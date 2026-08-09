package feishu

import (
	"context"
	"encoding/json"
	"errors"
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
	larknormalize "github.com/larksuite/oapi-sdk-go/v3/channel/normalize"
	larksafety "github.com/larksuite/oapi-sdk-go/v3/channel/safety"
	larktypes "github.com/larksuite/oapi-sdk-go/v3/channel/types"
	larkcore "github.com/larksuite/oapi-sdk-go/v3/core"
	"github.com/larksuite/oapi-sdk-go/v3/event/dispatcher"
	larkcallback "github.com/larksuite/oapi-sdk-go/v3/event/dispatcher/callback"
	larkim "github.com/larksuite/oapi-sdk-go/v3/service/im/v1"
	larkws "github.com/larksuite/oapi-sdk-go/v3/ws"

	"github.com/gitlink-org/gitlink-cli/shortcuts/common"
)

type reviewGatewayJSONOutput struct {
	mu     sync.Mutex
	writer io.Writer
	queue  chan interface{}
}

type reviewGatewayLifecycleEvent struct {
	SchemaVersion string `json:"schema_version"`
	Type          string `json:"type"`
	Message       string `json:"message,omitempty"`
	ObservedAt    string `json:"observed_at"`
}

type ReviewGatewayChatDiscovery struct {
	SchemaVersion string                        `json:"schema_version"`
	ReadOnly      bool                          `json:"read_only"`
	Chats         []ReviewGatewayDiscoveredChat `json:"chats"`
}

type ReviewGatewayDiscoveredChat struct {
	ChatID     string `json:"chat_id"`
	Name       string `json:"name,omitempty"`
	ChatMode   string `json:"chat_mode,omitempty"`
	ChatStatus string `json:"chat_status,omitempty"`
	External   bool   `json:"external"`
}

type reviewGatewayFeishuChannel struct {
	larktypes.Channel
	dispatcher *dispatcher.EventDispatcher
	policy     *larksafety.PolicyGate
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
			{Name: "discover-chats", Usage: "List groups visible to the self-built app for administrator pre-binding", Bool: true, Default: "false"},
			{Name: "execute-read-only", Usage: "In offline mode, execute the accepted GitLink GET-only job after printing its receipt", Bool: true, Default: "false"},
			{Name: "listen", Usage: "Listen through the Feishu Channel SDK persistent connection", Bool: true, Default: "false"},
			{Name: "app-id", Usage: "Feishu self-built app ID. Defaults to FEISHU_APP_ID"},
			{Name: "app-secret", Usage: "Feishu self-built app secret. Defaults to FEISHU_APP_SECRET"},
			{Name: "admin-users", Usage: "Comma-separated Feishu user IDs allowed to preview binding changes"},
			{Name: "state-db", Usage: "SQLite path for durable event dedupe and jobs in listen mode", Default: ".local/review-gateway.db"},
			{Name: "queue-size", Usage: "Maximum in-memory asynchronous job backlog", Default: "128"},
			{Name: "stale-minutes", Usage: "Reject message events older than this window", Default: "30"},
			{Name: "job-timeout-seconds", Usage: "Timeout for each GitLink GET-only job", Default: "60"},
			{Name: "handler-timeout-ms", Usage: "Total Feishu callback persistence budget", Default: "2000"},
			{Name: "sqlite-timeout-ms", Usage: "SQLite budget inside each Feishu callback", Default: "500"},
			{Name: "controlled-action-mode", Usage: "Controlled action execution mode: local or auto", Default: "local"},
		},
		Run: runReviewGateway,
	}
}

func runReviewGateway(runtime *common.RuntimeContext) error {
	fromEvent := strings.TrimSpace(runtime.Arg("from-event"))
	listen := parseBool(runtime.Arg("listen"))
	executeReadOnly := parseBool(runtime.Arg("execute-read-only"))
	discoverChats := parseBool(runtime.Arg("discover-chats"))
	controlledActionMode, err := normalizeControlledActionMode(runtime.Arg("controlled-action-mode"))
	if err != nil {
		return err
	}
	if discoverChats {
		if listen || fromEvent != "" || executeReadOnly {
			return fmt.Errorf("--discover-chats cannot be combined with --listen, --from-event, or --execute-read-only")
		}
		output := &reviewGatewayJSONOutput{writer: os.Stdout}
		return runReviewGatewayDiscoverChats(runtime, output)
	}
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
		result, executeErr := (&ReviewGatewayExecutor{Runtime: runtime, ControlledActionMode: controlledActionMode}).Execute(jobCtx, *receipt.Job)
		if emitErr := output.Emit(result); emitErr != nil {
			return emitErr
		}
		return executeErr
	}

	if enabledReviewGatewayBindingCount(bindings) == 0 {
		return fmt.Errorf("--listen requires at least one enabled binding in --bindings")
	}
	return runReviewGatewayChannel(runtime, bindings, config, output, controlledActionMode)
}

