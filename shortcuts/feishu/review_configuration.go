package feishu

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

var ErrReviewConfigurationConflict = errors.New("configuration revision conflict")

type ReviewConfigurationApplyOptions struct {
	Source           string
	ActorID          string
	ExpectedRevision *int
	WorkerConfig     map[string]int
	LimitConfig      map[string]int
	ServiceConfig    ReviewServiceConfig
}

type ReviewConfigurationState struct {
	ConfigRevision    int    `json:"config_revision"`
	ConfigFingerprint string `json:"config_fingerprint"`
	Source            string `json:"source"`
	SourceHash        string `json:"source_hash"`
	AppliedByHash     string `json:"applied_by_hash"`
	AppliedAt         string `json:"applied_at"`
}

type reviewConfigurationRevisionPayload struct {
	Fingerprint         string
	WorkerJSON          string
	LimitJSON           string
	ServiceJSON         string
	SubscriptionCount   int
	ResourcePolicyCount int
}

type reviewConfigurationEntitySnapshot struct {
	Fingerprint string
	Revision    int
	CreatedAt   string
}

type reviewConfigurationCanonical struct {
	Bindings         ReviewGatewayBindings    `json:"bindings"`
	IdentityHashes   []string                 `json:"identity_hashes"`
	Subscriptions    []map[string]interface{} `json:"subscriptions"`
	ResourcePolicies []map[string]interface{} `json:"resource_policies"`
	WorkerConfig     map[string]int           `json:"worker_config"`
	LimitConfig      map[string]int           `json:"limit_config"`
	ServiceConfig    reviewServiceSafeConfig  `json:"service_config"`
}

type reviewServiceSafeConfig struct {
	AdminListen             string `json:"admin_listen"`
	WebhookListen           string `json:"webhook_listen,omitempty"`
	ShutdownTimeout         string `json:"shutdown_timeout"`
	ReadDrainTimeout        string `json:"read_drain_timeout"`
	OperationDrainTimeout   string `json:"operation_drain_timeout"`
	WorkerHeartbeatInterval string `json:"worker_heartbeat_interval"`
	WorkerStaleAfter        string `json:"worker_stale_after"`
	MetricsEnabled          bool   `json:"metrics_enabled"`
	BackupInterval          string `json:"backup_interval,omitempty"`
	BackupRetentionCount    int    `json:"backup_retention_count"`
	RetentionInterval       string `json:"retention_interval,omitempty"`
	WALCheckpointInterval   string `json:"wal_checkpoint_interval,omitempty"`
	InstanceHeartbeat       string `json:"instance_heartbeat_interval"`
	ServiceVersion          string `json:"service_version,omitempty"`
	CommitSHA               string `json:"commit_sha,omitempty"`
}

func (s *SQLiteReviewGatewayStore) CurrentReviewConfigurationState(ctx context.Context) (ReviewConfigurationState, error) {
	if s == nil || s.db == nil {
		return ReviewConfigurationState{}, fmt.Errorf("review gateway state store is required")
	}
	return readReviewConfigurationState(ctx, s.db)
}

func readReviewConfigurationState(ctx context.Context, queryer interface {
	QueryRowContext(context.Context, string, ...interface{}) *sql.Row
}) (ReviewConfigurationState, error) {
	var state ReviewConfigurationState
	err := queryer.QueryRowContext(ctx, `SELECT config_revision,config_fingerprint,source,source_hash,applied_by_hash,applied_at
		FROM review_configuration_state WHERE singleton_id=1`).Scan(&state.ConfigRevision, &state.ConfigFingerprint, &state.Source, &state.SourceHash, &state.AppliedByHash, &state.AppliedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return ReviewConfigurationState{}, nil
	}
	return state, err
}

