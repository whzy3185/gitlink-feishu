package feishu

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

const reviewEventSanitizedPayloadLimit = 256 * 1024

type ReviewEventIngressMetadata struct {
	SignatureStatus  string
	TimestampStatus  string
	DeliveryStatus   string
	RawPayload       []byte
	ReplayRequestKey string
}

type ReviewEventInboxRecord struct {
	Event                NormalizedReviewEvent `json:"event"`
	Status               string                `json:"status"`
	ReasonCode           string                `json:"reason_code,omitempty"`
	SignatureStatus      string                `json:"signature_status"`
	TimestampStatus      string                `json:"timestamp_status"`
	DeliveryStatus       string                `json:"delivery_status"`
	CanonicalEventJSON   string                `json:"-"`
	SanitizedPayloadJSON string                `json:"-"`
	RouteRevision        int                   `json:"route_revision"`
	AttemptCount         int                   `json:"attempt_count"`
	NextAttemptAt        string                `json:"next_attempt_at,omitempty"`
	LeaseOwner           string                `json:"-"`
	LeaseExpiresAt       string                `json:"lease_expires_at,omitempty"`
	ErrorSummary         string                `json:"error_summary,omitempty"`
	ProcessedAt          string                `json:"processed_at,omitempty"`
	ReplayOfEventID      string                `json:"replay_of_event_id,omitempty"`
	ReplayRequestKey     string                `json:"replay_request_key,omitempty"`
}

type ReviewEventInboxInsertResult struct {
	Record    ReviewEventInboxRecord
	Duplicate bool
}

type ReviewEventClaimOptions struct {
	LeaseOwner    string
	Now           time.Time
	LeaseDuration time.Duration
}

type ReviewEventListFilters struct {
	Status         string
	InstallationID string
	Repository     string
	EventType      string
	Limit          int
}

func (s *SQLiteReviewGatewayStore) ListReviewEventInbox(ctx context.Context, filters ReviewEventListFilters) ([]ReviewEventInboxRecord, error) {
	query := reviewEventInboxSelect + ` WHERE 1=1`
	args := []interface{}{}
	for _, filter := range []struct{ column, value string }{
		{"status", filters.Status}, {"installation_id", filters.InstallationID},
		{"repository", filters.Repository}, {"event_type", filters.EventType},
	} {
		if strings.TrimSpace(filter.value) != "" {
			query += " AND " + filter.column + "=?"
			args = append(args, strings.TrimSpace(filter.value))
		}
	}
	limit := filters.Limit
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	query += ` ORDER BY received_at DESC, event_id DESC LIMIT ?`
	args = append(args, limit)
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []ReviewEventInboxRecord{}
	for rows.Next() {
		record, err := scanReviewEventInbox(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, record)
	}
	return result, rows.Err()
}

