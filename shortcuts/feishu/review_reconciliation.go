package feishu

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"
)

type ReviewReconciliationCandidate struct {
	CursorKey            string `json:"cursor_key"`
	InstallationID       string `json:"installation_id"`
	ChatIDHash           string `json:"chat_id_hash"`
	Repository           string `json:"repository"`
	PRNumber             int    `json:"pr_number"`
	CanonicalMessageHash string `json:"canonical_message_hash"`
	SourceFingerprint    string `json:"source_fingerprint,omitempty"`
	chatID               string
}

type ReviewReconciliationCursor struct {
	CursorKey             string `json:"cursor_key"`
	InstallationID        string `json:"installation_id"`
	ChatIDHash            string `json:"chat_id_hash"`
	Repository            string `json:"repository"`
	PRNumber              int    `json:"pr_number"`
	LastCheckedAt         string `json:"last_checked_at,omitempty"`
	LastEnqueuedAt        string `json:"last_enqueued_at,omitempty"`
	NextCheckAt           string `json:"next_check_at,omitempty"`
	LastSourceFingerprint string `json:"last_source_fingerprint,omitempty"`
	Status                string `json:"status"`
	ErrorSummary          string `json:"error_summary,omitempty"`
	UpdatedAt             string `json:"updated_at"`
	chatID                string
}

type ReviewReconciliationRunResult struct {
	DryRun     bool                            `json:"dry_run"`
	Candidates []ReviewReconciliationCandidate `json:"candidates"`
	Enqueued   int                             `json:"enqueued"`
	Duplicates int                             `json:"duplicates"`
	Skipped    int                             `json:"skipped"`
}

type ReviewReconciliationScheduler struct {
	Store    *SQLiteReviewGatewayStore
	Queue    ReviewPreparedJobEnqueuer
	Interval time.Duration
	Now      func() time.Time
}

func validateReviewReconciliationInterval(interval time.Duration) error {
	if interval == 0 {
		return nil
	}
	if interval < 5*time.Minute || interval > 24*time.Hour {
		return fmt.Errorf("Review reconciliation interval must be 0 or between 5m and 24h")
	}
	return nil
}

