package feishu

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	_ "modernc.org/sqlite"
)

const (
	reviewGatewayDefaultMaxAttempts = 3
	reviewGatewayReplyMaxAttempts   = 3
	reviewGatewaySQLiteBusyMS       = 400
)

const reviewGatewayStateSchema = `
CREATE TABLE IF NOT EXISTS review_gateway_events (
    dedupe_key TEXT PRIMARY KEY,
    received_at TEXT NOT NULL,
    expires_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS review_gateway_events_expires_at
    ON review_gateway_events(expires_at);

CREATE TABLE IF NOT EXISTS review_collaboration_items (
    repository TEXT NOT NULL,
    pr_number INTEGER NOT NULL,
    chat_id TEXT NOT NULL,
    assigned_to TEXT NOT NULL DEFAULT '',
    assigned_display_name TEXT NOT NULL DEFAULT '',
    collaboration_status TEXT NOT NULL DEFAULT 'unassigned',
    due_at TEXT NOT NULL DEFAULT '',
    updated_by TEXT NOT NULL DEFAULT '',
    updated_at TEXT NOT NULL,
    PRIMARY KEY(chat_id, repository, pr_number)
);

CREATE TABLE IF NOT EXISTS review_event_inbox (
    delivery_id TEXT PRIMARY KEY,
    event_type TEXT NOT NULL,
    repository TEXT NOT NULL,
    pr_number INTEGER NOT NULL,
    head_sha TEXT NOT NULL DEFAULT '',
    payload_json TEXT NOT NULL,
    status TEXT NOT NULL,
    attempt_count INTEGER NOT NULL DEFAULT 0,
    next_attempt_at TEXT NOT NULL,
    lease_owner TEXT NOT NULL DEFAULT '',
    lease_expires_at TEXT NOT NULL DEFAULT '',
    error_summary TEXT NOT NULL DEFAULT '',
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS review_event_inbox_ready
    ON review_event_inbox(status, next_attempt_at, lease_expires_at);

CREATE TABLE IF NOT EXISTS review_operations (
    operation_id TEXT PRIMARY KEY,
    dedupe_key TEXT NOT NULL UNIQUE,
    kind TEXT NOT NULL,
    status TEXT NOT NULL,
    payload_json TEXT NOT NULL,
    attempt_count INTEGER NOT NULL DEFAULT 0,
    max_attempts INTEGER NOT NULL DEFAULT 5,
    next_attempt_at TEXT NOT NULL,
    lease_owner TEXT NOT NULL DEFAULT '',
    lease_expires_at TEXT NOT NULL DEFAULT '',
    mutation_status TEXT NOT NULL DEFAULT 'not_started',
    reconciliation_state TEXT NOT NULL DEFAULT 'not_required',
    last_error TEXT NOT NULL DEFAULT '',
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS review_operations_ready
    ON review_operations(status, next_attempt_at, lease_expires_at);

CREATE TABLE IF NOT EXISTS review_dead_letters (
    dead_letter_id TEXT PRIMARY KEY,
    source_type TEXT NOT NULL,
    source_id TEXT NOT NULL,
    payload_json TEXT NOT NULL,
    error_summary TEXT NOT NULL,
    created_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS review_gateway_jobs (
    job_id TEXT PRIMARY KEY,
    dedupe_key TEXT NOT NULL,
    status TEXT NOT NULL,
    action TEXT NOT NULL,
    repository TEXT,
    pr_number INTEGER,
    requested_by TEXT NOT NULL,
    chat_id TEXT NOT NULL,
    payload_json TEXT NOT NULL,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    error_summary TEXT NOT NULL DEFAULT '',
    attempt_count INTEGER NOT NULL DEFAULT 0,
    max_attempts INTEGER NOT NULL DEFAULT 3,
    next_attempt_at TEXT NOT NULL DEFAULT '',
    lease_owner TEXT NOT NULL DEFAULT '',
    lease_expires_at TEXT NOT NULL DEFAULT '',
    result_json TEXT NOT NULL DEFAULT '',
    handler_latency_ms INTEGER NOT NULL DEFAULT 0,
    reply_status TEXT NOT NULL DEFAULT 'none',
    reply_attempt_count INTEGER NOT NULL DEFAULT 0,
    reply_next_attempt_at TEXT NOT NULL DEFAULT '',
    reply_lease_owner TEXT NOT NULL DEFAULT '',
    reply_lease_expires_at TEXT NOT NULL DEFAULT '',
    reply_message_id TEXT NOT NULL DEFAULT '',
    reply_error_summary TEXT NOT NULL DEFAULT ''
);
CREATE INDEX IF NOT EXISTS review_gateway_jobs_status
    ON review_gateway_jobs(status);
`

var reviewGatewayJobMigrations = map[string]string{
	"attempt_count":          "INTEGER NOT NULL DEFAULT 0",
	"max_attempts":           "INTEGER NOT NULL DEFAULT 3",
	"next_attempt_at":        "TEXT NOT NULL DEFAULT ''",
	"lease_owner":            "TEXT NOT NULL DEFAULT ''",
	"lease_expires_at":       "TEXT NOT NULL DEFAULT ''",
	"result_json":            "TEXT NOT NULL DEFAULT ''",
	"handler_latency_ms":     "INTEGER NOT NULL DEFAULT 0",
	"reply_status":           "TEXT NOT NULL DEFAULT 'none'",
	"reply_attempt_count":    "INTEGER NOT NULL DEFAULT 0",
	"reply_next_attempt_at":  "TEXT NOT NULL DEFAULT ''",
	"reply_lease_owner":      "TEXT NOT NULL DEFAULT ''",
	"reply_lease_expires_at": "TEXT NOT NULL DEFAULT ''",
	"reply_message_id":       "TEXT NOT NULL DEFAULT ''",
	"reply_error_summary":    "TEXT NOT NULL DEFAULT ''",
}

