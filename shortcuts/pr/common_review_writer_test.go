package pr

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/gitlink-org/gitlink-cli/internal/client"
	"github.com/gitlink-org/gitlink-cli/shortcuts/common"
)

type commonWriterFixture struct {
	mu             sync.Mutex
	head, state    string
	postMode       string
	readbackAbsent bool
	nullCommit     bool
	posts          int
	posted         map[string]interface{}
	existing       []map[string]interface{}
}

func (f *commonWriterFixture) server(t *testing.T) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/owner/repo/pulls/123.json":
			common.WriteJSON(t, w, map[string]interface{}{"pull_request": map[string]interface{}{
				"number": 123, "title": "fixture", "state": f.state, "head_commit_sha": f.head,
			}})
		case r.Method == http.MethodGet && r.URL.Path == "/v1/owner/repo/pulls/123/versions.json":
			common.WriteJSON(t, w, map[string]interface{}{"versions": []map[string]interface{}{{"id": 1, "head_commit_sha": f.head}}})
		case r.Method == http.MethodGet && r.URL.Path == "/users/me.json":
			common.WriteJSON(t, w, map[string]interface{}{"login": "alice", "id": 7})
		case r.Method == http.MethodGet && r.URL.Path == "/v1/owner/repo/pulls/123/reviews.json":
			f.mu.Lock()
			reviews := append([]map[string]interface{}{}, f.existing...)
			if f.posts > 0 && !f.readbackAbsent {
				commit := interface{}(f.head)
				if f.nullCommit {
					commit = nil
				}
				reviews = append(reviews, map[string]interface{}{
					"id": 91, "status": "common", "content": f.posted["content"], "commit_id": commit,
					"user": map[string]interface{}{"login": "alice"},
				})
			}
			f.mu.Unlock()
			common.WriteJSON(t, w, map[string]interface{}{"total_count": len(reviews), "reviews": reviews})
		case r.Method == http.MethodPost && r.URL.Path == "/v1/owner/repo/pulls/123/reviews.json":
			f.mu.Lock()
			f.posts++
			f.posted = common.DecodeJSON(t, r)
			mode := f.postMode
			f.mu.Unlock()
			if mode == "error" {
				http.Error(w, `{"message":"timeout"}`, http.StatusGatewayTimeout)
				return
			}
			common.WriteJSON(t, w, map[string]interface{}{"review": map[string]interface{}{"id": 91}})
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.String())
		}
	}))
}

func (f *commonWriterFixture) runtime(server *httptest.Server) *common.RuntimeContext {
	return &common.RuntimeContext{Client: &client.Client{HTTP: server.Client(), BaseURL: server.URL}, Owner: "owner", Repo: "repo"}
}

func (f *commonWriterFixture) postCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.posts
}

func TestExecuteCommonReviewVerifiedAutoHeadAndTrustedRef(t *testing.T) {
	fixture := &commonWriterFixture{head: strings.Repeat("a", 40), state: "open"}
	server := fixture.server(t)
	defer server.Close()
	result := ExecuteCommonReview(fixture.runtime(server), CommonReviewOptions{
		Owner: "owner", Repository: "repo", PRNumber: 123,
		Content: "Evidence\n\nRef: RW-FFFFFF", RequestID: "rw-82f31a",
	})
	if result.Status != "verified" || result.POSTCount != 1 || fixture.postCount() != 1 ||
		result.ExpectedHead != fixture.head || result.Actor != "alice" || result.ReviewID != "91" {
		t.Fatalf("result = %#v, posts=%d", result, fixture.postCount())
	}
	posted, _ := fixture.posted["content"].(string)
	if strings.Contains(posted, "RW-FFFFFF") || !strings.HasSuffix(posted, "Ref: RW-82F31A") {
		t.Fatalf("trusted content = %q", posted)
	}
}

