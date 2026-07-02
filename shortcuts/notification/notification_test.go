package notification

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/gitlink-org/gitlink-cli/internal/client"
	"github.com/gitlink-org/gitlink-cli/internal/i18n"
	"github.com/gitlink-org/gitlink-cli/shortcuts/common"
)

func runShortcut(t *testing.T, server *httptest.Server, name string, args map[string]string) error {
	t.Helper()
	shortcut := findShortcut(t, name)
	ctx := &common.RuntimeContext{
		Client: &client.Client{HTTP: server.Client(), BaseURL: server.URL},
		Format: "json",
		Args:   args,
		Tr:     i18n.Default(),
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
		t.Fatalf("write JSON: %v", err)
	}
}

func decodeJSON(t *testing.T, r *http.Request) map[string]interface{} {
	t.Helper()
	var payload map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		t.Fatalf("decode request body: %v", err)
	}
	return payload
}

func assertPath(t *testing.T, r *http.Request, method, path string) {
	t.Helper()
	if r.Method != method {
		t.Fatalf("method = %s, want %s", r.Method, method)
	}
	if r.URL.Path != path {
		t.Fatalf("path = %s, want %s", r.URL.Path, path)
	}
}

// --- list ---

func TestNotificationListExplicitUser(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertPath(t, r, "GET", "/api/users/alice/messages.json")
		if got := r.URL.Query().Get("type"); got != "atme" {
			t.Fatalf("type = %q, want atme", got)
		}
		if got := r.URL.Query().Get("status"); got != "1" {
			t.Fatalf("status = %q, want 1", got)
		}
		if got := r.URL.Query().Get("page"); got != "2" {
			t.Fatalf("page = %q, want 2", got)
		}
		if got := r.URL.Query().Get("limit"); got != "50" {
			t.Fatalf("limit = %q, want 50", got)
		}
		writeJSON(t, w, map[string]interface{}{"total_count": 1, "messages": []interface{}{}})
	}))
	defer server.Close()

	err := runShortcut(t, server, "list", map[string]string{
		"user":   "alice",
		"type":   "atme",
		"status": "unread",
		"page":   "2",
		"limit":  "50",
	})
	if err != nil {
		t.Fatalf("list failed: %v", err)
	}
}

func TestNotificationListDefaultsToCurrentUser(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/users/me.json":
			writeJSON(t, w, map[string]interface{}{"login": "current"})
		case "/api/users/current/messages.json":
			if got := r.URL.Query().Get("status"); got != "2" {
				t.Fatalf("status = %q, want 2", got)
			}
			writeJSON(t, w, map[string]interface{}{"messages": []interface{}{}})
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	if err := runShortcut(t, server, "list", map[string]string{"status": "read"}); err != nil {
		t.Fatalf("list default user failed: %v", err)
	}
}

func TestNotificationDefaultUserMissingLogin(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertPath(t, r, "GET", "/users/me.json")
		writeJSON(t, w, map[string]interface{}{"name": "No Login"})
	}))
	defer server.Close()

	if err := runShortcut(t, server, "list", nil); err == nil {
		t.Fatal("expected error when current user login is unavailable")
	}
}

// --- read ---

func TestNotificationReadDryRunDoesNotCallRemoteWrite(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatalf("dry-run should not call remote API: %s %s", r.Method, r.URL.Path)
	}))
	defer server.Close()

	err := runShortcut(t, server, "read", map[string]string{
		"user":    "alice",
		"ids":     "1,2,3",
		"dry-run": "true",
	})
	if err != nil {
		t.Fatalf("read dry-run failed: %v", err)
	}
}

func TestNotificationReadRequiresYesForRemoteWrite(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatalf("read without --yes should not call remote API: %s %s", r.Method, r.URL.Path)
	}))
	defer server.Close()

	err := runShortcut(t, server, "read", map[string]string{"user": "alice", "ids": "1"})
	if err == nil {
		t.Fatal("expected error when read is missing --yes")
	}
}

func TestNotificationReadPostsIDs(t *testing.T) {
	var payload map[string]interface{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertPath(t, r, "POST", "/api/users/alice/messages/read.json")
		payload = decodeJSON(t, r)
		writeJSON(t, w, map[string]interface{}{"status": 0, "message": "success"})
	}))
	defer server.Close()

	err := runShortcut(t, server, "read", map[string]string{
		"user": "alice",
		"type": "atme",
		"ids":  "4,5",
		"yes":  "true",
	})
	if err != nil {
		t.Fatalf("read failed: %v", err)
	}
	if payload["type"] != "atme" {
		t.Fatalf("type = %#v, want atme", payload["type"])
	}
	if got := floatSliceToInts(payload["ids"]); !reflect.DeepEqual(got, []int{4, 5}) {
		t.Fatalf("ids = %#v, want [4 5]", payload["ids"])
	}
}

func TestNotificationReadAllUnread(t *testing.T) {
	var payload map[string]interface{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertPath(t, r, "POST", "/api/users/alice/messages/read.json")
		payload = decodeJSON(t, r)
		writeJSON(t, w, map[string]interface{}{"status": 0})
	}))
	defer server.Close()

	err := runShortcut(t, server, "read", map[string]string{
		"user":       "alice",
		"all-unread": "true",
		"yes":        "true",
	})
	if err != nil {
		t.Fatalf("read all-unread failed: %v", err)
	}
	if got := floatSliceToInts(payload["ids"]); !reflect.DeepEqual(got, []int{-1}) {
		t.Fatalf("ids = %#v, want [-1]", payload["ids"])
	}
}

