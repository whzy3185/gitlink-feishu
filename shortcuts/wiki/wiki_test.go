package wiki

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gitlink-org/gitlink-cli/internal/client"
	"github.com/gitlink-org/gitlink-cli/shortcuts/common"
)

func TestWikiList(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertRequest(t, r, "GET", "/wiki/open/wikiPages")
		assertEqual(t, r.URL.Query().Get("owner"), "owner")
		assertEqual(t, r.URL.Query().Get("repo"), "repo")
		assertEqual(t, r.URL.Query().Get("projectId"), "12345")
		writeJSON(t, w, map[string]interface{}{"status": 0, "data": []interface{}{}})
	}))
	defer server.Close()

	err := runWikiShortcut(t, server, "list", map[string]string{
		"project-id": "12345",
	})
	if err != nil {
		t.Fatalf("list shortcut failed: %v", err)
	}
}

func TestWikiView(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertRequest(t, r, "GET", "/wiki/open/getWiki")
		assertEqual(t, r.URL.Query().Get("owner"), "owner")
		assertEqual(t, r.URL.Query().Get("repo"), "repo")
		assertEqual(t, r.URL.Query().Get("projectId"), "12345")
		assertEqual(t, r.URL.Query().Get("pageName"), "home")
		writeJSON(t, w, map[string]interface{}{"status": 0, "data": map[string]interface{}{"title": "home"}})
	}))
	defer server.Close()

	err := runWikiShortcut(t, server, "view", map[string]string{
		"project-id": "12345",
		"page-name":  "home",
	})
	if err != nil {
		t.Fatalf("view shortcut failed: %v", err)
	}
}

func TestWikiCreate(t *testing.T) {
	var payload map[string]interface{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertRequest(t, r, "POST", "/wiki/open/createWiki")
		payload = decodeJSON(t, r)
		writeJSON(t, w, map[string]interface{}{"status": 0, "message": "success"})
	}))
	defer server.Close()

	err := runWikiShortcut(t, server, "create", map[string]string{
		"project-id": "12345",
		"page-name":  "new-page",
		"title":      "New Page",
		"content":    "# Hello",
	})
	if err != nil {
		t.Fatalf("create shortcut failed: %v", err)
	}

	assertEqual(t, payload["owner"], "owner")
	assertEqual(t, payload["repo"], "repo")
	assertEqual(t, payload["pageName"], "new-page")
	assertEqual(t, payload["title"], "New Page")
	if _, ok := payload["content_base64"]; !ok {
		t.Fatal("body missing content_base64")
	}
}

func TestWikiUpdate(t *testing.T) {
	var payload map[string]interface{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertRequest(t, r, "PUT", "/wiki/open/updateWiki")
		payload = decodeJSON(t, r)
		writeJSON(t, w, map[string]interface{}{"status": 0, "message": "success"})
	}))
	defer server.Close()

	err := runWikiShortcut(t, server, "update", map[string]string{
		"project-id": "12345",
		"page-name":  "home",
		"title":      "Updated Title",
	})
	if err != nil {
		t.Fatalf("update shortcut failed: %v", err)
	}

	assertEqual(t, payload["owner"], "owner")
	assertEqual(t, payload["pageName"], "home")
	assertEqual(t, payload["title"], "Updated Title")
}

func TestWikiUpdateRequiresTitle(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatalf("server should not be called when title is missing: %s %s", r.Method, r.URL.Path)
	}))
	defer server.Close()

	err := runWikiShortcut(t, server, "update", map[string]string{
		"project-id": "12345",
		"page-name":  "home",
	})
	if err == nil {
		t.Fatal("expected update without --title to return an error")
	}
	if err.Error() != "--title is required" {
		t.Fatalf("unexpected error message: %s", err.Error())
	}
}

func TestWikiUpdateWithContentOnly(t *testing.T) {
	var payload map[string]interface{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertRequest(t, r, "PUT", "/wiki/open/updateWiki")
		payload = decodeJSON(t, r)
		writeJSON(t, w, map[string]interface{}{"status": 0, "message": "success"})
	}))
	defer server.Close()

	err := runWikiShortcut(t, server, "update", map[string]string{
		"project-id": "12345",
		"page-name":  "home",
		"title":      "Existing Title",
		"content":    "# Updated content",
	})
	if err != nil {
		t.Fatalf("update with content failed: %v", err)
	}
	if _, ok := payload["content_base64"]; !ok {
		t.Fatal("body missing content_base64 when --content provided")
	}
}

