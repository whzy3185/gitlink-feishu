package feishu

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	larktypes "github.com/larksuite/oapi-sdk-go/v3/channel/types"
)

type reviewOperationOpenAPIClient interface {
	reviewTenantTokenClient
	SearchBitableRecord(context.Context, string, string, string, string) (BitableSearchResult, error)
	CreateBitableRecord(context.Context, string, string, string, map[string]interface{}) (BitableWriteResult, error)
	UpdateBitableRecord(context.Context, string, string, string, string, map[string]interface{}) (BitableWriteResult, error)
	CreateDocument(context.Context, string, string, string) (CreatedDocument, error)
	CreateBlocks(context.Context, string, string, string, []DocBlock) (CreatedBlocks, error)
	CreateTask(context.Context, string, TaskCandidate) (CreatedTask, error)
	PatchTask(context.Context, string, string, TaskCandidate, string) error
}

type ReviewOperationExecutionResult struct {
	RemoteID           string
	AppliedFingerprint string
	Unchanged          bool
}

type ReviewOperationHandler struct {
	Store         *SQLiteReviewGatewayStore
	Client        reviewOperationOpenAPIClient
	Sender        reviewGatewayMessageSender
	TokenProvider *ReviewTenantTokenProvider
	Config        ReviewCollaborationPublisherConfig
	Now           func() time.Time
}

func (h *ReviewOperationHandler) Execute(ctx context.Context, operation ReviewOperation) (ReviewOperationExecutionResult, error) {
	if h == nil || h.Store == nil {
		return ReviewOperationExecutionResult{}, fmt.Errorf("review operation handler store is required")
	}
	switch operation.OperationKind {
	case ReviewOperationCanonicalCardUpsert:
		return h.executeCanonicalCard(ctx, operation)
	case ReviewOperationReplySend:
		return h.executeReply(ctx, operation)
	case ReviewOperationBitableUpsert:
		return h.executeBitable(ctx, operation)
	case ReviewOperationDocSnapshotUpsert:
		return h.executeDoc(ctx, operation)
	case ReviewOperationTaskUpsert:
		return h.executeTask(ctx, operation)
	default:
		return ReviewOperationExecutionResult{}, &ReviewOperationError{Class: ReviewOperationErrorTerminal, Code: "unsupported_operation_kind", Err: fmt.Errorf("unsupported operation kind %q", operation.OperationKind)}
	}
}

