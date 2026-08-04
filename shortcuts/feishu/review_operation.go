package feishu

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"
)

const reviewOperationDesiredJSONMaxBytes = 256 * 1024

type ReviewOperationKind string

const (
	ReviewOperationCanonicalCardUpsert ReviewOperationKind = "canonical_card_upsert"
	ReviewOperationReplySend           ReviewOperationKind = "reply_send"
	ReviewOperationBitableUpsert       ReviewOperationKind = "bitable_upsert"
	ReviewOperationDocSnapshotUpsert   ReviewOperationKind = "doc_snapshot_upsert"
	ReviewOperationTaskUpsert          ReviewOperationKind = "task_upsert"
)

type ReviewOperationStatus string

const (
	ReviewOperationPending             ReviewOperationStatus = "pending"
	ReviewOperationLeased              ReviewOperationStatus = "leased"
	ReviewOperationWriting             ReviewOperationStatus = "writing"
	ReviewOperationSucceeded           ReviewOperationStatus = "succeeded"
	ReviewOperationUnchanged           ReviewOperationStatus = "unchanged"
	ReviewOperationRetryScheduled      ReviewOperationStatus = "retry_scheduled"
	ReviewOperationFailedTerminal      ReviewOperationStatus = "failed_terminal"
	ReviewOperationUnknown             ReviewOperationStatus = "unknown"
	ReviewOperationNeedsReconciliation ReviewOperationStatus = "needs_reconciliation"
	ReviewOperationStale               ReviewOperationStatus = "stale"
	ReviewOperationDeadLetter          ReviewOperationStatus = "dead_letter"
	ReviewOperationCancelled           ReviewOperationStatus = "cancelled"
	ReviewOperationBlocked             ReviewOperationStatus = "blocked"
)

type ReviewOperationMutationStatus string

const (
	ReviewMutationNotStarted      ReviewOperationMutationStatus = "not_started"
	ReviewMutationRequestStarted  ReviewOperationMutationStatus = "request_started"
	ReviewMutationRemoteConfirmed ReviewOperationMutationStatus = "remote_confirmed"
	ReviewMutationRemoteUnknown   ReviewOperationMutationStatus = "remote_unknown"
	ReviewMutationLocalConfirmed  ReviewOperationMutationStatus = "local_confirmed"
)

type ReviewOperationRetrySafety string

const (
	ReviewRetryIdempotent    ReviewOperationRetrySafety = "idempotent"
	ReviewRetryReconcilable  ReviewOperationRetrySafety = "reconcilable"
	ReviewRetryNonIdempotent ReviewOperationRetrySafety = "non_idempotent"
)

type ReviewOperationDependencyPolicy string

const (
	ReviewDependencyNone     ReviewOperationDependencyPolicy = "none"
	ReviewDependencySuccess  ReviewOperationDependencyPolicy = "success"
	ReviewDependencyTerminal ReviewOperationDependencyPolicy = "terminal"
)

type ReviewOperation struct {
	OperationID            string                          `json:"operation_id"`
	OperationKind          ReviewOperationKind             `json:"operation_kind"`
	QueueClass             string                          `json:"queue_class"`
	InstallationID         string                          `json:"installation_id,omitempty"`
	ChatID                 string                          `json:"chat_id,omitempty"`
	Repository             string                          `json:"repository,omitempty"`
	PRNumber               int                             `json:"pr_number,omitempty"`
	SourceJobID            string                          `json:"source_job_id,omitempty"`
	SourceEventID          string                          `json:"source_event_id,omitempty"`
	ConsumerID             string                          `json:"consumer_id,omitempty"`
	WorkItemKey            string                          `json:"work_item_key,omitempty"`
	ResourceType           string                          `json:"resource_type,omitempty"`
	IdempotencyKey         string                          `json:"idempotency_key"`
	DesiredFingerprint     string                          `json:"desired_fingerprint,omitempty"`
	AppliedFingerprint     string                          `json:"applied_fingerprint,omitempty"`
	ExpectedRemoteID       string                          `json:"expected_remote_id,omitempty"`
	RemoteID               string                          `json:"remote_id,omitempty"`
	DesiredJSON            string                          `json:"desired_json"`
	DependencyOperationID  string                          `json:"dependency_operation_id,omitempty"`
	DependencyPolicy       ReviewOperationDependencyPolicy `json:"dependency_policy"`
	RetrySafety            ReviewOperationRetrySafety      `json:"retry_safety"`
	Status                 ReviewOperationStatus           `json:"status"`
	ErrorClass             string                          `json:"error_class,omitempty"`
	ErrorCode              string                          `json:"error_code,omitempty"`
	ErrorSummary           string                          `json:"error_summary,omitempty"`
	HTTPStatus             int                             `json:"http_status,omitempty"`
	MutationStatus         ReviewOperationMutationStatus   `json:"mutation_status"`
	RequiresReconciliation bool                            `json:"requires_reconciliation"`
	AttemptCount           int                             `json:"attempt_count"`
	MaxAttempts            int                             `json:"max_attempts"`
	NextAttemptAt          string                          `json:"next_attempt_at,omitempty"`
	RetryAfterAt           string                          `json:"retry_after_at,omitempty"`
	LeaseOwner             string                          `json:"lease_owner,omitempty"`
	LeaseExpiresAt         string                          `json:"lease_expires_at,omitempty"`
	CreatedAt              string                          `json:"created_at"`
	UpdatedAt              string                          `json:"updated_at"`
	CompletedAt            string                          `json:"completed_at,omitempty"`
}