func TestExecuteCommonReviewPreWriteSafetyGates(t *testing.T) {
	tests := []struct {
		name, state, expected, wantStatus string
		dryRun                            bool
		before                            func() error
	}{
		{name: "stale head", state: "open", expected: strings.Repeat("b", 40), wantStatus: "stale"},
		{name: "closed", state: "closed", wantStatus: "failed"},
		{name: "dry run", state: "open", dryRun: true, wantStatus: "dry_run"},
		{name: "boundary persistence", state: "open", wantStatus: "failed", before: func() error { return context.DeadlineExceeded }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			fixture := &commonWriterFixture{head: strings.Repeat("a", 40), state: test.state}
			server := fixture.server(t)
			defer server.Close()
			result := ExecuteCommonReview(fixture.runtime(server), CommonReviewOptions{
				Owner: "owner", Repository: "repo", PRNumber: 123, Content: "Evidence",
				ExpectedHead: test.expected, DryRun: test.dryRun, BeforePOST: test.before,
			})
			if result.Status != test.wantStatus || fixture.postCount() != 0 {
				t.Fatalf("result=%#v posts=%d", result, fixture.postCount())
			}
		})
	}
}

func TestExecuteCommonReviewDuplicateAndUncertainNeverReposts(t *testing.T) {
	head := strings.Repeat("a", 40)
	t.Run("duplicate", func(t *testing.T) {
		fixture := &commonWriterFixture{head: head, state: "open", existing: []map[string]interface{}{{
			"id": 17, "status": "common", "content": "Evidence\n\nRef: RW-82F31A", "commit_id": head,
			"user": map[string]interface{}{"login": "alice"},
		}}}
		server := fixture.server(t)
		defer server.Close()
		result := ExecuteCommonReview(fixture.runtime(server), CommonReviewOptions{
			Owner: "owner", Repository: "repo", PRNumber: 123, Content: "Evidence", RequestID: "RW-82F31A",
		})
		if result.Status != "duplicate" || result.ReviewID != "17" || fixture.postCount() != 0 {
			t.Fatalf("result=%#v posts=%d", result, fixture.postCount())
		}
	})
	for _, mode := range []string{"post error", "readback absent"} {
		t.Run(mode, func(t *testing.T) {
			fixture := &commonWriterFixture{head: head, state: "open", postMode: "error", readbackAbsent: true}
			if mode == "readback absent" {
				fixture.postMode = ""
			}
			server := fixture.server(t)
			defer server.Close()
			result := ExecuteCommonReview(fixture.runtime(server), CommonReviewOptions{
				Owner: "owner", Repository: "repo", PRNumber: 123, Content: "Evidence",
			})
			if result.Status != "unknown" || result.POSTCount != 1 || fixture.postCount() != 1 {
				t.Fatalf("result=%#v posts=%d", result, fixture.postCount())
			}
		})
	}
}

func TestExecuteCommonReviewPreservesNullCommit(t *testing.T) {
	fixture := &commonWriterFixture{head: strings.Repeat("a", 40), state: "open", nullCommit: true}
	server := fixture.server(t)
	defer server.Close()
	result := ExecuteCommonReview(fixture.runtime(server), CommonReviewOptions{
		Owner: "owner", Repository: "repo", PRNumber: 123, Content: "Evidence",
	})
	encoded, _ := json.Marshal(result)
	if result.Status != "verified" || result.RemoteCommitID != nil || len(result.Warnings) != 1 ||
		!strings.Contains(string(encoded), `"remote_commit_id":null`) || fixture.postCount() != 1 {
		t.Fatalf("result=%s posts=%d", encoded, fixture.postCount())
	}
}

func TestBuildCommonReviewContentRejectsInvalidRequestID(t *testing.T) {
	if _, err := BuildCommonReviewContent("Evidence", "user supplied"); err == nil {
		t.Fatal("invalid request ID accepted")
	}
}

func TestPRCommonReviewShortcutUsesGuardedWriter(t *testing.T) {
	fixture := &commonWriterFixture{head: strings.Repeat("a", 40), state: "open"}
	server := fixture.server(t)
	defer server.Close()
	if err := runPRShortcut(t, server, "review", map[string]string{
		"id": "123", "content": "Evidence", "status": "common", "request-id": "RW-82F31A",
	}); err != nil {
		t.Fatalf("common Review shortcut: %v", err)
	}
	if fixture.postCount() != 1 {
		t.Fatalf("POST count = %d", fixture.postCount())
	}
}
