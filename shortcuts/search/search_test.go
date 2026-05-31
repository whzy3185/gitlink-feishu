package search

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gitlink-org/gitlink-cli/internal/client"
	"github.com/gitlink-org/gitlink-cli/shortcuts/common"
)

func runShortcut(t *testing.T, server *httptest.Server, name string, args map[string]string) error {
	t.Helper()
	shortcut := findShortcut(t, name)
	ctx := &common.RuntimeContext{
		Client: &client.Client{HTTP: server.Client(), BaseURL: server.URL},
		Owner:  "owner",
		Repo:   "repo",
		Format: "json",
		Args:   args,
	}
	return shortcut.Run(ctx)
}

func findShortcut(t *testing.T, name string) *common.Shortcut {
	t.Helper()
	for _, s := range Shortcuts() {
		if s.Name == name {
			return s
		}
	}
	t.Fatalf("shortcut %q not found", name)
	return nil
}

func writeJSON(w http.ResponseWriter, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v)
}

// --- repos ---

func TestSearchRepos(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/projects.json" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if r.URL.Query().Get("search") != "golang" {
			t.Fatalf("expected search=golang, got %s", r.URL.Query().Get("search"))
		}
		writeJSON(w, []interface{}{
			map[string]interface{}{"name": "golang-project"},
		})
	}))
	defer server.Close()

	err := runShortcut(t, server, "repos", map[string]string{"keyword": "golang", "page": "1", "limit": "20"})
	if err != nil {
		t.Fatalf("repos failed: %v", err)
	}
}

// --- users ---

func TestSearchUsers(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/users/list.json" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if r.URL.Query().Get("search") != "alice" {
			t.Fatalf("expected search=alice, got %s", r.URL.Query().Get("search"))
		}
		writeJSON(w, []interface{}{
			map[string]interface{}{"login": "alice"},
		})
	}))
	defer server.Close()

	err := runShortcut(t, server, "users", map[string]string{"keyword": "alice", "page": "1", "limit": "20"})
	if err != nil {
		t.Fatalf("users failed: %v", err)
	}
}

// --- HTTP error paths ---

func TestSearchReposHTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("server error"))
	}))
	defer server.Close()

	err := runShortcut(t, server, "repos", map[string]string{"keyword": "test", "page": "1", "limit": "20"})
	if err == nil {
		t.Fatal("expected error for HTTP 500")
	}
}

func TestSearchUsersHTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("server error"))
	}))
	defer server.Close()

	err := runShortcut(t, server, "users", map[string]string{"keyword": "test", "page": "1", "limit": "20"})
	if err == nil {
		t.Fatal("expected error for HTTP 500")
	}
}
