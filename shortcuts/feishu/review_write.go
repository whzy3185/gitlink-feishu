package feishu

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/gitlink-org/gitlink-cli/shortcuts/workflow"
)

func (e *ReviewGatewayExecutor) prepareCommonReview(
	ctx context.Context,
	job ReviewGatewayJob,
	result ReviewGatewayExecutionResult,
	now time.Time,
) (ReviewGatewayExecutionResult, error) {
	if e.ActionPlans == nil {
		return reviewGatewayExecutionFailure(result, fmt.Errorf("review action plan runtime and store are required"))
	}
	runtime, runtimeErr := e.runtimeForJob(job)
	if runtimeErr != nil {
		return reviewGatewayExecutionFailure(result, runtimeErr)
	}
	identity, ok := findReviewIdentity(e.IdentityBindings, job.RequestedBy, job.InstallationID)
	if !ok {
		return reviewGatewayExecutionFailure(result, fmt.Errorf("Feishu account is not bound to a GitLink account"))
	}
	owner, repo, err := splitReviewGatewayRepository(job.Repository)
	if err != nil {
		return reviewGatewayExecutionFailure(result, err)
	}
	reviewContext, err := workflow.FetchReviewContext(runtime, workflow.ReviewContextOptions{
		Owner:           owner,
		Repo:            repo,
		Number:          job.PRNumber,
		VersionLimit:    100,
		ThreadLimit:     100,
		IncludePR:       true,
		IncludeFiles:    true,
		IncludeVersions: true,
		IncludeReviews:  true,
		IncludeThreads:  true,
	})
	if err != nil {
		return reviewGatewayExecutionFailure(result, err)
	}
	populateReviewGatewayContextResult(&result, reviewContext)
	if reviewContext.Partial || reviewContext.CollectionStatus != "complete" ||
		strings.TrimSpace(reviewContext.CurrentHeadSHA) == "" {
		return reviewGatewayExecutionFailure(result, fmt.Errorf(
			"complete Review context and current head are required before write planning",
		))
	}
	draft := buildReviewDraftPreview(reviewContext)
	content := strings.TrimSpace(strings.Join([]string{
		draft.Summary,
		"当前判断：" + firstNonEmpty(draft.Decision, "pending"),
		"建议下一步：" + firstNonEmpty(draft.NextStep, "人工复核"),
		"来源 head：" + reviewContext.CurrentHeadSHA,
		"该 Review 由飞书协作流程生成并经账号本人二次确认。",
	}, "\n"))
	plan := NewReviewActionPlan(
		job,
		identity.GitLinkLogin,
		reviewContext.CurrentHeadSHA,
		reviewContext.WorkItem.SourceFingerprint,
		content,
		now,
	)
	plan, err = e.ActionPlans.CreateReviewActionPlan(ctx, plan)
	if err != nil {
		return reviewGatewayExecutionFailure(result, err)
	}
	result.ActionPlan = &plan
	result.Draft = &draft
	result.ReadOnlyGitLink = true
	result.MutatesGitLink = false
	result.Message = fmt.Sprintf(
		"已生成 common Review ActionPlan %s；15 分钟内由同一飞书账号发送“确认 Review %s”。",
		plan.PlanID,
		plan.PlanID,
	)
	return result, nil
}

