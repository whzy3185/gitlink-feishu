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

func TestUserStatisticsHTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("server error"))
	}))
	defer server.Close()

	for _, shortcut := range []string{"activity", "headmap", "develop", "role", "major"} {
		t.Run(shortcut, func(t *testing.T) {
			err := runShortcut(t, server, shortcut, map[string]string{"login": "alice"})
			if err == nil {
				t.Fatal("expected error for HTTP 500")
			}
		})
	}
}

func TestUserActivityUsesLogin(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" || r.URL.Path != "/users/alice/statistics/activity.json" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		writeJSON(w, map[string]interface{}{"dates": []string{"2026.06.13"}, "issues_count": []int{1}})
	}))
	defer server.Close()

	if err := runShortcut(t, server, "activity", map[string]string{"login": "alice"}); err != nil {
		t.Fatalf("activity failed: %v", err)
	}
}

func TestUserHeadmapBuildsYearQuery(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" || r.URL.Path != "/users/alice/headmaps.json" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		if got := r.URL.Query().Get("year"); got != "2026" {
			t.Fatalf("year = %q", got)
		}
		writeJSON(w, map[string]interface{}{"total_contributions": 1, "headmaps": []interface{}{}})
	}))
	defer server.Close()

	if err := runShortcut(t, server, "headmap", map[string]string{"login": "alice", "year": "2026"}); err != nil {
		t.Fatalf("headmap failed: %v", err)
	}
}

func TestUserDevelopBuildsTimeRangeQuery(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" || r.URL.Path != "/users/alice/statistics/develop.json" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		if got := r.URL.Query().Get("start_time"); got != "1717200000" {
			t.Fatalf("start_time = %q", got)
		}
		if got := r.URL.Query().Get("end_time"); got != "1719800000" {
			t.Fatalf("end_time = %q", got)
		}
		writeJSON(w, map[string]interface{}{"user": map[string]interface{}{"activity": 90}})
	}))
	defer server.Close()

	err := runShortcut(t, server, "develop", map[string]string{"login": "alice", "start-time": "1717200000", "end-time": "1719800000"})
	if err != nil {
		t.Fatalf("develop failed: %v", err)
	}
}

func TestUserRoleAndMajorUseStatisticsEndpoints(t *testing.T) {
	calls := []string{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls = append(calls, r.URL.Path)
		writeJSON(w, map[string]interface{}{"ok": true})
	}))
	defer server.Close()

	if err := runShortcut(t, server, "role", map[string]string{"login": "alice"}); err != nil {
		t.Fatalf("role failed: %v", err)
	}
	if err := runShortcut(t, server, "major", map[string]string{"login": "alice"}); err != nil {
		t.Fatalf("major failed: %v", err)
	}
	want := []string{"/users/alice/statistics/role.json", "/users/alice/statistics/major.json"}
	if len(calls) != len(want) {
		t.Fatalf("calls = %#v", calls)
	}
	for i := range want {
		if calls[i] != want[i] {
			t.Fatalf("calls = %#v, want %#v", calls, want)
		}
	}
}

func TestUserStatisticsResolveCurrentUser(t *testing.T) {
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		switch requests {
		case 1:
			if r.URL.Path != "/users/me.json" {
				t.Fatalf("unexpected path: %s", r.URL.Path)
			}
			writeJSON(w, map[string]interface{}{"login": "current"})
		case 2:
			if r.URL.Path != "/users/current/statistics/activity.json" {
				t.Fatalf("unexpected path: %s", r.URL.Path)
			}
			writeJSON(w, map[string]interface{}{"dates": []interface{}{}})
		default:
			t.Fatalf("unexpected extra request: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	shortcut := findShortcut(t, "activity")
	ctx := &common.RuntimeContext{Client: &client.Client{HTTP: server.Client(), BaseURL: server.URL}, Format: "json", Args: map[string]string{}}
	if err := shortcut.Run(ctx); err != nil {
		t.Fatalf("activity failed: %v", err)
	}
}

func TestUserStatisticsValidation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatalf("invalid input should not call API, got %s %s", r.Method, r.URL.Path)
	}))
	defer server.Close()

	cases := []struct {
		name     string
		shortcut string
		args     map[string]string
	}{
		{name: "bad year", shortcut: "headmap", args: map[string]string{"login": "alice", "year": "abcd"}},
		{name: "negative start", shortcut: "develop", args: map[string]string{"login": "alice", "start-time": "-1"}},
		{name: "start after end", shortcut: "role", args: map[string]string{"login": "alice", "start-time": "20", "end-time": "10"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if err := runShortcut(t, server, tc.shortcut, tc.args); err == nil {
				t.Fatal("expected validation error")
			}
		})
	}
}

