package feishu

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	_ "modernc.org/sqlite"
)

const reviewGatewayStateSchema = `
CREATE TABLE IF NOT EXISTS review_gateway_events (
    dedupe_key TEXT PRIMARY KEY,
    received_at TEXT NOT NULL,
    expires_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS review_gateway_events_expires_at
    ON review_gateway_events(expires_at);

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
    error_summary TEXT
);
CREATE INDEX IF NOT EXISTS review_gateway_jobs_status
    ON review_gateway_jobs(status);
`

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

type ReviewGatewayJobStore interface {
	SaveJob(ctx context.Context, job ReviewGatewayJob) error
	UpdateJobStatus(ctx context.Context, jobID, status, errorSummary string) error
}

type ReviewGatewayStateStore interface {
	ReviewGatewayDeduper
	ReviewGatewayJobStore
	Close() error
}

type MemoryReviewGatewayJobStore struct {
	mu     sync.Mutex
	Jobs   map[string]ReviewGatewayJob
	Status map[string]string
	Errors map[string]string
}

type SQLiteReviewGatewayStore struct {
	db *sql.DB
}

type ReviewGatewayQueue struct {
	gateway  *ReviewGateway
	store    ReviewGatewayJobStore
	jobs     chan ReviewGatewayJob
	onResult func(ReviewGatewayJob, error)
}

func NewMemoryReviewGatewayJobStore() *MemoryReviewGatewayJobStore {
	return &MemoryReviewGatewayJobStore{
		Jobs:   map[string]ReviewGatewayJob{},
		Status: map[string]string{},
		Errors: map[string]string{},
	}
}

func (s *MemoryReviewGatewayJobStore) SaveJob(_ context.Context, job ReviewGatewayJob) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Jobs[job.JobID] = job
	s.Status[job.JobID] = job.Status
	return nil
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
	for _, statement := range []string{
		"PRAGMA journal_mode=WAL",
		"PRAGMA busy_timeout=5000",
		reviewGatewayStateSchema,
	} {
		if _, err := db.Exec(statement); err != nil {
			_ = db.Close()
			return nil, fmt.Errorf("initialize review gateway state database: %w", err)
		}
	}
	return &SQLiteReviewGatewayStore{db: db}, nil
}

func (s *SQLiteReviewGatewayStore) Reserve(key string, now time.Time, ttl time.Duration) (bool, error) {
	if s == nil || s.db == nil || strings.TrimSpace(key) == "" {
		return false, fmt.Errorf("review gateway state store and dedupe key are required")
	}
	if ttl <= 0 {
		ttl = 24 * time.Hour
	}
	tx, err := s.db.Begin()
	if err != nil {
		return false, fmt.Errorf("begin review gateway dedupe transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	_, _ = tx.Exec("DELETE FROM review_gateway_events WHERE expires_at <= ?", now.UTC().Format(time.RFC3339Nano))
	result, err := tx.Exec(
		"INSERT OR IGNORE INTO review_gateway_events(dedupe_key, received_at, expires_at) VALUES(?, ?, ?)",
		key,
		now.UTC().Format(time.RFC3339Nano),
		now.Add(ttl).UTC().Format(time.RFC3339Nano),
	)
	if err != nil {
		return false, fmt.Errorf("reserve review gateway event: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil || affected != 1 {
		if err != nil {
			return false, fmt.Errorf("read review gateway event reservation: %w", err)
		}
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
	_, _ = s.db.Exec("DELETE FROM review_gateway_events WHERE dedupe_key = ?", key)
}

func (s *SQLiteReviewGatewayStore) SaveJob(ctx context.Context, job ReviewGatewayJob) error {
	payload, err := json.Marshal(job)
	if err != nil {
		return fmt.Errorf("encode review gateway job: %w", err)
	}
	now := time.Now().UTC().Format(time.RFC3339Nano)
	_, err = s.db.ExecContext(
		ctx,
		`INSERT INTO review_gateway_jobs(
			job_id, dedupe_key, status, action, repository, pr_number,
			requested_by, chat_id, payload_json, created_at, updated_at, error_summary
		) VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, '')
		ON CONFLICT(job_id) DO UPDATE SET
			status=excluded.status,
			payload_json=excluded.payload_json,
			updated_at=excluded.updated_at,
			error_summary=''`,
		job.JobID,
		job.DedupeKey,
		job.Status,
		job.Action,
		job.Repository,
		job.PRNumber,
		job.RequestedBy,
		job.ChatID,
		string(payload),
		job.CreatedAt,
		now,
	)
	if err != nil {
		return fmt.Errorf("save review gateway job: %w", err)
	}
	return nil
}

func (s *SQLiteReviewGatewayStore) UpdateJobStatus(ctx context.Context, jobID, status, errorSummary string) error {
	result, err := s.db.ExecContext(
		ctx,
		"UPDATE review_gateway_jobs SET status = ?, updated_at = ?, error_summary = ? WHERE job_id = ?",
		status,
		time.Now().UTC().Format(time.RFC3339Nano),
		redactReviewGatewayError(errorSummary),
		jobID,
	)
	if err != nil {
		return fmt.Errorf("update review gateway job: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("read review gateway job update result: %w", err)
	}
	if affected != 1 {
		return fmt.Errorf("review gateway job %q not found", jobID)
	}
	return nil
}

func (s *SQLiteReviewGatewayStore) Close() error {
	if s == nil || s.db == nil {
		return nil
	}
	return s.db.Close()
}

func NewReviewGatewayQueue(gateway *ReviewGateway, store ReviewGatewayJobStore, capacity int, onResult func(ReviewGatewayJob, error)) *ReviewGatewayQueue {
	if capacity <= 0 {
		capacity = 128
	}
	if store == nil {
		store = NewMemoryReviewGatewayJobStore()
	}
	return &ReviewGatewayQueue{
		gateway:  gateway,
		store:    store,
		jobs:     make(chan ReviewGatewayJob, capacity),
		onResult: onResult,
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
	receipt, err := q.gateway.Plan(event)
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
	select {
	case q.jobs <- job:
		return receipt
	default:
		q.gateway.Release(receipt)
		_ = q.store.UpdateJobStatus(ctx, job.JobID, "failed", "review gateway queue is full")
		receipt.Accepted = false
		receipt.Reason = "queue_full"
		receipt.Job.Status = "failed"
		return receipt
	}
}

func (q *ReviewGatewayQueue) Run(ctx context.Context, handler func(context.Context, ReviewGatewayJob) error) {
	for {
		select {
		case <-ctx.Done():
			return
		case job := <-q.jobs:
			_ = q.store.UpdateJobStatus(ctx, job.JobID, "running", "")
			var err error
			if handler == nil {
				err = fmt.Errorf("review gateway job handler is not configured")
			} else {
				err = handler(ctx, job)
			}
			if err != nil {
				_ = q.store.UpdateJobStatus(ctx, job.JobID, "failed", err.Error())
			} else {
				_ = q.store.UpdateJobStatus(ctx, job.JobID, "completed", "")
			}
			if q.onResult != nil {
				q.onResult(job, err)
			}
		}
	}
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
