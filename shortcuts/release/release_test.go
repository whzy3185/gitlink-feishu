package release

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

func TestReleaseList(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/owner/repo/releases.json" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		writeJSON(w, []interface{}{
			map[string]interface{}{"tag_name": "v1.0"},
			map[string]interface{}{"tag_name": "v1.1"},
		})
	}))
	defer server.Close()

	err := runShortcut(t, server, "list", map[string]string{"page": "1", "limit": "20"})
	if err != nil {
		t.Fatalf("list failed: %v", err)
	}
}

// --- create ---

func TestReleaseCreate(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/owner/repo/releases.json" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		writeJSON(w, map[string]interface{}{"tag_name": "v2.0"})
	}))
	defer server.Close()

	err := runShortcut(t, server, "create", map[string]string{
		"tag":  "v2.0",
		"name": "Version 2.0",
	})
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}
}

func TestReleaseCreateWithBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, map[string]interface{}{"tag_name": "v2.1"})
	}))
	defer server.Close()

	err := runShortcut(t, server, "create", map[string]string{
		"tag":        "v2.1",
		"name":       "Version 2.1",
		"body":       "Release notes here",
		"target":     "develop",
		"prerelease": "true",
	})
	if err != nil {
		t.Fatalf("create with body failed: %v", err)
	}
}

// --- view ---

func TestReleaseView(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/owner/repo/releases/v1.0.json" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		writeJSON(w, map[string]interface{}{"tag_name": "v1.0", "name": "Version 1.0"})
	}))
	defer server.Close()

	err := runShortcut(t, server, "view", map[string]string{"id": "v1.0"})
	if err != nil {
		t.Fatalf("view failed: %v", err)
	}
}

// --- delete (normal path: delete succeeds) ---

func TestReleaseDeleteSuccess(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "DELETE" {
			writeJSON(w, map[string]interface{}{"message": "deleted"})
			return
		}
		t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
	}))
	defer server.Close()

	err := runShortcut(t, server, "delete", map[string]string{"id": "1"})
	if err != nil {
		t.Fatalf("delete failed: %v", err)
	}
}

// --- delete (API bug workaround: delete fails but release was actually deleted) ---

func TestReleaseDeleteBugWorkaround(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "DELETE":
			// API bug: delete returns error even when successful
			w.WriteHeader(http.StatusInternalServerError)
			writeJSON(w, map[string]interface{}{"status": float64(500), "message": "server error"})
		case "GET":
			// Verify shows the release no longer exists
			w.WriteHeader(http.StatusNotFound)
			writeJSON(w, map[string]interface{}{"status": float64(404), "message": "not found"})
		default:
			t.Fatalf("unexpected method: %s", r.Method)
		}
	}))
	defer server.Close()

	err := runShortcut(t, server, "delete", map[string]string{"id": "1"})
	if err != nil {
		t.Fatalf("delete bug workaround failed: %v", err)
	}
}

// --- delete (delete truly fails: release still exists) ---

func TestReleaseDeleteTrulyFails(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "DELETE":
			w.WriteHeader(http.StatusInternalServerError)
			writeJSON(w, map[string]interface{}{"status": float64(500), "message": "server error"})
		case "GET":
			// Release still exists — delete truly failed
			writeJSON(w, map[string]interface{}{"id": float64(1), "tag_name": "v1.0"})
		default:
			t.Fatalf("unexpected method: %s", r.Method)
		}
	}))
	defer server.Close()

	err := runShortcut(t, server, "delete", map[string]string{"id": "1"})
	if err == nil {
		t.Fatal("expected error when delete truly fails")
	}
}

// --- HTTP error paths ---

func TestReleaseListHTTPError(t *testing.T) {
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

func TestReleaseCreateHTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("server error"))
	}))
	defer server.Close()

	err := runShortcut(t, server, "create", map[string]string{"tag": "v1.0", "name": "v1.0"})
	if err == nil {
		t.Fatal("expected error for HTTP 500")
	}
}

func TestReleaseViewHTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("server error"))
	}))
	defer server.Close()

	err := runShortcut(t, server, "view", map[string]string{"id": "v1.0"})
	if err == nil {
		t.Fatal("expected error for HTTP 500")
	}
}