func (h *ReviewOperationHandler) executeCanonicalCard(ctx context.Context, operation ReviewOperation) (ReviewOperationExecutionResult, error) {
	if h.Sender == nil {
		return ReviewOperationExecutionResult{}, terminalReviewOperationError("message_sender_not_configured", "Feishu message sender is required")
	}
	var desired reviewCanonicalCardDesired
	if err := jsonUnmarshalBounded(operation.DesiredJSON, &desired); err != nil {
		return ReviewOperationExecutionResult{}, terminalReviewOperationError("invalid_card_payload", err.Error())
	}
	job := operationReviewGatewayJob(operation, desired.JobID)
	statuses, err := h.Store.ListReviewResourceProjectionStatuses(ctx, operation.InstallationID, operation.ChatID, operation.Repository, operation.PRNumber)
	if err != nil {
		return ReviewOperationExecutionResult{}, err
	}
	desired.Card = reviewCardWithProjectionStatuses(desired.Card, statuses)
	fingerprint := reviewGatewayCardFingerprint(desired.Card)
	state, err := h.Store.GetChatPRPresentation(ctx, job)
	if err != nil {
		return ReviewOperationExecutionResult{}, err
	}
	if state.CanonicalMessageID != "" && state.ContentFingerprint == fingerprint && state.CardStatus == "active" {
		return ReviewOperationExecutionResult{RemoteID: state.CanonicalMessageID, AppliedFingerprint: operation.DesiredFingerprint, Unchanged: true}, nil
	}
	now := h.now()
	if state.CanonicalMessageID == "" {
		_, acquired, err := h.Store.AcquireChatPRPresentationCreate(ctx, job, operation.OperationID, now)
		if err != nil || !acquired {
			if err != nil {
				return ReviewOperationExecutionResult{}, err
			}
			return ReviewOperationExecutionResult{}, staleReviewOperationError("canonical_create_not_acquired")
		}
		cardJSON, err := encodeReviewGatewayCard(desired.Card)
		if err != nil {
			return ReviewOperationExecutionResult{}, terminalReviewOperationError("invalid_card_payload", err.Error())
		}
		sent, sendErr := h.Sender.Send(ctx, &larktypes.SendInput{ChatID: operation.ChatID, MsgType: "interactive", Card: cardJSON})
		if sendErr != nil {
			classified := h.classify(sendErr, operation, true)
			if classified.RemoteSideEffectPossible {
				_ = h.Store.MarkChatPRPresentationUnknown(context.Background(), job, operation.OperationID, "", classified.Error(), now)
			}
			return ReviewOperationExecutionResult{}, classified
		}
		if sent == nil || strings.TrimSpace(sent.MessageID) == "" {
			err := &ReviewOperationError{Class: ReviewOperationErrorUnknownSideEffect, Code: "card_create_missing_message_id", RemoteSideEffectPossible: true, Err: fmt.Errorf("Feishu card create response missing message ID")}
			_ = h.Store.MarkChatPRPresentationUnknown(context.Background(), job, operation.OperationID, "", err.Error(), now)
			return ReviewOperationExecutionResult{}, err
		}
		if err := h.Store.CompleteChatPRPresentationCreate(ctx, job, operation.OperationID, sent.MessageID, fingerprint, desired.SourceCompletedAt, false, now); err != nil {
			_ = h.Store.MarkChatPRPresentationUnknown(context.Background(), job, operation.OperationID, sent.MessageID, err.Error(), now)
			return ReviewOperationExecutionResult{RemoteID: sent.MessageID}, unknownReviewOperationError("card_create_local_persist_failed", err)
		}
		return ReviewOperationExecutionResult{RemoteID: sent.MessageID, AppliedFingerprint: operation.DesiredFingerprint}, nil
	}

	updater, ok := h.Sender.(reviewGatewayMessageUpdater)
	if !ok {
		return ReviewOperationExecutionResult{}, terminalReviewOperationError("message_updater_not_configured", "Feishu interactive message updater is required")
	}
	_, acquired, err := h.Store.AcquireChatPRPresentationPatch(ctx, job, state.ContentFingerprint, operation.OperationID, desired.SourceCompletedAt, now)
	if err != nil || !acquired {
		if err != nil {
			return ReviewOperationExecutionResult{}, err
		}
		return ReviewOperationExecutionResult{RemoteID: state.CanonicalMessageID, AppliedFingerprint: operation.DesiredFingerprint, Unchanged: true}, nil
	}
	if err := updater.UpdateInteractiveMessage(ctx, state.CanonicalMessageID, desired.Card); err != nil {
		patchOperation := operation
		patchOperation.RetrySafety = ReviewRetryIdempotent
		classified := h.classify(err, patchOperation, true)
		_ = h.Store.ReleaseChatPRPresentationPatch(context.Background(), job, operation.OperationID, state.CardStatus, classified.Error(), now)
		return ReviewOperationExecutionResult{RemoteID: state.CanonicalMessageID}, classified
	}
	if err := h.Store.CompleteChatPRPresentationPatch(ctx, job, operation.OperationID, fingerprint, desired.SourceCompletedAt, false, now); err != nil {
		_ = h.Store.MarkChatPRPresentationUnknown(context.Background(), job, operation.OperationID, state.CanonicalMessageID, err.Error(), now)
		return ReviewOperationExecutionResult{RemoteID: state.CanonicalMessageID}, unknownReviewOperationError("card_patch_local_persist_failed", err)
	}
	return ReviewOperationExecutionResult{RemoteID: state.CanonicalMessageID, AppliedFingerprint: operation.DesiredFingerprint}, nil
}

