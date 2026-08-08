package feishu

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/gitlink-org/gitlink-cli/shortcuts/common"
	"github.com/gitlink-org/gitlink-cli/shortcuts/workflow"
)

const reviewGatewayResultSchema = "feishu.review-result/v2"

type ReviewGatewayExecutionResult struct {
	SchemaVersion       string                        `json:"schema_version"`
	JobID               string                        `json:"job_id"`
	Status              string                        `json:"status"`
	Mode                string                        `json:"mode"`
	Action              string                        `json:"action"`
	Repository          string                        `json:"repository,omitempty"`
	PRNumber            int                           `json:"pr_number,omitempty"`
	RequestedBy         string                        `json:"requested_by"`
	ReadOnlyGitLink     bool                          `json:"read_only_gitlink"`
	MutatesGitLink      bool                          `json:"mutates_gitlink"`
	PublicRead          bool                          `json:"public_read,omitempty"`
	CompletedAt         string                        `json:"completed_at"`
	Message             string                        `json:"message,omitempty"`
	CollectionStatus    string                        `json:"collection_status,omitempty"`
	Partial             bool                          `json:"partial,omitempty"`
	HeadSHA             string                        `json:"head_sha,omitempty"`
	SourceFingerprint   string                        `json:"source_fingerprint,omitempty"`
	ReviewStage         string                        `json:"review_stage,omitempty"`
	Decision            string                        `json:"decision,omitempty"`
	GitLinkState        string                        `json:"gitlink_state,omitempty"`
	ReviewCount         int                           `json:"review_count,omitempty"`
	ThreadCount         int                           `json:"thread_count,omitempty"`
	OpenThreadCount     int                           `json:"open_thread_count,omitempty"`
	PullRequest         *ReviewGatewayPullRequestView `json:"pull_request,omitempty"`
	ResultCard          Card                          `json:"result_card,omitempty"`
	Queue               *ReviewGatewayQueueView       `json:"queue,omitempty"`
	SnapshotPlan        *ReviewSnapshotPlan           `json:"snapshot_plan,omitempty"`
	Draft               *ReviewDraftPreview           `json:"draft,omitempty"`
	Collaboration       *ReviewCollaborationBundle    `json:"collaboration,omitempty"`
	CollaborationItems  []ReviewCollaborationItem     `json:"collaboration_items,omitempty"`
	ActionPlan          *ReviewActionPlan             `json:"action_plan,omitempty"`
	WriteResult         *ReviewWriteResult            `json:"write_result,omitempty"`
	Subscriptions       []ReviewChatSubscription      `json:"subscriptions,omitempty"`
	ResourceSync        []ReviewResourceSyncResult    `json:"resource_sync,omitempty"`
	AgentRun            *workflow.ReviewAgentRun      `json:"agent_run,omitempty"`
	Warnings            []string                      `json:"warnings,omitempty"`
	Error               string                        `json:"error,omitempty"`
	AttemptCount        int                           `json:"attempt_count,omitempty"`
	ConfirmationStateDB string                        `json:"-"`
}

// ReviewGatewayPullRequestView is the bounded presentation contract consumed
// by Feishu cards and text fallbacks. It intentionally excludes raw API
// payloads, full message/user identifiers, credentials, and unbounded content.
type ReviewGatewayPullRequestView struct {
	Title               string                      `json:"title,omitempty"`
	Author              string                      `json:"author,omitempty"`
	BaseBranch          string                      `json:"base_branch,omitempty"`
	HeadBranch          string                      `json:"head_branch,omitempty"`
	GitLinkURL          string                      `json:"gitlink_url,omitempty"`
	PatchsetID          string                      `json:"patchset_id,omitempty"`
	FilesCount          int                         `json:"files_count"`
	CommitsCount        int                         `json:"commits_count"`
	Additions           int                         `json:"additions"`
	Deletions           int                         `json:"deletions"`
	RiskLevel           string                      `json:"risk_level,omitempty"`
	RecommendedNextStep string                      `json:"recommended_next_step,omitempty"`
	Unknowns            []string                    `json:"unknowns,omitempty"`
	Reviewers           []ReviewGatewayReviewerView `json:"reviewers,omitempty"`
}

