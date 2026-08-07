package feishu

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

const (
	ReviewQueueGitLinkRead     = "gitlink_read"
	ReviewQueueCollaboration   = "collaboration"
	ReviewQueueControlledWrite = "controlled_write"
	ReviewQueueAgent           = "agent"
)

type ReviewQueueLimits struct {
	MaxJobs               int
	MaxOperations         int
	MaxJobsPerChat        int
	MaxJobsPerUser        int
	MaxOperationsPerChat  int
	UserRequestsPerMinute int
	ChatRequestsPerMinute int
	RefreshCoalesceWindow time.Duration
}

func DefaultReviewQueueLimits() ReviewQueueLimits {
	return ReviewQueueLimits{MaxJobs: 2000, MaxOperations: 5000, MaxJobsPerChat: 100, MaxJobsPerUser: 20, MaxOperationsPerChat: 200, UserRequestsPerMinute: 10, ChatRequestsPerMinute: 60, RefreshCoalesceWindow: 10 * time.Second}
}

func (l ReviewQueueLimits) normalized() ReviewQueueLimits {
	d := DefaultReviewQueueLimits()
	if l.MaxJobs < 1 || l.MaxJobs > 100000 {
		l.MaxJobs = d.MaxJobs
	}
	if l.MaxOperations < 1 || l.MaxOperations > 200000 {
		l.MaxOperations = d.MaxOperations
	}
	if l.MaxJobsPerChat < 1 || l.MaxJobsPerChat > 10000 {
		l.MaxJobsPerChat = d.MaxJobsPerChat
	}
	if l.MaxJobsPerUser < 1 || l.MaxJobsPerUser > 1000 {
		l.MaxJobsPerUser = d.MaxJobsPerUser
	}
	if l.MaxOperationsPerChat < 1 || l.MaxOperationsPerChat > 20000 {
		l.MaxOperationsPerChat = d.MaxOperationsPerChat
	}
	if l.UserRequestsPerMinute < 1 || l.UserRequestsPerMinute > 10000 {
		l.UserRequestsPerMinute = d.UserRequestsPerMinute
	}
	if l.ChatRequestsPerMinute < 1 || l.ChatRequestsPerMinute > 50000 {
		l.ChatRequestsPerMinute = d.ChatRequestsPerMinute
	}
	if l.RefreshCoalesceWindow < time.Second || l.RefreshCoalesceWindow > time.Minute {
		l.RefreshCoalesceWindow = d.RefreshCoalesceWindow
	}
	return l
}

type ReviewQueueAdmissionError struct {
	Code string
	Err  error
}

func (e *ReviewQueueAdmissionError) Error() string {
	if e.Err != nil {
		return e.Err.Error()
	}
	return e.Code
}
func (e *ReviewQueueAdmissionError) Unwrap() error { return e.Err }

type ReviewJobAdmissionResult struct {
	Created    bool   `json:"created"`
	Coalesced  bool   `json:"coalesced"`
	Duplicate  bool   `json:"duplicate"`
	JobID      string `json:"job_id"`
	ConsumerID string `json:"consumer_id,omitempty"`
}

type ReviewJobConsumer struct {
	ConsumerID       string `json:"consumer_id"`
	JobID            string `json:"job_id"`
	SourceType       string `json:"source_type"`
	ChatID           string `json:"-"`
	ChatIDHash       string `json:"chat_id_hash"`
	SourceMessageID  string `json:"source_message_id,omitempty"`
	RequestedByHash  string `json:"requested_by_hash,omitempty"`
	NotificationMode string `json:"notification_mode"`
	Status           string `json:"status"`
	CreatedAt        string `json:"created_at"`
	UpdatedAt        string `json:"updated_at"`
}

