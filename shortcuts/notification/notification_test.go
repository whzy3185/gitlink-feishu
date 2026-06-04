package notification

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gitlink-org/gitlink-cli/internal/client"
	"github.com/gitlink-org/gitlink-cli/shortcuts/common"
)

func TestNotificationList(t *testing.T) {
	server := newNotificationTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		assertRequest(t, r, "GET", "/users/testuser/messages.json")
		writeJSON(t, w, map[string]interface{}{"total_count": 1, "messages": []interface{}{}})
	})
	defer server.Close()

	if err := runNotificationShortcut(t, server, "list", nil); err != nil {
		t.Fatalf("list shortcut failed: %v", err)
	}
}

func TestNotificationView(t *testing.T) {
	server := newNotificationTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		assertRequest(t, r, "GET", "/users/testuser/messages/42.json")
		writeJSON(t, w, map[string]interface{}{"id": 42, "subject": "test notification"})
	})
	defer server.Close()

	if err := runNotificationShortcut(t, server, "view", map[string]string{"id": "42"}); err != nil {
		t.Fatalf("view shortcut failed: %v", err)
	}
}

func TestNotificationRead(t *testing.T) {
	server := newNotificationTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		assertRequest(t, r, "POST", "/users/testuser/messages/read.json")
		writeJSON(t, w, map[string]interface{}{"status": 0, "message": "success"})
	})
	defer server.Close()

	if err := runNotificationShortcut(t, server, "read", map[string]string{"id": "42"}); err != nil {
		t.Fatalf("read shortcut failed: %v", err)
	}
}

func TestNotificationDelete(t *testing.T) {
	server := newNotificationTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		assertRequest(t, r, "DELETE", "/users/testuser/messages/42.json")
		writeJSON(t, w, map[string]interface{}{"status": 0, "message": "success"})
	})
	defer server.Close()

	if err := runNotificationShortcut(t, server, "delete", map[string]string{"id": "42"}); err != nil {
		t.Fatalf("delete shortcut failed: %v", err)
	}
}

// --- test helpers ---

func runNotificationShortcut(t *testing.T, server *httptest.Server, name string, args map[string]string) error {
	t.Helper()
	shortcut := findNotificationShortcut(t, name)
	ctx := &common.RuntimeContext{
		Client: &client.Client{
			HTTP:    server.Client(),
			BaseURL: server.URL,
		},
		Owner:  "testuser",
		Format: "json",
		Args:   args,
	}
	if ctx.Args == nil {
		ctx.Args = map[string]string{}
	}
	return shortcut.Run(ctx)
}

func findNotificationShortcut(t *testing.T, name string) *common.Shortcut {
	t.Helper()
	for _, shortcut := range Shortcuts() {
		if shortcut.Name == name {
			return shortcut
		}
	}
	t.Fatalf("shortcut %q not found", name)
	return nil
}

func newNotificationTestServer(t *testing.T, handler http.HandlerFunc) *httptest.Server {
	t.Helper()
	return httptest.NewServer(handler)
}

func assertRequest(t *testing.T, r *http.Request, method, path string) {
	t.Helper()
	if r.Method != method || r.URL.Path != path {
		t.Fatalf("got request %s %s, want %s %s", r.Method, r.URL.Path, method, path)
	}
}

func writeJSON(t *testing.T, w http.ResponseWriter, payload interface{}) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		t.Fatalf("failed to write response: %v", err)
	}
}
