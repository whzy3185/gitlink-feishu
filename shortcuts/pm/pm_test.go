package pm

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gitlink-org/gitlink-cli/internal/client"
	"github.com/gitlink-org/gitlink-cli/shortcuts/common"
)

func TestPmDashboards(t *testing.T) {
	tests := []struct {
		name        string
		mockStatus  int
		mockBody    string
		wantErr     bool
		errContains string
	}{
		{"正常返回", 200, `{"dashboards": []}`, false, ""},
		{"API 404", 404, `{"error": "not found"}`, true, "404"},
		{"返回 HTML", 200, `<!DOCTYPE html><html><body>Login</body></html>`, true, "HTML"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != "GET" {
					t.Errorf("expected GET, got %s", r.Method)
				}
				if !strings.Contains(r.URL.Path, "/pm/dashboards") {
					t.Errorf("expected path containing /pm/dashboards, got %s", r.URL.Path)
				}
				if got := r.URL.Query().Get("project_id"); got != "123" {
					t.Errorf("expected project_id=123, got %s", got)
				}
				w.WriteHeader(tt.mockStatus)
				w.Write([]byte(tt.mockBody))
			}))
			defer server.Close()

			shortcut := findPmShortcut(t, "dashboards")
			ctx := &common.RuntimeContext{
				Client: &client.Client{HTTP: server.Client(), BaseURL: server.URL},
				Owner:  "test", Repo: "test", Format: "json",
				Args: map[string]string{"project": "123"},
			}
			err := shortcut.Run(ctx)

			if tt.wantErr && err == nil {
				t.Fatal("期望错误但为 nil")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("不期望错误: %v", err)
			}
			if tt.wantErr && tt.errContains != "" && err != nil {
				if !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("错误应包含 %q: %s", tt.errContains, err.Error())
				}
			}
		})
	}
}

func TestPmSprints(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if !strings.Contains(r.URL.Path, "/pm/sprint_issues") {
			t.Errorf("expected path containing /pm/sprint_issues, got %s", r.URL.Path)
		}
		w.WriteHeader(200)
		w.Write([]byte(`{"sprint_issues": []}`))
	}))
	defer server.Close()

	shortcut := findPmShortcut(t, "sprints")
	ctx := &common.RuntimeContext{
		Client: &client.Client{HTTP: server.Client(), BaseURL: server.URL},
		Owner:  "test", Repo: "test", Format: "json",
		Args: map[string]string{"project": "456"},
	}
	if err := shortcut.Run(ctx); err != nil {
		t.Fatalf("sprints shortcut failed: %v", err)
	}
}

func TestPmWeekly(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, "/pm/weekly_issues") {
			t.Errorf("expected path containing /pm/weekly_issues, got %s", r.URL.Path)
		}
		w.WriteHeader(200)
		w.Write([]byte(`{"weekly_issues": []}`))
	}))
	defer server.Close()

	shortcut := findPmShortcut(t, "weekly")
	ctx := &common.RuntimeContext{
		Client: &client.Client{HTTP: server.Client(), BaseURL: server.URL},
		Owner:  "test", Repo: "test", Format: "json",
		Args: map[string]string{"project": "789"},
	}
	if err := shortcut.Run(ctx); err != nil {
		t.Fatalf("weekly shortcut failed: %v", err)
	}
}

func TestPmTags(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, "/pm/issue_tags") {
			t.Errorf("expected path containing /pm/issue_tags, got %s", r.URL.Path)
		}
		w.WriteHeader(200)
		w.Write([]byte(`{"issue_tags": []}`))
	}))
	defer server.Close()

	shortcut := findPmShortcut(t, "tags")
	ctx := &common.RuntimeContext{
		Client: &client.Client{HTTP: server.Client(), BaseURL: server.URL},
		Owner:  "test", Repo: "test", Format: "json",
		Args: map[string]string{"project": "100"},
	}
	if err := shortcut.Run(ctx); err != nil {
		t.Fatalf("tags shortcut failed: %v", err)
	}
}

func TestPmPipelines(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, "/pm/pipelines") {
			t.Errorf("expected path containing /pm/pipelines, got %s", r.URL.Path)
		}
		w.WriteHeader(200)
		w.Write([]byte(`{"pipelines": []}`))
	}))
	defer server.Close()

	shortcut := findPmShortcut(t, "pipelines")
	ctx := &common.RuntimeContext{
		Client: &client.Client{HTTP: server.Client(), BaseURL: server.URL},
		Owner:  "test", Repo: "test", Format: "json",
		Args: map[string]string{"project": "200"},
	}
	if err := shortcut.Run(ctx); err != nil {
		t.Fatalf("pipelines shortcut failed: %v", err)
	}
}

func TestPmRuns(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, "/pm/action_runs") {
			t.Errorf("expected path containing /pm/action_runs, got %s", r.URL.Path)
		}
		w.WriteHeader(200)
		w.Write([]byte(`{"action_runs": []}`))
	}))
	defer server.Close()

	shortcut := findPmShortcut(t, "runs")
	ctx := &common.RuntimeContext{
		Client: &client.Client{HTTP: server.Client(), BaseURL: server.URL},
		Owner:  "test", Repo: "test", Format: "json",
		Args: map[string]string{"project": "300"},
	}
	if err := shortcut.Run(ctx); err != nil {
		t.Fatalf("runs shortcut failed: %v", err)
	}
}

func TestPmMissingProject(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("should not call API when --project is missing")
	}))
	defer server.Close()

	shortcut := findPmShortcut(t, "dashboards")
	ctx := &common.RuntimeContext{
		Client: &client.Client{HTTP: server.Client(), BaseURL: server.URL},
		Owner:  "test", Repo: "test", Format: "json",
		Args: map[string]string{},
	}
	err := shortcut.Run(ctx)
	if err == nil {
		t.Fatal("expected error when --project is missing")
	}
}

func findPmShortcut(t *testing.T, name string) *common.Shortcut {
	t.Helper()
	for _, shortcut := range Shortcuts() {
		if shortcut.Name == name {
			return shortcut
		}
	}
	t.Fatalf("shortcut %q not found", name)
	return nil
}

func assertPmRequest(t *testing.T, r *http.Request, method, pathPrefix string) {
	t.Helper()
	if r.Method != method {
		t.Fatalf("got method %s, want %s", r.Method, method)
	}
	if !strings.HasPrefix(r.URL.Path, pathPrefix) {
		t.Fatalf("got path %s, want prefix %s", r.URL.Path, pathPrefix)
	}
}

func decodePmJSON(t *testing.T, r *http.Request) map[string]interface{} {
	t.Helper()
	var payload map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		t.Fatalf("failed to decode request body: %v", err)
	}
	return payload
}