func (s *SQLiteReviewGatewayStore) ListReviewReconciliationCandidates(ctx context.Context) ([]ReviewReconciliationCandidate, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT cp.installation_id, cp.chat_id, cp.repository,
		cp.pr_number, cp.canonical_message_id, p.source_fingerprint
		FROM chat_pr_presentations cp
		JOIN review_pr_presentations p ON p.installation_id=cp.installation_id
		 AND p.repository=cp.repository AND p.pr_number=cp.pr_number
		JOIN review_chat_subscriptions s ON s.installation_id=cp.installation_id
		 AND s.chat_id=cp.chat_id AND s.repository=cp.repository AND s.enabled=1
		JOIN chat_repository_bindings b ON b.installation_id=cp.installation_id
		 AND b.chat_id=cp.chat_id AND b.repository=cp.repository AND b.enabled=1
		JOIN gitlink_installations i ON i.installation_id=cp.installation_id AND i.enabled=1
		JOIN installation_repositories ir ON ir.installation_id=cp.installation_id AND ir.repository=cp.repository
		WHERE cp.card_status='active' AND cp.canonical_message_id<>''
		AND p.gitlink_state NOT IN ('merged','closed')
		ORDER BY cp.installation_id, cp.chat_id, cp.repository, cp.pr_number`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []ReviewReconciliationCandidate{}
	for rows.Next() {
		var candidate ReviewReconciliationCandidate
		var messageID string
		if err := rows.Scan(&candidate.InstallationID, &candidate.chatID, &candidate.Repository,
			&candidate.PRNumber, &messageID, &candidate.SourceFingerprint); err != nil {
			return nil, err
		}
		candidate.CursorKey = reviewReconciliationCursorKey(candidate.InstallationID, candidate.chatID, candidate.Repository, candidate.PRNumber)
		candidate.ChatIDHash = reviewResourceIdentifierHash(candidate.chatID)
		candidate.CanonicalMessageHash = reviewResourceIdentifierHash(messageID)
		result = append(result, candidate)
	}
	return result, rows.Err()
}

func reviewReconciliationCursorKey(installationID, chatID, repository string, prNumber int) string {
	return stableKey("review-reconciliation", installationID, reviewResourceIdentifierHash(chatID), repository, fmt.Sprintf("%d", prNumber))
}

func (s *ReviewReconciliationScheduler) RunOnce(ctx context.Context, dryRun bool) (ReviewReconciliationRunResult, error) {
	if s == nil || s.Store == nil {
		return ReviewReconciliationRunResult{}, fmt.Errorf("Review reconciliation store is required")
	}
	interval := s.Interval
	if interval == 0 {
		interval = 20 * time.Minute
	}
	if err := validateReviewReconciliationInterval(interval); err != nil {
		return ReviewReconciliationRunResult{}, err
	}
	now := time.Now().UTC()
	if s.Now != nil {
		now = s.Now().UTC()
	}
	candidates, err := s.Store.ListReviewReconciliationCandidates(ctx)
	result := ReviewReconciliationRunResult{DryRun: dryRun, Candidates: candidates}
	if err != nil || dryRun {
		return result, err
	}
	if s.Queue == nil {
		return result, fmt.Errorf("Review reconciliation queue is required")
	}
	bucket := now.Truncate(interval)
	bucketText := reviewGatewayTimestamp(bucket)
	for _, candidate := range candidates {
		cursor, cursorErr := s.Store.getReviewReconciliationCursor(ctx, candidate.CursorKey)
		if cursorErr != nil && !errors.Is(cursorErr, sql.ErrNoRows) {
			return result, cursorErr
		}
		if cursor.LastEnqueuedAt == bucketText {
			result.Duplicates++
			continue
		}
		dedupeKey := strings.Join([]string{
			"review:reconciliation", candidate.InstallationID, reviewResourceIdentifierHash(candidate.chatID),
			candidate.Repository, fmt.Sprintf("%d", candidate.PRNumber), bucketText,
		}, ":")
		job := ReviewGatewayJob{
			SchemaVersion: reviewGatewayJobSchema, JobID: stableKey("review-reconciliation-job", dedupeKey),
			DedupeKey: dedupeKey, Status: "queued", Mode: "preview", Action: "refresh_review_context",
			InstallationID: candidate.InstallationID, Repository: candidate.Repository,
			PRNumber: candidate.PRNumber, ChatID: candidate.chatID,
			RequestedBy:   "scheduled-reconciliation:" + candidate.InstallationID,
			SourceEventID: candidate.CursorKey, NotifyChat: false,
			NotificationMode: ReviewNotificationCanonicalOnly,
			CreatedAt:        reviewGatewayTimestamp(now), MutatesGitLink: false,
			MaxAttempts: reviewGatewayDefaultMaxAttempts, NextAttemptAt: reviewGatewayTimestamp(now),
		}
		saved := false
		admission := ReviewJobAdmissionResult{}
		var enqueueErr error
		if queue, ok := s.Queue.(ReviewPreparedJobAdmissionEnqueuer); ok {
			admission, enqueueErr = queue.EnqueuePreparedJobWithAdmission(ctx, job, "reconciliation")
			saved = admission.Created
		} else {
			saved, enqueueErr = s.Queue.EnqueuePreparedJob(ctx, job)
		}
		if enqueueErr != nil {
			_ = s.Store.saveReviewReconciliationCursor(ctx, candidate, cursor, "failed", enqueueErr.Error(), "", now, interval)
			return result, enqueueErr
		}
		if saved {
			result.Enqueued++
		} else if admission.Coalesced || admission.Duplicate {
			result.Duplicates++
		} else {
			result.Skipped++
		}
		if err := s.Store.saveReviewReconciliationCursor(ctx, candidate, cursor, "scheduled", "", bucketText, now, interval); err != nil {
			return result, err
		}
	}
	return result, nil
}

func (s *SQLiteReviewGatewayStore) saveReviewReconciliationCursor(
	ctx context.Context, candidate ReviewReconciliationCandidate, current ReviewReconciliationCursor,
	status, errorSummary, lastEnqueued string, now time.Time, interval time.Duration,
) error {
	if lastEnqueued == "" {
		lastEnqueued = current.LastEnqueuedAt
	}
	_, err := s.db.ExecContext(ctx, `INSERT INTO review_reconciliation_cursors (
		cursor_key, installation_id, chat_id, repository, pr_number, last_checked_at,
		last_enqueued_at, next_check_at, last_source_fingerprint, status, error_summary, updated_at
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	ON CONFLICT(cursor_key) DO UPDATE SET last_checked_at=excluded.last_checked_at,
		last_enqueued_at=excluded.last_enqueued_at, next_check_at=excluded.next_check_at,
		last_source_fingerprint=excluded.last_source_fingerprint, status=excluded.status,
		error_summary=excluded.error_summary, updated_at=excluded.updated_at`,
		candidate.CursorKey, candidate.InstallationID, candidate.chatID, candidate.Repository,
		candidate.PRNumber, reviewGatewayTimestamp(now), lastEnqueued,
		reviewGatewayTimestamp(now.Add(interval)), candidate.SourceFingerprint, status,
		redactReviewGatewayError(errorSummary), reviewGatewayTimestamp(now))
	return err
}

func (s *SQLiteReviewGatewayStore) getReviewReconciliationCursor(ctx context.Context, key string) (ReviewReconciliationCursor, error) {
	var cursor ReviewReconciliationCursor
	err := s.db.QueryRowContext(ctx, `SELECT cursor_key, installation_id, chat_id, repository,
		pr_number, last_checked_at, last_enqueued_at, next_check_at, last_source_fingerprint,
		status, error_summary, updated_at FROM review_reconciliation_cursors WHERE cursor_key=?`, key).
		Scan(&cursor.CursorKey, &cursor.InstallationID, &cursor.chatID, &cursor.Repository,
			&cursor.PRNumber, &cursor.LastCheckedAt, &cursor.LastEnqueuedAt, &cursor.NextCheckAt,
			&cursor.LastSourceFingerprint, &cursor.Status, &cursor.ErrorSummary, &cursor.UpdatedAt)
	cursor.ChatIDHash = reviewResourceIdentifierHash(cursor.chatID)
	return cursor, err
}

func (s *SQLiteReviewGatewayStore) ListReviewReconciliationCursors(ctx context.Context) ([]ReviewReconciliationCursor, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT cursor_key, installation_id, chat_id, repository,
		pr_number, last_checked_at, last_enqueued_at, next_check_at, last_source_fingerprint,
		status, error_summary, updated_at FROM review_reconciliation_cursors ORDER BY cursor_key`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []ReviewReconciliationCursor{}
	for rows.Next() {
		var cursor ReviewReconciliationCursor
		if err := rows.Scan(&cursor.CursorKey, &cursor.InstallationID, &cursor.chatID,
			&cursor.Repository, &cursor.PRNumber, &cursor.LastCheckedAt, &cursor.LastEnqueuedAt,
			&cursor.NextCheckAt, &cursor.LastSourceFingerprint, &cursor.Status,
			&cursor.ErrorSummary, &cursor.UpdatedAt); err != nil {
			return nil, err
		}
		cursor.ChatIDHash = reviewResourceIdentifierHash(cursor.chatID)
		result = append(result, cursor)
	}
	return result, rows.Err()
}

func (s *ReviewReconciliationScheduler) Run(ctx context.Context) {
	if s == nil || s.Interval == 0 || validateReviewReconciliationInterval(s.Interval) != nil {
		return
	}
	ticker := time.NewTicker(s.Interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			_, _ = s.RunOnce(ctx, false)
		}
	}
}