type ReviewGatewayReviewerView struct {
	Reviewer string `json:"reviewer"`
	Decision string `json:"decision"`
}

type ReviewGatewayQueueView struct {
	TotalPRs       int                      `json:"total_prs"`
	HighPriority   int                      `json:"high_priority"`
	MediumPriority int                      `json:"medium_priority"`
	LowPriority    int                      `json:"low_priority"`
	TopItems       []ReviewGatewayQueueItem `json:"top_items"`
}

type ReviewGatewayQueueItem struct {
	Rank        int      `json:"rank"`
	Number      int      `json:"number,omitempty"`
	Title       string   `json:"title"`
	Priority    string   `json:"priority"`
	RiskLevel   string   `json:"risk_level"`
	Reasons     []string `json:"reasons,omitempty"`
	ReviewFocus []string `json:"review_focus,omitempty"`
}

type ReviewDraftPreview struct {
	TemplateVersion string                `json:"template_version"`
	Title           string                `json:"title"`
	Summary         string                `json:"summary"`
	Decision        string                `json:"decision"`
	ReviewerStates  []ReviewDraftReviewer `json:"reviewer_states"`
	OpenThreads     []ReviewDraftThread   `json:"open_threads"`
	Unknowns        []string              `json:"unknowns"`
	NextStep        string                `json:"next_step"`
}

type ReviewDraftReviewer struct {
	Reviewer string `json:"reviewer"`
	Decision string `json:"decision"`
}

type ReviewDraftThread struct {
	Author  string `json:"author,omitempty"`
	Content string `json:"content,omitempty"`
	Path    string `json:"path,omitempty"`
	Line    string `json:"line,omitempty"`
}

type ReviewGatewayExecutor struct {
	Runtime                    *common.RuntimeContext
	Now                        func() time.Time
	Collaboration              ReviewCollaborationStore
	ActionPlans                ReviewActionPlanStore
	Subscriptions              *SQLiteReviewGatewayStore
	IdentityBindings           []ReviewIdentityBinding
	EnableGitLinkWrite         bool
	Publisher                  ReviewCollaborationPublisher
	Installations              map[string]GitLinkInstallation
	RequireInstallationRuntime bool
	AgentProvider              workflow.ReviewAgentProvider
	AgentTaskTimeout           time.Duration
	AgentMaxConcurrency        int
	ConfirmationStateDB        string
}

