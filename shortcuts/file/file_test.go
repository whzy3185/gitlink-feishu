package file

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/gitlink-org/gitlink-cli/internal/client"
	"github.com/gitlink-org/gitlink-cli/shortcuts/common"
)

func TestFileView(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" || r.URL.Path != "/owner/repo/sub_entries.json" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		if got := r.URL.Query().Get("filepath"); got != "README.md" {
			t.Fatalf("filepath query = %q, want README.md", got)
		}
		writeJSON(t, w, map[string]interface{}{
			"entries": map[string]interface{}{
				"name": "README.md", "type": "file", "content": "# hello",
			},
		})
	}))
	defer server.Close()

	if err := runFileShortcut(t, server, "view", map[string]string{"path": "README.md"}); err != nil {
		t.Fatalf("view shortcut failed: %v", err)
	}
}

func TestFileSearch(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" || r.URL.Path != "/owner/repo/files.json" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		if got := r.URL.Query().Get("search"); got != "main" {
			t.Fatalf("search query = %q, want main", got)
		}
		writeJSON(t, w, []interface{}{})
	}))
	defer server.Close()

	if err := runFileShortcut(t, server, "search", map[string]string{"keyword": "main"}); err != nil {
		t.Fatalf("search shortcut failed: %v", err)
	}
}

func TestFileCreate(t *testing.T) {
	var payload map[string]interface{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" || r.URL.Path != "/v1/owner/repo/contents/batch.json" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		json.NewDecoder(r.Body).Decode(&payload)
		writeJSON(t, w, map[string]interface{}{"commit": map[string]interface{}{"sha": "abc"}})
	}))
	defer server.Close()

	err := runFileShortcut(t, server, "create", map[string]string{
		"path": "notes.md", "content": "hello", "branch": "master", "message": "add notes",
	})
	if err != nil {
		t.Fatalf("create shortcut failed: %v", err)
	}
	if payload["branch"] != "master" || payload["message"] != "add notes" {
		t.Fatalf("payload = %v", payload)
	}
	files := payload["files"].([]interface{})
	f := files[0].(map[string]interface{})
	if f["action_type"] != "create" || f["file_path"] != "notes.md" || f["encoding"] != "text" {
		t.Fatalf("file entry = %v", f)
	}
	if f["content"] != "hello" {
		t.Fatalf("content = %v, want hello", f["content"])
	}
}

func TestFileUpdateFromContentFile(t *testing.T) {
	dir := t.TempDir()
	local := filepath.Join(dir, "input.txt")
	os.WriteFile(local, []byte("updated"), 0600)

	var payload map[string]interface{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" || r.URL.Path != "/v1/owner/repo/contents/batch.json" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		json.NewDecoder(r.Body).Decode(&payload)
		writeJSON(t, w, map[string]interface{}{"commit": map[string]interface{}{"sha": "def"}})
	}))
	defer server.Close()

	err := runFileShortcut(t, server, "update", map[string]string{
		"path": "notes.md", "content-file": local, "branch": "master", "new-branch": "feature/x",
	})
	if err != nil {
		t.Fatalf("update shortcut failed: %v", err)
	}
	if payload["new_branch"] != "feature/x" {
		t.Fatalf("new_branch = %v", payload["new_branch"])
	}
	f := payload["files"].([]interface{})[0].(map[string]interface{})
	if f["action_type"] != "update" {
		t.Fatalf("action_type = %v", f["action_type"])
	}
	if f["content"] != "updated" {
		t.Fatalf("content = %v, want updated", f["content"])
	}
}

func TestFileDelete(t *testing.T) {
	var payload map[string]interface{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" || r.URL.Path != "/v1/owner/repo/contents/batch.json" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		json.NewDecoder(r.Body).Decode(&payload)
		writeJSON(t, w, map[string]interface{}{"commit": map[string]interface{}{"sha": "ghi"}})
	}))
	defer server.Close()

	err := runFileShortcut(t, server, "delete", map[string]string{
		"path": "notes.md", "branch": "master",
	})
	if err != nil {
		t.Fatalf("delete shortcut failed: %v", err)
	}
	f := payload["files"].([]interface{})[0].(map[string]interface{})
	if f["action_type"] != "delete" || f["file_path"] != "notes.md" {
		t.Fatalf("file entry = %v", f)
	}
	if payload["message"] != "delete notes.md" {
		t.Fatalf("default message = %v", payload["message"])
	}
}

