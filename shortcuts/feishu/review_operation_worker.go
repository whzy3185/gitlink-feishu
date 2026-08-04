package feishu

import (
	"context"
	"fmt"
	"strings"
	"time"
)

type ReviewOperationExecutor interface {
	Execute(context.Context, ReviewOperation) (ReviewOperationExecutionResult, error)
}

type ReviewOperationWorker struct {
	Store         *SQLiteReviewGatewayStore
	Handler       ReviewOperationExecutor
	QueueClass    string
	LeaseOwner    string
	LeaseDuration time.Duration
	PollInterval  time.Duration
	BatchSize     int
	Now           func() time.Time
	wake          chan struct{}
}

func NewReviewOperationWorker(store *SQLiteReviewGatewayStore, handler ReviewOperationExecutor, queueClass, leaseOwner string) *ReviewOperationWorker {
	return &ReviewOperationWorker{
		Store:         store,
		Handler:       handler,
		QueueClass:    strings.TrimSpace(queueClass),
		LeaseOwner:    strings.TrimSpace(leaseOwner),
		LeaseDuration: 2 * time.Minute,
		PollInterval:  time.Second,
		BatchSize:     1,
		Now:           time.Now,
		wake:          make(chan struct{}, 1),
	}
}

func (w *ReviewOperationWorker) Wake() {
	if w == nil {
		return
	}
	select {
	case w.wake <- struct{}{}:
	default:
	}
}

func (w *ReviewOperationWorker) Run(ctx context.Context) error {
	if err := w.validate(); err != nil {
		return err
	}
	ticker := time.NewTicker(w.PollInterval)
	defer ticker.Stop()
	w.Wake()
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-w.wake:
			w.runReady(ctx)
		case <-ticker.C:
			w.runReady(ctx)
		}
	}
}

func (w *ReviewOperationWorker) RunOnce(ctx context.Context) (bool, error) {
	if err := w.validate(); err != nil {
		return false, err
	}
	operations, err := w.Store.ClaimReviewOperations(ctx, ReviewOperationClaimOptions{
		QueueClass:    w.QueueClass,
		LeaseOwner:    w.LeaseOwner,
		Now:           w.now(),
		LeaseDuration: w.LeaseDuration,
		Limit:         w.BatchSize,
	})
	if err != nil || len(operations) == 0 {
		return false, err
	}
	for _, operation := range operations {
		attempt, startErr := w.Store.StartReviewOperationAttempt(ctx, operation, w.now())
		if startErr != nil {
			return true, startErr
		}
		// The network boundary is intentionally outside every SQLite
		// transaction. Claim and attempt creation have already committed.
		outcome, executeErr := w.Handler.Execute(ctx, operation)
		if finishErr := w.Store.FinishReviewOperation(ctx, operation, attempt, outcome, executeErr, w.now()); finishErr != nil {
			return true, finishErr
		}
	}
	return true, nil
}

func (w *ReviewOperationWorker) runReady(ctx context.Context) {
	for {
		ran, err := w.RunOnce(ctx)
		if err != nil || !ran {
			return
		}
	}
}

func (w *ReviewOperationWorker) validate() error {
	if w == nil || w.Store == nil || w.Handler == nil {
		return fmt.Errorf("review operation worker store and handler are required")
	}
	if w.QueueClass != "canonical_card" && w.QueueClass != "reply" && w.QueueClass != "resource" {
		return fmt.Errorf("unsupported review operation queue class %q", w.QueueClass)
	}
	if w.LeaseOwner == "" {
		return fmt.Errorf("review operation worker lease owner is required")
	}
	if w.Now == nil {
		w.Now = time.Now
	}
	if w.PollInterval <= 0 {
		w.PollInterval = time.Second
	}
	if w.LeaseDuration <= 0 {
		w.LeaseDuration = 2 * time.Minute
	}
	if w.BatchSize <= 0 || w.BatchSize > 50 {
		w.BatchSize = 1
	}
	if w.wake == nil {
		w.wake = make(chan struct{}, 1)
	}
	return nil
}

func (w *ReviewOperationWorker) now() time.Time {
	if w.Now == nil {
		return time.Now().UTC()
	}
	return w.Now().UTC()
}
