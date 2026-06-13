package user

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gitlink-org/gitlink-cli/internal/client"
	"github.com/gitlink-org/gitlink-cli/shortcuts/common"
)

func runShortcut(t *testing.T, server *httptest.Server, name string, args map[string]string) error {
	t.Helper()
	shortcut := findShortcut(t, name)
	ctx := &common.RuntimeContext{
		Client: &client.Client{HTTP: server.Client(), BaseURL: server.URL},
		Owner:  "owner",
		Repo:   "repo",
		Format: "json",
		Args:   args,
	}
	return shortcut.Run(ctx)
}

func findShortcut(t *testing.T, name string) *common.Shortcut {
	t.Helper()
	for _, s := range Shortcuts() {
		if s.Name == name {
			return s
		}
	}
	t.Fatalf("shortcut %q not found", name)
	return nil
}

func writeJSON(w http.ResponseWriter, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v)
}

// --- me ---

func TestUserMe(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/users/me.json" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		writeJSON(w, map[string]interface{}{
			"login": "currentuser",
			"name":  "Current User",
			"id":    float64(1),
		})
	}))
	defer server.Close()

	err := runShortcut(t, server, "me", nil)
	if err != nil {
		t.Fatalf("me failed: %v", err)
	}
}

// --- info ---

func TestUserInfo(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/users/alice.json" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		writeJSON(w, map[string]interface{}{
			"login": "alice",
			"name":  "Alice",
		})
	}))
	defer server.Close()

	err := runShortcut(t, server, "info", map[string]string{"login": "alice"})
	if err != nil {
		t.Fatalf("info failed: %v", err)
	}
}

func TestUserInfoMissingLogin(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("no API call expected")
	}))
	defer server.Close()

	err := runShortcut(t, server, "info", map[string]string{})
	if err == nil {
		t.Fatal("expected error for missing login")
	}
}

// --- HTTP error paths ---

func TestUserMeHTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("server error"))
	}))
	defer server.Close()

	err := runShortcut(t, server, "me", nil)
	if err == nil {
		t.Fatal("expected error for HTTP 500")
	}
}

func TestUserInfoHTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("server error"))
	}))
	defer server.Close()

	err := runShortcut(t, server, "info", map[string]string{"login": "alice"})
	if err == nil {
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