func (h *ReviewOperationHandler) executeReply(ctx context.Context, operation ReviewOperation) (ReviewOperationExecutionResult, error) {
	if h.Sender == nil {
		return ReviewOperationExecutionResult{}, terminalReviewOperationError("message_sender_not_configured", "Feishu message sender is required")
	}
	var desired reviewReplyDesired
	if err := jsonUnmarshalBounded(operation.DesiredJSON, &desired); err != nil || strings.TrimSpace(desired.Text) == "" {
		return ReviewOperationExecutionResult{}, terminalReviewOperationError("invalid_reply_payload", "reply text is required")
	}
	sent, err := h.Sender.Send(ctx, &larktypes.SendInput{ChatID: operation.ChatID, ReplyMessageID: desired.SourceMessageID, MsgType: "text", Text: desired.Text})
	if err != nil {
		return ReviewOperationExecutionResult{}, h.classify(err, operation, true)
	}
	if sent == nil || strings.TrimSpace(sent.MessageID) == "" {
		return ReviewOperationExecutionResult{}, unknownReviewOperationError("reply_missing_message_id", fmt.Errorf("Feishu reply response missing message ID"))
	}
	return ReviewOperationExecutionResult{RemoteID: sent.MessageID, AppliedFingerprint: operation.DesiredFingerprint}, nil
}

func (h *ReviewOperationHandler) executeBitable(ctx context.Context, operation ReviewOperation) (ReviewOperationExecutionResult, error) {
	var desired reviewBitableDesired
	if err := jsonUnmarshalBounded(operation.DesiredJSON, &desired); err != nil || desired.ResourceKey == "" {
		return ReviewOperationExecutionResult{}, terminalReviewOperationError("invalid_bitable_payload", "Base resource key is required")
	}
	token, err := h.token(ctx)
	if err != nil {
		return ReviewOperationExecutionResult{}, err
	}
	state, err := h.Store.GetReviewResourceState(ctx, desired.ResourceKey, ReviewResourceBitable)
	if err != nil {
		return ReviewOperationExecutionResult{}, err
	}
	if state.RemoteID != "" && state.ContentFingerprint == operation.DesiredFingerprint {
		return ReviewOperationExecutionResult{RemoteID: state.RemoteID, AppliedFingerprint: operation.DesiredFingerprint, Unchanged: true}, nil
	}
	remoteID := state.RemoteID
	if remoteID == "" {
		search, searchErr := h.Client.SearchBitableRecord(ctx, token, h.Config.BaseAppToken, h.Config.ReviewTableID, desired.ResourceKey)
		if searchErr != nil {
			return ReviewOperationExecutionResult{}, h.classify(searchErr, operation, false)
		}
		if search.Matches > 1 {
			return ReviewOperationExecutionResult{}, unknownReviewOperationError("base_multiple_matches", fmt.Errorf("multiple Base records require reconciliation"))
		}
		if search.Found {
			remoteID = search.RecordID
		}
	}
	if remoteID == "" {
		created, createErr := h.Client.CreateBitableRecord(ctx, token, h.Config.BaseAppToken, h.Config.ReviewTableID, desired.Fields)
		if createErr != nil {
			return ReviewOperationExecutionResult{}, h.classify(createErr, operation, true)
		}
		remoteID = created.RecordID
		if remoteID == "" {
			return ReviewOperationExecutionResult{}, unknownReviewOperationError("base_create_missing_record_id", fmt.Errorf("Base create response missing record ID"))
		}
	} else {
		_, updateErr := h.Client.UpdateBitableRecord(ctx, token, h.Config.BaseAppToken, h.Config.ReviewTableID, remoteID, desired.Fields)
		if updateErr != nil {
			updateOperation := operation
			updateOperation.RetrySafety = ReviewRetryIdempotent
			return ReviewOperationExecutionResult{RemoteID: remoteID}, h.classify(updateErr, updateOperation, true)
		}
	}
	if err := h.Store.SaveReviewResourceState(ctx, desired.ResourceKey, ReviewResourceBitable, remoteID, operation.DesiredFingerprint, h.now()); err != nil {
		return ReviewOperationExecutionResult{RemoteID: remoteID}, unknownReviewOperationError("base_local_persist_failed", err)
	}
	return ReviewOperationExecutionResult{RemoteID: remoteID, AppliedFingerprint: operation.DesiredFingerprint}, nil
}

