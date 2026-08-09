package feishu

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/gitlink-org/gitlink-cli/internal/client"
	"github.com/gitlink-org/gitlink-cli/shortcuts/common"
)

const (
	controlledActionModeLocal = "local"
	controlledActionModeAuto  = "auto"

	reviewExecutionPathGatewayDirect     = "gateway_direct"
	reviewExecutionPathLocalConfirmation = "local_confirmation"

	reviewFallbackModeLocal             = "mode_local"
	reviewFallbackCredentialUnavailable = "credential_unavailable"
	reviewFallbackIdentityUnverified    = "identity_unverified"
	reviewFallbackIdentityMismatch      = "identity_mismatch"
)

func (e *ReviewGatewayExecutor) prepareControlledReviewAction(ctx context.Context, job ReviewGatewayJob, result ReviewGatewayExecutionResult, now time.Time) (ReviewGatewayExecutionResult, error) {
	if e.ActionPlans == nil {
		return reviewGatewayExecutionFailure(result, fmt.Errorf("review action plan store is required"))
	}
	if strings.TrimSpace(job.GitLinkLogin) == "" {
		return reviewGatewayExecutionFailure(result, fmt.Errorf("当前飞书用户尚未绑定 GitLink 身份"))
	}
	owner, repo, err := splitReviewGatewayRepository(job.Repository)
	if err != nil {
		return reviewGatewayExecutionFailure(result, err)
	}
	provider := e.DataProvider
	if provider == nil {
		provider = GitLinkReviewDataProvider{}
	}
	data, err := provider.FetchReviewData(ctx, e.Runtime, ReviewDataRequest{Owner: owner, Repository: repo, PullRequest: job.PRNumber})
	if err != nil {
		return reviewGatewayExecutionFailure(result, err)
	}
	if data.Partial || data.CollectionStatus != "complete" || data.HeadSHA == "" || data.SourceFingerprint == "" {
		return reviewGatewayExecutionFailure(result, fmt.Errorf("生成写操作计划前必须取得完整 PR 数据、当前 Head 和来源指纹"))
	}
	if data.State != "open" {
		return reviewGatewayExecutionFailure(result, fmt.Errorf("只有开放中的 PR 可以生成写操作计划"))
	}
	action := reviewActionFromPrepareJob(job.Action)
	content := strings.TrimSpace(job.Argument)
	plan, err := NewReviewActionPlan(job, data, action, content, now)
	if err != nil {
		return reviewGatewayExecutionFailure(result, err)
	}
	plan, err = e.ActionPlans.CreateReviewActionPlan(ctx, plan)
	if err != nil {
		return reviewGatewayExecutionFailure(result, err)
	}
	result.ActionPlan = &plan
	result.ReviewData = &data
	result.HeadSHA = data.HeadSHA
	result.SourceFingerprint = data.SourceFingerprint
	result.Message = fmt.Sprintf("%s PR #%d 操作计划已生成；当前尚未修改 GitLink。", reviewActionDisplayName(action), job.PRNumber)
	mode := strings.ToLower(strings.TrimSpace(e.ControlledActionMode))
	if mode == "" {
		mode = controlledActionModeLocal
	}
	if mode != controlledActionModeAuto {
		result.ExecutionPath = reviewExecutionPathLocalConfirmation
		result.FallbackReason = reviewFallbackModeLocal
		return result, nil
	}
	if e.Runtime == nil || e.Runtime.Client == nil {
		result.ExecutionPath = reviewExecutionPathLocalConfirmation
		result.FallbackReason = reviewFallbackCredentialUnavailable
		return result, nil
	}
	actualLogin, identityErr := fetchCurrentGitLinkLogin(e.Runtime)
	if identityErr != nil {
		result.ExecutionPath = reviewExecutionPathLocalConfirmation
		result.FallbackReason = reviewFallbackIdentityUnverified
		return result, nil
	}
	if !strings.EqualFold(actualLogin, plan.GitLinkLogin) {
		result.ExecutionPath = reviewExecutionPathLocalConfirmation
		result.FallbackReason = reviewFallbackIdentityMismatch
		return result, nil
	}
	result.ExecutionPath = reviewExecutionPathGatewayDirect
	updated, executeErr := executeReviewActionPlan(ctx, e.Runtime, e.ActionPlans, provider, plan, false, now)
	result.ActionPlan = &updated
	result.ReadOnlyGitLink = false
	result.MutatesGitLink = updated.Status == "completed"
	if executeErr != nil {
		if updated.Status == "pending_confirmation" {
			_ = e.ActionPlans.FailPendingReviewActionPlan(ctx, plan.PlanID, executeErr, now)
			updated, _ = e.ActionPlans.GetReviewActionPlan(ctx, plan.PlanID)
			result.ActionPlan = &updated
		}
		result.Status = "failed"
		result.Error = redactReviewGatewayError(executeErr.Error())
		return result, nil
	}
	result.Message = fmt.Sprintf("%s PR #%d 已完成并通过 GitLink 回读验证。", reviewActionDisplayName(action), job.PRNumber)
	return result, nil
}

