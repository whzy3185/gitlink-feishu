package pr

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/gitlink-org/gitlink-cli/internal/client"
	"github.com/gitlink-org/gitlink-cli/shortcuts/common"
)

type controlledActionFixture struct {
	mu          sync.Mutex
	head        string
	state       string
	actor       string
	postError   bool
	remoteMoves bool
	posts       int
	postPath    string
	payload     map[string]interface{}
}

func (f *controlledActionFixture) server(t *testing.T) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		f.mu.Lock()
		state, head, actor := f.state, f.head, f.actor
		f.mu.Unlock()
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/owner/repo/pulls/123.json":
			common.WriteJSON(t, w, map[string]interface{}{"pull_request": map[string]interface{}{
				"number": 123, "title": "controlled action", "state": state, "head_commit_sha": head,
				"merged": state == "merged",
			}})
		case r.Method == http.MethodGet && r.URL.Path == "/v1/owner/repo/pulls/123/versions.json":
			common.WriteJSON(t, w, map[string]interface{}{"versions": []map[string]interface{}{{"id": 1, "head_commit_sha": head}}})
		case r.Method == http.MethodGet && r.URL.Path == "/users/me.json":
			common.WriteJSON(t, w, map[string]interface{}{"login": actor})
		case r.Method == http.MethodPost && (r.URL.Path == "/owner/repo/pulls/123/refuse_merge.json" || r.URL.Path == "/owner/repo/pulls/123/pr_merge.json"):
			f.mu.Lock()
			f.posts++
			f.postPath = r.URL.Path
			if r.Body != nil {
				_ = json.NewDecoder(r.Body).Decode(&f.payload)
			}
			if f.remoteMoves {
				if strings.Contains(r.URL.Path, "pr_merge") {
					f.state = "merged"
				} else {
					f.state = "closed"
				}
			}
			postError := f.postError
			f.mu.Unlock()
			if postError {
				http.Error(w, `{"message":"ambiguous timeout"}`, http.StatusGatewayTimeout)
				return
			}
			common.WriteJSON(t, w, map[string]interface{}{"message": "accepted"})
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.String())
		}
	}))
}

func (f *controlledActionFixture) runtime(server *httptest.Server) *common.RuntimeContext {
	return &common.RuntimeContext{Client: &client.Client{HTTP: server.Client(), BaseURL: server.URL}, Owner: "owner", Repo: "repo"}
}

func (f *controlledActionFixture) postCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.posts
}

func TestExecuteControlledPRActionVerifiedSingleMutation(t *testing.T) {
	for _, action := range []string{ControlledActionRejectClose, ControlledActionMerge} {
		t.Run(action, func(t *testing.T) {
			fixture := &controlledActionFixture{head: strings.Repeat("a", 40), state: "open", actor: "alice", remoteMoves: true}
			server := fixture.server(t)
			defer server.Close()
			result := ExecuteControlledPRAction(fixture.runtime(server), ControlledPRActionOptions{
				Owner: "owner", Repository: "repo", PRNumber: 123, Action: action,
				ExpectedHead: fixture.head, ExpectedActor: "alice",
			})
			if result.Status != "verified" || result.POSTCount != 1 || fixture.postCount() != 1 || result.MutationStatus != "confirmed" {
				t.Fatalf("result=%#v posts=%d", result, fixture.postCount())
			}
			if action == ControlledActionMerge {
				if fixture.postPath != "/owner/repo/pulls/123/pr_merge.json" || fixture.payload["do"] != "merge" {
					t.Fatalf("merge request path=%q payload=%#v", fixture.postPath, fixture.payload)
				}
			} else if fixture.postPath != "/owner/repo/pulls/123/refuse_merge.json" {
				t.Fatalf("refuse request path=%q", fixture.postPath)
			}
		})
	}
}

func TestExecuteControlledPRActionPreWriteGates(t *testing.T) {
	head := strings.Repeat("a", 40)
	for _, test := range []struct {
		name, state, actor, expected string
		dryRun                       bool
	}{
		{name: "dry run", state: "open", actor: "alice", expected: head, dryRun: true},
		{name: "wrong actor", state: "open", actor: "mallory", expected: head},
		{name: "stale head", state: "open", actor: "alice", expected: strings.Repeat("b", 40)},
		{name: "closed", state: "closed", actor: "alice", expected: head},
		{name: "already merged", state: "merged", actor: "alice", expected: head},
	} {
		t.Run(test.name, func(t *testing.T) {
			fixture := &controlledActionFixture{head: head, state: test.state, actor: test.actor}
			server := fixture.server(t)
			defer server.Close()
			result := ExecuteControlledPRAction(fixture.runtime(server), ControlledPRActionOptions{
				Owner: "owner", Repository: "repo", PRNumber: 123, Action: ControlledActionMerge,
				ExpectedHead: test.expected, ExpectedActor: "alice", DryRun: test.dryRun,
			})
			if fixture.postCount() != 0 {
				t.Fatalf("result=%#v posts=%d", result, fixture.postCount())
			}
		})
	}
}

func TestExecuteControlledPRActionAmbiguousPOSTNeverRetries(t *testing.T) {
	for _, remoteMoves := range []bool{false, true} {
		fixture := &controlledActionFixture{head: strings.Repeat("a", 40), state: "open", actor: "alice", postError: true, remoteMoves: remoteMoves}
		server := fixture.server(t)
		result := ExecuteControlledPRAction(fixture.runtime(server), ControlledPRActionOptions{
			Owner: "owner", Repository: "repo", PRNumber: 123, Action: ControlledActionMerge,
			ExpectedHead: fixture.head, ExpectedActor: "alice",
		})
		server.Close()
		if fixture.postCount() != 1 {
			t.Fatalf("remoteMoves=%t result=%#v posts=%d", remoteMoves, result, fixture.postCount())
		}
		if remoteMoves && result.Status != "verified" || !remoteMoves && result.Status != "unknown" {
			t.Fatalf("remoteMoves=%t result=%#v", remoteMoves, result)
		}
	}
}

func TestControlledPRActionExplicitCIFailureGate(t *testing.T) {
	for _, state := range []string{"failed", "failure", "error"} {
		if !controlledPRCIHasFailed(state) {
			t.Fatalf("CI state %q was not rejected", state)
		}
	}
	if controlledPRCIHasFailed("unknown") || controlledPRCIHasFailed("success") {
		t.Fatal("non-failed CI state was rejected")
	}
}
