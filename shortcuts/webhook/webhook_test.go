package webhook

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gitlink-org/gitlink-cli/internal/client"
	"github.com/gitlink-org/gitlink-cli/shortcuts/common"
)

func TestWebhookCreateBuildsPayload(t *testing.T) {
	var createPayload map[string]interface{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == "POST" && r.URL.Path == "/owner/repo/webhooks.json":
			createPayload = decodeWebhookJSON(t, r)
			writeWebhookJSON(t, w, map[string]interface{}{"id": float64(1)})
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	err := runWebhookShortcut(t, server, "create", map[string]string{
		"url":           "https://example.com/hook",
		"events":        "push,create",
		"type":          "slack",
		"content-type":  "form",
		"http-method":   "GET",
		"secret":        "abc123",
		"branch-filter": "master",
		"active":        "false",
	})
	if err != nil {
		t.Fatalf("create shortcut failed: %v", err)
	}

	assertWebhookEqual(t, createPayload["url"], "https://example.com/hook")
	assertWebhookEqual(t, createPayload["type"], "slack")
	assertWebhookEqual(t, createPayload["content_type"], "form")
	assertWebhookEqual(t, createPayload["http_method"], "GET")
	assertWebhookEqual(t, createPayload["secret"], "abc123")
	assertWebhookEqual(t, createPayload["branch_filter"], "master")
	assertWebhookEqual(t, createPayload["active"], false)
}

func TestWebhookUpdatePreservesExistingFields(t *testing.T) {
	var updatePayload map[string]interface{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == "GET" && r.URL.Path == "/owner/repo/webhooks/68.json":
			writeWebhookJSON(t, w, map[string]interface{}{
				"id":            float64(68),
				"type":          "gitea",
				"url":           "https://old.example.com/hook",
				"content_type":  "json",
				"http_method":   "POST",
				"secret":        "old-secret",
				"branch_filter": "*",
				"events":        []interface{}{"push", "create"},
				"active":        true,
			})
		case r.Method == "PUT" && r.URL.Path == "/owner/repo/webhooks/68.json":
			updatePayload = decodeWebhookJSON(t, r)
			writeWebhookJSON(t, w, map[string]interface{}{"id": float64(68)})
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	err := runWebhookShortcut(t, server, "update", map[string]string{
		"id":     "68",
		"url":    "https://new.example.com/hook",
		"active": "false",
	})
	if err != nil {
		t.Fatalf("update shortcut failed: %v", err)
	}

	assertWebhookEqual(t, updatePayload["url"], "https://new.example.com/hook")
	assertWebhookEqual(t, updatePayload["type"], "gitea")
	assertWebhookEqual(t, updatePayload["content_type"], "json")
	assertWebhookEqual(t, updatePayload["http_method"], "POST")
	assertWebhookEqual(t, updatePayload["secret"], "old-secret")
	assertWebhookEqual(t, updatePayload["branch_filter"], "*")
	assertWebhookEqual(t, updatePayload["active"], false)
}

func TestWebhookTestTriggersDelivery(t *testing.T) {
	called := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == "POST" && r.URL.Path == "/owner/repo/webhooks/68/tests.json":
			called = true
			writeWebhookJSON(t, w, map[string]interface{}{"status": float64(0), "message": "success"})
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	err := runWebhookShortcut(t, server, "test", map[string]string{"id": "68"})
	if err != nil {
		t.Fatalf("test shortcut failed: %v", err)
	}
	if !called {
		t.Fatal("webhook test endpoint was not called")
	}
}

func runWebhookShortcut(t *testing.T, server *httptest.Server, name string, args map[string]string) error {
	t.Helper()
	shortcut := findWebhookShortcut(t, name)
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

func findWebhookShortcut(t *testing.T, name string) *common.Shortcut {
	t.Helper()
	for _, shortcut := range Shortcuts() {
		if shortcut.Name == name {
			return shortcut
		}
	}
	t.Fatalf("shortcut %q not found", name)
	return nil
}

func decodeWebhookJSON(t *testing.T, r *http.Request) map[string]interface{} {
	t.Helper()
	var payload map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		t.Fatalf("failed to decode request body: %v", err)
	}
	return payload
}

func writeWebhookJSON(t *testing.T, w http.ResponseWriter, payload interface{}) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		t.Fatalf("failed to write response: %v", err)
	}
}

func assertWebhookEqual(t *testing.T, got interface{}, want interface{}) {
	t.Helper()
	if got != want {
		t.Fatalf("got %v (%T), want %v (%T)", got, got, want, want)
	}
}
