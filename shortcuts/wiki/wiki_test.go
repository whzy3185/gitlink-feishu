package wiki

import (
	"encoding/base64"
	"net/http"
	"testing"

	"github.com/gitlink-org/gitlink-cli/shortcuts/common"
)

func TestWikiList(t *testing.T) {
	server := common.NewTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == "GET" && r.URL.Path == "/owner/repo.json":
			common.WriteJSON(t, w, map[string]interface{}{
				"id":   float64(123),
				"name": "repo",
			})
		case r.Method == "GET" && r.URL.Path == "/wiki/open/wikiPages":
			common.WriteJSON(t, w, map[string]interface{}{
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

	ctx := common.NewTestContext(t, server, "owner", "repo", map[string]string{})
	err := common.RunShortcut(t, Shortcuts(), "list", ctx)
	if err != nil {
		t.Fatalf("list failed: %v", err)
	}
}

func TestWikiView(t *testing.T) {
	server := common.NewTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == "GET" && r.URL.Path == "/owner/repo.json":
			common.WriteJSON(t, w, map[string]interface{}{
				"id": float64(123),
			})
		case r.Method == "GET" && r.URL.Path == "/wiki/open/getWiki":
			pageName := r.URL.Query().Get("pageName")
			if pageName != "Home" {
				t.Fatalf("expected pageName=Home, got %s", pageName)
			}
			common.WriteJSON(t, w, map[string]interface{}{
				"data": map[string]interface{}{
					"title":          "Home",
					"content_base64": base64.StdEncoding.EncodeToString([]byte("Welcome")),
				},
			})
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	})
	defer server.Close()

	ctx := common.NewTestContext(t, server, "owner", "repo", map[string]string{
		"name": "Home",
	})
	err := common.RunShortcut(t, Shortcuts(), "view", ctx)
	if err != nil {
		t.Fatalf("view failed: %v", err)
	}
}

func TestWikiCreate(t *testing.T) {
	var createPayload map[string]interface{}
	server := common.NewTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == "GET" && r.URL.Path == "/owner/repo.json":
			common.WriteJSON(t, w, map[string]interface{}{
				"id": float64(123),
			})
		case r.Method == "POST" && r.URL.Path == "/wiki/open/createWiki":
			createPayload = common.DecodeJSON(t, r)
			common.WriteJSON(t, w, map[string]interface{}{
				"code": 201,
				"data": map[string]interface{}{"title": "NewPage"},
			})
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	})
	defer server.Close()

	ctx := common.NewTestContext(t, server, "owner", "repo", map[string]string{
		"name":    "NewPage",
		"content": "Hello Wiki",
	})
	err := common.RunShortcut(t, Shortcuts(), "create", ctx)
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}

	common.AssertEqual(t, createPayload["pageName"], "NewPage")
	common.AssertEqual(t, createPayload["owner"], "owner")
	common.AssertEqual(t, createPayload["repo"], "repo")

	expectedContent := base64.StdEncoding.EncodeToString([]byte("Hello Wiki"))
	common.AssertEqual(t, createPayload["content_base64"], expectedContent)
}

func TestWikiUpdate(t *testing.T) {
	var updatePayload map[string]interface{}
	server := common.NewTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == "GET" && r.URL.Path == "/owner/repo.json":
			common.WriteJSON(t, w, map[string]interface{}{
				"id": float64(123),
			})
		case r.Method == "PUT" && r.URL.Path == "/wiki/open/updateWiki":
			updatePayload = common.DecodeJSON(t, r)
			common.WriteJSON(t, w, map[string]interface{}{
				"code": 200,
			})
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	})
	defer server.Close()

	ctx := common.NewTestContext(t, server, "owner", "repo", map[string]string{
		"name":    "Home",
		"content": "Updated content",
		"message": "Update wiki page",
	})
	err := common.RunShortcut(t, Shortcuts(), "update", ctx)
	if err != nil {
		t.Fatalf("update failed: %v", err)
	}

	common.AssertEqual(t, updatePayload["pageName"], "Home")
	common.AssertEqual(t, updatePayload["message"], "Update wiki page")
}

func TestWikiDelete(t *testing.T) {
	var deletePayload, sidebarUpdatePayload map[string]interface{}
	server := common.NewTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == "GET" && r.URL.Path == "/owner/repo.json":
			common.WriteJSON(t, w, map[string]interface{}{
				"id": float64(123),
			})
		case r.Method == "DELETE" && r.URL.Path == "/wiki/open/deleteWiki":
			deletePayload = common.DecodeJSON(t, r)
			common.WriteJSON(t, w, map[string]interface{}{
				"code": 204,
			})
		case r.Method == "GET" && r.URL.Path == "/wiki/open/getWiki":
			pageName := r.URL.Query().Get("pageName")
			if pageName != "_Sidebar" {
				t.Fatalf("expected pageName=_Sidebar, got %s", pageName)
			}
			common.WriteJSON(t, w, map[string]interface{}{
				"code": 200,
				"data": map[string]interface{}{
					"content_base64": base64.StdEncoding.EncodeToString([]byte("[[OldPage]]\n[[OtherPage]]")),
				},
			})
		case r.Method == "PUT" && r.URL.Path == "/wiki/open/updateWiki":
			sidebarUpdatePayload = common.DecodeJSON(t, r)
			common.WriteJSON(t, w, map[string]interface{}{
				"code": 200,
			})
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	})
	defer server.Close()

	ctx := common.NewTestContext(t, server, "owner", "repo", map[string]string{
		"name": "OldPage",
	})
	err := common.RunShortcut(t, Shortcuts(), "delete", ctx)
	if err != nil {
		t.Fatalf("delete failed: %v", err)
	}

	common.AssertEqual(t, deletePayload["pageName"], "OldPage")
	common.AssertEqual(t, deletePayload["projectId"], float64(123))

	// Verify sidebar was updated to remove the deleted page link
	common.AssertEqual(t, sidebarUpdatePayload["pageName"], "_Sidebar")
	expectedSidebar := base64.StdEncoding.EncodeToString([]byte("[[OtherPage]]"))
	common.AssertEqual(t, sidebarUpdatePayload["content_base64"], expectedSidebar)
}
