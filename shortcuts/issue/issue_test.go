package issue

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

func decodeJSON(t *testing.T, r *http.Request) map[string]interface{} {
	t.Helper()
	var payload map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		t.Fatalf("failed to decode body: %v", err)
	}
	return payload
}

func assertEqual(t *testing.T, got interface{}, want interface{}) {
	t.Helper()
	if got != want {
		t.Fatalf("got %v (%T), want %v (%T)", got, got, want, want)
	}
}

// --- list ---

func TestIssueList(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Fatalf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/v1/owner/repo/issues.json" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if r.URL.Query().Get("state") != "open" {
			t.Fatalf("expected state=open, got %s", r.URL.Query().Get("state"))
		}
		writeJSON(w, []interface{}{
			map[string]interface{}{"id": float64(1), "subject": "bug"},
		})
	}))
	defer server.Close()

	err := runShortcut(t, server, "list", map[string]string{"state": "open", "page": "1", "limit": "20"})
	if err != nil {
		t.Fatalf("list failed: %v", err)
	}
}

// --- create ---

func TestIssueCreate(t *testing.T) {
	var payload map[string]interface{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Fatalf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/v1/owner/repo/issues.json" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		payload = decodeJSON(t, r)
		writeJSON(w, map[string]interface{}{"id": float64(1), "subject": "bug"})
	}))
	defer server.Close()

	err := runShortcut(t, server, "create", map[string]string{
		"title":    "bug: crash",
		"body":     "description",
		"assignee": "alice",
	})
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}
	assertEqual(t, payload["subject"], "bug: crash")
	assertEqual(t, payload["status_id"], float64(1))
	assertEqual(t, payload["assigned_to_id"], "alice")
}

func TestIssueCreateMissingTitle(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("no API call expected")
	}))
	defer server.Close()

	err := runShortcut(t, server, "create", map[string]string{})
	if err == nil {
		t.Fatal("expected error for missing title")
	}
}

// --- view ---

func TestIssueView(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Fatalf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/v1/owner/repo/issues/42.json" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		writeJSON(w, map[string]interface{}{"id": float64(42), "subject": "bug"})
	}))
	defer server.Close()

	err := runShortcut(t, server, "view", map[string]string{"number": "42"})
	if err != nil {
		t.Fatalf("view failed: %v", err)
	}
}

func TestIssueViewMissingNumber(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("no API call expected")
	}))
	defer server.Close()

	err := runShortcut(t, server, "view", map[string]string{})
	if err == nil {
		t.Fatal("expected error for missing number")
	}
}

// --- close ---

func TestIssueClose(t *testing.T) {
	var patchPayload map[string]interface{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == "GET" && r.URL.Path == "/v1/owner/repo/issues/42.json":
			writeJSON(w, map[string]interface{}{
				"id":          float64(42),
				"subject":     "Existing title",
				"description": "Existing description",
			})
		case r.Method == "PATCH" && r.URL.Path == "/v1/owner/repo/issues/42.json":
			patchPayload = decodeJSON(t, r)
			writeJSON(w, patchPayload)
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	err := runShortcut(t, server, "close", map[string]string{"number": "42"})
	if err != nil {
		t.Fatalf("close failed: %v", err)
	}
	assertEqual(t, patchPayload["subject"], "Existing title")
	assertEqual(t, patchPayload["description"], "Existing description")
	assertEqual(t, patchPayload["status_id"], float64(5))
}

func TestIssueCloseFetchFails(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		writeJSON(w, map[string]interface{}{"error": "not found"})
	}))
	defer server.Close()

	err := runShortcut(t, server, "close", map[string]string{"number": "999"})
	if err == nil {
		t.Fatal("expected error when issue not found")
	}
}

// --- update ---

