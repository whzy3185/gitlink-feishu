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
		result.Message = "PR head、Review 事实指纹或数据完整性已变化，ActionPlan 已失效；GitLink 写入为 0。"
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
			now,
		)
		if finishErr != nil {
			_ = e.ActionPlans.MarkReviewActionPlanUnknown(
				ctx,
				plan.PlanID,
				"GitLink POST result and local reconciliation persistence are both uncertain",
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
			Mutated:        false,
			Reconciliation: "query current reviews by head and content before retrying; automatic retry is disabled",
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
		_ = e.ActionPlans.MarkReviewActionPlanUnknown(
			ctx,
			plan.PlanID,
			"GitLink Review may exist but local completion state was not persisted",
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
