package pr

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gitlink-org/gitlink-cli/internal/client"
	"github.com/gitlink-org/gitlink-cli/internal/output"
	"github.com/gitlink-org/gitlink-cli/shortcuts/common"
)

func TestPRCommentPostsToCorrectIssueJournal(t *testing.T) {
	var journalPayload map[string]interface{}
	var journalPath string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == "GET" && r.URL.Path == "/owner/repo/pulls/13.json":
			common.WriteJSON(t, w, map[string]interface{}{
				"issue": map[string]interface{}{
					"id":      float64(142301),
					"subject": "test PR",
				},
				"pull_request": map[string]interface{}{
					"id": float64(14791),
				},
			})
		case r.Method == "POST" && r.URL.Path == "/v1/owner/repo/issues/142301/journals.json":
			journalPath = r.URL.Path
			journalPayload = common.DecodeJSON(t, r)
			common.WriteJSON(t, w, map[string]interface{}{
				"id":      float64(12345),
				"message": "评论成功",
			})
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	err := runPRShortcut(t, server, "comment", map[string]string{
		"id":   "13",
		"body": "LGTM, looks good!",
	})
	if err != nil {
		t.Fatalf("comment shortcut failed: %v", err)
	}

	if journalPath == "" {
		t.Fatal("journal endpoint was not called")
	}
	common.AssertEqual(t, journalPayload["notes"], "LGTM, looks good!")
}

func TestPRCommentFailsWhenPRNotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		common.WriteJSON(t, w, map[string]interface{}{
			"status": 404,
			"error":  "Not Found",
		})
	}))
	defer server.Close()

	err := runPRShortcut(t, server, "comment", map[string]string{
		"id":   "999",
		"body": "test",
	})
	if err == nil {
		t.Fatal("expected error for non-existent PR, got nil")
	}
}

func TestPRCommentFailsWhenIssueFieldMissing(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		common.WriteJSON(t, w, map[string]interface{}{
			"pull_request": map[string]interface{}{
				"id": float64(14791),
			},
		})
	}))
	defer server.Close()

	err := runPRShortcut(t, server, "comment", map[string]string{
		"id":   "13",
		"body": "test",
	})
	if err == nil {
		t.Fatal("expected error when issue field is missing, got nil")
	}
}

// --- list ---

func TestPRList(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Fatalf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/v1/owner/repo/pulls.json" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if r.URL.Query().Get("status") != "0" {
			t.Fatalf("expected status=0, got %s", r.URL.Query().Get("status"))
		}
		if r.URL.Query().Get("page") != "1" {
			t.Fatalf("expected page=1, got %s", r.URL.Query().Get("page"))
		}
		writeJSON(t, w, []interface{}{
			map[string]interface{}{"id": float64(1), "title": "PR 1"},
		})
	}))
	defer server.Close()

	err := runPRShortcut(t, server, "list", map[string]string{"state": "open", "page": "1", "limit": "20"})
	if err != nil {
		t.Fatalf("list failed: %v", err)
	}
}

func TestPRListWithFilters(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/owner/repo/pulls.json" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		query := r.URL.Query()
		assertEqual(t, query.Get("status"), "1")
		assertEqual(t, query.Get("keyword"), "release")
		assertEqual(t, query.Get("priority_id"), "2")
		assertEqual(t, query.Get("issue_tag_id"), "3")
		assertEqual(t, query.Get("version_id"), "4")
		assertEqual(t, query.Get("reviewer_id"), "5")
		assertEqual(t, query.Get("assign_user_id"), "6")
		assertEqual(t, query.Get("sort_by"), "updated_at")
		assertEqual(t, query.Get("sort_direction"), "desc")
		writeJSON(t, w, map[string]interface{}{"pulls": []interface{}{}})
	}))
	defer server.Close()

	err := runPRShortcut(t, server, "list", map[string]string{
		"state":          "merged",
		"keyword":        "release",
		"priority-id":    "2",
		"tag-id":         "3",
		"milestone-id":   "4",
		"reviewer-id":    "5",
		"assignee-id":    "6",
		"sort-by":        "updated_at",
		"sort-direction": "desc",
		"page":           "2",
		"limit":          "50",
	})
	if err != nil {
		t.Fatalf("list with filters failed: %v", err)
	}
}

func TestPRListStateAllOmitsStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/owner/repo/pulls.json" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if got := r.URL.Query().Get("status"); got != "" {
			t.Fatalf("status should be omitted for all, got %q", got)
		}
		writeJSON(t, w, map[string]interface{}{"pulls": []interface{}{}})
	}))
	defer server.Close()

	err := runPRShortcut(t, server, "list", map[string]string{"state": "all", "page": "1", "limit": "20"})
	if err != nil {
		t.Fatalf("list all failed: %v", err)
	}
}

// --- create ---

func TestPRCreate(t *testing.T) {
	var payload map[string]interface{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Fatalf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/owner/repo/pulls.json" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		payload = decodeJSON(t, r)
		writeJSON(t, w, map[string]interface{}{"id": float64(42), "title": "feat: new"})
	}))
	defer server.Close()

	err := runPRShortcut(t, server, "create", map[string]string{
		"title": "feat: new",
		"head":  "feature/x",
		"base":  "master",
		"body":  "description",
	})
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}
	assertEqual(t, payload["title"], "feat: new")
	assertEqual(t, payload["head"], "feature/x")
	assertEqual(t, payload["base"], "master")
	assertEqual(t, payload["body"], "description")
}

func TestPRCreateNoBody(t *testing.T) {
	var payload map[string]interface{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == "GET" && r.URL.Path == "/owner/repo.json":
			writeJSON(t, w, map[string]interface{}{"default_branch": "master"})
		case r.Method == "POST" && r.URL.Path == "/owner/repo/pulls.json":
			payload = decodeJSON(t, r)
			writeJSON(t, w, map[string]interface{}{"id": float64(43), "title": "feat: nob"})
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	err := runPRShortcut(t, server, "create", map[string]string{
		"title": "feat: nob",
		"head":  "feature/y",
	})
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}
	if _, ok := payload["body"]; ok {
		t.Fatal("body should not be in payload when not provided")
	}
	assertEqual(t, payload["base"], "master")
}

// --- view ---

func TestPRView(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Fatalf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/owner/repo/pulls/42.json" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		writeJSON(t, w, map[string]interface{}{
			"id":    float64(42),
			"title": "feat: new",
		})
	}))
	defer server.Close()

	err := runPRShortcut(t, server, "view", map[string]string{"id": "42"})
	if err != nil {
		t.Fatalf("view failed: %v", err)
	}
}

// --- merge ---

func TestPRMerge(t *testing.T) {
	var payload map[string]interface{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Fatalf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/owner/repo/pulls/42/pr_merge.json" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		payload = decodeJSON(t, r)
		writeJSON(t, w, map[string]interface{}{"message": "merged"})
	}))
	defer server.Close()

	err := runPRShortcut(t, server, "merge", map[string]string{"id": "42"})
	if err != nil {
		t.Fatalf("merge failed: %v", err)
	}
	assertEqual(t, payload["do"], "merge")
}

func TestPRMergeSquash(t *testing.T) {
	var payload map[string]interface{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		payload = decodeJSON(t, r)
		writeJSON(t, w, map[string]interface{}{"message": "squashed"})
	}))
	defer server.Close()

	err := runPRShortcut(t, server, "merge", map[string]string{"id": "42", "method": "squash"})
	if err != nil {
		t.Fatalf("merge squash failed: %v", err)
	}
	assertEqual(t, payload["do"], "squash")
}

// --- refuse ---

func TestPRRefuse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Fatalf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/owner/repo/pulls/42/refuse_merge.json" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		writeJSON(t, w, map[string]interface{}{"message": "closed"})
	}))
	defer server.Close()

	err := runPRShortcut(t, server, "refuse", map[string]string{"id": "42"})
	if err != nil {
		t.Fatalf("refuse failed: %v", err)
	}
}

// --- files ---

func TestPRFiles(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Fatalf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/owner/repo/pulls/42/files.json" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		writeJSON(t, w, []interface{}{
			map[string]interface{}{"filename": "main.go", "status": "modified"},
		})
	}))
	defer server.Close()

	err := runPRShortcut(t, server, "files", map[string]string{"id": "42"})
	if err != nil {
		t.Fatalf("files failed: %v", err)
	}
}

