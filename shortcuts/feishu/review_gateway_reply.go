package feishu

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	larktypes "github.com/larksuite/oapi-sdk-go/v3/channel/types"
)

type reviewGatewayMessageSender interface {
	Send(ctx context.Context, input *larktypes.SendInput) (*larktypes.SendResult, error)
}

type reviewGatewayMessageUpdater interface {
	UpdateInteractiveMessage(ctx context.Context, messageID string, card Card) error
}

type reviewGatewayResourceStore interface {
	GetReviewResourceState(context.Context, string, string) (ReviewResourceState, error)
	SaveReviewResourceState(context.Context, string, string, string, string, time.Time) error
}

type reviewGatewayCanonicalCardStore interface {
	GetChatPRPresentation(context.Context, ReviewGatewayJob) (ReviewChatPRPresentation, error)
	AcquireChatPRPresentationCreate(context.Context, ReviewGatewayJob, string, time.Time) (ReviewChatPRPresentation, bool, error)
	CompleteChatPRPresentationCreate(context.Context, ReviewGatewayJob, string, string, string, string, bool, time.Time) error
	AcquireChatPRPresentationPatch(context.Context, ReviewGatewayJob, string, string, string, time.Time) (ReviewChatPRPresentation, bool, error)
	CompleteChatPRPresentationPatch(context.Context, ReviewGatewayJob, string, string, string, bool, time.Time) error
	MarkChatPRPresentationUnknown(context.Context, ReviewGatewayJob, string, string, string, time.Time) error
}

type reviewGatewayLiveSender struct {
	client        OpenAPIClient
	appID         string
	appSecret     string
	tokenProvider *ReviewTenantTokenProvider
}

func (s *reviewGatewayLiveSender) tenantToken(ctx context.Context) (string, error) {
	if s.tokenProvider != nil {
		return s.tokenProvider.Token(ctx, s.appID, s.appSecret)
	}
	token, err := s.client.TenantAccessToken(ctx, s.appID, s.appSecret)
	return token.Value, err
}

func (s *reviewGatewayLiveSender) Send(
	ctx context.Context,
	input *larktypes.SendInput,
) (*larktypes.SendResult, error) {
	if input == nil {
		return nil, fmt.Errorf("Feishu send input is required")
	}
	msgType := strings.TrimSpace(input.MsgType)
	content := ""
	switch {
	case strings.TrimSpace(input.Card) != "":
		msgType = "interactive"
		content = input.Card
	case strings.TrimSpace(input.Text) != "":
		msgType = "text"
		encoded, err := json.Marshal(map[string]string{"text": input.Text})
		if err != nil {
			return nil, fmt.Errorf("encode Feishu reply text: %w", err)
		}
		content = string(encoded)
	default:
		return nil, fmt.Errorf("review gateway only sends text or interactive messages")
	}
	token, err := s.tenantToken(ctx)
	if err != nil {
		return nil, err
	}
	sent, err := s.client.SendMessage(
		ctx,
		token,
		input.ChatID,
		"chat_id",
		input.ReplyMessageID,
		msgType,
		content,
	)
	if err != nil {
		return nil, err
	}
	return &larktypes.SendResult{MessageID: sent.MessageID, ChatID: sent.ChatID}, nil
}

func (s *reviewGatewayLiveSender) UpdateInteractiveMessage(
	ctx context.Context,
	messageID string,
	card Card,
) error {
	token, err := s.tenantToken(ctx)
	if err != nil {
		return err
	}
	return s.client.PatchInteractiveMessage(ctx, token, messageID, card)
}

type ReviewGatewayReplyEvent struct {
	SchemaVersion string `json:"schema_version"`
	Type          string `json:"type"`
	JobID         string `json:"job_id"`
	Status        string `json:"status"`
	MessageIDHash string `json:"message_id_hash,omitempty"`
	Error         string `json:"error,omitempty"`
	ObservedAt    string `json:"observed_at"`
}

type ReviewGatewayNotice struct {
	ChatID          string
	SourceMessageID string
	Reason          string
}

type ReviewGatewayReplyDispatcher struct {
	sender        reviewGatewayMessageSender
	store         ReviewGatewayJobStore
	output        *reviewGatewayJSONOutput
	acks          chan ReviewGatewayJob
	notices       chan ReviewGatewayNotice
	wake          chan struct{}
	instanceID    string
	leaseOwner    string
	leaseDuration time.Duration
	pollInterval  time.Duration
	now           func() time.Time
	uncertain     sync.Map
}