var (
	reviewGatewayErrorURLPattern = regexp.MustCompile(`https?://[^\s"'<>]+`)
	reviewGatewayErrorRules      = []struct {
		pattern     *regexp.Regexp
		replacement string
	}{
		{
			pattern:     regexp.MustCompile(`(?i)(authorization\s*[:=]\s*bearer\s+)[^\s,;]+`),
			replacement: "${1}***",
		},
		{
			pattern:     regexp.MustCompile(`(?i)(cookie|set-cookie|x-auth-token|private-token|access_token|refresh_token|client_secret|app_secret)(\s*[:=]\s*)[^\s,;]+`),
			replacement: "${1}${2}***",
		},
		{
			pattern:     regexp.MustCompile(`(?i)(token|secret|password)(\s*[:=]\s*)[^\s,;]+`),
			replacement: "${1}${2}***",
		},
	}
)

type ReviewGatewayClaimOptions struct {
	LeaseOwner    string
	Now           time.Time
	LeaseDuration time.Duration
	Limit         int
}

type ReviewGatewayJobOutcome struct {
	Job       ReviewGatewayJob
	Result    ReviewGatewayExecutionResult
	Err       error
	WillRetry bool
}

type ReviewGatewayPendingReply struct {
	Job          ReviewGatewayJob
	Result       ReviewGatewayExecutionResult
	AttemptCount int
}

type ReviewGatewayJobStore interface {
	SaveJob(ctx context.Context, job ReviewGatewayJob) error
	ClaimReadyJobs(ctx context.Context, opts ReviewGatewayClaimOptions) ([]ReviewGatewayJob, error)
	CompleteJob(ctx context.Context, job ReviewGatewayJob, result ReviewGatewayExecutionResult) error
	RetryOrFailJob(ctx context.Context, job ReviewGatewayJob, result ReviewGatewayExecutionResult, errorSummary string, now time.Time) (bool, error)
	RecordHandlerLatency(ctx context.Context, jobID string, latencyMs int64) error
	ClaimPendingReplies(ctx context.Context, opts ReviewGatewayClaimOptions) ([]ReviewGatewayPendingReply, error)
	MarkReplySent(ctx context.Context, jobID, messageID string, now time.Time) error
	MarkReplyFailed(ctx context.Context, pending ReviewGatewayPendingReply, errorSummary string, now time.Time) (bool, error)
	UpdateJobStatus(ctx context.Context, jobID, status, errorSummary string) error
}

type ReviewGatewayStateStore interface {
	ReviewGatewayDeduper
	ReviewGatewayJobStore
	Close() error
}

type MemoryReviewGatewayJobStore struct {
	mu            sync.Mutex
	Jobs          map[string]ReviewGatewayJob
	Status        map[string]string
	Errors        map[string]string
	Results       map[string]ReviewGatewayExecutionResult
	ReplyStatus   map[string]string
	ReplyAttempts map[string]int
	ReplyLeases   map[string]string
}

type SQLiteReviewGatewayStore struct {
	db *sql.DB
}

type reviewGatewayLatencyObservation struct {
	JobID     string
	LatencyMs int64
}

type ReviewGatewayQueue struct {
	gateway       *ReviewGateway
	store         ReviewGatewayJobStore
	wake          chan struct{}
	latencies     chan reviewGatewayLatencyObservation
	leaseOwner    string
	leaseDuration time.Duration
	pollInterval  time.Duration
	onResult      func(ReviewGatewayJobOutcome)
	now           func() time.Time
}

func NewMemoryReviewGatewayJobStore() *MemoryReviewGatewayJobStore {
	return &MemoryReviewGatewayJobStore{
		Jobs:          map[string]ReviewGatewayJob{},
		Status:        map[string]string{},
		Errors:        map[string]string{},
		Results:       map[string]ReviewGatewayExecutionResult{},
		ReplyStatus:   map[string]string{},
		ReplyAttempts: map[string]int{},
		ReplyLeases:   map[string]string{},
	}
}

func (s *MemoryReviewGatewayJobStore) SaveJob(_ context.Context, job ReviewGatewayJob) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.Jobs[job.JobID]; exists {
		return nil
	}
	if job.MaxAttempts <= 0 {
		job.MaxAttempts = reviewGatewayDefaultMaxAttempts
	}
	if job.NextAttemptAt == "" {
		job.NextAttemptAt = job.CreatedAt
	}
	s.Jobs[job.JobID] = job
	s.Status[job.JobID] = "queued"
	s.ReplyStatus[job.JobID] = "none"
	return nil
}

func (s *MemoryReviewGatewayJobStore) ClaimReadyJobs(_ context.Context, opts ReviewGatewayClaimOptions) ([]ReviewGatewayJob, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	normalizeReviewGatewayClaimOptions(&opts)
	ids := make([]string, 0, len(s.Jobs))
	for id := range s.Jobs {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	result := []ReviewGatewayJob{}
	for _, id := range ids {
		if len(result) >= opts.Limit {
			break
		}
		job := s.Jobs[id]
		status := s.Status[id]
		if status == "running" && reviewGatewayTimeDue(job.LeaseExpiresAt, opts.Now) {
			status = "queued"
			s.Status[id] = status
		}
		if status != "queued" || !reviewGatewayTimeDue(job.NextAttemptAt, opts.Now) || job.AttemptCount >= job.MaxAttempts {
			continue
		}
		job.AttemptCount++
		job.Status = "running"
		job.LeaseOwner = opts.LeaseOwner
		job.LeaseExpiresAt = opts.Now.Add(opts.LeaseDuration).UTC().Format(time.RFC3339Nano)
		s.Jobs[id] = job
		s.Status[id] = "running"
		result = append(result, job)
	}
	return result, nil
}

func (s *MemoryReviewGatewayJobStore) CompleteJob(_ context.Context, job ReviewGatewayJob, result ReviewGatewayExecutionResult) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Results[job.JobID] = result
	s.Status[job.JobID] = "completed"
	job.Status = "completed"
	job.LeaseOwner = ""
	job.LeaseExpiresAt = ""
	s.Jobs[job.JobID] = job
	if job.SourceMessageID != "" {
		s.ReplyStatus[job.JobID] = "pending"
	}
	return nil
}

