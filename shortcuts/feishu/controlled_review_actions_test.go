package feishu

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gitlink-org/gitlink-cli/internal/client"
	"github.com/gitlink-org/gitlink-cli/shortcuts/common"
	"github.com/gitlink-org/gitlink-cli/shortcuts/workflow"
)

func TestControlledActionPlanMatrixAndLegacyCompatibility(t *testing.T) {
	store, err := OpenSQLiteReviewGatewayStore(filepath.Join(t.TempDir(), "plans.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	now := time.Now().UTC()
	for i, action := range []string{reviewActionCommon, reviewActionApprove, reviewActionReject, reviewActionRejectClose, reviewActionMerge} {
		job := testReviewGatewayJob(now, "plan-"+action)
		content := "evidence"
		if action == reviewActionMerge {
			content = ""
		}
		plan, createErr := store.CreateReviewActionPlan(context.Background(), NewControlledReviewActionPlan(
			job, "alice", "head", "fingerprint-"+action, action, content, now.Add(time.Duration(i)*time.Second),
		))
		if createErr != nil || plan.Action != action || plan.ReviewStatus != reviewStatusForAction(action) {
			t.Fatalf("action=%s plan=%#v err=%v", action, plan, createErr)
		}
	}
	bad := NewControlledReviewActionPlan(testReviewGatewayJob(now, "bad"), "alice", "head", "fp", "delete_repository", "bad", now)
	if _, err := store.CreateReviewActionPlan(context.Background(), bad); err == nil {
		t.Fatal("unknown action was stored")
	}

	legacy := NewReviewActionPlan(testReviewGatewayJob(now, "legacy"), "alice", "head", "legacy-fp", "legacy", now)
	legacy, err = store.CreateReviewActionPlan(context.Background(), legacy)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.db.Exec("UPDATE review_action_plans SET action='' WHERE plan_id=?", legacy.PlanID); err != nil {
		t.Fatal(err)
	}
	loaded, err := store.GetReviewActionPlan(context.Background(), legacy.PlanID)
	if err != nil || loaded.Action != reviewActionCommon || loaded.ReviewStatus != "common" {
		t.Fatalf("legacy plan=%#v err=%v", loaded, err)
	}
	replayed, err := store.CreateReviewActionPlan(context.Background(), NewReviewActionPlan(testReviewGatewayJob(now, "legacy"), "alice", "head", "legacy-fp", "legacy", now))
	if err != nil || replayed.PlanID != legacy.PlanID {
		t.Fatalf("legacy replay created a different plan: %#v err=%v", replayed, err)
	}
}

func TestControlledActionPlansShareTerminalAndLeaseGuards(t *testing.T) {
	store, err := OpenSQLiteReviewGatewayStore(filepath.Join(t.TempDir(), "terminals.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	now := time.Now().UTC()
	for _, terminal := range []string{"completed", "stale", "cancelled", "unknown"} {
		job := testReviewGatewayJob(now, "terminal-"+terminal)
		plan, err := store.CreateReviewActionPlan(context.Background(), NewControlledReviewActionPlan(job, "alice", "head", "fp-"+terminal, reviewActionMerge, "", now))
		if err != nil {
			t.Fatal(err)
		}
		claimed, err := store.ClaimReviewActionPlan(context.Background(), ReviewActionPlanClaimOptions{PlanID: plan.PlanID, ActorID: plan.ActorID, LeaseOwner: "owner", Now: now, LeaseDuration: time.Minute})
		if err != nil {
			t.Fatal(err)
		}
		mutation := reviewMutationNone
		if terminal == "unknown" {
			mutation = reviewMutationPossible
		}
		if err := store.FinishReviewActionPlan(context.Background(), claimed.PlanID, terminal, "", "", mutation, now); err != nil {
			t.Fatal(err)
		}
		if _, err := store.ClaimReviewActionPlan(context.Background(), ReviewActionPlanClaimOptions{PlanID: plan.PlanID, ActorID: plan.ActorID, LeaseOwner: "second", Now: now.Add(time.Second)}); err == nil {
			t.Fatalf("terminal plan %s was claimable", terminal)
		}
	}
}

func TestPrepareControlledActionsCreatesZeroWritePlansAndRiskCards(t *testing.T) {
	state := &commonReviewTestServerState{}
	server := newCommonReviewTestServer(t, state)
	defer server.Close()
	runtime := &common.RuntimeContext{Client: &client.Client{HTTP: server.Client(), BaseURL: server.URL}, Owner: "owner", Repo: "repo"}
	store, err := OpenSQLiteReviewGatewayStore(filepath.Join(t.TempDir(), "prepare.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	executor := &ReviewGatewayExecutor{Runtime: runtime, ActionPlans: store, IdentityBindings: []ReviewIdentityBinding{{InstallationID: "test", FeishuUserID: "ou_owner", GitLinkLogin: "gitlink-reviewer", Enabled: true}}}
	now := time.Now().UTC()
	for _, test := range []struct {
		jobAction, planAction, argument string
	}{
		{"prepare_common_review", reviewActionCommon, "common evidence"},
		{"prepare_review_approve", reviewActionApprove, "approve evidence"},
		{"prepare_review_reject", reviewActionReject, "reject reason"},
		{"prepare_reject_close", reviewActionRejectClose, "close reason"},
		{"prepare_merge", reviewActionMerge, ""},
	} {
		t.Run(test.planAction, func(t *testing.T) {
			job := testReviewGatewayJob(now, "prepare-"+test.planAction)
			job.Action, job.Argument = test.jobAction, test.argument
			result, runErr := executor.prepareControlledReviewAction(context.Background(), job, ReviewGatewayExecutionResult{}, now)
			if runErr != nil || result.ActionPlan == nil || result.ActionPlan.Action != test.planAction || state.writes() != 0 {
				t.Fatalf("result=%#v err=%v writes=%d", result, runErr, state.writes())
			}
			card, ok := safeReviewGatewayCardJSON(result.ResultCard)
			if !ok || !strings.Contains(card, result.ActionPlan.PlanID) || !strings.Contains(card, result.ActionPlan.RequestID) {
				t.Fatalf("card=%s", card)
			}
			if (test.planAction == reviewActionMerge || test.planAction == reviewActionRejectClose) && !strings.Contains(card, "HIGH RISK") {
				t.Fatalf("high-risk card missing warning: %s", card)
			}
		})
	}
}

func TestLocalControlledReviewApproveRejectAndDuplicate(t *testing.T) {
	for _, action := range []string{reviewActionApprove, reviewActionReject} {
		t.Run(action, func(t *testing.T) {
			state := &commonReviewTestServerState{}
			server := newCommonReviewTestServer(t, state)
			defer server.Close()
			runtime := &common.RuntimeContext{Client: &client.Client{HTTP: server.Client(), BaseURL: server.URL}, Owner: "owner", Repo: "repo"}
			current, err := workflow.FetchReviewContext(runtime, workflow.ReviewContextOptions{Owner: "owner", Repo: "repo", Number: 42, VersionLimit: 100, ThreadLimit: 100, IncludePR: true, IncludeFiles: true, IncludeVersions: true, IncludeReviews: true, IncludeThreads: true})
			if err != nil {
				t.Fatal(err)
			}
			store, err := OpenSQLiteReviewGatewayStore(filepath.Join(t.TempDir(), "review.db"))
			if err != nil {
				t.Fatal(err)
			}
			defer store.Close()
			now := time.Now().UTC()
			job := testReviewGatewayJob(now, "local-"+action)
			plan, err := store.CreateReviewActionPlan(context.Background(), NewControlledReviewActionPlan(job, "gitlink-reviewer", current.CurrentHeadSHA, current.WorkItem.SourceFingerprint, action, "decision evidence", now))
			if err != nil {
				t.Fatal(err)
			}
			dry, err := executeLocalReviewConfirmation(context.Background(), runtime, store, plan, true, false, now.Add(time.Second))
			if err != nil || dry.WriteResult.Status != "dry_run" || state.writes() != 0 {
				t.Fatalf("dry=%#v err=%v writes=%d", dry, err, state.writes())
			}
			result, err := executeLocalReviewConfirmation(context.Background(), runtime, store, plan, false, true, now.Add(2*time.Second))
			if err != nil || result.WriteResult.Status != "completed" || state.writes() != 1 || state.postedStatus != reviewStatusForAction(action) {
				t.Fatalf("result=%#v err=%v writes=%d status=%q", result, err, state.writes(), state.postedStatus)
			}
			if _, err := executeLocalReviewConfirmation(context.Background(), runtime, store, plan, false, true, now.Add(3*time.Second)); err == nil || state.writes() != 1 {
				t.Fatalf("completed plan reran: err=%v writes=%d", err, state.writes())
			}
		})
	}
}

type lifecycleGatewayFixture struct {
	mu        sync.Mutex
	state     string
	head      string
	posts     int
	postError bool
}

func (f *lifecycleGatewayFixture) server(t *testing.T) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		f.mu.Lock()
		state, head := f.state, f.head
		f.mu.Unlock()
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/owner/repo/pulls/42.json":
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"pull_request": map[string]interface{}{"number": 42, "title": "Lifecycle", "state": state, "merged": state == "merged", "head_commit_sha": head}})
		case r.Method == http.MethodGet && r.URL.Path == "/v1/owner/repo/pulls/42/versions.json":
			_, _ = w.Write([]byte(`{"versions":[{"id":1,"head_commit_sha":"` + head + `"}]}`))
		case r.Method == http.MethodGet && r.URL.Path == "/owner/repo/pulls/42/files.json":
			_, _ = w.Write([]byte(`{"files":[{"filename":"test.md"}]}`))
		case r.Method == http.MethodGet && r.URL.Path == "/v1/owner/repo/pulls/42/reviews.json":
			_, _ = w.Write([]byte(`{"reviews":[]}`))
		case r.Method == http.MethodGet && r.URL.Path == "/v1/owner/repo/pulls/42/journals.json":
			_, _ = w.Write([]byte(`{"journals":[]}`))
		case r.Method == http.MethodGet && r.URL.Path == "/users/me.json":
			_, _ = w.Write([]byte(`{"login":"gitlink-reviewer"}`))
		case r.Method == http.MethodPost && (strings.Contains(r.URL.Path, "refuse_merge") || strings.Contains(r.URL.Path, "pr_merge")):
			f.mu.Lock()
			f.posts++
			if !f.postError {
				if strings.Contains(r.URL.Path, "pr_merge") {
					f.state = "merged"
				} else {
					f.state = "closed"
				}
			}
			postError := f.postError
			f.mu.Unlock()
			if postError {
				http.Error(w, `{"message":"timeout"}`, http.StatusGatewayTimeout)
				return
			}
			_, _ = w.Write([]byte(`{"message":"accepted"}`))
		default:
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
		}
	}))
}