func TestIssueUpdateTitle(t *testing.T) {
	var patchPayload map[string]interface{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == "GET" && r.URL.Path == "/v1/owner/repo/issues/42.json":
			writeJSON(w, map[string]interface{}{
				"id":          float64(42),
				"subject":     "Existing title",
				"description": "Existing description",
			})
		case r.Method == "PATCH" && r.URL.Path == "/v1/owner/repo/issues/42.json":
			patchPayload = decodeJSON(t, r)
			writeJSON(w, patchPayload)
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	err := runShortcut(t, server, "update", map[string]string{"number": "42", "title": "New title", "state": "closed"})
	if err != nil {
		t.Fatalf("update failed: %v", err)
	}
	assertEqual(t, patchPayload["subject"], "New title")
	assertEqual(t, patchPayload["description"], "Existing description")
	assertEqual(t, patchPayload["status_id"], float64(5))
}

func TestIssueUpdateDescription(t *testing.T) {
	var patchPayload map[string]interface{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == "GET" && r.URL.Path == "/v1/owner/repo/issues/42.json":
			writeJSON(w, map[string]interface{}{
				"id":          float64(42),
				"subject":     "Existing title",
				"description": "Existing description",
			})
		case r.Method == "PATCH" && r.URL.Path == "/v1/owner/repo/issues/42.json":
			patchPayload = decodeJSON(t, r)
			writeJSON(w, patchPayload)
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	err := runShortcut(t, server, "update", map[string]string{"number": "42", "body": "New description"})
	if err != nil {
		t.Fatalf("update failed: %v", err)
	}
	assertEqual(t, patchPayload["subject"], "Existing title")
	assertEqual(t, patchPayload["description"], "New description")
}

func TestIssueUpdateNumericState(t *testing.T) {
	var patchPayload map[string]interface{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == "GET" && r.URL.Path == "/v1/owner/repo/issues/42.json":
			writeJSON(w, map[string]interface{}{
				"id":          float64(42),
				"subject":     "bug",
				"description": "desc",
			})
		case r.Method == "PATCH" && r.URL.Path == "/v1/owner/repo/issues/42.json":
			patchPayload = decodeJSON(t, r)
			writeJSON(w, map[string]interface{}{"id": float64(42)})
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	err := runShortcut(t, server, "update", map[string]string{"number": "42", "state": "3"})
	if err != nil {
		t.Fatalf("update numeric state failed: %v", err)
	}
	assertEqual(t, patchPayload["status_id"], float64(3))
}

func TestIssueUpdateInvalidState(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == "GET" && r.URL.Path == "/v1/owner/repo/issues/42.json":
			writeJSON(w, map[string]interface{}{
				"id":          float64(42),
				"subject":     "bug",
				"description": "desc",
			})
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	err := runShortcut(t, server, "update", map[string]string{"number": "42", "state": "invalid"})
	if err == nil {
		t.Fatal("expected error for invalid state")
	}
}

func TestIssueUpdateNoChanges(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("no API call expected")
	}))
	defer server.Close()

	err := runShortcut(t, server, "update", map[string]string{"number": "42"})
	if err == nil {
		t.Fatal("expected error when no changes specified")
	}
}

// --- comment ---

func TestIssueComment(t *testing.T) {
	var payload map[string]interface{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Fatalf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/v1/owner/repo/issues/42/journals.json" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		payload = decodeJSON(t, r)
		writeJSON(w, map[string]interface{}{"id": float64(1), "message": "ok"})
	}))
	defer server.Close()

	err := runShortcut(t, server, "comment", map[string]string{"number": "42", "body": "test comment"})
	if err != nil {
		t.Fatalf("comment failed: %v", err)
	}
	assertEqual(t, payload["notes"], "test comment")
}

func TestIssueCommentMissingBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("no API call expected")
	}))
	defer server.Close()

	err := runShortcut(t, server, "comment", map[string]string{"number": "42"})
	if err == nil {
		t.Fatal("expected error for missing body")
	}
}

// --- batch-close ---

func TestBatchClosePreservesCurrentDescription(t *testing.T) {
	var updatePayload map[string]interface{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == "GET" && r.URL.Path == "/v1/owner/repo/issues/42.json":
			writeJSON(w, map[string]interface{}{
				"subject":     "Existing title",
				"description": "Existing description",
			})
		case r.Method == "PATCH" && r.URL.Path == "/v1/owner/repo/issues/42.json":
			updatePayload = decodeJSON(t, r)
			writeJSON(w, updatePayload)
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	err := runShortcut(t, server, "batch-close", map[string]string{
		"numbers": "42",
		"dry-run": "false",
	})
	if err != nil {
		t.Fatalf("batch-close shortcut failed: %v", err)
	}

	assertEqual(t, updatePayload["subject"], "Existing title")
	assertEqual(t, updatePayload["description"], "Existing description")
	assertEqual(t, updatePayload["status_id"], float64(5))
}

func TestBatchCloseDryRun(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("no API call expected in dry-run mode")
	}))
	defer server.Close()

	err := runShortcut(t, server, "batch-close", map[string]string{
		"numbers": "1, 2, 3",
		"dry-run": "true",
	})
	if err != nil {
		t.Fatalf("batch-close dry-run failed: %v", err)
	}
}

func TestBatchCloseNoNumbers(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("no API call expected")
	}))
	defer server.Close()

	err := runShortcut(t, server, "batch-close", map[string]string{})
	if err == nil {
		t.Fatal("expected error when no issue numbers provided")
	}
}

func TestBatchCloseFetchFails(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("not found"))
	}))
	defer server.Close()

	err := runShortcut(t, server, "batch-close", map[string]string{
		"numbers": "99",
	})
	if err == nil {
		t.Fatal("expected error when fetch fails")
	}
}

