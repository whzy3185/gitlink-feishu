package workflow

import (
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/gitlink-org/gitlink-cli/shortcuts/common"
)

type StaleFetchOptions struct {
	Owner         string
	Repo          string
	State         string
	StaleDays     int
	IssueLimit    int
	PRLimit       int
	IncludeIssues bool
	IncludePRs    bool
}

type stalePullRequestProbe struct {
	Input   StalePullRequestInput
	IssueID int
}

func FetchStaleInput(ctx *common.RuntimeContext, opts StaleFetchOptions) (StaleInput, []string, error) {
	owner, repo, err := resolveFetchRepo(ctx, opts.Owner, opts.Repo)
	if err != nil {
		return StaleInput{}, nil, fmt.Errorf("workflow +stale remote mode requires --owner and --repo or a Git remote: %w", err)
	}
	if !opts.IncludeIssues && !opts.IncludePRs {
		return StaleInput{}, nil, fmt.Errorf("workflow +stale requires at least one of --include-issues or --include-prs")
	}

	state := strings.TrimSpace(opts.State)
	if state == "" {
		state = "open"
	}
	input := StaleInput{
		Repository: fmt.Sprintf("%s/%s", owner, repo),
		Source:     "remote-read-only-fetch",
	}
	notes := []string{}
	successes := 0

	if opts.IncludeIssues {
		issues, err := fetchStaleIssueInputs(ctx, owner, repo, state, opts.IssueLimit)
		if err != nil {
			notes = append(notes, fmt.Sprintf("issue fetch failed: %v", err))
		} else {
			input.Issues = issues
			successes++
		}
	}

	if opts.IncludePRs {
		prs, prNotes, err := fetchStalePullRequestInputs(ctx, owner, repo, state, opts.PRLimit)
		notes = append(notes, prNotes...)
		if err != nil {
			notes = append(notes, fmt.Sprintf("pull request fetch failed: %v", err))
		} else {
			input.PullRequests = prs
			successes++
		}
	}

	if successes == 0 {
		return StaleInput{}, uniqueStrings(notes), fmt.Errorf("fetch stale input: all enabled sections failed")
	}
	return input, uniqueStrings(notes), nil
}

func fetchStaleIssueInputs(ctx *common.RuntimeContext, owner, repo, state string, limit int) ([]IssueInput, error) {
	if limit <= 0 {
		limit = 20
	}

	issues := make([]IssueInput, 0, limit)
	pageSize := minInt(limit, 100)
	for page := 1; len(issues) < limit; page++ {
		query := url.Values{}
		query.Set("state", state)
		query.Set("page", fmt.Sprintf("%d", page))
		query.Set("limit", fmt.Sprintf("%d", pageSize))

		env, err := ctx.CallAPIWithQuery("GET", workflowRepoPath(owner, repo)+"/issues", query)
		if err != nil {
			return nil, err
		}
		items := apiList(env.Data)
		if len(items) == 0 {
			break
		}

		rawCount := len(items)
		for _, raw := range items {
			issue, ok := normalizeIssueItem(raw)
			if !ok {
				continue
			}
			issues = append(issues, issue)
			if len(issues) >= limit {
				break
			}
		}
		if rawCount < pageSize {
			break
		}
	}
	return issues, nil
}

func fetchStalePullRequestInputs(ctx *common.RuntimeContext, owner, repo, state string, limit int) ([]StalePullRequestInput, []string, error) {
	if limit <= 0 {
		limit = 20
	}

	probes := make([]stalePullRequestProbe, 0, limit)
	pageSize := minInt(limit, 50)
	for page := 1; len(probes) < limit; page++ {
		query := url.Values{}
		query.Set("state", state)
		query.Set("page", fmt.Sprintf("%d", page))
		query.Set("limit", fmt.Sprintf("%d", pageSize))

		env, err := ctx.CallAPIWithQuery("GET", workflowRepoPath(owner, repo)+"/pulls", query)
		if err != nil {
			return nil, nil, err
		}
		items := apiList(env.Data)
		if len(items) == 0 {
			break
		}

		rawCount := len(items)
		for _, raw := range items {
			probe, ok := normalizeStalePullRequest(raw)
			if !ok {
				continue
			}
			probes = append(probes, probe)
			if len(probes) >= limit {
				break
			}
		}
		if rawCount < pageSize {
			break
		}
	}

	notes := []string{}
	prs := make([]StalePullRequestInput, 0, len(probes))
	for _, probe := range probes {
		pr := probe.Input
		if probe.IssueID > 0 {
			latestJournal, err := fetchIssueJournalActivity(ctx, probe.IssueID)
			if err != nil {
				notes = append(notes, fmt.Sprintf("PR #%d journal fallback failed: %v", pr.Number, err))
			} else if latestJournal.After(pr.LastActivityAt) {
				pr.LastActivityAt = latestJournal
				pr.ActivitySource = "issue_journal"
			}
		} else if pr.ActivitySource == "created_at" || pr.ActivitySource == "pr_created_unix" {
			notes = append(notes, fmt.Sprintf("PR #%d has no issue journal id; stale age fell back to creation time", pr.Number))
		}
		prs = append(prs, pr)
	}

	return prs, uniqueStrings(notes), nil
}

