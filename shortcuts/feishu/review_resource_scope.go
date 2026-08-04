package feishu

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"
)

const (
	ReviewResourceCard    = "feishu_card"
	ReviewResourceBitable = "feishu_bitable"
	ReviewResourceDoc     = "feishu_doc"
	ReviewResourceTask    = "feishu_task"
)

type ReviewResourceScope string

const (
	ReviewResourceScopeChat         ReviewResourceScope = "chat"
	ReviewResourceScopeInstallation ReviewResourceScope = "installation"
	ReviewResourceScopeDisabled     ReviewResourceScope = "disabled"
)

var (
	ErrReviewResourceProjectionDisabled = errors.New("review resource projection is disabled")
	ErrReviewResourcePolicyRequired     = errors.New("explicit review resource migration policy is required")
	ErrReviewResourcePolicyConflict     = errors.New("review resource policy changed concurrently")
)

type ReviewResourceScopePolicy struct {
	InstallationID   string              `json:"installation_id"`
	ResourceType     string              `json:"resource_type"`
	TargetScope      ReviewResourceScope `json:"target_scope"`
	MigrationEnabled bool                `json:"migration_enabled"`
	Revision         int                 `json:"revision"`
	UpdatedBy        string              `json:"updated_by,omitempty"`
	UpdatedAt        string              `json:"updated_at"`
}

func ResolveReviewResourceTarget(
	resourceType, installationID, chatID, repository string,
	prNumber int,
	policy ReviewResourceScope,
) (string, error) {
	resourceType = strings.TrimSpace(resourceType)
	installationID = firstNonEmpty(strings.TrimSpace(installationID), "legacy")
	chatID = strings.TrimSpace(chatID)
	repository = strings.TrimSpace(repository)
	if repository == "" || prNumber <= 0 {
		return "", fmt.Errorf("review resource repository and PR number are required")
	}
	switch resourceType {
	case ReviewResourceCard, ReviewResourceTask:
		policy = ReviewResourceScopeChat
	case ReviewResourceBitable, ReviewResourceDoc:
		if policy == "" {
			return "", ErrReviewResourcePolicyRequired
		}
	default:
		return "", fmt.Errorf("unsupported review resource type %q", resourceType)
	}
	switch policy {
	case ReviewResourceScopeChat:
		if chatID == "" {
			return "", fmt.Errorf("chat-scoped review resource requires chat ID")
		}
		return stableKey("review-work-item", reviewCollaborationScopeKey(
			installationID, chatID, repository, prNumber,
		)), nil
	case ReviewResourceScopeInstallation:
		if resourceType == ReviewResourceCard || resourceType == ReviewResourceTask {
			return "", fmt.Errorf("%s is always chat scoped", resourceType)
		}
		return stableKey("review-work-item", reviewSnapshotScopeKey(
			installationID, repository, prNumber,
		)), nil
	case ReviewResourceScopeDisabled:
		return "", ErrReviewResourceProjectionDisabled
	default:
		return "", fmt.Errorf("unsupported review resource scope %q", policy)
	}
}

func RuntimeReviewResourceScope(resourceType string, configured ReviewResourceScope) (ReviewResourceScope, error) {
	switch strings.TrimSpace(resourceType) {
	case ReviewResourceCard, ReviewResourceTask:
		return ReviewResourceScopeChat, nil
	case ReviewResourceBitable, ReviewResourceDoc:
		if configured == "" {
			return ReviewResourceScopeChat, nil
		}
		if configured == ReviewResourceScopeChat || configured == ReviewResourceScopeInstallation || configured == ReviewResourceScopeDisabled {
			return configured, nil
		}
		return "", fmt.Errorf("unsupported review resource scope %q", configured)
	default:
		return "", fmt.Errorf("unsupported review resource type %q", resourceType)
	}
}

