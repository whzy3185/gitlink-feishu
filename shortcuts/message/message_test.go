package message

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/gitlink-org/gitlink-cli/internal/client"
	"github.com/gitlink-org/gitlink-cli/shortcuts/common"
)

func runMessageShortcut(t *testing.T, server *httptest.Server, name string, args map[string]string) error {
	t.Helper()
	shortcut := findMessageShortcut(t, name)
	ctx := &common.RuntimeContext{
		Client: &client.Client{HTTP: server.Client(), BaseURL: server.URL},
		Format: "json",
		Args:   args,
	}
	if ctx.Args == nil {
		ctx.Args = map[string]string{}
	}
	return shortcut.Run(ctx)
}

func findMessageShortcut(t *testing.T, name string) *common.Shortcut {
	t.Helper()
	for _, shortcut := range Shortcuts() {
		if shortcut.Name == name {
			return shortcut
		}
	}
	t.Fatalf("shortcut %q not found", name)
	return nil
}

func writeMessageJSON(t *testing.T, w http.ResponseWriter, value interface{}) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(value); err != nil {
		t.Fatalf("encode json: %v", err)
	}
}

func decodeMessageJSON(t *testing.T, r *http.Request) map[string]interface{} {
	t.Helper()
	defer r.Body.Close()
	var value map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&value); err != nil {
		t.Fatalf("decode request body: %v", err)
	}
	return value
}

func TestMessageListUsesCurrentUser(t *testing.T) {
	var sawList bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/users/me.json":
			writeMessageJSON(t, w, map[string]interface{}{"login": "alice"})
		case r.Method == http.MethodGet && r.URL.Path == "/api/users/alice/messages.json":
			sawList = true
			if got := r.URL.Query().Get("type"); got != messageTypeNotification {
				t.Fatalf("type query = %q, want %q", got, messageTypeNotification)
			}
			if got := r.URL.Query().Get("status"); got != "1" {
				t.Fatalf("status query = %q, want 1", got)
			}
			if got := r.URL.Query().Get("page"); got != "2" {
				t.Fatalf("page query = %q, want 2", got)
			}
			if got := r.URL.Query().Get("limit"); got != "5" {
				t.Fatalf("limit query = %q, want 5", got)
			}
			writeMessageJSON(t, w, messageListFixture())
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.String())
		}
	}))
	defer server.Close()

	err := runMessageShortcut(t, server, "list", map[string]string{
		"type":   messageTypeNotification,
		"status": messageStatusUnread,
		"page":   "2",
		"limit":  "5",
	})
	if err != nil {
		t.Fatalf("list failed: %v", err)
	}
	if !sawList {
		t.Fatal("expected message list request to be sent")
	}
}

func TestMessageStatsUsesSelectedLogin(t *testing.T) {
	var sawStats bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/api/users/alice/messages.json" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.String())
		}
		sawStats = true
		if got := r.URL.Query().Get("page"); got != "1" {
			t.Fatalf("page query = %q, want 1", got)
		}
		if got := r.URL.Query().Get("limit"); got != "1" {
			t.Fatalf("limit query = %q, want 1", got)
		}
		if got := r.URL.Query().Get("type"); got != messageTypeAtme {
			t.Fatalf("type query = %q, want %q", got, messageTypeAtme)
		}
		writeMessageJSON(t, w, messageListFixture())
	}))
	defer server.Close()

	err := runMessageShortcut(t, server, "stats", map[string]string{
		"login": "alice",
		"type":  messageTypeAtme,
	})
	if err != nil {
		t.Fatalf("stats failed: %v", err)
	}
	if !sawStats {
		t.Fatal("expected message stats request to be sent")
	}
}

func TestMessageReadDryRunDoesNotWrite(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatalf("dry-run should not write, got %s %s", r.Method, r.URL.String())
	}))
	defer server.Close()

	err := runMessageShortcut(t, server, "read", map[string]string{
		"login":   "alice",
		"type":    messageTypeNotification,
		"ids":     "101,202",
		"dry-run": "true",
	})
	if err != nil {
		t.Fatalf("read dry-run failed: %v", err)
	}
}

