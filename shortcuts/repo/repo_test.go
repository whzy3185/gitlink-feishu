package repo

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
	s := findShortcut(t, name)
	ctx := &common.RuntimeContext{
		Client: &client.Client{HTTP: server.Client(), BaseURL: server.URL},
		Owner:  "owner",
		Repo:   "repo",
		Format: "json",
		Args:   args,
	}
	return s.Run(ctx)
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

// --- list ---

func TestRepoListDefault(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Fatalf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/projects.json" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		writeJSON(w, map[string]interface{}{
			"total_count": float64(2),
			"data":        []interface{}{map[string]interface{}{"name": "repo1"}},
		})
	}))
	defer server.Close()

	err := runShortcut(t, server, "list", map[string]string{"page": "1", "limit": "20"})
	if err != nil {
		t.Fatalf("list failed: %v", err)
	}
}

func TestRepoListForUser(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/users/alice/projects.json" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		writeJSON(w, map[string]interface{}{"total_count": float64(0), "data": []interface{}{}})
	}))
	defer server.Close()

	err := runShortcut(t, server, "list", map[string]string{"user": "alice", "page": "1", "limit": "20"})
	if err != nil {
		t.Fatalf("list for user failed: %v", err)
	}
}

func TestRepoListWithCategory(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("category") != "mirror" {
			t.Fatalf("expected category=mirror, got %s", r.URL.Query().Get("category"))
		}
		writeJSON(w, map[string]interface{}{"data": []interface{}{}})
	}))
	defer server.Close()

	err := runShortcut(t, server, "list", map[string]string{"category": "mirror", "page": "1", "limit": "20"})
	if err != nil {
		t.Fatalf("list with category failed: %v", err)
	}
}

// --- info ---

func TestRepoInfo(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/owner/repo.json" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		writeJSON(w, map[string]interface{}{
			"name":        "repo",
			"description": "test repo",
		})
	}))
	defer server.Close()

	err := runShortcut(t, server, "info", nil)
	if err != nil {
		t.Fatalf("info failed: %v", err)
	}
}

// --- fork ---

func TestRepoFork(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Fatalf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/owner/repo/forks.json" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		writeJSON(w, map[string]interface{}{"message": "forked"})
	}))
	defer server.Close()

	err := runShortcut(t, server, "fork", nil)
	if err != nil {
		t.Fatalf("fork failed: %v", err)
	}
}

// --- delete ---

func TestRepoDelete(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "DELETE" {
			t.Fatalf("expected DELETE, got %s", r.Method)
		}
		if r.URL.Path != "/owner/repo.json" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		writeJSON(w, map[string]interface{}{"message": "deleted"})
	}))
	defer server.Close()

	err := runShortcut(t, server, "delete", nil)
	if err != nil {
		t.Fatalf("delete failed: %v", err)
	}
}

// --- create ---

func TestRepoCreate(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/users/me.json" && r.Method == "GET":
			writeJSON(w, map[string]interface{}{
				"login":   "creator",
				"user_id": float64(42),
			})
		case r.URL.Path == "/creator/new-repo.json" && r.Method == "POST":
			writeJSON(w, map[string]interface{}{"name": "new-repo"})
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	err := runShortcut(t, server, "create", map[string]string{"name": "new-repo"})
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}
}

func TestRepoCreateWithOptions(t *testing.T) {
	var body map[string]interface{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/users/me.json":
			writeJSON(w, map[string]interface{}{
				"login":   "creator",
				"user_id": float64(42),
			})
		case r.URL.Path == "/creator/my-repo.json":
			json.NewDecoder(r.Body).Decode(&body)
			writeJSON(w, map[string]interface{}{"name": "my-repo"})
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	err := runShortcut(t, server, "create", map[string]string{
		"name":        "my-repo",
		"description": "a test repo",
		"private":     "true",
	})
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}
	if body["description"] != "a test repo" {
		t.Fatalf("description = %v", body["description"])
	}
	if body["private"] != true {
		t.Fatalf("private = %v", body["private"])
	}
}

func TestRepoCreateFailsWithoutName(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("no API call should be made")
	}))
	defer server.Close()

	err := runShortcut(t, server, "create", map[string]string{})
	if err == nil {
		t.Fatal("expected error for missing name")
	}
}

// --- HTTP error paths ---

func TestRepoListHTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("server error"))
	}))
	defer server.Close()

	err := runShortcut(t, server, "list", map[string]string{"page": "1", "limit": "20"})
	if err == nil {
		t.Fatal("expected error for HTTP 500")
	}
}

func TestRepoInfoHTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("server error"))
	}))
	defer server.Close()

	err := runShortcut(t, server, "info", nil)
	if err == nil {
		t.Fatal("expected error for HTTP 500")
	}
}

func TestRepoForkHTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("server error"))
	}))
	defer server.Close()

	err := runShortcut(t, server, "fork", nil)
	if err == nil {
		t.Fatal("expected error for HTTP 500")
	}
}

func TestRepoDeleteHTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("server error"))
	}))
	defer server.Close()

	err := runShortcut(t, server, "delete", nil)
	if err == nil {
		t.Fatal("expected error for HTTP 500")
	}
}

func TestRepoCreateGetUserHTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("server error"))
	}))
	defer server.Close()

	err := runShortcut(t, server, "create", map[string]string{"name": "new-repo"})
	if err == nil {
		t.Fatal("expected error when get user fails")
	}
}

func TestRepoCreateUserNoLogin(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, map[string]interface{}{"user_id": float64(42)})
	}))
	defer server.Close()

	err := runShortcut(t, server, "create", map[string]string{"name": "new-repo"})
	if err == nil {
		t.Fatal("expected error when user response has no login")
	}
}