func (s *SQLiteReviewGatewayStore) InsertReviewEventInbox(
	ctx context.Context,
	normalization ReviewEventNormalization,
	metadata ReviewEventIngressMetadata,
) (ReviewEventInboxInsertResult, error) {
	event := normalization.Event
	if event.EventID == "" {
		event.EventID = reviewEventSyntheticDeliveryKey("ignored:"+normalization.ReasonCode, event.InstallationID, event.DeliveryKey)
	}
	status := strings.TrimSpace(normalization.Status)
	if status != "normalized" && status != "ignored" {
		return ReviewEventInboxInsertResult{}, fmt.Errorf("unsupported initial Review event status %q", status)
	}
	canonicalJSON := ""
	if status == "normalized" {
		if err := validateNormalizedReviewEvent(event); err != nil {
			return ReviewEventInboxInsertResult{}, err
		}
		encoded, err := json.Marshal(event)
		if err != nil {
			return ReviewEventInboxInsertResult{}, fmt.Errorf("encode canonical Review event: %w", err)
		}
		canonicalJSON = string(encoded)
	}
	sanitizedJSON := sanitizeReviewEventPayload(event, metadata.RawPayload)
	record := ReviewEventInboxRecord{
		Event: event, Status: status, ReasonCode: normalization.ReasonCode,
		SignatureStatus:    firstNonEmpty(metadata.SignatureStatus, "not_checked"),
		TimestampStatus:    firstNonEmpty(metadata.TimestampStatus, "not_checked"),
		DeliveryStatus:     firstNonEmpty(metadata.DeliveryStatus, "header"),
		CanonicalEventJSON: canonicalJSON, SanitizedPayloadJSON: sanitizedJSON,
		ReplayOfEventID: event.ReplayOf, ReplayRequestKey: strings.TrimSpace(metadata.ReplayRequestKey),
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return ReviewEventInboxInsertResult{}, err
	}
	defer tx.Rollback()
	result, err := tx.ExecContext(ctx, `INSERT OR IGNORE INTO review_event_inbox (
		event_id, installation_id, delivery_key, delivery_hash, source, schema_version,
		event_type, action, repository, pr_number, head_sha, actor_hash, occurred_at,
		received_at, signature_status, timestamp_status, delivery_status,
		payload_fingerprint, canonical_event_json, sanitized_payload_json,
		status, reason_code, replay_of_event_id, replay_request_key
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		event.EventID, event.InstallationID, event.DeliveryKey, event.DeliveryHash,
		event.Source, event.SchemaVersion, event.EventType, event.Action, event.Repository,
		event.PRNumber, event.HeadSHA, event.ActorHash, event.OccurredAt, event.ReceivedAt,
		record.SignatureStatus, record.TimestampStatus, record.DeliveryStatus,
		event.PayloadFingerprint, canonicalJSON, sanitizedJSON, status, normalization.ReasonCode,
		event.ReplayOf, record.ReplayRequestKey)
	if err != nil {
		return ReviewEventInboxInsertResult{}, fmt.Errorf("insert Review event inbox: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return ReviewEventInboxInsertResult{}, err
	}
	if affected == 0 {
		var existing ReviewEventInboxRecord
		if record.ReplayRequestKey != "" {
			existing, err = scanReviewEventInbox(tx.QueryRowContext(ctx, reviewEventInboxSelect+` WHERE replay_request_key=?`, record.ReplayRequestKey))
		} else {
			existing, err = getReviewEventInboxTx(ctx, tx, event.InstallationID, event.DeliveryKey)
		}
		if err != nil {
			return ReviewEventInboxInsertResult{}, err
		}
		if err := tx.Commit(); err != nil {
			return ReviewEventInboxInsertResult{}, err
		}
		return ReviewEventInboxInsertResult{Record: existing, Duplicate: true}, nil
	}
	if err := tx.Commit(); err != nil {
		return ReviewEventInboxInsertResult{}, err
	}
	record.Event.EventID = event.EventID
	return ReviewEventInboxInsertResult{Record: record}, nil
}

func sanitizeReviewEventPayload(event NormalizedReviewEvent, _ []byte) string {
	// Keep only bounded routing diagnostics. Raw actors, headers, credentials,
	// cookies, tokens, and arbitrary payload fields never enter long-term state.
	summary := map[string]interface{}{
		"repository": event.Repository,
		"pr_number":  event.PRNumber,
		"event_type": event.EventType,
		"action":     event.Action,
		"head_sha":   event.HeadSHA,
	}
	encoded, _ := json.Marshal(summary)
	if len(encoded) > reviewEventSanitizedPayloadLimit {
		return `{"truncated":true}`
	}
	return string(encoded)
}

func (s *SQLiteReviewGatewayStore) GetReviewEventInbox(ctx context.Context, eventID string) (ReviewEventInboxRecord, error) {
	return scanReviewEventInbox(s.db.QueryRowContext(ctx, reviewEventInboxSelect+` WHERE event_id=?`, eventID))
}

func getReviewEventInboxTx(ctx context.Context, tx *sql.Tx, installationID, deliveryKey string) (ReviewEventInboxRecord, error) {
	return scanReviewEventInbox(tx.QueryRowContext(ctx, reviewEventInboxSelect+
		` WHERE installation_id=? AND delivery_key=?`, installationID, deliveryKey))
}

const reviewEventInboxSelect = `SELECT event_id, installation_id, delivery_key, delivery_hash,
	source, schema_version, event_type, action, repository, pr_number, head_sha, actor_hash,
	occurred_at, received_at, payload_fingerprint, canonical_event_json, sanitized_payload_json,
	status, reason_code, signature_status, timestamp_status, delivery_status, route_revision,
	attempt_count, next_attempt_at, lease_owner, lease_expires_at, error_summary, processed_at,
	replay_of_event_id, replay_request_key FROM review_event_inbox`

type reviewEventRow interface {
	Scan(dest ...interface{}) error
}

func scanReviewEventInbox(row reviewEventRow) (ReviewEventInboxRecord, error) {
	var record ReviewEventInboxRecord
	err := row.Scan(&record.Event.EventID, &record.Event.InstallationID,
		&record.Event.DeliveryKey, &record.Event.DeliveryHash, &record.Event.Source,
		&record.Event.SchemaVersion, &record.Event.EventType, &record.Event.Action,
		&record.Event.Repository, &record.Event.PRNumber, &record.Event.HeadSHA,
		&record.Event.ActorHash, &record.Event.OccurredAt, &record.Event.ReceivedAt,
		&record.Event.PayloadFingerprint, &record.CanonicalEventJSON,
		&record.SanitizedPayloadJSON, &record.Status, &record.ReasonCode,
		&record.SignatureStatus, &record.TimestampStatus, &record.DeliveryStatus,
		&record.RouteRevision, &record.AttemptCount, &record.NextAttemptAt,
		&record.LeaseOwner, &record.LeaseExpiresAt, &record.ErrorSummary,
		&record.ProcessedAt, &record.ReplayOfEventID, &record.ReplayRequestKey)
	if err != nil {
		return ReviewEventInboxRecord{}, err
	}
	record.Event.ReplayOf = record.ReplayOfEventID
	record.Event.Synthetic = record.Event.Source != ReviewEventSourceGitLinkWebhook
	return record, nil
}

func (s *SQLiteReviewGatewayStore) ClaimReviewEventInbox(
	ctx context.Context, options ReviewEventClaimOptions,
) (ReviewEventInboxRecord, error) {
	if options.Now.IsZero() {
		options.Now = time.Now().UTC()
	}
	if options.LeaseDuration <= 0 {
		options.LeaseDuration = 2 * time.Minute
	}
	if strings.TrimSpace(options.LeaseOwner) == "" {
		return ReviewEventInboxRecord{}, fmt.Errorf("Review event lease owner is required")
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return ReviewEventInboxRecord{}, err
	}
	defer tx.Rollback()
	nowText := reviewGatewayTimestamp(options.Now)
	var eventID string
	err = tx.QueryRowContext(ctx, `SELECT event_id FROM review_event_inbox
		WHERE status IN ('normalized','retry_scheduled')
		AND (next_attempt_at='' OR next_attempt_at<=?)
		AND (lease_expires_at='' OR lease_expires_at<=?)
		ORDER BY received_at, event_id LIMIT 1`, nowText, nowText).Scan(&eventID)
	if errors.Is(err, sql.ErrNoRows) {
		return ReviewEventInboxRecord{}, sql.ErrNoRows
	}
	if err != nil {
		return ReviewEventInboxRecord{}, err
	}
	leaseExpires := reviewGatewayTimestamp(options.Now.Add(options.LeaseDuration))
	result, err := tx.ExecContext(ctx, `UPDATE review_event_inbox SET
		status='routing', lease_owner=?, lease_expires_at=?, attempt_count=attempt_count+1
		WHERE event_id=? AND status IN ('normalized','retry_scheduled')
		AND (lease_expires_at='' OR lease_expires_at<=?)`, options.LeaseOwner, leaseExpires, eventID, nowText)
	if err != nil {
		return ReviewEventInboxRecord{}, err
	}
	affected, _ := result.RowsAffected()
	if affected != 1 {
		return ReviewEventInboxRecord{}, sql.ErrNoRows
	}
	record, err := scanReviewEventInbox(tx.QueryRowContext(ctx, reviewEventInboxSelect+` WHERE event_id=?`, eventID))
	if err != nil {
		return ReviewEventInboxRecord{}, err
	}
	if err := tx.Commit(); err != nil {
		return ReviewEventInboxRecord{}, err
	}
	return record, nil
}

func (s *SQLiteReviewGatewayStore) finishReviewEventInbox(
	ctx context.Context, eventID, leaseOwner, status, reason, errorSummary string, now time.Time,
) error {
	result, err := s.db.ExecContext(ctx, `UPDATE review_event_inbox SET status=?, reason_code=?,
		error_summary=?, processed_at=?, lease_owner='', lease_expires_at='', next_attempt_at=''
		WHERE event_id=? AND status IN ('routing','routed') AND lease_owner=?`,
		status, reason, redactReviewGatewayError(errorSummary), reviewGatewayTimestamp(now), eventID, leaseOwner)
	if err != nil {
		return err
	}
	affected, _ := result.RowsAffected()
	if affected != 1 {
		return fmt.Errorf("Review event lease was lost")
	}
	return nil
}

func (s *SQLiteReviewGatewayStore) retryReviewEventInbox(
	ctx context.Context, record ReviewEventInboxRecord, leaseOwner string, processingErr error, now time.Time,
) error {
	status := "retry_scheduled"
	next := reviewGatewayTimestamp(now.Add(time.Duration(record.AttemptCount) * time.Minute))
	if record.AttemptCount >= 3 {
		status, next = "failed", ""
	}
	_, err := s.db.ExecContext(ctx, `UPDATE review_event_inbox SET status=?, next_attempt_at=?,
		error_summary=?, lease_owner='', lease_expires_at='' WHERE event_id=? AND lease_owner=?`,
		status, next, redactReviewGatewayError(processingErr.Error()), record.Event.EventID, leaseOwner)
	return err
}
