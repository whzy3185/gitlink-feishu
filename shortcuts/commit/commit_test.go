package commit

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gitlink-org/gitlink-cli/internal/client"
	"github.com/gitlink-org/gitlink-cli/shortcuts/common"
)

func TestCommitList(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" || r.URL.Path != "/v1/owner/repo/commits.json" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		writeJSON(t, w, map[string]interface{}{
			"total_count": 1,
			"commits": []map[string]interface{}{
				{"sha": "abc123", "commit_message": "initial commit"},
			},
		})
	}))
	defer server.Close()

	err := runCommitShortcut(t, server, "list", map[string]string{})
	if err != nil {
		t.Fatalf("list shortcut failed: %v", err)
	}
}

func TestCommitListWithSHA(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" || r.URL.Path != "/v1/owner/repo/commits.json" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		if r.URL.Query().Get("sha") != "develop" {
			t.Fatalf("expected sha=develop, got %s", r.URL.Query().Get("sha"))
		}
		writeJSON(t, w, map[string]interface{}{
			"total_count": 0,
			"commits":     []map[string]interface{}{},
		})
	}))
	defer server.Close()

	err := runCommitShortcut(t, server, "list", map[string]string{"sha": "develop"})
	if err != nil {
		t.Fatalf("list with sha failed: %v", err)
	}
}

func TestCommitView(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" && r.URL.Path == "/v1/owner/repo/commits/abc123/files.json" {
			writeJSON(t, w, map[string]interface{}{
				"file_nums": 1,
				"files": []map[string]interface{}{
					{"filename": "main.go", "additions": 10, "deletions": 2},
				},
			})
			return
		}
		t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
	}))
	defer server.Close()

	err := runCommitShortcut(t, server, "view", map[string]string{"sha": "abc123"})
	if err != nil {
		t.Fatalf("view shortcut failed: %v", err)
	}
}

func TestCommitDiff(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" && r.URL.Path == "/v1/owner/repo/commits/abc123/diff.json" {
			writeJSON(t, w, map[string]interface{}{
				"file_nums":      1,
				"total_addition": 10,
				"total_deletion": 2,
				"files":          []map[string]interface{}{{"name": "main.go"}},
			})
			return
		}
		t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
	}))
	defer server.Close()

	err := runCommitShortcut(t, server, "diff", map[string]string{"sha": "abc123"})
	if err != nil {
		t.Fatalf("diff shortcut failed: %v", err)
	}
}

func TestCommitBlame(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" || r.URL.Path != "/v1/owner/repo/blame.json" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		if r.URL.Query().Get("filepath") != "main.go" {
			t.Fatalf("expected filepath=main.go, got %s", r.URL.Query().Get("filepath"))
		}
		writeJSON(t, w, map[string]interface{}{
			"file_name": "main.go",
			"num_lines": 20,
		})
	}))
	defer server.Close()

	err := runCommitShortcut(t, server, "blame", map[string]string{"path": "main.go", "sha": "master"})
	if err != nil {
		t.Fatalf("blame shortcut failed: %v", err)
	}
}

// === helpers ===

func runCommitShortcut(t *testing.T, server *httptest.Server, name string, args map[string]string) error {
	t.Helper()
	shortcut := findCommitShortcut(t, name)
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

func findCommitShortcut(t *testing.T, name string) *common.Shortcut {
	t.Helper()
	for _, s := range Shortcuts() {
		if s.Name == name {
			return s
		}
	}
	t.Fatalf("shortcut %q not found", name)
	return nil
}

func writeJSON(t *testing.T, w http.ResponseWriter, payload interface{}) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		t.Fatalf("failed to write response: %v", err)
	}
}

func assertEqual(t *testing.T, got, want interface{}) {
	t.Helper()
	if fmt.Sprintf("%v", got) != fmt.Sprintf("%v", want) {
		t.Fatalf("got %v (%T), want %v (%T)", got, got, want, want)
	}
}
