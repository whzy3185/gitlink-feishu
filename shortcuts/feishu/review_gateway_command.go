package feishu

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
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
	"github.com/gitlink-org/gitlink-cli/shortcuts/workflow"
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
	dispatcher      *dispatcher.EventDispatcher
	policy          *larksafety.PolicyGate
	output          *reviewGatewayJSONOutput
	instanceID      string
	apiClient       *lark.Client
	wsClient        *larkws.Client
	botMu           sync.RWMutex
	botIdentity     *larktypes.BotIdentity
	onReadyHandlers []func()
	onErrorHandlers []func(error)
}

func newReviewGatewayShortcut() *common.Shortcut {
	return &common.Shortcut{
		Name:        "review-gateway",
		Description: "Run the durable Feishu-to-GitLink Review collaboration gateway",
		Long: "P2-P3 Review collaboration gateway. Offline mode reads one normalized Feishu event from JSON. " +
			"Listen mode uses the official Feishu Channel SDK and persistent connection. " +
			"GitLink is GET-only by default; only an identity-bound, current-head, explicitly confirmed common Review " +
			"can write when --enable-gitlink-review-write is set. Approval, rejection, comments, reviewer changes, and merge remain disabled.",
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
			{Name: "enable-gitlink-review-write", Usage: "Explicitly enable confirmed common Review writes; approved/rejected/comment/merge remain disabled", Bool: true, Default: "false"},
			{Name: "sync-feishu-resources", Usage: "Explicitly sync configured Base/Doc/Task resources from the canonical WorkItem", Bool: true, Default: "false"},
			{Name: "base-app-token", Usage: "Review Base app token. Defaults to FEISHU_REVIEW_BASE_APP_TOKEN, then FEISHU_BASE_APP_TOKEN"},
			{Name: "review-table-id", Usage: "Feishu Base table for Review WorkItems. Defaults to FEISHU_REVIEW_TABLE_ID"},
			{Name: "review-document-id", Usage: "Feishu DocX receiving Review snapshots. Defaults to FEISHU_REVIEW_DOCUMENT_ID"},
			{Name: "review-document-folder-token", Usage: "Feishu folder for one durable Review document per PR. Defaults to FEISHU_REVIEW_DOCUMENT_FOLDER_TOKEN"},
			{Name: "sync-feishu-task", Usage: "Create at most one Feishu Task for each active Review WorkItem", Bool: true, Default: "false"},
			{Name: "webhook-listen-address", Usage: "Optional local address for signed GitLink webhook ingress, for example 127.0.0.1:8787"},
			{Name: "webhook-path", Usage: "Installation-scoped GitLink webhook path prefix", Default: reviewWebhookPathPrefix},
			{Name: "allow-public-webhook-listen", Usage: "Allow webhook listener to bind a non-loopback address; use only behind TLS and a trusted reverse proxy", Bool: true, Default: "false"},
			{Name: "reconciliation-interval", Usage: "Periodic active canonical-card reconciliation interval; 0 disables it", Default: "0"},
			{Name: "enable-agent-runner", Usage: "Enable provider-neutral read-only Agent review requests", Bool: true, Default: "false"},
			{Name: "agent-endpoint", Usage: "HTTPS endpoint implementing review.agent-invocation/v1"},
			{Name: "agent-credential-ref", Usage: "Optional Agent bearer credential as env:VARIABLE"},
			{Name: "agent-task-timeout-seconds", Usage: "Timeout per Agent specialist", Default: "120"},
			{Name: "agent-max-concurrency", Usage: "Maximum parallel Agent specialists", Default: "3"},
		},
		Run: runReviewGateway,
	}
}

