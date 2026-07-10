package org

import (
	"net/http"
	"testing"

	"github.com/gitlink-org/gitlink-cli/shortcuts/common"
)

func TestOrgList(t *testing.T) {
	server := common.NewTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" && r.URL.Path == "/organizations.json" {
			common.WriteJSON(t, w, map[string]interface{}{
				"total_count": float64(1),
				"organizations": []interface{}{
					map[string]interface{}{
						"id":   float64(1),
						"name": "test-org",
					},
				},
			})
		} else {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	})
	defer server.Close()

	ctx := common.NewTestContext(t, server, "", "", map[string]string{})
	err := common.RunShortcut(t, Shortcuts(), "list", ctx)
	if err != nil {
		t.Fatalf("list failed: %v", err)
	}
}

func TestOrgInfo(t *testing.T) {
	server := common.NewTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" && r.URL.Path == "/organizations/5.json" {
			common.WriteJSON(t, w, map[string]interface{}{
				"id":   float64(5),
				"name": "test-org",
			})
		} else {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	})
	defer server.Close()

	ctx := common.NewTestContext(t, server, "", "", map[string]string{
		"id": "5",
	})
	err := common.RunShortcut(t, Shortcuts(), "info", ctx)
	if err != nil {
		t.Fatalf("info failed: %v", err)
	}
}

func TestOrgMembers(t *testing.T) {
	server := common.NewTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" && r.URL.Path == "/organizations/5/organization_users.json" {
			common.WriteJSON(t, w, map[string]interface{}{
				"total_count": float64(2),
				"organization_users": []interface{}{
					map[string]interface{}{
						"user": map[string]interface{}{"login": "alice"},
					},
				},
			})
		} else {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	})
	defer server.Close()

	ctx := common.NewTestContext(t, server, "", "", map[string]string{
		"id": "5",
	})
	err := common.RunShortcut(t, Shortcuts(), "members", ctx)
	if err != nil {
		t.Fatalf("members failed: %v", err)
	}
}

func TestOrgCreate(t *testing.T) {
	var requestMethod string
	server := common.NewTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		requestMethod = r.Method
		payload := common.DecodeJSON(t, r)
		if payload["name"] != "new-org" {
			t.Fatalf("expected name=new-org, got %v", payload["name"])
		}
		if payload["nickname"] != "new-org" {
			t.Fatalf("expected nickname=new-org, got %v", payload["nickname"])
		}
		if payload["visibility"] != "common" {
			t.Fatalf("expected visibility=common, got %v", payload["visibility"])
		}
		common.WriteJSON(t, w, map[string]interface{}{
			"id":   float64(10),
			"name": "new-org",
		})
	})
	defer server.Close()

	ctx := common.NewTestContext(t, server, "", "", map[string]string{
		"name": "new-org",
	})
	err := common.RunShortcut(t, Shortcuts(), "create", ctx)
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}
	if requestMethod != "POST" {
		t.Errorf("expected POST, got %s", requestMethod)
	}
}

// --- teams ---

func TestOrgTeams(t *testing.T) {
	server := common.NewTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" && r.URL.Path == "/organizations/5/teams.json" {
			common.WriteJSON(t, w, map[string]interface{}{
				"total_count": float64(1),
				"teams": []interface{}{
					map[string]interface{}{
						"id":   float64(1),
						"name": "dev-team",
					},
				},
			})
		} else {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	})
	defer server.Close()

	ctx := common.NewTestContext(t, server, "", "", map[string]string{
		"id": "5",
	})
	err := common.RunShortcut(t, Shortcuts(), "teams", ctx)
	if err != nil {
		t.Fatalf("teams failed: %v", err)
	}
}

// --- create-team ---

func TestOrgCreateTeam(t *testing.T) {
	var requestMethod string
	server := common.NewTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		requestMethod = r.Method
		payload := common.DecodeJSON(t, r)
		if payload["name"] != "new-team" {
			t.Fatalf("expected name=new-team, got %v", payload["name"])
		}
		common.WriteJSON(t, w, map[string]interface{}{
			"id":   float64(1),
			"name": "new-team",
		})
	})
	defer server.Close()

	ctx := common.NewTestContext(t, server, "", "", map[string]string{
		"id":   "5",
		"name": "new-team",
	})
	err := common.RunShortcut(t, Shortcuts(), "create-team", ctx)
	if err != nil {
		t.Fatalf("create-team failed: %v", err)
	}
	if requestMethod != "POST" {
		t.Errorf("expected POST, got %s", requestMethod)
	}
}

// --- remove-member ---

func TestOrgRemoveMember(t *testing.T) {
	var requestMethod string
	server := common.NewTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		requestMethod = r.Method
		common.WriteJSON(t, w, map[string]interface{}{
			"ok": true,
		})
	})
	defer server.Close()

	ctx := common.NewTestContext(t, server, "", "", map[string]string{
		"id":  "5",
		"uid": "42",
	})
	err := common.RunShortcut(t, Shortcuts(), "remove-member", ctx)
	if err != nil {
		t.Fatalf("remove-member failed: %v", err)
	}
	if requestMethod != "DELETE" {
		t.Errorf("expected DELETE, got %s", requestMethod)
	}
}

// --- nickname (view) ---

func TestOrgNicknameView(t *testing.T) {
	server := common.NewTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" && r.URL.Path == "/organizations/5/organization_users/42.json" {
			common.WriteJSON(t, w, map[string]interface{}{
				"nickname": "thename",
			})
		} else {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	})
	defer server.Close()

	ctx := common.NewTestContext(t, server, "", "", map[string]string{
		"id":  "5",
		"uid": "42",
	})
	err := common.RunShortcut(t, Shortcuts(), "nickname", ctx)
	if err != nil {
		t.Fatalf("nickname failed: %v", err)
	}
}

// --- nickname (set) ---

func TestOrgNicknameSet(t *testing.T) {
	var requestMethod string
	server := common.NewTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		requestMethod = r.Method
		payload := common.DecodeJSON(t, r)
		if payload["nickname"] != "newname" {
			t.Fatalf("expected nickname=newname, got %v", payload["nickname"])
		}
		common.WriteJSON(t, w, map[string]interface{}{
			"nickname": "newname",
		})
	})
	defer server.Close()

	ctx := common.NewTestContext(t, server, "", "", map[string]string{
		"id":       "5",
		"uid":      "42",
		"nickname": "newname",
	})
	err := common.RunShortcut(t, Shortcuts(), "nickname", ctx)
	if err != nil {
		t.Fatalf("nickname set failed: %v", err)
	}
	if requestMethod != "PUT" {
		t.Errorf("expected PUT, got %s", requestMethod)
	}
}

// --- uid ---

func TestOrgUID(t *testing.T) {
	server := common.NewTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" && r.URL.Path == "/users/baoerjun.json" {
			common.WriteJSON(t, w, map[string]interface{}{
				"id":    float64(148287),
				"login": "baoerjun",
			})
		} else {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	})
	defer server.Close()

	ctx := common.NewTestContext(t, server, "", "", map[string]string{
		"login": "baoerjun",
	})
	err := common.RunShortcut(t, Shortcuts(), "uid", ctx)
	if err != nil {
		t.Fatalf("uid failed: %v", err)
	}
}
