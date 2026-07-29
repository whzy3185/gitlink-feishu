package workflow

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"
)

const (
	reviewContextSchemaVersion = "review.context/v1"

	reviewFreshnessCurrent  = "current"
	reviewFreshnessOutdated = "outdated"
	reviewFreshnessUnknown  = "unknown"
)

// PullRequestReviewContext is the stable, read-only schema emitted by
// workflow +review-context. Raw API sections remain available for backward
// compatibility, while ReviewRecords, Threads, Summary, and WorkItem provide
// deterministic fields for collaboration clients and agent hosts.
type PullRequestReviewContext struct {
	SchemaVersion    string                     `json:"schema_version"`
	Repository       string                     `json:"repository"`
	PullRequest      int                        `json:"pull_request"`
	Source           string                     `json:"source"`
	Sections         []string                   `json:"sections"`
	CurrentHeadSHA   string                     `json:"current_head_sha,omitempty"`
	CurrentVersionID string                     `json:"current_version_id,omitempty"`
	RepositoryInfo   map[string]interface{}     `json:"repository_info,omitempty"`
	PR               map[string]interface{}     `json:"pr,omitempty"`
	Files            []map[string]interface{}   `json:"files,omitempty"`
	Reviews          []map[string]interface{}   `json:"reviews,omitempty"`
	ReviewRecords    []ReviewContextReview      `json:"review_records,omitempty"`
	Threads          []ReviewContextThread      `json:"threads,omitempty"`
	Summary          ReviewCollaborationSummary `json:"review_summary"`
	WorkItem         ReviewWorkItem             `json:"work_item"`
	OpenIssues       []map[string]interface{}   `json:"open_issues,omitempty"`
	Labels           []map[string]interface{}   `json:"labels,omitempty"`
	Notes            []ScoringNote              `json:"notes,omitempty"`
	fileLimit        int
	reviewLimit      int
	threadLimit      int
}

// ReviewContext preserves the existing Go API name while the JSON contract
// uses the more explicit PullRequestReviewContext schema.
type ReviewContext = PullRequestReviewContext

type ReviewContextReview struct {
	ID         string `json:"id,omitempty"`
	Actor      string `json:"actor,omitempty"`
	Status     string `json:"status"`
	CommitID   string `json:"commit_id,omitempty"`
	Freshness  string `json:"freshness"`
	Content    string `json:"content,omitempty"`
	CreatedAt  string `json:"created_at,omitempty"`
	UpdatedAt  string `json:"updated_at,omitempty"`
	SourceType string `json:"source_type"`
}

type ReviewContextThread struct {
	ID            string `json:"id,omitempty"`
	ReviewID      string `json:"review_id,omitempty"`
	ParentID      string `json:"parent_id,omitempty"`
	Author        string `json:"author,omitempty"`
	Type          string `json:"type,omitempty"`
	State         string `json:"state"`
	NeedRespond   *bool  `json:"need_respond,omitempty"`
	Path          string `json:"path,omitempty"`
	LineCode      string `json:"line_code,omitempty"`
	CommitID      string `json:"commit_id,omitempty"`
	Freshness     string `json:"freshness"`
	CreatedAt     string `json:"created_at,omitempty"`
	UpdatedAt     string `json:"updated_at,omitempty"`
	UnknownParent bool   `json:"unknown_parent,omitempty"`
}

type ReviewCollaborationSummary struct {
	GitLinkPRState       string `json:"gitlink_pr_state"`
	GitLinkReviewStatus  string `json:"gitlink_review_status"`
	ReviewFreshness      string `json:"review_freshness"`
	ThreadState          string `json:"thread_state"`
	Decision             string `json:"decision"`
	ReviewsAvailable     bool   `json:"reviews_available"`
	ThreadsAvailable     bool   `json:"threads_available"`
	TotalReviews         int    `json:"total_reviews"`
	CurrentReviews       int    `json:"current_reviews"`
	OutdatedReviews      int    `json:"outdated_reviews"`
	UnknownReviews       int    `json:"unknown_reviews"`
	TotalThreads         int    `json:"total_threads"`
	CurrentThreads       int    `json:"current_threads"`
	OpenThreads          int    `json:"open_threads"`
	PendingResponse      int    `json:"pending_response_threads"`
	ResolvedThreads      int    `json:"resolved_threads"`
	DisabledThreads      int    `json:"disabled_threads"`
	OutdatedThreads      int    `json:"outdated_threads"`
	UnknownThreads       int    `json:"unknown_threads"`
	UnknownParentReplies int    `json:"unknown_parent_replies"`
}

