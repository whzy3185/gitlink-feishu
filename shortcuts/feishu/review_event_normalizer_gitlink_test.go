package feishu

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"
)

var reviewEventTestTime = time.Date(2026, 8, 4, 10, 0, 0, 0, time.UTC)

func normalizeReviewEventTest(t *testing.T, eventName, action string) ReviewEventNormalization {
	t.Helper()
	body, _ := json.Marshal(map[string]interface{}{
		"action":     action,
		"repository": map[string]interface{}{"full_name": "owner/repo"},
		"pull_request": map[string]interface{}{
			"number": 42,
			"head":   map[string]interface{}{"sha": "head-42"},
		},
		"sender": map[string]interface{}{"login": "alice@example.invalid"},
	})
	headers := http.Header{}
	headers.Set("X-Gitlink-Event", eventName)
	headers.Set("X-Gitlink-Delivery", "raw-delivery-42")
	result, err := NormalizeGitLinkReviewEvent("installation-a", headers, body, reviewEventTestTime)
	if err != nil {
		t.Fatalf("NormalizeGitLinkReviewEvent: %v", err)
	}
	return result
}

func requireNormalizedReviewAction(t *testing.T, eventName, action, want string) {
	t.Helper()
	result := normalizeReviewEventTest(t, eventName, action)
	if result.Status != "normalized" || result.Event.Action != want {
		t.Fatalf("normalization = status:%q action:%q reason:%q", result.Status, result.Event.Action, result.ReasonCode)
	}
}

func TestNormalizePullRequestOpened(t *testing.T) {
	requireNormalizedReviewAction(t, "pull_request", "opened", "pull_request.opened")
}

func TestNormalizePullRequestSynchronized(t *testing.T) {
	requireNormalizedReviewAction(t, "pull_request", "synchronize", "pull_request.synchronized")
}

func TestNormalizePullRequestMerged(t *testing.T) {
	requireNormalizedReviewAction(t, "pull_request", "merged", "pull_request.merged")
}

func TestNormalizeReviewCreated(t *testing.T) {
	requireNormalizedReviewAction(t, "pull_request_review", "submitted", "review.created")
}

func TestNormalizeReviewDismissed(t *testing.T) {
	requireNormalizedReviewAction(t, "review", "dismissed", "review.dismissed")
}

func TestNormalizeThreadResolved(t *testing.T) {
	requireNormalizedReviewAction(t, "review_thread", "resolved", "review_thread.resolved")
}

func TestNormalizeCICompleted(t *testing.T) {
	requireNormalizedReviewAction(t, "check_run", "completed", "ci.completed")
}

func TestUnknownActionIsIgnored(t *testing.T) {
	result := normalizeReviewEventTest(t, "pull_request", "teleported")
	if result.Status != "ignored" || result.ReasonCode != "unsupported_action" || result.Event.EventID != "" {
		t.Fatalf("unknown action normalization = %+v", result)
	}
}

func TestMissingPRTargetIsIgnored(t *testing.T) {
	body := []byte(`{"action":"opened","repository":{"full_name":"owner/repo"}}`)
	headers := http.Header{"X-Gitlink-Event": []string{"pull_request"}, "X-Gitlink-Delivery": []string{"delivery-missing-target"}}
	result, err := NormalizeGitLinkReviewEvent("installation-a", headers, body, reviewEventTestTime)
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != "ignored" || result.ReasonCode != "missing_pr_target" {
		t.Fatalf("missing target normalization = %+v", result)
	}
}

func TestEventIDIsDeterministic(t *testing.T) {
	first := normalizeReviewEventTest(t, "pull_request", "opened")
	second := normalizeReviewEventTest(t, "pull_request", "opened")
	if first.Event.EventID == "" || first.Event.EventID != second.Event.EventID {
		t.Fatalf("event IDs are not deterministic: %q %q", first.Event.EventID, second.Event.EventID)
	}
}

func TestDeliveryKeyDoesNotPersistRawDeliveryID(t *testing.T) {
	result := normalizeReviewEventTest(t, "pull_request", "opened")
	encoded, _ := json.Marshal(result.Event)
	if strings.Contains(string(encoded), "raw-delivery-42") || len(result.Event.DeliveryKey) != 64 || len(result.Event.DeliveryHash) != 64 {
		t.Fatalf("raw delivery identity leaked: %s", encoded)
	}
}

func TestActorIsPersistedOnlyAsHash(t *testing.T) {
	result := normalizeReviewEventTest(t, "pull_request", "opened")
	encoded, _ := json.Marshal(result.Event)
	if result.Event.ActorHash == "" || strings.Contains(string(encoded), "alice") || strings.Contains(string(encoded), "example.invalid") {
		t.Fatalf("actor was not reduced to a hash: %s", encoded)
	}
}