func TestWikiDelete(t *testing.T) {
	var payload map[string]interface{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertRequest(t, r, "DELETE", "/wiki/open/deleteWiki")
		payload = decodeJSON(t, r)
		writeJSON(t, w, map[string]interface{}{"status": 0, "message": "success"})
	}))
	defer server.Close()

	err := runWikiShortcut(t, server, "delete", map[string]string{
		"project-id": "12345",
		"page-name":  "old-page",
	})
	if err != nil {
		t.Fatalf("delete shortcut failed: %v", err)
	}

	assertEqual(t, payload["owner"], "owner")
	assertEqual(t, payload["repo"], "repo")
	assertEqual(t, payload["pageName"], "old-page")
}

func TestWikiListAutoResolvesProjectID(t *testing.T) {
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		switch requests {
		case 1:
			assertRequest(t, r, "GET", "/owner/repo.json")
			writeJSON(t, w, map[string]interface{}{"project_id": float64(1549132)})
		case 2:
			assertRequest(t, r, "GET", "/wiki/open/wikiPages")
			assertEqual(t, r.URL.Query().Get("projectId"), "1549132")
			writeJSON(t, w, map[string]interface{}{"status": 0, "data": []interface{}{}})
		default:
			t.Fatalf("unexpected extra request: %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	err := runWikiShortcut(t, server, "list", nil)
	if err != nil {
		t.Fatalf("list with auto-resolved project id failed: %v", err)
	}
	if requests != 2 {
		t.Fatalf("expected 2 requests, got %d", requests)
	}
}

func TestWikiCreateAutoResolvesProjectID(t *testing.T) {
	requests := 0
	var payload map[string]interface{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		switch requests {
		case 1:
			assertRequest(t, r, "GET", "/owner/repo.json")
			writeJSON(t, w, map[string]interface{}{"project_id": float64(789)})
		case 2:
			assertRequest(t, r, "POST", "/wiki/open/createWiki")
			payload = decodeJSON(t, r)
			writeJSON(t, w, map[string]interface{}{"status": 0, "message": "success"})
		default:
			t.Fatalf("unexpected extra request: %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	err := runWikiShortcut(t, server, "create", map[string]string{
		"page-name": "new-page",
		"title":     "New Page",
		"content":   "# Hello",
	})
	if err != nil {
		t.Fatalf("create with auto-resolved project id failed: %v", err)
	}
	assertEqual(t, payload["projectId"], "789")
}

func TestWikiInvalidProjectID(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatalf("server should not be called for invalid --project-id: %s %s", r.Method, r.URL.Path)
	}))
	defer server.Close()

	err := runWikiShortcut(t, server, "list", map[string]string{
		"project-id": "abc",
	})
	if err == nil {
		t.Fatal("expected invalid --project-id to return an error")
	}
}

func TestWikiResolveProjectIDMissingField(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertRequest(t, r, "GET", "/owner/repo.json")
		writeJSON(t, w, map[string]interface{}{"identifier": "repo"})
	}))
	defer server.Close()

	err := runWikiShortcut(t, server, "list", nil)
	if err == nil {
		t.Fatal("expected missing project_id in repository response to return an error")
	}
}

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

func assertRequest(t *testing.T, r *http.Request, method, path string) {
	t.Helper()
	if r.Method != method || r.URL.Path != path {
		t.Fatalf("got request %s %s, want %s %s", r.Method, r.URL.Path, method, path)
	}
}

func decodeJSON(t *testing.T, r *http.Request) map[string]interface{} {
	t.Helper()
	var payload map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		t.Fatalf("failed to decode request body: %v", err)
	}
	return payload
}

func writeJSON(t *testing.T, w http.ResponseWriter, payload interface{}) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		t.Fatalf("failed to write response: %v", err)
	}
}

func assertEqual(t *testing.T, got interface{}, want interface{}) {
	t.Helper()
	if got != want {
		t.Fatalf("got %v (%T), want %v (%T)", got, got, want, want)
	}
}
