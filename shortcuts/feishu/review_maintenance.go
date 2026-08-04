package feishu

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	_ "modernc.org/sqlite"
)

const reviewBackupManifestSchema = "review.backup-manifest/v1"

type ReviewMaintenanceRun struct {
	RunID              string `json:"run_id"`
	MaintenanceKind    string `json:"maintenance_kind"`
	Status             string `json:"status"`
	StartedAt          string `json:"started_at"`
	FinishedAt         string `json:"finished_at,omitempty"`
	SourceDatabaseHash string `json:"source_database_hash,omitempty"`
	ArtifactPathHash   string `json:"artifact_path_hash,omitempty"`
	ArtifactSHA256     string `json:"artifact_sha256,omitempty"`
	RowsAffected       int64  `json:"rows_affected"`
	ErrorSummary       string `json:"error_summary,omitempty"`
}

type ReviewBackupManifest struct {
	SchemaVersion       string `json:"schema_version"`
	CreatedAt           string `json:"created_at"`
	DatabaseSHA256      string `json:"database_sha256"`
	DatabaseSize        int64  `json:"database_size"`
	ReviewSchemaVersion int    `json:"review_schema_version"`
	ConfigRevision      int    `json:"config_revision"`
	ServiceVersion      string `json:"service_version,omitempty"`
	CommitSHA           string `json:"commit_sha,omitempty"`
	SourcePathHash      string `json:"source_path_hash"`
}

type ReviewBackupOptions struct {
	OutputDirectory string
	Name            string
	Verify          bool
	RetentionCount  int
	ServiceVersion  string
	CommitSHA       string
	Now             time.Time
}

type ReviewBackupResult struct {
	DatabasePath string               `json:"database_path"`
	ManifestPath string               `json:"manifest_path"`
	Manifest     ReviewBackupManifest `json:"manifest"`
}

type ReviewRestoreOptions struct {
	StateDB    string
	Backup     string
	Manifest   string
	VerifyOnly bool
	Replace    bool
	Confirmed  bool
	Now        time.Time
}

type ReviewRetentionOptions struct {
	DryRun    bool
	BatchSize int
	Now       time.Time
}

type ReviewRetentionResult struct {
	DryRun       bool             `json:"dry_run"`
	RowsAffected int64            `json:"rows_affected"`
	ByTable      map[string]int64 `json:"by_table"`
}

type ReviewWALCheckpointResult struct {
	Mode           string `json:"mode"`
	Busy           bool   `json:"busy"`
	LogFrames      int    `json:"log_frames"`
	Checkpointed   int    `json:"checkpointed_frames"`
	MaintenanceRun string `json:"maintenance_run"`
}

var reviewMaintenanceLocks sync.Map
var reviewBackupNamePattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{0,96}$`)

func reviewMaintenanceLock(path, kind string) *sync.Mutex {
	key := reviewGatewayHashIdentifier(path) + ":" + kind
	value, _ := reviewMaintenanceLocks.LoadOrStore(key, &sync.Mutex{})
	return value.(*sync.Mutex)
}

func (s *SQLiteReviewGatewayStore) startReviewMaintenanceRun(ctx context.Context, kind string, now time.Time) (ReviewMaintenanceRun, error) {
	run := ReviewMaintenanceRun{RunID: stableKey("review-maintenance", kind, fmt.Sprintf("%d", now.UnixNano())), MaintenanceKind: kind, Status: "running", StartedAt: reviewGatewayTimestamp(now), SourceDatabaseHash: reviewGatewayHashIdentifier(s.path)}
	_, err := s.db.ExecContext(ctx, `INSERT INTO review_maintenance_runs(run_id,maintenance_kind,status,started_at,source_database_hash) VALUES(?,?,?,?,?)`, run.RunID, run.MaintenanceKind, run.Status, run.StartedAt, run.SourceDatabaseHash)
	return run, err
}

func (s *SQLiteReviewGatewayStore) finishReviewMaintenanceRun(ctx context.Context, run ReviewMaintenanceRun, status string, rows int64, artifactPath, artifactSHA string, runErr error, now time.Time) error {
	errorSummary := ""
	if runErr != nil {
		errorSummary = redactReviewMaintenanceError(runErr.Error())
	}
	_, err := s.db.ExecContext(ctx, `UPDATE review_maintenance_runs SET status=?,finished_at=?,artifact_path_hash=?,artifact_sha256=?,rows_affected=?,error_summary=? WHERE run_id=?`, status, reviewGatewayTimestamp(now), reviewGatewayHashIdentifier(artifactPath), artifactSHA, rows, errorSummary, run.RunID)
	return err
}

func redactReviewMaintenanceError(message string) string {
	message = redactReviewGatewayError(message)
	for _, part := range strings.Fields(message) {
		if filepath.IsAbs(strings.Trim(part, `"':,;()`)) {
			message = strings.ReplaceAll(message, part, "[path]")
		}
	}
	return message
}

