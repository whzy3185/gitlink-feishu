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
	return &common.Shortcut{Name: "review-confirm-local", Description: "Confirm one prepared controlled PR action with local GitLink credentials", Flags: []common.Flag{{Name: "plan-id", Required: true}, {Name: "state-db", Default: ".local/review-gateway.db"}, {Name: "dry-run", Bool: true, Default: "false"}, {Name: "yes", Bool: true, Default: "false"}}, Run: runReviewConfirmLocal}
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
		owner, repo, splitErr := splitReviewGatewayRepository(plan.Repository)
		if splitErr != nil {
			return splitErr
		}
		current, fetchErr := workflow.FetchReviewContext(runtime, workflow.ReviewContextOptions{Owner: owner, Repo: repo, Number: plan.PRNumber, IncludePR: true, IncludeVersions: true})
		if fetchErr != nil {
			return fetchErr
		}
		warning := ""
		switch plan.Action {
		case reviewActionMerge:
			warning = "\nTHIS ACTION WILL MERGE THE PR AND CHANGE THE TARGET BRANCH.\n"
		case reviewActionRejectClose:
			warning = "\nTHIS ACTION WILL REJECT AND CLOSE THE PR.\n"
		}
		fmt.Fprintf(os.Stderr, "ACTION: %s\nRepository: %s\nPR: #%d\nTitle: %s\nBase: %s\nHead branch: %s\nExpected SHA: %s\nCurrent SHA: %s\nGitLink actor: %s\nRequest ID: %s\nContent / Reason:\n%s\n%sType yes to continue: ", reviewActionLabel(plan.Action), plan.Repository, plan.PRNumber, current.WorkItem.Title, current.WorkItem.BaseBranch, current.WorkItem.HeadBranch, plan.ExpectedHeadSHA, current.CurrentHeadSHA, plan.GitLinkLogin, plan.RequestID, plan.Content, warning)
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
	action, actionErr := normalizeReviewAction(plan.Action, plan.ReviewStatus)
	if plan.Status != "pending_confirmation" || actionErr != nil {
		return reviewGatewayExecutionFailure(result, fmt.Errorf("ActionPlan is not pending controlled action confirmation"))
	}
	plan.Action, plan.ReviewStatus = action, reviewStatusForAction(action)
	expires, err := time.Parse(time.RFC3339Nano, plan.ExpiresAt)
	if err != nil || !expires.After(now) {
		return reviewGatewayExecutionFailure(result, fmt.Errorf("ActionPlan expired"))
	}
	owner, repo, err := splitReviewGatewayRepository(plan.Repository)
	if err != nil {
		return reviewGatewayExecutionFailure(result, err)
	}
	current, err := workflow.FetchReviewContext(runtime, workflow.ReviewContextOptions{Owner: owner, Repo: repo, Number: plan.PRNumber, VersionLimit: 100, ThreadLimit: 100, IncludePR: true, IncludeFiles: true, IncludeVersions: true, IncludeReviews: true, IncludeThreads: true})
	if err != nil {
		return reviewGatewayExecutionFailure(result, err)
	}
	populateReviewGatewayContextResult(&result, current)
	if current.Partial || current.CollectionStatus != "complete" {
		return reviewGatewayExecutionFailure(result, fmt.Errorf("ActionPlan is stale or the current Review context is incomplete"))
	}
	if strings.ToLower(strings.TrimSpace(current.WorkItem.GitLinkState)) != "open" {
		return reviewGatewayExecutionFailure(result, fmt.Errorf("pull request must remain open"))
	}
	if current.CurrentHeadSHA != plan.ExpectedHeadSHA || current.WorkItem.SourceFingerprint != plan.SourceFingerprint {
		staleLeaseOwner := "local-review-stale:" + plan.RequestID
		plan, err = store.ClaimReviewActionPlan(ctx, ReviewActionPlanClaimOptions{PlanID: plan.PlanID, ActorID: plan.ActorID, LeaseOwner: staleLeaseOwner, Now: now, LeaseDuration: 2 * time.Minute})
		if err != nil {
			return reviewGatewayExecutionFailure(result, err)
		}
		if err := store.FinishReviewActionPlan(ctx, plan.PlanID, "stale", "", "PR context changed", reviewMutationNone, now); err != nil {
			return reviewGatewayExecutionFailure(result, err)
		}
		if latest, err := store.GetReviewActionPlan(ctx, plan.PlanID); err == nil {
			plan = latest
		} else {
			plan.Status = "stale"
		}
		result.ActionPlan = &plan
		result.WriteResult = &ReviewWriteResult{PlanID: plan.PlanID, Status: "stale", Outcome: "stale", Action: plan.Action, Repository: plan.Repository, PRNumber: plan.PRNumber, HeadSHA: current.CurrentHeadSHA, ReviewStatus: plan.ReviewStatus, MutationStatus: reviewMutationNone, RequestID: plan.RequestID}
		result.Message = "PR context changed; ActionPlan is stale and GitLink writes remain zero"
		return planLocalReviewResultOperations(ctx, store, plan, result, now), nil
	}
	actor, err := resolveLocalReviewActor(runtime)
	if err != nil {
		return reviewGatewayExecutionFailure(result, err)
	}
	if actor != plan.GitLinkLogin {
		return reviewGatewayExecutionFailure(result, fmt.Errorf("active GitLink identity does not match the expected actor"))
	}
	if dryRun || !approved {
		result.WriteResult = &ReviewWriteResult{PlanID: plan.PlanID, Status: map[bool]string{true: "dry_run", false: "cancelled"}[dryRun], Outcome: "not_started", Action: plan.Action, Repository: plan.Repository, PRNumber: plan.PRNumber, HeadSHA: plan.ExpectedHeadSHA, ReviewStatus: plan.ReviewStatus, MutationStatus: reviewMutationNone, RequestID: plan.RequestID}
		return result, nil
	}
	leaseOwner := "local-review:" + plan.RequestID
	plan, err = store.ClaimReviewActionPlan(ctx, ReviewActionPlanClaimOptions{PlanID: plan.PlanID, ActorID: plan.ActorID, LeaseOwner: leaseOwner, Now: now, LeaseDuration: 2 * time.Minute})
	if err != nil {
		return reviewGatewayExecutionFailure(result, err)
	}
	executor := &ReviewGatewayExecutor{ActionPlans: store}
	beforePOST := func() error { return store.MarkReviewActionPlanWriteStarted(ctx, plan.PlanID, leaseOwner, now) }
	if plan.ReviewStatus != "" {
		write := prshortcut.ExecuteControlledReview(runtime, prshortcut.CommonReviewOptions{Owner: owner, Repository: repo, PRNumber: plan.PRNumber, Content: plan.Content, ReviewStatus: plan.ReviewStatus, ExpectedHead: plan.ExpectedHeadSHA, ExpectedActor: plan.GitLinkLogin, RequestID: plan.RequestID, BeforePOST: beforePOST})
		result, _ = executor.finishCommonReviewWrite(ctx, result, plan, write, now)
	} else {
		write := prshortcut.ExecuteControlledPRAction(runtime, prshortcut.ControlledPRActionOptions{Owner: owner, Repository: repo, PRNumber: plan.PRNumber, Action: plan.Action, ExpectedHead: plan.ExpectedHeadSHA, ExpectedActor: plan.GitLinkLogin, BeforePOST: beforePOST})
		result, _ = executor.finishControlledPRAction(ctx, result, plan, write, now)
	}
	return planLocalReviewResultOperations(ctx, store, plan, result, now), nil
}

