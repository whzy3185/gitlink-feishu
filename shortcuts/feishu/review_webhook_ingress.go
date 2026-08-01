package feishu

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

const reviewWebhookMaxBodyBytes = 1 << 20

type GitLinkWebhookIngress struct {
	bindings ReviewGatewayBindings
	queue    *ReviewGatewayQueue
	now      func() time.Time
}

type GitLinkWebhookIngressResult struct {
	SchemaVersion string `json:"schema_version"`
	Accepted      bool   `json:"accepted"`
	Duplicate     bool   `json:"duplicate"`
	Queued        int    `json:"queued"`
	Repository    string `json:"repository,omitempty"`
	PRNumber      int    `json:"pr_number,omitempty"`
	DeliveryHash  string `json:"delivery_hash,omitempty"`
	Reason        string `json:"reason,omitempty"`
}

func NewGitLinkWebhookIngress(bindings ReviewGatewayBindings, queue *ReviewGatewayQueue) (*GitLinkWebhookIngress, error) {
	normalized, err := normalizeReviewGatewayBindings(bindings)
	if err != nil {
		return nil, err
	}
	if queue == nil {
		return nil, fmt.Errorf("review gateway queue is required for GitLink webhook ingress")
	}
	return &GitLinkWebhookIngress{bindings: normalized, queue: queue, now: time.Now}, nil
}

func (h *GitLinkWebhookIngress) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	writer.Header().Set("Content-Type", "application/json; charset=utf-8")
	if request.Method != http.MethodPost {
		writer.Header().Set("Allow", http.MethodPost)
		writeGitLinkWebhookResponse(writer, http.StatusMethodNotAllowed, GitLinkWebhookIngressResult{
			SchemaVersion: "gitlink.review-webhook/v1", Reason: "method_not_allowed",
		})
		return
	}
	body, err := io.ReadAll(io.LimitReader(request.Body, reviewWebhookMaxBodyBytes+1))
	if err != nil || len(body) == 0 || len(body) > reviewWebhookMaxBodyBytes {
		writeGitLinkWebhookResponse(writer, http.StatusBadRequest, GitLinkWebhookIngressResult{
			SchemaVersion: "gitlink.review-webhook/v1", Reason: "invalid_payload",
		})
		return
	}
	var payload map[string]interface{}
	if err := json.Unmarshal(body, &payload); err != nil {
		writeGitLinkWebhookResponse(writer, http.StatusBadRequest, GitLinkWebhookIngressResult{
			SchemaVersion: "gitlink.review-webhook/v1", Reason: "invalid_json",
		})
		return
	}
	repository, prNumber := gitLinkWebhookTarget(payload)
	result := GitLinkWebhookIngressResult{
		SchemaVersion: "gitlink.review-webhook/v1",
		Repository:    repository,
		PRNumber:      prNumber,
	}
	if repository == "" || prNumber <= 0 || !gitLinkWebhookIsPullRequest(request, payload) {
		result.Reason = "unsupported_event"
		writeGitLinkWebhookResponse(writer, http.StatusAccepted, result)
		return
	}
	deliveryID := firstNonEmpty(
		request.Header.Get("X-Gitlink-Delivery"),
		request.Header.Get("X-Gitea-Delivery"),
		request.Header.Get("X-GitHub-Delivery"),
	)
	if deliveryID == "" {
		digest := sha256.Sum256(body)
		deliveryID = "body:" + hex.EncodeToString(digest[:16])
	}
	result.DeliveryHash = reviewGatewayHashIdentifier(deliveryID)

	verifiedInstallations, verifyErr := verifyGitLinkWebhookInstallations(
		h.bindings,
		repository,
		body,
		firstNonEmpty(
			request.Header.Get("X-Gitlink-Signature"),
			request.Header.Get("X-Gitea-Signature"),
			request.Header.Get("X-Hub-Signature-256"),
		),
	)
	if verifyErr != nil || len(verifiedInstallations) == 0 {
		result.Reason = "signature_verification_failed"
		writeGitLinkWebhookResponse(writer, http.StatusUnauthorized, result)
		return
	}

	now := h.now().UTC()
	duplicateCount := 0
	for _, binding := range h.bindings.Bindings {
		installation, verified := verifiedInstallations[binding.InstallationID]
		if !verified || !binding.Enabled || !containsReviewGatewayString(binding.Repositories, repository) {
			continue
		}
		dedupeKey := strings.Join([]string{
			"gitlink:webhook", deliveryID, binding.ChatID, repository, strconv.Itoa(prNumber),
		}, ":")
		digest := sha256.Sum256([]byte(dedupeKey))
		job := ReviewGatewayJob{
			SchemaVersion:    reviewGatewayJobSchema,
			JobID:            "job-webhook-" + hex.EncodeToString(digest[:8]),
			DedupeKey:        dedupeKey,
			Status:           "queued",
			Mode:             "preview",
			Action:           "refresh_review_context",
			InstallationID:   installation.InstallationID,
			InstallationMode: installation.OperationMode,
			Repository:       repository,
			Repositories:     append([]string(nil), binding.Repositories...),
			PRNumber:         prNumber,
			ChatID:           binding.ChatID,
			RequestedBy:      "gitlink-webhook:" + installation.InstallationID,
			SourceEventID:    deliveryID,
			NotifyChat:       true,
			CreatedAt:        now.Format(time.RFC3339),
			MutatesGitLink:   false,
			MaxAttempts:      reviewGatewayDefaultMaxAttempts,
			NextAttemptAt:    now.Format(time.RFC3339Nano),
		}
		saved, err := h.queue.EnqueuePreparedJob(request.Context(), job)
		if err != nil {
			result.Reason = "state_store_failed"
			writeGitLinkWebhookResponse(writer, http.StatusServiceUnavailable, result)
			return
		}
		if saved {
			result.Queued++
		} else {
			duplicateCount++
		}
	}
	result.Accepted = result.Queued > 0
	result.Duplicate = result.Queued == 0 && duplicateCount > 0
	switch {
	case result.Accepted:
		result.Reason = "queued"
	case result.Duplicate:
		result.Reason = "duplicate_delivery"
	default:
		result.Reason = "repository_not_bound"
	}
	writeGitLinkWebhookResponse(writer, http.StatusAccepted, result)
}