func (e *ReviewGatewayExecutor) Execute(ctx context.Context, job ReviewGatewayJob) (ReviewGatewayExecutionResult, error) {
	now := e.Now
	if now == nil {
		now = time.Now
	}
	result := ReviewGatewayExecutionResult{
		SchemaVersion:       reviewGatewayResultSchema,
		JobID:               job.JobID,
		Status:              "completed",
		Mode:                firstNonEmpty(job.Mode, "preview"),
		Action:              job.Action,
		Repository:          job.Repository,
		PRNumber:            job.PRNumber,
		RequestedBy:         job.RequestedBy,
		ReadOnlyGitLink:     true,
		MutatesGitLink:      false,
		PublicRead:          job.PublicRead,
		CompletedAt:         now().UTC().Format(time.RFC3339),
		AttemptCount:        job.AttemptCount,
		ConfirmationStateDB: firstNonEmpty(e.ConfirmationStateDB, ".local/review-gateway.db"),
	}
	select {
	case <-ctx.Done():
		result.Status = "failed"
		result.Error = redactReviewGatewayError(ctx.Err().Error())
		return result, ctx.Err()
	default:
	}
	if job.PublicRead && !reviewGatewayActionAllowsPublicRepository(job.Action) {
		return reviewGatewayExecutionFailure(
			result,
			fmt.Errorf("public repository discovery is not allowed for action %q", job.Action),
		)
	}

	switch job.Action {
	case "read_review_queue":
		runtime, runtimeErr := e.runtimeForJob(job)
		if runtimeErr != nil {
			return reviewGatewayExecutionFailure(result, runtimeErr)
		}
		owner, repo, err := splitReviewGatewayRepository(job.Repository)
		if err != nil {
			return reviewGatewayExecutionFailure(result, err)
		}
		queue, err := workflow.FetchReviewQueue(runtime, workflow.ReviewQueueFetchOptions{
			Owner:     owner,
			Repo:      repo,
			State:     "open",
			StartPage: 1,
			PageSize:  30,
			MaxItems:  100,
			Language:  "zh-CN",
		})
		if err != nil {
			return reviewGatewayExecutionFailure(result, err)
		}
		result.Queue = buildReviewGatewayQueueView(queue)
		result.Message = "已完成 GitLink GET-only Review Queue 读取。"
		return result, nil

	case "read_review_context", "refresh_review_context", "generate_review_draft", "run_agent_review":
		runtime, runtimeErr := e.runtimeForJob(job)
		if runtimeErr != nil {
			return reviewGatewayExecutionFailure(result, runtimeErr)
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
			IncludeRepo:     true,
			IncludePR:       true,
			IncludeFiles:    true,
			IncludeVersions: true,
			IncludeReviews:  true,
			IncludeThreads:  true,
		})
		if err != nil {
			return reviewGatewayExecutionFailure(result, err)
		}
		if job.PublicRead && !reviewContextRepositoryIsPublic(reviewContext) {
			return reviewGatewayExecutionFailure(result, fmt.Errorf("public repository status could not be verified"))
		}
		populateReviewGatewayContextResult(&result, reviewContext)
		result.ResultCard = buildReviewGatewayResultCard(job, result, nil)
		plan := PlanReviewSnapshotSync(reviewContext, nil)
		result.SnapshotPlan = &plan
		if !job.PublicRead && e.Collaboration != nil {
			item, collaborationErr := e.Collaboration.UpsertCollaborationFacts(ctx, job, result, now().UTC())
			if collaborationErr != nil {
				return reviewGatewayExecutionFailure(result, collaborationErr)
			}
			bundle := BuildReviewCollaborationBundle(item)
			bundle.Card = buildReviewGatewayResultCard(job, result, &item)
			result.Collaboration = &bundle
			result.ResultCard = bundle.Card
			e.publishCollaboration(ctx, &result, bundle)
		} else if job.PublicRead && e.Collaboration != nil {
			if _, presentationErr := e.Collaboration.SaveReviewPRPresentation(ctx, job, result, now().UTC()); presentationErr != nil {
				return reviewGatewayExecutionFailure(result, presentationErr)
			}
		}
		result.Message = "已完成 GitLink GET-only PR 上下文读取；未执行 Review、评论、Reviewer 或合并写入。"
		if job.PublicRead {
			result.Message = "已完成无凭据 GitLink 公共 PR 读取；未创建协作资源，GitLink 写入 0。"
		}
		if job.Action == "generate_review_draft" {
			draft := buildReviewDraftPreview(reviewContext)
			result.Draft = &draft
			result.Message = "已生成确定性 Review 草稿模板；草稿不会写回 GitLink。"
		}
		if job.Action == "run_agent_review" {
			if e.AgentProvider == nil {
				return reviewGatewayExecutionFailure(result, fmt.Errorf("Review Agent runner is not enabled"))
			}
			maxConcurrency := e.AgentMaxConcurrency
			if maxConcurrency <= 0 {
				maxConcurrency = 3
			}
			plan := workflow.BuildReviewAgentPlan(reviewContext, nil, maxConcurrency, now().UTC())
			run := workflow.RunReviewAgentPlan(
				ctx,
				plan,
				e.AgentProvider,
				e.AgentTaskTimeout,
				now,
			)
			result.AgentRun = &run
			result.Message = fmt.Sprintf(
				"Agent 审查状态：%s；收到 %d/%d 份有效评估；建议由维护团队完成最终复核；本次操作未修改 GitLink。",
				run.Status,
				run.Synthesis.AssessmentsReceived,
				run.Synthesis.AssessmentsExpected,
			)
			return result, nil
		}
		return result, nil

	case "prepare_common_review", "prepare_review_approve", "prepare_review_reject", "prepare_reject_close", "prepare_merge":
		return e.prepareControlledReviewAction(ctx, job, result, now().UTC())

	case "confirm_common_review":
		return e.confirmCommonReview(ctx, job, result, now().UTC())

	case "show_local_review_plan", "cancel_common_review":
		if e.ActionPlans == nil {
			return reviewGatewayExecutionFailure(result, fmt.Errorf("review action plan store is required"))
		}
		plan, err := e.ActionPlans.GetReviewActionPlan(ctx, job.Argument)
		if err != nil || plan.ActorID != job.RequestedBy || plan.SourceChatID != job.ChatID {
			return reviewGatewayExecutionFailure(result, fmt.Errorf("review action plan is unavailable in this scope"))
		}
		if job.Action == "cancel_common_review" {
			plan, err = e.ActionPlans.ClaimReviewActionPlan(ctx, ReviewActionPlanClaimOptions{PlanID: plan.PlanID, ActorID: job.RequestedBy, LeaseOwner: "cancel:" + job.JobID, Now: now().UTC()})
			if err == nil {
				err = e.ActionPlans.FinishReviewActionPlan(ctx, plan.PlanID, "cancelled", "", "", reviewMutationNone, now().UTC())
			}
			if err != nil {
				return reviewGatewayExecutionFailure(result, err)
			}
			plan.Status = "cancelled"
		}
		result.ActionPlan, result.Repository, result.PRNumber = &plan, plan.Repository, plan.PRNumber
		if job.Action == "cancel_common_review" {
			result.Message = "操作已取消；本次未修改 GitLink。"
		} else {
			result.Message = "已显示本地确认方式；查看操作不会修改 GitLink。"
		}
		result.ResultCard = buildReviewGatewayResultCard(job, result, nil)
		return result, nil

	case "subscribe_review_events", "unsubscribe_review_events", "set_review_notification_mode":
		if e.Subscriptions == nil {
			return reviewGatewayExecutionFailure(result, fmt.Errorf("review subscription store is required"))
		}
		change := ReviewSubscriptionChange{
			InstallationID: job.InstallationID,
			ChatID:         job.ChatID,
			Repository:     job.Repository,
			ActorID:        job.RequestedBy,
		}
		switch job.Action {
		case "subscribe_review_events":
			change.AddGroups = splitReviewSubscriptionArgument(job.Argument)
		case "unsubscribe_review_events":
			change.RemoveGroups = splitReviewSubscriptionArgument(job.Argument)
		case "set_review_notification_mode":
			change.NotificationMode = job.Argument
		}
		subscription, err := e.Subscriptions.ChangeReviewChatSubscription(ctx, change, now().UTC())
		if err != nil {
			return reviewGatewayExecutionFailure(result, err)
		}
		result.Subscriptions = []ReviewChatSubscription{subscription}
		result.Message = fmt.Sprintf("Review subscription revision %d saved for %s.", subscription.Revision, subscription.Repository)

	case "show_review_subscriptions":
		if e.Subscriptions == nil {
			return reviewGatewayExecutionFailure(result, fmt.Errorf("review subscription store is required"))
		}
		subscriptions, err := e.Subscriptions.ListReviewChatSubscriptions(
			ctx, job.InstallationID, job.ChatID, job.Repository, false,
		)
		if err != nil {
			return reviewGatewayExecutionFailure(result, err)
		}
		result.Subscriptions = subscriptions
		result.Message = fmt.Sprintf("Loaded %d Review subscription(s).", len(subscriptions))

	case "set_default_review_repository":
		if e.Subscriptions == nil {
			return reviewGatewayExecutionFailure(result, fmt.Errorf("review subscription store is required"))
		}
		if err := e.Subscriptions.SetDefaultReviewRepository(
			ctx, job.InstallationID, job.ChatID, job.Repository, job.RequestedBy,
		); err != nil {
			return reviewGatewayExecutionFailure(result, err)
		}
		result.Message = fmt.Sprintf("Default Review repository set to %s.", job.Repository)

	case "help":
		result.Message = reviewGatewayHelpText()
	case "show_binding":
		result.Message = formatReviewGatewayRepositoryBindings(job.Repositories, job.Repository)
	case "list_repositories":
		result.Message = formatReviewGatewayRepositoryBindings(job.Repositories, job.Repository)
	case "read_my_review_tasks":
		if e.Collaboration == nil {
			return reviewGatewayExecutionFailure(result, fmt.Errorf("review collaboration store is required"))
		}
		items, err := e.Collaboration.ListCollaborationItems(ctx, job.InstallationID, job.ChatID, job.Repository, job.RequestedBy)
		if err != nil {
			return reviewGatewayExecutionFailure(result, err)
		}
		result.CollaborationItems = items
		result.Message = fmt.Sprintf("已读取 %d 个由当前飞书账号认领的 Review 任务。", len(items))
	case "plan_bind_repository":
		result.Message = fmt.Sprintf("已生成仓库绑定变更计划：%s；该命令仅供离线预览，长连接首次绑定必须由管理员预配置。", job.Repository)
	case "claim_review", "release_review", "set_review_deadline":
		if e.Collaboration == nil {
			return reviewGatewayExecutionFailure(result, fmt.Errorf("review collaboration store is required"))
		}
		presentation, err := e.Collaboration.GetReviewPRPresentation(ctx, job)
		if err != nil {
			if errors.Is(err, ErrReviewPRPresentationNotFound) {
				return reviewGatewayExecutionFailure(result, ErrCompleteReviewPRPresentationRequired)
			}
			return reviewGatewayExecutionFailure(result, err)
		}
		if !presentation.Complete() || !presentation.Matches(job) {
			return reviewGatewayExecutionFailure(result, ErrCompleteReviewPRPresentationRequired)
		}
		result, err = reviewGatewayResultFromPresentation(job, presentation)
		if err != nil {
			return reviewGatewayExecutionFailure(result, err)
		}
		item, err := e.Collaboration.ApplyCollaborationAction(ctx, job, now().UTC())
		if err != nil {
			return reviewGatewayExecutionFailure(result, err)
		}
		bundle := BuildReviewCollaborationBundle(item)
		bundle.Card = buildReviewGatewayResultCard(job, result, &item)
		result.Collaboration = &bundle
		result.ResultCard = bundle.Card
		e.publishCollaboration(ctx, &result, bundle)
		result.Message = fmt.Sprintf(
			"PR #%d 协作状态已更新为 %s；变更仅发生在协作状态库，GitLink 写入为 0。",
			job.PRNumber,
			item.CollaborationStatus,
		)
	default:
		return reviewGatewayExecutionFailure(result, fmt.Errorf("unsupported review gateway action %q", job.Action))
	}
	return result, nil
}