// --- diff ---

func TestPRDiff(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Fatalf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/owner/repo/pulls/42/files.json" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		writeJSON(t, w, []interface{}{
			map[string]interface{}{"filename": "main.go", "patch": "@@ -1 +1 @@"},
		})
	}))
	defer server.Close()

	err := runPRShortcut(t, server, "diff", map[string]string{"id": "42"})
	if err != nil {
		t.Fatalf("diff failed: %v", err)
	}
}

// --- extractIssueID ---

func TestExtractIssueID(t *testing.T) {
	id, err := extractIssueID(&output.Envelope{Data: map[string]interface{}{
		"issue": map[string]interface{}{"id": float64(42)},
	}})
	if err != nil {
		t.Fatalf("extractIssueID error: %v", err)
	}
	if id != 42 {
		t.Fatalf("= %d, want 42", id)
	}
}

func TestExtractIssueIDNotMap(t *testing.T) {
	_, err := extractIssueID(&output.Envelope{Data: "not a map"})
	if err == nil {
		t.Fatal("expected error for non-map data")
	}
}

func TestExtractIssueIDMissingIssue(t *testing.T) {
	_, err := extractIssueID(&output.Envelope{Data: map[string]interface{}{"pr": map[string]interface{}{}}})
	if err == nil {
		t.Fatal("expected error for missing issue field")
	}
}

func TestExtractIssueIDMissingID(t *testing.T) {
	_, err := extractIssueID(&output.Envelope{Data: map[string]interface{}{
		"issue": map[string]interface{}{"subject": "test"},
	}})
	if err == nil {
		t.Fatal("expected error for missing issue.id")
	}
}

// --- HTTP error paths ---

func TestPRListHTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("server error"))
	}))
	defer server.Close()

	err := runPRShortcut(t, server, "list", map[string]string{"page": "1", "limit": "20"})
	if err == nil {
		t.Fatal("expected error for HTTP 500")
	}
}

func TestPRCreateHTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("server error"))
	}))
	defer server.Close()

	err := runPRShortcut(t, server, "create", map[string]string{"title": "test", "head": "feature/x"})
	if err == nil {
		t.Fatal("expected error for HTTP 500")
	}
}

func TestPRViewHTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("server error"))
	}))
	defer server.Close()

	err := runPRShortcut(t, server, "view", map[string]string{"id": "42"})
	if err == nil {
		t.Fatal("expected error for HTTP 500")
	}
}

func TestPRMergeHTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("server error"))
	}))
	defer server.Close()

	err := runPRShortcut(t, server, "merge", map[string]string{"id": "42"})
	if err == nil {
		t.Fatal("expected error for HTTP 500")
	}
}

func TestPRRefuseHTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("server error"))
	}))
	defer server.Close()

	err := runPRShortcut(t, server, "refuse", map[string]string{"id": "42"})
	if err == nil {
		t.Fatal("expected error for HTTP 500")
	}
}

func TestPRFilesHTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("server error"))
	}))
	defer server.Close()

	err := runPRShortcut(t, server, "files", map[string]string{"id": "42"})
	if err == nil {
		t.Fatal("expected error for HTTP 500")
	}
}

func TestPRDiffHTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("server error"))
	}))
	defer server.Close()

	err := runPRShortcut(t, server, "diff", map[string]string{"id": "42"})
	if err == nil {
		t.Fatal("expected error for HTTP 500")
	}
}

// --- commits ---

func TestPRCommits(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Fatalf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/v1/owner/repo/pulls/42/commits.json" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		writeJSON(t, w, []interface{}{
			map[string]interface{}{"sha": "abc1234", "message": "fix: bug"},
		})
	}))
	defer server.Close()

	err := runPRShortcut(t, server, "commits", map[string]string{"id": "42"})
	if err != nil {
		t.Fatalf("commits failed: %v", err)
	}
}

func TestPRCommitsHTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("server error"))
	}))
	defer server.Close()

	err := runPRShortcut(t, server, "commits", map[string]string{"id": "42"})
	if err == nil {
		t.Fatal("expected error for HTTP 500")
	}
}

