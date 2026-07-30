package workflow

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/gitlink-org/gitlink-cli/cmd/cmdutil"
	"github.com/gitlink-org/gitlink-cli/internal/client"
	"github.com/gitlink-org/gitlink-cli/shortcuts/common"
)

type ReviewContextOptions struct {
	Owner           string
	Repo            string
	Number          int
	IssueLimit      int
	LabelLimit      int
	VersionLimit    int
	ThreadLimit     int
	IncludeRepo     bool
	IncludePR       bool
	IncludeFiles    bool
	IncludeVersions bool
	IncludeReviews  bool
	IncludeThreads  bool
	IncludeIssues   bool
	IncludeLabels   bool
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
			{Name: "version-limit", Usage: "Maximum pull request patchset versions to include", Default: "100"},
			{Name: "thread-limit", Usage: "Maximum review thread records to include", Default: "100"},
			{Name: "include-repo", Usage: "Include repository info", Bool: true, Default: "true"},
			{Name: "include-pr", Usage: "Include pull request details", Bool: true, Default: "true"},
			{Name: "include-files", Usage: "Include pull request changed files", Bool: true, Default: "true"},
			{Name: "include-versions", Usage: "Include pull request patchset versions", Bool: true, Default: "true"},
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
	versionLimit, err := parseIntArg(ctx.Arg("version-limit"), 100, "version-limit")
	if err != nil {
		return err
	}
	threadLimit, err := parseIntArg(ctx.Arg("thread-limit"), 100, "thread-limit")
	if err != nil {
		return err
	}

	context, err = FetchReviewContext(ctx, ReviewContextOptions{
		Number:          number,
		IssueLimit:      issueLimit,
		LabelLimit:      labelLimit,
		VersionLimit:    versionLimit,
		ThreadLimit:     threadLimit,
		IncludeRepo:     parseBoolDefault(ctx.Arg("include-repo"), true),
		IncludePR:       parseBoolDefault(ctx.Arg("include-pr"), true),
		IncludeFiles:    parseBoolDefault(ctx.Arg("include-files"), true),
		IncludeVersions: parseBoolDefault(ctx.Arg("include-versions"), true),
		IncludeReviews:  parseBoolDefault(ctx.Arg("include-reviews"), true),
		IncludeThreads:  parseBoolDefault(ctx.Arg("include-threads"), true),
		IncludeIssues:   parseBoolDefault(ctx.Arg("include-issues"), true),
		IncludeLabels:   parseBoolDefault(ctx.Arg("include-labels"), true),
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
	if opts.VersionLimit <= 0 {
		opts.VersionLimit = 100
	}
	if opts.ThreadLimit <= 0 {
		opts.ThreadLimit = 100
	}

	result := ReviewContext{
		SchemaVersion:     reviewContextSchemaVersion,
		Repository:        fmt.Sprintf("%s/%s", owner, repo),
		PullRequest:       opts.Number,
		Source:            "shortcut-backed-read-only-fetch",
		Sections:          []string{},
		ReviewerSummaries: []ReviewContextReviewerSummary{},
		SectionStatuses:   initialReviewContextSectionStatuses(opts),
		FetchErrors:       []ReviewContextFetchError{},
		Notes:             []ScoringNote{},
		fileLimit:         100,
		versionLimit:      opts.VersionLimit,
		reviewLimit:       100,
		threadLimit:       opts.ThreadLimit,
	}
	successes := 0

	if opts.IncludeRepo {
		path := workflowRepoPath(owner, repo)
		if info, err := fetchRepoInfo(ctx, owner, repo); err != nil {
			recordReviewContextFetchError(&result, "repo_info", "GET", path, err, "repo +info equivalent failed")
		} else {
			result.RepositoryInfo = info
			result.Sections = append(result.Sections, "repo_info")
			recordReviewContextSectionSuccess(&result, "repo_info", 1, 0)
			successes++
		}
	}
	if opts.IncludePR {
		path := fmt.Sprintf("%s/pulls/%d", plainRepoPath(owner, repo), opts.Number)
		if pr, err := fetchReviewContextPR(ctx, owner, repo, opts.Number); err != nil {
			recordReviewContextFetchError(&result, "pr", "GET", path, err, "pr +view equivalent failed")
			result.Notes = uniqueScoringNotes(result.Notes)
			finalizeReviewContext(&result, time.Now().UTC())
			return result, fmt.Errorf("fetch required pull request: %s", safeReviewContextErrorMessage(err))
		} else {
			result.PR = pr
			result.Sections = append(result.Sections, "pr")
			recordReviewContextSectionSuccess(&result, "pr", 1, 0)
			successes++
		}
	}
	if opts.IncludeFiles {
		path := fmt.Sprintf("%s/pulls/%d/files", plainRepoPath(owner, repo), opts.Number)
		if files, err := fetchReviewContextList(ctx, path, nil, 100); err != nil {
			recordReviewContextFetchError(&result, "files", "GET", path, err, "pr +files equivalent failed")
		} else {
			result.Files = files
			result.Sections = append(result.Sections, "files")
			recordReviewContextSectionSuccess(&result, "files", len(files), 100)
			successes++
		}
	}
	if opts.IncludeVersions {
		path := fmt.Sprintf("%s/pulls/%d/versions", workflowRepoPath(owner, repo), opts.Number)
		if versions, err := fetchReviewContextList(ctx, path, nil, opts.VersionLimit); err != nil {
			recordReviewContextFetchError(&result, "versions", "GET", path, err, "pr +versions equivalent failed")
		} else {
			result.Versions = versions
			result.CurrentPatchset = normalizeCurrentReviewContextPatchset(versions)
			result.CurrentHeadSHA = result.CurrentPatchset.HeadSHA
			result.CurrentVersionID = result.CurrentPatchset.ID
			result.Sections = append(result.Sections, "versions")
			recordReviewContextSectionSuccess(&result, "versions", len(versions), opts.VersionLimit)
			successes++
		}
	}
	if opts.IncludeReviews {
		path := fmt.Sprintf("%s/pulls/%d/reviews", workflowRepoPath(owner, repo), opts.Number)
		if reviews, err := fetchReviewContextList(ctx, path, nil, 100); err != nil {
			recordReviewContextFetchError(&result, "reviews", "GET", path, err, "pr +reviews equivalent failed")
		} else {
			result.Reviews = reviews
			result.Sections = append(result.Sections, "reviews")
			recordReviewContextSectionSuccess(&result, "reviews", len(reviews), 100)
			successes++
		}
	}
	if opts.IncludeThreads {
		query := url.Values{}
		query.Set("is_full", "true")
		path := fmt.Sprintf("%s/pulls/%d/journals", workflowRepoPath(owner, repo), opts.Number)
		if threads, err := fetchReviewContextList(ctx, path, query, opts.ThreadLimit); err != nil {
			recordReviewContextFetchError(&result, "threads", "GET", path, err, "pr +review-comments equivalent failed")
		} else {
			result.Threads = normalizeReviewContextThreads(threads, result.CurrentHeadSHA)
			result.Sections = append(result.Sections, "threads")
			recordReviewContextSectionSuccess(&result, "threads", len(threads), opts.ThreadLimit)
			successes++
		}
	}
	if opts.IncludeIssues {
		query := url.Values{}
		query.Set("category", "opened")
		path := workflowRepoPath(owner, repo) + "/issues"
		if issues, err := fetchReviewContextList(ctx, path, query, opts.IssueLimit); err != nil {
			recordReviewContextFetchError(&result, "open_issues", "GET", path, err, "issue +list equivalent failed")
		} else {
			result.OpenIssues = issues
			result.Sections = append(result.Sections, "open_issues")
			recordReviewContextSectionSuccess(&result, "open_issues", len(issues), opts.IssueLimit)
			successes++
		}
	}
	if opts.IncludeLabels {
		path := workflowRepoPath(owner, repo) + "/issue_tags"
		if labels, err := fetchReviewContextList(ctx, path, nil, opts.LabelLimit); err != nil {
			recordReviewContextFetchError(&result, "labels", "GET", path, err, "label +list equivalent failed")
		} else {
			result.Labels = labels
			result.Sections = append(result.Sections, "labels")
			recordReviewContextSectionSuccess(&result, "labels", len(labels), opts.LabelLimit)
			successes++
		}
	}

	result.Notes = uniqueScoringNotes(result.Notes)
	if successes == 0 {
		finalizeReviewContext(&result, time.Now().UTC())
		return result, fmt.Errorf("fetch review context: all enabled sections failed")
	}
	finalizeReviewContext(&result, time.Now().UTC())
	return result, nil
}

func initialReviewContextSectionStatuses(opts ReviewContextOptions) []ReviewContextSectionStatus {
	definitions := []struct {
		name     string
		enabled  bool
		required bool
		limit    int
	}{
		{name: "repo_info", enabled: opts.IncludeRepo},
		{name: "pr", enabled: opts.IncludePR, required: true},
		{name: "files", enabled: opts.IncludeFiles, required: true, limit: 100},
		{name: "versions", enabled: opts.IncludeVersions, required: true, limit: opts.VersionLimit},
		{name: "reviews", enabled: opts.IncludeReviews, required: true, limit: 100},
		{name: "threads", enabled: opts.IncludeThreads, required: true, limit: opts.ThreadLimit},
		{name: "open_issues", enabled: opts.IncludeIssues, limit: opts.IssueLimit},
		{name: "labels", enabled: opts.IncludeLabels, limit: opts.LabelLimit},
	}
	statuses := make([]ReviewContextSectionStatus, 0, len(definitions))
	for _, definition := range definitions {
		status := reviewSectionDisabled
		if definition.enabled {
			status = reviewSectionPending
		}
		statuses = append(statuses, ReviewContextSectionStatus{
			Section:  definition.name,
			Status:   status,
			Required: definition.required,
			Limit:    definition.limit,
		})
	}
	return statuses
}

func recordReviewContextSectionSuccess(context *ReviewContext, section string, itemCount, limit int) {
	status := reviewSectionLoaded
	if itemCount == 0 {
		status = reviewSectionEmpty
	} else if limit > 0 && itemCount >= limit {
		status = reviewSectionSampled
	}
	updateReviewContextSectionStatus(context, section, status, itemCount, limit)
}

func recordReviewContextFetchError(context *ReviewContext, section, method, path string, err error, notePrefix string) {
	safeMessage := safeReviewContextErrorMessage(err)
	fetchError := ReviewContextFetchError{
		Section:   section,
		Method:    method,
		Path:      path,
		Code:      "request_failed",
		Retryable: true,
		Message:   safeMessage,
	}
	var apiErr *client.APIError
	if errors.As(err, &apiErr) {
		fetchError.StatusCode = apiErr.StatusCode
		fetchError.Code = fmt.Sprint(apiErr.Code)
		fetchError.Retryable = apiErr.StatusCode == 408 ||
			apiErr.StatusCode == 429 ||
			apiErr.StatusCode >= 500
	}
	context.FetchErrors = append(context.FetchErrors, fetchError)
	context.Notes = append(context.Notes, ScoringNote{
		Metric: reviewContextLegacyMetric(section),
		Note:   fmt.Sprintf("%s: %s", notePrefix, safeMessage),
	})
	updateReviewContextSectionStatus(context, section, reviewSectionFailed, 0, 0)
}

func updateReviewContextSectionStatus(context *ReviewContext, section, status string, itemCount, limit int) {
	for i := range context.SectionStatuses {
		if context.SectionStatuses[i].Section != section {
			continue
		}
		context.SectionStatuses[i].Status = status
		context.SectionStatuses[i].ItemCount = itemCount
		if limit > 0 {
			context.SectionStatuses[i].Limit = limit
		}
		return
	}
	context.SectionStatuses = append(context.SectionStatuses, ReviewContextSectionStatus{
		Section:   section,
		Status:    status,
		Required:  isRequiredReviewContextSection(section),
		ItemCount: itemCount,
		Limit:     limit,
	})
}

func reviewContextLegacyMetric(section string) string {
	switch section {
	case "pr":
		return "pr_view"
	case "files":
		return "pr_files"
	case "versions":
		return "pr_versions"
	case "reviews":
		return "pr_reviews"
	case "threads":
		return "pr_review_threads"
	default:
		return section
	}
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
	_, _ = fmt.Fprintf(w, "- Patchset version: `%s`\n", fallbackReviewContextValue(context.CurrentVersionID))
	_, _ = fmt.Fprintf(w, "- GitLink state: `%s`\n", fallbackReviewContextValue(context.WorkItem.GitLinkState))
	_, _ = fmt.Fprintf(w, "- Review stage: `%s`\n", fallbackReviewContextValue(context.WorkItem.ReviewStage))
	_, _ = fmt.Fprintf(w, "- Source fingerprint: `%s`\n", context.WorkItem.SourceFingerprint)
	_, _ = fmt.Fprintf(w, "- Collection status: `%s`\n", fallbackReviewContextValue(context.CollectionStatus))
	_, _ = fmt.Fprintf(w, "- Partial: `%t`\n", context.Partial)
	_, _ = fmt.Fprintf(w, "- Sections: `%s`\n", strings.Join(context.Sections, ", "))
	_, _ = fmt.Fprintf(w, "\n## Summary\n\n")
	_, _ = fmt.Fprintf(w, "- Changed files: `%d`\n", len(context.Files))
	_, _ = fmt.Fprintf(w, "- Patchset versions: `%d`\n", len(context.Versions))
	_, _ = fmt.Fprintf(w, "- Reviews: `%d`\n", reviewContextReviewCount(context))
	_, _ = fmt.Fprintf(w, "- Reviewers: `%d`\n", len(context.ReviewerSummaries))
	_, _ = fmt.Fprintf(w, "- Current reviews: `%d`\n", context.Summary.CurrentReviews)
	_, _ = fmt.Fprintf(w, "- Outdated reviews: `%d`\n", context.Summary.OutdatedReviews)
	_, _ = fmt.Fprintf(w, "- Review freshness: `%s`\n", context.Summary.ReviewFreshness)
	_, _ = fmt.Fprintf(w, "- Review decision: `%s`\n", context.Summary.Decision)
	_, _ = fmt.Fprintf(w, "- Review threads: `%d`\n", len(context.Threads))
	_, _ = fmt.Fprintf(w, "- Pending responses: `%d`\n", context.Summary.PendingResponse)
	_, _ = fmt.Fprintf(w, "- Open issues included: `%d`\n", len(context.OpenIssues))
	_, _ = fmt.Fprintf(w, "- Labels included: `%d`\n", len(context.Labels))
	if len(context.ReviewerSummaries) > 0 {
		_, _ = fmt.Fprintf(w, "\n## Reviewer Decisions\n\n")
		for _, reviewer := range context.ReviewerSummaries {
			identity := reviewer.Actor
			if strings.TrimSpace(identity) == "" {
				identity = reviewer.ReviewerKey
			}
			latestID := "unknown"
			if reviewer.LatestEffectiveReview != nil {
				latestID = fallbackReviewContextValue(reviewer.LatestEffectiveReview.ID)
			}
			_, _ = fmt.Fprintf(
				w,
				"- `%s`: decision=`%s`, latest=`%s`, current=`%d`, outdated=`%d`, unknown=`%d`, order_known=`%t`\n",
				identity,
				reviewer.CurrentDecision,
				latestID,
				reviewer.CurrentCount,
				reviewer.OutdatedCount,
				reviewer.UnknownCount,
				reviewer.DecisionOrderKnown,
			)
		}
	}
	if len(context.Threads) > 0 {
		_, _ = fmt.Fprintf(w, "\n## Review Threads\n\n")
		for _, thread := range context.Threads {
			content := strings.TrimSpace(firstLine(thread.Content))
			if content == "" {
				content = "(no content returned)"
			}
			location := strings.TrimSpace(thread.Path)
			if location == "" {
				location = "general"
			}
			_, _ = fmt.Fprintf(
				w,
				"- `#%s` `%s/%s` `%s` by `%s`: %s\n",
				fallbackReviewContextValue(thread.ID),
				thread.State,
				thread.Freshness,
				location,
				fallbackReviewContextValue(thread.Author),
				content,
			)
		}
	}
	if len(context.SectionStatuses) > 0 {
		_, _ = fmt.Fprintf(w, "\n## Collection Sections\n\n")
		for _, section := range context.SectionStatuses {
			_, _ = fmt.Fprintf(
				w,
				"- `%s`: status=`%s`, required=`%t`, items=`%d`, limit=`%d`\n",
				section.Section,
				section.Status,
				section.Required,
				section.ItemCount,
				section.Limit,
			)
		}
	}
	if len(context.FetchErrors) > 0 {
		_, _ = fmt.Fprintf(w, "\n## Fetch Errors\n\n")
		for _, fetchError := range context.FetchErrors {
			_, _ = fmt.Fprintf(
				w,
				"- `%s` `%s %s`: code=`%s`, status=`%d`, retryable=`%t` — %s\n",
				fetchError.Section,
				fetchError.Method,
				fetchError.Path,
				fetchError.Code,
				fetchError.StatusCode,
				fetchError.Retryable,
				fetchError.Message,
			)
		}
	}
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
	_, _ = fmt.Fprintf(w, "REPOSITORY\tPR\tHEAD\tSTATE\tSTAGE\tCOLLECTION\tDECISION\tFRESHNESS\tFILES\tREVIEWS\tREVIEWERS\tTHREADS\tPENDING\tERRORS\n")
	_, _ = fmt.Fprintf(w, "%s\t#%d\t%s\t%s\t%s\t%s\t%s\t%s\t%d\t%d\t%d\t%d\t%d\t%d\n",
		context.Repository,
		context.PullRequest,
		fallbackReviewContextValue(context.CurrentHeadSHA),
		fallbackReviewContextValue(context.WorkItem.GitLinkState),
		fallbackReviewContextValue(context.WorkItem.ReviewStage),
		fallbackReviewContextValue(context.CollectionStatus),
		context.Summary.Decision,
		context.Summary.ReviewFreshness,
		len(context.Files),
		reviewContextReviewCount(context),
		len(context.ReviewerSummaries),
		len(context.Threads),
		context.Summary.PendingResponse,
		len(context.FetchErrors),
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