func (s *MemoryReviewGatewayJobStore) RetryOrFailJob(_ context.Context, job ReviewGatewayJob, result ReviewGatewayExecutionResult, errorSummary string, now time.Time) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Results[job.JobID] = result
	s.Errors[job.JobID] = redactReviewGatewayError(errorSummary)
	job.LeaseOwner = ""
	job.LeaseExpiresAt = ""
	willRetry := job.AttemptCount < job.MaxAttempts
	if willRetry {
		job.Status = "queued"
		job.NextAttemptAt = now.Add(reviewGatewayRetryDelay(job.AttemptCount)).UTC().Format(time.RFC3339Nano)
		s.Status[job.JobID] = "queued"
	} else {
		job.Status = "failed"
		s.Status[job.JobID] = "failed"
		if job.SourceMessageID != "" {
			s.ReplyStatus[job.JobID] = "pending"
		}
	}
	s.Jobs[job.JobID] = job
	return willRetry, nil
}

func (s *MemoryReviewGatewayJobStore) RecordHandlerLatency(_ context.Context, jobID string, latencyMs int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	job, ok := s.Jobs[jobID]
	if !ok {
		return fmt.Errorf("review gateway job %q not found", jobID)
	}
	job.HandlerLatencyMs = latencyMs
	s.Jobs[jobID] = job
	return nil
}

func (s *MemoryReviewGatewayJobStore) ClaimPendingReplies(_ context.Context, opts ReviewGatewayClaimOptions) ([]ReviewGatewayPendingReply, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	normalizeReviewGatewayClaimOptions(&opts)
	ids := make([]string, 0, len(s.Jobs))
	for id := range s.Jobs {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	result := []ReviewGatewayPendingReply{}
	for _, id := range ids {
		if len(result) >= opts.Limit {
			break
		}
		status := s.ReplyStatus[id]
		job := s.Jobs[id]
		if status == "sending" && reviewGatewayTimeDue(s.ReplyLeases[id], opts.Now) {
			status = "pending"
			s.ReplyStatus[id] = status
		}
		if status != "pending" || s.ReplyAttempts[id] >= reviewGatewayReplyMaxAttempts {
			continue
		}
		s.ReplyStatus[id] = "sending"
		s.ReplyAttempts[id]++
		s.ReplyLeases[id] = opts.Now.Add(opts.LeaseDuration).UTC().Format(time.RFC3339Nano)
		result = append(result, ReviewGatewayPendingReply{
			Job:          job,
			Result:       s.Results[id],
			AttemptCount: s.ReplyAttempts[id],
		})
	}
	return result, nil
}

func (s *MemoryReviewGatewayJobStore) MarkReplySent(_ context.Context, jobID, _ string, _ time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ReplyStatus[jobID] = "sent"
	s.ReplyLeases[jobID] = ""
	return nil
}

func (s *MemoryReviewGatewayJobStore) MarkReplyFailed(_ context.Context, pending ReviewGatewayPendingReply, errorSummary string, _ time.Time) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Errors[pending.Job.JobID] = redactReviewGatewayError(errorSummary)
	willRetry := pending.AttemptCount < reviewGatewayReplyMaxAttempts
	if willRetry {
		s.ReplyStatus[pending.Job.JobID] = "pending"
	} else {
		s.ReplyStatus[pending.Job.JobID] = "failed"
	}
	s.ReplyLeases[pending.Job.JobID] = ""
	return willRetry, nil
}

func (s *MemoryReviewGatewayJobStore) UpdateJobStatus(_ context.Context, jobID, status, errorSummary string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.Jobs[jobID]; !ok {
		return fmt.Errorf("review gateway job %q not found", jobID)
	}
	s.Status[jobID] = status
	s.Errors[jobID] = redactReviewGatewayError(errorSummary)
	return nil
}

func OpenSQLiteReviewGatewayStore(path string) (*SQLiteReviewGatewayStore, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return nil, fmt.Errorf("review gateway state database path is required")
	}
	absolute, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("resolve review gateway state database: %w", err)
	}
	if parent := filepath.Dir(absolute); parent != "." {
		if err := os.MkdirAll(parent, 0700); err != nil {
			return nil, fmt.Errorf("create review gateway state directory: %w", err)
		}
	}
	db, err := sql.Open("sqlite", absolute)
	if err != nil {
		return nil, fmt.Errorf("open review gateway state database: %w", err)
	}
	db.SetMaxOpenConns(1)
	for _, statement := range []string{
		"PRAGMA journal_mode=WAL",
		fmt.Sprintf("PRAGMA busy_timeout=%d", reviewGatewaySQLiteBusyMS),
		reviewGatewayStateSchema,
	} {
		if _, err := db.Exec(statement); err != nil {
			_ = db.Close()
			return nil, fmt.Errorf("initialize review gateway state database: %w", err)
		}
	}
	if err := ensureReviewGatewayJobColumns(db); err != nil {
		_ = db.Close()
		return nil, err
	}
	for _, statement := range []string{
		`CREATE INDEX IF NOT EXISTS review_gateway_jobs_ready
		 ON review_gateway_jobs(status, next_attempt_at)`,
		`CREATE INDEX IF NOT EXISTS review_gateway_jobs_reply_ready
		 ON review_gateway_jobs(reply_status, reply_next_attempt_at)`,
	} {
		if _, err := db.Exec(statement); err != nil {
			_ = db.Close()
			return nil, fmt.Errorf("initialize review gateway state index: %w", err)
		}
	}
	return &SQLiteReviewGatewayStore{db: db}, nil
}

