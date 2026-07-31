package feishu

import (
	"context"
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
	if e.Runtime == nil || e.ActionPlans == nil {
		return reviewGatewayExecutionFailure(result, fmt.Errorf("review action plan runtime and store are required"))
	}
	identity, ok := findReviewIdentity(e.IdentityBindings, job.RequestedBy)
	if !ok {
		return reviewGatewayExecutionFailure(result, fmt.Errorf("Feishu account is not bound to a GitLink account"))
	}
	owner, repo, err := splitReviewGatewayRepository(job.Repository)
	if err != nil {
		return reviewGatewayExecutionFailure(result, err)
	}
	reviewContext, err := workflow.FetchReviewContext(e.Runtime, workflow.ReviewContextOptions{
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
	if e.Runtime == nil || e.ActionPlans == nil {
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
	identity, ok := findReviewIdentity(e.IdentityBindings, job.RequestedBy)
	if !ok || identity.GitLinkLogin != plan.GitLinkLogin {
		return reviewGatewayExecutionFailure(result, fmt.Errorf(
			"GitLink identity binding changed; create a new action plan",
		))
	}
	if !e.EnableGitLinkWrite {
		result.WriteResult = &ReviewWriteResult{
			PlanID:       plan.PlanID,
			Status:       "write_disabled",
			Repository:   plan.Repository,
			PRNumber:     plan.PRNumber,
			HeadSHA:      plan.ExpectedHeadSHA,
			ReviewStatus: plan.ReviewStatus,
			Mutated:      false,
		}
		result.Message = "ActionPlan 校验通过，但启动参数未显式启用 GitLink common Review 写入；GitLink 写入为 0。"
		return result, nil
	}
	plan, err = e.ActionPlans.ClaimReviewActionPlan(ctx, plan.PlanID, job.RequestedBy, now)
	if err != nil {
		return reviewGatewayExecutionFailure(result, err)
	}
	result.ActionPlan = &plan
	owner, repo, err := splitReviewGatewayRepository(plan.Repository)
	if err != nil {
		_ = e.ActionPlans.FinishReviewActionPlan(ctx, plan.PlanID, "failed", "", err.Error(), now)
		return reviewGatewayExecutionFailure(result, err)
	}
	reviewContext, err := workflow.FetchReviewContext(e.Runtime, workflow.ReviewContextOptions{
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
		_ = e.ActionPlans.FinishReviewActionPlan(ctx, plan.PlanID, "failed", "", err.Error(), now)
		return reviewGatewayExecutionFailure(result, err)
	}
	if reviewContext.Partial || reviewContext.CollectionStatus != "complete" ||
		reviewContext.CurrentHeadSHA != plan.ExpectedHeadSHA {
		_ = e.ActionPlans.FinishReviewActionPlan(
			ctx,
			plan.PlanID,
			"stale",
			"",
			"PR head or completeness changed",
			now,
		)
		result.WriteResult = &ReviewWriteResult{
			PlanID:       plan.PlanID,
			Status:       "stale",
			Repository:   plan.Repository,
			PRNumber:     plan.PRNumber,
			HeadSHA:      reviewContext.CurrentHeadSHA,
			ReviewStatus: plan.ReviewStatus,
			Mutated:      false,
		}
		result.Message = "PR head 或数据完整性已变化，ActionPlan 已失效；GitLink 写入为 0。"
		return result, nil
	}
	runtimeCopy := *e.Runtime
	runtimeCopy.Owner = owner
	runtimeCopy.Repo = repo
	userEnvelope, err := runtimeCopy.CallAPI("GET", "/users/me", nil)
	if err != nil {
		_ = e.ActionPlans.FinishReviewActionPlan(ctx, plan.PlanID, "failed", "", err.Error(), now)
		return reviewGatewayExecutionFailure(result, err)
	}
	if currentGitLinkLogin(userEnvelope.Data) != plan.GitLinkLogin {
		err = fmt.Errorf("active GitLink token identity does not match the bound GitLink account")
		_ = e.ActionPlans.FinishReviewActionPlan(ctx, plan.PlanID, "failed", "", err.Error(), now)
		return reviewGatewayExecutionFailure(result, err)
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
		_ = e.ActionPlans.FinishReviewActionPlan(ctx, plan.PlanID, "unknown", "", writeErr.Error(), now)
		result.WriteResult = &ReviewWriteResult{
			PlanID:         plan.PlanID,
			Status:         "unknown",
			Repository:     plan.Repository,
			PRNumber:       plan.PRNumber,
			HeadSHA:        plan.ExpectedHeadSHA,
			ReviewStatus:   plan.ReviewStatus,
			Mutated:        false,
			Reconciliation: "query current reviews before retrying; automatic retry is disabled",
		}
		result.Message = "GitLink 返回不确定结果；ActionPlan 标记 unknown，禁止自动重试以避免重复 Review。"
		return result, nil
	}
	reviewID := extractReviewWriteID(envelope.Data)
	terminalStatus := "completed"
	reconciliation := ""
	if reviewID == "" {
		terminalStatus = "unknown"
		reconciliation = "write returned success without Review ID; query current reviews"
	}
	if err := e.ActionPlans.FinishReviewActionPlan(
		ctx,
		plan.PlanID,
		terminalStatus,
		reviewID,
		reconciliation,
		now,
	); err != nil {
		return reviewGatewayExecutionFailure(result, err)
	}
	result.ReadOnlyGitLink = false
	result.MutatesGitLink = true
	result.WriteResult = &ReviewWriteResult{
		PlanID:         plan.PlanID,
		Status:         terminalStatus,
		ReviewID:       reviewID,
		Repository:     plan.Repository,
		PRNumber:       plan.PRNumber,
		HeadSHA:        plan.ExpectedHeadSHA,
		ReviewStatus:   "common",
		Mutated:        true,
		Reconciliation: reconciliation,
	}
	result.Message = fmt.Sprintf(
		"GitLink common Review 已提交；Review ID：%s。",
		firstNonEmpty(reviewID, "待回读"),
	)
	return result, nil
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