func TestBatchCloseWithFailedClose(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == "GET" && r.URL.Path == "/v1/owner/repo/issues/1.json":
			writeJSON(w, map[string]interface{}{"subject": "Issue 1", "description": "desc1"})
		case r.Method == "GET" && r.URL.Path == "/v1/owner/repo/issues/2.json":
			writeJSON(w, map[string]interface{}{"subject": "Issue 2", "description": "desc2"})
		case r.Method == "PATCH" && r.URL.Path == "/v1/owner/repo/issues/1.json":
			writeJSON(w, map[string]interface{}{"subject": "Issue 1", "description": "desc1", "status_id": float64(5)})
		case r.Method == "PATCH" && r.URL.Path == "/v1/owner/repo/issues/2.json":
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("server error"))
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	err := runShortcut(t, server, "batch-close", map[string]string{
		"numbers": "1, 2",
	})
	if err == nil {
		t.Fatal("expected error when some issues fail to close")
	}
}

// --- HTTP error paths ---

func TestIssueListHTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("server error"))
	}))
	defer server.Close()

	err := runShortcut(t, server, "list", map[string]string{"page": "1", "limit": "20"})
	if err == nil {
		t.Fatal("expected error for HTTP 500")
	}
}

func TestIssueCreateHTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("server error"))
	}))
	defer server.Close()

	err := runShortcut(t, server, "create", map[string]string{"title": "test"})
	if err == nil {
		t.Fatal("expected error for HTTP 500")
	}
}

func TestIssueViewHTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("server error"))
	}))
	defer server.Close()

	err := runShortcut(t, server, "view", map[string]string{"number": "42"})
	if err == nil {
		t.Fatal("expected error for HTTP 500")
	}
}

func TestIssueCommentHTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("server error"))
	}))
	defer server.Close()

	err := runShortcut(t, server, "comment", map[string]string{"number": "42", "body": "test"})
	if err == nil {
		t.Fatal("expected error for HTTP 500")
	}
}

func TestIssueUpdateHTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == "GET" && r.URL.Path == "/v1/owner/repo/issues/42.json":
			writeJSON(w, map[string]interface{}{
				"id": float64(42), "subject": "bug", "description": "desc",
			})
		case r.Method == "PATCH" && r.URL.Path == "/v1/owner/repo/issues/42.json":
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("server error"))
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	err := runShortcut(t, server, "update", map[string]string{"number": "42", "title": "new"})
	if err == nil {
		t.Fatal("expected error for PATCH HTTP 500")
	}
}

func TestIssueCloseHTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == "GET" && r.URL.Path == "/v1/owner/repo/issues/42.json":
			writeJSON(w, map[string]interface{}{
				"id": float64(42), "subject": "bug", "description": "desc",
			})
		case r.Method == "PATCH" && r.URL.Path == "/v1/owner/repo/issues/42.json":
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("server error"))
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	err := runShortcut(t, server, "close", map[string]string{"number": "42"})
	if err == nil {
		t.Fatal("expected error for PATCH HTTP 500")
	}
}

func TestFetchExistingIssueBadData(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, "not a map")
	}))
	defer server.Close()

	ctx := &common.RuntimeContext{
		Client: &client.Client{HTTP: server.Client(), BaseURL: server.URL},
		Owner:  "owner",
		Repo:   "repo",
	}
	_, err := fetchExistingIssue(ctx, "1")
	if err == nil {
		t.Fatal("expected error for non-map response")
	}
}

func TestFetchExistingIssueNoSubject(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, map[string]interface{}{"id": float64(1)})
	}))
	defer server.Close()

	ctx := &common.RuntimeContext{
		Client: &client.Client{HTTP: server.Client(), BaseURL: server.URL},
		Owner:  "owner",
		Repo:   "repo",
	}
	_, err := fetchExistingIssue(ctx, "1")
	if err == nil {
		t.Fatal("expected error for missing subject")
	}
}

// --- normalizeIssueStatus ---

func TestNormalizeIssueStatus(t *testing.T) {
	tests := []struct {
		input   string
		want    interface{}
		wantErr bool
	}{
		{"open", 1, false},
		{"OPEN", 1, false},
		{"  open  ", 1, false},
		{"closed", 5, false},
		{"CLOSED", 5, false},
		{"0", 0, false},
		{"10", 10, false},
		{"invalid", nil, true},
		{"", nil, true},
	}
	for _, tt := range tests {
		got, err := normalizeIssueStatus(tt.input)
		if tt.wantErr {
			if err == nil {
				t.Errorf("normalizeIssueStatus(%q) expected error", tt.input)
			}
		} else {
			if err != nil {
				t.Errorf("normalizeIssueStatus(%q) error: %v", tt.input, err)
			}
			if got != tt.want {
				t.Errorf("normalizeIssueStatus(%q) = %v, want %v", tt.input, got, tt.want)
			}
		}
	}
}
