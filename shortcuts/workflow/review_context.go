package workflow

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/gitlink-org/gitlink-cli/cmd/cmdutil"
	"github.com/gitlink-org/gitlink-cli/shortcuts/common"
)

type ReviewContextOptions struct {
	Owner          string
	Repo           string
	Number         int
	IssueLimit     int
	LabelLimit     int
	ThreadLimit    int
	IncludeRepo    bool
	IncludePR      bool
	IncludeFiles   bool
	IncludeReviews bool
	IncludeThreads bool
	IncludeIssues  bool
	IncludeLabels  bool
}

func newReviewContextShortcut() *common.Shortcut {
	return &common.Shortcut{
		Name:        "review-context",
		Description: "Build read-only PR review context from GitLink or local JSON",
		Flags: []common.Flag{
			{Name: "from", Usage: "Read review context from a local JSON file"},
			{Name: "number", Short: "n", Usage: "Pull request number"},
			{Name: "issue-limit", Usage: "Maximum open issues to include", Default: "20"},
			{Name: "label-limit", Usage: "Maximum labels to include", Default: "50"},
			{Name: "thread-limit", Usage: "Maximum review thread records to include", Default: "100"},
			{Name: "include-repo", Usage: "Include repository info", Bool: true, Default: "true"},
			{Name: "include-pr", Usage: "Include pull request details", Bool: true, Default: "true"},
			{Name: "include-files", Usage: "Include pull request changed files", Bool: true, Default: "true"},
			{Name: "include-reviews", Usage: "Include pull request reviews", Bool: true, Default: "true"},
			{Name: "include-threads", Usage: "Include pull request review threads", Bool: true, Default: "true"},
			{Name: "include-issues", Usage: "Include open issue context", Bool: true, Default: "true"},
			{Name: "include-labels", Usage: "Include issue labels", Bool: true, Default: "true"},
		},
		Run: runReviewContext,
	}
}

func runReviewContext(ctx *common.RuntimeContext) error {
	var context ReviewContext
	if path := strings.TrimSpace(ctx.Arg("from")); path != "" {
		loaded, err := readReviewContextInput(path)
		if err != nil {
			return err
		}
		context = loaded
		finalizeReviewContext(&context, time.Now().UTC())
		return renderReviewContextToStdout(ctx, context)
	}

	number, err := parseIntArg(ctx.Arg("number"), 0, "number")
	if err != nil {
		return err
	}
	if number <= 0 {
		return fmt.Errorf("workflow +review-context requires --number with --owner and --repo for read-only fetch")
	}
	issueLimit, err := parseIntArg(ctx.Arg("issue-limit"), 20, "issue-limit")
	if err != nil {
		return err
	}
	labelLimit, err := parseIntArg(ctx.Arg("label-limit"), 50, "label-limit")
	if err != nil {
		return err
	}
	threadLimit, err := parseIntArg(ctx.Arg("thread-limit"), 100, "thread-limit")
	if err != nil {
		return err
	}

	context, err = FetchReviewContext(ctx, ReviewContextOptions{
		Number:         number,
		IssueLimit:     issueLimit,
		LabelLimit:     labelLimit,
		ThreadLimit:    threadLimit,
		IncludeRepo:    parseBoolDefault(ctx.Arg("include-repo"), true),
		IncludePR:      parseBoolDefault(ctx.Arg("include-pr"), true),
		IncludeFiles:   parseBoolDefault(ctx.Arg("include-files"), true),
		IncludeReviews: parseBoolDefault(ctx.Arg("include-reviews"), true),
		IncludeThreads: parseBoolDefault(ctx.Arg("include-threads"), true),
		IncludeIssues:  parseBoolDefault(ctx.Arg("include-issues"), true),
		IncludeLabels:  parseBoolDefault(ctx.Arg("include-labels"), true),
	})
	if err != nil {
		return err
	}
	return renderReviewContextToStdout(ctx, context)
}