func (f *lifecycleGatewayFixture) writeCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.posts
}

func TestLocalRejectCloseAndMergeUseOneMutationAndPersistUnknown(t *testing.T) {
	for _, action := range []string{reviewActionRejectClose, reviewActionMerge} {
		t.Run(action, func(t *testing.T) {
			fixture := &lifecycleGatewayFixture{state: "open", head: "head-action"}
			server := fixture.server(t)
			defer server.Close()
			runtime := &common.RuntimeContext{Client: &client.Client{HTTP: server.Client(), BaseURL: server.URL}, Owner: "owner", Repo: "repo"}
			current, err := workflow.FetchReviewContext(runtime, workflow.ReviewContextOptions{Owner: "owner", Repo: "repo", Number: 42, VersionLimit: 100, ThreadLimit: 100, IncludePR: true, IncludeFiles: true, IncludeVersions: true, IncludeReviews: true, IncludeThreads: true})
			if err != nil {
				t.Fatal(err)
			}
			store, err := OpenSQLiteReviewGatewayStore(filepath.Join(t.TempDir(), "lifecycle.db"))
			if err != nil {
				t.Fatal(err)
			}
			defer store.Close()
			now := time.Now().UTC()
			job := testReviewGatewayJob(now, "local-"+action)
			content := "reject and close reason"
			if action == reviewActionMerge {
				content = ""
			}
			plan, err := store.CreateReviewActionPlan(context.Background(), NewControlledReviewActionPlan(job, "gitlink-reviewer", current.CurrentHeadSHA, current.WorkItem.SourceFingerprint, action, content, now))
			if err != nil {
				t.Fatal(err)
			}
			result, err := executeLocalReviewConfirmation(context.Background(), runtime, store, plan, false, true, now.Add(time.Second))
			if err != nil || result.WriteResult.Status != "completed" || fixture.writeCount() != 1 {
				t.Fatalf("result=%#v err=%v writes=%d", result, err, fixture.writeCount())
			}
			if _, err := executeLocalReviewConfirmation(context.Background(), runtime, store, plan, false, true, now.Add(2*time.Second)); err == nil || fixture.writeCount() != 1 {
				t.Fatalf("duplicate err=%v writes=%d", err, fixture.writeCount())
			}
		})
	}

	fixture := &lifecycleGatewayFixture{state: "open", head: "head-action", postError: true}
	server := fixture.server(t)
	defer server.Close()
	runtime := &common.RuntimeContext{Client: &client.Client{HTTP: server.Client(), BaseURL: server.URL}, Owner: "owner", Repo: "repo"}
	current, _ := workflow.FetchReviewContext(runtime, workflow.ReviewContextOptions{Owner: "owner", Repo: "repo", Number: 42, VersionLimit: 100, ThreadLimit: 100, IncludePR: true, IncludeFiles: true, IncludeVersions: true, IncludeReviews: true, IncludeThreads: true})
	store, _ := OpenSQLiteReviewGatewayStore(filepath.Join(t.TempDir(), "unknown.db"))
	defer store.Close()
	now := time.Now().UTC()
	job := testReviewGatewayJob(now, "unknown-merge")
	plan, _ := store.CreateReviewActionPlan(context.Background(), NewControlledReviewActionPlan(job, "gitlink-reviewer", current.CurrentHeadSHA, current.WorkItem.SourceFingerprint, reviewActionMerge, "", now))
	result, err := executeLocalReviewConfirmation(context.Background(), runtime, store, plan, false, true, now.Add(time.Second))
	stored, _ := store.GetReviewActionPlan(context.Background(), plan.PlanID)
	if err != nil || result.WriteResult.Status != "unknown_needs_reconciliation" || fixture.writeCount() != 1 || stored.Status != "unknown" || stored.Reconciliation != "required" {
		t.Fatalf("result=%#v err=%v writes=%d stored=%#v", result, err, fixture.writeCount(), stored)
	}
}