func runReviewGatewayDiscoverChats(runtime *common.RuntimeContext, output *reviewGatewayJSONOutput) error {
	appID := firstNonEmpty(runtime.Arg("app-id"), os.Getenv("FEISHU_APP_ID"))
	appSecret := firstNonEmpty(runtime.Arg("app-secret"), os.Getenv("FEISHU_APP_SECRET"))
	if appID == "" || appSecret == "" {
		return fmt.Errorf("--discover-chats requires --app-id and --app-secret or FEISHU_APP_ID and FEISHU_APP_SECRET")
	}
	client := lark.NewClient(appID, appSecret, lark.WithLogLevel(larkcore.LogLevelWarn))
	result := ReviewGatewayChatDiscovery{
		SchemaVersion: "feishu.review-chat-discovery/v1",
		ReadOnly:      true,
		Chats:         []ReviewGatewayDiscoveredChat{},
	}
	pageToken := ""
	for {
		builder := larkim.NewListChatReqBuilder().PageSize(100).Types("group")
		if pageToken != "" {
			builder.PageToken(pageToken)
		}
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		response, err := client.Im.V1.Chat.List(ctx, builder.Build())
		cancel()
		if err != nil {
			return fmt.Errorf("discover Feishu chats: %s", redactReviewGatewayError(err.Error()))
		}
		if !response.Success() {
			return fmt.Errorf("discover Feishu chats failed (code %d): %s", response.Code, redactReviewGatewayError(response.Msg))
		}
		if response.Data == nil {
			break
		}
		for _, chat := range response.Data.Items {
			if chat == nil || chat.ChatId == nil || strings.TrimSpace(*chat.ChatId) == "" {
				continue
			}
			result.Chats = append(result.Chats, ReviewGatewayDiscoveredChat{
				ChatID:     strings.TrimSpace(*chat.ChatId),
				Name:       reviewGatewayStringPointer(chat.Name),
				ChatMode:   reviewGatewayStringPointer(chat.ChatMode),
				ChatStatus: reviewGatewayStringPointer(chat.ChatStatus),
				External:   chat.External != nil && *chat.External,
			})
		}
		if response.Data.HasMore == nil || !*response.Data.HasMore || response.Data.PageToken == nil {
			break
		}
		pageToken = strings.TrimSpace(*response.Data.PageToken)
		if pageToken == "" {
			break
		}
	}
	return output.Emit(result)
}

func reviewGatewayStringPointer(value *string) string {
	if value == nil {
		return ""
	}
	return strings.TrimSpace(*value)
}

