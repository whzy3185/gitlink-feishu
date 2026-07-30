package feishu

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/gitlink-org/gitlink-cli/shortcuts/common"
	"github.com/gitlink-org/gitlink-cli/shortcuts/workflow"
)

const reviewGatewayResultSchema = "feishu.review-result/v1"

type ReviewGatewayExecutionResult struct {
	SchemaVersion     string                  `json:"schema_version"`
	JobID             string                  `json:"job_id"`
	Status            string                  `json:"status"`
	Mode              string                  `json:"mode"`
	Action            string                  `json:"action"`
	Repository        string                  `json:"repository,omitempty"`
	PRNumber          int                     `json:"pr_number,omitempty"`
	RequestedBy       string                  `json:"requested_by"`
	ReadOnlyGitLink   bool                    `json:"read_only_gitlink"`
	MutatesGitLink    bool                    `json:"mutates_gitlink"`
	CompletedAt       string                  `json:"completed_at"`
	Message           string                  `json:"message,omitempty"`
	CollectionStatus  string                  `json:"collection_status,omitempty"`
	Partial           bool                    `json:"partial,omitempty"`
	HeadSHA           string                  `json:"head_sha,omitempty"`
	SourceFingerprint string                  `json:"source_fingerprint,omitempty"`
	ReviewStage       string                  `json:"review_stage,omitempty"`
	Decision          string                  `json:"decision,omitempty"`
	ReviewCount       int                     `json:"review_count,omitempty"`
	ThreadCount       int                     `json:"thread_count,omitempty"`
	OpenThreadCount   int                     `json:"open_thread_count,omitempty"`
	Queue             *ReviewGatewayQueueView `json:"queue,omitempty"`
	SnapshotPlan      *ReviewSnapshotPlan     `json:"snapshot_plan,omitempty"`
	Draft             *ReviewDraftPreview     `json:"draft,omitempty"`
	Error             string                  `json:"error,omitempty"`
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
	Runtime *common.RuntimeContext
	Now     func() time.Time
}

func (e *ReviewGatewayExecutor) Execute(ctx context.Context, job ReviewGatewayJob) (ReviewGatewayExecutionResult, error) {
	now := e.Now
	if now == nil {
		now = time.Now
	}
	result := ReviewGatewayExecutionResult{
		SchemaVersion:   reviewGatewayResultSchema,
		JobID:           job.JobID,
		Status:          "completed",
		Mode:            "preview",
		Action:          job.Action,
		Repository:      job.Repository,
		PRNumber:        job.PRNumber,
		RequestedBy:     job.RequestedBy,
		ReadOnlyGitLink: true,
		MutatesGitLink:  false,
		CompletedAt:     now().UTC().Format(time.RFC3339),
	}
	select {
	case <-ctx.Done():
		result.Status = "failed"
		result.Error = redactReviewGatewayError(ctx.Err().Error())
		return result, ctx.Err()
	default:
	}

	switch job.Action {
	case "read_review_queue":
		if e.Runtime == nil {
			return reviewGatewayExecutionFailure(result, fmt.Errorf("GitLink runtime is required"))
		}
		owner, repo, err := splitReviewGatewayRepository(job.Repository)
		if err != nil {
			return reviewGatewayExecutionFailure(result, err)
		}
		queue, err := workflow.FetchReviewQueue(e.Runtime, workflow.ReviewQueueFetchOptions{
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
		result.Message = "已完成 GitLink GET-only Review Queue 读取；结果仅输出为本地预览。"
		return result, nil

	case "read_review_context", "refresh_review_context", "generate_review_draft":
		if e.Runtime == nil {
			return reviewGatewayExecutionFailure(result, fmt.Errorf("GitLink runtime is required"))
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
		plan := PlanReviewSnapshotSync(reviewContext, nil)
		result.SnapshotPlan = &plan
		result.Message = "已完成 GitLink GET-only PR 上下文读取；未执行 Review、评论、Reviewer 或合并写入。"
		if job.Action == "generate_review_draft" {
			draft := buildReviewDraftPreview(reviewContext)
			result.Draft = &draft
			result.Message = "已生成确定性 Review 草稿模板；草稿仅本地预览，未写回 GitLink 或飞书。"
		}
		return result, nil

	case "help":
		result.Message = "支持：查看待审查、查看 PR #编号、刷新 PR #编号、生成 PR #编号 Review 草稿、查看绑定、领取 PR #编号。P2 当前仅执行 GitLink GET-only 读取与协作变更预览。"
	case "show_binding":
		result.Message = fmt.Sprintf("当前群已绑定仓库 %s；绑定来源是受控配置文件。", job.Repository)
	case "read_my_review_tasks":
		result.Message = "个人 Review 任务视图已进入任务模型，本阶段尚未写入飞书 Task。"
	case "plan_bind_repository":
		result.Message = fmt.Sprintf("已生成仓库绑定变更计划：%s；本阶段不会修改绑定配置。", job.Repository)
	case "plan_claim_review":
		result.Message = fmt.Sprintf("已生成 PR #%d 认领变更计划；本阶段不会写入飞书 Base、Task 或 GitLink。", job.PRNumber)
	default:
		return reviewGatewayExecutionFailure(result, fmt.Errorf("unsupported review gateway action %q", job.Action))
	}
	return result, nil
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
	result.ReviewCount = reviewContext.Summary.TotalReviews
	result.ThreadCount = reviewContext.Summary.TotalThreads
	result.OpenThreadCount = reviewContext.Summary.OpenThreads
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
