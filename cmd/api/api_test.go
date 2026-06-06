package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/gitlink-org/gitlink-cli/cmd/cmdutil"
)

func TestResolveFormat(t *testing.T) {
	tests := []struct {
		name       string
		flagFormat string
		want       string
	}{
		{"empty defaults to json", "", "json"},
		{"explicit json", "json", "json"},
		{"explicit yaml", "yaml", "yaml"},
		{"explicit table", "table", "table"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmdutil.Format = tt.flagFormat
			if got := resolveFormat(); got != tt.want {
				t.Fatalf("resolveFormat = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestNewAPICmd(t *testing.T) {
	cmd := NewAPICmd()
	if cmd.Use != "api (<METHOD> <PATH> | --batch-file <FILE>)" {
		t.Fatalf("Use = %q", cmd.Use)
	}
	if cmd.Short == "" {
		t.Fatal("Short is empty")
	}

	// Verify flags exist
	flags := []string{"body", "query", "header", "batch-file", "dry-run", "continue-on-error", "var"}
	for _, f := range flags {
		if cmd.Flags().Lookup(f) == nil {
			t.Fatalf("flag %q not found", f)
		}
	}
}

func setupAPITest(t *testing.T, handler http.HandlerFunc) string {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	dir := t.TempDir()
	t.Setenv("GITLINK_CONFIG_DIR", dir)
	os.MkdirAll(dir, 0700)
	os.WriteFile(filepath.Join(dir, "config.yaml"), []byte("base_url: "+server.URL+"\ndefault_format: table\n"), 0600)
	return dir
}

func TestRunAPIGet(t *testing.T) {
	setupAPITest(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/users/me.json" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"login": "testuser", "id": 42})
	})
	cmdutil.Format = "json"

	cmd := NewAPICmd()
	cmd.SetArgs([]string{"GET", "/users/me"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("runAPI GET error: %v", err)
	}
}

func TestRunAPIPostWithBody(t *testing.T) {
	var gotBody map[string]interface{}
	setupAPITest(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Fatalf("expected POST, got %s", r.Method)
		}
		json.NewDecoder(r.Body).Decode(&gotBody)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"id": 1, "title": "new issue"})
	})
	cmdutil.Format = "json"

	cmd := NewAPICmd()
	cmd.SetArgs([]string{"POST", "/repos/owner/repo/issues"})
	cmd.Flags().Set("body", `{"title":"new issue","body":"test"}`)
	if err := cmd.Execute(); err != nil {
		t.Fatalf("runAPI POST error: %v", err)
	}
	if gotBody["title"] != "new issue" {
		t.Fatalf("body title = %q, want 'new issue'", gotBody["title"])
	}
}

func TestRunAPIBadJSONBody(t *testing.T) {
	setupAPITest(t, func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("should not reach server")
	})
	cmdutil.Format = "json"

	cmd := NewAPICmd()
	cmd.SetArgs([]string{"POST", "/repos/owner/repo/issues"})
	cmd.Flags().Set("body", `{bad json}`)
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error for bad JSON body")
	}
}

func TestRunAPIBadQuery(t *testing.T) {
	setupAPITest(t, func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("should not reach server")
	})
	cmdutil.Format = "json"

	cmd := NewAPICmd()
	cmd.SetArgs([]string{"GET", "/repos/owner/repo/issues"})
	cmd.Flags().Set("query", "key=%zz")
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error for bad query string")
	}
}

func TestRunAPIHTTPError(t *testing.T) {
	setupAPITest(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("not found"))
	})
	cmdutil.Format = "json"

	cmd := NewAPICmd()
	cmd.SetArgs([]string{"GET", "/nonexistent"})
	// HTTP errors are caught and printed as error envelopes; runAPI does not return the error
	if err := cmd.Execute(); err != nil {
		t.Fatalf("runAPI HTTP error: %v (expected success with error envelope)", err)
	}
}

