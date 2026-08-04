package feishu

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func startReviewShutdownTestService(t *testing.T) *ReviewService {
	t.Helper()
	service := newReviewServiceTestHarness(t)
	service.Config.OperationDrainTimeout = time.Second
	service.Config.ReadDrainTimeout = time.Second
	requireReviewServiceStart(t, service)
	return service
}

func shutdownReviewServiceForTest(t *testing.T, service *ReviewService) error {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	return service.Shutdown(ctx)
}

func TestGracefulShutdownClosesAdmissionFirst(t *testing.T) {
	service := startReviewShutdownTestService(t)
	if err := shutdownReviewServiceForTest(t, service); err != nil {
		t.Fatal(err)
	}
	if service.Status.Snapshot().Admission != ReviewAdmissionClosed || service.Store.reviewAdmissionAllowed() {
		t.Fatal("shutdown left admission open")
	}
}

func TestGracefulShutdownMakesReadyReturn503(t *testing.T) {
	service := startReviewShutdownTestService(t)
	if err := shutdownReviewServiceForTest(t, service); err != nil {
		t.Fatal(err)
	}
	recorder := httptest.NewRecorder()
	service.handleReady(recorder, httptest.NewRequest(http.MethodGet, "/readyz", nil))
	if recorder.Code != http.StatusServiceUnavailable {
		t.Fatalf("ready status=%d", recorder.Code)
	}
}

func TestGracefulShutdownStopsNewWebhookAdmission(t *testing.T) {
	service := startReviewShutdownTestService(t)
	if err := shutdownReviewServiceForTest(t, service); err != nil {
		t.Fatal(err)
	}
	_, err := service.Store.ReserveAndSaveJob(context.Background(), ReviewGatewayJob{JobID: "after", DedupeKey: "after", CreatedAt: reviewGatewayTimestamp(service.now())}, service.now(), time.Hour)
	if !errors.Is(err, ErrReviewServiceDraining) {
		t.Fatalf("admission err=%v", err)
	}
}

func TestGracefulShutdownStopsNewJobClaims(t *testing.T) {
	service := startReviewShutdownTestService(t)
	if err := shutdownReviewServiceForTest(t, service); err != nil {
		t.Fatal(err)
	}
	jobs, err := service.Store.ClaimReadyJobs(context.Background(), ReviewGatewayClaimOptions{LeaseOwner: "after", Now: service.now(), Limit: 1})
	if err != nil || len(jobs) != 0 {
		t.Fatalf("jobs=%#v err=%v", jobs, err)
	}
}

func TestGracefulShutdownDrainsSafeReadJobs(t *testing.T) {
	service := startReviewShutdownTestService(t)
	now := reviewGatewayTimestamp(service.now())
	if _, err := service.Store.db.Exec(`INSERT INTO review_gateway_jobs(job_id,dedupe_key,status,action,requested_by,chat_id,payload_json,created_at,updated_at,queue_class,lease_owner,lease_expires_at) VALUES('read-running','read-running','running','view','u','c','{}',?,?,'gitlink_read','worker',?)`, now, now, reviewGatewayTimestamp(service.now().Add(time.Minute))); err != nil {
		t.Fatal(err)
	}
	go func() {
		time.Sleep(60 * time.Millisecond)
		_, _ = service.Store.db.Exec(`UPDATE review_gateway_jobs SET status='completed',lease_owner='',lease_expires_at='',updated_at=? WHERE job_id='read-running'`, reviewGatewayTimestamp(service.now()))
	}()
	if err := shutdownReviewServiceForTest(t, service); err != nil {
		t.Fatal(err)
	}
}

func TestGracefulShutdownReleasesNotStartedOperationLease(t *testing.T) {
	service := startReviewShutdownTestService(t)
	op := seedReviewShutdownOperation(t, service, ReviewRetryIdempotent, "leased", "not_started")
	path := service.Config.StateDB
	if err := shutdownReviewServiceForTest(t, service); err != nil {
		t.Fatal(err)
	}
	reopened, err := OpenSQLiteReviewGatewayStore(path)
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.Close()
	got, err := reopened.GetReviewOperation(context.Background(), op.OperationID)
	if err != nil || got.Status != ReviewOperationPending || got.LeaseOwner != "" {
		t.Fatalf("operation=%#v err=%v", got, err)
	}
}

func TestGracefulShutdownKeepsNonIdempotentInFlightUnknown(t *testing.T) {
	assertReviewShutdownUncertainOperation(t, ReviewRetryNonIdempotent, ReviewOperationUnknown)
}

func TestGracefulShutdownSendsReconcilableOperationToReconciliation(t *testing.T) {
	assertReviewShutdownUncertainOperation(t, ReviewRetryReconcilable, ReviewOperationNeedsReconciliation)
}

func assertReviewShutdownUncertainOperation(t *testing.T, safety ReviewOperationRetrySafety, want ReviewOperationStatus) {
	t.Helper()
	service := startReviewShutdownTestService(t)
	op := seedReviewShutdownOperation(t, service, safety, "writing", "request_started")
	path := service.Config.StateDB
	if err := shutdownReviewServiceForTest(t, service); err != nil {
		t.Fatal(err)
	}
	reopened, err := OpenSQLiteReviewGatewayStore(path)
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.Close()
	got, err := reopened.GetReviewOperation(context.Background(), op.OperationID)
	if err != nil || got.Status != want || !got.RequiresReconciliation || got.MutationStatus != ReviewMutationRemoteUnknown {
		t.Fatalf("operation=%#v err=%v", got, err)
	}
	var count int
	if err := reopened.db.QueryRow(`SELECT COUNT(*) FROM review_operation_reconciliation_tasks WHERE operation_id=?`, op.OperationID).Scan(&count); err != nil || count != 1 {
		t.Fatalf("reconciliation count=%d err=%v", count, err)
	}
}