func ensureReviewGatewayJobColumns(db *sql.DB) error {
	rows, err := db.Query("PRAGMA table_info(review_gateway_jobs)")
	if err != nil {
		return fmt.Errorf("inspect review gateway job schema: %w", err)
	}
	existing := map[string]bool{}
	for rows.Next() {
		var cid, notNull, primaryKey int
		var name, columnType string
		var defaultValue sql.NullString
		if err := rows.Scan(&cid, &name, &columnType, &notNull, &defaultValue, &primaryKey); err != nil {
			_ = rows.Close()
			return fmt.Errorf("read review gateway job schema: %w", err)
		}
		existing[name] = true
	}
	if err := rows.Close(); err != nil {
		return fmt.Errorf("close review gateway job schema rows: %w", err)
	}
	names := make([]string, 0, len(reviewGatewayJobMigrations))
	for name := range reviewGatewayJobMigrations {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		if existing[name] {
			continue
		}
		statement := fmt.Sprintf("ALTER TABLE review_gateway_jobs ADD COLUMN %s %s", name, reviewGatewayJobMigrations[name])
		if _, err := db.Exec(statement); err != nil {
			return fmt.Errorf("migrate review gateway job column %s: %w", name, err)
		}
	}
	return nil
}

func (s *SQLiteReviewGatewayStore) Reserve(key string, now time.Time, ttl time.Duration) (bool, error) {
	return s.ReserveContext(context.Background(), key, now, ttl)
}