type ReviewWorkItem struct {
	SchemaVersion       string                 `json:"schema_version"`
	PRKey               string                 `json:"pr_key"`
	Repository          string                 `json:"repository"`
	Number              int                    `json:"number"`
	Title               string                 `json:"title,omitempty"`
	Author              string                 `json:"author,omitempty"`
	GitLinkURL          string                 `json:"gitlink_url,omitempty"`
	BaseBranch          string                 `json:"base_branch,omitempty"`
	HeadBranch          string                 `json:"head_branch,omitempty"`
	HeadSHA             string                 `json:"head_sha,omitempty"`
	GitLinkState        string                 `json:"gitlink_state"`
	ReviewStage         string                 `json:"review_stage"`
	Priority            string                 `json:"priority"`
	PriorityReasons     []string               `json:"priority_reasons"`
	RiskLevel           string                 `json:"risk_level"`
	ChangeMetrics       ReviewChangeMetrics    `json:"change_metrics"`
	CISummary           ReviewCISummary        `json:"ci_summary"`
	Mergeability        string                 `json:"mergeability"`
	Evidence            []ReviewEvidence       `json:"evidence"`
	Unknowns            []string               `json:"unknowns"`
	RecommendedNextStep string                 `json:"recommended_next_step"`
	SourceScope         ReviewSourceScope      `json:"source_scope"`
	SourceFingerprint   string                 `json:"source_fingerprint"`
	GeneratedAt         string                 `json:"generated_at"`
	ReviewSummary       ReviewDecisionSnapshot `json:"review_summary"`
}

type ReviewChangeMetrics struct {
	ChangedFiles int `json:"changed_files"`
	Reviews      int `json:"reviews"`
	Threads      int `json:"threads"`
}

type ReviewCISummary struct {
	State  string `json:"state"`
	Source string `json:"source"`
}

type ReviewEvidence struct {
	Type       string                  `json:"type"`
	Source     string                  `json:"source"`
	Summary    string                  `json:"summary"`
	Location   *ReviewEvidenceLocation `json:"location,omitempty"`
	SourceURL  string                  `json:"source_url,omitempty"`
	ObservedAt string                  `json:"observed_at,omitempty"`
	Confidence string                  `json:"confidence"`
}

type ReviewEvidenceLocation struct {
	Path string `json:"path"`
	Line int    `json:"line,omitempty"`
}

type ReviewSourceScope struct {
	Complete bool `json:"complete"`
	Sampled  bool `json:"sampled"`
}

type ReviewDecisionSnapshot struct {
	Status           string `json:"status"`
	Freshness        string `json:"freshness"`
	Decision         string `json:"decision"`
	PendingResponses int    `json:"pending_responses"`
}

// ReviewRun and ReviewTask establish the P1 transport contracts. They are
// intentionally not persisted or executed by the CLI in this read-only phase.
type ReviewRun struct {
	SchemaVersion    string   `json:"schema_version"`
	RunID            string   `json:"run_id"`
	Scope            string   `json:"scope"`
	Targets          []string `json:"targets"`
	AgentHost        string   `json:"agent_host,omitempty"`
	SkillsUsed       []string `json:"skills_used"`
	InputFingerprint string   `json:"input_fingerprint"`
	Status           string   `json:"status"`
	Findings         []string `json:"findings"`
	Verification     []string `json:"verification"`
	Unknowns         []string `json:"unknowns"`
	StartedAt        string   `json:"started_at,omitempty"`
	FinishedAt       string   `json:"finished_at,omitempty"`
}

