package workflow

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/gitlink-org/gitlink-cli/shortcuts/common"
)

type RepoReportFetchOptions struct {
	Owner         string
	Repo          string
	IssueLimit    int
	PRLimit       int
	StaleDays     int
	IncludeIssues bool
	IncludePRs    bool
	IncludeHealth bool
}

func FetchRepoReportInput(ctx *common.RuntimeContext, opts RepoReportFetchOptions) (RepoReportInput, []ScoringNote, error) {
	owner, repo, err := resolveFetchRepo(ctx, opts.Owner, opts.Repo)
	if err != nil {
		return RepoReportInput{}, nil, fmt.Errorf("workflow +repo-report remote mode requires --owner and --repo or a Git remote: %w", err)
	}
	if opts.IssueLimit <= 0 {
		opts.IssueLimit = 20
	}
	if opts.PRLimit <= 0 {
		opts.PRLimit = 10
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
		issues, err := FetchIssuesForTriage(ctx, TriageFetchOptions{
			Owner: owner,
			Repo:  repo,
			State: "open",
			Limit: opts.IssueLimit,
			Page:  1,
		})
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

func fetchPRListForReport(ctx *common.RuntimeContext, owner, repo string, limit int) ([]PRSummaryInput, error) {
	if limit <= 0 {
		limit = 10
	}
	query := url.Values{}
	query.Set("state", "open")
	query.Set("page", "1")
	query.Set("limit", fmt.Sprintf("%d", limit))

	env, err := ctx.CallAPIWithQuery("GET", workflowRepoPath(owner, repo)+"/pulls", query)
	if err != nil {
		return nil, err
	}
	items := apiList(env.Data)
	inputs := make([]PRSummaryInput, 0, len(items))
	for _, raw := range items {
		item, ok := raw.(map[string]interface{})
		if !ok {
			continue
		}
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
		if len(inputs) >= limit {
			break
		}
	}
	return inputs, nil
}
