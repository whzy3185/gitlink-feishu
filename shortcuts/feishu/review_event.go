package feishu

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"
	"time"
)

const normalizedReviewEventSchema = "review.event/v1"

const (
	ReviewEventSourceGitLinkWebhook          = "gitlink_webhook"
	ReviewEventSourceManualReplay            = "manual_replay"
	ReviewEventSourceScheduledReconciliation = "scheduled_reconciliation"
)

type NormalizedReviewEvent struct {
	SchemaVersion      string `json:"schema_version"`
	EventID            string `json:"event_id"`
	Source             string `json:"source"`
	InstallationID     string `json:"installation_id"`
	DeliveryKey        string `json:"delivery_key"`
	DeliveryHash       string `json:"delivery_hash"`
	Repository         string `json:"repository"`
	PRNumber           int    `json:"pr_number"`
	EventType          string `json:"event_type"`
	Action             string `json:"action"`
	HeadSHA            string `json:"head_sha,omitempty"`
	ActorHash          string `json:"actor_hash,omitempty"`
	OccurredAt         string `json:"occurred_at,omitempty"`
	ReceivedAt         string `json:"received_at"`
	PayloadFingerprint string `json:"payload_fingerprint"`
	Synthetic          bool   `json:"synthetic"`
	ReplayOf           string `json:"replay_of,omitempty"`
}

type ReviewEventNormalization struct {
	Event      NormalizedReviewEvent
	Status     string
	ReasonCode string
}

func reviewEventPayloadFingerprint(body []byte) string {
	digest := sha256.Sum256(body)
	return hex.EncodeToString(digest[:])
}

func reviewEventDeliveryKey(installationID, rawDeliveryID, payloadFingerprint string) (string, string, string) {
	rawDeliveryID = strings.TrimSpace(rawDeliveryID)
	material := installationID + ":" + rawDeliveryID
	status := "header"
	if rawDeliveryID == "" {
		material = installationID + ":body:" + payloadFingerprint
		status = "body_hash_fallback"
	}
	keyDigest := sha256.Sum256([]byte(material))
	hashDigest := sha256.Sum256([]byte(firstNonEmpty(rawDeliveryID, "body:"+payloadFingerprint)))
	return hex.EncodeToString(keyDigest[:]), hex.EncodeToString(hashDigest[:]), status
}

func reviewEventID(event NormalizedReviewEvent) string {
	material := strings.Join([]string{
		event.InstallationID, event.DeliveryKey, event.EventType, event.Action,
		event.Repository, strconv.Itoa(event.PRNumber),
	}, "\x00")
	digest := sha256.Sum256([]byte(material))
	return hex.EncodeToString(digest[:])
}

func reviewEventSyntheticDeliveryKey(source, installationID, requestKey string) string {
	digest := sha256.Sum256([]byte(strings.Join([]string{source, installationID, requestKey}, "\x00")))
	return hex.EncodeToString(digest[:])
}

func validateNormalizedReviewEvent(event NormalizedReviewEvent) error {
	if event.SchemaVersion != normalizedReviewEventSchema {
		return fmt.Errorf("unsupported normalized Review event schema %q", event.SchemaVersion)
	}
	switch event.Source {
	case ReviewEventSourceGitLinkWebhook, ReviewEventSourceManualReplay, ReviewEventSourceScheduledReconciliation:
	default:
		return fmt.Errorf("unsupported normalized Review event source %q", event.Source)
	}
	switch event.EventType {
	case "pull_request", "review", "review_thread", "ci":
	default:
		return fmt.Errorf("unsupported normalized Review event type %q", event.EventType)
	}
	if event.EventID == "" || event.InstallationID == "" || event.DeliveryKey == "" ||
		event.Repository == "" || event.PRNumber <= 0 || event.Action == "" || event.ReceivedAt == "" ||
		event.PayloadFingerprint == "" {
		return fmt.Errorf("normalized Review event is incomplete")
	}
	if _, err := time.Parse(time.RFC3339Nano, event.ReceivedAt); err != nil {
		return fmt.Errorf("invalid received_at: %w", err)
	}
	return nil
}
