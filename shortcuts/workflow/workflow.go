package workflow

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/gitlink-org/gitlink-cli/shortcuts/common"
)

type TriageReport struct {
	Repository string         `json:"repository"`
	State      string         `json:"state"`
	Limit      int            `json:"limit"`
	DryRun     bool           `json:"dry_run"`
	Language   string         `json:"language"`
	Results    []TriageResult `json:"results"`
}

func Shortcuts() []*common.Shortcut {
	return []*common.Shortcut{
		newTriageShortcut(),
		newHealthShortcut(),
		newPRSummaryShortcut(),
		newRepoReportShortcut(),
	}
}

func newTriageShortcut() *common.Shortcut {
	return &common.Shortcut{
		Name:        "triage",
		Description: "Analyze issues with local workflow triage rules",
		Flags: []common.Flag{
			{Name: "from", Usage: "Read issue inputs from a JSON file. Supports a single issue, an array, or an object with an issues field"},
			{Name: "title", Short: "t", Usage: "Issue title for single-issue local analysis"},
			{Name: "body", Short: "b", Usage: "Issue body for single-issue local analysis"},
			{Name: "number", Short: "n", Usage: "Issue number for single-issue local analysis"},
			{Name: "author", Usage: "Issue author for single-issue local analysis"},
			{Name: "url", Usage: "Issue URL for single-issue local analysis"},
			{Name: "labels", Usage: "Comma-separated labels for single-issue local analysis"},
			{Name: "state", Short: "s", Usage: "Filter or assign issue state", Default: "open"},
			{Name: "page", Short: "p", Usage: "API page number for remote triage", Default: "1"},
			{Name: "limit", Short: "l", Usage: "Maximum issues to analyze", Default: "30"},
			{Name: "since", Usage: "Optional remote issue filter for updated time"},
			{Name: "dry-run", Usage: "Preview workflow recommendations without remote writes", Bool: true, Default: "true"},
			{Name: "lang", Usage: "Output language: en or zh-CN", Default: langEN},
		},
		Run: runTriage,
	}
}

func newHealthShortcut() *common.Shortcut {
	return &common.Shortcut{
		Name:        "health",
		Description: "Score repository health with local workflow rules",
		Flags: []common.Flag{
			{Name: "from", Usage: "Read health input from a JSON file"},
			{Name: "repository", Usage: "Repository name, for example owner/repo"},
			{Name: "open-issues", Usage: "Open issue count", Default: "0"},
			{Name: "open-prs", Usage: "Open pull request count", Default: "0"},
			{Name: "stale-issues", Usage: "Stale issue count", Default: "0"},
			{Name: "stale-prs", Usage: "Stale pull request count", Default: "0"},
			{Name: "recent-activity-known", Usage: "Whether recent activity is known", Bool: true, Default: "false"},
			{Name: "recent-activity-days", Usage: "Days since recent activity", Default: "0"},
			{Name: "release-known", Usage: "Whether release status is known", Bool: true, Default: "false"},
			{Name: "has-recent-release", Usage: "Whether a recent release exists", Bool: true, Default: "false"},
			{Name: "ci-known", Usage: "Whether CI status is known", Bool: true, Default: "false"},
			{Name: "ci-passing", Usage: "Whether CI is passing", Bool: true, Default: "false"},
			{Name: "has-readme", Usage: "Whether README exists", Bool: true, Default: "false"},
			{Name: "has-license", Usage: "Whether LICENSE exists", Bool: true, Default: "false"},
			{Name: "has-contributing", Usage: "Whether CONTRIBUTING exists", Bool: true, Default: "false"},
			{Name: "agent-readiness-known", Usage: "Whether agent readiness score is known", Bool: true, Default: "false"},
			{Name: "agent-readiness-score", Usage: "Agent readiness score from 0 to 10", Default: "0"},
			{Name: "stale-days", Usage: "Days before an issue or PR is considered stale", Default: "30"},
			{Name: "lang", Usage: "Output language: en or zh-CN", Default: langEN},
		},
		Run: runHealth,
	}
}

