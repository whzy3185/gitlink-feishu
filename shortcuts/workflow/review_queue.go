package workflow

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"sort"
	"strings"
	"text/tabwriter"

	"github.com/gitlink-org/gitlink-cli/cmd/cmdutil"
	"github.com/gitlink-org/gitlink-cli/shortcuts/common"
)

type ReviewQueueInput struct {
	Repository   string           `json:"repository"`
	PullRequests []PRSummaryInput `json:"pull_requests"`
	Source       string           `json:"source"`
}

type ReviewQueueResult struct {
	Repository      string            `json:"repository"`
	TotalPRs        int               `json:"total_prs"`
	HighPriority    int               `json:"high_priority"`
	MediumPriority  int               `json:"medium_priority"`
	LowPriority     int               `json:"low_priority"`
	Items           []ReviewQueueItem `json:"items"`
	TopFocus        []string          `json:"top_focus"`
	Recommendations []string          `json:"recommendations"`
	Source          string            `json:"source"`
}

type ReviewQueueItem struct {
	Rank            int      `json:"rank"`
	Number          int      `json:"number,omitempty"`
	Title           string   `json:"title"`
	Author          string   `json:"author,omitempty"`
	State           string   `json:"state,omitempty"`
	ChangeType      string   `json:"change_type"`
	RiskLevel       string   `json:"risk_level"`
	Priority        string   `json:"priority"`
	PriorityScore   int      `json:"priority_score"`
	ChangedFiles    int      `json:"changed_files"`
	Commits         int      `json:"commits"`
	Additions       int      `json:"additions"`
	Deletions       int      `json:"deletions"`
	Reasons         []string `json:"reasons"`
	SuggestedAction string   `json:"suggested_action"`
	ReviewFocus     []string `json:"review_focus,omitempty"`
}

type ReviewQueueFetchOptions struct {
	Owner     string
	Repo      string
	State     string
	StartPage int
	PageSize  int
	MaxItems  int
	Language  string
}

// FetchReviewQueue builds a prioritized review queue using GET-only GitLink
// requests. It exists as a stable entry point for collaboration gateways so
// they do not need to emulate CLI flags or mutate RuntimeContext repository
// state.
func FetchReviewQueue(ctx *common.RuntimeContext, opts ReviewQueueFetchOptions) (ReviewQueueResult, error) {
	if ctx == nil {
		return ReviewQueueResult{}, fmt.Errorf("review queue runtime context is required")
	}
	owner, repo, err := resolveFetchRepo(ctx, opts.Owner, opts.Repo)
	if err != nil {
		return ReviewQueueResult{}, fmt.Errorf("resolve review queue repository: %w", err)
	}
	state := strings.TrimSpace(opts.State)
	if state == "" {
		state = "open"
	}
	startPage := opts.StartPage
	if startPage <= 0 {
		startPage = 1
	}
	pageSize := opts.PageSize
	if pageSize <= 0 {
		pageSize = 30
	}
	if pageSize > 100 {
		pageSize = 100
	}
	maxItems := opts.MaxItems
	if maxItems <= 0 {
		maxItems = 100
	}

	prs := make([]PRSummaryInput, 0, minReviewQueueInt(pageSize, maxItems))
	seen := map[string]bool{}
	for page := startPage; len(prs) < maxItems; page++ {
		items, fetchErr := fetchReviewQueuePage(ctx, owner, repo, state, page, pageSize)
		if fetchErr != nil {
			return ReviewQueueResult{}, fmt.Errorf("fetch pull requests for review queue page %d: %w", page, fetchErr)
		}
		if len(items) == 0 {
			break
		}
		for _, item := range items {
			key := reviewQueuePRKey(item)
			if seen[key] {
				continue
			}
			seen[key] = true
			prs = append(prs, item)
			if len(prs) >= maxItems {
				break
			}
		}
		if len(items) < pageSize {
			break
		}
	}
	input := ReviewQueueInput{
		Repository:   fmt.Sprintf("%s/%s", owner, repo),
		PullRequests: prs,
		Source:       "remote-read-only-fetch",
	}
	return AnalyzeReviewQueue(input, normalizeLang(opts.Language)), nil
}

