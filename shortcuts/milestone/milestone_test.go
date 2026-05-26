package milestone

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gitlink-org/gitlink-cli/internal/client"
	"github.com/gitlink-org/gitlink-cli/shortcuts/common"
)

func TestMilestoneList(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertRequest(t, r, "GET", "/v1/owner/repo/milestones.json")
		assertEqual(t, r.URL.Query().Get("category"), "opening")
		assertEqual(t, r.URL.Query().Get("keyword"), "v1")
		assertEqual(t, r.URL.Query().Get("page"), "2")
		assertEqual(t, r.URL.Query().Get("limit"), "50")
		writeJSON(t, w, map[string]interface{}{"total_count": 0, "milestones": []interface{}{}})
	}))
	defer server.Close()

	err := runMilestoneShortcut(t, server, "list", map[string]string{
		"category": "opening",
		"keyword":  "v1",
		"page":     "2",
		"limit":    "50",
	})
	if err != nil {
		t.Fatalf("list shortcut failed: %v", err)
	}
}

func TestMilestoneCreatePayload(t *testing.T) {
	var payload map[string]interface{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertRequest(t, r, "POST", "/v1/owner/repo/milestones.json")
		payload = decodeJSON(t, r)
		writeJSON(t, w, map[string]interface{}{"status": 0, "message": "success"})
	}))
	defer server.Close()

	err := runMilestoneShortcut(t, server, "create", map[string]string{
		"name":        "v1.0",
		"description": "first release",
		"due-date":    "2026-07-01",
	})
	if err != nil {
		t.Fatalf("create shortcut failed: %v", err)
	}

	assertEqual(t, payload["name"], "v1.0")
	assertEqual(t, payload["description"], "first release")
	assertEqual(t, payload["effective_date"], "2026-07-01")
}

func TestMilestoneViewWithIssueFilters(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertRequest(t, r, "GET", "/v1/owner/repo/milestones/7.json")
		assertEqual(t, r.URL.Query().Get("category"), "opened")
		assertEqual(t, r.URL.Query().Get("author_id"), "11")
		assertEqual(t, r.URL.Query().Get("assigner_id"), "22")
		assertEqual(t, r.URL.Query().Get("issue_tag_ids"), "1,2,3")
		writeJSON(t, w, map[string]interface{}{"milestone": map[string]interface{}{"id": 7}})
	}))
	defer server.Close()

	err := runMilestoneShortcut(t, server, "view", map[string]string{
		"id":            "7",
		"category":      "opened",
		"author-id":     "11",
		"assigner-id":   "22",
		"issue-tag-ids": "1, 2,3",
	})
	if err != nil {
		t.Fatalf("view shortcut failed: %v", err)
	}
}

func TestMilestoneUpdatePayload(t *testing.T) {
	var payload map[string]interface{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertRequest(t, r, "PATCH", "/v1/owner/repo/milestones/7.json")
		payload = decodeJSON(t, r)
		writeJSON(t, w, map[string]interface{}{"status": 0, "message": "success"})
	}))
	defer server.Close()

	err := runMilestoneShortcut(t, server, "update", map[string]string{
		"id":       "7",
		"due-date": "2026-08-01",
	})
	if err != nil {
		t.Fatalf("update shortcut failed: %v", err)
	}

	if _, ok := payload["name"]; ok {
		t.Fatal("update payload should omit empty name")
	}
	assertEqual(t, payload["effective_date"], "2026-08-01")
}

func TestMilestoneUpdateRequiresChange(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatalf("server should not be called when update payload is empty: %s %s", r.Method, r.URL.Path)
	}))
	defer server.Close()

	err := runMilestoneShortcut(t, server, "update", map[string]string{"id": "7"})
	if err == nil {
		t.Fatal("expected update without fields to return an error")
	}
}

func TestMilestoneDelete(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertRequest(t, r, "DELETE", "/v1/owner/repo/milestones/7.json")
		writeJSON(t, w, map[string]interface{}{"status": 0, "message": "success"})
	}))
	defer server.Close()

	if err := runMilestoneShortcut(t, server, "delete", map[string]string{"id": "7"}); err != nil {
		t.Fatalf("delete shortcut failed: %v", err)
	}
}

func TestMilestoneCloseAndReopen(t *testing.T) {
	gotStatuses := []string{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertRequest(t, r, "POST", "/owner/repo/milestones/7/update_status.json")
		payload := decodeJSON(t, r)
		gotStatuses = append(gotStatuses, payload["status"].(string))
		writeJSON(t, w, map[string]interface{}{"status": 0, "message": "success"})
	}))
	defer server.Close()

	if err := runMilestoneShortcut(t, server, "close", map[string]string{"id": "7"}); err != nil {
		t.Fatalf("close shortcut failed: %v", err)
	}
	if err := runMilestoneShortcut(t, server, "reopen", map[string]string{"id": "7"}); err != nil {
		t.Fatalf("reopen shortcut failed: %v", err)
	}
	assertEqual(t, gotStatuses[0], "closed")
	assertEqual(t, gotStatuses[1], "open")
}

func runMilestoneShortcut(t *testing.T, server *httptest.Server, name string, args map[string]string) error {
	t.Helper()
	shortcut := findMilestoneShortcut(t, name)
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

func findMilestoneShortcut(t *testing.T, name string) *common.Shortcut {
	t.Helper()
	for _, shortcut := range Shortcuts() {
		if shortcut.Name == name {
			return shortcut
		}
	}
	t.Fatalf("shortcut %q not found", name)
	return nil
}

func assertRequest(t *testing.T, r *http.Request, method, path string) {
	t.Helper()
	if r.Method != method || r.URL.Path != path {
		t.Fatalf("got request %s %s, want %s %s", r.Method, r.URL.Path, method, path)
	}
}

func decodeJSON(t *testing.T, r *http.Request) map[string]interface{} {
	t.Helper()
	var payload map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		t.Fatalf("failed to decode request body: %v", err)
	}
	return payload
}

func writeJSON(t *testing.T, w http.ResponseWriter, payload interface{}) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		t.Fatalf("failed to write response: %v", err)
	}
}

func assertEqual(t *testing.T, got interface{}, want interface{}) {
	t.Helper()
	if got != want {
		t.Fatalf("got %v (%T), want %v (%T)", got, got, want, want)
	}
}
