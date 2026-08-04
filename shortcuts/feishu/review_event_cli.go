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

type ReviewEventCommandOptions struct {
	Action         string
	EventID        string
	Status         string
	InstallationID string
	Repository     string
	EventType      string
	Limit          int
	RequestID      string
	Reason         string
	Confirmed      bool
	BindingsPath   string
}

type reviewEventCommandResult struct {
	SchemaVersion string                   `json:"schema_version"`
	Action        string                   `json:"action"`
	ReadOnly      bool                     `json:"read_only"`
	Events        []ReviewEventInboxRecord `json:"events,omitempty"`
	Event         *ReviewEventInboxRecord  `json:"event,omitempty"`
	Routes        []ReviewEventRoute       `json:"routes,omitempty"`
	Processed     bool                     `json:"processed,omitempty"`
}

func newReviewEventShortcut() *common.Shortcut {
	return &common.Shortcut{
		Name: "review-event", Description: "Inspect, replay, or process the local Review Event Inbox",
		Long: "Local Event Inbox administration. Replay never revalidates HMAC and only creates a new normalized inbox event; external writes remain zero.",
		Flags: []common.Flag{
			{Name: "action", Usage: "list, inspect, replay, or process", Required: true},
			{Name: "state-db", Usage: "SQLite Review gateway state database", Default: ".local/review-gateway.db"},
			{Name: "event-id", Usage: "Event ID for inspect or replay"},
			{Name: "status", Usage: "Optional inbox status filter"},
			{Name: "installation", Usage: "Optional Installation filter"},
			{Name: "repository", Usage: "Optional owner/repo filter"},
			{Name: "event-type", Usage: "Optional normalized event type filter"},
			{Name: "limit", Usage: "Maximum list rows", Default: "100"},
			{Name: "request-id", Usage: "Explicit idempotency key for replay"},
			{Name: "reason", Usage: "Required operator reason for replay"},
			{Name: "yes", Usage: "Explicitly confirm replay", Bool: true, Default: "false"},
			{Name: "bindings", Usage: "Bindings JSON required by process"},
		},
		Run: runReviewEventCommand,
	}
}

func runReviewEventCommand(runtime *common.RuntimeContext) error {
	store, err := OpenSQLiteReviewGatewayStore(runtime.Arg("state-db"))
	if err != nil {
		return err
	}
	defer store.Close()
	limit := 0
	_, _ = fmt.Sscanf(runtime.Arg("limit"), "%d", &limit)
	result, err := executeReviewEventCommand(context.Background(), store, ReviewEventCommandOptions{
		Action: runtime.Arg("action"), EventID: runtime.Arg("event-id"), Status: runtime.Arg("status"),
		InstallationID: runtime.Arg("installation"), Repository: runtime.Arg("repository"),
		EventType: runtime.Arg("event-type"), Limit: limit, RequestID: runtime.Arg("request-id"),
		Reason: runtime.Arg("reason"), Confirmed: parseBool(runtime.Arg("yes")), BindingsPath: runtime.Arg("bindings"),
	}, time.Now().UTC())
	if err != nil {
		return err
	}
	return json.NewEncoder(os.Stdout).Encode(result)
}

func executeReviewEventCommand(
	ctx context.Context, store *SQLiteReviewGatewayStore, options ReviewEventCommandOptions, now time.Time,
) (reviewEventCommandResult, error) {
	result := reviewEventCommandResult{
		SchemaVersion: "feishu.review-event-command/v1",
		Action:        strings.ToLower(strings.TrimSpace(options.Action)), ReadOnly: true,
	}
	switch result.Action {
	case "list":
		events, err := store.ListReviewEventInbox(ctx, ReviewEventListFilters{
			Status: options.Status, InstallationID: options.InstallationID,
			Repository: options.Repository, EventType: options.EventType, Limit: options.Limit,
		})
		result.Events = events
		return result, err
	case "inspect":
		event, err := store.GetReviewEventInbox(ctx, strings.TrimSpace(options.EventID))
		if err != nil {
			return result, err
		}
		routes, err := store.ListReviewEventRoutes(ctx, event.Event.EventID)
		result.Event, result.Routes = &event, routes
		return result, err
	case "replay":
		result.ReadOnly = false
		replay, err := store.ReplayReviewEvent(ctx, ReviewEventReplayRequest{
			EventID: options.EventID, RequestID: options.RequestID,
			Reason: options.Reason, Confirmed: options.Confirmed,
		}, now)
		result.Event = &replay.Record
		return result, err
	case "process":
		result.ReadOnly = false
		bindings, err := readReviewGatewayBindings(strings.TrimSpace(options.BindingsPath))
		if err != nil {
			return result, err
		}
		gateway, err := NewReviewGateway(bindings, ReviewGatewayConfig{Now: func() time.Time { return now }}, store)
		if err != nil {
			return result, err
		}
		queue := NewReviewGatewayQueue(gateway, store, 1, nil)
		processor := NewReviewEventInboxProcessor(store, queue, "review-event-cli")
		processor.Now = func() time.Time { return now }
		result.Processed, err = processor.ProcessOne(ctx)
		return result, err
	default:
		return result, fmt.Errorf("unsupported review-event action %q", result.Action)
	}
}