func (h *ReviewOperationHandler) executeDoc(ctx context.Context, operation ReviewOperation) (ReviewOperationExecutionResult, error) {
	var desired reviewDocDesired
	if err := jsonUnmarshalBounded(operation.DesiredJSON, &desired); err != nil || desired.ResourceKey == "" {
		return ReviewOperationExecutionResult{}, terminalReviewOperationError("invalid_doc_payload", "Doc resource key is required")
	}
	token, err := h.token(ctx)
	if err != nil {
		return ReviewOperationExecutionResult{}, err
	}
	state, err := h.Store.GetReviewResourceState(ctx, desired.ResourceKey, ReviewResourceDoc)
	if err != nil {
		return ReviewOperationExecutionResult{}, err
	}
	documentID := firstNonEmpty(state.RemoteID, h.Config.DocumentID)
	if documentID != "" && state.ContentFingerprint == operation.DesiredFingerprint {
		return ReviewOperationExecutionResult{RemoteID: documentID, AppliedFingerprint: operation.DesiredFingerprint, Unchanged: true}, nil
	}
	if documentID == "" {
		created, createErr := h.Client.CreateDocument(ctx, token, h.Config.DocumentFolderToken, desired.Title)
		if createErr != nil {
			return ReviewOperationExecutionResult{}, h.classify(createErr, operation, true)
		}
		documentID = created.DocumentID
		if documentID == "" {
			return ReviewOperationExecutionResult{}, unknownReviewOperationError("doc_create_missing_document_id", fmt.Errorf("Doc create response missing document ID"))
		}
		if err := h.Store.SaveReviewResourceState(ctx, desired.ResourceKey, ReviewResourceDoc, documentID, "", h.now()); err != nil {
			return ReviewOperationExecutionResult{RemoteID: documentID}, unknownReviewOperationError("doc_id_local_persist_failed", err)
		}
	}
	if _, err := h.Client.CreateBlocks(ctx, token, documentID, documentID, []DocBlock{textBlock(desired.Markdown)}); err != nil {
		return ReviewOperationExecutionResult{RemoteID: documentID}, h.classify(err, operation, true)
	}
	if err := h.Store.SaveReviewResourceState(ctx, desired.ResourceKey, ReviewResourceDoc, documentID, operation.DesiredFingerprint, h.now()); err != nil {
		return ReviewOperationExecutionResult{RemoteID: documentID}, unknownReviewOperationError("doc_snapshot_local_persist_failed", err)
	}
	return ReviewOperationExecutionResult{RemoteID: documentID, AppliedFingerprint: operation.DesiredFingerprint}, nil
}

