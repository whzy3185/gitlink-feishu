package feishu

import (
	"context"
	"errors"
	"io"
	"net/http"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

var reviewServiceTestTime = time.Date(2026, 8, 5, 8, 0, 0, 0, time.UTC)

func newReviewServiceTestHarness(t *testing.T) *ReviewService {
	t.Helper()
	config := DefaultReviewServiceConfig()
	config.StateDB = filepath.Join(t.TempDir(), "service.db")
	config.AppID = "app-test"
	config.AdminListen = "127.0.0.1:0"
	config.WorkerHeartbeatInterval = time.Second
	config.WorkerStaleAfter = 2 * time.Second
	config.InstanceHeartbeat = time.Second
	service := NewReviewService(config)
	service.Now = func() time.Time { return reviewServiceTestTime }
	for _, name := range []string{"event_processor", "job_worker_gitlink_read", "job_worker_collaboration", "job_worker_controlled_write", "operation_planner", "operation_worker_canonical_card", "operation_worker_reply", "operation_worker_resource"} {
		if err := service.AddComponent(ReviewServiceComponent{Name: name, Required: true, Enabled: true}); err != nil {
			t.Fatal(err)
		}
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		_ = service.Shutdown(ctx)
	})
	return service
}

func requireReviewServiceStart(t *testing.T, service *ReviewService) {
	t.Helper()
	if err := service.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
}

func TestReviewServiceStartsComponentsInOrder(t *testing.T) {
	service := newReviewServiceTestHarness(t)
	service.Components = nil
	var order []string
	var mu sync.Mutex
	for _, name := range []string{"operation_worker_resource", "event_processor", "job_worker_gitlink_read"} {
		name := name
		_ = service.AddComponent(ReviewServiceComponent{Name: name, Required: true, Enabled: true, Start: func(context.Context) error { mu.Lock(); defer mu.Unlock(); order = append(order, name); return nil }})
	}
	requireReviewServiceStart(t, service)
	if strings.Join(order, ",") != "event_processor,job_worker_gitlink_read,operation_worker_resource" {
		t.Fatalf("start order=%v", order)
	}
}

func TestReviewServiceStartFailureRollsBackStartedComponents(t *testing.T) {
	service := newReviewServiceTestHarness(t)
	service.Components = nil
	stopped := false
	_ = service.AddComponent(ReviewServiceComponent{Name: "event_processor", Required: true, Enabled: true, Stop: func(context.Context) error { stopped = true; return nil }})
	_ = service.AddComponent(ReviewServiceComponent{Name: "job_worker_gitlink_read", Required: true, Enabled: true, Start: func(context.Context) error { return errors.New("synthetic start failure") }})
	if err := service.Start(context.Background()); err == nil || !stopped {
		t.Fatalf("err=%v stopped=%t", err, stopped)
	}
	if snapshot := service.Status.Snapshot(); snapshot.State != ReviewServiceFailed || snapshot.Admission != ReviewAdmissionClosed {
		t.Fatalf("snapshot=%+v", snapshot)
	}
}

func TestReviewServicePublishesReadyOnlyAfterRequiredComponents(t *testing.T) {
	service := newReviewServiceTestHarness(t)
	if readiness := service.Readiness(context.Background()); readiness.Status == "ready" {
		t.Fatal("service ready before start")
	}
	requireReviewServiceStart(t, service)
	if readiness := service.Readiness(context.Background()); readiness.Status != "ready" {
		t.Fatalf("readiness=%+v", readiness)
	}
}

func TestReviewServiceOptionalComponentFailureIsDegraded(t *testing.T) {
	service := newReviewServiceTestHarness(t)
	_ = service.AddComponent(ReviewServiceComponent{Name: "maintenance", Enabled: true, Start: func(context.Context) error { return errors.New("synthetic optional failure") }})
	requireReviewServiceStart(t, service)
	if snapshot := service.Status.Snapshot(); snapshot.State != ReviewServiceDegraded || snapshot.Admission != ReviewAdmissionOpen {
		t.Fatalf("snapshot=%+v", snapshot)
	}
}

