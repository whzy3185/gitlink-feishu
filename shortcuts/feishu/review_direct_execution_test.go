package feishu

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gitlink-org/gitlink-cli/internal/client"
	"github.com/gitlink-org/gitlink-cli/shortcuts/common"
)

type stagedReviewDataProvider struct {
	calls int
	first ReviewData
	next  ReviewData
}

func (p *stagedReviewDataProvider) FetchReviewData(context.Context, *common.RuntimeContext, ReviewDataRequest) (ReviewData, error) {
	p.calls++
	if p.calls == 1 {
		return p.first, nil
	}
	return p.next, nil
}

func TestControlledActionAutoExecutesAllSupportedActions(t *testing.T) {
	now := time.Date(2026, 8, 9, 12, 0, 0, 0, time.UTC)
	for _, test := range []struct {
		jobAction string
		argument  string
	}{
		{jobAction: "prepare_common_review", argument: "review result"},
		{jobAction: "prepare_review_approve", argument: "approved"},
		{jobAction: "prepare_review_reject", argument: "add tests"},
		{jobAction: "prepare_reject_close", argument: "out of scope"},
		{jobAction: "prepare_merge"},
	} {
		t.Run(test.jobAction, func(t *testing.T) {
			store := openActionPlanTestStore(t)
			defer store.Close()
			postApplied, posts := false, 0
			provider := &actionReviewDataProvider{
				base: completeActionReviewData(), postApplied: &postApplied,
				action: reviewActionFromPrepareJob(test.jobAction),
			}
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				path := strings.TrimSuffix(r.URL.Path, ".json")
				w.Header().Set("Content-Type", "application/json")
				switch {
				case r.Method == http.MethodGet && path == "/v1/users/me":
					_ = json.NewEncoder(w).Encode(map[string]interface{}{"code": 0, "data": map[string]interface{}{"login": "alice"}})
				case r.Method == http.MethodGet && path == "/owner/repo/pulls/42":
					_ = json.NewEncoder(w).Encode(map[string]interface{}{"code": 0, "data": map[string]interface{}{"state": "open", "ci_state": "success"}})
				case r.Method == http.MethodPost:
					posts++
					provider.mu.Lock()
					postApplied = true
					if provider.action == reviewActionCommon || provider.action == reviewActionApprove || provider.action == reviewActionReject {
						var body map[string]interface{}
						_ = json.NewDecoder(r.Body).Decode(&body)
						content, _ := body["content"].(string)
						if marker := strings.LastIndex(content, "Ref: "); marker >= 0 {
							provider.requestID = strings.TrimSpace(content[marker+len("Ref: "):])
						}
					}
					provider.mu.Unlock()
					_ = json.NewEncoder(w).Encode(map[string]interface{}{"code": 0, "data": map[string]interface{}{"id": 130}})
				default:
					http.NotFound(w, r)
				}
			}))
			defer server.Close()
			runtime := &common.RuntimeContext{Client: &client.Client{BaseURL: server.URL, HTTP: server.Client()}}
			executor := ReviewGatewayExecutor{
				Runtime: runtime, DataProvider: provider, ActionPlans: store,
				ControlledActionMode: controlledActionModeAuto, Now: func() time.Time { return now },
			}
			result, err := executor.Execute(context.Background(), ReviewGatewayJob{
				JobID: "job-auto-" + test.jobAction, Action: test.jobAction, Repository: "owner/repo", PRNumber: 42,
				ChatID: "oc_fixture", RequestedBy: "ou_alice", GitLinkLogin: "alice", Argument: test.argument,
			})
			if err != nil || result.Status != "completed" || result.ActionPlan == nil || result.ActionPlan.Status != "completed" {
				t.Fatalf("result=%#v err=%v", result, err)
			}
			if result.ExecutionPath != reviewExecutionPathGatewayDirect || !result.MutatesGitLink || posts != 1 {
				t.Fatalf("path=%q mutates=%v posts=%d", result.ExecutionPath, result.MutatesGitLink, posts)
			}
		})
	}
}

