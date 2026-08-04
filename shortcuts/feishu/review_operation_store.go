package feishu

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"
)

var ErrReviewOperationNotFound = errors.New("review operation not found")

type ReviewOperationClaimOptions struct {
	QueueClass    string
	LeaseOwner    string
	Now           time.Time
	LeaseDuration time.Duration
	Limit         int
}

func (s *SQLiteReviewGatewayStore) SaveReviewOperations(ctx context.Context, operations []ReviewOperation) error {
	if s == nil || s.db == nil {
		return fmt.Errorf("review operation store is required")
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin review operation transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	if err := saveReviewOperationsTx(ctx, tx, operations); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit review operation transaction: %w", err)
	}
	return nil
}

func saveReviewOperationsTx(ctx context.Context, tx *sql.Tx, operations []ReviewOperation) error {
	for _, operation := range operations {
		if err := operation.Validate(); err != nil {
			return err
		}
		// A newer desired state supersedes only work that has not started. An
		// in-flight or uncertain write remains independently auditable.
		if operation.WorkItemKey != "" {
			if _, err := tx.ExecContext(ctx, `UPDATE review_operations
				SET status='stale', completed_at=?, updated_at=?
				WHERE work_item_key=? AND operation_kind=?
				  AND desired_fingerprint<>?
				  AND status IN ('pending','retry_scheduled','blocked')`,
				operation.CreatedAt,
				operation.CreatedAt,
				operation.WorkItemKey,
				operation.OperationKind,
				operation.DesiredFingerprint,
			); err != nil {
				return fmt.Errorf("stale superseded review operations: %w", err)
			}
		}
		result, err := tx.ExecContext(ctx, `INSERT INTO review_operations (
			operation_id, operation_kind, queue_class,
			installation_id, chat_id, repository, pr_number,
			source_job_id, source_event_id, consumer_id,
			work_item_key, resource_type,
			idempotency_key, desired_fingerprint, applied_fingerprint,
			expected_remote_id, remote_id, desired_json,
			dependency_operation_id, dependency_policy, retry_safety, status,
			error_class, error_code, error_summary, http_status,
			mutation_status, requires_reconciliation,
			attempt_count, max_attempts, next_attempt_at, retry_after_at,
			lease_owner, lease_expires_at, created_at, updated_at, completed_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?,
			'', '', '', 0, ?, 0, 0, ?, ?, '', '', '', ?, ?, '')
		ON CONFLICT(idempotency_key) DO NOTHING`,
			operation.OperationID,
			operation.OperationKind,
			operation.QueueClass,
			operation.InstallationID,
			operation.ChatID,
			operation.Repository,
			operation.PRNumber,
			operation.SourceJobID,
			operation.SourceEventID,
			operation.ConsumerID,
			operation.WorkItemKey,
			operation.ResourceType,
			operation.IdempotencyKey,
			operation.DesiredFingerprint,
			operation.AppliedFingerprint,
			operation.ExpectedRemoteID,
			operation.RemoteID,
			operation.DesiredJSON,
			operation.DependencyOperationID,
			operation.DependencyPolicy,
			operation.RetrySafety,
			operation.Status,
			operation.MutationStatus,
			operation.MaxAttempts,
			operation.NextAttemptAt,
			operation.CreatedAt,
			operation.UpdatedAt,
		)
		if err != nil {
			return fmt.Errorf("save review operation %s: %w", operation.OperationID, err)
		}
		inserted, _ := result.RowsAffected()
		if inserted == 1 {
			createdAt, _ := time.Parse(time.RFC3339Nano, operation.CreatedAt)
			if createdAt.IsZero() {
				createdAt = time.Now().UTC()
			}
			if err := updateReviewProjectionStatusTx(ctx, tx, operation, ReviewProjectionPending, "", "", "", false, createdAt); err != nil {
				return fmt.Errorf("save review operation projection: %w", err)
			}
		}
	}
	return nil
}

func (s *SQLiteReviewGatewayStore) GetReviewOperation(ctx context.Context, operationID string) (ReviewOperation, error) {
	if s == nil || s.db == nil {
		return ReviewOperation{}, fmt.Errorf("review operation store is required")
	}
	return scanReviewOperation(s.db.QueryRowContext(ctx, reviewOperationSelect+` WHERE operation_id=?`, strings.TrimSpace(operationID)))
}

