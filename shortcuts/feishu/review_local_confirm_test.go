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
	result := ReviewGatewayExecutionResult{Repository: plan.Repository, PRNumber: plan.PRNumber, ActionPlan: &plan}
	cardJSON, ok := safeReviewGatewayCardJSON(buildReviewGatewayResultCard(job, result, nil))
	for _, expected := range []string{plan.RequestID, plan.GitLinkLogin, "review-confirm-local", "本地执行 Review", "取消 Review", "刷新 owner/repo PR #42"} {
		if !ok || !strings.Contains(cardJSON, expected) {
			t.Fatalf("ActionPlan card missing %q: %s", expected, cardJSON)
		}
	}
	for _, unsafe := range []string{"approve owner/repo#42", "reject owner/repo#42", "merge owner/repo#42"} {
		if got := parseReviewGatewayIntent(unsafe); got.Name != "unknown" {
			t.Fatalf("unsafe command %q mapped to %#v", unsafe, got)
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
	stale := plan
	stale.SourceFingerprint = "changed"
	if _, err := executeLocalReviewConfirmation(context.Background(), runtime, store, stale, false, true, now.Add(time.Second)); err == nil || state.writes() != 0 {
		t.Fatalf("stale err=%v writes=%d", err, state.writes())
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