func (e *ReviewGatewayExecutor) confirmCommonReview(
	ctx context.Context,
	job ReviewGatewayJob,
	result ReviewGatewayExecutionResult,
	now time.Time,
) (ReviewGatewayExecutionResult, error) {
	if e.ActionPlans == nil {
		return reviewGatewayExecutionFailure(result, fmt.Errorf("review action plan runtime and store are required"))
	}
	plan, err := e.ActionPlans.GetReviewActionPlan(ctx, job.Argument)
	if err != nil {
		return reviewGatewayExecutionFailure(result, fmt.Errorf("load review action plan: %w", err))
	}
	result.ActionPlan = &plan
	result.Repository = plan.Repository
	result.PRNumber = plan.PRNumber
	result.ReadOnlyGitLink = !e.EnableGitLinkWrite
	result.MutatesGitLink = false
	if plan.ActorID != job.RequestedBy {
		return reviewGatewayExecutionFailure(result, fmt.Errorf("review action plan belongs to another Feishu user"))
	}
	if plan.ReviewStatus != "common" {
		return reviewGatewayExecutionFailure(result, fmt.Errorf("only common Review writes are enabled"))
	}
	if err := e.validateReviewActionPlanScope(job, plan); err != nil {
		return reviewGatewayExecutionFailure(result, err)
	}
	if job.InstallationMode != "write" {
		return reviewGatewayExecutionFailure(result, fmt.Errorf(
			"GitLink installation does not allow Review writes",
		))
	}
	identity, ok := findReviewIdentity(e.IdentityBindings, job.RequestedBy, job.InstallationID)
	if !ok || identity.GitLinkLogin != plan.GitLinkLogin {
		return reviewGatewayExecutionFailure(result, fmt.Errorf(
			"GitLink identity binding changed; create a new action plan",
		))
	}
	if !e.EnableGitLinkWrite {
		result.WriteResult = &ReviewWriteResult{
			PlanID:         plan.PlanID,
			Status:         "write_disabled",
			Repository:     plan.Repository,
			PRNumber:       plan.PRNumber,
			HeadSHA:        plan.ExpectedHeadSHA,
			ReviewStatus:   plan.ReviewStatus,
			MutationStatus: reviewMutationNone,
			Mutated:        false,
		}
		result.Message = "ActionPlan 校验通过，但启动参数未显式启用 GitLink common Review 写入；GitLink 写入为 0。"
		return result, nil
	}
	runtime, runtimeErr := e.runtimeForJob(job)
	if runtimeErr != nil {
		return reviewGatewayExecutionFailure(result, runtimeErr)
	}
	leaseOwner := "review-write:" + job.JobID
	plan, err = e.ActionPlans.ClaimReviewActionPlan(ctx, ReviewActionPlanClaimOptions{
		PlanID:        plan.PlanID,
		ActorID:       job.RequestedBy,
		LeaseOwner:    leaseOwner,
		Now:           now,
		LeaseDuration: 2 * time.Minute,
	})
	if err != nil {
		return reviewGatewayExecutionFailure(result, err)
	}
	result.ActionPlan = &plan
	owner, repo, err := splitReviewGatewayRepository(plan.Repository)
	if err != nil {
		_ = e.ActionPlans.FinishReviewActionPlan(ctx, plan.PlanID, "failed", "", err.Error(), reviewMutationNone, now)
		return reviewGatewayExecutionFailure(result, err)
	}
	reviewContext, err := workflow.FetchReviewContext(runtime, workflow.ReviewContextOptions{
		Owner:           owner,
		Repo:            repo,
		Number:          plan.PRNumber,
		VersionLimit:    100,
		ThreadLimit:     100,
		IncludePR:       true,
		IncludeFiles:    true,
		IncludeVersions: true,
		IncludeReviews:  true,
		IncludeThreads:  true,
	})
	if err != nil {
		_ = e.ActionPlans.FinishReviewActionPlan(ctx, plan.PlanID, "failed", "", err.Error(), reviewMutationNone, now)
		return reviewGatewayExecutionFailure(result, err)
	}
	populateReviewGatewayContextResult(&result, reviewContext)
	if reviewContext.Partial || reviewContext.CollectionStatus != "complete" ||
		reviewContext.CurrentHeadSHA != plan.ExpectedHeadSHA ||
		reviewContext.WorkItem.SourceFingerprint != plan.SourceFingerprint {
		_ = e.ActionPlans.FinishReviewActionPlan(
			ctx,
			plan.PlanID,
			"stale",
			"",
			"PR head, source fingerprint, or completeness changed",
			reviewMutationNone,
			now,
		)
		result.WriteResult = &ReviewWriteResult{
			PlanID:         plan.PlanID,
			Status:         "stale",
			Repository:     plan.Repository,
			PRNumber:       plan.PRNumber,
			HeadSHA:        reviewContext.CurrentHeadSHA,
			ReviewStatus:   plan.ReviewStatus,
			MutationStatus: reviewMutationNone,
			Mutated:        false,
		}
		result.Message = "PR head、Review 事实指纹或数据完整性已变化，ActionPlan 已失效；GitLink 写入为 0。"
		return result, nil
	}
	runtimeCopy := *runtime
	runtimeCopy.Owner = owner
	runtimeCopy.Repo = repo
	userEnvelope, err := runtimeCopy.CallAPI("GET", "/users/me", nil)
	if err != nil {
		_ = e.ActionPlans.FinishReviewActionPlan(ctx, plan.PlanID, "failed", "", err.Error(), reviewMutationNone, now)
		return reviewGatewayExecutionFailure(result, err)
	}
	if currentGitLinkLogin(userEnvelope.Data) != plan.GitLinkLogin {
		err = fmt.Errorf("active GitLink token identity does not match the bound GitLink account")
		_ = e.ActionPlans.FinishReviewActionPlan(ctx, plan.PlanID, "failed", "", err.Error(), reviewMutationNone, now)
		return reviewGatewayExecutionFailure(result, err)
	}
	if err := e.ActionPlans.MarkReviewActionPlanWriteStarted(
		ctx,
		plan.PlanID,
		leaseOwner,
		now,
	); err != nil {
		return reviewGatewayExecutionFailure(result, fmt.Errorf(
			"persist Review write boundary before POST: %w",
			err,
		))
	}
	envelope, writeErr := runtimeCopy.CallAPI(
		"POST",
		fmt.Sprintf("/v1/%s/%s/pulls/%d/reviews", owner, repo, plan.PRNumber),
		map[string]interface{}{
			"content":   plan.Content,
			"status":    "common",
			"commit_id": plan.ExpectedHeadSHA,
		},
	)
	if writeErr != nil {
		finishErr := e.ActionPlans.FinishReviewActionPlan(
			ctx,
			plan.PlanID,
			"unknown",
			"",
			writeErr.Error(),
			reviewMutationPossible,
			now,
		)
		if finishErr != nil {
			_ = e.ActionPlans.MarkReviewActionPlanUnknown(
				ctx,
				plan.PlanID,
				"GitLink POST result and local reconciliation persistence are both uncertain",
				reviewMutationPossible,
				now,
			)
		}
		result.WriteResult = &ReviewWriteResult{
			PlanID:         plan.PlanID,
			Status:         "unknown_needs_reconciliation",
			Repository:     plan.Repository,
			PRNumber:       plan.PRNumber,
			HeadSHA:        plan.ExpectedHeadSHA,
			ReviewStatus:   plan.ReviewStatus,
			MutationStatus: reviewMutationPossible,
			Mutated:        false,
			Reconciliation: "query current reviews by head and content before retrying; automatic retry is disabled",
		}
		result.Message = "GitLink 返回不确定结果；ActionPlan 标记 unknown，禁止自动重试以避免重复 Review。"
		return result, nil
	}
	reviewID := extractReviewWriteID(envelope.Data)
	if reviewID == "" {
		return e.finishConfirmedReviewNeedsReconciliation(
			ctx,
			result,
			plan,
			"",
			"GitLink POST succeeded without a Review ID",
			now,
		)
	}
	readback, readbackErr := runtimeCopy.CallAPI(
		"GET",
		fmt.Sprintf("/v1/%s/%s/pulls/%d/reviews", owner, repo, plan.PRNumber),
		nil,
	)
	if readbackErr != nil {
		return e.finishConfirmedReviewNeedsReconciliation(
			ctx,
			result,
			plan,
			reviewID,
			"GitLink Review read-back failed after a confirmed POST: "+readbackErr.Error(),
			now,
		)
	}
	if verified, reason := verifyReviewWriteReadback(readback.Data, reviewID, plan); !verified {
		return e.finishConfirmedReviewNeedsReconciliation(
			ctx,
			result,
			plan,
			reviewID,
			"GitLink Review read-back did not match the confirmed write: "+reason,
			now,
		)
	}
	if err := e.ActionPlans.FinishReviewActionPlan(
		ctx,
		plan.PlanID,
		"completed",
		reviewID,
		"",
		reviewMutationConfirmed,
		now,
	); err != nil {
		_ = e.ActionPlans.MarkReviewActionPlanUnknown(
			ctx,
			plan.PlanID,
			"GitLink Review may exist but local completion state was not persisted",
			reviewMutationConfirmed,
			now,
		)
		result.ReadOnlyGitLink = false
		result.MutatesGitLink = true
		result.WriteResult = &ReviewWriteResult{
			PlanID:         plan.PlanID,
			Status:         "unknown_needs_reconciliation",
			ReviewID:       reviewID,
			Repository:     plan.Repository,
			PRNumber:       plan.PRNumber,
			HeadSHA:        plan.ExpectedHeadSHA,
			ReviewStatus:   "common",
			MutationStatus: reviewMutationConfirmed,
			Mutated:        true,
			Reconciliation: "GitLink POST succeeded; verify Review ID/content before any retry",
		}
		result.Message = "GitLink POST 已返回成功，但本地完成状态保存失败；已禁止自动重试并要求人工对账。"
		return result, nil
	}
	result.ReadOnlyGitLink = false
	result.MutatesGitLink = true
	result.WriteResult = &ReviewWriteResult{
		PlanID:         plan.PlanID,
		Status:         "completed",
		ReviewID:       reviewID,
		Repository:     plan.Repository,
		PRNumber:       plan.PRNumber,
		HeadSHA:        plan.ExpectedHeadSHA,
		ReviewStatus:   "common",
		MutationStatus: reviewMutationConfirmed,
		Mutated:        true,
		Reconciliation: "verified by GitLink GET read-back",
	}
	result.Message = fmt.Sprintf(
		"GitLink common Review 已提交并回读验证；Review ID：%s。",
		reviewID,
	)
	return result, nil
}