func (h *ReviewOperationHandler) executeTask(ctx context.Context, operation ReviewOperation) (ReviewOperationExecutionResult, error) {
	var desired reviewTaskDesired
	if err := jsonUnmarshalBounded(operation.DesiredJSON, &desired); err != nil || desired.ResourceKey == "" {
		return ReviewOperationExecutionResult{}, terminalReviewOperationError("invalid_task_payload", "Task resource key is required")
	}
	token, err := h.token(ctx)
	if err != nil {
		return ReviewOperationExecutionResult{}, err
	}
	state, err := h.Store.GetReviewResourceState(ctx, desired.ResourceKey, ReviewResourceTask)
	if err != nil {
		return ReviewOperationExecutionResult{}, err
	}
	if state.RemoteID != "" && state.ContentFingerprint == operation.DesiredFingerprint {
		return ReviewOperationExecutionResult{RemoteID: state.RemoteID, AppliedFingerprint: operation.DesiredFingerprint, Unchanged: true}, nil
	}
	if state.RemoteID == "" && desired.Archived {
		return ReviewOperationExecutionResult{Unchanged: true, AppliedFingerprint: operation.DesiredFingerprint}, nil
	}
	remoteID := state.RemoteID
	if remoteID == "" {
		if desired.Task == nil {
			return ReviewOperationExecutionResult{}, terminalReviewOperationError("task_candidate_missing", "Task candidate is required")
		}
		created, createErr := h.Client.CreateTask(ctx, token, *desired.Task)
		if createErr != nil {
			return ReviewOperationExecutionResult{}, h.classify(createErr, operation, true)
		}
		remoteID = created.TaskID
		if remoteID == "" {
			return ReviewOperationExecutionResult{}, unknownReviewOperationError("task_create_missing_id", fmt.Errorf("Task create response missing ID"))
		}
	} else {
		task := desired.Task
		completedAt := "0"
		if desired.Archived {
			archived := reviewCollaborationArchivedTask(ReviewCollaborationBundle{UniqueKey: desired.ResourceKey, Item: ReviewCollaborationItem{Repository: operation.Repository, PRNumber: operation.PRNumber}})
			task = &archived
			completedAt = fmt.Sprintf("%d", h.now().UnixMilli())
		}
		if task == nil {
			return ReviewOperationExecutionResult{}, terminalReviewOperationError("task_candidate_missing", "Task candidate is required")
		}
		if err := h.Client.PatchTask(ctx, token, remoteID, *task, completedAt); err != nil {
			patchOperation := operation
			patchOperation.RetrySafety = ReviewRetryIdempotent
			return ReviewOperationExecutionResult{RemoteID: remoteID}, h.classify(err, patchOperation, true)
		}
	}
	if err := h.Store.SaveReviewResourceState(ctx, desired.ResourceKey, ReviewResourceTask, remoteID, operation.DesiredFingerprint, h.now()); err != nil {
		return ReviewOperationExecutionResult{RemoteID: remoteID}, unknownReviewOperationError("task_local_persist_failed", err)
	}
	return ReviewOperationExecutionResult{RemoteID: remoteID, AppliedFingerprint: operation.DesiredFingerprint}, nil
}

func (h *ReviewOperationHandler) token(ctx context.Context) (string, error) {
	if h.Client == nil {
		return "", terminalReviewOperationError("openapi_client_not_configured", "Feishu OpenAPI client is required")
	}
	if h.TokenProvider == nil {
		h.TokenProvider = NewReviewTenantTokenProvider(h.Client, h.Now)
	}
	token, err := h.TokenProvider.Token(ctx, h.Config.AppID, h.Config.AppSecret)
	if err != nil {
		return "", h.classify(err, ReviewOperation{RetrySafety: ReviewRetryIdempotent}, false)
	}
	return token, nil
}

func (h *ReviewOperationHandler) classify(err error, operation ReviewOperation, requestStarted bool) *ReviewOperationError {
	classified := ClassifyReviewOperationError(err, operation, requestStarted)
	if classified != nil && classified.HTTPStatus == 401 && h.TokenProvider != nil {
		h.TokenProvider.Invalidate(h.Config.AppID)
	}
	return classified
}

func (h *ReviewOperationHandler) now() time.Time {
	if h.Now != nil {
		return h.Now().UTC()
	}
	return time.Now().UTC()
}

func operationReviewGatewayJob(operation ReviewOperation, jobID string) ReviewGatewayJob {
	return ReviewGatewayJob{JobID: firstNonEmpty(jobID, operation.SourceJobID), InstallationID: operation.InstallationID, ChatID: operation.ChatID, Repository: operation.Repository, PRNumber: operation.PRNumber}
}

func terminalReviewOperationError(code, message string) *ReviewOperationError {
	return &ReviewOperationError{Class: ReviewOperationErrorTerminal, Code: code, Err: fmt.Errorf("%s", message)}
}

func unknownReviewOperationError(code string, err error) *ReviewOperationError {
	return &ReviewOperationError{Class: ReviewOperationErrorUnknownSideEffect, Code: code, RemoteSideEffectPossible: true, Err: err}
}

func encodeReviewGatewayCard(card Card) (string, error) {
	encoded, err := json.Marshal(card)
	if err != nil {
		return "", fmt.Errorf("encode Feishu card: %w", err)
	}
	return string(encoded), nil
}
