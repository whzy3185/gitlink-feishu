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

const reviewActionPlanSchema = "review.action-plan/v1"

type ReviewActionPlan struct {
	SchemaVersion     string `json:"schema_version"`
	PlanID            string `json:"plan_id"`
	Repository        string `json:"repository"`
	PRNumber          int    `json:"pr_number"`
	ActorID           string `json:"actor_id"`
	GitLinkLogin      string `json:"gitlink_login"`
	ExpectedHeadSHA   string `json:"expected_head_sha"`
	SourceFingerprint string `json:"source_fingerprint"`
	ReviewStatus      string `json:"review_status"`
	Content           string `json:"content"`
	Status            string `json:"status"`
	IdempotencyKey    string `json:"idempotency_key"`
	SourceJobID       string `json:"source_job_id"`
	ReviewID          string `json:"review_id,omitempty"`
	Error             string `json:"error,omitempty"`
	CreatedAt         string `json:"created_at"`
	ExpiresAt         string `json:"expires_at"`
	UpdatedAt         string `json:"updated_at"`
}

type ReviewWriteResult struct {
	PlanID         string `json:"plan_id"`
	Status         string `json:"status"`
	ReviewID       string `json:"review_id,omitempty"`
	Repository     string `json:"repository"`
	PRNumber       int    `json:"pr_number"`
	HeadSHA        string `json:"head_sha"`
	ReviewStatus   string `json:"review_status"`
	Mutated        bool   `json:"mutated"`
	Reconciliation string `json:"reconciliation,omitempty"`
}

type ReviewActionPlanStore interface {
	CreateReviewActionPlan(context.Context, ReviewActionPlan) (ReviewActionPlan, error)
	GetReviewActionPlan(context.Context, string) (ReviewActionPlan, error)
	ClaimReviewActionPlan(context.Context, string, string, time.Time) (ReviewActionPlan, error)
	FinishReviewActionPlan(context.Context, string, string, string, string, time.Time) error
}

func NewReviewActionPlan(
	job ReviewGatewayJob,
	gitLinkLogin,
	headSHA,
	sourceFingerprint,
	content string,
	now time.Time,
) ReviewActionPlan {
	now = now.UTC()
	idempotencySeed := strings.Join([]string{
		job.Repository,
		fmt.Sprintf("%d", job.PRNumber),
		job.RequestedBy,
		gitLinkLogin,
		headSHA,
		sourceFingerprint,
		"common",
		content,
	}, "\x00")
	digest := sha256.Sum256([]byte(idempotencySeed))
	key := hex.EncodeToString(digest[:16])
	return ReviewActionPlan{
		SchemaVersion:     reviewActionPlanSchema,
		PlanID:            "review-plan-" + key[:16],
		Repository:        job.Repository,
		PRNumber:          job.PRNumber,
		ActorID:           job.RequestedBy,
		GitLinkLogin:      gitLinkLogin,
		ExpectedHeadSHA:   headSHA,
		SourceFingerprint: sourceFingerprint,
		ReviewStatus:      "common",
		Content:           strings.TrimSpace(content),
		Status:            "pending_confirmation",
		IdempotencyKey:    key,
		SourceJobID:       job.JobID,
		CreatedAt:         now.Format(time.RFC3339Nano),
		ExpiresAt:         now.Add(15 * time.Minute).Format(time.RFC3339Nano),
		UpdatedAt:         now.Format(time.RFC3339Nano),
	}
}

func (s *SQLiteReviewGatewayStore) CreateReviewActionPlan(
	ctx context.Context,
	plan ReviewActionPlan,
) (ReviewActionPlan, error) {
	if plan.ReviewStatus != "common" {
		return ReviewActionPlan{}, fmt.Errorf("only common Review action plans are allowed")
	}
	if plan.ExpectedHeadSHA == "" || plan.Content == "" || plan.ActorID == "" || plan.GitLinkLogin == "" {
		return ReviewActionPlan{}, fmt.Errorf("review action plan identity, head, and content are required")
	}
	_, err := s.db.ExecContext(ctx, `INSERT INTO review_action_plans (
		plan_id, repository, pr_number, actor_id, gitlink_login,
		expected_head_sha, source_fingerprint, review_status, content, status,
		idempotency_key, source_job_id, created_at, expires_at, updated_at
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	ON CONFLICT(idempotency_key) DO NOTHING`,
		plan.PlanID,
		plan.Repository,
		plan.PRNumber,
		plan.ActorID,
		plan.GitLinkLogin,
		plan.ExpectedHeadSHA,
		plan.SourceFingerprint,
		plan.ReviewStatus,
		plan.Content,
		plan.Status,
		plan.IdempotencyKey,
		plan.SourceJobID,
		plan.CreatedAt,
		plan.ExpiresAt,
		plan.UpdatedAt,
	)
	if err != nil {
		return ReviewActionPlan{}, err
	}
	return s.getReviewActionPlanByIdempotencyKey(ctx, plan.IdempotencyKey)
}

