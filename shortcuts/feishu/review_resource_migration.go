package feishu

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"
)

const (
	ReviewResourceMigrationDiscovered          = "discovered"
	ReviewResourceMigrationNeedsReconciliation = "needs_reconciliation"
	ReviewResourceMigrationVerified            = "verified"
	ReviewResourceMigrationApplying            = "applying"
	ReviewResourceMigrationMigrated            = "migrated"
	ReviewResourceMigrationSkippedAmbiguous    = "skipped_ambiguous"
	ReviewResourceMigrationSkippedDisabled     = "skipped_disabled"
	ReviewResourceMigrationFailed              = "failed"
	ReviewResourceMigrationSuperseded          = "superseded"
)

var ErrReviewResourceMigrationConflict = errors.New("review resource migration state changed concurrently")

type ReviewResourceMigration struct {
	MigrationID        string              `json:"migration_id"`
	LegacyWorkItemKey  string              `json:"legacy_work_item_key"`
	TargetWorkItemKey  string              `json:"target_work_item_key,omitempty"`
	ResourceType       string              `json:"resource_type"`
	TargetScope        ReviewResourceScope `json:"target_scope,omitempty"`
	InstallationID     string              `json:"installation_id,omitempty"`
	Repository         string              `json:"repository,omitempty"`
	PRNumber           int                 `json:"pr_number,omitempty"`
	ChatIDHash         string              `json:"chat_id_hash,omitempty"`
	RemoteIDHash       string              `json:"remote_id_hash,omitempty"`
	Status             string              `json:"status"`
	ReasonCode         string              `json:"reason_code,omitempty"`
	VerificationMethod string              `json:"verification_method,omitempty"`
	VerifiedByHash     string              `json:"verified_by_hash,omitempty"`
	VerifiedAt         string              `json:"verified_at,omitempty"`
	AttemptCount       int                 `json:"attempt_count"`
	ErrorSummary       string              `json:"error_summary,omitempty"`
	CreatedAt          string              `json:"created_at"`
	UpdatedAt          string              `json:"updated_at"`
}

type ReviewResourceMigrationAudit struct {
	AuditID     string          `json:"audit_id"`
	MigrationID string          `json:"migration_id"`
	Action      string          `json:"action"`
	FromStatus  string          `json:"from_status"`
	ToStatus    string          `json:"to_status"`
	ActorHash   string          `json:"actor_hash,omitempty"`
	DetailJSON  json.RawMessage `json:"detail_json"`
	CreatedAt   string          `json:"created_at"`
}

type LegacyReviewResourceVerificationInput struct {
	MigrationID          string              `json:"migration_id"`
	ExpectedInstallation string              `json:"expected_installation"`
	ExpectedScope        ReviewResourceScope `json:"expected_scope"`
	ExpectedChatID       string              `json:"-"`
	Method               string              `json:"method"`
	Actor                string              `json:"-"`
	Confirmed            bool                `json:"confirmed"`
}

type LegacyReviewResourceVerifier interface {
	Verify(context.Context, ReviewResourceMigration, LegacyReviewResourceVerificationInput) error
}

type OperatorConfirmedLegacyReviewResourceVerifier struct{}

func (OperatorConfirmedLegacyReviewResourceVerifier) Verify(
	_ context.Context,
	migration ReviewResourceMigration,
	input LegacyReviewResourceVerificationInput,
) error {
	if !input.Confirmed {
		return fmt.Errorf("explicit --yes confirmation is required")
	}
	if input.Method != "operator_confirmed" {
		return fmt.Errorf("unsupported verification method %q", input.Method)
	}
	if strings.TrimSpace(input.MigrationID) != migration.MigrationID {
		return fmt.Errorf("migration ID confirmation does not match")
	}
	if strings.TrimSpace(input.ExpectedInstallation) != migration.InstallationID {
		return fmt.Errorf("installation confirmation does not match migration plan")
	}
	if input.ExpectedScope != migration.TargetScope {
		return fmt.Errorf("scope confirmation does not match migration plan")
	}
	if migration.TargetScope == ReviewResourceScopeChat && reviewResourceIdentifierHash(input.ExpectedChatID) != migration.ChatIDHash {
		return fmt.Errorf("chat confirmation does not match migration plan")
	}
	return nil
}

