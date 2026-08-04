package feishu

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

const (
	reviewCardPresentationV1 = 1
	reviewGatewayAppScope    = "gitlink-feishu-review"
)

// ReviewChatPRPresentation owns the single canonical full card for one
// app/installation/chat/PR scope. A reply message ID is never stored here.
type ReviewChatPRPresentation struct {
	PresentationKey        string `json:"presentation_key"`
	AppScope               string `json:"app_scope"`
	InstallationID         string `json:"installation_id"`
	ChatID                 string `json:"chat_id"`
	Repository             string `json:"repository"`
	PRNumber               int    `json:"pr_number"`
	CanonicalMessageID     string `json:"canonical_message_id,omitempty"`
	ContentFingerprint     string `json:"content_fingerprint,omitempty"`
	SourceCompletedAt      string `json:"source_completed_at,omitempty"`
	PresentationVersion    int    `json:"presentation_version"`
	CardStatus             string `json:"card_status"`
	LastOperationID        string `json:"last_operation_id,omitempty"`
	LastPatchAt            string `json:"last_patch_at,omitempty"`
	RequiresReconciliation bool   `json:"requires_reconciliation"`
	LastErrorSummary       string `json:"last_error_summary,omitempty"`
	CreatedAt              string `json:"created_at"`
	UpdatedAt              string `json:"updated_at"`
}

func reviewChatPRPresentationKey(job ReviewGatewayJob) string {
	return stableKey(
		"chat-pr-presentation",
		reviewGatewayAppScope,
		firstNonEmpty(job.InstallationID, "legacy"),
		job.ChatID,
		job.Repository,
		fmt.Sprintf("%d", job.PRNumber),
	)
}

func reviewGatewayCardFingerprint(card Card) string {
	encoded, _ := json.Marshal(card)
	sum := sha256.Sum256(encoded)
	return hex.EncodeToString(sum[:])
}

func (s *SQLiteReviewGatewayStore) GetChatPRPresentation(ctx context.Context, job ReviewGatewayJob) (ReviewChatPRPresentation, error) {
	var state ReviewChatPRPresentation
	var reconciliation int
	err := s.db.QueryRowContext(ctx, `SELECT presentation_key, app_scope, installation_id,
		chat_id, repository, pr_number, canonical_message_id, content_fingerprint, source_completed_at,
		presentation_version, card_status, last_operation_id, last_patch_at,
		requires_reconciliation, last_error_summary, created_at, updated_at
		FROM chat_pr_presentations WHERE presentation_key = ?`,
		reviewChatPRPresentationKey(job),
	).Scan(
		&state.PresentationKey, &state.AppScope, &state.InstallationID, &state.ChatID,
		&state.Repository, &state.PRNumber, &state.CanonicalMessageID,
		&state.ContentFingerprint, &state.SourceCompletedAt, &state.PresentationVersion, &state.CardStatus,
		&state.LastOperationID, &state.LastPatchAt, &reconciliation,
		&state.LastErrorSummary, &state.CreatedAt, &state.UpdatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return ReviewChatPRPresentation{}, nil
	}
	state.RequiresReconciliation = reconciliation != 0
	return state, err
}

func (s *SQLiteReviewGatewayStore) AcquireChatPRPresentationCreate(
	ctx context.Context,
	job ReviewGatewayJob,
	operationID string,
	now time.Time,
) (ReviewChatPRPresentation, bool, error) {
	if strings.TrimSpace(operationID) == "" {
		return ReviewChatPRPresentation{}, false, fmt.Errorf("canonical card create operation ID is required")
	}
	nowText := now.UTC().Format(time.RFC3339Nano)
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return ReviewChatPRPresentation{}, false, err
	}
	defer tx.Rollback()
	key := reviewChatPRPresentationKey(job)
	if _, err := tx.ExecContext(ctx, `INSERT INTO chat_pr_presentations (
		presentation_key, app_scope, installation_id, chat_id, repository, pr_number,
		presentation_version, card_status, created_at, updated_at
	) VALUES (?, ?, ?, ?, ?, ?, 0, 'pending', ?, ?)
	ON CONFLICT(presentation_key) DO NOTHING`,
		key, reviewGatewayAppScope, firstNonEmpty(job.InstallationID, "legacy"), job.ChatID,
		job.Repository, job.PRNumber, nowText, nowText,
	); err != nil {
		return ReviewChatPRPresentation{}, false, err
	}
	update, err := tx.ExecContext(ctx, `UPDATE chat_pr_presentations SET
		card_status='creating', last_operation_id=?, last_error_summary='', updated_at=?
		WHERE presentation_key=? AND card_status='pending' AND requires_reconciliation=0`,
		operationID, nowText, key,
	)
	if err != nil {
		return ReviewChatPRPresentation{}, false, err
	}
	affected, err := update.RowsAffected()
	if err != nil {
		return ReviewChatPRPresentation{}, false, err
	}
	if err := tx.Commit(); err != nil {
		return ReviewChatPRPresentation{}, false, err
	}
	state, err := s.GetChatPRPresentation(ctx, job)
	return state, affected == 1, err
}

