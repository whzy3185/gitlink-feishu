package trace

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gitlink-org/gitlink-cli/internal/client"
	"github.com/gitlink-org/gitlink-cli/shortcuts/common"
)

func runTraceShortcut(t *testing.T, server *httptest.Server, name string, args map[string]string) error {
	t.Helper()
	s := findTraceShortcut(t, name)
	ctx := &common.RuntimeContext{
		Client: &client.Client{
			HTTP:    server.Client(),
			BaseURL: server.URL,
		},
		Owner:  "owner",
		Repo:   "repo",
		Format: "json",
		Args:   args,
	}
	if ctx.Args == nil {
		ctx.Args = map[string]string{}
	}
	return s.Run(ctx)
}

func findTraceShortcut(t *testing.T, name string) *common.Shortcut {
	t.Helper()
	for _, s := range Shortcuts() {
		if s.Name == name {
			return s
		}
	}
	t.Fatalf("shortcut %q not found", name)
	return nil
}

func writeJSON(t *testing.T, w http.ResponseWriter, value interface{}) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(value); err != nil {
		t.Fatalf("write response: %v", err)
	}
}

func decodeJSON(t *testing.T, r *http.Request) map[string]interface{} {
	t.Helper()
	var payload map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		t.Fatalf("decode request: %v", err)
	}
	return payload
}

func TestTraceInit(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Fatalf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/api/traces/trace_users.json" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		writeJSON(t, w, map[string]interface{}{"status": float64(0), "message": "success"})
	}))
	defer server.Close()

	if err := runTraceShortcut(t, server, "init", nil); err != nil {
		t.Fatalf("init failed: %v", err)
	}
}

func TestTraceResults(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Fatalf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/api/traces/owner/repo/task_results.json" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if r.URL.Query().Get("page") != "2" {
			t.Fatalf("expected page=2, got %s", r.URL.Query().Get("page"))
		}
		if r.URL.Query().Get("limit") != "50" {
			t.Fatalf("expected limit=50, got %s", r.URL.Query().Get("limit"))
		}
		writeJSON(t, w, map[string]interface{}{"data": []interface{}{map[string]interface{}{"task_id": "task-1"}}})
	}))
	defer server.Close()

	err := runTraceShortcut(t, server, "results", map[string]string{"page": "2", "limit": "50"})
	if err != nil {
		t.Fatalf("results failed: %v", err)
	}
}

func TestTraceStart(t *testing.T) {
	var payload map[string]interface{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Fatalf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/api/traces/owner/repo/tasks.json" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		payload = decodeJSON(t, r)
		writeJSON(t, w, map[string]interface{}{"status": float64(0), "message": "success"})
	}))
	defer server.Close()

	err := runTraceShortcut(t, server, "start", map[string]string{"branch": " feature/scan "})
	if err != nil {
		t.Fatalf("start failed: %v", err)
	}
	if payload["branch_name"] != "feature/scan" {
		t.Fatalf("unexpected branch_name: %#v", payload["branch_name"])
	}
}

func TestTraceStartDryRunDoesNotCallAPI(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatalf("dry-run should not call API: %s %s", r.Method, r.URL.Path)
	}))
	defer server.Close()

	err := runTraceShortcut(t, server, "start", map[string]string{"branch": "main", "dry-run": "true"})
	if err != nil {
		t.Fatalf("start dry-run failed: %v", err)
	}
}

func TestTraceStartRejectsBlankBranch(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatalf("blank branch should not call API")
	}))
	defer server.Close()

	err := runTraceShortcut(t, server, "start", map[string]string{"branch": "   "})
	if err == nil {
		t.Fatal("expected blank branch error, got nil")
	}
}

func TestTraceRescan(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Fatalf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/api/traces/owner/repo/reload_task.json" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if r.URL.Query().Get("project_id") != "project-1" {
			t.Fatalf("unexpected project_id: %s", r.URL.Query().Get("project_id"))
		}
		writeJSON(t, w, map[string]interface{}{"status": float64(0), "message": "success"})
	}))
	defer server.Close()

	err := runTraceShortcut(t, server, "rescan", map[string]string{"project-id": " project-1 "})
	if err != nil {
		t.Fatalf("rescan failed: %v", err)
	}
}

func TestTraceRescanDryRunDoesNotCallAPI(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatalf("dry-run should not call API: %s %s", r.Method, r.URL.Path)
	}))
	defer server.Close()

	err := runTraceShortcut(t, server, "rescan", map[string]string{"project-id": "project-1", "dry-run": "true"})
	if err != nil {
		t.Fatalf("rescan dry-run failed: %v", err)
	}
}

func TestTraceReport(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Fatalf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/api/traces/owner/repo/task_pdf.json" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if r.URL.Query().Get("task_id") != "task-1" {
			t.Fatalf("unexpected task_id: %s", r.URL.Query().Get("task_id"))
		}
		writeJSON(t, w, map[string]interface{}{"status": float64(0), "message": "success"})
	}))
	defer server.Close()

	err := runTraceShortcut(t, server, "report", map[string]string{"task-id": "task-1"})
	if err != nil {
		t.Fatalf("report failed: %v", err)
	}
}
