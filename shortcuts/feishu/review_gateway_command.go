package feishu

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"time"

	lark "github.com/larksuite/oapi-sdk-go/v3"
	larkchannel "github.com/larksuite/oapi-sdk-go/v3/channel"
	larktypes "github.com/larksuite/oapi-sdk-go/v3/channel/types"
	larkcore "github.com/larksuite/oapi-sdk-go/v3/core"
	"github.com/larksuite/oapi-sdk-go/v3/event/dispatcher"
	larkcallback "github.com/larksuite/oapi-sdk-go/v3/event/dispatcher/callback"
	larkws "github.com/larksuite/oapi-sdk-go/v3/ws"

	"github.com/gitlink-org/gitlink-cli/shortcuts/common"
)

type reviewGatewayJSONOutput struct {
	mu     sync.Mutex
	writer io.Writer
}

type reviewGatewayLifecycleEvent struct {
	SchemaVersion string `json:"schema_version"`
	Type          string `json:"type"`
	Message       string `json:"message,omitempty"`
	ObservedAt    string `json:"observed_at"`
}

func newReviewGatewayShortcut() *common.Shortcut {
	return &common.Shortcut{
		Name:        "review-gateway",
		Description: "Preview Feishu-to-GitLink review collaboration through a GET-only gateway",
		Long: "P2 preview gateway. Offline mode reads one normalized Feishu event from JSON. " +
			"Listen mode uses the official Feishu Channel SDK and persistent connection. " +
			"GitLink access remains GET-only; collaboration mutations are plans only.",
		Flags: []common.Flag{
			{Name: "from-event", Usage: "Read one normalized Feishu event from a local JSON file"},
			{Name: "bindings", Usage: "Read controlled chat-to-repository bindings from JSON"},
			{Name: "execute-read-only", Usage: "In offline mode, execute the accepted GitLink GET-only job after printing its receipt", Bool: true, Default: "false"},
			{Name: "listen", Usage: "Listen through the Feishu Channel SDK persistent connection", Bool: true, Default: "false"},
			{Name: "app-id", Usage: "Feishu self-built app ID. Defaults to FEISHU_APP_ID"},
			{Name: "app-secret", Usage: "Feishu self-built app secret. Defaults to FEISHU_APP_SECRET"},
			{Name: "admin-users", Usage: "Comma-separated Feishu user IDs allowed to preview binding changes"},
			{Name: "state-db", Usage: "SQLite path for durable event dedupe and jobs in listen mode", Default: ".local/review-gateway.db"},
			{Name: "queue-size", Usage: "Maximum in-memory asynchronous job backlog", Default: "128"},
			{Name: "stale-minutes", Usage: "Reject message events older than this window", Default: "30"},
			{Name: "job-timeout-seconds", Usage: "Timeout for each GitLink GET-only job", Default: "60"},
		},
		Run: runReviewGateway,
	}
}

func runReviewGateway(runtime *common.RuntimeContext) error {
	fromEvent := strings.TrimSpace(runtime.Arg("from-event"))
	listen := parseBool(runtime.Arg("listen"))
	executeReadOnly := parseBool(runtime.Arg("execute-read-only"))
	if listen && fromEvent != "" {
		return fmt.Errorf("--listen and --from-event cannot be used together")
	}
	if listen && executeReadOnly {
		return fmt.Errorf("--execute-read-only is implicit in --listen and cannot be specified together")
	}
	if !listen && fromEvent == "" {
		return fmt.Errorf("feishu +review-gateway requires --from-event for offline preview or --listen")
	}
	bindings, err := readReviewGatewayBindings(strings.TrimSpace(runtime.Arg("bindings")))
	if err != nil {
		return err
	}
	staleMinutes, err := positiveReviewGatewayInt(runtime.Arg("stale-minutes"), 30, "stale-minutes")
	if err != nil {
		return err
	}
	config := ReviewGatewayConfig{
		AdminUserIDs: parseReviewGatewayList(runtime.Arg("admin-users")),
		StaleWindow:  time.Duration(staleMinutes) * time.Minute,
	}
	output := &reviewGatewayJSONOutput{writer: os.Stdout}

	if !listen {
		event, readErr := readReviewGatewayEvent(fromEvent)
		if readErr != nil {
			return readErr
		}
		gateway, gatewayErr := NewReviewGateway(bindings, config, nil)
		if gatewayErr != nil {
			return gatewayErr
		}
		receipt, planErr := gateway.Plan(event)
		if emitErr := output.Emit(receipt); emitErr != nil {
			return emitErr
		}
		if planErr != nil || !executeReadOnly || !receipt.Accepted || receipt.Job == nil {
			return planErr
		}
		jobTimeoutSeconds, timeoutErr := positiveReviewGatewayInt(runtime.Arg("job-timeout-seconds"), 60, "job-timeout-seconds")
		if timeoutErr != nil {
			return timeoutErr
		}
		jobCtx, cancel := context.WithTimeout(context.Background(), time.Duration(jobTimeoutSeconds)*time.Second)
		defer cancel()
		result, executeErr := (&ReviewGatewayExecutor{Runtime: runtime}).Execute(jobCtx, *receipt.Job)
		if emitErr := output.Emit(result); emitErr != nil {
			return emitErr
		}
		return executeErr
	}

	if enabledReviewGatewayBindingCount(bindings) == 0 {
		return fmt.Errorf("--listen requires at least one enabled binding in --bindings")
	}
	return runReviewGatewayChannel(runtime, bindings, config, output)
}

