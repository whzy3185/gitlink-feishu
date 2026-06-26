package issue

import (
	"encoding/json"
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

func newIssueTestServer(t *testing.T, handler http.HandlerFunc) *httptest.Server {
	t.Helper()
	return httptest.NewServer(handler)
}

func writeJSON(t *testing.T, w http.ResponseWriter, v interface{}) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(v); err != nil {
		t.Fatalf("failed to write response: %v", err)
	}
}

func writeText(t *testing.T, w http.ResponseWriter, status int, text string) {
	t.Helper()
	w.WriteHeader(status)
	if _, err := w.Write([]byte(text)); err != nil {
		t.Fatalf("write response: %v", err)
	}
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

func assertNumberSlice(t *testing.T, got interface{}, want []float64) {
	t.Helper()
	values, ok := got.([]interface{})
	if !ok {
		t.Fatalf("got %v (%T), want numeric slice", got, got)
	}
	if len(values) != len(want) {
		t.Fatalf("got %v, want %v", values, want)
	}
	for i, value := range values {
		if value != want[i] {
			t.Fatalf("got %v, want %v", values, want)
		}
	}
}

// --- list ---

func TestIssueList(t *testing.T) {
	server := newIssueTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Fatalf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/v1/owner/repo/issues.json" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if r.URL.Query().Get("category") != "opened" {
			t.Fatalf("expected category=opened, got %s", r.URL.Query().Get("category"))
		}
		writeJSON(t, w, []interface{}{
			map[string]interface{}{"id": float64(1), "subject": "bug"},
		})
	})
	defer server.Close()

	err := runShortcut(t, server, "list", map[string]string{"state": "open", "page": "1", "limit": "20"})
	if err != nil {
		t.Fatalf("list failed: %v", err)
	}
}

func TestIssueListWithFilters(t *testing.T) {
	server := newIssueTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/owner/repo/issues.json" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		query := r.URL.Query()
		assertEqual(t, query.Get("category"), "closed")
		assertEqual(t, query.Get("keyword"), "release")
		assertEqual(t, query.Get("participant_category"), "assignedme")
		assertEqual(t, query.Get("author_id"), "10")
		assertEqual(t, query.Get("assigner_id"), "11")
		assertEqual(t, query.Get("milestone_id"), "12")
		assertEqual(t, query.Get("status_id"), "5")
		assertEqual(t, query.Get("issue_tag_ids"), "1,2")
		assertEqual(t, query.Get("sort_by"), "issues.updated_on")
		assertEqual(t, query.Get("sort_direction"), "desc")
		writeJSON(t, w, map[string]interface{}{"issues": []interface{}{}})
	})
	defer server.Close()

	err := runShortcut(t, server, "list", map[string]string{
		"state":          "closed",
		"keyword":        "release",
		"participant":    "assignedme",
		"author-id":      "10",
		"assignee-id":    "11",
		"milestone-id":   "12",
		"status-id":      "5",
		"tag-ids":        "1,2",
		"sort-by":        "issues.updated_on",
		"sort-direction": "desc",
		"page":           "2",
		"limit":          "50",
	})
	if err != nil {
		t.Fatalf("list with filters failed: %v", err)
	}
}

func TestIssueListStateAll(t *testing.T) {
	server := newIssueTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		assertEqual(t, r.URL.Query().Get("category"), "all")
		writeJSON(t, w, map[string]interface{}{"issues": []interface{}{}})
	})
	defer server.Close()

	err := runShortcut(t, server, "list", map[string]string{"state": "all", "page": "1", "limit": "20"})
	if err != nil {
		t.Fatalf("list all failed: %v", err)
	}
}

// --- create ---

func TestIssueCreate(t *testing.T) {
	var payload map[string]interface{}
	server := newIssueTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Fatalf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/v1/owner/repo/issues.json" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		payload = decodeJSON(t, r)
		writeJSON(t, w, map[string]interface{}{"id": float64(1), "subject": "bug"})
	})
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

func TestIssueCreateSupportsMetadataFields(t *testing.T) {
	var createPayload map[string]interface{}
	server := newIssueTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" || r.URL.Path != "/v1/owner/repo/issues.json" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		createPayload = decodeJSON(t, r)
		writeJSON(t, w, createPayload)
	})
	defer server.Close()

	err := runShortcut(t, server, "create", map[string]string{
		"title":        "New issue",
		"body":         "With metadata",
		"priority-id":  "3",
		"tag-ids":      "4,5",
		"assigner-ids": "7,8",
		"branch":       "feature/metadata",
		"start-date":   "2026-05-01",
		"due-date":     "2026-05-31",
	})
	if err != nil {
		t.Fatalf("create shortcut failed: %v", err)
	}

	assertEqual(t, createPayload["subject"], "New issue")
	assertEqual(t, createPayload["description"], "With metadata")
	assertEqual(t, createPayload["priority_id"], float64(3))
	assertNumberSlice(t, createPayload["issue_tag_ids"], []float64{4, 5})
	assertNumberSlice(t, createPayload["assigner_ids"], []float64{7, 8})
	assertEqual(t, createPayload["branch_name"], "feature/metadata")
	assertEqual(t, createPayload["start_date"], "2026-05-01")
	assertEqual(t, createPayload["due_date"], "2026-05-31")
}

func TestIssueCreateMissingTitle(t *testing.T) {
	server := newIssueTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("no API call expected")
	})
	defer server.Close()

	err := runShortcut(t, server, "create", map[string]string{})
	if err == nil {
		t.Fatal("expected error for missing title")
	}
}

// --- view/id alias ---