func renderReviewContextToStdout(ctx *common.RuntimeContext, context ReviewContext) error {
	format := ctx.Format
	if strings.TrimSpace(cmdutil.Format) == "" {
		format = "json"
	}
	rendered, err := RenderReviewContext(context, format)
	if err != nil {
		return err
	}
	_, err = fmt.Fprint(os.Stdout, rendered)
	return err
}

func readReviewContextInput(path string) (ReviewContext, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return ReviewContext{}, fmt.Errorf("read review context input: %w", err)
	}
	var context ReviewContext
	if err := json.Unmarshal(data, &context); err != nil {
		return ReviewContext{}, fmt.Errorf("parse review context input: %w", err)
	}
	if strings.TrimSpace(context.Repository) == "" {
		return ReviewContext{}, fmt.Errorf("parse review context input: repository is required")
	}
	if context.PullRequest <= 0 {
		return ReviewContext{}, fmt.Errorf("parse review context input: pull_request must be positive")
	}
	if strings.TrimSpace(context.Source) == "" {
		context.Source = "local-json"
	}
	return context, nil
}

func FetchReviewContext(ctx *common.RuntimeContext, opts ReviewContextOptions) (ReviewContext, error) {
	owner, repo, err := resolveFetchRepo(ctx, opts.Owner, opts.Repo)
	if err != nil {
		return ReviewContext{}, fmt.Errorf("workflow +review-context remote mode requires --owner and --repo or a Git remote: %w", err)
	}
	if opts.Number <= 0 {
		return ReviewContext{}, fmt.Errorf("pull request number is required")
	}
	if opts.IssueLimit <= 0 {
		opts.IssueLimit = 20
	}
	if opts.LabelLimit <= 0 {
		opts.LabelLimit = 50
	}
	if opts.ThreadLimit <= 0 {
		opts.ThreadLimit = 100
	}

	result := ReviewContext{
		SchemaVersion: reviewContextSchemaVersion,
		Repository:    fmt.Sprintf("%s/%s", owner, repo),
		PullRequest:   opts.Number,
		Source:        "shortcut-backed-read-only-fetch",
		Sections:      []string{},
		Notes:         []ScoringNote{},
		fileLimit:     100,
		reviewLimit:   100,
		threadLimit:   opts.ThreadLimit,
	}
	successes := 0

	if opts.IncludeRepo {
		if info, err := fetchRepoInfo(ctx, owner, repo); err != nil {
			result.Notes = append(result.Notes, ScoringNote{Metric: "repo_info", Note: fmt.Sprintf("repo +info equivalent failed: %v", err)})
		} else {
			result.RepositoryInfo = info
			result.Sections = append(result.Sections, "repo_info")
			successes++
		}
	}
	if opts.IncludePR {
		if pr, err := fetchReviewContextPR(ctx, owner, repo, opts.Number); err != nil {
			result.Notes = append(result.Notes, ScoringNote{Metric: "pr_view", Note: fmt.Sprintf("pr +view equivalent failed: %v", err)})
		} else {
			result.PR = pr
			result.Sections = append(result.Sections, "pr")
			successes++
		}
	}
	if opts.IncludeFiles {
		if files, err := fetchReviewContextList(ctx, fmt.Sprintf("%s/pulls/%d/files", plainRepoPath(owner, repo), opts.Number), nil, 100); err != nil {
			result.Notes = append(result.Notes, ScoringNote{Metric: "pr_files", Note: fmt.Sprintf("pr +files equivalent failed: %v", err)})
		} else {
			result.Files = files
			result.Sections = append(result.Sections, "files")
			successes++
		}
	}
	if opts.IncludeReviews {
		if reviews, err := fetchReviewContextList(ctx, fmt.Sprintf("%s/pulls/%d/reviews", workflowRepoPath(owner, repo), opts.Number), nil, 100); err != nil {
			result.Notes = append(result.Notes, ScoringNote{Metric: "pr_reviews", Note: fmt.Sprintf("pr +reviews equivalent failed: %v", err)})
		} else {
			result.Reviews = reviews
			result.Sections = append(result.Sections, "reviews")
			successes++
		}
	}
	if opts.IncludeThreads {
		query := url.Values{}
		query.Set("is_full", "true")
		path := fmt.Sprintf("%s/pulls/%d/journals", workflowRepoPath(owner, repo), opts.Number)
		if threads, err := fetchReviewContextList(ctx, path, query, opts.ThreadLimit); err != nil {
			result.Notes = append(result.Notes, ScoringNote{Metric: "pr_review_threads", Note: fmt.Sprintf("pr +review-comments equivalent failed: %v", err)})
		} else {
			result.Threads = normalizeReviewContextThreads(threads, result.CurrentHeadSHA)
			result.Sections = append(result.Sections, "threads")
			successes++
		}
	}
	if opts.IncludeIssues {
		query := url.Values{}
		query.Set("category", "opened")
		if issues, err := fetchReviewContextList(ctx, workflowRepoPath(owner, repo)+"/issues", query, opts.IssueLimit); err != nil {
			result.Notes = append(result.Notes, ScoringNote{Metric: "open_issues", Note: fmt.Sprintf("issue +list equivalent failed: %v", err)})
		} else {
			result.OpenIssues = issues
			result.Sections = append(result.Sections, "open_issues")
			successes++
		}
	}
	if opts.IncludeLabels {
		if labels, err := fetchReviewContextList(ctx, workflowRepoPath(owner, repo)+"/issue_tags", nil, opts.LabelLimit); err != nil {
			result.Notes = append(result.Notes, ScoringNote{Metric: "labels", Note: fmt.Sprintf("label +list equivalent failed: %v", err)})
		} else {
			result.Labels = labels
			result.Sections = append(result.Sections, "labels")
			successes++
		}
	}

	result.Notes = uniqueScoringNotes(result.Notes)
	if successes == 0 {
		return ReviewContext{}, fmt.Errorf("fetch review context: all enabled sections failed")
	}
	finalizeReviewContext(&result, time.Now().UTC())
	return result, nil
}

