package attachment

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/gitlink-org/gitlink-cli/internal/client"
	"github.com/gitlink-org/gitlink-cli/internal/i18n"
	"github.com/gitlink-org/gitlink-cli/shortcuts/common"
)

func TestShortcutsRegistered(t *testing.T) {
	shortcuts := Shortcuts()
	if len(shortcuts) != 2 {
		t.Fatalf("expected 2 shortcuts, got %d", len(shortcuts))
	}
	names := map[string]bool{}
	for _, s := range shortcuts {
		names[s.Name] = true
		if s.Description == "" {
			t.Fatalf("shortcut %q has empty description", s.Name)
		}
	}
	for _, want := range []string{"upload", "download"} {
		if !names[want] {
			t.Fatalf("missing shortcut %q", want)
		}
	}
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

func newTestContext(t *testing.T, handler http.HandlerFunc, args map[string]string) *common.RuntimeContext {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	return &common.RuntimeContext{
		Client: &client.Client{HTTP: server.Client(), BaseURL: server.URL},
		Format: "json",
		Args:   args,
	}
}

func TestUploadMultipart(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "asset.txt")
	if err := os.WriteFile(src, []byte("hello attachment"), 0644); err != nil {
		t.Fatal(err)
	}

	var gotFilename, gotDescription string
	ctx := newTestContext(t, func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseMultipartForm(1 << 20); err != nil {
			t.Fatalf("parse multipart: %v", err)
		}
		file, header, err := r.FormFile("file")
		if err != nil {
			t.Fatalf("form file: %v", err)
		}
		defer file.Close()
		data, _ := io.ReadAll(file)
		if string(data) != "hello attachment" {
			t.Fatalf("unexpected file content: %q", data)
		}
		gotFilename = header.Filename
		gotDescription = r.FormValue("description")
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"id": 123, "filename": header.Filename})
	}, map[string]string{"file": src, "description": "test asset"})
	ctx.Tr = i18n.Default()

	if err := findShortcut(t, "upload").Run(ctx); err != nil {
		t.Fatalf("upload error: %v", err)
	}
	if gotFilename != "asset.txt" {
		t.Fatalf("filename = %q", gotFilename)
	}
	if gotDescription != "test asset" {
		t.Fatalf("description = %q", gotDescription)
	}
}

func TestUploadMissingFile(t *testing.T) {
	ctx := newTestContext(t, func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("server should not be reached")
	}, map[string]string{"file": "/nonexistent/path/file.bin"})
	ctx.Tr = i18n.Default()

	if err := findShortcut(t, "upload").Run(ctx); err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestUploadRejectsDirectory(t *testing.T) {
	dir := t.TempDir()
	ctx := newTestContext(t, func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("server should not be reached")
	}, map[string]string{"file": dir})
	ctx.Tr = i18n.Default()

	if err := findShortcut(t, "upload").Run(ctx); err == nil {
		t.Fatal("expected error for directory")
	}
}

func TestDownloadWritesFile(t *testing.T) {
	dir := t.TempDir()
	dest := filepath.Join(dir, "out.bin")
	ctx := newTestContext(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/attachments/42" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		w.Write([]byte("binary-content"))
	}, map[string]string{"id": "42", "output": dest})
	ctx.Tr = i18n.Default()

	if err := findShortcut(t, "download").Run(ctx); err != nil {
		t.Fatalf("download error: %v", err)
	}
	data, err := os.ReadFile(dest)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "binary-content" {
		t.Fatalf("unexpected content: %q", data)
	}
}

func TestDownloadJSONErrorBody(t *testing.T) {
	dir := t.TempDir()
	dest := filepath.Join(dir, "out.bin")
	ctx := newTestContext(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.Write([]byte(`{"status":404,"message":"不存在或已被删除"}`))
	}, map[string]string{"id": "deleted", "output": dest})
	ctx.Tr = i18n.Default()

	if err := findShortcut(t, "download").Run(ctx); err == nil {
		t.Fatal("expected error for JSON error body")
	}
	if _, err := os.Stat(dest); err == nil {
		t.Fatal("output file should not be created on JSON error body")
	}
}

func TestDownloadHTMLFallback(t *testing.T) {
	dir := t.TempDir()
	dest := filepath.Join(dir, "out.bin")
	ctx := newTestContext(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write([]byte("<!doctype html><html></html>"))
	}, map[string]string{"id": "unknown", "output": dest})
	ctx.Tr = i18n.Default()

	if err := findShortcut(t, "download").Run(ctx); err == nil {
		t.Fatal("expected error for HTML fallback page")
	}
	if _, err := os.Stat(dest); err == nil {
		t.Fatal("output file should not be created on HTML fallback")
	}
}

func TestDownloadHTTPError(t *testing.T) {
	dir := t.TempDir()
	dest := filepath.Join(dir, "out.bin")
	ctx := newTestContext(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("not found"))
	}, map[string]string{"id": "999", "output": dest})
	ctx.Tr = i18n.Default()

	if err := findShortcut(t, "download").Run(ctx); err == nil {
		t.Fatal("expected error for HTTP 404")
	}
	if _, err := os.Stat(dest); err == nil {
		t.Fatal("output file should not be created on HTTP error")
	}
}
