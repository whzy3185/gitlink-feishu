package label

import (
	"net/http"
	"testing"

	"github.com/gitlink-org/gitlink-cli/shortcuts/common"
)

func TestLabelList(t *testing.T) {
	server := common.NewTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" && r.URL.Path == "/v1/owner/repo/issue_tags.json" {
			common.WriteJSON(t, w, map[string]interface{}{
				"issue_tags": []interface{}{
					map[string]interface{}{
						"name":  "bug",
						"color": "#FF0000",
					},
				},
				"total_count": 1,
			})
		} else {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	})
	defer server.Close()

	ctx := common.NewTestContext(t, server, "owner", "repo", map[string]string{
		"order-by":        "created_on",
		"order-direction": "desc",
	})
	err := common.RunShortcut(t, Shortcuts(), "list", ctx)
	if err != nil {
		t.Fatalf("list failed: %v", err)
	}
}

func TestLabelCreate(t *testing.T) {
	var createPayload map[string]interface{}
	server := common.NewTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" && r.URL.Path == "/v1/owner/repo/issue_tags.json" {
			createPayload = common.DecodeJSON(t, r)
			common.WriteJSON(t, w, map[string]interface{}{
				"status":  0,
				"message": "创建成功",
			})
		} else {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	})
	defer server.Close()

	ctx := common.NewTestContext(t, server, "owner", "repo", map[string]string{
		"name":        "enhancement",
		"color":       "#00FF00",
		"description": "New feature",
	})
	err := common.RunShortcut(t, Shortcuts(), "create", ctx)
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}

	common.AssertEqual(t, createPayload["name"], "enhancement")
	common.AssertEqual(t, createPayload["color"], "#00FF00")
}

func TestLabelDelete(t *testing.T) {
	server := common.NewTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "DELETE" && r.URL.Path == "/v1/owner/repo/issue_tags/3.json" {
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
		"id": "3",
	})
	err := common.RunShortcut(t, Shortcuts(), "delete", ctx)
	if err != nil {
		t.Fatalf("delete failed: %v", err)
	}
}

func TestLabelUpdate(t *testing.T) {
	var updatePayload map[string]interface{}
	server := common.NewTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "PATCH" && r.URL.Path == "/v1/owner/repo/issue_tags/7.json" {
			updatePayload = common.DecodeJSON(t, r)
			common.WriteJSON(t, w, map[string]interface{}{
				"status":  0,
				"message": "更新成功",
			})
		} else {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	})
	defer server.Close()

	ctx := common.NewTestContext(t, server, "owner", "repo", map[string]string{
		"id":          "7",
		"name":        "enhancement",
		"color":       "#0000FF",
		"description": "New feature",
	})
	err := common.RunShortcut(t, Shortcuts(), "update", ctx)
	if err != nil {
		t.Fatalf("update failed: %v", err)
	}

	common.AssertEqual(t, updatePayload["name"], "enhancement")
	common.AssertEqual(t, updatePayload["color"], "#0000FF")
	common.AssertEqual(t, updatePayload["description"], "New feature")
}

func TestLabelUpdateRequiresAtLeastOneField(t *testing.T) {
	server := common.NewTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("no request should be made without update fields")
	})
	defer server.Close()

	ctx := common.NewTestContext(t, server, "owner", "repo", map[string]string{
		"id": "1",
	})
	err := common.RunShortcut(t, Shortcuts(), "update", ctx)
	if err == nil {
		t.Fatal("expected error when no fields provided, got nil")
	}
}
