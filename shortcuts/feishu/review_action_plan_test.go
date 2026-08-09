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
)

type actionReviewDataProvider struct {
	mu          sync.Mutex
	base        ReviewData
	postApplied *bool
	requestID   string
	action      string
}

func (p *actionReviewDataProvider) FetchReviewData(context.Context, *common.RuntimeContext, ReviewDataRequest) (ReviewData, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	data := p.base
	if p.postApplied != nil && *p.postApplied {
		switch p.action {
		case reviewActionCommon, reviewActionApprove, reviewActionReject:
			data.Reviews = []ReviewDataReview{{
				ID: "130", Status: reviewStatusForAction(p.action), CommitID: data.HeadSHA,
				Content: "review result\n\nRef: " + p.requestID, Freshness: "current",
			}}
		case reviewActionRejectClose:
			data.State = "closed"
		case reviewActionMerge:
			data.State = "merged"
		}
	}
	return data, nil
}

func completeActionReviewData() ReviewData {
	return ReviewData{
		SchemaVersion: reviewDataSchemaVersion, Repository: "owner/repo", PullRequest: 42,
		State: "open", HeadSHA: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		CollectionStatus: "complete", SourceFingerprint: "sha256:fixture-fingerprint",
	}
}

func TestParseControlledReviewCommandsAndAliases(t *testing.T) {
	tests := []struct{ input, action, argument string }{
		{"提交审查意见 owner/repo PR #42 looks good", "prepare_common_review", "looks good"},
		{"批准 owner/repo PR #42 approved", "prepare_review_approve", "approved"},
		{"需要修改 owner/repo PR #42 add tests", "prepare_review_reject", "add tests"},
		{"要求修改 owner/repo PR #42 add tests", "prepare_review_reject", "add tests"},
		{"拒绝并关闭 owner/repo PR #42 invalid", "prepare_reject_close", "invalid"},
		{"合并 owner/repo PR #42", "prepare_merge", ""},
		{"review owner/repo#42 looks good", "prepare_common_review", "looks good"},
		{"approve owner/repo#42 approved", "prepare_review_approve", "approved"},
		{"reject owner/repo#42 add tests", "prepare_review_reject", "add tests"},
		{"refuse owner/repo#42 invalid", "prepare_reject_close", "invalid"},
		{"merge owner/repo#42", "prepare_merge", ""},
	}
	for _, test := range tests {
		intent := parseReviewGatewayIntent(test.input)
		if intent.Name != test.action || intent.Repository != "owner/repo" || intent.PRNumber != 42 || intent.Argument != test.argument {
			t.Fatalf("parse %q = %#v", test.input, intent)
		}
	}
}

func TestPrepareControlledReviewActionCreatesPlanWithZeroGitLinkWrites(t *testing.T) {
	store := openActionPlanTestStore(t)
	defer store.Close()
	now := time.Date(2026, 8, 9, 10, 0, 0, 0, time.UTC)
	executor := ReviewGatewayExecutor{
		Runtime: &common.RuntimeContext{}, DataProvider: staticReviewDataProvider{data: completeActionReviewData()},
		ActionPlans: store, Now: func() time.Time { return now },
	}
	result, err := executor.Execute(context.Background(), ReviewGatewayJob{
		JobID: "job-plan", Action: "prepare_review_approve", Repository: "owner/repo", PRNumber: 42,
		ChatID: "oc_fixture", RequestedBy: "ou_alice", GitLinkLogin: "alice", Argument: "approved",
	})
	if err != nil || result.ActionPlan == nil || result.ActionPlan.Status != "pending_confirmation" || result.MutatesGitLink {
		t.Fatalf("result = %#v, err=%v", result, err)
	}
	if result.ActionPlan.ExpectedHeadSHA != completeActionReviewData().HeadSHA || result.ActionPlan.RequestID == "" {
		t.Fatalf("plan = %#v", result.ActionPlan)
	}
}

