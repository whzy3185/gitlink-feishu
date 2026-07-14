package branch

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
	shortcuts := Shortcuts()
	for _, s := range shortcuts {
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

func TestBranchList(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/owner/repo/branches.json" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		writeJSON(w, []interface{}{
			map[string]interface{}{"name": "master"},
			map[string]interface{}{"name": "develop"},
		})
	}))
	defer server.Close()

	err := runShortcut(t, server, "list", map[string]string{"page": "1", "limit": "20"})
	if err != nil {
		t.Fatalf("list failed: %v", err)
	}
}

func TestBranchListFilters(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/owner/repo/branches.json" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if got := r.URL.Query().Get("keyword"); got != "feature" {
			t.Fatalf("keyword = %q, want feature", got)
		}
		if got := r.URL.Query().Get("state"); got != "deleted" {
			t.Fatalf("state = %q, want deleted", got)
		}
		writeJSON(w, map[string]interface{}{"total_count": 0, "branches": []interface{}{}})
	}))
	defer server.Close()

	err := runShortcut(t, server, "list", map[string]string{
		"page": "1", "limit": "20", "keyword": "feature", "state": "deleted",
	})
	if err != nil {
		t.Fatalf("list with filters failed: %v", err)
	}
}

func TestBranchListRejectsInvalidState(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatalf("invalid state should not call API, got %s %s", r.Method, r.URL.Path)
	}))
	defer server.Close()

	err := runShortcut(t, server, "list", map[string]string{"page": "1", "limit": "20", "state": "open"})
	if err == nil {
		t.Fatal("expected validation error")
	}
}

func TestBranchAll(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" || r.URL.Path != "/v1/owner/repo/branches/all.json" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		writeJSON(w, []interface{}{map[string]interface{}{"name": "master"}})
	}))
	defer server.Close()

	err := runShortcut(t, server, "all", nil)
	if err != nil {
		t.Fatalf("all failed: %v", err)
	}
}

// --- create ---

func TestBranchCreate(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Fatalf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/v1/owner/repo/branches.json" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		writeJSON(w, map[string]interface{}{"name": "feature-x"})
	}))
	defer server.Close()

	err := runShortcut(t, server, "create", map[string]string{"name": "feature-x", "from": "master"})
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}
}

func TestBranchCreateDefaultFrom(t *testing.T) {
	// When 'from' is not set, it falls back to the repository default branch.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/owner/repo.json":
			writeJSON(w, map[string]interface{}{"default_branch": "main"})
		case "/v1/owner/repo/branches.json":
			var payload map[string]interface{}
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
				t.Fatalf("decode payload: %v", err)
			}
			if payload["old_branch_name"] != "main" {
				t.Fatalf("expected old_branch_name to be default branch main, got %v", payload["old_branch_name"])
			}
			writeJSON(w, map[string]interface{}{"name": "feature-y"})
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	err := runShortcut(t, server, "create", map[string]string{"name": "feature-y"})
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}
}

// --- delete ---

func TestBranchDelete(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/owner/repo/branches/delete.json" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		writeJSON(w, map[string]interface{}{"message": "deleted"})
	}))
	defer server.Close()

	err := runShortcut(t, server, "delete", map[string]string{"name": "old-branch"})
	if err != nil {
		t.Fatalf("delete failed: %v", err)
	}
}

// --- protect ---

func TestBranchProtect(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/owner/repo/protected_branches.json" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		writeJSON(w, map[string]interface{}{"message": "protected"})
	}))
	defer server.Close()

	err := runShortcut(t, server, "protect", map[string]string{"name": "master"})
	if err != nil {
		t.Fatalf("protect failed: %v", err)
	}
}

// --- unprotect ---

func TestBranchUnprotect(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "DELETE" {
			t.Fatalf("expected DELETE, got %s", r.Method)
		}
		if r.URL.Path != "/owner/repo/protected_branches/master.json" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		writeJSON(w, map[string]interface{}{"message": "unprotected"})
	}))
	defer server.Close()

	err := runShortcut(t, server, "unprotect", map[string]string{"name": "master"})
	if err != nil {
		t.Fatalf("unprotect failed: %v", err)
	}
}