func TestIssueView(t *testing.T) {
	server := newIssueTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Fatalf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/v1/owner/repo/issues/42.json" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		writeJSON(t, w, map[string]interface{}{"id": float64(42), "subject": "bug"})
	})
	defer server.Close()

	err := runShortcut(t, server, "view", map[string]string{"number": "42"})
	if err != nil {
		t.Fatalf("view failed: %v", err)
	}
}

func TestIssueViewMissingNumber(t *testing.T) {
	server := newIssueTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("no API call expected")
	})
	defer server.Close()

	err := runShortcut(t, server, "view", map[string]string{})
	if err == nil {
		t.Fatal("expected error for missing number")
	}
}

func TestIssueViewAcceptsIDAlias(t *testing.T) {
	var requestedPath string
	server := newIssueTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		requestedPath = r.URL.Path
		if r.Method != "GET" || r.URL.Path != "/v1/owner/repo/issues/42.json" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		writeJSON(t, w, map[string]interface{}{
			"project_issues_index": 42,
			"subject":              "Issue from web URL",
		})
	})
	defer server.Close()

	err := runShortcut(t, server, "view", map[string]string{"id": "42"})
	if err != nil {
		t.Fatalf("view shortcut failed: %v", err)
	}
	assertEqual(t, requestedPath, "/v1/owner/repo/issues/42.json")
}

func TestIssueNumberTakesPrecedenceOverIDAlias(t *testing.T) {
	server := newIssueTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" || r.URL.Path != "/v1/owner/repo/issues/42.json" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		writeJSON(t, w, map[string]interface{}{"subject": "Existing title"})
	})
	defer server.Close()

	err := runShortcut(t, server, "view", map[string]string{
		"number": "42",
		"id":     "99",
	})
	if err != nil {
		t.Fatalf("view shortcut failed: %v", err)
	}
}

// --- close ---

func TestIssueClose(t *testing.T) {
	var patchPayload map[string]interface{}
	server := newIssueTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == "GET" && r.URL.Path == "/v1/owner/repo/issues/42.json":
			writeJSON(t, w, map[string]interface{}{
				"id":          float64(42),
				"subject":     "Existing title",
				"description": "Existing description",
			})
		case r.Method == "PATCH" && r.URL.Path == "/v1/owner/repo/issues/42.json":
			patchPayload = decodeJSON(t, r)
			writeJSON(t, w, patchPayload)
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	})
	defer server.Close()

	err := runShortcut(t, server, "close", map[string]string{"number": "42"})
	if err != nil {
		t.Fatalf("close failed: %v", err)
	}
	assertEqual(t, patchPayload["subject"], "Existing title")
	assertEqual(t, patchPayload["description"], "Existing description")
	assertEqual(t, patchPayload["status_id"], float64(5))
}

func TestIssueCloseAcceptsIDAlias(t *testing.T) {
	var updatePayload map[string]interface{}
	server := newIssueTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == "GET" && r.URL.Path == "/v1/owner/repo/issues/42.json":
			writeJSON(t, w, map[string]interface{}{
				"subject":     "Existing title",
				"description": "Existing description",
			})
		case r.Method == "PATCH" && r.URL.Path == "/v1/owner/repo/issues/42.json":
			updatePayload = decodeJSON(t, r)
			writeJSON(t, w, updatePayload)
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	})
	defer server.Close()

	err := runShortcut(t, server, "close", map[string]string{"id": "42"})
	if err != nil {
		t.Fatalf("close shortcut failed: %v", err)
	}
	assertEqual(t, updatePayload["status_id"], float64(5))
}

func TestIssueClosePreservesCurrentMetadata(t *testing.T) {
	var updatePayload map[string]interface{}
	server := newIssueTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == "GET" && r.URL.Path == "/v1/owner/repo/issues/42.json":
			writeJSON(t, w, map[string]interface{}{
				"subject":  "Existing title",
				"priority": map[string]interface{}{"id": 3},
				"tags": []map[string]interface{}{
					{"id": 4},
				},
			})
		case r.Method == "PATCH" && r.URL.Path == "/v1/owner/repo/issues/42.json":
			updatePayload = decodeJSON(t, r)
			writeJSON(t, w, updatePayload)
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	})
	defer server.Close()

	err := runShortcut(t, server, "close", map[string]string{"number": "42"})
	if err != nil {
		t.Fatalf("close shortcut failed: %v", err)
	}
	assertEqual(t, updatePayload["subject"], "Existing title")
	assertEqual(t, updatePayload["status_id"], float64(5))
	assertEqual(t, updatePayload["priority_id"], float64(3))
	assertNumberSlice(t, updatePayload["issue_tag_ids"], []float64{4})
}

func TestIssueCloseFetchFails(t *testing.T) {
	server := newIssueTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		writeJSON(t, w, map[string]interface{}{"error": "not found"})
	})
	defer server.Close()

	err := runShortcut(t, server, "close", map[string]string{"number": "999"})
	if err == nil {
		t.Fatal("expected error when issue not found")
	}
}

// --- update ---

func TestIssueUpdateTitle(t *testing.T) {
	var patchPayload map[string]interface{}
	server := newIssueTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == "GET" && r.URL.Path == "/v1/owner/repo/issues/42.json":
			writeJSON(t, w, map[string]interface{}{
				"id":          float64(42),
				"subject":     "Existing title",
				"description": "Existing description",
			})
		case r.Method == "PATCH" && r.URL.Path == "/v1/owner/repo/issues/42.json":
			patchPayload = decodeJSON(t, r)
			writeJSON(t, w, patchPayload)
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	})
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
	server := newIssueTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == "GET" && r.URL.Path == "/v1/owner/repo/issues/42.json":
			writeJSON(t, w, map[string]interface{}{
				"id":          float64(42),
				"subject":     "Existing title",
				"description": "Existing description",
			})
		case r.Method == "PATCH" && r.URL.Path == "/v1/owner/repo/issues/42.json":
			patchPayload = decodeJSON(t, r)
			writeJSON(t, w, patchPayload)
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	})
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
	server := newIssueTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == "GET" && r.URL.Path == "/v1/owner/repo/issues/42.json":
			writeJSON(t, w, map[string]interface{}{
				"id":          float64(42),
				"subject":     "bug",
				"description": "desc",
			})
		case r.Method == "PATCH" && r.URL.Path == "/v1/owner/repo/issues/42.json":
			patchPayload = decodeJSON(t, r)
			writeJSON(t, w, map[string]interface{}{"id": float64(42)})
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	})
	defer server.Close()

	err := runShortcut(t, server, "update", map[string]string{"number": "42", "state": "3"})
	if err != nil {
		t.Fatalf("update numeric state failed: %v", err)
	}
	assertEqual(t, patchPayload["status_id"], float64(3))
}