func (s *SQLiteReviewGatewayStore) SetReviewResourceScopePolicy(
	ctx context.Context,
	policy ReviewResourceScopePolicy,
	now time.Time,
) (ReviewResourceScopePolicy, error) {
	if s == nil || s.db == nil {
		return ReviewResourceScopePolicy{}, fmt.Errorf("review resource policy store is unavailable")
	}
	policy.InstallationID = strings.TrimSpace(policy.InstallationID)
	policy.ResourceType = strings.TrimSpace(policy.ResourceType)
	policy.UpdatedBy = strings.TrimSpace(policy.UpdatedBy)
	if policy.InstallationID == "" {
		return ReviewResourceScopePolicy{}, fmt.Errorf("review resource policy installation is required")
	}
	if policy.ResourceType != ReviewResourceBitable && policy.ResourceType != ReviewResourceDoc {
		return ReviewResourceScopePolicy{}, fmt.Errorf("policies are only configurable for Base and Doc")
	}
	if policy.TargetScope != ReviewResourceScopeChat && policy.TargetScope != ReviewResourceScopeInstallation && policy.TargetScope != ReviewResourceScopeDisabled {
		return ReviewResourceScopePolicy{}, fmt.Errorf("invalid review resource policy scope %q", policy.TargetScope)
	}
	nowText := now.UTC().Format(time.RFC3339Nano)
	if policy.Revision > 0 {
		result, err := s.db.ExecContext(ctx, `UPDATE review_resource_scope_policies SET
			target_scope=?, migration_enabled=?, revision=revision+1, updated_by=?, updated_at=?
			WHERE installation_id=? AND resource_type=? AND revision=?`, policy.TargetScope,
			boolToReviewCollaborationInt(policy.MigrationEnabled), policy.UpdatedBy, nowText,
			policy.InstallationID, policy.ResourceType, policy.Revision)
		if err != nil {
			return ReviewResourceScopePolicy{}, err
		}
		affected, err := result.RowsAffected()
		if err != nil || affected != 1 {
			return ReviewResourceScopePolicy{}, ErrReviewResourcePolicyConflict
		}
		return s.GetReviewResourceScopePolicy(ctx, policy.InstallationID, policy.ResourceType)
	}
	_, err := s.db.ExecContext(ctx, `INSERT INTO review_resource_scope_policies (
		installation_id, resource_type, target_scope, migration_enabled,
		revision, updated_by, updated_at
	) VALUES (?, ?, ?, ?, 1, ?, ?)
	ON CONFLICT(installation_id, resource_type) DO UPDATE SET
		target_scope=excluded.target_scope,
		migration_enabled=excluded.migration_enabled,
		revision=review_resource_scope_policies.revision+1,
		updated_by=excluded.updated_by,
		updated_at=excluded.updated_at`,
		policy.InstallationID, policy.ResourceType, policy.TargetScope,
		boolToReviewCollaborationInt(policy.MigrationEnabled), policy.UpdatedBy, nowText,
	)
	if err != nil {
		return ReviewResourceScopePolicy{}, err
	}
	return s.GetReviewResourceScopePolicy(ctx, policy.InstallationID, policy.ResourceType)
}

func (s *SQLiteReviewGatewayStore) GetReviewResourceScopePolicy(
	ctx context.Context,
	installationID, resourceType string,
) (ReviewResourceScopePolicy, error) {
	var policy ReviewResourceScopePolicy
	var enabled int
	err := s.db.QueryRowContext(ctx, `SELECT installation_id, resource_type, target_scope,
		migration_enabled, revision, updated_by, updated_at
		FROM review_resource_scope_policies
		WHERE installation_id=? AND resource_type=?`,
		strings.TrimSpace(installationID), strings.TrimSpace(resourceType),
	).Scan(&policy.InstallationID, &policy.ResourceType, &policy.TargetScope,
		&enabled, &policy.Revision, &policy.UpdatedBy, &policy.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return ReviewResourceScopePolicy{}, nil
	}
	policy.MigrationEnabled = enabled != 0
	return policy, err
}

func (s *SQLiteReviewGatewayStore) ListReviewResourceScopePolicies(ctx context.Context) ([]ReviewResourceScopePolicy, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT installation_id, resource_type, target_scope,
		migration_enabled, revision, updated_by, updated_at
		FROM review_resource_scope_policies ORDER BY installation_id, resource_type`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	policies := []ReviewResourceScopePolicy{}
	for rows.Next() {
		var policy ReviewResourceScopePolicy
		var enabled int
		if err := rows.Scan(&policy.InstallationID, &policy.ResourceType, &policy.TargetScope,
			&enabled, &policy.Revision, &policy.UpdatedBy, &policy.UpdatedAt); err != nil {
			return nil, err
		}
		policy.MigrationEnabled = enabled != 0
		policies = append(policies, policy)
	}
	return policies, rows.Err()
}
