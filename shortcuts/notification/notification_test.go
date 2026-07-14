package notification

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/gitlink-org/gitlink-cli/internal/client"
	"github.com/gitlink-org/gitlink-cli/shortcuts/common"
)

func TestNotificationListBuildsQuery(t *testing.T) {
	server := newNotificationTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		assertRequest(t, r, "GET", "/users/alice/messages.json")
		query := r.URL.Query()
		assertEqual(t, query.Get("page"), "2")
		assertEqual(t, query.Get("limit"), "5")
		assertEqual(t, query.Get("type"), "atme")
		assertEqual(t, query.Get("status"), "1")
		writeJSON(t, w, map[string]interface{}{"total_count": 0, "messages": []interface{}{}})
	})
	defer server.Close()

	err := runNotificationShortcut(t, server, "list", map[string]string{
		"user":   "alice",
		"type":   "atme",
		"status": "unread",
		"page":   "2",
		"limit":  "5",
	})
	if err != nil {
		t.Fatalf("list shortcut failed: %v", err)
	}
}

func TestNotificationMarkReadPayload(t *testing.T) {
	var payload map[string]interface{}
	server := newNotificationTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		assertRequest(t, r, "POST", "/users/alice/messages/read.json")
		payload = decodeJSON(t, r)
		writeJSON(t, w, map[string]interface{}{"status": 0, "message": "success"})
	})
	defer server.Close()

	err := runNotificationShortcut(t, server, "mark-read", map[string]string{
		"user": "alice",
		"ids":  "101,102,101",
		"type": "notification",
	})
	if err != nil {
		t.Fatalf("mark-read shortcut failed: %v", err)
	}

	assertEqual(t, payload["type"], "notification")
	assertNumberSlice(t, payload["ids"], []float64{101, 102})
}

func TestNotificationMarkReadAllUnreadDryRunDoesNotCallAPI(t *testing.T) {
	called := false
	server := newNotificationTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		called = true
		t.Fatalf("dry-run should not call API, got: %s %s", r.Method, r.URL.Path)
	})
	defer server.Close()

	err := runNotificationShortcut(t, server, "mark-read", map[string]string{
		"user":       "alice",
		"all-unread": "true",
		"dry-run":    "true",
	})
	if err != nil {
		t.Fatalf("mark-read dry-run failed: %v", err)
	}
	if called {
		t.Fatal("dry-run called API")
	}
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
		"ids":  "201,202",
		"type": "atme",
	})
	if err != nil {
		t.Fatalf("delete shortcut failed: %v", err)
	}

	assertEqual(t, payload["type"], "atme")
	assertNumberSlice(t, payload["ids"], []float64{201, 202})
}

func TestNotificationCreateAtmePayload(t *testing.T) {
	var payload map[string]interface{}
	server := newNotificationTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		assertRequest(t, r, "POST", "/users/alice/messages.json")
		payload = decodeJSON(t, r)
		writeJSON(t, w, map[string]interface{}{"status": 0, "message": "success"})
	})
	defer server.Close()

	err := runNotificationShortcut(t, server, "create-atme", map[string]string{
		"user":          "alice",
		"receivers":     "bob,carol,bob",
		"atmeable-type": "issue",
		"atmeable-id":   "99",
	})
	if err != nil {
		t.Fatalf("create-atme shortcut failed: %v", err)
	}

	assertEqual(t, payload["type"], "atme")
	assertStringSlice(t, payload["receivers_login"], []string{"bob", "carol"})
	assertEqual(t, payload["atmeable_type"], "Issue")
	assertEqual(t, payload["atmeable_id"], float64(99))
}

func TestNotificationPlatformSettings(t *testing.T) {
	server := newNotificationTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		assertRequest(t, r, "GET", "/template_message_settings.json")
		writeJSON(t, w, map[string]interface{}{"status": 0, "setting_types": []interface{}{}})
	})
	defer server.Close()

	if err := runNotificationShortcut(t, server, "platform-settings", nil); err != nil {
		t.Fatalf("platform-settings shortcut failed: %v", err)
	}
}