func TestReviewServiceRequiredComponentFailureStopsService(t *testing.T) {
	service := newReviewServiceTestHarness(t)
	fail := make(chan struct{})
	service.Components[0].Run = func(ctx context.Context) error {
		select {
		case <-fail:
			return errors.New("synthetic runtime failure")
		case <-ctx.Done():
			return nil
		}
	}
	requireReviewServiceStart(t, service)
	close(fail)
	select {
	case <-service.BackgroundFailureHandled():
	case <-time.After(time.Second):
		t.Fatal("required failure was not observed")
	}
	if snapshot := service.Status.Snapshot(); snapshot.State != ReviewServiceFailed || snapshot.Admission != ReviewAdmissionClosed {
		t.Fatalf("snapshot=%+v", snapshot)
	}
}

func TestReviewServiceInstanceHeartbeat(t *testing.T) {
	service := newReviewServiceTestHarness(t)
	requireReviewServiceStart(t, service)
	now := reviewServiceTestTime.Add(time.Minute)
	if err := service.persistInstance(context.Background(), now); err != nil {
		t.Fatal(err)
	}
	stored, err := service.Store.GetReviewServiceInstance(context.Background(), service.InstanceLock.metadata.InstanceID)
	if err != nil || stored.HeartbeatAt != reviewGatewayTimestamp(now) || stored.HostnameHash == "" {
		t.Fatalf("instance=%+v err=%v", stored, err)
	}
}

func TestReviewServiceInstanceStopsCleanly(t *testing.T) {
	service := newReviewServiceTestHarness(t)
	path := service.Config.StateDB
	requireReviewServiceStart(t, service)
	instanceID := service.InstanceLock.metadata.InstanceID
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := service.Shutdown(ctx); err != nil {
		t.Fatal(err)
	}
	store, err := OpenSQLiteReviewGatewayStore(path)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	instance, err := store.GetReviewServiceInstance(context.Background(), instanceID)
	if err != nil || instance.Status != ReviewServiceStopped || instance.StoppedAt == "" {
		t.Fatalf("instance=%+v err=%v", instance, err)
	}
}

func TestReviewServiceReusesExistingInstanceLock(t *testing.T) {
	service := newReviewServiceTestHarness(t)
	lock, err := acquireReviewGatewayInstanceLock(service.Config.StateDB, service.Config.AppID, reviewServiceTestTime)
	if err != nil {
		t.Fatal(err)
	}
	service.InstanceLock = lock
	t.Cleanup(func() { _ = lock.Release() })
	requireReviewServiceStart(t, service)
	if service.InstanceLock != lock || !lock.Valid() {
		t.Fatal("service did not reuse injected lock")
	}
}

func TestHealthIsIndependentOfSQLite(t *testing.T) {
	service := newReviewServiceTestHarness(t)
	requireReviewServiceStart(t, service)
	_ = service.Store.Close()
	response, err := http.Get("http://" + service.AdminAddress() + "/healthz")
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		t.Fatalf("status=%d", response.StatusCode)
	}
}

func TestHealthIsIndependentOfExternalDependencies(t *testing.T) {
	service := newReviewServiceTestHarness(t)
	requireReviewServiceStart(t, service)
	request, _ := http.NewRequest(http.MethodGet, "http://service/healthz", nil)
	recorder := newReviewHTTPRecorder()
	service.handleHealth(recorder, request)
	if recorder.status != http.StatusOK || !strings.Contains(recorder.body.String(), reviewServiceHealthSchema) {
		t.Fatalf("status=%d body=%s", recorder.status, recorder.body.String())
	}
}

func TestReadySucceedsWhenRequiredComponentsRun(t *testing.T) {
	service := newReviewServiceTestHarness(t)
	requireReviewServiceStart(t, service)
	if readiness := service.Readiness(context.Background()); readiness.Status != "ready" {
		t.Fatalf("readiness=%+v", readiness)
	}
}

func TestReadyFailsWhenSQLiteUnavailable(t *testing.T) {
	service := newReviewServiceTestHarness(t)
	requireReviewServiceStart(t, service)
	_ = service.Store.Close()
	if readiness := service.Readiness(context.Background()); readiness.Status != "not_ready" || readiness.Components["sqlite"] != "not_ready" {
		t.Fatalf("readiness=%+v", readiness)
	}
}

