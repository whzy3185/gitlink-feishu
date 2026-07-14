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

// --- heatmap ---

func TestUserHeatmapExplicitUserWithYear(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/users/alice/headmaps.json" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if got := r.URL.Query().Get("year"); got != "2026" {
			t.Fatalf("year = %q, want 2026", got)
		}
		writeJSON(w, map[string]interface{}{
			"total_contributions": float64(12),
		})
	}))
	defer server.Close()

	err := runShortcut(t, server, "heatmap", map[string]string{"user": "alice", "year": "2026"})
	if err != nil {
		t.Fatalf("heatmap failed: %v", err)
	}
}

func TestUserHeatmapDefaultsToCurrentUser(t *testing.T) {
	var calls []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls = append(calls, r.URL.Path)
		switch r.URL.Path {
		case "/users/me.json":
			writeJSON(w, map[string]interface{}{"login": "currentuser"})
		case "/users/currentuser/headmaps.json":
			writeJSON(w, map[string]interface{}{"headmaps": []interface{}{}})
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	err := runShortcut(t, server, "heatmap", map[string]string{})
	if err != nil {
		t.Fatalf("heatmap failed: %v", err)
	}
	if len(calls) != 2 {
		t.Fatalf("calls = %v, want 2 calls", calls)
	}
}

// --- statistics ---

func TestUserStatisticsWithTimeWindow(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/users/alice/statistics.json" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if got := r.URL.Query().Get("start_time"); got != "100" {
			t.Fatalf("start_time = %q, want 100", got)
		}
		if got := r.URL.Query().Get("end_time"); got != "200" {
			t.Fatalf("end_time = %q, want 200", got)
		}
		writeJSON(w, map[string]interface{}{
			"issues_count": float64(3),
		})
	}))
	defer server.Close()

	args := map[string]string{"user": "alice", "start-time": "100", "end-time": "200"}
	err := runShortcut(t, server, "statistics", args)
	if err != nil {
		t.Fatalf("statistics failed: %v", err)
	}
}

func TestUserStatsAlias(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/users/alice/statistics.json" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		writeJSON(w, map[string]interface{}{})
	}))
	defer server.Close()

	err := runShortcut(t, server, "stats", map[string]string{"user": "alice"})
	if err != nil {
		t.Fatalf("stats alias failed: %v", err)
	}
}

// --- project trends ---

func TestUserProjectTrendsWithTimeWindow(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/users/alice/project_trends.json" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if got := r.URL.Query().Get("start_time"); got != "100" {
			t.Fatalf("start_time = %q, want 100", got)
		}
		if got := r.URL.Query().Get("end_time"); got != "200" {
			t.Fatalf("end_time = %q, want 200", got)
		}
		writeJSON(w, map[string]interface{}{
			"trends": []interface{}{},
		})
	}))
	defer server.Close()

	args := map[string]string{"user": "alice", "start-time": "100", "end-time": "200"}
	err := runShortcut(t, server, "project-trends", args)
	if err != nil {
		t.Fatalf("project-trends failed: %v", err)
	}
}

func TestUserTrendsAlias(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/users/alice/project_trends.json" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		writeJSON(w, map[string]interface{}{})
	}))
	defer server.Close()

	err := runShortcut(t, server, "trends", map[string]string{"user": "alice"})
	if err != nil {
		t.Fatalf("trends alias failed: %v", err)
	}
}

func TestUserDefaultUserMissingLogin(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/users/me.json" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		writeJSON(w, map[string]interface{}{"name": "no login"})
	}))
	defer server.Close()

	err := runShortcut(t, server, "statistics", map[string]string{})
	if err == nil {
		t.Fatal("expected error when /users/me has no login")
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

func TestUserHeatmapHTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("server error"))
	}))
	defer server.Close()

	err := runShortcut(t, server, "heatmap", map[string]string{"user": "alice"})
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