func (s *SQLiteReviewGatewayStore) CompleteChatPRPresentationCreate(
	ctx context.Context,
	job ReviewGatewayJob,
	operationID,
	messageID,
	fingerprint string,
	sourceCompletedAt string,
	archived bool,
	now time.Time,
) error {
	status := "active"
	if archived {
		status = "archived"
	}
	update, err := s.db.ExecContext(ctx, `UPDATE chat_pr_presentations SET
		canonical_message_id=?, content_fingerprint=?, source_completed_at=?, presentation_version=1,
		card_status=?, requires_reconciliation=0, last_error_summary='', updated_at=?
		WHERE presentation_key=? AND card_status='creating' AND last_operation_id=?`,
		messageID, fingerprint, sourceCompletedAt, status, now.UTC().Format(time.RFC3339Nano),
		reviewChatPRPresentationKey(job), operationID,
	)
	return requireReviewChatPRPresentationUpdate(update, err, job, "complete canonical card create")
}

func (s *SQLiteReviewGatewayStore) AcquireChatPRPresentationPatch(
	ctx context.Context,
	job ReviewGatewayJob,
	expectedFingerprint,
	operationID string,
	sourceCompletedAt string,
	now time.Time,
) (ReviewChatPRPresentation, bool, error) {
	if strings.TrimSpace(operationID) == "" {
		return ReviewChatPRPresentation{}, false, fmt.Errorf("canonical card patch operation ID is required")
	}
	state, err := s.GetChatPRPresentation(ctx, job)
	if err != nil {
		return ReviewChatPRPresentation{}, false, err
	}
	if stale, err := reviewChatPresentationSourceIsStale(state.SourceCompletedAt, sourceCompletedAt); err != nil {
		return ReviewChatPRPresentation{}, false, err
	} else if stale {
		return state, false, ErrStaleReviewPRPresentation
	}
	update, err := s.db.ExecContext(ctx, `UPDATE chat_pr_presentations SET
		card_status='patching', last_operation_id=?, last_error_summary='', updated_at=?
		WHERE presentation_key=? AND card_status IN ('active', 'archived')
		  AND content_fingerprint=? AND requires_reconciliation=0`,
		operationID, now.UTC().Format(time.RFC3339Nano), reviewChatPRPresentationKey(job), expectedFingerprint,
	)
	if err != nil {
		return ReviewChatPRPresentation{}, false, err
	}
	affected, err := update.RowsAffected()
	if err != nil {
		return ReviewChatPRPresentation{}, false, err
	}
	state, err = s.GetChatPRPresentation(ctx, job)
	return state, affected == 1, err
}

func (s *SQLiteReviewGatewayStore) CompleteChatPRPresentationPatch(
	ctx context.Context,
	job ReviewGatewayJob,
	operationID,
	fingerprint string,
	sourceCompletedAt string,
	archived bool,
	now time.Time,
) error {
	status := "active"
	if archived {
		status = "archived"
	}
	nowText := now.UTC().Format(time.RFC3339Nano)
	update, err := s.db.ExecContext(ctx, `UPDATE chat_pr_presentations SET
		content_fingerprint=?, source_completed_at=?, presentation_version=presentation_version+1,
		card_status=?, last_patch_at=?, requires_reconciliation=0,
		last_error_summary='', updated_at=?
		WHERE presentation_key=? AND card_status='patching' AND last_operation_id=?`,
		fingerprint, sourceCompletedAt, status, nowText, nowText, reviewChatPRPresentationKey(job), operationID,
	)
	return requireReviewChatPRPresentationUpdate(update, err, job, "complete canonical card patch")
}

func reviewChatPresentationSourceIsStale(current, incoming string) (bool, error) {
	if strings.TrimSpace(current) == "" {
		return false, nil
	}
	currentTime, currentErr := time.Parse(time.RFC3339Nano, strings.TrimSpace(current))
	incomingTime, incomingErr := time.Parse(time.RFC3339Nano, strings.TrimSpace(incoming))
	if currentErr != nil || incomingErr != nil {
		return false, fmt.Errorf("compare canonical card source freshness: %w", ErrStaleReviewPRPresentation)
	}
	return incomingTime.Before(currentTime), nil
}

func (s *SQLiteReviewGatewayStore) MarkChatPRPresentationUnknown(
	ctx context.Context,
	job ReviewGatewayJob,
	operationID,
	messageID,
	errorSummary string,
	now time.Time,
) error {
	update, err := s.db.ExecContext(ctx, `UPDATE chat_pr_presentations SET
		canonical_message_id=CASE WHEN ?<>'' THEN ? ELSE canonical_message_id END,
		card_status='unknown', requires_reconciliation=1,
		last_error_summary=?, updated_at=?
		WHERE presentation_key=? AND last_operation_id=?
		  AND card_status IN ('creating', 'patching')`,
		messageID, messageID, redactReviewGatewayError(errorSummary), now.UTC().Format(time.RFC3339Nano),
		reviewChatPRPresentationKey(job), operationID,
	)
	return requireReviewChatPRPresentationUpdate(update, err, job, "mark canonical card unknown")
}

func requireReviewChatPRPresentationUpdate(result sql.Result, err error, job ReviewGatewayJob, action string) error {
	if err != nil {
		return fmt.Errorf("%s: %w", action, err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("%s rows: %w", action, err)
	}
	if affected != 1 {
		return fmt.Errorf("%s for %s PR #%d affected %d rows", action, job.Repository, job.PRNumber, affected)
	}
	return nil
}