func formatReviewGatewayRepositoryBindings(repositories []string, defaultRepository string) string {
	if len(repositories) == 0 {
		return "当前群没有可用仓库绑定。"
	}
	lines := []string{"当前群已绑定仓库："}
	for _, repository := range repositories {
		suffix := ""
		if repository == defaultRepository {
			suffix = "（默认）"
		}
		lines = append(lines, "- "+repository+suffix)
	}
	lines = append(lines, "多仓库命令示例：查看 owner/repo PR #编号。")
	return strings.Join(lines, "\n")
}

func (e *ReviewGatewayExecutor) publishCollaboration(
	ctx context.Context,
	result *ReviewGatewayExecutionResult,
	bundle ReviewCollaborationBundle,
) {
	if e.Publisher == nil || result == nil {
		return
	}
	switch result.Action {
	case "claim_review", "release_review", "set_review_deadline":
		bundle.HumanFieldsAuthoritative = true
	}
	syncResults, err := e.Publisher.Publish(ctx, bundle)
	result.ResourceSync = syncResults
	if err != nil {
		result.Warnings = append(result.Warnings, redactReviewGatewayError(err.Error()))
	}
	for _, syncResult := range syncResults {
		if syncResult.Error != "" {
			result.Warnings = append(
				result.Warnings,
				fmt.Sprintf("%s: %s", syncResult.Resource, syncResult.Error),
			)
		}
	}
}

