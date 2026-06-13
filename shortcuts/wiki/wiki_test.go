package wiki

import (
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
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

// decodeBody reads the request body into a map.
func decodeBody(t *testing.T, r *http.Request) map[string]interface{} {
	t.Helper()
	data, _ := io.ReadAll(r.Body)
	var m map[string]interface{}
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("decode body: %v (raw: %s)", err, string(data))
	}
	return m
}

// --- list / view resolve project id from repo info ---

func TestWikiListResolvesProjectID(t *testing.T) {
	var sawRepo, sawList bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/alice/demo.json":
			sawRepo = true
			writeJSON(w, map[string]interface{}{"id": float64(4242)})
		case "/wiki/open/wikiPages":
			sawList = true
			if got := r.URL.Query().Get("projectId"); got != "4242" {
				t.Fatalf("projectId = %q, want 4242", got)
			}
			if got := r.URL.Query().Get("owner"); got != "alice" {
				t.Fatalf("owner = %q, want alice", got)
			}
			writeJSON(w, map[string]interface{}{"data": map[string]interface{}{}})
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	if err := runShortcut(t, server, "list", map[string]string{}); err != nil {
		t.Fatalf("list failed: %v", err)
	}
	if !sawRepo || !sawList {
		t.Fatalf("expected repo+list calls, got repo=%v list=%v", sawRepo, sawList)
	}
}

func TestWikiViewExplicitProjectID(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/wiki/open/getWiki" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if got := r.URL.Query().Get("pageName"); got != "Home" {
			t.Fatalf("pageName = %q, want Home", got)
		}
		if got := r.URL.Query().Get("projectId"); got != "10" {
			t.Fatalf("projectId = %q, want 10", got)
		}
		writeJSON(w, map[string]interface{}{"data": map[string]interface{}{}})
	}))
	defer server.Close()

	args := map[string]string{"project-id": "10", "page": "Home"}
	if err := runShortcut(t, server, "view", args); err != nil {
		t.Fatalf("view failed: %v", err)
	}
}

func TestWikiViewMissingPage(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, map[string]interface{}{"data": map[string]interface{}{}})
	}))
	defer server.Close()

	if err := runShortcut(t, server, "view", map[string]string{"project-id": "10"}); err == nil {
		t.Fatal("expected error for missing --page")
	}
}

// --- create encodes content as base64 ---

func TestWikiCreateEncodesContent(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/wiki/open/createWiki" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		body := decodeBody(t, r)
		if body["pageName"] != "Home" {
			t.Fatalf("pageName = %v", body["pageName"])
		}
		if body["projectId"] != float64(10) {
			t.Fatalf("projectId = %v, want 10", body["projectId"])
		}
		want := base64.StdEncoding.EncodeToString([]byte("hello"))
		if body["content_base64"] != want {
			t.Fatalf("content_base64 = %v, want %v", body["content_base64"], want)
		}
		writeJSON(w, map[string]interface{}{"message": "201"})
	}))
	defer server.Close()

	args := map[string]string{"project-id": "10", "page": "Home", "content": "hello"}
	if err := runShortcut(t, server, "create", args); err != nil {
		t.Fatalf("create failed: %v", err)
	}
}

func TestWikiCreateRequiresContent(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("no API call expected when content is missing")
	}))
	defer server.Close()

	args := map[string]string{"project-id": "10", "page": "Home"}
	if err := runShortcut(t, server, "create", args); err == nil {
		t.Fatal("expected error for missing content")
	}
}

func TestWikiCreateContentFile(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "page.md")
	if err := os.WriteFile(file, []byte("# Title"), 0o600); err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body := decodeBody(t, r)
		want := base64.StdEncoding.EncodeToString([]byte("# Title"))
		if body["content_base64"] != want {
			t.Fatalf("content_base64 = %v, want %v", body["content_base64"], want)
		}
		writeJSON(w, map[string]interface{}{"message": "201"})
	}))
	defer server.Close()

	args := map[string]string{"project-id": "10", "page": "Home", "content-file": file}
	if err := runShortcut(t, server, "create", args); err != nil {
		t.Fatalf("create from file failed: %v", err)
	}
}

// --- update allows omitting content ---

func TestWikiUpdateWithoutContent(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/wiki/open/updateWiki" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		body := decodeBody(t, r)
		if _, ok := body["content_base64"]; ok {
			t.Fatalf("content_base64 should be absent when not provided")
		}
		if body["title"] != "Renamed" {
			t.Fatalf("title = %v, want Renamed", body["title"])
		}
		writeJSON(w, map[string]interface{}{"message": "ok"})
	}))
	defer server.Close()

	args := map[string]string{"project-id": "10", "page": "Home", "title": "Renamed"}
	if err := runShortcut(t, server, "update", args); err != nil {
		t.Fatalf("update failed: %v", err)
	}
}

// --- delete dry-run does not call the API ---

func TestWikiDeleteDryRun(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("no API call expected in dry-run")
	}))
	defer server.Close()

	args := map[string]string{"project-id": "10", "page": "Home", "dry-run": "true"}
	if err := runShortcut(t, server, "delete", args); err != nil {
		t.Fatalf("delete dry-run failed: %v", err)
	}
}

func TestWikiDelete(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/wiki/open/deleteWiki" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if r.Method != http.MethodDelete {
			t.Fatalf("method = %s, want DELETE", r.Method)
		}
		body := decodeBody(t, r)
		if body["pageName"] != "Home" {
			t.Fatalf("pageName = %v", body["pageName"])
		}
		writeJSON(w, map[string]interface{}{"message": "ok"})
	}))
	defer server.Close()

	args := map[string]string{"project-id": "10", "page": "Home"}
	if err := runShortcut(t, server, "delete", args); err != nil {
		t.Fatalf("delete failed: %v", err)
	}
}

// --- export ---

func TestWikiExportDefaults(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/wikiExport/wikiExport-wrapper.json" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		q := r.URL.Query()
		if q.Get("type") != "markdown" {
			t.Fatalf("type = %q, want markdown", q.Get("type"))
		}
		if q.Get("repoName") != "demo" {
			t.Fatalf("repoName = %q, want demo", q.Get("repoName"))
		}
		if q.Get("projectName") != "demo" {
			t.Fatalf("projectName = %q, want demo (default to repo)", q.Get("projectName"))
		}
		writeJSON(w, map[string]interface{}{"data": map[string]interface{}{}})
	}))
	defer server.Close()

	if err := runShortcut(t, server, "export", map[string]string{"project-id": "10"}); err != nil {
		t.Fatalf("export failed: %v", err)
	}
}

func TestWikiExportInvalidType(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("no API call expected for invalid type")
	}))
	defer server.Close()

	args := map[string]string{"project-id": "10", "type": "docx"}
	err := runShortcut(t, server, "export", args)
	if err == nil || !strings.Contains(err.Error(), "type") {
		t.Fatalf("expected invalid type error, got %v", err)
	}
}

// --- invalid project id ---

func TestWikiInvalidProjectID(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("no API call expected for invalid project id")
	}))
	defer server.Close()

	args := map[string]string{"project-id": "abc"}
	if err := runShortcut(t, server, "list", args); err == nil {
		t.Fatal("expected error for invalid --project-id")
	}
}
