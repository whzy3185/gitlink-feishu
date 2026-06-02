package wiki

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gitlink-org/gitlink-cli/internal/client"
	"github.com/gitlink-org/gitlink-cli/shortcuts/common"
)

// --- List ---

func TestWikiList(t *testing.T) {
	server := newWikiTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		assertRequest(t, r, "GET", "/owner/repo/wiki/pages.json")
		writeJSON(t, w, map[string]interface{}{
			"total_count": 2,
			"wiki_pages": []interface{}{
				map[string]interface{}{"title": "Home", "slug": "home"},
				map[string]interface{}{"title": "Getting Started", "slug": "getting-started"},
			},
		})
	})
	defer server.Close()

	if err := runWikiShortcut(t, server, "list", nil); err != nil {
		t.Fatalf("list shortcut failed: %v", err)
	}
}

// --- View ---

func TestWikiView(t *testing.T) {
	server := newWikiTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		assertRequest(t, r, "GET", "/owner/repo/wiki/pages/home.json")
		writeJSON(t, w, map[string]interface{}{
			"title":          "Home",
			"slug":           "home",
			"content_base64": base64.StdEncoding.EncodeToString([]byte("Welcome to the wiki")),
		})
	})
	defer server.Close()

	if err := runWikiShortcut(t, server, "view", map[string]string{"slug": "home"}); err != nil {
		t.Fatalf("view shortcut failed: %v", err)
	}
}

// --- Create ---

func TestWikiCreate(t *testing.T) {
	server := newWikiTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		assertRequest(t, r, "POST", "/owner/repo/wiki/pages.json")
		body := decodeJSON(t, r)
		assertEqual(t, body["title"], "TestPage")
		decoded, err := base64.StdEncoding.DecodeString(body["content_base64"].(string))
		if err != nil {
			t.Fatalf("failed to decode content_base64: %v", err)
		}
		assertEqual(t, string(decoded), "Hello World")

		writeJSON(t, w, map[string]interface{}{
			"title": "TestPage",
			"slug":  "TestPage",
		})
	})
	defer server.Close()

	if err := runWikiShortcut(t, server, "create", map[string]string{
		"title": "TestPage",
		"body":  "Hello World",
	}); err != nil {
		t.Fatalf("create shortcut failed: %v", err)
	}
}

// --- Update ---

func TestWikiUpdate(t *testing.T) {
	server := newWikiTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		assertRequest(t, r, "PATCH", "/owner/repo/wiki/pages/home.json")
		body := decodeJSON(t, r)
		assertEqual(t, body["title"], "Updated Title")
		decoded, err := base64.StdEncoding.DecodeString(body["content_base64"].(string))
		if err != nil {
			t.Fatalf("failed to decode content_base64: %v", err)
		}
		assertEqual(t, string(decoded), "Updated content")

		writeJSON(t, w, map[string]interface{}{
			"title": "Updated Title",
			"slug":  "home",
		})
	})
	defer server.Close()

	if err := runWikiShortcut(t, server, "update", map[string]string{
		"slug":  "home",
		"title": "Updated Title",
		"body":  "Updated content",
	}); err != nil {
		t.Fatalf("update shortcut failed: %v", err)
	}
}

// --- Delete ---

func TestWikiDelete(t *testing.T) {
	server := newWikiTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		assertRequest(t, r, "DELETE", "/owner/repo/wiki/pages/old-page.json")
		writeJSON(t, w, map[string]interface{}{
			"message": "deleted",
		})
	})
	defer server.Close()

	if err := runWikiShortcut(t, server, "delete", map[string]string{
		"slug": "old-page",
	}); err != nil {
		t.Fatalf("delete shortcut failed: %v", err)
	}
}

// --- test helpers ---

func runWikiShortcut(t *testing.T, server *httptest.Server, name string, args map[string]string) error {
	t.Helper()
	shortcut := findWikiShortcut(t, name)
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
	if ctx.Args == nil {
		ctx.Args = map[string]string{}
	}
	return shortcut.Run(ctx)
}

func findWikiShortcut(t *testing.T, name string) *common.Shortcut {
	t.Helper()
	for _, shortcut := range Shortcuts() {
		if shortcut.Name == name {
			return shortcut
		}
	}
	t.Fatalf("shortcut %q not found", name)
	return nil
}

func newWikiTestServer(t *testing.T, handler http.HandlerFunc) *httptest.Server {
	t.Helper()
	return httptest.NewServer(handler)
}

func assertRequest(t *testing.T, r *http.Request, method, path string) {
	t.Helper()
	if r.Method != method || r.URL.Path != path {
		t.Fatalf("got request %s %s, want %s %s", r.Method, r.URL.Path, method, path)
	}
}

func writeJSON(t *testing.T, w http.ResponseWriter, payload interface{}) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		t.Fatalf("failed to write response: %v", err)
	}
}

func decodeJSON(t *testing.T, r *http.Request) map[string]interface{} {
	t.Helper()
	var m map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&m); err != nil {
		t.Fatalf("failed to decode request body: %v", err)
	}
	return m
}

func assertEqual(t *testing.T, got, want interface{}) {
	t.Helper()
	if got != want {
		t.Fatalf("got %v, want %v", got, want)
	}
}
