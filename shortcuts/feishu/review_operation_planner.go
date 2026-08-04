package feishu

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
)

// ReviewOperationPlanner converts a completed local job into durable remote
// side-effect intents. Planning is deliberately network-free.
type ReviewOperationPlanner struct {
	Store  *SQLiteReviewGatewayStore
	Config ReviewCollaborationPublisherConfig
	Now    func() time.Time
}

type reviewCanonicalCardDesired struct {
	JobID               string `json:"job_id"`
	PresentationKey     string `json:"presentation_key"`
	PresentationVersion int    `json:"presentation_version"`
	SourceCompletedAt   string `json:"source_completed_at,omitempty"`
	Card                Card   `json:"card"`
}

type reviewReplyDesired struct {
	JobID           string `json:"job_id"`
	SourceMessageID string `json:"source_message_id,omitempty"`
	Text            string `json:"text"`
}

type reviewBitableDesired struct {
	ResourceKey string                 `json:"resource_key"`
	Fields      map[string]interface{} `json:"fields"`
}

type reviewDocDesired struct {
	ResourceKey string `json:"resource_key"`
	Title       string `json:"title"`
	Markdown    string `json:"markdown"`
}

type reviewTaskDesired struct {
	ResourceKey string         `json:"resource_key"`
	Archived    bool           `json:"archived"`
	Task        *TaskCandidate `json:"task,omitempty"`
}

func (p *ReviewOperationPlanner) Plan(ctx context.Context, job ReviewGatewayJob, result ReviewGatewayExecutionResult) ([]ReviewOperation, error) {
	if p == nil || p.Store == nil {
		return nil, fmt.Errorf("review operation planner store is required")
	}
	now := time.Now
	if p.Now != nil {
		now = p.Now
	}
	operations, err := p.Build(job, result, now().UTC())
	if err != nil {
		return nil, err
	}
	if err := p.Store.SaveReviewOperations(ctx, operations); err != nil {
		return nil, err
	}
	return operations, nil
}

func (p *ReviewOperationPlanner) Build(job ReviewGatewayJob, result ReviewGatewayExecutionResult, now time.Time) ([]ReviewOperation, error) {
	if strings.TrimSpace(job.JobID) == "" {
		return nil, fmt.Errorf("review operation planner requires a source job")
	}
	operations := []ReviewOperation{}
	var cardOperationID string
	if result.Status == "completed" && reviewOperationHasCard(result.ResultCard) && job.ChatID != "" && job.Repository != "" && job.PRNumber > 0 {
		presentationVersion := reviewCardPresentationV1
		desired := reviewCanonicalCardDesired{
			JobID:               job.JobID,
			PresentationKey:     reviewChatPRPresentationKey(job),
			PresentationVersion: presentationVersion,
			SourceCompletedAt:   result.CompletedAt,
			Card:                result.ResultCard,
		}
		operation, err := NewReviewOperation(
			ReviewOperationCanonicalCardUpsert,
			job,
			reviewChatPRPresentationKey(job),
			ReviewResourceCard,
			desired,
			ReviewRetryNonIdempotent,
			now,
		)
		if err != nil {
			return nil, err
		}
		operations = append(operations, operation)
		cardOperationID = operation.OperationID
	}

	if reviewOperationNeedsReply(job) {
		consumerID := stableKey("review-job-consumer", job.JobID, firstNonEmpty(job.SourceMessageID, job.SourceEventID, job.RequestedBy))
		desired := reviewReplyDesired{
			JobID:           job.JobID,
			SourceMessageID: job.SourceMessageID,
			Text:            truncateReviewGatewayText(formatReviewGatewayResultReply(job, result), 3000),
		}
		operation, err := NewReviewOperation(ReviewOperationReplySend, job, consumerID, "feishu_reply", desired, ReviewRetryNonIdempotent, now)
		if err != nil {
			return nil, err
		}
		operation.ConsumerID = consumerID
		if cardOperationID != "" {
			operation.DependencyOperationID = cardOperationID
			operation.DependencyPolicy = ReviewDependencySuccess
		}
		operations = append(operations, operation)
	}

	if result.Collaboration != nil {
		resourceOperations, err := p.buildResourceOperations(job, *result.Collaboration, now)
		if err != nil {
			return nil, err
		}
		operations = append(operations, resourceOperations...)
	}
	return operations, nil
}

func (p *ReviewOperationPlanner) buildResourceOperations(job ReviewGatewayJob, bundle ReviewCollaborationBundle, now time.Time) ([]ReviewOperation, error) {
	operations := []ReviewOperation{}
	if p.Config.BaseAppToken != "" && p.Config.ReviewTableID != "" {
		resourceKey, err := reviewPublisherResourceKey(bundle, ReviewResourceBitable, p.Config.BaseScope)
		if !errors.Is(err, ErrReviewResourceProjectionDisabled) {
			if err != nil {
				return nil, err
			}
			desired := reviewBitableDesired{
				ResourceKey: resourceKey,
				Fields:      reviewCollaborationBitableFields(bundle, resourceKey, bundle.HumanFieldsAuthoritative),
			}
			operation, buildErr := NewReviewOperation(ReviewOperationBitableUpsert, job, resourceKey, ReviewResourceBitable, desired, ReviewRetryReconcilable, now)
			if buildErr != nil {
				return nil, buildErr
			}
			operations = append(operations, operation)
		}
	}

	if p.Config.DocumentID != "" || p.Config.DocumentFolderToken != "" {
		resourceKey, err := reviewPublisherResourceKey(bundle, ReviewResourceDoc, p.Config.DocScope)
		if !errors.Is(err, ErrReviewResourceProjectionDisabled) {
			if err != nil {
				return nil, err
			}
			desired := reviewDocDesired{
				ResourceKey: resourceKey,
				Title:       fmt.Sprintf("%s PR #%d Review", bundle.Item.Repository, bundle.Item.PRNumber),
				Markdown:    truncateReviewGatewayText(bundle.DocMarkdown, 128*1024),
			}
			operation, buildErr := NewReviewOperation(ReviewOperationDocSnapshotUpsert, job, resourceKey, ReviewResourceDoc, desired, ReviewRetryNonIdempotent, now)
			if buildErr != nil {
				return nil, buildErr
			}
			operations = append(operations, operation)
		}
	}

	if p.Config.EnableTask && (bundle.Task != nil || bundle.Item.Archived) {
		if bundle.Task == nil && bundle.Item.Archived {
			state, err := p.Store.GetReviewResourceState(context.Background(), bundle.UniqueKey, ReviewResourceTask)
			if err != nil {
				return nil, err
			}
			if state.RemoteID == "" {
				return operations, nil
			}
		}
		desired := reviewTaskDesired{ResourceKey: bundle.UniqueKey, Archived: bundle.Item.Archived, Task: bundle.Task}
		operation, err := NewReviewOperation(ReviewOperationTaskUpsert, job, bundle.UniqueKey, ReviewResourceTask, desired, ReviewRetryNonIdempotent, now)
		if err != nil {
			return nil, err
		}
		operations = append(operations, operation)
	}
	return operations, nil
}

func reviewOperationNeedsReply(job ReviewGatewayJob) bool {
	if job.SourceMessageID != "" {
		return true
	}
	return job.NotificationMode == ReviewNotificationCanonicalAndNotice
}

func reviewOperationHasCard(card Card) bool {
	return len(card) > 0
}
