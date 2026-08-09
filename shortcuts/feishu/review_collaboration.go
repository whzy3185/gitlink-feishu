package feishu

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/gitlink-org/gitlink-cli/shortcuts/common"
)

const reviewCollaborationSchema = "feishu.review-collaboration/v1"

type ReviewIdentityBinding struct {
	FeishuUserID       string `json:"feishu_user_id"`
	GitLinkLogin       string `json:"gitlink_login"`
	VerificationMethod string `json:"verification_method,omitempty"`
	Enabled            bool   `json:"enabled"`
}

type ReviewCollaborationItem struct {
	SchemaVersion       string `json:"schema_version"`
	Repository          string `json:"repository"`
	PRNumber            int    `json:"pr_number"`
	ChatID              string `json:"chat_id"`
	AssignedTo          string `json:"assigned_to,omitempty"`
	AssignedDisplayName string `json:"assigned_display_name,omitempty"`
	CollaborationStatus string `json:"collaboration_status"`
	DueAt               string `json:"due_at,omitempty"`
	ActionOutcome       string `json:"action_outcome,omitempty"`
	UpdatedBy           string `json:"updated_by,omitempty"`
	UpdatedAt           string `json:"updated_at"`
}

type ReviewCollaborationStore interface {
	ApplyCollaborationAction(context.Context, ReviewGatewayJob, time.Time) (ReviewCollaborationItem, error)
	GetCollaborationItem(context.Context, string, string, int) (ReviewCollaborationItem, error)
}

type ReviewRepositoryCollaborator struct {
	Login string `json:"login"`
	Name  string `json:"name,omitempty"`
}

type ReviewRepositoryCollaboratorReader interface {
	ListRepositoryCollaborators(context.Context, *common.RuntimeContext, string, string) ([]ReviewRepositoryCollaborator, error)
}

type GitLinkReviewRepositoryCollaboratorReader struct{}