func normalizeStalePullRequest(raw interface{}) (stalePullRequestProbe, bool) {
	item, ok := raw.(map[string]interface{})
	if !ok {
		return stalePullRequestProbe{}, false
	}

	number := firstPRInt(item, "number", "index", "iid", "pull_request_number")
	title := firstPRString(item, "title", "subject")
	if number == 0 && strings.TrimSpace(title) == "" {
		return stalePullRequestProbe{}, false
	}

	issueObject, _ := item["issue"].(map[string]interface{})
	author := firstPRAuthor(item)
	if author == "" && issueObject != nil {
		author = apiAuthor(issueObject["author"])
	}
	comments := firstPRInt(item, "journals_count")
	if comments == 0 && issueObject != nil {
		comments = firstPRInt(issueObject, "journals_count", "comment_journals_count")
	}

	updatedAt := firstPRTime(item, "updated_at", "updated", "last_updated_at")
	createdAt := firstPRTime(item, "created_at", "pr_created_unix")
	lastActivity := apiLatestTime(updatedAt, createdAt)
	activitySource := ""
	switch {
	case !updatedAt.IsZero():
		activitySource = "updated_at"
	case !createdAt.IsZero():
		if item["pr_created_unix"] != nil {
			activitySource = "pr_created_unix"
		} else {
			activitySource = "created_at"
		}
	}

	return stalePullRequestProbe{
		Input: StalePullRequestInput{
			Number:         number,
			Title:          title,
			Author:         author,
			State:          firstNonEmpty(firstPRString(item, "status", "state"), "open"),
			URL:            firstPRString(item, "html_url", "url", "web_url"),
			BaseBranch:     firstPRBranch(item, "base_branch", "target_branch", "base"),
			HeadBranch:     firstPRBranch(item, "head_branch", "source_branch", "head"),
			CreatedAt:      createdAt,
			UpdatedAt:      updatedAt,
			LastActivityAt: lastActivity,
			CommentsCount:  comments,
			ActivitySource: activitySource,
		},
		IssueID: firstPRInt(issueObject, "id"),
	}, true
}

func fetchIssueJournalActivity(ctx *common.RuntimeContext, issueID int) (time.Time, error) {
	if issueID <= 0 {
		return time.Time{}, fmt.Errorf("issue id is required")
	}

	latest := time.Time{}
	for page := 1; ; page++ {
		query := url.Values{}
		query.Set("page", fmt.Sprintf("%d", page))
		query.Set("limit", "100")

		env, err := ctx.CallAPIWithQuery("GET", fmt.Sprintf("/issues/%d/journals", issueID), query)
		if err != nil {
			return time.Time{}, err
		}

		items := apiList(env.Data)
		if len(items) == 0 {
			break
		}

		pageCount := 0
		for _, raw := range items {
			item, ok := raw.(map[string]interface{})
			if !ok {
				continue
			}
			pageCount++
			latest = apiLatestTime(latest, journalActivityTime(item))
		}
		if pageCount < 100 {
			break
		}
	}
	return latest, nil
}

func journalActivityTime(item map[string]interface{}) time.Time {
	if item == nil {
		return time.Time{}
	}
	return apiLatestTime(
		apiTime(item["format_time"]),
		apiTime(item["updated_at"]),
		apiTime(item["created_at"]),
		apiTime(item["created_on"]),
		apiTime(item["updated_on"]),
	)
}
