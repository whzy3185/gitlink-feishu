package feishu

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

const reviewWebhookTestSecret = "stage-three-webhook-secret"

type reviewWebhookHarness struct {
	ingress *GitLinkWebhookIngress
	store   *SQLiteReviewGatewayStore
	now     time.Time
}

func newReviewWebhookHarness(t *testing.T, configure func(*GitLinkInstallation)) reviewWebhookHarness {
	t.Helper()
	t.Setenv("STAGE3_WEBHOOK_SECRET", reviewWebhookTestSecret)
	now := time.Date(2026, 8, 4, 11, 0, 0, 0, time.UTC)
	installation := GitLinkInstallation{
		InstallationID: "installation-a", GitLinkHost: "https://www.gitlink.org.cn/",
		OperationMode: "collaborate", AllowedRepositories: []string{"owner/repo"},
		WebhookSecretRef: "env:STAGE3_WEBHOOK_SECRET", Enabled: true,
	}
	if configure != nil {
		configure(&installation)
	}
	bindings := ReviewGatewayBindings{
		SchemaVersion: reviewGatewayBindingSchema,
		Installations: []GitLinkInstallation{installation},
		Bindings: []ReviewChatBinding{{
			ChatID: "chat-a", InstallationID: installation.InstallationID,
			Repositories: []string{"owner/repo"}, Enabled: true,
		}},
	}
	store, err := OpenSQLiteReviewGatewayStore(filepath.Join(t.TempDir(), "webhook.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })
	queue := &recordingReviewEventQueue{}
	processor := NewReviewEventInboxProcessor(store, queue, "webhook-test")
	ingress, err := NewGitLinkWebhookIngress(bindings, store, processor)
	if err != nil {
		t.Fatal(err)
	}
	ingress.now = func() time.Time { return now }
	return reviewWebhookHarness{ingress: ingress, store: store, now: now}
}

func reviewWebhookPayload(action string) []byte {
	payload, _ := json.Marshal(map[string]interface{}{
		"action":       action,
		"repository":   map[string]interface{}{"full_name": "owner/repo"},
		"pull_request": map[string]interface{}{"number": 42},
	})
	return payload
}

func signReviewWebhook(body []byte, timestamp string) string {
	material := body
	if timestamp != "" {
		material = append([]byte(timestamp+"."), body...)
	}
	mac := hmac.New(sha256.New, []byte(reviewWebhookTestSecret))
	_, _ = mac.Write(material)
	return hex.EncodeToString(mac.Sum(nil))
}

func deliverReviewWebhook(h reviewWebhookHarness, body []byte, mutate func(*http.Request)) (int, GitLinkWebhookIngressResult) {
	request := httptest.NewRequest(http.MethodPost, reviewWebhookPathPrefix+"installation-a", bytes.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("X-Gitlink-Event", "pull_request")
	request.Header.Set("X-Gitlink-Delivery", "delivery-42")
	request.Header.Set("X-Gitlink-Signature", signReviewWebhook(body, ""))
	if mutate != nil {
		mutate(request)
	}
	response := httptest.NewRecorder()
	h.ingress.ServeHTTP(response, request)
	var result GitLinkWebhookIngressResult
	_ = json.Unmarshal(response.Body.Bytes(), &result)
	return response.Code, result
}

func TestWebhookRejectsNonJSONContentType(t *testing.T) {
	h := newReviewWebhookHarness(t, nil)
	status, _ := deliverReviewWebhook(h, reviewWebhookPayload("opened"), func(request *http.Request) {
		request.Header.Set("Content-Type", "text/plain")
	})
	if status != http.StatusUnsupportedMediaType {
		t.Fatalf("status=%d", status)
	}
}

func TestWebhookRejectsOversizedBody(t *testing.T) {
	h := newReviewWebhookHarness(t, nil)
	status, _ := deliverReviewWebhook(h, []byte(strings.Repeat("x", reviewWebhookMaxBodyBytes+1)), nil)
	if status != http.StatusRequestEntityTooLarge {
		t.Fatalf("status=%d", status)
	}
}

func TestWebhookRejectsInvalidJSON(t *testing.T) {
	h := newReviewWebhookHarness(t, nil)
	status, result := deliverReviewWebhook(h, []byte(`{"broken":`), nil)
	if status != http.StatusBadRequest || result.Reason != "invalid_json" {
		t.Fatalf("status=%d result=%+v", status, result)
	}
}

func TestWebhookRejectsUnknownInstallation(t *testing.T) {
	h := newReviewWebhookHarness(t, nil)
	body := reviewWebhookPayload("opened")
	status, _ := deliverReviewWebhook(h, body, func(request *http.Request) { request.URL.Path = reviewWebhookPathPrefix + "unknown" })
	if status != http.StatusNotFound {
		t.Fatalf("status=%d", status)
	}
}

func TestWebhookRejectsDisabledInstallation(t *testing.T) {
	h := newReviewWebhookHarness(t, func(installation *GitLinkInstallation) { installation.Enabled = false })
	status, _ := deliverReviewWebhook(h, reviewWebhookPayload("opened"), nil)
	if status != http.StatusNotFound {
		t.Fatalf("status=%d", status)
	}
}

func TestWebhookRejectsInvalidSignature(t *testing.T) {
	h := newReviewWebhookHarness(t, nil)
	status, _ := deliverReviewWebhook(h, reviewWebhookPayload("opened"), func(request *http.Request) {
		request.Header.Set("X-Gitlink-Signature", strings.Repeat("0", sha256.Size*2))
	})
	if status != http.StatusUnauthorized {
		t.Fatalf("status=%d", status)
	}
}

func TestWebhookBodySignatureMode(t *testing.T) {
	h := newReviewWebhookHarness(t, func(installation *GitLinkInstallation) { installation.WebhookSignatureMode = "body_sha256" })
	status, result := deliverReviewWebhook(h, reviewWebhookPayload("opened"), nil)
	if status != http.StatusAccepted || !result.Accepted {
		t.Fatalf("status=%d result=%+v", status, result)
	}
}

func TestWebhookTimestampBodySignatureMode(t *testing.T) {
	h := newReviewWebhookHarness(t, func(installation *GitLinkInstallation) {
		installation.WebhookSignatureMode = "timestamp_body_sha256"
		installation.WebhookTimestampMode = "required"
	})
	body := reviewWebhookPayload("opened")
	timestamp := strconv.FormatInt(h.now.Unix(), 10)
	status, result := deliverReviewWebhook(h, body, func(request *http.Request) {
		request.Header.Set("X-Gitlink-Timestamp", timestamp)
		request.Header.Set("X-Gitlink-Signature", signReviewWebhook(body, timestamp))
	})
	if status != http.StatusAccepted || !result.Accepted {
		t.Fatalf("status=%d result=%+v", status, result)
	}
}

func TestWebhookRequiredTimestampRejectsMissing(t *testing.T) {
	h := newReviewWebhookHarness(t, func(installation *GitLinkInstallation) { installation.WebhookTimestampMode = "required" })
	status, _ := deliverReviewWebhook(h, reviewWebhookPayload("opened"), nil)
	if status != http.StatusUnauthorized {
		t.Fatalf("status=%d", status)
	}
}

func TestWebhookRejectsExpiredTimestamp(t *testing.T) {
	h := newReviewWebhookHarness(t, nil)
	status, _ := deliverReviewWebhook(h, reviewWebhookPayload("opened"), func(request *http.Request) {
		request.Header.Set("X-Gitlink-Timestamp", h.now.Add(-10*time.Minute).Format(time.RFC3339))
	})
	if status != http.StatusUnauthorized {
		t.Fatalf("status=%d", status)
	}
}

func TestWebhookRejectsFutureTimestamp(t *testing.T) {
	h := newReviewWebhookHarness(t, nil)
	status, _ := deliverReviewWebhook(h, reviewWebhookPayload("opened"), func(request *http.Request) {
		request.Header.Set("X-Gitlink-Timestamp", h.now.Add(10*time.Minute).Format(time.RFC3339))
	})
	if status != http.StatusUnauthorized {
		t.Fatalf("status=%d", status)
	}
}

func TestWebhookOptionalTimestampAllowsMissing(t *testing.T) {
	h := newReviewWebhookHarness(t, nil)
	status, result := deliverReviewWebhook(h, reviewWebhookPayload("opened"), nil)
	if status != http.StatusAccepted || !result.Accepted {
		t.Fatalf("status=%d result=%+v", status, result)
	}
}

func TestWebhookRequiredDeliveryRejectsMissing(t *testing.T) {
	h := newReviewWebhookHarness(t, func(installation *GitLinkInstallation) { installation.WebhookDeliveryRequired = true })
	status, _ := deliverReviewWebhook(h, reviewWebhookPayload("opened"), func(request *http.Request) {
		request.Header.Del("X-Gitlink-Delivery")
	})
	if status != http.StatusBadRequest {
		t.Fatalf("status=%d", status)
	}
}

func TestWebhookBodyHashFallback(t *testing.T) {
	h := newReviewWebhookHarness(t, nil)
	status, result := deliverReviewWebhook(h, reviewWebhookPayload("opened"), func(request *http.Request) {
		request.Header.Del("X-Gitlink-Delivery")
	})
	if status != http.StatusAccepted || !result.Accepted {
		t.Fatalf("status=%d result=%+v", status, result)
	}
	record, err := h.store.GetReviewEventInbox(context.Background(), result.EventID)
	if err != nil || record.DeliveryStatus != "body_hash_fallback" {
		t.Fatalf("fallback record=%+v err=%v", record, err)
	}
}

func TestWebhookDuplicateReturnsAccepted(t *testing.T) {
	h := newReviewWebhookHarness(t, nil)
	body := reviewWebhookPayload("opened")
	_, _ = deliverReviewWebhook(h, body, nil)
	status, result := deliverReviewWebhook(h, body, nil)
	if status != http.StatusAccepted || !result.Duplicate {
		t.Fatalf("status=%d result=%+v", status, result)
	}
}

func TestWebhookHandlerDoesNotEnqueueJobSynchronously(t *testing.T) {
	h := newReviewWebhookHarness(t, nil)
	status, _ := deliverReviewWebhook(h, reviewWebhookPayload("opened"), nil)
	var count int
	if err := h.store.db.QueryRow(`SELECT COUNT(*) FROM review_gateway_jobs`).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if status != http.StatusAccepted || count != 0 {
		t.Fatalf("status=%d synchronous jobs=%d", status, count)
	}
}

func TestWebhookHandlerReturns202AfterInboxWrite(t *testing.T) {
	h := newReviewWebhookHarness(t, nil)
	status, result := deliverReviewWebhook(h, reviewWebhookPayload("opened"), nil)
	var count int
	_ = h.store.db.QueryRow(`SELECT COUNT(*) FROM review_event_inbox WHERE event_id=?`, result.EventID).Scan(&count)
	if status != http.StatusAccepted || count != 1 {
		t.Fatalf("status=%d inbox=%d result=%+v", status, count, result)
	}
}
