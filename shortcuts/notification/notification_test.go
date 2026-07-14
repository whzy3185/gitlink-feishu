package notification

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gitlink-org/gitlink-cli/internal/client"
	"github.com/gitlink-org/gitlink-cli/shortcuts/common"
)

func TestNotificationListResolvesCurrentUser(t *testing.T) {
	requests := 0
	server := newNotificationTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		requests++
		switch requests {
		case 1:
			assertRequest(t, r, "GET", "/users/me.json")
			writeJSON(t, w, map[string]interface{}{"login": "mengz"})
		case 2:
			assertRequest(t, r, "GET", "/users/mengz/messages.json")
			assertEqual(t, r.URL.Query().Get("type"), "notification")
			assertEqual(t, r.URL.Query().Get("status"), "1")
			assertEqual(t, r.URL.Query().Get("page"), "2")
			assertEqual(t, r.URL.Query().Get("limit"), "50")
			writeJSON(t, w, map[string]interface{}{"total_count": 0, "messages": []interface{}{}})
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	})
	defer server.Close()

	err := runNotificationShortcut(t, server, "list", map[string]string{
		"type":   "notification",
		"status": "unread",
		"page":   "2",
		"limit":  "50",
	})
	if err != nil {
		t.Fatalf("list shortcut failed: %v", err)
	}
	assertEqual(t, requests, 2)
}

func TestNotificationListUsesExplicitUser(t *testing.T) {
	server := newNotificationTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		assertRequest(t, r, "GET", "/users/alice/messages.json")
		assertEqual(t, r.URL.Query().Get("page"), "1")
		assertEqual(t, r.URL.Query().Get("limit"), "20")
		assertEqual(t, r.URL.Query().Get("type"), "")
		assertEqual(t, r.URL.Query().Get("status"), "")
		writeJSON(t, w, map[string]interface{}{"total_count": 0, "messages": []interface{}{}})
	})
	defer server.Close()

	if err := runNotificationShortcut(t, server, "list", map[string]string{"user": "alice"}); err != nil {
		t.Fatalf("list shortcut failed: %v", err)
	}
}

func TestNotificationReadPayload(t *testing.T) {
	var payload map[string]interface{}
	server := newNotificationTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		assertRequest(t, r, "POST", "/users/alice/messages/read.json")
		payload = decodeJSON(t, r)
		writeJSON(t, w, map[string]interface{}{"status": 0, "message": "success"})
	})
	defer server.Close()

	err := runNotificationShortcut(t, server, "read", map[string]string{
		"user": "alice",
		"type": "atme",
		"ids":  "1,2,2",
	})
	if err != nil {
		t.Fatalf("read shortcut failed: %v", err)
	}
	assertEqual(t, payload["type"], "atme")
	assertIntSlice(t, payload["ids"], []int{1, 2})
}

func TestNotificationReadAllUnread(t *testing.T) {
	var payload map[string]interface{}
	server := newNotificationTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		assertRequest(t, r, "POST", "/users/alice/messages/read.json")
		payload = decodeJSON(t, r)
		writeJSON(t, w, map[string]interface{}{"status": 0, "message": "success"})
	})
	defer server.Close()

	err := runNotificationShortcut(t, server, "read", map[string]string{
		"user": "alice",
		"type": "notification",
		"ids":  "-1",
	})
	if err != nil {
		t.Fatalf("read shortcut failed: %v", err)
	}
	assertIntSlice(t, payload["ids"], []int{-1})
}

func TestNotificationDeletePayload(t *testing.T) {
	var payload map[string]interface{}
	server := newNotificationTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		assertRequest(t, r, "DELETE", "/users/alice/messages.json")
		payload = decodeJSON(t, r)
		writeJSON(t, w, map[string]interface{}{"status": 0, "message": "success"})
	})
	defer server.Close()

	err := runNotificationShortcut(t, server, "delete", map[string]string{
		"user": "alice",
		"type": "notification",
		"ids":  "7,8",
	})
	if err != nil {
		t.Fatalf("delete shortcut failed: %v", err)
	}
	assertEqual(t, payload["type"], "notification")
	assertIntSlice(t, payload["ids"], []int{7, 8})
}

func TestNotificationValidation(t *testing.T) {
	server := newNotificationTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		t.Fatalf("invalid input should not call API, got: %s %s", r.Method, r.URL.Path)
	})
	defer server.Close()

	cases := []struct {
		name     string
		shortcut string
		args     map[string]string
	}{
		{name: "invalid type", shortcut: "list", args: map[string]string{"user": "alice", "type": "other"}},
		{name: "invalid status", shortcut: "list", args: map[string]string{"user": "alice", "status": "maybe"}},
		{name: "invalid page", shortcut: "list", args: map[string]string{"user": "alice", "page": "0"}},
		{name: "invalid ids", shortcut: "read", args: map[string]string{"user": "alice", "type": "atme", "ids": "abc"}},
		{name: "delete all unread rejected", shortcut: "delete", args: map[string]string{"user": "alice", "type": "atme", "ids": "-1"}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if err := runNotificationShortcut(t, server, tc.shortcut, tc.args); err == nil {
				t.Fatal("expected validation error")
			}
		})
	}
}

func runNotificationShortcut(t *testing.T, server *httptest.Server, name string, args map[string]string) error {
	t.Helper()
	shortcut := findNotificationShortcut(t, name)
	ctx := &common.RuntimeContext{
		Client: &client.Client{
			HTTP:    server.Client(),
			BaseURL: server.URL,
		},
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
		t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
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
	if fmt.Sprintf("%v", got) != fmt.Sprintf("%v", want) {
		t.Fatalf("got %v (%T), want %v (%T)", got, got, want, want)
	}
}

func assertIntSlice(t *testing.T, got interface{}, want []int) {
	t.Helper()
	values, ok := got.([]interface{})
	if !ok {
		t.Fatalf("got ids %T, want []interface{}", got)
	}
	if len(values) != len(want) {
		t.Fatalf("got ids length %d, want %d", len(values), len(want))
	}
	for i, value := range values {
		assertEqual(t, value, float64(want[i]))
	}
}