func TestBranchSetDefault(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "PATCH" || r.URL.Path != "/v1/owner/repo/branches/update_default_branch.json" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		if got := r.URL.Query().Get("name"); got != "main" {
			t.Fatalf("name query = %q, want main", got)
		}
		writeJSON(w, map[string]interface{}{"status": 0, "message": "success"})
	}))
	defer server.Close()

	if err := runShortcut(t, server, "set-default", map[string]string{"name": "main"}); err != nil {
		t.Fatalf("set-default failed: %v", err)
	}
}

func TestBranchSetDefaultDryRunDoesNotCallAPI(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatalf("dry-run should not call API, got %s %s", r.Method, r.URL.Path)
	}))
	defer server.Close()

	if err := runShortcut(t, server, "set-default", map[string]string{"name": "main", "dry-run": "true"}); err != nil {
		t.Fatalf("set-default dry-run failed: %v", err)
	}
}

func TestBranchRestore(t *testing.T) {
	var payload map[string]interface{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" || r.URL.Path != "/v1/owner/repo/branches/restore.json" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		writeJSON(w, map[string]interface{}{"status": 0, "message": "success"})
	}))
	defer server.Close()

	if err := runShortcut(t, server, "restore", map[string]string{"id": "7", "name": "feature/deleted"}); err != nil {
		t.Fatalf("restore failed: %v", err)
	}
	if payload["branch_id"] != float64(7) {
		t.Fatalf("branch_id = %v, want 7", payload["branch_id"])
	}
	if payload["branch_name"] != "feature/deleted" {
		t.Fatalf("branch_name = %v, want feature/deleted", payload["branch_name"])
	}
}

func TestBranchRestoreValidation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatalf("invalid restore should not call API, got %s %s", r.Method, r.URL.Path)
	}))
	defer server.Close()

	if err := runShortcut(t, server, "restore", map[string]string{"id": "0", "name": "feature/deleted"}); err == nil {
		t.Fatal("expected invalid id error")
	}
}

// --- HTTP error paths ---

func TestBranchListHTTPError(t *testing.T) {
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

func TestBranchCreateHTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("server error"))
	}))
	defer server.Close()

	err := runShortcut(t, server, "create", map[string]string{"name": "feature-x"})
	if err == nil {
		t.Fatal("expected error for HTTP 500")
	}
}

func TestBranchDeleteHTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("server error"))
	}))
	defer server.Close()

	err := runShortcut(t, server, "delete", map[string]string{"name": "old-branch"})
	if err == nil {
		t.Fatal("expected error for HTTP 500")
	}
}

func TestBranchProtectHTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("server error"))
	}))
	defer server.Close()

	err := runShortcut(t, server, "protect", map[string]string{"name": "master"})
	if err == nil {
		t.Fatal("expected error for HTTP 500")
	}
}

func TestBranchUnprotectHTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("server error"))
	}))
	defer server.Close()

	err := runShortcut(t, server, "unprotect", map[string]string{"name": "master"})
	if err == nil {
		t.Fatal("expected error for HTTP 500")
	}
}

func TestBranchAllUsesAllEndpoint(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" || r.URL.Path != "/v1/owner/repo/branches/all.json" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		writeJSON(w, []interface{}{map[string]interface{}{"name": "master"}})
	}))
	defer server.Close()

	if err := runShortcut(t, server, "all", nil); err != nil {
		t.Fatalf("all failed: %v", err)
	}
}

func TestBranchSetDefaultPatchesName(t *testing.T) {
	var payload map[string]interface{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "PATCH" || r.URL.Path != "/v1/owner/repo/branches/update_default_branch.json" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("decode payload: %v", err)
		}
		writeJSON(w, map[string]interface{}{"status": float64(0), "message": "success"})
	}))
	defer server.Close()

	if err := runShortcut(t, server, "set-default", map[string]string{"name": "develop"}); err != nil {
		t.Fatalf("set-default failed: %v", err)
	}
	if payload["name"] != "develop" {
		t.Fatalf("expected name=develop, got %v", payload["name"])
	}
}
