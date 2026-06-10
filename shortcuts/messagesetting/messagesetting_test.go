package messagesetting

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gitlink-org/gitlink-cli/internal/client"
	"github.com/gitlink-org/gitlink-cli/shortcuts/common"
)

func runMessageSettingShortcut(t *testing.T, server *httptest.Server, name string, args map[string]string) error {
	t.Helper()
	shortcut := findMessageSettingShortcut(t, name)
	ctx := &common.RuntimeContext{
		Client: &client.Client{HTTP: server.Client(), BaseURL: server.URL},
		Format: "json",
		Args:   args,
	}
	return shortcut.Run(ctx)
}

func findMessageSettingShortcut(t *testing.T, name string) *common.Shortcut {
	t.Helper()
	for _, shortcut := range Shortcuts() {
		if shortcut.Name == name {
			return shortcut
		}
	}
	t.Fatalf("shortcut %q not found", name)
	return nil
}

func writeMessageSettingJSON(t *testing.T, w http.ResponseWriter, value interface{}) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(value); err != nil {
		t.Fatalf("encode json: %v", err)
	}
}

func decodeMessageSettingJSON(t *testing.T, r *http.Request) map[string]interface{} {
	t.Helper()
	defer r.Body.Close()
	var value map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&value); err != nil {
		t.Fatalf("decode request body: %v", err)
	}
	return value
}

func TestMessageSettingsCatalog(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/api/template_message_settings.json" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		writeMessageSettingJSON(t, w, catalogFixture())
	}))
	defer server.Close()

	if err := runMessageSettingShortcut(t, server, "catalog", nil); err != nil {
		t.Fatalf("catalog failed: %v", err)
	}
}

func TestMessageSettingsViewUsesCurrentUserWhenLoginMissing(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/users/me.json":
			writeMessageSettingJSON(t, w, map[string]interface{}{"login": "alice"})
		case r.Method == http.MethodGet && r.URL.Path == "/api/template_message_settings.json":
			writeMessageSettingJSON(t, w, catalogFixture())
		case r.Method == http.MethodGet && r.URL.Path == "/api/users/alice/template_message_settings.json":
			writeMessageSettingJSON(t, w, userSettingFixture())
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	if err := runMessageSettingShortcut(t, server, "view", nil); err != nil {
		t.Fatalf("view failed: %v", err)
	}
}

func TestMessageSettingsUpdateDryRunDoesNotPost(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/users/me.json":
			writeMessageSettingJSON(t, w, map[string]interface{}{"login": "alice"})
		case r.Method == http.MethodGet && r.URL.Path == "/api/template_message_settings.json":
			writeMessageSettingJSON(t, w, catalogFixture())
		case r.Method == http.MethodGet && r.URL.Path == "/api/users/alice/template_message_settings.json":
			writeMessageSettingJSON(t, w, userSettingFixture())
		default:
			t.Fatalf("dry-run should not write, got %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	err := runMessageSettingShortcut(t, server, "update", map[string]string{
		"channel": "notification",
		"state":   "off",
		"keys":    "Normal::Permission",
		"dry-run": "true",
	})
	if err != nil {
		t.Fatalf("update dry-run failed: %v", err)
	}
}

