package feishu

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"
)

type ReviewDeadLetter struct {
	DeadLetterID     string `json:"dead_letter_id"`
	EntityType       string `json:"entity_type"`
	EntityID         string `json:"entity_id"`
	QueueClass       string `json:"queue_class,omitempty"`
	InstallationID   string `json:"installation_id,omitempty"`
	ChatIDHash       string `json:"chat_id_hash,omitempty"`
	Repository       string `json:"repository,omitempty"`
	PRNumber         int    `json:"pr_number,omitempty"`
	ErrorClass       string `json:"error_class"`
	ErrorCode        string `json:"error_code,omitempty"`
	ErrorSummary     string `json:"error_summary,omitempty"`
	AttemptCount     int    `json:"attempt_count"`
	Status           string `json:"status"`
	ResolutionAction string `json:"resolution_action,omitempty"`
	ResolvedByHash   string `json:"resolved_by_hash,omitempty"`
	ResolvedAt       string `json:"resolved_at,omitempty"`
	CreatedAt        string `json:"created_at"`
	UpdatedAt        string `json:"updated_at"`
}

func createReviewOperationDeadLetterTx(ctx context.Context, tx *sql.Tx, operation ReviewOperation, now time.Time) error {
	_, err := tx.ExecContext(ctx, `INSERT INTO review_dead_letters (
		dead_letter_id, entity_type, entity_id, queue_class, installation_id,
		chat_id_hash, repository, pr_number, error_class, error_code, error_summary,
		attempt_count, status, created_at, updated_at
	) VALUES (?, 'operation', ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 'open', ?, ?)
	ON CONFLICT(entity_type, entity_id) DO UPDATE SET
		error_class=excluded.error_class,
		error_code=excluded.error_code,
		error_summary=excluded.error_summary,
		attempt_count=excluded.attempt_count,
		updated_at=excluded.updated_at`,
		stableKey("review-dead-letter", "operation", operation.OperationID),
		operation.OperationID, operation.QueueClass, operation.InstallationID,
		reviewGatewayHashIdentifier(operation.ChatID), operation.Repository, operation.PRNumber,
		operation.ErrorClass, operation.ErrorCode, redactReviewGatewayError(operation.ErrorSummary),
		operation.AttemptCount, reviewGatewayTimestamp(now), reviewGatewayTimestamp(now))
	return err
}

func (s *SQLiteReviewGatewayStore) ListReviewDeadLetters(ctx context.Context, status, entityType string, limit int) ([]ReviewDeadLetter, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	query := reviewDeadLetterSelect + ` WHERE 1=1`
	args := []interface{}{}
	if strings.TrimSpace(status) != "" {
		query += ` AND status=?`
		args = append(args, strings.TrimSpace(status))
	}
	if strings.TrimSpace(entityType) != "" {
		query += ` AND entity_type=?`
		args = append(args, strings.TrimSpace(entityType))
	}
	query += ` ORDER BY created_at, dead_letter_id LIMIT ?`
	args = append(args, limit)
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []ReviewDeadLetter{}
	for rows.Next() {
		item, err := scanReviewDeadLetter(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, item)
	}
	return result, rows.Err()
}

func (s *SQLiteReviewGatewayStore) GetReviewDeadLetter(ctx context.Context, id string) (ReviewDeadLetter, error) {
	return scanReviewDeadLetter(s.db.QueryRowContext(ctx, reviewDeadLetterSelect+` WHERE dead_letter_id=?`, strings.TrimSpace(id)))
}