func TestFileCreateContentConflicts(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("should not reach server")
	}))
	defer server.Close()

	err := runFileShortcut(t, server, "create", map[string]string{
		"path": "a", "content": "x", "content-file": "y", "branch": "master",
	})
	if err == nil {
		t.Fatal("expected error for both --content and --content-file")
	}

	err = runFileShortcut(t, server, "create", map[string]string{
		"path": "a", "branch": "master",
	})
	if err == nil {
		t.Fatal("expected error when no content source is provided")
	}
}

func TestExtractContent(t *testing.T) {
	tests := []struct {
		name         string
		data         interface{}
		wantContent  string
		wantEncoding string
		wantOK       bool
	}{
		{"entries object", map[string]interface{}{"entries": map[string]interface{}{"type": "file", "content": "abc"}}, "abc", "", true},
		{"readme object", map[string]interface{}{"type": "file", "content": "abc", "encoding": "base64"}, "abc", "base64", true},
		{"directory", map[string]interface{}{"type": "dir", "content": "x"}, "", "", false},
		{"no content", map[string]interface{}{"type": "file"}, "", "", false},
		{"not a map", []interface{}{}, "", "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			content, encoding, ok := extractContent(tt.data)
			if content != tt.wantContent || encoding != tt.wantEncoding || ok != tt.wantOK {
				t.Fatalf("extractContent() = (%q, %q, %v), want (%q, %q, %v)",
					content, encoding, ok, tt.wantContent, tt.wantEncoding, tt.wantOK)
			}
		})
	}
}

func runFileShortcut(t *testing.T, server *httptest.Server, name string, args map[string]string) error {
	t.Helper()
	shortcut := findFileShortcut(t, name)
	ctx := &common.RuntimeContext{
		Client: &client.Client{
			HTTP:    server.Client(),
			BaseURL: server.URL,
		},
		Owner:  "owner",
		Repo:   "repo",
		Format: "json",
		Args:   args,
	}
	return shortcut.Run(ctx)
}

func findFileShortcut(t *testing.T, name string) *common.Shortcut {
	t.Helper()
	for _, shortcut := range Shortcuts() {
		if shortcut.Name == name {
			return shortcut
		}
	}
	t.Fatalf("shortcut %q not found", name)
	return nil
}

func writeJSON(t *testing.T, w http.ResponseWriter, payload interface{}) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		t.Fatalf("failed to write response: %v", err)
	}
}

func TestFileBatchPostsAllOperations(t *testing.T) {
	dir := t.TempDir()
	spec := filepath.Join(dir, "spec.json")
	os.WriteFile(spec, []byte(`[
  {"action_type": "create", "file_path": "a.txt", "content": "A"},
  {"action_type": "delete", "file_path": "b.txt"}
]`), 0600)

	var payload map[string]interface{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" || r.URL.Path != "/v1/owner/repo/contents/batch.json" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		json.NewDecoder(r.Body).Decode(&payload)
		writeJSON(t, w, map[string]interface{}{"commit": map[string]interface{}{"sha": "abc"}})
	}))
	defer server.Close()

	err := runFileShortcut(t, server, "batch", map[string]string{
		"spec": spec, "branch": "master", "message": "batch ops",
	})
	if err != nil {
		t.Fatalf("batch shortcut failed: %v", err)
	}
	files := payload["files"].([]interface{})
	if len(files) != 2 {
		t.Fatalf("expected 2 files, got %d", len(files))
	}
	del := files[1].(map[string]interface{})
	if del["action_type"] != "delete" || del["content"] != "" || del["encoding"] != "text" {
		t.Fatalf("delete entry not normalized: %v", del)
	}

	bad := filepath.Join(dir, "bad.json")
	os.WriteFile(bad, []byte(`[{"action_type": "rename", "file_path": "x"}]`), 0600)
	if err := runFileShortcut(t, server, "batch", map[string]string{
		"spec": bad, "branch": "master", "message": "m",
	}); err == nil {
		t.Fatal("expected error for invalid action_type")
	}
}
