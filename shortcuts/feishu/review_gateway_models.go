package feishu

import (
	"context"
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
	SchemaVersion    string                  `json:"schema_version"`
	Bindings         []ReviewChatBinding     `json:"bindings"`
	IdentityBindings []ReviewIdentityBinding `json:"identity_bindings,omitempty"`
}

type ReviewGatewayIntent struct {
	Name               string `json:"name"`
	Repository         string `json:"repository,omitempty"`
	PRNumber           int    `json:"pr_number,omitempty"`
	Argument           string `json:"argument,omitempty"`
	ExplicitRepository bool   `json:"explicit_repository,omitempty"`
}

type ReviewGatewayJob struct {
	SchemaVersion           string `json:"schema_version"`
	JobID                   string `json:"job_id"`
	DedupeKey               string `json:"dedupe_key"`
	Status                  string `json:"status"`
	Mode                    string `json:"mode"`
	Action                  string `json:"action"`
	Repository              string `json:"repository,omitempty"`
	PRNumber                int    `json:"pr_number,omitempty"`
	ChatID                  string `json:"chat_id"`
	RequestedBy             string `json:"requested_by"`
	SourceEventID           string `json:"source_event_id,omitempty"`
	SourceMessageID         string `json:"source_message_id,omitempty"`
	CreatedAt               string `json:"created_at"`
	MutatesGitLink          bool   `json:"mutates_gitlink"`
	CollaborationMutation   bool   `json:"collaboration_mutation"`
	RequiresAdmin           bool   `json:"requires_admin"`
	RequirePublicRepo       bool   `json:"require_public_repository,omitempty"`
	AttemptCount            int    `json:"attempt_count"`
	MaxAttempts             int    `json:"max_attempts"`
	NextAttemptAt           string `json:"next_attempt_at,omitempty"`
	LeaseOwner              string `json:"lease_owner,omitempty"`
	LeaseExpiresAt          string `json:"lease_expires_at,omitempty"`
	HandlerLatencyMs        int64  `json:"handler_latency_ms,omitempty"`
	Argument                string `json:"argument,omitempty"`
	GitLinkLogin            string `json:"gitlink_login,omitempty"`
	RequestedDisplayName    string `json:"requested_display_name,omitempty"`
	CollaborationAuthorized bool   `json:"collaboration_authorized,omitempty"`
	CollaborationAdmin      bool   `json:"collaboration_admin,omitempty"`
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
	SchemaVersion    string                `json:"schema_version"`
	Mode             string                `json:"mode"`
	Accepted         bool                  `json:"accepted"`
	Duplicate        bool                  `json:"duplicate"`
	Stale            bool                  `json:"stale"`
	Bound            bool                  `json:"bound"`
	Reason           string                `json:"reason,omitempty"`
	DedupeKey        string                `json:"dedupe_key,omitempty"`
	Event            ReviewGatewayEventRef `json:"event"`
	Binding          *ReviewChatBinding    `json:"binding,omitempty"`
	Intent           ReviewGatewayIntent   `json:"intent"`
	Job              *ReviewGatewayJob     `json:"job,omitempty"`
	HandlerLatencyMs int64                 `json:"handler_latency_ms,omitempty"`
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
	bindings   map[string]ReviewChatBinding
	admins     map[string]bool
	identities map[string]ReviewIdentityBinding
	deduper    ReviewGatewayDeduper
	now        func() time.Time
	stale      time.Duration
}

