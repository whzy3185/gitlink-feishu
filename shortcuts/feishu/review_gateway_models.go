package feishu

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gitlink-org/gitlink-cli/shortcuts/workflow"
)

const (
	reviewGatewaySchemaVersion = "feishu.review-gateway/v1"
	reviewGatewayJobSchema     = "feishu.review-job/v1"
	reviewGatewayBindingSchema = "feishu.review-bindings/v1"
)

type ReviewGatewayEvent struct {
	EventID      string `json:"event_id,omitempty"`
	MessageID    string `json:"message_id,omitempty"`
	EventType    string `json:"event_type"`
	ChatID       string `json:"chat_id"`
	ChatType     string `json:"chat_type,omitempty"`
	UserID       string `json:"user_id"`
	Content      string `json:"content"`
	CreateTimeMs int64  `json:"create_time_ms,omitempty"`
}

type ReviewChatBinding struct {
	ChatID          string   `json:"chat_id"`
	Repository      string   `json:"repository"`
	Enabled         bool     `json:"enabled"`
	AdminUserIDs    []string `json:"admin_user_ids,omitempty"`
	AllowedUserIDs  []string `json:"allowed_user_ids,omitempty"`
	DeadlineHours   int      `json:"deadline_hours,omitempty"`
	BindingRevision string   `json:"binding_revision,omitempty"`
}

type ReviewGatewayBindings struct {
	SchemaVersion string              `json:"schema_version"`
	Bindings      []ReviewChatBinding `json:"bindings"`
}

type ReviewGatewayIntent struct {
	Name       string `json:"name"`
	Repository string `json:"repository,omitempty"`
	PRNumber   int    `json:"pr_number,omitempty"`
}

type ReviewGatewayJob struct {
	SchemaVersion         string `json:"schema_version"`
	JobID                 string `json:"job_id"`
	DedupeKey             string `json:"dedupe_key"`
	Status                string `json:"status"`
	Mode                  string `json:"mode"`
	Action                string `json:"action"`
	Repository            string `json:"repository,omitempty"`
	PRNumber              int    `json:"pr_number,omitempty"`
	ChatID                string `json:"chat_id"`
	RequestedBy           string `json:"requested_by"`
	SourceEventID         string `json:"source_event_id,omitempty"`
	SourceMessageID       string `json:"source_message_id,omitempty"`
	CreatedAt             string `json:"created_at"`
	MutatesGitLink        bool   `json:"mutates_gitlink"`
	CollaborationMutation bool   `json:"collaboration_mutation"`
	RequiresAdmin         bool   `json:"requires_admin"`
}

type ReviewGatewayEventRef struct {
	EventID      string `json:"event_id,omitempty"`
	MessageID    string `json:"message_id,omitempty"`
	EventType    string `json:"event_type"`
	ChatID       string `json:"chat_id"`
	UserID       string `json:"user_id"`
	CreateTimeMs int64  `json:"create_time_ms,omitempty"`
}

type ReviewGatewayReceipt struct {
	SchemaVersion string                `json:"schema_version"`
	Mode          string                `json:"mode"`
	Accepted      bool                  `json:"accepted"`
	Duplicate     bool                  `json:"duplicate"`
	Stale         bool                  `json:"stale"`
	Bound         bool                  `json:"bound"`
	Reason        string                `json:"reason,omitempty"`
	DedupeKey     string                `json:"dedupe_key,omitempty"`
	Event         ReviewGatewayEventRef `json:"event"`
	Binding       *ReviewChatBinding    `json:"binding,omitempty"`
	Intent        ReviewGatewayIntent   `json:"intent"`
	Job           *ReviewGatewayJob     `json:"job,omitempty"`
}

type ReviewSnapshotState struct {
	SourceFingerprint string `json:"source_fingerprint,omitempty"`
	HeadSHA           string `json:"head_sha,omitempty"`
	Complete          bool   `json:"complete"`
	Archived          bool   `json:"archived"`
}

