package feishu

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"
)

type ReviewResourceProjectionStatusValue string

const (
	ReviewProjectionNotConfigured       ReviewResourceProjectionStatusValue = "not_configured"
	ReviewProjectionDisabled            ReviewResourceProjectionStatusValue = "disabled"
	ReviewProjectionPending             ReviewResourceProjectionStatusValue = "pending"
	ReviewProjectionSyncing             ReviewResourceProjectionStatusValue = "syncing"
	ReviewProjectionSucceeded           ReviewResourceProjectionStatusValue = "succeeded"
	ReviewProjectionUnchanged           ReviewResourceProjectionStatusValue = "unchanged"
	ReviewProjectionFailed              ReviewResourceProjectionStatusValue = "failed"
	ReviewProjectionNeedsReconciliation ReviewResourceProjectionStatusValue = "needs_reconciliation"
)

type ReviewResourceProjectionStatus struct {
	ProjectionKey          string                              `json:"projection_key"`
	InstallationID         string                              `json:"installation_id"`
	ChatID                 string                              `json:"chat_id,omitempty"`
	Repository             string                              `json:"repository"`
	PRNumber               int                                 `json:"pr_number"`
	ResourceType           string                              `json:"resource_type"`
	TargetScope            string                              `json:"target_scope"`
	DesiredFingerprint     string                              `json:"desired_fingerprint,omitempty"`
	AppliedFingerprint     string                              `json:"applied_fingerprint,omitempty"`
	OperationID            string                              `json:"operation_id,omitempty"`
	Status                 ReviewResourceProjectionStatusValue `json:"status"`
	ErrorClass             string                              `json:"error_class,omitempty"`
	ErrorSummary           string                              `json:"error_summary,omitempty"`
	RequiresReconciliation bool                                `json:"requires_reconciliation"`
	UpdatedAt              string                              `json:"updated_at"`
}

func reviewProjectionKey(operation ReviewOperation, targetScope string) string {
	return stableKey("review-resource-projection", operation.InstallationID, operation.ChatID, operation.Repository, fmt.Sprintf("%d", operation.PRNumber), operation.ResourceType, targetScope)
}