func (s *SQLiteReviewGatewayStore) BackupReviewSQLite(ctx context.Context, options ReviewBackupOptions) (result ReviewBackupResult, err error) {
	if s == nil || s.db == nil {
		return result, fmt.Errorf("review backup store is required")
	}
	lock := reviewMaintenanceLock(s.path, "backup")
	if !lock.TryLock() {
		return result, fmt.Errorf("review backup is already running")
	}
	defer lock.Unlock()
	if options.Now.IsZero() {
		options.Now = time.Now().UTC()
	}
	if options.RetentionCount <= 0 {
		options.RetentionCount = 7
	}
	output, err := filepath.Abs(strings.TrimSpace(options.OutputDirectory))
	if err != nil || strings.TrimSpace(options.OutputDirectory) == "" {
		return result, fmt.Errorf("review backup output directory is required")
	}
	if err := os.MkdirAll(output, 0o700); err != nil {
		return result, fmt.Errorf("create review backup directory: %w", err)
	}
	name := strings.TrimSpace(options.Name)
	if name == "" {
		name = "review-" + options.Now.UTC().Format("20060102T150405Z")
	}
	if !reviewBackupNamePattern.MatchString(name) {
		return result, fmt.Errorf("review backup name contains unsupported characters")
	}
	run, startErr := s.startReviewMaintenanceRun(ctx, "backup", options.Now)
	if startErr != nil {
		return result, startErr
	}
	status := "failed"
	defer func() {
		finishCtx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		if err == nil {
			status = "succeeded"
		}
		_ = s.finishReviewMaintenanceRun(finishCtx, run, status, 0, result.DatabasePath, result.Manifest.DatabaseSHA256, err, time.Now().UTC())
	}()

	finalDB := filepath.Join(output, name+".db")
	finalManifest := finalDB + ".manifest.json"
	tempDB := finalDB + ".tmp-" + fmt.Sprintf("%d", options.Now.UnixNano())
	tempManifest := finalManifest + ".tmp-" + fmt.Sprintf("%d", options.Now.UnixNano())
	defer func() { _ = os.Remove(tempDB); _ = os.Remove(tempManifest) }()
	if _, statErr := os.Stat(finalDB); statErr == nil {
		return result, fmt.Errorf("review backup artifact already exists")
	}
	if _, err = s.db.ExecContext(ctx, `VACUUM INTO ?`, tempDB); err != nil {
		return result, fmt.Errorf("create consistent review SQLite backup: %w", err)
	}
	if options.Verify {
		if err = verifyReviewSQLiteIntegrity(ctx, tempDB, true); err != nil {
			return result, fmt.Errorf("verify review backup quick check: %w", err)
		}
	}
	digest, size, err := reviewFileSHA256(tempDB)
	if err != nil {
		return result, err
	}
	schema, err := reviewSQLiteSchemaVersion(ctx, tempDB)
	if err != nil {
		return result, err
	}
	state, stateErr := s.CurrentReviewConfigurationState(ctx)
	if stateErr != nil {
		return result, fmt.Errorf("read review configuration state for backup: %w", stateErr)
	}
	result.Manifest = ReviewBackupManifest{SchemaVersion: reviewBackupManifestSchema, CreatedAt: reviewGatewayTimestamp(options.Now), DatabaseSHA256: digest, DatabaseSize: size, ReviewSchemaVersion: schema, ConfigRevision: state.ConfigRevision, ServiceVersion: strings.TrimSpace(options.ServiceVersion), CommitSHA: strings.TrimSpace(options.CommitSHA), SourcePathHash: reviewGatewayHashIdentifier(s.path)}
	manifestBytes, err := json.MarshalIndent(result.Manifest, "", "  ")
	if err != nil {
		return result, err
	}
	manifestBytes = append(manifestBytes, '\n')
	if err = writeReviewFileSynced(tempManifest, manifestBytes, 0o600); err != nil {
		return result, err
	}
	if err = syncReviewFile(tempDB); err != nil {
		return result, err
	}
	if err = os.Rename(tempDB, finalDB); err != nil {
		return result, fmt.Errorf("publish review backup database: %w", err)
	}
	if err = os.Rename(tempManifest, finalManifest); err != nil {
		_ = os.Remove(finalDB)
		return result, fmt.Errorf("publish review backup manifest: %w", err)
	}
	result.DatabasePath, result.ManifestPath = finalDB, finalManifest
	if err = retainReviewBackups(output, options.RetentionCount, finalDB); err != nil {
		return result, err
	}
	return result, nil
}