type ReviewSnapshotPlan struct {
	Action                string `json:"action"`
	Reason                string `json:"reason"`
	OverwriteGitLinkFacts bool   `json:"overwrite_gitlink_facts"`
	PreserveManualFields  bool   `json:"preserve_manual_fields"`
	Archive               bool   `json:"archive"`
}

type ReviewGatewayConfig struct {
	AdminUserIDs []string
	StaleWindow  time.Duration
	Now          func() time.Time
}

type reviewGatewayDedupeEntry struct {
	ExpiresAt time.Time
}

type ReviewGatewayDeduper interface {
	Reserve(key string, now time.Time, ttl time.Duration) (bool, error)
	Release(key string)
}

type MemoryReviewGatewayDeduper struct {
	mu      sync.Mutex
	entries map[string]reviewGatewayDedupeEntry
}

type ReviewGateway struct {
	bindings map[string]ReviewChatBinding
	admins   map[string]bool
	deduper  ReviewGatewayDeduper
	now      func() time.Time
	stale    time.Duration
}

var (
	reviewGatewayRepositoryPattern = regexp.MustCompile(`^[A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+$`)
	reviewGatewayPRPattern         = regexp.MustCompile(`(?i)^查看\s*PR\s*#?(\d+)$`)
	reviewGatewayClaimPattern      = regexp.MustCompile(`(?i)^领取\s*PR\s*#?(\d+)$`)
	reviewGatewayDraftPattern      = regexp.MustCompile(`(?i)^生成\s*PR\s*#?(\d+)\s*Review\s*草稿$`)
	reviewGatewayRefreshPattern    = regexp.MustCompile(`(?i)^刷新\s*PR\s*#?(\d+)$`)
	reviewGatewayBindPattern       = regexp.MustCompile(`(?i)^绑定仓库\s+([A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+)$`)
	reviewGatewayIntentPatterns    = []struct {
		pattern *regexp.Regexp
		name    string
	}{
		{pattern: reviewGatewayBindPattern, name: "plan_bind_repository"},
		{pattern: reviewGatewayPRPattern, name: "read_review_context"},
		{pattern: reviewGatewayClaimPattern, name: "plan_claim_review"},
		{pattern: reviewGatewayDraftPattern, name: "generate_review_draft"},
		{pattern: reviewGatewayRefreshPattern, name: "refresh_review_context"},
	}
)

func NewMemoryReviewGatewayDeduper() *MemoryReviewGatewayDeduper {
	return &MemoryReviewGatewayDeduper{entries: map[string]reviewGatewayDedupeEntry{}}
}

