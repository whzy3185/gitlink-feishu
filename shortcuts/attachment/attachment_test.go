package attachment

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/gitlink-org/gitlink-cli/internal/client"
	"github.com/gitlink-org/gitlink-cli/shortcuts/common"
)

func TestAttachmentUploadSendsMultipartForm(t *testing.T) {
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "sample.txt")
	if err := os.WriteFile(filePath, []byte("hello attachment"), 0o644); err != nil {
		t.Fatalf("write temp file: %v", err)
	}

	called := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" || r.URL.Path != "/attachments.json" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		called = true
		if err := r.ParseMultipartForm(1 << 20); err != nil {
			t.Fatalf("parse multipart: %v", err)
		}
		if got := r.FormValue("description"); got != "release asset" {
			t.Fatalf("description = %q, want %q", got, "release asset")
		}
		if got := r.FormValue("container_id"); got != "42" {
			t.Fatalf("container_id = %q, want %q", got, "42")
		}
		if got := r.FormValue("container_type"); got != "VersionRelease" {
			t.Fatalf("container_type = %q, want %q", got, "VersionRelease")
		}
		file, header, err := r.FormFile("file")
		if err != nil {
			t.Fatalf("read form file: %v", err)
		}
		defer file.Close()
		if header.Filename != "sample.txt" {
			t.Fatalf("filename = %q, want %q", header.Filename, "sample.txt")
		}
		content, err := io.ReadAll(file)
		if err != nil {
			t.Fatalf("read uploaded file: %v", err)
		}
		if string(content) != "hello attachment" {
			t.Fatalf("content = %q, want %q", string(content), "hello attachment")
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"att-1","title":"sample.txt","url":"https://example.com/a/att-1"}`))
	}))
	defer server.Close()

	err := runAttachmentShortcut(t, server, "upload", map[string]string{
		"file":           filePath,
		"description":    "release asset",
		"container-id":   "42",
		"container-type": "VersionRelease",
	})
	if err != nil {
		t.Fatalf("upload shortcut failed: %v", err)
	}
	if !called {
		t.Fatal("upload endpoint was not called")
	}
}

func TestAttachmentDeleteCallsAPI(t *testing.T) {
	called := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "DELETE" || r.URL.Path != "/attachments/att-1.json" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		called = true
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":0,"message":"deleted"}`))
	}))
	defer server.Close()

	err := runAttachmentShortcut(t, server, "delete", map[string]string{
		"id": "att-1",
	})
	if err != nil {
		t.Fatalf("delete shortcut failed: %v", err)
	}
	if !called {
		t.Fatal("delete endpoint was not called")
	}
}

func runAttachmentShortcut(t *testing.T, server *httptest.Server, name string, args map[string]string) error {
	t.Helper()
	shortcut := findAttachmentShortcut(t, name)
	ctx := &common.RuntimeContext{
		Client: &client.Client{
			HTTP:    server.Client(),
			BaseURL: server.URL,
		},
		Format: "json",
		Args:   args,
	}
	return shortcut.Run(ctx)
}

func findAttachmentShortcut(t *testing.T, name string) *common.Shortcut {
	t.Helper()
	for _, shortcut := range Shortcuts() {
		if shortcut.Name == name {
			return shortcut
		}
	}
	t.Fatalf("shortcut %q not found", name)
	return nil
}
