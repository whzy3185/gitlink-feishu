package profile

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
	_ = json.NewEncoder(w).Encode(v)
}

// --- ability / role / major share statRun ---

func TestProfileAbilityExplicitUser(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/users/alice/statistics/develop.json" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		writeJSON(w, map[string]interface{}{
			"user": map[string]interface{}{"contribution": float64(78)},
		})
	}))
	defer server.Close()

	if err := runShortcut(t, server, "ability", map[string]string{"user": "alice"}); err != nil {
		t.Fatalf("ability failed: %v", err)
	}
}

func TestProfileAbilityTimeWindow(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("start_time"); got != "100" {
			t.Fatalf("start_time = %q, want 100", got)
		}
		if got := r.URL.Query().Get("end_time"); got != "200" {
			t.Fatalf("end_time = %q, want 200", got)
		}
		writeJSON(w, map[string]interface{}{"user": map[string]interface{}{}})
	}))
	defer server.Close()

	args := map[string]string{"user": "alice", "start-time": "100", "end-time": "200"}
	if err := runShortcut(t, server, "ability", args); err != nil {
		t.Fatalf("ability with window failed: %v", err)
	}
}

func TestProfileMajor(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/users/bob/statistics/major.json" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		writeJSON(w, map[string]interface{}{"categories": []string{"深度学习"}})
	}))
	defer server.Close()

	if err := runShortcut(t, server, "major", map[string]string{"user": "bob"}); err != nil {
		t.Fatalf("major failed: %v", err)
	}
}

func TestProfileRole(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/users/bob/statistics/role.json" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		writeJSON(w, map[string]interface{}{})
	}))
	defer server.Close()

	if err := runShortcut(t, server, "role", map[string]string{"user": "bob"}); err != nil {
		t.Fatalf("role failed: %v", err)
	}
}

// --- activity ---

func TestProfileActivity(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/users/bob/statistics/activity.json" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		writeJSON(w, map[string]interface{}{"dates": []string{}})
	}))
	defer server.Close()

	if err := runShortcut(t, server, "activity", map[string]string{"user": "bob"}); err != nil {
		t.Fatalf("activity failed: %v", err)
	}
}

// --- contribution ---

func TestProfileContributionWithYear(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/users/bob/headmaps.json" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if got := r.URL.Query().Get("year"); got != "2025" {
			t.Fatalf("year = %q, want 2025", got)
		}
		writeJSON(w, map[string]interface{}{"total_contributions": float64(12)})
	}))
	defer server.Close()

	if err := runShortcut(t, server, "contribution", map[string]string{"user": "bob", "year": "2025"}); err != nil {
		t.Fatalf("contribution failed: %v", err)
	}
}

// --- resolveUser falls back to /users/me ---

func TestProfileDefaultsToCurrentUser(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/users/me.json":
			writeJSON(w, map[string]interface{}{"login": "currentuser"})
		case "/users/currentuser/statistics/develop.json":
			writeJSON(w, map[string]interface{}{"user": map[string]interface{}{}})
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	if err := runShortcut(t, server, "ability", nil); err != nil {
		t.Fatalf("ability (default user) failed: %v", err)
	}
}

func TestProfileDefaultUserMissingLogin(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/users/me.json" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		writeJSON(w, map[string]interface{}{"name": "no login here"})
	}))
	defer server.Close()

	if err := runShortcut(t, server, "ability", nil); err == nil {
		t.Fatal("expected error when /users/me has no login")
	}
}

// --- HTTP error path ---

func TestProfileHTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("server error"))
	}))
	defer server.Close()

	if err := runShortcut(t, server, "major", map[string]string{"user": "bob"}); err == nil {
		t.Fatal("expected error for HTTP 500")
	}
}
