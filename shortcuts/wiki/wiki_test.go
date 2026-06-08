package wiki

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gitlink-org/gitlink-cli/internal/client"
	"github.com/gitlink-org/gitlink-cli/shortcuts/common"
)

// --- list ---

func TestWikiList(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		switch {
		case r.Method == "GET" && r.URL.Path == "/owner/repo.json":
			writeJSON(t, w, map[string]interface{}{"id": float64(42)})
		case r.Method == "GET" && r.URL.Path == "/wiki/open/wikiPages":
			if r.URL.Query().Get("projectId") != "42" {
				t.Fatalf("expected projectId=42, got %s", r.URL.Query().Get("projectId"))
			}
			writeJSON(t, w, map[string]interface{}{
				"data": []interface{}{
					map[string]interface{}{"title": "Home", "sub_url": "Home"},
					map[string]interface{}{"title": "Guide", "sub_url": "Guide"},
				},
			})
		default:
			t.Fatalf("unexpected request #%d: %s %s", callCount, r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	if err := runWikiShortcut(t, server, "list", map[string]string{}); err != nil {
		t.Fatalf("list failed: %v", err)
	}
}

func TestWikiListWithProjectID(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		switch {
		case r.Method == "GET" && r.URL.Path == "/owner/repo.json":
			writeJSON(t, w, map[string]interface{}{"project_id": float64(99)})
		case r.Method == "GET" && r.URL.Path == "/wiki/open/wikiPages":
			if r.URL.Query().Get("projectId") != "99" {
				t.Fatalf("expected projectId=99, got %s", r.URL.Query().Get("projectId"))
			}
			writeJSON(t, w, map[string]interface{}{"data": []interface{}{}})
		default:
			t.Fatalf("unexpected request #%d: %s %s", callCount, r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	if err := runWikiShortcut(t, server, "list", map[string]string{}); err != nil {
		t.Fatalf("list with project_id failed: %v", err)
	}
}

// --- view ---

func TestWikiView(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		switch {
		case r.Method == "GET" && r.URL.Path == "/owner/repo.json":
			writeJSON(t, w, map[string]interface{}{"id": float64(42)})
		case r.Method == "GET" && r.URL.Path == "/wiki/open/getWiki":
			if r.URL.Query().Get("pageName") != "Home" {
				t.Fatalf("expected pageName=Home, got %s", r.URL.Query().Get("pageName"))
			}
			writeJSON(t, w, map[string]interface{}{
				"data": map[string]interface{}{
					"title":          "Home",
					"content_base64": base64.StdEncoding.EncodeToString([]byte("Welcome to wiki")),
				},
			})
		default:
			t.Fatalf("unexpected request #%d: %s %s", callCount, r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	err := runWikiShortcut(t, server, "view", map[string]string{"name": "Home"})
	if err != nil {
		t.Fatalf("view failed: %v", err)
	}
}

func TestWikiViewRequiresName(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("no request should be made without --name")
	}))
	defer server.Close()

	err := runWikiShortcut(t, server, "view", map[string]string{})
	if err == nil {
		t.Fatal("expected error for missing --name")
	}
}

// --- create ---

func TestWikiCreate(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		switch {
		case r.Method == "GET" && r.URL.Path == "/owner/repo.json":
			writeJSON(t, w, map[string]interface{}{"id": float64(42)})
		case r.Method == "POST" && r.URL.Path == "/wiki/open/createWiki":
			var payload map[string]interface{}
			json.NewDecoder(r.Body).Decode(&payload)
			if payload["pageName"] != "NewPage" {
				t.Fatalf("expected pageName=NewPage, got %v", payload["pageName"])
			}
			if payload["title"] != "NewPage" {
				t.Fatalf("expected title=NewPage, got %v", payload["title"])
			}
			if payload["owner"] != "owner" {
				t.Fatalf("expected owner=owner, got %v", payload["owner"])
			}
			if payload["repo"] != "repo" {
				t.Fatalf("expected repo=repo, got %v", payload["repo"])
			}
			if payload["projectId"].(float64) != 42 {
				t.Fatalf("expected projectId=42, got %v", payload["projectId"])
			}
			expectedContent := base64.StdEncoding.EncodeToString([]byte("Hello Wiki!"))
			if payload["content_base64"] != expectedContent {
				t.Fatalf("content_base64 mismatch: got %v", payload["content_base64"])
			}
			writeJSON(t, w, map[string]interface{}{
				"code": 201,
				"data": map[string]interface{}{"title": "NewPage"},
			})
		default:
			t.Fatalf("unexpected request #%d: %s %s", callCount, r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	err := runWikiShortcut(t, server, "create", map[string]string{
		"name": "NewPage", "content": "Hello Wiki!", "message": "create page",
	})
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}
}

func TestWikiCreateRequiresName(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("no request should be made without --name")
	}))
	defer server.Close()

	err := runWikiShortcut(t, server, "create", map[string]string{"content": "test"})
	if err == nil {
		t.Fatal("expected error for missing --name")
	}
}

func TestWikiCreateRequiresContent(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("no request should be made without --content")
	}))
	defer server.Close()

	err := runWikiShortcut(t, server, "create", map[string]string{"name": "Test"})
	if err == nil {
		t.Fatal("expected error for missing --content")
	}
}