var (
	reviewGatewayRepositoryPattern    = regexp.MustCompile(`^[A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+$`)
	reviewGatewayPRPattern            = regexp.MustCompile(`(?i)^查看\s*PR\s*#?(\d+)$`)
	reviewGatewayScopedPRPattern      = regexp.MustCompile(`(?i)^查看\s+([A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+)\s+PR\s*#?(\d+)$`)
	reviewGatewayRefreshPattern       = regexp.MustCompile(`(?i)^刷新\s*PR\s*#?(\d+)$`)
	reviewGatewayClaimPattern         = regexp.MustCompile(`(?i)^领取\s+([A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+)\s+PR\s*#?(\d+)$`)
	reviewGatewayReleasePattern       = regexp.MustCompile(`(?i)^(?:取消领取|释放)\s+([A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+)\s+PR\s*#?(\d+)$`)
	reviewGatewayDeadlinePattern      = regexp.MustCompile(`(?i)^设置\s+([A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+)\s+PR\s*#?(\d+)\s+(?:审查截止|截止)\s+(\d{4}-\d{2}-\d{2})$`)
	reviewGatewayClearDeadlinePattern = regexp.MustCompile(`(?i)^清除\s+([A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+)\s+PR\s*#?(\d+)\s+审查截止$`)
	reviewGatewayCommonReviewPattern  = regexp.MustCompile(`(?i)^(?:提交审查意见|review)\s+([A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+)(?:\s+PR\s*#?|#)(\d+)\s+(.+)$`)
	reviewGatewayApprovePattern       = regexp.MustCompile(`(?i)^(?:批准|approve)\s+([A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+)(?:\s+PR\s*#?|#)(\d+)\s+(.+)$`)
	reviewGatewayRejectPattern        = regexp.MustCompile(`(?i)^(?:需要修改|要求修改|reject)\s+([A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+)(?:\s+PR\s*#?|#)(\d+)\s+(.+)$`)
	reviewGatewayRefusePattern        = regexp.MustCompile(`(?i)^(?:拒绝并关闭|refuse)\s+([A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+)(?:\s+PR\s*#?|#)(\d+)\s+(.+)$`)
	reviewGatewayMergePattern         = regexp.MustCompile(`(?i)^(?:合并|merge)\s+([A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+)(?:\s+PR\s*#?|#)(\d+)$`)
	reviewGatewayBindPattern          = regexp.MustCompile(`(?i)^绑定仓库\s+([A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+)$`)
	reviewGatewayIntentPatterns       = []struct {
		pattern *regexp.Regexp
		name    string
	}{
		{pattern: reviewGatewayBindPattern, name: "plan_bind_repository"},
		{pattern: reviewGatewayPRPattern, name: "read_review_context"},
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
		bindings:   map[string]ReviewChatBinding{},
		admins:     stringSet(config.AdminUserIDs),
		identities: map[string]ReviewIdentityBinding{},
		deduper:    deduper,
		now:        config.Now,
		stale:      config.StaleWindow,
	}
	for _, identity := range bindings.IdentityBindings {
		identity.FeishuUserID = strings.TrimSpace(identity.FeishuUserID)
		identity.GitLinkLogin = strings.TrimSpace(identity.GitLinkLogin)
		if identity.FeishuUserID == "" || identity.GitLinkLogin == "" {
			return nil, fmt.Errorf("review identity binding requires feishu_user_id and gitlink_login")
		}
		if _, exists := result.identities[identity.FeishuUserID]; exists {
			return nil, fmt.Errorf("duplicate review identity binding for user %q", identity.FeishuUserID)
		}
		result.identities[identity.FeishuUserID] = identity
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
	return g.PlanContext(context.Background(), event)
}

func (g *ReviewGateway) PlanContext(ctx context.Context, event ReviewGatewayEvent) (ReviewGatewayReceipt, error) {
	return g.planContext(ctx, event, true)
}

func (g *ReviewGateway) planWithoutReserveContext(ctx context.Context, event ReviewGatewayEvent) (ReviewGatewayReceipt, error) {
	return g.planContext(ctx, event, false)
}

func (g *ReviewGateway) planContext(ctx context.Context, event ReviewGatewayEvent, reserve bool) (ReviewGatewayReceipt, error) {
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
	if intent.Name == "unknown" {
		intent.Name = "unsupported_command"
	}
	receipt.Intent = intent
	if isReviewCollaborationAction(intent.Name) || isControlledReviewPrepareAction(intent.Name) {
		identity, ok := g.identities[event.UserID]
		if !ok || !identity.Enabled {
			receipt.Reason = "collaboration_identity_required"
			return receipt, nil
		}
		intent.Repository = strings.TrimSpace(intent.Repository)
		receipt.Intent = intent
	}

	binding, bound := g.bindings[event.ChatID]
	receipt.Bound = bound && binding.Enabled
	if bound {
		copy := binding
		receipt.Binding = &copy
	}
	if intent.Name != "help" && intent.Name != "unsupported_command" && intent.Name != "plan_bind_repository" {
		explicitPublicRead := intent.Name == "read_review_context" && intent.ExplicitRepository
		if (!bound || !binding.Enabled) && !explicitPublicRead {
			receipt.Reason = "chat_not_bound"
			return receipt, nil
		}
		if !explicitPublicRead && !isReviewCollaborationAction(intent.Name) && !isControlledReviewPrepareAction(intent.Name) && len(binding.AllowedUserIDs) > 0 && !containsReviewGatewayString(binding.AllowedUserIDs, event.UserID) && !g.isAdmin(binding, event.UserID) {
			receipt.Reason = "sender_not_allowed"
			return receipt, nil
		}
		if !intent.ExplicitRepository {
			intent.Repository = binding.Repository
		}
		receipt.Intent = intent
	}
	if intent.Name == "plan_bind_repository" && !g.isAdmin(binding, event.UserID) {
		receipt.Reason = "binding_requires_admin"
		return receipt, nil
	}

	dedupeKey := reviewGatewayDedupeKey(event)
	receipt.DedupeKey = dedupeKey
	if reserve {
		var reserved bool
		var err error
		if contextual, ok := g.deduper.(interface {
			ReserveContext(context.Context, string, time.Time, time.Duration) (bool, error)
		}); ok {
			reserved, err = contextual.ReserveContext(ctx, dedupeKey, now, 24*time.Hour)
		} else {
			reserved, err = g.deduper.Reserve(dedupeKey, now, 24*time.Hour)
		}
		if err != nil {
			receipt.Reason = "state_store_failed"
			return receipt, err
		}
		if !reserved {
			receipt.Duplicate = true
			receipt.Reason = "duplicate_event"
			return receipt, nil
		}
	}

	job := newReviewGatewayJob(event, intent, dedupeKey, now)
	if isReviewCollaborationAction(intent.Name) || isControlledReviewPrepareAction(intent.Name) {
		identity := g.identities[event.UserID]
		job.GitLinkLogin = identity.GitLinkLogin
		job.CollaborationAuthorized = true
		job.CollaborationAdmin = g.isAdmin(binding, event.UserID)
	}
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
	}
	if match := reviewGatewayScopedPRPattern.FindStringSubmatch(content); len(match) == 3 {
		number, _ := strconv.Atoi(match[2])
		return ReviewGatewayIntent{
			Name: "read_review_context", Repository: match[1], PRNumber: number, ExplicitRepository: true,
		}
	}
	for _, candidate := range []struct {
		pattern *regexp.Regexp
		name    string
	}{
		{reviewGatewayClaimPattern, "claim_review"},
		{reviewGatewayReleasePattern, "release_review"},
		{reviewGatewayDeadlinePattern, "set_review_deadline"},
		{reviewGatewayClearDeadlinePattern, "clear_review_deadline"},
	} {
		match := candidate.pattern.FindStringSubmatch(content)
		if len(match) < 3 {
			continue
		}
		number, _ := strconv.Atoi(match[2])
		intent := ReviewGatewayIntent{Name: candidate.name, Repository: match[1], PRNumber: number, ExplicitRepository: true}
		if len(match) > 3 {
			intent.Argument = match[3]
		}
		return intent
	}
	for _, candidate := range []struct {
		pattern *regexp.Regexp
		name    string
	}{
		{reviewGatewayCommonReviewPattern, "prepare_common_review"},
		{reviewGatewayApprovePattern, "prepare_review_approve"},
		{reviewGatewayRejectPattern, "prepare_review_reject"},
		{reviewGatewayRefusePattern, "prepare_reject_close"},
		{reviewGatewayMergePattern, "prepare_merge"},
	} {
		match := candidate.pattern.FindStringSubmatch(content)
		if len(match) < 3 {
			continue
		}
		number, _ := strconv.Atoi(match[2])
		intent := ReviewGatewayIntent{Name: candidate.name, Repository: match[1], PRNumber: number, ExplicitRepository: true}
		if len(match) > 3 {
			intent.Argument = strings.TrimSpace(match[3])
		}
		return intent
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
	collaborationMutation := intent.Name == "plan_bind_repository" || isReviewCollaborationAction(intent.Name)
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
		RequirePublicRepo:     intent.ExplicitRepository,
		MaxAttempts:           3,
		NextAttemptAt:         now.Format(time.RFC3339Nano),
		Argument:              intent.Argument,
	}
}

