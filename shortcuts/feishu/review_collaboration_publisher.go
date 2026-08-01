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

type ReviewResourceSyncResult struct {
	Resource string `json:"resource"`
	Action   string `json:"action"`
	RemoteID string `json:"remote_id,omitempty"`
	Error    string `json:"error,omitempty"`
}

type ReviewCollaborationPublisher interface {
	Publish(context.Context, ReviewCollaborationBundle) ([]ReviewResourceSyncResult, error)
}

type ReviewCollaborationPublisherConfig struct {
	AppID               string
	AppSecret           string
	BaseAppToken        string
	ReviewTableID       string
	DocumentID          string
	DocumentFolderToken string
	EnableTask          bool
}

type FeishuReviewCollaborationPublisher struct {
	Client OpenAPIClient
	Store  *SQLiteReviewGatewayStore
	Config ReviewCollaborationPublisherConfig
	Now    func() time.Time
}

func (p *FeishuReviewCollaborationPublisher) Publish(
	ctx context.Context,
	bundle ReviewCollaborationBundle,
) ([]ReviewResourceSyncResult, error) {
	if p == nil || p.Store == nil {
		return nil, fmt.Errorf("Feishu Review collaboration publisher store is required")
	}
	if strings.TrimSpace(p.Config.AppID) == "" || strings.TrimSpace(p.Config.AppSecret) == "" {
		return nil, fmt.Errorf("Feishu app credentials are required for resource sync")
	}
	if p.Now == nil {
		p.Now = time.Now
	}
	token, err := p.Client.TenantAccessToken(ctx, p.Config.AppID, p.Config.AppSecret)
	if err != nil {
		return nil, err
	}
	fingerprint := reviewCollaborationBundleFingerprint(bundle)
	results := []ReviewResourceSyncResult{}
	if p.Config.BaseAppToken != "" && p.Config.ReviewTableID != "" {
		result := ReviewResourceSyncResult{Resource: "feishu_bitable"}
		state, stateErr := p.Store.GetReviewResourceState(ctx, bundle.UniqueKey, result.Resource)
		remoteID := state.RemoteID
		switch {
		case stateErr != nil:
			result.Action = "failed"
			result.Error = redactReviewGatewayError(stateErr.Error())
		case state.ContentFingerprint == fingerprint && remoteID != "":
			result.Action = "unchanged"
			result.RemoteID = remoteID
		default:
			if remoteID == "" {
				search, searchErr := p.Client.SearchBitableRecord(
					ctx,
					token.Value,
					p.Config.BaseAppToken,
					p.Config.ReviewTableID,
					bundle.UniqueKey,
				)
				switch {
				case searchErr != nil:
					result.Action = "failed"
					result.Error = redactReviewGatewayError(searchErr.Error())
				case search.Matches > 1:
					result.Action = "failed"
					result.Error = "multiple Feishu Base records share the same unique_key; manual reconciliation is required"
				case search.Found:
					remoteID = search.RecordID
				}
			}
			if result.Action == "failed" {
				break
			}
			if remoteID == "" {
				fields := reviewCollaborationBitableFields(bundle, true)
				created, createErr := p.Client.CreateBitableRecord(
					ctx,
					token.Value,
					p.Config.BaseAppToken,
					p.Config.ReviewTableID,
					fields,
				)
				if createErr != nil || strings.TrimSpace(created.RecordID) == "" {
					result.Action = "failed"
					if createErr != nil {
						result.Error = redactReviewGatewayError(createErr.Error())
					} else {
						result.Error = "Feishu Base create response missing record_id"
					}
				} else {
					result.Action = "created"
					result.RemoteID = created.RecordID
				}
				break
			}
			fields := reviewCollaborationBitableFields(bundle, bundle.HumanFieldsAuthoritative)
			updated, updateErr := p.Client.UpdateBitableRecord(
				ctx,
				token.Value,
				p.Config.BaseAppToken,
				p.Config.ReviewTableID,
				remoteID,
				fields,
			)
			if updateErr != nil {
				result.Action = "failed"
				result.Error = redactReviewGatewayError(updateErr.Error())
			} else {
				result.Action = "updated"
				result.RemoteID = updated.RecordID
			}
		}
		if result.Action != "failed" && result.Action != "unchanged" {
			p.persistRemoteWrite(
				ctx,
				&result,
				bundle.UniqueKey,
				fingerprint,
			)
		}
		results = append(results, result)
	}
	if p.Config.DocumentID != "" || p.Config.DocumentFolderToken != "" {
		result := ReviewResourceSyncResult{Resource: "feishu_doc"}
		state, stateErr := p.Store.GetReviewResourceState(ctx, bundle.UniqueKey, result.Resource)
		documentID := state.RemoteID
		if documentID == "" {
			documentID = strings.TrimSpace(p.Config.DocumentID)
		}
		switch {
		case stateErr != nil:
			result.Action = "failed"
			result.Error = redactReviewGatewayError(stateErr.Error())
		case state.ContentFingerprint == fingerprint && documentID != "":
			result.Action = "unchanged"
			result.RemoteID = documentID
		default:
			if documentID == "" {
				created, createErr := p.Client.CreateDocument(
					ctx,
					token.Value,
					p.Config.DocumentFolderToken,
					fmt.Sprintf("%s PR #%d Review", bundle.Item.Repository, bundle.Item.PRNumber),
				)
				if createErr != nil {
					result.Action = "failed"
					result.Error = redactReviewGatewayError(createErr.Error())
					break
				}
				documentID = created.DocumentID
				result.Action = "created"
				result.RemoteID = documentID
				p.persistRemoteWrite(ctx, &result, bundle.UniqueKey, "")
				if result.Action == "unknown" {
					break
				}
			}
			_, appendErr := p.Client.CreateBlocks(
				ctx,
				token.Value,
				documentID,
				documentID,
				[]DocBlock{textBlock(bundle.DocMarkdown)},
			)
			if appendErr != nil {
				result.Action = "failed"
				result.Error = redactReviewGatewayError(appendErr.Error())
			} else {
				if result.Action == "created" {
					result.Action = "created_and_appended"
				} else {
					result.Action = "appended"
				}
				result.RemoteID = documentID
				p.persistRemoteWrite(
					ctx,
					&result,
					bundle.UniqueKey,
					fingerprint,
				)
			}
		}
		results = append(results, result)
	}
	if p.Config.EnableTask && (bundle.Task != nil || bundle.Item.Archived) {
		result := ReviewResourceSyncResult{Resource: "feishu_task"}
		state, stateErr := p.Store.GetReviewResourceState(ctx, bundle.UniqueKey, result.Resource)
		switch {
		case stateErr != nil:
			result.Action = "failed"
			result.Error = redactReviewGatewayError(stateErr.Error())
		case state.ContentFingerprint == fingerprint && state.RemoteID != "":
			result.Action = "unchanged"
			result.RemoteID = state.RemoteID
		case state.RemoteID == "" && bundle.Task == nil:
			result.Action = "not_created"
		case state.RemoteID == "":
			created, createErr := p.Client.CreateTask(ctx, token.Value, *bundle.Task)
			if createErr != nil || strings.TrimSpace(created.TaskID) == "" {
				result.Action = "failed"
				if createErr != nil {
					result.Error = redactReviewGatewayError(createErr.Error())
				} else {
					result.Error = "Feishu Task create response missing task GUID"
				}
			} else {
				result.Action = "created"
				result.RemoteID = created.TaskID
				p.persistRemoteWrite(
					ctx,
					&result,
					bundle.UniqueKey,
					fingerprint,
				)
			}
		default:
			task := bundle.Task
			completedAt := "0"
			if bundle.Item.Archived {
				archived := reviewCollaborationArchivedTask(bundle)
				task = &archived
				completedAt = fmt.Sprintf("%d", p.Now().UTC().UnixMilli())
			}
			patchErr := p.Client.PatchTask(ctx, token.Value, state.RemoteID, *task, completedAt)
			if patchErr != nil {
				result.Action = "failed"
				result.Error = redactReviewGatewayError(patchErr.Error())
			} else {
				result.Action = "updated"
				if bundle.Item.Archived {
					result.Action = "completed"
				}
				result.RemoteID = state.RemoteID
				p.persistRemoteWrite(ctx, &result, bundle.UniqueKey, fingerprint)
			}
		}
		results = append(results, result)
	}
	return results, nil
}