func TestControlledActionAutoFallsBackWithoutSafeIdentityMatch(t *testing.T) {
	now := time.Date(2026, 8, 9, 12, 0, 0, 0, time.UTC)
	for _, test := range []struct {
		name, login, reason string
		status              int
		missingRuntime      bool
	}{
		{name: "identity mismatch", login: "bob", status: http.StatusOK, reason: reviewFallbackIdentityMismatch},
		{name: "identity unavailable", status: http.StatusUnauthorized, reason: reviewFallbackIdentityUnverified},
		{name: "missing runtime", missingRuntime: true, reason: reviewFallbackCredentialUnavailable},
	} {
		t.Run(test.name, func(t *testing.T) {
			store := openActionPlanTestStore(t)
			defer store.Close()
			posts := 0
			var runtime *common.RuntimeContext
			if !test.missingRuntime {
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if r.Method == http.MethodPost {
						posts++
					}
					if test.status != http.StatusOK {
						w.WriteHeader(test.status)
						_, _ = w.Write([]byte(`{"message":"not authenticated"}`))
						return
					}
					_ = json.NewEncoder(w).Encode(map[string]interface{}{"code": 0, "data": map[string]interface{}{"login": test.login}})
				}))
				defer server.Close()
				runtime = &common.RuntimeContext{Client: &client.Client{BaseURL: server.URL, HTTP: server.Client()}}
			}
			executor := ReviewGatewayExecutor{
				Runtime: runtime, DataProvider: staticReviewDataProvider{data: completeActionReviewData()}, ActionPlans: store,
				ControlledActionMode: controlledActionModeAuto, Now: func() time.Time { return now },
			}
			result, err := executor.Execute(context.Background(), ReviewGatewayJob{
				JobID: "job-fallback", Action: "prepare_review_approve", Repository: "owner/repo", PRNumber: 42,
				ChatID: "oc_fixture", RequestedBy: "ou_alice", GitLinkLogin: "alice", Argument: "approved",
			})
			if err != nil || result.ActionPlan == nil || result.ActionPlan.Status != "pending_confirmation" || posts != 0 {
				t.Fatalf("result=%#v err=%v posts=%d", result, err, posts)
			}
			if result.ExecutionPath != reviewExecutionPathLocalConfirmation || result.FallbackReason != test.reason {
				t.Fatalf("path=%q reason=%q", result.ExecutionPath, result.FallbackReason)
			}
			card, cardErr := buildReviewActionPlanCard(ReviewGatewayJob{}, *result.ActionPlan, result.ExecutionPath, result.FallbackReason, result.Error)
			if cardErr != nil || !strings.Contains(card, "本地确认") {
				t.Fatalf("fallback card=%q err=%v", card, cardErr)
			}
		})
	}
}

func TestControlledActionLocalModeKeepsExistingBehavior(t *testing.T) {
	store := openActionPlanTestStore(t)
	defer store.Close()
	result, err := (&ReviewGatewayExecutor{
		DataProvider: staticReviewDataProvider{data: completeActionReviewData()}, ActionPlans: store,
		ControlledActionMode: controlledActionModeLocal,
	}).Execute(context.Background(), ReviewGatewayJob{
		JobID: "job-local", Action: "prepare_common_review", Repository: "owner/repo", PRNumber: 42,
		ChatID: "oc_fixture", RequestedBy: "ou_alice", GitLinkLogin: "alice", Argument: "review result",
	})
	if err != nil || result.ActionPlan == nil || result.ActionPlan.Status != "pending_confirmation" {
		t.Fatalf("result=%#v err=%v", result, err)
	}
	if result.ExecutionPath != reviewExecutionPathLocalConfirmation || result.FallbackReason != reviewFallbackModeLocal {
		t.Fatalf("path=%q reason=%q", result.ExecutionPath, result.FallbackReason)
	}
}