func TestMessageSettingsUpdatePostsMergedPayload(t *testing.T) {
	var payload map[string]interface{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/users/me.json":
			writeMessageSettingJSON(t, w, map[string]interface{}{"login": "alice"})
		case r.Method == http.MethodGet && r.URL.Path == "/api/template_message_settings.json":
			writeMessageSettingJSON(t, w, catalogFixture())
		case r.Method == http.MethodGet && r.URL.Path == "/api/users/alice/template_message_settings.json":
			writeMessageSettingJSON(t, w, userSettingFixture())
		case r.Method == http.MethodPost && r.URL.Path == "/api/users/alice/template_message_settings/update_setting.json":
			payload = decodeMessageSettingJSON(t, r)
			writeMessageSettingJSON(t, w, map[string]interface{}{"status": 0, "message": "updated"})
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	err := runMessageSettingShortcut(t, server, "update", map[string]string{
		"channel": "email",
		"state":   "on",
		"group":   "ManageProject",
	})
	if err != nil {
		t.Fatalf("update failed: %v", err)
	}

	setting, ok := payload["setting"].(map[string]interface{})
	if !ok {
		t.Fatalf("missing setting payload: %#v", payload)
	}
	notificationBody := setting["notification_body"].(map[string]interface{})
	emailBody := setting["email_body"].(map[string]interface{})

	if notificationBody["ManageProject::Issue"] != true {
		t.Fatalf("notification should stay true for ManageProject::Issue, got %#v", notificationBody["ManageProject::Issue"])
	}
	if emailBody["ManageProject::Issue"] != true {
		t.Fatalf("email should be enabled for ManageProject::Issue, got %#v", emailBody["ManageProject::Issue"])
	}
	if emailBody["Normal::Permission"] != false {
		t.Fatalf("email should preserve unrelated keys, got %#v", emailBody["Normal::Permission"])
	}
	if notificationBody["ManageProject::Praised"] != true {
		t.Fatalf("notification defaults should be preserved for missing keys, got %#v", notificationBody["ManageProject::Praised"])
	}
	if emailBody["ManageProject::Praised"] != true {
		t.Fatalf("selected group keys should be updated even when missing from current settings, got %#v", emailBody["ManageProject::Praised"])
	}
}

func TestMessageSettingsPresetPostsSelectedKeys(t *testing.T) {
	var payload map[string]interface{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/users/me.json":
			writeMessageSettingJSON(t, w, map[string]interface{}{"login": "alice"})
		case r.Method == http.MethodGet && r.URL.Path == "/api/template_message_settings.json":
			writeMessageSettingJSON(t, w, catalogFixture())
		case r.Method == http.MethodGet && r.URL.Path == "/api/users/alice/template_message_settings.json":
			writeMessageSettingJSON(t, w, userSettingFixture())
		case r.Method == http.MethodPost && r.URL.Path == "/api/users/alice/template_message_settings/update_setting.json":
			payload = decodeMessageSettingJSON(t, r)
			writeMessageSettingJSON(t, w, map[string]interface{}{"status": 0, "message": "updated"})
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	err := runMessageSettingShortcut(t, server, "preset", map[string]string{
		"name":  "all-off",
		"keys":  "Permission,ManageProject::Issue",
		"all":   "false",
	})
	if err != nil {
		t.Fatalf("preset failed: %v", err)
	}

	setting := payload["setting"].(map[string]interface{})
	notificationBody := setting["notification_body"].(map[string]interface{})
	emailBody := setting["email_body"].(map[string]interface{})

	if notificationBody["Normal::Permission"] != false || emailBody["Normal::Permission"] != false {
		t.Fatalf("Normal::Permission should be turned off by preset")
	}
	if notificationBody["ManageProject::Issue"] != false || emailBody["ManageProject::Issue"] != false {
		t.Fatalf("ManageProject::Issue should be turned off by preset")
	}
	if notificationBody["ManageProject::PullRequest"] != true {
		t.Fatalf("unselected keys should remain unchanged, got %#v", notificationBody["ManageProject::PullRequest"])
	}
}

func catalogFixture() map[string]interface{} {
	return map[string]interface{}{
		"setting_types": []map[string]interface{}{
			{
				"type":      "TemplateMessageSetting::Normal",
				"type_name": "My status",
				"settings": []map[string]interface{}{
					{
						"name":                  "Permission changed",
						"key":                   "Permission",
						"notification_disabled": false,
						"email_disabled":        false,
					},
				},
			},
			{
				"type":      "TemplateMessageSetting::ManageProject",
				"type_name": "Managed repositories",
				"settings": []map[string]interface{}{
					{
						"name":                  "New issue",
						"key":                   "Issue",
						"notification_disabled": false,
						"email_disabled":        false,
					},
					{
						"name":                  "New pull request",
						"key":                   "PullRequest",
						"notification_disabled": false,
						"email_disabled":        false,
					},
					{
						"name":                  "Praised",
						"key":                   "Praised",
						"notification_disabled": false,
						"email_disabled":        true,
					},
				},
			},
		},
	}
}

func userSettingFixture() map[string]interface{} {
	return map[string]interface{}{
		"user": map[string]interface{}{
			"login": "alice",
			"name":  "Alice",
		},
		"notification_body": map[string]interface{}{
			"Normal::Permission":        true,
			"ManageProject::Issue":      true,
			"ManageProject::PullRequest": true,
		},
		"email_body": map[string]interface{}{
			"Normal::Permission":        false,
			"ManageProject::Issue":      false,
			"ManageProject::PullRequest": false,
		},
	}
}
