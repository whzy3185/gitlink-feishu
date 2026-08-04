package feishu

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

const reviewWebhookMaxBodyBytes = 1 << 20
const reviewWebhookPathPrefix = "/webhooks/gitlink/"

type GitLinkWebhookIngress struct {
	installations map[string]GitLinkInstallation
	store         *SQLiteReviewGatewayStore
	processor     *ReviewEventInboxProcessor
	now           func() time.Time
}

type GitLinkWebhookIngressResult struct {
	SchemaVersion string `json:"schema_version"`
	Accepted      bool   `json:"accepted"`
	Duplicate     bool   `json:"duplicate"`
	EventID       string `json:"event_id,omitempty"`
	Repository    string `json:"repository,omitempty"`
	PRNumber      int    `json:"pr_number,omitempty"`
	DeliveryHash  string `json:"delivery_hash,omitempty"`
	Reason        string `json:"reason,omitempty"`
}

func NewGitLinkWebhookIngress(
	bindings ReviewGatewayBindings,
	store *SQLiteReviewGatewayStore,
	processor *ReviewEventInboxProcessor,
) (*GitLinkWebhookIngress, error) {
	normalized, err := normalizeReviewGatewayBindings(bindings)
	if err != nil {
		return nil, err
	}
	if store == nil || processor == nil {
		return nil, fmt.Errorf("Review event inbox store and processor are required for GitLink webhook ingress")
	}
	return &GitLinkWebhookIngress{
		installations: reviewGatewayInstallationMap(normalized),
		store:         store, processor: processor, now: time.Now,
	}, nil
}

func (h *GitLinkWebhookIngress) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	writer.Header().Set("Content-Type", "application/json; charset=utf-8")
	respond := func(status int, result GitLinkWebhookIngressResult) {
		result.SchemaVersion = "gitlink.review-webhook/v2"
		writer.WriteHeader(status)
		_ = json.NewEncoder(writer).Encode(result)
	}
	if request.Method != http.MethodPost {
		writer.Header().Set("Allow", http.MethodPost)
		respond(http.StatusMethodNotAllowed, GitLinkWebhookIngressResult{Reason: "method_not_allowed"})
		return
	}
	mediaType, _, err := mime.ParseMediaType(request.Header.Get("Content-Type"))
	if err != nil || !strings.EqualFold(mediaType, "application/json") {
		respond(http.StatusUnsupportedMediaType, GitLinkWebhookIngressResult{Reason: "json_content_type_required"})
		return
	}
	installationID, ok := gitLinkWebhookInstallationFromPath(request.URL.Path)
	if !ok {
		respond(http.StatusNotFound, GitLinkWebhookIngressResult{Reason: "unknown_installation"})
		return
	}
	installation, ok := h.installations[installationID]
	if !ok || !installation.Enabled {
		respond(http.StatusNotFound, GitLinkWebhookIngressResult{Reason: "unknown_installation"})
		return
	}
	body, err := io.ReadAll(io.LimitReader(request.Body, reviewWebhookMaxBodyBytes+1))
	if err != nil || len(body) == 0 {
		respond(http.StatusBadRequest, GitLinkWebhookIngressResult{Reason: "invalid_payload"})
		return
	}
	if len(body) > reviewWebhookMaxBodyBytes {
		respond(http.StatusRequestEntityTooLarge, GitLinkWebhookIngressResult{Reason: "payload_too_large"})
		return
	}
	now := h.now().UTC()
	timestamp, timestampStatus, err := validateGitLinkWebhookTimestamp(installation, request.Header, now)
	if err != nil {
		respond(http.StatusUnauthorized, GitLinkWebhookIngressResult{Reason: err.Error()})
		return
	}
	if err := verifyGitLinkWebhookSignature(installation, request.Header, timestamp, body); err != nil {
		respond(http.StatusUnauthorized, GitLinkWebhookIngressResult{Reason: "signature_verification_failed"})
		return
	}
	var validJSON interface{}
	if err := json.Unmarshal(body, &validJSON); err != nil {
		respond(http.StatusBadRequest, GitLinkWebhookIngressResult{Reason: "invalid_json"})
		return
	}
	deliveryID, _ := firstReviewEventHeader(request.Header,
		"X-Gitlink-Delivery", "X-Gitea-Delivery", "X-GitHub-Delivery")
	if installation.WebhookDeliveryRequired && strings.TrimSpace(deliveryID) == "" {
		respond(http.StatusBadRequest, GitLinkWebhookIngressResult{Reason: "missing_delivery"})
		return
	}
	normalization, err := NormalizeGitLinkReviewEvent(installationID, request.Header, body, now)
	if err != nil {
		respond(http.StatusBadRequest, GitLinkWebhookIngressResult{Reason: "invalid_json"})
		return
	}
	if normalization.Event.Repository != "" &&
		!containsReviewGatewayString(installation.AllowedRepositories, normalization.Event.Repository) {
		normalization.Status = "ignored"
		normalization.ReasonCode = "repository_not_authorized"
		normalization.Event.EventID = ""
	}
	_, _, deliveryStatus := reviewEventDeliveryKey(
		installationID, deliveryID, normalization.Event.PayloadFingerprint,
	)
	insert, err := h.store.InsertReviewEventInbox(request.Context(), normalization, ReviewEventIngressMetadata{
		SignatureStatus: "verified", TimestampStatus: timestampStatus,
		DeliveryStatus: deliveryStatus, RawPayload: body,
	})
	if err != nil {
		respond(http.StatusServiceUnavailable, GitLinkWebhookIngressResult{Reason: "inbox_save_failed"})
		return
	}
	h.processor.Wake()
	result := GitLinkWebhookIngressResult{
		Accepted: true, Duplicate: insert.Duplicate,
		EventID:    insert.Record.Event.EventID,
		Repository: insert.Record.Event.Repository, PRNumber: insert.Record.Event.PRNumber,
		DeliveryHash: insert.Record.Event.DeliveryHash,
		Reason:       firstNonEmpty(insert.Record.ReasonCode, "accepted"),
	}
	if insert.Duplicate {
		result.Reason = "duplicate_delivery"
	}
	respond(http.StatusAccepted, result)
}

