package webhook

import (
	"net/http"
	"testing"

	"github.com/gitlink-org/gitlink-cli/shortcuts/common"
)

func TestWebhookList(t *testing.T) {
	server := common.NewTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" && r.URL.Path == "/v1/owner/repo/webhooks.json" {
			common.WriteJSON(t, w, map[string]interface{}{
				"webhooks": []interface{}{
					map[string]interface{}{
						"id":     float64(1),
						"url":    "https://example.com/hook",
						"active": true,
					},
				},
			})
		} else {
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

func TestWebhookCreate(t *testing.T) {
	var createPayload map[string]interface{}
	server := common.NewTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" && r.URL.Path == "/v1/owner/repo/webhooks.json" {
			createPayload = common.DecodeJSON(t, r)
			common.WriteJSON(t, w, map[string]interface{}{
				"id":      float64(2),
				"message": "创建成功",
			})
		} else {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	})
	defer server.Close()

	ctx := common.NewTestContext(t, server, "owner", "repo", map[string]string{
		"url":         "https://example.com/hook",
		"content-type": "json",
		"events":      "push,issues",
	})
	err := common.RunShortcut(t, Shortcuts(), "create", ctx)
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}

	common.AssertEqual(t, createPayload["url"], "https://example.com/hook")
}

func TestWebhookDelete(t *testing.T) {
	server := common.NewTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "DELETE" && r.URL.Path == "/v1/owner/repo/webhooks/1.json" {
			common.WriteJSON(t, w, map[string]interface{}{
				"status":  0,
				"message": "删除成功",
			})
		} else {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	})
	defer server.Close()

	ctx := common.NewTestContext(t, server, "owner", "repo", map[string]string{
		"id": "1",
	})
	err := common.RunShortcut(t, Shortcuts(), "delete", ctx)
	if err != nil {
		t.Fatalf("delete failed: %v", err)
	}
}

func TestWebView(t *testing.T) {
	server := common.NewTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" && r.URL.Path == "/v1/owner/repo/webhooks/1.json" {
			common.WriteJSON(t, w, map[string]interface{}{
				"id":           float64(1),
				"url":          "https://example.com/hook",
				"active":       true,
				"content_type": "json",
				"events":       []string{"push"},
			})
		} else {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	})
	defer server.Close()

	ctx := common.NewTestContext(t, server, "owner", "repo", map[string]string{
		"id": "1",
	})
	err := common.RunShortcut(t, Shortcuts(), "view", ctx)
	if err != nil {
		t.Fatalf("view failed: %v", err)
	}
}

func TestWebUpdate(t *testing.T) {
	var updatePayload map[string]interface{}
	server := common.NewTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "PUT" && r.URL.Path == "/v1/owner/repo/webhooks/1.json" {
			updatePayload = common.DecodeJSON(t, r)
			common.WriteJSON(t, w, map[string]interface{}{
				"id":      float64(1),
				"url":     "https://example.com/updated",
				"message": "更新成功",
			})
		} else {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	})
	defer server.Close()

	ctx := common.NewTestContext(t, server, "owner", "repo", map[string]string{
		"id":     "1",
		"url":    "https://example.com/updated",
		"events": "push,issues",
	})
	err := common.RunShortcut(t, Shortcuts(), "update", ctx)
	if err != nil {
		t.Fatalf("update failed: %v", err)
	}

	common.AssertEqual(t, updatePayload["url"], "https://example.com/updated")
	common.AssertEqual(t, updatePayload["http_method"], "POST")
}

func TestWebHistory(t *testing.T) {
	server := common.NewTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" && r.URL.Path == "/v1/owner/repo/webhooks/1/hooktasks.json" {
			common.WriteJSON(t, w, map[string]interface{}{
				"total_count": 2,
				"hooktasks": []interface{}{
					map[string]interface{}{"id": float64(10), "status": "succeeded"},
					map[string]interface{}{"id": float64(11), "status": "failed"},
				},
			})
		} else {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	})
	defer server.Close()

	ctx := common.NewTestContext(t, server, "owner", "repo", map[string]string{
		"id": "1",
	})
	err := common.RunShortcut(t, Shortcuts(), "history", ctx)
	if err != nil {
		t.Fatalf("history failed: %v", err)
	}
}

func TestWebTest(t *testing.T) {
	server := common.NewTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" && r.URL.Path == "/v1/owner/repo/webhooks/1/tests.json" {
			common.WriteJSON(t, w, map[string]interface{}{
				"status":  0,
				"message": "success",
			})
		} else {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	})
	defer server.Close()

	ctx := common.NewTestContext(t, server, "owner", "repo", map[string]string{
		"id": "1",
	})
	err := common.RunShortcut(t, Shortcuts(), "test", ctx)
	if err != nil {
		t.Fatalf("test failed: %v", err)
	}
}