func NewReviewGatewayReplyDispatcher(sender reviewGatewayMessageSender, store ReviewGatewayJobStore, output *reviewGatewayJSONOutput, capacity int) *ReviewGatewayReplyDispatcher {
	if capacity <= 0 {
		capacity = 128
	}
	return &ReviewGatewayReplyDispatcher{
		sender:        sender,
		store:         store,
		output:        output,
		acks:          make(chan ReviewGatewayJob, capacity),
		notices:       make(chan ReviewGatewayNotice, capacity),
		wake:          make(chan struct{}, 1),
		leaseOwner:    fmt.Sprintf("reply-%d", time.Now().UnixNano()),
		leaseDuration: time.Minute,
		pollInterval:  time.Second,
		now:           time.Now,
	}
}

func (d *ReviewGatewayReplyDispatcher) TryNotice(notice ReviewGatewayNotice) bool {
	if d == nil || d.sender == nil || notice.SourceMessageID == "" || notice.ChatID == "" {
		return false
	}
	select {
	case d.notices <- notice:
		return true
	default:
		return false
	}
}

func (d *ReviewGatewayReplyDispatcher) TryAcknowledge(job ReviewGatewayJob) bool {
	if d == nil || d.sender == nil || job.SourceMessageID == "" {
		return false
	}
	select {
	case d.acks <- job:
		return true
	default:
		return false
	}
}

func (d *ReviewGatewayReplyDispatcher) Wake() {
	if d == nil {
		return
	}
	select {
	case d.wake <- struct{}{}:
	default:
	}
}

func (d *ReviewGatewayReplyDispatcher) Run(ctx context.Context) {
	if d == nil || d.sender == nil || d.store == nil {
		return
	}
	ticker := time.NewTicker(d.pollInterval)
	defer ticker.Stop()
	d.Wake()
	for {
		select {
		case <-ctx.Done():
			return
		case job := <-d.acks:
			d.sendAcknowledgement(ctx, job)
		case notice := <-d.notices:
			d.sendNotice(ctx, notice)
		case <-d.wake:
			d.drainAcknowledgements(ctx)
			d.deliverPendingReplies(ctx)
		case <-ticker.C:
			d.drainAcknowledgements(ctx)
			d.deliverPendingReplies(ctx)
		}
	}
}

func (d *ReviewGatewayReplyDispatcher) sendNotice(ctx context.Context, notice ReviewGatewayNotice) {
	text := formatReviewGatewayNotice(notice.Reason)
	if text == "" {
		return
	}
	sendCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	result, err := d.sender.Send(sendCtx, &larktypes.SendInput{
		ChatID:         notice.ChatID,
		ReplyMessageID: notice.SourceMessageID,
		MsgType:        "text",
		Text:           text,
	})
	event := ReviewGatewayReplyEvent{
		SchemaVersion: reviewGatewaySchemaVersion,
		Type:          "rejection_notice",
		Status:        "sent",
		ObservedAt:    d.now().UTC().Format(time.RFC3339),
	}
	if err != nil {
		event.Status = "failed"
		event.Error = redactReviewGatewayError(err.Error())
	} else if result != nil {
		event.MessageIDHash = reviewGatewayHashIdentifier(result.MessageID)
	}
	d.output.TryEmit(event)
}

func (d *ReviewGatewayReplyDispatcher) drainAcknowledgements(ctx context.Context) {
	for {
		select {
		case job := <-d.acks:
			d.sendAcknowledgement(ctx, job)
		default:
			return
		}
	}
}

func (d *ReviewGatewayReplyDispatcher) sendAcknowledgement(ctx context.Context, job ReviewGatewayJob) {
	sendCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	result, err := d.sender.Send(sendCtx, &larktypes.SendInput{
		ChatID:         job.ChatID,
		ReplyMessageID: job.SourceMessageID,
		MsgType:        "text",
		Text:           formatReviewGatewayAcknowledgement(job),
	})
	event := ReviewGatewayReplyEvent{
		SchemaVersion: reviewGatewaySchemaVersion,
		Type:          "ack_reply",
		JobID:         job.JobID,
		Status:        "sent",
		ObservedAt:    d.now().UTC().Format(time.RFC3339),
	}
	if err != nil {
		event.Status = "failed"
		event.Error = redactReviewGatewayError(err.Error())
	} else if result != nil {
		event.MessageIDHash = reviewGatewayHashIdentifier(result.MessageID)
	}
	d.output.TryEmit(event)
}

