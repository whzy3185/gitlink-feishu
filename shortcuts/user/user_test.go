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