func (e *ReviewGatewayExecutor) finishConfirmedReviewNeedsReconciliation(
	ctx context.Context,
	result ReviewGatewayExecutionResult,
	plan ReviewActionPlan,
	reviewID,
	errorSummary string,
	now time.Time,
) (ReviewGatewayExecutionResult, error) {
	finishErr := e.ActionPlans.FinishReviewActionPlan(
		ctx,
		plan.PlanID,
		"unknown",
		reviewID,
		errorSummary,
		reviewMutationConfirmed,
		now,
	)
	if finishErr != nil {
		_ = e.ActionPlans.MarkReviewActionPlanUnknown(
			ctx,
			plan.PlanID,
			"GitLink POST is confirmed but read-back reconciliation state could not be persisted",
			reviewMutationConfirmed,
			now,
		)
	}
	result.ReadOnlyGitLink = false
	result.MutatesGitLink = true
	result.WriteResult = &ReviewWriteResult{
		PlanID:         plan.PlanID,
		Status:         "unknown_needs_reconciliation",
		ReviewID:       reviewID,
		Repository:     plan.Repository,
		PRNumber:       plan.PRNumber,
		HeadSHA:        plan.ExpectedHeadSHA,
		ReviewStatus:   "common",
		MutationStatus: reviewMutationConfirmed,
		Mutated:        true,
		Reconciliation: "POST confirmed; verify Review ID, status, head, content, and actor before any retry",
	}
	result.Message = "GitLink POST 已确认，但写后回读未完成严格对账；ActionPlan 标记 unknown，禁止自动重试。"
	return result, nil
}

