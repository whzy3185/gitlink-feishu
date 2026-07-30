package workflow

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"
)

const (
	reviewContextSchemaVersion = "review.context/v1"

	reviewFreshnessCurrent  = "current"
	reviewFreshnessOutdated = "outdated"
	reviewFreshnessUnknown  = "unknown"

	reviewCollectionComplete = "complete"
	reviewCollectionPartial  = "partial"
	reviewCollectionFailed   = "failed"

	reviewSectionLoaded   = "loaded"
	reviewSectionEmpty    = "empty"
	reviewSectionDisabled = "disabled"
	reviewSectionFailed   = "failed"
	reviewSectionPending  = "pending"
	reviewSectionSampled  = "sampled"
)

// PullRequestReviewContext is the stable, read-only schema emitted by
// workflow +review-context. Raw API sections remain available for backward
// compatibility, while ReviewRecords, Threads, Summary, and WorkItem provide
// deterministic fields for collaboration clients and agent hosts.
type PullRequestReviewContext struct {
	SchemaVersion     string                         `json:"schema_version"`
	Repository        string                         `json:"repository"`
	PullRequest       int                            `json:"pull_request"`
	Source            string                         `json:"source"`
	Sections          []string                       `json:"sections"`
	CurrentHeadSHA    string                         `json:"current_head_sha,omitempty"`
	CurrentVersionID  string                         `json:"current_version_id,omitempty"`
	RepositoryInfo    map[string]interface{}         `json:"repository_info,omitempty"`
	PR                map[string]interface{}         `json:"pr,omitempty"`
	Files             []map[string]interface{}       `json:"files,omitempty"`
	Versions          []map[string]interface{}       `json:"versions,omitempty"`
	CurrentPatchset   ReviewContextPatchset          `json:"current_patchset"`
	Reviews           []map[string]interface{}       `json:"reviews,omitempty"`
	ReviewRecords     []ReviewContextReview          `json:"review_records,omitempty"`
	ReviewerSummaries []ReviewContextReviewerSummary `json:"reviewer_summaries"`
	Threads           []ReviewContextThread          `json:"threads,omitempty"`
	Summary           ReviewCollaborationSummary     `json:"review_summary"`
	WorkItem          ReviewWorkItem                 `json:"work_item"`
	OpenIssues        []map[string]interface{}       `json:"open_issues,omitempty"`
	Labels            []map[string]interface{}       `json:"labels,omitempty"`
	CollectionStatus  string                         `json:"collection_status"`
	Partial           bool                           `json:"partial"`
	SectionStatuses   []ReviewContextSectionStatus   `json:"section_statuses"`
	FetchErrors       []ReviewContextFetchError      `json:"fetch_errors"`
	Notes             []ScoringNote                  `json:"notes,omitempty"`
	fileLimit         int
	versionLimit      int
	reviewLimit       int
	threadLimit       int
}

// ReviewContext preserves the existing Go API name while the JSON contract
// uses the more explicit PullRequestReviewContext schema.
type ReviewContext = PullRequestReviewContext

type ReviewContextReview struct {
	ID         string `json:"id,omitempty"`
	ActorID    string `json:"actor_id,omitempty"`
	Actor      string `json:"actor,omitempty"`
	Status     string `json:"status"`
	CommitID   string `json:"commit_id,omitempty"`
	Freshness  string `json:"freshness"`
	Content    string `json:"content,omitempty"`
	CreatedAt  string `json:"created_at,omitempty"`
	UpdatedAt  string `json:"updated_at,omitempty"`
	SourceType string `json:"source_type"`
}

type ReviewContextPatchset struct {
	ID           string `json:"id,omitempty"`
	HeadSHA      string `json:"head_sha,omitempty"`
	BaseSHA      string `json:"base_sha,omitempty"`
	StartSHA     string `json:"start_sha,omitempty"`
	FilesCount   int    `json:"files_count"`
	CommitsCount int    `json:"commits_count"`
	Additions    int    `json:"additions"`
	Deletions    int    `json:"deletions"`
	CreatedAt    string `json:"created_at,omitempty"`
	UpdatedAt    string `json:"updated_at,omitempty"`
}