func reviewGatewayExecutionFailure(result ReviewGatewayExecutionResult, err error) (ReviewGatewayExecutionResult, error) {
	result.Status = "failed"
	result.Error = redactReviewGatewayError(err.Error())
	return result, fmt.Errorf("%s", result.Error)
}

func splitReviewGatewayRepository(repository string) (string, string, error) {
	parts := strings.Split(strings.TrimSpace(repository), "/")
	if len(parts) != 2 || strings.TrimSpace(parts[0]) == "" || strings.TrimSpace(parts[1]) == "" {
		return "", "", fmt.Errorf("review gateway repository %q must use owner/repo", repository)
	}
	return parts[0], parts[1], nil
}

func populateReviewGatewayContextResult(result *ReviewGatewayExecutionResult, reviewContext workflow.ReviewContext) {
	result.CollectionStatus = reviewContext.CollectionStatus
	result.Partial = reviewContext.Partial
	result.HeadSHA = reviewContext.CurrentHeadSHA
	result.SourceFingerprint = reviewContext.WorkItem.SourceFingerprint
	result.ReviewStage = reviewContext.WorkItem.ReviewStage
	result.Decision = reviewContext.Summary.Decision
	result.GitLinkState = reviewContext.WorkItem.GitLinkState
	result.ReviewCount = reviewContext.Summary.TotalReviews
	result.ThreadCount = reviewContext.Summary.TotalThreads
	result.OpenThreadCount = reviewContext.Summary.OpenThreads
	view := ReviewGatewayPullRequestView{
		Title: truncateReviewGatewayText(firstNonEmpty(
			reviewContext.WorkItem.Title,
			reviewContext.WorkItem.HeadBranch,
			fmt.Sprintf("PR #%d", reviewContext.PullRequest),
		), 160),
		Author:              truncateReviewGatewayText(reviewContext.WorkItem.Author, 80),
		BaseBranch:          truncateReviewGatewayText(reviewContext.WorkItem.BaseBranch, 100),
		HeadBranch:          truncateReviewGatewayText(reviewContext.WorkItem.HeadBranch, 100),
		GitLinkURL:          reviewGatewayGitLinkURL(reviewContext.Repository, reviewContext.PullRequest),
		PatchsetID:          truncateReviewGatewayText(reviewContext.CurrentVersionID, 48),
		FilesCount:          reviewContext.CurrentPatchset.FilesCount,
		CommitsCount:        reviewContext.CurrentPatchset.CommitsCount,
		Additions:           reviewContext.CurrentPatchset.Additions,
		Deletions:           reviewContext.CurrentPatchset.Deletions,
		RiskLevel:           truncateReviewGatewayText(reviewContext.WorkItem.RiskLevel, 32),
		RecommendedNextStep: truncateReviewGatewayText(reviewContext.WorkItem.RecommendedNextStep, 240),
		Unknowns:            []string{},
		Reviewers:           []ReviewGatewayReviewerView{},
	}
	if result.GitLinkState == "closed" || result.GitLinkState == "merged" {
		result.ReviewStage = result.GitLinkState
		result.Decision = "none"
		view.RecommendedNextStep = "none"
	}
	for _, unknown := range reviewContext.WorkItem.Unknowns {
		if len(view.Unknowns) >= 5 {
			break
		}
		unknown = truncateReviewGatewayText(redactReviewGatewayError(unknown), 180)
		if unknown != "" {
			view.Unknowns = append(view.Unknowns, unknown)
		}
	}
	for _, reviewer := range reviewContext.ReviewerSummaries {
		if len(view.Reviewers) >= 8 {
			break
		}
		name := truncateReviewGatewayText(firstNonEmpty(reviewer.Actor, reviewer.ReviewerKey), 64)
		if name == "" {
			continue
		}
		view.Reviewers = append(view.Reviewers, ReviewGatewayReviewerView{
			Reviewer: name,
			Decision: truncateReviewGatewayText(firstNonEmpty(reviewer.CurrentDecision, "unknown"), 32),
		})
	}
	result.PullRequest = &view
}

