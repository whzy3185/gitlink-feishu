package org

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

// --- list ---

func TestOrgList(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/organizations.json" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		writeJSON(w, []interface{}{
			map[string]interface{}{"login": "org1"},
			map[string]interface{}{"login": "org2"},
		})
	}))
	defer server.Close()

	err := runShortcut(t, server, "list", map[string]string{"page": "1", "limit": "20"})
	if err != nil {
		t.Fatalf("list failed: %v", err)
	}
}

// --- info ---

func TestOrgInfo(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/organizations/myorg.json" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		writeJSON(w, map[string]interface{}{"login": "myorg", "name": "My Org"})
	}))
	defer server.Close()

	err := runShortcut(t, server, "info", map[string]string{"id": "myorg"})
	if err != nil {
		t.Fatalf("info failed: %v", err)
	}
}

// --- members ---

func TestOrgMembers(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/organizations/myorg/organization_users.json" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		writeJSON(w, []interface{}{
			map[string]interface{}{"login": "user1"},
		})
	}))
	defer server.Close()

	err := runShortcut(t, server, "members", map[string]string{"id": "myorg", "page": "1", "limit": "20"})
	if err != nil {
		t.Fatalf("members failed: %v", err)
	}
}

// --- create ---

func TestOrgCreate(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/organizations.json" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if r.Method != "POST" {
			t.Fatalf("expected POST, got %s", r.Method)
		}
		writeJSON(w, map[string]interface{}{"login": "neworg"})
	}))
	defer server.Close()

	err := runShortcut(t, server, "create", map[string]string{"name": "neworg", "description": "A new org"})
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}
}

func TestOrgCreateNoDescription(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, map[string]interface{}{"login": "neworg"})
	}))
	defer server.Close()

	err := runShortcut(t, server, "create", map[string]string{"name": "neworg"})
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}
}

// --- HTTP error paths ---

func TestOrgListHTTPError(t *testing.T) {
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

func TestOrgInfoHTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("server error"))
	}))
	defer server.Close()

	err := runShortcut(t, server, "info", map[string]string{"id": "myorg"})
	if err == nil {
		t.Fatal("expected error for HTTP 500")
	}
}

func TestOrgMembersHTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("server error"))
	}))
	defer server.Close()

	err := runShortcut(t, server, "members", map[string]string{"id": "myorg", "page": "1", "limit": "20"})
	if err == nil {
		t.Fatal("expected error for HTTP 500")
	}
}

func TestOrgCreateHTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("server error"))
	}))
	defer server.Close()

	err := runShortcut(t, server, "create", map[string]string{"name": "neworg"})
	if err == nil {
		t.Fatal("expected error for HTTP 500")
	}
}
