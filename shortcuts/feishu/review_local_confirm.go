package feishu

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/gitlink-org/gitlink-cli/shortcuts/common"
	prshortcut "github.com/gitlink-org/gitlink-cli/shortcuts/pr"
	"github.com/gitlink-org/gitlink-cli/shortcuts/workflow"
	"os"
	"strings"
	"time"
)

func newReviewConfirmLocalShortcut() *common.Shortcut {
	return &common.Shortcut{Name: "review-confirm-local", Description: "Confirm one prepared common Review with local GitLink credentials", Flags: []common.Flag{{Name: "plan-id", Required: true}, {Name: "state-db", Default: ".local/review-gateway.db"}, {Name: "dry-run", Bool: true, Default: "false"}, {Name: "yes", Bool: true, Default: "false"}}, Run: runReviewConfirmLocal}
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
	dryRun, approved := parseBool(runtime.Arg("dry-run")), parseBool(runtime.Arg("yes"))
	if !dryRun && !approved {
		fmt.Fprintf(os.Stderr, "%s PR #%d\nHead: %s\nGitLink actor: %s\nRef: %s\nReview:\n%s\nType yes to continue: ", plan.Repository, plan.PRNumber, plan.ExpectedHeadSHA, plan.GitLinkLogin, plan.RequestID, plan.Content)
		var answer string
		_, _ = fmt.Fscanln(os.Stdin, &answer)
		approved = strings.EqualFold(strings.TrimSpace(answer), "yes")
	}
	result, err := executeLocalReviewConfirmation(context.Background(), runtime, store, plan, dryRun, approved, time.Now().UTC())
	if err != nil {
		return err
	}
	return runtime.OutputData(result)
}
func executeLocalReviewConfirmation(ctx context.Context, runtime *common.RuntimeContext, store *SQLiteReviewGatewayStore, plan ReviewActionPlan, dryRun, approved bool, now time.Time) (ReviewGatewayExecutionResult, error) {
	result := ReviewGatewayExecutionResult{SchemaVersion: reviewGatewayResultSchema, JobID: plan.SourceJobID, Status: "completed", Mode: "local_confirmation", Action: "confirm_common_review", Repository: plan.Repository, PRNumber: plan.PRNumber, RequestedBy: plan.ActorID, ReadOnlyGitLink: true, CompletedAt: now.Format(time.RFC3339)}
	if plan.Status != "pending_confirmation" || plan.ReviewStatus != "common" {
		return reviewGatewayExecutionFailure(result, fmt.Errorf("ActionPlan is not pending common Review confirmation"))
	}
	expires, err := time.Parse(time.RFC3339Nano, plan.ExpiresAt)
	if err != nil || !expires.After(now) {
		return reviewGatewayExecutionFailure(result, fmt.Errorf("ActionPlan expired"))
	}
	owner, repo, err := splitReviewGatewayRepository(plan.Repository)
	if err != nil {
		return reviewGatewayExecutionFailure(result, err)
	}
	current, err := workflow.FetchReviewContext(runtime, workflow.ReviewContextOptions{Owner: owner, Repo: repo, Number: plan.PRNumber, VersionLimit: 100, ThreadLimit: 100, IncludePR: true, IncludeFiles: true, IncludeVersions: true, IncludeReviews: true, IncludeThreads: true})
	if err != nil || current.Partial || current.CollectionStatus != "complete" || current.CurrentHeadSHA != plan.ExpectedHeadSHA || current.WorkItem.SourceFingerprint != plan.SourceFingerprint {
		return reviewGatewayExecutionFailure(result, fmt.Errorf("ActionPlan is stale or the current Review context is incomplete"))
	}
	populateReviewGatewayContextResult(&result, current)
	if dryRun || !approved {
		result.WriteResult = &ReviewWriteResult{PlanID: plan.PlanID, Status: map[bool]string{true: "dry_run", false: "cancelled"}[dryRun], Outcome: "not_started", Repository: plan.Repository, PRNumber: plan.PRNumber, HeadSHA: plan.ExpectedHeadSHA, ReviewStatus: "common", MutationStatus: reviewMutationNone, RequestID: plan.RequestID}
		result.Message = "Local confirmation did not write to GitLink"
		return result, nil
	}
	leaseOwner := "local-review:" + plan.RequestID
	plan, err = store.ClaimReviewActionPlan(ctx, ReviewActionPlanClaimOptions{PlanID: plan.PlanID, ActorID: plan.ActorID, LeaseOwner: leaseOwner, Now: now, LeaseDuration: 2 * time.Minute})
	if err != nil {
		return reviewGatewayExecutionFailure(result, err)
	}
	write := prshortcut.ExecuteCommonReview(runtime, prshortcut.CommonReviewOptions{Owner: owner, Repository: repo, PRNumber: plan.PRNumber, Content: plan.Content, ExpectedHead: plan.ExpectedHeadSHA, ExpectedActor: plan.GitLinkLogin, RequestID: plan.RequestID, BeforePOST: func() error { return store.MarkReviewActionPlanWriteStarted(ctx, plan.PlanID, leaseOwner, now) }})
	result, _ = (&ReviewGatewayExecutor{ActionPlans: store}).finishCommonReviewWrite(ctx, result, plan, write, now)
	result.ResultCard = buildReviewGatewayResultCard(ReviewGatewayJob{Repository: plan.Repository, PRNumber: plan.PRNumber}, result, nil)
	var source ReviewGatewayJob
	var payload string
	if store.db.QueryRowContext(ctx, "SELECT payload_json FROM review_gateway_jobs WHERE job_id=?", plan.SourceJobID).Scan(&payload) == nil && json.Unmarshal([]byte(payload), &source) == nil {
		planner := &ReviewOperationPlanner{Store: store, Now: func() time.Time { return now }}
		_, _ = planner.Plan(ctx, source, result)
	}
	return result, nil
}
