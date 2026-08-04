package feishu

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

// ReviewOperationPlanner converts a completed local job into durable remote
// side-effect intents. Planning is deliberately network-free.
type ReviewOperationPlanner struct {
	Store         *SQLiteReviewGatewayStore
	Config        ReviewCollaborationPublisherConfig
	Now           func() time.Time
	PollInterval  time.Duration
	LeaseDuration time.Duration
	OnPlanned     func([]ReviewOperation)
	wake          chan struct{}
}

type ReviewOperationPlanItem struct {
	Job          ReviewGatewayJob
	Result       ReviewGatewayExecutionResult
	LeaseOwner   string
	AttemptCount int
}

func (p *ReviewOperationPlanner) Wake() {
	if p == nil {
		return
	}
	if p.wake == nil {
		p.wake = make(chan struct{}, 1)
	}
	select {
	case p.wake <- struct{}{}:
	default:
	}
}

func (p *ReviewOperationPlanner) Run(ctx context.Context, instanceID string, workers int) error {
	if p == nil || p.Store == nil {
		return fmt.Errorf("review operation planner store is required")
	}
	if workers < 1 || workers > 32 {
		return fmt.Errorf("review operation planner workers must be between 1 and 32")
	}
	if p.PollInterval <= 0 {
		p.PollInterval = time.Second
	}
	if p.LeaseDuration <= 0 {
		p.LeaseDuration = time.Minute
	}
	if p.wake == nil {
		p.wake = make(chan struct{}, 1)
	}
	for index := 0; index < workers; index++ {
		owner := fmt.Sprintf("%s-planner-%d", firstNonEmpty(instanceID, "review"), index+1)
		go p.runWorker(ctx, owner)
	}
	p.Wake()
	<-ctx.Done()
	return nil
}

func (p *ReviewOperationPlanner) runWorker(ctx context.Context, owner string) {
	ticker := time.NewTicker(p.PollInterval)
	defer ticker.Stop()
	for {
		for {
			item, err := p.claim(ctx, owner)
			if err != nil || item == nil {
				break
			}
			operations, planErr := p.planClaimed(ctx, *item)
			if planErr != nil {
				_ = p.failClaim(ctx, *item, planErr)
				break
			}
			if p.OnPlanned != nil {
				p.OnPlanned(operations)
			}
		}
		select {
		case <-ctx.Done():
			return
		case <-p.wake:
		case <-ticker.C:
		}
	}
}

func (p *ReviewOperationPlanner) claim(ctx context.Context, owner string) (*ReviewOperationPlanItem, error) {
	now := p.now()
	tx, err := p.Store.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()
	nowText := reviewGatewayTimestamp(now)
	_, err = tx.ExecContext(ctx, `UPDATE review_gateway_jobs SET operation_plan_status='retry_scheduled',operation_plan_lease_owner='',operation_plan_lease_expires_at='',operation_plan_next_attempt_at=?,updated_at=? WHERE operation_plan_status='planning' AND operation_plan_lease_expires_at<>'' AND operation_plan_lease_expires_at<=?`, nowText, nowText, nowText)
	if err != nil {
		return nil, err
	}
	var jobID, payload, resultJSON string
	var attempts int
	err = tx.QueryRowContext(ctx, `SELECT job_id,payload_json,result_json,operation_plan_attempt_count FROM review_gateway_jobs WHERE status='completed' AND operation_plan_status IN ('pending','retry_scheduled') AND (operation_plan_next_attempt_at='' OR operation_plan_next_attempt_at<=?) ORDER BY updated_at,job_id LIMIT 1`, nowText).Scan(&jobID, &payload, &resultJSON, &attempts)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	updated, err := tx.ExecContext(ctx, `UPDATE review_gateway_jobs SET operation_plan_status='planning',operation_plan_attempt_count=operation_plan_attempt_count+1,operation_plan_lease_owner=?,operation_plan_lease_expires_at=?,updated_at=? WHERE job_id=? AND operation_plan_status IN ('pending','retry_scheduled')`, owner, reviewGatewayTimestamp(now.Add(p.LeaseDuration)), nowText, jobID)
	if err != nil {
		return nil, err
	}
	affected, _ := updated.RowsAffected()
	if affected != 1 {
		return nil, nil
	}
	var job ReviewGatewayJob
	var result ReviewGatewayExecutionResult
	if err := json.Unmarshal([]byte(payload), &job); err != nil {
		return nil, err
	}
	if err := json.Unmarshal([]byte(resultJSON), &result); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return &ReviewOperationPlanItem{Job: job, Result: result, LeaseOwner: owner, AttemptCount: attempts + 1}, nil
}