func (d *MemoryReviewGatewayDeduper) Reserve(key string, now time.Time, ttl time.Duration) (bool, error) {
	if strings.TrimSpace(key) == "" {
		return false, fmt.Errorf("review gateway dedupe key is required")
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.entries == nil {
		d.entries = map[string]reviewGatewayDedupeEntry{}
	}
	for existing, entry := range d.entries {
		if !entry.ExpiresAt.After(now) {
			delete(d.entries, existing)
		}
	}
	if entry, ok := d.entries[key]; ok && entry.ExpiresAt.After(now) {
		return false, nil
	}
	d.entries[key] = reviewGatewayDedupeEntry{ExpiresAt: now.Add(ttl)}
	return true, nil
}

func (d *MemoryReviewGatewayDeduper) Release(key string) {
	d.mu.Lock()
	defer d.mu.Unlock()
	delete(d.entries, key)
}

func NewReviewGateway(bindings ReviewGatewayBindings, config ReviewGatewayConfig, deduper ReviewGatewayDeduper) (*ReviewGateway, error) {
	if bindings.SchemaVersion != "" && bindings.SchemaVersion != reviewGatewayBindingSchema {
		return nil, fmt.Errorf(
			"unsupported review gateway bindings schema %q, want %q",
			bindings.SchemaVersion,
			reviewGatewayBindingSchema,
		)
	}
	if config.Now == nil {
		config.Now = time.Now
	}
	if config.StaleWindow <= 0 {
		config.StaleWindow = 30 * time.Minute
	}
	if deduper == nil {
		deduper = NewMemoryReviewGatewayDeduper()
	}
	result := &ReviewGateway{
		bindings: map[string]ReviewChatBinding{},
		admins:   stringSet(config.AdminUserIDs),
		deduper:  deduper,
		now:      config.Now,
		stale:    config.StaleWindow,
	}
	for _, binding := range bindings.Bindings {
		binding.ChatID = strings.TrimSpace(binding.ChatID)
		binding.Repository = strings.TrimSpace(binding.Repository)
		if binding.ChatID == "" {
			return nil, fmt.Errorf("review gateway binding chat_id is required")
		}
		if !reviewGatewayRepositoryPattern.MatchString(binding.Repository) {
			return nil, fmt.Errorf("review gateway binding repository %q must use owner/repo", binding.Repository)
		}
		if _, exists := result.bindings[binding.ChatID]; exists {
			return nil, fmt.Errorf("duplicate review gateway binding for chat %q", binding.ChatID)
		}
		binding.AdminUserIDs = sortedUniqueReviewGatewayStrings(binding.AdminUserIDs)
		binding.AllowedUserIDs = sortedUniqueReviewGatewayStrings(binding.AllowedUserIDs)
		result.bindings[binding.ChatID] = binding
	}
	return result, nil
}

func (g *ReviewGateway) Plan(event ReviewGatewayEvent) (ReviewGatewayReceipt, error) {
	now := g.now().UTC()
	event = normalizeReviewGatewayEvent(event)
	receipt := ReviewGatewayReceipt{
		SchemaVersion: reviewGatewaySchemaVersion,
		Mode:          "preview",
		Event: ReviewGatewayEventRef{
			EventID:      event.EventID,
			MessageID:    event.MessageID,
			EventType:    event.EventType,
			ChatID:       event.ChatID,
			UserID:       event.UserID,
			CreateTimeMs: event.CreateTimeMs,
		},
	}
	if event.ChatID == "" || event.UserID == "" {
		receipt.Reason = "invalid_event_identity"
		return receipt, nil
	}
	if event.EventType == "" {
		receipt.Reason = "event_type_required"
		return receipt, nil
	}
	if event.EventType != "message" && event.EventType != "card_action" {
		receipt.Reason = "unsupported_event_type"
		return receipt, nil
	}
	if event.CreateTimeMs > 0 && now.Sub(time.UnixMilli(event.CreateTimeMs)) > g.stale {
		receipt.Stale = true
		receipt.Reason = "stale_event"
		return receipt, nil
	}

	intent := parseReviewGatewayIntent(event.Content)
	receipt.Intent = intent
	if intent.Name == "unknown" {
		receipt.Reason = "unsupported_read_only_command"
		return receipt, nil
	}

	binding, bound := g.bindings[event.ChatID]
	receipt.Bound = bound && binding.Enabled
	if bound {
		copy := binding
		receipt.Binding = &copy
	}
	if intent.Name != "help" && intent.Name != "plan_bind_repository" {
		if !bound || !binding.Enabled {
			receipt.Reason = "chat_not_bound"
			return receipt, nil
		}
		if len(binding.AllowedUserIDs) > 0 && !containsReviewGatewayString(binding.AllowedUserIDs, event.UserID) && !g.isAdmin(binding, event.UserID) {
			receipt.Reason = "sender_not_allowed"
			return receipt, nil
		}
		intent.Repository = binding.Repository
		receipt.Intent = intent
	}
	if intent.Name == "plan_bind_repository" && !g.isAdmin(binding, event.UserID) {
		receipt.Reason = "binding_requires_admin"
		return receipt, nil
	}

	dedupeKey := reviewGatewayDedupeKey(event)
	receipt.DedupeKey = dedupeKey
	reserved, err := g.deduper.Reserve(dedupeKey, now, 24*time.Hour)
	if err != nil {
		receipt.Reason = "state_store_failed"
		return receipt, err
	}
	if !reserved {
		receipt.Duplicate = true
		receipt.Reason = "duplicate_event"
		return receipt, nil
	}

	job := newReviewGatewayJob(event, intent, dedupeKey, now)
	receipt.Accepted = true
	receipt.Reason = "queued_preview"
	receipt.Job = &job
	return receipt, nil
}

func (g *ReviewGateway) Release(receipt ReviewGatewayReceipt) {
	if receipt.DedupeKey != "" {
		g.deduper.Release(receipt.DedupeKey)
	}
}

func (g *ReviewGateway) isAdmin(binding ReviewChatBinding, userID string) bool {
	return g.admins[userID] || containsReviewGatewayString(binding.AdminUserIDs, userID)
}

func parseReviewGatewayIntent(content string) ReviewGatewayIntent {
	content = strings.Join(strings.Fields(strings.TrimSpace(content)), " ")
	switch strings.ToLower(content) {
	case "帮助", "help", "/help":
		return ReviewGatewayIntent{Name: "help"}
	case "查看绑定", "show binding":
		return ReviewGatewayIntent{Name: "show_binding"}
	case "查看待 review", "查看待审查", "review queue":
		return ReviewGatewayIntent{Name: "read_review_queue"}
	case "查看我的 review 任务", "我的 review 任务", "my review tasks":
		return ReviewGatewayIntent{Name: "read_my_review_tasks"}
	}
	for _, rule := range reviewGatewayIntentPatterns {
		match := rule.pattern.FindStringSubmatch(content)
		if len(match) != 2 {
			continue
		}
		intent := ReviewGatewayIntent{Name: rule.name}
		if rule.name == "plan_bind_repository" {
			intent.Repository = match[1]
			return intent
		}
		intent.PRNumber, _ = strconv.Atoi(match[1])
		return intent
	}
	return ReviewGatewayIntent{Name: "unknown"}
}

func newReviewGatewayJob(event ReviewGatewayEvent, intent ReviewGatewayIntent, dedupeKey string, now time.Time) ReviewGatewayJob {
	collaborationMutation := intent.Name == "plan_bind_repository" || intent.Name == "plan_claim_review"
	requiresAdmin := intent.Name == "plan_bind_repository"
	jobSeed := strings.Join([]string{dedupeKey, intent.Name, intent.Repository, strconv.Itoa(intent.PRNumber)}, "\x00")
	digest := sha256.Sum256([]byte(jobSeed))
	return ReviewGatewayJob{
		SchemaVersion:         reviewGatewayJobSchema,
		JobID:                 "job-" + hex.EncodeToString(digest[:8]),
		DedupeKey:             dedupeKey,
		Status:                "queued",
		Mode:                  "preview",
		Action:                intent.Name,
		Repository:            intent.Repository,
		PRNumber:              intent.PRNumber,
		ChatID:                event.ChatID,
		RequestedBy:           event.UserID,
		SourceEventID:         event.EventID,
		SourceMessageID:       event.MessageID,
		CreatedAt:             now.Format(time.RFC3339),
		MutatesGitLink:        false,
		CollaborationMutation: collaborationMutation,
		RequiresAdmin:         requiresAdmin,
	}
}

func PlanReviewSnapshotSync(current workflow.ReviewContext, previous *ReviewSnapshotState) ReviewSnapshotPlan {
	state := strings.ToLower(strings.TrimSpace(current.WorkItem.GitLinkState))
	if state == "merged" || state == "closed" {
		return ReviewSnapshotPlan{
			Action:                "archive",
			Reason:                "gitlink_pr_" + state,
			OverwriteGitLinkFacts: false,
			PreserveManualFields:  true,
			Archive:               true,
		}
	}
	if current.Partial || current.CollectionStatus != "complete" || !current.WorkItem.SourceScope.Complete {
		return ReviewSnapshotPlan{
			Action:                "preserve_previous",
			Reason:                "partial_or_incomplete_snapshot",
			OverwriteGitLinkFacts: false,
			PreserveManualFields:  true,
		}
	}
	if previous != nil && previous.Complete &&
		previous.SourceFingerprint == current.WorkItem.SourceFingerprint &&
		previous.HeadSHA == current.CurrentHeadSHA {
		return ReviewSnapshotPlan{
			Action:                "unchanged",
			Reason:                "source_fingerprint_unchanged",
			OverwriteGitLinkFacts: false,
			PreserveManualFields:  true,
			Archive:               previous.Archived,
		}
	}
	return ReviewSnapshotPlan{
		Action:                "apply",
		Reason:                "complete_snapshot_changed",
		OverwriteGitLinkFacts: true,
		PreserveManualFields:  true,
	}
}

func readReviewGatewayEvent(path string) (ReviewGatewayEvent, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return ReviewGatewayEvent{}, fmt.Errorf("read review gateway event: %w", err)
	}
	data, err = normalizeJSONBytes(data)
	if err != nil {
		return ReviewGatewayEvent{}, err
	}
	var event ReviewGatewayEvent
	if err := json.Unmarshal(data, &event); err != nil {
		return ReviewGatewayEvent{}, fmt.Errorf("parse review gateway event: %w", err)
	}
	return event, nil
}

