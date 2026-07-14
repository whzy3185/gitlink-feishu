package member

import (
	"net/http"
	"testing"

	"github.com/gitlink-org/gitlink-cli/shortcuts/common"
)

func TestMemberList(t *testing.T) {
	server := common.NewTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" && r.URL.Path == "/owner/repo/collaborators.json" {
			common.WriteJSON(t, w, map[string]interface{}{
				"members": []interface{}{
					map[string]interface{}{
						"id":   float64(1),
						"login": "developer",
						"role":  "Manager",
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

func TestMemberAdd(t *testing.T) {
	var addPayload map[string]interface{}
	server := common.NewTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" && r.URL.Path == "/owner/repo/collaborators.json" {
			addPayload = common.DecodeJSON(t, r)
			common.WriteJSON(t, w, map[string]interface{}{
				"status":  0,
				"message": "添加成功",
			})
		} else {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	})
	defer server.Close()

	ctx := common.NewTestContext(t, server, "owner", "repo", map[string]string{
		"user-id": "42",
	})
	err := common.RunShortcut(t, Shortcuts(), "add", ctx)
	if err != nil {
		t.Fatalf("add failed: %v", err)
	}

	common.AssertEqual(t, addPayload["user_id"], "42")
}

func TestMemberRemove(t *testing.T) {
	var removePayload map[string]interface{}
	server := common.NewTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "DELETE" && r.URL.Path == "/owner/repo/collaborators/remove.json" {
			removePayload = common.DecodeJSON(t, r)
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
		"user-id": "42",
	})
	err := common.RunShortcut(t, Shortcuts(), "remove", ctx)
	if err != nil {
		t.Fatalf("remove failed: %v", err)
	}

	common.AssertEqual(t, removePayload["user_id"], "42")
}
