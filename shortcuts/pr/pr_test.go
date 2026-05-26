package pr

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gitlink-org/gitlink-cli/internal/client"
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

func TestPRVersionsUsesV1Endpoint(t *testing.T) {
	var calledPath string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" || r.URL.Path != "/v1/owner/repo/pulls/13/versions.json" {
			t.Fatalf("unexpected request: %s %s?%s", r.Method, r.URL.Path, r.URL.RawQuery)
		}
		calledPath = r.URL.Path
		writeJSON(t, w, map[string]interface{}{
			"total_count": float64(2),
			"versions": []map[string]interface{}{
				{
					"id":              float64(16039),
					"head_commit_sha": "aaaaaaaa",
				},
				{
					"id":              float64(16040),
					"head_commit_sha": "bbbbbbbb",
				},
			},
		})
	}))
	defer server.Close()

	err := runPRShortcut(t, server, "versions", map[string]string{
		"id": "13",
	})
	if err != nil {
		t.Fatalf("versions shortcut failed: %v", err)
	}
	assertEqual(t, calledPath, "/v1/owner/repo/pulls/13/versions.json")
}

func TestPRVersionDiffUsesV1EndpointWithFileFilter(t *testing.T) {
	var calledPath string
	var filepath string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" || r.URL.Path != "/v1/owner/repo/pulls/13/versions/16040/diff.json" {
			t.Fatalf("unexpected request: %s %s?%s", r.Method, r.URL.Path, r.URL.RawQuery)
		}
		calledPath = r.URL.Path
		filepath = r.URL.Query().Get("filepath")
		writeJSON(t, w, map[string]interface{}{
			"diff": "--- a/shortcuts/pr/pr.go\n+++ b/shortcuts/pr/pr.go\n",
		})
	}))
	defer server.Close()

	err := runPRShortcut(t, server, "version-diff", map[string]string{
		"id":         "13",
		"version-id": "16040",
		"file":       "shortcuts/pr/pr.go",
	})
	if err != nil {
		t.Fatalf("version-diff shortcut failed: %v", err)
	}
	assertEqual(t, calledPath, "/v1/owner/repo/pulls/13/versions/16040/diff.json")
	assertEqual(t, filepath, "shortcuts/pr/pr.go")
}

func TestPRVersionDiffRequiresVersionID(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatalf("server should not be called when version-id is missing: %s %s", r.Method, r.URL.Path)
	}))
	defer server.Close()

	err := runPRShortcut(t, server, "version-diff", map[string]string{
		"id": "13",
	})
	if err == nil {
		t.Fatal("expected error when version-id is missing, got nil")
	}
}

func TestPRReviewsUsesV1EndpointWithStatusFilter(t *testing.T) {
	var calledPath string
	var status string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" || r.URL.Path != "/v1/owner/repo/pulls/13/reviews.json" {
			t.Fatalf("unexpected request: %s %s?%s", r.Method, r.URL.Path, r.URL.RawQuery)
		}
		calledPath = r.URL.Path
		status = r.URL.Query().Get("status")
		writeJSON(t, w, map[string]interface{}{
			"total_count": float64(1),
			"reviews": []map[string]interface{}{
				{
					"id":      float64(100),
					"content": "LGTM",
					"status":  "approved",
				},
			},
		})
	}))
	defer server.Close()

	err := runPRShortcut(t, server, "reviews", map[string]string{
		"id":     "13",
		"status": "approved",
	})
	if err != nil {
		t.Fatalf("reviews shortcut failed: %v", err)
	}
	assertEqual(t, calledPath, "/v1/owner/repo/pulls/13/reviews.json")
	assertEqual(t, status, "approved")
}

func TestPRReviewPostsReviewPayload(t *testing.T) {
	var reviewPayload map[string]interface{}
	var reviewPath string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" || r.URL.Path != "/v1/owner/repo/pulls/13/reviews.json" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		reviewPath = r.URL.Path
		reviewPayload = decodeJSON(t, r)
		writeJSON(t, w, map[string]interface{}{
			"id":        float64(101),
			"content":   "Looks good",
			"status":    "approved",
			"commit_id": "abc123",
		})
	}))
	defer server.Close()

	err := runPRShortcut(t, server, "review", map[string]string{
		"id":      "13",
		"status":  "approved",
		"content": "Looks good",
		"commit":  "abc123",
	})
	if err != nil {
		t.Fatalf("review shortcut failed: %v", err)
	}
	assertEqual(t, reviewPath, "/v1/owner/repo/pulls/13/reviews.json")
	assertEqual(t, reviewPayload["content"], "Looks good")
	assertEqual(t, reviewPayload["status"], "approved")
	assertEqual(t, reviewPayload["commit_id"], "abc123")
}

func TestPRReviewDryRunDoesNotCallAPI(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatalf("server should not be called during dry-run: %s %s", r.Method, r.URL.Path)
	}))
	defer server.Close()

	err := runPRShortcut(t, server, "review", map[string]string{
		"id":      "13",
		"status":  "rejected",
		"content": "Please fix the failing tests",
		"dry-run": "true",
	})
	if err != nil {
		t.Fatalf("review dry-run failed: %v", err)
	}
}

func TestPRReviewRejectsInvalidStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatalf("server should not be called for invalid status: %s %s", r.Method, r.URL.Path)
	}))
	defer server.Close()

	err := runPRShortcut(t, server, "review", map[string]string{
		"id":      "13",
		"status":  "approve",
		"content": "LGTM",
	})
	if err == nil {
		t.Fatal("expected error for invalid review status, got nil")
	}
}

func TestPRReopenUsesV1Endpoint(t *testing.T) {
	var calledPath string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" || r.URL.Path != "/v1/owner/repo/pulls/13/reopen.json" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		calledPath = r.URL.Path
		writeJSON(t, w, map[string]interface{}{
			"status":  0,
			"message": "success",
		})
	}))
	defer server.Close()

	err := runPRShortcut(t, server, "reopen", map[string]string{
		"id": "13",
	})
	if err != nil {
		t.Fatalf("reopen shortcut failed: %v", err)
	}
	assertEqual(t, calledPath, "/v1/owner/repo/pulls/13/reopen.json")
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