func runReviewGateway(runtime *common.RuntimeContext) error {
	fromEvent := strings.TrimSpace(runtime.Arg("from-event"))
	listen := parseBool(runtime.Arg("listen"))
	executeReadOnly := parseBool(runtime.Arg("execute-read-only"))
	discoverChats := parseBool(runtime.Arg("discover-chats"))
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
	bindings, err = normalizeReviewGatewayBindings(bindings)
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
	instanceLock, err := acquireReviewGatewayInstanceLock(statePath, appID, time.Now())
	if err != nil {
		return err
	}
	defer instanceLock.Release()
	store, err := OpenSQLiteReviewGatewayStore(statePath)
	if err != nil {
		return err
	}
	defer store.Close()
	if err := store.SyncReviewGatewayConfiguration(
		context.Background(),
		bindings,
		"bindings_file",
		time.Now(),
	); err != nil {
		return err
	}

	gateway, err := NewReviewGateway(bindings, config, store)
	if err != nil {
		return err
	}
	liveCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	output.Start(liveCtx, queueSize)

	channel := newFeishuReviewGatewayChannel(appID, appSecret, bindings, output, instanceLock.metadata.InstanceID)
	liveSender := &reviewGatewayLiveSender{
		client:    NewOpenAPIClient(nil),
		appID:     appID,
		appSecret: appSecret,
	}
	replyDispatcher := NewReviewGatewayReplyDispatcher(liveSender, store, output, queueSize)
	replyDispatcher.instanceID = instanceLock.metadata.InstanceID
	publisherConfig := ReviewCollaborationPublisherConfig{AppID: appID, AppSecret: appSecret}
	syncFeishuResources := parseBool(runtime.Arg("sync-feishu-resources"))
	if syncFeishuResources {
		publisherConfig.BaseAppToken = firstNonEmpty(
			runtime.Arg("base-app-token"),
			os.Getenv("FEISHU_REVIEW_BASE_APP_TOKEN"),
			os.Getenv("FEISHU_BASE_APP_TOKEN"),
		)
		publisherConfig.ReviewTableID = firstNonEmpty(runtime.Arg("review-table-id"), os.Getenv("FEISHU_REVIEW_TABLE_ID"))
		publisherConfig.DocumentID = firstNonEmpty(runtime.Arg("review-document-id"), os.Getenv("FEISHU_REVIEW_DOCUMENT_ID"))
		publisherConfig.DocumentFolderToken = firstNonEmpty(runtime.Arg("review-document-folder-token"), os.Getenv("FEISHU_REVIEW_DOCUMENT_FOLDER_TOKEN"))
		publisherConfig.EnableTask = parseBool(runtime.Arg("sync-feishu-task"))
		if publisherConfig.DocumentID != "" && publisherConfig.DocumentFolderToken != "" {
			return fmt.Errorf("configure either --review-document-id or --review-document-folder-token, not both")
		}
		if (publisherConfig.BaseAppToken == "") != (publisherConfig.ReviewTableID == "") {
			return fmt.Errorf("Base Review sync requires both --base-app-token and --review-table-id")
		}
		if publisherConfig.BaseAppToken == "" && publisherConfig.DocumentID == "" && publisherConfig.DocumentFolderToken == "" && !publisherConfig.EnableTask {
			return fmt.Errorf("--sync-feishu-resources requires at least one Base, Doc, or Task target")
		}
	}
	var agentProvider workflow.ReviewAgentProvider
	agentTaskTimeoutSeconds, err := boundedReviewGatewayInt(runtime.Arg("agent-task-timeout-seconds"), 120, 1, 900, "agent-task-timeout-seconds")
	if err != nil {
		return err
	}
	agentMaxConcurrency, err := boundedReviewGatewayInt(runtime.Arg("agent-max-concurrency"), 3, 1, 8, "agent-max-concurrency")
	if err != nil {
		return err
	}
	agentRunnerEnabled := parseBool(runtime.Arg("enable-agent-runner"))
	if agentRunnerEnabled {
		provider, providerErr := workflow.NewHTTPReviewAgentProvider(
			runtime.Arg("agent-endpoint"),
			runtime.Arg("agent-credential-ref"),
			nil,
		)
		if providerErr != nil {
			return providerErr
		}
		agentProvider = provider
	}
	executor := &ReviewGatewayExecutor{
		Runtime:            runtime,
		Collaboration:      store,
		ActionPlans:        store,
		Subscriptions:      store,
		IdentityBindings:   bindings.IdentityBindings,
		EnableGitLinkWrite: parseBool(runtime.Arg("enable-gitlink-review-write")),
		// Production resource writes are planned after completion and executed
		// by durable Operation workers. Publisher remains a compatibility-only
		// synchronous wrapper for offline tests.
		Publisher:                  nil,
		Installations:              reviewGatewayInstallationMap(bindings),
		RequireInstallationRuntime: true,
		AgentProvider:              agentProvider,
		AgentTaskTimeout:           time.Duration(agentTaskTimeoutSeconds) * time.Second,
		AgentMaxConcurrency:        agentMaxConcurrency,
	}
	queue := NewReviewGatewayQueue(gateway, store, queueSize, func(outcome ReviewGatewayJobOutcome) {
		resultObservation := newReviewGatewayObservation("result", instanceLock.metadata.InstanceID)
		resultObservation.MessageIDHash = reviewGatewayHashIdentifier(outcome.Job.SourceMessageID)
		resultObservation.ChatIDHash = reviewGatewayHashIdentifier(outcome.Job.ChatID)
		resultObservation.SenderIDHash = reviewGatewayHashIdentifier(outcome.Job.RequestedBy)
		resultObservation.Allowed = reviewGatewayBoolPointer(outcome.Err == nil)
		resultObservation.Reason = outcome.Result.Status
		output.TryEmit(resultObservation)
		if !outcome.WillRetry {
			replyDispatcher.Wake()
		}
	})
	operationClient := NewOpenAPIClient(nil)
	tokenProvider := NewReviewTenantTokenProvider(operationClient, time.Now)
	liveSender.tokenProvider = tokenProvider
	operationHandler := &ReviewOperationHandler{
		Store: store, Client: operationClient, Sender: liveSender,
		TokenProvider: tokenProvider, Config: publisherConfig, Now: time.Now,
	}
	operationPlanner := &ReviewOperationPlanner{Store: store, Config: publisherConfig, Now: time.Now}
	workerConcurrency := DefaultReviewWorkerConcurrency()
	if agentRunnerEnabled {
		workerConcurrency.Agent = 1
	}
	operationWorkers := NewReviewOperationWorkerPool(store, operationHandler, workerConcurrency, instanceLock.metadata.InstanceID)
	workerManager := &ReviewWorkerManager{Queue: queue, Concurrency: workerConcurrency, InstanceID: instanceLock.metadata.InstanceID}
	operationReconciler := &ReviewOperationReconciler{
		Store: store, Client: operationClient, TokenProvider: tokenProvider, Config: publisherConfig,
		LeaseOwner: instanceLock.metadata.InstanceID + "-operation-reconciliation", Now: time.Now,
	}
	queue.UseOperationOutbox(operationPlanner, func() {
		for _, worker := range operationWorkers {
			worker.Wake()
		}
	})
	eventProcessor := NewReviewEventInboxProcessor(store, queue, instanceLock.metadata.InstanceID+"-events")
	reconciliationInterval := time.Duration(0)
	if value := strings.TrimSpace(runtime.Arg("reconciliation-interval")); value != "" && value != "0" {
		reconciliationInterval, err = time.ParseDuration(value)
		if err != nil {
			return fmt.Errorf("parse --reconciliation-interval: %w", err)
		}
	}
	if err := validateReviewReconciliationInterval(reconciliationInterval); err != nil {
		return err
	}
	reconciliation := &ReviewReconciliationScheduler{
		Store: store, Queue: queue, Interval: reconciliationInterval, Now: time.Now,
	}
	go replyDispatcher.Run(liveCtx)
	go func() { _ = operationPlanner.Run(liveCtx, instanceLock.metadata.InstanceID, workerConcurrency.Planner) }()
	for _, worker := range operationWorkers {
		go func(current *ReviewOperationWorker) { _ = current.Run(liveCtx) }(worker)
	}
	go operationReconciler.Run(liveCtx)
	go func() {
		_ = workerManager.Run(liveCtx, func(_ context.Context, job ReviewGatewayJob) (ReviewGatewayExecutionResult, error) {
			jobCtx, cancel := context.WithTimeout(liveCtx, time.Duration(jobTimeoutSeconds)*time.Second)
			defer cancel()
			return executor.Execute(jobCtx, job)
		})
	}()
	go eventProcessor.Run(liveCtx)
	if reconciliationInterval > 0 {
		go reconciliation.Run(liveCtx)
	}
	webhookAddress := strings.TrimSpace(runtime.Arg("webhook-listen-address"))
	var webhookServer *http.Server
	var webhookListener net.Listener
	if webhookAddress != "" {
		if err := validateReviewWebhookListenAddress(
			webhookAddress,
			parseBool(runtime.Arg("allow-public-webhook-listen")),
		); err != nil {
			return err
		}
		webhookPath := strings.TrimSpace(runtime.Arg("webhook-path"))
		if webhookPath != reviewWebhookPathPrefix {
			return fmt.Errorf("--webhook-path must be %q for installation-scoped routing", reviewWebhookPathPrefix)
		}
		ingress, err := NewGitLinkWebhookIngress(bindings, store, eventProcessor)
		if err != nil {
			return err
		}
		webhookListener, err = net.Listen("tcp", webhookAddress)
		if err != nil {
			return fmt.Errorf("listen for GitLink webhooks: %w", err)
		}
		mux := http.NewServeMux()
		mux.Handle(webhookPath, ingress)
		webhookServer = &http.Server{
			Handler:           mux,
			ReadHeaderTimeout: 5 * time.Second,
			ReadTimeout:       5 * time.Second,
			WriteTimeout:      5 * time.Second,
			IdleTimeout:       30 * time.Second,
		}
		go func() {
			if serveErr := webhookServer.Serve(webhookListener); serveErr != nil && !errors.Is(serveErr, http.ErrServerClosed) {
				output.TryEmit(reviewGatewayLifecycleEvent{
					SchemaVersion: reviewGatewaySchemaVersion,
					Type:          "webhook_error",
					Message:       redactReviewGatewayError(serveErr.Error()),
					ObservedAt:    time.Now().UTC().Format(time.RFC3339),
				})
			}
		}()
		defer func() {
			shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			_ = webhookServer.Shutdown(shutdownCtx)
		}()
		output.TryEmit(reviewGatewayLifecycleEvent{
			SchemaVersion: reviewGatewaySchemaVersion,
			Type:          "webhook_ready",
			Message:       fmt.Sprintf("signed GitLink PR webhook ingress listening on %s%s", webhookAddress, webhookPath),
			ObservedAt:    time.Now().UTC().Format(time.RFC3339),
		})
	}
	output.TryEmit(reviewGatewayLifecycleEvent{
		SchemaVersion: reviewGatewaySchemaVersion,
		Type:          "preflight",
		Message: fmt.Sprintf(
			"instance=%s enabled_bindings=%d allowed_users=%d admin_users=%d state_db=%s handler_budget_ms=%d sqlite_budget_ms=%d gitlink_common_review_write=%t feishu_resource_sync=%t",
			instanceLock.metadata.InstanceID,
			countEnabledReviewGatewayBindings(bindings),
			countReviewGatewayBindingUsers(bindings, false),
			countReviewGatewayBindingUsers(bindings, true),
			instanceLock.metadata.StateDBHash,
			handlerTimeoutMS,
			sqliteTimeoutMS,
			executor.EnableGitLinkWrite,
			syncFeishuResources,
		),
		ObservedAt: time.Now().UTC().Format(time.RFC3339),
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
		boundary := "GitLink execution is GET-only by default."
		if executor.EnableGitLinkWrite {
			boundary = "Confirmed common Review write is enabled; every other GitLink mutation remains disabled."
		}
		output.TryEmit(reviewGatewayLifecycleEvent{
			SchemaVersion: reviewGatewaySchemaVersion,
			Type:          "ready",
			Message:       "Feishu Channel SDK connected; inbound jobs are durable and replies are asynchronous. " + boundary,
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

func validateReviewWebhookListenAddress(address string, allowPublic bool) error {
	host, _, err := net.SplitHostPort(strings.TrimSpace(address))
	if err != nil {
		return fmt.Errorf("invalid --webhook-listen-address: %w", err)
	}
	if allowPublic {
		return nil
	}
	host = strings.Trim(host, "[]")
	if strings.EqualFold(host, "localhost") {
		return nil
	}
	ip := net.ParseIP(host)
	if ip == nil || !ip.IsLoopback() {
		return fmt.Errorf("public webhook listen address requires --allow-public-webhook-listen")
	}
	return nil
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
	instanceID := ""
	if replies != nil {
		instanceID = replies.instanceID
	}
	output.TryEmit(reviewGatewayReceiptObservation(instanceID, receipt))
	if receipt.Accepted && receipt.Job != nil {
		replies.TryAcknowledge(*receipt.Job)
	} else if replies != nil && shouldNotifyReviewGatewayRejection(receipt) {
		replies.TryNotice(ReviewGatewayNotice{
			ChatID:          event.ChatID,
			SourceMessageID: event.MessageID,
			Reason:          receipt.Reason,
		})
	}
	if receipt.Reason == "state_store_failed" {
		return fmt.Errorf("review gateway callback persistence failed within %d ms", sqliteBudget.Milliseconds())
	}
	if handlerCtx.Err() != nil || time.Since(startedAt) > handlerBudget {
		return fmt.Errorf("review gateway callback exceeded %d ms budget", handlerBudget.Milliseconds())
	}
	return nil
}

func newFeishuReviewGatewayChannel(
	appID,
	appSecret string,
	bindings ReviewGatewayBindings,
	output *reviewGatewayJSONOutput,
	instanceID string,
) *reviewGatewayFeishuChannel {
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
		output:     output,
		instanceID: instanceID,
		apiClient:  apiClient,
		wsClient:   wsClient,
	}
}

// Start deliberately starts the underlying WebSocket client directly instead
// of channel.Channel.Start. Channel SDK v3.9.9 logs the raw bot open_id from a
// hard-coded event logger during startup, even when its configured log level is
// Warn. Fetching the identity here preserves Channel normalization and outbound
// helpers without allowing that identifier into process logs.
func (c *reviewGatewayFeishuChannel) Start(ctx context.Context) error {
	if c == nil || c.wsClient == nil || c.apiClient == nil {
		return fmt.Errorf("Feishu review gateway channel is not configured")
	}
	c.wsClient.SetOnReady(func() {
		identity, err := fetchReviewGatewayBotIdentity(ctx, c.apiClient)
		if err != nil {
			for _, handler := range c.onErrorHandlers {
				handler(fmt.Errorf("resolve Feishu bot identity: %w", err))
			}
			c.wsClient.Close()
			return
		}
		c.botMu.Lock()
		c.botIdentity = identity
		c.botMu.Unlock()
		c.policy.SetBotIdentity(identity)
		for _, handler := range c.onReadyHandlers {
			handler()
		}
	})
	c.wsClient.SetOnError(func(err error) {
		for _, handler := range c.onErrorHandlers {
			handler(err)
		}
	})
	return c.wsClient.Start(ctx)
}

func (c *reviewGatewayFeishuChannel) Stop(_ context.Context) error {
	if c != nil && c.wsClient != nil {
		c.wsClient.Close()
	}
	return nil
}

func (c *reviewGatewayFeishuChannel) OnReady(handler func()) {
	if c != nil && handler != nil {
		c.onReadyHandlers = append(c.onReadyHandlers, handler)
	}
}

func (c *reviewGatewayFeishuChannel) OnError(handler func(error)) {
	if c != nil && handler != nil {
		c.onErrorHandlers = append(c.onErrorHandlers, handler)
	}
}

func (c *reviewGatewayFeishuChannel) GetBotIdentity(_ context.Context) *larktypes.BotIdentity {
	if c == nil {
		return nil
	}
	c.botMu.RLock()
	defer c.botMu.RUnlock()
	if c.botIdentity == nil {
		return nil
	}
	copy := *c.botIdentity
	return &copy
}

func fetchReviewGatewayBotIdentity(ctx context.Context, client *lark.Client) (*larktypes.BotIdentity, error) {
	if client == nil {
		return nil, fmt.Errorf("Feishu API client is required")
	}
	response, err := client.Get(ctx, "/open-apis/bot/v3/info", nil, larkcore.AccessTokenTypeTenant)
	if err != nil {
		return nil, fmt.Errorf("bot info request failed")
	}
	if response == nil || response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("bot info returned an unexpected HTTP status")
	}
	var result struct {
		Code int `json:"code"`
		Bot  struct {
			OpenID         string `json:"open_id"`
			AppName        string `json:"app_name"`
			ActivateStatus int    `json:"activate_status"`
		} `json:"bot"`
	}
	if err := json.Unmarshal(response.RawBody, &result); err != nil {
		return nil, fmt.Errorf("decode bot info response")
	}
	if result.Code != 0 || strings.TrimSpace(result.Bot.OpenID) == "" {
		return nil, fmt.Errorf("bot info response is incomplete")
	}
	name := strings.TrimSpace(result.Bot.AppName)
	if name == "" {
		name = "bot"
	}
	return &larktypes.BotIdentity{
		OpenID:         result.Bot.OpenID,
		Name:           name,
		ActivateStatus: result.Bot.ActivateStatus,
	}, nil
}

func (c *reviewGatewayFeishuChannel) OnGatewayMessage(handler func(context.Context, *larktypes.NormalizedMessage) error) {
	if c == nil || c.dispatcher == nil || handler == nil {
		return
	}
	c.dispatcher.OnP2MessageReceiveV1(func(ctx context.Context, event *larkim.P2MessageReceiveV1) error {
		c.output.TryEmit(reviewGatewayRawObservation(event, c.instanceID))
		message := larknormalize.ParseMessage(event)
		if message == nil {
			observation := newReviewGatewayObservation("normalized", c.instanceID)
			observation.Allowed = reviewGatewayBoolPointer(false)
			observation.Reason = "normalization_failed"
			c.output.TryEmit(observation)
			return nil
		}
		c.output.TryEmit(reviewGatewayNormalizedObservation("normalized", c.instanceID, message))
		bot := c.GetBotIdentity(ctx)
		if bot == nil {
			observation := reviewGatewayNormalizedObservation("identity", c.instanceID, message)
			observation.Allowed = reviewGatewayBoolPointer(false)
			observation.Reason = "bot_identity_unavailable"
			c.output.TryEmit(observation)
			return fmt.Errorf("Feishu bot identity is unavailable")
		}
		if message.UserID == bot.OpenID {
			observation := reviewGatewayNormalizedObservation("identity", c.instanceID, message)
			observation.Allowed = reviewGatewayBoolPointer(false)
			observation.Reason = "self_message"
			c.output.TryEmit(observation)
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
		c.output.TryEmit(reviewGatewayNormalizedObservation("identity", c.instanceID, message))
		decision := c.policy.Evaluate(message)
		observation := reviewGatewayNormalizedObservation("policy", c.instanceID, message)
		observation.Allowed = reviewGatewayBoolPointer(decision.Allowed)
		observation.Reason = string(decision.Reason)
		if decision.Allowed {
			observation.Reason = "allowed"
		}
		c.output.TryEmit(observation)
		if !decision.Allowed {
			return nil
		}
		return handler(ctx, message)
	})
}

func countEnabledReviewGatewayBindings(bindings ReviewGatewayBindings) int {
	count := 0
	for _, binding := range bindings.Bindings {
		if binding.Enabled {
			count++
		}
	}
	return count
}

func countReviewGatewayBindingUsers(bindings ReviewGatewayBindings, admins bool) int {
	users := map[string]bool{}
	for _, binding := range bindings.Bindings {
		values := binding.AllowedUserIDs
		if admins {
			values = binding.AdminUserIDs
		}
		for _, value := range values {
			value = strings.TrimSpace(value)
			if value != "" {
				users[value] = true
			}
		}
	}
	return len(users)
}

func shouldNotifyReviewGatewayRejection(receipt ReviewGatewayReceipt) bool {
	if !receipt.Bound || strings.TrimSpace(receipt.Event.MessageID) == "" {
		return false
	}
	switch receipt.Reason {
	case "sender_not_allowed", "unsupported_read_only_command", "binding_requires_admin", "repository_qualification_required":
		return true
	default:
		return false
	}
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