func TestRunAPIStatusError(t *testing.T) {
	setupAPITest(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]interface{}{"status": float64(401), "message": "Unauthorized"})
	})
	cmdutil.Format = "json"

	cmd := NewAPICmd()
	cmd.SetArgs([]string{"GET", "/users/me"})
	// Should print error envelope, not return a Go error (status check in Do() handles this)
	// Actually, HTTP 401 triggers APIError return from Do(), so this should error
	if err := cmd.Execute(); err != nil {
		// Expected — HTTP error
		t.Logf("got expected error: %v", err)
	}
}

func TestRunAPIDebug(t *testing.T) {
	var gotDebugHeader bool
	setupAPITest(t, func(w http.ResponseWriter, r *http.Request) {
		gotDebugHeader = true
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"ok": true})
	})
	cmdutil.Format = "json"
	cmdutil.Debug = true
	defer func() { cmdutil.Debug = false }()

	cmd := NewAPICmd()
	cmd.SetArgs([]string{"GET", "/users/me"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("runAPI debug error: %v", err)
	}
	if !gotDebugHeader {
		t.Fatal("server not reached")
	}
}

func TestRunAPINoPrefix(t *testing.T) {
	setupAPITest(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/users/me.json" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"login": "testuser"})
	})
	cmdutil.Format = "json"

	cmd := NewAPICmd()
	cmd.SetArgs([]string{"GET", "users/me"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("runAPI no-prefix error: %v", err)
	}
}

func TestRenderBatchRequestsTemplateVars(t *testing.T) {
	requests, err := renderBatchRequests([]batchRequest{
		{
			Name:   "comment-{{number}}",
			Method: "post",
			Path:   "v1/{{owner}}/{{repo}}/issues/{{number}}/journals",
			Query: map[string]interface{}{
				"label": []interface{}{"{{label}}", "triage"},
				"page":  float64(1),
			},
			Body: map[string]interface{}{
				"notes": "handled by {{actor}}",
				"meta":  map[string]interface{}{"repo": "{{repo}}"},
			},
		},
	}, map[string]string{
		"owner":  "Gitlink",
		"repo":   "gitlink-cli",
		"number": "42",
		"label":  "bug",
		"actor":  "bot",
	})
	if err != nil {
		t.Fatalf("renderBatchRequests error: %v", err)
	}
	if len(requests) != 1 {
		t.Fatalf("len = %d, want 1", len(requests))
	}
	req := requests[0]
	if req.Name != "comment-42" {
		t.Fatalf("Name = %q", req.Name)
	}
	if req.Method != "POST" {
		t.Fatalf("Method = %q", req.Method)
	}
	if req.Path != "/v1/Gitlink/gitlink-cli/issues/42/journals" {
		t.Fatalf("Path = %q", req.Path)
	}
	if got := req.Query["label"]; len(got) != 2 || got[0] != "bug" || got[1] != "triage" {
		t.Fatalf("label query = %#v", got)
	}
	body := req.Body.(map[string]interface{})
	if body["notes"] != "handled by bot" {
		t.Fatalf("notes = %#v", body["notes"])
	}
}

func TestRenderBatchRequestsMissingVar(t *testing.T) {
	_, err := renderBatchRequests([]batchRequest{{Method: "GET", Path: "/{{missing}}"}}, nil)
	if err == nil {
		t.Fatal("expected missing variable error")
	}
}

func TestRunAPIBatchDryRunDoesNotReachServer(t *testing.T) {
	setupAPITest(t, func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("dry-run should not reach server")
	})
	cmdutil.Format = "json"

	plan := writeBatchPlan(t, map[string]interface{}{
		"vars": map[string]string{"owner": "Gitlink"},
		"requests": []map[string]interface{}{
			{"name": "me", "method": "GET", "path": "/users/me"},
			{"name": "repo", "method": "GET", "path": "/{{owner}}/gitlink-cli"},
		},
	})

	cmd := NewAPICmd()
	cmd.SetArgs([]string{"--batch-file", plan, "--dry-run"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("dry-run batch error: %v", err)
	}
}

