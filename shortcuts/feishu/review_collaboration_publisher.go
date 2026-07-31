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
	AppID         string
	AppSecret     string
	BaseAppToken  string
	ReviewTableID string
	DocumentID    string
	EnableTask    bool
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
		search, searchErr := p.Client.SearchBitableRecord(
			ctx,
			token.Value,
			p.Config.BaseAppToken,
			p.Config.ReviewTableID,
			bundle.UniqueKey,
		)
		fields := normalizeBitableWriteFields(bundle.BitableRecord.Fields)
		fields["unique_key"] = bundle.UniqueKey
		if searchErr != nil {
			result.Action = "failed"
			result.Error = redactReviewGatewayError(searchErr.Error())
		} else if search.Found {
			updated, updateErr := p.Client.UpdateBitableRecord(
				ctx,
				token.Value,
				p.Config.BaseAppToken,
				p.Config.ReviewTableID,
				search.RecordID,
				fields,
			)
			if updateErr != nil {
				result.Action = "failed"
				result.Error = redactReviewGatewayError(updateErr.Error())
			} else {
				result.Action = "updated"
				result.RemoteID = updated.RecordID
			}
		} else {
			created, createErr := p.Client.CreateBitableRecord(
				ctx,
				token.Value,
				p.Config.BaseAppToken,
				p.Config.ReviewTableID,
				fields,
			)
			if createErr != nil {
				result.Action = "failed"
				result.Error = redactReviewGatewayError(createErr.Error())
			} else {
				result.Action = "created"
				result.RemoteID = created.RecordID
			}
		}
		if result.Action != "failed" {
			p.persistRemoteWrite(
				ctx,
				&result,
				bundle.UniqueKey,
				fingerprint,
			)
		}
		results = append(results, result)
	}
	if p.Config.DocumentID != "" {
		result := ReviewResourceSyncResult{Resource: "feishu_doc"}
		state, stateErr := p.Store.GetReviewResourceState(ctx, bundle.UniqueKey, result.Resource)
		switch {
		case stateErr != nil:
			result.Action = "failed"
			result.Error = redactReviewGatewayError(stateErr.Error())
		case state.ContentFingerprint == fingerprint:
			result.Action = "unchanged"
			result.RemoteID = p.Config.DocumentID
		default:
			_, appendErr := p.Client.CreateBlocks(
				ctx,
				token.Value,
				p.Config.DocumentID,
				p.Config.DocumentID,
				[]DocBlock{textBlock(bundle.DocMarkdown)},
			)
			if appendErr != nil {
				result.Action = "failed"
				result.Error = redactReviewGatewayError(appendErr.Error())
			} else {
				result.Action = "appended"
				result.RemoteID = p.Config.DocumentID
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
	if p.Config.EnableTask && bundle.Task != nil {
		result := ReviewResourceSyncResult{Resource: "feishu_task"}
		state, stateErr := p.Store.GetReviewResourceState(ctx, bundle.UniqueKey, result.Resource)
		switch {
		case stateErr != nil:
			result.Action = "failed"
			result.Error = redactReviewGatewayError(stateErr.Error())
		case state.RemoteID != "":
			result.Action = "existing"
			result.RemoteID = state.RemoteID
		default:
			created, createErr := p.Client.CreateTask(ctx, token.Value, *bundle.Task)
			if createErr != nil {
				result.Action = "failed"
				result.Error = redactReviewGatewayError(createErr.Error())
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
		}
		results = append(results, result)
	}
	return results, nil
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