func isReviewCollaborationAction(action string) bool {
	switch action {
	case "claim_review", "release_review", "set_review_deadline", "clear_review_deadline":
		return true
	default:
		return false
	}
}

func isControlledReviewPrepareAction(action string) bool {
	switch action {
	case "prepare_common_review", "prepare_review_approve", "prepare_review_reject", "prepare_reject_close", "prepare_merge":
		return true
	default:
		return false
	}
}

func PlanReviewSnapshotSync(current ReviewData, previous *ReviewSnapshotState) ReviewSnapshotPlan {
	state := strings.ToLower(strings.TrimSpace(current.State))
	if state == "merged" || state == "closed" {
		return ReviewSnapshotPlan{
			Action:                "archive",
			Reason:                "gitlink_pr_" + state,
			OverwriteGitLinkFacts: false,
			PreserveManualFields:  true,
			Archive:               true,
		}
	}
	if current.Partial || current.CollectionStatus != "complete" {
		return ReviewSnapshotPlan{
			Action:                "preserve_previous",
			Reason:                "partial_or_incomplete_snapshot",
			OverwriteGitLinkFacts: false,
			PreserveManualFields:  true,
		}
	}
	if previous != nil && previous.Complete &&
		previous.SourceFingerprint == current.SourceFingerprint &&
		previous.HeadSHA == current.HeadSHA {
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
