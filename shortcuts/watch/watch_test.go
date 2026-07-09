package watch

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gitlink-org/gitlink-cli/internal/client"
	"github.com/gitlink-org/gitlink-cli/shortcuts/common"
)

func TestWatch(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		switch {
		case r.Method == "GET" && r.URL.Path == "/owner/repo.json":
			writeJSON(t, w, map[string]interface{}{"id": float64(100), "name": "repo"})
		case r.Method == "POST" && r.URL.Path == "/watchers/follow.json":
			if r.URL.Query().Get("target_type") != "project" {
				t.Fatal("expected target_type=project")
			}
			if r.URL.Query().Get("id") != "100" {
				t.Fatalf("expected id=100, got %s", r.URL.Query().Get("id"))
			}
			writeJSON(t, w, map[string]interface{}{"watched": true})
		default:
			t.Fatalf("unexpected request #%d: %s %s", callCount, r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	if err := runWatchShortcut(t, server, "watch", map[string]string{}); err != nil {
		t.Fatalf("watch failed: %v", err)
	}
	if callCount != 2 {
		t.Fatalf("expected 2 calls (GET project + POST watch), got %d", callCount)
	}
}

func TestUnwatch(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		switch {
		case r.Method == "GET" && r.URL.Path == "/owner/repo.json":
			writeJSON(t, w, map[string]interface{}{"id": float64(100)})
		case r.Method == "DELETE" && r.URL.Path == "/watchers/unfollow.json":
			writeJSON(t, w, map[string]interface{}{"watched": false})
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	if err := runWatchShortcut(t, server, "unwatch", map[string]string{}); err != nil {
		t.Fatalf("unwatch failed: %v", err)
	}
}

func TestWatchers(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" || r.URL.Path != "/owner/repo/watchers.json" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		writeJSON(t, w, map[string]interface{}{
			"count": 1,
			"users": []map[string]interface{}{{"login": "alice", "is_watch": true}},
		})
	}))
	defer server.Close()

	err := runWatchShortcut(t, server, "watchers", map[string]string{
		"owner": "owner", "repo": "repo",
	})
	if err != nil {
		t.Fatalf("watchers failed: %v", err)
	}
}

// === helpers ===

func runWatchShortcut(t *testing.T, server *httptest.Server, name string, args map[string]string) error {
	t.Helper()
	shortcut := findWatchShortcut(t, name)
	ctx := &common.RuntimeContext{
		Client: &client.Client{HTTP: server.Client(), BaseURL: server.URL},
		Owner:  "owner", Repo: "repo", Format: "json", Args: args,
	}
	return shortcut.Run(ctx)
}

func findWatchShortcut(t *testing.T, name string) *common.Shortcut {
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
	json.NewEncoder(w).Encode(payload)
}