func newReviewQueueShortcut() *common.Shortcut {
	return &common.Shortcut{
		Name:        "review-queue",
		Description: "Prioritize open pull requests for maintainer review",
		Flags: []common.Flag{
			{Name: "from", Usage: "Read review queue input from a JSON file"},
			{Name: "state", Usage: "Remote pull request state to fetch", Default: "open"},
			{Name: "page", Short: "p", Usage: "Remote pull request page", Default: "1"},
			{Name: "limit", Short: "l", Usage: "Maximum pull requests to include", Default: "30"},
			{Name: "all", Usage: "Fetch consecutive pull request pages with read-only requests", Bool: true, Default: "false"},
			{Name: "max-items", Usage: "Safety cap when --all is enabled", Default: "1000"},
			{Name: "lang", Usage: "Output language: en or zh-CN", Default: langEN},
		},
		Run: runReviewQueue,
	}
}

func runReviewQueue(ctx *common.RuntimeContext) error {
	lang := normalizeLang(ctx.Arg("lang"))
	input, err := collectReviewQueueInput(ctx)
	if err != nil {
		return err
	}
	result := AnalyzeReviewQueue(input, lang)
	format := ctx.Format
	if strings.TrimSpace(cmdutil.Format) == "" {
		format = "table"
	}
	rendered, err := RenderReviewQueue(result, format, lang)
	if err != nil {
		return err
	}
	_, err = fmt.Fprint(os.Stdout, rendered)
	return err
}

func collectReviewQueueInput(ctx *common.RuntimeContext) (ReviewQueueInput, error) {
	if path := strings.TrimSpace(ctx.Arg("from")); path != "" {
		input, err := readReviewQueueInput(path)
		if err != nil {
			return ReviewQueueInput{}, err
		}
		if strings.TrimSpace(input.Source) == "" {
			input.Source = "local-json"
		}
		return input, nil
	}
	limit, err := parseIntArg(ctx.Arg("limit"), 30, "limit")
	if err != nil {
		return ReviewQueueInput{}, err
	}
	page, err := parseIntArg(ctx.Arg("page"), 1, "page")
	if err != nil {
		return ReviewQueueInput{}, err
	}
	state := strings.TrimSpace(ctx.Arg("state"))
	if state == "" {
		state = "open"
	}
	var prs []PRSummaryInput
	var owner, repo string
	if parseBoolDefault(ctx.Arg("all"), false) {
		maxItems, parseErr := parseIntArg(ctx.Arg("max-items"), 1000, "max-items")
		if parseErr != nil {
			return ReviewQueueInput{}, parseErr
		}
		prs, owner, repo, err = fetchAllReviewQueuePullRequests(ctx, state, page, limit, maxItems)
	} else {
		prs, owner, repo, err = fetchReviewQueuePullRequests(ctx, state, page, limit)
	}
	if err != nil {
		return ReviewQueueInput{}, err
	}
	return ReviewQueueInput{
		Repository:   fmt.Sprintf("%s/%s", owner, repo),
		PullRequests: prs,
		Source:       "remote-read-only-fetch",
	}, nil
}

func readReviewQueueInput(path string) (ReviewQueueInput, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return ReviewQueueInput{}, fmt.Errorf("read review queue input: %w", err)
	}
	var input ReviewQueueInput
	if err := json.Unmarshal(data, &input); err == nil && (len(input.PullRequests) > 0 || strings.TrimSpace(input.Repository) != "") {
		return input, nil
	}
	var prs []PRSummaryInput
	if err := json.Unmarshal(data, &prs); err != nil {
		return ReviewQueueInput{}, fmt.Errorf("parse review queue input: expected ReviewQueueInput or []PRSummaryInput: %w", err)
	}
	return ReviewQueueInput{PullRequests: prs, Source: "local-json"}, nil
}

func fetchReviewQueuePullRequests(ctx *common.RuntimeContext, state string, page, limit int) ([]PRSummaryInput, string, string, error) {
	owner, repo, err := resolveFetchRepo(ctx, "", "")
	if err != nil {
		return nil, "", "", fmt.Errorf("workflow +review-queue remote mode requires --owner and --repo or a Git remote: %w", err)
	}
	if limit <= 0 {
		limit = 30
	}
	if page <= 0 {
		page = 1
	}
	prs, err := fetchReviewQueuePage(ctx, owner, repo, state, page, limit)
	if err != nil {
		return nil, "", "", err
	}
	return prs, owner, repo, nil
}

