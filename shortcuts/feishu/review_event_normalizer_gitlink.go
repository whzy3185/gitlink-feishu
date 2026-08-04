package feishu

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
)

func NormalizeGitLinkReviewEvent(
	installationID string,
	headers http.Header,
	body []byte,
	receivedAt time.Time,
) (ReviewEventNormalization, error) {
	installationID = strings.TrimSpace(installationID)
	if installationID == "" {
		return ReviewEventNormalization{}, fmt.Errorf("installation ID is required")
	}
	var payload map[string]interface{}
	if err := json.Unmarshal(body, &payload); err != nil {
		return ReviewEventNormalization{}, fmt.Errorf("decode GitLink event payload: %w", err)
	}
	fingerprint := reviewEventPayloadFingerprint(body)
	rawDelivery, _ := firstReviewEventHeader(headers, "X-Gitlink-Delivery", "X-Gitea-Delivery", "X-GitHub-Delivery")
	deliveryKey, deliveryHash, _ := reviewEventDeliveryKey(installationID, rawDelivery, fingerprint)
	eventName, _ := firstReviewEventHeader(headers, "X-Gitlink-Event", "X-Gitea-Event", "X-GitHub-Event")
	eventType, action := normalizeGitLinkReviewAction(eventName, payload)
	repository, prNumber := extractGitLinkReviewTarget(payload)
	event := NormalizedReviewEvent{
		SchemaVersion: normalizedReviewEventSchema,
		Source:        ReviewEventSourceGitLinkWebhook, InstallationID: installationID,
		DeliveryKey: deliveryKey, DeliveryHash: deliveryHash,
		Repository: repository, PRNumber: prNumber,
		EventType: eventType, Action: action,
		HeadSHA: extractGitLinkReviewString(payload,
			[]string{"pull_request", "head", "sha"},
			[]string{"object_attributes", "last_commit", "id"},
			[]string{"head_sha"}),
		ActorHash:          reviewEventActorHash(payload),
		OccurredAt:         extractGitLinkReviewOccurredAt(payload),
		ReceivedAt:         receivedAt.UTC().Format(time.RFC3339Nano),
		PayloadFingerprint: fingerprint,
	}
	result := ReviewEventNormalization{Event: event, Status: "normalized"}
	if eventType == "" || action == "" {
		result.Status = "ignored"
		result.ReasonCode = "unsupported_action"
		return result, nil
	}
	if repository == "" || prNumber <= 0 {
		result.Status = "ignored"
		result.ReasonCode = "missing_pr_target"
		return result, nil
	}
	event.EventID = reviewEventID(event)
	result.Event = event
	return result, nil
}

func firstReviewEventHeader(headers http.Header, names ...string) (string, string) {
	for _, name := range names {
		if value := strings.TrimSpace(headers.Get(name)); value != "" {
			return value, name
		}
	}
	return "", ""
}

func normalizeGitLinkReviewAction(eventName string, payload map[string]interface{}) (string, string) {
	eventName = strings.ToLower(strings.TrimSpace(eventName))
	rawAction := strings.ToLower(strings.TrimSpace(firstNonEmpty(
		extractGitLinkReviewString(payload, []string{"action"}),
		extractGitLinkReviewString(payload, []string{"object_attributes", "action"}),
		extractGitLinkReviewString(payload, []string{"pull_request", "action"}),
	)))
	if strings.Contains(eventName, ".") && rawAction == "" {
		parts := strings.SplitN(eventName, ".", 2)
		eventName, rawAction = parts[0], parts[1]
	}
	typeName := ""
	switch eventName {
	case "pull_request", "pull_request_hook", "pullrequest":
		typeName = "pull_request"
	case "pull_request_review", "review", "review_hook":
		typeName = "review"
	case "pull_request_review_thread", "review_thread", "review_thread_hook":
		typeName = "review_thread"
	case "ci", "pipeline", "pipeline_hook", "check_run", "status":
		typeName = "ci"
	}
	aliases := map[string]map[string]string{
		"pull_request": {
			"open": "opened", "opened": "opened", "update": "updated", "updated": "updated",
			"synchronize": "synchronized", "synchronized": "synchronized", "sync": "synchronized",
			"reopen": "reopened", "reopened": "reopened", "ready_for_review": "ready_for_review",
			"merge": "merged", "merged": "merged", "close": "closed", "closed": "closed",
		},
		"review": {
			"create": "created", "created": "created", "submitted": "created",
			"update": "updated", "updated": "updated", "edited": "updated",
			"dismiss": "dismissed", "dismissed": "dismissed",
		},
		"review_thread": {
			"create": "created", "created": "created", "update": "updated", "updated": "updated",
			"resolve": "resolved", "resolved": "resolved",
		},
		"ci": {
			"start": "started", "started": "started", "requested": "started", "in_progress": "started",
			"complete": "completed", "completed": "completed", "success": "completed", "succeeded": "completed",
			"fail": "failed", "failed": "failed", "failure": "failed",
		},
	}
	if canonical := aliases[typeName][rawAction]; canonical != "" {
		return typeName, typeName + "." + canonical
	}
	return "", ""
}

func extractGitLinkReviewTarget(payload map[string]interface{}) (string, int) {
	repository := extractGitLinkReviewString(payload,
		[]string{"repository", "full_name"},
		[]string{"repository", "path_with_namespace"})
	if repository == "" {
		owner := extractGitLinkReviewString(payload,
			[]string{"repository", "owner", "login"},
			[]string{"repository", "owner", "username"},
			[]string{"repository", "owner"})
		name := extractGitLinkReviewString(payload, []string{"repository", "name"})
		if owner != "" && name != "" {
			repository = owner + "/" + name
		}
	}
	number := extractGitLinkReviewInt(payload,
		[]string{"pull_request", "number"},
		[]string{"pull_request", "index"},
		[]string{"object_attributes", "iid"},
		[]string{"number"})
	return repository, number
}

func extractGitLinkReviewString(payload map[string]interface{}, paths ...[]string) string {
	for _, path := range paths {
		var current interface{} = payload
		for _, key := range path {
			object, ok := current.(map[string]interface{})
			if !ok {
				current = nil
				break
			}
			current = object[key]
		}
		switch value := current.(type) {
		case string:
			if strings.TrimSpace(value) != "" {
				return strings.TrimSpace(value)
			}
		case json.Number:
			return value.String()
		case float64:
			return strconv.FormatInt(int64(value), 10)
		}
	}
	return ""
}

func extractGitLinkReviewInt(payload map[string]interface{}, paths ...[]string) int {
	value := extractGitLinkReviewString(payload, paths...)
	number, _ := strconv.Atoi(value)
	return number
}

func reviewEventActorHash(payload map[string]interface{}) string {
	actor := extractGitLinkReviewString(payload,
		[]string{"sender", "id"}, []string{"sender", "login"},
		[]string{"user", "id"}, []string{"user", "username"},
		[]string{"actor", "id"}, []string{"actor", "login"})
	if actor == "" {
		return ""
	}
	return reviewResourceIdentifierHash(actor)
}

func extractGitLinkReviewOccurredAt(payload map[string]interface{}) string {
	value := extractGitLinkReviewString(payload,
		[]string{"timestamp"}, []string{"updated_at"}, []string{"created_at"},
		[]string{"object_attributes", "updated_at"})
	if value == "" {
		return ""
	}
	for _, layout := range []string{time.RFC3339Nano, time.RFC3339} {
		if parsed, err := time.Parse(layout, value); err == nil {
			return parsed.UTC().Format(time.RFC3339Nano)
		}
	}
	return ""
}