func runReviewGatewayChannel(runtime *common.RuntimeContext, bindings ReviewGatewayBindings, config ReviewGatewayConfig, output *reviewGatewayJSONOutput, controlledActionMode string) error {
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
	handlerTimeoutMS, err := boundedReviewGatewayInt(runtime.Arg("handler-timeout-ms"), 2000, 100, 2500, "handler-timeout-ms")
	if err != nil {
		return err
	}
	sqliteTimeoutMS, err := boundedReviewGatewayInt(runtime.Arg("sqlite-timeout-ms"), 500, 50, 1000, "sqlite-timeout-ms")
	if err != nil {
		return err
	}
	if sqliteTimeoutMS >= handlerTimeoutMS {
		return fmt.Errorf("--sqlite-timeout-ms must be lower than --handler-timeout-ms")
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
	liveCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	output.Start(liveCtx, queueSize)

	channel := newFeishuReviewGatewayChannel(appID, appSecret, bindings)
	replyDispatcher := NewReviewGatewayReplyDispatcher(channel, store, output, queueSize)
	executor := &ReviewGatewayExecutor{
		Runtime:       runtime,
		Collaboration: store,
		ActionPlans:   store,
		DisplayNames: ReviewFeishuDisplayNameResolver{
			Client: NewOpenAPIClient(nil), AppID: appID, AppSecret: appSecret,
		},
		ControlledActionMode: controlledActionMode,
	}
	queue := NewReviewGatewayQueue(gateway, store, queueSize, func(outcome ReviewGatewayJobOutcome) {
		output.TryEmit(outcome.Result)
		if !outcome.WillRetry {
			replyDispatcher.Wake()
		}
	})
	go replyDispatcher.Run(liveCtx)
	go queue.Run(liveCtx, func(_ context.Context, job ReviewGatewayJob) (ReviewGatewayExecutionResult, error) {
		jobCtx, cancel := context.WithTimeout(liveCtx, time.Duration(jobTimeoutSeconds)*time.Second)
		defer cancel()
		return executor.Execute(jobCtx, job)
	})

	channel.OnGatewayMessage(func(ctx context.Context, message *larktypes.NormalizedMessage) error {
		if message == nil {
			return nil
		}
		return handleReviewGatewayInbound(
			ctx,
			reviewGatewayEventFromMessage(message),
			queue,
			replyDispatcher,
			output,
			time.Duration(handlerTimeoutMS)*time.Millisecond,
			time.Duration(sqliteTimeoutMS)*time.Millisecond,
		)
	})
	channel.OnCardAction(func(ctx context.Context, action *larktypes.CardActionEvent) error {
		event, ok := reviewGatewayEventFromCardAction(action)
		if !ok {
			return nil
		}
		return handleReviewGatewayInbound(
			ctx,
			event,
			queue,
			replyDispatcher,
			output,
			time.Duration(handlerTimeoutMS)*time.Millisecond,
			time.Duration(sqliteTimeoutMS)*time.Millisecond,
		)
	})
	channel.OnReady(func() {
		message := "Feishu Channel SDK connected; inbound jobs are durable and replies are asynchronous."
		if controlledActionMode == controlledActionModeAuto {
			message += " Controlled actions may execute directly only after GitLink identity verification."
		} else {
			message += " Controlled actions require local confirmation."
		}
		output.TryEmit(reviewGatewayLifecycleEvent{
			SchemaVersion: reviewGatewaySchemaVersion,
			Type:          "ready",
			Message:       message,
			ObservedAt:    time.Now().UTC().Format(time.RFC3339),
		})
	})
	channel.OnError(func(channelErr error) {
		output.TryEmit(reviewGatewayLifecycleEvent{
			SchemaVersion: reviewGatewaySchemaVersion,
			Type:          "channel_error",
			Message:       redactReviewGatewayError(channelErr.Error()),
			ObservedAt:    time.Now().UTC().Format(time.RFC3339),
		})
	})
	defer channel.Stop(context.Background())
	if err := channel.Start(liveCtx); err != nil && !errors.Is(err, context.Canceled) {
		return fmt.Errorf("Feishu review gateway channel stopped: %s", redactReviewGatewayError(err.Error()))
	}
	return nil
}

func normalizeControlledActionMode(value string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", controlledActionModeLocal:
		return controlledActionModeLocal, nil
	case controlledActionModeAuto:
		return controlledActionModeAuto, nil
	default:
		return "", fmt.Errorf("--controlled-action-mode must be local or auto")
	}
}