// --- heatmap ---

func TestUserHeatmap(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/users/alice/headmaps.json" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if r.URL.Query().Get("year") != "" {
			t.Fatalf("expected no year query param, got %s", r.URL.Query().Get("year"))
		}
		writeJSON(w, map[string]interface{}{
			"contributions": []interface{}{
				map[string]interface{}{"date": "2026-01-01", "count": float64(5)},
			},
		})
	}))
	defer server.Close()

	err := runShortcut(t, server, "heatmap", map[string]string{"login": "alice"})
	if err != nil {
		t.Fatalf("heatmap failed: %v", err)
	}
}

func TestUserHeatmapWithYear(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/users/alice/headmaps.json" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if r.URL.Query().Get("year") != "2025" {
			t.Fatalf("expected year=2025, got %s", r.URL.Query().Get("year"))
		}
		writeJSON(w, map[string]interface{}{
			"contributions": []interface{}{},
		})
	}))
	defer server.Close()

	err := runShortcut(t, server, "heatmap", map[string]string{"login": "alice", "year": "2025"})
	if err != nil {
		t.Fatalf("heatmap with year failed: %v", err)
	}
}

func TestUserHeatmapMissingLogin(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("no API call expected")
	}))
	defer server.Close()

	err := runShortcut(t, server, "heatmap", map[string]string{})
	if err == nil {
		t.Fatal("expected error for missing login")
	}
}

func TestUserHeatmapHTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("server error"))
	}))
	defer server.Close()

	err := runShortcut(t, server, "heatmap", map[string]string{"login": "alice"})
	if err == nil {
		t.Fatal("expected error for HTTP 500")
	}
}

// --- stats ---

func TestUserStats(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/users/alice/statistics/develop.json" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if r.URL.Query().Get("start_time") != "" || r.URL.Query().Get("end_time") != "" {
			t.Fatal("expected no time query params")
		}
		writeJSON(w, map[string]interface{}{
			"pull_request_count": float64(10),
			"commit_count":       float64(42),
		})
	}))
	defer server.Close()

	err := runShortcut(t, server, "stats", map[string]string{"login": "alice"})
	if err != nil {
		t.Fatalf("stats failed: %v", err)
	}
}

func TestUserStatsWithTimeRange(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/users/alice/statistics/develop.json" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if r.URL.Query().Get("start_time") != "2026-01-01" {
			t.Fatalf("expected start_time=2026-01-01, got %s", r.URL.Query().Get("start_time"))
		}
		if r.URL.Query().Get("end_time") != "2026-03-31" {
			t.Fatalf("expected end_time=2026-03-31, got %s", r.URL.Query().Get("end_time"))
		}
		writeJSON(w, map[string]interface{}{
			"pull_request_count": float64(5),
			"commit_count":       float64(20),
		})
	}))
	defer server.Close()

	err := runShortcut(t, server, "stats", map[string]string{
		"login":      "alice",
		"start-time": "2026-01-01",
		"end-time":   "2026-03-31",
	})
	if err != nil {
		t.Fatalf("stats with time range failed: %v", err)
	}
}

func TestUserStatsMissingLogin(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("no API call expected")
	}))
	defer server.Close()

	err := runShortcut(t, server, "stats", map[string]string{})
	if err == nil {
		t.Fatal("expected error for missing login")
	}
}

func TestUserStatsHTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("server error"))
	}))
	defer server.Close()

	err := runShortcut(t, server, "stats", map[string]string{"login": "alice"})
	if err == nil {
		t.Fatal("expected error for HTTP 500")
	}
}

// --- trends ---

func TestUserTrends(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/users/alice/project_trends.json" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		writeJSON(w, []interface{}{
			map[string]interface{}{"id": float64(1), "name": "created project"},
		})
	}))
	defer server.Close()

	err := runShortcut(t, server, "trends", map[string]string{"login": "alice"})
	if err != nil {
		t.Fatalf("trends failed: %v", err)
	}
}

func TestUserTrendsMissingLogin(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("no API call expected")
	}))
	defer server.Close()

	err := runShortcut(t, server, "trends", map[string]string{})
	if err == nil {
		t.Fatal("expected error for missing login")
	}
}

func TestUserTrendsHTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("server error"))
	}))
	defer server.Close()

	err := runShortcut(t, server, "trends", map[string]string{"login": "alice"})
	if err == nil {
		t.Fatal("expected error for HTTP 500")
	}
}
