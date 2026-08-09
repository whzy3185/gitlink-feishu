package wiki

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gitlink-org/gitlink-cli/internal/client"
	"github.com/gitlink-org/gitlink-cli/shortcuts/common"
)

func TestWikiListAutoResolvesProjectID(t *testing.T) {
	called := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == "GET" && r.URL.Path == "/owner/repo.json":
			writeJSON(t, w, map[string]interface{}{"project_id": float64(1546652)})
		case r.Method == "GET" && r.URL.Path == "/api/wiki/wikiPages.json":
			called = true
			query := r.URL.Query()
			assertEqual(t, query.Get("owner"), "owner")
			assertEqual(t, query.Get("repo"), "repo")
			assertEqual(t, query.Get("projectId"), "1546652")
			writeJSON(t, w, map[string]interface{}{"message": "success", "data": []interface{}{}})
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	err := runWikiShortcut(t, server, "list", map[string]string{})
	if err != nil {
		t.Fatalf("list shortcut failed: %v", err)
	}
	if !called {
		t.Fatal("wiki list endpoint was not called")
	}
}

func TestWikiViewUsesExplicitProjectID(t *testing.T) {
	called := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" || r.URL.Path != "/api/wiki/getWiki.json" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		called = true
		query := r.URL.Query()
		assertEqual(t, query.Get("owner"), "owner")
		assertEqual(t, query.Get("repo"), "repo")
		assertEqual(t, query.Get("projectId"), "42")
		assertEqual(t, query.Get("pageName"), "Home")
		writeJSON(t, w, map[string]interface{}{"message": "success", "data": map[string]interface{}{"title": "Home"}})
	}))
	defer server.Close()

	err := runWikiShortcut(t, server, "view", map[string]string{
		"page":       "Home",
		"project-id": "42",
	})
	if err != nil {
		t.Fatalf("view shortcut failed: %v", err)
	}
	if !called {
		t.Fatal("wiki view endpoint was not called")
	}
}

func TestWikiCreateBuildsPayloadFromContent(t *testing.T) {
	var payload map[string]interface{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == "GET" && r.URL.Path == "/owner/repo.json":
			writeJSON(t, w, map[string]interface{}{"project_id": float64(1546652)})
		case r.Method == "POST" && r.URL.Path == "/api/wiki/createWiki.json":
			payload = decodeJSON(t, r)
			writeJSON(t, w, map[string]interface{}{"message": "201"})
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	err := runWikiShortcut(t, server, "create", map[string]string{
		"page":    "Home",
		"title":   "Welcome",
		"content": "hello wiki",
		"message": "Add Home",
	})
	if err != nil {
		t.Fatalf("create shortcut failed: %v", err)
	}

	assertEqual(t, payload["owner"], "owner")
	assertEqual(t, payload["repo"], "repo")
	assertEqual(t, payload["projectId"], float64(1546652))
	assertEqual(t, payload["pageName"], "Home")
	assertEqual(t, payload["title"], "Welcome")
	assertEqual(t, payload["message"], "Add Home")
	assertEqual(t, payload["content_base64"], base64.StdEncoding.EncodeToString([]byte("hello wiki")))
}

func TestWikiUpdateBuildsPayloadFromFile(t *testing.T) {
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "wiki.md")
	if err := os.WriteFile(filePath, []byte("# Updated\n"), 0o644); err != nil {
		t.Fatalf("write wiki file: %v", err)
	}

	var payload map[string]interface{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "PUT" || r.URL.Path != "/api/wiki/updateWiki.json" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		payload = decodeJSON(t, r)
		writeJSON(t, w, map[string]interface{}{"message": "success"})
	}))
	defer server.Close()

	err := runWikiShortcut(t, server, "update", map[string]string{
		"page":       "Home",
		"file":       filePath,
		"project-id": "42",
	})
	if err != nil {
		t.Fatalf("update shortcut failed: %v", err)
	}

	assertEqual(t, payload["projectId"], float64(42))
	assertEqual(t, payload["pageName"], "Home")
	assertEqual(t, payload["title"], "Home")
	assertEqual(t, payload["content_base64"], base64.StdEncoding.EncodeToString([]byte("# Updated\n")))
}

func TestWikiDeleteSendsBody(t *testing.T) {
	var payload map[string]interface{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "DELETE" || r.URL.Path != "/api/wiki/deleteWiki.json" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		payload = decodeJSON(t, r)
		writeJSON(t, w, map[string]interface{}{"message": "success"})
	}))
	defer server.Close()

	err := runWikiShortcut(t, server, "delete", map[string]string{
		"page":       "Home",
		"project-id": "42",
	})
	if err != nil {
		t.Fatalf("delete shortcut failed: %v", err)
	}

	assertEqual(t, payload["owner"], "owner")
	assertEqual(t, payload["repo"], "repo")
	assertEqual(t, payload["projectId"], float64(42))
	assertEqual(t, payload["pageName"], "Home")
}

func TestWikiOpenAPI404HasActionableError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" || r.URL.Path != "/api/wiki/wikiPages.json" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		writeJSON(t, w, map[string]interface{}{"status": 404, "message": "not found"})
	}))
	defer server.Close()

	err := runWikiShortcut(t, server, "list", map[string]string{
		"project-id": "42",
	})
	if err == nil {
		t.Fatal("expected wiki API 404 to fail")
	}
	if !strings.Contains(err.Error(), "GitLink Wiki API /api/wiki/wikiPages returned 404") {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(err.Error(), "Wiki OpenAPI is available") {
		t.Fatalf("expected actionable wiki API hint, got: %v", err)
	}
}

func TestWikiRejectsInvalidInput(t *testing.T) {
	server := httptest.NewServer(http.NotFoundHandler())
	defer server.Close()

	if err := runWikiShortcut(t, server, "view", map[string]string{
		"page":       "Home",
		"project-id": "abc",
	}); err == nil {
		t.Fatal("expected invalid project-id to fail")
	}
	if err := runWikiShortcut(t, server, "create", map[string]string{
		"page":       "Home",
		"project-id": "42",
	}); err == nil {
		t.Fatal("expected missing content to fail")
	}
	if err := runWikiShortcut(t, server, "create", map[string]string{
		"page":       "Home",
		"project-id": "42",
		"content":    "hello",
		"file":       "wiki.md",
	}); err == nil {
		t.Fatal("expected content/file conflict to fail")
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
	if fmt.Sprintf("%v", got) != fmt.Sprintf("%v", want) {
		t.Fatalf("got %v (%T), want %v (%T)", got, got, want, want)
	}
}