func gitLinkWebhookInstallationFromPath(path string) (string, bool) {
	if !strings.HasPrefix(path, reviewWebhookPathPrefix) {
		return "", false
	}
	installationID := strings.Trim(strings.TrimPrefix(path, reviewWebhookPathPrefix), "/")
	if installationID == "" || strings.Contains(installationID, "/") {
		return "", false
	}
	return installationID, true
}

func validateGitLinkWebhookTimestamp(
	installation GitLinkInstallation, headers http.Header, now time.Time,
) (string, string, error) {
	if installation.WebhookTimestampMode == "disabled" {
		return "", "disabled", nil
	}
	raw, _ := firstReviewEventHeader(headers,
		"X-Gitlink-Timestamp", "X-Gitea-Timestamp", "X-Webhook-Timestamp")
	if raw == "" {
		if installation.WebhookTimestampMode == "required" {
			return "", "missing", fmt.Errorf("missing_timestamp")
		}
		return "", "missing_allowed", nil
	}
	parsed, err := parseGitLinkWebhookTimestamp(raw)
	if err != nil {
		return "", "invalid", fmt.Errorf("invalid_timestamp")
	}
	skew := now.Sub(parsed)
	limit := time.Duration(installation.WebhookMaxSkewSeconds) * time.Second
	if skew > limit {
		return "", "expired", fmt.Errorf("expired_timestamp")
	}
	if skew < -limit {
		return "", "future", fmt.Errorf("future_timestamp")
	}
	return raw, "valid", nil
}

func parseGitLinkWebhookTimestamp(value string) (time.Time, error) {
	value = strings.TrimSpace(value)
	if seconds, err := strconv.ParseInt(value, 10, 64); err == nil {
		return time.Unix(seconds, 0).UTC(), nil
	}
	for _, layout := range []string{time.RFC3339Nano, time.RFC3339} {
		if parsed, err := time.Parse(layout, value); err == nil {
			return parsed.UTC(), nil
		}
	}
	return time.Time{}, fmt.Errorf("invalid webhook timestamp")
}

func verifyGitLinkWebhookSignature(
	installation GitLinkInstallation, headers http.Header, timestamp string, body []byte,
) error {
	signature, _ := firstReviewEventHeader(headers,
		"X-Gitlink-Signature", "X-Gitea-Signature", "X-Hub-Signature-256")
	signature = strings.TrimPrefix(strings.TrimSpace(signature), "sha256=")
	provided, err := hex.DecodeString(signature)
	if err != nil || len(provided) != sha256.Size {
		return fmt.Errorf("invalid GitLink webhook signature")
	}
	secret, err := resolveReviewGatewaySecretReference(installation.WebhookSecretRef)
	if err != nil {
		return err
	}
	material := body
	if installation.WebhookSignatureMode == "timestamp_body_sha256" {
		if timestamp == "" {
			return fmt.Errorf("timestamp is required by signature mode")
		}
		material = append(append([]byte(timestamp+"."), body...), []byte{}...)
	}
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write(material)
	if !hmac.Equal(provided, mac.Sum(nil)) {
		return fmt.Errorf("GitLink webhook signature mismatch")
	}
	return nil
}

func resolveReviewGatewaySecretReference(reference string) (string, error) {
	reference = strings.TrimSpace(reference)
	if !reviewGatewaySecretReferencePattern.MatchString(reference) {
		return "", fmt.Errorf("secret reference must use env:VARIABLE")
	}
	name := strings.TrimPrefix(reference, "env:")
	value, ok := os.LookupEnv(name)
	if !ok || strings.TrimSpace(value) == "" {
		return "", fmt.Errorf("secret reference %q is not configured", reference)
	}
	return value, nil
}

var _ http.Handler = (*GitLinkWebhookIngress)(nil)
