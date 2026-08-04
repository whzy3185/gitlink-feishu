package feishu

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"
)

const (
	reviewServiceHealthSchema    = "review.service-health/v1"
	reviewServiceReadinessSchema = "review.service-readiness/v1"
)

type ReviewServiceReadiness struct {
	Status        string            `json:"status"`
	SchemaVersion string            `json:"schema_version"`
	Components    map[string]string `json:"components"`
}

func (s *ReviewService) startAdminHTTP() error {
	listener, err := net.Listen("tcp", s.Config.AdminListen)
	if err != nil {
		return fmt.Errorf("listen for review service administration: %w", err)
	}
	s.adminListener = listener
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", s.handleHealth)
	mux.HandleFunc("/readyz", s.handleReady)
	s.AdminServer = &http.Server{Handler: mux, ReadHeaderTimeout: 3 * time.Second, ReadTimeout: 5 * time.Second, WriteTimeout: 5 * time.Second, IdleTimeout: 30 * time.Second}
	s.Status.RegisterComponent("admin_http", 0, true, true, s.now())
	s.Status.UpdateComponent("admin_http", 0, ReviewComponentRunning, s.now(), nil)
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		serveErr := s.AdminServer.Serve(listener)
		if serveErr != nil && serveErr != http.ErrServerClosed {
			s.Status.UpdateComponent("admin_http", 0, ReviewComponentFailed, s.now(), serveErr)
			select {
			case s.componentErrors <- reviewServiceComponentError{component: ReviewServiceComponent{Name: "admin_http", Required: true, Enabled: true}, err: serveErr}:
			case <-s.serviceContext().Done():
			}
		}
	}()
	return nil
}

func (s *ReviewService) handleHealth(writer http.ResponseWriter, _ *http.Request) {
	writeReviewServiceJSON(writer, http.StatusOK, map[string]interface{}{"status": "ok", "service": "gitlink-feishu-review", "version": reviewServiceHealthSchema})
}

func (s *ReviewService) handleReady(writer http.ResponseWriter, request *http.Request) {
	readiness := s.Readiness(request.Context())
	status := http.StatusOK
	if readiness.Status != "ready" {
		status = http.StatusServiceUnavailable
	}
	writeReviewServiceJSON(writer, status, readiness)
}

func (s *ReviewService) Readiness(parent context.Context) ReviewServiceReadiness {
	result := ReviewServiceReadiness{Status: "ready", SchemaVersion: reviewServiceReadinessSchema, Components: map[string]string{}}
	if s == nil || s.Status == nil {
		result.Status = "not_ready"
		result.Components["service"] = "not_ready"
		return result
	}
	snapshot := s.Status.Snapshot()
	if (snapshot.State != ReviewServiceReady && snapshot.State != ReviewServiceDegraded) || snapshot.Admission != ReviewAdmissionOpen {
		result.Status = "not_ready"
		result.Components["service"] = string(snapshot.State)
	}
	if s.InstanceLock == nil || !s.InstanceLock.Valid() {
		result.Status = "not_ready"
		result.Components["instance_lock"] = "not_ready"
	} else {
		result.Components["instance_lock"] = "ready"
	}
	ctx, cancel := context.WithTimeout(parent, 250*time.Millisecond)
	defer cancel()
	if s.Store == nil || s.Store.Ping(ctx) != nil {
		result.Status = "not_ready"
		result.Components["sqlite"] = "not_ready"
	} else {
		result.Components["sqlite"] = "ready"
		version, err := s.Store.CurrentSchemaVersion(ctx)
		if err != nil || version != latestReviewGatewaySchemaVersion() {
			result.Status = "not_ready"
			result.Components["migration"] = "not_ready"
		} else {
			result.Components["migration"] = "ready"
		}
	}
	s.mu.Lock()
	configurationLoaded := s.configurationLoaded && strings.TrimSpace(s.configFingerprint) != ""
	s.mu.Unlock()
	if !configurationLoaded {
		result.Status = "not_ready"
		result.Components["configuration"] = "not_ready"
	} else {
		result.Components["configuration"] = "ready"
	}
	if s.adminListener == nil {
		result.Status = "not_ready"
		result.Components["admin_http"] = "not_ready"
	}
	now := s.now()
	for _, component := range snapshot.Components {
		if !component.Required || component.Status == ReviewComponentDisabled {
			continue
		}
		name := component.Component
		componentReady := component.Status == ReviewComponentRunning || component.Status == ReviewComponentIdle
		if heartbeat, err := time.Parse(time.RFC3339Nano, component.LastHeartbeatAt); err != nil || now.Sub(heartbeat) > s.Config.WorkerStaleAfter {
			componentReady = false
		}
		if componentReady {
			result.Components[name] = "ready"
		} else {
			result.Status = "not_ready"
			result.Components[name] = "not_ready"
		}
	}
	return result
}

func writeReviewServiceJSON(writer http.ResponseWriter, status int, value interface{}) {
	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(status)
	_ = json.NewEncoder(writer).Encode(value)
}
