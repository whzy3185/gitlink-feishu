package workflow

import (
	"fmt"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/gitlink-org/gitlink-cli/shortcuts/common"
)

const (
	reviewStandardFormal          = "formal_review"
	reviewStandardReviewerJournal = "reviewer_journal_feedback"
	reviewStandardUnreviewed      = "unreviewed"

	actorSubmitter   = "submitter"
	actorReviewer    = "reviewer"
	actorParticipant = "participant"
	actorBot         = "bot"
	actorSystem      = "system"
	actorUnknown     = "unknown"
)

type normalizedPRReview struct {
	Actor  string
	Status string
	At     time.Time
}

func fetchPRReviewAudit(ctx *common.RuntimeContext, owner, repo string, prs []PRSummaryInput, maxItems int) (*RepoPRReviewAudit, []ScoringNote) {
	if maxItems > 0 && len(prs) > maxItems {
		prs = prs[:maxItems]
	}
	audit := &RepoPRReviewAudit{
		Source:       "remote-read-only-fetch:reviews-and-journals",
		PullRequests: make([]PRReviewAudit, 0, len(prs)),
	}
	notes := []ScoringNote{}
	for _, pr := range prs {
		item, itemNotes := fetchOnePRReviewAudit(ctx, owner, repo, pr)
		notes = append(notes, itemNotes...)
		audit.PullRequests = append(audit.PullRequests, item)
		audit.Audited++
		if item.Reviewed {
			audit.Reviewed++
		} else {
			audit.Unreviewed++
		}
		if item.NeedsReReview {
			audit.NeedsReReview++
		}
		audit.FormalReviews += item.FormalReviewCount
		audit.ReviewerComments += item.ReviewerComments
		audit.SubmitterComments += item.SubmitterComments
		audit.ParticipantComments += item.ParticipantComments
		audit.BotEvents += item.BotEvents
		audit.SystemEvents += item.SystemEvents
		audit.UnknownActorEvents += item.UnknownActorEvents
		if len(item.Notes) > 0 {
			audit.Errors += len(item.Notes)
		}
	}
	return audit, uniqueScoringNotes(notes)
}