type ReviewTask struct {
	SchemaVersion    string   `json:"schema_version"`
	TaskID           string   `json:"task_id"`
	PRKey            string   `json:"pr_key"`
	Type             string   `json:"type"`
	State            string   `json:"state"`
	OwnerIdentityRef string   `json:"owner_identity_ref,omitempty"`
	Collaborators    []string `json:"collaborators"`
	DependsOn        []string `json:"depends_on"`
	StartAt          string   `json:"start_at,omitempty"`
	DueAt            string   `json:"due_at,omitempty"`
	ResultRef        string   `json:"result_ref,omitempty"`
}

type reviewContextPRFacts struct {
	Title       string
	Author      string
	State       string
	BaseBranch  string
	HeadBranch  string
	HeadSHA     string
	VersionID   string
	GitLinkURL  string
	GeneratedAt time.Time
}

func extractReviewContextPRFacts(repository string, number int, pr map[string]interface{}) reviewContextPRFacts {
	facts := reviewContextPRFacts{}
	if pr == nil {
		return facts
	}
	if summary, ok := normalizePRSummaryItem(pr); ok {
		facts.Title = summary.Title
		facts.Author = summary.Author
		facts.State = normalizeGitLinkPRState(summary.State)
		facts.BaseBranch = summary.BaseBranch
		facts.HeadBranch = summary.HeadBranch
	}
	facts.HeadSHA = firstPRString(pr, "head_commit_sha", "head_sha", "commit_sha")
	if facts.HeadSHA == "" {
		if head, ok := pr["head"].(map[string]interface{}); ok {
			facts.HeadSHA = firstPRString(head, "sha", "commit_id", "head_commit_sha")
			if facts.HeadBranch == "" {
				facts.HeadBranch = firstPRString(head, "ref", "branch", "name")
			}
		}
	}
	facts.VersionID = firstPRString(pr, "version_id", "current_version_id", "patchset_id")
	facts.GitLinkURL = firstPRString(pr, "html_url", "web_url", "url")
	if facts.GitLinkURL == "" && strings.TrimSpace(repository) != "" && number > 0 {
		facts.GitLinkURL = fmt.Sprintf("https://www.gitlink.org.cn/%s/pulls/%d", repository, number)
	}
	if facts.State == "" {
		facts.State = "unknown"
	}
	return facts
}

func normalizeReviewContextReviews(items []map[string]interface{}, headSHA string) []ReviewContextReview {
	out := make([]ReviewContextReview, 0, len(items))
	for _, item := range items {
		record := ReviewContextReview{
			ID:         firstPRString(item, "id", "review_id"),
			Actor:      firstReviewActor(item),
			Status:     normalizeGitLinkReviewStatus(firstPRString(item, "status", "state", "review_status")),
			CommitID:   firstPRString(item, "commit_id", "commit_sha", "sha"),
			Content:    firstPRString(item, "content", "body", "note", "notes"),
			CreatedAt:  formatReviewContextTime(firstPRTime(item, "created_at", "createdAt")),
			UpdatedAt:  formatReviewContextTime(firstPRTime(item, "updated_at", "updatedAt")),
			SourceType: "formal_review",
		}
		record.Freshness = evaluateReviewFreshness(record.CommitID, headSHA)
		out = append(out, record)
	}
	sort.SliceStable(out, func(i, j int) bool {
		return reviewRecordSortKey(out[i]) < reviewRecordSortKey(out[j])
	})
	return out
}

func normalizeReviewContextThreads(items []map[string]interface{}, headSHA string) []ReviewContextThread {
	out := make([]ReviewContextThread, 0, len(items))
	knownIDs := map[string]bool{}
	for _, item := range items {
		record := ReviewContextThread{
			ID:          firstPRString(item, "id", "journal_id", "comment_id"),
			ReviewID:    firstPRString(item, "review_id"),
			ParentID:    firstPRString(item, "parent_id"),
			Author:      firstReviewActor(item),
			Type:        strings.ToLower(strings.TrimSpace(firstPRString(item, "type", "comment_type"))),
			State:       normalizeReviewThreadState(firstPRString(item, "state", "status")),
			NeedRespond: reviewContextBoolPointer(item, "need_respond", "needRespond"),
			Path:        firstPRString(item, "path", "file_path", "filename"),
			LineCode:    firstPRString(item, "line_code", "lineCode"),
			CommitID:    firstPRString(item, "commit_id", "commit_sha", "sha"),
			CreatedAt:   formatReviewContextTime(firstPRTime(item, "created_at", "createdAt")),
			UpdatedAt:   formatReviewContextTime(firstPRTime(item, "updated_at", "updatedAt")),
		}
		record.Freshness = evaluateReviewFreshness(record.CommitID, headSHA)
		if record.ID != "" {
			knownIDs[record.ID] = true
		}
		out = append(out, record)
	}
	for i := range out {
		if out[i].ParentID != "" && !knownIDs[out[i].ParentID] {
			out[i].UnknownParent = true
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		return reviewThreadSortKey(out[i]) < reviewThreadSortKey(out[j])
	})
	return out
}