func reviewGatewayGitLinkURL(repository string, number int) string {
	owner, repo, err := splitReviewGatewayRepository(repository)
	if err != nil || number <= 0 {
		return ""
	}
	return fmt.Sprintf("https://www.gitlink.org.cn/%s/%s/pulls/%d", owner, repo, number)
}

func reviewContextRepositoryIsPublic(reviewContext workflow.ReviewContext) bool {
	value, exists := reviewContext.RepositoryInfo["is_public"]
	if !exists {
		return false
	}
	public, ok := value.(bool)
	return ok && public
}

func buildReviewGatewayQueueView(queue workflow.ReviewQueueResult) *ReviewGatewayQueueView {
	view := &ReviewGatewayQueueView{
		TotalPRs:       queue.TotalPRs,
		HighPriority:   queue.HighPriority,
		MediumPriority: queue.MediumPriority,
		LowPriority:    queue.LowPriority,
		TopItems:       []ReviewGatewayQueueItem{},
	}
	limit := len(queue.Items)
	if limit > 10 {
		limit = 10
	}
	for _, item := range queue.Items[:limit] {
		view.TopItems = append(view.TopItems, ReviewGatewayQueueItem{
			Rank:        item.Rank,
			Number:      item.Number,
			Title:       item.Title,
			Priority:    item.Priority,
			RiskLevel:   item.RiskLevel,
			Reasons:     append([]string(nil), item.Reasons...),
			ReviewFocus: append([]string(nil), item.ReviewFocus...),
		})
	}
	return view
}

