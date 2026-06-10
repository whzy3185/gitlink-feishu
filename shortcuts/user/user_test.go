package user

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gitlink-org/gitlink-cli/internal/client"
	"github.com/gitlink-org/gitlink-cli/shortcuts/common"
)

func runUserShortcut(t *testing.T, server *httptest.Server, name string, args map[string]string) error {
	t.Helper()
	shortcut := findUserShortcut(t, name)
	ctx := &common.RuntimeContext{
		Client: &client.Client{HTTP: server.Client(), BaseURL: server.URL},
		Format: "json",
		Args:   args,
	}
	return shortcut.Run(ctx)
}

func findUserShortcut(t *testing.T, name string) *common.Shortcut {
	t.Helper()
	for _, shortcut := range Shortcuts() {
		if shortcut.Name == name {
			return shortcut
		}
	}
	t.Fatalf("shortcut %q not found", name)
	return nil
}

func writeUserJSON(w http.ResponseWriter, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}

func TestUserMe(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/users/me.json" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		writeUserJSON(w, map[string]interface{}{
			"login": "currentuser",
			"name":  "Current User",
			"id":    1,
		})
	}))
	defer server.Close()

	if err := runUserShortcut(t, server, "me", nil); err != nil {
		t.Fatalf("me failed: %v", err)
	}
}

func TestUserInfo(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/users/alice.json" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		writeUserJSON(w, map[string]interface{}{
			"login": "alice",
			"name":  "Alice",
		})
	}))
	defer server.Close()

	if err := runUserShortcut(t, server, "info", map[string]string{"login": "alice"}); err != nil {
		t.Fatalf("info failed: %v", err)
	}
}

func TestUserInfoMissingLogin(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("no API call expected")
	}))
	defer server.Close()

	if err := runUserShortcut(t, server, "info", map[string]string{}); err == nil {
		t.Fatal("expected error for missing login")
	}
}

func TestUserHeadmapShortcutBuildsYearQuery(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/users/alice/headmaps.json" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if r.URL.Query().Get("year") != "2026" {
			t.Fatalf("unexpected year query: %q", r.URL.Query().Get("year"))
		}
		writeUserJSON(w, map[string]interface{}{
			"headmaps": []interface{}{
				map[string]interface{}{"date": "2026-06-01", "contributions": 5},
			},
			"total_contributions": 5,
		})
	}))
	defer server.Close()

	if err := runUserShortcut(t, server, "headmap", map[string]string{"login": "alice", "year": "2026"}); err != nil {
		t.Fatalf("headmap failed: %v", err)
	}
}

func TestUserDevelopShortcutBuildsTimeRangeQuery(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/users/alice/statistics/develop.json" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if r.URL.Query().Get("start_time") != "100" || r.URL.Query().Get("end_time") != "200" {
			t.Fatalf("unexpected time range query: %s", r.URL.RawQuery)
		}
		writeUserJSON(w, map[string]interface{}{
			"platform": map[string]interface{}{"activity": 90},
			"user": map[string]interface{}{
				"activity":            70,
				"languages_percent":   map[string]interface{}{"Go": 0.8},
				"each_language_score": map[string]interface{}{"Go": 88},
			},
		})
	}))
	defer server.Close()

	if err := runUserShortcut(t, server, "develop", map[string]string{
		"login":      "alice",
		"start-time": "100",
		"end-time":   "200",
	}); err != nil {
		t.Fatalf("develop failed: %v", err)
	}
}

func TestUserTrendsFetchesAllPagesWhenFiltersPresent(t *testing.T) {
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		page := r.URL.Query().Get("page")
		switch page {
		case "1":
			writeUserJSON(w, map[string]interface{}{
				"project_trends": []interface{}{
					map[string]interface{}{
						"id":          1,
						"trend_id":    11,
						"trend_type":  "CommitLog",
						"name":        "initial commit",
						"action_type": "创建了代码提交(Commit)",
						"user_login":  "alice",
						"user_name":   "Alice",
						"project": map[string]interface{}{
							"identifier": "repo-a",
							"owner":      map[string]interface{}{"login": "Gitlink"},
						},
					},
				},
				"total_count": 2,
			})
		case "2":
			writeUserJSON(w, map[string]interface{}{
				"project_trends": []interface{}{
					map[string]interface{}{
						"id":          2,
						"trend_id":    12,
						"trend_type":  "PullRequest",
						"name":        "improve docs",
						"action_type": "创建了合并请求(PR)",
						"user_login":  "alice",
						"user_name":   "Alice",
						"project": map[string]interface{}{
							"identifier": "repo-b",
							"owner":      map[string]interface{}{"login": "Gitlink"},
						},
					},
				},
				"total_count": 2,
			})
		default:
			t.Fatalf("unexpected page query: %s", page)
		}
	}))
	defer server.Close()

	if err := runUserShortcut(t, server, "trends", map[string]string{
		"login":      "alice",
		"page":       "1",
		"limit":      "1",
		"trend-type": "PullRequest",
	}); err != nil {
		t.Fatalf("trends failed: %v", err)
	}
	if requests != 2 {
		t.Fatalf("expected 2 requests for filtered trends, got %d", requests)
	}
}

