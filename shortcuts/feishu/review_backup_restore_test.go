package feishu

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func openReviewMaintenanceTestStore(t *testing.T) *SQLiteReviewGatewayStore {
	t.Helper()
	store, err := OpenSQLiteReviewGatewayStore(filepath.Join(t.TempDir(), "maintenance.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })
	return store
}

func createReviewTestBackup(t *testing.T, store *SQLiteReviewGatewayStore, name string) ReviewBackupResult {
	t.Helper()
	result, err := store.BackupReviewSQLite(context.Background(), ReviewBackupOptions{OutputDirectory: t.TempDir(), Name: name, Verify: true, RetentionCount: 7, Now: reviewConfigurationTestTime})
	if err != nil {
		t.Fatal(err)
	}
	return result
}

func TestBackupUsesConsistentSQLiteSnapshot(t *testing.T) {
	store := openReviewMaintenanceTestStore(t)
	if _, err := store.db.Exec(`INSERT INTO review_gateway_events(dedupe_key,received_at,expires_at) VALUES('committed','2026-01-01T00:00:00Z','2027-01-01T00:00:00Z')`); err != nil {
		t.Fatal(err)
	}
	backup := createReviewTestBackup(t, store, "consistent")
	db, err := OpenSQLiteReviewGatewayStore(backup.DatabasePath)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	var count int
	if err := db.db.QueryRow(`SELECT COUNT(*) FROM review_gateway_events WHERE dedupe_key='committed'`).Scan(&count); err != nil || count != 1 {
		t.Fatalf("count=%d err=%v", count, err)
	}
}

func TestBackupIncludesCommittedWALData(t *testing.T) {
	store := openReviewMaintenanceTestStore(t)
	if _, err := store.db.Exec(`PRAGMA journal_mode=WAL`); err != nil {
		t.Fatal(err)
	}
	if _, err := store.db.Exec(`INSERT INTO review_gateway_events(dedupe_key,received_at,expires_at) VALUES('wal-row','2026-01-01T00:00:00Z','2027-01-01T00:00:00Z')`); err != nil {
		t.Fatal(err)
	}
	backup := createReviewTestBackup(t, store, "wal")
	db, _ := OpenSQLiteReviewGatewayStore(backup.DatabasePath)
	defer db.Close()
	var count int
	if err := db.db.QueryRow(`SELECT COUNT(*) FROM review_gateway_events WHERE dedupe_key='wal-row'`).Scan(&count); err != nil || count != 1 {
		t.Fatalf("count=%d err=%v", count, err)
	}
}

func TestBackupManifestHasChecksum(t *testing.T) {
	store := openReviewMaintenanceTestStore(t)
	backup := createReviewTestBackup(t, store, "checksum")
	digest, size, err := reviewFileSHA256(backup.DatabasePath)
	if err != nil || backup.Manifest.DatabaseSHA256 != digest || backup.Manifest.DatabaseSize != size {
		t.Fatalf("manifest=%#v digest=%s size=%d err=%v", backup.Manifest, digest, size, err)
	}
}

func TestBackupManifestContainsNoAbsolutePath(t *testing.T) {
	store := openReviewMaintenanceTestStore(t)
	backup := createReviewTestBackup(t, store, "no-path")
	payload, _ := os.ReadFile(backup.ManifestPath)
	if strings.Contains(string(payload), filepath.Dir(store.path)) || strings.Contains(string(payload), store.path) {
		t.Fatalf("manifest exposed path: %s", payload)
	}
}

func TestBackupManifestContainsNoSecrets(t *testing.T) {
	store := openReviewMaintenanceTestStore(t)
	backup := createReviewTestBackup(t, store, "no-secret")
	payload, _ := os.ReadFile(backup.ManifestPath)
	for _, forbidden := range []string{"token", "secret", "authorization", "cookie", "chat_id", "remote_id"} {
		if strings.Contains(strings.ToLower(string(payload)), forbidden) {
			t.Fatalf("manifest exposed %q: %s", forbidden, payload)
		}
	}
}

func TestBackupQuickCheckPasses(t *testing.T) {
	store := openReviewMaintenanceTestStore(t)
	backup := createReviewTestBackup(t, store, "quick")
	if err := verifyReviewSQLiteIntegrity(context.Background(), backup.DatabasePath, true); err != nil {
		t.Fatal(err)
	}
}

func TestBackupIsAtomic(t *testing.T) {
	store := openReviewMaintenanceTestStore(t)
	backup := createReviewTestBackup(t, store, "atomic")
	entries, _ := filepath.Glob(filepath.Join(filepath.Dir(backup.DatabasePath), "*.tmp-*"))
	if len(entries) != 0 {
		t.Fatalf("temporary backup artifacts remain: %v", entries)
	}
}

func TestConcurrentBackupIsRejected(t *testing.T) {
	store := openReviewMaintenanceTestStore(t)
	lock := reviewMaintenanceLock(store.path, "backup")
	lock.Lock()
	defer lock.Unlock()
	if _, err := store.BackupReviewSQLite(context.Background(), ReviewBackupOptions{OutputDirectory: t.TempDir(), Name: "concurrent", Now: reviewConfigurationTestTime}); err == nil {
		t.Fatal("concurrent backup was accepted")
	}
}

func TestBackupRetentionKeepsLatestSuccessfulBackup(t *testing.T) {
	store := openReviewMaintenanceTestStore(t)
	directory := t.TempDir()
	paths := []string{}
	for index, name := range []string{"one", "two", "three"} {
		backup, err := store.BackupReviewSQLite(context.Background(), ReviewBackupOptions{OutputDirectory: directory, Name: name, Verify: true, RetentionCount: 2, Now: reviewConfigurationTestTime.Add(time.Duration(index) * time.Minute)})
		if err != nil {
			t.Fatal(err)
		}
		paths = append(paths, backup.DatabasePath)
	}
	if _, err := os.Stat(paths[0]); !os.IsNotExist(err) {
		t.Fatalf("old backup was retained: %v", err)
	}
	for _, path := range paths[1:] {
		if _, err := os.Stat(path); err != nil {
			t.Fatal(err)
		}
	}
}

func TestBackupSurvivesServiceRestart(t *testing.T) {
	path := filepath.Join(t.TempDir(), "source.db")
	store, _ := OpenSQLiteReviewGatewayStore(path)
	backup := createReviewTestBackup(t, store, "restart")
	_ = store.Close()
	store, _ = OpenSQLiteReviewGatewayStore(path)
	defer store.Close()
	if _, err := VerifyReviewBackup(context.Background(), backup.DatabasePath, backup.ManifestPath); err != nil {
		t.Fatal(err)
	}
}

func prepareReviewRestoreFixture(t *testing.T) (ReviewBackupResult, string) {
	t.Helper()
	source := openReviewMaintenanceTestStore(t)
	op, err := NewReviewOperation(ReviewOperationReplySend, ReviewGatewayJob{Repository: "owner/repo", ChatID: "chat"}, "restore-operation", "reply", map[string]string{"text": "restore"}, ReviewRetryNonIdempotent, reviewConfigurationTestTime)
	if err != nil {
		t.Fatal(err)
	}
	if err := source.SaveReviewOperations(context.Background(), []ReviewOperation{op}); err != nil {
		t.Fatal(err)
	}
	if _, err := source.db.Exec(`INSERT INTO review_operation_reconciliation_tasks(reconciliation_id,operation_id,resource_type,reason_code,status,created_at,updated_at) VALUES('restore-reconciliation',?,'reply','test','pending',?,?)`, op.OperationID, reviewGatewayTimestamp(reviewConfigurationTestTime), reviewGatewayTimestamp(reviewConfigurationTestTime)); err != nil {
		t.Fatal(err)
	}
	return createReviewTestBackup(t, source, "restore-source"), op.OperationID
}

func createReviewRestoreTarget(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "target.db")
	store, err := OpenSQLiteReviewGatewayStore(path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.db.Exec(`INSERT INTO review_gateway_events(dedupe_key,received_at,expires_at) VALUES('original-target','2026-01-01T00:00:00Z','2027-01-01T00:00:00Z')`); err != nil {
		t.Fatal(err)
	}
	_ = store.Close()
	return path
}

func TestRestoreVerifyOnlyDoesNotModifyDatabase(t *testing.T) {
	backup, _ := prepareReviewRestoreFixture(t)
	target := createReviewRestoreTarget(t)
	before, _, _ := reviewFileSHA256(target)
	if err := RestoreReviewSQLite(context.Background(), ReviewRestoreOptions{StateDB: target, Backup: backup.DatabasePath, Manifest: backup.ManifestPath, VerifyOnly: true}); err != nil {
		t.Fatal(err)
	}
	after, _, _ := reviewFileSHA256(target)
	if before != after {
		t.Fatal("verify-only changed target")
	}
}

func TestRestoreRejectsChecksumMismatch(t *testing.T) {
	backup, _ := prepareReviewRestoreFixture(t)
	file, _ := os.OpenFile(backup.DatabasePath, os.O_APPEND|os.O_WRONLY, 0)
	_, _ = file.Write([]byte("tamper"))
	_ = file.Close()
	if err := RestoreReviewSQLite(context.Background(), ReviewRestoreOptions{StateDB: createReviewRestoreTarget(t), Backup: backup.DatabasePath, Manifest: backup.ManifestPath, VerifyOnly: true}); err == nil {
		t.Fatal("checksum mismatch accepted")
	}
}

func TestRestoreRejectsInvalidSQLite(t *testing.T) {
	directory := t.TempDir()
	backupPath := filepath.Join(directory, "invalid.db")
	_ = os.WriteFile(backupPath, []byte("not sqlite"), 0o600)
	digest := sha256.Sum256([]byte("not sqlite"))
	manifest := ReviewBackupManifest{SchemaVersion: reviewBackupManifestSchema, DatabaseSHA256: hex.EncodeToString(digest[:]), DatabaseSize: int64(len("not sqlite"))}
	payload, _ := json.Marshal(manifest)
	manifestPath := backupPath + ".manifest.json"
	_ = os.WriteFile(manifestPath, payload, 0o600)
	if err := RestoreReviewSQLite(context.Background(), ReviewRestoreOptions{StateDB: createReviewRestoreTarget(t), Backup: backupPath, Manifest: manifestPath, VerifyOnly: true}); err == nil {
		t.Fatal("invalid SQLite accepted")
	}
}

func TestRestoreRejectsUnsupportedFutureSchema(t *testing.T) {
	backup, _ := prepareReviewRestoreFixture(t)
	db, _ := OpenSQLiteReviewGatewayStore(backup.DatabasePath)
	_, _ = db.db.Exec(`INSERT INTO schema_migrations(version,name,applied_at) VALUES(999,'future','2026-01-01T00:00:00Z')`)
	_ = db.Close()
	rewriteReviewBackupManifest(t, &backup)
	if err := RestoreReviewSQLite(context.Background(), ReviewRestoreOptions{StateDB: createReviewRestoreTarget(t), Backup: backup.DatabasePath, Manifest: backup.ManifestPath, VerifyOnly: true}); err == nil {
		t.Fatal("future schema accepted")
	}
}

func rewriteReviewBackupManifest(t *testing.T, backup *ReviewBackupResult) {
	t.Helper()
	digest, size, err := reviewFileSHA256(backup.DatabasePath)
	if err != nil {
		t.Fatal(err)
	}
	backup.Manifest.DatabaseSHA256, backup.Manifest.DatabaseSize = digest, size
	schema, _ := reviewSQLiteSchemaVersion(context.Background(), backup.DatabasePath)
	backup.Manifest.ReviewSchemaVersion = schema
	payload, _ := json.Marshal(backup.Manifest)
	if err := os.WriteFile(backup.ManifestPath, payload, 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestRestoreRejectsRunningService(t *testing.T) {
	backup, _ := prepareReviewRestoreFixture(t)
	target := createReviewRestoreTarget(t)
	lock, err := acquireReviewGatewayInstanceLock(target, "test", reviewConfigurationTestTime)
	if err != nil {
		t.Fatal(err)
	}
	defer lock.Release()
	if err := RestoreReviewSQLite(context.Background(), ReviewRestoreOptions{StateDB: target, Backup: backup.DatabasePath, Manifest: backup.ManifestPath, Replace: true, Confirmed: true}); err == nil {
		t.Fatal("running service restore accepted")
	}
}

func TestRestoreCreatesPreRestoreBackup(t *testing.T) {
	backup, _ := prepareReviewRestoreFixture(t)
	target := createReviewRestoreTarget(t)
	if err := RestoreReviewSQLite(context.Background(), ReviewRestoreOptions{StateDB: target, Backup: backup.DatabasePath, Manifest: backup.ManifestPath, Replace: true, Confirmed: true, Now: reviewConfigurationTestTime}); err != nil {
		t.Fatal(err)
	}
	artifacts, _ := filepath.Glob(filepath.Join(filepath.Dir(target), "pre-restore-backups", "*.db"))
	if len(artifacts) != 1 {
		t.Fatalf("pre-restore backups=%v", artifacts)
	}
}

func TestRestoreAtomicallyReplacesDatabase(t *testing.T) {
	backup, operationID := prepareReviewRestoreFixture(t)
	target := createReviewRestoreTarget(t)
	if err := RestoreReviewSQLite(context.Background(), ReviewRestoreOptions{StateDB: target, Backup: backup.DatabasePath, Manifest: backup.ManifestPath, Replace: true, Confirmed: true, Now: reviewConfigurationTestTime}); err != nil {
		t.Fatal(err)
	}
	store, _ := OpenSQLiteReviewGatewayStore(target)
	defer store.Close()
	if _, err := store.GetReviewOperation(context.Background(), operationID); err != nil {
		t.Fatal(err)
	}
	var original int
	_ = store.db.QueryRow(`SELECT COUNT(*) FROM review_gateway_events WHERE dedupe_key='original-target'`).Scan(&original)
	if original != 0 {
		t.Fatal("original database was not replaced")
	}
}

func TestRestoreRunsMigrationsAfterReplace(t *testing.T) {
	backup, _ := prepareReviewRestoreFixture(t)
	target := createReviewRestoreTarget(t)
	if err := RestoreReviewSQLite(context.Background(), ReviewRestoreOptions{StateDB: target, Backup: backup.DatabasePath, Manifest: backup.ManifestPath, Replace: true, Confirmed: true, Now: reviewConfigurationTestTime}); err != nil {
		t.Fatal(err)
	}
	store, _ := OpenSQLiteReviewGatewayStore(target)
	defer store.Close()
	version, err := store.CurrentSchemaVersion(context.Background())
	if err != nil || version != latestReviewGatewaySchemaVersion() {
		t.Fatalf("version=%d err=%v", version, err)
	}
}

func TestRestoreFailureRestoresOriginalDatabase(t *testing.T) {
	backup, _ := prepareReviewRestoreFixture(t)
	target := createReviewRestoreTarget(t)
	file, _ := os.OpenFile(backup.DatabasePath, os.O_APPEND|os.O_WRONLY, 0)
	_, _ = file.Write([]byte("tamper"))
	_ = file.Close()
	if err := RestoreReviewSQLite(context.Background(), ReviewRestoreOptions{StateDB: target, Backup: backup.DatabasePath, Manifest: backup.ManifestPath, Replace: true, Confirmed: true}); err == nil {
		t.Fatal("invalid restore succeeded")
	}
	store, _ := OpenSQLiteReviewGatewayStore(target)
	defer store.Close()
	var count int
	if err := store.db.QueryRow(`SELECT COUNT(*) FROM review_gateway_events WHERE dedupe_key='original-target'`).Scan(&count); err != nil || count != 1 {
		t.Fatalf("original count=%d err=%v", count, err)
	}
}

func TestRestoredDatabasePreservesOperationsAndReconciliation(t *testing.T) {
	backup, operationID := prepareReviewRestoreFixture(t)
	target := createReviewRestoreTarget(t)
	if err := RestoreReviewSQLite(context.Background(), ReviewRestoreOptions{StateDB: target, Backup: backup.DatabasePath, Manifest: backup.ManifestPath, Replace: true, Confirmed: true, Now: reviewConfigurationTestTime}); err != nil {
		t.Fatal(err)
	}
	store, _ := OpenSQLiteReviewGatewayStore(target)
	defer store.Close()
	var operations, reconciliations int
	_ = store.db.QueryRow(`SELECT COUNT(*) FROM review_operations WHERE operation_id=?`, operationID).Scan(&operations)
	_ = store.db.QueryRow(`SELECT COUNT(*) FROM review_operation_reconciliation_tasks WHERE operation_id=?`, operationID).Scan(&reconciliations)
	if operations != 1 || reconciliations != 1 {
		t.Fatalf("operations=%d reconciliations=%d", operations, reconciliations)
	}
}
