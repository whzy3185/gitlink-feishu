package wiki

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/gitlink-org/gitlink-cli/internal/client"
	"github.com/gitlink-org/gitlink-cli/shortcuts/common"
)

func TestWikiList(t *testing.T) {
	server := newWikiTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		assertRequest(t, r, "GET", "/api/wiki/wikiPages.json")
		assertQuery(t, r, "owner", "owner")
		assertQuery(t, r, "repo", "repo")
		assertQuery(t, r, "projectId", "12345")
		writeJSON(t, w, map[string]interface{}{
			"message": "success",
			"data": map[string]interface{}{
				"wiki_pages": []interface{}{
					map[string]interface{}{"title": "Home", "slug": "home"},
					map[string]interface{}{"title": "Getting Started", "slug": "getting-started"},
				},
			},
		})
	})
	defer server.Close()

	if err := runWikiShortcut(t, server, "list", map[string]string{
		"project-id": "12345",
	}); err != nil {
		t.Fatalf("list shortcut failed: %v", err)
	}
}

// --- test helpers ---

func runWikiShortcut(t *testing.T, server *httptest.Server, name string, args map[string]string) error {
	t.Helper()
	shortcut := findWikiShortcut(t, name)
	ctx := &common.RuntimeContext{
		Client: &client.Client{
			HTTP:    server.Client(),
			BaseURL: server.URL + "/api",
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

func assertQuery(t *testing.T, r *http.Request, key, value string) {
	t.Helper()
	got := r.URL.Query().Get(key)
	if got != value {
		t.Fatalf("query param %q: got %q, want %q", key, got, value)
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

var _ = url.Values{} // ensure net/url import is used if needed later
