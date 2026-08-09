package feishu

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/gitlink-org/gitlink-cli/shortcuts/common"
)

const reviewGatewayResultSchema = "feishu.review-result/v1"

type ReviewGatewayExecutionResult struct {
	SchemaVersion     string                   `json:"schema_version"`
	JobID             string                   `json:"job_id"`
	Status            string                   `json:"status"`
	Mode              string                   `json:"mode"`
	Action            string                   `json:"action"`
	Repository        string                   `json:"repository,omitempty"`
	PRNumber          int                      `json:"pr_number,omitempty"`
	RequestedBy       string                   `json:"requested_by"`
	ReadOnlyGitLink   bool                     `json:"read_only_gitlink"`
	MutatesGitLink    bool                     `json:"mutates_gitlink"`
	CompletedAt       string                   `json:"completed_at"`
	Message           string                   `json:"message,omitempty"`
	CollectionStatus  string                   `json:"collection_status,omitempty"`
	Partial           bool                     `json:"partial,omitempty"`
	HeadSHA           string                   `json:"head_sha,omitempty"`
	SourceFingerprint string                   `json:"source_fingerprint,omitempty"`
	ReviewStage       string                   `json:"review_stage,omitempty"`
	Decision          string                   `json:"decision,omitempty"`
	ReviewCount       int                      `json:"review_count,omitempty"`
	ThreadCount       int                      `json:"thread_count,omitempty"`
	OpenThreadCount   int                      `json:"open_thread_count,omitempty"`
	ReviewData        *ReviewData              `json:"review_data,omitempty"`
	SnapshotPlan      *ReviewSnapshotPlan      `json:"snapshot_plan,omitempty"`
	Collaboration     *ReviewCollaborationItem `json:"collaboration,omitempty"`
	ActionPlan        *ReviewActionPlan        `json:"action_plan,omitempty"`
	Error             string                   `json:"error,omitempty"`
	AttemptCount      int                      `json:"attempt_count,omitempty"`
}

type ReviewGatewayExecutor struct {
	Runtime       *common.RuntimeContext
	DataProvider  ReviewDataProvider
	Collaboration ReviewCollaborationStore
	ActionPlans   *SQLiteReviewGatewayStore
	Collaborators ReviewRepositoryCollaboratorReader
	DisplayNames  FeishuDisplayNameResolver
	Now           func() time.Time
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
		AttemptCount:    job.AttemptCount,
	}
	if err := ctx.Err(); err != nil {
		result.Status = "failed"
		result.Error = redactReviewGatewayError(err.Error())
		return result, err
	}

	switch job.Action {
	case "read_review_context", "refresh_review_context":
		if e.Runtime == nil {
			return reviewGatewayExecutionFailure(result, fmt.Errorf("GitLink runtime is required"))
		}
		owner, repo, err := splitReviewGatewayRepository(job.Repository)
		if err != nil {
			return reviewGatewayExecutionFailure(result, err)
		}
		provider := e.DataProvider
		if provider == nil {
			provider = GitLinkReviewDataProvider{}
		}
		data, err := provider.FetchReviewData(ctx, e.Runtime, ReviewDataRequest{
			Owner: owner, Repository: repo, PullRequest: job.PRNumber,
			VersionLimit: 100, ReviewLimit: 100, ThreadLimit: 100,
		})
		if err != nil {
			return reviewGatewayExecutionFailure(result, err)
		}
		if job.RequirePublicRepo && (data.RepositoryPublic == nil || !*data.RepositoryPublic) {
			return reviewGatewayExecutionFailure(result, fmt.Errorf("explicit repository queries require a confirmed public repository"))
		}
		populateReviewGatewayResult(&result, data)
		if e.Collaboration != nil {
			item, itemErr := e.Collaboration.GetCollaborationItem(ctx, job.ChatID, job.Repository, job.PRNumber)
			if itemErr != nil {
				return reviewGatewayExecutionFailure(result, itemErr)
			}
			if item.SchemaVersion != "" {
				result.Collaboration = &item
			}
		}
		plan := PlanReviewSnapshotSync(data, nil)
		result.SnapshotPlan = &plan
		result.Message = "已完成 GitLink PR 只读查询。"
		return result, nil

	case "help":
		result.Message = strings.Join([]string{
			"GitLink PR Review 助手",
			"",
			"查询与协作",
			"查看 <拥有者>/<仓库> PR #<编号>",
			"领取 <拥有者>/<仓库> PR #<编号>",
			"取消领取 <拥有者>/<仓库> PR #<编号>",
			"设置 <拥有者>/<仓库> PR #<编号> 审查截止 <YYYY-MM-DD>",
			"清除 <拥有者>/<仓库> PR #<编号> 审查截止",
			"",
			"受控写操作",
			"提交审查意见 <拥有者>/<仓库> PR #<编号> <意见>",
			"批准 <拥有者>/<仓库> PR #<编号> <说明>",
			"需要修改 <拥有者>/<仓库> PR #<编号> <原因>",
			"拒绝并关闭 <拥有者>/<仓库> PR #<编号> <原因>",
			"合并 <拥有者>/<仓库> PR #<编号>",
			"",
			"受控写操作只生成计划，最终执行需要绑定的 GitLink 身份在本地确认。",
		}, "\n")
	case "unsupported_command":
		return reviewGatewayExecutionFailure(result, fmt.Errorf("不支持的命令，请发送“帮助”查看可用命令"))
	case "show_binding":
		result.Message = fmt.Sprintf("当前群已绑定仓库 %s。", job.Repository)
	case "plan_bind_repository":
		result.Message = fmt.Sprintf("已生成仓库绑定预览：%s；未修改远程资源。", job.Repository)
	case "claim_review", "release_review", "set_review_deadline", "clear_review_deadline":
		if !job.CollaborationAuthorized || strings.TrimSpace(job.GitLinkLogin) == "" {
			return reviewGatewayExecutionFailure(result, fmt.Errorf("当前飞书用户尚未绑定 GitLink 身份"))
		}
		if e.Collaboration == nil {
			return reviewGatewayExecutionFailure(result, fmt.Errorf("review collaboration store is required"))
		}
		owner, repo, err := splitReviewGatewayRepository(job.Repository)
		if err != nil {
			return reviewGatewayExecutionFailure(result, err)
		}
		provider := e.DataProvider
		if provider == nil {
			provider = GitLinkReviewDataProvider{}
		}
		if _, err := provider.FetchReviewData(ctx, e.Runtime, ReviewDataRequest{Owner: owner, Repository: repo, PullRequest: job.PRNumber}); err != nil {
			return reviewGatewayExecutionFailure(result, err)
		}
		if job.Action == "claim_review" {
			reader := e.Collaborators
			if reader == nil {
				reader = GitLinkReviewRepositoryCollaboratorReader{}
			}
			collaborators, err := reader.ListRepositoryCollaborators(ctx, e.Runtime, owner, repo)
			if err != nil {
				return reviewGatewayExecutionFailure(result, err)
			}
			if !reviewLoginIsCollaborator(job.GitLinkLogin, collaborators) {
				return reviewGatewayExecutionFailure(result, fmt.Errorf("绑定的 GitLink 用户不是当前仓库协作者"))
			}
		}
		if e.DisplayNames != nil {
			if name, err := e.DisplayNames.ResolveFeishuDisplayName(ctx, job.RequestedBy); err == nil {
				job.RequestedDisplayName = name
			}
		}
		if job.RequestedDisplayName == "" {
			job.RequestedDisplayName = job.GitLinkLogin
		}
		item, err := e.Collaboration.ApplyCollaborationAction(ctx, job, now().UTC())
		if err != nil {
			return reviewGatewayExecutionFailure(result, err)
		}
		result.Collaboration = &item
		result.Message = formatReviewCollaborationOutcome(item)
	case "prepare_common_review", "prepare_review_approve", "prepare_review_reject", "prepare_reject_close", "prepare_merge":
		return e.prepareControlledReviewAction(ctx, job, result, now().UTC())
	default:
		return reviewGatewayExecutionFailure(result, fmt.Errorf("unsupported review gateway action %q", job.Action))
	}
	return result, nil
}