func fetchAllReviewQueuePullRequests(ctx *common.RuntimeContext, state string, startPage, pageSize, maxItems int) ([]PRSummaryInput, string, string, error) {
	owner, repo, err := resolveFetchRepo(ctx, "", "")
	if err != nil {
		return nil, "", "", fmt.Errorf("workflow +review-queue remote mode requires --owner and --repo or a Git remote: %w", err)
	}
	if startPage <= 0 {
		startPage = 1
	}
	if pageSize <= 0 {
		pageSize = 30
	}
	if pageSize > 100 {
		pageSize = 100
	}
	if maxItems <= 0 {
		maxItems = 1000
	}
	out := make([]PRSummaryInput, 0, minReviewQueueInt(pageSize, maxItems))
	seen := map[string]bool{}
	for page := startPage; len(out) < maxItems; page++ {
		items, err := fetchReviewQueuePage(ctx, owner, repo, state, page, pageSize)
		if err != nil {
			return nil, "", "", fmt.Errorf("fetch pull requests for review queue page %d: %w", page, err)
		}
		if len(items) == 0 {
			break
		}
		for _, item := range items {
			key := reviewQueuePRKey(item)
			if seen[key] {
				continue
			}
			seen[key] = true
			out = append(out, item)
			if len(out) >= maxItems {
				break
			}
		}
		if len(items) < pageSize {
			break
		}
	}
	return out, owner, repo, nil
}

func fetchReviewQueuePage(ctx *common.RuntimeContext, owner, repo, state string, page, limit int) ([]PRSummaryInput, error) {
	query := url.Values{}
	query.Set("state", state)
	query.Set("page", fmt.Sprintf("%d", page))
	query.Set("limit", fmt.Sprintf("%d", limit))
	env, err := ctx.CallAPIWithQuery("GET", workflowRepoPath(owner, repo)+"/pulls", query)
	if err != nil {
		return nil, fmt.Errorf("fetch pull requests for review queue: %w", err)
	}
	items := apiList(env.Data)
	prs := make([]PRSummaryInput, 0, len(items))
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
			input.State = state
		}
		prs = append(prs, input)
		if len(prs) >= limit {
			break
		}
	}
	return prs, nil
}

func reviewQueuePRKey(item PRSummaryInput) string {
	if item.Number > 0 {
		return fmt.Sprintf("number:%d", item.Number)
	}
	return "fallback:" + strings.ToLower(strings.TrimSpace(item.Title)) + "\x00" + strings.ToLower(strings.TrimSpace(item.Author))
}

func minReviewQueueInt(left, right int) int {
	if left < right {
		return left
	}
	return right
}

func AnalyzeReviewQueue(input ReviewQueueInput, lang string) ReviewQueueResult {
	lang = normalizeLang(lang)
	source := strings.TrimSpace(input.Source)
	if source == "" {
		source = "local"
	}
	repository := strings.TrimSpace(input.Repository)
	if repository == "" && len(input.PullRequests) > 0 {
		repository = input.PullRequests[0].Repository
	}
	if repository == "" {
		repository = "local"
	}

	items := make([]ReviewQueueItem, 0, len(input.PullRequests))
	focus := []string{}
	for _, pr := range input.PullRequests {
		summary := AnalyzePRSummary(pr, lang)
		item := buildReviewQueueItem(pr, summary, lang)
		items = append(items, item)
		focus = append(focus, item.ReviewFocus...)
	}
	sort.SliceStable(items, func(i, j int) bool {
		if items[i].PriorityScore != items[j].PriorityScore {
			return items[i].PriorityScore > items[j].PriorityScore
		}
		if items[i].RiskLevel != items[j].RiskLevel {
			return reviewQueueRiskWeight(items[i].RiskLevel) > reviewQueueRiskWeight(items[j].RiskLevel)
		}
		return items[i].Number < items[j].Number
	})

	result := ReviewQueueResult{
		Repository: repository,
		TotalPRs:   len(items),
		Items:      items,
		TopFocus:   limitStringsForReviewQueue(uniqueStrings(focus), 10),
		Source:     source,
	}
	for i := range result.Items {
		result.Items[i].Rank = i + 1
		switch result.Items[i].Priority {
		case "high":
			result.HighPriority++
		case "medium":
			result.MediumPriority++
		default:
			result.LowPriority++
		}
	}
	result.Recommendations = buildReviewQueueRecommendations(result, lang)
	return result
}

