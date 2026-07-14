package user

import (
	"net/http"
	"testing"

	"github.com/gitlink-org/gitlink-cli/shortcuts/common"
)

func TestUserMe(t *testing.T) {
	server := common.NewTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" && r.URL.Path == "/users/me.json" {
			common.WriteJSON(t, w, map[string]interface{}{
				"login":   "alice",
				"user_id": float64(42),
				"name":    "Alice",
			})
		} else {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	})
	defer server.Close()

	ctx := common.NewTestContext(t, server, "", "", map[string]string{})
	err := common.RunShortcut(t, Shortcuts(), "me", ctx)
	if err != nil {
		t.Fatalf("me failed: %v", err)
	}
}

func TestUserInfo(t *testing.T) {
	server := common.NewTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" && r.URL.Path == "/users/bob.json" {
			common.WriteJSON(t, w, map[string]interface{}{
				"login":   "bob",
				"user_id": float64(7),
				"name":    "Bob",
			})
		} else {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	})
	defer server.Close()

	ctx := common.NewTestContext(t, server, "", "", map[string]string{
		"login": "bob",
	})
	err := common.RunShortcut(t, Shortcuts(), "info", ctx)
	if err != nil {
		t.Fatalf("info failed: %v", err)
	}
}

func TestUserShortcutsMissingLogin(t *testing.T) {
	tests := []string{"info", "headmaps", "trends"}
	for _, name := range tests {
		t.Run(name, func(t *testing.T) {
			server := common.NewTestServer(t, func(w http.ResponseWriter, r *http.Request) {
				t.Fatalf("no request should be made: %s %s", r.Method, r.URL.Path)
			})
			defer server.Close()

			ctx := common.NewTestContext(t, server, "", "", map[string]string{})
			err := common.RunShortcut(t, Shortcuts(), name, ctx)
			if err == nil {
				t.Fatal("expected error for missing --login")
			}
		})
	}
}

func TestUserHeadmaps(t *testing.T) {
	server := common.NewTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" && r.URL.Path == "/users/alice/headmaps.json" {
			common.WriteJSON(t, w, map[string]interface{}{
				"headmaps": []interface{}{
					map[string]interface{}{"date": "2025-01-01", "count": float64(5)},
				},
			})
		} else {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	})
	defer server.Close()

	ctx := common.NewTestContext(t, server, "", "", map[string]string{
		"login": "alice",
	})
	err := common.RunShortcut(t, Shortcuts(), "headmaps", ctx)
	if err != nil {
		t.Fatalf("headmaps failed: %v", err)
	}
}

func TestUserStatsEndpoints(t *testing.T) {
	tests := []struct {
		name     string
		shortcut string
		wantPath string
		respData map[string]interface{}
	}{
		{
			name: "activity", shortcut: "stats-activity",
			wantPath: "/users/alice/statistics/activity.json",
			respData: map[string]interface{}{"dates": []string{"2025-01-01"}, "commits_count": []float64{3}},
		},
		{
			name: "develop", shortcut: "stats-develop",
			wantPath: "/users/alice/statistics/develop.json",
			respData: map[string]interface{}{"score": float64(80)},
		},
		{
			name: "role", shortcut: "stats-role",
			wantPath: "/users/alice/statistics/role.json",
			respData: map[string]interface{}{"role": "developer"},
		},
		{
			name: "major", shortcut: "stats-major",
			wantPath: "/users/alice/statistics/major.json",
			respData: map[string]interface{}{"major": "backend"},
		},
	}
	for _, tt := range tests {
		// 功能测试
		t.Run(tt.name, func(t *testing.T) {
			server := common.NewTestServer(t, func(w http.ResponseWriter, r *http.Request) {
				if r.Method == "GET" && r.URL.Path == tt.wantPath {
					common.WriteJSON(t, w, tt.respData)
				} else {
					t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
				}
			})
			defer server.Close()

			ctx := common.NewTestContext(t, server, "", "", map[string]string{"login": "alice"})
			err := common.RunShortcut(t, Shortcuts(), tt.shortcut, ctx)
			if err != nil {
				t.Fatalf("%s failed: %v", tt.shortcut, err)
			}
		})

		// MissingLogin 测试
		t.Run(tt.name+"_missing_login", func(t *testing.T) {
			server := common.NewTestServer(t, func(w http.ResponseWriter, r *http.Request) {
				t.Fatalf("no request should be made: %s %s", r.Method, r.URL.Path)
			})
			defer server.Close()

			ctx := common.NewTestContext(t, server, "", "", map[string]string{})
			err := common.RunShortcut(t, Shortcuts(), tt.shortcut, ctx)
			if err == nil {
				t.Fatal("expected error for missing --login")
			}
		})
	}
}

func TestUserTrends(t *testing.T) {
	server := common.NewTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" && r.URL.Path == "/users/alice/project_trends.json" {
			// 验证分页参数
			if r.URL.Query().Get("page") != "1" {
				t.Fatalf("expected page=1, got %s", r.URL.Query().Get("page"))
			}
			if r.URL.Query().Get("limit") != "20" {
				t.Fatalf("expected limit=20, got %s", r.URL.Query().Get("limit"))
			}
			common.WriteJSON(t, w, map[string]interface{}{
				"total_count": float64(69),
				"project_trends": []interface{}{
					map[string]interface{}{"id": float64(1), "name": "trend1"},
				},
			})
		} else {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	})
	defer server.Close()

	ctx := common.NewTestContext(t, server, "", "", map[string]string{
		"login": "alice",
		"page":  "1",
		"limit": "20",
	})
	err := common.RunShortcut(t, Shortcuts(), "trends", ctx)
	if err != nil {
		t.Fatalf("trends failed: %v", err)
	}
}

func TestUserTrendsDefaultPagination(t *testing.T) {
	server := common.NewTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" && r.URL.Path == "/users/alice/project_trends.json" {
			// 不传 page/limit 时，Arg 返回空字符串，url.Values.Set 设置空值
			common.WriteJSON(t, w, map[string]interface{}{
				"total_count":    float64(69),
				"project_trends": []interface{}{},
			})
		} else {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	})
	defer server.Close()

	ctx := common.NewTestContext(t, server, "", "", map[string]string{
		"login": "alice",
	})
	err := common.RunShortcut(t, Shortcuts(), "trends", ctx)
	if err != nil {
		t.Fatalf("trends failed: %v", err)
	}
}