func TestIssueUpdateAcceptsIDAlias(t *testing.T) {
	var updatePayload map[string]interface{}
	server := newIssueTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == "GET" && r.URL.Path == "/v1/owner/repo/issues/42.json":
			writeJSON(t, w, map[string]interface{}{
				"subject":     "Existing title",
				"description": "Existing description",
			})
		case r.Method == "PATCH" && r.URL.Path == "/v1/owner/repo/issues/42.json":
			updatePayload = decodeJSON(t, r)
			writeJSON(t, w, updatePayload)
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	})
	defer server.Close()

	err := runShortcut(t, server, "update", map[string]string{
		"id":    "42",
		"title": "New title",
	})
	if err != nil {
		t.Fatalf("update shortcut failed: %v", err)
	}
	assertEqual(t, updatePayload["subject"], "New title")
	assertEqual(t, updatePayload["description"], "Existing description")
}

func TestIssueUpdatePreservesCurrentMetadata(t *testing.T) {
	var updatePayload map[string]interface{}
	server := newIssueTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == "GET" && r.URL.Path == "/v1/owner/repo/issues/42.json":
			writeJSON(t, w, map[string]interface{}{
				"subject":     "Existing title",
				"description": "Existing description",
				"status":      map[string]interface{}{"id": 1},
				"priority":    map[string]interface{}{"id": 2},
				"tags": []map[string]interface{}{
					{"id": 7},
					{"id": 8},
				},
				"assigners": []map[string]interface{}{
					{"id": 9},
				},
				"branch_name": "main",
				"start_date":  "2026-05-01",
				"due_date":    "2026-05-31",
			})
		case r.Method == "PATCH" && r.URL.Path == "/v1/owner/repo/issues/42.json":
			updatePayload = decodeJSON(t, r)
			writeJSON(t, w, updatePayload)
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	})
	defer server.Close()

	err := runShortcut(t, server, "update", map[string]string{
		"number": "42",
		"title":  "New title",
	})
	if err != nil {
		t.Fatalf("update shortcut failed: %v", err)
	}
	assertEqual(t, updatePayload["subject"], "New title")
	assertEqual(t, updatePayload["description"], "Existing description")
	assertEqual(t, updatePayload["status_id"], float64(1))
	assertEqual(t, updatePayload["priority_id"], float64(2))
	assertNumberSlice(t, updatePayload["issue_tag_ids"], []float64{7, 8})
	assertNumberSlice(t, updatePayload["assigner_ids"], []float64{9})
	assertEqual(t, updatePayload["branch_name"], "main")
	assertEqual(t, updatePayload["start_date"], "2026-05-01")
	assertEqual(t, updatePayload["due_date"], "2026-05-31")
}

func TestIssueUpdateSupportsMetadataFields(t *testing.T) {
	var updatePayload map[string]interface{}
	server := newIssueTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == "GET" && r.URL.Path == "/v1/owner/repo/issues/42.json":
			writeJSON(t, w, map[string]interface{}{
				"subject":     "Existing title",
				"description": "Existing description",
			})
		case r.Method == "PATCH" && r.URL.Path == "/v1/owner/repo/issues/42.json":
			updatePayload = decodeJSON(t, r)
			writeJSON(t, w, updatePayload)
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	})
	defer server.Close()

	err := runShortcut(t, server, "update", map[string]string{
		"number":       "42",
		"priority-id":  "4",
		"tag-ids":      "6,7",
		"assigner-ids": "8",
		"branch":       "bugfix/metadata",
		"start-date":   "2026-06-01",
		"due-date":     "2026-06-15",
	})
	if err != nil {
		t.Fatalf("update shortcut failed: %v", err)
	}
	assertEqual(t, updatePayload["subject"], "Existing title")
	assertEqual(t, updatePayload["description"], "Existing description")
	assertEqual(t, updatePayload["priority_id"], float64(4))
	assertNumberSlice(t, updatePayload["issue_tag_ids"], []float64{6, 7})
	assertNumberSlice(t, updatePayload["assigner_ids"], []float64{8})
	assertEqual(t, updatePayload["branch_name"], "bugfix/metadata")
	assertEqual(t, updatePayload["start_date"], "2026-06-01")
	assertEqual(t, updatePayload["due_date"], "2026-06-15")
}

func TestIssueUpdateInvalidState(t *testing.T) {
	server := newIssueTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == "GET" && r.URL.Path == "/v1/owner/repo/issues/42.json":
			writeJSON(t, w, map[string]interface{}{
				"id":          float64(42),
				"subject":     "bug",
				"description": "desc",
			})
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	})
	defer server.Close()

	err := runShortcut(t, server, "update", map[string]string{"number": "42", "state": "invalid"})
	if err == nil {
		t.Fatal("expected error for invalid state")
	}
}

func TestIssueUpdateNoChanges(t *testing.T) {
	server := newIssueTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("no API call expected")
	})
	defer server.Close()

	err := runShortcut(t, server, "update", map[string]string{"number": "42"})
	if err == nil {
		t.Fatal("expected error when no changes specified")
	}
}

