package pr

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
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
			writeJSON(t, w, map[string]interface{}{
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
			journalPayload = decodeJSON(t, r)
			writeJSON(t, w, map[string]interface{}{
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
	assertEqual(t, journalPayload["notes"], "LGTM, looks good!")
}

func TestPRCommentFailsWhenPRNotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		writeJSON(t, w, map[string]interface{}{
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
		writeJSON(t, w, map[string]interface{}{
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
		payload = decodeJSON(t, r)
		writeJSON(t, w, map[string]interface{}{"id": float64(43), "title": "feat: nob"})
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

// --- status ---

func TestPRStatusGrouping(t *testing.T) {
	tests := []struct {
		name        string
		openPulls   []interface{}
		reviewPulls []interface{}
		wantCreated []string
		wantReview  []string
	}{
		{
			name: "groups created and review-requested",
			openPulls: []interface{}{
				makePull(1, "mine one", "currentuser"),
				makePull(2, "theirs", "someoneelse"),
				makePull(3, "mine two", "currentuser"),
			},
			reviewPulls: []interface{}{
				makePull(4, "review me", "author4"),
			},
			wantCreated: []string{"mine one", "mine two"},
			wantReview:  []string{"review me"},
		},
		{
			name: "author filter excludes other people",
			openPulls: []interface{}{
				makePull(2, "theirs", "someoneelse"),
			},
			reviewPulls: []interface{}{},
			wantCreated: []string{},
			wantReview:  []string{},
		},
		{
			name:        "empty repository yields empty groups",
			openPulls:   []interface{}{},
			reviewPulls: []interface{}{},
			wantCreated: []string{},
			wantReview:  []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var reviewerID string
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				switch {
				case r.URL.Path == "/users/me.json":
					writeJSON(t, w, map[string]interface{}{"login": "currentuser", "id": float64(7)})
				case r.URL.Path == "/v1/owner/repo/pulls.json":
					if got := r.URL.Query().Get("status"); got != "0" {
						t.Fatalf("expected status=0, got %q", got)
					}
					if rid := r.URL.Query().Get("reviewer_id"); rid != "" {
						reviewerID = rid
						writeJSON(t, w, map[string]interface{}{"pulls": tt.reviewPulls})
						return
					}
					writeJSON(t, w, map[string]interface{}{"pulls": tt.openPulls})
				default:
					t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
				}
			}))
			defer server.Close()

			result, err := collectPullStatus(statusContext(server))
			if err != nil {
				t.Fatalf("collectPullStatus failed: %v", err)
			}

			assertEqual(t, result["login"], "currentuser")
			assertPullTitles(t, result["created"], tt.wantCreated)
			assertPullTitles(t, result["review_requested"], tt.wantReview)
			if reviewerID != "7" {
				t.Fatalf("expected reviewer_id=7 sent to API, got %q", reviewerID)
			}
		})
	}
}

func TestPRStatusPropagatesUserError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("server error"))
	}))
	defer server.Close()

	if _, err := collectPullStatus(statusContext(server)); err == nil {
		t.Fatal("expected error when /users/me fails")
	}
}

func TestPRStatusMissingLogin(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(t, w, map[string]interface{}{"id": float64(7)})
	}))
	defer server.Close()

	if _, err := collectPullStatus(statusContext(server)); err == nil {
		t.Fatal("expected error when login is missing")
	}
}

func TestPRStatusRun(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/users/me.json":
			writeJSON(t, w, map[string]interface{}{"login": "currentuser", "id": float64(7)})
		case "/v1/owner/repo/pulls.json":
			writeJSON(t, w, map[string]interface{}{"pulls": []interface{}{}})
		default:
			t.Fatalf("unexpected request: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	if err := runPRShortcut(t, server, "status", nil); err != nil {
		t.Fatalf("status run failed: %v", err)
	}
}

func statusContext(server *httptest.Server) *common.RuntimeContext {
	return &common.RuntimeContext{
		Client: &client.Client{HTTP: server.Client(), BaseURL: server.URL},
		Owner:  "owner",
		Repo:   "repo",
		Format: "json",
	}
}

func makePull(id int, title, authorLogin string) map[string]interface{} {
	return map[string]interface{}{
		"id":    float64(id),
		"title": title,
		"issue": map[string]interface{}{
			"author": map[string]interface{}{"login": authorLogin},
		},
	}
}

func assertPullTitles(t *testing.T, raw interface{}, want []string) {
	t.Helper()
	items, ok := raw.([]interface{})
	if !ok {
		t.Fatalf("expected []interface{}, got %T", raw)
	}
	if len(items) != len(want) {
		t.Fatalf("expected %d pulls, got %d", len(want), len(items))
	}
	for i, it := range items {
		m, ok := it.(map[string]interface{})
		if !ok {
			t.Fatalf("pull %d not a map: %T", i, it)
		}
		if got := stringField(m, "title"); got != want[i] {
			t.Fatalf("pull %d title = %q, want %q", i, got, want[i])
		}
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

func decodeJSON(t *testing.T, r *http.Request) map[string]interface{} {
	t.Helper()
	var payload map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		t.Fatalf("failed to decode request body: %v", err)
	}
	return payload
}

func writeJSON(t *testing.T, w http.ResponseWriter, payload interface{}) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		t.Fatalf("failed to write response: %v", err)
	}
}

func assertEqual(t *testing.T, got interface{}, want interface{}) {
	t.Helper()
	if fmt.Sprintf("%v", got) != fmt.Sprintf("%v", want) {
		t.Fatalf("got %v (%T), want %v (%T)", got, got, want, want)
	}
}