func (d *ReviewGatewayReplyDispatcher) deliverPendingReplies(ctx context.Context) {
	for {
		claimCtx, cancel := context.WithTimeout(ctx, 500*time.Millisecond)
		pending, err := d.store.ClaimPendingReplies(claimCtx, ReviewGatewayClaimOptions{
			LeaseOwner:    d.leaseOwner,
			Now:           d.now().UTC(),
			LeaseDuration: d.leaseDuration,
			Limit:         1,
		})
		cancel()
		if err != nil || len(pending) == 0 {
			return
		}
		item := pending[0]
		if _, blocked := d.uncertain.Load(item.Job.JobID); blocked {
			persistCtx, persistCancel := context.WithTimeout(ctx, 500*time.Millisecond)
			if err := d.store.MarkReplyUnknown(
				persistCtx, item.Job.JobID, "", "reply outcome remains uncertain in this process", d.now().UTC(),
			); err == nil {
				d.uncertain.Delete(item.Job.JobID)
			}
			persistCancel()
			return
		}
		sendCtx, sendCancel := context.WithTimeout(ctx, 10*time.Second)
		sendInput := &larktypes.SendInput{
			ChatID:         item.Job.ChatID,
			ReplyMessageID: item.Job.SourceMessageID,
			MsgType:        "text",
			Text:           formatReviewGatewayResultReply(item.Job, item.Result),
		}
		cardResourceStore, hasCardResourceStore := d.store.(reviewGatewayResourceStore)
		canonicalStore, hasCanonicalStore := d.store.(reviewGatewayCanonicalCardStore)
		resultCard := item.Result.ResultCard
		if item.Result.Collaboration != nil {
			resultCard = item.Result.Collaboration.Card
		}
		if len(resultCard) == 0 && item.Result.Collaboration == nil && item.Job.PRNumber > 0 {
			resultCard = buildReviewGatewayResultCard(item.Job, item.Result, nil)
		}
		completePRCard := item.Job.PRNumber > 0 && item.Result.Status == "completed" &&
			item.Result.CollectionStatus == "complete" && !item.Result.Partial &&
			item.Result.PullRequest != nil
		interactiveCardReady := false
		if item.Result.Status != "failed" && !item.Result.Partial && len(resultCard) > 0 {
			if cardJSON, ok := safeReviewGatewayCardJSON(resultCard); ok {
				sendInput.MsgType = "interactive"
				sendInput.Text = ""
				sendInput.Card = cardJSON
				interactiveCardReady = true
			}
		}
		canonicalReady := completePRCard && interactiveCardReady && hasCanonicalStore
		archivedCard := item.Result.GitLinkState == "merged" || item.Result.GitLinkState == "closed"
		cardFingerprint := reviewGatewayCardFingerprint(resultCard)
		var sendResult *larktypes.SendResult
		var sendErr error
		canonicalMessageID := ""
		replyAction := "created"
		remoteCardWriteCompleted := false
		notifyEvent := item.Job.SourceMessageID != "" || item.Job.NotifyChat
		notifyCurrentMessage := func(text string) (*larktypes.SendResult, error) {
			return d.sender.Send(sendCtx, &larktypes.SendInput{
				ChatID:         item.Job.ChatID,
				ReplyMessageID: item.Job.SourceMessageID,
				MsgType:        "text",
				Text:           text,
			})
		}
		var canonicalState ReviewChatPRPresentation
		if canonicalReady {
			canonicalState, sendErr = canonicalStore.GetChatPRPresentation(sendCtx, item.Job)
		}
		switch {
		case sendErr != nil:
			sendErr = fmt.Errorf("load canonical PR card: %w", sendErr)
		case canonicalReady && canonicalState.PresentationKey == "":
			_, acquired, acquireErr := canonicalStore.AcquireChatPRPresentationCreate(
				sendCtx, item.Job, item.Job.JobID, d.now().UTC(),
			)
			if acquireErr != nil {
				sendErr = acquireErr
			}
			if sendErr == nil && !acquired {
				sendErr = fmt.Errorf("canonical PR card create is already owned by another worker")
			}
			if sendErr != nil {
				sendErr = fmt.Errorf("reserve canonical PR card: %w", sendErr)
				break
			}
			sendResult, sendErr = d.sender.Send(sendCtx, sendInput)
			remoteCardWriteCompleted = true
			if sendErr != nil {
				_ = canonicalStore.MarkChatPRPresentationUnknown(
					sendCtx, item.Job, item.Job.JobID, "", sendErr.Error(), d.now().UTC(),
				)
			}
			if sendErr == nil && sendResult != nil {
				canonicalMessageID = sendResult.MessageID
				if persistErr := canonicalStore.CompleteChatPRPresentationCreate(
					sendCtx, item.Job, item.Job.JobID, canonicalMessageID, cardFingerprint,
					item.Result.CompletedAt, archivedCard, d.now().UTC(),
				); persistErr != nil {
					_ = canonicalStore.MarkChatPRPresentationUnknown(
						sendCtx, item.Job, item.Job.JobID, canonicalMessageID, persistErr.Error(), d.now().UTC(),
					)
					sendErr = fmt.Errorf("canonical card created but local mapping failed: %w", persistErr)
				}
			}
		case canonicalReady && (canonicalState.RequiresReconciliation ||
			canonicalState.CardStatus == "unknown" || canonicalState.CardStatus == "needs_reconciliation"):
			canonicalMessageID = canonicalState.CanonicalMessageID
			sendErr = fmt.Errorf("canonical PR card requires reconciliation before another remote write")
			remoteCardWriteCompleted = true
		case canonicalReady && (canonicalState.CardStatus == "creating" || canonicalState.CardStatus == "patching"):
			sendErr = fmt.Errorf("canonical PR card operation is already owned by another worker")
		case canonicalReady && canonicalState.CanonicalMessageID != "" &&
			canonicalState.ContentFingerprint == cardFingerprint:
			replyAction = "unchanged"
			canonicalMessageID = canonicalState.CanonicalMessageID
			if notifyEvent {
				replyAction = "unchanged_and_notified"
				sendResult, sendErr = notifyCurrentMessage(formatReviewGatewayCanonicalNotice(item.Job, false))
			}
		case canonicalReady && canonicalState.CanonicalMessageID != "":
			replyAction = "updated"
			canonicalMessageID = canonicalState.CanonicalMessageID
			updater, ok := d.sender.(reviewGatewayMessageUpdater)
			if !ok {
				sendErr = fmt.Errorf("review gateway sender cannot update an existing interactive card")
				break
			}
			_, acquired, acquireErr := canonicalStore.AcquireChatPRPresentationPatch(
				sendCtx, item.Job, canonicalState.ContentFingerprint, item.Job.JobID,
				item.Result.CompletedAt, d.now().UTC(),
			)
			if acquireErr != nil {
				sendErr = acquireErr
			} else if !acquired {
				sendErr = fmt.Errorf("canonical PR card patch is already owned by another worker")
			}
			if sendErr != nil {
				break
			}
			sendErr = updater.UpdateInteractiveMessage(sendCtx, canonicalMessageID, resultCard)
			remoteCardWriteCompleted = true
			if sendErr != nil {
				_ = canonicalStore.MarkChatPRPresentationUnknown(
					sendCtx, item.Job, item.Job.JobID, canonicalMessageID, sendErr.Error(), d.now().UTC(),
				)
			}
			if sendErr == nil {
				if persistErr := canonicalStore.CompleteChatPRPresentationPatch(
					sendCtx, item.Job, item.Job.JobID, cardFingerprint,
					item.Result.CompletedAt, archivedCard, d.now().UTC(),
				); persistErr != nil {
					_ = canonicalStore.MarkChatPRPresentationUnknown(
						sendCtx, item.Job, item.Job.JobID, canonicalMessageID, persistErr.Error(), d.now().UTC(),
					)
					sendErr = fmt.Errorf("canonical card patched but local state failed: %w", persistErr)
					break
				}
				if hasCardResourceStore {
					_ = cardResourceStore.SaveReviewResourceState(
						sendCtx, reviewGatewayCardResourceKey(item), "feishu_card",
						canonicalMessageID, cardFingerprint, d.now().UTC(),
					)
				}
				// The canonical PATCH is now durably reflected locally. A failure
				// below belongs only to the lightweight current-message notice, so
				// retrying the reply must not PATCH the canonical card again.
				remoteCardWriteCompleted = false
				if notifyEvent {
					replyAction = "updated_and_notified"
					sendResult, sendErr = notifyCurrentMessage(formatReviewGatewayCanonicalNotice(item.Job, true))
				}
			}
		default:
			sendResult, sendErr = d.sender.Send(sendCtx, sendInput)
		}
		sendCancel()
		event := ReviewGatewayReplyEvent{
			SchemaVersion: reviewGatewaySchemaVersion,
			Type:          "result_reply",
			JobID:         item.Job.JobID,
			Status:        "sent",
			ObservedAt:    d.now().UTC().Format(time.RFC3339),
		}
		if sendErr == nil {
			event.Status = replyAction
		}
		persistCtx, persistCancel := context.WithTimeout(ctx, 500*time.Millisecond)
		if sendErr != nil && remoteCardWriteCompleted {
			event.Status = "unknown"
			event.Error = redactReviewGatewayError(sendErr.Error())
			messageID := canonicalMessageID
			if sendResult != nil && sendResult.MessageID != "" {
				messageID = sendResult.MessageID
			}
			if unknownErr := d.store.MarkReplyUnknown(
				persistCtx, item.Job.JobID, messageID, sendErr.Error(), d.now().UTC(),
			); unknownErr != nil {
				d.uncertain.Store(item.Job.JobID, struct{}{})
			}
		} else if sendErr != nil {
			event.Status = "failed"
			event.Error = redactReviewGatewayError(sendErr.Error())
			_, _ = d.store.MarkReplyFailed(persistCtx, item, sendErr.Error(), d.now().UTC())
		} else {
			messageID := ""
			if sendResult != nil {
				messageID = sendResult.MessageID
				event.MessageIDHash = reviewGatewayHashIdentifier(messageID)
			}
			if persistErr := d.store.MarkReplySent(persistCtx, item.Job.JobID, messageID, d.now().UTC()); persistErr != nil {
				event.Status = "unknown"
				event.Error = redactReviewGatewayError(fmt.Sprintf("reply sent but local state failed: %v", persistErr))
				if unknownErr := d.store.MarkReplyUnknown(
					persistCtx, item.Job.JobID, messageID, persistErr.Error(), d.now().UTC(),
				); unknownErr != nil {
					d.uncertain.Store(item.Job.JobID, struct{}{})
				}
			}
			if canonicalReady && hasCardResourceStore && canonicalMessageID != "" && replyAction == "created" {
				if resourceErr := cardResourceStore.SaveReviewResourceState(
					persistCtx,
					reviewGatewayCardResourceKey(item),
					"feishu_card",
					canonicalMessageID,
					cardFingerprint,
					d.now().UTC(),
				); resourceErr != nil {
					event.Error = redactReviewGatewayError(fmt.Sprintf("compatibility card mapping failed: %v", resourceErr))
				}
			}
		}
		persistCancel()
		d.output.TryEmit(event)
		if sendErr != nil {
			return
		}
	}
}

