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

type reviewReconciliationCommandResult struct {
	SchemaVersion string                         `json:"schema_version"`
	Action        string                         `json:"action"`
	ReadOnly      bool                           `json:"read_only"`
	Cursors       []ReviewReconciliationCursor   `json:"cursors,omitempty"`
	Run           *ReviewReconciliationRunResult `json:"run,omitempty"`
}

func newReviewReconciliationShortcut() *common.Shortcut {
	return &common.Shortcut{
		Name: "review-reconciliation", Description: "List or run active canonical Review card reconciliation",
		Flags: []common.Flag{
			{Name: "action", Usage: "list or run", Required: true},
			{Name: "state-db", Usage: "SQLite Review gateway state database", Default: ".local/review-gateway.db"},
			{Name: "bindings", Usage: "Bindings JSON required by run"},
			{Name: "interval", Usage: "Time bucket interval", Default: "20m"},
			{Name: "dry-run", Usage: "List candidates without jobs or cursor changes", Bool: true, Default: "false"},
			{Name: "once", Usage: "Explicitly run one scheduling pass", Bool: true, Default: "false"},
		},
		Run: runReviewReconciliationCommand,
	}
}

func runReviewReconciliationCommand(runtime *common.RuntimeContext) error {
	store, err := OpenSQLiteReviewGatewayStore(runtime.Arg("state-db"))
	if err != nil {
		return err
	}
	defer store.Close()
	action := strings.ToLower(strings.TrimSpace(runtime.Arg("action")))
	result := reviewReconciliationCommandResult{
		SchemaVersion: "feishu.review-reconciliation-command/v1", Action: action, ReadOnly: true,
	}
	switch action {
	case "list":
		result.Cursors, err = store.ListReviewReconciliationCursors(context.Background())
	case "run":
		dryRun := parseBool(runtime.Arg("dry-run"))
		if !dryRun && !parseBool(runtime.Arg("once")) {
			return fmt.Errorf("run requires --dry-run or --once")
		}
		interval, parseErr := time.ParseDuration(runtime.Arg("interval"))
		if parseErr != nil {
			return parseErr
		}
		var queue ReviewPreparedJobEnqueuer
		if !dryRun {
			bindings, readErr := readReviewGatewayBindings(strings.TrimSpace(runtime.Arg("bindings")))
			if readErr != nil {
				return readErr
			}
			gateway, gatewayErr := NewReviewGateway(bindings, ReviewGatewayConfig{}, store)
			if gatewayErr != nil {
				return gatewayErr
			}
			queue = NewReviewGatewayQueue(gateway, store, 1, nil)
		}
		scheduler := &ReviewReconciliationScheduler{Store: store, Queue: queue, Interval: interval, Now: time.Now}
		run, runErr := scheduler.RunOnce(context.Background(), dryRun)
		result.ReadOnly, result.Run, err = dryRun, &run, runErr
	default:
		return fmt.Errorf("unsupported review-reconciliation action %q", action)
	}
	if err != nil {
		return err
	}
	return json.NewEncoder(os.Stdout).Encode(result)
}