func TestControlledActionPermissionFailureIsTerminalWithoutCredentialFallback(t *testing.T) {
	store := openActionPlanTestStore(t)
	defer store.Close()
	now := time.Date(2026, 8, 9, 12, 0, 0, 0, time.UTC)
	posts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimSuffix(r.URL.Path, ".json")
		w.Header().Set("Content-Type", "application/json")
		if r.Method == http.MethodGet && path == "/v1/users/me" {
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"code": 0, "data": map[string]interface{}{"login": "alice"}})
			return
		}
		if r.Method == http.MethodPost {
			posts++
			w.WriteHeader(http.StatusForbidden)
			_, _ = w.Write([]byte(`{"message":"permission denied"}`))
			return
		}
		http.NotFound(w, r)
	}))
	defer server.Close()
	runtime := &common.RuntimeContext{Client: &client.Client{BaseURL: server.URL, HTTP: server.Client()}}
	result, err := (&ReviewGatewayExecutor{
		Runtime: runtime, DataProvider: staticReviewDataProvider{data: completeActionReviewData()}, ActionPlans: store,
		ControlledActionMode: controlledActionModeAuto, Now: func() time.Time { return now },
	}).Execute(context.Background(), ReviewGatewayJob{
		JobID: "job-forbidden", Action: "prepare_review_approve", Repository: "owner/repo", PRNumber: 42,
		ChatID: "oc_fixture", RequestedBy: "ou_alice", GitLinkLogin: "alice", Argument: "approved",
	})
	if err != nil || result.Status != "failed" || result.ActionPlan == nil || result.ActionPlan.Status != "failed" || posts != 1 {
		t.Fatalf("result=%#v err=%v posts=%d", result, err, posts)
	}
	if !strings.Contains(result.Error, "GitLink 拒绝该操作") || result.ExecutionPath != reviewExecutionPathGatewayDirect {
		t.Fatalf("error=%q path=%q", result.Error, result.ExecutionPath)
	}
	card, cardErr := buildReviewActionPlanCard(ReviewGatewayJob{}, *result.ActionPlan, result.ExecutionPath, result.FallbackReason, result.Error)
	if cardErr != nil || !strings.Contains(card, "GitLink 拒绝该操作") || strings.Contains(card, "待本地确认") {
		t.Fatalf("permission card=%q err=%v", card, cardErr)
	}
}

func TestControlledActionAutoRejectsStalePlanWithZeroMutation(t *testing.T) {
	store := openActionPlanTestStore(t)
	defer store.Close()
	now := time.Date(2026, 8, 9, 12, 0, 0, 0, time.UTC)
	current := completeActionReviewData()
	changed := current
	changed.HeadSHA = "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
	changed.SourceFingerprint = "sha256:changed"
	provider := &stagedReviewDataProvider{first: current, next: changed}
	posts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			posts++
		}
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"code": 0, "data": map[string]interface{}{"login": "alice"}})
	}))
	defer server.Close()
	runtime := &common.RuntimeContext{Client: &client.Client{BaseURL: server.URL, HTTP: server.Client()}}
	result, err := (&ReviewGatewayExecutor{
		Runtime: runtime, DataProvider: provider, ActionPlans: store,
		ControlledActionMode: controlledActionModeAuto, Now: func() time.Time { return now },
	}).Execute(context.Background(), ReviewGatewayJob{
		JobID: "job-stale", Action: "prepare_review_approve", Repository: "owner/repo", PRNumber: 42,
		ChatID: "oc_fixture", RequestedBy: "ou_alice", GitLinkLogin: "alice", Argument: "approved",
	})
	if err != nil || result.Status != "failed" || result.ActionPlan == nil || result.ActionPlan.Status != "failed" || posts != 0 {
		t.Fatalf("result=%#v err=%v posts=%d", result, err, posts)
	}
	if !strings.Contains(result.Error, "stale") || result.ActionPlan.MutationStatus != "none" {
		t.Fatalf("result=%#v", result)
	}
}

func TestControlledActionModeValidation(t *testing.T) {
	for input, expected := range map[string]string{"": controlledActionModeLocal, "LOCAL": controlledActionModeLocal, "auto": controlledActionModeAuto} {
		actual, err := normalizeControlledActionMode(input)
		if err != nil || actual != expected {
			t.Fatalf("normalize %q = %q, %v", input, actual, err)
		}
	}
	if _, err := normalizeControlledActionMode("unsafe"); err == nil {
		t.Fatal("invalid controlled action mode accepted")
	}
}