func verifyReviewWriteReadback(
	data interface{},
	reviewID string,
	plan ReviewActionPlan,
) (bool, string) {
	reviewID = strings.TrimSpace(reviewID)
	if reviewID == "" {
		return false, "Review ID is empty"
	}
	records := reviewWriteReadbackRecords(data)
	for _, record := range records {
		if reviewWriteScalarString(record, "id", "review_id") != reviewID {
			continue
		}
		if !strings.EqualFold(reviewWriteScalarString(record, "status", "state", "review_status"), "common") {
			return false, "Review status is not common"
		}
		if reviewWriteScalarString(record, "commit_id", "commit_sha", "sha") != strings.TrimSpace(plan.ExpectedHeadSHA) {
			return false, "Review commit does not match the planned head"
		}
		content := reviewWriteScalarString(record, "content", "body", "note", "notes")
		if reviewWriteContentFingerprint(content) != reviewWriteContentFingerprint(plan.Content) {
			return false, "Review content fingerprint does not match"
		}
		if actor := reviewWriteActorLogin(record); actor != "" && actor != strings.TrimSpace(plan.GitLinkLogin) {
			return false, "Review actor does not match the bound GitLink login"
		}
		return true, ""
	}
	return false, "Review ID was not present in the read-back response"
}

func reviewWriteReadbackRecords(data interface{}) []map[string]interface{} {
	switch typed := data.(type) {
	case []interface{}:
		result := make([]map[string]interface{}, 0, len(typed))
		for _, item := range typed {
			if record, ok := item.(map[string]interface{}); ok {
				result = append(result, record)
			}
		}
		return result
	case []map[string]interface{}:
		return typed
	case map[string]interface{}:
		for _, key := range []string{"reviews", "items", "data"} {
			if records := reviewWriteReadbackRecords(typed[key]); len(records) > 0 {
				return records
			}
		}
		if reviewWriteScalarString(typed, "id", "review_id") != "" {
			return []map[string]interface{}{typed}
		}
	}
	return nil
}

