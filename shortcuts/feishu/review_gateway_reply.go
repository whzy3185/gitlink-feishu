package feishu

import (
	"context"
	"fmt"
	"strings"
	"time"

	larktypes "github.com/larksuite/oapi-sdk-go/v3/channel/types"
)

type reviewGatewayMessageSender interface {
	Send(ctx context.Context, input *larktypes.SendInput) (*larktypes.SendResult, error)
}

type ReviewGatewayReplyEvent struct {
	SchemaVersion string `json:"schema_version"`
	Type          string `json:"type"`
	JobID         string `json:"job_id"`
	Status        string `json:"status"`
	MessageID     string `json:"message_id,omitempty"`
	Error         string `json:"error,omitempty"`
	ObservedAt    string `json:"observed_at"`
}

type ReviewGatewayReplyDispatcher struct {
	sender        reviewGatewayMessageSender
	store         ReviewGatewayJobStore
	output        *reviewGatewayJSONOutput
	acks          chan ReviewGatewayJob
	wake          chan struct{}
	leaseOwner    string
	leaseDuration time.Duration
	pollInterval  time.Duration
	now           func() time.Time
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
		wake:          make(chan struct{}, 1),
		leaseOwner:    fmt.Sprintf("reply-%d", time.Now().UnixNano()),
		leaseDuration: time.Minute,
		pollInterval:  time.Second,
		now:           time.Now,
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
		case <-d.wake:
			d.drainAcknowledgements(ctx)
			d.deliverPendingReplies(ctx)
		case <-ticker.C:
			d.drainAcknowledgements(ctx)
			d.deliverPendingReplies(ctx)
		}
	}
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
		event.MessageID = result.MessageID
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
		sendCtx, sendCancel := context.WithTimeout(ctx, 10*time.Second)
		sendResult, sendErr := d.sender.Send(sendCtx, &larktypes.SendInput{
			ChatID:         item.Job.ChatID,
			ReplyMessageID: item.Job.SourceMessageID,
			MsgType:        "text",
			Text:           formatReviewGatewayResultReply(item.Job, item.Result),
		})
		sendCancel()
		event := ReviewGatewayReplyEvent{
			SchemaVersion: reviewGatewaySchemaVersion,
			Type:          "result_reply",
			JobID:         item.Job.JobID,
			Status:        "sent",
			ObservedAt:    d.now().UTC().Format(time.RFC3339),
		}
		persistCtx, persistCancel := context.WithTimeout(ctx, 500*time.Millisecond)
		if sendErr != nil {
			event.Status = "failed"
			event.Error = redactReviewGatewayError(sendErr.Error())
			_, _ = d.store.MarkReplyFailed(persistCtx, item, sendErr.Error(), d.now().UTC())
		} else {
			messageID := ""
			if sendResult != nil {
				messageID = sendResult.MessageID
				event.MessageID = messageID
			}
			_ = d.store.MarkReplySent(persistCtx, item.Job.JobID, messageID, d.now().UTC())
		}
		persistCancel()
		d.output.TryEmit(event)
		if sendErr != nil {
			return
		}
	}
}

func formatReviewGatewayAcknowledgement(job ReviewGatewayJob) string {
	target := job.Repository
	if job.PRNumber > 0 {
		target = fmt.Sprintf("%s PR #%d", job.Repository, job.PRNumber)
	}
	return fmt.Sprintf(
		"已接收只读 Review 请求：%s\n任务：%s\nGitLink 写入：0\n完成后会回复本消息。",
		target,
		job.JobID,
	)
}

func formatReviewGatewayResultReply(job ReviewGatewayJob, result ReviewGatewayExecutionResult) string {
	if result.Status == "failed" {
		return truncateReviewGatewayText(fmt.Sprintf(
			"只读 Review 任务失败\n任务：%s\n目标：%s PR #%d\n错误：%s\nGitLink 写入：0",
			job.JobID,
			job.Repository,
			job.PRNumber,
			firstNonEmpty(result.Error, "unknown"),
		), 3000)
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

	lines := []string{
		fmt.Sprintf("%s PR #%d 只读 Review 已完成", job.Repository, job.PRNumber),
		fmt.Sprintf("阶段：%s；决定：%s", firstNonEmpty(result.ReviewStage, "unknown"), firstNonEmpty(result.Decision, "unknown")),
		fmt.Sprintf("数据：%s；partial=%t", firstNonEmpty(result.CollectionStatus, "unknown"), result.Partial),
		fmt.Sprintf("Review：%d；线程：%d；未解决：%d", result.ReviewCount, result.ThreadCount, result.OpenThreadCount),
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
	lines = append(lines, "GitLink 写入：0")
	return truncateReviewGatewayText(strings.Join(lines, "\n"), 3000)
}