func handleReviewGatewayInbound(
	ctx context.Context,
	event ReviewGatewayEvent,
	queue *ReviewGatewayQueue,
	replies *ReviewGatewayReplyDispatcher,
	output *reviewGatewayJSONOutput,
	handlerBudget time.Duration,
	sqliteBudget time.Duration,
) error {
	startedAt := time.Now()
	handlerCtx, handlerCancel := context.WithTimeout(ctx, handlerBudget)
	defer handlerCancel()
	dbCtx, dbCancel := context.WithTimeout(handlerCtx, sqliteBudget)
	receipt := queue.Enqueue(dbCtx, event)
	dbCancel()
	latencyMs := time.Since(startedAt).Milliseconds()
	receipt.HandlerLatencyMs = latencyMs
	if receipt.Job != nil {
		receipt.Job.HandlerLatencyMs = latencyMs
		queue.ObserveHandlerLatency(receipt.Job.JobID, latencyMs)
	}
	output.TryEmit(receipt)
	if !receipt.Accepted && receipt.Reason != "duplicate_event" {
		replies.TryNotice(event, formatReviewGatewayNotice(receipt.Reason))
	}
	if receipt.Reason == "state_store_failed" {
		return fmt.Errorf("review gateway callback persistence failed within %d ms", sqliteBudget.Milliseconds())
	}
	if handlerCtx.Err() != nil || time.Since(startedAt) > handlerBudget {
		return fmt.Errorf("review gateway callback exceeded %d ms budget", handlerBudget.Milliseconds())
	}
	return nil
}

func newFeishuReviewGatewayChannel(appID, appSecret string, bindings ReviewGatewayBindings) *reviewGatewayFeishuChannel {
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
	channel := larkchannel.NewChannel(apiClient, wsClient, larktypes.WithPolicyConfig(policy))
	return &reviewGatewayFeishuChannel{
		Channel:    channel,
		dispatcher: eventDispatcher,
		policy:     larksafety.NewPolicyGate(&policy, nil),
	}
}

func (c *reviewGatewayFeishuChannel) OnGatewayMessage(handler func(context.Context, *larktypes.NormalizedMessage) error) {
	if c == nil || c.dispatcher == nil || handler == nil {
		return
	}
	c.dispatcher.OnP2MessageReceiveV1(func(ctx context.Context, event *larkim.P2MessageReceiveV1) error {
		message := larknormalize.ParseMessage(event)
		if message == nil {
			return nil
		}
		bot := c.GetBotIdentity(ctx)
		if bot == nil {
			return fmt.Errorf("Feishu bot identity is unavailable")
		}
		if message.UserID == bot.OpenID {
			return nil
		}
		for i := range message.Mentions {
			mention := &message.Mentions[i]
			if mention.OpenID == bot.OpenID ||
				mention.UserID == bot.OpenID ||
				(bot.UserID != "" && mention.UserID == bot.UserID) {
				message.MentionedBot = true
				mention.IsBot = true
			}
		}
		if decision := c.policy.Evaluate(message); !decision.Allowed {
			return nil
		}
		return handler(ctx, message)
	})
}

func reviewGatewayEventFromMessage(message *larktypes.NormalizedMessage) ReviewGatewayEvent {
	if message == nil {
		return ReviewGatewayEvent{}
	}
	content := message.Content
	for _, mention := range message.Mentions {
		if mention.IsBot && strings.TrimSpace(mention.Key) != "" {
			content = strings.ReplaceAll(content, mention.Key, "")
		}
	}
	return ReviewGatewayEvent{
		EventID:      message.EventID,
		MessageID:    message.MessageID,
		EventType:    "message",
		ChatID:       message.ChatID,
		ChatType:     message.ChatType,
		UserID:       message.UserID,
		Content:      strings.TrimSpace(content),
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
	if o == nil {
		return nil
	}
	o.mu.Lock()
	defer o.mu.Unlock()
	encoder := json.NewEncoder(o.writer)
	encoder.SetEscapeHTML(false)
	return encoder.Encode(value)
}

func (o *reviewGatewayJSONOutput) Start(ctx context.Context, capacity int) {
	if o == nil || o.queue != nil {
		return
	}
	if capacity <= 0 {
		capacity = 128
	}
	o.queue = make(chan interface{}, capacity)
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case value := <-o.queue:
				_ = o.Emit(value)
			}
		}
	}()
}

func (o *reviewGatewayJSONOutput) TryEmit(value interface{}) bool {
	if o == nil {
		return false
	}
	if o.queue == nil {
		return o.Emit(value) == nil
	}
	select {
	case o.queue <- value:
		return true
	default:
		return false
	}
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

func boundedReviewGatewayInt(value string, fallback, minimum, maximum int, name string) (int, error) {
	parsed, err := positiveReviewGatewayInt(value, fallback, name)
	if err != nil {
		return 0, err
	}
	if parsed < minimum || parsed > maximum {
		return 0, fmt.Errorf("--%s must be between %d and %d", name, minimum, maximum)
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