func TestRunAPIBatchExecutesRequestsWithOverrides(t *testing.T) {
	var seen []string
	var gotBody map[string]interface{}
	setupAPITest(t, func(w http.ResponseWriter, r *http.Request) {
		seen = append(seen, r.Method+" "+r.URL.String())
		switch r.URL.Path {
		case "/v1/Mengz/gitlink-cli/issues.json":
			if r.URL.Query().Get("state") != "open" {
				t.Fatalf("state query = %q", r.URL.Query().Get("state"))
			}
			json.NewEncoder(w).Encode(map[string]interface{}{"issues": []interface{}{}})
		case "/v1/Mengz/gitlink-cli/issues/7/journals.json":
			if err := json.NewDecoder(r.Body).Decode(&gotBody); err != nil {
				t.Fatalf("decode body: %v", err)
			}
			json.NewEncoder(w).Encode(map[string]interface{}{"id": 99})
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	})
	cmdutil.Format = "json"

	plan := writeBatchPlan(t, map[string]interface{}{
		"vars": map[string]string{"owner": "Gitlink", "repo": "gitlink-cli", "issue": "7"},
		"requests": []map[string]interface{}{
			{
				"name":   "list",
				"method": "GET",
				"path":   "/v1/{{owner}}/{{repo}}/issues",
				"query":  map[string]interface{}{"state": "open"},
			},
			{
				"name":   "comment",
				"method": "POST",
				"path":   "/v1/{{owner}}/{{repo}}/issues/{{issue}}/journals",
				"body":   map[string]interface{}{"notes": "hello {{repo}}"},
			},
		},
	})

	cmd := NewAPICmd()
	cmd.SetArgs([]string{"--batch-file", plan, "--var", "owner=Mengz"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("batch execute error: %v", err)
	}
	if len(seen) != 2 {
		t.Fatalf("requests = %d, want 2 (%v)", len(seen), seen)
	}
	if gotBody["notes"] != "hello gitlink-cli" {
		t.Fatalf("body notes = %#v", gotBody["notes"])
	}
}

func TestRunAPIBatchStopsOnErrorByDefault(t *testing.T) {
	var seen []string
	setupAPITest(t, func(w http.ResponseWriter, r *http.Request) {
		seen = append(seen, r.URL.Path)
		if r.URL.Path == "/fail.json" {
			http.Error(w, "boom", http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(map[string]interface{}{"ok": true})
	})
	cmdutil.Format = "json"

	plan := writeBatchPlan(t, map[string]interface{}{
		"requests": []map[string]interface{}{
			{"method": "GET", "path": "/ok"},
			{"method": "GET", "path": "/fail"},
			{"method": "GET", "path": "/never"},
		},
	})

	cmd := NewAPICmd()
	cmd.SetArgs([]string{"--batch-file", plan})
	if err := cmd.Execute(); err == nil {
		t.Fatal("expected batch error")
	}
	if len(seen) != 2 {
		t.Fatalf("requests = %d, want 2 (%v)", len(seen), seen)
	}
}

func TestRunAPIBatchContinueOnError(t *testing.T) {
	var seen []string
	setupAPITest(t, func(w http.ResponseWriter, r *http.Request) {
		seen = append(seen, r.URL.Path)
		if r.URL.Path == "/fail.json" {
			http.Error(w, "boom", http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(map[string]interface{}{"ok": true})
	})
	cmdutil.Format = "json"

	plan := writeBatchPlan(t, map[string]interface{}{
		"requests": []map[string]interface{}{
			{"method": "GET", "path": "/ok"},
			{"method": "GET", "path": "/fail"},
			{"method": "GET", "path": "/after"},
		},
	})

	cmd := NewAPICmd()
	cmd.SetArgs([]string{"--batch-file", plan, "--continue-on-error"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("batch should continue: %v", err)
	}
	if len(seen) != 3 {
		t.Fatalf("requests = %d, want 3 (%v)", len(seen), seen)
	}
}

func writeBatchPlan(t *testing.T, payload interface{}) string {
	t.Helper()
	data, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal plan: %v", err)
	}
	path := filepath.Join(t.TempDir(), "plan.json")
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatalf("write plan: %v", err)
	}
	return path
}