func finalizeReviewContext(context *ReviewContext, now time.Time) {
	if context == nil {
		return
	}
	context.SchemaVersion = reviewContextSchemaVersion
	facts := extractReviewContextPRFacts(context.Repository, context.PullRequest, context.PR)
	if facts.HeadSHA == "" {
		facts.HeadSHA = context.CurrentHeadSHA
	}
	if facts.VersionID == "" {
		facts.VersionID = context.CurrentVersionID
	}
	if facts.Title == "" {
		facts.Title = context.WorkItem.Title
	}
	if facts.Author == "" {
		facts.Author = context.WorkItem.Author
	}
	if facts.BaseBranch == "" {
		facts.BaseBranch = context.WorkItem.BaseBranch
	}
	if facts.HeadBranch == "" {
		facts.HeadBranch = context.WorkItem.HeadBranch
	}
	if facts.GitLinkURL == "" {
		facts.GitLinkURL = context.WorkItem.GitLinkURL
	}
	if facts.State == "unknown" && context.WorkItem.GitLinkState != "" {
		facts.State = context.WorkItem.GitLinkState
	}
	context.CurrentHeadSHA = facts.HeadSHA
	context.CurrentVersionID = facts.VersionID
	if context.Reviews != nil || len(context.ReviewRecords) == 0 {
		context.ReviewRecords = normalizeReviewContextReviews(context.Reviews, facts.HeadSHA)
	} else {
		for i := range context.ReviewRecords {
			context.ReviewRecords[i].Freshness = evaluateReviewFreshness(context.ReviewRecords[i].CommitID, facts.HeadSHA)
		}
		sort.SliceStable(context.ReviewRecords, func(i, j int) bool {
			return reviewRecordSortKey(context.ReviewRecords[i]) < reviewRecordSortKey(context.ReviewRecords[j])
		})
	}
	for i := range context.Threads {
		context.Threads[i].Freshness = evaluateReviewFreshness(context.Threads[i].CommitID, facts.HeadSHA)
	}
	sort.SliceStable(context.Threads, func(i, j int) bool {
		return reviewThreadSortKey(context.Threads[i]) < reviewThreadSortKey(context.Threads[j])
	})
	context.Summary = summarizeReviewCollaboration(
		facts.State,
		context.ReviewRecords,
		context.Threads,
		containsString(context.Sections, "reviews"),
		containsString(context.Sections, "threads"),
	)
	context.WorkItem = buildReviewWorkItem(*context, facts, now)
}