type ReviewOperationAttempt struct {
	AttemptID         string                        `json:"attempt_id"`
	OperationID       string                        `json:"operation_id"`
	AttemptNumber     int                           `json:"attempt_number"`
	LeaseOwnerHash    string                        `json:"lease_owner_hash,omitempty"`
	StartedAt         string                        `json:"started_at"`
	FinishedAt        string                        `json:"finished_at,omitempty"`
	MutationStatus    ReviewOperationMutationStatus `json:"mutation_status"`
	ErrorClass        string                        `json:"error_class,omitempty"`
	ErrorCode         string                        `json:"error_code,omitempty"`
	ErrorSummary      string                        `json:"error_summary,omitempty"`
	HTTPStatus        int                           `json:"http_status,omitempty"`
	RetryAfterAt      string                        `json:"retry_after_at,omitempty"`
	ResultFingerprint string                        `json:"result_fingerprint,omitempty"`
}

var reviewOperationSecretKeyPattern = regexp.MustCompile(`(?i)(authorization|app_secret|tenant_token|gitlink_token|cookie|webhook_secret|access_token|refresh_token)`)

func NewReviewOperation(kind ReviewOperationKind, scope ReviewGatewayJob, workItemKey, resourceType string, desired interface{}, retrySafety ReviewOperationRetrySafety, now time.Time) (ReviewOperation, error) {
	desiredJSON, fingerprint, err := boundedReviewOperationPayload(desired)
	if err != nil {
		return ReviewOperation{}, err
	}
	idempotencyKey := reviewOperationIdempotencyKey(kind, workItemKey, fingerprint)
	operationID := stableKey("review-operation", string(kind), idempotencyKey)
	op := ReviewOperation{
		OperationID:        operationID,
		OperationKind:      kind,
		QueueClass:         reviewOperationQueueClass(kind),
		InstallationID:     strings.TrimSpace(scope.InstallationID),
		ChatID:             strings.TrimSpace(scope.ChatID),
		Repository:         strings.TrimSpace(scope.Repository),
		PRNumber:           scope.PRNumber,
		SourceJobID:        strings.TrimSpace(scope.JobID),
		SourceEventID:      strings.TrimSpace(scope.SourceEventID),
		WorkItemKey:        strings.TrimSpace(workItemKey),
		ResourceType:       strings.TrimSpace(resourceType),
		IdempotencyKey:     idempotencyKey,
		DesiredFingerprint: fingerprint,
		DesiredJSON:        desiredJSON,
		DependencyPolicy:   ReviewDependencyNone,
		RetrySafety:        retrySafety,
		Status:             ReviewOperationPending,
		MutationStatus:     ReviewMutationNotStarted,
		MaxAttempts:        3,
		NextAttemptAt:      reviewGatewayTimestamp(now),
		CreatedAt:          reviewGatewayTimestamp(now),
		UpdatedAt:          reviewGatewayTimestamp(now),
	}
	if err := op.Validate(); err != nil {
		return ReviewOperation{}, err
	}
	return op, nil
}

