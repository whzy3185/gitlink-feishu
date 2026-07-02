package pm

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gitlink-org/gitlink-cli/internal/client"
	"github.com/gitlink-org/gitlink-cli/internal/i18n"
	"github.com/gitlink-org/gitlink-cli/shortcuts/common"
)

func runShortcut(t *testing.T, server *httptest.Server, name string, args map[string]string) error {
	t.Helper()
	shortcut := findShortcut(t, name)
	ctx := &common.RuntimeContext{
		Client: &client.Client{HTTP: server.Client(), BaseURL: server.URL},
		Format: "json",
		Args:   args,
		Tr:     i18n.Default(),
	}
	return shortcut.Run(ctx)
}

func findShortcut(t *testing.T, name string) *common.Shortcut {
	t.Helper()
	for _, shortcut := range Shortcuts() {
		if shortcut.Name == name {
			return shortcut
		}
	}
	t.Fatalf("shortcut %q not found", name)
	return nil
}

func writeJSON(t *testing.T, w http.ResponseWriter, v interface{}) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(v); err != nil {
		t.Fatalf("write JSON: %v", err)
	}
}

func TestPMReadShortcuts(t *testing.T) {
	tests := []struct {
		name string
		path string
	}{
		{name: "dashboards", path: "/pm/dashboards.json"},
		{name: "sprint-issues", path: "/pm/sprint_issues.json"},
		{name: "weekly-issues", path: "/pm/weekly_issues.json"},
		{name: "issue-tags", path: "/pm/issue_tags.json"},
		{name: "pipelines", path: "/pm/pipelines.json"},
		{name: "action-runs", path: "/pm/action_runs.json"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodGet {
					t.Fatalf("method = %s, want GET", r.Method)
				}
				if r.URL.Path != tt.path {
					t.Fatalf("path = %s, want %s", r.URL.Path, tt.path)
				}
				if got := r.URL.Query().Get("project_id"); got != "123" {
					t.Fatalf("project_id = %q, want 123", got)
				}
				if got := r.URL.Query().Get("page"); got != "2" {
					t.Fatalf("page = %q, want 2", got)
				}
				if got := r.URL.Query().Get("limit"); got != "50" {
					t.Fatalf("limit = %q, want 50", got)
				}
				writeJSON(t, w, map[string]interface{}{"status": 0, "message": "success"})
			}))
			defer server.Close()

			err := runShortcut(t, server, tt.name, map[string]string{
				"project-id": "123",
				"page":       "2",
				"limit":      "50",
			})
			if err != nil {
				t.Fatalf("%s failed: %v", tt.name, err)
			}
		})
	}
}

func TestPMReadShortcutsDefaultPagination(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/pm/dashboards.json" {
			t.Fatalf("path = %s, want /pm/dashboards.json", r.URL.Path)
		}
		if got := r.URL.Query().Get("page"); got != "1" {
			t.Fatalf("page = %q, want 1", got)
		}
		if got := r.URL.Query().Get("limit"); got != "20" {
			t.Fatalf("limit = %q, want 20", got)
		}
		writeJSON(t, w, map[string]interface{}{"status": 0})
	}))
	defer server.Close()

	if err := runShortcut(t, server, "dashboards", map[string]string{"project-id": "123"}); err != nil {
		t.Fatalf("dashboards failed: %v", err)
	}
}

func TestPMReadShortcutRequiresProjectID(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatalf("missing project-id should fail before remote API: %s %s", r.Method, r.URL.Path)
	}))
	defer server.Close()

	if err := runShortcut(t, server, "dashboards", nil); err == nil {
		t.Fatal("expected missing project-id error")
	}
}

func TestPMReadShortcutHTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("server error"))
	}))
	defer server.Close()

	if err := runShortcut(t, server, "dashboards", map[string]string{"project-id": "123"}); err == nil {
		t.Fatal("expected error for HTTP 500")
	}
}
