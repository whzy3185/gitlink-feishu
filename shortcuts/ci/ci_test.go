package ci

import (
	"net/http"
	"testing"

	"github.com/gitlink-org/gitlink-cli/shortcuts/common"
)

func TestCIBuilds(t *testing.T) {
	server := common.NewTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" && r.URL.Path == "/owner/repo/builds.json" {
			common.WriteJSON(t, w, map[string]interface{}{
				"total_count": float64(1),
				"builds": []interface{}{
					map[string]interface{}{
						"id":     float64(10),
						"status": "success",
					},
				},
			})
		} else {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	})
	defer server.Close()

	ctx := common.NewTestContext(t, server, "owner", "repo", map[string]string{})
	err := common.RunShortcut(t, Shortcuts(), "builds", ctx)
	if err != nil {
		t.Fatalf("builds failed: %v", err)
	}
}

func TestCILogs(t *testing.T) {
	server := common.NewTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" && r.URL.Path == "/owner/repo/builds/10/logs/1/1.json" {
			common.WriteJSON(t, w, map[string]interface{}{
				"build_id": float64(10),
				"stage":    float64(1),
				"step":     float64(1),
				"lines":    []interface{}{"Building..."},
			})
		} else {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	})
	defer server.Close()

	ctx := common.NewTestContext(t, server, "owner", "repo", map[string]string{
		"build": "10",
		"stage": "1",
		"step":  "1",
	})
	err := common.RunShortcut(t, Shortcuts(), "logs", ctx)
	if err != nil {
		t.Fatalf("logs failed: %v", err)
	}
}

func TestCIRestart(t *testing.T) {
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
		"build": "10",
	})
	err := common.RunShortcut(t, Shortcuts(), "restart", ctx)
	if err != nil {
		t.Fatalf("restart failed: %v", err)
	}
	if requestMethod != "POST" {
		t.Errorf("expected POST, got %s", requestMethod)
	}
}

func TestCIStop(t *testing.T) {
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
		"build": "10",
	})
	err := common.RunShortcut(t, Shortcuts(), "stop", ctx)
	if err != nil {
		t.Fatalf("stop failed: %v", err)
	}
	if requestMethod != "DELETE" {
		t.Errorf("expected DELETE, got %s", requestMethod)
	}
}

func TestCIToggle(t *testing.T) {
	tests := []struct {
		name       string
		shortcut   string
		wantMethod string
		wantPath   string
	}{
		{"enable", "enable", "POST", "/v1/owner/repo/actions/enable.json"},
		{"disable", "disable", "POST", "/v1/owner/repo/actions/disable.json"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var requestMethod string
			var requestPath string
			server := common.NewTestServer(t, func(w http.ResponseWriter, r *http.Request) {
				requestMethod = r.Method
				requestPath = r.URL.Path
				common.WriteJSON(t, w, map[string]interface{}{
					"status":  float64(0),
					"message": "success",
				})
			})
			defer server.Close()

			ctx := common.NewTestContext(t, server, "owner", "repo", map[string]string{})
			err := common.RunShortcut(t, Shortcuts(), tt.shortcut, ctx)
			if err != nil {
				t.Fatalf("%s failed: %v", tt.shortcut, err)
			}
			if requestMethod != tt.wantMethod {
				t.Errorf("expected %s, got %s", tt.wantMethod, requestMethod)
			}
			if requestPath != tt.wantPath {
				t.Errorf("expected %s, got %s", tt.wantPath, requestPath)
			}
		})
	}
}

func TestCIAuthorize(t *testing.T) {
	var requestMethod string
	var requestPath string
	server := common.NewTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		requestMethod = r.Method
		requestPath = r.URL.Path
		common.WriteJSON(t, w, map[string]interface{}{
			"status":  float64(0),
			"message": "success",
		})
	})
	defer server.Close()

	ctx := common.NewTestContext(t, server, "owner", "repo", map[string]string{})
	err := common.RunShortcut(t, Shortcuts(), "authorize", ctx)
	if err != nil {
		t.Fatalf("authorize failed: %v", err)
	}
	if requestMethod != "GET" {
		t.Errorf("expected GET, got %s", requestMethod)
	}
	if requestPath != "/owner/repo/ci_authorize.json" {
		t.Errorf("expected /owner/repo/ci_authorize.json, got %s", requestPath)
	}
}