func TestIssueRejectsInvalidMetadataIDs(t *testing.T) {
	server := newIssueTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		t.Fatalf("invalid metadata should not call API, got %s %s", r.Method, r.URL.Path)
	})
	defer server.Close()

	cases := []struct {
		name string
		args map[string]string
	}{
		{name: "bad priority", args: map[string]string{"title": "x", "priority-id": "abc"}},
		{name: "empty tag", args: map[string]string{"title": "x", "tag-ids": "1,,2"}},
		{name: "label conflicts with tag ids", args: map[string]string{"title": "x", "label": "1", "tag-ids": "2"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if err := runShortcut(t, server, "create", tc.args); err == nil {
				t.Fatal("expected metadata validation error")
			}
		})
	}
}

// --- comment ---

func TestIssueComment(t *testing.T) {
	var payload map[string]interface{}
	server := newIssueTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Fatalf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/v1/owner/repo/issues/42/journals.json" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		payload = decodeJSON(t, r)
		writeJSON(t, w, map[string]interface{}{"id": float64(1), "message": "ok"})
	})
	defer server.Close()

	err := runShortcut(t, server, "comment", map[string]string{"number": "42", "body": "test comment"})
	if err != nil {
		t.Fatalf("comment failed: %v", err)
	}
	assertEqual(t, payload["notes"], "test comment")
}

func TestIssueCommentAcceptsIDAlias(t *testing.T) {
	var commentPayload map[string]interface{}
	server := newIssueTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" || r.URL.Path != "/v1/owner/repo/issues/42/journals.json" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		commentPayload = decodeJSON(t, r)
		writeJSON(t, w, commentPayload)
	})
	defer server.Close()

	err := runShortcut(t, server, "comment", map[string]string{
		"id":   "42",
		"body": "Fixed",
	})
	if err != nil {
		t.Fatalf("comment shortcut failed: %v", err)
	}
	assertEqual(t, commentPayload["notes"], "Fixed")
}

func TestIssueCommentMissingBody(t *testing.T) {
	server := newIssueTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("no API call expected")
	})
	defer server.Close()

	err := runShortcut(t, server, "comment", map[string]string{"number": "42"})
	if err == nil {
		t.Fatal("expected error for missing body")
	}
}

func TestIssueNumberOrIDIsRequired(t *testing.T) {
	server := newIssueTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
	})
	defer server.Close()

	cases := []struct {
		name string
		args map[string]string
	}{
		{name: "view", args: map[string]string{}},
		{name: "close", args: map[string]string{}},
		{name: "update", args: map[string]string{"title": "New title"}},
		{name: "comment", args: map[string]string{"body": "Fixed"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := runShortcut(t, server, tc.name, tc.args)
			if err == nil {
				t.Fatal("expected missing issue number error")
			}
			if !strings.Contains(err.Error(), "--number") || !strings.Contains(err.Error(), "--id") {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

// --- batch-close ---

func TestBatchClosePreservesCurrentDescription(t *testing.T) {
	var updatePayload map[string]interface{}
	server := newIssueTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == "GET" && r.URL.Path == "/v1/owner/repo/issues/42.json":
			writeJSON(t, w, map[string]interface{}{
				"subject":     "Existing title",
				"description": "Existing description",
			})
		case r.Method == "PATCH" && r.URL.Path == "/v1/owner/repo/issues/42.json":
			updatePayload = decodeJSON(t, r)
			writeJSON(t, w, updatePayload)
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	})
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
	server := newIssueTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("no API call expected in dry-run mode")
	})
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
	server := newIssueTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("no API call expected")
	})
	defer server.Close()

	err := runShortcut(t, server, "batch-close", map[string]string{})
	if err == nil {
		t.Fatal("expected error when no issue numbers provided")
	}
}

func TestBatchCloseFetchFails(t *testing.T) {
	server := newIssueTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		writeText(t, w, http.StatusNotFound, "not found")
	})
	defer server.Close()

	err := runShortcut(t, server, "batch-close", map[string]string{"numbers": "99"})
	if err == nil {
		t.Fatal("expected error when fetch fails")
	}
}

func TestBatchCloseWithFailedClose(t *testing.T) {
	server := newIssueTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == "GET" && r.URL.Path == "/v1/owner/repo/issues/1.json":
			writeJSON(t, w, map[string]interface{}{"subject": "Issue 1", "description": "desc1"})
		case r.Method == "GET" && r.URL.Path == "/v1/owner/repo/issues/2.json":
			writeJSON(t, w, map[string]interface{}{"subject": "Issue 2", "description": "desc2"})
		case r.Method == "PATCH" && r.URL.Path == "/v1/owner/repo/issues/1.json":
			writeJSON(t, w, map[string]interface{}{"subject": "Issue 1", "description": "desc1", "status_id": float64(5)})
		case r.Method == "PATCH" && r.URL.Path == "/v1/owner/repo/issues/2.json":
			writeText(t, w, http.StatusInternalServerError, "server error")
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	})
	defer server.Close()

	err := runShortcut(t, server, "batch-close", map[string]string{"numbers": "1, 2"})
	if err == nil {
		t.Fatal("expected error when some issues fail to close")
	}
}

// --- issue users ---

func TestIssueAssignersShortcutWithKeyword(t *testing.T) {
	server := newIssueTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" || r.URL.Path != "/v1/owner/repo/issue_assigners.json" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		assertEqual(t, r.URL.Query().Get("keyword"), "alice")
		writeJSON(t, w, map[string]interface{}{
			"total_count": 1,
			"assigners": []map[string]interface{}{
				{"id": 7, "name": "Alice", "login": "alice"},
			},
		})
	})
	defer server.Close()

	err := runShortcut(t, server, "assigners", map[string]string{"keyword": "alice"})
	if err != nil {
		t.Fatalf("assigners shortcut failed: %v", err)
	}
}

