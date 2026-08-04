package feishu

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"
)

type ReviewServiceComponent struct {
	Name        string
	WorkerIndex int
	Required    bool
	Enabled     bool
	Start       func(context.Context) error
	Run         func(context.Context) error
	Stop        func(context.Context) error
}

type ReviewService struct {
	Store        *SQLiteReviewGatewayStore
	InstanceLock *reviewGatewayInstanceLock

	PublicServer *http.Server
	AdminServer  *http.Server
	Status       *ReviewServiceStatusRegistry
	Metrics      *ReviewMetrics
	Config       ReviewServiceConfig
	Now          func() time.Time

	Components []ReviewServiceComponent

	mu                       sync.Mutex
	ctx                      context.Context
	cancel                   context.CancelFunc
	wg                       sync.WaitGroup
	adminListener            net.Listener
	publicListener           net.Listener
	started                  bool
	shutdown                 bool
	ownedStore               bool
	ownedLock                bool
	configurationLoaded      bool
	configRevision           int
	configFingerprint        string
	componentErrors          chan reviewServiceComponentError
	backgroundFailureHandled chan struct{}
	adminToken               []byte
}

type reviewServiceComponentError struct {
	component ReviewServiceComponent
	err       error
}

func NewReviewService(config ReviewServiceConfig) *ReviewService {
	config = config.normalized()
	return &ReviewService{Config: config, Status: NewReviewServiceStatusRegistry(), Now: time.Now, componentErrors: make(chan reviewServiceComponentError, 16), backgroundFailureHandled: make(chan struct{}, 1)}
}

func (s *ReviewService) AddComponent(component ReviewServiceComponent) error {
	if s == nil {
		return fmt.Errorf("review service is required")
	}
	component.Name = strings.TrimSpace(component.Name)
	if component.Name == "" {
		return fmt.Errorf("review service component name is required")
	}
	for _, existing := range s.Components {
		if existing.Name == component.Name && existing.WorkerIndex == component.WorkerIndex {
			return fmt.Errorf("review service component %s/%d is already registered", component.Name, component.WorkerIndex)
		}
	}
	s.Components = append(s.Components, component)
	return nil
}

func (s *ReviewService) SetConfigurationState(revision int, fingerprint string, loaded bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.configRevision = revision
	s.configFingerprint = strings.TrimSpace(fingerprint)
	s.configurationLoaded = loaded && s.configFingerprint != ""
}

