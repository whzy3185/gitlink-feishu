package feishu

import (
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

type ReviewServiceState string
type ReviewServiceAdmissionState string
type ReviewServiceComponentState string

const (
	ReviewServiceCreated  ReviewServiceState = "created"
	ReviewServiceStarting ReviewServiceState = "starting"
	ReviewServiceReady    ReviewServiceState = "ready"
	ReviewServiceDegraded ReviewServiceState = "degraded"
	ReviewServiceDraining ReviewServiceState = "draining"
	ReviewServiceStopping ReviewServiceState = "stopping"
	ReviewServiceStopped  ReviewServiceState = "stopped"
	ReviewServiceFailed   ReviewServiceState = "failed"

	ReviewAdmissionClosed   ReviewServiceAdmissionState = "closed"
	ReviewAdmissionOpen     ReviewServiceAdmissionState = "open"
	ReviewAdmissionDraining ReviewServiceAdmissionState = "draining"

	ReviewComponentStarting ReviewServiceComponentState = "starting"
	ReviewComponentRunning  ReviewServiceComponentState = "running"
	ReviewComponentIdle     ReviewServiceComponentState = "idle"
	ReviewComponentDegraded ReviewServiceComponentState = "degraded"
	ReviewComponentStopping ReviewServiceComponentState = "stopping"
	ReviewComponentStopped  ReviewServiceComponentState = "stopped"
	ReviewComponentFailed   ReviewServiceComponentState = "failed"
	ReviewComponentDisabled ReviewServiceComponentState = "disabled"
)

type ReviewServiceComponentStatus struct {
	Component        string                      `json:"component"`
	WorkerIndex      int                         `json:"worker_index,omitempty"`
	Required         bool                        `json:"required"`
	Status           ReviewServiceComponentState `json:"status"`
	StartedAt        string                      `json:"started_at,omitempty"`
	LastHeartbeatAt  string                      `json:"last_heartbeat_at,omitempty"`
	LastSuccessAt    string                      `json:"last_success_at,omitempty"`
	LastErrorAt      string                      `json:"last_error_at,omitempty"`
	LastErrorSummary string                      `json:"last_error_summary,omitempty"`
}

type ReviewServiceStatusSnapshot struct {
	State             ReviewServiceState             `json:"state"`
	Admission         ReviewServiceAdmissionState    `json:"admission"`
	StartedAt         string                         `json:"started_at,omitempty"`
	ReadyAt           string                         `json:"ready_at,omitempty"`
	ShutdownStartedAt string                         `json:"shutdown_started_at,omitempty"`
	StoppedAt         string                         `json:"stopped_at,omitempty"`
	LastErrorSummary  string                         `json:"last_error_summary,omitempty"`
	Components        []ReviewServiceComponentStatus `json:"components"`
}

type ReviewServiceStatusRegistry struct {
	mu         sync.RWMutex
	state      ReviewServiceState
	admission  ReviewServiceAdmissionState
	startedAt  time.Time
	readyAt    time.Time
	shutdownAt time.Time
	stoppedAt  time.Time
	lastError  string
	components map[string]ReviewServiceComponentStatus
}

func NewReviewServiceStatusRegistry() *ReviewServiceStatusRegistry {
	return &ReviewServiceStatusRegistry{state: ReviewServiceCreated, admission: ReviewAdmissionClosed, components: map[string]ReviewServiceComponentStatus{}}
}

func reviewServiceComponentKey(name string, index int) string {
	return strings.TrimSpace(name) + ":" + reviewGatewayIntString(index)
}

func (r *ReviewServiceStatusRegistry) SetService(state ReviewServiceState, admission ReviewServiceAdmissionState, now time.Time, serviceErr error) {
	if r == nil {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.state, r.admission = state, admission
	switch state {
	case ReviewServiceStarting:
		if r.startedAt.IsZero() {
			r.startedAt = now.UTC()
		}
	case ReviewServiceReady, ReviewServiceDegraded:
		if r.readyAt.IsZero() {
			r.readyAt = now.UTC()
		}
	case ReviewServiceDraining, ReviewServiceStopping:
		if r.shutdownAt.IsZero() {
			r.shutdownAt = now.UTC()
		}
	case ReviewServiceStopped:
		r.stoppedAt = now.UTC()
	}
	if serviceErr != nil {
		r.lastError = redactReviewGatewayError(serviceErr.Error())
	}
}

func (r *ReviewServiceStatusRegistry) RegisterComponent(name string, index int, required, enabled bool, now time.Time) {
	if r == nil {
		return
	}
	state := ReviewComponentStarting
	if !enabled {
		state = ReviewComponentDisabled
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.components[reviewServiceComponentKey(name, index)] = ReviewServiceComponentStatus{Component: strings.TrimSpace(name), WorkerIndex: index, Required: required, Status: state, StartedAt: reviewGatewayTimestamp(now), LastHeartbeatAt: reviewGatewayTimestamp(now)}
}

func (r *ReviewServiceStatusRegistry) UpdateComponent(name string, index int, state ReviewServiceComponentState, now time.Time, componentErr error) {
	if r == nil {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	key := reviewServiceComponentKey(name, index)
	value := r.components[key]
	value.Component, value.WorkerIndex, value.Status = strings.TrimSpace(name), index, state
	value.LastHeartbeatAt = reviewGatewayTimestamp(now)
	if state == ReviewComponentRunning || state == ReviewComponentIdle {
		value.LastSuccessAt = reviewGatewayTimestamp(now)
	}
	if componentErr != nil {
		value.LastErrorAt = reviewGatewayTimestamp(now)
		value.LastErrorSummary = redactReviewGatewayError(componentErr.Error())
	}
	r.components[key] = value
}

func (r *ReviewServiceStatusRegistry) Heartbeat(name string, index int, now time.Time) {
	if r == nil {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	key := reviewServiceComponentKey(name, index)
	value := r.components[key]
	value.LastHeartbeatAt = reviewGatewayTimestamp(now)
	r.components[key] = value
}

func (r *ReviewServiceStatusRegistry) Snapshot() ReviewServiceStatusSnapshot {
	if r == nil {
		return ReviewServiceStatusSnapshot{State: ReviewServiceFailed, Admission: ReviewAdmissionClosed}
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	snapshot := ReviewServiceStatusSnapshot{State: r.state, Admission: r.admission, StartedAt: reviewServiceTime(r.startedAt), ReadyAt: reviewServiceTime(r.readyAt), ShutdownStartedAt: reviewServiceTime(r.shutdownAt), StoppedAt: reviewServiceTime(r.stoppedAt), LastErrorSummary: r.lastError}
	for _, value := range r.components {
		snapshot.Components = append(snapshot.Components, value)
	}
	sort.Slice(snapshot.Components, func(i, j int) bool {
		if snapshot.Components[i].Component == snapshot.Components[j].Component {
			return snapshot.Components[i].WorkerIndex < snapshot.Components[j].WorkerIndex
		}
		return snapshot.Components[i].Component < snapshot.Components[j].Component
	})
	return snapshot
}

func reviewGatewayIntString(value int) string {
	return strconv.Itoa(value)
}

func reviewServiceTime(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return reviewGatewayTimestamp(value)
}