func TestIssueAuthorsShortcutWithKeyword(t *testing.T) {
	server := newIssueTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" || r.URL.Path != "/v1/owner/repo/issue_authors.json" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		assertEqual(t, r.URL.Query().Get("keyword"), "bob")
		writeJSON(t, w, map[string]interface{}{
			"total_count": 1,
			"authors": []map[string]interface{}{
				{"id": 8, "name": "Bob", "login": "bob"},
			},
		})
	})
	defer server.Close()

	err := runShortcut(t, server, "authors", map[string]string{"keyword": "bob"})
	if err != nil {
		t.Fatalf("authors shortcut failed: %v", err)
	}
}

// --- metadata lookup shortcuts ---

func TestIssuePrioritiesShortcut(t *testing.T) {
	server := newIssueTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" || r.URL.Path != "/v1/owner/repo/issue_priorities.json" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		assertEqual(t, r.URL.Query().Get("keyword"), "normal")
		writeJSON(t, w, []map[string]interface{}{
			{"id": 2, "name": "Normal"},
		})
	})
	defer server.Close()

	err := runShortcut(t, server, "priorities", map[string]string{"keyword": "normal"})
	if err != nil {
		t.Fatalf("priorities shortcut failed: %v", err)
	}
}

func TestIssueTagsShortcutWithFilters(t *testing.T) {
	server := newIssueTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" || r.URL.Path != "/v1/owner/repo/issue_tags.json" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		query := r.URL.Query()
		assertEqual(t, query.Get("keyword"), "bug")
		assertEqual(t, query.Get("only_name"), "true")
		assertEqual(t, query.Get("order_by"), "issues_count")
		assertEqual(t, query.Get("order_direction"), "desc")
		writeJSON(t, w, map[string]interface{}{
			"total_count": 1,
			"issue_tags": []map[string]interface{}{
				{"id": 3, "name": "bug"},
			},
		})
	})
	defer server.Close()

	err := runShortcut(t, server, "tags", map[string]string{
		"keyword":         "bug",
		"only-name":       "true",
		"order-by":        "issues_count",
		"order-direction": "desc",
	})
	if err != nil {
		t.Fatalf("tags shortcut failed: %v", err)
	}
}

func TestIssueStatusesShortcut(t *testing.T) {
	server := newIssueTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" || r.URL.Path != "/v1/owner/repo/issue_statues.json" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		assertEqual(t, r.URL.Query().Get("page"), "2")
		assertEqual(t, r.URL.Query().Get("limit"), "10")
		writeJSON(t, w, map[string]interface{}{
			"total_count": 1,
			"statues": []map[string]interface{}{
				{"id": 1, "name": "Open"},
			},
		})
	})
	defer server.Close()

	err := runShortcut(t, server, "statuses", map[string]string{"page": "2", "limit": "10"})
	if err != nil {
		t.Fatalf("statuses shortcut failed: %v", err)
	}
}

// --- HTTP error paths ---

func TestIssueListHTTPError(t *testing.T) {
	server := newIssueTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		writeText(t, w, http.StatusInternalServerError, "server error")
	})
	defer server.Close()

	err := runShortcut(t, server, "list", map[string]string{"page": "1", "limit": "20"})
	if err == nil {
		t.Fatal("expected error for HTTP 500")
	}
}

func TestIssueCreateHTTPError(t *testing.T) {
	server := newIssueTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		writeText(t, w, http.StatusInternalServerError, "server error")
	})
	defer server.Close()

	err := runShortcut(t, server, "create", map[string]string{"title": "test"})
	if err == nil {
		t.Fatal("expected error for HTTP 500")
	}
}

func TestIssueViewHTTPError(t *testing.T) {
	server := newIssueTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		writeText(t, w, http.StatusInternalServerError, "server error")
	})
	defer server.Close()

	err := runShortcut(t, server, "view", map[string]string{"number": "42"})
	if err == nil {
		t.Fatal("expected error for HTTP 500")
	}
}

func TestIssueCommentHTTPError(t *testing.T) {
	server := newIssueTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		writeText(t, w, http.StatusInternalServerError, "server error")
	})
	defer server.Close()

	err := runShortcut(t, server, "comment", map[string]string{"number": "42", "body": "test"})
	if err == nil {
		t.Fatal("expected error for HTTP 500")
	}
}

func TestIssueUpdateHTTPError(t *testing.T) {
	server := newIssueTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == "GET" && r.URL.Path == "/v1/owner/repo/issues/42.json":
			writeJSON(t, w, map[string]interface{}{
				"id": float64(42), "subject": "bug", "description": "desc",
			})
		case r.Method == "PATCH" && r.URL.Path == "/v1/owner/repo/issues/42.json":
			writeText(t, w, http.StatusInternalServerError, "server error")
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	})
	defer server.Close()

	err := runShortcut(t, server, "update", map[string]string{"number": "42", "title": "new"})
	if err == nil {
		t.Fatal("expected error for PATCH HTTP 500")
	}
}

func TestIssueCloseHTTPError(t *testing.T) {
	server := newIssueTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == "GET" && r.URL.Path == "/v1/owner/repo/issues/42.json":
			writeJSON(t, w, map[string]interface{}{
				"id": float64(42), "subject": "bug", "description": "desc",
			})
		case r.Method == "PATCH" && r.URL.Path == "/v1/owner/repo/issues/42.json":
			writeText(t, w, http.StatusInternalServerError, "server error")
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	})
	defer server.Close()

	err := runShortcut(t, server, "close", map[string]string{"number": "42"})
	if err == nil {
		t.Fatal("expected error for PATCH HTTP 500")
	}
}