func buildReviewQueueItem(pr PRSummaryInput, summary PRSummaryResult, lang string) ReviewQueueItem {
	score, reasons := scoreReviewQueueItem(pr, summary)
	priority := "low"
	if score >= 70 {
		priority = "high"
	} else if score >= 40 {
		priority = "medium"
	}
	return ReviewQueueItem{
		Number:          summary.Number,
		Title:           summary.Title,
		Author:          summary.Author,
		State:           summary.State,
		ChangeType:      summary.ChangeType,
		RiskLevel:       summary.RiskLevel,
		Priority:        priority,
		PriorityScore:   score,
		ChangedFiles:    summary.ChangedFilesCount,
		Commits:         summary.CommitCount,
		Additions:       summary.Additions,
		Deletions:       summary.Deletions,
		Reasons:         reasons,
		SuggestedAction: reviewQueueSuggestedAction(priority, summary.RiskLevel, summary.ChangeType, lang),
		ReviewFocus:     summary.ReviewFocus,
	}
}

func scoreReviewQueueItem(pr PRSummaryInput, summary PRSummaryResult) (int, []string) {
	score := 0
	reasons := []string{}
	switch summary.RiskLevel {
	case PRRiskCritical:
		score += 70
		reasons = append(reasons, "critical risk")
	case PRRiskHigh:
		score += 55
		reasons = append(reasons, "high risk")
	case PRRiskMedium:
		score += 30
		reasons = append(reasons, "medium risk")
	case PRRiskLow:
		score += 10
		reasons = append(reasons, "low risk")
	}
	switch summary.ChangeType {
	case PRChangeTypeFeature:
		score += 18
		reasons = append(reasons, "feature work")
	case PRChangeTypeFix:
		score += 16
		reasons = append(reasons, "bug fix")
	case PRChangeTypeRefactor:
		score += 14
		reasons = append(reasons, "refactor")
	case PRChangeTypeCI:
		score += 12
		reasons = append(reasons, "ci/build change")
	case PRChangeTypeDocs:
		score += 4
		reasons = append(reasons, "docs-only candidate")
	}
	size := summary.Additions + summary.Deletions
	if size >= 800 || summary.ChangedFilesCount >= 25 {
		score += 22
		reasons = append(reasons, "large diff")
	} else if size >= 200 || summary.ChangedFilesCount >= 8 {
		score += 12
		reasons = append(reasons, "medium diff")
	}
	if summary.CommitCount >= 10 {
		score += 8
		reasons = append(reasons, "many commits")
	}
	if summary.ChangeType != PRChangeTypeDocs && !hasReviewQueueTestSignal(pr) {
		score += 8
		reasons = append(reasons, "test signal not obvious")
	}
	if score > 100 {
		score = 100
	}
	return score, uniqueStrings(reasons)
}

func hasReviewQueueTestSignal(pr PRSummaryInput) bool {
	for _, file := range pr.ChangedFiles {
		if isTestPath(normalizedPath(file.Filename)) {
			return true
		}
	}
	text := strings.ToLower(prTextCorpus(pr, true))
	for _, keyword := range []string{"test", "tests", "coverage", "verified", "go test", "passed"} {
		if strings.Contains(text, keyword) {
			return true
		}
	}
	return false
}

func reviewQueueRiskWeight(risk string) int {
	switch risk {
	case PRRiskCritical:
		return 4
	case PRRiskHigh:
		return 3
	case PRRiskMedium:
		return 2
	case PRRiskLow:
		return 1
	default:
		return 0
	}
}

func reviewQueueSuggestedAction(priority, risk, changeType, lang string) string {
	if normalizeLang(lang) == langZH {
		switch {
		case priority == "high":
			return "优先安排维护者审查，合并前确认测试和风险点"
		case risk == PRRiskMedium || changeType == PRChangeTypeFeature:
			return "安排常规审查，重点确认行为变更和测试覆盖"
		default:
			return "可作为低风险队列处理，快速确认后推进"
		}
	}
	switch {
	case priority == "high":
		return "Prioritize maintainer review and verify tests plus risk areas before merge."
	case risk == PRRiskMedium || changeType == PRChangeTypeFeature:
		return "Schedule normal review with focus on behavior changes and test coverage."
	default:
		return "Treat as a low-risk queue item and move forward after a quick check."
	}
}

