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
		switch {
		case r.Method == "GET" && r.URL.Path == "/owner/repo.json":
			writeJSON(t, w, map[string]interface{}{
				"id":         float64(1547460),
				"project_id": float64(1547460),
				"name":       "gitlink-cli",
			})
		case r.Method == "GET" && r.URL.Path == "/wiki/open/wikiPages":
			writeJSON(t, w, map[string]interface{}{
				"data": []interface{}{
					map[string]interface{}{"title": "Home", "sub_url": "Home"},
					map[string]interface{}{"title": "Guide", "sub_url": "Guide"},
				},
			})
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	})
	defer server.Close()

	if err := runWikiShortcut(t, server, "list", nil); err != nil {
		t.Fatalf("list failed: %v", err)
	}
}

// --- View ---

func TestWikiView(t *testing.T) {
	server := newWikiTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == "GET" && r.URL.Path == "/owner/repo.json":
			writeJSON(t, w, map[string]interface{}{
				"id":         float64(1547460),
				"project_id": float64(1547460),
			})
		case r.Method == "GET" && r.URL.Path == "/wiki/open/getWiki":
			if r.URL.Query().Get("pageName") != "Home" {
				t.Fatalf("expected pageName=Home, got %s", r.URL.Query().Get("pageName"))
			}
			writeJSON(t, w, map[string]interface{}{
				"data": map[string]interface{}{
					"title":          "Home",
					"content_base64": base64.StdEncoding.EncodeToString([]byte("Welcome to the wiki")),
				},
			})
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	})
	defer server.Close()

	if err := runWikiShortcut(t, server, "view", map[string]string{"name": "Home"}); err != nil {
		t.Fatalf("view failed: %v", err)
	}
}

// --- Create ---

func TestWikiCreate(t *testing.T) {
	var createPayload map[string]interface{}
	server := newWikiTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == "GET" && r.URL.Path == "/owner/repo.json":
			writeJSON(t, w, map[string]interface{}{
				"id":         float64(1547460),
				"project_id": float64(1547460),
			})
		case r.Method == "POST" && r.URL.Path == "/wiki/open/createWiki":
			createPayload = decodeJSON(t, r)
			writeJSON(t, w, map[string]interface{}{
				"code": 201,
				"data": map[string]interface{}{"title": "NewPage"},
			})
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	})
	defer server.Close()

	if err := runWikiShortcut(t, server, "create", map[string]string{
		"name":    "NewPage",
		"content": "Hello Wiki",
	}); err != nil {
		t.Fatalf("create failed: %v", err)
	}

	assertEqual(t, createPayload["pageName"], "NewPage")
	assertEqual(t, createPayload["owner"], "owner")
	assertEqual(t, createPayload["repo"], "repo")
	expectedContent := base64.StdEncoding.EncodeToString([]byte("Hello Wiki"))
	assertEqual(t, createPayload["content_base64"], expectedContent)
}

// --- Update ---

func TestWikiUpdate(t *testing.T) {
	var updatePayload map[string]interface{}
	server := newWikiTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == "GET" && r.URL.Path == "/owner/repo.json":
			writeJSON(t, w, map[string]interface{}{
				"id":         float64(1547460),
				"project_id": float64(1547460),
			})
		case r.Method == "POST" && r.URL.Path == "/wiki/open/updateWiki":
			updatePayload = decodeJSON(t, r)
			writeJSON(t, w, map[string]interface{}{
				"code": 200,
			})
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	})
	defer server.Close()

	if err := runWikiShortcut(t, server, "update", map[string]string{
		"name":    "Home",
		"content": "Updated content",
		"message": "Update wiki page",
	}); err != nil {
		t.Fatalf("update failed: %v", err)
	}

	assertEqual(t, updatePayload["pageName"], "Home")
	assertEqual(t, updatePayload["message"], "Update wiki page")
}

func TestWikiUpdateWithoutMessage(t *testing.T) {
	var updatePayload map[string]interface{}
	server := newWikiTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == "GET" && r.URL.Path == "/owner/repo.json":
			writeJSON(t, w, map[string]interface{}{
				"id":         float64(1547460),
				"project_id": float64(1547460),
			})
		case r.Method == "POST" && r.URL.Path == "/wiki/open/updateWiki":
			updatePayload = decodeJSON(t, r)
			writeJSON(t, w, map[string]interface{}{
				"code": 200,
			})
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	})
	defer server.Close()

	if err := runWikiShortcut(t, server, "update", map[string]string{
		"name":    "Home",
		"content": "Updated content",
	}); err != nil {
		t.Fatalf("update without message failed: %v", err)
	}

	assertEqual(t, updatePayload["pageName"], "Home")
	// message field should be present as empty string, not omitted
	assertEqual(t, updatePayload["message"], "")
}

// --- Delete ---

func TestWikiDelete(t *testing.T) {
	var deletePayload map[string]interface{}
	server := newWikiTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == "GET" && r.URL.Path == "/owner/repo.json":
			writeJSON(t, w, map[string]interface{}{
				"id":         float64(1547460),
				"project_id": float64(1547460),
			})
		case r.Method == "DELETE" && r.URL.Path == "/wiki/open/deleteWiki":
			deletePayload = decodeJSON(t, r)
			writeJSON(t, w, map[string]interface{}{
				"code": 204,
			})
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	})
	defer server.Close()

	if err := runWikiShortcut(t, server, "delete", map[string]string{
		"name": "OldPage",
	}); err != nil {
		t.Fatalf("delete failed: %v", err)
	}

	assertEqual(t, deletePayload["pageName"], "OldPage")
	assertEqual(t, deletePayload["projectId"], float64(1547460))
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
