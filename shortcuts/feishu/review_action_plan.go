package feishu

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"
)

const (
	reviewActionPlanSchema  = "feishu.review-action-plan/v1"
	reviewActionCommon      = "review_common"
	reviewActionApprove     = "review_approve"
	reviewActionReject      = "review_reject"
	reviewActionRejectClose = "reject_close"
	reviewActionMerge       = "merge"
)

type ReviewActionPlan struct {
	SchemaVersion     string `json:"schema_version"`
	PlanID            string `json:"plan_id"`
	RequestID         string `json:"request_id"`
	IdempotencyKey    string `json:"idempotency_key"`
	SourceJobID       string `json:"source_job_id"`
	SourceChatID      string `json:"source_chat_id"`
	ActorID           string `json:"actor_id"`
	GitLinkLogin      string `json:"gitlink_login"`
	Repository        string `json:"repository"`
	PRNumber          int    `json:"pr_number"`
	Action            string `json:"action"`
	ReviewStatus      string `json:"review_status,omitempty"`
	Content           string `json:"content,omitempty"`
	ExpectedHeadSHA   string `json:"expected_head_sha"`
	SourceFingerprint string `json:"source_fingerprint"`
	Status            string `json:"status"`
	ReviewID          string `json:"review_id,omitempty"`
	MutationStatus    string `json:"mutation_status"`
	Reconciliation    string `json:"reconciliation_status"`
	AttemptCount      int    `json:"attempt_count"`
	LeaseOwner        string `json:"lease_owner,omitempty"`
	LeaseExpiresAt    string `json:"lease_expires_at,omitempty"`
	Error             string `json:"error,omitempty"`
	CreatedAt         string `json:"created_at"`
	ExpiresAt         string `json:"expires_at"`
	UpdatedAt         string `json:"updated_at"`
}

func NewReviewActionPlan(job ReviewGatewayJob, data ReviewData, action, content string, now time.Time) (ReviewActionPlan, error) {
	action = strings.TrimSpace(action)
	if normalizeReviewAction(action) == "" {
		return ReviewActionPlan{}, fmt.Errorf("unsupported review action %q", action)
	}
	if job.Repository == "" || job.PRNumber <= 0 || job.RequestedBy == "" || job.GitLinkLogin == "" || data.HeadSHA == "" || data.SourceFingerprint == "" {
		return ReviewActionPlan{}, fmt.Errorf("review action plan requires actor, repository, PR, head, fingerprint, and identity")
	}
	if action != reviewActionMerge && strings.TrimSpace(content) == "" {
		return ReviewActionPlan{}, fmt.Errorf("review content or reason is required")
	}
	idempotency := strings.Join([]string{job.ChatID, job.RequestedBy, job.GitLinkLogin, job.Repository, fmt.Sprint(job.PRNumber), action, data.HeadSHA, data.SourceFingerprint, strings.TrimSpace(content)}, "\x00")
	digest := sha256.Sum256([]byte(idempotency))
	planID := "review-plan-" + hex.EncodeToString(digest[:8])
	requestID := "RW-" + strings.ToUpper(hex.EncodeToString(digest[:3]))
	return ReviewActionPlan{
		SchemaVersion: reviewActionPlanSchema, PlanID: planID, RequestID: requestID,
		IdempotencyKey: "sha256:" + hex.EncodeToString(digest[:]), SourceJobID: job.JobID, SourceChatID: job.ChatID,
		ActorID: job.RequestedBy, GitLinkLogin: job.GitLinkLogin, Repository: job.Repository, PRNumber: job.PRNumber,
		Action: action, ReviewStatus: reviewStatusForAction(action), Content: strings.TrimSpace(content),
		ExpectedHeadSHA: data.HeadSHA, SourceFingerprint: data.SourceFingerprint,
		Status: "pending_confirmation", MutationStatus: "none", Reconciliation: "not_required",
		CreatedAt: now.UTC().Format(time.RFC3339Nano), ExpiresAt: now.Add(30 * time.Minute).UTC().Format(time.RFC3339Nano), UpdatedAt: now.UTC().Format(time.RFC3339Nano),
	}, nil
}

func normalizeReviewAction(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case reviewActionCommon:
		return reviewActionCommon
	case reviewActionApprove:
		return reviewActionApprove
	case reviewActionReject:
		return reviewActionReject
	case reviewActionRejectClose:
		return reviewActionRejectClose
	case reviewActionMerge:
		return reviewActionMerge
	default:
		return ""
	}
}

func reviewStatusForAction(action string) string {
	switch action {
	case reviewActionCommon:
		return "common"
	case reviewActionApprove:
		return "approved"
	case reviewActionReject:
		return "rejected"
	default:
		return ""
	}
}

func (s *SQLiteReviewGatewayStore) CreateReviewActionPlan(ctx context.Context, plan ReviewActionPlan) (ReviewActionPlan, error) {
	if plan.SchemaVersion != reviewActionPlanSchema || plan.PlanID == "" || plan.IdempotencyKey == "" {
		return ReviewActionPlan{}, fmt.Errorf("invalid review action plan")
	}
	_, err := s.db.ExecContext(ctx, `INSERT OR IGNORE INTO review_action_plans (
		plan_id, request_id, idempotency_key, source_job_id, source_chat_id, actor_id, gitlink_login,
		repository, pr_number, action, review_status, content, expected_head_sha, source_fingerprint,
		status, review_id, mutation_status, reconciliation_status, attempt_count,
		lease_owner, lease_expires_at, error_summary, created_at, expires_at, updated_at
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, '', 'none', 'not_required', 0, '', '', '', ?, ?, ?)`,
		plan.PlanID, plan.RequestID, plan.IdempotencyKey, plan.SourceJobID, plan.SourceChatID, plan.ActorID, plan.GitLinkLogin,
		plan.Repository, plan.PRNumber, plan.Action, plan.ReviewStatus, plan.Content, plan.ExpectedHeadSHA, plan.SourceFingerprint,
		plan.Status, plan.CreatedAt, plan.ExpiresAt, plan.UpdatedAt)
	if err != nil {
		return ReviewActionPlan{}, err
	}
	return s.GetReviewActionPlan(ctx, plan.PlanID)
}