func fetchOnePRReviewAudit(ctx *common.RuntimeContext, owner, repo string, pr PRSummaryInput) (PRReviewAudit, []ScoringNote) {
	result := PRReviewAudit{
		Number:             pr.Number,
		Author:             pr.Author,
		IssueID:            pr.IssueID,
		ReviewStandard:     reviewStandardUnreviewed,
		FormalReviewStatus: "unreviewed",
	}
	notes := []ScoringNote{}
	if result.Number <= 0 {
		result.Notes = append(result.Notes, "missing PR number")
		return result, []ScoringNote{{Metric: "repo_report_pr_review_audit", Note: "skipped PR review audit: missing PR number"}}
	}

	if result.IssueID == 0 || strings.TrimSpace(result.Author) == "" {
		base, err := fetchPRBase(ctx, owner, repo, result.Number)
		if err != nil {
			note := fmt.Sprintf("PR #%d base detail unavailable for review audit: %v", result.Number, err)
			result.Notes = append(result.Notes, note)
			notes = append(notes, ScoringNote{Metric: "repo_report_pr_review_audit", Note: note})
		} else {
			if result.IssueID == 0 {
				result.IssueID = base.IssueID
			}
			if strings.TrimSpace(result.Author) == "" {
				result.Author = base.Author
			}
			if pr.CreatedAt.IsZero() {
				pr.CreatedAt = base.CreatedAt
			}
			if pr.UpdatedAt.IsZero() {
				pr.UpdatedAt = base.UpdatedAt
			}
		}
	}

	reviews, err := fetchFormalPRReviews(ctx, owner, repo, result.Number)
	if err != nil {
		note := fmt.Sprintf("PR #%d formal reviews unavailable: %v", result.Number, err)
		result.Notes = append(result.Notes, note)
		notes = append(notes, ScoringNote{Metric: "repo_report_pr_review_audit", Note: note})
	} else {
		result.FormalReviewCount = len(reviews)
		result.FormalReviewStatus = summarizeFormalReviewStatus(reviews)
		result.Reviewers = uniqueReviewerNames(reviews)
		if latest := latestReviewTime(reviews); !latest.IsZero() {
			result.LatestReviewerAt = latest.Format(time.RFC3339)
		}
		if result.FormalReviewCount > 0 {
			result.Reviewed = true
			result.ReviewStandard = reviewStandardFormal
		}
	}

	reviewerSet := reviewerSet(reviews)
	var latestReviewerAt time.Time
	if result.LatestReviewerAt != "" {
		latestReviewerAt = apiTime(result.LatestReviewerAt)
	}
	var latestSubmitterAt time.Time
	if result.IssueID == 0 {
		note := fmt.Sprintf("PR #%d conversation journal skipped: missing associated issue id", result.Number)
		result.Notes = append(result.Notes, note)
		notes = append(notes, ScoringNote{Metric: "repo_report_pr_review_audit", Note: note})
	} else {
		journals, err := fetchPRJournals(ctx, owner, repo, result.IssueID)
		if err != nil {
			note := fmt.Sprintf("PR #%d conversation journal unavailable: %v", result.Number, err)
			result.Notes = append(result.Notes, note)
			notes = append(notes, ScoringNote{Metric: "repo_report_pr_review_audit", Note: note})
		} else {
			for _, journal := range journals {
				switch classifyJournalActor(journal, result.Author, reviewerSet) {
				case actorReviewer:
					result.ReviewerComments++
					latestReviewerAt = apiLatestTime(latestReviewerAt, journalTime(journal))
				case actorSubmitter:
					result.SubmitterComments++
					latestSubmitterAt = apiLatestTime(latestSubmitterAt, journalTime(journal))
				case actorParticipant:
					result.ParticipantComments++
				case actorBot:
					result.BotEvents++
				case actorSystem:
					result.SystemEvents++
				default:
					result.UnknownActorEvents++
				}
			}
			if !result.Reviewed && result.ReviewerComments > 0 {
				result.Reviewed = true
				result.ReviewStandard = reviewStandardReviewerJournal
			}
		}
	}
	if !latestReviewerAt.IsZero() {
		result.LatestReviewerAt = latestReviewerAt.Format(time.RFC3339)
	}
	if !latestSubmitterAt.IsZero() {
		result.LatestSubmitterAt = latestSubmitterAt.Format(time.RFC3339)
	}
	latestCommitAt := latestCommitTime(pr.Commits)
	if !latestCommitAt.IsZero() {
		result.LatestCommitAt = latestCommitAt.Format(time.RFC3339)
	}
	if !pr.UpdatedAt.IsZero() {
		result.LatestPRUpdateAt = pr.UpdatedAt.Format(time.RFC3339)
	}
	result.NeedsReReview = needsReReview(latestReviewerAt, latestSubmitterAt, latestCommitAt, pr.UpdatedAt)

	return result, notes
}

func fetchFormalPRReviews(ctx *common.RuntimeContext, owner, repo string, number int) ([]normalizedPRReview, error) {
	items, err := fetchListItems(ctx, prPath(owner, repo, number)+"/reviews", url.Values{}, 50, 0)
	if err != nil {
		return nil, err
	}
	reviews := make([]normalizedPRReview, 0, len(items))
	for _, item := range items {
		review := normalizedPRReview{
			Actor:  firstReviewActor(item),
			Status: strings.ToLower(strings.TrimSpace(firstPRString(item, "status", "state", "review_status"))),
			At:     journalTime(item),
		}
		if review.Actor == "" && review.Status == "" {
			continue
		}
		if review.Status == "" {
			review.Status = "common"
		}
		reviews = append(reviews, review)
	}
	return reviews, nil
}

func latestReviewTime(reviews []normalizedPRReview) time.Time {
	var latest time.Time
	for _, review := range reviews {
		latest = apiLatestTime(latest, review.At)
	}
	return latest
}

func journalTime(item map[string]interface{}) time.Time {
	return apiLatestTime(
		firstPRTime(item, "updated_at", "updatedAt"),
		firstPRTime(item, "created_at", "createdAt"),
	)
}

