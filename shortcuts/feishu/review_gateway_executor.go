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
	SchemaVersion     string              `json:"schema_version"`
	JobID             string              `json:"job_id"`
	Status            string              `json:"status"`
	Mode              string              `json:"mode"`
	Action            string              `json:"action"`
	Repository        string              `json:"repository,omitempty"`
	PRNumber          int                 `json:"pr_number,omitempty"`
	RequestedBy       string              `json:"requested_by"`
	ReadOnlyGitLink   bool                `json:"read_only_gitlink"`
	MutatesGitLink    bool                `json:"mutates_gitlink"`
	CompletedAt       string              `json:"completed_at"`
	Message           string              `json:"message,omitempty"`
	CollectionStatus  string              `json:"collection_status,omitempty"`
	Partial           bool                `json:"partial,omitempty"`
	HeadSHA           string              `json:"head_sha,omitempty"`
	SourceFingerprint string              `json:"source_fingerprint,omitempty"`
	ReviewStage       string              `json:"review_stage,omitempty"`
	Decision          string              `json:"decision,omitempty"`
	ReviewCount       int                 `json:"review_count,omitempty"`
	ThreadCount       int                 `json:"thread_count,omitempty"`
	OpenThreadCount   int                 `json:"open_thread_count,omitempty"`
	ReviewData        *ReviewData         `json:"review_data,omitempty"`
	SnapshotPlan      *ReviewSnapshotPlan `json:"snapshot_plan,omitempty"`
	Error             string              `json:"error,omitempty"`
	AttemptCount      int                 `json:"attempt_count,omitempty"`
}

type ReviewGatewayExecutor struct {
	Runtime      *common.RuntimeContext
	DataProvider ReviewDataProvider
	Now          func() time.Time
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
		plan := PlanReviewSnapshotSync(data, nil)
		result.SnapshotPlan = &plan
		result.Message = "已完成 GitLink PR 只读查询。"
		return result, nil

	case "help":
		result.Message = "支持：查看 PR #编号、刷新 PR #编号、查看绑定。GitLink 查询保持只读。"
	case "unsupported_command":
		return reviewGatewayExecutionFailure(result, fmt.Errorf("不支持的命令，请发送“帮助”查看可用命令"))
	case "show_binding":
		result.Message = fmt.Sprintf("当前群已绑定仓库 %s。", job.Repository)
	case "plan_bind_repository":
		result.Message = fmt.Sprintf("已生成仓库绑定预览：%s；未修改远程资源。", job.Repository)
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