func (s *SQLiteReviewGatewayStore) GetReviewActionPlan(ctx context.Context, planID string) (ReviewActionPlan, error) {
	return scanReviewActionPlan(s.db.QueryRowContext(ctx, `SELECT plan_id, request_id, idempotency_key,
		source_job_id, source_chat_id, actor_id, gitlink_login, repository, pr_number, action, review_status,
		content, expected_head_sha, source_fingerprint, status, review_id, mutation_status,
		reconciliation_status, attempt_count, lease_owner, lease_expires_at, error_summary,
		created_at, expires_at, updated_at FROM review_action_plans WHERE plan_id=?`, strings.TrimSpace(planID)))
}

func (s *SQLiteReviewGatewayStore) AcquireReviewActionPlan(ctx context.Context, planID, owner string, now time.Time, duration time.Duration) (ReviewActionPlan, error) {
	if owner == "" {
		return ReviewActionPlan{}, fmt.Errorf("review action lease owner is required")
	}
	if duration <= 0 {
		duration = time.Minute
	}
	result, err := s.db.ExecContext(ctx, `UPDATE review_action_plans SET status='executing',
		lease_owner=?, lease_expires_at=?, attempt_count=attempt_count+1, updated_at=?
		WHERE plan_id=? AND status='pending_confirmation' AND expires_at>?`, owner,
		reviewGatewayTimestamp(now.Add(duration)), reviewGatewayTimestamp(now), planID, reviewGatewayTimestamp(now))
	if err := requireReviewGatewayJobUpdate(result, err, planID, "acquire review action plan"); err != nil {
		return ReviewActionPlan{}, err
	}
	return s.GetReviewActionPlan(ctx, planID)
}

func (s *SQLiteReviewGatewayStore) CompleteReviewActionPlan(ctx context.Context, planID, owner, reviewID string, now time.Time) error {
	result, err := s.db.ExecContext(ctx, `UPDATE review_action_plans SET status='completed', review_id=?,
		mutation_status='confirmed', reconciliation_status='verified', lease_owner='', lease_expires_at='', updated_at=?
		WHERE plan_id=? AND status='executing' AND lease_owner=?`, reviewID, reviewGatewayTimestamp(now), planID, owner)
	return requireReviewGatewayJobUpdate(result, err, planID, "complete review action plan")
}

func (s *SQLiteReviewGatewayStore) MarkReviewActionPlanUnknown(ctx context.Context, planID, owner string, failure error, now time.Time) error {
	result, err := s.db.ExecContext(ctx, `UPDATE review_action_plans SET status='unknown_needs_reconciliation',
		mutation_status='possible', reconciliation_status='required', error_summary=?, lease_owner='', lease_expires_at='', updated_at=?
		WHERE plan_id=? AND status='executing' AND lease_owner=?`, redactReviewGatewayError(failure.Error()), reviewGatewayTimestamp(now), planID, owner)
	return requireReviewGatewayJobUpdate(result, err, planID, "mark review action plan unknown")
}

func (s *SQLiteReviewGatewayStore) FailReviewActionPlan(ctx context.Context, planID, owner string, failure error, now time.Time) error {
	result, err := s.db.ExecContext(ctx, `UPDATE review_action_plans SET status='failed', mutation_status='none',
		reconciliation_status='not_required', error_summary=?, lease_owner='', lease_expires_at='', updated_at=?
		WHERE plan_id=? AND status='executing' AND lease_owner=?`, redactReviewGatewayError(failure.Error()), reviewGatewayTimestamp(now), planID, owner)
	return requireReviewGatewayJobUpdate(result, err, planID, "fail review action plan")
}

func (s *SQLiteReviewGatewayStore) CancelReviewActionPlan(ctx context.Context, planID, actorID string, now time.Time) error {
	result, err := s.db.ExecContext(ctx, `UPDATE review_action_plans SET status='cancelled', updated_at=?
		WHERE plan_id=? AND actor_id=? AND status='pending_confirmation'`, reviewGatewayTimestamp(now), planID, actorID)
	return requireReviewGatewayJobUpdate(result, err, planID, "cancel review action plan")
}

func scanReviewActionPlan(scanner interface{ Scan(...interface{}) error }) (ReviewActionPlan, error) {
	plan := ReviewActionPlan{SchemaVersion: reviewActionPlanSchema}
	err := scanner.Scan(&plan.PlanID, &plan.RequestID, &plan.IdempotencyKey, &plan.SourceJobID,
		&plan.SourceChatID, &plan.ActorID, &plan.GitLinkLogin, &plan.Repository, &plan.PRNumber,
		&plan.Action, &plan.ReviewStatus, &plan.Content, &plan.ExpectedHeadSHA, &plan.SourceFingerprint,
		&plan.Status, &plan.ReviewID, &plan.MutationStatus, &plan.Reconciliation, &plan.AttemptCount,
		&plan.LeaseOwner, &plan.LeaseExpiresAt, &plan.Error, &plan.CreatedAt, &plan.ExpiresAt, &plan.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return ReviewActionPlan{}, fmt.Errorf("review action plan %q not found", plan.PlanID)
	}
	return plan, err
}
