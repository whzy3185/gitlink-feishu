package action

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
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

func writeJSON(t *testing.T, w http.ResponseWriter, v interface{}) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(v); err != nil {
		t.Fatalf("write response: %v", err)
	}
}

func TestActionListUsesActionsEndpoint(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" || r.URL.Path != "/v1/owner/repo/actions.json" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		writeJSON(t, w, map[string]interface{}{"total_count": 1, "files": []interface{}{}})
	}))
	defer server.Close()

	if err := runShortcut(t, server, "list", map[string]string{}); err != nil {
		t.Fatalf("list failed: %v", err)
	}
}

func TestActionRunsRequiresWorkflowAndForwardsQuery(t *testing.T) {
	var query string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" || r.URL.Path != "/v1/owner/repo/actions/runs.json" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		query = r.URL.RawQuery
		writeJSON(t, w, map[string]interface{}{"total_data": 0, "runs": []interface{}{}})
	}))
	defer server.Close()

	if err := runShortcut(t, server, "runs", map[string]string{"workflow": "ci.yml", "page": "1", "limit": "20"}); err != nil {
		t.Fatalf("runs failed: %v", err)
	}
	if !strings.Contains(query, "workflow=ci.yml") {
		t.Fatalf("expected workflow in query, got %q", query)
	}

	if err := runShortcut(t, server, "runs", map[string]string{}); err == nil {
		t.Fatal("expected error when --workflow missing")
	}
}

func TestActionRunPostsWorkflowAndRef(t *testing.T) {
	var query string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" || r.URL.Path != "/v1/owner/repo/actions/runs.json" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		query = r.URL.RawQuery
		writeJSON(t, w, map[string]interface{}{"status": float64(0)})
	}))
	defer server.Close()

	if err := runShortcut(t, server, "run", map[string]string{"workflow": "ci.yml", "ref": "master"}); err != nil {
		t.Fatalf("run failed: %v", err)
	}
	if !strings.Contains(query, "workflow=ci.yml") || !strings.Contains(query, "ref=master") {
		t.Fatalf("unexpected query: %q", query)
	}
}

func TestActionRerunValidatesRunID(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" || r.URL.Path != "/v1/owner/repo/actions/runs/7/rerun.json" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		writeJSON(t, w, map[string]interface{}{"status": float64(0)})
	}))
	defer server.Close()

	if err := runShortcut(t, server, "rerun", map[string]string{"run-id": "7"}); err != nil {
		t.Fatalf("rerun failed: %v", err)
	}
	if err := runShortcut(t, server, "rerun", map[string]string{"run-id": "abc"}); err == nil {
		t.Fatal("expected error for non-integer --run-id")
	}
}

func TestActionJobRerunBuildsNestedPath(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" || r.URL.Path != "/v1/owner/repo/actions/runs/7/jobs/build/rerun.json" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		writeJSON(t, w, map[string]interface{}{"status": float64(0)})
	}))
	defer server.Close()

	if err := runShortcut(t, server, "job-rerun", map[string]string{"run-id": "7", "job": "build"}); err != nil {
		t.Fatalf("job-rerun failed: %v", err)
	}
}

func TestActionEnableDisableEndpoints(t *testing.T) {
	var paths []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Fatalf("expected POST, got %s", r.Method)
		}
		paths = append(paths, r.URL.Path)
		writeJSON(t, w, map[string]interface{}{"status": float64(0)})
	}))
	defer server.Close()

	if err := runShortcut(t, server, "enable", map[string]string{"workflow": "ci.yml"}); err != nil {
		t.Fatalf("enable failed: %v", err)
	}
	if err := runShortcut(t, server, "disable", map[string]string{"workflow": "ci.yml"}); err != nil {
		t.Fatalf("disable failed: %v", err)
	}
	if err := runShortcut(t, server, "disable", map[string]string{}); err == nil {
		t.Fatal("expected error when --workflow missing")
	}
	want := []string{"/v1/owner/repo/actions/enable.json", "/v1/owner/repo/actions/disable.json"}
	for i, p := range want {
		if paths[i] != p {
			t.Fatalf("expected %s, got %s", p, paths[i])
		}
	}
}

func TestActionLogsBuildsNestedLogsPath(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" || r.URL.Path != "/v1/owner/repo/actions/runs/6/jobs/0/logs.json" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		w.Header().Set("Content-Type", "text/plain")
		fmt.Fprint(w, "log line 1\nlog line 2\n")
	}))
	defer server.Close()

	if err := runShortcut(t, server, "logs", map[string]string{"run-id": "6", "job": "0"}); err != nil {
		t.Fatalf("logs failed: %v", err)
	}

	if err := runShortcut(t, server, "logs", map[string]string{"run-id": "abc", "job": "0"}); err == nil {
		t.Fatal("expected error for non-integer --run-id")
	}
}
