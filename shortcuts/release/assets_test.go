package release

import (
	"io"
	"net/http"
	"os"
	"path/filepath"
	"testing"
)

func TestReleaseAssets(t *testing.T) {
	server := newReleaseTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		assertReleaseRequest(t, r, "GET", "/owner/repo/releases/v1.0.json")
		writeReleaseJSON(t, w, releaseViewFixture("7"))
	})
	defer server.Close()

	if err := runReleaseShortcut(t, server, "assets", map[string]string{"id": "v1.0"}); err != nil {
		t.Fatalf("assets shortcut failed: %v", err)
	}
}

func TestReleaseAttachDryRunDoesNotWrite(t *testing.T) {
	server := newReleaseTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/owner/repo/releases/v1.0.json":
			writeReleaseJSON(t, w, releaseViewFixture("7"))
		case r.Method == http.MethodGet && r.URL.Path == "/owner/repo/releases/7/edit.json":
			writeReleaseJSON(t, w, releaseEditFixture())
		default:
			t.Fatalf("dry-run should not write, got %s %s", r.Method, r.URL.Path)
		}
	})
	defer server.Close()

	err := runReleaseShortcut(t, server, "attach", map[string]string{
		"id":             "v1.0",
		"attachment-ids": "34,56",
		"dry-run":        "true",
	})
	if err != nil {
		t.Fatalf("attach dry-run failed: %v", err)
	}
}

func TestReleaseAttachWritesMergedAttachmentIDs(t *testing.T) {
	var payload map[string]interface{}
	server := newReleaseTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/owner/repo/releases/v1.0.json":
			writeReleaseJSON(t, w, releaseViewFixture("7"))
		case r.Method == http.MethodGet && r.URL.Path == "/owner/repo/releases/7/edit.json":
			writeReleaseJSON(t, w, releaseEditFixture())
		case r.Method == http.MethodPut && r.URL.Path == "/owner/repo/releases/7.json":
			payload = decodeReleaseJSON(t, r)
			writeReleaseJSON(t, w, map[string]interface{}{"status": 0, "message": "updated"})
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	})
	defer server.Close()

	err := runReleaseShortcut(t, server, "attach", map[string]string{
		"id":             "v1.0",
		"attachment-ids": "34,56",
	})
	if err != nil {
		t.Fatalf("attach shortcut failed: %v", err)
	}

	assertReleaseEqual(t, payload["name"], "Old release")
	assertReleaseEqual(t, payload["tag_name"], "v1.0.0")
	assertReleaseStringSlice(t, payload["attachment_ids"], []string{"12", "34", "56"})
}

func TestReleaseDetachWritesRemainingAttachmentIDs(t *testing.T) {
	var payload map[string]interface{}
	server := newReleaseTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/owner/repo/releases/v1.0.json":
			writeReleaseJSON(t, w, releaseViewFixture("7"))
		case r.Method == http.MethodGet && r.URL.Path == "/owner/repo/releases/7/edit.json":
			writeReleaseJSON(t, w, releaseEditFixture())
		case r.Method == http.MethodPut && r.URL.Path == "/owner/repo/releases/7.json":
			payload = decodeReleaseJSON(t, r)
			writeReleaseJSON(t, w, map[string]interface{}{"status": 0, "message": "updated"})
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	})
	defer server.Close()

	err := runReleaseShortcut(t, server, "detach", map[string]string{
		"id":             "v1.0",
		"attachment-ids": "12",
	})
	if err != nil {
		t.Fatalf("detach shortcut failed: %v", err)
	}

	assertReleaseStringSlice(t, payload["attachment_ids"], []string{"34"})
}