func (s *SQLiteReviewGatewayStore) ReserveContext(ctx context.Context, key string, now time.Time, ttl time.Duration) (bool, error) {
	if s == nil || s.db == nil || strings.TrimSpace(key) == "" {
		return false, fmt.Errorf("review gateway state store and dedupe key are required")
	}
	if ttl <= 0 {
		ttl = 24 * time.Hour
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return false, fmt.Errorf("begin review gateway dedupe transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	_, _ = tx.ExecContext(ctx, "DELETE FROM review_gateway_events WHERE expires_at <= ?", reviewGatewayTimestamp(now))
	result, err := tx.ExecContext(
		ctx,
		"INSERT OR IGNORE INTO review_gateway_events(dedupe_key, received_at, expires_at) VALUES(?, ?, ?)",
		key,
		reviewGatewayTimestamp(now),
		reviewGatewayTimestamp(now.Add(ttl)),
	)
	if err != nil {
		return false, fmt.Errorf("reserve review gateway event: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("read review gateway event reservation: %w", err)
	}
	if affected != 1 {
		return false, nil
	}
	if err := tx.Commit(); err != nil {
		return false, fmt.Errorf("commit review gateway event reservation: %w", err)
	}
	return true, nil
}

func (s *SQLiteReviewGatewayStore) Release(key string) {
	if s == nil || s.db == nil || strings.TrimSpace(key) == "" {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()
	_, _ = s.db.ExecContext(ctx, "DELETE FROM review_gateway_events WHERE dedupe_key = ?", key)
}

func (s *SQLiteReviewGatewayStore) SaveJob(ctx context.Context, job ReviewGatewayJob) error {
	if job.MaxAttempts <= 0 {
		job.MaxAttempts = reviewGatewayDefaultMaxAttempts
	}
	if job.NextAttemptAt == "" {
		job.NextAttemptAt = job.CreatedAt
	}
	payload, err := json.Marshal(job)
	if err != nil {
		return fmt.Errorf("encode review gateway job: %w", err)
	}
	now := reviewGatewayTimestamp(time.Now())
	_, err = s.db.ExecContext(
		ctx,
		`INSERT INTO review_gateway_jobs(
			job_id, dedupe_key, status, action, repository, pr_number,
			requested_by, chat_id, payload_json, created_at, updated_at,
			error_summary, attempt_count, max_attempts, next_attempt_at
		) VALUES(?, ?, 'queued', ?, ?, ?, ?, ?, ?, ?, ?, '', 0, ?, ?)
		ON CONFLICT(job_id) DO NOTHING`,
		job.JobID,
		job.DedupeKey,
		job.Action,
		job.Repository,
		job.PRNumber,
		job.RequestedBy,
		job.ChatID,
		string(payload),
		job.CreatedAt,
		now,
		job.MaxAttempts,
		job.NextAttemptAt,
	)
	if err != nil {
		return fmt.Errorf("save review gateway job: %w", err)
	}
	return nil
}

func (s *SQLiteReviewGatewayStore) ReserveAndSaveJob(ctx context.Context, job ReviewGatewayJob, now time.Time, ttl time.Duration) (bool, error) {
	if s == nil || s.db == nil || strings.TrimSpace(job.DedupeKey) == "" {
		return false, fmt.Errorf("review gateway state store and dedupe key are required")
	}
	if ttl <= 0 {
		ttl = 24 * time.Hour
	}
	if job.MaxAttempts <= 0 {
		job.MaxAttempts = reviewGatewayDefaultMaxAttempts
	}
	if job.NextAttemptAt == "" {
		job.NextAttemptAt = job.CreatedAt
	}
	payload, err := json.Marshal(job)
	if err != nil {
		return false, fmt.Errorf("encode review gateway job: %w", err)
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return false, fmt.Errorf("begin atomic review gateway enqueue: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	nowText := reviewGatewayTimestamp(now)
	_, _ = tx.ExecContext(ctx, "DELETE FROM review_gateway_events WHERE expires_at <= ?", nowText)
	reservation, err := tx.ExecContext(
		ctx,
		"INSERT OR IGNORE INTO review_gateway_events(dedupe_key, received_at, expires_at) VALUES(?, ?, ?)",
		job.DedupeKey,
		nowText,
		reviewGatewayTimestamp(now.Add(ttl)),
	)
	if err != nil {
		return false, fmt.Errorf("reserve atomic review gateway event: %w", err)
	}
	affected, err := reservation.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("read atomic review gateway reservation: %w", err)
	}
	if affected == 0 {
		var existingJobID string
		err := tx.QueryRowContext(
			ctx,
			"SELECT job_id FROM review_gateway_jobs WHERE dedupe_key=? LIMIT 1",
			job.DedupeKey,
		).Scan(&existingJobID)
		if err == nil {
			return false, nil
		}
		if !errors.Is(err, sql.ErrNoRows) {
			return false, fmt.Errorf("inspect atomic review gateway reservation: %w", err)
		}
		// Recover a reservation left by the P2.0 two-transaction enqueue path.
		if _, err := tx.ExecContext(
			ctx,
			"UPDATE review_gateway_events SET received_at=?, expires_at=? WHERE dedupe_key=?",
			nowText,
			reviewGatewayTimestamp(now.Add(ttl)),
			job.DedupeKey,
		); err != nil {
			return false, fmt.Errorf("recover orphan review gateway reservation: %w", err)
		}
	}
	saved, err := tx.ExecContext(
		ctx,
		`INSERT INTO review_gateway_jobs(
			job_id, dedupe_key, status, action, repository, pr_number,
			requested_by, chat_id, payload_json, created_at, updated_at,
			error_summary, attempt_count, max_attempts, next_attempt_at
		) VALUES(?, ?, 'queued', ?, ?, ?, ?, ?, ?, ?, ?, '', 0, ?, ?)
		ON CONFLICT(job_id) DO NOTHING`,
		job.JobID,
		job.DedupeKey,
		job.Action,
		job.Repository,
		job.PRNumber,
		job.RequestedBy,
		job.ChatID,
		string(payload),
		job.CreatedAt,
		nowText,
		job.MaxAttempts,
		job.NextAttemptAt,
	)
	if err != nil {
		return false, fmt.Errorf("save atomic review gateway job: %w", err)
	}
	savedCount, err := saved.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("read atomic review gateway job insert: %w", err)
	}
	if savedCount != 1 {
		return false, nil
	}
	if err := tx.Commit(); err != nil {
		return false, fmt.Errorf("commit atomic review gateway enqueue: %w", err)
	}
	return true, nil
}

func (s *SQLiteReviewGatewayStore) ClaimReadyJobs(ctx context.Context, opts ReviewGatewayClaimOptions) ([]ReviewGatewayJob, error) {
	normalizeReviewGatewayClaimOptions(&opts)
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin review gateway job claim: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	nowText := reviewGatewayTimestamp(opts.Now)
	_, err = tx.ExecContext(
		ctx,
		`UPDATE review_gateway_jobs
		 SET status='queued', lease_owner='', lease_expires_at='', updated_at=?
		 WHERE status='running' AND lease_expires_at != '' AND lease_expires_at <= ?`,
		nowText,
		nowText,
	)
	if err != nil {
		return nil, fmt.Errorf("recover expired review gateway jobs: %w", err)
	}
	rows, err := tx.QueryContext(
		ctx,
		`SELECT job_id, payload_json, attempt_count, max_attempts
		 FROM review_gateway_jobs
		 WHERE status='queued'
		   AND (next_attempt_at='' OR next_attempt_at <= ?)
		   AND attempt_count < max_attempts
		 ORDER BY created_at, job_id
		 LIMIT ?`,
		nowText,
		opts.Limit,
	)
	if err != nil {
		return nil, fmt.Errorf("list ready review gateway jobs: %w", err)
	}
	type candidate struct {
		id          string
		payload     string
		attempts    int
		maxAttempts int
	}
	candidates := []candidate{}
	for rows.Next() {
		var item candidate
		if err := rows.Scan(&item.id, &item.payload, &item.attempts, &item.maxAttempts); err != nil {
			_ = rows.Close()
			return nil, fmt.Errorf("scan ready review gateway job: %w", err)
		}
		candidates = append(candidates, item)
	}
	if err := rows.Close(); err != nil {
		return nil, fmt.Errorf("close ready review gateway jobs: %w", err)
	}
	result := []ReviewGatewayJob{}
	for _, candidate := range candidates {
		leaseExpiresAt := reviewGatewayTimestamp(opts.Now.Add(opts.LeaseDuration))
		update, err := tx.ExecContext(
			ctx,
			`UPDATE review_gateway_jobs
			 SET status='running',
			     attempt_count=attempt_count+1,
			     lease_owner=?,
			     lease_expires_at=?,
			     updated_at=?
			 WHERE job_id=? AND status='queued'`,
			opts.LeaseOwner,
			leaseExpiresAt,
			nowText,
			candidate.id,
		)
		if err != nil {
			return nil, fmt.Errorf("lease review gateway job %s: %w", candidate.id, err)
		}
		affected, err := update.RowsAffected()
		if err != nil {
			return nil, fmt.Errorf("read review gateway job lease %s: %w", candidate.id, err)
		}
		if affected != 1 {
			continue
		}
		var job ReviewGatewayJob
		if err := json.Unmarshal([]byte(candidate.payload), &job); err != nil {
			return nil, fmt.Errorf("decode review gateway job %s: %w", candidate.id, err)
		}
		job.Status = "running"
		job.AttemptCount = candidate.attempts + 1
		job.MaxAttempts = candidate.maxAttempts
		job.LeaseOwner = opts.LeaseOwner
		job.LeaseExpiresAt = leaseExpiresAt
		result = append(result, job)
	}
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit review gateway job claims: %w", err)
	}
	return result, nil
}

func (s *SQLiteReviewGatewayStore) CompleteJob(ctx context.Context, job ReviewGatewayJob, result ReviewGatewayExecutionResult) error {
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return fmt.Errorf("encode review gateway result: %w", err)
	}
	replyStatus := "none"
	replyNext := ""
	if job.SourceMessageID != "" {
		replyStatus = "pending"
		replyNext = reviewGatewayTimestamp(time.Now())
	}
	update, err := s.db.ExecContext(
		ctx,
		`UPDATE review_gateway_jobs
		 SET status='completed',
		     result_json=?,
		     error_summary='',
		     lease_owner='',
		     lease_expires_at='',
		     reply_status=?,
		     reply_next_attempt_at=?,
		     updated_at=?
		 WHERE job_id=? AND status='running' AND lease_owner=?`,
		string(resultJSON),
		replyStatus,
		replyNext,
		reviewGatewayTimestamp(time.Now()),
		job.JobID,
		job.LeaseOwner,
	)
	return requireReviewGatewayJobUpdate(update, err, job.JobID, "complete")
}

func (s *SQLiteReviewGatewayStore) RetryOrFailJob(ctx context.Context, job ReviewGatewayJob, result ReviewGatewayExecutionResult, errorSummary string, now time.Time) (bool, error) {
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return false, fmt.Errorf("encode failed review gateway result: %w", err)
	}
	willRetry := job.AttemptCount < job.MaxAttempts
	status := "failed"
	nextAttempt := ""
	replyStatus := "none"
	replyNext := ""
	if willRetry {
		status = "queued"
		nextAttempt = reviewGatewayTimestamp(now.Add(reviewGatewayRetryDelay(job.AttemptCount)))
	} else if job.SourceMessageID != "" {
		replyStatus = "pending"
		replyNext = reviewGatewayTimestamp(now)
	}
	update, err := s.db.ExecContext(
		ctx,
		`UPDATE review_gateway_jobs
		 SET status=?,
		     next_attempt_at=?,
		     lease_owner='',
		     lease_expires_at='',
		     result_json=?,
		     error_summary=?,
		     reply_status=?,
		     reply_next_attempt_at=?,
		     updated_at=?
		 WHERE job_id=? AND status='running' AND lease_owner=?`,
		status,
		nextAttempt,
		string(resultJSON),
		redactReviewGatewayError(errorSummary),
		replyStatus,
		replyNext,
		reviewGatewayTimestamp(now),
		job.JobID,
		job.LeaseOwner,
	)
	if err := requireReviewGatewayJobUpdate(update, err, job.JobID, "retry or fail"); err != nil {
		return false, err
	}
	return willRetry, nil
}

func (s *SQLiteReviewGatewayStore) RecordHandlerLatency(ctx context.Context, jobID string, latencyMs int64) error {
	if strings.TrimSpace(jobID) == "" {
		return nil
	}
	_, err := s.db.ExecContext(
		ctx,
		"UPDATE review_gateway_jobs SET handler_latency_ms=? WHERE job_id=?",
		latencyMs,
		jobID,
	)
	if err != nil {
		return fmt.Errorf("record review gateway handler latency: %w", err)
	}
	return nil
}

func (s *SQLiteReviewGatewayStore) ClaimPendingReplies(ctx context.Context, opts ReviewGatewayClaimOptions) ([]ReviewGatewayPendingReply, error) {
	normalizeReviewGatewayClaimOptions(&opts)
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin review gateway reply claim: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	nowText := reviewGatewayTimestamp(opts.Now)
	_, err = tx.ExecContext(
		ctx,
		`UPDATE review_gateway_jobs
		 SET reply_status='pending', reply_lease_owner='', reply_lease_expires_at='', updated_at=?
		 WHERE reply_status='sending'
		   AND reply_lease_expires_at != ''
		   AND reply_lease_expires_at <= ?`,
		nowText,
		nowText,
	)
	if err != nil {
		return nil, fmt.Errorf("recover expired review gateway replies: %w", err)
	}
	rows, err := tx.QueryContext(
		ctx,
		`SELECT job_id, payload_json, result_json, reply_attempt_count
		 FROM review_gateway_jobs
		 WHERE reply_status='pending'
		   AND result_json != ''
		   AND (reply_next_attempt_at='' OR reply_next_attempt_at <= ?)
		   AND reply_attempt_count < ?
		 ORDER BY updated_at, job_id
		 LIMIT ?`,
		nowText,
		reviewGatewayReplyMaxAttempts,
		opts.Limit,
	)
	if err != nil {
		return nil, fmt.Errorf("list pending review gateway replies: %w", err)
	}
	type replyCandidate struct {
		id         string
		jobJSON    string
		resultJSON string
		attempts   int
	}
	candidates := []replyCandidate{}
	for rows.Next() {
		var candidate replyCandidate
		if err := rows.Scan(&candidate.id, &candidate.jobJSON, &candidate.resultJSON, &candidate.attempts); err != nil {
			_ = rows.Close()
			return nil, fmt.Errorf("scan pending review gateway reply: %w", err)
		}
		candidates = append(candidates, candidate)
	}
	if err := rows.Close(); err != nil {
		return nil, fmt.Errorf("close pending review gateway replies: %w", err)
	}
	result := []ReviewGatewayPendingReply{}
	for _, candidate := range candidates {
		leaseExpiresAt := reviewGatewayTimestamp(opts.Now.Add(opts.LeaseDuration))
		update, err := tx.ExecContext(
			ctx,
			`UPDATE review_gateway_jobs
			 SET reply_status='sending',
			     reply_attempt_count=reply_attempt_count+1,
			     reply_lease_owner=?,
			     reply_lease_expires_at=?,
			     updated_at=?
			 WHERE job_id=? AND reply_status='pending'`,
			opts.LeaseOwner,
			leaseExpiresAt,
			nowText,
			candidate.id,
		)
		if err != nil {
			return nil, fmt.Errorf("lease review gateway reply %s: %w", candidate.id, err)
		}
		affected, err := update.RowsAffected()
		if err != nil {
			return nil, fmt.Errorf("read review gateway reply lease %s: %w", candidate.id, err)
		}
		if affected != 1 {
			continue
		}
		var job ReviewGatewayJob
		var executionResult ReviewGatewayExecutionResult
		if err := json.Unmarshal([]byte(candidate.jobJSON), &job); err != nil {
			return nil, fmt.Errorf("decode review gateway reply job %s: %w", candidate.id, err)
		}
		if err := json.Unmarshal([]byte(candidate.resultJSON), &executionResult); err != nil {
			return nil, fmt.Errorf("decode review gateway result %s: %w", candidate.id, err)
		}
		result = append(result, ReviewGatewayPendingReply{
			Job:          job,
			Result:       executionResult,
			AttemptCount: candidate.attempts + 1,
		})
	}
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit review gateway reply claims: %w", err)
	}
	return result, nil
}