func runReviewGatewayChannel(runtime *common.RuntimeContext, bindings ReviewGatewayBindings, config ReviewGatewayConfig, output *reviewGatewayJSONOutput) error {
	appID := firstNonEmpty(runtime.Arg("app-id"), os.Getenv("FEISHU_APP_ID"))
	appSecret := firstNonEmpty(runtime.Arg("app-secret"), os.Getenv("FEISHU_APP_SECRET"))
	if appID == "" || appSecret == "" {
		return fmt.Errorf("--listen requires --app-id and --app-secret or FEISHU_APP_ID and FEISHU_APP_SECRET")
	}
	queueSize, err := positiveReviewGatewayInt(runtime.Arg("queue-size"), 128, "queue-size")
	if err != nil {
		return err
	}
	jobTimeoutSeconds, err := positiveReviewGatewayInt(runtime.Arg("job-timeout-seconds"), 60, "job-timeout-seconds")
	if err != nil {
		return err
	}
	statePath := firstNonEmpty(runtime.Arg("state-db"), ".local/review-gateway.db")
	store, err := OpenSQLiteReviewGatewayStore(statePath)
	if err != nil {
		return err
	}
	defer store.Close()

	gateway, err := NewReviewGateway(bindings, config, store)
	if err != nil {
		return err
	}
	executor := &ReviewGatewayExecutor{Runtime: runtime}
	queue := NewReviewGatewayQueue(gateway, store, queueSize, nil)
	liveCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	go queue.Run(liveCtx, func(_ context.Context, job ReviewGatewayJob) error {
		jobCtx, cancel := context.WithTimeout(liveCtx, time.Duration(jobTimeoutSeconds)*time.Second)
		defer cancel()
		result, executeErr := executor.Execute(jobCtx, job)
		_ = output.Emit(result)
		return executeErr
	})

	channel := newFeishuReviewGatewayChannel(appID, appSecret, bindings)
	channel.OnMessage(func(ctx context.Context, message *larktypes.NormalizedMessage) error {
		if message == nil {
			return nil
		}
		receipt := queue.Enqueue(ctx, reviewGatewayEventFromMessage(message))
		_ = output.Emit(receipt)
		return nil
	})
	channel.OnCardAction(func(ctx context.Context, action *larktypes.CardActionEvent) error {
		event, ok := reviewGatewayEventFromCardAction(action)
		if !ok {
			return nil
		}
		receipt := queue.Enqueue(ctx, event)
		_ = output.Emit(receipt)
		return nil
	})
	channel.OnReady(func() {
		_ = output.Emit(reviewGatewayLifecycleEvent{
			SchemaVersion: reviewGatewaySchemaVersion,
			Type:          "ready",
			Message:       "Feishu Channel SDK connected; GitLink execution is GET-only and output remains local preview.",
			ObservedAt:    time.Now().UTC().Format(time.RFC3339),
		})
	})
	channel.OnError(func(channelErr error) {
		_ = output.Emit(reviewGatewayLifecycleEvent{
			SchemaVersion: reviewGatewaySchemaVersion,
			Type:          "channel_error",
			Message:       redactReviewGatewayError(channelErr.Error()),
			ObservedAt:    time.Now().UTC().Format(time.RFC3339),
		})
	})
	defer channel.Stop(context.Background())
	return channel.Start(liveCtx)
}

