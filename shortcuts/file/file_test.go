package file

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gitlink-org/gitlink-cli/internal/client"
	"github.com/gitlink-org/gitlink-cli/shortcuts/common"
)

func TestFileList(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" || r.URL.Path != "/owner/repo/files.json" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		writeJSON(t, w, []map[string]interface{}{
			{"name": "README.md", "path": "README.md", "type": "file"},
			{"name": "src", "path": "src", "type": "dir"},
		})
	}))
	defer server.Close()

	if err := runFileShortcut(t, server, "list", map[string]string{}); err != nil {
		t.Fatalf("list failed: %v", err)
	}
}

func TestFileListWithRef(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("ref") != "dev" {
			t.Fatalf("expected ref=dev, got %s", r.URL.Query().Get("ref"))
		}
		writeJSON(t, w, []map[string]interface{}{})
	}))
	defer server.Close()

	if err := runFileShortcut(t, server, "list", map[string]string{"ref": "dev"}); err != nil {
		t.Fatalf("list with ref failed: %v", err)
	}
}

func TestFileTree(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" || r.URL.Path != "/v1/owner/repo/git/trees/master.json" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		writeJSON(t, w, map[string]interface{}{
			"total_count": 1,
			"entries":     []map[string]interface{}{{"name": "main.go", "type": "file"}},
		})
	}))
	defer server.Close()

	if err := runFileShortcut(t, server, "tree", map[string]string{}); err != nil {
		t.Fatalf("tree failed: %v", err)
	}
}

func TestFileTreeRecursive(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("recursive") != "true" {
			t.Fatalf("expected recursive=true")
		}
		writeJSON(t, w, map[string]interface{}{"entries": []map[string]interface{}{}})
	}))
	defer server.Close()

	if err := runFileShortcut(t, server, "tree", map[string]string{"recursive": "true"}); err != nil {
		t.Fatalf("tree recursive failed: %v", err)
	}
}

func TestFileGet(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" || r.URL.Path != "/owner/repo/sub_entries.json" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		if r.URL.Query().Get("filepath") != "README.md" {
			t.Fatalf("expected filepath=README.md, got %s", r.URL.Query().Get("filepath"))
		}
		writeJSON(t, w, map[string]interface{}{"name": "README.md", "type": "file"})
	}))
	defer server.Close()

	err := runFileShortcut(t, server, "get", map[string]string{"path": "README.md"})
	if err != nil {
		t.Fatalf("get failed: %v", err)
	}
}

func TestFileGetRequiresPath(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("no request should be made without --path")
	}))
	defer server.Close()

	err := runFileShortcut(t, server, "get", map[string]string{})
	if err == nil {
		t.Fatal("expected error for missing --path")
	}
}

func TestFileCreate(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" && r.URL.Path == "/owner/repo/create_file.json" {
			var payload map[string]interface{}
			json.NewDecoder(r.Body).Decode(&payload)
			if payload["filepath"] != "test.txt" {
				t.Fatalf("expected filepath=test.txt, got %v", payload["filepath"])
			}
			if payload["message"] != "add test" {
				t.Fatalf("expected message=add test, got %v", payload["message"])
			}
			if payload["branch"] != "master" {
				t.Fatalf("expected branch=master, got %v", payload["branch"])
			}
			if _, ok := payload["content"].(string); !ok || payload["content"] == "" {
				t.Fatal("content should be a non-empty Base64 string")
			}
			writeJSON(t, w, map[string]interface{}{"name": "test.txt", "sha": "abc123"})
			return
		}
		t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
	}))
	defer server.Close()

	err := runFileShortcut(t, server, "create", map[string]string{
		"path": "test.txt", "content": "hello world", "message": "add test",
	})
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}
}

func TestFileDelete(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "DELETE" && r.URL.Path == "/owner/repo/delete_file.json" {
			var payload map[string]interface{}
			json.NewDecoder(r.Body).Decode(&payload)
			if payload["filepath"] != "old.txt" {
				t.Fatalf("expected filepath=old.txt, got %v", payload["filepath"])
			}
			if payload["sha"] != "def456" {
				t.Fatalf("expected sha=def456, got %v", payload["sha"])
			}
			writeJSON(t, w, map[string]interface{}{"status": 0, "message": "success"})
			return
		}
		t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
	}))
	defer server.Close()

	err := runFileShortcut(t, server, "delete", map[string]string{
		"path": "old.txt", "sha": "def456", "message": "remove old",
	})
	if err != nil {
		t.Fatalf("delete failed: %v", err)
	}
}

func TestFileDeleteRequiresPath(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("no request should be made without --path")
	}))
	defer server.Close()

	err := runFileShortcut(t, server, "delete", map[string]string{})
	if err == nil {
		t.Fatal("expected error for missing --path")
	}
}

// === helpers ===

func runFileShortcut(t *testing.T, server *httptest.Server, name string, args map[string]string) error {
	t.Helper()
	shortcut := findFileShortcut(t, name)
	ctx := &common.RuntimeContext{
		Client: &client.Client{HTTP: server.Client(), BaseURL: server.URL},
		Owner: "owner", Repo: "repo", Format: "json", Args: args,
	}
	return shortcut.Run(ctx)
}

func findFileShortcut(t *testing.T, name string) *common.Shortcut {
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