func (s *SQLiteReviewGatewayStore) MarkReplySent(ctx context.Context, jobID, messageID string, now time.Time) error {
	update, err := s.db.ExecContext(
		ctx,
		`UPDATE review_gateway_jobs
		 SET reply_status='sent',
		     reply_message_id=?,
		     reply_error_summary='',
		     reply_lease_owner='',
		     reply_lease_expires_at='',
		     updated_at=?
		 WHERE job_id=? AND reply_status='sending'`,
		messageID,
		reviewGatewayTimestamp(now),
		jobID,
	)
	return requireReviewGatewayJobUpdate(update, err, jobID, "mark reply sent")
}

func (s *SQLiteReviewGatewayStore) MarkReplyFailed(ctx context.Context, pending ReviewGatewayPendingReply, errorSummary string, now time.Time) (bool, error) {
	willRetry := pending.AttemptCount < reviewGatewayReplyMaxAttempts
	status := "failed"
	nextAttempt := ""
	if willRetry {
		status = "pending"
		nextAttempt = reviewGatewayTimestamp(now.Add(reviewGatewayRetryDelay(pending.AttemptCount)))
	}
	update, err := s.db.ExecContext(
		ctx,
		`UPDATE review_gateway_jobs
		 SET reply_status=?,
		     reply_next_attempt_at=?,
		     reply_error_summary=?,
		     reply_lease_owner='',
		     reply_lease_expires_at='',
		     updated_at=?
		 WHERE job_id=? AND reply_status='sending'`,
		status,
		nextAttempt,
		redactReviewGatewayError(errorSummary),
		reviewGatewayTimestamp(now),
		pending.Job.JobID,
	)
	if err := requireReviewGatewayJobUpdate(update, err, pending.Job.JobID, "mark reply failed"); err != nil {
		return false, err
	}
	return willRetry, nil
}