func fetchReviewContextPR(ctx *common.RuntimeContext, owner, repo string, number int) (map[string]interface{}, error) {
	env, err := ctx.CallAPI("GET", fmt.Sprintf("%s/pulls/%d", plainRepoPath(owner, repo), number), nil)
	if err != nil {
		return nil, err
	}
	item := prAPIObject(env.Data)
	if item == nil {
		return nil, fmt.Errorf("PR response did not contain an object")
	}
	return item, nil
}

func fetchReviewContextList(ctx *common.RuntimeContext, path string, query url.Values, limit int) ([]map[string]interface{}, error) {
	if limit <= 0 {
		limit = 100
	}
	q := cloneValues(query)
	q.Set("page", "1")
	q.Set("limit", fmt.Sprintf("%d", limit))

	env, err := ctx.CallAPIWithQuery("GET", path, q)
	if err != nil {
		return nil, err
	}
	items := apiList(env.Data)
	out := make([]map[string]interface{}, 0, len(items))
	for _, raw := range items {
		item, ok := raw.(map[string]interface{})
		if !ok {
			continue
		}
		out = append(out, item)
		if len(out) >= limit {
			break
		}
	}
	return out, nil
}

func RenderReviewContext(context ReviewContext, format string) (string, error) {
	var rendered strings.Builder
	switch normalizeFormat(format) {
	case "json":
		if err := writeJSON(&rendered, context); err != nil {
			return "", err
		}
	case "markdown":
		if err := writeReviewContextMarkdown(&rendered, context); err != nil {
			return "", err
		}
	case "table":
		if err := writeReviewContextTable(&rendered, context); err != nil {
			return "", err
		}
	default:
		return "", fmt.Errorf("unsupported workflow output format %q", format)
	}
	return rendered.String(), nil
}

