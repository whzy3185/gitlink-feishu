package feishu

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/gitlink-org/gitlink-cli/shortcuts/common"
)

type reviewOperationAdminResult struct {
	SchemaVersion   string                              `json:"schema_version"`
	Action          string                              `json:"action"`
	ReadOnly        bool                                `json:"read_only"`
	Operations      []reviewOperationAdminView          `json:"operations,omitempty"`
	DeadLetters     []ReviewDeadLetter                  `json:"dead_letters,omitempty"`
	Reconciliations []ReviewOperationReconciliationTask `json:"reconciliations,omitempty"`
	QueueStatus     *ReviewQueueStatus                  `json:"queue_status,omitempty"`
}

type ReviewQueueStatus struct {
	Jobs                   map[string]int `json:"jobs"`
	Operations             map[string]int `json:"operations"`
	Consumers              map[string]int `json:"consumers"`
	OpenDeadLetters        int            `json:"open_dead_letters"`
	PendingReconciliations int            `json:"pending_reconciliations"`
}

type reviewOperationAdminView struct {
	OperationID            string                        `json:"operation_id"`
	Kind                   ReviewOperationKind           `json:"kind"`
	QueueClass             string                        `json:"queue_class"`
	Repository             string                        `json:"repository,omitempty"`
	PRNumber               int                           `json:"pr_number,omitempty"`
	ChatIDHash             string                        `json:"chat_id_hash,omitempty"`
	Status                 ReviewOperationStatus         `json:"status"`
	MutationStatus         ReviewOperationMutationStatus `json:"mutation_status"`
	RetrySafety            ReviewOperationRetrySafety    `json:"retry_safety"`
	RemoteIDHash           string                        `json:"remote_id_hash,omitempty"`
	ErrorClass             string                        `json:"error_class,omitempty"`
	ErrorCode              string                        `json:"error_code,omitempty"`
	ErrorSummary           string                        `json:"error_summary,omitempty"`
	AttemptCount           int                           `json:"attempt_count"`
	RequiresReconciliation bool                          `json:"requires_reconciliation"`
}

func newReviewOperationShortcut() *common.Shortcut {
	return &common.Shortcut{
		Name: "review-operation", Description: "Inspect and safely administer durable Review operations in local SQLite",
		Flags: []common.Flag{{Name: "action", Usage: "list, inspect, retry, or reconcile", Required: true}, {Name: "state-db", Usage: "SQLite state database", Default: ".local/review-gateway.db"}, {Name: "id", Usage: "Operation ID"}, {Name: "status", Usage: "Status filter"}, {Name: "queue-class", Usage: "Queue class filter"}, {Name: "reason", Usage: "Required audit reason"}, {Name: "yes", Usage: "Explicit confirmation", Bool: true, Default: "false"}}, Run: runReviewOperationAdmin,
	}
}

func newReviewDeadLetterShortcut() *common.Shortcut {
	return &common.Shortcut{
		Name: "review-dead-letter", Description: "Inspect and safely resolve local Review dead letters",
		Flags: []common.Flag{{Name: "action", Usage: "list, inspect, retry, resolve, or ignore", Required: true}, {Name: "state-db", Usage: "SQLite state database", Default: ".local/review-gateway.db"}, {Name: "id", Usage: "Dead letter ID"}, {Name: "status", Usage: "Status filter"}, {Name: "entity-type", Usage: "Entity type filter"}, {Name: "reason", Usage: "Required resolution reason"}, {Name: "actor", Usage: "Local operator; stored only as a hash"}, {Name: "yes", Usage: "Explicit confirmation", Bool: true, Default: "false"}}, Run: runReviewDeadLetterAdmin,
	}
}

func newReviewOperationReconciliationShortcut() *common.Shortcut {
	return &common.Shortcut{
		Name: "review-operation-reconciliation", Description: "Inspect or manually resolve external side-effect reconciliation tasks",
		Flags: []common.Flag{{Name: "action", Usage: "list, inspect, or resolve", Required: true}, {Name: "state-db", Usage: "SQLite state database", Default: ".local/review-gateway.db"}, {Name: "id", Usage: "Reconciliation ID"}, {Name: "status", Usage: "Status filter"}, {Name: "reason", Usage: "Required resolution summary"}, {Name: "actor", Usage: "Local operator; stored only as a hash"}, {Name: "success", Usage: "Mark manually verified success", Bool: true, Default: "false"}, {Name: "yes", Usage: "Explicit confirmation", Bool: true, Default: "false"}}, Run: runReviewOperationReconciliationAdmin,
	}
}