func resolveLocalReviewActor(runtime *common.RuntimeContext) (string, error) {
	user, err := runtime.CallAPI("GET", "/users/me", nil)
	if err != nil {
		return "", fmt.Errorf("resolve active GitLink identity: %w", err)
	}
	userMap, _ := user.Data.(map[string]interface{})
	for _, key := range []string{"login", "username"} {
		if value := strings.TrimSpace(fmt.Sprint(userMap[key])); value != "" && value != "<nil>" {
			return value, nil
		}
	}
	return "", fmt.Errorf("active GitLink identity is missing login")
}

func planLocalReviewResultOperations(ctx context.Context, store *SQLiteReviewGatewayStore, plan ReviewActionPlan, result ReviewGatewayExecutionResult, now time.Time) ReviewGatewayExecutionResult {
	result.ResultCard = buildReviewGatewayResultCard(ReviewGatewayJob{Repository: plan.Repository, PRNumber: plan.PRNumber}, result, nil)
	var source ReviewGatewayJob
	var payload string
	if store.db.QueryRowContext(ctx, "SELECT payload_json FROM review_gateway_jobs WHERE job_id=?", plan.SourceJobID).Scan(&payload) == nil && json.Unmarshal([]byte(payload), &source) == nil {
		planner := &ReviewOperationPlanner{Store: store, Now: func() time.Time { return now }}
		_, _ = planner.Plan(ctx, source, result)
	}
	return result
}