func (o ReviewOperation) Validate() error {
	if strings.TrimSpace(o.OperationID) == "" || strings.TrimSpace(o.IdempotencyKey) == "" {
		return fmt.Errorf("review operation identity is required")
	}
	if !validReviewOperationKind(o.OperationKind) {
		return fmt.Errorf("unsupported review operation kind %q", o.OperationKind)
	}
	if !validReviewOperationStatus(o.Status) {
		return fmt.Errorf("unsupported review operation status %q", o.Status)
	}
	if !validReviewRetrySafety(o.RetrySafety) {
		return fmt.Errorf("unsupported review operation retry safety %q", o.RetrySafety)
	}
	if !validReviewDependencyPolicy(o.DependencyPolicy) {
		return fmt.Errorf("unsupported review operation dependency policy %q", o.DependencyPolicy)
	}
	if len(o.DesiredJSON) > reviewOperationDesiredJSONMaxBytes {
		return fmt.Errorf("desired_payload_too_large: review operation payload is %d bytes", len(o.DesiredJSON))
	}
	if strings.TrimSpace(o.DesiredJSON) == "" || !json.Valid([]byte(o.DesiredJSON)) {
		return fmt.Errorf("review operation desired_json must be valid JSON")
	}
	if reviewOperationPayloadContainsSecrets([]byte(o.DesiredJSON)) {
		return fmt.Errorf("review operation desired_json contains a forbidden secret field")
	}
	return nil
}

func boundedReviewOperationPayload(value interface{}) (string, string, error) {
	payload, err := json.Marshal(value)
	if err != nil {
		return "", "", fmt.Errorf("encode review operation desired payload: %w", err)
	}
	if len(payload) > reviewOperationDesiredJSONMaxBytes {
		return "", "", fmt.Errorf("desired_payload_too_large: review operation payload is %d bytes", len(payload))
	}
	if reviewOperationPayloadContainsSecrets(payload) {
		return "", "", fmt.Errorf("review operation desired payload contains a forbidden secret field")
	}
	digest := sha256.Sum256(payload)
	return string(payload), hex.EncodeToString(digest[:16]), nil
}

func reviewOperationPayloadContainsSecrets(payload []byte) bool {
	var value interface{}
	if json.Unmarshal(payload, &value) != nil {
		return true
	}
	var visit func(interface{}) bool
	visit = func(candidate interface{}) bool {
		switch current := candidate.(type) {
		case map[string]interface{}:
			keys := make([]string, 0, len(current))
			for key := range current {
				keys = append(keys, key)
			}
			sort.Strings(keys)
			for _, key := range keys {
				if reviewOperationSecretKeyPattern.MatchString(key) || visit(current[key]) {
					return true
				}
			}
		case []interface{}:
			for _, item := range current {
				if visit(item) {
					return true
				}
			}
		}
		return false
	}
	return visit(value)
}

func reviewOperationIdempotencyKey(kind ReviewOperationKind, workItemKey, desiredFingerprint string) string {
	return stableKey("review-operation-intent", string(kind), strings.TrimSpace(workItemKey), strings.TrimSpace(desiredFingerprint))
}

func reviewOperationQueueClass(kind ReviewOperationKind) string {
	switch kind {
	case ReviewOperationCanonicalCardUpsert:
		return "canonical_card"
	case ReviewOperationReplySend:
		return "reply"
	default:
		return "resource"
	}
}

func validReviewOperationKind(value ReviewOperationKind) bool {
	switch value {
	case ReviewOperationCanonicalCardUpsert, ReviewOperationReplySend, ReviewOperationBitableUpsert, ReviewOperationDocSnapshotUpsert, ReviewOperationTaskUpsert:
		return true
	default:
		return false
	}
}

func validReviewOperationStatus(value ReviewOperationStatus) bool {
	switch value {
	case ReviewOperationPending, ReviewOperationLeased, ReviewOperationWriting, ReviewOperationSucceeded,
		ReviewOperationUnchanged, ReviewOperationRetryScheduled, ReviewOperationFailedTerminal,
		ReviewOperationUnknown, ReviewOperationNeedsReconciliation, ReviewOperationStale,
		ReviewOperationDeadLetter, ReviewOperationCancelled, ReviewOperationBlocked:
		return true
	default:
		return false
	}
}

func validReviewRetrySafety(value ReviewOperationRetrySafety) bool {
	return value == ReviewRetryIdempotent || value == ReviewRetryReconcilable || value == ReviewRetryNonIdempotent
}

func validReviewDependencyPolicy(value ReviewOperationDependencyPolicy) bool {
	return value == ReviewDependencyNone || value == ReviewDependencySuccess || value == ReviewDependencyTerminal
}
