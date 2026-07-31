package collab

import (
	"fmt"
	"strings"
	"time"
)

const WorkItemSchema = "gitlink.review-work-item/v1"

type WorkItem struct {
	SchemaVersion       string     `json:"schema_version"`
	PRKey               string     `json:"pr_key"`
	Repository          string     `json:"repository"`
	PRNumber            int        `json:"pr_number"`
	GitLinkURL          string     `json:"gitlink_url"`
	GitLinkState        string     `json:"gitlink_state,omitempty"`
	ReviewStage         string     `json:"review_stage"`
	Decision            string     `json:"decision"`
	CollectionStatus    string     `json:"collection_status"`
	HeadSHA             string     `json:"head_sha,omitempty"`
	SourceFingerprint   string     `json:"source_fingerprint,omitempty"`
	AssignedTo          string     `json:"assigned_to,omitempty"`
	CollaborationStatus string     `json:"collaboration_status"`
	DueAt               string     `json:"due_at,omitempty"`
	Archived            bool       `json:"archived"`
	NextStep            string     `json:"next_step"`
	Evidence            []Evidence `json:"evidence,omitempty"`
	UpdatedAt           string     `json:"updated_at"`
}

type Evidence struct {
	Kind  string `json:"kind"`
	Label string `json:"label"`
	Value string `json:"value"`
}

type InboundEvent struct {
	SchemaVersion string            `json:"schema_version"`
	EventID       string            `json:"event_id"`
	Platform      string            `json:"platform"`
	TenantID      string            `json:"tenant_id,omitempty"`
	AppID         string            `json:"app_id,omitempty"`
	ChatID        string            `json:"chat_id"`
	UserID        string            `json:"user_id"`
	Conversation  string            `json:"conversation,omitempty"`
	Kind          string            `json:"kind"`
	Text          string            `json:"text,omitempty"`
	ActionKey     string            `json:"action_key,omitempty"`
	ActionValue   map[string]string `json:"action_value,omitempty"`
	ReceivedAt    string            `json:"received_at"`
}

func (item WorkItem) Validate() error {
	if item.SchemaVersion != WorkItemSchema {
		return fmt.Errorf("unsupported work item schema %q", item.SchemaVersion)
	}
	if strings.TrimSpace(item.Repository) == "" || item.PRNumber <= 0 || strings.TrimSpace(item.PRKey) == "" {
		return fmt.Errorf("work item repository, PR number, and key are required")
	}
	if strings.TrimSpace(item.CollectionStatus) == "" || strings.TrimSpace(item.ReviewStage) == "" {
		return fmt.Errorf("work item collection status and Review stage are required")
	}
	return nil
}

func NewInboundEvent(platform, eventID, chatID, userID, kind string) InboundEvent {
	return InboundEvent{
		SchemaVersion: "gitlink.collab-inbound/v1",
		EventID:       eventID,
		Platform:      platform,
		ChatID:        chatID,
		UserID:        userID,
		Kind:          kind,
		ReceivedAt:    time.Now().UTC().Format(time.RFC3339Nano),
	}
}