func readReviewGatewayBindings(path string) (ReviewGatewayBindings, error) {
	if strings.TrimSpace(path) == "" {
		return ReviewGatewayBindings{SchemaVersion: reviewGatewayBindingSchema, Bindings: []ReviewChatBinding{}}, nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return ReviewGatewayBindings{}, fmt.Errorf("read review gateway bindings: %w", err)
	}
	data, err = normalizeJSONBytes(data)
	if err != nil {
		return ReviewGatewayBindings{}, err
	}
	var bindings ReviewGatewayBindings
	if err := json.Unmarshal(data, &bindings); err != nil {
		return ReviewGatewayBindings{}, fmt.Errorf("parse review gateway bindings: %w", err)
	}
	if bindings.SchemaVersion == "" {
		bindings.SchemaVersion = reviewGatewayBindingSchema
	}
	return bindings, nil
}

func normalizeReviewGatewayEvent(event ReviewGatewayEvent) ReviewGatewayEvent {
	event.EventID = strings.TrimSpace(event.EventID)
	event.MessageID = strings.TrimSpace(event.MessageID)
	event.EventType = strings.ToLower(strings.TrimSpace(event.EventType))
	event.ChatID = strings.TrimSpace(event.ChatID)
	event.ChatType = strings.ToLower(strings.TrimSpace(event.ChatType))
	event.UserID = strings.TrimSpace(event.UserID)
	event.Content = strings.TrimSpace(event.Content)
	return event
}

func reviewGatewayDedupeKey(event ReviewGatewayEvent) string {
	if event.EventType == "message" && event.MessageID != "" {
		return "feishu:message:" + event.MessageID
	}
	if event.EventID != "" {
		return "feishu:event:" + event.EventID
	}
	digest := sha256.Sum256([]byte(strings.Join([]string{
		event.EventType,
		event.MessageID,
		event.ChatID,
		event.UserID,
		event.Content,
		strconv.FormatInt(event.CreateTimeMs, 10),
	}, "\x00")))
	return "feishu:fallback:" + hex.EncodeToString(digest[:16])
}

func stringSet(values []string) map[string]bool {
	result := map[string]bool{}
	for _, value := range values {
		if value = strings.TrimSpace(value); value != "" {
			result[value] = true
		}
	}
	return result
}

func sortedUniqueReviewGatewayStrings(values []string) []string {
	set := stringSet(values)
	result := make([]string, 0, len(set))
	for value := range set {
		result = append(result, value)
	}
	sort.Strings(result)
	return result
}

func containsReviewGatewayString(values []string, target string) bool {
	for _, value := range values {
		if strings.TrimSpace(value) == target {
			return true
		}
	}
	return false
}
