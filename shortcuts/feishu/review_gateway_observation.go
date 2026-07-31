package feishu

import (
	"strconv"
	"strings"
	"time"

	larktypes "github.com/larksuite/oapi-sdk-go/v3/channel/types"
	larkim "github.com/larksuite/oapi-sdk-go/v3/service/im/v1"
)

const reviewGatewayObservationSchema = "feishu.review-observation/v1"

type ReviewGatewayObservation struct {
	SchemaVersion  string `json:"schema_version"`
	Stage          string `json:"stage"`
	InstanceID     string `json:"instance_id,omitempty"`
	EventIDHash    string `json:"event_id_hash,omitempty"`
	MessageIDHash  string `json:"message_id_hash,omitempty"`
	ChatIDHash     string `json:"chat_id_hash,omitempty"`
	SenderIDHash   string `json:"sender_id_hash,omitempty"`
	ChatType       string `json:"chat_type,omitempty"`
	MentionCount   int    `json:"mention_count,omitempty"`
	MentionedBot   bool   `json:"mentioned_bot,omitempty"`
	Allowed        *bool  `json:"allowed,omitempty"`
	Reason         string `json:"reason,omitempty"`
	HandlerLatency int64  `json:"handler_latency_ms,omitempty"`
	ObservedAt     string `json:"observed_at"`
}

func newReviewGatewayObservation(stage, instanceID string) ReviewGatewayObservation {
	return ReviewGatewayObservation{
		SchemaVersion: reviewGatewayObservationSchema,
		Stage:         stage,
		InstanceID:    instanceID,
		ObservedAt:    time.Now().UTC().Format(time.RFC3339Nano),
	}
}

func reviewGatewayRawObservation(event *larkim.P2MessageReceiveV1, instanceID string) ReviewGatewayObservation {
	observation := newReviewGatewayObservation("raw", instanceID)
	if event == nil {
		return observation
	}
	if event.EventV2Base != nil && event.EventV2Base.Header != nil {
		observation.EventIDHash = reviewGatewayHashIdentifier(event.EventV2Base.Header.EventID)
	}
	if event.Event == nil {
		return observation
	}
	if event.Event.Message != nil {
		message := event.Event.Message
		observation.MessageIDHash = reviewGatewayHashIdentifier(reviewGatewayStringPointer(message.MessageId))
		observation.ChatIDHash = reviewGatewayHashIdentifier(reviewGatewayStringPointer(message.ChatId))
		observation.ChatType = reviewGatewayStringPointer(message.ChatType)
		observation.MentionCount = len(message.Mentions)
	}
	if event.Event.Sender != nil && event.Event.Sender.SenderId != nil {
		sender := event.Event.Sender.SenderId
		observation.SenderIDHash = reviewGatewayHashIdentifier(firstNonEmpty(
			reviewGatewayStringPointer(sender.OpenId),
			reviewGatewayStringPointer(sender.UserId),
			reviewGatewayStringPointer(sender.UnionId),
		))
	}
	return observation
}

func reviewGatewayNormalizedObservation(stage, instanceID string, message *larktypes.NormalizedMessage) ReviewGatewayObservation {
	observation := newReviewGatewayObservation(stage, instanceID)
	if message == nil {
		return observation
	}
	observation.EventIDHash = reviewGatewayHashIdentifier(message.EventID)
	observation.MessageIDHash = reviewGatewayHashIdentifier(message.MessageID)
	observation.ChatIDHash = reviewGatewayHashIdentifier(message.ChatID)
	observation.SenderIDHash = reviewGatewayHashIdentifier(message.UserID)
	observation.ChatType = message.ChatType
	observation.MentionCount = len(message.Mentions)
	observation.MentionedBot = message.MentionedBot
	return observation
}

func reviewGatewayReceiptObservation(instanceID string, receipt ReviewGatewayReceipt) ReviewGatewayObservation {
	observation := newReviewGatewayObservation("gateway", instanceID)
	observation.EventIDHash = reviewGatewayHashIdentifier(receipt.Event.EventID)
	observation.MessageIDHash = reviewGatewayHashIdentifier(receipt.Event.MessageID)
	observation.ChatIDHash = reviewGatewayHashIdentifier(receipt.Event.ChatID)
	observation.SenderIDHash = reviewGatewayHashIdentifier(receipt.Event.UserID)
	observation.Allowed = reviewGatewayBoolPointer(receipt.Accepted)
	observation.Reason = firstNonEmpty(receipt.Reason, "accepted")
	observation.HandlerLatency = receipt.HandlerLatencyMs
	return observation
}

func reviewGatewayBoolPointer(value bool) *bool {
	return &value
}

func reviewGatewayRawCreateTime(event *larkim.P2MessageReceiveV1) int64 {
	if event == nil || event.EventV2Base == nil || event.EventV2Base.Header == nil {
		return 0
	}
	value, _ := strconv.ParseInt(strings.TrimSpace(event.EventV2Base.Header.CreateTime), 10, 64)
	return value
}