// --- branches ---

func TestPRBranches(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Fatalf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/owner/repo/pulls/get_branches.json" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		writeJSON(t, w, []interface{}{
			map[string]interface{}{"name": "master"},
			map[string]interface{}{"name": "develop"},
		})
	}))
	defer server.Close()

	err := runPRShortcut(t, server, "branches", map[string]string{})
	if err != nil {
		t.Fatalf("branches failed: %v", err)
	}
}

func TestPRBranchesHTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("server error"))
	}))
	defer server.Close()

	err := runPRShortcut(t, server, "branches", map[string]string{})
	if err == nil {
		t.Fatal("expected error for HTTP 500")
	}
}

// --- check-merge ---

func TestPRCheckMerge(t *testing.T) {
	var payload map[string]interface{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Fatalf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/owner/repo/pulls/check_can_merge.json" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		payload = decodeJSON(t, r)
		writeJSON(t, w, map[string]interface{}{"can_merge": true})
	}))
	defer server.Close()

	err := runPRShortcut(t, server, "check-merge", map[string]string{
		"head": "feature/x",
		"base": "master",
	})
	if err != nil {
		t.Fatalf("check-merge failed: %v", err)
	}
	assertEqual(t, payload["head"], "feature/x")
	assertEqual(t, payload["base"], "master")
}

func TestPRCheckMergeHTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("server error"))
	}))
	defer server.Close()

	err := runPRShortcut(t, server, "check-merge", map[string]string{
		"head": "feature/x",
		"base": "master",
	})
	if err == nil {
		t.Fatal("expected error for HTTP 500")
	}
}

func runPRShortcut(t *testing.T, server *httptest.Server, name string, args map[string]string) error {
	t.Helper()
	shortcut := findPRShortcut(t, name)
	ctx := &common.RuntimeContext{
		Client: &client.Client{
			HTTP:    server.Client(),
			BaseURL: server.URL,
		},
		Owner:  "owner",
		Repo:   "repo",
		Format: "json",
		Args:   args,
	}
	return shortcut.Run(ctx)
}

func findPRShortcut(t *testing.T, name string) *common.Shortcut {
	t.Helper()
	for _, shortcut := range Shortcuts() {
		if shortcut.Name == name {
			return shortcut
		}
	}
	t.Fatalf("shortcut %q not found", name)
	return nil
}

// --- Review tests (from master) ---

func TestPRReviewsList(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" && r.URL.Path == "/v1/owner/repo/pulls/13/reviews.json" {
			common.WriteJSON(t, w, map[string]interface{}{
				"total_count": 1,
				"reviews": []interface{}{
					map[string]interface{}{
						"id":      float64(1),
						"content": "LGTM",
						"status":  "approved",
					},
				},
			})
		} else {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	err := runPRShortcut(t, server, "reviews", map[string]string{
		"id": "13",
	})
	if err != nil {
		t.Fatalf("reviews list failed: %v", err)
	}
}

func TestPRReviewCreate(t *testing.T) {
	var reviewPayload map[string]interface{}
	requestCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		if r.Method == "POST" && r.URL.Path == "/v1/owner/repo/pulls/13/reviews.json" {
			reviewPayload = common.DecodeJSON(t, r)
			common.WriteJSON(t, w, map[string]interface{}{
				"id":      float64(2),
				"content": "Looks good",
				"status":  "approved",
			})
		} else {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	err := runPRShortcut(t, server, "review", map[string]string{
		"id":      "13",
		"content": "Looks good",
		"status":  "approved",
		"commit":  "abc123",
	})
	if err != nil {
		t.Fatalf("review create failed: %v", err)
	}

	common.AssertEqual(t, requestCount, 1)
	common.AssertEqual(t, reviewPayload["content"], "Looks good")
	common.AssertEqual(t, reviewPayload["status"], "approved")
	common.AssertEqual(t, reviewPayload["commit_id"], "abc123")
}

func TestPRReviewDryRunDoesNotCallServer(t *testing.T) {
	requestCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		t.Fatalf("dry-run must not send a request: %s %s", r.Method, r.URL.Path)
	}))
	defer server.Close()

	err := runPRShortcut(t, server, "review", map[string]string{
		"id":      "13",
		"content": "Looks good",
		"status":  "approved",
		"commit":  "abc123",
		"dry-run": "true",
	})
	if err != nil {
		t.Fatalf("review dry-run failed: %v", err)
	}

	common.AssertEqual(t, requestCount, 0)
}

