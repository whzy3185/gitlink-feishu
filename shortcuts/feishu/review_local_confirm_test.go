package feishu

import (
	"context"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gitlink-org/gitlink-cli/internal/client"
	"github.com/gitlink-org/gitlink-cli/shortcuts/common"
	"github.com/gitlink-org/gitlink-cli/shortcuts/workflow"
)

func TestCommonReviewInputBuildsBoundedActionPlanCard(t *testing.T) {
	intent := parseReviewGatewayIntent("review owner/repo#42 Useful evidence GITLINK_TOKEN=secret123 Ref: RW-AAAAAA")
	if intent.Name != "prepare_common_review" || intent.Repository != "owner/repo" || intent.PRNumber != 42 {
		t.Fatalf("intent = %#v", intent)
	}
	if strings.Contains(intent.Argument, "secret123") {
		t.Fatalf("secret reached the durable Review body: %q", intent.Argument)
	}
	job := testReviewGatewayJob(time.Now().UTC(), "new-review-input")
	job.Repository, job.PRNumber, job.Argument = intent.Repository, intent.PRNumber, intent.Argument
	plan := NewReviewActionPlan(job, "reviewer", "head", "fingerprint", job.Argument, time.Now().UTC())
	if !strings.HasPrefix(plan.RequestID, "RW-") || strings.Count(plan.Content, "Ref:") != 1 || strings.Contains(plan.Content, "RW-AAAAAA") {
		t.Fatalf("plan request identity/content = %#v", plan)
	}
	result := ReviewGatewayExecutionResult{Status: "completed", Repository: plan.Repository, PRNumber: plan.PRNumber, ActionPlan: &plan, ConfirmationStateDB: ".local/review-gateway-live.db"}
	cardJSON, ok := safeReviewGatewayCardJSON(buildReviewGatewayResultCard(job, result, nil))
	if !ok || strings.Contains(cardJSON, ".local/review-gateway-live.db") || strings.Contains(cardJSON, "review-confirm-local") {
		t.Fatalf("ordinary ActionPlan card leaked advanced local confirmation details: %s", cardJSON)
	}
	for _, expected := range []string{plan.RequestID, plan.GitLinkLogin, "查看本地确认方式", "取消操作", "刷新 PR 状态", plan.PlanID} {
		if !ok || !strings.Contains(cardJSON, expected) {
			t.Fatalf("ActionPlan card missing %q: %s", expected, cardJSON)
		}
	}
	advanced := result
	advanced.Action = "show_local_review_plan"
	advancedJSON, ok := safeReviewGatewayCardJSON(buildReviewGatewayResultCard(job, advanced, nil))
	if !ok || !strings.Contains(advancedJSON, "review-confirm-local") || !strings.Contains(advancedJSON, ".local/review-gateway-live.db") || strings.Contains(advancedJSON, "--state-db .local/review-gateway.db") {
		t.Fatalf("advanced confirmation card did not preserve the configured state DB: %s", advancedJSON)
	}
	result.PullRequest = &ReviewGatewayPullRequestView{RecommendedNextStep: "assign_human_reviewer"}
	result.ResultCard = buildReviewGatewayResultCard(job, result, nil)
	reply := formatReviewGatewayResultReply(job, result)
	if strings.Contains(reply, plan.PlanID) || !strings.Contains(reply, plan.RequestID) || strings.Contains(reply, "assign_human_reviewer") {
		t.Fatalf("ActionPlan reply was downgraded to a PR summary: %s", reply)
	}
	ack := formatReviewGatewayAcknowledgement(ReviewGatewayJob{Action: "prepare_common_review", Repository: job.Repository, PRNumber: job.PRNumber, JobID: job.JobID})
	if !strings.Contains(ack, "提交审查意见") || strings.Contains(ack, "ActionPlan") || strings.Contains(ack, "只读 Review 请求") {
		t.Fatalf("ActionPlan acknowledgement = %q", ack)
	}
	planner := &ReviewOperationPlanner{}
	operations, err := planner.BuildWithConsumers(job, result, []ReviewJobConsumer{{
		ConsumerID: "consumer", JobID: job.JobID, SourceType: "user_command", ChatID: job.ChatID, SourceMessageID: job.SourceMessageID,
	}}, time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}
	var cardPlanned bool
	for _, operation := range operations {
		if operation.OperationKind == ReviewOperationCanonicalCardUpsert && strings.Contains(operation.DesiredJSON, plan.PlanID) && strings.Contains(operation.DesiredJSON, plan.RequestID) {
			cardPlanned = true
		}
	}
	if !cardPlanned {
		t.Fatalf("ActionPlan confirmation card operation was not planned: %#v", operations)
	}
	for input, want := range map[string]string{
		"approve owner/repo#42 evidence": "prepare_review_approve",
		"reject owner/repo#42 reason":    "prepare_review_reject",
		"refuse owner/repo#42 reason":    "prepare_reject_close",
		"merge owner/repo#42":            "prepare_merge",
	} {
		if got := parseReviewGatewayIntent(input); got.Name != want {
			t.Fatalf("controlled command %q mapped to %#v, want %s", input, got, want)
		}
	}
	for _, incomplete := range []string{"approve owner/repo#42", "reject owner/repo#42", "refuse owner/repo#42"} {
		if got := parseReviewGatewayIntent(incomplete); got.Name != "unknown" {
			t.Fatalf("missing required reason %q mapped to %#v", incomplete, got)
		}
	}
}