func reviewGatewayQueueClassForAction(action string) (string, error) {
	switch strings.TrimSpace(action) {
	case "read_review_context", "refresh_review_context", "read_review_queue", "generate_review_draft", "show_review_subscriptions":
		return ReviewQueueGitLinkRead, nil
	case "claim_review", "release_review", "set_review_deadline", "prepare_common_review", "show_local_review_plan", "cancel_common_review", "subscribe_review_events", "unsubscribe_review_events", "set_review_notification_mode", "set_default_review_repository", "help", "show_binding", "list_repositories", "read_my_review_tasks", "plan_bind_repository":
		return ReviewQueueCollaboration, nil
	case "confirm_common_review":
		return ReviewQueueControlledWrite, nil
	case "run_agent_review":
		return ReviewQueueAgent, nil
	default:
		return "", &ReviewQueueAdmissionError{Code: "unsupported_queue_action", Err: fmt.Errorf("unsupported queue action %q", action)}
	}
}
func reviewGatewayDefaultPriority(queueClass string) int {
	switch queueClass {
	case ReviewQueueCollaboration:
		return 20
	case ReviewQueueControlledWrite:
		return 10
	case ReviewQueueGitLinkRead:
		return 50
	case ReviewQueueAgent:
		return 90
	default:
		return 100
	}
}
func reviewRefreshCoalesceKey(job ReviewGatewayJob) string {
	if job.Action != "refresh_review_context" {
		return ""
	}
	return stableKey("review-refresh", job.InstallationID, job.ChatID, job.Repository, fmt.Sprintf("%d", job.PRNumber))
}