func newReviewQueueStatusShortcut() *common.Shortcut {
	return &common.Shortcut{Name: "review-queue-status", Description: "Show bounded local Review job and operation queue counts", Flags: []common.Flag{{Name: "state-db", Usage: "SQLite state database", Default: ".local/review-gateway.db"}}, Run: func(runtime *common.RuntimeContext) error {
		store, err := OpenSQLiteReviewGatewayStore(runtime.Arg("state-db"))
		if err != nil {
			return err
		}
		defer store.Close()
		status, err := store.ReviewQueueStatus(context.Background())
		if err != nil {
			return err
		}
		return json.NewEncoder(os.Stdout).Encode(reviewOperationAdminResult{SchemaVersion: "feishu.review-queue-status/v1", Action: "status", ReadOnly: true, QueueStatus: &status})
	}}
}

func (s *SQLiteReviewGatewayStore) ReviewQueueStatus(ctx context.Context) (ReviewQueueStatus, error) {
	result := ReviewQueueStatus{Jobs: map[string]int{}, Operations: map[string]int{}, Consumers: map[string]int{}}
	rows, err := s.db.QueryContext(ctx, `SELECT queue_class||':'||status,COUNT(*) FROM review_gateway_jobs GROUP BY queue_class,status`)
	if err != nil {
		return result, err
	}
	for rows.Next() {
		var key string
		var count int
		if err := rows.Scan(&key, &count); err != nil {
			_ = rows.Close()
			return result, err
		}
		result.Jobs[key] = count
	}
	_ = rows.Close()
	rows, err = s.db.QueryContext(ctx, `SELECT queue_class||':'||status,COUNT(*) FROM review_operations GROUP BY queue_class,status`)
	if err != nil {
		return result, err
	}
	for rows.Next() {
		var key string
		var count int
		if err := rows.Scan(&key, &count); err != nil {
			_ = rows.Close()
			return result, err
		}
		result.Operations[key] = count
	}
	_ = rows.Close()
	rows, err = s.db.QueryContext(ctx, `SELECT status,COUNT(*) FROM review_job_consumers GROUP BY status`)
	if err != nil {
		return result, err
	}
	for rows.Next() {
		var key string
		var count int
		if err := rows.Scan(&key, &count); err != nil {
			_ = rows.Close()
			return result, err
		}
		result.Consumers[key] = count
	}
	_ = rows.Close()
	_ = s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM review_dead_letters WHERE status='open'`).Scan(&result.OpenDeadLetters)
	_ = s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM review_operation_reconciliation_tasks WHERE status IN ('pending','checking')`).Scan(&result.PendingReconciliations)
	return result, nil
}

func runReviewOperationAdmin(runtime *common.RuntimeContext) error {
	store, err := OpenSQLiteReviewGatewayStore(runtime.Arg("state-db"))
	if err != nil {
		return err
	}
	defer store.Close()
	ctx := context.Background()
	action := strings.TrimSpace(runtime.Arg("action"))
	result := reviewOperationAdminResult{SchemaVersion: "feishu.review-operation-admin/v1", Action: action, ReadOnly: true}
	switch action {
	case "list":
		operations, err := store.ListReviewOperations(ctx, runtime.Arg("queue-class"), runtime.Arg("status"), 100)
		if err != nil {
			return err
		}
		result.Operations = reviewOperationAdminViews(operations)
	case "inspect":
		operation, err := store.GetReviewOperation(ctx, runtime.Arg("id"))
		if err != nil {
			return err
		}
		result.Operations = reviewOperationAdminViews([]ReviewOperation{operation})
	case "retry":
		result.ReadOnly = false
		if err := store.RetryReviewOperation(ctx, runtime.Arg("id"), runtime.Arg("reason"), parseBool(runtime.Arg("yes")), time.Now().UTC()); err != nil {
			return err
		}
	case "reconcile":
		result.ReadOnly = false
		if !parseBool(runtime.Arg("yes")) || strings.TrimSpace(runtime.Arg("reason")) == "" {
			return fmt.Errorf("operation reconcile requires --reason and --yes")
		}
		if err := store.RequestReviewOperationReconciliation(ctx, runtime.Arg("id"), runtime.Arg("reason"), time.Now().UTC()); err != nil {
			return err
		}
	default:
		return fmt.Errorf("unsupported review-operation action %q", action)
	}
	return json.NewEncoder(os.Stdout).Encode(result)
}

func runReviewDeadLetterAdmin(runtime *common.RuntimeContext) error {
	store, err := OpenSQLiteReviewGatewayStore(runtime.Arg("state-db"))
	if err != nil {
		return err
	}
	defer store.Close()
	ctx := context.Background()
	action := strings.TrimSpace(runtime.Arg("action"))
	result := reviewOperationAdminResult{SchemaVersion: "feishu.review-dead-letter-admin/v1", Action: action, ReadOnly: true}
	now := time.Now().UTC()
	switch action {
	case "list":
		result.DeadLetters, err = store.ListReviewDeadLetters(ctx, runtime.Arg("status"), runtime.Arg("entity-type"), 100)
	case "inspect":
		item, readErr := store.GetReviewDeadLetter(ctx, runtime.Arg("id"))
		err = readErr
		result.DeadLetters = []ReviewDeadLetter{item}
	case "retry":
		result.ReadOnly = false
		err = store.RetryReviewDeadLetter(ctx, runtime.Arg("id"), runtime.Arg("reason"), parseBool(runtime.Arg("yes")), runtime.Arg("actor"), now)
	case "resolve", "ignore":
		result.ReadOnly = false
		if !parseBool(runtime.Arg("yes")) {
			return fmt.Errorf("dead-letter resolution requires --yes")
		}
		err = store.ResolveReviewDeadLetter(ctx, runtime.Arg("id"), runtime.Arg("reason"), runtime.Arg("actor"), action == "ignore", now)
	default:
		return fmt.Errorf("unsupported review-dead-letter action %q", action)
	}
	if err != nil {
		return err
	}
	return json.NewEncoder(os.Stdout).Encode(result)
}