func (p *ReviewOperationPlanner) planClaimed(ctx context.Context, item ReviewOperationPlanItem) ([]ReviewOperation, error) {
	now := p.now()
	consumers, err := p.Store.ListReviewJobConsumers(ctx, item.Job.JobID)
	if err != nil {
		return nil, err
	}
	operations, err := p.BuildWithConsumers(item.Job, item.Result, consumers, now)
	if err != nil {
		return nil, err
	}
	tx, err := p.Store.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()
	if err := saveReviewOperationsTx(ctx, tx, operations); err != nil {
		return nil, err
	}
	if reviewOperationHasCard(item.Result.ResultCard) {
		if err := p.ensureProjectionConfigurationTx(ctx, tx, item.Job, now); err != nil {
			return nil, err
		}
	}
	updated, err := tx.ExecContext(ctx, `UPDATE review_gateway_jobs SET operation_plan_status='planned',operation_plan_lease_owner='',operation_plan_lease_expires_at='',operation_plan_error_summary='',updated_at=? WHERE job_id=? AND operation_plan_status='planning' AND operation_plan_lease_owner=?`, reviewGatewayTimestamp(now), item.Job.JobID, item.LeaseOwner)
	if err != nil {
		return nil, err
	}
	affected, _ := updated.RowsAffected()
	if affected != 1 {
		return nil, fmt.Errorf("review operation plan lease is no longer current")
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return operations, nil
}

func (p *ReviewOperationPlanner) failClaim(ctx context.Context, item ReviewOperationPlanItem, planErr error) error {
	now := p.now()
	status := "retry_scheduled"
	next := reviewGatewayTimestamp(now.Add(reviewGatewayRetryDelay(item.AttemptCount)))
	if item.AttemptCount >= 3 {
		status = "failed"
		next = ""
	}
	_, err := p.Store.db.ExecContext(ctx, `UPDATE review_gateway_jobs SET operation_plan_status=?,operation_plan_next_attempt_at=?,operation_plan_lease_owner='',operation_plan_lease_expires_at='',operation_plan_error_summary=?,updated_at=? WHERE job_id=? AND operation_plan_status='planning' AND operation_plan_lease_owner=?`, status, next, redactReviewGatewayError(planErr.Error()), reviewGatewayTimestamp(now), item.Job.JobID, item.LeaseOwner)
	return err
}

func (p *ReviewOperationPlanner) now() time.Time {
	if p.Now != nil {
		return p.Now().UTC()
	}
	return time.Now().UTC()
}

type reviewCanonicalCardDesired struct {
	JobID                 string `json:"job_id"`
	PresentationKey       string `json:"presentation_key"`
	PresentationVersion   int    `json:"presentation_version"`
	SourceCompletedAt     string `json:"source_completed_at,omitempty"`
	ProjectionFingerprint string `json:"projection_fingerprint,omitempty"`
	Card                  Card   `json:"card"`
}

type reviewReplyDesired struct {
	JobID           string `json:"job_id"`
	SourceMessageID string `json:"source_message_id,omitempty"`
	Text            string `json:"text"`
}

type reviewBitableDesired struct {
	ResourceKey string                 `json:"resource_key"`
	TargetScope string                 `json:"target_scope"`
	Fields      map[string]interface{} `json:"fields"`
}

type reviewDocDesired struct {
	ResourceKey string `json:"resource_key"`
	TargetScope string `json:"target_scope"`
	Title       string `json:"title"`
	Markdown    string `json:"markdown"`
}

type reviewTaskDesired struct {
	ResourceKey string         `json:"resource_key"`
	TargetScope string         `json:"target_scope"`
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
	consumers, _ := p.Store.ListReviewJobConsumers(ctx, job.JobID)
	operations, err := p.BuildWithConsumers(job, result, consumers, now().UTC())
	if err != nil {
		return nil, err
	}
	if err := p.Store.SaveReviewOperations(ctx, operations); err != nil {
		return nil, err
	}
	if reviewOperationHasCard(result.ResultCard) {
		if err := p.ensureProjectionConfiguration(ctx, job, now().UTC()); err != nil {
			return nil, err
		}
	}
	return operations, nil
}

func (p *ReviewOperationPlanner) ensureProjectionConfiguration(ctx context.Context, job ReviewGatewayJob, now time.Time) error {
	return p.ensureProjectionConfigurationWith(ctx, p.Store.db, job, now)
}

func (p *ReviewOperationPlanner) ensureProjectionConfigurationTx(ctx context.Context, tx *sql.Tx, job ReviewGatewayJob, now time.Time) error {
	return p.ensureProjectionConfigurationWith(ctx, tx, job, now)
}

func (p *ReviewOperationPlanner) ensureProjectionConfigurationWith(ctx context.Context, execer reviewProjectionExecer, job ReviewGatewayJob, now time.Time) error {
	baseScope, _ := RuntimeReviewResourceScope(ReviewResourceBitable, p.Config.BaseScope)
	baseStatus := ReviewProjectionPending
	if baseScope == ReviewResourceScopeDisabled {
		baseStatus = ReviewProjectionDisabled
	} else if p.Config.BaseAppToken == "" || p.Config.ReviewTableID == "" {
		baseStatus = ReviewProjectionNotConfigured
	}
	docScope, _ := RuntimeReviewResourceScope(ReviewResourceDoc, p.Config.DocScope)
	docStatus := ReviewProjectionPending
	if docScope == ReviewResourceScopeDisabled {
		docStatus = ReviewProjectionDisabled
	} else if p.Config.DocumentID == "" && p.Config.DocumentFolderToken == "" {
		docStatus = ReviewProjectionNotConfigured
	}
	taskStatus := ReviewProjectionDisabled
	if p.Config.EnableTask {
		taskStatus = ReviewProjectionPending
	}
	for _, item := range []struct {
		resource string
		scope    ReviewResourceScope
		status   ReviewResourceProjectionStatusValue
	}{
		{ReviewResourceBitable, baseScope, baseStatus},
		{ReviewResourceDoc, docScope, docStatus},
		{ReviewResourceTask, ReviewResourceScopeChat, taskStatus},
	} {
		if err := ensureReviewProjectionConfigurationExec(ctx, execer, job, item.resource, item.scope, item.status, now); err != nil {
			return err
		}
	}
	return nil
}

func (p *ReviewOperationPlanner) Build(job ReviewGatewayJob, result ReviewGatewayExecutionResult, now time.Time) ([]ReviewOperation, error) {
	return p.BuildWithConsumers(job, result, nil, now)
}

func (p *ReviewOperationPlanner) BuildWithConsumers(job ReviewGatewayJob, result ReviewGatewayExecutionResult, consumers []ReviewJobConsumer, now time.Time) ([]ReviewOperation, error) {
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

	if len(consumers) == 0 && reviewOperationNeedsReply(job) {
		consumers = []ReviewJobConsumer{{ConsumerID: stableKey("review-job-consumer", job.JobID, firstNonEmpty(job.SourceMessageID, job.SourceEventID, job.RequestedBy)), JobID: job.JobID, SourceType: normalizeReviewConsumerSourceType("", job), ChatID: job.ChatID, SourceMessageID: job.SourceMessageID, NotificationMode: firstNonEmpty(job.NotificationMode, ReviewNotificationCanonicalOnly)}}
	}
	for _, consumer := range consumers {
		if !reviewOperationConsumerNeedsReply(consumer) {
			continue
		}
		consumerID := consumer.ConsumerID
		desired := reviewReplyDesired{
			JobID:           job.JobID,
			SourceMessageID: consumer.SourceMessageID,
			Text:            truncateReviewGatewayText(formatReviewGatewayResultReply(job, result), 3000),
		}
		operation, err := NewReviewOperation(ReviewOperationReplySend, job, consumerID, "feishu_reply", desired, ReviewRetryNonIdempotent, now)
		if err != nil {
			return nil, err
		}
		operation.ConsumerID = consumerID
		operation.ChatID = firstNonEmpty(consumer.ChatID, job.ChatID)
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

func reviewOperationConsumerNeedsReply(consumer ReviewJobConsumer) bool {
	switch consumer.SourceType {
	case "user_command":
		return consumer.SourceMessageID != ""
	case "event_route", "replay":
		return consumer.NotificationMode == ReviewNotificationCanonicalAndNotice
	default:
		return false
	}
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
				TargetScope: string(p.Config.BaseScope),
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
				TargetScope: string(p.Config.DocScope),
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
		desired := reviewTaskDesired{ResourceKey: bundle.UniqueKey, TargetScope: string(ReviewResourceScopeChat), Archived: bundle.Item.Archived, Task: bundle.Task}
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