func retainReviewBackups(directory string, keep int, current string) error {
	entries, err := filepath.Glob(filepath.Join(directory, "*.db.manifest.json"))
	if err != nil {
		return err
	}
	type artifact struct {
		db, manifest string
		created      time.Time
	}
	items := []artifact{}
	for _, manifestPath := range entries {
		payload, readErr := os.ReadFile(manifestPath)
		if readErr != nil {
			continue
		}
		var manifest ReviewBackupManifest
		if json.Unmarshal(payload, &manifest) != nil || manifest.SchemaVersion != reviewBackupManifestSchema {
			continue
		}
		dbPath := strings.TrimSuffix(manifestPath, ".manifest.json")
		if _, statErr := os.Stat(dbPath); statErr != nil {
			continue
		}
		created, _ := time.Parse(time.RFC3339Nano, manifest.CreatedAt)
		items = append(items, artifact{db: dbPath, manifest: manifestPath, created: created})
	}
	sort.Slice(items, func(i, j int) bool { return items[i].created.After(items[j].created) })
	for index := keep; index < len(items); index++ {
		if sameReviewPath(items[index].db, current) {
			continue
		}
		if err := os.Remove(items[index].db); err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
		if err := os.Remove(items[index].manifest); err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
	}
	return nil
}

func VerifyReviewBackup(ctx context.Context, backupPath, manifestPath string) (ReviewBackupManifest, error) {
	var manifest ReviewBackupManifest
	payload, err := os.ReadFile(manifestPath)
	if err != nil {
		return manifest, fmt.Errorf("read review backup manifest: %w", err)
	}
	if err := json.Unmarshal(payload, &manifest); err != nil || manifest.SchemaVersion != reviewBackupManifestSchema {
		return manifest, fmt.Errorf("unsupported review backup manifest")
	}
	digest, size, err := reviewFileSHA256(backupPath)
	if err != nil {
		return manifest, err
	}
	if digest != manifest.DatabaseSHA256 || size != manifest.DatabaseSize {
		return manifest, fmt.Errorf("review backup checksum mismatch")
	}
	header := make([]byte, 16)
	file, err := os.Open(backupPath)
	if err != nil {
		return manifest, err
	}
	_, readErr := io.ReadFull(file, header)
	_ = file.Close()
	if readErr != nil || string(header) != "SQLite format 3\x00" {
		return manifest, fmt.Errorf("review backup has invalid SQLite header")
	}
	if err := verifyReviewSQLiteIntegrity(ctx, backupPath, false); err != nil {
		return manifest, err
	}
	schema, err := reviewSQLiteSchemaVersion(ctx, backupPath)
	if err != nil {
		return manifest, err
	}
	if schema != manifest.ReviewSchemaVersion || schema > latestReviewGatewaySchemaVersion() {
		return manifest, fmt.Errorf("review backup schema version is unsupported")
	}
	db, err := sql.Open("sqlite", "file:"+filepath.ToSlash(backupPath)+"?mode=ro")
	if err != nil {
		return manifest, err
	}
	defer db.Close()
	for _, table := range []string{"schema_migrations", "review_gateway_jobs", "review_operations"} {
		var count int
		if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name=?`, table).Scan(&count); err != nil || count != 1 {
			return manifest, fmt.Errorf("review backup is missing required table %s", table)
		}
	}
	if schema >= 17 {
		if _, err := readReviewConfigurationState(ctx, db); err != nil {
			return manifest, fmt.Errorf("read review backup configuration state: %w", err)
		}
	}
	return manifest, nil
}

func RestoreReviewSQLite(ctx context.Context, options ReviewRestoreOptions) (err error) {
	if options.Now.IsZero() {
		options.Now = time.Now().UTC()
	}
	statePath, err := filepath.Abs(strings.TrimSpace(options.StateDB))
	if err != nil || strings.TrimSpace(options.StateDB) == "" {
		return fmt.Errorf("review restore state database is required")
	}
	manifest, err := VerifyReviewBackup(ctx, options.Backup, options.Manifest)
	if err != nil {
		return err
	}
	if options.VerifyOnly {
		return nil
	}
	if !options.Replace || !options.Confirmed {
		return fmt.Errorf("review restore replacement requires --replace and explicit confirmation")
	}
	if _, lockErr := readReviewGatewayInstanceMetadata(statePath + ".lock"); lockErr == nil {
		return fmt.Errorf("review restore rejected because the service instance lock is active")
	}
	lock := reviewMaintenanceLock(statePath, "restore")
	if !lock.TryLock() {
		return fmt.Errorf("review restore is already running")
	}
	defer lock.Unlock()
	if sameReviewPath(statePath, options.Backup) {
		return fmt.Errorf("review restore backup and target must be different files")
	}
	if err := os.MkdirAll(filepath.Dir(statePath), 0o700); err != nil {
		return err
	}
	preRestoreDirectory := filepath.Join(filepath.Dir(statePath), "pre-restore-backups")
	if _, statErr := os.Stat(statePath); statErr == nil {
		current, openErr := OpenSQLiteReviewGatewayStore(statePath)
		if openErr != nil {
			return fmt.Errorf("open current review database before restore: %w", openErr)
		}
		_, backupErr := current.BackupReviewSQLite(ctx, ReviewBackupOptions{OutputDirectory: preRestoreDirectory, Name: "pre-restore-" + options.Now.UTC().Format("20060102T150405.000000000Z"), Verify: true, RetentionCount: 7, Now: options.Now})
		_ = current.Close()
		if backupErr != nil {
			return fmt.Errorf("create pre-restore review backup: %w", backupErr)
		}
	}
	temp := statePath + ".restore-tmp"
	rollback := statePath + ".restore-original"
	_ = os.Remove(temp)
	_ = os.Remove(rollback)
	if err := copyReviewFileSynced(options.Backup, temp); err != nil {
		return err
	}
	if err := verifyReviewSQLiteIntegrity(ctx, temp, false); err != nil {
		_ = os.Remove(temp)
		return err
	}
	hadOriginal := false
	if _, statErr := os.Stat(statePath); statErr == nil {
		if err := os.Rename(statePath, rollback); err != nil {
			_ = os.Remove(temp)
			return fmt.Errorf("stage current review database for restore: %w", err)
		}
		hadOriginal = true
	}
	restoreOriginal := func() {
		_ = os.Remove(statePath)
		if hadOriginal {
			_ = os.Rename(rollback, statePath)
		}
	}
	if err := os.Rename(temp, statePath); err != nil {
		restoreOriginal()
		return fmt.Errorf("replace review database: %w", err)
	}
	_ = os.Remove(statePath + "-wal")
	_ = os.Remove(statePath + "-shm")
	restored, openErr := OpenSQLiteReviewGatewayStore(statePath)
	if openErr != nil {
		restoreOriginal()
		return fmt.Errorf("open restored review database: %w", openErr)
	}
	if integrityErr := verifyReviewSQLiteDB(ctx, restored.db, false); integrityErr != nil {
		_ = restored.Close()
		restoreOriginal()
		return integrityErr
	}
	run, runErr := restored.startReviewMaintenanceRun(ctx, "restore_verify", options.Now)
	if runErr == nil {
		runErr = restored.finishReviewMaintenanceRun(ctx, run, "succeeded", 0, options.Backup, manifest.DatabaseSHA256, nil, time.Now().UTC())
	}
	_ = restored.Close()
	if runErr != nil {
		restoreOriginal()
		return fmt.Errorf("record review restore verification: %w", runErr)
	}
	_ = os.Remove(rollback)
	return nil
}

func (s *SQLiteReviewGatewayStore) RunReviewRetention(ctx context.Context, options ReviewRetentionOptions) (result ReviewRetentionResult, err error) {
	result = ReviewRetentionResult{DryRun: options.DryRun, ByTable: map[string]int64{}}
	if options.Now.IsZero() {
		options.Now = time.Now().UTC()
	}
	if options.BatchSize <= 0 || options.BatchSize > 1000 {
		options.BatchSize = 1000
	}
	lock := reviewMaintenanceLock(s.path, "retention")
	if !lock.TryLock() {
		return result, fmt.Errorf("review retention is already running")
	}
	defer lock.Unlock()
	run, err := s.startReviewMaintenanceRun(ctx, "retention", options.Now)
	if err != nil {
		return result, err
	}
	status := "failed"
	defer func() {
		if err == nil {
			status = "succeeded"
		}
		_ = s.finishReviewMaintenanceRun(context.Background(), run, status, result.RowsAffected, "", "", err, time.Now().UTC())
	}()
	cutoff30 := reviewGatewayTimestamp(options.Now.Add(-30 * 24 * time.Hour))
	cutoff2 := reviewGatewayTimestamp(options.Now.Add(-2 * 24 * time.Hour))
	statements := []struct {
		table, count, remove string
		args                 []interface{}
	}{
		{"review_operation_attempts", `SELECT COUNT(*) FROM review_operation_attempts a JOIN review_operations o ON o.operation_id=a.operation_id WHERE a.finished_at<>'' AND a.finished_at<? AND o.status IN ('succeeded','unchanged','stale','failed_terminal','dead_letter','cancelled')`, `DELETE FROM review_operation_attempts WHERE rowid IN (SELECT a.rowid FROM review_operation_attempts a JOIN review_operations o ON o.operation_id=a.operation_id WHERE a.finished_at<>'' AND a.finished_at<? AND o.status IN ('succeeded','unchanged','stale','failed_terminal','dead_letter','cancelled') LIMIT ?)`, []interface{}{cutoff30}},
		{"review_operations", `SELECT COUNT(*) FROM review_operations WHERE completed_at<>'' AND completed_at<? AND status IN ('succeeded','unchanged','stale','failed_terminal','dead_letter','cancelled')`, `DELETE FROM review_operations WHERE rowid IN (SELECT rowid FROM review_operations WHERE completed_at<>'' AND completed_at<? AND status IN ('succeeded','unchanged','stale','failed_terminal','dead_letter','cancelled') LIMIT ?)`, []interface{}{cutoff30}},
		{"review_job_consumers", `SELECT COUNT(*) FROM review_job_consumers WHERE updated_at<? AND status IN ('sent','failed')`, `DELETE FROM review_job_consumers WHERE rowid IN (SELECT rowid FROM review_job_consumers WHERE updated_at<? AND status IN ('sent','failed') LIMIT ?)`, []interface{}{cutoff30}},
		{"review_gateway_jobs", `SELECT COUNT(*) FROM review_gateway_jobs WHERE updated_at<? AND status IN ('completed','failed')`, `DELETE FROM review_gateway_jobs WHERE rowid IN (SELECT rowid FROM review_gateway_jobs WHERE updated_at<? AND status IN ('completed','failed') LIMIT ?)`, []interface{}{cutoff30}},
		{"review_rate_limit_buckets", `SELECT COUNT(*) FROM review_rate_limit_buckets WHERE updated_at<?`, `DELETE FROM review_rate_limit_buckets WHERE rowid IN (SELECT rowid FROM review_rate_limit_buckets WHERE updated_at<? LIMIT ?)`, []interface{}{cutoff2}},
		{"review_dead_letters", `SELECT COUNT(*) FROM review_dead_letters WHERE updated_at<? AND status='resolved'`, `DELETE FROM review_dead_letters WHERE rowid IN (SELECT rowid FROM review_dead_letters WHERE updated_at<? AND status='resolved' LIMIT ?)`, []interface{}{cutoff30}},
		{"review_operation_reconciliation_tasks", `SELECT COUNT(*) FROM review_operation_reconciliation_tasks WHERE resolved_at<>'' AND resolved_at<? AND status='resolved'`, `DELETE FROM review_operation_reconciliation_tasks WHERE rowid IN (SELECT rowid FROM review_operation_reconciliation_tasks WHERE resolved_at<>'' AND resolved_at<? AND status='resolved' LIMIT ?)`, []interface{}{cutoff30}},
	}
	if options.DryRun {
		for _, statement := range statements {
			var count int64
			if err := s.db.QueryRowContext(ctx, statement.count, statement.args...).Scan(&count); err != nil {
				return result, err
			}
			if count > int64(options.BatchSize) {
				count = int64(options.BatchSize)
			}
			result.ByTable[statement.table], result.RowsAffected = count, result.RowsAffected+count
		}
		return result, nil
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return result, err
	}
	defer tx.Rollback()
	for _, statement := range statements {
		args := append(append([]interface{}{}, statement.args...), options.BatchSize)
		res, executeErr := tx.ExecContext(ctx, statement.remove, args...)
		if executeErr != nil {
			return result, executeErr
		}
		count, _ := res.RowsAffected()
		result.ByTable[statement.table], result.RowsAffected = count, result.RowsAffected+count
	}
	if err := tx.Commit(); err != nil {
		return result, err
	}
	return result, nil
}

func (s *SQLiteReviewGatewayStore) WALCheckpoint(ctx context.Context, mode string, metrics *ReviewMetrics, now time.Time) (result ReviewWALCheckpointResult, err error) {
	mode = strings.ToUpper(strings.TrimSpace(mode))
	if mode != "PASSIVE" && mode != "FULL" && mode != "TRUNCATE" {
		return result, fmt.Errorf("unsupported review WAL checkpoint mode")
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}
	lock := reviewMaintenanceLock(s.path, "wal_checkpoint")
	if !lock.TryLock() {
		return result, fmt.Errorf("review WAL checkpoint is already running")
	}
	defer lock.Unlock()
	run, err := s.startReviewMaintenanceRun(ctx, "wal_checkpoint", now)
	if err != nil {
		return result, err
	}
	result.Mode, result.MaintenanceRun = mode, run.RunID
	status := "failed"
	defer func() {
		if err == nil {
			status = "succeeded"
			if metrics != nil {
				metrics.walSuccess.Store(time.Now().Unix())
			}
		}
		_ = s.finishReviewMaintenanceRun(context.Background(), run, status, 0, "", "", err, time.Now().UTC())
	}()
	var busy int
	if err = s.db.QueryRowContext(ctx, `PRAGMA wal_checkpoint(`+mode+`)`).Scan(&busy, &result.LogFrames, &result.Checkpointed); err != nil {
		return result, err
	}
	result.Busy = busy != 0
	return result, nil
}

func (s *SQLiteReviewGatewayStore) IntegrityCheck(ctx context.Context, quick bool, now time.Time) (err error) {
	if now.IsZero() {
		now = time.Now().UTC()
	}
	run, err := s.startReviewMaintenanceRun(ctx, "integrity_check", now)
	if err != nil {
		return err
	}
	status := "failed"
	defer func() {
		if err == nil {
			status = "succeeded"
		}
		_ = s.finishReviewMaintenanceRun(context.Background(), run, status, 0, "", "", err, time.Now().UTC())
	}()
	return verifyReviewSQLiteDB(ctx, s.db, quick)
}

func verifyReviewSQLiteIntegrity(ctx context.Context, path string, quick bool) error {
	db, err := sql.Open("sqlite", "file:"+filepath.ToSlash(path)+"?mode=ro")
	if err != nil {
		return err
	}
	defer db.Close()
	return verifyReviewSQLiteDB(ctx, db, quick)
}

func verifyReviewSQLiteDB(ctx context.Context, db *sql.DB, quick bool) error {
	pragma := "integrity_check"
	if quick {
		pragma = "quick_check"
	}
	rows, err := db.QueryContext(ctx, `PRAGMA `+pragma)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var result string
		if err := rows.Scan(&result); err != nil {
			return err
		}
		if !strings.EqualFold(strings.TrimSpace(result), "ok") {
			return fmt.Errorf("review SQLite %s failed", pragma)
		}
	}
	return rows.Err()
}

func reviewSQLiteSchemaVersion(ctx context.Context, path string) (int, error) {
	db, err := sql.Open("sqlite", "file:"+filepath.ToSlash(path)+"?mode=ro")
	if err != nil {
		return 0, err
	}
	defer db.Close()
	var version sql.NullInt64
	if err := db.QueryRowContext(ctx, `SELECT MAX(version) FROM schema_migrations`).Scan(&version); err != nil {
		return 0, err
	}
	return int(version.Int64), nil
}

func reviewFileSHA256(path string) (string, int64, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", 0, err
	}
	defer file.Close()
	hash := sha256.New()
	size, err := io.Copy(hash, file)
	if err != nil {
		return "", 0, err
	}
	return hex.EncodeToString(hash.Sum(nil)), size, nil
}

func writeReviewFileSynced(path string, payload []byte, mode os.FileMode) error {
	file, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, mode)
	if err != nil {
		return err
	}
	if _, err := file.Write(payload); err != nil {
		_ = file.Close()
		return err
	}
	if err := file.Sync(); err != nil {
		_ = file.Close()
		return err
	}
	return file.Close()
}

func syncReviewFile(path string) error {
	file, err := os.OpenFile(path, os.O_RDWR, 0)
	if err != nil {
		return err
	}
	defer file.Close()
	return file.Sync()
}

func copyReviewFileSynced(source, destination string) error {
	input, err := os.Open(source)
	if err != nil {
		return err
	}
	defer input.Close()
	output, err := os.OpenFile(destination, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	if _, err := io.Copy(output, input); err != nil {
		_ = output.Close()
		return err
	}
	if err := output.Sync(); err != nil {
		_ = output.Close()
		return err
	}
	return output.Close()
}

func sameReviewPath(left, right string) bool {
	leftAbs, _ := filepath.Abs(left)
	rightAbs, _ := filepath.Abs(right)
	return strings.EqualFold(filepath.Clean(leftAbs), filepath.Clean(rightAbs))
}

type ReviewMaintenanceScheduler struct {
	Store             *SQLiteReviewGatewayStore
	Metrics           *ReviewMetrics
	BackupOptions     ReviewBackupOptions
	BackupInterval    time.Duration
	RetentionInterval time.Duration
	WALInterval       time.Duration
	IntegrityInterval time.Duration
	Now               func() time.Time

	locks sync.Map
}

func (m *ReviewMaintenanceScheduler) Run(ctx context.Context) {
	if m == nil || m.Store == nil {
		return
	}
	if m.Now == nil {
		m.Now = time.Now
	}
	type scheduled struct {
		kind     string
		interval time.Duration
	}
	entries := []scheduled{{"backup", m.BackupInterval}, {"retention", m.RetentionInterval}, {"wal_checkpoint", m.WALInterval}, {"integrity_check", m.IntegrityInterval}}
	var wg sync.WaitGroup
	for _, entry := range entries {
		if entry.interval <= 0 {
			continue
		}
		entry := entry
		wg.Add(1)
		go func() {
			defer wg.Done()
			ticker := time.NewTicker(entry.interval)
			defer ticker.Stop()
			for {
				select {
				case <-ctx.Done():
					return
				case <-ticker.C:
					go func() { _ = m.RunKind(ctx, entry.kind) }()
				}
			}
		}()
	}
	<-ctx.Done()
	wg.Wait()
}

func (m *ReviewMaintenanceScheduler) RunKind(ctx context.Context, kind string) error {
	value, _ := m.locks.LoadOrStore(kind, &sync.Mutex{})
	lock := value.(*sync.Mutex)
	if !lock.TryLock() {
		return fmt.Errorf("review maintenance %s is already running", kind)
	}
	defer lock.Unlock()
	now := time.Now().UTC()
	if m.Now != nil {
		now = m.Now().UTC()
	}
	switch kind {
	case "backup":
		options := m.BackupOptions
		options.Now = now
		_, err := m.Store.BackupReviewSQLite(ctx, options)
		return err
	case "retention":
		_, err := m.Store.RunReviewRetention(ctx, ReviewRetentionOptions{DryRun: false, BatchSize: 1000, Now: now})
		return err
	case "wal_checkpoint":
		_, err := m.Store.WALCheckpoint(ctx, "PASSIVE", m.Metrics, now)
		return err
	case "integrity_check":
		return m.Store.IntegrityCheck(ctx, true, now)
	default:
		return fmt.Errorf("unsupported review maintenance kind")
	}
}