// --- Diff tests ---

func TestPRDiffWithFileFilter(t *testing.T) {
	var requestPath string
	var requestQuery string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestPath = r.URL.Path
		requestQuery = r.URL.RawQuery
		common.WriteJSON(t, w, []interface{}{
			map[string]interface{}{"filename": "src/main.go", "patch": "@@ -1 +1 @@"},
		})
	}))
	defer server.Close()

	err := runPRShortcut(t, server, "diff", map[string]string{
		"id":   "42",
		"file": "src/main.go",
	})
	if err != nil {
		t.Fatalf("diff with file filter failed: %v", err)
	}

	if requestPath != "/owner/repo/pulls/42/files.json" {
		t.Fatalf("unexpected path: %s", requestPath)
	}
	if !strings.Contains(requestQuery, "filepath=src") {
		t.Errorf("expected filepath query param, got: %s", requestQuery)
	}
}

func TestPRCheckMergePostsCorrectPayload(t *testing.T) {
	var requestMethod string
	var requestPath string
	var checkPayload map[string]interface{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestMethod = r.Method
		requestPath = r.URL.Path
		if r.Body != nil {
			checkPayload = common.DecodeJSON(t, r)
		}
		common.WriteJSON(t, w, map[string]interface{}{
			"can_merge": true,
		})
	}))
	defer server.Close()

	err := runPRShortcut(t, server, "check-merge", map[string]string{
		"head": "feature-branch",
		"base": "master",
	})
	if err != nil {
		t.Fatalf("check-merge shortcut failed: %v", err)
	}

	common.AssertEqual(t, requestMethod, "POST")
	common.AssertEqual(t, requestPath, "/owner/repo/pulls/check_can_merge.json")
	common.AssertEqual(t, checkPayload["head"], "feature-branch")
	common.AssertEqual(t, checkPayload["base"], "master")
}

func TestPRBranchesList(t *testing.T) {
	var requestMethod string
	var requestPath string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestMethod = r.Method
		requestPath = r.URL.Path
		common.WriteJSON(t, w, map[string]interface{}{
			"branches": []interface{}{
				"master",
				"develop",
			},
		})
	}))
	defer server.Close()

	err := runPRShortcut(t, server, "branches", map[string]string{})
	if err != nil {
		t.Fatalf("branches shortcut failed: %v", err)
	}

	common.AssertEqual(t, requestMethod, "GET")
	common.AssertEqual(t, requestPath, "/owner/repo/pulls/get_branches.json")
}

func TestPRDiffFailsWhenPRNotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		common.WriteJSON(t, w, map[string]interface{}{
			"status": float64(404),
			"error":  "Not Found",
		})
	}))
	defer server.Close()

	err := runPRShortcut(t, server, "diff", map[string]string{
		"id": "999",
	})
	if err == nil {
		t.Fatal("expected error for non-existent PR, got nil")
	}
}

// writeJSON is a thin local alias used by the upstream-merged PR tests; it
// delegates to common.WriteJSON to avoid a second copy of the implementation.
func writeJSON(t *testing.T, w http.ResponseWriter, payload interface{}) {
	t.Helper()
	common.WriteJSON(t, w, payload)
}

// assertEqual is a thin local alias used by the upstream-merged PR tests; it
// delegates to common.AssertEqual.
func assertEqual(t *testing.T, got interface{}, want interface{}) {
	t.Helper()
	common.AssertEqual(t, got, want)
}

// decodeJSON decodes an HTTP request body into a map; used by the
// upstream-merged PR tests that assert on request payloads.
func decodeJSON(t *testing.T, r *http.Request) map[string]interface{} {
	t.Helper()
	var payload map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		t.Fatalf("failed to decode request body: %v", err)
	}
	return payload
}