func runTriage(ctx *common.RuntimeContext) error {
	lang := normalizeLang(ctx.Arg("lang"))
	limit, err := parseIntArg(ctx.Arg("limit"), 30, "limit")
	if err != nil {
		return err
	}
	state := ctx.Arg("state")
	if state == "" {
		state = "open"
	}

	if hasLocalTriageInput(ctx) {
		issues, err := collectIssuesFromArgs(ctx)
		if err != nil {
			return err
		}

		filtered := filterIssueInputs(issues, state, limit)
		results := make([]TriageResult, 0, len(filtered))
		for _, issue := range filtered {
			results = append(results, AnalyzeIssue(issue, lang))
		}

		report := TriageReport{
			Repository: repositoryFromContext(ctx, ""),
			State:      state,
			Limit:      limit,
			DryRun:     parseBoolArg(ctx.Arg("dry-run")),
			Language:   lang,
			Results:    results,
		}
		return renderTriageReport(os.Stdout, report, ctx.Format)
	}

	issues, err := FetchIssuesForTriage(ctx, TriageFetchOptions{
		State:  state,
		Limit:  limit,
		Page:   mustParseInt(ctx.Arg("page"), 1),
		Labels: parseCSV(ctx.Arg("labels")),
		Since:  ctx.Arg("since"),
	})
	if err != nil {
		return fmt.Errorf("%w\nhint: use --title or --from issues.json for local rule analysis", err)
	}
	results := make([]TriageResult, 0, len(issues))
	for _, issue := range issues {
		results = append(results, AnalyzeIssue(issue, lang))
	}
	if err := ctx.ResolveOwnerRepo(); err != nil {
		return fmt.Errorf("workflow +triage remote mode requires --owner and --repo or a Git remote: %w", err)
	}
	report := TriageReport{
		Repository: repositoryFromContext(ctx, ""),
		State:      state,
		Limit:      limit,
		DryRun:     parseBoolArg(ctx.Arg("dry-run")),
		Language:   lang,
		Results:    results,
	}
	return renderTriageReport(os.Stdout, report, ctx.Format)
}

func runHealth(ctx *common.RuntimeContext) error {
	lang := normalizeLang(ctx.Arg("lang"))
	if hasLocalHealthInput(ctx) {
		input, err := collectHealthFromArgs(ctx)
		if err != nil {
			return err
		}
		input.Repository = repositoryFromContext(ctx, input.Repository)

		result := ScoreHealth(input, lang)
		return renderHealthResult(os.Stdout, result, ctx.Format)
	}

	if err := ctx.ResolveOwnerRepo(); err != nil {
		return fmt.Errorf("workflow +health remote mode requires --owner and --repo or a Git remote: %w", err)
	}
	input, notes, err := FetchHealthInput(ctx, HealthFetchOptions{
		StaleDays:      mustParseInt(ctx.Arg("stale-days"), 30),
		IncludeCI:      true,
		IncludeRelease: true,
		IncludeDocs:    true,
	})
	if err != nil {
		return err
	}
	result := ScoreHealth(input, lang)
	result.ScoringNotes = append(notes, result.ScoringNotes...)
	return renderHealthResult(os.Stdout, result, ctx.Format)
}

func collectIssuesFromArgs(ctx *common.RuntimeContext) ([]IssueInput, error) {
	if path := ctx.Arg("from"); path != "" {
		return readIssueInputs(path)
	}
	if strings.TrimSpace(ctx.Arg("title")) == "" {
		return nil, fmt.Errorf("workflow +triage currently requires --from issues.json or --title for local rule analysis")
	}
	number, err := parseIntArg(ctx.Arg("number"), 0, "number")
	if err != nil {
		return nil, err
	}
	state := ctx.Arg("state")
	if state == "" {
		state = "open"
	}
	return []IssueInput{{
		Number: number,
		Title:  ctx.Arg("title"),
		Body:   ctx.Arg("body"),
		State:  state,
		Author: ctx.Arg("author"),
		URL:    ctx.Arg("url"),
		Labels: parseCSV(ctx.Arg("labels")),
	}}, nil
}

func hasLocalTriageInput(ctx *common.RuntimeContext) bool {
	if strings.TrimSpace(ctx.Arg("from")) != "" || strings.TrimSpace(ctx.Arg("title")) != "" {
		return true
	}
	// When user passes triage-specific flags without --title or --from,
	// treat it as local input so argument validation kicks in early.
	for _, name := range []string{"body", "number", "author", "url", "labels"} {
		if strings.TrimSpace(ctx.Arg(name)) != "" {
			return true
		}
	}
	return false
}

func hasLocalHealthInput(ctx *common.RuntimeContext) bool {
	if strings.TrimSpace(ctx.Arg("from")) != "" || strings.TrimSpace(ctx.Arg("repository")) != "" {
		return true
	}
	// When user passes any health-flag value (even invalid ones like
	// --open-issues abc), stay in local mode so parseIntArg validates them.
	for _, name := range []string{"open-issues", "stale-issues", "open-prs", "stale-prs", "recent-activity-known", "recent-activity-days", "release-known", "has-recent-release", "ci-known", "ci-passing", "has-readme", "has-license", "has-contributing", "agent-readiness-known", "agent-readiness-score"} {
		if strings.TrimSpace(ctx.Arg(name)) != "" {
			return true
		}
	}
	return false
}