func TestLocalReviewConfirmationDryRunStaleAndConcurrentSingleWrite(t *testing.T) {
	state := &commonReviewTestServerState{}
	server := newCommonReviewTestServer(t, state)
	defer server.Close()
	runtime := &common.RuntimeContext{Client: &client.Client{HTTP: server.Client(), BaseURL: server.URL}, Owner: "owner", Repo: "repo"}
	current, err := workflow.FetchReviewContext(runtime, workflow.ReviewContextOptions{Owner: "owner", Repo: "repo", Number: 42,
		VersionLimit: 100, ThreadLimit: 100, IncludePR: true, IncludeFiles: true, IncludeVersions: true, IncludeReviews: true, IncludeThreads: true})
	if err != nil {
		t.Fatal(err)
	}
	store, err := OpenSQLiteReviewGatewayStore(filepath.Join(t.TempDir(), "local-confirm.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	now := time.Now().UTC()
	job := testReviewGatewayJob(now, "local-confirm")
	job.SourceMessageID, job.NotifyChat = "om_source", true
	if err := store.SaveJob(context.Background(), job); err != nil {
		t.Fatal(err)
	}
	plan, err := store.CreateReviewActionPlan(context.Background(), NewReviewActionPlan(job, "gitlink-reviewer", current.CurrentHeadSHA, current.WorkItem.SourceFingerprint, "Evidence", now))
	if err != nil {
		t.Fatal(err)
	}
	if result, err := executeLocalReviewConfirmation(context.Background(), runtime, store, plan, true, false, now.Add(time.Second)); err != nil || result.WriteResult.Status != "dry_run" || state.writes() != 0 {
		t.Fatalf("dry run result=%#v err=%v writes=%d", result, err, state.writes())
	}
	staleJob := testReviewGatewayJob(now, "local-confirm-stale")
	if err := store.SaveJob(context.Background(), staleJob); err != nil {
		t.Fatal(err)
	}
	stalePlan, err := store.CreateReviewActionPlan(context.Background(), NewReviewActionPlan(staleJob, "gitlink-reviewer", current.CurrentHeadSHA, "changed", "Evidence", now))
	if err != nil {
		t.Fatal(err)
	}
	staleResult, err := executeLocalReviewConfirmation(context.Background(), runtime, store, stalePlan, false, true, now.Add(time.Second))
	if err != nil || staleResult.WriteResult == nil || staleResult.WriteResult.Status != "stale" || state.writes() != 0 {
		t.Fatalf("stale result=%#v err=%v writes=%d", staleResult, err, state.writes())
	}
	storedStale, _ := store.GetReviewActionPlan(context.Background(), stalePlan.PlanID)
	if storedStale.Status != "stale" || storedStale.MutationStatus != reviewMutationNone {
		t.Fatalf("stored stale plan = %#v", storedStale)
	}
	var wg sync.WaitGroup
	results := make(chan ReviewGatewayExecutionResult, 2)
	errors := make(chan error, 2)
	for range 2 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			result, runErr := executeLocalReviewConfirmation(context.Background(), runtime, store, plan, false, true, now.Add(2*time.Second))
			results <- result
			errors <- runErr
		}()
	}
	wg.Wait()
	close(results)
	close(errors)
	successes := 0
	for result := range results {
		if result.WriteResult != nil && result.WriteResult.Status == "completed" {
			successes++
		}
	}
	if successes != 1 || state.writes() != 1 {
		t.Fatalf("successes=%d writes=%d errors=%v,%v", successes, state.writes(), <-errors, <-errors)
	}
	stored, _ := store.GetReviewActionPlan(context.Background(), plan.PlanID)
	if stored.Status != "completed" || stored.Reconciliation != "verified" || stored.RequestID == "" {
		t.Fatalf("stored plan = %#v", stored)
	}
}

func TestCancelReviewActionPlanIsActorScopedAndTerminal(t *testing.T) {
	store, err := OpenSQLiteReviewGatewayStore(filepath.Join(t.TempDir(), "cancel.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	now := time.Now().UTC()
	job := testReviewGatewayJob(now, "cancel")
	plan, _ := store.CreateReviewActionPlan(context.Background(), NewReviewActionPlan(job, "reviewer", "head", "fingerprint", "body", now))
	if _, err := store.ClaimReviewActionPlan(context.Background(), ReviewActionPlanClaimOptions{PlanID: plan.PlanID, ActorID: "wrong", LeaseOwner: "cancel", Now: now}); err == nil {
		t.Fatal("another actor cancelled the plan")
	}
	claimed, err := store.ClaimReviewActionPlan(context.Background(), ReviewActionPlanClaimOptions{PlanID: plan.PlanID, ActorID: job.RequestedBy, LeaseOwner: "cancel", Now: now})
	if err != nil {
		t.Fatal(err)
	}
	if err := store.FinishReviewActionPlan(context.Background(), claimed.PlanID, "cancelled", "", "", reviewMutationNone, now); err != nil {
		t.Fatal(err)
	}
	if _, err := store.ClaimReviewActionPlan(context.Background(), ReviewActionPlanClaimOptions{PlanID: plan.PlanID, ActorID: job.RequestedBy, LeaseOwner: "test", Now: now, LeaseDuration: time.Minute}); err == nil {
		t.Fatal("cancelled plan was claimable")
	}
}
