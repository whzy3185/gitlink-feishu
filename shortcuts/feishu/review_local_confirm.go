package feishu

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/gitlink-org/gitlink-cli/shortcuts/common"
)

func newReviewConfirmLocalShortcut() *common.Shortcut {
	return &common.Shortcut{
		Name:        "review-confirm-local",
		Description: "Confirm one prepared Feishu PR action with the current local GitLink credential",
		Flags: []common.Flag{
			{Name: "plan-id", Required: true},
			{Name: "state-db", Default: ".local/review-gateway.db"},
			{Name: "dry-run", Bool: true, Default: "false"},
			{Name: "yes", Bool: true, Default: "false"},
		},
		Run: runReviewConfirmLocal,
	}
}

func runReviewConfirmLocal(runtime *common.RuntimeContext) error {
	store, err := OpenSQLiteReviewGatewayStore(runtime.Arg("state-db"))
	if err != nil {
		return err
	}
	defer store.Close()
	plan, err := store.GetReviewActionPlan(context.Background(), runtime.Arg("plan-id"))
	if err != nil {
		return err
	}
	dryRun := parseBool(runtime.Arg("dry-run"))
	confirmed := parseBool(runtime.Arg("yes"))
	if !dryRun && !confirmed {
		fmt.Fprintf(os.Stderr, "操作：%s\n仓库：%s\nPR：#%d\nGitLink 身份：%s\n基于版本：%s\n操作编号：%s\n输入 yes 继续：", reviewActionDisplayName(plan.Action), plan.Repository, plan.PRNumber, plan.GitLinkLogin, plan.ExpectedHeadSHA, plan.RequestID)
		var answer string
		_, _ = fmt.Fscanln(os.Stdin, &answer)
		confirmed = strings.EqualFold(strings.TrimSpace(answer), "yes")
	}
	if !dryRun && !confirmed {
		return fmt.Errorf("local confirmation was not approved")
	}
	now := time.Now().UTC()
	updated, executeErr := executeReviewActionPlan(context.Background(), runtime, store, GitLinkReviewDataProvider{}, plan, dryRun, now)
	result := ReviewGatewayExecutionResult{
		SchemaVersion: reviewGatewayResultSchema, JobID: plan.SourceJobID, Status: "completed",
		Mode: "local_confirmation", Action: plan.Action, Repository: plan.Repository, PRNumber: plan.PRNumber,
		RequestedBy: plan.ActorID, ReadOnlyGitLink: dryRun, MutatesGitLink: !dryRun && updated.MutationStatus == "confirmed",
		CompletedAt: now.Format(time.RFC3339Nano), ActionPlan: &updated,
	}
	if dryRun {
		result.Status = "dry_run"
		result.Message = "试运行完成；GitLink 写入为 0。"
	} else if executeErr != nil {
		result.Status = updated.Status
		result.Error = redactReviewGatewayError(executeErr.Error())
		result.Message = "操作结果暂无法确认；系统已停止自动重试。"
	} else {
		result.Message = "GitLink 操作已完成并通过回读验证。"
	}
	if !dryRun {
		_ = store.PublishReviewActionResult(context.Background(), plan.SourceJobID, result, now)
	}
	if outputErr := runtime.OutputData(result); outputErr != nil {
		return outputErr
	}
	return executeErr
}

func (s *SQLiteReviewGatewayStore) PublishReviewActionResult(ctx context.Context, jobID string, result ReviewGatewayExecutionResult, now time.Time) error {
	encoded, err := jsonMarshalReviewResult(result)
	if err != nil {
		return err
	}
	update, err := s.db.ExecContext(ctx, `UPDATE review_gateway_jobs SET result_json=?, reply_status='pending',
		reply_attempt_count=0, reply_next_attempt_at=?, updated_at=? WHERE job_id=?`,
		encoded, reviewGatewayTimestamp(now), reviewGatewayTimestamp(now), jobID)
	return requireReviewGatewayJobUpdate(update, err, jobID, "publish review action result")
}

func jsonMarshalReviewResult(result ReviewGatewayExecutionResult) (string, error) {
	encoded, err := json.Marshal(result)
	if err != nil {
		return "", fmt.Errorf("encode review action result: %w", err)
	}
	return string(encoded), nil
}