func reviewCollaborationBitableFields(bundle ReviewCollaborationBundle, includeHuman bool) map[string]interface{} {
	manual := map[string]bool{
		"assigned_to":          true,
		"collaboration_status": true,
		"due_at":               true,
	}
	selected := map[string]interface{}{}
	for key, value := range bundle.BitableRecord.Fields {
		if !includeHuman && manual[key] {
			continue
		}
		selected[key] = value
	}
	fields := normalizeBitableWriteFields(selected)
	fields["unique_key"] = bundle.UniqueKey
	return fields
}

func reviewCollaborationArchivedTask(bundle ReviewCollaborationBundle) TaskCandidate {
	return TaskCandidate{
		UniqueKey:   bundle.UniqueKey,
		Title:       fmt.Sprintf("Review %s PR #%d", bundle.Item.Repository, bundle.Item.PRNumber),
		Description: "GitLink PR is closed or merged; collaboration task archived by the Review gateway.",
		SourceType:  "gitlink_pr_review",
		SourceKey:   bundle.Item.PRKey,
		Repository:  bundle.Item.Repository,
		GitLinkURL:  fmt.Sprintf("https://www.gitlink.org.cn/%s/pulls/%d", bundle.Item.Repository, bundle.Item.PRNumber),
	}
}

func (p *FeishuReviewCollaborationPublisher) persistRemoteWrite(
	ctx context.Context,
	result *ReviewResourceSyncResult,
	workItemKey,
	fingerprint string,
) {
	err := p.Store.SaveReviewResourceState(
		ctx,
		workItemKey,
		result.Resource,
		result.RemoteID,
		fingerprint,
		p.Now().UTC(),
	)
	if err == nil {
		return
	}
	result.Action = "unknown"
	result.Error = redactReviewGatewayError(fmt.Sprintf(
		"remote write completed but local idempotency state was not persisted: %v",
		err,
	))
}