type ReviewContextReviewerSummary struct {
	ReviewerKey           string               `json:"reviewer_key"`
	ActorID               string               `json:"actor_id,omitempty"`
	Actor                 string               `json:"actor,omitempty"`
	CurrentDecision       string               `json:"current_decision"`
	CurrentCount          int                  `json:"current_count"`
	OutdatedCount         int                  `json:"outdated_count"`
	UnknownCount          int                  `json:"unknown_count"`
	DecisionOrderKnown    bool                 `json:"decision_order_known"`
	LatestEffectiveReview *ReviewContextReview `json:"latest_effective_review,omitempty"`
}

type ReviewContextThread struct {
	ID            string `json:"id,omitempty"`
	ReviewID      string `json:"review_id,omitempty"`
	ParentID      string `json:"parent_id,omitempty"`
	AuthorID      string `json:"author_id,omitempty"`
	Author        string `json:"author,omitempty"`
	Content       string `json:"content,omitempty"`
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

type ReviewContextSectionStatus struct {
	Section   string `json:"section"`
	Status    string `json:"status"`
	Required  bool   `json:"required"`
	ItemCount int    `json:"item_count,omitempty"`
	Limit     int    `json:"limit,omitempty"`
}

type ReviewContextFetchError struct {
	Section    string `json:"section"`
	Method     string `json:"method"`
	Path       string `json:"path"`
	Code       string `json:"code"`
	StatusCode int    `json:"status_code,omitempty"`
	Retryable  bool   `json:"retryable"`
	Message    string `json:"message"`
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
	if facts.Title == "" {
		facts.Title = firstPRString(pr, "title", "subject")
	}
	if facts.Author == "" {
		facts.Author = firstPRString(pr, "create_user", "fork_project_user", "author_name")
	}
	if facts.BaseBranch == "" {
		facts.BaseBranch = firstPRBranch(pr, "base_branch", "target_branch", "base")
	}
	if facts.HeadBranch == "" {
		facts.HeadBranch = firstPRBranch(pr, "head_branch", "source_branch", "head")
	}
	if apiBool(pr["merged"]) {
		facts.State = "merged"
	} else if state := firstPRString(pr, "state", "pull_request_staus", "status"); state != "" {
		facts.State = normalizeGitLinkPRState(state)
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

func normalizeCurrentReviewContextPatchset(items []map[string]interface{}) ReviewContextPatchset {
	var current ReviewContextPatchset
	for _, item := range items {
		candidate := ReviewContextPatchset{
			ID:           firstPRString(item, "id", "version_id"),
			HeadSHA:      firstPRString(item, "head_commit_sha", "head_sha", "commit_sha"),
			BaseSHA:      firstPRString(item, "base_commit_sha", "base_sha"),
			StartSHA:     firstPRString(item, "start_commit_sha", "start_sha"),
			FilesCount:   firstPRInt(item, "files_count", "changed_files"),
			CommitsCount: firstPRInt(item, "commits_count", "commits"),
			Additions:    firstPRInt(item, "add_line_num", "additions"),
			Deletions:    firstPRInt(item, "del_line_num", "deletions"),
			CreatedAt:    formatReviewContextTime(firstPRTime(item, "created_time", "created_at")),
			UpdatedAt:    formatReviewContextTime(firstPRTime(item, "updated_time", "updated_at")),
		}
		if current.ID == "" || reviewContextPatchsetLater(candidate, current) {
			current = candidate
		}
	}
	return current
}

func reviewContextPatchsetLater(left, right ReviewContextPatchset) bool {
	leftTime := firstNonZeroReviewContextTime(left.UpdatedAt, left.CreatedAt)
	rightTime := firstNonZeroReviewContextTime(right.UpdatedAt, right.CreatedAt)
	if !leftTime.IsZero() && !rightTime.IsZero() && !leftTime.Equal(rightTime) {
		return leftTime.After(rightTime)
	}
	leftID, leftOK := reviewContextNumericID(left.ID)
	rightID, rightOK := reviewContextNumericID(right.ID)
	if leftOK && rightOK {
		return leftID > rightID
	}
	return left.ID > right.ID
}

func firstNonZeroReviewContextTime(values ...string) time.Time {
	for _, value := range values {
		if parsed := parseAPIStringTime(value); !parsed.IsZero() {
			return parsed
		}
	}
	return time.Time{}
}

func normalizeReviewContextReviews(items []map[string]interface{}, headSHA string) []ReviewContextReview {
	out := make([]ReviewContextReview, 0, len(items))
	for _, item := range items {
		record := ReviewContextReview{
			ID:         firstPRString(item, "id", "review_id"),
			ActorID:    firstReviewActorID(item),
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
			ReviewID:    firstReviewContextReviewID(item),
			ParentID:    firstPRString(item, "parent_id"),
			AuthorID:    firstReviewActorID(item),
			Author:      firstReviewActor(item),
			Content:     firstPRString(item, "note", "content", "body"),
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

func firstReviewContextReviewID(item map[string]interface{}) string {
	if value := firstPRString(item, "review_id"); value != "" {
		return value
	}
	if review, ok := item["review"].(map[string]interface{}); ok {
		return firstPRString(review, "id", "review_id")
	}
	return ""
}

func finalizeReviewContext(context *ReviewContext, now time.Time) {
	if context == nil {
		return
	}
	sanitizeReviewContextDiagnostics(context)
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
	if context.CurrentPatchset.ID == "" && len(context.Versions) > 0 {
		context.CurrentPatchset = normalizeCurrentReviewContextPatchset(context.Versions)
	}
	if facts.HeadSHA == "" {
		facts.HeadSHA = context.CurrentPatchset.HeadSHA
	}
	if facts.VersionID == "" {
		facts.VersionID = context.CurrentPatchset.ID
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
	context.ReviewerSummaries = buildReviewContextReviewerSummaries(context.ReviewRecords)
	context.Summary = summarizeReviewCollaborationWithReviewers(
		facts.State,
		context.ReviewRecords,
		context.ReviewerSummaries,
		context.Threads,
		containsString(context.Sections, "reviews"),
		containsString(context.Sections, "threads"),
	)
	context.WorkItem = buildReviewWorkItem(*context, facts, now)
	finalizeReviewContextCollection(context)
	context.WorkItem.SourceFingerprint = reviewContextFingerprint(*context, context.WorkItem)
}

func summarizeReviewCollaboration(prState string, reviews []ReviewContextReview, threads []ReviewContextThread, reviewsAvailable, threadsAvailable bool) ReviewCollaborationSummary {
	return summarizeReviewCollaborationWithReviewers(
		prState,
		reviews,
		buildReviewContextReviewerSummaries(reviews),
		threads,
		reviewsAvailable,
		threadsAvailable,
	)
}

func summarizeReviewCollaborationWithReviewers(
	prState string,
	reviews []ReviewContextReview,
	reviewerSummaries []ReviewContextReviewerSummary,
	threads []ReviewContextThread,
	reviewsAvailable bool,
	threadsAvailable bool,
) ReviewCollaborationSummary {
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
	hasCurrentUnknownDecision := false
	for _, review := range reviews {
		switch review.Freshness {
		case reviewFreshnessCurrent:
			summary.CurrentReviews++
		case reviewFreshnessOutdated:
			summary.OutdatedReviews++
		default:
			summary.UnknownReviews++
		}
	}
	for _, reviewer := range reviewerSummaries {
		if reviewer.CurrentCount == 0 {
			continue
		}
		switch reviewer.CurrentDecision {
		case "approved":
			hasCurrentApproved = true
		case "rejected":
			hasCurrentRejected = true
		case "common":
			hasCurrentCommon = true
		default:
			hasCurrentUnknownDecision = true
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
	case hasCurrentUnknownDecision:
		summary.GitLinkReviewStatus = "unknown"
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
	case hasCurrentUnknownDecision:
		summary.Decision = "pending"
	case hasCurrentApproved:
		summary.Decision = "approved"
	case hasCurrentCommon:
		summary.Decision = "commented"
	default:
		summary.Decision = "pending"
	}
	return summary
}

func buildReviewContextReviewerSummaries(reviews []ReviewContextReview) []ReviewContextReviewerSummary {
	grouped := map[string][]ReviewContextReview{}
	for _, review := range reviews {
		key := reviewContextReviewerKey(review)
		grouped[key] = append(grouped[key], review)
	}
	keys := make([]string, 0, len(grouped))
	for key := range grouped {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	out := make([]ReviewContextReviewerSummary, 0, len(keys))
	for _, key := range keys {
		records := grouped[key]
		summary := ReviewContextReviewerSummary{
			ReviewerKey:     key,
			CurrentDecision: "none",
		}
		if len(records) > 0 {
			summary.ActorID = records[0].ActorID
			summary.Actor = records[0].Actor
		}
		current := make([]ReviewContextReview, 0, len(records))
		for _, review := range records {
			switch review.Freshness {
			case reviewFreshnessCurrent:
				summary.CurrentCount++
				current = append(current, review)
			case reviewFreshnessOutdated:
				summary.OutdatedCount++
			default:
				summary.UnknownCount++
			}
		}
		if len(current) > 0 {
			latest, orderKnown, decision := latestEffectiveReviewerReview(current)
			summary.CurrentDecision = decision
			summary.DecisionOrderKnown = orderKnown
			if latest != nil {
				record := *latest
				summary.LatestEffectiveReview = &record
				if strings.TrimSpace(record.ActorID) != "" {
					summary.ActorID = record.ActorID
				}
				if strings.TrimSpace(record.Actor) != "" {
					summary.Actor = record.Actor
				}
			}
		}
		out = append(out, summary)
	}
	return out
}

func latestEffectiveReviewerReview(reviews []ReviewContextReview) (*ReviewContextReview, bool, string) {
	if len(reviews) == 0 {
		return nil, false, "none"
	}
	if len(reviews) == 1 {
		record := reviews[0]
		return &record, true, record.Status
	}

	latest := reviews[0]
	orderKnown := true
	statuses := map[string]bool{latest.Status: true}
	for _, candidate := range reviews[1:] {
		statuses[candidate.Status] = true
		comparison, comparable := compareReviewContextRecency(candidate, latest)
		if !comparable {
			orderKnown = false
			if reviewRecordSortKey(candidate) > reviewRecordSortKey(latest) {
				latest = candidate
			}
			continue
		}
		if comparison > 0 {
			latest = candidate
		}
	}
	if !orderKnown {
		if len(statuses) == 1 {
			return nil, false, latest.Status
		}
		return nil, false, "unknown"
	}
	record := latest
	return &record, true, record.Status
}

func compareReviewContextRecency(left, right ReviewContextReview) (int, bool) {
	leftTime := effectiveReviewContextTime(left)
	rightTime := effectiveReviewContextTime(right)
	if !leftTime.IsZero() || !rightTime.IsZero() {
		if !leftTime.IsZero() && !rightTime.IsZero() && !leftTime.Equal(rightTime) {
			if leftTime.After(rightTime) {
				return 1, true
			}
			return -1, true
		}
		return 0, false
	}
	leftID, leftOK := reviewContextNumericID(left.ID)
	rightID, rightOK := reviewContextNumericID(right.ID)
	if leftOK && rightOK && leftID != rightID {
		if leftID > rightID {
			return 1, true
		}
		return -1, true
	}
	return 0, false
}

func effectiveReviewContextTime(review ReviewContextReview) time.Time {
	for _, value := range []string{review.UpdatedAt, review.CreatedAt} {
		if parsed := parseAPIStringTime(value); !parsed.IsZero() {
			return parsed
		}
	}
	return time.Time{}
}

func reviewContextNumericID(value string) (int64, bool) {
	parsed, err := strconv.ParseInt(strings.TrimSpace(value), 10, 64)
	return parsed, err == nil
}

func reviewContextReviewerKey(review ReviewContextReview) string {
	if value := strings.TrimSpace(review.ActorID); value != "" {
		return "id:" + strings.ToLower(value)
	}
	if value := strings.TrimSpace(review.Actor); value != "" {
		return "actor:" + strings.ToLower(value)
	}
	if value := strings.TrimSpace(review.ID); value != "" {
		return "unknown-review:" + value
	}
	return "unknown-review:" + reviewRecordSortKey(review)
}

func firstReviewActorID(item map[string]interface{}) string {
	if value := firstPRString(item, "reviewer_id", "user_id", "author_id", "creator_id"); value != "" {
		return value
	}
	for _, key := range []string{"reviewer", "user", "author", "creator"} {
		nested, ok := item[key].(map[string]interface{})
		if !ok {
			continue
		}
		if value := firstPRString(nested, "id", "user_id", "uid"); value != "" {
			return value
		}
	}
	return ""
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
	if !containsString(context.Sections, "versions") {
		unknowns = append(unknowns, "patchset_versions_not_loaded")
	} else if context.CurrentPatchset.HeadSHA == "" {
		unknowns = append(unknowns, "current_patchset_head_not_returned")
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
	versionLimit := context.versionLimit
	if versionLimit <= 0 {
		versionLimit = 100
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
		(containsString(context.Sections, "versions") && len(context.Versions) >= versionLimit) ||
		(context.Summary.ReviewsAvailable && reviewContextReviewCount(context) >= reviewLimit) ||
		(context.Summary.ThreadsAvailable && len(context.Threads) >= threadLimit)
	complete := len(context.Notes) == 0 &&
		containsString(context.Sections, "pr") &&
		containsString(context.Sections, "files") &&
		containsString(context.Sections, "versions") &&
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
	return item
}

func reviewContextFingerprint(context ReviewContext, item ReviewWorkItem) string {
	type fingerprintFetchError struct {
		Section    string `json:"section"`
		Method     string `json:"method"`
		Path       string `json:"path"`
		Code       string `json:"code"`
		StatusCode int    `json:"status_code"`
		Retryable  bool   `json:"retryable"`
	}
	fetchErrors := make([]fingerprintFetchError, 0, len(context.FetchErrors))
	for _, fetchError := range context.FetchErrors {
		fetchErrors = append(fetchErrors, fingerprintFetchError{
			Section:    fetchError.Section,
			Method:     fetchError.Method,
			Path:       fetchError.Path,
			Code:       fetchError.Code,
			StatusCode: fetchError.StatusCode,
			Retryable:  fetchError.Retryable,
		})
	}
	payload := struct {
		SchemaVersion     string                         `json:"schema_version"`
		Repository        string                         `json:"repository"`
		Number            int                            `json:"number"`
		HeadSHA           string                         `json:"head_sha"`
		CurrentPatchset   ReviewContextPatchset          `json:"current_patchset"`
		Reviews           []ReviewContextReview          `json:"reviews"`
		ReviewerSummaries []ReviewContextReviewerSummary `json:"reviewer_summaries"`
		Threads           []ReviewContextThread          `json:"threads"`
		Summary           ReviewCollaborationSummary     `json:"summary"`
		CollectionStatus  string                         `json:"collection_status"`
		SectionStatuses   []ReviewContextSectionStatus   `json:"section_statuses"`
		FetchErrors       []fingerprintFetchError        `json:"fetch_errors"`
	}{
		SchemaVersion:     reviewContextSchemaVersion,
		Repository:        context.Repository,
		Number:            context.PullRequest,
		HeadSHA:           item.HeadSHA,
		CurrentPatchset:   context.CurrentPatchset,
		Reviews:           context.ReviewRecords,
		ReviewerSummaries: context.ReviewerSummaries,
		Threads:           context.Threads,
		Summary:           context.Summary,
		CollectionStatus:  context.CollectionStatus,
		SectionStatuses:   context.SectionStatuses,
		FetchErrors:       fetchErrors,
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
	switch summary.GitLinkPRState {
	case "merged":
		return "merged", "none", "low", []string{"gitlink_pr_merged"}
	case "closed":
		return "closed", "none", "low", []string{"gitlink_pr_closed"}
	}
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
		if summary.CurrentReviews > 0 {
			return "human_reviewing", "resolve_ambiguous_reviewer_decision", "high", []string{"current_review_order_unknown"}
		}
		if summary.OutdatedReviews > 0 {
			return "waiting_for_re_review", "request_re_review_for_current_head", "high", []string{"only_outdated_reviews"}
		}
		return "triaged", "assign_human_reviewer", "normal", []string{"no_current_decision"}
	}
}

func finalizeReviewContextCollection(context *ReviewContext) {
	if context == nil {
		return
	}
	if len(context.SectionStatuses) == 0 {
		for _, section := range context.Sections {
			context.SectionStatuses = append(context.SectionStatuses, ReviewContextSectionStatus{
				Section:   section,
				Status:    reviewSectionLoaded,
				Required:  isRequiredReviewContextSection(section),
				ItemCount: reviewContextSectionItemCount(*context, section),
			})
		}
	}
	sort.SliceStable(context.SectionStatuses, func(i, j int) bool {
		return context.SectionStatuses[i].Section < context.SectionStatuses[j].Section
	})
	sort.SliceStable(context.FetchErrors, func(i, j int) bool {
		left := context.FetchErrors[i]
		right := context.FetchErrors[j]
		return strings.Join([]string{left.Section, left.Method, left.Path, left.Code}, "\x00") <
			strings.Join([]string{right.Section, right.Method, right.Path, right.Code}, "\x00")
	})
	switch {
	case len(context.Sections) == 0 && len(context.FetchErrors) > 0:
		context.CollectionStatus = reviewCollectionFailed
	case len(context.FetchErrors) > 0 || !context.WorkItem.SourceScope.Complete:
		context.CollectionStatus = reviewCollectionPartial
	default:
		context.CollectionStatus = reviewCollectionComplete
	}
	context.Partial = context.CollectionStatus != reviewCollectionComplete
}

func isRequiredReviewContextSection(section string) bool {
	switch section {
	case "pr", "files", "versions", "reviews", "threads":
		return true
	default:
		return false
	}
}

func reviewContextSectionItemCount(context ReviewContext, section string) int {
	switch section {
	case "repo_info", "pr":
		return 1
	case "files":
		return len(context.Files)
	case "versions":
		return len(context.Versions)
	case "reviews":
		return reviewContextReviewCount(context)
	case "threads":
		return len(context.Threads)
	case "open_issues":
		return len(context.OpenIssues)
	case "labels":
		return len(context.Labels)
	default:
		return 0
	}
}

func reviewContextReviewCount(context ReviewContext) int {
	if len(context.Reviews) > 0 {
		return len(context.Reviews)
	}
	return len(context.ReviewRecords)
}

func reviewRecordSortKey(record ReviewContextReview) string {
	return strings.Join([]string{
		record.ID,
		record.ActorID,
		record.Actor,
		record.CreatedAt,
		record.UpdatedAt,
		record.CommitID,
		record.Status,
		record.Content,
	}, "\x00")
}

func reviewThreadSortKey(record ReviewContextThread) string {
	return strings.Join([]string{
		record.ID,
		record.ParentID,
		record.AuthorID,
		record.CreatedAt,
		record.UpdatedAt,
		record.Path,
		record.LineCode,
		record.Content,
	}, "\x00")
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
