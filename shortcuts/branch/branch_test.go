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