func TestNotificationSettings(t *testing.T) {
	server := newNotificationTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		assertRequest(t, r, "GET", "/users/alice/template_message_settings.json")
		writeJSON(t, w, map[string]interface{}{
			"status":            0,
			"notification_body": map[string]bool{"Normal::Project": true},
			"email_body":        map[string]bool{"Normal::Project": false},
		})
	})
	defer server.Close()

	if err := runNotificationShortcut(t, server, "settings", map[string]string{"user": "alice"}); err != nil {
		t.Fatalf("settings shortcut failed: %v", err)
	}
}

func TestNotificationSettingsUpdatePreservesExistingKeys(t *testing.T) {
	var payload map[string]interface{}
	server := newNotificationTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == "GET" && r.URL.Path == "/users/alice/template_message_settings.json":
			writeJSON(t, w, map[string]interface{}{
				"status": 0,
				"notification_body": map[string]bool{
					"Normal::Project":      true,
					"ManageProject::Issue": true,
				},
				"email_body": map[string]bool{
					"Normal::Project":      false,
					"ManageProject::Issue": false,
				},
			})
		case r.Method == "POST" && r.URL.Path == "/users/alice/template_message_settings/update_setting.json":
			payload = decodeJSON(t, r)
			writeJSON(t, w, map[string]interface{}{"status": 0, "message": "响应成功"})
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	})
	defer server.Close()

	err := runNotificationShortcut(t, server, "settings-update", map[string]string{
		"user":         "alice",
		"notification": "ManageProject::Issue=false",
		"email":        "Normal::Project=true",
	})
	if err != nil {
		t.Fatalf("settings-update shortcut failed: %v", err)
	}

	setting, ok := payload["setting"].(map[string]interface{})
	if !ok {
		t.Fatalf("payload setting type = %T, want map", payload["setting"])
	}
	notificationBody, ok := setting["notification_body"].(map[string]interface{})
	if !ok {
		t.Fatalf("notification_body type = %T, want map", setting["notification_body"])
	}
	emailBody, ok := setting["email_body"].(map[string]interface{})
	if !ok {
		t.Fatalf("email_body type = %T, want map", setting["email_body"])
	}
	assertEqual(t, notificationBody["Normal::Project"], true)
	assertEqual(t, notificationBody["ManageProject::Issue"], false)
	assertEqual(t, emailBody["Normal::Project"], true)
	assertEqual(t, emailBody["ManageProject::Issue"], false)
}

func TestNotificationRejectsInvalidInputs(t *testing.T) {
	if _, err := normalizeMessageType("chat"); err == nil {
		t.Fatal("expected invalid message type to fail")
	}
	if _, err := normalizeMessageStatus("done"); err == nil {
		t.Fatal("expected invalid status to fail")
	}
	if _, err := normalizeAtmeableType("Repository"); err == nil {
		t.Fatal("expected invalid atmeable type to fail")
	}
	if _, err := parseBoolPairs("Normal::Project=yes"); err == nil {
		t.Fatal("expected invalid boolean setting to fail")
	}
	if _, err := parseMessageIDs("1", true, true); err == nil {
		t.Fatal("expected --ids with --all-unread to fail")
	}
	if _, err := parseMessageIDs("", true, false); err == nil {
		t.Fatal("expected all-unread to be rejected when not allowed")
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

func assertStringSlice(t *testing.T, got interface{}, want []string) {
	t.Helper()
	values, ok := got.([]interface{})
	if !ok {
		t.Fatalf("got %T, want []interface{}", got)
	}
	result := make([]string, 0, len(values))
	for _, value := range values {
		text, ok := value.(string)
		if !ok {
			t.Fatalf("got value %v (%T), want string", value, value)
		}
		result = append(result, text)
	}
	if !reflect.DeepEqual(result, want) {
		t.Fatalf("got %v, want %v", result, want)
	}
}

func assertNumberSlice(t *testing.T, got interface{}, want []float64) {
	t.Helper()
	values, ok := got.([]interface{})
	if !ok {
		t.Fatalf("got %T, want []interface{}", got)
	}
	result := make([]float64, 0, len(values))
	for _, value := range values {
		number, ok := value.(float64)
		if !ok {
			t.Fatalf("got value %v (%T), want float64", value, value)
		}
		result = append(result, number)
	}
	if !reflect.DeepEqual(result, want) {
		t.Fatalf("got %v, want %v", result, want)
	}
}