func summarizeReviewCollaboration(prState string, reviews []ReviewContextReview, threads []ReviewContextThread, reviewsAvailable, threadsAvailable bool) ReviewCollaborationSummary {
	summary := ReviewCollaborationSummary{
		GitLinkPRState:      normalizeGitLinkPRState(prState),
		GitLinkReviewStatus: "unknown",
		ReviewFreshness:     reviewFreshnessUnknown,
		ThreadState:         "unknown",
		Decision:            "pending",
		ReviewsAvailable:    reviewsAvailable,
		ThreadsAvailable:    threadsAvailable,
		TotalReviews:        len(reviews),
		TotalThreads:        len(threads),
	}
	hasCurrentApproved := false
	hasCurrentCommon := false
	hasCurrentRejected := false
	for _, review := range reviews {
		switch review.Freshness {
		case reviewFreshnessCurrent:
			summary.CurrentReviews++
			switch review.Status {
			case "approved":
				hasCurrentApproved = true
			case "rejected":
				hasCurrentRejected = true
			case "common":
				hasCurrentCommon = true
			}
		case reviewFreshnessOutdated:
			summary.OutdatedReviews++
		default:
			summary.UnknownReviews++
		}
	}
	switch {
	case summary.CurrentReviews > 0:
		summary.ReviewFreshness = reviewFreshnessCurrent
	case summary.OutdatedReviews > 0:
		summary.ReviewFreshness = reviewFreshnessOutdated
	default:
		summary.ReviewFreshness = reviewFreshnessUnknown
	}
	switch {
	case !reviewsAvailable:
		summary.GitLinkReviewStatus = "unknown"
	case hasCurrentRejected:
		summary.GitLinkReviewStatus = "rejected"
	case hasCurrentApproved:
		summary.GitLinkReviewStatus = "approved"
	case hasCurrentCommon:
		summary.GitLinkReviewStatus = "common"
	case len(reviews) == 0:
		summary.GitLinkReviewStatus = "none"
	default:
		summary.GitLinkReviewStatus = "unknown"
	}

	hasCurrentOpenedThread := false
	hasBlockingThread := false
	for _, thread := range threads {
		switch thread.Freshness {
		case reviewFreshnessCurrent:
			summary.CurrentThreads++
		case reviewFreshnessOutdated:
			summary.OutdatedThreads++
		default:
			summary.UnknownThreads++
		}
		switch thread.State {
		case "opened":
			if thread.Freshness == reviewFreshnessCurrent {
				summary.OpenThreads++
				hasCurrentOpenedThread = true
				if (thread.NeedRespond != nil && *thread.NeedRespond) || thread.Type == "problem" {
					summary.PendingResponse++
					hasBlockingThread = true
				}
			}
		case "resolved":
			summary.ResolvedThreads++
		case "disabled":
			summary.DisabledThreads++
		}
		if thread.UnknownParent {
			summary.UnknownParentReplies++
		}
	}
	switch {
	case !threadsAvailable:
		summary.ThreadState = "unknown"
	case hasCurrentOpenedThread:
		summary.ThreadState = "opened"
	case summary.ResolvedThreads > 0:
		summary.ThreadState = "resolved"
	case summary.DisabledThreads > 0:
		summary.ThreadState = "disabled"
	case len(threads) == 0:
		summary.ThreadState = "none"
	default:
		summary.ThreadState = "unknown"
	}
	switch {
	case hasCurrentRejected:
		summary.Decision = "blocked"
	case hasBlockingThread:
		summary.Decision = "changes_pending"
	case hasCurrentApproved:
		summary.Decision = "approved"
	case hasCurrentCommon:
		summary.Decision = "commented"
	default:
		summary.Decision = "pending"
	}
	return summary
}