type legacyReviewResourceCandidate struct {
	LegacyKey    string
	ResourceType string
	RemoteID     string
	Repository   string
	PRNumber     int
	ChatID       string
}

func DiscoverLegacyReviewResourceMigrations(ctx context.Context, tx *sql.Tx, now time.Time) error {
	if tx == nil {
		return fmt.Errorf("review resource migration discovery transaction is required")
	}
	items, err := readLegacyReviewResourceCandidates(ctx, tx)
	if err != nil {
		return err
	}
	for _, candidate := range items {
		migration, err := classifyLegacyReviewResource(ctx, tx, candidate, now)
		if err != nil {
			return err
		}
		inserted, err := insertReviewResourceMigration(ctx, tx, migration)
		if err != nil {
			return err
		}
		if !inserted {
			continue
		}
		if err := appendReviewResourceMigrationAudit(ctx, tx, migration, "discovered", "", ReviewResourceMigrationDiscovered, "", map[string]interface{}{
			"resource_type":  migration.ResourceType,
			"remote_id_hash": migration.RemoteIDHash,
		}, now); err != nil {
			return err
		}
		if migration.Status != ReviewResourceMigrationDiscovered {
			if err := appendReviewResourceMigrationAudit(ctx, tx, migration, "classified", ReviewResourceMigrationDiscovered, migration.Status, "", map[string]interface{}{
				"reason_code":  migration.ReasonCode,
				"target_scope": migration.TargetScope,
			}, now); err != nil {
				return err
			}
		}
	}
	return nil
}

