package feishu

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"
)

func (s *SQLiteReviewGatewayStore) WaitForReviewReadJobs(ctx context.Context) error {
	return s.waitForReviewCount(ctx, `SELECT COUNT(*) FROM review_gateway_jobs WHERE status='running' AND queue_class='gitlink_read'`)
}

func (s *SQLiteReviewGatewayStore) WaitForReviewOperations(ctx context.Context) error {
	return s.waitForReviewCount(ctx, `SELECT COUNT(*) FROM review_operations WHERE status IN ('leased','writing')`)
}

func (s *SQLiteReviewGatewayStore) ReleaseNotStartedReviewWork(ctx context.Context, now time.Time) error {
	if s == nil || s.db == nil {
		return nil
	}
	nowText := reviewGatewayTimestamp(now)
	if _, err := s.db.ExecContext(ctx, `UPDATE review_operations SET status='pending',lease_owner='',lease_expires_at='',updated_at=?
		WHERE status='leased' AND mutation_status='not_started'`, nowText); err != nil {
		return fmt.Errorf("release not-started review operation leases: %w", err)
	}
	return nil
}

func (s *SQLiteReviewGatewayStore) waitForReviewCount(ctx context.Context, query string) error {
	ticker := time.NewTicker(20 * time.Millisecond)
	defer ticker.Stop()
	for {
		var count int
		if err := s.db.QueryRowContext(ctx, query).Scan(&count); err != nil {
			return err
		}
		if count == 0 {
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
}

// PrepareReviewWorkForShutdown releases only work that is proven not to have
// crossed a remote mutation boundary. Ambiguous operations and replies are
// made reconcilable and are never returned to a blindly retryable state.
func (s *SQLiteReviewGatewayStore) PrepareReviewWorkForShutdown(ctx context.Context, now time.Time) error {
	if s == nil || s.db == nil {
		return nil
	}
	nowText := reviewGatewayTimestamp(now)
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, `UPDATE review_gateway_jobs SET status='queued',lease_owner='',lease_expires_at='',updated_at=?
		WHERE status='running' AND queue_class IN ('gitlink_read','collaboration')`, nowText); err != nil {
		return fmt.Errorf("release review job leases during shutdown: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `UPDATE review_gateway_jobs SET reply_status='unknown',reply_requires_reconciliation=1,
		reply_lease_owner='',reply_lease_expires_at='',reply_error_summary='shutdown interrupted an in-flight reply',updated_at=?
		WHERE reply_status='sending'`, nowText); err != nil {
		return fmt.Errorf("protect in-flight review replies during shutdown: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `UPDATE review_operations SET status='pending',lease_owner='',lease_expires_at='',updated_at=?
		WHERE status='leased' AND mutation_status='not_started'`, nowText); err != nil {
		return fmt.Errorf("release not-started review operation leases: %w", err)
	}
	type uncertainOperation struct {
		id, resourceType, retrySafety string
	}
	rows, err := tx.QueryContext(ctx, `SELECT operation_id,resource_type,retry_safety FROM review_operations
		WHERE status='writing' AND mutation_status='request_started'`)
	if err != nil {
		return err
	}
	operations := []uncertainOperation{}
	for rows.Next() {
		var operation uncertainOperation
		if err := rows.Scan(&operation.id, &operation.resourceType, &operation.retrySafety); err != nil {
			_ = rows.Close()
			return err
		}
		operations = append(operations, operation)
	}
	if err := rows.Close(); err != nil {
		return err
	}
	for _, operation := range operations {
		status, mutation, reconciliation := string(ReviewOperationRetryScheduled), string(ReviewMutationNotStarted), 0
		errorClass := string(ReviewOperationErrorTransient)
		if operation.retrySafety == string(ReviewRetryReconcilable) {
			status, mutation, reconciliation = string(ReviewOperationNeedsReconciliation), string(ReviewMutationRemoteUnknown), 1
			errorClass = string(ReviewOperationErrorUnknownSideEffect)
		} else if operation.retrySafety == string(ReviewRetryNonIdempotent) {
			status, mutation, reconciliation = string(ReviewOperationUnknown), string(ReviewMutationRemoteUnknown), 1
			errorClass = string(ReviewOperationErrorUnknownSideEffect)
		}
		if _, err := tx.ExecContext(ctx, `UPDATE review_operations SET status=?,mutation_status=?,requires_reconciliation=?,
			error_class=?,error_code='shutdown_interrupted',error_summary='shutdown interrupted an in-flight operation',
			lease_owner='',lease_expires_at='',next_attempt_at=?,updated_at=? WHERE operation_id=? AND status='writing'`,
			status, mutation, reconciliation, errorClass, func() string {
				if status == string(ReviewOperationRetryScheduled) {
					return nowText
				}
				return ""
			}(), nowText, operation.id); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `UPDATE review_operation_attempts SET finished_at=?,mutation_status=?,error_class=?,
			error_code='shutdown_interrupted',error_summary='shutdown interrupted an in-flight operation'
			WHERE operation_id=? AND finished_at=''`, nowText, mutation, errorClass, operation.id); err != nil {
			return err
		}
		if reconciliation != 0 {
			reconciliationID := stableKey("review-operation-reconciliation", operation.id)
			if _, err := tx.ExecContext(ctx, `INSERT INTO review_operation_reconciliation_tasks(
				reconciliation_id,operation_id,resource_type,reason_code,status,next_attempt_at,created_at,updated_at
			) VALUES(?,?,?,'shutdown_interrupted','pending',?,?,?) ON CONFLICT(operation_id) DO NOTHING`, reconciliationID, operation.id, operation.resourceType, nowText, nowText, nowText); err != nil {
				return err
			}
		}
	}
	// Controlled GitLink writes use a separate action-plan state machine.
	// pre_write is safe to release; remote_write_possible remains explicitly
	// unknown and cannot be auto-claimed by the normal execution path.
	if _, err := tx.ExecContext(ctx, `UPDATE review_action_plans SET status='pending_confirmation',lease_owner='',lease_expires_at='',updated_at=?
		WHERE status='executing' AND reconciliation_status='pre_write' AND mutation_status='not_started'`, nowText); err != nil && !strings.Contains(strings.ToLower(err.Error()), "no such table") {
		return err
	}
	if _, err := tx.ExecContext(ctx, `UPDATE review_action_plans SET status='unknown_needs_reconciliation',reconciliation_status='unknown_needs_reconciliation',
		mutation_status='possible',lease_owner='',lease_expires_at='',error_summary='shutdown interrupted a possible remote write',updated_at=?
		WHERE status='executing' AND reconciliation_status='remote_write_possible'`, nowText); err != nil && !strings.Contains(strings.ToLower(err.Error()), "no such table") {
		return err
	}
	return tx.Commit()
}

func reviewContextWithMaximum(parent context.Context, maximum time.Duration) (context.Context, context.CancelFunc) {
	if maximum <= 0 {
		return context.WithCancel(parent)
	}
	if deadline, ok := parent.Deadline(); ok && time.Until(deadline) <= maximum {
		return context.WithCancel(parent)
	}
	return context.WithTimeout(parent, maximum)
}

func reviewShutdownError(current, next error) error {
	if current != nil {
		return current
	}
	return next
}

func reviewDatabaseOpen(db *sql.DB) bool {
	if db == nil {
		return false
	}
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	return db.PingContext(ctx) == nil
}
