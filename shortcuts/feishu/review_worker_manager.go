package feishu

import (
	"context"
	"fmt"
	"time"
)

type ReviewWorkerConcurrency struct {
	Event                   int `json:"event"`
	GitLinkRead             int `json:"gitlink_read"`
	Collaboration           int `json:"collaboration"`
	Planner                 int `json:"planner"`
	CanonicalCard           int `json:"canonical_card"`
	Reply                   int `json:"reply"`
	Resource                int `json:"resource"`
	ControlledWrite         int `json:"controlled_write"`
	Agent                   int `json:"agent"`
	OperationReconciliation int `json:"operation_reconciliation"`
}

func DefaultReviewWorkerConcurrency() ReviewWorkerConcurrency {
	return ReviewWorkerConcurrency{Event: 2, GitLinkRead: 3, Collaboration: 1, Planner: 2, CanonicalCard: 2, Reply: 2, Resource: 2, ControlledWrite: 1, Agent: 0, OperationReconciliation: 1}
}

func reviewWorkerConcurrencyConfiguration(c ReviewWorkerConcurrency) map[string]int {
	return map[string]int{
		"event": c.Event, "gitlink_read": c.GitLinkRead, "collaboration": c.Collaboration,
		"planner": c.Planner, "canonical_card": c.CanonicalCard, "reply": c.Reply,
		"resource": c.Resource, "controlled_write": c.ControlledWrite, "agent": c.Agent,
		"operation_reconciliation": c.OperationReconciliation,
	}
}

func (c ReviewWorkerConcurrency) Validate() error {
	values := map[string]int{"event": c.Event, "gitlink_read": c.GitLinkRead, "collaboration": c.Collaboration, "planner": c.Planner, "canonical_card": c.CanonicalCard, "reply": c.Reply, "resource": c.Resource, "controlled_write": c.ControlledWrite, "agent": c.Agent, "operation_reconciliation": c.OperationReconciliation}
	for name, value := range values {
		if value < 0 || value > 32 {
			return fmt.Errorf("Review worker concurrency %s=%d is outside 0..32", name, value)
		}
	}
	if c.GitLinkRead == 0 || c.Collaboration == 0 || c.Planner == 0 || c.CanonicalCard == 0 || c.Reply == 0 || c.Resource == 0 || c.ControlledWrite == 0 {
		return fmt.Errorf("required Review worker pools must have at least one worker")
	}
	return nil
}

type ReviewWorkerManager struct {
	Queue        *ReviewGatewayQueue
	Concurrency  ReviewWorkerConcurrency
	PollInterval time.Duration
	InstanceID   string
}

func (m *ReviewWorkerManager) Run(ctx context.Context, handler func(context.Context, ReviewGatewayJob) (ReviewGatewayExecutionResult, error)) error {
	if m == nil || m.Queue == nil {
		return fmt.Errorf("Review worker manager queue is required")
	}
	if m.Concurrency == (ReviewWorkerConcurrency{}) {
		m.Concurrency = DefaultReviewWorkerConcurrency()
	}
	if err := m.Concurrency.Validate(); err != nil {
		return err
	}
	if m.PollInterval <= 0 {
		m.PollInterval = time.Second
	}
	classes := []struct {
		name  string
		count int
	}{{ReviewQueueGitLinkRead, m.Concurrency.GitLinkRead}, {ReviewQueueCollaboration, m.Concurrency.Collaboration}, {ReviewQueueControlledWrite, m.Concurrency.ControlledWrite}, {ReviewQueueAgent, m.Concurrency.Agent}}
	for _, class := range classes {
		for index := 0; index < class.count; index++ {
			owner := fmt.Sprintf("%s-%s-%d", firstNonEmpty(m.InstanceID, "review-worker"), class.name, index+1)
			go m.runJobWorker(ctx, class.name, owner, handler)
		}
	}
	<-ctx.Done()
	return nil
}

func (m *ReviewWorkerManager) RunClass(ctx context.Context, queueClass string, count int, handler func(context.Context, ReviewGatewayJob) (ReviewGatewayExecutionResult, error)) error {
	if m == nil || m.Queue == nil {
		return fmt.Errorf("Review worker manager queue is required")
	}
	if count < 1 || count > 32 {
		return fmt.Errorf("Review worker class %s concurrency must be between 1 and 32", queueClass)
	}
	if m.PollInterval <= 0 {
		m.PollInterval = time.Second
	}
	for index := 0; index < count; index++ {
		owner := fmt.Sprintf("%s-%s-%d", firstNonEmpty(m.InstanceID, "review-worker"), queueClass, index+1)
		go m.runJobWorker(ctx, queueClass, owner, handler)
	}
	<-ctx.Done()
	return nil
}

func (m *ReviewWorkerManager) runJobWorker(ctx context.Context, queueClass, owner string, handler func(context.Context, ReviewGatewayJob) (ReviewGatewayExecutionResult, error)) {
	ticker := time.NewTicker(m.PollInterval)
	defer ticker.Stop()
	for {
		m.Queue.runReadyJobsForClass(ctx, handler, queueClass, owner)
		select {
		case <-ctx.Done():
			return
		case <-m.Queue.wake:
		case <-ticker.C:
		}
	}
}

func NewReviewOperationWorkerPool(store *SQLiteReviewGatewayStore, handler ReviewOperationExecutor, concurrency ReviewWorkerConcurrency, instanceID string) []*ReviewOperationWorker {
	workers := []*ReviewOperationWorker{}
	specs := []struct {
		queue string
		count int
	}{{"canonical_card", concurrency.CanonicalCard}, {"reply", concurrency.Reply}, {"resource", concurrency.Resource}}
	for _, spec := range specs {
		for index := 0; index < spec.count; index++ {
			workers = append(workers, NewReviewOperationWorker(store, handler, spec.queue, fmt.Sprintf("%s-%s-%d", instanceID, spec.queue, index+1)))
		}
	}
	return workers
}
