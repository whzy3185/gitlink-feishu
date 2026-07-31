package wecom

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gitlink-org/gitlink-cli/internal/collab"
)

func TestBuildReviewPayloadPreservesCanonicalSemantics(t *testing.T) {
	item := collab.WorkItem{
		SchemaVersion:       collab.WorkItemSchema,
		PRKey:               "Gitlink/gitlink-cli#431",
		Repository:          "Gitlink/gitlink-cli",
		PRNumber:            431,
		GitLinkURL:          "https://www.gitlink.org.cn/Gitlink/gitlink-cli/pulls/431",
		ReviewStage:         "human_reviewing",
		Decision:            "pending",
		CollectionStatus:    "complete",
		CollaborationStatus: "reviewing",
		AssignedTo:          "reviewer",
		NextStep:            "complete current patchset Review",
	}
	payload := BuildReviewPayload(item)
	if payload.MsgType != "template_card" ||
		payload.TemplateCard.MainTitle.Title != "Gitlink/gitlink-cli PR #431" {
		t.Fatalf("payload = %#v", payload)
	}
	encoded, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}
	text := string(encoded)
	for _, expected := range []string{
		"Gitlink/gitlink-cli#431",
		"human_reviewing",
		"complete current patchset Review",
		"GitLink 写入 0",
	} {
		if !strings.Contains(text, expected) {
			t.Fatalf("payload missing %q: %s", expected, text)
		}
	}
}

func TestSendWebhookUsesTemplateCardAndHandlesResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodPost {
			t.Fatalf("method = %s", request.Method)
		}
		var payload ReviewPayload
		if err := json.NewDecoder(request.Body).Decode(&payload); err != nil {
			t.Fatalf("decode payload: %v", err)
		}
		if payload.MsgType != "template_card" {
			t.Fatalf("msgtype = %s", payload.MsgType)
		}
		_, _ = w.Write([]byte(`{"errcode":0,"errmsg":"ok"}`))
	}))
	defer server.Close()
	errCode, errMsg, err := sendWebhook(
		context.Background(),
		server.Client(),
		server.URL,
		BuildReviewPayload(collab.WorkItem{
			Repository: "owner/repo",
			PRNumber:   42,
			PRKey:      "owner/repo#42",
		}),
	)
	if err != nil || errCode != 0 || errMsg != "ok" {
		t.Fatalf("sendWebhook = %d, %q, %v", errCode, errMsg, err)
	}
}

func TestWebhookRedactionRemovesKey(t *testing.T) {
	redacted := redactWebhook(
		"https://qyapi.weixin.qq.com/cgi-bin/webhook/send?key=super-secret",
	)
	if strings.Contains(redacted, "super-secret") || !strings.Contains(redacted, "%2A%2A%2A") {
		t.Fatalf("redacted webhook = %s", redacted)
	}
}

func TestReviewCoreRequiresLoopbackAndBearerToken(t *testing.T) {
	handler := &ReviewCoreHandler{CoreToken: "local-core-token"}
	body := `{
		"schema_version":"gitlink.collab-inbound/v1",
		"event_id":"event-1",
		"platform":"wecom",
		"chat_id":"chat-1",
		"user_id":"user-1",
		"kind":"message",
		"text":"帮助",
		"received_at":"2026-07-31T08:00:00Z"
	}`

	unauthorized := httptest.NewRequest(http.MethodPost, "/v1/review/inbound", strings.NewReader(body))
	unauthorized.RemoteAddr = "127.0.0.1:12345"
	unauthorizedResult := httptest.NewRecorder()
	handler.ServeHTTP(unauthorizedResult, unauthorized)
	if unauthorizedResult.Code != http.StatusUnauthorized {
		t.Fatalf("unauthorized status = %d", unauthorizedResult.Code)
	}

	nonLoopback := httptest.NewRequest(http.MethodPost, "/v1/review/inbound", strings.NewReader(body))
	nonLoopback.RemoteAddr = "192.0.2.10:12345"
	nonLoopback.Header.Set("Authorization", "Bearer local-core-token")
	nonLoopbackResult := httptest.NewRecorder()
	handler.ServeHTTP(nonLoopbackResult, nonLoopback)
	if nonLoopbackResult.Code != http.StatusForbidden {
		t.Fatalf("non-loopback status = %d", nonLoopbackResult.Code)
	}

	allowed := httptest.NewRequest(http.MethodPost, "/v1/review/inbound", strings.NewReader(body))
	allowed.RemoteAddr = "127.0.0.1:12345"
	allowed.Header.Set("Authorization", "Bearer local-core-token")
	allowedResult := httptest.NewRecorder()
	handler.ServeHTTP(allowedResult, allowed)
	if allowedResult.Code != http.StatusOK {
		t.Fatalf("allowed status = %d, body=%s", allowedResult.Code, allowedResult.Body.String())
	}
	var result ReviewCoreResult
	if err := json.Unmarshal(allowedResult.Body.Bytes(), &result); err != nil {
		t.Fatalf("decode result: %v", err)
	}
	if result.SchemaVersion != reviewCoreResultSchema ||
		result.GitLinkWrites != 0 ||
		!strings.Contains(result.Markdown, "GitLink 写入：0") {
		t.Fatalf("result = %#v", result)
	}
}

func TestReviewCoreListenAddressIsLoopbackOnly(t *testing.T) {
	for _, allowed := range []string{"127.0.0.1:8765", "localhost:8765", "[::1]:8765"} {
		if err := validateLoopbackListenAddress(allowed); err != nil {
			t.Fatalf("%s rejected: %v", allowed, err)
		}
	}
	if err := validateLoopbackListenAddress("0.0.0.0:8765"); err == nil {
		t.Fatal("non-loopback listen address must be rejected")
	}
}