func TestReviewActionDryRunIdentityMismatchStaleAndSingleMutation(t *testing.T) {
	now := time.Date(2026, 8, 9, 10, 0, 0, 0, time.UTC)
	data := completeActionReviewData()
	for _, test := range []struct {
		name         string
		login        string
		providerData ReviewData
		dryRun       bool
		wantErr      bool
		wantPosts    int
		wantStatus   string
	}{
		{name: "dry run", login: "alice", providerData: data, dryRun: true, wantStatus: "pending_confirmation"},
		{name: "identity mismatch", login: "bob", providerData: data, wantErr: true, wantStatus: "pending_confirmation"},
		{name: "stale head", login: "alice", providerData: func() ReviewData {
			changed := data
			changed.HeadSHA = "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
			changed.SourceFingerprint = "sha256:changed"
			return changed
		}(), wantErr: true, wantStatus: "pending_confirmation"},
		{name: "single mutation", login: "alice", providerData: data, wantPosts: 1, wantStatus: "completed"},
	} {
		t.Run(test.name, func(t *testing.T) {
			store := openActionPlanTestStore(t)
			defer store.Close()
			plan, _ := NewReviewActionPlan(ReviewGatewayJob{JobID: "job", ChatID: "chat", RequestedBy: "user", GitLinkLogin: "alice", Repository: "owner/repo", PRNumber: 42}, data, reviewActionCommon, "review result", now)
			plan, _ = store.CreateReviewActionPlan(context.Background(), plan)
			postApplied := false
			posts := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				path := strings.TrimSuffix(r.URL.Path, ".json")
				w.Header().Set("Content-Type", "application/json")
				switch {
				case r.Method == http.MethodGet && path == "/v1/users/me":
					_ = json.NewEncoder(w).Encode(map[string]interface{}{"code": 0, "data": map[string]interface{}{"login": test.login}})
				case r.Method == http.MethodPost && path == "/v1/owner/repo/pulls/42/reviews":
					posts++
					postApplied = true
					_ = json.NewEncoder(w).Encode(map[string]interface{}{"code": 0, "data": map[string]interface{}{"id": 130}})
				default:
					http.NotFound(w, r)
				}
			}))
			defer server.Close()
			runtime := &common.RuntimeContext{Client: &client.Client{BaseURL: server.URL, HTTP: server.Client()}}
			provider := &actionReviewDataProvider{base: test.providerData, postApplied: &postApplied, requestID: plan.RequestID, action: plan.Action}
			updated, err := executeReviewActionPlan(context.Background(), runtime, store, provider, plan, test.dryRun, now.Add(time.Minute))
			if (err != nil) != test.wantErr || posts != test.wantPosts || updated.Status != test.wantStatus {
				t.Fatalf("updated=%#v err=%v posts=%d", updated, err, posts)
			}
			if test.wantStatus == "completed" {
				if _, err := executeReviewActionPlan(context.Background(), runtime, store, provider, updated, false, now.Add(2*time.Minute)); err == nil || posts != 1 {
					t.Fatalf("terminal plan rerun err=%v posts=%d", err, posts)
				}
			}
		})
	}
}

func TestReviewActionNetworkUncertaintyStopsRetry(t *testing.T) {
	store := openActionPlanTestStore(t)
	defer store.Close()
	now := time.Date(2026, 8, 9, 10, 0, 0, 0, time.UTC)
	data := completeActionReviewData()
	plan, _ := NewReviewActionPlan(ReviewGatewayJob{JobID: "job", ChatID: "chat", RequestedBy: "user", GitLinkLogin: "alice", Repository: "owner/repo", PRNumber: 42}, data, reviewActionApprove, "approved", now)
	plan, _ = store.CreateReviewActionPlan(context.Background(), plan)
	posts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimSuffix(r.URL.Path, ".json")
		if r.Method == http.MethodGet && path == "/v1/users/me" {
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"code": 0, "data": map[string]interface{}{"login": "alice"}})
			return
		}
		if r.Method == http.MethodPost {
			posts++
			hj, _, _ := w.(http.Hijacker).Hijack()
			_ = hj.Close()
			return
		}
		http.NotFound(w, r)
	}))
	defer server.Close()
	runtime := &common.RuntimeContext{Client: &client.Client{BaseURL: server.URL, HTTP: server.Client()}}
	provider := &actionReviewDataProvider{base: data}
	updated, err := executeReviewActionPlan(context.Background(), runtime, store, provider, plan, false, now.Add(time.Minute))
	if err == nil || updated.Status != "unknown_needs_reconciliation" || updated.MutationStatus != "possible" || posts != 1 {
		t.Fatalf("updated=%#v err=%v posts=%d", updated, err, posts)
	}
	if _, err := executeReviewActionPlan(context.Background(), runtime, store, provider, updated, false, now.Add(2*time.Minute)); err == nil || posts != 1 {
		t.Fatalf("unknown plan retried err=%v posts=%d", err, posts)
	}
}

