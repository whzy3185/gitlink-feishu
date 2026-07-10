package branch

import (
	"net/http"
	"strings"
	"testing"

	"github.com/gitlink-org/gitlink-cli/shortcuts/common"
)

func TestBranchList(t *testing.T) {
	server := common.NewTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" && strings.Contains(r.URL.Path, "/branches") {
			common.WriteJSON(t, w, map[string]interface{}{
				"total_count": float64(1),
				"branches": []interface{}{
					map[string]interface{}{
						"name":      "master",
						"protected": false,
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

func TestBranchCreate(t *testing.T) {
	var requestMethod string
	server := common.NewTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		requestMethod = r.Method
		common.WriteJSON(t, w, map[string]interface{}{
			"name":      "feature-1",
			"protected": false,
		})
	})
	defer server.Close()

	ctx := common.NewTestContext(t, server, "owner", "repo", map[string]string{
		"name": "feature-1",
	})
	err := common.RunShortcut(t, Shortcuts(), "create", ctx)
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}
	if requestMethod != "POST" {
		t.Errorf("expected POST, got %s", requestMethod)
	}
}

func TestBranchDelete(t *testing.T) {
	var requestMethod string
	server := common.NewTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		requestMethod = r.Method
		common.WriteJSON(t, w, map[string]interface{}{
			"status":  float64(0),
			"message": "success",
		})
	})
	defer server.Close()

	ctx := common.NewTestContext(t, server, "owner", "repo", map[string]string{
		"name": "old-branch",
	})
	err := common.RunShortcut(t, Shortcuts(), "delete", ctx)
	if err != nil {
		t.Fatalf("delete failed: %v", err)
	}
	if requestMethod != "POST" {
		t.Errorf("expected POST, got %s", requestMethod)
	}
}

func TestBranchProtect(t *testing.T) {
	var requestMethod string
	server := common.NewTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		requestMethod = r.Method
		common.WriteJSON(t, w, map[string]interface{}{
			"status":  float64(0),
			"message": "success",
		})
	})
	defer server.Close()

	ctx := common.NewTestContext(t, server, "owner", "repo", map[string]string{
		"name": "master",
	})
	err := common.RunShortcut(t, Shortcuts(), "protect", ctx)
	if err != nil {
		t.Fatalf("protect failed: %v", err)
	}
	if requestMethod != "POST" {
		t.Errorf("expected POST, got %s", requestMethod)
	}
}

func TestBranchUnprotect(t *testing.T) {
	var requestMethod string
	server := common.NewTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		requestMethod = r.Method
		common.WriteJSON(t, w, map[string]interface{}{
			"status":  float64(0),
			"message": "success",
		})
	})
	defer server.Close()

	ctx := common.NewTestContext(t, server, "owner", "repo", map[string]string{
		"name": "master",
	})
	err := common.RunShortcut(t, Shortcuts(), "unprotect", ctx)
	if err != nil {
		t.Fatalf("unprotect failed: %v", err)
	}
	if requestMethod != "DELETE" {
		t.Errorf("expected DELETE, got %s", requestMethod)
	}
}
