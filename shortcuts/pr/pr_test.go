package pr

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gitlink-org/gitlink-cli/internal/client"
	"github.com/gitlink-org/gitlink-cli/internal/output"
	"github.com/gitlink-org/gitlink-cli/shortcuts/common"
)

func TestPRCommentPostsToPullJournals(t *testing.T) {
	var journalPayload map[string]interface{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" || r.URL.Path != "/v1/owner/repo/pulls/13/journals.json" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		journalPayload = decodeJSON(t, r)
		writeJSON(t, w, map[string]interface{}{
			"id":   float64(12345),
			"note": "LGTM, looks good!",
		})
	}))
	defer server.Close()

	err := runPRShortcut(t, server, "comment", map[string]string{
		"id":   "13",
		"body": "LGTM, looks good!",
	})
	if err != nil {
		t.Fatalf("comment shortcut failed: %v", err)
	}
	assertEqual(t, journalPayload["note"], "LGTM, looks good!")
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

func TestPRCheckoutFetchesHeadBranch(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}
	// Upstream repo with a PR head branch.
	upstream := t.TempDir()
	run := func(args ...string) {
		t.Helper()
		if out, err := exec.Command("git", args...).CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v (%s)", args, err, out)
		}
	}
	run("init", "-q", "--initial-branch=master", upstream)
	run("-C", upstream, "-c", "user.name=t", "-c", "user.email=t@t", "commit", "-q", "--allow-empty", "-m", "init")
	run("-C", upstream, "branch", "feat/x")

	// Local clone where checkout happens.
	local := filepath.Join(t.TempDir(), "local")
	run("clone", "-q", upstream, local)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/owner/repo/pulls/42.json" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		writeJSON(t, w, map[string]interface{}{"head": "feat/x", "base": "master"})
	}))
	defer server.Close()

	cwd, _ := os.Getwd()
	if err := os.Chdir(local); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(cwd)

	if err := runPRShortcut(t, server, "checkout", map[string]string{"id": "42"}); err != nil {
		t.Fatalf("checkout shortcut failed: %v", err)
	}
	out, err := exec.Command("git", "-C", local, "branch", "--show-current").Output()
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.TrimSpace(string(out)); got != "feat/x" {
		t.Fatalf("current branch = %q, want feat/x", got)
	}
}

func TestPRCheckoutMissingHead(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(t, w, map[string]interface{}{"base": "master"})
	}))
	defer server.Close()

	if err := runPRShortcut(t, server, "checkout", map[string]string{"id": "42"}); err == nil {
		t.Fatal("expected error when head missing")
	}
}