func reviewGatewayCardResourceKey(item ReviewGatewayPendingReply) string {
	if item.Result.Collaboration != nil && strings.TrimSpace(item.Result.Collaboration.UniqueKey) != "" {
		return item.Result.Collaboration.UniqueKey
	}
	return stableKey(
		"review-work-item",
		reviewCollaborationScopeKey(
			item.Job.InstallationID,
			item.Job.ChatID,
			item.Job.Repository,
			item.Job.PRNumber,
		),
	)
}

func formatReviewGatewayCanonicalNotice(job ReviewGatewayJob, refreshed bool) string {
	switch job.Action {
	case "claim_review":
		return fmt.Sprintf("PR #%d 领取成功。", job.PRNumber)
	case "release_review":
		return fmt.Sprintf("PR #%d 已取消领取。", job.PRNumber)
	case "set_review_deadline":
		return fmt.Sprintf("PR #%d 的审查截止时间已更新为 %s。", job.PRNumber, strings.TrimSpace(job.Argument))
	}
	if refreshed {
		return fmt.Sprintf("PR #%d 已刷新，正式卡片已更新。", job.PRNumber)
	}
	return fmt.Sprintf("PR #%d 状态未变化，正式卡片保持最新。", job.PRNumber)
}

func formatReviewGatewayAcknowledgement(job ReviewGatewayJob) string {
	switch job.Action {
	case "read_review_context", "refresh_review_context":
		return "已收到 PR 查询请求，正在获取 GitLink 最新状态。"
	case "claim_review":
		return fmt.Sprintf("正在领取 PR #%d。", job.PRNumber)
	case "release_review":
		return fmt.Sprintf("正在取消 PR #%d 的领取状态。", job.PRNumber)
	case "set_review_deadline":
		return fmt.Sprintf("正在更新 PR #%d 的审查截止时间。", job.PRNumber)
	}
	if action := reviewGatewayPrepareAction(job.Action); action != "" {
		return fmt.Sprintf("已收到“%s PR #%d”请求，正在生成操作计划。\n当前尚未修改 GitLink。", reviewActionDisplayName(action), job.PRNumber)
	}
	return "请求已收到，正在处理。\n当前尚未修改 GitLink。"
}