func verifyGitLinkWebhookInstallations(
	bindings ReviewGatewayBindings,
	repository string,
	body []byte,
	signature string,
) (map[string]GitLinkInstallation, error) {
	signature = strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(signature), "sha256="))
	provided, err := hex.DecodeString(signature)
	if err != nil || len(provided) != sha256.Size {
		return nil, fmt.Errorf("invalid GitLink webhook signature")
	}
	verified := map[string]GitLinkInstallation{}
	for _, installation := range bindings.Installations {
		if !installation.Enabled || !containsReviewGatewayString(installation.AllowedRepositories, repository) ||
			installation.WebhookSecretRef == "" {
			continue
		}
		secret, err := resolveReviewGatewaySecretReference(installation.WebhookSecretRef)
		if err != nil {
			continue
		}
		mac := hmac.New(sha256.New, []byte(secret))
		_, _ = mac.Write(body)
		if hmac.Equal(provided, mac.Sum(nil)) {
			verified[installation.InstallationID] = installation
		}
	}
	if len(verified) == 0 {
		return nil, fmt.Errorf("GitLink webhook signature did not match an enabled installation")
	}
	return verified, nil
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

func gitLinkWebhookTarget(payload map[string]interface{}) (string, int) {
	repositoryObject, _ := payload["repository"].(map[string]interface{})
	repository := firstNonEmpty(
		stringFromWebhookMap(repositoryObject, "full_name"),
		stringFromWebhookMap(repositoryObject, "path_with_namespace"),
	)
	if repository == "" {
		ownerObject, _ := repositoryObject["owner"].(map[string]interface{})
		owner := firstNonEmpty(
			stringFromWebhookMap(ownerObject, "login"),
			stringFromWebhookMap(ownerObject, "username"),
			stringFromWebhookMap(ownerObject, "name"),
		)
		name := stringFromWebhookMap(repositoryObject, "name")
		if owner != "" && name != "" {
			repository = owner + "/" + name
		}
	}
	pullRequest, _ := payload["pull_request"].(map[string]interface{})
	prNumber := integerFromWebhookMap(pullRequest, "number")
	if prNumber <= 0 {
		prNumber = integerFromWebhookMap(pullRequest, "index")
	}
	if prNumber <= 0 {
		prNumber = integerFromWebhookMap(payload, "number")
	}
	if !reviewGatewayRepositoryPattern.MatchString(repository) {
		return "", 0
	}
	return repository, prNumber
}

func gitLinkWebhookIsPullRequest(request *http.Request, payload map[string]interface{}) bool {
	event := strings.ToLower(firstNonEmpty(
		request.Header.Get("X-Gitlink-Event"),
		request.Header.Get("X-Gitea-Event"),
		request.Header.Get("X-GitHub-Event"),
		stringFromWebhookMap(payload, "hook_name"),
	))
	if _, ok := payload["pull_request"].(map[string]interface{}); !ok {
		return false
	}
	return event == "" || strings.Contains(event, "pull_request")
}

func stringFromWebhookMap(values map[string]interface{}, key string) string {
	if values == nil {
		return ""
	}
	value, _ := values[key].(string)
	return strings.TrimSpace(value)
}

func integerFromWebhookMap(values map[string]interface{}, key string) int {
	if values == nil {
		return 0
	}
	switch value := values[key].(type) {
	case float64:
		return int(value)
	case json.Number:
		result, _ := strconv.Atoi(value.String())
		return result
	case string:
		result, _ := strconv.Atoi(strings.TrimSpace(value))
		return result
	default:
		return 0
	}
}

func writeGitLinkWebhookResponse(writer http.ResponseWriter, status int, result GitLinkWebhookIngressResult) {
	writer.WriteHeader(status)
	_ = json.NewEncoder(writer).Encode(result)
}

var _ http.Handler = (*GitLinkWebhookIngress)(nil)