func buildReviewQueueRecommendations(result ReviewQueueResult, lang string) []string {
	if normalizeLang(lang) == langZH {
		recs := []string{}
		if result.HighPriority > 0 {
			recs = append(recs, fmt.Sprintf("先处理 %d 个高优先级 PR，避免高风险变更长时间积压。", result.HighPriority))
		}
		if len(result.TopFocus) > 0 {
			recs = append(recs, "审查时优先关注队列中反复出现的风险点。")
		}
		if len(recs) == 0 {
			recs = append(recs, "当前队列整体风险较低，可按提交顺序推进。")
		}
		return recs
	}
	recs := []string{}
	if result.HighPriority > 0 {
		recs = append(recs, fmt.Sprintf("Review %d high-priority PR(s) first to avoid high-risk backlog.", result.HighPriority))
	}
	if len(result.TopFocus) > 0 {
		recs = append(recs, "Use the repeated review focus items as the first pass checklist.")
	}
	if len(recs) == 0 {
		recs = append(recs, "The queue is mostly low risk; process it in normal submission order.")
	}
	return recs
}

func limitStringsForReviewQueue(values []string, limit int) []string {
	if limit <= 0 || len(values) <= limit {
		return values
	}
	return values[:limit]
}

func RenderReviewQueue(result ReviewQueueResult, format string, lang string) (string, error) {
	var buf bytes.Buffer
	switch normalizeFormat(format) {
	case "json":
		if err := writeJSON(&buf, result); err != nil {
			return "", err
		}
	case "markdown":
		if err := writeReviewQueueMarkdown(&buf, result, lang); err != nil {
			return "", err
		}
	case "table":
		if err := writeReviewQueueTable(&buf, result); err != nil {
			return "", err
		}
	default:
		return "", fmt.Errorf("unsupported workflow output format %q", format)
	}
	return buf.String(), nil
}

func writeReviewQueueMarkdown(buf *bytes.Buffer, result ReviewQueueResult, lang string) error {
	title := "Pull Request Review Queue"
	if normalizeLang(lang) == langZH {
		title = "PR 审查队列"
	}
	if _, err := fmt.Fprintf(buf, "# %s\n\n", title); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(buf, "- Repository: `%s`\n- Pull requests: `%d`\n- High priority: `%d`\n- Medium priority: `%d`\n- Low priority: `%d`\n- Source: `%s`\n\n",
		result.Repository,
		result.TotalPRs,
		result.HighPriority,
		result.MediumPriority,
		result.LowPriority,
		result.Source,
	); err != nil {
		return err
	}
	if len(result.Recommendations) > 0 {
		if _, err := fmt.Fprintln(buf, "## Recommendations"); err != nil {
			return err
		}
		if _, err := fmt.Fprintln(buf); err != nil {
			return err
		}
		for _, rec := range result.Recommendations {
			if _, err := fmt.Fprintf(buf, "- %s\n", rec); err != nil {
				return err
			}
		}
		if _, err := fmt.Fprintln(buf); err != nil {
			return err
		}
	}
	if _, err := fmt.Fprintln(buf, "## Queue"); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(buf); err != nil {
		return err
	}
	for _, item := range result.Items {
		number := ""
		if item.Number > 0 {
			number = fmt.Sprintf("#%d ", item.Number)
		}
		if _, err := fmt.Fprintf(buf, "%d. %s%s `%s/%s score:%d`\n", item.Rank, number, item.Title, item.Priority, item.RiskLevel, item.PriorityScore); err != nil {
			return err
		}
		if len(item.Reasons) > 0 {
			if _, err := fmt.Fprintf(buf, "   - Reasons: %s\n", strings.Join(item.Reasons, ", ")); err != nil {
				return err
			}
		}
		if _, err := fmt.Fprintf(buf, "   - Action: %s\n", item.SuggestedAction); err != nil {
			return err
		}
	}
	return nil
}

func writeReviewQueueTable(buf *bytes.Buffer, result ReviewQueueResult) error {
	tw := tabwriter.NewWriter(buf, 0, 0, 2, ' ', 0)
	if _, err := fmt.Fprintln(tw, "RANK\tPR\tPRIORITY\tSCORE\tRISK\tTYPE\tFILES\tCOMMITS\tTITLE"); err != nil {
		return err
	}
	for _, item := range result.Items {
		number := "-"
		if item.Number > 0 {
			number = fmt.Sprintf("#%d", item.Number)
		}
		if _, err := fmt.Fprintf(tw, "%d\t%s\t%s\t%d\t%s\t%s\t%d\t%d\t%s\n",
			item.Rank,
			number,
			item.Priority,
			item.PriorityScore,
			item.RiskLevel,
			item.ChangeType,
			item.ChangedFiles,
			item.Commits,
			truncateTableText(item.Title, 80),
		); err != nil {
			return err
		}
	}
	return tw.Flush()
}