func formatReviewGatewayNotice(reason string) string {
	switch reason {
	case "sender_not_allowed":
		return "当前账号没有操作本群 GitLink Review 助手的权限，请联系群管理员加入允许名单。"
	case "binding_requires_admin":
		return "仓库绑定只能由已配置的管理员执行；当前请求没有修改任何绑定。"
	case "repository_qualification_required":
		return "请明确指定仓库，所有仓库均为平等作用域。示例：查看 Gitlink/gitlink-cli PR #431。"
	case "unsupported_read_only_command":
		return "暂不支持该指令。请输入“帮助”查看正式命令。PR 级命令必须包含 owner/repo，例如：查看 Gitlink/gitlink-cli PR #431。Reviewer 修改、行级评论写入和讨论解决暂未支持。"
	case "review_queue_busy", "review_rate_limited":
		return "当前群的 Review 请求较多，请稍后重试。"
	default:
		return ""
	}
}

func formatReviewGatewayResultReply(job ReviewGatewayJob, result ReviewGatewayExecutionResult) string {
	if result.Status == "failed" {
		return truncateReviewGatewayText(fmt.Sprintf(
			"PR 请求处理失败\n目标：%s PR #%d\n错误：%s\n本次操作未修改 GitLink。",
			job.Repository,
			job.PRNumber,
			firstNonEmpty(result.Error, "unknown"),
		), 3000)
	}
	if result.WriteResult != nil {
		return formatReviewWriteResultReply(*result.WriteResult)
	}
	if result.Queue != nil {
		lines := []string{
			"GitLink Review Queue 已更新",
			fmt.Sprintf("仓库：%s", job.Repository),
			fmt.Sprintf("总数：%d；高优先级：%d；中优先级：%d；低优先级：%d",
				result.Queue.TotalPRs,
				result.Queue.HighPriority,
				result.Queue.MediumPriority,
				result.Queue.LowPriority,
			),
		}
		for _, item := range result.Queue.TopItems {
			lines = append(lines, fmt.Sprintf(
				"%d. PR #%d [%s] %s",
				item.Rank,
				item.Number,
				item.Priority,
				item.Title,
			))
			if len(lines) >= 8 {
				break
			}
		}
		lines = append(lines, "GitLink 写入：0")
		return truncateReviewGatewayText(strings.Join(lines, "\n"), 3000)
	}
	if plan := result.ActionPlan; plan != nil {
		return formatReviewActionPlanReply(*plan)
	}
	if result.Collaboration != nil && (job.Action == "claim_review" || job.Action == "release_review" || job.Action == "set_review_deadline") {
		return formatReviewCollaborationReply(job, result.Collaboration.Item)
	}
	if result.PullRequest != nil {
		view := result.PullRequest
		lines := []string{
			fmt.Sprintf("%s PR #%d · %s", job.Repository, job.PRNumber, firstNonEmpty(view.Title, "无标题")),
			fmt.Sprintf("作者：%s", firstNonEmpty(view.Author, "待确认")),
			fmt.Sprintf("PR 状态：%s", reviewPRStateDisplayName(result.GitLinkState)),
			fmt.Sprintf("目标分支：%s；来源分支：%s", firstNonEmpty(view.BaseBranch, "待确认"), firstNonEmpty(view.HeadBranch, "待确认")),
			fmt.Sprintf("当前版本：%s", shortReviewGatewaySHA(result.HeadSHA)),
			fmt.Sprintf("变更：%d 个文件 · +%d / -%d · %d 次提交", view.FilesCount, view.Additions, view.Deletions, view.CommitsCount),
			fmt.Sprintf("审查阶段：%s；当前结论：%s", reviewStageDisplayName(result.ReviewStage), reviewDecisionDisplayName(result.Decision)),
			fmt.Sprintf("审查记录：%d；未解决讨论：%d", result.ReviewCount, result.OpenThreadCount),
			fmt.Sprintf("数据状态：%s；风险：%s", reviewCollectionStatusDisplayName(result.CollectionStatus, result.Partial), reviewRiskDisplayName(view.RiskLevel)),
		}
		if len(view.Reviewers) > 0 {
			reviewers := make([]string, 0, len(view.Reviewers))
			for _, reviewer := range view.Reviewers {
				reviewers = append(reviewers, reviewer.Reviewer+"="+reviewReviewerDecisionDisplayName(reviewer.Decision))
			}
			lines = append(lines, "Reviewer："+strings.Join(reviewers, "；"))
		}
		if len(view.Unknowns) > 0 {
			lines = append(lines, "待确认："+strings.Join(view.Unknowns, "；"))
		}
		if view.RecommendedNextStep != "" {
			lines = append(lines, "建议下一步："+reviewNextStepDisplayName(view.RecommendedNextStep))
		}
		if result.Collaboration != nil {
			item := result.Collaboration.Item
			lines = append(lines, fmt.Sprintf(
				"协作：%s；负责人：%s；截止：%s",
				reviewCollaborationStatusDisplayName(item.CollaborationStatus),
				reviewGatewayAssigneePlainLabel(item),
				firstNonEmpty(item.DueAt, "未设置"),
			))
		}
		if view.GitLinkURL != "" {
			lines = append(lines, "GitLink："+view.GitLinkURL)
		}
		lines = append(lines, "本次操作未修改 GitLink。")
		return truncateReviewGatewayText(strings.Join(lines, "\n"), 3000)
	}
	if result.Collaboration != nil {
		item := result.Collaboration.Item
		return truncateReviewGatewayText(fmt.Sprintf(
			"%s PR #%d 协作状态已更新\n协作状态：%s\n负责人：%s\n审查截止：%s\n审查阶段：%s\n本次操作未修改 GitLink。",
			item.Repository,
			item.PRNumber,
			reviewCollaborationStatusDisplayName(item.CollaborationStatus),
			reviewGatewayAssigneePlainLabel(item),
			firstNonEmpty(item.DueAt, "未设置"),
			reviewStageDisplayName(item.ReviewStage),
		), 3000)
	}
	if result.CollaborationItems != nil {
		lines := []string{fmt.Sprintf("我的 Review 任务：%d", len(result.CollaborationItems))}
		for _, item := range result.CollaborationItems {
			lines = append(lines, fmt.Sprintf(
				"- %s PR #%d：%s，截止 %s",
				item.Repository,
				item.PRNumber,
				reviewCollaborationStatusDisplayName(item.CollaborationStatus),
				firstNonEmpty(item.DueAt, "未设置"),
			))
			if len(lines) >= 10 {
				break
			}
		}
		lines = append(lines, "本次操作未修改 GitLink。")
		return truncateReviewGatewayText(strings.Join(lines, "\n"), 3000)
	}
	if job.Action == "help" && strings.TrimSpace(result.Message) != "" {
		return truncateReviewGatewayText(result.Message, 3000)
	}

	lines := []string{
		fmt.Sprintf("%s PR #%d 查询完成", job.Repository, job.PRNumber),
		fmt.Sprintf("审查阶段：%s；当前结论：%s", reviewStageDisplayName(result.ReviewStage), reviewDecisionDisplayName(result.Decision)),
		fmt.Sprintf("数据状态：%s", reviewCollectionStatusDisplayName(result.CollectionStatus, result.Partial)),
		fmt.Sprintf("审查记录：%d；讨论：%d；未解决：%d", result.ReviewCount, result.ThreadCount, result.OpenThreadCount),
	}
	if result.HeadSHA != "" {
		lines = append(lines, "Head："+truncateReviewGatewayText(result.HeadSHA, 12))
	}
	if result.SnapshotPlan != nil {
		lines = append(lines, fmt.Sprintf(
			"同步计划：%s；保留人工字段=%t",
			result.SnapshotPlan.Action,
			result.SnapshotPlan.PreserveManualFields,
		))
	}
	if result.Draft != nil {
		lines = append(lines,
			"Review 草稿："+result.Draft.Summary,
			"建议下一步："+firstNonEmpty(result.Draft.NextStep, "人工复核"),
		)
	}
	lines = append(lines, "本次操作未修改 GitLink。")
	return truncateReviewGatewayText(strings.Join(lines, "\n"), 3000)
}