func buildReviewWorkItem(context ReviewContext, facts reviewContextPRFacts, now time.Time) ReviewWorkItem {
	if now.IsZero() {
		now = time.Now().UTC()
	}
	unknowns := []string{}
	if !containsString(context.Sections, "pr") {
		unknowns = append(unknowns, "pull_request_details_not_loaded")
	}
	if facts.HeadSHA == "" {
		unknowns = append(unknowns, "current_head_sha_not_returned")
	}
	if !containsString(context.Sections, "files") {
		unknowns = append(unknowns, "changed_files_not_loaded")
	}
	if !context.Summary.ReviewsAvailable {
		unknowns = append(unknowns, "formal_reviews_not_loaded")
	}
	if !context.Summary.ThreadsAvailable {
		unknowns = append(unknowns, "review_threads_not_loaded")
	}
	if context.Summary.UnknownReviews > 0 {
		unknowns = append(unknowns, "review_commit_binding_unknown")
	}
	if context.Summary.UnknownThreads > 0 {
		unknowns = append(unknowns, "review_thread_commit_binding_unknown")
	}
	if context.Summary.UnknownParentReplies > 0 {
		unknowns = append(unknowns, "orphan_review_thread_reply")
	}
	for _, note := range context.Notes {
		unknowns = append(unknowns, note.Metric)
	}
	unknowns = sortedUniqueStrings(unknowns)

	stage, next, priority, reasons := reviewWorkItemRouting(context.Summary)
	fileLimit := context.fileLimit
	if fileLimit <= 0 {
		fileLimit = 100
	}
	reviewLimit := context.reviewLimit
	if reviewLimit <= 0 {
		reviewLimit = 100
	}
	threadLimit := context.threadLimit
	if threadLimit <= 0 {
		threadLimit = 100
	}
	sampled := (containsString(context.Sections, "files") && len(context.Files) >= fileLimit) ||
		(context.Summary.ReviewsAvailable && len(context.Reviews) >= reviewLimit) ||
		(context.Summary.ThreadsAvailable && len(context.Threads) >= threadLimit)
	complete := len(context.Notes) == 0 &&
		containsString(context.Sections, "pr") &&
		containsString(context.Sections, "files") &&
		facts.HeadSHA != "" &&
		context.Summary.ReviewsAvailable &&
		context.Summary.ThreadsAvailable &&
		context.Summary.UnknownReviews == 0 &&
		context.Summary.UnknownThreads == 0 &&
		context.Summary.UnknownParentReplies == 0 &&
		!sampled
	evidence := []ReviewEvidence{}
	if facts.HeadSHA != "" {
		evidence = append(evidence, ReviewEvidence{
			Type:       "diff",
			Source:     "gitlink",
			Summary:    fmt.Sprintf("Current pull request head is %s", facts.HeadSHA),
			Confidence: "verified",
		})
	}
	if context.Summary.ReviewsAvailable {
		evidence = append(evidence, ReviewEvidence{
			Type:       "formal_review",
			Source:     "gitlink",
			Summary:    fmt.Sprintf("%d formal review records loaded; %d current", context.Summary.TotalReviews, context.Summary.CurrentReviews),
			Confidence: "verified",
		})
	}
	if context.Summary.ThreadsAvailable {
		evidence = append(evidence, ReviewEvidence{
			Type:       "journal",
			Source:     "gitlink",
			Summary:    fmt.Sprintf("%d review thread records loaded; %d pending response", context.Summary.TotalThreads, context.Summary.PendingResponse),
			Confidence: "verified",
		})
	}
	item := ReviewWorkItem{
		SchemaVersion:   "review.work-item/v1",
		PRKey:           fmt.Sprintf("gitlink:%s:pr:%d", context.Repository, context.PullRequest),
		Repository:      context.Repository,
		Number:          context.PullRequest,
		Title:           facts.Title,
		Author:          facts.Author,
		GitLinkURL:      facts.GitLinkURL,
		BaseBranch:      facts.BaseBranch,
		HeadBranch:      facts.HeadBranch,
		HeadSHA:         facts.HeadSHA,
		GitLinkState:    normalizeGitLinkPRState(facts.State),
		ReviewStage:     stage,
		Priority:        priority,
		PriorityReasons: reasons,
		RiskLevel:       "unknown",
		ChangeMetrics: ReviewChangeMetrics{
			ChangedFiles: len(context.Files),
			Reviews:      len(context.ReviewRecords),
			Threads:      len(context.Threads),
		},
		CISummary:           ReviewCISummary{State: "unknown", Source: "not_returned"},
		Mergeability:        "unknown",
		Evidence:            evidence,
		Unknowns:            unknowns,
		RecommendedNextStep: next,
		SourceScope:         ReviewSourceScope{Complete: complete, Sampled: sampled},
		GeneratedAt:         now.UTC().Format(time.RFC3339),
		ReviewSummary: ReviewDecisionSnapshot{
			Status:           context.Summary.GitLinkReviewStatus,
			Freshness:        context.Summary.ReviewFreshness,
			Decision:         context.Summary.Decision,
			PendingResponses: context.Summary.PendingResponse,
		},
	}
	item.SourceFingerprint = reviewContextFingerprint(context, item)
	return item
}