func newFeishuReviewGatewayChannel(appID, appSecret string, bindings ReviewGatewayBindings) larktypes.Channel {
	eventDispatcher := dispatcher.NewEventDispatcher("", "")
	apiClient := lark.NewClient(appID, appSecret, lark.WithLogLevel(larkcore.LogLevelWarn))
	wsClient := larkws.NewClient(
		appID,
		appSecret,
		larkws.WithEventHandler(eventDispatcher),
		larkws.WithLogLevel(larkcore.LogLevelWarn),
	)
	requireMention := true
	respondToMentionAll := false
	policy := larktypes.PolicyConfig{
		GroupAllowlist:      enabledReviewGatewayChatIDs(bindings),
		RequireMention:      &requireMention,
		RespondToMentionAll: &respondToMentionAll,
		DMMode:              "disabled",
	}
	return larkchannel.NewChannel(apiClient, wsClient, larktypes.WithPolicyConfig(policy))
}

func reviewGatewayEventFromMessage(message *larktypes.NormalizedMessage) ReviewGatewayEvent {
	if message == nil {
		return ReviewGatewayEvent{}
	}
	return ReviewGatewayEvent{
		EventID:      message.EventID,
		MessageID:    message.MessageID,
		EventType:    "message",
		ChatID:       message.ChatID,
		ChatType:     message.ChatType,
		UserID:       message.UserID,
		Content:      message.Content,
		CreateTimeMs: message.CreateTimeMs,
	}
}

func reviewGatewayEventFromCardAction(action *larktypes.CardActionEvent) (ReviewGatewayEvent, bool) {
	if action == nil {
		return ReviewGatewayEvent{}, false
	}
	command, ok := action.Action.Value["command"].(string)
	if !ok || strings.TrimSpace(command) == "" {
		return ReviewGatewayEvent{}, false
	}
	userID := firstNonEmpty(action.Operator.OpenID, action.Operator.UserID)
	chatID := firstNonEmpty(action.ChatID, action.Context.OpenChatID)
	messageID := firstNonEmpty(action.MessageID, action.Context.OpenMessageID)
	var createTimeMs int64
	if raw, ok := action.RawEvent.(*larkcallback.CardActionTriggerEvent); ok &&
		raw.EventV2Base != nil &&
		raw.EventV2Base.Header != nil {
		createTimeMs, _ = strconv.ParseInt(strings.TrimSpace(raw.EventV2Base.Header.CreateTime), 10, 64)
	}
	return ReviewGatewayEvent{
		EventID:      action.EventID,
		MessageID:    messageID,
		EventType:    "card_action",
		ChatID:       chatID,
		UserID:       userID,
		Content:      command,
		CreateTimeMs: createTimeMs,
	}, true
}

func (o *reviewGatewayJSONOutput) Emit(value interface{}) error {
	o.mu.Lock()
	defer o.mu.Unlock()
	encoder := json.NewEncoder(o.writer)
	encoder.SetEscapeHTML(false)
	return encoder.Encode(value)
}

func positiveReviewGatewayInt(value string, fallback int, name string) (int, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return fallback, nil
	}
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed <= 0 {
		return 0, fmt.Errorf("--%s must be a positive integer", name)
	}
	return parsed, nil
}

func parseReviewGatewayList(value string) []string {
	return sortedUniqueReviewGatewayStrings(strings.Split(value, ","))
}

func enabledReviewGatewayChatIDs(bindings ReviewGatewayBindings) []string {
	result := []string{}
	for _, binding := range bindings.Bindings {
		if binding.Enabled {
			result = append(result, strings.TrimSpace(binding.ChatID))
		}
	}
	return sortedUniqueReviewGatewayStrings(result)
}

func enabledReviewGatewayBindingCount(bindings ReviewGatewayBindings) int {
	return len(enabledReviewGatewayChatIDs(bindings))
}