// --- update ---

func TestWikiUpdate(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		switch {
		case r.Method == "GET" && r.URL.Path == "/owner/repo.json":
			writeJSON(t, w, map[string]interface{}{"id": float64(42)})
		case r.Method == "PUT" && r.URL.Path == "/wiki/open/updateWiki":
			var payload map[string]interface{}
			json.NewDecoder(r.Body).Decode(&payload)
			if payload["pageName"] != "Home" {
				t.Fatalf("expected pageName=Home, got %v", payload["pageName"])
			}
			if payload["title"] != "Home" {
				t.Fatalf("expected title=Home, got %v", payload["title"])
			}
			if payload["message"] != "update page" {
				t.Fatalf("expected message=update page, got %v", payload["message"])
			}
			writeJSON(t, w, map[string]interface{}{"code": 200})
		default:
			t.Fatalf("unexpected request #%d: %s %s", callCount, r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	err := runWikiShortcut(t, server, "update", map[string]string{
		"name": "Home", "content": "Updated content", "message": "update page",
	})
	if err != nil {
		t.Fatalf("update failed: %v", err)
	}
}

func TestWikiUpdateRequiresName(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("no request should be made without --name")
	}))
	defer server.Close()

	err := runWikiShortcut(t, server, "update", map[string]string{"content": "test"})
	if err == nil {
		t.Fatal("expected error for missing --name")
	}
}

// --- delete ---

func TestWikiDelete(t *testing.T) {
	callCount := 0
	sidebarContent := "[[Home]]\n[[OldPage]]\n[[Guide]]"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		switch {
		case r.Method == "GET" && r.URL.Path == "/owner/repo.json":
			writeJSON(t, w, map[string]interface{}{"id": float64(42)})
		case r.Method == "DELETE" && r.URL.Path == "/wiki/open/deleteWiki":
			var payload map[string]interface{}
			json.NewDecoder(r.Body).Decode(&payload)
			if payload["pageName"] != "OldPage" {
				t.Fatalf("expected pageName=OldPage, got %v", payload["pageName"])
			}
			writeJSON(t, w, map[string]interface{}{"code": 200})
		case r.Method == "GET" && r.URL.Path == "/wiki/open/getWiki":
			if r.URL.Query().Get("pageName") != "_Sidebar" {
				t.Fatalf("expected pageName=_Sidebar, got %s", r.URL.Query().Get("pageName"))
			}
			// Return sidebar with the page still in it
			writeJSON(t, w, map[string]interface{}{
				"code": 200,
				"data": fmt.Sprintf(`{"content_base64":"%s"}`, base64.StdEncoding.EncodeToString([]byte(sidebarContent))),
			})
		case r.Method == "PUT" && r.URL.Path == "/wiki/open/updateWiki":
			var payload map[string]interface{}
			json.NewDecoder(r.Body).Decode(&payload)
			if payload["pageName"] != "_Sidebar" {
				t.Fatalf("expected pageName=_Sidebar, got %v", payload["pageName"])
			}
			// Verify OldPage is removed from sidebar
			updated, _ := base64.StdEncoding.DecodeString(payload["content_base64"].(string))
			if strings.Contains(string(updated), "[[OldPage]]") {
				t.Fatal("sidebar should not contain [[OldPage]] after delete")
			}
			if !strings.Contains(string(updated), "[[Home]]") {
				t.Fatal("sidebar should still contain [[Home]]")
			}
			writeJSON(t, w, map[string]interface{}{"code": 200})
		default:
			t.Fatalf("unexpected request #%d: %s %s", callCount, r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	err := runWikiShortcut(t, server, "delete", map[string]string{"name": "OldPage"})
	if err != nil {
		t.Fatalf("delete failed: %v", err)
	}
}

func TestWikiDeleteRequiresName(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("no request should be made without --name")
	}))
	defer server.Close()

	err := runWikiShortcut(t, server, "delete", map[string]string{})
	if err == nil {
		t.Fatal("expected error for missing --name")
	}
}

// --- helpers ---

func runWikiShortcut(t *testing.T, server *httptest.Server, name string, args map[string]string) error {
	t.Helper()
	shortcut := findWikiShortcut(t, name)
	ctx := &common.RuntimeContext{
		Client: &client.Client{HTTP: server.Client(), BaseURL: server.URL},
		Owner:  "owner", Repo: "repo", Format: "json", Args: args,
	}
	return shortcut.Run(ctx)
}

func findWikiShortcut(t *testing.T, name string) *common.Shortcut {
	t.Helper()
	for _, s := range Shortcuts() {
		if s.Name == name {
			return s
		}
	}
	t.Fatalf("shortcut %q not found", name)
	return nil
}

func writeJSON(t *testing.T, w http.ResponseWriter, payload interface{}) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(payload)
}