func TestBuildTimeRangeQueryRejectsInvalidOrder(t *testing.T) {
	if _, _, err := buildTimeRangeQuery("200", "100"); err == nil {
		t.Fatal("expected invalid time range error")
	}
}

func TestNormalizeActivityDataBuildsTimelineSummary(t *testing.T) {
	result, err := normalizeActivityData("alice", map[string]interface{}{
		"dates":               []interface{}{"2026.06.01", "2026.06.02"},
		"commits_count":       []interface{}{3, 5},
		"issues_count":        []interface{}{1, 0},
		"pull_requests_count": []interface{}{0, 2},
	})
	if err != nil {
		t.Fatalf("normalizeActivityData failed: %v", err)
	}

	totals := result["totals"].(map[string]interface{})
	if totals["all"].(int) != 11 {
		t.Fatalf("unexpected total activity count: %#v", totals)
	}

	peakDay := result["peak_day"].(map[string]interface{})
	if peakDay["date"].(string) != "2026.06.02" {
		t.Fatalf("unexpected peak day: %#v", peakDay)
	}
}

func TestNormalizeDevelopDataSortsLanguages(t *testing.T) {
	result, err := normalizeDevelopData("alice", map[string]interface{}{"start_time": 100}, map[string]interface{}{
		"platform": map[string]interface{}{
			"activity": 90,
		},
		"user": map[string]interface{}{
			"activity":            70,
			"languages_percent":   map[string]interface{}{"Go": 0.4, "Python": 0.6},
			"each_language_score": map[string]interface{}{"Go": 80, "Python": 95},
		},
	})
	if err != nil {
		t.Fatalf("normalizeDevelopData failed: %v", err)
	}

	languages := result["languages"].([]map[string]interface{})
	if len(languages) != 2 || languages[0]["name"].(string) != "Python" {
		t.Fatalf("languages not sorted by percent: %#v", languages)
	}
	if result["period"].(map[string]interface{})["start_time"].(int) != 100 {
		t.Fatalf("unexpected period: %#v", result["period"])
	}
}

func TestNormalizeRoleDataSortsRoleCounts(t *testing.T) {
	result, err := normalizeRoleData("alice", nil, map[string]interface{}{
		"role": map[string]interface{}{
			"owner":     map[string]interface{}{"count": 3, "percent": 0.6},
			"developer": map[string]interface{}{"count": 2, "percent": 0.4},
		},
		"total_projects_count": 5,
	})
	if err != nil {
		t.Fatalf("normalizeRoleData failed: %v", err)
	}

	roles := result["roles"].([]map[string]interface{})
	if roles[0]["name"].(string) != "owner" {
		t.Fatalf("expected owner to be primary role: %#v", roles)
	}
}

func TestNormalizeTrendDataAppliesFilters(t *testing.T) {
	result, err := normalizeTrendData("alice", 1, 20, 2, true, trendFilters{TrendType: "PullRequest"}, []interface{}{
		map[string]interface{}{
			"id":          1,
			"trend_id":    11,
			"trend_type":  "CommitLog",
			"name":        "initial commit",
			"action_type": "创建了代码提交(Commit)",
			"user_login":  "alice",
			"user_name":   "Alice",
			"project": map[string]interface{}{
				"identifier": "repo-a",
				"owner":      map[string]interface{}{"login": "Gitlink"},
			},
		},
		map[string]interface{}{
			"id":          2,
			"trend_id":    12,
			"trend_type":  "PullRequest",
			"name":        "improve docs",
			"action_type": "创建了合并请求(PR)",
			"user_login":  "alice",
			"user_name":   "Alice",
			"project": map[string]interface{}{
				"identifier": "repo-b",
				"owner":      map[string]interface{}{"login": "Gitlink"},
			},
		},
	})
	if err != nil {
		t.Fatalf("normalizeTrendData failed: %v", err)
	}

	items := result["items"].([]map[string]interface{})
	if len(items) != 1 || items[0]["trend_type"].(string) != "PullRequest" {
		t.Fatalf("unexpected filtered trend items: %#v", items)
	}
}

func TestUserInfoHTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("server error"))
	}))
	defer server.Close()

	if err := runUserShortcut(t, server, "info", map[string]string{"login": "alice"}); err == nil {
		t.Fatal("expected error for HTTP 500")
	}
}
