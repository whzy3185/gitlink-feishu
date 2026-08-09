package feishu

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

const (
	reviewRepositoryEventSchema = "gitlink.review-event/v1"
	reviewOperationSchema       = "feishu.review-operation/v1"
)

type ReviewRepositoryEvent struct {
	SchemaVersion string                 `json:"schema_version"`
	DeliveryID    string                 `json:"delivery_id"`
	EventType     string                 `json:"event_type"`
	Repository    string                 `json:"repository"`
	PRNumber      int                    `json:"pr_number"`
	HeadSHA       string                 `json:"head_sha,omitempty"`
	OccurredAt    string                 `json:"occurred_at,omitempty"`
	Payload       map[string]interface{} `json:"payload,omitempty"`
}

type ReviewEventInboxItem struct {
	Event          ReviewRepositoryEvent `json:"event"`
	Status         string                `json:"status"`
	AttemptCount   int                   `json:"attempt_count"`
	NextAttemptAt  string                `json:"next_attempt_at,omitempty"`
	LeaseOwner     string                `json:"lease_owner,omitempty"`
	LeaseExpiresAt string                `json:"lease_expires_at,omitempty"`
	ErrorSummary   string                `json:"error_summary,omitempty"`
}

type ReviewOperation struct {
	SchemaVersion       string          `json:"schema_version"`
	OperationID         string          `json:"operation_id"`
	DedupeKey           string          `json:"dedupe_key"`
	Kind                string          `json:"kind"`
	Status              string          `json:"status"`
	Payload             json.RawMessage `json:"payload"`
	AttemptCount        int             `json:"attempt_count"`
	MaxAttempts         int             `json:"max_attempts"`
	NextAttemptAt       string          `json:"next_attempt_at,omitempty"`
	LeaseOwner          string          `json:"lease_owner,omitempty"`
	LeaseExpiresAt      string          `json:"lease_expires_at,omitempty"`
	MutationStatus      string          `json:"mutation_status"`
	ReconciliationState string          `json:"reconciliation_state"`
	LastError           string          `json:"last_error,omitempty"`
}

type ReviewOperationError struct {
	Kind                 string
	Retryable            bool
	RetryAfter           time.Duration
	RemoteMayHaveApplied bool
	Err                  error
}

func (e *ReviewOperationError) Error() string {
	if e == nil || e.Err == nil {
		return "review operation failed"
	}
	return e.Err.Error()
}