func collectHealthFromArgs(ctx *common.RuntimeContext) (HealthInput, error) {
	if path := ctx.Arg("from"); path != "" {
		input, err := readHealthInput(path)
		if err != nil {
			return HealthInput{}, err
		}
		return input, nil
	}

	openIssues, err := parseIntArg(ctx.Arg("open-issues"), 0, "open-issues")
	if err != nil {
		return HealthInput{}, err
	}
	openPRs, err := parseIntArg(ctx.Arg("open-prs"), 0, "open-prs")
	if err != nil {
		return HealthInput{}, err
	}
	staleIssues, err := parseIntArg(ctx.Arg("stale-issues"), 0, "stale-issues")
	if err != nil {
		return HealthInput{}, err
	}
	stalePRs, err := parseIntArg(ctx.Arg("stale-prs"), 0, "stale-prs")
	if err != nil {
		return HealthInput{}, err
	}
	recentActivityDays, err := parseIntArg(ctx.Arg("recent-activity-days"), 0, "recent-activity-days")
	if err != nil {
		return HealthInput{}, err
	}
	agentReadinessScore, err := parseIntArg(ctx.Arg("agent-readiness-score"), 0, "agent-readiness-score")
	if err != nil {
		return HealthInput{}, err
	}

	return HealthInput{
		Repository:          ctx.Arg("repository"),
		OpenIssues:          openIssues,
		OpenPRs:             openPRs,
		StaleIssues:         staleIssues,
		StalePRs:            stalePRs,
		RecentActivityKnown: parseBoolArg(ctx.Arg("recent-activity-known")),
		RecentActivityDays:  recentActivityDays,
		ReleaseKnown:        parseBoolArg(ctx.Arg("release-known")),
		HasRecentRelease:    parseBoolArg(ctx.Arg("has-recent-release")),
		CIKnown:             parseBoolArg(ctx.Arg("ci-known")),
		CIPassing:           parseBoolArg(ctx.Arg("ci-passing")),
		HasReadme:           parseBoolArg(ctx.Arg("has-readme")),
		HasLicense:          parseBoolArg(ctx.Arg("has-license")),
		HasContributing:     parseBoolArg(ctx.Arg("has-contributing")),
		AgentReadinessKnown: parseBoolArg(ctx.Arg("agent-readiness-known")),
		AgentReadinessScore: agentReadinessScore,
	}, nil
}

func readIssueInputs(path string) ([]IssueInput, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read issue inputs: %w", err)
	}

	var issues []IssueInput
	if err := json.Unmarshal(data, &issues); err == nil {
		return issues, nil
	}

	var wrapper struct {
		Issues []IssueInput `json:"issues"`
	}
	if err := json.Unmarshal(data, &wrapper); err == nil && wrapper.Issues != nil {
		return wrapper.Issues, nil
	}

	var issue IssueInput
	if err := json.Unmarshal(data, &issue); err == nil && issue.Title != "" {
		return []IssueInput{issue}, nil
	}

	return nil, fmt.Errorf("parse issue inputs: expected a single issue, an array, or an object with an issues field")
}

func readHealthInput(path string) (HealthInput, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return HealthInput{}, fmt.Errorf("read health input: %w", err)
	}
	var input HealthInput
	if err := json.Unmarshal(data, &input); err != nil {
		return HealthInput{}, fmt.Errorf("parse health input: %w", err)
	}
	return input, nil
}

func filterIssueInputs(issues []IssueInput, state string, limit int) []IssueInput {
	filtered := make([]IssueInput, 0, len(issues))
	for _, issue := range issues {
		if state != "" && state != "all" && issue.State != "" && !strings.EqualFold(issue.State, state) {
			continue
		}
		filtered = append(filtered, issue)
		if limit > 0 && len(filtered) >= limit {
			break
		}
	}
	return filtered
}

func repositoryFromContext(ctx *common.RuntimeContext, fallback string) string {
	if ctx.Owner != "" && ctx.Repo != "" {
		return ctx.Owner + "/" + ctx.Repo
	}
	if fallback != "" {
		return fallback
	}
	return "local"
}

func parseCSV(value string) []string {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	parts := strings.Split(value, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			result = append(result, part)
		}
	}
	return result
}

func parseBoolArg(value string) bool {
	parsed, err := strconv.ParseBool(strings.TrimSpace(value))
	return err == nil && parsed
}

func parseIntArg(value string, defaultValue int, name string) (int, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return defaultValue, nil
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return 0, fmt.Errorf("invalid --%s %q: must be an integer", name, value)
	}
	if parsed < 0 {
		return 0, fmt.Errorf("invalid --%s %q: must be non-negative", name, value)
	}
	return parsed, nil
}

func mustParseInt(value string, defaultValue int) int {
	parsed, err := parseIntArg(value, defaultValue, "value")
	if err != nil {
		return defaultValue
	}
	return parsed
}
