package feishu

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"
)

type ReviewOperationReconciliationTask struct {
	ReconciliationID   string `json:"reconciliation_id"`
	OperationID        string `json:"operation_id"`
	ResourceType       string `json:"resource_type"`
	ReasonCode         string `json:"reason_code"`
	Status             string `json:"status"`
	VerificationMethod string `json:"verification_method,omitempty"`
	AttemptCount       int    `json:"attempt_count"`
	MaxAttempts        int    `json:"max_attempts"`
	NextAttemptAt      string `json:"next_attempt_at,omitempty"`
	LeaseOwner         string `json:"lease_owner,omitempty"`
	LeaseExpiresAt     string `json:"lease_expires_at,omitempty"`
	ResultSummary      string `json:"result_summary,omitempty"`
	ErrorSummary       string `json:"error_summary,omitempty"`
	CreatedAt          string `json:"created_at"`
	UpdatedAt          string `json:"updated_at"`
	ResolvedAt         string `json:"resolved_at,omitempty"`
}

func createReviewOperationReconciliationTx(ctx context.Context, tx *sql.Tx, operation ReviewOperation, reasonCode string, now time.Time) error {
	_, err := tx.ExecContext(ctx, `INSERT INTO review_operation_reconciliation_tasks (
		reconciliation_id, operation_id, resource_type, reason_code, status,
		max_attempts, next_attempt_at, created_at, updated_at
	) VALUES (?, ?, ?, ?, 'pending', 3, ?, ?, ?)
	ON CONFLICT(operation_id) DO UPDATE SET
		reason_code=excluded.reason_code,
		status=CASE WHEN review_operation_reconciliation_tasks.status IN ('resolved_success','manual_required')
			THEN review_operation_reconciliation_tasks.status ELSE 'pending' END,
		next_attempt_at=excluded.next_attempt_at,
		updated_at=excluded.updated_at`,
		stableKey("review-operation-reconciliation", operation.OperationID), operation.OperationID,
		operation.ResourceType, firstNonEmpty(reasonCode, "remote_side_effect_unknown"),
		reviewGatewayTimestamp(now), reviewGatewayTimestamp(now), reviewGatewayTimestamp(now))
	return err
}

func (s *SQLiteReviewGatewayStore) ListReviewOperationReconciliations(ctx context.Context, status string, limit int) ([]ReviewOperationReconciliationTask, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	query := reviewOperationReconciliationSelect + ` WHERE 1=1`
	args := []interface{}{}
	if strings.TrimSpace(status) != "" {
		query += ` AND status=?`
		args = append(args, strings.TrimSpace(status))
	}
	query += ` ORDER BY created_at, reconciliation_id LIMIT ?`
	args = append(args, limit)
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []ReviewOperationReconciliationTask{}
	for rows.Next() {
		item, err := scanReviewOperationReconciliation(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, item)
	}
	return result, rows.Err()
}

func (s *SQLiteReviewGatewayStore) GetReviewOperationReconciliation(ctx context.Context, id string) (ReviewOperationReconciliationTask, error) {
	return scanReviewOperationReconciliation(s.db.QueryRowContext(ctx, reviewOperationReconciliationSelect+` WHERE reconciliation_id=?`, strings.TrimSpace(id)))
}