func formatReviewWriteResultReply(write ReviewWriteResult) string {
	action := write.Action
	if action == "" {
		action, _ = normalizeReviewAction("", write.ReviewStatus)
	}
	status := strings.ToLower(strings.TrimSpace(write.Status))
	if write.MutationStatus == reviewMutationPossible {
		return truncateReviewGatewayText(strings.Join([]string{
			fmt.Sprintf("%s PR #%d 操作结果暂无法确认", write.Repository, write.PRNumber),
			"操作：" + reviewActionDisplayName(action),
			"系统已停止自动重试，以避免重复写入。",
			"请先核对 GitLink 当前状态。",
		}, "\n"), 3000)
	}
	switch status {
	case "unknown", "unknown_needs_reconciliation":
		return truncateReviewGatewayText(strings.Join([]string{
			fmt.Sprintf("%s PR #%d 操作结果暂无法确认", write.Repository, write.PRNumber),
			"操作：" + reviewActionDisplayName(action),
			"系统已停止自动重试，以避免重复写入。",
			"请先核对 GitLink 当前状态。",
		}, "\n"), 3000)
	case "stale":
		return truncateReviewGatewayText(strings.Join([]string{
			fmt.Sprintf("%s PR #%d 操作计划已失效", write.Repository, write.PRNumber),
			"PR 的代码版本已经发生变化。",
			"本次未修改 GitLink。",
			"请刷新 PR 后重新发起操作。",
		}, "\n"), 3000)
	case "cancelled":
		return fmt.Sprintf("%s PR #%d 操作已取消\n本次未修改 GitLink。", write.Repository, write.PRNumber)
	case "failed":
		return truncateReviewGatewayText(strings.Join([]string{
			fmt.Sprintf("%s PR #%d 操作执行失败", write.Repository, write.PRNumber),
			"操作：" + reviewActionDisplayName(action),
			"结果：执行失败",
			"本次未确认 GitLink 写入。",
			"请刷新 PR 状态并检查失败原因后重新发起操作。",
		}, "\n"), 3000)
	case "write_disabled":
		return truncateReviewGatewayText(strings.Join([]string{
			fmt.Sprintf("%s PR #%d 操作未执行", write.Repository, write.PRNumber),
			"操作：" + reviewActionDisplayName(action),
			"当前 Gateway 未启用 GitLink 写操作。",
			"本次未修改 GitLink。",
		}, "\n"), 3000)
	case "dry_run":
		return truncateReviewGatewayText(strings.Join([]string{
			fmt.Sprintf("%s PR #%d 试运行完成", write.Repository, write.PRNumber),
			"操作：" + reviewActionDisplayName(action),
			"结果：校验完成",
			"GitLink 写入：0",
		}, "\n"), 3000)
	case "completed", "verified":
		// Only explicit terminal success states may reach the success reply.
	default:
		return truncateReviewGatewayText(strings.Join([]string{
			fmt.Sprintf("%s PR #%d 操作结果暂无法确认", write.Repository, write.PRNumber),
			"操作：" + reviewActionDisplayName(action),
			"系统已停止自动重试，以避免重复写入。",
			"请先核对 GitLink 当前状态。",
		}, "\n"), 3000)
	}
	lines := []string{
		fmt.Sprintf("%s %s", write.Repository, reviewActionCompletedTitle(action, write.PRNumber)),
		"操作：" + reviewActionDisplayName(action),
	}
	if action == reviewActionReject {
		lines = append(lines, "PR 状态：保持开放")
	} else if action == reviewActionRejectClose {
		lines = append(lines, "PR 状态：已关闭")
	} else if action == reviewActionMerge {
		lines = append(lines, "PR 状态：已合并")
	}
	lines = append(lines, "结果：GitLink 写入已完成", "远程回读：验证通过")
	if strings.TrimSpace(write.ReviewID) != "" {
		lines = append(lines, "审查记录：#"+write.ReviewID)
	}
	return truncateReviewGatewayText(strings.Join(lines, "\n"), 3000)
}

