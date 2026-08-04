package feishu

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gitlink-org/gitlink-cli/shortcuts/common"
)

func newReviewServiceShortcut() *common.Shortcut {
	return &common.Shortcut{
		Name: "review-service", Description: "Inspect and maintain the local GitLink Feishu Review service",
		Flags: []common.Flag{
			{Name: "action", Usage: "status, doctor, backup, restore, retention, wal-checkpoint, or integrity-check", Required: true},
			{Name: "state-db", Usage: "SQLite Review service database", Default: ".local/review-gateway.db"},
			{Name: "output-directory", Usage: "Backup output directory"}, {Name: "name", Usage: "Backup artifact name"},
			{Name: "backup", Usage: "Restore backup database"}, {Name: "manifest", Usage: "Restore backup manifest"},
			{Name: "verify", Usage: "Verify a backup after creation", Bool: true, Default: "false"},
			{Name: "verify-only", Usage: "Verify restore inputs without changing the target", Bool: true, Default: "false"},
			{Name: "replace", Usage: "Replace the stopped target database", Bool: true, Default: "false"},
			{Name: "apply", Usage: "Apply retention; dry-run is the default", Bool: true, Default: "false"},
			{Name: "yes", Usage: "Explicitly confirm destructive maintenance", Bool: true, Default: "false"},
			{Name: "checkpoint-mode", Usage: "PASSIVE, FULL, or TRUNCATE", Default: "PASSIVE"},
		}, Run: runReviewServiceAdmin,
	}
}

func newReviewConfigurationShortcut() *common.Shortcut {
	return &common.Shortcut{Name: "review-config", Description: "Inspect versioned local Review configuration", Flags: []common.Flag{
		{Name: "action", Usage: "status, revisions, or inspect", Required: true},
		{Name: "state-db", Usage: "SQLite Review service database", Default: ".local/review-gateway.db"},
		{Name: "revision", Usage: "Configuration revision to inspect"},
	}, Run: runReviewConfigurationAdmin}
}

func runReviewServiceAdmin(runtime *common.RuntimeContext) error {
	action := strings.ToLower(strings.TrimSpace(runtime.Arg("action")))
	stateDB := firstNonEmpty(runtime.Arg("state-db"), ".local/review-gateway.db")
	if action == "restore" {
		return runReviewServiceRestore(runtime, stateDB)
	}
	store, err := OpenSQLiteReviewGatewayStore(stateDB)
	if err != nil {
		return err
	}
	defer store.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	switch action {
	case "status":
		return emitReviewServiceJSON(reviewServiceStatusOutput(ctx, store))
	case "doctor":
		return emitReviewServiceJSON(reviewServiceDoctorOutput(ctx, store))
	case "backup":
		result, err := store.BackupReviewSQLite(ctx, ReviewBackupOptions{OutputDirectory: runtime.Arg("output-directory"), Name: runtime.Arg("name"), Verify: parseBool(runtime.Arg("verify")), RetentionCount: 7, Now: time.Now().UTC()})
		if err != nil {
			return err
		}
		return emitReviewServiceJSON(map[string]interface{}{"schema_version": "review.service-maintenance/v1", "action": action, "database_path_hash": reviewGatewayHashIdentifier(result.DatabasePath), "manifest_path_hash": reviewGatewayHashIdentifier(result.ManifestPath), "manifest": result.Manifest})
	case "retention":
		apply := parseBool(runtime.Arg("apply"))
		if apply && !parseBool(runtime.Arg("yes")) {
			return fmt.Errorf("applying review retention requires --yes")
		}
		result, err := store.RunReviewRetention(ctx, ReviewRetentionOptions{DryRun: !apply, BatchSize: 1000, Now: time.Now().UTC()})
		if err != nil {
			return err
		}
		return emitReviewServiceJSON(result)
	case "wal-checkpoint":
		result, err := store.WALCheckpoint(ctx, runtime.Arg("checkpoint-mode"), nil, time.Now().UTC())
		if err != nil {
			return err
		}
		return emitReviewServiceJSON(result)
	case "integrity-check":
		if err := store.IntegrityCheck(ctx, false, time.Now().UTC()); err != nil {
			return err
		}
		return emitReviewServiceJSON(map[string]interface{}{"schema_version": "review.integrity-check/v1", "status": "ok"})
	default:
		return fmt.Errorf("unsupported review service action %q", action)
	}
}