func TestNotificationReadRejectsIDsWithAllUnread(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatalf("invalid args should fail before remote API: %s %s", r.Method, r.URL.Path)
	}))
	defer server.Close()

	err := runShortcut(t, server, "read", map[string]string{
		"user":       "alice",
		"ids":        "1",
		"all-unread": "true",
		"dry-run":    "true",
	})
	if err == nil {
		t.Fatal("expected error when --ids and --all-unread are combined")
	}
}

func TestNotificationReadRequiresIDsOrAllUnread(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatalf("invalid args should fail before remote API: %s %s", r.Method, r.URL.Path)
	}))
	defer server.Close()

	err := runShortcut(t, server, "read", map[string]string{"user": "alice", "dry-run": "true"})
	if err == nil {
		t.Fatal("expected error when read has no ids")
	}
}

// --- delete ---

func TestNotificationDeleteDryRunDoesNotCallRemoteWrite(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatalf("dry-run should not call remote API: %s %s", r.Method, r.URL.Path)
	}))
	defer server.Close()

	err := runShortcut(t, server, "delete", map[string]string{
		"user":    "alice",
		"ids":     "9",
		"dry-run": "true",
	})
	if err != nil {
		t.Fatalf("delete dry-run failed: %v", err)
	}
}

func TestNotificationDeleteRequiresYes(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatalf("delete without --yes should not call remote API: %s %s", r.Method, r.URL.Path)
	}))
	defer server.Close()

	err := runShortcut(t, server, "delete", map[string]string{"user": "alice", "ids": "9"})
	if err == nil {
		t.Fatal("expected error when delete is missing --yes")
	}
}

func TestNotificationDeleteCallsEndpoint(t *testing.T) {
	var payload map[string]interface{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertPath(t, r, "DELETE", "/api/users/alice/messages.json")
		payload = decodeJSON(t, r)
		writeJSON(t, w, map[string]interface{}{"status": 0, "message": "success"})
	}))
	defer server.Close()

	err := runShortcut(t, server, "delete", map[string]string{
		"user": "alice",
		"type": "notification",
		"ids":  "10,11",
		"yes":  "true",
	})
	if err != nil {
		t.Fatalf("delete failed: %v", err)
	}
	if got := floatSliceToInts(payload["ids"]); !reflect.DeepEqual(got, []int{10, 11}) {
		t.Fatalf("ids = %#v, want [10 11]", payload["ids"])
	}
}

// --- send-atme ---

func TestNotificationSendAtmeDryRunDoesNotCallRemoteWrite(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatalf("dry-run should not call remote API: %s %s", r.Method, r.URL.Path)
	}))
	defer server.Close()

	err := runShortcut(t, server, "send-atme", map[string]string{
		"user":          "alice",
		"receivers":     "bob,carol",
		"atmeable-type": "Issue",
		"atmeable-id":   "42",
		"dry-run":       "true",
	})
	if err != nil {
		t.Fatalf("send-atme dry-run failed: %v", err)
	}
}

func TestNotificationSendAtmeRequiresYes(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatalf("send-atme without --yes should not call remote API: %s %s", r.Method, r.URL.Path)
	}))
	defer server.Close()

	err := runShortcut(t, server, "send-atme", map[string]string{
		"user":          "alice",
		"receivers":     "bob",
		"atmeable-type": "Issue",
		"atmeable-id":   "42",
	})
	if err == nil {
		t.Fatal("expected error when send-atme is missing --yes")
	}
}

func TestNotificationSendAtmeCallsEndpoint(t *testing.T) {
	var payload map[string]interface{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertPath(t, r, "POST", "/api/users/alice/messages.json")
		payload = decodeJSON(t, r)
		writeJSON(t, w, map[string]interface{}{"status": 0, "message": "success"})
	}))
	defer server.Close()

	err := runShortcut(t, server, "send-atme", map[string]string{
		"user":          "alice",
		"receivers":     "bob, carol",
		"atmeable-type": "PullRequest",
		"atmeable-id":   "77",
		"yes":           "true",
	})
	if err != nil {
		t.Fatalf("send-atme failed: %v", err)
	}
	if payload["type"] != "atme" {
		t.Fatalf("type = %#v, want atme", payload["type"])
	}
	if got, ok := payload["receivers_login"].([]interface{}); !ok || len(got) != 2 || got[0] != "bob" || got[1] != "carol" {
		t.Fatalf("receivers_login = %#v, want [bob carol]", payload["receivers_login"])
	}
	if payload["atmeable_type"] != "PullRequest" {
		t.Fatalf("atmeable_type = %#v, want PullRequest", payload["atmeable_type"])
	}
	if payload["atmeable_id"] != float64(77) {
		t.Fatalf("atmeable_id = %#v, want 77", payload["atmeable_id"])
	}
}

func TestNotificationSendAtmeRejectsInvalidID(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatalf("invalid args should fail before remote API: %s %s", r.Method, r.URL.Path)
	}))
	defer server.Close()

	err := runShortcut(t, server, "send-atme", map[string]string{
		"user":          "alice",
		"receivers":     "bob",
		"atmeable-type": "Issue",
		"atmeable-id":   "0",
		"dry-run":       "true",
	})
	if err == nil {
		t.Fatal("expected invalid atmeable-id error")
	}
}

func floatSliceToInts(raw interface{}) []int {
	values, ok := raw.([]interface{})
	if !ok {
		return nil
	}
	ints := make([]int, 0, len(values))
	for _, value := range values {
		if number, ok := value.(float64); ok {
			ints = append(ints, int(number))
		}
	}
	return ints
}