func formatReviewActionPlanReply(plan ReviewActionPlan) string {
	switch reviewActionPlanPresentationState(plan, nil) {
	case "completed":
		return fmt.Sprintf("%s %s\n结果：操作已完成并通过 GitLink 回读验证", plan.Repository, reviewActionCompletedTitle(plan.Action, plan.PRNumber))
	case "stale":
		return fmt.Sprintf("%s PR #%d 操作计划已失效\nPR 的代码版本已经发生变化。\n本次未修改 GitLink。\n请刷新 PR 后重新发起操作。", plan.Repository, plan.PRNumber)
	case "cancelled":
		return fmt.Sprintf("%s PR #%d 操作已取消\n本次未修改 GitLink。", plan.Repository, plan.PRNumber)
	case "unknown":
		return fmt.Sprintf("%s PR #%d 操作结果暂无法确认\n系统已停止自动重试，以避免重复写入。\n请先核对 GitLink 当前状态。", plan.Repository, plan.PRNumber)
	case "failed":
		return fmt.Sprintf("%s PR #%d 操作执行失败\n本次未确认 GitLink 写入。\n请刷新 PR 状态并检查失败原因后重新发起操作。", plan.Repository, plan.PRNumber)
	default:
		return truncateReviewGatewayText(strings.Join([]string{
			fmt.Sprintf("%s PR #%d “%s”操作计划已生成", plan.Repository, plan.PRNumber, reviewActionDisplayName(plan.Action)),
			"操作编号：" + plan.RequestID,
			"GitLink 身份：" + firstNonEmpty(plan.GitLinkLogin, "待确认"),
			"基于版本：" + shortReviewGatewaySHA(plan.ExpectedHeadSHA),
			"状态：待本地确认",
			"操作计划已生成，尚未修改 GitLink。",
			"最终执行需要使用绑定的 GitLink 身份完成本地确认。",
		}, "\n"), 3000)
	}
}

func formatReviewCollaborationReply(job ReviewGatewayJob, item ReviewCollaborationItem) string {
	switch job.Action {
	case "claim_review":
		return fmt.Sprintf("%s PR #%d 领取成功\n负责人：%s\n协作状态：%s\n本次操作未修改 GitLink。", item.Repository, item.PRNumber, reviewGatewayAssigneePlainLabel(item), reviewCollaborationStatusDisplayName(item.CollaborationStatus))
	case "release_review":
		return fmt.Sprintf("%s PR #%d 已取消领取\n负责人：未领取\n审查截止：未设置\n协作状态：未领取\n本次操作未修改 GitLink。", item.Repository, item.PRNumber)
	case "set_review_deadline":
		return fmt.Sprintf("%s PR #%d 的审查截止时间已更新\n审查截止：%s\n本次操作未修改 GitLink。", item.Repository, item.PRNumber, firstNonEmpty(item.DueAt, "未设置"))
	default:
		return "协作状态已更新。\n本次操作未修改 GitLink。"
	}
}