func (s *SQLiteReviewGatewayStore) ApplyReviewGatewayConfiguration(ctx context.Context, bindings ReviewGatewayBindings, options ReviewConfigurationApplyOptions, now time.Time) (ReviewConfigurationState, bool, error) {
	if s == nil || s.db == nil {
		return ReviewConfigurationState{}, false, fmt.Errorf("review gateway state store is required")
	}
	normalized, err := normalizeReviewGatewayBindings(bindings)
	if err != nil {
		return ReviewConfigurationState{}, false, fmt.Errorf("validate review gateway configuration: %w", err)
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return ReviewConfigurationState{}, false, fmt.Errorf("begin review gateway configuration sync: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	current, err := readReviewConfigurationState(ctx, tx)
	if err != nil {
		return ReviewConfigurationState{}, false, fmt.Errorf("read review configuration revision: %w", err)
	}
	if options.ExpectedRevision != nil && current.ConfigRevision != *options.ExpectedRevision {
		return current, false, ErrReviewConfigurationConflict
	}
	payload, err := buildReviewConfigurationRevisionPayload(ctx, tx, normalized, options)
	if err != nil {
		return ReviewConfigurationState{}, false, err
	}
	if current.ConfigRevision > 0 && current.ConfigFingerprint == payload.Fingerprint {
		return current, false, tx.Commit()
	}

	appliedAt := reviewGatewayTimestamp(now)
	actorHash := reviewConfigurationIdentifierHash(options.ActorID)
	snapshots, err := loadReviewConfigurationEntitySnapshots(ctx, tx)
	if err != nil {
		return ReviewConfigurationState{}, false, err
	}
	for _, statement := range []string{"DELETE FROM review_identity_bindings", "DELETE FROM chat_repository_bindings", "DELETE FROM installation_repositories", "DELETE FROM gitlink_installations"} {
		if _, err := tx.ExecContext(ctx, statement); err != nil {
			return ReviewConfigurationState{}, false, fmt.Errorf("clear review gateway configuration snapshot: %w", err)
		}
	}
	repositoryCount, err := writeReviewConfigurationEntities(ctx, tx, normalized, snapshots, actorHash, appliedAt)
	if err != nil {
		return ReviewConfigurationState{}, false, err
	}
	source, sourceHash := reviewConfigurationSource(options.Source)
	next := ReviewConfigurationState{ConfigRevision: current.ConfigRevision + 1, ConfigFingerprint: payload.Fingerprint, Source: source, SourceHash: sourceHash, AppliedByHash: actorHash, AppliedAt: appliedAt}
	revisionID := fmt.Sprintf("configuration-%d-%s", next.ConfigRevision, payload.Fingerprint[:12])
	if _, err := tx.ExecContext(ctx, `INSERT INTO review_configuration_revisions(
		revision_id,config_revision,config_fingerprint,source,source_hash,installation_count,
		repository_count,binding_count,subscription_count,identity_binding_count,resource_policy_count,
		worker_config_json,limit_config_json,service_config_json,applied_by_hash,applied_at
	) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`, revisionID, next.ConfigRevision, next.ConfigFingerprint, next.Source, next.SourceHash,
		len(normalized.Installations), repositoryCount, countReviewConfigurationBindings(normalized), payload.SubscriptionCount,
		len(normalized.IdentityBindings), payload.ResourcePolicyCount, payload.WorkerJSON, payload.LimitJSON, payload.ServiceJSON, actorHash, appliedAt); err != nil {
		return ReviewConfigurationState{}, false, fmt.Errorf("append review configuration revision: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO review_configuration_state(singleton_id,config_revision,config_fingerprint,source,source_hash,applied_by_hash,applied_at)
		VALUES(1,?,?,?,?,?,?) ON CONFLICT(singleton_id) DO UPDATE SET config_revision=excluded.config_revision,
		config_fingerprint=excluded.config_fingerprint,source=excluded.source,source_hash=excluded.source_hash,
		applied_by_hash=excluded.applied_by_hash,applied_at=excluded.applied_at`, next.ConfigRevision, next.ConfigFingerprint, next.Source, next.SourceHash, next.AppliedByHash, next.AppliedAt); err != nil {
		return ReviewConfigurationState{}, false, fmt.Errorf("update review configuration state: %w", err)
	}
	auditID := fmt.Sprintf("config-%d-%s", now.UTC().UnixNano(), payload.Fingerprint[:12])
	if _, err := tx.ExecContext(ctx, `INSERT INTO review_gateway_configuration_audit(audit_id,config_fingerprint,source,installation_count,binding_count,repository_count,applied_at)
		VALUES(?,?,?,?,?,?,?)`, auditID, payload.Fingerprint, source, len(normalized.Installations), len(normalized.Bindings), repositoryCount, appliedAt); err != nil {
		return ReviewConfigurationState{}, false, fmt.Errorf("audit review gateway configuration: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return ReviewConfigurationState{}, false, fmt.Errorf("commit review gateway configuration sync: %w", err)
	}
	return next, true, nil
}

func buildReviewConfigurationRevisionPayload(ctx context.Context, tx *sql.Tx, bindings ReviewGatewayBindings, options ReviewConfigurationApplyOptions) (reviewConfigurationRevisionPayload, error) {
	workerJSON, err := reviewConfigurationMapJSON(options.WorkerConfig)
	if err != nil {
		return reviewConfigurationRevisionPayload{}, err
	}
	limitJSON, err := reviewConfigurationMapJSON(options.LimitConfig)
	if err != nil {
		return reviewConfigurationRevisionPayload{}, err
	}
	safeService := safeReviewServiceConfig(options.ServiceConfig.normalized())
	serviceBytes, err := json.Marshal(safeService)
	if err != nil {
		return reviewConfigurationRevisionPayload{}, fmt.Errorf("encode safe review service configuration: %w", err)
	}
	subscriptions, err := readReviewConfigurationRows(ctx, tx, `SELECT installation_id,chat_id,repository,pull_events,review_events,thread_events,merge_events,ci_events,notification_mode,enabled,revision FROM review_chat_subscriptions ORDER BY installation_id,chat_id,repository`)
	if err != nil {
		return reviewConfigurationRevisionPayload{}, fmt.Errorf("read subscriptions for configuration fingerprint: %w", err)
	}
	policies, err := readReviewConfigurationRows(ctx, tx, `SELECT installation_id,resource_type,target_scope,migration_enabled,revision FROM review_resource_scope_policies ORDER BY installation_id,resource_type`)
	if err != nil {
		return reviewConfigurationRevisionPayload{}, fmt.Errorf("read resource policies for configuration fingerprint: %w", err)
	}
	identityHashes := make([]string, 0, len(bindings.IdentityBindings))
	canonicalBindings := bindings
	canonicalBindings.IdentityBindings = nil
	for _, identity := range bindings.IdentityBindings {
		identityHashes = append(identityHashes, reviewConfigurationIdentifierHash(identity.InstallationID+"\x00"+identity.FeishuUserID+"\x00"+identity.GitLinkLogin+"\x00"+identity.VerificationMethod+"\x00"+identity.VerifiedAt+fmt.Sprintf("\x00%t", identity.Enabled)))
	}
	sort.Strings(identityHashes)
	canonical := reviewConfigurationCanonical{Bindings: canonicalBindings, IdentityHashes: identityHashes, Subscriptions: subscriptions, ResourcePolicies: policies, WorkerConfig: nonNilReviewConfigurationMap(options.WorkerConfig), LimitConfig: nonNilReviewConfigurationMap(options.LimitConfig), ServiceConfig: safeService}
	encoded, err := json.Marshal(canonical)
	if err != nil {
		return reviewConfigurationRevisionPayload{}, fmt.Errorf("encode review configuration fingerprint: %w", err)
	}
	digest := sha256.Sum256(encoded)
	return reviewConfigurationRevisionPayload{Fingerprint: hex.EncodeToString(digest[:]), WorkerJSON: workerJSON, LimitJSON: limitJSON, ServiceJSON: string(serviceBytes), SubscriptionCount: len(subscriptions), ResourcePolicyCount: len(policies)}, nil
}

func readReviewConfigurationRows(ctx context.Context, tx *sql.Tx, query string) ([]map[string]interface{}, error) {
	rows, err := tx.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	columns, err := rows.Columns()
	if err != nil {
		return nil, err
	}
	result := []map[string]interface{}{}
	for rows.Next() {
		values := make([]interface{}, len(columns))
		pointers := make([]interface{}, len(columns))
		for index := range values {
			pointers[index] = &values[index]
		}
		if err := rows.Scan(pointers...); err != nil {
			return nil, err
		}
		item := map[string]interface{}{}
		for index, name := range columns {
			switch value := values[index].(type) {
			case []byte:
				item[name] = string(value)
			default:
				item[name] = value
			}
		}
		result = append(result, item)
	}
	return result, rows.Err()
}

func reviewConfigurationMapJSON(values map[string]int) (string, error) {
	encoded, err := json.Marshal(nonNilReviewConfigurationMap(values))
	if err != nil {
		return "", fmt.Errorf("encode review configuration limits: %w", err)
	}
	return string(encoded), nil
}

func nonNilReviewConfigurationMap(values map[string]int) map[string]int {
	if values == nil {
		return map[string]int{}
	}
	return values
}

func safeReviewServiceConfig(config ReviewServiceConfig) reviewServiceSafeConfig {
	return reviewServiceSafeConfig{AdminListen: config.AdminListen, WebhookListen: config.WebhookListen, ShutdownTimeout: config.ShutdownTimeout.String(), ReadDrainTimeout: config.ReadDrainTimeout.String(), OperationDrainTimeout: config.OperationDrainTimeout.String(), WorkerHeartbeatInterval: config.WorkerHeartbeatInterval.String(), WorkerStaleAfter: config.WorkerStaleAfter.String(), MetricsEnabled: config.MetricsEnabled, BackupInterval: config.BackupInterval.String(), BackupRetentionCount: config.BackupRetentionCount, RetentionInterval: config.RetentionInterval.String(), WALCheckpointInterval: config.WALCheckpointInterval.String(), InstanceHeartbeat: config.InstanceHeartbeat.String(), ServiceVersion: strings.TrimSpace(config.ServiceVersion), CommitSHA: strings.TrimSpace(config.CommitSHA)}
}

func reviewConfigurationSource(source string) (string, string) {
	original := strings.TrimSpace(source)
	if original == "" {
		original = "unknown"
	}
	safe := original
	if filepath.IsAbs(original) || strings.ContainsAny(original, `/\\`) {
		safe = filepath.Base(original)
	}
	if safe == "." || safe == string(filepath.Separator) || safe == "" {
		safe = "file"
	}
	return safe, reviewConfigurationIdentifierHash(original)
}

func reviewConfigurationIdentifierHash(value string) string {
	if strings.TrimSpace(value) == "" {
		return ""
	}
	digest := sha256.Sum256([]byte(value))
	return hex.EncodeToString(digest[:])[:16]
}

func countReviewConfigurationBindings(bindings ReviewGatewayBindings) int {
	count := 0
	for _, binding := range bindings.Bindings {
		count += len(binding.Repositories)
	}
	return count
}

func reviewEntityState(fingerprint string, snapshot reviewConfigurationEntitySnapshot, actorHash, appliedAt string) (int, string, string) {
	if snapshot.Revision == 0 {
		return 1, appliedAt, actorHash
	}
	if snapshot.Fingerprint == fingerprint {
		return snapshot.Revision, snapshot.CreatedAt, actorHash
	}
	return snapshot.Revision + 1, snapshot.CreatedAt, actorHash
}

func reviewEntityFingerprint(values ...interface{}) string {
	encoded, _ := json.Marshal(values)
	digest := sha256.Sum256(encoded)
	return hex.EncodeToString(digest[:])
}

func loadReviewConfigurationEntitySnapshots(ctx context.Context, tx *sql.Tx) (map[string]reviewConfigurationEntitySnapshot, error) {
	result := map[string]reviewConfigurationEntitySnapshot{}
	queries := []struct {
		prefix string
		query  string
		read   func(*sql.Rows) (string, string, int, string, error)
	}{
		{"installation", `SELECT installation_id,gitlink_host,owner,credential_ref,operation_mode,allow_public_read,webhook_id,webhook_secret_ref,webhook_signature_mode,webhook_timestamp_mode,webhook_max_skew_seconds,webhook_delivery_required,enabled,revision,created_at FROM gitlink_installations`, readReviewInstallationSnapshot},
		{"repository", `SELECT installation_id,repository,revision,created_at FROM installation_repositories`, readReviewRepositorySnapshot},
		{"binding", `SELECT chat_id,installation_id,repository,is_default,enabled,allow_public_read,admin_user_ids_json,allowed_user_ids_json,revision,created_at FROM chat_repository_bindings`, readReviewBindingSnapshot},
		{"identity", `SELECT installation_id,feishu_user_id,gitlink_login,verification_method,verified_at,enabled,revision,created_at FROM review_identity_bindings`, readReviewIdentitySnapshot},
	}
	for _, item := range queries {
		rows, err := tx.QueryContext(ctx, item.query)
		if err != nil {
			return nil, fmt.Errorf("read %s configuration revisions: %w", item.prefix, err)
		}
		for rows.Next() {
			key, fingerprint, revision, createdAt, err := item.read(rows)
			if err != nil {
				_ = rows.Close()
				return nil, err
			}
			result[item.prefix+":"+key] = reviewConfigurationEntitySnapshot{Fingerprint: fingerprint, Revision: revision, CreatedAt: createdAt}
		}
		if err := rows.Close(); err != nil {
			return nil, err
		}
	}
	return result, nil
}

func readReviewInstallationSnapshot(rows *sql.Rows) (string, string, int, string, error) {
	var id, host, owner, credential, mode, webhookID, webhookRef, signatureMode, timestampMode, created string
	var publicRead, skew, delivery, enabled, revision int
	err := rows.Scan(&id, &host, &owner, &credential, &mode, &publicRead, &webhookID, &webhookRef, &signatureMode, &timestampMode, &skew, &delivery, &enabled, &revision, &created)
	return id, reviewEntityFingerprint(host, owner, credential, mode, publicRead, webhookID, webhookRef, signatureMode, timestampMode, skew, delivery, enabled), revision, created, err
}

func readReviewRepositorySnapshot(rows *sql.Rows) (string, string, int, string, error) {
	var installation, repository, created string
	var revision int
	err := rows.Scan(&installation, &repository, &revision, &created)
	return installation + "\x00" + repository, reviewEntityFingerprint(installation, repository), revision, created, err
}

func readReviewBindingSnapshot(rows *sql.Rows) (string, string, int, string, error) {
	var chat, installation, repository, admins, allowed, created string
	var isDefault, enabled, publicRead, revision int
	err := rows.Scan(&chat, &installation, &repository, &isDefault, &enabled, &publicRead, &admins, &allowed, &revision, &created)
	return chat + "\x00" + installation + "\x00" + repository, reviewEntityFingerprint(chat, installation, repository, isDefault, enabled, publicRead, admins, allowed), revision, created, err
}

func readReviewIdentitySnapshot(rows *sql.Rows) (string, string, int, string, error) {
	var installation, user, login, method, verified, created string
	var enabled, revision int
	err := rows.Scan(&installation, &user, &login, &method, &verified, &enabled, &revision, &created)
	return installation + "\x00" + user, reviewEntityFingerprint(installation, user, login, method, verified, enabled), revision, created, err
}

func writeReviewConfigurationEntities(ctx context.Context, tx *sql.Tx, bindings ReviewGatewayBindings, snapshots map[string]reviewConfigurationEntitySnapshot, actorHash, appliedAt string) (int, error) {
	repositoryCount := 0
	for _, installation := range bindings.Installations {
		fingerprint := reviewEntityFingerprint(installation.GitLinkHost, installation.Owner, installation.CredentialRef, installation.OperationMode, boolToSQLiteInteger(installation.AllowPublicRead), installation.WebhookID, installation.WebhookSecretRef, installation.WebhookSignatureMode, installation.WebhookTimestampMode, installation.WebhookMaxSkewSeconds, boolToSQLiteInteger(installation.WebhookDeliveryRequired), boolToSQLiteInteger(installation.Enabled))
		revision, createdAt, updatedBy := reviewEntityState(fingerprint, snapshots["installation:"+installation.InstallationID], actorHash, appliedAt)
		if _, err := tx.ExecContext(ctx, `INSERT INTO gitlink_installations(installation_id,gitlink_host,owner,credential_ref,operation_mode,allow_public_read,webhook_id,webhook_secret_ref,webhook_signature_mode,webhook_timestamp_mode,webhook_max_skew_seconds,webhook_delivery_required,enabled,updated_at,revision,created_at,updated_by_hash) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`, installation.InstallationID, installation.GitLinkHost, installation.Owner, installation.CredentialRef, installation.OperationMode, boolToSQLiteInteger(installation.AllowPublicRead), installation.WebhookID, installation.WebhookSecretRef, installation.WebhookSignatureMode, installation.WebhookTimestampMode, installation.WebhookMaxSkewSeconds, boolToSQLiteInteger(installation.WebhookDeliveryRequired), boolToSQLiteInteger(installation.Enabled), appliedAt, revision, createdAt, updatedBy); err != nil {
			return 0, fmt.Errorf("store GitLink installation %q: %w", installation.InstallationID, err)
		}
		for _, repository := range installation.AllowedRepositories {
			key := installation.InstallationID + "\x00" + repository
			fingerprint := reviewEntityFingerprint(installation.InstallationID, repository)
			revision, createdAt, updatedBy := reviewEntityState(fingerprint, snapshots["repository:"+key], actorHash, appliedAt)
			if _, err := tx.ExecContext(ctx, `INSERT INTO installation_repositories(installation_id,repository,updated_at,revision,created_at,updated_by_hash) VALUES(?,?,?,?,?,?)`, installation.InstallationID, repository, appliedAt, revision, createdAt, updatedBy); err != nil {
				return 0, fmt.Errorf("store GitLink installation repository %q: %w", repository, err)
			}
			repositoryCount++
		}
	}
	for _, binding := range bindings.Bindings {
		adminJSON, _ := json.Marshal(binding.AdminUserIDs)
		allowedJSON, _ := json.Marshal(binding.AllowedUserIDs)
		for _, repository := range binding.Repositories {
			isDefault := boolToSQLiteInteger(repository == binding.DefaultRepository)
			enabled := boolToSQLiteInteger(binding.Enabled)
			publicRead := boolToSQLiteInteger(binding.AllowPublicRead)
			key := binding.ChatID + "\x00" + binding.InstallationID + "\x00" + repository
			fingerprint := reviewEntityFingerprint(binding.ChatID, binding.InstallationID, repository, isDefault, enabled, publicRead, string(adminJSON), string(allowedJSON))
			revision, createdAt, updatedBy := reviewEntityState(fingerprint, snapshots["binding:"+key], actorHash, appliedAt)
			if _, err := tx.ExecContext(ctx, `INSERT INTO chat_repository_bindings(chat_id,installation_id,repository,is_default,enabled,allow_public_read,admin_user_ids_json,allowed_user_ids_json,updated_at,revision,created_at,updated_by_hash) VALUES(?,?,?,?,?,?,?,?,?,?,?,?)`, binding.ChatID, binding.InstallationID, repository, isDefault, enabled, publicRead, string(adminJSON), string(allowedJSON), appliedAt, revision, createdAt, updatedBy); err != nil {
				return 0, fmt.Errorf("store chat repository binding %q/%q: %w", binding.ChatID, repository, err)
			}
		}
	}
	for _, identity := range bindings.IdentityBindings {
		enabled := boolToSQLiteInteger(identity.Enabled)
		key := identity.InstallationID + "\x00" + identity.FeishuUserID
		fingerprint := reviewEntityFingerprint(identity.InstallationID, identity.FeishuUserID, identity.GitLinkLogin, identity.VerificationMethod, identity.VerifiedAt, enabled)
		revision, createdAt, updatedBy := reviewEntityState(fingerprint, snapshots["identity:"+key], actorHash, appliedAt)
		if _, err := tx.ExecContext(ctx, `INSERT INTO review_identity_bindings(installation_id,feishu_user_id,gitlink_login,verification_method,verified_at,enabled,updated_at,revision,created_at,updated_by_hash) VALUES(?,?,?,?,?,?,?,?,?,?)`, identity.InstallationID, identity.FeishuUserID, identity.GitLinkLogin, identity.VerificationMethod, identity.VerifiedAt, enabled, appliedAt, revision, createdAt, updatedBy); err != nil {
			return 0, fmt.Errorf("store Review identity binding for %q: %w", identity.FeishuUserID, err)
		}
	}
	return repositoryCount, nil
}

// CompareAndSwapReviewEntityRevision is the common optimistic-concurrency gate
// for local entity update commands. Callers perform their value update in the
// same transaction after this succeeds.
func (s *SQLiteReviewGatewayStore) CompareAndSwapReviewEntityRevision(ctx context.Context, table, where string, keyArgs []interface{}, expectedRevision int, actorID string, now time.Time) (int, error) {
	allowed := map[string]bool{"gitlink_installations": true, "installation_repositories": true, "chat_repository_bindings": true, "review_identity_bindings": true}
	allowedWhere := map[string]bool{"installation_id=?": true, "installation_id=? AND repository=?": true, "chat_id=? AND installation_id=? AND repository=?": true, "installation_id=? AND feishu_user_id=?": true}
	if !allowed[table] || !allowedWhere[where] || expectedRevision < 1 {
		return 0, fmt.Errorf("invalid review configuration entity revision request")
	}
	args := []interface{}{reviewConfigurationIdentifierHash(actorID), reviewGatewayTimestamp(now)}
	args = append(args, keyArgs...)
	args = append(args, expectedRevision)
	result, err := s.db.ExecContext(ctx, `UPDATE `+table+` SET revision=revision+1,updated_by_hash=?,updated_at=? WHERE `+where+` AND revision=?`, args...)
	if err != nil {
		return 0, err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return 0, err
	}
	if affected != 1 {
		return 0, ErrReviewConfigurationConflict
	}
	return expectedRevision + 1, nil
}