func (s *SQLiteReviewGatewayStore) UpdateJobStatus(ctx context.Context, jobID, status, errorSummary string) error {
	update, err := s.db.ExecContext(
		ctx,
		`UPDATE review_gateway_jobs
		 SET status=?, updated_at=?, error_summary=?,
		     lease_owner='', lease_expires_at=''
		 WHERE job_id=?`,
		status,
		reviewGatewayTimestamp(time.Now()),
		redactReviewGatewayError(errorSummary),
		jobID,
	)
	return requireReviewGatewayJobUpdate(update, err, jobID, "update status")
}

func (s *SQLiteReviewGatewayStore) Close() error {
	if s == nil || s.db == nil {
		return nil
	}
	return s.db.Close()
}

func NewReviewGatewayQueue(gateway *ReviewGateway, store ReviewGatewayJobStore, capacity int, onResult func(ReviewGatewayJobOutcome)) *ReviewGatewayQueue {
	if capacity <= 0 {
		capacity = 128
	}
	if store == nil {
		store = NewMemoryReviewGatewayJobStore()
	}
	return &ReviewGatewayQueue{
		gateway:       gateway,
		store:         store,
		wake:          make(chan struct{}, 1),
		latencies:     make(chan reviewGatewayLatencyObservation, capacity),
		leaseOwner:    fmt.Sprintf("worker-%d-%d", os.Getpid(), time.Now().UnixNano()),
		leaseDuration: 2 * time.Minute,
		pollInterval:  time.Second,
		onResult:      onResult,
		now:           time.Now,
	}
}

func (q *ReviewGatewayQueue) Enqueue(ctx context.Context, event ReviewGatewayEvent) ReviewGatewayReceipt {
	if q == nil || q.gateway == nil {
		return ReviewGatewayReceipt{
			SchemaVersion: reviewGatewaySchemaVersion,
			Mode:          "preview",
			Reason:        "gateway_not_configured",
		}
	}
	if sqliteStore, ok := q.store.(*SQLiteReviewGatewayStore); ok {
		if dedupeStore, sameStore := q.gateway.deduper.(*SQLiteReviewGatewayStore); sameStore && dedupeStore == sqliteStore {
			receipt, err := q.gateway.planWithoutReserveContext(ctx, event)
			if err != nil {
				receipt.Accepted = false
				receipt.Reason = "state_store_failed"
				return receipt
			}
			if !receipt.Accepted || receipt.Job == nil {
				return receipt
			}
			saved, err := sqliteStore.ReserveAndSaveJob(ctx, *receipt.Job, q.now().UTC(), 24*time.Hour)
			if err != nil {
				receipt.Accepted = false
				receipt.Reason = "state_store_failed"
				receipt.Job = nil
				return receipt
			}
			if !saved {
				receipt.Accepted = false
				receipt.Duplicate = true
				receipt.Reason = "duplicate_event"
				receipt.Job = nil
				return receipt
			}
			q.Wake()
			return receipt
		}
	}
	receipt, err := q.gateway.PlanContext(ctx, event)
	if err != nil {
		receipt.Accepted = false
		receipt.Reason = "state_store_failed"
		return receipt
	}
	if !receipt.Accepted || receipt.Job == nil {
		return receipt
	}
	job := *receipt.Job
	if err := q.store.SaveJob(ctx, job); err != nil {
		q.gateway.Release(receipt)
		receipt.Accepted = false
		receipt.Reason = "state_store_failed"
		receipt.Job = nil
		return receipt
	}
	q.Wake()
	return receipt
}

func (q *ReviewGatewayQueue) Wake() {
	if q == nil {
		return
	}
	select {
	case q.wake <- struct{}{}:
	default:
	}
}

func (q *ReviewGatewayQueue) ObserveHandlerLatency(jobID string, latencyMs int64) {
	if q == nil || strings.TrimSpace(jobID) == "" {
		return
	}
	select {
	case q.latencies <- reviewGatewayLatencyObservation{JobID: jobID, LatencyMs: latencyMs}:
	default:
	}
}

func (q *ReviewGatewayQueue) Run(ctx context.Context, handler func(context.Context, ReviewGatewayJob) (ReviewGatewayExecutionResult, error)) {
	ticker := time.NewTicker(q.pollInterval)
	defer ticker.Stop()
	go q.runLatencyWriter(ctx)
	q.Wake()
	for {
		select {
		case <-ctx.Done():
			return
		case <-q.wake:
			q.runReadyJobs(ctx, handler)
		case <-ticker.C:
			q.runReadyJobs(ctx, handler)
		}
	}
}