func TestControlledLifecycleActionsUseOneMutationAndReadBack(t *testing.T) {
	now := time.Date(2026, 8, 9, 10, 0, 0, 0, time.UTC)
	for _, test := range []struct {
		name, action, path string
	}{
		{"need changes", reviewActionReject, "/v1/owner/repo/pulls/42/reviews"},
		{"reject and close", reviewActionRejectClose, "/owner/repo/pulls/42/refuse_merge"},
		{"merge", reviewActionMerge, "/owner/repo/pulls/42/pr_merge"},
	} {
		t.Run(test.name, func(t *testing.T) {
			store := openActionPlanTestStore(t)
			defer store.Close()
			data := completeActionReviewData()
			content := "reason"
			if test.action == reviewActionMerge {
				content = ""
			}
			plan, _ := NewReviewActionPlan(ReviewGatewayJob{JobID: "job", ChatID: "chat", RequestedBy: "user", GitLinkLogin: "alice", Repository: "owner/repo", PRNumber: 42}, data, test.action, content, now)
			plan, _ = store.CreateReviewActionPlan(context.Background(), plan)
			postApplied, posts := false, 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				path := strings.TrimSuffix(r.URL.Path, ".json")
				w.Header().Set("Content-Type", "application/json")
				if r.Method == http.MethodGet && path == "/v1/users/me" {
					_ = json.NewEncoder(w).Encode(map[string]interface{}{"code": 0, "data": map[string]interface{}{"login": "alice"}})
					return
				}
				if r.Method == http.MethodGet && path == "/owner/repo/pulls/42" {
					_ = json.NewEncoder(w).Encode(map[string]interface{}{"code": 0, "data": map[string]interface{}{"state": "open", "ci_state": "success"}})
					return
				}
				if r.Method == http.MethodPost && path == test.path {
					posts++
					postApplied = true
					_ = json.NewEncoder(w).Encode(map[string]interface{}{"code": 0, "data": map[string]interface{}{"id": 130}})
					return
				}
				http.NotFound(w, r)
			}))
			defer server.Close()
			runtime := &common.RuntimeContext{Client: &client.Client{BaseURL: server.URL, HTTP: server.Client()}}
			provider := &actionReviewDataProvider{base: data, postApplied: &postApplied, requestID: plan.RequestID, action: plan.Action}
			updated, err := executeReviewActionPlan(context.Background(), runtime, store, provider, plan, false, now.Add(time.Minute))
			if err != nil || updated.Status != "completed" || posts != 1 {
				t.Fatalf("updated=%#v err=%v posts=%d", updated, err, posts)
			}
			if test.action == reviewActionReject && provider.base.State != "open" {
				t.Fatalf("need changes closed PR")
			}
		})
	}
}

func TestReviewActionPlanCardSeparatesPendingCompletedAndUnknown(t *testing.T) {
	base := ReviewActionPlan{
		PlanID: "review-plan-fixture", RequestID: "RW-ABC123", Repository: "owner/repo", PRNumber: 42,
		Action: reviewActionApprove, GitLinkLogin: "alice", ExpectedHeadSHA: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
	}
	for _, test := range []struct {
		status      string
		want        []string
		notExpected []string
	}{
		{status: "pending_confirmation", want: []string{"待本地确认", "尚未修改 GitLink", "RW-ABC123"}, notExpected: []string{"回读验证"}},
		{status: "completed", want: []string{"已完成", "回读验证"}, notExpected: []string{"待本地确认", "本地确认标识"}},
		{status: "unknown_needs_reconciliation", want: []string{"结果待核对", "停止自动重试"}, notExpected: []string{"回读验证"}},
	} {
		plan := base
		plan.Status = test.status
		card, err := buildReviewActionPlanCard(ReviewGatewayJob{}, plan, reviewExecutionPathLocalConfirmation, reviewFallbackModeLocal, "")
		if err != nil {
			t.Fatalf("build card: %v", err)
		}
		for _, value := range test.want {
			if !strings.Contains(card, value) {
				t.Fatalf("%s card missing %q: %s", test.status, value, card)
			}
		}
		for _, value := range test.notExpected {
			if strings.Contains(card, value) {
				t.Fatalf("%s card leaked %q: %s", test.status, value, card)
			}
		}
	}
}

func TestMergeIsBlockedByExplicitFailedCIWithZeroMutation(t *testing.T) {
	store := openActionPlanTestStore(t)
	defer store.Close()
	now := time.Date(2026, 8, 9, 10, 0, 0, 0, time.UTC)
	data := completeActionReviewData()
	plan, _ := NewReviewActionPlan(ReviewGatewayJob{JobID: "job", ChatID: "chat", RequestedBy: "user", GitLinkLogin: "alice", Repository: "owner/repo", PRNumber: 42}, data, reviewActionMerge, "", now)
	plan, _ = store.CreateReviewActionPlan(context.Background(), plan)
	posts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimSuffix(r.URL.Path, ".json")
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && path == "/v1/users/me":
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"code": 0, "data": map[string]interface{}{"login": "alice"}})
		case r.Method == http.MethodGet && path == "/owner/repo/pulls/42":
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"code": 0, "data": map[string]interface{}{"state": "open", "ci_state": "failed"}})
		case r.Method == http.MethodPost:
			posts++
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"code": 0})
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	runtime := &common.RuntimeContext{Client: &client.Client{BaseURL: server.URL, HTTP: server.Client()}}
	updated, err := executeReviewActionPlan(context.Background(), runtime, store, staticReviewDataProvider{data: data}, plan, false, now.Add(time.Minute))
	if err == nil || !strings.Contains(err.Error(), "CI") || posts != 0 || updated.Status != "pending_confirmation" {
		t.Fatalf("updated=%#v err=%v posts=%d", updated, err, posts)
	}
}

func openActionPlanTestStore(t *testing.T) *SQLiteReviewGatewayStore {
	t.Helper()
	store, err := OpenSQLiteReviewGatewayStore(filepath.Join(t.TempDir(), "gateway.db"))
	if err != nil {
		t.Fatalf("OpenSQLiteReviewGatewayStore: %v", err)
	}
	return store
}