func (s *ReviewService) Start(parent context.Context) error {
	if s == nil {
		return fmt.Errorf("review service is required")
	}
	s.mu.Lock()
	if s.started {
		s.mu.Unlock()
		return fmt.Errorf("review service is already started")
	}
	s.Config = s.Config.normalized()
	if err := s.Config.Validate(); err != nil {
		s.mu.Unlock()
		return err
	}
	s.started = true
	s.ctx, s.cancel = context.WithCancel(parent)
	s.mu.Unlock()

	now := s.now()
	s.Status.SetService(ReviewServiceStarting, ReviewAdmissionClosed, now, nil)
	if s.InstanceLock == nil {
		lock, err := acquireReviewGatewayInstanceLock(s.Config.StateDB, s.Config.AppID, now)
		if err != nil {
			return s.startFailed(err, nil)
		}
		s.InstanceLock, s.ownedLock = lock, true
	}
	if s.Store == nil {
		store, err := OpenSQLiteReviewGatewayStore(s.Config.StateDB)
		if err != nil {
			return s.startFailed(err, nil)
		}
		s.Store, s.ownedStore = store, true
	}
	if !s.configurationLoaded {
		fingerprint, err := reviewServiceRuntimeConfigFingerprint(s.Config)
		if err != nil {
			return s.startFailed(err, nil)
		}
		s.SetConfigurationState(0, fingerprint, true)
	}
	s.Metrics = NewReviewMetrics(s.Store, s.Status, s.Config.StateDB, s.Now)
	if err := s.persistInstance(s.ctx, now); err != nil {
		return s.startFailed(err, nil)
	}
	if err := s.startAdminHTTP(); err != nil {
		return s.startFailed(err, nil)
	}

	started := []ReviewServiceComponent{}
	sorted := append([]ReviewServiceComponent(nil), s.Components...)
	sort.SliceStable(sorted, func(i, j int) bool {
		return reviewServiceComponentOrder(sorted[i].Name) < reviewServiceComponentOrder(sorted[j].Name)
	})
	optionalFailed := false
	for _, component := range sorted {
		s.Status.RegisterComponent(component.Name, component.WorkerIndex, component.Required, component.Enabled, s.now())
		if !component.Enabled {
			continue
		}
		if component.Start != nil {
			if err := component.Start(s.ctx); err != nil {
				s.Status.UpdateComponent(component.Name, component.WorkerIndex, ReviewComponentFailed, s.now(), err)
				if component.Required {
					return s.startFailed(fmt.Errorf("start required review component %s: %w", component.Name, err), started)
				}
				optionalFailed = true
				continue
			}
		}
		started = append(started, component)
		s.Status.UpdateComponent(component.Name, component.WorkerIndex, ReviewComponentRunning, s.now(), nil)
		s.runComponent(component)
	}
	state := ReviewServiceReady
	if optionalFailed {
		state = ReviewServiceDegraded
	}
	s.Status.SetService(state, ReviewAdmissionOpen, s.now(), nil)
	if err := s.persistInstance(s.ctx, s.now()); err != nil {
		return s.startFailed(err, started)
	}
	s.runHeartbeat()
	s.runErrorMonitor()
	return nil
}

func (s *ReviewService) runComponent(component ReviewServiceComponent) {
	if component.Run == nil {
		return
	}
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		err := component.Run(s.ctx)
		if s.ctx.Err() != nil {
			s.Status.UpdateComponent(component.Name, component.WorkerIndex, ReviewComponentStopped, s.now(), nil)
			return
		}
		if err == nil {
			err = fmt.Errorf("review component exited unexpectedly")
		}
		s.Status.UpdateComponent(component.Name, component.WorkerIndex, ReviewComponentFailed, s.now(), err)
		select {
		case s.componentErrors <- reviewServiceComponentError{component: component, err: err}:
		case <-s.ctx.Done():
		}
	}()
}

func (s *ReviewService) runHeartbeat() {
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		instanceTicker := time.NewTicker(s.Config.InstanceHeartbeat)
		componentTicker := time.NewTicker(s.Config.WorkerHeartbeatInterval)
		defer instanceTicker.Stop()
		defer componentTicker.Stop()
		for {
			select {
			case <-s.ctx.Done():
				return
			case now := <-instanceTicker.C:
				_ = s.persistInstance(context.Background(), now)
			case now := <-componentTicker.C:
				for _, component := range s.Status.Snapshot().Components {
					if component.Status == ReviewComponentRunning || component.Status == ReviewComponentIdle {
						s.Status.Heartbeat(component.Component, component.WorkerIndex, now)
					}
				}
			}
		}
	}()
}

func (s *ReviewService) runErrorMonitor() {
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		for {
			select {
			case <-s.ctx.Done():
				return
			case failure := <-s.componentErrors:
				if failure.component.Required {
					s.Status.SetService(ReviewServiceFailed, ReviewAdmissionClosed, s.now(), failure.err)
					s.cancel()
					select {
					case s.backgroundFailureHandled <- struct{}{}:
					default:
					}
					return
				}
				s.Status.SetService(ReviewServiceDegraded, ReviewAdmissionOpen, s.now(), failure.err)
			}
		}
	}()
}