func runReviewOperationReconciliationAdmin(runtime *common.RuntimeContext) error {
	store, err := OpenSQLiteReviewGatewayStore(runtime.Arg("state-db"))
	if err != nil {
		return err
	}
	defer store.Close()
	ctx := context.Background()
	action := strings.TrimSpace(runtime.Arg("action"))
	result := reviewOperationAdminResult{SchemaVersion: "feishu.review-operation-reconciliation-admin/v1", Action: action, ReadOnly: true}
	switch action {
	case "list":
		result.Reconciliations, err = store.ListReviewOperationReconciliations(ctx, runtime.Arg("status"), 100)
	case "inspect":
		item, readErr := store.GetReviewOperationReconciliation(ctx, runtime.Arg("id"))
		err = readErr
		result.Reconciliations = []ReviewOperationReconciliationTask{item}
	case "resolve":
		result.ReadOnly = false
		if !parseBool(runtime.Arg("yes")) {
			return fmt.Errorf("reconciliation resolve requires --yes")
		}
		err = store.ResolveReviewOperationReconciliation(ctx, runtime.Arg("id"), runtime.Arg("reason"), runtime.Arg("actor"), parseBool(runtime.Arg("success")), time.Now().UTC())
	default:
		return fmt.Errorf("unsupported review-operation-reconciliation action %q", action)
	}
	if err != nil {
		return err
	}
	return json.NewEncoder(os.Stdout).Encode(result)
}

func reviewOperationAdminViews(operations []ReviewOperation) []reviewOperationAdminView {
	result := make([]reviewOperationAdminView, 0, len(operations))
	for _, operation := range operations {
		result = append(result, reviewOperationAdminView{OperationID: operation.OperationID, Kind: operation.OperationKind, QueueClass: operation.QueueClass, Repository: operation.Repository, PRNumber: operation.PRNumber, ChatIDHash: reviewGatewayHashIdentifier(operation.ChatID), Status: operation.Status, MutationStatus: operation.MutationStatus, RetrySafety: operation.RetrySafety, RemoteIDHash: reviewGatewayHashIdentifier(operation.RemoteID), ErrorClass: operation.ErrorClass, ErrorCode: operation.ErrorCode, ErrorSummary: operation.ErrorSummary, AttemptCount: operation.AttemptCount, RequiresReconciliation: operation.RequiresReconciliation})
	}
	return result
}

func (s *SQLiteReviewGatewayStore) RetryReviewOperation(ctx context.Context, id, reason string, confirmed bool, now time.Time) error {
	if strings.TrimSpace(reason) == "" || !confirmed {
		return fmt.Errorf("operation retry requires --reason and --yes")
	}
	operation, err := s.GetReviewOperation(ctx, id)
	if err != nil {
		return err
	}
	if operation.Status == ReviewOperationUnknown || operation.Status == ReviewOperationNeedsReconciliation || operation.MutationStatus == ReviewMutationRemoteUnknown {
		return fmt.Errorf("unknown operation must be reconciled before retry")
	}
	result, err := s.db.ExecContext(ctx, `UPDATE review_operations SET status='retry_scheduled',error_class='',error_code='',error_summary='',next_attempt_at=?,completed_at='',updated_at=? WHERE operation_id=? AND status IN ('failed_terminal','dead_letter','stale')`, reviewGatewayTimestamp(now), reviewGatewayTimestamp(now), id)
	if err != nil {
		return err
	}
	affected, _ := result.RowsAffected()
	if affected != 1 {
		return fmt.Errorf("operation %s is not retryable", id)
	}
	return nil
}

func (s *SQLiteReviewGatewayStore) RequestReviewOperationReconciliation(ctx context.Context, id, reason string, now time.Time) error {
	operation, err := s.GetReviewOperation(ctx, id)
	if err != nil {
		return err
	}
	if operation.Status != ReviewOperationUnknown && operation.Status != ReviewOperationNeedsReconciliation {
		return fmt.Errorf("operation %s does not require reconciliation", id)
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	if err := createReviewOperationReconciliationTx(ctx, tx, operation, truncateReviewGatewayText(reason, 120), now); err != nil {
		return err
	}
	return tx.Commit()
}