func (s *SQLiteReviewGatewayStore) ListReviewOperations(ctx context.Context, queueClass, status string, limit int) ([]ReviewOperation, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	query := reviewOperationSelect + ` WHERE 1=1`
	args := []interface{}{}
	if strings.TrimSpace(queueClass) != "" {
		query += ` AND queue_class=?`
		args = append(args, strings.TrimSpace(queueClass))
	}
	if strings.TrimSpace(status) != "" {
		query += ` AND status=?`
		args = append(args, strings.TrimSpace(status))
	}
	query += ` ORDER BY created_at, operation_id LIMIT ?`
	args = append(args, limit)
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list review operations: %w", err)
	}
	defer rows.Close()
	result := []ReviewOperation{}
	for rows.Next() {
		operation, scanErr := scanReviewOperation(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		result = append(result, operation)
	}
	return result, rows.Err()
}

func (s *SQLiteReviewGatewayStore) ClaimReviewOperations(ctx context.Context, opts ReviewOperationClaimOptions) ([]ReviewOperation, error) {
	if strings.TrimSpace(opts.QueueClass) == "" {
		return nil, fmt.Errorf("review operation queue class is required")
	}
	if strings.TrimSpace(opts.LeaseOwner) == "" {
		return nil, fmt.Errorf("review operation lease owner is required")
	}
	if opts.Now.IsZero() {
		opts.Now = time.Now().UTC()
	}
	if opts.LeaseDuration <= 0 {
		opts.LeaseDuration = 2 * time.Minute
	}
	if opts.Limit <= 0 || opts.Limit > 50 {
		opts.Limit = 1
	}
	nowText := reviewGatewayTimestamp(opts.Now)
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin review operation claim: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	// Expired writes are never blindly reset here. Only work that had not
	// started can be safely recovered by the normal claimer.
	if _, err := tx.ExecContext(ctx, `UPDATE review_operations
		SET status='retry_scheduled', lease_owner='', lease_expires_at='', updated_at=?
		WHERE queue_class=? AND status='leased' AND mutation_status='not_started'
		  AND lease_expires_at<>'' AND lease_expires_at<=?`, nowText, opts.QueueClass, nowText); err != nil {
		return nil, fmt.Errorf("recover expired review operation leases: %w", err)
	}
	rows, err := tx.QueryContext(ctx, `SELECT operation_id FROM review_operations o
		WHERE o.queue_class=? AND o.status IN ('pending','retry_scheduled')
		  AND (o.next_attempt_at='' OR o.next_attempt_at<=?)
		  AND (
			o.dependency_policy='none'
			OR (o.dependency_policy='success' AND EXISTS (
				SELECT 1 FROM review_operations p WHERE p.operation_id=o.dependency_operation_id
				AND p.status IN ('succeeded','unchanged')
			))
			OR (o.dependency_policy='terminal' AND EXISTS (
				SELECT 1 FROM review_operations p WHERE p.operation_id=o.dependency_operation_id
				AND p.status IN ('succeeded','unchanged','failed_terminal','unknown','needs_reconciliation','stale','dead_letter','cancelled')
			))
		  )
		ORDER BY o.created_at, o.operation_id LIMIT ?`, opts.QueueClass, nowText, opts.Limit)
	if err != nil {
		return nil, fmt.Errorf("select ready review operations: %w", err)
	}
	ids := []string{}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			_ = rows.Close()
			return nil, err
		}
		ids = append(ids, id)
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	leaseExpiry := reviewGatewayTimestamp(opts.Now.Add(opts.LeaseDuration))
	claimed := []ReviewOperation{}
	for _, id := range ids {
		result, updateErr := tx.ExecContext(ctx, `UPDATE review_operations
			SET status='leased', lease_owner=?, lease_expires_at=?, updated_at=?
			WHERE operation_id=? AND status IN ('pending','retry_scheduled')`,
			opts.LeaseOwner, leaseExpiry, nowText, id)
		if updateErr != nil {
			return nil, updateErr
		}
		affected, _ := result.RowsAffected()
		if affected != 1 {
			continue
		}
		operation, scanErr := scanReviewOperation(tx.QueryRowContext(ctx, reviewOperationSelect+` WHERE operation_id=?`, id))
		if scanErr != nil {
			return nil, scanErr
		}
		claimed = append(claimed, operation)
	}
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit review operation claim: %w", err)
	}
	return claimed, nil
}