func writeReviewContextMarkdown(w *strings.Builder, context ReviewContext) error {
	_, _ = fmt.Fprintf(w, "# PR Review Context\n\n")
	_, _ = fmt.Fprintf(w, "- Repository: `%s`\n", context.Repository)
	_, _ = fmt.Fprintf(w, "- Pull request: `#%d`\n", context.PullRequest)
	_, _ = fmt.Fprintf(w, "- Source: `%s`\n", context.Source)
	_, _ = fmt.Fprintf(w, "- Head SHA: `%s`\n", fallbackReviewContextValue(context.CurrentHeadSHA))
	_, _ = fmt.Fprintf(w, "- Source fingerprint: `%s`\n", context.WorkItem.SourceFingerprint)
	_, _ = fmt.Fprintf(w, "- Sections: `%s`\n", strings.Join(context.Sections, ", "))
	_, _ = fmt.Fprintf(w, "\n## Summary\n\n")
	_, _ = fmt.Fprintf(w, "- Changed files: `%d`\n", len(context.Files))
	_, _ = fmt.Fprintf(w, "- Reviews: `%d`\n", len(context.Reviews))
	_, _ = fmt.Fprintf(w, "- Current reviews: `%d`\n", context.Summary.CurrentReviews)
	_, _ = fmt.Fprintf(w, "- Outdated reviews: `%d`\n", context.Summary.OutdatedReviews)
	_, _ = fmt.Fprintf(w, "- Review freshness: `%s`\n", context.Summary.ReviewFreshness)
	_, _ = fmt.Fprintf(w, "- Review decision: `%s`\n", context.Summary.Decision)
	_, _ = fmt.Fprintf(w, "- Review threads: `%d`\n", len(context.Threads))
	_, _ = fmt.Fprintf(w, "- Pending responses: `%d`\n", context.Summary.PendingResponse)
	_, _ = fmt.Fprintf(w, "- Open issues included: `%d`\n", len(context.OpenIssues))
	_, _ = fmt.Fprintf(w, "- Labels included: `%d`\n", len(context.Labels))
	if len(context.WorkItem.Unknowns) > 0 {
		_, _ = fmt.Fprintf(w, "\n## Unknowns\n\n")
		for _, unknown := range context.WorkItem.Unknowns {
			_, _ = fmt.Fprintf(w, "- `%s`\n", unknown)
		}
	}
	if len(context.Notes) > 0 {
		_, _ = fmt.Fprintf(w, "\n## Notes\n\n")
		for _, note := range context.Notes {
			_, _ = fmt.Fprintf(w, "- `%s`: %s\n", note.Metric, note.Note)
		}
	}
	return nil
}

func writeReviewContextTable(w *strings.Builder, context ReviewContext) error {
	_, _ = fmt.Fprintf(w, "REPOSITORY\tPR\tHEAD\tDECISION\tFRESHNESS\tFILES\tREVIEWS\tTHREADS\tPENDING\tNOTES\n")
	_, _ = fmt.Fprintf(w, "%s\t#%d\t%s\t%s\t%s\t%d\t%d\t%d\t%d\t%d\n",
		context.Repository,
		context.PullRequest,
		fallbackReviewContextValue(context.CurrentHeadSHA),
		context.Summary.Decision,
		context.Summary.ReviewFreshness,
		len(context.Files),
		len(context.Reviews),
		len(context.Threads),
		context.Summary.PendingResponse,
		len(context.Notes),
	)
	return nil
}

func fallbackReviewContextValue(value string) string {
	if strings.TrimSpace(value) == "" {
		return "unknown"
	}
	return value
}

func plainRepoPath(owner, repo string) string {
	return fmt.Sprintf("/%s/%s", strings.TrimSpace(owner), strings.TrimSpace(repo))
}

func parseBoolDefault(value string, defaultValue bool) bool {
	value = strings.TrimSpace(value)
	if value == "" {
		return defaultValue
	}
	return parseBoolArg(value)
}