func (s *SQLiteReviewGatewayStore) AdmitReviewGatewayJob(ctx context.Context, job ReviewGatewayJob, sourceType string, now time.Time, limits ReviewQueueLimits) (ReviewJobAdmissionResult, error) {
	limits = limits.normalized()
	queueClass, err := reviewGatewayQueueClassForAction(job.Action)
	if err != nil {
		return ReviewJobAdmissionResult{}, err
	}
	job.QueueClass = queueClass
	if job.Priority <= 0 {
		job.Priority = reviewGatewayDefaultPriority(queueClass)
	}
	job.CoalesceKey = reviewRefreshCoalesceKey(job)
	if job.CoalesceKey != "" {
		job.CoalesceUntil = reviewGatewayTimestamp(now.Add(limits.RefreshCoalesceWindow))
	}
	if job.OperationPlanStatus == "" {
		job.OperationPlanStatus = "none"
	}
	if job.MaxAttempts <= 0 {
		job.MaxAttempts = reviewGatewayDefaultMaxAttempts
	}
	if job.NextAttemptAt == "" {
		job.NextAttemptAt = reviewGatewayTimestamp(now)
	}
	sourceType = normalizeReviewConsumerSourceType(sourceType, job)
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return ReviewJobAdmissionResult{}, err
	}
	defer func() { _ = tx.Rollback() }()
	nowText := reviewGatewayTimestamp(now)
	var existingReservation string
	reserveErr := tx.QueryRowContext(ctx, `SELECT dedupe_key FROM review_gateway_events WHERE dedupe_key=? AND expires_at>?`, job.DedupeKey, nowText).Scan(&existingReservation)
	if reserveErr == nil {
		var existingJob string
		jobErr := tx.QueryRowContext(ctx, `SELECT job_id FROM review_gateway_jobs WHERE dedupe_key=? ORDER BY created_at DESC LIMIT 1`, job.DedupeKey).Scan(&existingJob)
		if jobErr == nil {
			return ReviewJobAdmissionResult{Duplicate: true, JobID: existingJob}, nil
		}
		if !errors.Is(jobErr, sql.ErrNoRows) {
			return ReviewJobAdmissionResult{}, jobErr
		}
		// Recover reservations left by the historical two-transaction path.
		if _, err := tx.ExecContext(ctx, `DELETE FROM review_gateway_events WHERE dedupe_key=?`, job.DedupeKey); err != nil {
			return ReviewJobAdmissionResult{}, err
		}
		reserveErr = sql.ErrNoRows
	}
	if !errors.Is(reserveErr, sql.ErrNoRows) {
		return ReviewJobAdmissionResult{}, reserveErr
	}
	if err := checkReviewQueueCapacityTx(ctx, tx, job, limits); err != nil {
		return ReviewJobAdmissionResult{}, err
	}
	if sourceType == "user_command" {
		if err := consumeReviewRateLimitsTx(ctx, tx, job, now, limits); err != nil {
			return ReviewJobAdmissionResult{}, err
		}
	}
	if job.CoalesceKey != "" {
		var existingJobID string
		err := tx.QueryRowContext(ctx, `SELECT job_id FROM review_gateway_jobs WHERE coalesce_key=? AND status IN ('queued','running') AND coalesce_until>=? ORDER BY created_at LIMIT 1`, job.CoalesceKey, nowText).Scan(&existingJobID)
		if err == nil {
			consumerID, err := saveReviewJobConsumerTx(ctx, tx, existingJobID, job, sourceType, now)
			if err != nil {
				return ReviewJobAdmissionResult{}, err
			}
			if err := reserveReviewJobDedupeTx(ctx, tx, job.DedupeKey, now); err != nil {
				return ReviewJobAdmissionResult{}, err
			}
			if err := tx.Commit(); err != nil {
				return ReviewJobAdmissionResult{}, err
			}
			return ReviewJobAdmissionResult{Coalesced: true, JobID: existingJobID, ConsumerID: consumerID}, nil
		}
		if !errors.Is(err, sql.ErrNoRows) {
			return ReviewJobAdmissionResult{}, err
		}
	}
	payload, err := json.Marshal(job)
	if err != nil {
		return ReviewJobAdmissionResult{}, err
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO review_gateway_jobs(job_id,dedupe_key,status,action,repository,pr_number,requested_by,chat_id,payload_json,created_at,updated_at,error_summary,attempt_count,max_attempts,next_attempt_at,queue_class,priority,coalesce_key,coalesce_until,operation_plan_status)
		VALUES(?,?,'queued',?,?,?,?,?,?,?,?,'',0,?,?,?,?,?,?,?)`, job.JobID, job.DedupeKey, job.Action, job.Repository, job.PRNumber, job.RequestedBy, job.ChatID, string(payload), job.CreatedAt, nowText, job.MaxAttempts, job.NextAttemptAt, job.QueueClass, job.Priority, job.CoalesceKey, job.CoalesceUntil, job.OperationPlanStatus)
	if err != nil {
		return ReviewJobAdmissionResult{}, fmt.Errorf("save admitted Review job: %w", err)
	}
	consumerID, err := saveReviewJobConsumerTx(ctx, tx, job.JobID, job, sourceType, now)
	if err != nil {
		return ReviewJobAdmissionResult{}, err
	}
	if err := reserveReviewJobDedupeTx(ctx, tx, job.DedupeKey, now); err != nil {
		return ReviewJobAdmissionResult{}, err
	}
	if err := tx.Commit(); err != nil {
		return ReviewJobAdmissionResult{}, err
	}
	return ReviewJobAdmissionResult{Created: true, JobID: job.JobID, ConsumerID: consumerID}, nil
}

func reserveReviewJobDedupeTx(ctx context.Context, tx *sql.Tx, key string, now time.Time) error {
	result, err := tx.ExecContext(ctx, `INSERT OR IGNORE INTO review_gateway_events(dedupe_key,received_at,expires_at)VALUES(?,?,?)`, key, reviewGatewayTimestamp(now), reviewGatewayTimestamp(now.Add(24*time.Hour)))
	if err != nil {
		return err
	}
	affected, _ := result.RowsAffected()
	if affected != 1 {
		return fmt.Errorf("review job dedupe reservation changed concurrently")
	}
	return nil
}

func checkReviewQueueCapacityTx(ctx context.Context, tx *sql.Tx, job ReviewGatewayJob, l ReviewQueueLimits) error {
	checks := []struct {
		query string
		args  []interface{}
		limit int
		code  string
	}{{`SELECT COUNT(*) FROM review_gateway_jobs WHERE status IN ('queued','running')`, nil, l.MaxJobs, "global_job_capacity"}, {`SELECT COUNT(*) FROM review_gateway_jobs WHERE chat_id=? AND status IN ('queued','running')`, []interface{}{job.ChatID}, l.MaxJobsPerChat, "chat_job_capacity"}, {`SELECT COUNT(*) FROM review_gateway_jobs WHERE requested_by=? AND status IN ('queued','running')`, []interface{}{job.RequestedBy}, l.MaxJobsPerUser, "user_job_capacity"}, {`SELECT COUNT(*) FROM review_operations WHERE status IN ('pending','leased','writing','retry_scheduled','blocked')`, nil, l.MaxOperations, "global_operation_capacity"}, {`SELECT COUNT(*) FROM review_operations WHERE chat_id=? AND status IN ('pending','leased','writing','retry_scheduled','blocked')`, []interface{}{job.ChatID}, l.MaxOperationsPerChat, "chat_operation_capacity"}}
	for _, check := range checks {
		var count int
		if err := tx.QueryRowContext(ctx, check.query, check.args...).Scan(&count); err != nil {
			return err
		}
		if count >= check.limit {
			return &ReviewQueueAdmissionError{Code: check.code, Err: fmt.Errorf("Review queue capacity exceeded: %s", check.code)}
		}
	}
	return nil
}

func consumeReviewRateLimitsTx(ctx context.Context, tx *sql.Tx, job ReviewGatewayJob, now time.Time, l ReviewQueueLimits) error {
	window := now.UTC().Truncate(time.Minute)
	for _, item := range []struct {
		kind, subject string
		limit         int
	}{{"user", job.RequestedBy, l.UserRequestsPerMinute}, {"chat", job.ChatID, l.ChatRequestsPerMinute}} {
		hash := reviewGatewayHashIdentifier(item.subject)
		key := stableKey("review-rate", item.kind, hash, reviewGatewayTimestamp(window))
		var count int
		err := tx.QueryRowContext(ctx, `SELECT request_count FROM review_rate_limit_buckets WHERE bucket_key=?`, key).Scan(&count)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return err
		}
		if count >= item.limit {
			return &ReviewQueueAdmissionError{Code: item.kind + "_rate_limited", Err: fmt.Errorf("Review %s rate limit exceeded", item.kind)}
		}
		_, err = tx.ExecContext(ctx, `INSERT INTO review_rate_limit_buckets(bucket_key,bucket_type,subject_hash,window_started_at,request_count,updated_at)VALUES(?,?,?,?,1,?) ON CONFLICT(bucket_key)DO UPDATE SET request_count=request_count+1,updated_at=excluded.updated_at`, key, item.kind, hash, reviewGatewayTimestamp(window), reviewGatewayTimestamp(now))
		if err != nil {
			return err
		}
	}
	return nil
}

func saveReviewJobConsumerTx(ctx context.Context, tx *sql.Tx, jobID string, job ReviewGatewayJob, sourceType string, now time.Time) (string, error) {
	sourceIdentity := firstNonEmpty(job.SourceMessageID, job.SourceEventID, job.DedupeKey)
	consumerID := stableKey("review-job-consumer", jobID, sourceType, job.ChatID, sourceIdentity, job.NotificationMode)
	notification := firstNonEmpty(job.NotificationMode, ReviewNotificationCanonicalOnly)
	_, err := tx.ExecContext(ctx, `INSERT OR IGNORE INTO review_job_consumers(consumer_id,job_id,source_type,chat_id,source_message_id,requested_by_hash,notification_mode,status,created_at,updated_at)VALUES(?,?,?,?,?,?,?,'pending',?,?)`, consumerID, jobID, sourceType, job.ChatID, job.SourceMessageID, reviewGatewayHashIdentifier(job.RequestedBy), notification, reviewGatewayTimestamp(now), reviewGatewayTimestamp(now))
	return consumerID, err
}

func normalizeReviewConsumerSourceType(sourceType string, job ReviewGatewayJob) string {
	switch sourceType {
	case "user_command", "event_route", "reconciliation", "replay":
		return sourceType
	}
	if job.SourceMessageID != "" {
		return "user_command"
	}
	if strings.HasPrefix(job.SourceEventID, "reconcile") || strings.HasPrefix(job.DedupeKey, "review-reconciliation") {
		return "reconciliation"
	}
	return "event_route"
}

func (s *SQLiteReviewGatewayStore) ListReviewJobConsumers(ctx context.Context, jobID string) ([]ReviewJobConsumer, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT consumer_id,job_id,source_type,chat_id,source_message_id,requested_by_hash,notification_mode,status,created_at,updated_at FROM review_job_consumers WHERE job_id=? ORDER BY created_at,consumer_id`, jobID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []ReviewJobConsumer{}
	for rows.Next() {
		var item ReviewJobConsumer
		if err := rows.Scan(&item.ConsumerID, &item.JobID, &item.SourceType, &item.ChatID, &item.SourceMessageID, &item.RequestedByHash, &item.NotificationMode, &item.Status, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, err
		}
		item.ChatIDHash = reviewGatewayHashIdentifier(item.ChatID)
		result = append(result, item)
	}
	return result, rows.Err()
}
