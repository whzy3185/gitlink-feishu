package dataset

import (
	"encoding/json"
	"io"
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
		Owner:  "alice",
		Repo:   "demo",
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
	_ = json.NewEncoder(w).Encode(v)
}

func decodeBody(t *testing.T, r *http.Request) map[string]interface{} {
	t.Helper()
	data, _ := io.ReadAll(r.Body)
	var m map[string]interface{}
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("decode body: %v (raw: %s)", err, string(data))
	}
	return m
}

// --- view ---

func TestDatasetView(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/alice/demo/dataset.json" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if got := r.URL.Query().Get("page"); got != "2" {
			t.Fatalf("page = %q, want 2", got)
		}
		if got := r.URL.Query().Get("limit"); got != "5" {
			t.Fatalf("limit = %q, want 5", got)
		}
		writeJSON(w, map[string]interface{}{"id": float64(1), "attachments": []interface{}{}})
	}))
	defer server.Close()

	if err := runShortcut(t, server, "view", map[string]string{"page": "2", "limit": "5"}); err != nil {
		t.Fatalf("view failed: %v", err)
	}
}

// --- list ---

func TestDatasetListNormalizesIDs(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/project_datasets.json" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if got := r.URL.Query().Get("ids"); got != "1,2,3" {
			t.Fatalf("ids = %q, want 1,2,3", got)
		}
		writeJSON(w, map[string]interface{}{"total_count": float64(0), "project_datasets": []interface{}{}})
	}))
	defer server.Close()

	if err := runShortcut(t, server, "list", map[string]string{"ids": " 1, 2 ,3 "}); err != nil {
		t.Fatalf("list failed: %v", err)
	}
}

func TestDatasetListInvalidIDs(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("no API call expected for invalid ids")
	}))
	defer server.Close()

	if err := runShortcut(t, server, "list", map[string]string{"ids": "abc"}); err == nil {
		t.Fatal("expected error for non-numeric --ids")
	}
}

// --- create ---

func TestDatasetCreate(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/alice/demo/dataset.json" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if r.Method != http.MethodPost {
			t.Fatalf("method = %s, want POST", r.Method)
		}
		body := decodeBody(t, r)
		if body["title"] != "DS" || body["description"] != "desc" {
			t.Fatalf("unexpected body: %v", body)
		}
		if body["license_id"] != float64(359) {
			t.Fatalf("license_id = %v, want 359", body["license_id"])
		}
		writeJSON(w, map[string]interface{}{"status": float64(0), "message": "success"})
	}))
	defer server.Close()

	args := map[string]string{"title": "DS", "description": "desc", "license-id": "359", "paper-content": "x"}
	if err := runShortcut(t, server, "create", args); err != nil {
		t.Fatalf("create failed: %v", err)
	}
}

func TestDatasetCreateMissingTitle(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("no API call expected without required flags")
	}))
	defer server.Close()

	if err := runShortcut(t, server, "create", map[string]string{"description": "desc"}); err == nil {
		t.Fatal("expected error for missing --title")
	}
}

func TestDatasetCreateInvalidLicense(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("no API call expected for invalid license id")
	}))
	defer server.Close()

	args := map[string]string{"title": "DS", "description": "desc", "license-id": "abc"}
	if err := runShortcut(t, server, "create", args); err == nil {
		t.Fatal("expected error for invalid --license-id")
	}
}

func TestDatasetCreateDryRun(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("no API call expected in dry-run")
	}))
	defer server.Close()

	args := map[string]string{"title": "DS", "description": "desc", "dry-run": "true"}
	if err := runShortcut(t, server, "create", args); err != nil {
		t.Fatalf("create dry-run failed: %v", err)
	}
}

// --- update ---

func TestDatasetUpdate(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/alice/demo/dataset.json" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if r.Method != http.MethodPut {
			t.Fatalf("method = %s, want PUT", r.Method)
		}
		writeJSON(w, map[string]interface{}{"status": float64(0), "message": "success"})
	}))
	defer server.Close()

	args := map[string]string{"title": "DS2", "description": "desc2"}
	if err := runShortcut(t, server, "update", args); err != nil {
		t.Fatalf("update failed: %v", err)
	}
}

// --- delete-attachment ---

func TestDatasetDeleteAttachmentDryRun(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("no API call expected in dry-run")
	}))
	defer server.Close()

	args := map[string]string{"uuid": "abc-123", "dry-run": "true"}
	if err := runShortcut(t, server, "delete-attachment", args); err != nil {
		t.Fatalf("delete dry-run failed: %v", err)
	}
}

func TestDatasetDeleteAttachmentRequiresConfirm(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("no API call expected without --yes")
	}))
	defer server.Close()

	if err := runShortcut(t, server, "delete-attachment", map[string]string{"uuid": "abc-123"}); err == nil {
		t.Fatal("expected error without --yes confirmation")
	}
}

func TestDatasetDeleteAttachmentConfirmed(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/attachments/abc-123.json" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if r.Method != http.MethodDelete {
			t.Fatalf("method = %s, want DELETE", r.Method)
		}
		writeJSON(w, map[string]interface{}{"status": float64(0), "message": "删除成功"})
	}))
	defer server.Close()

	args := map[string]string{"uuid": "abc-123", "yes": "true"}
	if err := runShortcut(t, server, "delete-attachment", args); err != nil {
		t.Fatalf("delete failed: %v", err)
	}
}