func reviewWriteScalarString(item map[string]interface{}, keys ...string) string {
	for _, key := range keys {
		switch value := item[key].(type) {
		case string:
			if value = strings.TrimSpace(value); value != "" {
				return value
			}
		case float64:
			return strconv.FormatInt(int64(value), 10)
		case int:
			return strconv.Itoa(value)
		case int64:
			return strconv.FormatInt(value, 10)
		}
	}
	return ""
}

func reviewWriteActorLogin(item map[string]interface{}) string {
	for _, key := range []string{"user", "actor", "reviewer", "author"} {
		if actor, ok := item[key].(map[string]interface{}); ok {
			if login := reviewWriteScalarString(actor, "login", "username"); login != "" {
				return login
			}
		}
	}
	return reviewWriteScalarString(item, "login", "username")
}

func reviewWriteContentFingerprint(content string) string {
	content = strings.ReplaceAll(content, "\r\n", "\n")
	content = strings.TrimSpace(content)
	digest := sha256.Sum256([]byte(content))
	return "sha256:" + hex.EncodeToString(digest[:])
}

func (e *ReviewGatewayExecutor) validateReviewActionPlanScope(
	job ReviewGatewayJob,
	plan ReviewActionPlan,
) error {
	installationID := strings.TrimSpace(plan.InstallationID)
	sourceChatID := strings.TrimSpace(plan.SourceChatID)
	repository := strings.TrimSpace(plan.Repository)
	if installationID == "" || sourceChatID == "" || repository == "" {
		return fmt.Errorf("review action plan is missing its original installation, chat, or repository scope")
	}
	if strings.TrimSpace(job.InstallationID) != installationID {
		return fmt.Errorf("review action plan belongs to another GitLink installation")
	}
	if strings.TrimSpace(job.ChatID) != sourceChatID {
		return fmt.Errorf("review action plan must be confirmed in its source chat")
	}
	installation, exists := e.Installations[installationID]
	if !exists || !installation.Enabled {
		return fmt.Errorf("review action plan GitLink installation is unavailable")
	}
	if strings.TrimSpace(installation.OperationMode) != "write" {
		return fmt.Errorf("review action plan GitLink installation no longer allows writes")
	}
	if !containsReviewGatewayString(installation.AllowedRepositories, repository) {
		return fmt.Errorf("review action plan repository is outside its GitLink installation allowlist")
	}
	if !containsReviewGatewayString(job.Repositories, repository) {
		return fmt.Errorf("review action plan repository is no longer bound to its source chat")
	}
	return nil
}

func currentGitLinkLogin(data interface{}) string {
	item, ok := data.(map[string]interface{})
	if !ok {
		return ""
	}
	value, _ := item["login"].(string)
	return strings.TrimSpace(value)
}

func extractReviewWriteID(data interface{}) string {
	switch typed := data.(type) {
	case map[string]interface{}:
		for _, key := range []string{"id", "review_id"} {
			switch value := typed[key].(type) {
			case string:
				if strings.TrimSpace(value) != "" {
					return strings.TrimSpace(value)
				}
			case float64:
				return strconv.FormatInt(int64(value), 10)
			case int:
				return strconv.Itoa(value)
			}
		}
		for _, key := range []string{"review", "data"} {
			if id := extractReviewWriteID(typed[key]); id != "" {
				return id
			}
		}
	}
	return ""
}