func (s *ReviewService) startFailed(startErr error, started []ReviewServiceComponent) error {
	s.Status.SetService(ReviewServiceFailed, ReviewAdmissionClosed, s.now(), startErr)
	if s.cancel != nil {
		s.cancel()
	}
	for index := len(started) - 1; index >= 0; index-- {
		component := started[index]
		if component.Stop != nil {
			_ = component.Stop(context.Background())
		}
		s.Status.UpdateComponent(component.Name, component.WorkerIndex, ReviewComponentStopped, s.now(), nil)
	}
	s.closeHTTP(context.Background())
	if s.ownedStore && s.Store != nil {
		_ = s.Store.Close()
	}
	if s.ownedLock && s.InstanceLock != nil {
		_ = s.InstanceLock.Release()
	}
	return startErr
}

func (s *ReviewService) Shutdown(ctx context.Context) error {
	if s == nil {
		return nil
	}
	s.mu.Lock()
	if s.shutdown {
		s.mu.Unlock()
		return nil
	}
	s.shutdown = true
	s.mu.Unlock()
	shutdownCtx, shutdownCancel := reviewContextWithMaximum(ctx, s.Config.ShutdownTimeout)
	defer shutdownCancel()
	var shutdownErr error
	s.Status.SetService(ReviewServiceDraining, ReviewAdmissionDraining, s.now(), nil)
	if s.Store != nil {
		s.Store.CloseReviewAdmission()
		s.Store.CloseReviewClaims()
		releaseCtx, cancelRelease := context.WithTimeout(context.Background(), time.Second)
		shutdownErr = reviewShutdownError(shutdownErr, s.Store.ReleaseNotStartedReviewWork(releaseCtx, s.now()))
		cancelRelease()
	}
	// Stop public admission first. The loopback admin listener remains alive
	// long enough for readyz to report 503 while the service drains.
	if s.PublicServer != nil {
		shutdownErr = reviewShutdownError(shutdownErr, s.PublicServer.Shutdown(shutdownCtx))
	}
	if s.Store != nil {
		readCtx, cancelRead := reviewContextWithMaximum(shutdownCtx, s.Config.ReadDrainTimeout)
		readErr := s.Store.WaitForReviewReadJobs(readCtx)
		cancelRead()
		if readErr != nil && !errors.Is(readErr, context.DeadlineExceeded) && !errors.Is(readErr, context.Canceled) {
			shutdownErr = reviewShutdownError(shutdownErr, readErr)
		}
		operationCtx, cancelOperations := reviewContextWithMaximum(shutdownCtx, s.Config.OperationDrainTimeout)
		operationErr := s.Store.WaitForReviewOperations(operationCtx)
		cancelOperations()
		if operationErr != nil && !errors.Is(operationErr, context.DeadlineExceeded) && !errors.Is(operationErr, context.Canceled) {
			shutdownErr = reviewShutdownError(shutdownErr, operationErr)
		}
		transitionCtx, cancelTransition := context.WithTimeout(context.Background(), 2*time.Second)
		shutdownErr = reviewShutdownError(shutdownErr, s.Store.PrepareReviewWorkForShutdown(transitionCtx, s.now()))
		cancelTransition()
	}
	if s.AdminServer != nil {
		shutdownErr = reviewShutdownError(shutdownErr, s.AdminServer.Shutdown(shutdownCtx))
	}
	if s.cancel != nil {
		s.cancel()
	}
	for index := len(s.Components) - 1; index >= 0; index-- {
		component := s.Components[index]
		if !component.Enabled {
			continue
		}
		s.Status.UpdateComponent(component.Name, component.WorkerIndex, ReviewComponentStopping, s.now(), nil)
		if component.Stop != nil {
			shutdownErr = reviewShutdownError(shutdownErr, component.Stop(shutdownCtx))
		}
		s.Status.UpdateComponent(component.Name, component.WorkerIndex, ReviewComponentStopped, s.now(), nil)
	}
	done := make(chan struct{})
	go func() { s.wg.Wait(); close(done) }()
	select {
	case <-done:
	case <-shutdownCtx.Done():
		shutdownErr = reviewShutdownError(shutdownErr, shutdownCtx.Err())
	}
	s.Status.SetService(ReviewServiceStopping, ReviewAdmissionClosed, s.now(), nil)
	_ = s.persistInstance(context.Background(), s.now())
	if s.Store != nil && reviewDatabaseOpen(s.Store.db) {
		checkpointCtx, checkpointCancel := context.WithTimeout(context.Background(), 2*time.Second)
		_, checkpointErr := s.Store.WALCheckpoint(checkpointCtx, "TRUNCATE", s.Metrics, s.now())
		checkpointCancel()
		shutdownErr = reviewShutdownError(shutdownErr, checkpointErr)
	}
	s.Status.SetService(ReviewServiceStopped, ReviewAdmissionClosed, s.now(), nil)
	_ = s.persistInstance(context.Background(), s.now())
	if s.ownedStore && s.Store != nil {
		_ = s.Store.Close()
	}
	if s.ownedLock && s.InstanceLock != nil {
		shutdownErr = reviewShutdownError(shutdownErr, s.InstanceLock.Release())
	}
	return shutdownErr
}