func reviewActionFromPrepareJob(action string) string {
	switch action {
	case "prepare_common_review":
		return reviewActionCommon
	case "prepare_review_approve":
		return reviewActionApprove
	case "prepare_review_reject":
		return reviewActionReject
	case "prepare_reject_close":
		return reviewActionRejectClose
	case "prepare_merge":
		return reviewActionMerge
	default:
		return ""
	}
}

func reviewActionDisplayName(action string) string {
	switch action {
	case reviewActionCommon:
		return "提交审查意见"
	case reviewActionApprove:
		return "批准"
	case reviewActionReject:
		return "需要修改"
	case reviewActionRejectClose:
		return "拒绝并关闭"
	case reviewActionMerge:
		return "合并"
	default:
		return "待确认"
	}
}

func executeReviewActionPlan(
	ctx context.Context,
	runtime *common.RuntimeContext,
	store *SQLiteReviewGatewayStore,
	provider ReviewDataProvider,
	plan ReviewActionPlan,
	dryRun bool,
	now time.Time,
) (ReviewActionPlan, error) {
	if plan.Status != "pending_confirmation" {
		return plan, fmt.Errorf("review action plan is not pending confirmation")
	}
	expires, err := time.Parse(time.RFC3339Nano, plan.ExpiresAt)
	if err != nil || !expires.After(now) {
		return plan, fmt.Errorf("review action plan expired")
	}
	if dryRun {
		return plan, nil
	}
	if runtime == nil || runtime.Client == nil {
		return plan, fmt.Errorf("GitLink runtime is required")
	}
	login, err := fetchCurrentGitLinkLogin(runtime)
	if err != nil {
		return plan, err
	}
	if !strings.EqualFold(login, plan.GitLinkLogin) {
		return plan, fmt.Errorf("current GitLink credential belongs to %q, expected %q", login, plan.GitLinkLogin)
	}
	owner, repo, err := splitReviewGatewayRepository(plan.Repository)
	if err != nil {
		return plan, err
	}
	if provider == nil {
		provider = GitLinkReviewDataProvider{}
	}
	data, err := provider.FetchReviewData(ctx, runtime, ReviewDataRequest{Owner: owner, Repository: repo, PullRequest: plan.PRNumber})
	if err != nil {
		return plan, err
	}
	if data.Partial || data.CollectionStatus != "complete" || data.HeadSHA != plan.ExpectedHeadSHA || data.SourceFingerprint != plan.SourceFingerprint {
		return plan, fmt.Errorf("review action plan is stale; PR head or source fingerprint changed")
	}
	if plan.Action == reviewActionMerge {
		ciState, err := fetchControlledMergeCIState(runtime, owner, repo, plan.PRNumber)
		if err != nil {
			return plan, err
		}
		if controlledReviewCIHasFailed(ciState) {
			return plan, fmt.Errorf("merge is blocked by an explicitly failed CI state")
		}
	}
	leaseOwner := "local-confirm-" + plan.RequestID
	plan, err = store.AcquireReviewActionPlan(ctx, plan.PlanID, leaseOwner, now, time.Minute)
	if err != nil {
		return plan, err
	}
	reviewID, mutationErr := mutateGitLinkReviewAction(runtime, plan)
	if mutationErr != nil {
		if isGitLinkPermissionFailure(mutationErr) {
			_ = store.FailReviewActionPlan(ctx, plan.PlanID, leaseOwner, mutationErr, now)
			updated, _ := store.GetReviewActionPlan(ctx, plan.PlanID)
			return updated, fmt.Errorf("GitLink 拒绝该操作；当前身份没有所需权限，或当前 PR 状态不允许执行: %w", mutationErr)
		}
		_ = store.MarkReviewActionPlanUnknown(ctx, plan.PlanID, leaseOwner, mutationErr, now)
		updated, _ := store.GetReviewActionPlan(ctx, plan.PlanID)
		return updated, fmt.Errorf("GitLink 写入结果暂无法确认，已停止自动重试: %w", mutationErr)
	}
	verifiedID, readbackErr := verifyGitLinkReviewAction(ctx, runtime, provider, plan, reviewID)
	if readbackErr != nil {
		_ = store.MarkReviewActionPlanUnknown(ctx, plan.PlanID, leaseOwner, readbackErr, now)
		updated, _ := store.GetReviewActionPlan(ctx, plan.PlanID)
		return updated, fmt.Errorf("GitLink 写入后回读未确认，已停止自动重试: %w", readbackErr)
	}
	if err := store.CompleteReviewActionPlan(ctx, plan.PlanID, leaseOwner, verifiedID, now); err != nil {
		_ = store.MarkReviewActionPlanUnknown(ctx, plan.PlanID, leaseOwner, err, now)
		updated, _ := store.GetReviewActionPlan(ctx, plan.PlanID)
		return updated, fmt.Errorf("GitLink 可能已写入但本地完成状态保存失败: %w", err)
	}
	return store.GetReviewActionPlan(ctx, plan.PlanID)
}