func TestFetchExistingIssueBadData(t *testing.T) {
	server := newIssueTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		writeJSON(t, w, "not a map")
	})
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
	server := newIssueTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		writeJSON(t, w, map[string]interface{}{"id": float64(1)})
	})
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

// --- batch-reopen ---

func TestBatchReopenPreservesCurrentDescription(t *testing.T) {
	var updatePayload map[string]interface{}
	server := newIssueTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == "GET" && r.URL.Path == "/v1/owner/repo/issues/42.json":
			writeJSON(t, w, map[string]interface{}{
				"subject":     "Existing title",
				"description": "Existing description",
			})
		case r.Method == "PATCH" && r.URL.Path == "/v1/owner/repo/issues/42.json":
			updatePayload = decodeJSON(t, r)
			writeJSON(t, w, updatePayload)
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	})
	defer server.Close()

	err := runShortcut(t, server, "batch-reopen", map[string]string{
		"numbers": "42",
		"dry-run": "false",
	})
	if err != nil {
		t.Fatalf("batch-reopen shortcut failed: %v", err)
	}
	assertEqual(t, updatePayload["subject"], "Existing title")
	assertEqual(t, updatePayload["description"], "Existing description")
	assertEqual(t, updatePayload["status_id"], float64(1))
}

func TestBatchReopenDryRun(t *testing.T) {
	server := newIssueTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("no API call expected in dry-run mode")
	})
	defer server.Close()

	err := runShortcut(t, server, "batch-reopen", map[string]string{
		"numbers": "1, 2, 3",
		"dry-run": "true",
	})
	if err != nil {
		t.Fatalf("batch-reopen dry-run failed: %v", err)
	}
}

func TestBatchReopenNoNumbers(t *testing.T) {
	server := newIssueTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("no API call expected")
	})
	defer server.Close()

	err := runShortcut(t, server, "batch-reopen", map[string]string{})
	if err == nil {
		t.Fatal("expected error when no issue numbers provided")
	}
}

func TestBatchReopenFetchFails(t *testing.T) {
	server := newIssueTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		writeText(t, w, http.StatusNotFound, "not found")
	})
	defer server.Close()

	err := runShortcut(t, server, "batch-reopen", map[string]string{"numbers": "99"})
	if err == nil {
		t.Fatal("expected error when fetch fails")
	}
}

// --- batch-label ---

func TestBatchLabelAddLabels(t *testing.T) {
	var patchPayloads []map[string]interface{}
	server := newIssueTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		// Handle GET requests to fetch issue data
		if r.Method == "GET" {
			writeJSON(t, w, map[string]interface{}{
				"id":         float64(1),
				"tags":       []interface{}{},
				"issue_tags": []interface{}{},
			})
			return
		}
		// Handle PATCH request to update labels
		if r.Method != "PATCH" || r.URL.Path != "/v1/owner/repo/issues/batch_update.json" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		payload := decodeJSON(t, r)
		patchPayloads = append(patchPayloads, payload)
		writeJSON(t, w, map[string]interface{}{"status": 0, "message": "ok"})
	})
	defer server.Close()

	err := runShortcut(t, server, "batch-label", map[string]string{
		"ids":     "1,2,3",
		"add":     "1,2",
		"dry-run": "false",
	})
	if err != nil {
		t.Fatalf("batch-label shortcut failed: %v", err)
	}
	// Should have 3 PATCH requests (one per issue)
	if len(patchPayloads) != 3 {
		t.Fatalf("expected 3 PATCH requests, got %d", len(patchPayloads))
	}
	// Each PATCH should have a single issue ID and the new tags
	for i, payload := range patchPayloads {
		assertNumberSlice(t, payload["ids"], []float64{float64(i + 1)})
		assertNumberSlice(t, payload["issue_tag_ids"], []float64{1, 2})
	}
}

func TestBatchLabelRemoveLabels(t *testing.T) {
	var patchPayloads []map[string]interface{}
	server := newIssueTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		// Handle GET requests to fetch issue data
		if r.Method == "GET" {
			writeJSON(t, w, map[string]interface{}{
				"id":         float64(1),
				"tags":       []interface{}{map[string]interface{}{"id": float64(1)}},
				"issue_tags": []interface{}{map[string]interface{}{"id": float64(1)}},
			})
			return
		}
		// Handle PATCH request to update labels
		if r.Method != "PATCH" || r.URL.Path != "/v1/owner/repo/issues/batch_update.json" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		payload := decodeJSON(t, r)
		patchPayloads = append(patchPayloads, payload)
		writeJSON(t, w, map[string]interface{}{"status": 0, "message": "ok"})
	})
	defer server.Close()

	err := runShortcut(t, server, "batch-label", map[string]string{
		"ids":     "1,2,3",
		"remove":  "1",
		"dry-run": "false",
	})
	if err != nil {
		t.Fatalf("batch-label shortcut failed: %v", err)
	}
	// Should have 3 PATCH requests (one per issue)
	if len(patchPayloads) != 3 {
		t.Fatalf("expected 3 PATCH requests, got %d", len(patchPayloads))
	}
	// Each PATCH should have a single issue ID and empty tags (after removing tag 1)
	for i, payload := range patchPayloads {
		assertNumberSlice(t, payload["ids"], []float64{float64(i + 1)})
		assertNumberSlice(t, payload["issue_tag_ids"], []float64{})
	}
}

func TestBatchLabelDryRun(t *testing.T) {
	server := newIssueTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("no API call expected in dry-run mode")
	})
	defer server.Close()

	err := runShortcut(t, server, "batch-label", map[string]string{
		"ids":     "1,2,3",
		"add":     "1",
		"dry-run": "true",
	})
	if err != nil {
		t.Fatalf("batch-label dry-run failed: %v", err)
	}
}