func (s *SQLiteReviewGatewayStore) RetryReviewDeadLetter(ctx context.Context, id, reason string, confirmed bool, actor string, now time.Time) error {
	if strings.TrimSpace(reason) == "" || !confirmed {
		return fmt.Errorf("dead-letter retry requires --reason and --yes")
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	item, err := scanReviewDeadLetter(tx.QueryRowContext(ctx, reviewDeadLetterSelect+` WHERE dead_letter_id=?`, strings.TrimSpace(id)))
	if err != nil {
		return err
	}
	if item.Status != "open" || item.EntityType != "operation" {
		return fmt.Errorf("dead letter %s is not retryable", id)
	}
	operation, err := scanReviewOperation(tx.QueryRowContext(ctx, reviewOperationSelect+` WHERE operation_id=?`, item.EntityID))
	if err != nil {
		return err
	}
	if operation.Status == ReviewOperationUnknown || operation.Status == ReviewOperationNeedsReconciliation || operation.MutationStatus == ReviewMutationRemoteUnknown {
		return fmt.Errorf("unknown non-idempotent operation must be reconciled before retry")
	}
	if _, err := tx.ExecContext(ctx, `UPDATE review_operations SET status='retry_scheduled',
		error_class='', error_code='', error_summary='', next_attempt_at=?, completed_at='', updated_at=?
		WHERE operation_id=? AND status IN ('dead_letter','failed_terminal')`,
		reviewGatewayTimestamp(now), reviewGatewayTimestamp(now), operation.OperationID); err != nil {
		return err
	}
	result, err := tx.ExecContext(ctx, `UPDATE review_dead_letters SET status='resolved',
		resolution_action=?, resolved_by_hash=?, resolved_at=?, updated_at=?
		WHERE dead_letter_id=? AND status='open'`, truncateReviewGatewayText(reason, 300), reviewGatewayHashIdentifier(actor), reviewGatewayTimestamp(now), reviewGatewayTimestamp(now), id)
	if err != nil {
		return err
	}
	affected, _ := result.RowsAffected()
	if affected != 1 {
		return fmt.Errorf("dead letter %s changed concurrently", id)
	}
	return tx.Commit()
}

func (s *SQLiteReviewGatewayStore) ResolveReviewDeadLetter(ctx context.Context, id, action, actor string, ignored bool, now time.Time) error {
	if strings.TrimSpace(action) == "" {
		return fmt.Errorf("dead-letter resolution action is required")
	}
	status := "resolved"
	if ignored {
		status = "ignored"
	}
	result, err := s.db.ExecContext(ctx, `UPDATE review_dead_letters SET status=?, resolution_action=?,
		resolved_by_hash=?, resolved_at=?, updated_at=? WHERE dead_letter_id=? AND status IN ('open','resolving')`,
		status, truncateReviewGatewayText(action, 300), reviewGatewayHashIdentifier(actor), reviewGatewayTimestamp(now), reviewGatewayTimestamp(now), id)
	if err != nil {
		return err
	}
	affected, _ := result.RowsAffected()
	if affected != 1 {
		return fmt.Errorf("dead letter %s is not open", id)
	}
	return nil
}

const reviewDeadLetterSelect = `SELECT dead_letter_id, entity_type, entity_id, queue_class,
	installation_id, chat_id_hash, repository, pr_number, error_class, error_code,
	error_summary, attempt_count, status, resolution_action, resolved_by_hash,
	resolved_at, created_at, updated_at FROM review_dead_letters`

type reviewDeadLetterScanner interface{ Scan(...interface{}) error }

func scanReviewDeadLetter(scanner reviewDeadLetterScanner) (ReviewDeadLetter, error) {
	var item ReviewDeadLetter
	err := scanner.Scan(&item.DeadLetterID, &item.EntityType, &item.EntityID, &item.QueueClass,
		&item.InstallationID, &item.ChatIDHash, &item.Repository, &item.PRNumber,
		&item.ErrorClass, &item.ErrorCode, &item.ErrorSummary, &item.AttemptCount,
		&item.Status, &item.ResolutionAction, &item.ResolvedByHash, &item.ResolvedAt,
		&item.CreatedAt, &item.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return ReviewDeadLetter{}, sql.ErrNoRows
	}
	return item, err
}