func isGitLinkPermissionFailure(err error) bool {
	var apiErr *client.APIError
	if !errors.As(err, &apiErr) {
		return false
	}
	return apiErr.StatusCode == 401 || apiErr.StatusCode == 403
}

func fetchCurrentGitLinkLogin(runtime *common.RuntimeContext) (string, error) {
	envelope, err := runtime.CallAPI("GET", "/v1/users/me", nil)
	if err != nil {
		return "", err
	}
	item := reviewDataObject(envelope.Data)
	login := firstReviewDataString(item, "login", "username")
	if login == "" {
		return "", fmt.Errorf("GitLink /users/me response missing login")
	}
	return login, nil
}

func mutateGitLinkReviewAction(runtime *common.RuntimeContext, plan ReviewActionPlan) (string, error) {
	owner, repo, _ := splitReviewGatewayRepository(plan.Repository)
	switch plan.Action {
	case reviewActionCommon, reviewActionApprove, reviewActionReject:
		content := plan.Content
		marker := "Ref: " + plan.RequestID
		if !strings.Contains(content, marker) {
			content = strings.TrimSpace(content) + "\n\n" + marker
		}
		envelope, err := runtime.CallAPI("POST", fmt.Sprintf("/v1/%s/%s/pulls/%d/reviews", owner, repo, plan.PRNumber), map[string]interface{}{
			"content": content, "status": plan.ReviewStatus, "commit_id": plan.ExpectedHeadSHA,
		})
		if err != nil {
			return "", err
		}
		item := reviewDataObject(envelope.Data)
		return firstReviewDataString(item, "id", "review_id"), nil
	case reviewActionRejectClose:
		_, err := runtime.CallAPI("POST", fmt.Sprintf("/%s/%s/pulls/%d/refuse_merge", owner, repo, plan.PRNumber), nil)
		return "", err
	case reviewActionMerge:
		_, err := runtime.CallAPI("POST", fmt.Sprintf("/%s/%s/pulls/%d/pr_merge", owner, repo, plan.PRNumber), map[string]interface{}{"do": "merge"})
		return "", err
	default:
		return "", fmt.Errorf("unsupported review action %q", plan.Action)
	}
}

func verifyGitLinkReviewAction(ctx context.Context, runtime *common.RuntimeContext, provider ReviewDataProvider, plan ReviewActionPlan, reviewID string) (string, error) {
	owner, repo, _ := splitReviewGatewayRepository(plan.Repository)
	var lastErr error
	for attempt := 0; attempt < 3; attempt++ {
		data, err := provider.FetchReviewData(ctx, runtime, ReviewDataRequest{Owner: owner, Repository: repo, PullRequest: plan.PRNumber})
		if err != nil {
			lastErr = err
		} else {
			switch plan.Action {
			case reviewActionCommon, reviewActionApprove, reviewActionReject:
				marker := "Ref: " + plan.RequestID
				for _, review := range data.Reviews {
					if review.Status == plan.ReviewStatus && review.CommitID == plan.ExpectedHeadSHA && strings.Contains(review.Content, marker) {
						if reviewID != "" && review.ID != reviewID {
							continue
						}
						return review.ID, nil
					}
				}
				lastErr = fmt.Errorf("created review was not found by bounded GET read-back")
			case reviewActionRejectClose:
				if data.State == "closed" {
					return "", nil
				}
				lastErr = fmt.Errorf("PR state is %q after close", data.State)
			case reviewActionMerge:
				if data.State == "merged" {
					return "", nil
				}
				lastErr = fmt.Errorf("PR state is %q after merge", data.State)
			}
		}
		if attempt < 2 {
			time.Sleep(time.Duration(attempt+1) * 50 * time.Millisecond)
		}
	}
	return "", lastErr
}

func fetchControlledMergeCIState(runtime *common.RuntimeContext, owner, repository string, number int) (string, error) {
	envelope, err := runtime.CallAPI("GET", fmt.Sprintf("/%s/%s/pulls/%d", owner, repository, number), nil)
	if err != nil {
		return "", fmt.Errorf("read merge eligibility: %w", err)
	}
	item := reviewDataObject(envelope.Data)
	state := firstReviewDataString(item, "ci_state", "pipeline_status", "build_status")
	for _, key := range []string{"ci_summary", "pipeline", "build"} {
		if nested, ok := item[key].(map[string]interface{}); ok {
			state = firstReviewDataValue(state, firstReviewDataString(nested, "state", "status"))
		}
	}
	return state, nil
}

func controlledReviewCIHasFailed(state string) bool {
	switch strings.ToLower(strings.TrimSpace(state)) {
	case "failed", "failure", "error":
		return true
	default:
		return false
	}
}