func (s *SQLiteReviewGatewayStore) ListReviewResourceProjectionStatuses(ctx context.Context, installationID, chatID, repository string, prNumber int) ([]ReviewResourceProjectionStatus, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT projection_key, installation_id, chat_id, repository, pr_number,
		resource_type, target_scope, desired_fingerprint, applied_fingerprint,
		operation_id, status, error_class, error_summary, requires_reconciliation, updated_at
		FROM review_resource_projection_status
		WHERE installation_id=? AND chat_id=? AND repository=? AND pr_number=?
		ORDER BY resource_type`, installationID, chatID, repository, prNumber)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []ReviewResourceProjectionStatus{}
	for rows.Next() {
		item, scanErr := scanReviewResourceProjectionStatus(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		result = append(result, item)
	}
	return result, rows.Err()
}

func (s *SQLiteReviewGatewayStore) GetReviewResourceProjectionStatus(ctx context.Context, projectionKey string) (ReviewResourceProjectionStatus, error) {
	return scanReviewResourceProjectionStatus(s.db.QueryRowContext(ctx, `SELECT projection_key, installation_id, chat_id, repository, pr_number,
		resource_type, target_scope, desired_fingerprint, applied_fingerprint,
		operation_id, status, error_class, error_summary, requires_reconciliation, updated_at
		FROM review_resource_projection_status WHERE projection_key=?`, projectionKey))
}

func (s *SQLiteReviewGatewayStore) EnsureReviewProjectionConfiguration(ctx context.Context, job ReviewGatewayJob, resourceType string, targetScope ReviewResourceScope, status ReviewResourceProjectionStatusValue, now time.Time) error {
	return ensureReviewProjectionConfigurationExec(ctx, s.db, job, resourceType, targetScope, status, now)
}

type reviewProjectionExecer interface {
	ExecContext(context.Context, string, ...interface{}) (sql.Result, error)
}

func ensureReviewProjectionConfigurationExec(ctx context.Context, execer reviewProjectionExecer, job ReviewGatewayJob, resourceType string, targetScope ReviewResourceScope, status ReviewResourceProjectionStatusValue, now time.Time) error {
	operation := ReviewOperation{
		InstallationID: job.InstallationID, ChatID: job.ChatID, Repository: job.Repository,
		PRNumber: job.PRNumber, ResourceType: resourceType,
		DesiredJSON: fmt.Sprintf(`{"target_scope":%q}`, targetScope),
	}
	key := reviewProjectionKey(operation, string(targetScope))
	_, err := execer.ExecContext(ctx, `INSERT INTO review_resource_projection_status (
		projection_key, installation_id, chat_id, repository, pr_number,
		resource_type, target_scope, status, updated_at
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	ON CONFLICT(projection_key) DO UPDATE SET
		status=CASE
			WHEN excluded.status IN ('disabled','not_configured') THEN excluded.status
			ELSE review_resource_projection_status.status
		END,
		updated_at=excluded.updated_at`, key, job.InstallationID, job.ChatID, job.Repository,
		job.PRNumber, resourceType, targetScope, status, reviewGatewayTimestamp(now))
	return err
}

func updateReviewProjectionStatusTx(ctx context.Context, tx *sql.Tx, operation ReviewOperation, status ReviewResourceProjectionStatusValue, appliedFingerprint, errorClass, errorSummary string, reconciliation bool, now time.Time) error {
	if operation.ResourceType != ReviewResourceBitable && operation.ResourceType != ReviewResourceDoc && operation.ResourceType != ReviewResourceTask {
		return nil
	}
	targetScope := reviewOperationTargetScope(operation)
	_, err := tx.ExecContext(ctx, `INSERT INTO review_resource_projection_status (
		projection_key, installation_id, chat_id, repository, pr_number,
		resource_type, target_scope, desired_fingerprint, applied_fingerprint,
		operation_id, status, error_class, error_summary, requires_reconciliation, updated_at
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	ON CONFLICT(projection_key) DO UPDATE SET
		desired_fingerprint=excluded.desired_fingerprint,
		applied_fingerprint=CASE WHEN excluded.applied_fingerprint<>'' THEN excluded.applied_fingerprint ELSE review_resource_projection_status.applied_fingerprint END,
		operation_id=excluded.operation_id,
		status=excluded.status,
		error_class=excluded.error_class,
		error_summary=excluded.error_summary,
		requires_reconciliation=excluded.requires_reconciliation,
		updated_at=excluded.updated_at`,
		reviewProjectionKey(operation, targetScope), operation.InstallationID, operation.ChatID,
		operation.Repository, operation.PRNumber, operation.ResourceType, targetScope,
		operation.DesiredFingerprint, appliedFingerprint, operation.OperationID, status,
		errorClass, redactReviewGatewayError(errorSummary), boolToReviewCollaborationInt(reconciliation), reviewGatewayTimestamp(now))
	return err
}

func reviewOperationTargetScope(operation ReviewOperation) string {
	var target struct {
		TargetScope string `json:"target_scope"`
	}
	_ = jsonUnmarshalBounded(operation.DesiredJSON, &target)
	if target.TargetScope == "" {
		return string(ReviewResourceScopeChat)
	}
	return target.TargetScope
}

func jsonUnmarshalBounded(value string, target interface{}) error {
	if len(value) > reviewOperationDesiredJSONMaxBytes {
		return fmt.Errorf("desired_payload_too_large")
	}
	return json.Unmarshal([]byte(value), target)
}

type reviewProjectionScanner interface{ Scan(...interface{}) error }

func scanReviewResourceProjectionStatus(scanner reviewProjectionScanner) (ReviewResourceProjectionStatus, error) {
	var value ReviewResourceProjectionStatus
	var reconcile int
	err := scanner.Scan(&value.ProjectionKey, &value.InstallationID, &value.ChatID, &value.Repository,
		&value.PRNumber, &value.ResourceType, &value.TargetScope, &value.DesiredFingerprint,
		&value.AppliedFingerprint, &value.OperationID, &value.Status, &value.ErrorClass,
		&value.ErrorSummary, &reconcile, &value.UpdatedAt)
	value.RequiresReconciliation = reconcile != 0
	if errors.Is(err, sql.ErrNoRows) {
		return ReviewResourceProjectionStatus{}, sql.ErrNoRows
	}
	return value, err
}

func reviewCardWithProjectionStatuses(card Card, statuses []ReviewResourceProjectionStatus) Card {
	if len(card) == 0 || len(statuses) == 0 {
		return card
	}
	copy := cloneReviewCard(card)
	labels := map[string]string{ReviewResourceBitable: "Base", ReviewResourceDoc: "Doc", ReviewResourceTask: "Task"}
	states := map[ReviewResourceProjectionStatusValue]string{
		ReviewProjectionNotConfigured:       "未配置",
		ReviewProjectionDisabled:            "已禁用",
		ReviewProjectionPending:             "同步中",
		ReviewProjectionSyncing:             "同步中",
		ReviewProjectionSucceeded:           "已同步",
		ReviewProjectionUnchanged:           "已同步",
		ReviewProjectionFailed:              "失败",
		ReviewProjectionNeedsReconciliation: "待对账",
	}
	lines := []string{}
	for _, item := range statuses {
		label, ok := labels[item.ResourceType]
		if !ok {
			continue
		}
		lines = append(lines, label+"："+firstNonEmpty(states[item.Status], string(item.Status)))
	}
	sort.Strings(lines)
	if len(lines) == 0 {
		return copy
	}
	elements, _ := copy["elements"].([]interface{})
	elements = append(elements, map[string]interface{}{
		"tag":  "div",
		"text": map[string]interface{}{"tag": "lark_md", "content": strings.Join(lines, "　")},
	})
	copy["elements"] = elements
	return copy
}

func cloneReviewCard(card Card) Card {
	encoded, _ := json.Marshal(card)
	var cloned Card
	_ = json.Unmarshal(encoded, &cloned)
	return cloned
}

// QueueReviewProjectionCardRefresh creates a new deterministic card intent
// when an independently executed Base, Doc or Task projection changes. Card
// operation completion never calls this function, preventing refresh loops.
func (s *SQLiteReviewGatewayStore) QueueReviewProjectionCardRefresh(ctx context.Context, source ReviewOperation, now time.Time) error {
	if source.ResourceType != ReviewResourceBitable && source.ResourceType != ReviewResourceDoc && source.ResourceType != ReviewResourceTask {
		return nil
	}
	latest, err := scanReviewOperation(s.db.QueryRowContext(ctx, reviewOperationSelect+`
		WHERE installation_id=? AND chat_id=? AND repository=? AND pr_number=?
		  AND operation_kind='canonical_card_upsert'
		ORDER BY created_at DESC, operation_id DESC LIMIT 1`,
		source.InstallationID, source.ChatID, source.Repository, source.PRNumber))
	if errors.Is(err, ErrReviewOperationNotFound) {
		return nil
	}
	if err != nil {
		return err
	}
	var desired reviewCanonicalCardDesired
	if err := jsonUnmarshalBounded(latest.DesiredJSON, &desired); err != nil {
		return err
	}
	statuses, err := s.ListReviewResourceProjectionStatuses(ctx, source.InstallationID, source.ChatID, source.Repository, source.PRNumber)
	if err != nil {
		return err
	}
	_, projectionFingerprint, err := boundedReviewOperationPayload(statuses)
	if err != nil {
		return err
	}
	if desired.ProjectionFingerprint == projectionFingerprint {
		return nil
	}
	desired.ProjectionFingerprint = projectionFingerprint
	job := ReviewGatewayJob{
		JobID: source.SourceJobID, InstallationID: source.InstallationID, ChatID: source.ChatID,
		Repository: source.Repository, PRNumber: source.PRNumber, SourceEventID: source.SourceEventID,
	}
	operation, err := NewReviewOperation(ReviewOperationCanonicalCardUpsert, job, reviewChatPRPresentationKey(job), ReviewResourceCard, desired, ReviewRetryNonIdempotent, now)
	if err != nil {
		return err
	}
	return s.SaveReviewOperations(ctx, []ReviewOperation{operation})
}