func (s *SQLiteReviewGatewayStore) StartReviewOperationAttempt(ctx context.Context, operation ReviewOperation, now time.Time) (ReviewOperationAttempt, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return ReviewOperationAttempt{}, err
	}
	defer func() { _ = tx.Rollback() }()
	result, err := tx.ExecContext(ctx, `UPDATE review_operations
		SET status='writing', mutation_status='request_started', attempt_count=attempt_count+1, updated_at=?
		WHERE operation_id=? AND status='leased' AND lease_owner=? AND lease_expires_at>?`,
		reviewGatewayTimestamp(now), operation.OperationID, operation.LeaseOwner, reviewGatewayTimestamp(now))
	if err != nil {
		return ReviewOperationAttempt{}, err
	}
	affected, _ := result.RowsAffected()
	if affected != 1 {
		return ReviewOperationAttempt{}, fmt.Errorf("review operation %s lease is no longer current", operation.OperationID)
	}
	attemptNumber := operation.AttemptCount + 1
	attempt := ReviewOperationAttempt{
		AttemptID:      stableKey("review-operation-attempt", operation.OperationID, fmt.Sprintf("%d", attemptNumber)),
		OperationID:    operation.OperationID,
		AttemptNumber:  attemptNumber,
		LeaseOwnerHash: stableKey("lease-owner", operation.LeaseOwner),
		StartedAt:      reviewGatewayTimestamp(now),
		MutationStatus: ReviewMutationRequestStarted,
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO review_operation_attempts (
		attempt_id, operation_id, attempt_number, lease_owner_hash, started_at, mutation_status
	) VALUES (?, ?, ?, ?, ?, ?)`, attempt.AttemptID, attempt.OperationID, attempt.AttemptNumber, attempt.LeaseOwnerHash, attempt.StartedAt, attempt.MutationStatus)
	if err != nil {
		return ReviewOperationAttempt{}, fmt.Errorf("save review operation attempt: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return ReviewOperationAttempt{}, err
	}
	return attempt, nil
}

func (s *SQLiteReviewGatewayStore) FinishReviewOperation(ctx context.Context, operation ReviewOperation, attempt ReviewOperationAttempt, outcome ReviewOperationExecutionResult, executeErr error, now time.Time) error {
	classified := ClassifyReviewOperationError(executeErr, operation, true)
	status := ReviewOperationSucceeded
	mutationStatus := ReviewMutationLocalConfirmed
	appliedFingerprint := firstNonEmpty(outcome.AppliedFingerprint, operation.DesiredFingerprint)
	remoteID := firstNonEmpty(outcome.RemoteID, operation.RemoteID)
	errorClass, errorCode, errorSummary := "", "", ""
	httpStatus := 0
	retryAfterAt := ""
	nextAttemptAt := ""
	requiresReconciliation := false
	completedAt := reviewGatewayTimestamp(now)
	projectionStatus := ReviewProjectionSucceeded
	if outcome.Unchanged {
		status = ReviewOperationUnchanged
		projectionStatus = ReviewProjectionUnchanged
	}
	if classified != nil {
		errorClass = string(classified.Class)
		errorCode = classified.Code
		errorSummary = redactReviewGatewayError(classified.Error())
		httpStatus = classified.HTTPStatus
		appliedFingerprint = ""
		remoteID = firstNonEmpty(outcome.RemoteID, operation.RemoteID)
		switch classified.Class {
		case ReviewOperationErrorStale:
			status = ReviewOperationStale
			mutationStatus = ReviewMutationNotStarted
			projectionStatus = ReviewProjectionPending
		case ReviewOperationErrorUnknownSideEffect:
			status = ReviewOperationUnknown
			mutationStatus = ReviewMutationRemoteUnknown
			requiresReconciliation = true
			projectionStatus = ReviewProjectionNeedsReconciliation
		case ReviewOperationErrorTerminal:
			status = ReviewOperationFailedTerminal
			mutationStatus = ReviewMutationNotStarted
			projectionStatus = ReviewProjectionFailed
		case ReviewOperationErrorRateLimited, ReviewOperationErrorTransient:
			if operation.AttemptCount+1 < operation.MaxAttempts && !classified.RemoteSideEffectPossible {
				status = ReviewOperationRetryScheduled
				mutationStatus = ReviewMutationNotStarted
				delay := reviewGatewayRetryDelay(operation.AttemptCount + 1)
				if classified.RetryAfter > 0 {
					delay = classified.RetryAfter
					retryAfterAt = reviewGatewayTimestamp(now.Add(delay))
				}
				nextAttemptAt = reviewGatewayTimestamp(now.Add(delay))
				completedAt = ""
				projectionStatus = ReviewProjectionPending
			} else {
				status = ReviewOperationFailedTerminal
				mutationStatus = ReviewMutationNotStarted
				projectionStatus = ReviewProjectionFailed
			}
		}
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	result, err := tx.ExecContext(ctx, `UPDATE review_operation_attempts SET
		finished_at=?, mutation_status=?, error_class=?, error_code=?, error_summary=?,
		http_status=?, retry_after_at=?, result_fingerprint=?
		WHERE attempt_id=? AND operation_id=? AND attempt_number=? AND finished_at=''`,
		reviewGatewayTimestamp(now), mutationStatus, errorClass, errorCode, errorSummary,
		httpStatus, retryAfterAt, appliedFingerprint, attempt.AttemptID, operation.OperationID, attempt.AttemptNumber)
	if err != nil {
		return fmt.Errorf("finish review operation attempt: %w", err)
	}
	affected, _ := result.RowsAffected()
	if affected != 1 {
		return fmt.Errorf("review operation attempt %s is no longer writable", attempt.AttemptID)
	}
	result, err = tx.ExecContext(ctx, `UPDATE review_operations SET
		status=?, applied_fingerprint=?, remote_id=?, error_class=?, error_code=?,
		error_summary=?, http_status=?, mutation_status=?, requires_reconciliation=?,
		next_attempt_at=?, retry_after_at=?, lease_owner='', lease_expires_at='',
		updated_at=?, completed_at=?
		WHERE operation_id=? AND status='writing' AND lease_owner=? AND attempt_count=?`,
		status, appliedFingerprint, remoteID, errorClass, errorCode, errorSummary, httpStatus,
		mutationStatus, boolToReviewCollaborationInt(requiresReconciliation), nextAttemptAt,
		retryAfterAt, reviewGatewayTimestamp(now), completedAt, operation.OperationID,
		operation.LeaseOwner, attempt.AttemptNumber)
	if err != nil {
		return fmt.Errorf("finish review operation: %w", err)
	}
	affected, _ = result.RowsAffected()
	if affected != 1 {
		return fmt.Errorf("review operation %s lease is no longer current", operation.OperationID)
	}
	updated := operation
	updated.Status = status
	updated.AppliedFingerprint = appliedFingerprint
	updated.RemoteID = remoteID
	updated.ErrorClass = errorClass
	updated.ErrorSummary = errorSummary
	updated.RequiresReconciliation = requiresReconciliation
	if err := updateReviewProjectionStatusTx(ctx, tx, updated, projectionStatus, appliedFingerprint, errorClass, errorSummary, requiresReconciliation, now); err != nil {
		return fmt.Errorf("finish review operation projection: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	if err := s.QueueReviewProjectionCardRefresh(ctx, updated, now); err != nil {
		return fmt.Errorf("queue projection card refresh: %w", err)
	}
	return nil
}

const reviewOperationSelect = `SELECT
	operation_id, operation_kind, queue_class,
	installation_id, chat_id, repository, pr_number,
	source_job_id, source_event_id, consumer_id,
	work_item_key, resource_type, idempotency_key,
	desired_fingerprint, applied_fingerprint, expected_remote_id, remote_id, desired_json,
	dependency_operation_id, dependency_policy, retry_safety, status,
	error_class, error_code, error_summary, http_status,
	mutation_status, requires_reconciliation, attempt_count, max_attempts,
	next_attempt_at, retry_after_at, lease_owner, lease_expires_at,
	created_at, updated_at, completed_at
	FROM review_operations`

type reviewOperationScanner interface {
	Scan(...interface{}) error
}

func scanReviewOperation(scanner reviewOperationScanner) (ReviewOperation, error) {
	var operation ReviewOperation
	var requiresReconciliation int
	err := scanner.Scan(
		&operation.OperationID, &operation.OperationKind, &operation.QueueClass,
		&operation.InstallationID, &operation.ChatID, &operation.Repository, &operation.PRNumber,
		&operation.SourceJobID, &operation.SourceEventID, &operation.ConsumerID,
		&operation.WorkItemKey, &operation.ResourceType, &operation.IdempotencyKey,
		&operation.DesiredFingerprint, &operation.AppliedFingerprint, &operation.ExpectedRemoteID,
		&operation.RemoteID, &operation.DesiredJSON, &operation.DependencyOperationID,
		&operation.DependencyPolicy, &operation.RetrySafety, &operation.Status,
		&operation.ErrorClass, &operation.ErrorCode, &operation.ErrorSummary, &operation.HTTPStatus,
		&operation.MutationStatus, &requiresReconciliation, &operation.AttemptCount,
		&operation.MaxAttempts, &operation.NextAttemptAt, &operation.RetryAfterAt,
		&operation.LeaseOwner, &operation.LeaseExpiresAt, &operation.CreatedAt,
		&operation.UpdatedAt, &operation.CompletedAt,
	)
	operation.RequiresReconciliation = requiresReconciliation != 0
	if errors.Is(err, sql.ErrNoRows) {
		return ReviewOperation{}, ErrReviewOperationNotFound
	}
	if err != nil {
		return ReviewOperation{}, fmt.Errorf("scan review operation: %w", err)
	}
	return operation, nil
}
