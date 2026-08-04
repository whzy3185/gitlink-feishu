package feishu

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

func oldReviewMaintenanceTime() string {
	return reviewGatewayTimestamp(reviewConfigurationTestTime.Add(-60 * 24 * time.Hour))
}

func insertReviewRetentionJob(t *testing.T, store *SQLiteReviewGatewayStore, id, status string, updated string) {
	t.Helper()
	if _, err := store.db.Exec(`INSERT INTO review_gateway_jobs(job_id,dedupe_key,status,action,requested_by,chat_id,payload_json,created_at,updated_at,queue_class) VALUES(?,?,?,?,?,'chat','{}',?,?,'gitlink_read')`, id, id, status, "view", "user", updated, updated); err != nil {
		t.Fatal(err)
	}
}

func insertReviewRetentionOperation(t *testing.T, store *SQLiteReviewGatewayStore, key string, status ReviewOperationStatus) ReviewOperation {
	t.Helper()
	op, err := NewReviewOperation(ReviewOperationCanonicalCardUpsert, ReviewGatewayJob{Repository: "owner/repo", ChatID: "chat"}, key, "card", map[string]string{"title": key}, ReviewRetryIdempotent, reviewConfigurationTestTime.Add(-60*24*time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if err := store.SaveReviewOperations(context.Background(), []ReviewOperation{op}); err != nil {
		t.Fatal(err)
	}
	completed := ""
	if status == ReviewOperationSucceeded || status == ReviewOperationUnchanged || status == ReviewOperationStale {
		completed = oldReviewMaintenanceTime()
	}
	if _, err := store.db.Exec(`UPDATE review_operations SET status=?,completed_at=?,updated_at=? WHERE operation_id=?`, status, completed, oldReviewMaintenanceTime(), op.OperationID); err != nil {
		t.Fatal(err)
	}
	return op
}

func runReviewRetentionApply(t *testing.T, store *SQLiteReviewGatewayStore, batch int) ReviewRetentionResult {
	t.Helper()
	result, err := store.RunReviewRetention(context.Background(), ReviewRetentionOptions{DryRun: false, BatchSize: batch, Now: reviewConfigurationTestTime})
	if err != nil {
		t.Fatal(err)
	}
	return result
}

func tableReviewCount(t *testing.T, store *SQLiteReviewGatewayStore, table, where string, args ...interface{}) int {
	t.Helper()
	var count int
	if err := store.db.QueryRow(`SELECT COUNT(*) FROM `+table+` WHERE `+where, args...).Scan(&count); err != nil {
		t.Fatal(err)
	}
	return count
}

func TestRetentionDryRunDeletesNothing(t *testing.T) {
	store := openReviewMaintenanceTestStore(t)
	insertReviewRetentionJob(t, store, "dry-run", "completed", oldReviewMaintenanceTime())
	result, err := store.RunReviewRetention(context.Background(), ReviewRetentionOptions{DryRun: true, BatchSize: 1000, Now: reviewConfigurationTestTime})
	if err != nil || result.RowsAffected != 1 || tableReviewCount(t, store, "review_gateway_jobs", "job_id='dry-run'") != 1 {
		t.Fatalf("result=%#v err=%v", result, err)
	}
}

func TestRetentionDeletesOldCompletedJobs(t *testing.T) {
	store := openReviewMaintenanceTestStore(t)
	insertReviewRetentionJob(t, store, "completed", "completed", oldReviewMaintenanceTime())
	runReviewRetentionApply(t, store, 1000)
	if tableReviewCount(t, store, "review_gateway_jobs", "job_id='completed'") != 0 {
		t.Fatal("old completed job retained")
	}
}

func TestRetentionDeletesOldTerminalOperations(t *testing.T) {
	store := openReviewMaintenanceTestStore(t)
	op := insertReviewRetentionOperation(t, store, "terminal", ReviewOperationSucceeded)
	runReviewRetentionApply(t, store, 1000)
	if tableReviewCount(t, store, "review_operations", "operation_id=?", op.OperationID) != 0 {
		t.Fatal("old terminal operation retained")
	}
}

func TestRetentionDeletesExpiredRateLimitBuckets(t *testing.T) {
	store := openReviewMaintenanceTestStore(t)
	if _, err := store.db.Exec(`INSERT INTO review_rate_limit_buckets(bucket_key,bucket_type,subject_hash,window_started_at,updated_at) VALUES('old','chat','hash',?,?)`, oldReviewMaintenanceTime(), oldReviewMaintenanceTime()); err != nil {
		t.Fatal(err)
	}
	runReviewRetentionApply(t, store, 1000)
	if tableReviewCount(t, store, "review_rate_limit_buckets", "bucket_key='old'") != 0 {
		t.Fatal("expired rate-limit bucket retained")
	}
}

func TestRetentionKeepsPendingJobs(t *testing.T) {
	store := openReviewMaintenanceTestStore(t)
	insertReviewRetentionJob(t, store, "pending", "queued", oldReviewMaintenanceTime())
	runReviewRetentionApply(t, store, 1000)
	if tableReviewCount(t, store, "review_gateway_jobs", "job_id='pending'") != 1 {
		t.Fatal("pending job deleted")
	}
}

func TestRetentionKeepsUnknownOperations(t *testing.T) {
	store := openReviewMaintenanceTestStore(t)
	op := insertReviewRetentionOperation(t, store, "unknown", ReviewOperationUnknown)
	runReviewRetentionApply(t, store, 1000)
	if tableReviewCount(t, store, "review_operations", "operation_id=?", op.OperationID) != 1 {
		t.Fatal("unknown operation deleted")
	}
}

func TestRetentionKeepsOpenDeadLetters(t *testing.T) {
	store := openReviewMaintenanceTestStore(t)
	if _, err := store.db.Exec(`INSERT INTO review_dead_letters(dead_letter_id,entity_type,entity_id,error_class,status,created_at,updated_at) VALUES('open','job','j','permanent','open',?,?)`, oldReviewMaintenanceTime(), oldReviewMaintenanceTime()); err != nil {
		t.Fatal(err)
	}
	runReviewRetentionApply(t, store, 1000)
	if tableReviewCount(t, store, "review_dead_letters", "dead_letter_id='open'") != 1 {
		t.Fatal("open dead letter deleted")
	}
}

func TestRetentionKeepsUnresolvedReconciliation(t *testing.T) {
	store := openReviewMaintenanceTestStore(t)
	if _, err := store.db.Exec(`INSERT INTO review_operation_reconciliation_tasks(reconciliation_id,operation_id,resource_type,reason_code,status,created_at,updated_at) VALUES('pending','operation','card','test','pending',?,?)`, oldReviewMaintenanceTime(), oldReviewMaintenanceTime()); err != nil {
		t.Fatal(err)
	}
	runReviewRetentionApply(t, store, 1000)
	if tableReviewCount(t, store, "review_operation_reconciliation_tasks", "reconciliation_id='pending'") != 1 {
		t.Fatal("unresolved reconciliation deleted")
	}
}

func TestRetentionKeepsActivePresentations(t *testing.T) {
	store := openReviewMaintenanceTestStore(t)
	if _, err := store.db.Exec(`INSERT INTO review_pr_presentations(presentation_key,installation_id,repository,pr_number,schema_version,updated_at) VALUES('active','i','owner/repo',1,'v1',?)`, oldReviewMaintenanceTime()); err != nil {
		t.Fatal(err)
	}
	runReviewRetentionApply(t, store, 1000)
	if tableReviewCount(t, store, "review_pr_presentations", "presentation_key='active'") != 1 {
		t.Fatal("active presentation deleted")
	}
}

func TestRetentionKeepsResourceMigrationAudit(t *testing.T) {
	store := openReviewMaintenanceTestStore(t)
	if _, err := store.db.Exec(`INSERT INTO review_resource_migration_audit(audit_id,migration_id,action,from_status,to_status,created_at) VALUES('audit','migration','test','pending','completed',?)`, oldReviewMaintenanceTime()); err != nil {
		t.Fatal(err)
	}
	runReviewRetentionApply(t, store, 1000)
	if tableReviewCount(t, store, "review_resource_migration_audit", "audit_id='audit'") != 1 {
		t.Fatal("resource migration audit deleted")
	}
}

func TestRetentionUsesBoundedBatches(t *testing.T) {
	store := openReviewMaintenanceTestStore(t)
	for _, id := range []string{"batch-1", "batch-2", "batch-3"} {
		insertReviewRetentionJob(t, store, id, "completed", oldReviewMaintenanceTime())
	}
	result := runReviewRetentionApply(t, store, 2)
	if result.ByTable["review_gateway_jobs"] != 2 || tableReviewCount(t, store, "review_gateway_jobs", "job_id LIKE 'batch-%'") != 1 {
		t.Fatalf("result=%#v", result)
	}
}

func TestRetentionSurvivesRestart(t *testing.T) {
	path := filepath.Join(t.TempDir(), "retention.db")
	store, _ := OpenSQLiteReviewGatewayStore(path)
	insertReviewRetentionJob(t, store, "restart-completed", "completed", oldReviewMaintenanceTime())
	runReviewRetentionApply(t, store, 1000)
	_ = store.Close()
	store, _ = OpenSQLiteReviewGatewayStore(path)
	defer store.Close()
	if tableReviewCount(t, store, "review_gateway_jobs", "job_id='restart-completed'") != 0 {
		t.Fatal("retention deletion did not survive restart")
	}
}

func TestWALCheckpointRecordsMaintenanceRun(t *testing.T) {
	store := openReviewMaintenanceTestStore(t)
	if _, err := store.WALCheckpoint(context.Background(), "PASSIVE", nil, reviewConfigurationTestTime); err != nil {
		t.Fatal(err)
	}
	if tableReviewCount(t, store, "review_maintenance_runs", "maintenance_kind='wal_checkpoint' AND status='succeeded'") != 1 {
		t.Fatal("WAL maintenance run missing")
	}
}

func TestWALCheckpointBusyIsNotDataCorruption(t *testing.T) {
	store := openReviewMaintenanceTestStore(t)
	tx, err := store.db.Begin()
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()
	if _, err := tx.Exec(`INSERT INTO review_gateway_events(dedupe_key,received_at,expires_at) VALUES('busy','2026-01-01T00:00:00Z','2027-01-01T00:00:00Z')`); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()
	result, err := store.WALCheckpoint(ctx, "PASSIVE", nil, reviewConfigurationTestTime)
	if err != nil && !strings.Contains(strings.ToLower(err.Error()), "busy") && !strings.Contains(strings.ToLower(err.Error()), "locked") && ctx.Err() == nil {
		t.Fatalf("checkpoint=%#v err=%v", result, err)
	}
}

func TestIntegrityCheckReportsFailureSafely(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sensitive-invalid.db")
	if err := os.WriteFile(path, []byte("not sqlite"), 0o600); err != nil {
		t.Fatal(err)
	}
	err := verifyReviewSQLiteIntegrity(context.Background(), path, false)
	if err == nil {
		t.Fatal("invalid database passed integrity check")
	}
	if strings.Contains(redactReviewMaintenanceError(err.Error()), filepath.Dir(path)) {
		t.Fatal("integrity error exposed path")
	}
}

func TestMaintenanceSchedulerDoesNotOverlapSameTask(t *testing.T) {
	store := openReviewMaintenanceTestStore(t)
	scheduler := &ReviewMaintenanceScheduler{Store: store}
	value, _ := scheduler.locks.LoadOrStore("wal_checkpoint", &sync.Mutex{})
	lock := value.(*sync.Mutex)
	lock.Lock()
	defer lock.Unlock()
	if err := scheduler.RunKind(context.Background(), "wal_checkpoint"); err == nil {
		t.Fatal("overlapping maintenance was accepted")
	}
}

func TestMaintenanceSchedulerStopsOnContextCancel(t *testing.T) {
	store := openReviewMaintenanceTestStore(t)
	scheduler := &ReviewMaintenanceScheduler{Store: store, WALInterval: time.Hour}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() { scheduler.Run(ctx); close(done) }()
	cancel()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("maintenance scheduler did not stop")
	}
}

func TestMaintenanceMetricsUpdateAfterSuccess(t *testing.T) {
	store := openReviewMaintenanceTestStore(t)
	metrics := NewReviewMetrics(store, NewReviewServiceStatusRegistry(), store.path, time.Now)
	if _, err := store.WALCheckpoint(context.Background(), "PASSIVE", metrics, reviewConfigurationTestTime); err != nil {
		t.Fatal(err)
	}
	if metrics.walSuccess.Load() == 0 {
		t.Fatal("WAL success metric was not updated")
	}
}

func TestMaintenanceErrorContainsNoSensitivePath(t *testing.T) {
	path := `E:\sensitive\review.db`
	redacted := redactReviewMaintenanceError("failed to open " + path + ": access denied")
	if strings.Contains(redacted, path) || strings.Contains(redacted, "sensitive") {
		t.Fatalf("path not redacted: %s", redacted)
	}
}

func TestMaintenanceMigrationIsIdempotent(t *testing.T) {
	testReviewConfigurationMigrationIdempotent(t, 18)
}

func TestStageFiveMigrationRollbackLeavesDatabaseUsable(t *testing.T) {
	store := openReviewMaintenanceTestStore(t)
	bad := reviewGatewaySchemaMigration{Version: 20, Name: "intentional_failure", Statements: []string{`CREATE TABLE stage_five_rollback_probe(id TEXT)`, `INVALID SQL`}}
	if err := applyReviewGatewaySchemaMigrations(store.db, []reviewGatewaySchemaMigration{bad}); err == nil {
		t.Fatal("invalid migration succeeded")
	}
	var count int
	if err := store.db.QueryRow(`SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='stage_five_rollback_probe'`).Scan(&count); err != nil || count != 0 {
		t.Fatalf("partial migration remained count=%d err=%v", count, err)
	}
	if err := store.db.Ping(); err != nil {
		t.Fatal(err)
	}
}