func reviewContextFingerprint(context ReviewContext, item ReviewWorkItem) string {
	payload := struct {
		SchemaVersion string                     `json:"schema_version"`
		Repository    string                     `json:"repository"`
		Number        int                        `json:"number"`
		HeadSHA       string                     `json:"head_sha"`
		Reviews       []ReviewContextReview      `json:"reviews"`
		Threads       []ReviewContextThread      `json:"threads"`
		Summary       ReviewCollaborationSummary `json:"summary"`
	}{
		SchemaVersion: reviewContextSchemaVersion,
		Repository:    context.Repository,
		Number:        context.PullRequest,
		HeadSHA:       item.HeadSHA,
		Reviews:       context.ReviewRecords,
		Threads:       context.Threads,
		Summary:       context.Summary,
	}
	encoded, _ := json.Marshal(payload)
	digest := sha256.Sum256(encoded)
	return "sha256:" + hex.EncodeToString(digest[:])
}

func evaluateReviewFreshness(commitID, headSHA string) string {
	commitID = strings.ToLower(strings.TrimSpace(commitID))
	headSHA = strings.ToLower(strings.TrimSpace(headSHA))
	if commitID == "" || headSHA == "" {
		return reviewFreshnessUnknown
	}
	if commitIDsMatch(commitID, headSHA) {
		return reviewFreshnessCurrent
	}
	return reviewFreshnessOutdated
}

func commitIDsMatch(left, right string) bool {
	left = strings.ToLower(strings.TrimSpace(left))
	right = strings.ToLower(strings.TrimSpace(right))
	if left == "" || right == "" {
		return false
	}
	if left == right {
		return true
	}
	if len(left) >= 7 && len(right) >= 7 {
		return strings.HasPrefix(left, right) || strings.HasPrefix(right, left)
	}
	return false
}

func normalizeGitLinkPRState(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "open", "opened":
		return "open"
	case "merged", "merge":
		return "merged"
	case "closed", "close":
		return "closed"
	default:
		return "unknown"
	}
}

func normalizeGitLinkReviewStatus(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "common", "commented", "comment":
		return "common"
	case "approved", "approve":
		return "approved"
	case "rejected", "reject", "changes_requested", "request_changes":
		return "rejected"
	default:
		return "unknown"
	}
}

func normalizeReviewThreadState(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "open", "opened":
		return "opened"
	case "resolve", "resolved":
		return "resolved"
	case "disable", "disabled":
		return "disabled"
	default:
		return "unknown"
	}
}

func reviewContextBoolPointer(item map[string]interface{}, keys ...string) *bool {
	for _, key := range keys {
		value, ok := item[key]
		if !ok || value == nil {
			continue
		}
		switch typed := value.(type) {
		case bool:
			result := typed
			return &result
		case string:
			switch strings.ToLower(strings.TrimSpace(typed)) {
			case "true", "1":
				result := true
				return &result
			case "false", "0":
				result := false
				return &result
			}
		case float64:
			result := typed != 0
			return &result
		case int:
			result := typed != 0
			return &result
		}
	}
	return nil
}

func reviewWorkItemRouting(summary ReviewCollaborationSummary) (stage, next, priority string, reasons []string) {
	switch summary.Decision {
	case "blocked":
		return "waiting_for_contributor", "address_rejected_review", "high", []string{"current_rejected_review"}
	case "changes_pending":
		return "waiting_for_contributor", "resolve_pending_review_threads", "high", []string{"pending_review_response"}
	case "approved":
		return "ready_for_decision", "human_merge_readiness_check", "normal", []string{"current_approval_present"}
	case "commented":
		return "human_reviewing", "continue_human_review", "normal", []string{"current_common_review"}
	default:
		return "triaged", "assign_human_reviewer", "normal", []string{"no_current_decision"}
	}
}

func reviewRecordSortKey(record ReviewContextReview) string {
	return strings.Join([]string{record.ID, record.Actor, record.CreatedAt, record.CommitID, record.Status}, "\x00")
}

func reviewThreadSortKey(record ReviewContextThread) string {
	return strings.Join([]string{record.ID, record.ParentID, record.CreatedAt, record.Path, record.LineCode}, "\x00")
}

func formatReviewContextTime(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return value.UTC().Format(time.RFC3339)
}

func sortedUniqueStrings(values []string) []string {
	set := map[string]bool{}
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			set[value] = true
		}
	}
	out := make([]string, 0, len(set))
	for value := range set {
		out = append(out, value)
	}
	sort.Strings(out)
	return out
}
