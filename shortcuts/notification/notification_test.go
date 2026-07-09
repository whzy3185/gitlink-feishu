package notification

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gitlink-org/gitlink-cli/internal/client"
	"github.com/gitlink-org/gitlink-cli/shortcuts/common"
)

func runNotifShortcut(t *testing.T, server *httptest.Server, name string, args map[string]string) error {
	t.Helper()
	s := findNotifShortcut(t, name)
	ctx := &common.RuntimeContext{
		Client: &client.Client{HTTP: server.Client(), BaseURL: server.URL},
		Owner:  "owner",
		Repo:   "repo",
		Format: "json",
		Args:   args,
	}
	return s.Run(ctx)
}

func findNotifShortcut(t *testing.T, name string) *common.Shortcut {
	t.Helper()
	for _, s := range Shortcuts() {
		if s.Name == name {
			return s
		}
	}
	t.Fatalf("shortcut %q not found", name)
	return nil
}

func writeNotifJSON(w http.ResponseWriter, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v)
}

// --- list ---

func TestNotifListBasic(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Fatalf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/notifications.json" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if got := r.URL.Query().Get("page"); got != "1" {
			t.Fatalf("got page %q, want %q", got, "1")
		}
		if got := r.URL.Query().Get("limit"); got != "20" {
			t.Fatalf("got limit %q, want %q", got, "20")
		}
		writeNotifJSON(w, map[string]interface{}{
			"total_count": float64(1),
			"notifications": []interface{}{
				map[string]interface{}{"id": float64(1), "unread": true},
			},
		})
	}))
	defer server.Close()

	err := runNotifShortcut(t, server, "list", map[string]string{
		"page": "1", "limit": "20", "all": "false", "participating": "false",
	})
	if err != nil {
		t.Fatalf("list failed: %v", err)
	}
}

func TestNotifListWithAll(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("all"); got != "true" {
			t.Fatalf("expected all=true, got %q", got)
		}
		writeNotifJSON(w, map[string]interface{}{"total_count": float64(0), "notifications": []interface{}{}})
	}))
	defer server.Close()

	err := runNotifShortcut(t, server, "list", map[string]string{
		"page": "1", "limit": "20", "all": "true", "participating": "false",
	})
	if err != nil {
		t.Fatalf("list with all failed: %v", err)
	}
}

func TestNotifListWithParticipating(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("participating"); got != "true" {
			t.Fatalf("expected participating=true, got %q", got)
		}
		writeNotifJSON(w, map[string]interface{}{"total_count": float64(0), "notifications": []interface{}{}})
	}))
	defer server.Close()

	err := runNotifShortcut(t, server, "list", map[string]string{
		"page": "1", "limit": "20", "all": "false", "participating": "true",
	})
	if err != nil {
		t.Fatalf("list with participating failed: %v", err)
	}
}

// --- read ---

func TestNotifRead(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "PUT" {
			t.Fatalf("expected PUT, got %s", r.Method)
		}
		if r.URL.Path != "/notifications/42.json" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		writeNotifJSON(w, map[string]interface{}{"status": 0, "message": "success"})
	}))
	defer server.Close()

	err := runNotifShortcut(t, server, "read", map[string]string{"id": "42"})
	if err != nil {
		t.Fatalf("read failed: %v", err)
	}
}

// --- read-all ---

func TestNotifReadAll(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "PUT" {
			t.Fatalf("expected PUT, got %s", r.Method)
		}
		if r.URL.Path != "/notifications.json" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		writeNotifJSON(w, map[string]interface{}{"status": 0, "message": "success"})
	}))
	defer server.Close()

	err := runNotifShortcut(t, server, "read-all", map[string]string{})
	if err != nil {
		t.Fatalf("read-all failed: %v", err)
	}
}

// --- watch ---

func TestNotifWatch(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Fatalf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/watchers/alice/repo.json" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		writeNotifJSON(w, map[string]interface{}{"status": 0, "message": "success"})
	}))
	defer server.Close()

	err := runNotifShortcut(t, server, "watch", map[string]string{
		"owner": "alice", "repo": "repo", "unwatch": "false",
	})
	if err != nil {
		t.Fatalf("watch failed: %v", err)
	}
}

func TestNotifUnwatch(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "DELETE" {
			t.Fatalf("expected DELETE, got %s", r.Method)
		}
		if r.URL.Path != "/watchers/bob/project.json" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		writeNotifJSON(w, map[string]interface{}{"status": 0, "message": "success"})
	}))
	defer server.Close()

	err := runNotifShortcut(t, server, "watch", map[string]string{
		"owner": "bob", "repo": "project", "unwatch": "true",
	})
	if err != nil {
		t.Fatalf("unwatch failed: %v", err)
	}
}