type ReviewResourceState struct {
	WorkItemKey        string
	ResourceType       string
	RemoteID           string
	ContentFingerprint string
	UpdatedAt          string
}

func (s *SQLiteReviewGatewayStore) GetReviewResourceState(
	ctx context.Context,
	workItemKey,
	resourceType string,
) (ReviewResourceState, error) {
	state := ReviewResourceState{WorkItemKey: workItemKey, ResourceType: resourceType}
	err := s.db.QueryRowContext(ctx, `SELECT remote_id, content_fingerprint, updated_at
		FROM review_collaboration_resources
		WHERE work_item_key = ? AND resource_type = ?`,
		workItemKey,
		resourceType,
	).Scan(&state.RemoteID, &state.ContentFingerprint, &state.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return state, nil
		}
		return state, err
	}
	return state, nil
}

func (s *SQLiteReviewGatewayStore) SaveReviewResourceState(
	ctx context.Context,
	workItemKey,
	resourceType,
	remoteID,
	fingerprint string,
	now time.Time,
) error {
	_, err := s.db.ExecContext(ctx, `INSERT INTO review_collaboration_resources (
		work_item_key, resource_type, remote_id, content_fingerprint, updated_at
	) VALUES (?, ?, ?, ?, ?)
	ON CONFLICT(work_item_key, resource_type) DO UPDATE SET
		remote_id=excluded.remote_id,
		content_fingerprint=excluded.content_fingerprint,
		updated_at=excluded.updated_at`,
		workItemKey,
		resourceType,
		remoteID,
		fingerprint,
		now.UTC().Format(time.RFC3339Nano),
	)
	return err
}

func reviewCollaborationBundleFingerprint(bundle ReviewCollaborationBundle) string {
	payload, _ := json.Marshal(bundle.Canonical)
	digest := sha256.Sum256(payload)
	return hex.EncodeToString(digest[:16])
}