func readLegacyReviewResourceCandidates(ctx context.Context, tx *sql.Tx) ([]legacyReviewResourceCandidate, error) {
	rows, err := tx.QueryContext(ctx, `SELECT repository, pr_number, chat_id FROM review_collaboration_items`)
	if err != nil {
		return nil, fmt.Errorf("read legacy Review items for resource discovery: %w", err)
	}
	legacyItems := map[string]legacyReviewResourceCandidate{}
	for rows.Next() {
		var candidate legacyReviewResourceCandidate
		if err := rows.Scan(&candidate.Repository, &candidate.PRNumber, &candidate.ChatID); err != nil {
			_ = rows.Close()
			return nil, err
		}
		candidate.LegacyKey = stableKey("review-work-item", candidate.Repository, fmt.Sprintf("%d", candidate.PRNumber))
		legacyItems[candidate.LegacyKey] = candidate
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	resourceRows, err := tx.QueryContext(ctx, `SELECT work_item_key, resource_type, remote_id
		FROM review_collaboration_resources ORDER BY work_item_key, resource_type`)
	if err != nil {
		return nil, fmt.Errorf("read legacy Review resources: %w", err)
	}
	candidates := []legacyReviewResourceCandidate{}
	for resourceRows.Next() {
		var legacyKey, resourceType, remoteID string
		if err := resourceRows.Scan(&legacyKey, &resourceType, &remoteID); err != nil {
			_ = resourceRows.Close()
			return nil, err
		}
		item, ok := legacyItems[legacyKey]
		if !ok || strings.TrimSpace(remoteID) == "" {
			continue
		}
		item.LegacyKey = legacyKey
		item.ResourceType = resourceType
		item.RemoteID = remoteID
		candidates = append(candidates, item)
	}
	if err := resourceRows.Close(); err != nil {
		return nil, err
	}
	return candidates, nil
}

func classifyLegacyReviewResource(
	ctx context.Context,
	tx *sql.Tx,
	candidate legacyReviewResourceCandidate,
	now time.Time,
) (ReviewResourceMigration, error) {
	installations, err := candidateLegacyReviewInstallations(ctx, tx, candidate.ChatID, candidate.Repository)
	if err != nil {
		return ReviewResourceMigration{}, err
	}
	migration := ReviewResourceMigration{
		LegacyWorkItemKey: candidate.LegacyKey,
		ResourceType:      candidate.ResourceType,
		Repository:        candidate.Repository,
		PRNumber:          candidate.PRNumber,
		ChatIDHash:        reviewResourceIdentifierHash(candidate.ChatID),
		RemoteIDHash:      reviewResourceIdentifierHash(candidate.RemoteID),
		Status:            ReviewResourceMigrationDiscovered,
		CreatedAt:         now.UTC().Format(time.RFC3339Nano),
		UpdatedAt:         now.UTC().Format(time.RFC3339Nano),
	}
	candidateIdentity := strings.Join(installations, ",")
	if len(installations) != 1 {
		migration.Status = ReviewResourceMigrationSkippedAmbiguous
		migration.ReasonCode = "ambiguous_installation"
		migration.MigrationID = reviewResourceMigrationID(migration, candidateIdentity)
		return migration, nil
	}
	migration.InstallationID = installations[0]
	switch candidate.ResourceType {
	case ReviewResourceCard, ReviewResourceTask:
		migration.TargetScope = ReviewResourceScopeChat
	case ReviewResourceBitable, ReviewResourceDoc:
		var enabled int
		err := tx.QueryRowContext(ctx, `SELECT target_scope, migration_enabled
			FROM review_resource_scope_policies WHERE installation_id=? AND resource_type=?`,
			migration.InstallationID, candidate.ResourceType,
		).Scan(&migration.TargetScope, &enabled)
		if errors.Is(err, sql.ErrNoRows) {
			migration.Status = ReviewResourceMigrationNeedsReconciliation
			migration.ReasonCode = "explicit_scope_policy_required"
			migration.MigrationID = reviewResourceMigrationID(migration, candidateIdentity)
			return migration, nil
		}
		if err != nil {
			return ReviewResourceMigration{}, err
		}
		if migration.TargetScope == ReviewResourceScopeDisabled {
			migration.Status = ReviewResourceMigrationSkippedDisabled
			migration.ReasonCode = "resource_projection_disabled"
			migration.MigrationID = reviewResourceMigrationID(migration, candidateIdentity)
			return migration, nil
		}
		if enabled == 0 {
			migration.Status = ReviewResourceMigrationNeedsReconciliation
			migration.ReasonCode = "migration_not_enabled"
			migration.MigrationID = reviewResourceMigrationID(migration, candidateIdentity)
			return migration, nil
		}
	default:
		migration.Status = ReviewResourceMigrationNeedsReconciliation
		migration.ReasonCode = "unsupported_resource_type"
		migration.MigrationID = reviewResourceMigrationID(migration, candidateIdentity)
		return migration, nil
	}
	if migration.TargetScope == ReviewResourceScopeChat && strings.TrimSpace(candidate.ChatID) == "" {
		migration.Status = ReviewResourceMigrationNeedsReconciliation
		migration.ReasonCode = "missing_legacy_chat"
		migration.MigrationID = reviewResourceMigrationID(migration, candidateIdentity)
		return migration, nil
	}
	target, err := ResolveReviewResourceTarget(
		candidate.ResourceType, migration.InstallationID, candidate.ChatID,
		candidate.Repository, candidate.PRNumber, migration.TargetScope,
	)
	if err != nil {
		migration.Status = ReviewResourceMigrationNeedsReconciliation
		migration.ReasonCode = "target_resolution_failed"
	} else {
		migration.TargetWorkItemKey = reviewResourceMigrationTargetReference(target, migration.TargetScope)
		migration.Status = ReviewResourceMigrationNeedsReconciliation
		migration.ReasonCode = "explicit_verification_required"
	}
	migration.MigrationID = reviewResourceMigrationID(migration, candidateIdentity)
	return migration, nil
}

func reviewResourceMigrationTargetReference(target string, scope ReviewResourceScope) string {
	if scope == ReviewResourceScopeChat {
		return stableKey("review-resource-target", reviewResourceIdentifierHash(target))
	}
	return target
}

func candidateLegacyReviewInstallations(ctx context.Context, tx *sql.Tx, chatID, repository string) ([]string, error) {
	rows, err := tx.QueryContext(ctx, `SELECT DISTINCT installation_id FROM chat_repository_bindings
		WHERE chat_id=? AND repository=? AND enabled=1 ORDER BY installation_id`, chatID, repository)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	installations := []string{}
	for rows.Next() {
		var installation string
		if err := rows.Scan(&installation); err != nil {
			return nil, err
		}
		installations = append(installations, installation)
	}
	return installations, rows.Err()
}

func reviewResourceMigrationID(migration ReviewResourceMigration, candidateIdentity string) string {
	return stableKey(
		"review-resource-migration", migration.LegacyWorkItemKey, migration.ResourceType,
		candidateIdentity, migration.ChatIDHash, string(migration.TargetScope),
	)
}

func reviewResourceIdentifierHash(value string) string {
	if strings.TrimSpace(value) == "" {
		return ""
	}
	sum := sha256.Sum256([]byte(strings.TrimSpace(value)))
	return hex.EncodeToString(sum[:])
}

func insertReviewResourceMigration(ctx context.Context, tx *sql.Tx, migration ReviewResourceMigration) (bool, error) {
	result, err := tx.ExecContext(ctx, `INSERT INTO review_resource_migrations (
		migration_id, legacy_work_item_key, target_work_item_key, resource_type,
		target_scope, installation_id, repository, pr_number, chat_id_hash, remote_id_hash,
		status, reason_code, created_at, updated_at
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	ON CONFLICT(migration_id) DO NOTHING`,
		migration.MigrationID, migration.LegacyWorkItemKey, migration.TargetWorkItemKey,
		migration.ResourceType, migration.TargetScope, migration.InstallationID,
		migration.Repository, migration.PRNumber, migration.ChatIDHash, migration.RemoteIDHash,
		migration.Status, migration.ReasonCode, migration.CreatedAt, migration.UpdatedAt,
	)
	if err != nil {
		return false, err
	}
	affected, err := result.RowsAffected()
	return affected == 1, err
}

func (s *SQLiteReviewGatewayStore) ScanLegacyReviewResourceMigrations(ctx context.Context, now time.Time) error {
	if s == nil || s.db == nil {
		return fmt.Errorf("review resource migration store is unavailable")
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err := DiscoverLegacyReviewResourceMigrations(ctx, tx, now); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *SQLiteReviewGatewayStore) GetReviewResourceMigration(ctx context.Context, migrationID string) (ReviewResourceMigration, error) {
	return readReviewResourceMigration(ctx, s.db, migrationID)
}

type reviewResourceMigrationQueryer interface {
	QueryRowContext(context.Context, string, ...interface{}) *sql.Row
}

func readReviewResourceMigration(ctx context.Context, db reviewResourceMigrationQueryer, migrationID string) (ReviewResourceMigration, error) {
	var migration ReviewResourceMigration
	err := db.QueryRowContext(ctx, `SELECT migration_id, legacy_work_item_key, target_work_item_key,
		resource_type, target_scope, installation_id, repository, pr_number, chat_id_hash,
		remote_id_hash, status, reason_code, verification_method, verified_by_hash,
		verified_at, attempt_count, error_summary, created_at, updated_at
		FROM review_resource_migrations WHERE migration_id=?`, strings.TrimSpace(migrationID)).Scan(
		&migration.MigrationID, &migration.LegacyWorkItemKey, &migration.TargetWorkItemKey,
		&migration.ResourceType, &migration.TargetScope, &migration.InstallationID,
		&migration.Repository, &migration.PRNumber, &migration.ChatIDHash,
		&migration.RemoteIDHash, &migration.Status, &migration.ReasonCode,
		&migration.VerificationMethod, &migration.VerifiedByHash, &migration.VerifiedAt,
		&migration.AttemptCount, &migration.ErrorSummary, &migration.CreatedAt, &migration.UpdatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return ReviewResourceMigration{}, fmt.Errorf("review resource migration %q not found", migrationID)
	}
	return migration, err
}

func (s *SQLiteReviewGatewayStore) ListReviewResourceMigrations(
	ctx context.Context,
	status, resourceType, installationID string,
) ([]ReviewResourceMigration, error) {
	query := `SELECT migration_id FROM review_resource_migrations WHERE 1=1`
	args := []interface{}{}
	for _, filter := range []struct{ column, value string }{
		{"status", status}, {"resource_type", resourceType}, {"installation_id", installationID},
	} {
		if strings.TrimSpace(filter.value) != "" {
			query += " AND " + filter.column + "=?"
			args = append(args, strings.TrimSpace(filter.value))
		}
	}
	query += " ORDER BY updated_at, migration_id"
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	ids := []string{}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			_ = rows.Close()
			return nil, err
		}
		ids = append(ids, id)
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	result := make([]ReviewResourceMigration, 0, len(ids))
	for _, id := range ids {
		migration, err := s.GetReviewResourceMigration(ctx, id)
		if err != nil {
			return nil, err
		}
		result = append(result, migration)
	}
	return result, nil
}

func (s *SQLiteReviewGatewayStore) VerifyLegacyReviewResourceMigration(
	ctx context.Context,
	migrationID string,
	input LegacyReviewResourceVerificationInput,
	verifier LegacyReviewResourceVerifier,
	now time.Time,
) (ReviewResourceMigration, error) {
	if verifier == nil {
		return ReviewResourceMigration{}, fmt.Errorf("legacy Review resource verifier is required")
	}
	migration, err := s.GetReviewResourceMigration(ctx, migrationID)
	if err != nil {
		return ReviewResourceMigration{}, err
	}
	if migration.Status != ReviewResourceMigrationNeedsReconciliation {
		return migration, fmt.Errorf("%w: only needs_reconciliation can be verified", ErrReviewResourceMigrationConflict)
	}
	input.MigrationID = migrationID
	if err := verifier.Verify(ctx, migration, input); err != nil {
		return migration, err
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return migration, err
	}
	defer tx.Rollback()
	nowText := now.UTC().Format(time.RFC3339Nano)
	result, err := tx.ExecContext(ctx, `UPDATE review_resource_migrations SET
		status=?, reason_code='', verification_method=?, verified_by_hash=?, verified_at=?,
		error_summary='', updated_at=? WHERE migration_id=? AND status=?`,
		ReviewResourceMigrationVerified, input.Method, reviewResourceIdentifierHash(input.Actor),
		nowText, nowText, migrationID, ReviewResourceMigrationNeedsReconciliation,
	)
	if err != nil {
		return migration, err
	}
	if err := requireReviewResourceMigrationRows(result); err != nil {
		return migration, err
	}
	if err := appendReviewResourceMigrationAudit(ctx, tx, migration, "verified",
		ReviewResourceMigrationNeedsReconciliation, ReviewResourceMigrationVerified,
		input.Actor, map[string]interface{}{"method": input.Method}, now); err != nil {
		return migration, err
	}
	if err := tx.Commit(); err != nil {
		return migration, err
	}
	return s.GetReviewResourceMigration(ctx, migrationID)
}

func (s *SQLiteReviewGatewayStore) ApplyLegacyReviewResourceMigration(
	ctx context.Context,
	migrationID, actor string,
	now time.Time,
) (ReviewResourceMigration, error) {
	migration, err := s.GetReviewResourceMigration(ctx, migrationID)
	if err != nil {
		return ReviewResourceMigration{}, err
	}
	if migration.Status != ReviewResourceMigrationVerified {
		return migration, fmt.Errorf("%w: only verified migration can be applied", ErrReviewResourceMigrationConflict)
	}
	if err := s.beginReviewResourceMigrationApply(ctx, migration, actor, now); err != nil {
		return migration, err
	}
	result, err := s.applyLegacyReviewResourceMigration(ctx, migrationID, actor, now)
	if err == nil {
		return result, nil
	}
	_ = s.failReviewResourceMigration(ctx, migrationID, actor, err, now)
	failed, readErr := s.GetReviewResourceMigration(ctx, migrationID)
	if readErr != nil {
		return migration, err
	}
	return failed, err
}

func (s *SQLiteReviewGatewayStore) beginReviewResourceMigrationApply(
	ctx context.Context, migration ReviewResourceMigration, actor string, now time.Time,
) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	result, err := tx.ExecContext(ctx, `UPDATE review_resource_migrations SET
		status=?, attempt_count=attempt_count+1, updated_at=?
		WHERE migration_id=? AND status=?`, ReviewResourceMigrationApplying,
		now.UTC().Format(time.RFC3339Nano), migration.MigrationID, ReviewResourceMigrationVerified)
	if err != nil {
		return err
	}
	if err := requireReviewResourceMigrationRows(result); err != nil {
		return err
	}
	if err := appendReviewResourceMigrationAudit(ctx, tx, migration, "applied",
		ReviewResourceMigrationVerified, ReviewResourceMigrationApplying, actor,
		map[string]interface{}{"phase": "begin"}, now); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *SQLiteReviewGatewayStore) applyLegacyReviewResourceMigration(
	ctx context.Context, migrationID, actor string, now time.Time,
) (ReviewResourceMigration, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return ReviewResourceMigration{}, err
	}
	defer tx.Rollback()
	migration, err := readReviewResourceMigration(ctx, tx, migrationID)
	if err != nil {
		return migration, err
	}
	if migration.Status != ReviewResourceMigrationApplying {
		return migration, ErrReviewResourceMigrationConflict
	}
	var remoteID, fingerprint, sourceUpdatedAt string
	err = tx.QueryRowContext(ctx, `SELECT remote_id, content_fingerprint, updated_at
		FROM review_collaboration_resources WHERE work_item_key=? AND resource_type=?`,
		migration.LegacyWorkItemKey, migration.ResourceType,
	).Scan(&remoteID, &fingerprint, &sourceUpdatedAt)
	if err != nil || strings.TrimSpace(remoteID) == "" {
		return migration, fmt.Errorf("legacy Review resource source is unavailable")
	}
	if reviewResourceIdentifierHash(remoteID) != migration.RemoteIDHash {
		return migration, fmt.Errorf("legacy Review resource changed after verification")
	}
	var existing string
	targetWorkItemKey, _, err := resolveReviewResourceMigrationTarget(ctx, tx, migration)
	if err != nil {
		return migration, err
	}
	err = tx.QueryRowContext(ctx, `SELECT remote_id FROM review_collaboration_resources
		WHERE work_item_key=? AND resource_type=?`, targetWorkItemKey, migration.ResourceType).Scan(&existing)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return migration, err
	}
	if existing != "" {
		if existing == remoteID {
			if err := completeReviewResourceMigration(ctx, tx, migration, ReviewResourceMigrationMigrated,
				"target_already_matches", "applied", actor, now); err != nil {
				return migration, err
			}
		} else {
			if err := completeReviewResourceMigration(ctx, tx, migration, ReviewResourceMigrationSuperseded,
				"target_resource_already_exists", "superseded", actor, now); err != nil {
				return migration, err
			}
		}
		if err := tx.Commit(); err != nil {
			return migration, err
		}
		return s.GetReviewResourceMigration(ctx, migrationID)
	}
	if migration.ResourceType == ReviewResourceCard {
		outcome, err := adoptVerifiedLegacyCanonicalCard(ctx, tx, migration, remoteID, now)
		if err != nil {
			return migration, err
		}
		if outcome == ReviewResourceMigrationSuperseded {
			if err := completeReviewResourceMigration(ctx, tx, migration, outcome,
				"target_resource_already_exists", "superseded", actor, now); err != nil {
				return migration, err
			}
			if err := tx.Commit(); err != nil {
				return migration, err
			}
			return s.GetReviewResourceMigration(ctx, migrationID)
		}
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO review_collaboration_resources (
		work_item_key, resource_type, remote_id, content_fingerprint, updated_at
	) VALUES (?, ?, ?, ?, ?)`, targetWorkItemKey, migration.ResourceType,
		remoteID, fingerprint, sourceUpdatedAt); err != nil {
		return migration, err
	}
	if err := completeReviewResourceMigration(ctx, tx, migration, ReviewResourceMigrationMigrated,
		"", "applied", actor, now); err != nil {
		return migration, err
	}
	if err := tx.Commit(); err != nil {
		return migration, err
	}
	return s.GetReviewResourceMigration(ctx, migrationID)
}

func resolveReviewResourceMigrationTarget(
	ctx context.Context, tx *sql.Tx, migration ReviewResourceMigration,
) (string, string, error) {
	chatID := ""
	if migration.TargetScope == ReviewResourceScopeChat {
		err := tx.QueryRowContext(ctx, `SELECT chat_id FROM review_collaboration_items
			WHERE repository=? AND pr_number=?`, migration.Repository, migration.PRNumber).Scan(&chatID)
		if err != nil {
			return "", "", err
		}
		if reviewResourceIdentifierHash(chatID) != migration.ChatIDHash {
			return "", "", fmt.Errorf("legacy Review resource chat changed after verification")
		}
	}
	target, err := ResolveReviewResourceTarget(
		migration.ResourceType, migration.InstallationID, chatID,
		migration.Repository, migration.PRNumber, migration.TargetScope,
	)
	if err != nil {
		return "", "", err
	}
	if reviewResourceMigrationTargetReference(target, migration.TargetScope) != migration.TargetWorkItemKey {
		return "", "", fmt.Errorf("review resource migration target changed after verification")
	}
	return target, chatID, nil
}

func adoptVerifiedLegacyCanonicalCard(
	ctx context.Context, tx *sql.Tx, migration ReviewResourceMigration, remoteID string, now time.Time,
) (string, error) {
	_, chatID, err := resolveReviewResourceMigrationTarget(ctx, tx, migration)
	if err != nil {
		return "", err
	}
	job := ReviewGatewayJob{InstallationID: migration.InstallationID, ChatID: chatID,
		Repository: migration.Repository, PRNumber: migration.PRNumber}
	key := reviewChatPRPresentationKey(job)
	var existing string
	err = tx.QueryRowContext(ctx, `SELECT canonical_message_id FROM chat_pr_presentations
		WHERE presentation_key=?`, key).Scan(&existing)
	if err == nil {
		if existing == remoteID {
			return ReviewResourceMigrationMigrated, nil
		}
		return ReviewResourceMigrationSuperseded, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return "", err
	}
	nowText := now.UTC().Format(time.RFC3339Nano)
	_, err = tx.ExecContext(ctx, `INSERT INTO chat_pr_presentations (
		presentation_key, app_scope, installation_id, chat_id, repository, pr_number,
		canonical_message_id, content_fingerprint, source_completed_at,
		presentation_version, card_status, last_operation_id, requires_reconciliation,
		created_at, updated_at
	) VALUES (?, ?, ?, ?, ?, ?, ?, '', '', 1, 'active', ?, 0, ?, ?)`,
		key, reviewGatewayAppScope, migration.InstallationID, chatID, migration.Repository,
		migration.PRNumber, remoteID, migration.MigrationID, nowText, nowText)
	if err != nil {
		return "", err
	}
	return ReviewResourceMigrationMigrated, nil
}

func completeReviewResourceMigration(
	ctx context.Context, tx *sql.Tx, migration ReviewResourceMigration,
	status, reason, action, actor string, now time.Time,
) error {
	result, err := tx.ExecContext(ctx, `UPDATE review_resource_migrations SET
		status=?, reason_code=?, error_summary='', updated_at=?
		WHERE migration_id=? AND status=?`, status, reason,
		now.UTC().Format(time.RFC3339Nano), migration.MigrationID, ReviewResourceMigrationApplying)
	if err != nil {
		return err
	}
	if err := requireReviewResourceMigrationRows(result); err != nil {
		return err
	}
	return appendReviewResourceMigrationAudit(ctx, tx, migration, action,
		ReviewResourceMigrationApplying, status, actor, map[string]interface{}{"reason_code": reason}, now)
}

func (s *SQLiteReviewGatewayStore) failReviewResourceMigration(
	ctx context.Context, migrationID, actor string, applyErr error, now time.Time,
) error {
	migration, err := s.GetReviewResourceMigration(ctx, migrationID)
	if err != nil {
		return err
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	summary := redactReviewGatewayError(applyErr.Error())
	result, err := tx.ExecContext(ctx, `UPDATE review_resource_migrations SET
		status=?, reason_code='apply_failed', error_summary=?, updated_at=?
		WHERE migration_id=? AND status=?`, ReviewResourceMigrationFailed, summary,
		now.UTC().Format(time.RFC3339Nano), migrationID, ReviewResourceMigrationApplying)
	if err != nil {
		return err
	}
	if err := requireReviewResourceMigrationRows(result); err != nil {
		return err
	}
	if err := appendReviewResourceMigrationAudit(ctx, tx, migration, "failed",
		ReviewResourceMigrationApplying, ReviewResourceMigrationFailed, actor,
		map[string]interface{}{"error_summary": summary}, now); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *SQLiteReviewGatewayStore) RetryLegacyReviewResourceMigration(
	ctx context.Context, migrationID, actor string, now time.Time,
) (ReviewResourceMigration, error) {
	return s.transitionReviewResourceMigration(ctx, migrationID, ReviewResourceMigrationFailed,
		ReviewResourceMigrationVerified, "retried", actor, "", now)
}

func (s *SQLiteReviewGatewayStore) SkipLegacyReviewResourceMigration(
	ctx context.Context, migrationID, reason, actor string, now time.Time,
) (ReviewResourceMigration, error) {
	if strings.TrimSpace(reason) == "" {
		return ReviewResourceMigration{}, fmt.Errorf("migration skip reason is required")
	}
	migration, err := s.GetReviewResourceMigration(ctx, migrationID)
	if err != nil {
		return migration, err
	}
	if migration.Status != ReviewResourceMigrationDiscovered && migration.Status != ReviewResourceMigrationNeedsReconciliation {
		return migration, ErrReviewResourceMigrationConflict
	}
	target := ReviewResourceMigrationSkippedAmbiguous
	if migration.TargetScope == ReviewResourceScopeDisabled {
		target = ReviewResourceMigrationSkippedDisabled
	}
	return s.transitionReviewResourceMigration(ctx, migrationID, migration.Status, target, "skipped", actor, reason, now)
}

func (s *SQLiteReviewGatewayStore) transitionReviewResourceMigration(
	ctx context.Context, migrationID, from, to, action, actor, reason string, now time.Time,
) (ReviewResourceMigration, error) {
	migration, err := s.GetReviewResourceMigration(ctx, migrationID)
	if err != nil {
		return migration, err
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return migration, err
	}
	defer tx.Rollback()
	result, err := tx.ExecContext(ctx, `UPDATE review_resource_migrations SET
		status=?, reason_code=?, error_summary='', updated_at=?
		WHERE migration_id=? AND status=?`, to, reason, now.UTC().Format(time.RFC3339Nano), migrationID, from)
	if err != nil {
		return migration, err
	}
	if err := requireReviewResourceMigrationRows(result); err != nil {
		return migration, err
	}
	if err := appendReviewResourceMigrationAudit(ctx, tx, migration, action, from, to, actor,
		map[string]interface{}{"reason_code": reason}, now); err != nil {
		return migration, err
	}
	if err := tx.Commit(); err != nil {
		return migration, err
	}
	return s.GetReviewResourceMigration(ctx, migrationID)
}

func requireReviewResourceMigrationRows(result sql.Result) error {
	if result == nil {
		return ErrReviewResourceMigrationConflict
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected != 1 {
		return ErrReviewResourceMigrationConflict
	}
	return nil
}

func appendReviewResourceMigrationAudit(
	ctx context.Context, tx *sql.Tx, migration ReviewResourceMigration,
	action, from, to, actor string, detail map[string]interface{}, now time.Time,
) error {
	clean := map[string]interface{}{}
	keys := make([]string, 0, len(detail))
	for key := range detail {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		value := detail[key]
		if strings.Contains(strings.ToLower(key), "remote_id") || strings.Contains(strings.ToLower(key), "chat_id") {
			if text, ok := value.(string); ok && text != "" && len(text) != 64 {
				value = reviewResourceIdentifierHash(text)
			}
		}
		clean[key] = value
	}
	encoded, err := json.Marshal(clean)
	if err != nil {
		return fmt.Errorf("encode review migration audit: %w", err)
	}
	nowText := now.UTC().Format(time.RFC3339Nano)
	auditID := stableKey("review-resource-migration-audit", migration.MigrationID, action, from, to, nowText, fmt.Sprintf("%d", len(encoded)))
	_, err = tx.ExecContext(ctx, `INSERT INTO review_resource_migration_audit (
		audit_id, migration_id, action, from_status, to_status, actor_hash, detail_json, created_at
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`, auditID, migration.MigrationID, action, from, to,
		reviewResourceIdentifierHash(actor), string(encoded), nowText)
	return err
}

func (s *SQLiteReviewGatewayStore) ListReviewResourceMigrationAudit(
	ctx context.Context, migrationID string,
) ([]ReviewResourceMigrationAudit, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT audit_id, migration_id, action, from_status,
		to_status, actor_hash, detail_json, created_at
		FROM review_resource_migration_audit WHERE migration_id=? ORDER BY created_at, audit_id`, migrationID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []ReviewResourceMigrationAudit{}
	for rows.Next() {
		var audit ReviewResourceMigrationAudit
		var detail string
		if err := rows.Scan(&audit.AuditID, &audit.MigrationID, &audit.Action,
			&audit.FromStatus, &audit.ToStatus, &audit.ActorHash, &detail, &audit.CreatedAt); err != nil {
			return nil, err
		}
		if !json.Valid([]byte(detail)) {
			return nil, fmt.Errorf("invalid review migration audit JSON")
		}
		audit.DetailJSON = json.RawMessage(detail)
		result = append(result, audit)
	}
	return result, rows.Err()
}