func (s *ReviewService) persistInstance(ctx context.Context, now time.Time) error {
	if s.Store == nil || s.InstanceLock == nil {
		return nil
	}
	schema, err := s.Store.CurrentSchemaVersion(ctx)
	if err != nil {
		return err
	}
	s.mu.Lock()
	revision, fingerprint := s.configRevision, s.configFingerprint
	s.mu.Unlock()
	return s.Store.SaveReviewServiceInstance(ctx, newReviewServiceInstance(s.InstanceLock, s.Config, s.Status.Snapshot(), schema, revision, fingerprint, now))
}

func (s *ReviewService) now() time.Time {
	if s.Now != nil {
		return s.Now().UTC()
	}
	return time.Now().UTC()
}

func reviewServiceComponentOrder(name string) int {
	order := []string{"admin_http", "public_webhook", "feishu_channel", "event_processor", "job_worker_gitlink_read", "job_worker_collaboration", "job_worker_controlled_write", "operation_planner", "operation_worker_canonical_card", "operation_worker_reply", "operation_worker_resource", "operation_reconciliation", "pr_reconciliation", "maintenance", "agent_worker"}
	for index, item := range order {
		if name == item {
			return index
		}
	}
	return len(order) + 1
}

func reviewServiceRuntimeConfigFingerprint(config ReviewServiceConfig) (string, error) {
	value := map[string]interface{}{
		"admin_listen": config.AdminListen, "webhook_listen": config.WebhookListen,
		"shutdown_timeout": config.ShutdownTimeout.String(), "read_drain_timeout": config.ReadDrainTimeout.String(),
		"operation_drain_timeout": config.OperationDrainTimeout.String(), "worker_heartbeat_interval": config.WorkerHeartbeatInterval.String(),
		"worker_stale_after": config.WorkerStaleAfter.String(), "metrics_enabled": config.MetricsEnabled,
		"backup_interval": config.BackupInterval.String(), "backup_retention_count": config.BackupRetentionCount,
		"retention_interval": config.RetentionInterval.String(), "wal_checkpoint_interval": config.WALCheckpointInterval.String(),
	}
	_, fingerprint, err := boundedReviewOperationPayload(value)
	return fingerprint, err
}

func (s *ReviewService) closeHTTP(ctx context.Context) {
	if s.AdminServer != nil {
		_ = s.AdminServer.Shutdown(ctx)
	}
	if s.PublicServer != nil {
		_ = s.PublicServer.Shutdown(ctx)
	}
}

func (s *ReviewService) AdminAddress() string {
	if s == nil || s.adminListener == nil {
		return ""
	}
	return s.adminListener.Addr().String()
}

func (s *ReviewService) BackgroundFailureHandled() <-chan struct{} {
	return s.backgroundFailureHandled
}

func (s *ReviewService) serviceContext() context.Context {
	if s.ctx != nil {
		return s.ctx
	}
	return context.Background()
}