func (q *ReviewGatewayQueue) runLatencyWriter(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case observation := <-q.latencies:
			metricCtx, cancel := context.WithTimeout(ctx, 500*time.Millisecond)
			_ = q.store.RecordHandlerLatency(metricCtx, observation.JobID, observation.LatencyMs)
			cancel()
		}
	}
}

func (q *ReviewGatewayQueue) runReadyJobs(ctx context.Context, handler func(context.Context, ReviewGatewayJob) (ReviewGatewayExecutionResult, error)) {
	for {
		claimCtx, cancel := context.WithTimeout(ctx, 500*time.Millisecond)
		jobs, err := q.store.ClaimReadyJobs(claimCtx, ReviewGatewayClaimOptions{
			LeaseOwner:    q.leaseOwner,
			Now:           q.now().UTC(),
			LeaseDuration: q.leaseDuration,
			Limit:         1,
		})
		cancel()
		if err != nil || len(jobs) == 0 {
			return
		}
		job := jobs[0]
		var result ReviewGatewayExecutionResult
		var executeErr error
		if handler == nil {
			executeErr = fmt.Errorf("review gateway job handler is not configured")
			result = failedReviewGatewayResult(job, executeErr, q.now())
		} else {
			result, executeErr = handler(ctx, job)
		}
		outcome := ReviewGatewayJobOutcome{Job: job, Result: result, Err: executeErr}
		persistCtx, persistCancel := context.WithTimeout(ctx, 2*time.Second)
		if executeErr == nil {
			executeErr = q.store.CompleteJob(persistCtx, job, result)
			outcome.Err = executeErr
		} else {
			outcome.WillRetry, err = q.store.RetryOrFailJob(
				persistCtx,
				job,
				result,
				executeErr.Error(),
				q.now().UTC(),
			)
			if err != nil {
				outcome.Err = err
			}
		}
		persistCancel()
		if q.onResult != nil {
			q.onResult(outcome)
		}
		if outcome.WillRetry {
			return
		}
	}
}

func failedReviewGatewayResult(job ReviewGatewayJob, err error, now time.Time) ReviewGatewayExecutionResult {
	return ReviewGatewayExecutionResult{
		SchemaVersion:   reviewGatewayResultSchema,
		JobID:           job.JobID,
		Status:          "failed",
		Mode:            "preview",
		Action:          job.Action,
		Repository:      job.Repository,
		PRNumber:        job.PRNumber,
		RequestedBy:     job.RequestedBy,
		ReadOnlyGitLink: true,
		MutatesGitLink:  false,
		CompletedAt:     now.UTC().Format(time.RFC3339),
		Error:           redactReviewGatewayError(err.Error()),
		AttemptCount:    job.AttemptCount,
	}
}

func normalizeReviewGatewayClaimOptions(opts *ReviewGatewayClaimOptions) {
	if opts.Now.IsZero() {
		opts.Now = time.Now().UTC()
	}
	if opts.LeaseDuration <= 0 {
		opts.LeaseDuration = 2 * time.Minute
	}
	if opts.Limit <= 0 {
		opts.Limit = 1
	}
	if strings.TrimSpace(opts.LeaseOwner) == "" {
		opts.LeaseOwner = "review-gateway-worker"
	}
}

func reviewGatewayRetryDelay(attempt int) time.Duration {
	if attempt < 1 {
		attempt = 1
	}
	if attempt > 6 {
		attempt = 6
	}
	return time.Duration(1<<(attempt-1)) * 2 * time.Second
}

func reviewGatewayTimeDue(value string, now time.Time) bool {
	if strings.TrimSpace(value) == "" {
		return true
	}
	parsed, err := time.Parse(time.RFC3339Nano, value)
	return err != nil || !parsed.After(now)
}

func reviewGatewayTimestamp(value time.Time) string {
	return value.UTC().Format(time.RFC3339Nano)
}

func requireReviewGatewayJobUpdate(result sql.Result, err error, jobID, action string) error {
	if err != nil {
		return fmt.Errorf("%s review gateway job %s: %w", action, jobID, err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("read %s review gateway job %s: %w", action, jobID, err)
	}
	if affected != 1 {
		return fmt.Errorf("%s review gateway job %q: lease or status changed", action, jobID)
	}
	return nil
}

func redactReviewGatewayError(value string) string {
	value = strings.TrimSpace(strings.ReplaceAll(value, "\x00", ""))
	value = reviewGatewayErrorURLPattern.ReplaceAllStringFunc(value, func(raw string) string {
		parsed, err := url.Parse(raw)
		if err != nil || parsed.Host == "" {
			return "https://..."
		}
		parsed.User = nil
		parsed.RawQuery = ""
		parsed.Fragment = ""
		parts := strings.Split(strings.Trim(parsed.Path, "/"), "/")
		for i := range parts {
			if i > 0 && (strings.EqualFold(parts[i-1], "hook") || strings.EqualFold(parts[i-1], "webhook")) {
				parts[i] = "***"
			}
		}
		parsed.Path = "/" + strings.Join(parts, "/")
		if len(parts) == 0 || (len(parts) == 1 && parts[0] == "") {
			parsed.Path = ""
		}
		return parsed.String()
	})
	for _, rule := range reviewGatewayErrorRules {
		value = rule.pattern.ReplaceAllString(value, rule.replacement)
	}
	for _, name := range []string{
		"GITLINK_TOKEN",
		"GITLINK_ACCESS_TOKEN",
		"FEISHU_APP_SECRET",
		"FEISHU_WEBHOOK_SECRET",
		"FEISHU_WEBHOOK_URL",
	} {
		if secret := strings.TrimSpace(os.Getenv(name)); len(secret) >= 6 {
			value = strings.ReplaceAll(value, secret, "***")
		}
	}
	value = strings.Join(strings.Fields(value), " ")
	runes := []rune(value)
	if len(runes) > 512 {
		value = string(runes[:512]) + "..."
	}
	return value
}