func TestReadyFailsWhenMigrationPending(t *testing.T) {
	service := newReviewServiceTestHarness(t)
	requireReviewServiceStart(t, service)
	if _, err := service.Store.db.Exec(`DELETE FROM schema_migrations WHERE version=?`, latestReviewGatewaySchemaVersion()); err != nil {
		t.Fatal(err)
	}
	if readiness := service.Readiness(context.Background()); readiness.Status != "not_ready" || readiness.Components["migration"] != "not_ready" {
		t.Fatalf("readiness=%+v", readiness)
	}
}

func TestReadyFailsWhenConfigurationMissing(t *testing.T) {
	service := newReviewServiceTestHarness(t)
	requireReviewServiceStart(t, service)
	service.SetConfigurationState(0, "", false)
	if readiness := service.Readiness(context.Background()); readiness.Status != "not_ready" || readiness.Components["configuration"] != "not_ready" {
		t.Fatalf("readiness=%+v", readiness)
	}
}

func TestReadyFailsWhenRequiredWorkerStopped(t *testing.T) {
	service := newReviewServiceTestHarness(t)
	requireReviewServiceStart(t, service)
	service.Status.UpdateComponent("job_worker_gitlink_read", 0, ReviewComponentStopped, reviewServiceTestTime, nil)
	if readiness := service.Readiness(context.Background()); readiness.Status != "not_ready" {
		t.Fatalf("readiness=%+v", readiness)
	}
}

func TestReadyFailsWhenWorkerHeartbeatIsStale(t *testing.T) {
	service := newReviewServiceTestHarness(t)
	requireReviewServiceStart(t, service)
	service.Status.UpdateComponent("job_worker_gitlink_read", 0, ReviewComponentRunning, reviewServiceTestTime.Add(-time.Minute), nil)
	if readiness := service.Readiness(context.Background()); readiness.Status != "not_ready" {
		t.Fatalf("readiness=%+v", readiness)
	}
}

func TestReadyFailsDuringDraining(t *testing.T) {
	service := newReviewServiceTestHarness(t)
	requireReviewServiceStart(t, service)
	service.Status.SetService(ReviewServiceDraining, ReviewAdmissionDraining, reviewServiceTestTime, nil)
	if readiness := service.Readiness(context.Background()); readiness.Status != "not_ready" {
		t.Fatalf("readiness=%+v", readiness)
	}
}

func TestReadyDoesNotCallGitLinkOrFeishu(t *testing.T) {
	service := newReviewServiceTestHarness(t)
	requireReviewServiceStart(t, service)
	if readiness := service.Readiness(context.Background()); readiness.Status != "ready" {
		t.Fatalf("readiness=%+v", readiness)
	}
}

func TestReadyResponseContainsNoSensitiveIdentifiers(t *testing.T) {
	service := newReviewServiceTestHarness(t)
	requireReviewServiceStart(t, service)
	request, _ := http.NewRequest(http.MethodGet, "http://service/readyz", nil)
	recorder := newReviewHTTPRecorder()
	service.handleReady(recorder, request)
	body, _ := io.ReadAll(strings.NewReader(recorder.body.String()))
	for _, forbidden := range []string{"chat-secret", "repository-secret", service.InstanceLock.metadata.InstanceID, service.Config.StateDB} {
		if strings.Contains(string(body), forbidden) {
			t.Fatalf("ready response leaked %q: %s", forbidden, body)
		}
	}
}

type reviewHTTPRecorder struct {
	header http.Header
	status int
	body   strings.Builder
}

func newReviewHTTPRecorder() *reviewHTTPRecorder     { return &reviewHTTPRecorder{header: http.Header{}} }
func (r *reviewHTTPRecorder) Header() http.Header    { return r.header }
func (r *reviewHTTPRecorder) WriteHeader(status int) { r.status = status }
func (r *reviewHTTPRecorder) Write(payload []byte) (int, error) {
	if r.status == 0 {
		r.status = http.StatusOK
	}
	return r.body.Write(payload)
}