func buildReviewDraftPreview(reviewContext workflow.ReviewContext) ReviewDraftPreview {
	title := reviewContext.WorkItem.Title
	if strings.TrimSpace(title) == "" {
		title = fmt.Sprintf("PR #%d", reviewContext.PullRequest)
	}
	draft := ReviewDraftPreview{
		TemplateVersion: "review-draft/v1",
		Title:           fmt.Sprintf("%s #%d — %s", reviewContext.Repository, reviewContext.PullRequest, title),
		Summary: fmt.Sprintf(
			"当前 patchset %s，%d 个文件，%d 次 Review，%d 个未解决线程。",
			firstNonEmpty(reviewContext.CurrentVersionID, "unknown"),
			reviewContext.CurrentPatchset.FilesCount,
			reviewContext.Summary.TotalReviews,
			reviewContext.Summary.OpenThreads,
		),
		Decision:       reviewContext.Summary.Decision,
		ReviewerStates: []ReviewDraftReviewer{},
		OpenThreads:    []ReviewDraftThread{},
		Unknowns:       append([]string(nil), reviewContext.WorkItem.Unknowns...),
		NextStep:       reviewContext.WorkItem.RecommendedNextStep,
	}
	for _, reviewer := range reviewContext.ReviewerSummaries {
		name := firstNonEmpty(reviewer.Actor, reviewer.ActorID, reviewer.ReviewerKey)
		draft.ReviewerStates = append(draft.ReviewerStates, ReviewDraftReviewer{
			Reviewer: name,
			Decision: reviewer.CurrentDecision,
		})
	}
	for _, thread := range reviewContext.Threads {
		if len(draft.OpenThreads) >= 10 {
			break
		}
		if strings.EqualFold(thread.State, "resolved") || strings.EqualFold(thread.Freshness, "outdated") {
			continue
		}
		draft.OpenThreads = append(draft.OpenThreads, ReviewDraftThread{
			Author:  firstNonEmpty(thread.Author, thread.AuthorID),
			Content: truncateReviewGatewayText(thread.Content, 400),
			Path:    thread.Path,
			Line:    thread.LineCode,
		})
	}
	return draft
}

func truncateReviewGatewayText(value string, max int) string {
	value = strings.TrimSpace(value)
	runes := []rune(value)
	if max > 0 && len(runes) > max {
		return string(runes[:max]) + "…"
	}
	return value
}
