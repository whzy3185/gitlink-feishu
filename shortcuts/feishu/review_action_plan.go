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
	LeaseOwner        string `json:"lease_owner,omitempty"`
	LeaseExpiresAt    string `json:"lease_expires_at,omitempty"`
	AttemptCount      int    `json:"attempt_count"`
	MaxAttempts       int    `json:"max_attempts"`
	Reconciliation    string `json:"reconciliation_status"`
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
	ClaimReviewActionPlan(context.Context, ReviewActionPlanClaimOptions) (ReviewActionPlan, error)
	MarkReviewActionPlanWriteStarted(context.Context, string, string, time.Time) error
	FinishReviewActionPlan(context.Context, string, string, string, string, time.Time) error
	MarkReviewActionPlanUnknown(context.Context, string, string, time.Time) error
}

type ReviewActionPlanClaimOptions struct {
	PlanID        string
	ActorID       string
	LeaseOwner    string
	Now           time.Time
	LeaseDuration time.Duration
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
		MaxAttempts:       3,
		Reconciliation:    "not_required",
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
	if plan.ExpectedHeadSHA == "" || plan.SourceFingerprint == "" ||
		plan.Content == "" || plan.ActorID == "" || plan.GitLinkLogin == "" {
		return ReviewActionPlan{}, fmt.Errorf("review action plan identity, head, fingerprint, and content are required")
	}
	if plan.MaxAttempts <= 0 {
		plan.MaxAttempts = 3
	}
	if plan.Reconciliation == "" {
		plan.Reconciliation = "not_required"
	}
	_, err := s.db.ExecContext(ctx, `INSERT INTO review_action_plans (
		plan_id, repository, pr_number, actor_id, gitlink_login,
		expected_head_sha, source_fingerprint, review_status, content, status,
		idempotency_key, source_job_id, max_attempts, reconciliation_status,
		created_at, expires_at, updated_at
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
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
		plan.MaxAttempts,
		plan.Reconciliation,
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
		lease_owner, lease_expires_at, attempt_count, max_attempts, reconciliation_status,
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
		lease_owner, lease_expires_at, attempt_count, max_attempts, reconciliation_status,
		created_at, expires_at, updated_at
		FROM review_action_plans WHERE idempotency_key = ?`, key))
}

func (s *SQLiteReviewGatewayStore) ClaimReviewActionPlan(
	ctx context.Context,
	opts ReviewActionPlanClaimOptions,
) (ReviewActionPlan, error) {
	opts.Now = opts.Now.UTC()
	if opts.LeaseDuration <= 0 {
		opts.LeaseDuration = 2 * time.Minute
	}
	if strings.TrimSpace(opts.LeaseOwner) == "" {
		return ReviewActionPlan{}, fmt.Errorf("review action plan lease owner is required")
	}
	nowText := opts.Now.Format(time.RFC3339Nano)
	result, err := s.db.ExecContext(ctx, `UPDATE review_action_plans
		SET status = 'executing',
			lease_owner = ?,
			lease_expires_at = ?,
			attempt_count = attempt_count + 1,
			reconciliation_status = 'pre_write',
			updated_at = ?
		WHERE plan_id = ? AND actor_id = ? AND expires_at > ?
			AND attempt_count < max_attempts
			AND (
				status = 'pending_confirmation'
				OR (
					status = 'executing'
					AND reconciliation_status = 'pre_write'
					AND lease_expires_at != ''
					AND lease_expires_at <= ?
				)
			)`,
		opts.LeaseOwner,
		opts.Now.Add(opts.LeaseDuration).Format(time.RFC3339Nano),
		nowText,
		strings.TrimSpace(opts.PlanID),
		strings.TrimSpace(opts.ActorID),
		nowText,
		nowText,
	)
	if err != nil {
		return ReviewActionPlan{}, err
	}
	affected, _ := result.RowsAffected()
	if affected != 1 {
		plan, getErr := s.GetReviewActionPlan(ctx, opts.PlanID)
		if errors.Is(getErr, sql.ErrNoRows) {
			return ReviewActionPlan{}, fmt.Errorf("review action plan not found")
		}
		if getErr != nil {
			return ReviewActionPlan{}, getErr
		}
		switch {
		case plan.ActorID != opts.ActorID:
			return ReviewActionPlan{}, fmt.Errorf("review action plan belongs to another Feishu user")
		case plan.Status == "completed":
			return plan, fmt.Errorf("review action plan is already completed")
		case plan.Reconciliation == "remote_write_possible" || plan.Reconciliation == "required":
			return plan, fmt.Errorf("review action plan requires remote reconciliation before any retry")
		case plan.AttemptCount >= plan.MaxAttempts:
			return plan, fmt.Errorf("review action plan exhausted its execution attempts")
		case plan.Status != "pending_confirmation" && plan.Status != "executing":
			return plan, fmt.Errorf("review action plan status is %s", plan.Status)
		case plan.Status == "executing":
			return plan, fmt.Errorf("review action plan execution lease is still active")
		default:
			return plan, fmt.Errorf("review action plan expired")
		}
	}
	return s.GetReviewActionPlan(ctx, opts.PlanID)
}

func (s *SQLiteReviewGatewayStore) MarkReviewActionPlanWriteStarted(
	ctx context.Context,
	planID,
	leaseOwner string,
	now time.Time,
) error {
	result, err := s.db.ExecContext(ctx, `UPDATE review_action_plans
		SET reconciliation_status = 'remote_write_possible', updated_at = ?
		WHERE plan_id = ? AND status = 'executing' AND lease_owner = ?
			AND lease_expires_at > ?`,
		now.UTC().Format(time.RFC3339Nano),
		strings.TrimSpace(planID),
		strings.TrimSpace(leaseOwner),
		now.UTC().Format(time.RFC3339Nano),
	)
	if err != nil {
		return err
	}
	affected, _ := result.RowsAffected()
	if affected != 1 {
		return fmt.Errorf("review action plan %q no longer owns the execution lease", planID)
	}
	return nil
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
	reconciliation := "not_required"
	if status == "completed" {
		reconciliation = "verified"
	} else if status == "unknown" {
		reconciliation = "required"
	}
	result, err := s.db.ExecContext(ctx, `UPDATE review_action_plans
		SET status = ?, review_id = ?, error_summary = ?,
			lease_owner = '', lease_expires_at = '',
			reconciliation_status = ?, updated_at = ?
		WHERE plan_id = ? AND status = 'executing'`,
		status,
		reviewID,
		redactReviewGatewayError(errorSummary),
		reconciliation,
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

func (s *SQLiteReviewGatewayStore) MarkReviewActionPlanUnknown(
	ctx context.Context,
	planID,
	errorSummary string,
	now time.Time,
) error {
	result, err := s.db.ExecContext(ctx, `UPDATE review_action_plans
		SET status = 'unknown', error_summary = ?,
			lease_owner = '', lease_expires_at = '',
			reconciliation_status = 'required', updated_at = ?
		WHERE plan_id = ? AND status = 'executing'`,
		redactReviewGatewayError(errorSummary),
		now.UTC().Format(time.RFC3339Nano),
		strings.TrimSpace(planID),
	)
	if err != nil {
		return err
	}
	affected, _ := result.RowsAffected()
	if affected != 1 {
		return fmt.Errorf("review action plan %q could not enter reconciliation state", planID)
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
		&plan.LeaseOwner,
		&plan.LeaseExpiresAt,
		&plan.AttemptCount,
		&plan.MaxAttempts,
		&plan.Reconciliation,
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