func (s *SQLiteReviewGatewayStore) ClaimReviewOperationReconciliations(ctx context.Context, owner string, now time.Time, lease time.Duration, limit int) ([]ReviewOperationReconciliationTask, error) {
	if strings.TrimSpace(owner) == "" {
		return nil, fmt.Errorf("reconciliation lease owner is required")
	}
	if lease <= 0 {
		lease = time.Minute
	}
	if limit <= 0 || limit > 20 {
		limit = 1
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()
	nowText := reviewGatewayTimestamp(now)
	if _, err := tx.ExecContext(ctx, `UPDATE review_operation_reconciliation_tasks SET status='pending',
		lease_owner='', lease_expires_at='', updated_at=? WHERE status='checking' AND lease_expires_at<>'' AND lease_expires_at<=?`, nowText, nowText); err != nil {
		return nil, err
	}
	rows, err := tx.QueryContext(ctx, `SELECT reconciliation_id FROM review_operation_reconciliation_tasks
		WHERE status='pending' AND (next_attempt_at='' OR next_attempt_at<=?)
		ORDER BY created_at LIMIT ?`, nowText, limit)
	if err != nil {
		return nil, err
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
	_ = rows.Close()
	result := []ReviewOperationReconciliationTask{}
	for _, id := range ids {
		updated, err := tx.ExecContext(ctx, `UPDATE review_operation_reconciliation_tasks SET status='checking',
			lease_owner=?, lease_expires_at=?, attempt_count=attempt_count+1, updated_at=?
			WHERE reconciliation_id=? AND status='pending'`, owner, reviewGatewayTimestamp(now.Add(lease)), nowText, id)
		if err != nil {
			return nil, err
		}
		affected, _ := updated.RowsAffected()
		if affected != 1 {
			continue
		}
		item, err := scanReviewOperationReconciliation(tx.QueryRowContext(ctx, reviewOperationReconciliationSelect+` WHERE reconciliation_id=?`, id))
		if err != nil {
			return nil, err
		}
		result = append(result, item)
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return result, nil
}

func (s *SQLiteReviewGatewayStore) RecoverExpiredReviewOperationWrites(ctx context.Context, now time.Time) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	rows, err := tx.QueryContext(ctx, reviewOperationSelect+` WHERE status IN ('leased','writing') AND lease_expires_at<>'' AND lease_expires_at<=?`, reviewGatewayTimestamp(now))
	if err != nil {
		return err
	}
	operations := []ReviewOperation{}
	for rows.Next() {
		op, err := scanReviewOperation(rows)
		if err != nil {
			_ = rows.Close()
			return err
		}
		operations = append(operations, op)
	}
	_ = rows.Close()
	for _, op := range operations {
		status := ReviewOperationRetryScheduled
		mutation := ReviewMutationNotStarted
		reconcile := false
		if op.MutationStatus == ReviewMutationRequestStarted {
			switch op.RetrySafety {
			case ReviewRetryIdempotent:
				status = ReviewOperationRetryScheduled
			case ReviewRetryReconcilable:
				status = ReviewOperationNeedsReconciliation
				mutation = ReviewMutationRemoteUnknown
				reconcile = true
			default:
				status = ReviewOperationUnknown
				mutation = ReviewMutationRemoteUnknown
				reconcile = true
			}
		}
		if _, err := tx.ExecContext(ctx, `UPDATE review_operations SET status=?, mutation_status=?, requires_reconciliation=?,
			lease_owner='',lease_expires_at='',next_attempt_at=?,updated_at=? WHERE operation_id=? AND lease_expires_at<=?`, status, mutation, boolToReviewCollaborationInt(reconcile), reviewGatewayTimestamp(now), reviewGatewayTimestamp(now), op.OperationID, reviewGatewayTimestamp(now)); err != nil {
			return err
		}
		if reconcile {
			op.Status = status
			op.MutationStatus = mutation
			op.RequiresReconciliation = true
			if err := createReviewOperationReconciliationTx(ctx, tx, op, "expired_request_started", now); err != nil {
				return err
			}
		}
	}
	return tx.Commit()
}

type ReviewOperationReconciler struct {
	Store         *SQLiteReviewGatewayStore
	Client        reviewOperationOpenAPIClient
	TokenProvider *ReviewTenantTokenProvider
	Config        ReviewCollaborationPublisherConfig
	LeaseOwner    string
	LeaseDuration time.Duration
	Now           func() time.Time
	PollInterval  time.Duration
}

func (r *ReviewOperationReconciler) Run(ctx context.Context) {
	interval := r.PollInterval
	if interval <= 0 {
		interval = 2 * time.Second
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		for {
			ran, err := r.RunOnce(ctx)
			if err != nil || !ran {
				break
			}
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func (r *ReviewOperationReconciler) RunOnce(ctx context.Context) (bool, error) {
	if r == nil || r.Store == nil {
		return false, fmt.Errorf("operation reconciler store is required")
	}
	now := time.Now().UTC()
	if r.Now != nil {
		now = r.Now().UTC()
	}
	if r.LeaseDuration <= 0 {
		r.LeaseDuration = time.Minute
	}
	if r.LeaseOwner == "" {
		r.LeaseOwner = "operation-reconciler"
	}
	if err := r.Store.RecoverExpiredReviewOperationWrites(ctx, now); err != nil {
		return false, err
	}
	tasks, err := r.Store.ClaimReviewOperationReconciliations(ctx, r.LeaseOwner, now, r.LeaseDuration, 1)
	if err != nil || len(tasks) == 0 {
		return false, err
	}
	task := tasks[0]
	operation, err := r.Store.GetReviewOperation(ctx, task.OperationID)
	if err != nil {
		return true, r.fail(ctx, task, err, now)
	}
	if operation.ResourceType != ReviewResourceBitable {
		return true, r.manual(ctx, task, operation, "safe remote lookup is unavailable for this resource", now)
	}
	var desired reviewBitableDesired
	if err := jsonUnmarshalBounded(operation.DesiredJSON, &desired); err != nil {
		return true, r.manual(ctx, task, operation, "invalid Base reconciliation payload", now)
	}
	if r.Client == nil {
		return true, r.manual(ctx, task, operation, "Base reconciliation client is not configured", now)
	}
	if r.TokenProvider == nil {
		r.TokenProvider = NewReviewTenantTokenProvider(r.Client, r.Now)
	}
	token, err := r.TokenProvider.Token(ctx, r.Config.AppID, r.Config.AppSecret)
	if err != nil {
		return true, r.fail(ctx, task, err, now)
	}
	search, err := r.Client.SearchBitableRecord(ctx, token, r.Config.BaseAppToken, r.Config.ReviewTableID, desired.ResourceKey)
	if err != nil {
		return true, r.fail(ctx, task, err, now)
	}
	switch {
	case search.Matches > 1:
		return true, r.manual(ctx, task, operation, "multiple Base records match unique_key", now)
	case !search.Found:
		return true, r.retry(ctx, task, operation, "no Base record found; create may be retried", now)
	default:
		return true, r.resolveBase(ctx, task, operation, desired.ResourceKey, search.RecordID, now)
	}
}

func (r *ReviewOperationReconciler) resolveBase(ctx context.Context, task ReviewOperationReconciliationTask, operation ReviewOperation, key, remoteID string, now time.Time) error {
	tx, err := r.Store.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	if _, err := tx.ExecContext(ctx, `INSERT INTO review_collaboration_resources(work_item_key,resource_type,remote_id,content_fingerprint,updated_at)
		VALUES(?,?,?,?,?) ON CONFLICT(work_item_key,resource_type) DO UPDATE SET remote_id=excluded.remote_id,content_fingerprint=excluded.content_fingerprint,updated_at=excluded.updated_at`, key, ReviewResourceBitable, remoteID, operation.DesiredFingerprint, reviewGatewayTimestamp(now)); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `UPDATE review_operations SET status='succeeded',remote_id=?,applied_fingerprint=desired_fingerprint,
		mutation_status='local_confirmed',requires_reconciliation=0,error_class='',error_code='',error_summary='',completed_at=?,updated_at=? WHERE operation_id=?`, remoteID, reviewGatewayTimestamp(now), reviewGatewayTimestamp(now), operation.OperationID); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `UPDATE review_operation_reconciliation_tasks SET status='resolved_success',verification_method='base_unique_key',
		result_summary='one Base record adopted',lease_owner='',lease_expires_at='',resolved_at=?,updated_at=? WHERE reconciliation_id=? AND status='checking' AND lease_owner=?`, reviewGatewayTimestamp(now), reviewGatewayTimestamp(now), task.ReconciliationID, r.LeaseOwner); err != nil {
		return err
	}
	operation.RemoteID = remoteID
	operation.Status = ReviewOperationSucceeded
	if err := updateReviewProjectionStatusTx(ctx, tx, operation, ReviewProjectionSucceeded, operation.DesiredFingerprint, "", "", false, now); err != nil {
		return err
	}
	return tx.Commit()
}

func (r *ReviewOperationReconciler) retry(ctx context.Context, task ReviewOperationReconciliationTask, operation ReviewOperation, summary string, now time.Time) error {
	tx, err := r.Store.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	if _, err := tx.ExecContext(ctx, `UPDATE review_operations SET status='retry_scheduled',mutation_status='not_started',requires_reconciliation=0,
		next_attempt_at=?,lease_owner='',lease_expires_at='',updated_at=? WHERE operation_id=?`, reviewGatewayTimestamp(now), reviewGatewayTimestamp(now), operation.OperationID); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `UPDATE review_operation_reconciliation_tasks SET status='resolved_retry',verification_method='base_unique_key',
		result_summary=?,lease_owner='',lease_expires_at='',resolved_at=?,updated_at=? WHERE reconciliation_id=? AND status='checking' AND lease_owner=?`, summary, reviewGatewayTimestamp(now), reviewGatewayTimestamp(now), task.ReconciliationID, r.LeaseOwner); err != nil {
		return err
	}
	return tx.Commit()
}

func (r *ReviewOperationReconciler) manual(ctx context.Context, task ReviewOperationReconciliationTask, operation ReviewOperation, summary string, now time.Time) error {
	tx, err := r.Store.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	if _, err := tx.ExecContext(ctx, `UPDATE review_operations SET status='needs_reconciliation',requires_reconciliation=1,updated_at=? WHERE operation_id=?`, reviewGatewayTimestamp(now), operation.OperationID); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `UPDATE review_operation_reconciliation_tasks SET status='manual_required',verification_method='manual',result_summary=?,
		lease_owner='',lease_expires_at='',resolved_at=?,updated_at=? WHERE reconciliation_id=? AND status='checking' AND lease_owner=?`, summary, reviewGatewayTimestamp(now), reviewGatewayTimestamp(now), task.ReconciliationID, r.LeaseOwner); err != nil {
		return err
	}
	return tx.Commit()
}

func (r *ReviewOperationReconciler) fail(ctx context.Context, task ReviewOperationReconciliationTask, runErr error, now time.Time) error {
	status := "pending"
	next := reviewGatewayTimestamp(now.Add(reviewGatewayRetryDelay(task.AttemptCount)))
	if task.AttemptCount >= task.MaxAttempts {
		status = "dead_letter"
		next = ""
	}
	tx, err := r.Store.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	if _, err := tx.ExecContext(ctx, `UPDATE review_operation_reconciliation_tasks SET status=?,next_attempt_at=?,error_summary=?,
		lease_owner='',lease_expires_at='',updated_at=? WHERE reconciliation_id=? AND status='checking' AND lease_owner=?`, status, next, redactReviewGatewayError(runErr.Error()), reviewGatewayTimestamp(now), task.ReconciliationID, r.LeaseOwner); err != nil {
		return err
	}
	if status == "dead_letter" {
		if _, err := tx.ExecContext(ctx, `INSERT INTO review_dead_letters(dead_letter_id,entity_type,entity_id,queue_class,error_class,error_code,error_summary,attempt_count,status,created_at,updated_at)
		VALUES(?,'reconciliation',?,'operation_reconciliation','transient','reconciliation_exhausted',?,?,'open',?,?) ON CONFLICT(entity_type,entity_id) DO NOTHING`, stableKey("review-dead-letter", "reconciliation", task.ReconciliationID), task.ReconciliationID, redactReviewGatewayError(runErr.Error()), task.AttemptCount, reviewGatewayTimestamp(now), reviewGatewayTimestamp(now)); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (s *SQLiteReviewGatewayStore) ResolveReviewOperationReconciliation(ctx context.Context, id, summary, actor string, success bool, now time.Time) error {
	if strings.TrimSpace(summary) == "" {
		return fmt.Errorf("reconciliation resolution summary is required")
	}
	status := "manual_required"
	if success {
		status = "resolved_success"
	}
	result, err := s.db.ExecContext(ctx, `UPDATE review_operation_reconciliation_tasks SET status=?,verification_method='manual',result_summary=?,
		lease_owner='',lease_expires_at='',resolved_at=?,updated_at=? WHERE reconciliation_id=? AND status IN ('manual_required','pending','checking')`, status, truncateReviewGatewayText(summary+" by "+reviewGatewayHashIdentifier(actor), 300), reviewGatewayTimestamp(now), reviewGatewayTimestamp(now), id)
	if err != nil {
		return err
	}
	affected, _ := result.RowsAffected()
	if affected != 1 {
		return fmt.Errorf("reconciliation %s is not resolvable", id)
	}
	return nil
}

const reviewOperationReconciliationSelect = `SELECT reconciliation_id,operation_id,resource_type,reason_code,status,
	verification_method,attempt_count,max_attempts,next_attempt_at,lease_owner,lease_expires_at,result_summary,error_summary,created_at,updated_at,resolved_at FROM review_operation_reconciliation_tasks`

type reviewOperationReconciliationScanner interface{ Scan(...interface{}) error }

func scanReviewOperationReconciliation(scanner reviewOperationReconciliationScanner) (ReviewOperationReconciliationTask, error) {
	var item ReviewOperationReconciliationTask
	err := scanner.Scan(&item.ReconciliationID, &item.OperationID, &item.ResourceType, &item.ReasonCode, &item.Status, &item.VerificationMethod, &item.AttemptCount, &item.MaxAttempts, &item.NextAttemptAt, &item.LeaseOwner, &item.LeaseExpiresAt, &item.ResultSummary, &item.ErrorSummary, &item.CreatedAt, &item.UpdatedAt, &item.ResolvedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return item, sql.ErrNoRows
	}
	return item, err
}
