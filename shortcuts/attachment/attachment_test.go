package attachment

import (
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gitlink-org/gitlink-cli/internal/client"
	"github.com/gitlink-org/gitlink-cli/shortcuts/common"
)

func TestAttachmentUploadDryRunDoesNotCallAPI(t *testing.T) {
	server := newAttachmentTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		t.Fatalf("dry-run should not call API, got: %s %s", r.Method, r.URL.Path)
	})
	defer server.Close()

	err := runAttachmentShortcut(t, server, "upload", map[string]string{
		"file":           filepath.Join(t.TempDir(), "missing.txt"),
		"description":    "design screenshot",
		"container-id":   "123",
		"container-type": "Issue",
		"dry-run":        "true",
	})
	if err != nil {
		t.Fatalf("upload dry-run failed: %v", err)
	}
}

func TestAttachmentUploadMultipartPayload(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "note.txt")
	if err := os.WriteFile(filePath, []byte("hello attachment"), 0600); err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}

	server := newAttachmentTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		assertAttachmentRequest(t, r, "POST", "/attachments.json")
		if got := r.Header.Get("Content-Type"); !strings.HasPrefix(got, "multipart/form-data;") {
			t.Fatalf("got content-type %q, want multipart/form-data", got)
		}
		if err := r.ParseMultipartForm(1 << 20); err != nil {
			t.Fatalf("failed to parse multipart form: %v", err)
		}
		assertFormValue(t, r.MultipartForm, "description", "design screenshot")
		assertFormValue(t, r.MultipartForm, "container_id", "123")
		assertFormValue(t, r.MultipartForm, "container_type", "Issue")
		file, header, err := r.FormFile("file")
		if err != nil {
			t.Fatalf("file field missing: %v", err)
		}
		defer file.Close()
		if header.Filename != "note.txt" {
			t.Fatalf("got filename %q, want note.txt", header.Filename)
		}
		data, err := io.ReadAll(file)
		if err != nil {
			t.Fatalf("failed to read uploaded file: %v", err)
		}
		if string(data) != "hello attachment" {
			t.Fatalf("got file content %q", string(data))
		}
		writeAttachmentJSON(t, w, map[string]interface{}{
			"id":           "uuid-1",
			"title":        "note.txt",
			"filesize":     "16 Bytes",
			"is_pdf":       false,
			"url":          "/api/attachments/uuid-1",
			"content_type": "text/plain",
		})
	})
	defer server.Close()

	err := runAttachmentShortcut(t, server, "upload", map[string]string{
		"file":           filePath,
		"description":    "design screenshot",
		"container-id":   "123",
		"container-type": "Issue",
	})
	if err != nil {
		t.Fatalf("upload shortcut failed: %v", err)
	}
}

func TestAttachmentUploadMissingFile(t *testing.T) {
	server := newAttachmentTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		t.Fatalf("missing file should not call API, got: %s %s", r.Method, r.URL.Path)
	})
	defer server.Close()

	err := runAttachmentShortcut(t, server, "upload", map[string]string{"file": filepath.Join(t.TempDir(), "missing.txt")})
	if err == nil {
		t.Fatal("expected missing file to return an error")
	}
	if !strings.Contains(err.Error(), "open attachment file") {
		t.Fatalf("got error %q, want open attachment file", err.Error())
	}
}

func TestAttachmentDelete(t *testing.T) {
	server := newAttachmentTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		assertAttachmentRequest(t, r, "DELETE", "/attachments/uuid-1.json")
		writeAttachmentJSON(t, w, map[string]interface{}{"status": 0, "message": "删除成功"})
	})
	defer server.Close()

	if err := runAttachmentShortcut(t, server, "delete", map[string]string{"uuid": "uuid-1"}); err != nil {
		t.Fatalf("delete shortcut failed: %v", err)
	}
}

func TestAttachmentDeleteDryRunDoesNotCallAPI(t *testing.T) {
	server := newAttachmentTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		t.Fatalf("dry-run should not call API, got: %s %s", r.Method, r.URL.Path)
	})
	defer server.Close()

	if err := runAttachmentShortcut(t, server, "delete", map[string]string{"uuid": "uuid-1", "dry-run": "true"}); err != nil {
		t.Fatalf("delete dry-run failed: %v", err)
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
	if ctx.Args == nil {
		ctx.Args = map[string]string{}
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

func newAttachmentTestServer(t *testing.T, handler http.HandlerFunc) *httptest.Server {
	t.Helper()
	return httptest.NewServer(handler)
}

func assertAttachmentRequest(t *testing.T, r *http.Request, method, path string) {
	t.Helper()
	if r.Method != method || r.URL.Path != path {
		t.Fatalf("got request %s %s, want %s %s", r.Method, r.URL.Path, method, path)
	}
}

func assertFormValue(t *testing.T, form *multipart.Form, key, want string) {
	t.Helper()
	values := form.Value[key]
	if len(values) != 1 || values[0] != want {
		t.Fatalf("got form field %s=%v, want %q", key, values, want)
	}
}

func writeAttachmentJSON(t *testing.T, w http.ResponseWriter, payload interface{}) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		t.Fatalf("failed to write response: %v", err)
	}
}