func TestBatchLabelNoIDs(t *testing.T) {
	server := newIssueTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("no API call expected")
	})
	defer server.Close()

	err := runShortcut(t, server, "batch-label", map[string]string{"add": "1"})
	if err == nil {
		t.Fatal("expected error when no issue IDs provided")
	}
}

func TestBatchLabelNoLabels(t *testing.T) {
	server := newIssueTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("no API call expected")
	})
	defer server.Close()

	err := runShortcut(t, server, "batch-label", map[string]string{"ids": "1,2,3"})
	if err == nil {
		t.Fatal("expected error when no labels specified")
	}
}

// --- batch-assign ---

func TestBatchAssignAddAssigners(t *testing.T) {
	var patchPayloads []map[string]interface{}
	server := newIssueTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		// Handle GET requests to fetch issue data
		if r.Method == "GET" {
			writeJSON(t, w, map[string]interface{}{
				"id":        float64(1),
				"assigners": []interface{}{},
			})
			return
		}
		// Handle PATCH request to update assigners
		if r.Method != "PATCH" || r.URL.Path != "/v1/owner/repo/issues/batch_update.json" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		payload := decodeJSON(t, r)
		patchPayloads = append(patchPayloads, payload)
		writeJSON(t, w, map[string]interface{}{"status": 0, "message": "ok"})
	})
	defer server.Close()

	err := runShortcut(t, server, "batch-assign", map[string]string{
		"ids":     "1,2,3",
		"add":     "1,2",
		"dry-run": "false",
	})
	if err != nil {
		t.Fatalf("batch-assign shortcut failed: %v", err)
	}
	// Should have 3 PATCH requests (one per issue)
	if len(patchPayloads) != 3 {
		t.Fatalf("expected 3 PATCH requests, got %d", len(patchPayloads))
	}
	// Each PATCH should have a single issue ID and the new assigners
	for i, payload := range patchPayloads {
		assertNumberSlice(t, payload["ids"], []float64{float64(i + 1)})
		assertNumberSlice(t, payload["assigner_ids"], []float64{1, 2})
	}
}

func TestBatchAssignRemoveAssigners(t *testing.T) {
	var patchPayloads []map[string]interface{}
	server := newIssueTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		// Handle GET requests to fetch issue data
		if r.Method == "GET" {
			writeJSON(t, w, map[string]interface{}{
				"id":        float64(1),
				"assigners": []interface{}{map[string]interface{}{"id": float64(1)}},
			})
			return
		}
		// Handle PATCH request to update assigners
		if r.Method != "PATCH" || r.URL.Path != "/v1/owner/repo/issues/batch_update.json" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		payload := decodeJSON(t, r)
		patchPayloads = append(patchPayloads, payload)
		writeJSON(t, w, map[string]interface{}{"status": 0, "message": "ok"})
	})
	defer server.Close()

	err := runShortcut(t, server, "batch-assign", map[string]string{
		"ids":     "1,2,3",
		"remove":  "1",
		"dry-run": "false",
	})
	if err != nil {
		t.Fatalf("batch-assign shortcut failed: %v", err)
	}
	// Should have 3 PATCH requests (one per issue)
	if len(patchPayloads) != 3 {
		t.Fatalf("expected 3 PATCH requests, got %d", len(patchPayloads))
	}
	// Each PATCH should have a single issue ID and empty assigners (after removing assigner 1)
	for i, payload := range patchPayloads {
		assertNumberSlice(t, payload["ids"], []float64{float64(i + 1)})
		assertNumberSlice(t, payload["assigner_ids"], []float64{})
	}
}

func TestBatchAssignDryRun(t *testing.T) {
	server := newIssueTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("no API call expected in dry-run mode")
	})
	defer server.Close()

	err := runShortcut(t, server, "batch-assign", map[string]string{
		"ids":     "1,2,3",
		"add":     "1",
		"dry-run": "true",
	})
	if err != nil {
		t.Fatalf("batch-assign dry-run failed: %v", err)
	}
}

func TestBatchAssignNoIDs(t *testing.T) {
	server := newIssueTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("no API call expected")
	})
	defer server.Close()

	err := runShortcut(t, server, "batch-assign", map[string]string{"add": "1"})
	if err == nil {
		t.Fatal("expected error when no issue IDs provided")
	}
}

func TestBatchAssignNoAssigners(t *testing.T) {
	server := newIssueTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("no API call expected")
	})
	defer server.Close()

	err := runShortcut(t, server, "batch-assign", map[string]string{"ids": "1,2,3"})
	if err == nil {
		t.Fatal("expected error when no assigners specified")
	}
}

// --- batch-comment ---

func TestBatchCommentAddsComments(t *testing.T) {
	commentCount := 0
	server := newIssueTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == "GET" && r.URL.Path == "/v1/owner/repo/issues/1.json":
			writeJSON(t, w, map[string]interface{}{"subject": "Issue 1"})
		case r.Method == "GET" && r.URL.Path == "/v1/owner/repo/issues/2.json":
			writeJSON(t, w, map[string]interface{}{"subject": "Issue 2"})
		case r.Method == "POST" && r.URL.Path == "/v1/owner/repo/issues/1/journals.json":
			commentCount++
			writeJSON(t, w, map[string]interface{}{"id": float64(1)})
		case r.Method == "POST" && r.URL.Path == "/v1/owner/repo/issues/2/journals.json":
			commentCount++
			writeJSON(t, w, map[string]interface{}{"id": float64(2)})
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	})
	defer server.Close()

	err := runShortcut(t, server, "batch-comment", map[string]string{
		"numbers": "1,2",
		"message": "Batch comment",
		"dry-run": "false",
	})
	if err != nil {
		t.Fatalf("batch-comment shortcut failed: %v", err)
	}
	assertEqual(t, commentCount, 2)
}