func latestCommitTime(commits []PRCommit) time.Time {
	var latest time.Time
	for _, commit := range commits {
		latest = apiLatestTime(latest, commit.Date)
	}
	return latest
}

func needsReReview(latestReviewerAt, latestSubmitterAt, latestCommitAt, latestPRUpdateAt time.Time) bool {
	if latestReviewerAt.IsZero() {
		return false
	}
	return isAfter(latestSubmitterAt, latestReviewerAt) ||
		isAfter(latestCommitAt, latestReviewerAt) ||
		isAfter(latestPRUpdateAt, latestReviewerAt)
}

func isAfter(value, baseline time.Time) bool {
	return !value.IsZero() && !baseline.IsZero() && value.After(baseline)
}

func fetchPRJournals(ctx *common.RuntimeContext, owner, repo string, issueID int) ([]map[string]interface{}, error) {
	return fetchListItems(ctx, fmt.Sprintf("/v1/%s/%s/issues/%d/journals", owner, repo, issueID), url.Values{}, 50, 0)
}

func firstReviewActor(item map[string]interface{}) string {
	for _, key := range []string{"reviewer", "user", "author", "creator"} {
		if value, ok := item[key]; ok {
			if actor := apiAuthor(value); actor != "" {
				return actor
			}
		}
	}
	return ""
}

func summarizeFormalReviewStatus(reviews []normalizedPRReview) string {
	if len(reviews) == 0 {
		return "unreviewed"
	}
	hasApproved := false
	hasCommon := false
	for _, review := range reviews {
		switch strings.ToLower(strings.TrimSpace(review.Status)) {
		case "rejected", "reject", "changes_requested", "request_changes":
			return "rejected"
		case "approved", "approve":
			hasApproved = true
		default:
			hasCommon = true
		}
	}
	if hasApproved {
		return "approved"
	}
	if hasCommon {
		return "common"
	}
	return "reviewed"
}

func uniqueReviewerNames(reviews []normalizedPRReview) []string {
	set := map[string]string{}
	for _, review := range reviews {
		key := normalizeActorID(review.Actor)
		if key != "" {
			set[key] = review.Actor
		}
	}
	keys := make([]string, 0, len(set))
	for key := range set {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	out := make([]string, 0, len(keys))
	for _, key := range keys {
		out = append(out, set[key])
	}
	return out
}

func reviewerSet(reviews []normalizedPRReview) map[string]bool {
	set := map[string]bool{}
	for _, review := range reviews {
		if key := normalizeActorID(review.Actor); key != "" {
			set[key] = true
		}
	}
	return set
}

func classifyJournalActor(item map[string]interface{}, author string, reviewers map[string]bool) string {
	category := strings.ToLower(strings.TrimSpace(firstPRString(item, "operate_category", "category", "type", "event")))
	content := strings.TrimSpace(firstPRString(item, "notes", "note", "body", "content", "operate_content"))
	actor := firstReviewActor(item)
	if isSystemJournalEvent(category, content, actor) {
		return actorSystem
	}
	if actor == "" {
		return actorUnknown
	}
	if isBotActor(actor) {
		return actorBot
	}
	actorID := normalizeActorID(actor)
	if actorID != "" && actorID == normalizeActorID(author) {
		return actorSubmitter
	}
	if actorID != "" && reviewers[actorID] {
		return actorReviewer
	}
	return actorParticipant
}

func isSystemJournalEvent(category, content, actor string) bool {
	if strings.TrimSpace(content) == "" {
		return true
	}
	switch category {
	case "status", "state", "system", "relation", "assignee", "label", "milestone":
		return true
	}
	if strings.TrimSpace(actor) == "" && category != "" {
		return true
	}
	return false
}

func isBotActor(actor string) bool {
	actor = strings.ToLower(strings.TrimSpace(actor))
	return strings.Contains(actor, "bot") || strings.Contains(actor, "机器人") || strings.Contains(actor, "automation")
}

func normalizeActorID(actor string) string {
	return strings.ToLower(strings.TrimSpace(actor))
}