func formatReviewCollaborationOutcome(item ReviewCollaborationItem) string {
	switch item.ActionOutcome {
	case "claimed":
		return fmt.Sprintf("PR #%d 已由 %s 负责", item.PRNumber, firstNonEmpty(item.AssignedDisplayName, "飞书成员"))
	case "already_claimed_by_self":
		return fmt.Sprintf("PR #%d 当前已由你负责", item.PRNumber)
	case "released":
		return fmt.Sprintf("PR #%d 已取消负责人", item.PRNumber)
	case "already_unassigned":
		return fmt.Sprintf("PR #%d 当前没有负责人", item.PRNumber)
	case "deadline_set":
		return fmt.Sprintf("PR #%d 审查截止时间已设置为 %s", item.PRNumber, item.DueAt)
	case "deadline_updated":
		return fmt.Sprintf("PR #%d 审查截止时间已更新为 %s", item.PRNumber, item.DueAt)
	case "deadline_unchanged":
		return fmt.Sprintf("PR #%d 审查截止时间仍为 %s", item.PRNumber, item.DueAt)
	case "deadline_cleared":
		return fmt.Sprintf("PR #%d 的审查截止时间已清除", item.PRNumber)
	case "deadline_absent":
		return fmt.Sprintf("PR #%d 当前没有设置审查截止时间", item.PRNumber)
	default:
		return fmt.Sprintf("PR #%d 协作状态已更新", item.PRNumber)
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

func populateReviewGatewayResult(result *ReviewGatewayExecutionResult, data ReviewData) {
	result.CollectionStatus = data.CollectionStatus
	result.Partial = data.Partial
	result.HeadSHA = data.HeadSHA
	result.SourceFingerprint = data.SourceFingerprint
	result.ReviewStage = reviewGatewayReviewStage(data)
	result.Decision = data.Summary.Decision
	result.ReviewCount = data.Summary.TotalReviews
	result.ThreadCount = data.Summary.TotalThreads
	result.OpenThreadCount = data.Summary.OpenThreads
	copy := data
	result.ReviewData = &copy
}

func reviewGatewayReviewStage(data ReviewData) string {
	switch data.State {
	case "merged", "closed":
		return data.State
	}
	switch data.Summary.ReviewStatus {
	case "rejected":
		return "waiting_for_contributor"
	case "approved":
		return "ready_for_decision"
	case "common":
		return "human_reviewing"
	default:
		return "triaged"
	}
}