func runReviewServiceRestore(runtime *common.RuntimeContext, stateDB string) error {
	verifyOnly := parseBool(runtime.Arg("verify-only"))
	if !verifyOnly && (!parseBool(runtime.Arg("replace")) || !parseBool(runtime.Arg("yes"))) {
		return fmt.Errorf("review restore requires --replace and --yes; use --verify-only for validation")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	if err := RestoreReviewSQLite(ctx, ReviewRestoreOptions{StateDB: stateDB, Backup: runtime.Arg("backup"), Manifest: runtime.Arg("manifest"), VerifyOnly: verifyOnly, Replace: parseBool(runtime.Arg("replace")), Confirmed: parseBool(runtime.Arg("yes")), Now: time.Now().UTC()}); err != nil {
		return err
	}
	return emitReviewServiceJSON(map[string]interface{}{"schema_version": "review.restore/v1", "status": "verified", "replaced": !verifyOnly})
}

func reviewServiceStatusOutput(ctx context.Context, store *SQLiteReviewGatewayStore) map[string]interface{} {
	queue, _ := store.ReviewQueueStatus(ctx)
	schema, _ := store.CurrentSchemaVersion(ctx)
	configuration, _ := store.CurrentReviewConfigurationState(ctx)
	var latest ReviewServiceInstance
	_ = store.db.QueryRowContext(ctx, `SELECT instance_id,process_id,hostname_hash,service_version,commit_sha,schema_version,
		config_revision,config_fingerprint,status,admission_status,started_at,ready_at,heartbeat_at,shutdown_started_at,stopped_at,last_error_summary
		FROM review_service_instances ORDER BY heartbeat_at DESC LIMIT 1`).Scan(&latest.InstanceID, &latest.ProcessID, &latest.HostnameHash, &latest.ServiceVersion, &latest.CommitSHA, &latest.SchemaVersion, &latest.ConfigRevision, &latest.ConfigFingerprint, &latest.Status, &latest.AdmissionStatus, &latest.StartedAt, &latest.ReadyAt, &latest.HeartbeatAt, &latest.ShutdownStartedAt, &latest.StoppedAt, &latest.LastErrorSummary)
	return map[string]interface{}{"schema_version": "review.service-status/v1", "read_only": true, "database_hash": reviewGatewayHashIdentifier(store.path), "schema_revision": schema, "configuration": configuration, "latest_instance": latest, "queue": queue}
}

func reviewServiceDoctorOutput(ctx context.Context, store *SQLiteReviewGatewayStore) map[string]interface{} {
	checks := map[string]string{}
	if err := store.Ping(ctx); err == nil {
		checks["sqlite_ping"] = "ok"
	} else {
		checks["sqlite_ping"] = "failed"
	}
	version, versionErr := store.CurrentSchemaVersion(ctx)
	if versionErr == nil && version == latestReviewGatewaySchemaVersion() {
		checks["schema"] = "ok"
	} else {
		checks["schema"] = "failed"
	}
	if err := verifyReviewSQLiteDB(ctx, store.db, true); err == nil {
		checks["quick_check"] = "ok"
	} else {
		checks["quick_check"] = "failed"
	}
	state, err := store.CurrentReviewConfigurationState(ctx)
	if err == nil && (state.ConfigRevision == 0 || state.ConfigFingerprint != "") {
		checks["configuration"] = "ok"
	} else {
		checks["configuration"] = "failed"
	}
	lockStatus := "absent"
	if _, err := readReviewGatewayInstanceMetadata(store.path + ".lock"); err == nil {
		lockStatus = "active"
	}
	checks["instance_lock"] = lockStatus
	walSize := int64(0)
	if info, err := os.Stat(store.path + "-wal"); err == nil {
		walSize = info.Size()
	}
	return map[string]interface{}{"schema_version": "review.service-doctor/v1", "read_only": true, "checks": checks, "wal_size_bytes": walSize}
}

func runReviewConfigurationAdmin(runtime *common.RuntimeContext) error {
	store, err := OpenSQLiteReviewGatewayStore(firstNonEmpty(runtime.Arg("state-db"), ".local/review-gateway.db"))
	if err != nil {
		return err
	}
	defer store.Close()
	ctx := context.Background()
	switch strings.ToLower(strings.TrimSpace(runtime.Arg("action"))) {
	case "status":
		state, err := store.CurrentReviewConfigurationState(ctx)
		if err != nil {
			return err
		}
		return emitReviewServiceJSON(state)
	case "revisions":
		return emitReviewServiceJSON(queryReviewConfigurationRevisions(ctx, store, 0))
	case "inspect":
		revision, err := strconv.Atoi(strings.TrimSpace(runtime.Arg("revision")))
		if err != nil || revision < 1 {
			return fmt.Errorf("review config inspect requires a positive --revision")
		}
		return emitReviewServiceJSON(queryReviewConfigurationRevisions(ctx, store, revision))
	default:
		return fmt.Errorf("unsupported review config action")
	}
}

func queryReviewConfigurationRevisions(ctx context.Context, store *SQLiteReviewGatewayStore, revision int) map[string]interface{} {
	query := `SELECT config_revision,config_fingerprint,source,source_hash,installation_count,repository_count,binding_count,
		subscription_count,identity_binding_count,resource_policy_count,worker_config_json,limit_config_json,service_config_json,applied_by_hash,applied_at
		FROM review_configuration_revisions`
	args := []interface{}{}
	if revision > 0 {
		query += ` WHERE config_revision=?`
		args = append(args, revision)
	}
	query += ` ORDER BY config_revision DESC LIMIT 200`
	rows, err := store.db.QueryContext(ctx, query, args...)
	items := []map[string]interface{}{}
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var number, installations, repositories, bindings, subscriptions, identities, policies int
			var fingerprint, source, sourceHash, workerJSON, limitJSON, serviceJSON, actorHash, appliedAt string
			if rows.Scan(&number, &fingerprint, &source, &sourceHash, &installations, &repositories, &bindings, &subscriptions, &identities, &policies, &workerJSON, &limitJSON, &serviceJSON, &actorHash, &appliedAt) == nil {
				items = append(items, map[string]interface{}{"config_revision": number, "config_fingerprint": fingerprint, "source": source, "source_hash": sourceHash, "installation_count": installations, "repository_count": repositories, "binding_count": bindings, "subscription_count": subscriptions, "identity_binding_count": identities, "resource_policy_count": policies, "worker_config_json": json.RawMessage(workerJSON), "limit_config_json": json.RawMessage(limitJSON), "service_config_json": json.RawMessage(serviceJSON), "applied_by_hash": actorHash, "applied_at": appliedAt})
			}
		}
	}
	return map[string]interface{}{"schema_version": reviewAdminListSchema, "read_only": true, "items": items}
}

func emitReviewServiceJSON(value interface{}) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetEscapeHTML(false)
	return encoder.Encode(value)
}

func reviewServicePathHash(path string) string {
	absolute, _ := filepath.Abs(path)
	return reviewGatewayHashIdentifier(absolute)
}