func (GitLinkReviewRepositoryCollaboratorReader) ListRepositoryCollaborators(
	ctx context.Context,
	runtime *common.RuntimeContext,
	owner,
	repository string,
) ([]ReviewRepositoryCollaborator, error) {
	if runtime == nil || runtime.Client == nil {
		return nil, fmt.Errorf("GitLink runtime is required")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	envelope, err := runtime.CallAPI("GET", fmt.Sprintf("/v1/%s/%s/collaborators", owner, repository), nil)
	if err != nil {
		return nil, err
	}
	value := normalizeReviewDataJSON(envelope.Data)
	items := reviewDataList(value)
	if object, ok := value.(map[string]interface{}); ok && len(items) == 0 {
		items = reviewDataList(firstReviewDataValueInterface(object, "collaborators", "members", "data"))
	}
	result := make([]ReviewRepositoryCollaborator, 0, len(items))
	for _, item := range items {
		login := firstReviewDataString(item, "login", "username")
		if login == "" {
			continue
		}
		result = append(result, ReviewRepositoryCollaborator{Login: login, Name: firstReviewDataString(item, "name", "full_name")})
	}
	return result, nil
}

func firstReviewDataValueInterface(item map[string]interface{}, keys ...string) interface{} {
	for _, key := range keys {
		if value, ok := item[key]; ok && value != nil {
			return value
		}
	}
	return nil
}

type FeishuDisplayNameResolver interface {
	ResolveFeishuDisplayName(context.Context, string) (string, error)
}

type ReviewFeishuDisplayNameResolver struct {
	Client    OpenAPIClient
	AppID     string
	AppSecret string
}

func (r ReviewFeishuDisplayNameResolver) ResolveFeishuDisplayName(ctx context.Context, userID string) (string, error) {
	token, err := r.Client.TenantAccessToken(ctx, r.AppID, r.AppSecret)
	if err != nil {
		return "", err
	}
	return r.Client.GetUserDisplayName(ctx, token.Value, userID)
}

func (s *SQLiteReviewGatewayStore) ApplyCollaborationAction(ctx context.Context, job ReviewGatewayJob, now time.Time) (ReviewCollaborationItem, error) {
	if s == nil || s.db == nil {
		return ReviewCollaborationItem{}, fmt.Errorf("review collaboration store is required")
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return ReviewCollaborationItem{}, err
	}
	defer tx.Rollback()
	item, err := readReviewCollaborationItem(ctx, tx, job.ChatID, job.Repository, job.PRNumber)
	if err != nil {
		return ReviewCollaborationItem{}, err
	}
	if item.SchemaVersion == "" {
		item = newReviewCollaborationItem(job, now)
	}
	actor := strings.TrimSpace(job.RequestedBy)
	actorName := firstNonEmpty(strings.TrimSpace(job.RequestedDisplayName), strings.TrimSpace(job.GitLinkLogin), "飞书成员")
	switch job.Action {
	case "claim_review":
		switch {
		case item.AssignedTo == "":
			item.AssignedTo = actor
			item.AssignedDisplayName = actorName
			item.CollaborationStatus = "reviewing"
			item.ActionOutcome = "claimed"
		case item.AssignedTo == actor:
			if item.AssignedDisplayName == "" {
				item.AssignedDisplayName = actorName
			}
			item.ActionOutcome = "already_claimed_by_self"
		default:
			return ReviewCollaborationItem{}, fmt.Errorf("PR 当前由 %s 负责", firstNonEmpty(item.AssignedDisplayName, "其他成员"))
		}
	case "release_review":
		switch {
		case item.AssignedTo == "":
			item.ActionOutcome = "already_unassigned"
		case item.AssignedTo != actor && !job.CollaborationAdmin:
			return ReviewCollaborationItem{}, fmt.Errorf("只有当前负责人或 Gateway 管理员可以取消领取")
		default:
			item.AssignedTo = ""
			item.AssignedDisplayName = ""
			item.CollaborationStatus = "unassigned"
			item.ActionOutcome = "released"
		}
	case "set_review_deadline", "clear_review_deadline":
		if item.AssignedTo != "" && item.AssignedTo != actor && !job.CollaborationAdmin {
			return ReviewCollaborationItem{}, fmt.Errorf("只有当前负责人或 Gateway 管理员可以修改审查截止时间")
		}
		if job.Action == "clear_review_deadline" {
			if item.DueAt == "" {
				item.ActionOutcome = "deadline_absent"
			} else {
				item.DueAt = ""
				item.ActionOutcome = "deadline_cleared"
			}
		} else {
			if _, err := time.Parse("2006-01-02", strings.TrimSpace(job.Argument)); err != nil {
				return ReviewCollaborationItem{}, fmt.Errorf("审查截止时间必须使用 YYYY-MM-DD")
			}
			if item.DueAt == "" {
				item.ActionOutcome = "deadline_set"
			} else if item.DueAt == job.Argument {
				item.ActionOutcome = "deadline_unchanged"
			} else {
				item.ActionOutcome = "deadline_updated"
			}
			item.DueAt = job.Argument
		}
	default:
		return ReviewCollaborationItem{}, fmt.Errorf("unsupported collaboration action %q", job.Action)
	}
	item.UpdatedBy = actor
	item.UpdatedAt = now.UTC().Format(time.RFC3339Nano)
	if err := writeReviewCollaborationItem(ctx, tx, item); err != nil {
		return ReviewCollaborationItem{}, err
	}
	if err := tx.Commit(); err != nil {
		return ReviewCollaborationItem{}, err
	}
	return item, nil
}

func (s *SQLiteReviewGatewayStore) GetCollaborationItem(ctx context.Context, chatID, repository string, number int) (ReviewCollaborationItem, error) {
	row := s.db.QueryRowContext(ctx, `SELECT repository, pr_number, chat_id, assigned_to,
		assigned_display_name, collaboration_status, due_at, updated_by, updated_at
		FROM review_collaboration_items WHERE chat_id = ? AND repository = ? AND pr_number = ?`, chatID, repository, number)
	item := ReviewCollaborationItem{SchemaVersion: reviewCollaborationSchema}
	err := row.Scan(&item.Repository, &item.PRNumber, &item.ChatID, &item.AssignedTo,
		&item.AssignedDisplayName, &item.CollaborationStatus, &item.DueAt, &item.UpdatedBy, &item.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return ReviewCollaborationItem{}, nil
	}
	return item, err
}

func readReviewCollaborationItem(ctx context.Context, tx *sql.Tx, chatID, repository string, number int) (ReviewCollaborationItem, error) {
	row := tx.QueryRowContext(ctx, `SELECT repository, pr_number, chat_id, assigned_to,
		assigned_display_name, collaboration_status, due_at, updated_by, updated_at
		FROM review_collaboration_items WHERE chat_id = ? AND repository = ? AND pr_number = ?`, chatID, repository, number)
	item := ReviewCollaborationItem{SchemaVersion: reviewCollaborationSchema}
	err := row.Scan(&item.Repository, &item.PRNumber, &item.ChatID, &item.AssignedTo,
		&item.AssignedDisplayName, &item.CollaborationStatus, &item.DueAt, &item.UpdatedBy, &item.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return ReviewCollaborationItem{}, nil
	}
	return item, err
}

func writeReviewCollaborationItem(ctx context.Context, tx *sql.Tx, item ReviewCollaborationItem) error {
	_, err := tx.ExecContext(ctx, `INSERT INTO review_collaboration_items (
		repository, pr_number, chat_id, assigned_to, assigned_display_name,
		collaboration_status, due_at, updated_by, updated_at
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	ON CONFLICT(chat_id, repository, pr_number) DO UPDATE SET
		assigned_to=excluded.assigned_to,
		assigned_display_name=excluded.assigned_display_name,
		collaboration_status=excluded.collaboration_status,
		due_at=excluded.due_at,
		updated_by=excluded.updated_by,
		updated_at=excluded.updated_at`,
		item.Repository, item.PRNumber, item.ChatID, item.AssignedTo, item.AssignedDisplayName,
		item.CollaborationStatus, item.DueAt, item.UpdatedBy, item.UpdatedAt)
	return err
}

func newReviewCollaborationItem(job ReviewGatewayJob, now time.Time) ReviewCollaborationItem {
	return ReviewCollaborationItem{
		SchemaVersion: reviewCollaborationSchema, Repository: job.Repository, PRNumber: job.PRNumber,
		ChatID: job.ChatID, CollaborationStatus: "unassigned", UpdatedAt: now.UTC().Format(time.RFC3339Nano),
	}
}

func reviewIdentityForUser(bindings []ReviewIdentityBinding, userID string) (ReviewIdentityBinding, bool) {
	for _, binding := range bindings {
		if binding.Enabled && strings.TrimSpace(binding.FeishuUserID) == strings.TrimSpace(userID) && strings.TrimSpace(binding.GitLinkLogin) != "" {
			return binding, true
		}
	}
	return ReviewIdentityBinding{}, false
}

func reviewLoginIsCollaborator(login string, collaborators []ReviewRepositoryCollaborator) bool {
	for _, collaborator := range collaborators {
		if strings.EqualFold(strings.TrimSpace(login), strings.TrimSpace(collaborator.Login)) {
			return true
		}
	}
	return false
}

func marshalReviewCollaboration(item ReviewCollaborationItem) string {
	encoded, _ := json.Marshal(item)
	return string(encoded)
}