func (e *ReviewOperationError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

func NormalizeGitLinkReviewEvent(deliveryID, eventType string, payload []byte) (ReviewRepositoryEvent, error) {
	deliveryID = strings.TrimSpace(deliveryID)
	eventType = strings.ToLower(strings.TrimSpace(eventType))
	if deliveryID == "" || eventType == "" {
		return ReviewRepositoryEvent{}, fmt.Errorf("review event requires delivery ID and event type")
	}
	var body map[string]interface{}
	if err := json.Unmarshal(payload, &body); err != nil {
		return ReviewRepositoryEvent{}, fmt.Errorf("decode GitLink review event: %w", err)
	}
	repository := firstReviewDataString(body, "repository", "repo", "project_path")
	if nested, ok := body["repository"].(map[string]interface{}); ok {
		repository = firstReviewDataValue(repository, firstReviewDataString(nested, "full_name", "path_with_namespace", "name"))
	}
	pull := reviewDataObject(firstReviewDataValueInterface(body, "pull_request", "pull", "object_attributes"))
	if pull == nil {
		pull = body
	}
	number := firstReviewDataInt(pull, "number", "iid", "pull_request_id")
	if !reviewGatewayRepositoryPattern.MatchString(repository) || number <= 0 {
		return ReviewRepositoryEvent{}, fmt.Errorf("review event requires owner/repo and a positive PR number")
	}
	event := ReviewRepositoryEvent{
		SchemaVersion: reviewRepositoryEventSchema, DeliveryID: deliveryID, EventType: eventType,
		Repository: repository, PRNumber: number,
		HeadSHA:    firstReviewDataString(pull, "head_sha", "head_commit_sha", "last_commit_id"),
		OccurredAt: firstReviewDataString(body, "occurred_at", "updated_at", "created_at"),
		Payload:    body,
	}
	return event, nil
}

func (s *SQLiteReviewGatewayStore) EnqueueReviewEvent(ctx context.Context, event ReviewRepositoryEvent, now time.Time) (bool, error) {
	if event.SchemaVersion != reviewRepositoryEventSchema || strings.TrimSpace(event.DeliveryID) == "" {
		return false, fmt.Errorf("invalid normalized review event")
	}
	payload, err := json.Marshal(event)
	if err != nil {
		return false, err
	}
	result, err := s.db.ExecContext(ctx, `INSERT OR IGNORE INTO review_event_inbox (
		delivery_id, event_type, repository, pr_number, head_sha, payload_json,
		status, attempt_count, next_attempt_at, created_at, updated_at
	) VALUES (?, ?, ?, ?, ?, ?, 'queued', 0, ?, ?, ?)`,
		event.DeliveryID, event.EventType, event.Repository, event.PRNumber, event.HeadSHA, string(payload),
		reviewGatewayTimestamp(now), reviewGatewayTimestamp(now), reviewGatewayTimestamp(now))
	if err != nil {
		return false, err
	}
	rows, err := result.RowsAffected()
	return rows == 1, err
}

func (s *SQLiteReviewGatewayStore) ClaimReviewEvents(ctx context.Context, owner string, now, leaseUntil time.Time, limit int) ([]ReviewEventInboxItem, error) {
	if strings.TrimSpace(owner) == "" {
		return nil, fmt.Errorf("review event lease owner is required")
	}
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	rows, err := tx.QueryContext(ctx, `SELECT delivery_id, payload_json, attempt_count
		FROM review_event_inbox
		WHERE (status = 'queued' OR (status = 'running' AND lease_expires_at <= ?))
		AND next_attempt_at <= ? ORDER BY created_at, delivery_id LIMIT ?`,
		reviewGatewayTimestamp(now), reviewGatewayTimestamp(now), limit)
	if err != nil {
		return nil, err
	}
	type candidate struct {
		delivery, payload string
		attempts          int
	}
	candidates := []candidate{}
	for rows.Next() {
		var item candidate
		if err := rows.Scan(&item.delivery, &item.payload, &item.attempts); err != nil {
			rows.Close()
			return nil, err
		}
		candidates = append(candidates, item)
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	result := make([]ReviewEventInboxItem, 0, len(candidates))
	for _, candidate := range candidates {
		updated, err := tx.ExecContext(ctx, `UPDATE review_event_inbox SET status='running',
			attempt_count=attempt_count+1, lease_owner=?, lease_expires_at=?, updated_at=?
			WHERE delivery_id=? AND (status='queued' OR lease_expires_at <= ?)`,
			owner, reviewGatewayTimestamp(leaseUntil), reviewGatewayTimestamp(now), candidate.delivery, reviewGatewayTimestamp(now))
		if err != nil {
			return nil, err
		}
		count, _ := updated.RowsAffected()
		if count != 1 {
			continue
		}
		var event ReviewRepositoryEvent
		if err := json.Unmarshal([]byte(candidate.payload), &event); err != nil {
			return nil, err
		}
		result = append(result, ReviewEventInboxItem{Event: event, Status: "running", AttemptCount: candidate.attempts + 1, LeaseOwner: owner, LeaseExpiresAt: reviewGatewayTimestamp(leaseUntil)})
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return result, nil
}

func (s *SQLiteReviewGatewayStore) CompleteReviewEvent(ctx context.Context, deliveryID, owner string, now time.Time) error {
	result, err := s.db.ExecContext(ctx, `UPDATE review_event_inbox SET status='completed',
		lease_owner='', lease_expires_at='', updated_at=? WHERE delivery_id=? AND status='running' AND lease_owner=?`,
		reviewGatewayTimestamp(now), deliveryID, owner)
	return requireReviewGatewayJobUpdate(result, err, deliveryID, "complete review event")
}

func (s *SQLiteReviewGatewayStore) FailReviewEvent(ctx context.Context, item ReviewEventInboxItem, owner string, failure error, now time.Time, maxAttempts int) (string, error) {
	if maxAttempts <= 0 {
		maxAttempts = 5
	}
	status := "queued"
	next := now.Add(reviewGatewayRetryDelay(item.AttemptCount))
	if item.AttemptCount >= maxAttempts {
		status = "dead_letter"
		next = time.Time{}
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return "", err
	}
	defer tx.Rollback()
	result, err := tx.ExecContext(ctx, `UPDATE review_event_inbox SET status=?, next_attempt_at=?,
		lease_owner='', lease_expires_at='', error_summary=?, updated_at=?
		WHERE delivery_id=? AND status='running' AND lease_owner=?`, status, reviewGatewayTimestamp(next),
		redactReviewGatewayError(failure.Error()), reviewGatewayTimestamp(now), item.Event.DeliveryID, owner)
	if err := requireReviewGatewayJobUpdate(result, err, item.Event.DeliveryID, "fail review event"); err != nil {
		return "", err
	}
	if status == "dead_letter" {
		payload, _ := json.Marshal(item.Event)
		_, err = tx.ExecContext(ctx, `INSERT OR REPLACE INTO review_dead_letters (
			dead_letter_id, source_type, source_id, payload_json, error_summary, created_at
		) VALUES (?, 'review_event', ?, ?, ?, ?)`,
			stableReviewReliabilityID("dead", item.Event.DeliveryID), item.Event.DeliveryID, string(payload),
			redactReviewGatewayError(failure.Error()), reviewGatewayTimestamp(now))
		if err != nil {
			return "", err
		}
	}
	return status, tx.Commit()
}

func (s *SQLiteReviewGatewayStore) ReplayReviewEvent(ctx context.Context, deliveryID string, now time.Time) error {
	result, err := s.db.ExecContext(ctx, `UPDATE review_event_inbox SET status='queued',
		next_attempt_at=?, lease_owner='', lease_expires_at='', error_summary='', updated_at=?
		WHERE delivery_id=? AND status IN ('failed','dead_letter','completed')`,
		reviewGatewayTimestamp(now), reviewGatewayTimestamp(now), deliveryID)
	return requireReviewGatewayJobUpdate(result, err, deliveryID, "replay review event")
}

func NewReviewOperation(kind, dedupeKey string, payload interface{}, maxAttempts int, now time.Time) (ReviewOperation, error) {
	kind, dedupeKey = strings.TrimSpace(kind), strings.TrimSpace(dedupeKey)
	if kind == "" || dedupeKey == "" {
		return ReviewOperation{}, fmt.Errorf("review operation requires kind and dedupe key")
	}
	encoded, err := json.Marshal(payload)
	if err != nil {
		return ReviewOperation{}, err
	}
	if maxAttempts <= 0 {
		maxAttempts = 5
	}
	return ReviewOperation{
		SchemaVersion: reviewOperationSchema, OperationID: stableReviewReliabilityID("operation", kind, dedupeKey),
		DedupeKey: dedupeKey, Kind: kind, Status: "queued", Payload: encoded,
		MaxAttempts: maxAttempts, NextAttemptAt: reviewGatewayTimestamp(now), MutationStatus: "not_started", ReconciliationState: "not_required",
	}, nil
}

func (s *SQLiteReviewGatewayStore) EnqueueReviewOperation(ctx context.Context, operation ReviewOperation, now time.Time) (bool, error) {
	result, err := s.db.ExecContext(ctx, `INSERT OR IGNORE INTO review_operations (
		operation_id, dedupe_key, kind, status, payload_json, attempt_count, max_attempts,
		next_attempt_at, mutation_status, reconciliation_state, created_at, updated_at
	) VALUES (?, ?, ?, 'queued', ?, 0, ?, ?, 'not_started', 'not_required', ?, ?)`,
		operation.OperationID, operation.DedupeKey, operation.Kind, string(operation.Payload), operation.MaxAttempts,
		operation.NextAttemptAt, reviewGatewayTimestamp(now), reviewGatewayTimestamp(now))
	if err != nil {
		return false, err
	}
	rows, err := result.RowsAffected()
	return rows == 1, err
}

func (s *SQLiteReviewGatewayStore) ClaimReviewOperations(ctx context.Context, owner string, now, leaseUntil time.Time, limit int) ([]ReviewOperation, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	rows, err := tx.QueryContext(ctx, `SELECT operation_id FROM review_operations
		WHERE (status='queued' OR (status='running' AND lease_expires_at <= ?))
		AND next_attempt_at <= ? ORDER BY created_at, operation_id LIMIT ?`, reviewGatewayTimestamp(now), reviewGatewayTimestamp(now), limit)
	if err != nil {
		return nil, err
	}
	ids := []string{}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return nil, err
		}
		ids = append(ids, id)
	}
	rows.Close()
	result := []ReviewOperation{}
	for _, id := range ids {
		updated, err := tx.ExecContext(ctx, `UPDATE review_operations SET status='running', attempt_count=attempt_count+1,
			lease_owner=?, lease_expires_at=?, updated_at=? WHERE operation_id=? AND (status='queued' OR lease_expires_at <= ?)`,
			owner, reviewGatewayTimestamp(leaseUntil), reviewGatewayTimestamp(now), id, reviewGatewayTimestamp(now))
		if err != nil {
			return nil, err
		}
		count, _ := updated.RowsAffected()
		if count != 1 {
			continue
		}
		operation, err := scanReviewOperation(tx.QueryRowContext(ctx, `SELECT operation_id, dedupe_key, kind, status,
			payload_json, attempt_count, max_attempts, next_attempt_at, lease_owner, lease_expires_at,
			mutation_status, reconciliation_state, last_error FROM review_operations WHERE operation_id=?`, id))
		if err != nil {
			return nil, err
		}
		result = append(result, operation)
	}
	return result, tx.Commit()
}

func (s *SQLiteReviewGatewayStore) CompleteReviewOperation(ctx context.Context, operation ReviewOperation, owner string, now time.Time) error {
	result, err := s.db.ExecContext(ctx, `UPDATE review_operations SET status='completed', mutation_status='confirmed',
		reconciliation_state='verified', lease_owner='', lease_expires_at='', updated_at=?
		WHERE operation_id=? AND status='running' AND lease_owner=?`, reviewGatewayTimestamp(now), operation.OperationID, owner)
	return requireReviewGatewayJobUpdate(result, err, operation.OperationID, "complete review operation")
}

func (s *SQLiteReviewGatewayStore) FailReviewOperation(ctx context.Context, operation ReviewOperation, owner string, failure error, now time.Time) (string, error) {
	var typed *ReviewOperationError
	errors.As(failure, &typed)
	if typed != nil && typed.RemoteMayHaveApplied {
		result, err := s.db.ExecContext(ctx, `UPDATE review_operations SET status='unknown', mutation_status='possible',
			reconciliation_state='required', lease_owner='', lease_expires_at='', last_error=?, updated_at=?
			WHERE operation_id=? AND status='running' AND lease_owner=?`, redactReviewGatewayError(failure.Error()),
			reviewGatewayTimestamp(now), operation.OperationID, owner)
		return "unknown", requireReviewGatewayJobUpdate(result, err, operation.OperationID, "mark review operation unknown")
	}
	retryable := typed != nil && typed.Retryable && operation.AttemptCount < operation.MaxAttempts
	status := "failed"
	next := time.Time{}
	if retryable {
		status = "queued"
		delay := typed.RetryAfter
		if delay <= 0 {
			delay = reviewGatewayRetryDelay(operation.AttemptCount)
		}
		next = now.Add(delay)
	}
	result, err := s.db.ExecContext(ctx, `UPDATE review_operations SET status=?, next_attempt_at=?,
		lease_owner='', lease_expires_at='', last_error=?, updated_at=?
		WHERE operation_id=? AND status='running' AND lease_owner=?`, status, reviewGatewayTimestamp(next),
		redactReviewGatewayError(failure.Error()), reviewGatewayTimestamp(now), operation.OperationID, owner)
	return status, requireReviewGatewayJobUpdate(result, err, operation.OperationID, "fail review operation")
}

func (s *SQLiteReviewGatewayStore) MarkReviewOperationReconciled(ctx context.Context, operationID, outcome string, now time.Time) error {
	if outcome != "verified" && outcome != "not_applied" {
		return fmt.Errorf("review reconciliation outcome must be verified or not_applied")
	}
	status, mutation := "completed", "confirmed"
	if outcome == "not_applied" {
		status, mutation = "failed", "not_applied"
	}
	result, err := s.db.ExecContext(ctx, `UPDATE review_operations SET status=?, mutation_status=?,
		reconciliation_state=?, updated_at=? WHERE operation_id=? AND status='unknown'`,
		status, mutation, outcome, reviewGatewayTimestamp(now), operationID)
	return requireReviewGatewayJobUpdate(result, err, operationID, "reconcile review operation")
}

func scanReviewOperation(scanner interface{ Scan(...interface{}) error }) (ReviewOperation, error) {
	operation := ReviewOperation{SchemaVersion: reviewOperationSchema}
	var payload string
	err := scanner.Scan(&operation.OperationID, &operation.DedupeKey, &operation.Kind, &operation.Status,
		&payload, &operation.AttemptCount, &operation.MaxAttempts, &operation.NextAttemptAt,
		&operation.LeaseOwner, &operation.LeaseExpiresAt, &operation.MutationStatus,
		&operation.ReconciliationState, &operation.LastError)
	operation.Payload = json.RawMessage(payload)
	return operation, err
}

func stableReviewReliabilityID(prefix string, parts ...string) string {
	digest := sha256.Sum256([]byte(strings.Join(parts, "\x00")))
	return prefix + "-" + hex.EncodeToString(digest[:8])
}