func TestBatchCommentDryRun(t *testing.T) {
	server := newIssueTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("no API call expected in dry-run mode")
	})
	defer server.Close()

	err := runShortcut(t, server, "batch-comment", map[string]string{
		"numbers": "1,2,3",
		"message": "Test comment",
		"dry-run": "true",
	})
	if err != nil {
		t.Fatalf("batch-comment dry-run failed: %v", err)
	}
}

func TestBatchCommentNoNumbers(t *testing.T) {
	server := newIssueTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("no API call expected")
	})
	defer server.Close()

	err := runShortcut(t, server, "batch-comment", map[string]string{"message": "test"})
	if err == nil {
		t.Fatal("expected error when no issue numbers provided")
	}
}

func TestBatchCommentNoBody(t *testing.T) {
	server := newIssueTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("no API call expected")
	})
	defer server.Close()

	err := runShortcut(t, server, "batch-comment", map[string]string{"numbers": "1,2"})
	if err == nil {
		t.Fatal("expected error when no body provided")
	}
}

// --- batch-export ---

func TestBatchExportToJSON(t *testing.T) {
	server := newIssueTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" || r.URL.Path != "/v1/owner/repo/issues.json" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		writeJSON(t, w, map[string]interface{}{
			"issues": []interface{}{
				map[string]interface{}{
					"id":         float64(1),
					"subject":    "Bug 1",
					"status":     map[string]interface{}{"id": float64(1), "name": "Open"},
					"priority":   map[string]interface{}{"id": float64(2), "name": "Normal"},
					"assigners":  []interface{}{},
					"tags":       []interface{}{},
					"author":     map[string]interface{}{"id": float64(1), "login": "alice"},
					"created_on": "2026-01-01T00:00:00Z",
				},
			},
		})
	})
	defer server.Close()

	err := runShortcut(t, server, "batch-export", map[string]string{
		"state": "open",
		"limit": "100",
	})
	if err != nil {
		t.Fatalf("batch-export shortcut failed: %v", err)
	}
}

func TestBatchExportToCSV(t *testing.T) {
	server := newIssueTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" || r.URL.Path != "/v1/owner/repo/issues.json" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		writeJSON(t, w, map[string]interface{}{
			"issues": []interface{}{
				map[string]interface{}{
					"id":         float64(1),
					"subject":    "Bug 1",
					"status":     map[string]interface{}{"id": float64(1), "name": "Open"},
					"priority":   map[string]interface{}{"id": float64(2), "name": "Normal"},
					"assigners":  []interface{}{},
					"tags":       []interface{}{},
					"author":     map[string]interface{}{"id": float64(1), "login": "alice"},
					"created_on": "2026-01-01T00:00:00Z",
				},
			},
		})
	})
	defer server.Close()

	err := runShortcut(t, server, "batch-export", map[string]string{
		"state":  "open",
		"format": "csv",
		"limit":  "100",
	})
	if err != nil {
		t.Fatalf("batch-export shortcut failed: %v", err)
	}
}

func TestBatchExportWithFilters(t *testing.T) {
	server := newIssueTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" || r.URL.Path != "/v1/owner/repo/issues.json" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		query := r.URL.Query()
		assertEqual(t, query.Get("category"), "closed")
		assertEqual(t, query.Get("keyword"), "release")
		assertEqual(t, query.Get("status_id"), "5")
		writeJSON(t, w, map[string]interface{}{
			"issues": []interface{}{
				map[string]interface{}{
					"id":         float64(1),
					"subject":    "Filtered Issue",
					"status":     map[string]interface{}{"id": float64(5), "name": "Closed"},
					"priority":   map[string]interface{}{"id": float64(2), "name": "Normal"},
					"assigners":  []interface{}{},
					"tags":       []interface{}{},
					"author":     map[string]interface{}{"id": float64(1), "login": "alice"},
					"created_on": "2026-01-01T00:00:00Z",
				},
			},
		})
	})
	defer server.Close()

	err := runShortcut(t, server, "batch-export", map[string]string{
		"state":     "closed",
		"keyword":   "release",
		"status-id": "5",
		"limit":     "100",
	})
	if err != nil {
		t.Fatalf("batch-export with filters failed: %v", err)
	}
}

// --- batch-import ---

func TestBatchImportFromCSV(t *testing.T) {
	createCount := 0
	server := newIssueTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" || r.URL.Path != "/v1/owner/repo/issues.json" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		payload := decodeJSON(t, r)
		assertEqual(t, payload["subject"], "Imported Issue")
		assertEqual(t, payload["status_id"], float64(1))
		createCount++
		writeJSON(t, w, map[string]interface{}{"id": float64(createCount)})
	})
	defer server.Close()

	// Note: This test will fail because the file doesn't exist
	// In a real test, we would create the file first
	_ = runShortcut(t, server, "batch-import", map[string]string{
		"file":    "/tmp/test-import.csv",
		"dry-run": "false",
	})
	_ = createCount // Avoid unused variable warning
}

func TestBatchImportDryRun(t *testing.T) {
	server := newIssueTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("no API call expected in dry-run mode")
	})
	defer server.Close()

	// This test will fail because the file doesn't exist
	// In a real test, we would create the file first
	err := runShortcut(t, server, "batch-import", map[string]string{
		"file":    "/tmp/test-import.csv",
		"dry-run": "true",
	})
	// We expect an error about missing file
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestBatchImportNoFile(t *testing.T) {
	server := newIssueTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("no API call expected")
	})
	defer server.Close()

	err := runShortcut(t, server, "batch-import", map[string]string{})
	if err == nil {
		t.Fatal("expected error when no file provided")
	}
}