func seedReviewShutdownOperation(t *testing.T, service *ReviewService, safety ReviewOperationRetrySafety, status, mutation string) ReviewOperation {
	t.Helper()
	op, err := NewReviewOperation(ReviewOperationReplySend, ReviewGatewayJob{Repository: "owner/repo", ChatID: "chat"}, "shutdown-"+string(safety)+status, "reply", map[string]string{"text": "ok"}, safety, service.now())
	if err != nil {
		t.Fatal(err)
	}
	if err := service.Store.SaveReviewOperations(context.Background(), []ReviewOperation{op}); err != nil {
		t.Fatal(err)
	}
	if _, err := service.Store.db.Exec(`UPDATE review_operations SET status=?,mutation_status=?,lease_owner='worker',lease_expires_at=? WHERE operation_id=?`, status, mutation, reviewGatewayTimestamp(service.now().Add(time.Minute)), op.OperationID); err != nil {
		t.Fatal(err)
	}
	return op
}

func TestGracefulShutdownDoesNotBlindlyRetryReply(t *testing.T) {
	service := startReviewShutdownTestService(t)
	now := reviewGatewayTimestamp(service.now())
	if _, err := service.Store.db.Exec(`INSERT INTO review_gateway_jobs(job_id,dedupe_key,status,action,requested_by,chat_id,payload_json,created_at,updated_at,reply_status,reply_lease_owner,reply_lease_expires_at) VALUES('reply-running','reply-running','completed','view','u','c','{}',?,?,'sending','reply-worker',?)`, now, now, reviewGatewayTimestamp(service.now().Add(time.Minute))); err != nil {
		t.Fatal(err)
	}
	path := service.Config.StateDB
	if err := shutdownReviewServiceForTest(t, service); err != nil {
		t.Fatal(err)
	}
	reopened, _ := OpenSQLiteReviewGatewayStore(path)
	defer reopened.Close()
	var status string
	var reconciliation int
	if err := reopened.db.QueryRow(`SELECT reply_status,reply_requires_reconciliation FROM review_gateway_jobs WHERE job_id='reply-running'`).Scan(&status, &reconciliation); err != nil || status != "unknown" || reconciliation != 1 {
		t.Fatalf("reply status=%s reconciliation=%d err=%v", status, reconciliation, err)
	}
}

func TestGracefulShutdownCheckpointsWAL(t *testing.T) {
	service := startReviewShutdownTestService(t)
	path := service.Config.StateDB
	if err := shutdownReviewServiceForTest(t, service); err != nil {
		t.Fatal(err)
	}
	reopened, _ := OpenSQLiteReviewGatewayStore(path)
	defer reopened.Close()
	var count int
	if err := reopened.db.QueryRow(`SELECT COUNT(*) FROM review_maintenance_runs WHERE maintenance_kind='wal_checkpoint' AND status='succeeded'`).Scan(&count); err != nil || count == 0 {
		t.Fatalf("checkpoint count=%d err=%v", count, err)
	}
}

func TestGracefulShutdownClosesDatabase(t *testing.T) {
	service := startReviewShutdownTestService(t)
	if err := shutdownReviewServiceForTest(t, service); err != nil {
		t.Fatal(err)
	}
	if !service.Store.Closed() || service.Store.db.Ping() == nil {
		t.Fatal("shutdown left SQLite open")
	}
}

func TestGracefulShutdownReleasesInstanceLock(t *testing.T) {
	service := startReviewShutdownTestService(t)
	lock := service.InstanceLock
	if err := shutdownReviewServiceForTest(t, service); err != nil {
		t.Fatal(err)
	}
	if lock.Valid() {
		t.Fatal("shutdown left instance lock valid")
	}
}

func TestGracefulShutdownHonorsTimeout(t *testing.T) {
	service := startReviewShutdownTestService(t)
	now := reviewGatewayTimestamp(service.now())
	if _, err := service.Store.db.Exec(`INSERT INTO review_gateway_jobs(job_id,dedupe_key,status,action,requested_by,chat_id,payload_json,created_at,updated_at,queue_class,lease_owner,lease_expires_at) VALUES('read-timeout','read-timeout','running','view','u','c','{}',?,?,'gitlink_read','worker',?)`, now, now, reviewGatewayTimestamp(service.now().Add(time.Minute))); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 40*time.Millisecond)
	defer cancel()
	if err := service.Shutdown(ctx); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("shutdown err=%v", err)
	}
	if !service.Store.Closed() || service.InstanceLock.Valid() {
		t.Fatal("timed out shutdown did not finalize resources")
	}
}

func TestGracefulShutdownIsIdempotent(t *testing.T) {
	service := startReviewShutdownTestService(t)
	if err := shutdownReviewServiceForTest(t, service); err != nil {
		t.Fatal(err)
	}
	if err := service.Shutdown(context.Background()); err != nil {
		t.Fatal(err)
	}
}
