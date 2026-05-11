package issue

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gitlink-org/gitlink-cli/internal/client"
	"github.com/gitlink-org/gitlink-cli/shortcuts/common"
)

func TestIssueClosePreservesCurrentDescription(t *testing.T) {
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

	err := runIssueShortcut(t, server, "close", map[string]string{"number": "42"})
	if err != nil {
		t.Fatalf("close shortcut failed: %v", err)
	}

	assertEqual(t, updatePayload["subject"], "Existing title")
	assertEqual(t, updatePayload["description"], "Existing description")
	assertEqual(t, updatePayload["status_id"], float64(5))
}

func TestIssueUpdatePreservesCurrentDescriptionWhenChangingTitleAndState(t *testing.T) {
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

	err := runIssueShortcut(t, server, "update", map[string]string{
		"number": "42",
		"title":  "New title",
		"state":  "closed",
	})
	if err != nil {
		t.Fatalf("update shortcut failed: %v", err)
	}

	assertEqual(t, updatePayload["subject"], "New title")
	assertEqual(t, updatePayload["description"], "Existing description")
	assertEqual(t, updatePayload["status_id"], float64(5))
}

func TestIssueUpdatePreservesCurrentSubjectWhenChangingDescription(t *testing.T) {
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

	err := runIssueShortcut(t, server, "update", map[string]string{
		"number": "42",
		"body":   "New description",
	})
	if err != nil {
		t.Fatalf("update shortcut failed: %v", err)
	}

	assertEqual(t, updatePayload["subject"], "Existing title")
	assertEqual(t, updatePayload["description"], "New description")
}

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

	err := runIssueShortcut(t, server, "batch-close", map[string]string{
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

func runIssueShortcut(t *testing.T, server *httptest.Server, name string, args map[string]string) error {
	t.Helper()
	shortcut := findIssueShortcut(t, name)
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
	return shortcut.Run(ctx)
}

func findIssueShortcut(t *testing.T, name string) *common.Shortcut {
	t.Helper()
	for _, shortcut := range Shortcuts() {
		if shortcut.Name == name {
			return shortcut
		}
	}
	t.Fatalf("shortcut %q not found", name)
	return nil
}

func newIssueTestServer(t *testing.T, handler http.HandlerFunc) *httptest.Server {
	t.Helper()
	return httptest.NewServer(handler)
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
