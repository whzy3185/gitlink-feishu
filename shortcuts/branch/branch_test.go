package branch

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gitlink-org/gitlink-cli/internal/client"
	"github.com/gitlink-org/gitlink-cli/internal/i18n"
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
		Tr:     i18n.Default(),
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
		if got := r.URL.Query().Get("page"); got != "1" {
			t.Fatalf("unexpected page: %s", got)
		}
		if got := r.URL.Query().Get("limit"); got != "20" {
			t.Fatalf("unexpected limit: %s", got)
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

func TestBranchListWithStateAndKeyword(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/owner/repo/branches.json" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if got := r.URL.Query().Get("state"); got != "deleted" {
			t.Fatalf("unexpected state: %s", got)
		}
		if got := r.URL.Query().Get("keyword"); got != "release/" {
			t.Fatalf("unexpected keyword: %s", got)
		}
		writeJSON(w, map[string]interface{}{
			"total_count": 1,
			"branches": []interface{}{
				map[string]interface{}{"name": "release/1.0", "branch_id": 7, "is_deleted": true},
			},
		})
	}))
	defer server.Close()

	err := runShortcut(t, server, "list", map[string]string{
		"page":    "2",
		"limit":   "50",
		"state":   "deleted",
		"keyword": "release/",
	})
	if err != nil {
		t.Fatalf("list with filters failed: %v", err)
	}
}

func TestBranchAll(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/owner/repo/branches/all.json" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		writeJSON(w, []interface{}{
			map[string]interface{}{"name": "main"},
			map[string]interface{}{"name": "stable/1.0"},
		})
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
	// When 'from' is not set, it defaults to "master"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/owner/repo/branches.json" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		writeJSON(w, map[string]interface{}{"name": "feature-y"})
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

func TestBranchSetDefault(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "PATCH" {
			t.Fatalf("expected PATCH, got %s", r.Method)
		}
		if r.URL.Path != "/v1/owner/repo/branches/update_default_branch.json" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if got := r.URL.Query().Get("name"); got != "release/1.0" {
			t.Fatalf("unexpected default branch name: %s", got)
		}
		writeJSON(w, map[string]interface{}{"status": 0, "message": "success"})
	}))
	defer server.Close()

	err := runShortcut(t, server, "set-default", map[string]string{"name": "release/1.0"})
	if err != nil {
		t.Fatalf("set-default failed: %v", err)
	}
}

func TestBranchRestore(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Fatalf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/v1/owner/repo/branches/restore.json" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		var payload map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("decode restore payload: %v", err)
		}
		if got := payload["branch_name"]; got != "feature/restore-me" {
			t.Fatalf("unexpected branch_name: %v", got)
		}
		if got := payload["branch_id"]; got != float64(9) {
			t.Fatalf("unexpected branch_id: %v", got)
		}
		writeJSON(w, map[string]interface{}{"status": 0, "message": "success"})
	}))
	defer server.Close()

	err := runShortcut(t, server, "restore", map[string]string{
		"branch-id": "9",
		"name":      "feature/restore-me",
	})
	if err != nil {
		t.Fatalf("restore failed: %v", err)
	}
}

func TestBranchRestoreRejectsInvalidBranchID(t *testing.T) {
	var called bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	err := runShortcut(t, server, "restore", map[string]string{
		"branch-id": "abc",
		"name":      "feature/restore-me",
	})
	if err == nil {
		t.Fatal("expected invalid branch-id to fail")
	}
	if called {
		t.Fatal("restore should fail before making a request")
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

func TestBranchAllHTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("server error"))
	}))
	defer server.Close()

	err := runShortcut(t, server, "all", nil)
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

func TestBranchSetDefaultHTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("server error"))
	}))
	defer server.Close()

	err := runShortcut(t, server, "set-default", map[string]string{"name": "main"})
	if err == nil {
		t.Fatal("expected error for HTTP 500")
	}
}

func TestBranchRestoreHTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("server error"))
	}))
	defer server.Close()

	err := runShortcut(t, server, "restore", map[string]string{
		"branch-id": "3",
		"name":      "feature/restore-me",
	})
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
