package workflow

import (
	"fmt"
	"strings"

	"github.com/gitlink-org/gitlink-cli/shortcuts/common"
)

type RepoReportFetchOptions struct {
	Owner                string
	Repo                 string
	IssueLimit           int
	PRLimit              int
	StaleDays            int
	PRReviewAuditLimit   int
	IncludeIssues        bool
	IncludePRs           bool
	IncludePRLifecycle   bool
	IncludePRReviewAudit bool
	IncludeHealth        bool
}

func FetchRepoReportInput(ctx *common.RuntimeContext, opts RepoReportFetchOptions) (RepoReportInput, []ScoringNote, error) {
	owner, repo, err := resolveFetchRepo(ctx, opts.Owner, opts.Repo)
	if err != nil {
		return RepoReportInput{}, nil, fmt.Errorf("workflow +repo-report remote mode requires --owner and --repo or a Git remote: %w", err)
	}
	if opts.StaleDays <= 0 {
		opts.StaleDays = 30
	}

	input := RepoReportInput{
		Repository: fmt.Sprintf("%s/%s", owner, repo),
		Source:     "remote-read-only-fetch",
	}
	notes := []ScoringNote{}
	successes := 0

	if opts.IncludeHealth {
		health, healthNotes, err := FetchHealthInput(ctx, HealthFetchOptions{
			Owner:          owner,
			Repo:           repo,
			StaleDays:      opts.StaleDays,
			IncludeCI:      true,
			IncludeRelease: true,
			IncludeDocs:    true,
		})
		notes = append(notes, healthNotes...)
		if err != nil {
			notes = append(notes, ScoringNote{Metric: "repo_report_health", Note: fmt.Sprintf("health fetch failed: %v", err)})
		} else {
			input.Health = &health
			successes++
		}
	}

	if opts.IncludeIssues {
		issues, err := fetchIssueListForReport(ctx, owner, repo, opts.IssueLimit)
		if err != nil {
			notes = append(notes, ScoringNote{Metric: "repo_report_issues", Note: fmt.Sprintf("issue fetch failed: %v", err)})
		} else {
			input.Issues = issues
			successes++
		}
	}

	if opts.IncludePRs {
		prs, err := fetchPRListForReport(ctx, owner, repo, opts.PRLimit)
		if err != nil {
			notes = append(notes, ScoringNote{Metric: "repo_report_prs", Note: fmt.Sprintf("pull request list fetch failed: %v", err)})
		} else {
			input.PullRequests = prs
			successes++
			if opts.IncludePRLifecycle {
				lifecycle, lifecycleNotes := fetchPRLifecycle(ctx, owner, repo)
				input.PRLifecycle = lifecycle
				notes = append(notes, lifecycleNotes...)
			}
			if opts.IncludePRReviewAudit {
				audit, auditNotes := fetchPRReviewAudit(ctx, owner, repo, prs, opts.PRReviewAuditLimit)
				input.PRReviewAudit = audit
				notes = append(notes, auditNotes...)
			}
			if len(prs) > 0 {
				notes = append(notes, ScoringNote{
					Metric: "repo_report_prs",
					Note:   "PR report uses list metadata only; changed files and commits require workflow +pr-summary with a PR number.",
				})
			}
		}
	}

	if successes == 0 {
		if len(notes) == 0 {
			notes = append(notes, ScoringNote{Metric: "repo_report", Note: "no report sections were enabled or fetched"})
		}
		return RepoReportInput{}, uniqueScoringNotes(notes), fmt.Errorf("fetch repo report: all enabled sections failed")
	}
	return input, uniqueScoringNotes(notes), nil
}

func fetchPRLifecycle(ctx *common.RuntimeContext, owner, repo string) (*RepoPRLifecycle, []ScoringNote) {
	states := []struct {
		name  string
		value *int
	}{
		{name: "open"},
		{name: "merged"},
		{name: "closed"},
	}
	lifecycle := &RepoPRLifecycle{Source: "remote-read-only-fetch:list-totals"}
	states[0].value = &lifecycle.Open
	states[1].value = &lifecycle.Merged
	states[2].value = &lifecycle.ClosedOrRejected
	notes := []ScoringNote{}
	successes := 0
	for _, state := range states {
		total, err := fetchPRStateTotal(ctx, owner, repo, state.name)
		if err != nil {
			notes = append(notes, ScoringNote{
				Metric: "repo_report_pr_lifecycle",
				Note:   fmt.Sprintf("%s PR total unavailable: %v", state.name, err),
			})
			continue
		}
		*state.value = total
		successes++
	}
	if successes == 0 {
		return nil, notes
	}
	lifecycle.Total = lifecycle.Open + lifecycle.Merged + lifecycle.ClosedOrRejected
	return lifecycle, notes
}

func fetchPRStateTotal(ctx *common.RuntimeContext, owner, repo, state string) (int, error) {
	query := pullListQuery(state)
	query.Set("page", "1")
	query.Set("limit", "1")
	env, err := ctx.CallAPIWithQuery("GET", workflowRepoPath(owner, repo)+"/pulls", query)
	if err != nil {
		return 0, err
	}
	if env.Meta != nil && env.Meta.TotalCount > 0 {
		return env.Meta.TotalCount, nil
	}
	if total := apiListTotal(env.Data); total > 0 {
		return total, nil
	}
	return len(apiList(env.Data)), nil
}

func fetchPRListForReport(ctx *common.RuntimeContext, owner, repo string, limit int) ([]PRSummaryInput, error) {
	pageSize := reportPageSize(limit)
	items, err := fetchListItems(ctx, workflowRepoPath(owner, repo)+"/pulls", pullListQuery("open"), pageSize, limit)
	if err != nil {
		return nil, err
	}
	inputs := make([]PRSummaryInput, 0, len(items))
	for _, item := range items {
		input, ok := normalizePRSummaryItem(item)
		if !ok {
			continue
		}
		input.Repository = fmt.Sprintf("%s/%s", owner, repo)
		input.Source = "remote-read-only-fetch:list-metadata"
		if strings.TrimSpace(input.State) == "" {
			input.State = "open"
		}
		inputs = append(inputs, input)
		if limit > 0 && len(inputs) >= limit {
			break
		}
	}
	return inputs, nil
}

func fetchIssueListForReport(ctx *common.RuntimeContext, owner, repo string, limit int) ([]IssueInput, error) {
	pageSize := reportPageSize(limit)
	items, err := fetchListItems(ctx, workflowRepoPath(owner, repo)+"/issues", issueListQuery("open"), pageSize, limit)
	if err != nil {
		return nil, err
	}
	issues := make([]IssueInput, 0, len(items))
	for _, item := range items {
		issue, ok := normalizeIssueItem(item)
		if !ok {
			continue
		}
		issues = append(issues, issue)
		if limit > 0 && len(issues) >= limit {
			break
		}
	}
	return issues, nil
}

func reportPageSize(limit int) int {
	const maxPageSize = 50
	if limit > 0 && limit < maxPageSize {
		return limit
	}
	return maxPageSize
}