func TestReleaseUploadUploadsAndAttachesAsset(t *testing.T) {
	var payload map[string]interface{}
	path := filepath.Join(t.TempDir(), "artifact.zip")
	if err := os.WriteFile(path, []byte("artifact-bytes"), 0600); err != nil {
		t.Fatalf("write test asset: %v", err)
	}

	server := newReleaseTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/owner/repo/releases/v1.0.json":
			writeReleaseJSON(t, w, releaseViewFixture("7"))
		case r.Method == http.MethodGet && r.URL.Path == "/owner/repo/releases/7/edit.json":
			writeReleaseJSON(t, w, releaseEditFixture())
		case r.Method == http.MethodPost && r.URL.Path == "/attachments.json":
			if err := r.ParseMultipartForm(1 << 20); err != nil {
				t.Fatalf("ParseMultipartForm: %v", err)
			}
			if got := r.FormValue("description"); got != "CLI binary" {
				t.Fatalf("description = %q, want %q", got, "CLI binary")
			}
			files := r.MultipartForm.File["file"]
			if len(files) != 1 {
				t.Fatalf("expected 1 uploaded file, got %d", len(files))
			}
			if files[0].Filename != "gitlink-cli.zip" {
				t.Fatalf("filename = %q, want %q", files[0].Filename, "gitlink-cli.zip")
			}
			file, err := files[0].Open()
			if err != nil {
				t.Fatalf("open multipart file: %v", err)
			}
			defer file.Close()
			data, err := io.ReadAll(file)
			if err != nil {
				t.Fatalf("read multipart file: %v", err)
			}
			if string(data) != "artifact-bytes" {
				t.Fatalf("uploaded body = %q, want %q", string(data), "artifact-bytes")
			}
			writeReleaseJSON(t, w, map[string]interface{}{"id": "asset-99", "title": "gitlink-cli.zip"})
		case r.Method == http.MethodPut && r.URL.Path == "/owner/repo/releases/7.json":
			payload = decodeReleaseJSON(t, r)
			writeReleaseJSON(t, w, map[string]interface{}{"status": 0, "message": "updated"})
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	})
	defer server.Close()

	err := runReleaseShortcut(t, server, "upload", map[string]string{
		"id":          "v1.0",
		"file":        path,
		"asset-name":  "gitlink-cli.zip",
		"description": "CLI binary",
	})
	if err != nil {
		t.Fatalf("upload shortcut failed: %v", err)
	}

	assertReleaseStringSlice(t, payload["attachment_ids"], []string{"12", "34", "asset-99"})
}

func TestReleaseUploadCleansUpOnAttachFailure(t *testing.T) {
	path := filepath.Join(t.TempDir(), "artifact.zip")
	if err := os.WriteFile(path, []byte("artifact-bytes"), 0600); err != nil {
		t.Fatalf("write test asset: %v", err)
	}

	cleanupCalled := false
	server := newReleaseTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/owner/repo/releases/v1.0.json":
			writeReleaseJSON(t, w, releaseViewFixture("7"))
		case r.Method == http.MethodGet && r.URL.Path == "/owner/repo/releases/7/edit.json":
			writeReleaseJSON(t, w, releaseEditFixture())
		case r.Method == http.MethodPost && r.URL.Path == "/attachments.json":
			writeReleaseJSON(t, w, map[string]interface{}{"id": "asset-99", "title": "artifact.zip"})
		case r.Method == http.MethodPut && r.URL.Path == "/owner/repo/releases/7.json":
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte("update failed"))
		case r.Method == http.MethodDelete && r.URL.Path == "/attachments/asset-99.json":
			cleanupCalled = true
			writeReleaseJSON(t, w, map[string]interface{}{"status": 0, "message": "deleted"})
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	})
	defer server.Close()

	err := runReleaseShortcut(t, server, "upload", map[string]string{
		"id":   "v1.0",
		"file": path,
	})
	if err == nil {
		t.Fatal("expected upload shortcut to fail when release update fails")
	}
	if !cleanupCalled {
		t.Fatal("expected uploaded attachment cleanup to run after release update failure")
	}
}

func releaseViewFixture(versionID string) map[string]interface{} {
	return map[string]interface{}{
		"version_id": versionID,
		"id":         "release-gid",
		"tag_name":   "v1.0.0",
		"name":       "Old release",
		"attachments": []map[string]interface{}{
			{"id": 12, "title": "a.zip"},
			{"id": "34", "title": "b.zip"},
		},
	}
}