func (s *SQLiteReviewGatewayStore) GetReviewActionPlan(ctx context.Context, planID string) (ReviewActionPlan, error) {
	return scanReviewActionPlan(s.db.QueryRowContext(ctx, `SELECT
		plan_id, repository, pr_number, actor_id, gitlink_login,
		expected_head_sha, source_fingerprint, review_status, content, status,
		idempotency_key, source_job_id, review_id, error_summary,
		created_at, expires_at, updated_at
		FROM review_action_plans WHERE plan_id = ?`, strings.TrimSpace(planID)))
}

func (s *SQLiteReviewGatewayStore) getReviewActionPlanByIdempotencyKey(
	ctx context.Context,
	key string,
) (ReviewActionPlan, error) {
	return scanReviewActionPlan(s.db.QueryRowContext(ctx, `SELECT
		plan_id, repository, pr_number, actor_id, gitlink_login,
		expected_head_sha, source_fingerprint, review_status, content, status,
		idempotency_key, source_job_id, review_id, error_summary,
		created_at, expires_at, updated_at
		FROM review_action_plans WHERE idempotency_key = ?`, key))
}

func (s *SQLiteReviewGatewayStore) ClaimReviewActionPlan(
	ctx context.Context,
	planID,
	actorID string,
	now time.Time,
) (ReviewActionPlan, error) {
	now = now.UTC()
	result, err := s.db.ExecContext(ctx, `UPDATE review_action_plans
		SET status = 'executing', updated_at = ?
		WHERE plan_id = ? AND actor_id = ? AND status = 'pending_confirmation' AND expires_at > ?`,
		now.Format(time.RFC3339Nano),
		strings.TrimSpace(planID),
		strings.TrimSpace(actorID),
		now.Format(time.RFC3339Nano),
	)
	if err != nil {
		return ReviewActionPlan{}, err
	}
	affected, _ := result.RowsAffected()
	if affected != 1 {
		plan, getErr := s.GetReviewActionPlan(ctx, planID)
		if errors.Is(getErr, sql.ErrNoRows) {
			return ReviewActionPlan{}, fmt.Errorf("review action plan not found")
		}
		if getErr != nil {
			return ReviewActionPlan{}, getErr
		}
		switch {
		case plan.ActorID != actorID:
			return ReviewActionPlan{}, fmt.Errorf("review action plan belongs to another Feishu user")
		case plan.Status == "completed":
			return plan, fmt.Errorf("review action plan is already completed")
		case plan.Status != "pending_confirmation":
			return plan, fmt.Errorf("review action plan status is %s", plan.Status)
		default:
			return plan, fmt.Errorf("review action plan expired")
		}
	}
	return s.GetReviewActionPlan(ctx, planID)
}

func (s *SQLiteReviewGatewayStore) FinishReviewActionPlan(
	ctx context.Context,
	planID,
	status,
	reviewID,
	errorSummary string,
	now time.Time,
) error {
	switch status {
	case "completed", "stale", "failed", "unknown":
	default:
		return fmt.Errorf("unsupported review action plan terminal status %q", status)
	}
	result, err := s.db.ExecContext(ctx, `UPDATE review_action_plans
		SET status = ?, review_id = ?, error_summary = ?, updated_at = ?
		WHERE plan_id = ? AND status = 'executing'`,
		status,
		reviewID,
		redactReviewGatewayError(errorSummary),
		now.UTC().Format(time.RFC3339Nano),
		planID,
	)
	if err != nil {
		return err
	}
	affected, _ := result.RowsAffected()
	if affected != 1 {
		return fmt.Errorf("review action plan %q is no longer executing", planID)
	}
	return nil
}

func scanReviewActionPlan(scanner reviewCollaborationScanner) (ReviewActionPlan, error) {
	plan := ReviewActionPlan{SchemaVersion: reviewActionPlanSchema}
	err := scanner.Scan(
		&plan.PlanID,
		&plan.Repository,
		&plan.PRNumber,
		&plan.ActorID,
		&plan.GitLinkLogin,
		&plan.ExpectedHeadSHA,
		&plan.SourceFingerprint,
		&plan.ReviewStatus,
		&plan.Content,
		&plan.Status,
		&plan.IdempotencyKey,
		&plan.SourceJobID,
		&plan.ReviewID,
		&plan.Error,
		&plan.CreatedAt,
		&plan.ExpiresAt,
		&plan.UpdatedAt,
	)
	return plan, err
}

func findReviewIdentity(bindings []ReviewIdentityBinding, feishuUserID string) (ReviewIdentityBinding, bool) {
	for _, binding := range bindings {
		if binding.Enabled && strings.TrimSpace(binding.FeishuUserID) == strings.TrimSpace(feishuUserID) {
			return binding, strings.TrimSpace(binding.GitLinkLogin) != ""
		}
	}
	return ReviewIdentityBinding{}, false
}