func TestMessageReadMarksAllUnread(t *testing.T) {
	var payload map[string]interface{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/users/alice/messages/read.json" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.String())
		}
		payload = decodeMessageJSON(t, r)
		writeMessageJSON(t, w, map[string]interface{}{"status": 0, "message": "updated"})
	}))
	defer server.Close()

	err := runMessageShortcut(t, server, "read", map[string]string{
		"login": "alice",
		"type":  messageTypeNotification,
		"all":   "true",
	})
	if err != nil {
		t.Fatalf("read failed: %v", err)
	}

	if payload["type"] != messageTypeNotification {
		t.Fatalf("type = %#v, want %q", payload["type"], messageTypeNotification)
	}
	ids := payload["ids"].([]interface{})
	if len(ids) != 1 || ids[0].(float64) != -1 {
		t.Fatalf("ids = %#v, want [-1]", ids)
	}
}

func TestMessageDeletePostsSelectedIDs(t *testing.T) {
	var payload map[string]interface{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete || r.URL.Path != "/api/users/alice/messages.json" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.String())
		}
		payload = decodeMessageJSON(t, r)
		writeMessageJSON(t, w, map[string]interface{}{"status": 0, "message": "deleted"})
	}))
	defer server.Close()

	err := runMessageShortcut(t, server, "delete", map[string]string{
		"login": "alice",
		"type":  messageTypeAtme,
		"ids":   "202,101,202",
	})
	if err != nil {
		t.Fatalf("delete failed: %v", err)
	}

	if payload["type"] != messageTypeAtme {
		t.Fatalf("type = %#v, want %q", payload["type"], messageTypeAtme)
	}
	ids := payload["ids"].([]interface{})
	if len(ids) != 2 || ids[0].(float64) != 101 || ids[1].(float64) != 202 {
		t.Fatalf("ids = %#v, want [101 202]", ids)
	}
}

func TestParseMutationIDsRejectsInvalidInput(t *testing.T) {
	if _, _, err := parseMutationIDs("", false); err == nil {
		t.Fatal("expected error when neither --ids nor --all is provided")
	}
	if _, _, err := parseMutationIDs("1,abc", false); err == nil {
		t.Fatal("expected error for invalid message id")
	}
	if _, _, err := parseMutationIDs("1", true); err == nil {
		t.Fatal("expected error when --ids and --all are used together")
	}
}

func TestNormalizeMessageText(t *testing.T) {
	got := normalizeMessageText("someone <b>@you</b> &amp; <i>team</i>")
	want := "someone @you & team"
	if got != want {
		t.Fatalf("normalizeMessageText() = %q, want %q", got, want)
	}
}

func TestNormalizeMessagesIncludesPlainTextContent(t *testing.T) {
	got := normalizeMessages([]messageRow{
		{
			ID:              101,
			Type:            messageTypeNotification,
			Status:          1,
			Content:         "merged <b>successfully</b>",
			NotificationURL: "https://example.com/pulls/1",
			Source:          "PullRequestMerged",
			CreatedAt:       "2026-06-10 12:00:00",
			TimeAgo:         "1 hour ago",
			Sender: map[string]interface{}{
				"login": "alice",
			},
		},
	})
	want := []messageOutput{
		{
			ID:              101,
			Type:            messageTypeNotification,
			Status:          1,
			Read:            false,
			Source:          "PullRequestMerged",
			Content:         "merged <b>successfully</b>",
			ContentText:     "merged successfully",
			NotificationURL: "https://example.com/pulls/1",
			CreatedAt:       "2026-06-10 12:00:00",
			TimeAgo:         "1 hour ago",
			Sender: map[string]interface{}{
				"login": "alice",
			},
		},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("normalizeMessages() = %#v, want %#v", got, want)
	}
}

func messageListFixture() map[string]interface{} {
	return map[string]interface{}{
		"total_count":         2,
		"type":                messageTypeNotification,
		"unread_notification": 1,
		"unread_atme":         3,
		"messages": []map[string]interface{}{
			{
				"id":               101,
				"status":           1,
				"content":          "your pull request was <b>merged</b>",
				"notification_url": "https://example.com/pulls/1",
				"source":           "PullRequestMerged",
				"created_at":       "2026-06-10 12:00:00",
				"time_ago":         "1 hour ago",
				"type":             messageTypeNotification,
			},
			{
				"id":      202,
				"status":  2,
				"content": "someone <b>@you</b>",
				"type":    messageTypeAtme,
				"sender": map[string]interface{}{
					"login": "bob",
				},
			},
		},
	}
}
