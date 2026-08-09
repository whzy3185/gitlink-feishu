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
		sendResult, sendErr := d.sender.Send(sendCtx, reviewGatewayResultSendInput(item.Job, item.Result))
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
		if job.Action == "unsupported_command" {
			return truncateReviewGatewayText(firstNonEmpty(result.Error, "不支持的命令，请发送“帮助”查看可用命令"), 3000)
		}
		return truncateReviewGatewayText(fmt.Sprintf(
			"只读 Review 任务失败\n任务：%s\n目标：%s PR #%d\n错误：%s\nGitLink 写入：0",
			job.JobID,
			job.Repository,
			job.PRNumber,
			firstNonEmpty(result.Error, "unknown"),
		), 3000)
	}
	lines := []string{fmt.Sprintf("%s PR #%d", job.Repository, job.PRNumber)}
	if data := result.ReviewData; data != nil {
		if data.Title != "" {
			lines = append(lines, data.Title)
		}
		lines = append(lines,
			fmt.Sprintf("作者：%s", firstNonEmpty(data.Author, "未知")),
			fmt.Sprintf("状态：%s", reviewGatewayPRStateDisplay(data.State)),
			fmt.Sprintf("目标分支：%s", firstNonEmpty(data.BaseBranch, "未知")),
			fmt.Sprintf("来源分支：%s", firstNonEmpty(data.HeadBranch, "未知")),
			fmt.Sprintf("当前版本：%s", truncateReviewGatewayText(firstNonEmpty(data.HeadSHA, "未知"), 12)),
			fmt.Sprintf("变更：%d 个文件 · +%d / -%d · %d 次提交", data.Patchset.FilesCount, data.Patchset.Additions, data.Patchset.Deletions, data.Patchset.CommitsCount),
			fmt.Sprintf("Review：%d 条", data.Summary.TotalReviews),
			fmt.Sprintf("当前结论：%s", reviewGatewayDecisionDisplay(data.Summary.ReviewStatus)),
		)
		if data.GitLinkURL != "" {
			lines = append(lines, "GitLink："+data.GitLinkURL)
		}
		if item := result.Collaboration; item != nil {
			lines = append(lines, "负责人："+firstNonEmpty(item.AssignedDisplayName, "无"))
			if item.DueAt != "" {
				lines = append(lines, "审查截止："+item.DueAt)
			}
		}
	} else {
		lines = append(lines, firstNonEmpty(result.Message, "查询已完成"))
	}
	return truncateReviewGatewayText(strings.Join(lines, "\n"), 3000)
}

func truncateReviewGatewayText(value string, max int) string {
	value = strings.TrimSpace(value)
	runes := []rune(value)
	if max > 0 && len(runes) > max {
		return string(runes[:max]) + "…"
	}
	return value
}
