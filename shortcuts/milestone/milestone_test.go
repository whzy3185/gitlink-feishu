package milestone

import (
	"net/http"
	"testing"

	"github.com/gitlink-org/gitlink-cli/shortcuts/common"
)

func TestMilestoneList(t *testing.T) {
	server := common.NewTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" && r.URL.Path == "/v1/owner/repo/milestones.json" {
			common.WriteJSON(t, w, map[string]interface{}{
				"milestones": []interface{}{
					map[string]interface{}{
						"id":     float64(1),
						"name":   "v1.0",
						"status": "open",
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
		"status": "all",
		"page":   "1",
		"limit":  "20",
	})
	err := common.RunShortcut(t, Shortcuts(), "list", ctx)
	if err != nil {
		t.Fatalf("list failed: %v", err)
	}
}

func TestMilestoneCreate(t *testing.T) {
	var createPayload map[string]interface{}
	server := common.NewTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" && r.URL.Path == "/v1/owner/repo/milestones.json" {
			createPayload = common.DecodeJSON(t, r)
			common.WriteJSON(t, w, map[string]interface{}{
				"id":      float64(2),
				"name":    "v2.0",
				"status":  "open",
				"message": "创建成功",
			})
		} else {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	})
	defer server.Close()

	ctx := common.NewTestContext(t, server, "owner", "repo", map[string]string{
		"name":        "v2.0",
		"description": "Second release",
		"due-date":    "2026-07-01",
	})
	err := common.RunShortcut(t, Shortcuts(), "create", ctx)
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}

	common.AssertEqual(t, createPayload["name"], "v2.0")
	common.AssertEqual(t, createPayload["description"], "Second release")
	common.AssertEqual(t, createPayload["effective_date"], "2026-07-01")
}

func TestMilestoneView(t *testing.T) {
	server := common.NewTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" && r.URL.Path == "/v1/owner/repo/milestones/1.json" {
			common.WriteJSON(t, w, map[string]interface{}{
				"id":     float64(1),
				"name":   "v1.0",
				"status": "open",
				"issues": []interface{}{},
			})
		} else {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	})
	defer server.Close()

	ctx := common.NewTestContext(t, server, "owner", "repo", map[string]string{
		"id":       "1",
		"category": "all",
	})
	err := common.RunShortcut(t, Shortcuts(), "view", ctx)
	if err != nil {
		t.Fatalf("view failed: %v", err)
	}
}

func TestMilestoneDelete(t *testing.T) {
	server := common.NewTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "DELETE" && r.URL.Path == "/v1/owner/repo/milestones/1.json" {
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

func TestMilestoneClose(t *testing.T) {
	var closePayload map[string]interface{}
	server := common.NewTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		// Regression guard: +close must hit /v1/{owner}/{repo}/milestones/{id}/update_status
		// (previously malformed to /{owner}/{repo}/{owner}/milestones/{id}/update_status — Owner duplicated, Repo dropped, no /v1).
		if r.Method == "POST" && r.URL.Path == "/v1/owner/repo/milestones/1/update_status.json" {
			closePayload = common.DecodeJSON(t, r)
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
		"id": "1",
	})
	err := common.RunShortcut(t, Shortcuts(), "close", ctx)
	if err != nil {
		t.Fatalf("close failed: %v", err)
	}
	common.AssertEqual(t, closePayload["status"], "closed")
}
