package workflow

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/gitlink-org/gitlink-cli/cmd/cmdutil"
	"github.com/gitlink-org/gitlink-cli/shortcuts/common"
)

const (
	staleKindIssue = "issue"
	staleKindPR    = "pull_request"
)

const (
	staleBucketFresh  = "fresh"
	staleBucketWatch  = "watch"
	staleBucketStale  = "stale"
	staleBucketZombie = "zombie"
)

type StaleInput struct {
	Repository   string                  `json:"repository"`
	Source       string                  `json:"source"`
	Issues       []IssueInput            `json:"issues,omitempty"`
	PullRequests []StalePullRequestInput `json:"pull_requests,omitempty"`
}

type StalePullRequestInput struct {
	Number         int       `json:"number"`
	Title          string    `json:"title"`
	Author         string    `json:"author"`
	State          string    `json:"state"`
	URL            string    `json:"url"`
	BaseBranch     string    `json:"base_branch,omitempty"`
	HeadBranch     string    `json:"head_branch,omitempty"`
	CreatedAt      time.Time `json:"created_at,omitempty"`
	UpdatedAt      time.Time `json:"updated_at,omitempty"`
	LastActivityAt time.Time `json:"last_activity_at,omitempty"`
	CommentsCount  int       `json:"comments_count"`
	ActivitySource string    `json:"activity_source,omitempty"`
}

type StaleReport struct {
	Repository      string         `json:"repository"`
	Source          string         `json:"source"`
	State           string         `json:"state"`
	StaleDays       int            `json:"stale_days"`
	Top             int            `json:"top"`
	ScannedTotal    int            `json:"scanned_total"`
	IssuesScanned   int            `json:"issues_scanned"`
	PRsScanned      int            `json:"prs_scanned"`
	FlaggedTotal    int            `json:"flagged_total"`
	IssuesFlagged   int            `json:"issues_flagged"`
	PRsFlagged      int            `json:"prs_flagged"`
	ShownTotal      int            `json:"shown_total"`
	OmittedTotal    int            `json:"omitted_total"`
	OldestAgeDays   int            `json:"oldest_age_days"`
	ByBucket        map[string]int `json:"by_bucket"`
	Items           []StaleItem    `json:"items"`
	Recommendations []string       `json:"recommendations"`
	Notes           []string       `json:"notes,omitempty"`
}

type StaleItem struct {
	Kind             string    `json:"kind"`
	Number           int       `json:"number"`
	Title            string    `json:"title"`
	State            string    `json:"state"`
	Author           string    `json:"author,omitempty"`
	URL              string    `json:"url,omitempty"`
	Labels           []string  `json:"labels,omitempty"`
	BaseBranch       string    `json:"base_branch,omitempty"`
	HeadBranch       string    `json:"head_branch,omitempty"`
	CommentsCount    int       `json:"comments_count"`
	LastActivityAt   time.Time `json:"last_activity_at,omitempty"`
	ActivitySource   string    `json:"activity_source,omitempty"`
	AgeDays          int       `json:"age_days"`
	Bucket           string    `json:"bucket"`
	SuggestedAction  string    `json:"suggested_action"`
	SuggestedComment string    `json:"suggested_comment"`
	Reasoning        []string  `json:"reasoning,omitempty"`
}

type staleScanOptions struct {
	State         string
	StaleDays     int
	Top           int
	IncludeIssues bool
	IncludePRs    bool
}

type staleCandidate struct {
	Kind           string
	Number         int
	Title          string
	State          string
	Author         string
	URL            string
	Labels         []string
	BaseBranch     string
	HeadBranch     string
	CommentsCount  int
	LastActivityAt time.Time
	ActivitySource string
}

func newStaleShortcut() *common.Shortcut {
	return &common.Shortcut{
		Name:        "stale",
		Description: "Scan stale issues and pull requests with read-only workflow rules",
		Flags: []common.Flag{
			{Name: "from", Usage: "Read stale scan input from a JSON file"},
			{Name: "state", Short: "s", Usage: "Filter state for local or remote scan", Default: "open"},
			{Name: "stale-days", Usage: "Days before an item enters the watch bucket", Default: "30"},
			{Name: "issue-limit", Usage: "Maximum issues to fetch in remote mode", Default: "20"},
			{Name: "pr-limit", Usage: "Maximum pull requests to fetch in remote mode", Default: "20"},
			{Name: "top", Usage: "Maximum flagged items to show; 0 shows all", Default: "20"},
			{Name: "include-issues", Usage: "Include issues in the scan", Bool: true, Default: "true"},
			{Name: "include-prs", Usage: "Include pull requests in the scan", Bool: true, Default: "true"},
			{Name: "lang", Usage: "Output language: en or zh-CN", Default: langEN},
		},
		Run: runStale,
	}
}

func runStale(ctx *common.RuntimeContext) error {
	lang := normalizeLang(ctx.Arg("lang"))
	input, notes, opts, err := collectStaleInput(ctx)
	if err != nil {
		return err
	}

	report := AnalyzeStale(input, notes, opts, lang)
	format := ctx.Format
	if strings.TrimSpace(cmdutil.Format) == "" {
		format = "markdown"
	}
	return renderStaleReport(os.Stdout, report, format, lang)
}

func collectStaleInput(ctx *common.RuntimeContext) (StaleInput, []string, staleScanOptions, error) {
	opts := staleScanOptions{
		State:         strings.TrimSpace(ctx.Arg("state")),
		StaleDays:     30,
		Top:           20,
		IncludeIssues: true,
		IncludePRs:    true,
	}
	if value := strings.TrimSpace(ctx.Arg("include-issues")); value != "" {
		opts.IncludeIssues = parseBoolArg(value)
	}
	if value := strings.TrimSpace(ctx.Arg("include-prs")); value != "" {
		opts.IncludePRs = parseBoolArg(value)
	}
	if opts.State == "" {
		opts.State = "open"
	}
	if !opts.IncludeIssues && !opts.IncludePRs {
		return StaleInput{}, nil, staleScanOptions{}, fmt.Errorf("workflow +stale requires at least one of --include-issues or --include-prs")
	}

	staleDays, err := parseIntArg(ctx.Arg("stale-days"), 30, "stale-days")
	if err != nil {
		return StaleInput{}, nil, staleScanOptions{}, err
	}
	top, err := parseIntArg(ctx.Arg("top"), 20, "top")
	if err != nil {
		return StaleInput{}, nil, staleScanOptions{}, err
	}
	opts.StaleDays = staleDays
	opts.Top = top

	if path := strings.TrimSpace(ctx.Arg("from")); path != "" {
		input, err := readStaleInput(path)
		if err != nil {
			return StaleInput{}, nil, staleScanOptions{}, err
		}
		if strings.TrimSpace(input.Source) == "" {
			input.Source = "local-json"
		}
		input.Repository = repositoryFromContext(ctx, input.Repository)
		return input, nil, opts, nil
	}

	issueLimit, err := parseIntArg(ctx.Arg("issue-limit"), 20, "issue-limit")
	if err != nil {
		return StaleInput{}, nil, staleScanOptions{}, err
	}
	prLimit, err := parseIntArg(ctx.Arg("pr-limit"), 20, "pr-limit")
	if err != nil {
		return StaleInput{}, nil, staleScanOptions{}, err
	}

	input, notes, err := FetchStaleInput(ctx, StaleFetchOptions{
		State:         opts.State,
		StaleDays:     opts.StaleDays,
		IssueLimit:    issueLimit,
		PRLimit:       prLimit,
		IncludeIssues: opts.IncludeIssues,
		IncludePRs:    opts.IncludePRs,
	})
	if err != nil {
		return StaleInput{}, nil, staleScanOptions{}, err
	}
	return input, notes, opts, nil
}

func readStaleInput(path string) (StaleInput, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return StaleInput{}, fmt.Errorf("read stale input: %w", err)
	}

	var input StaleInput
	if err := json.Unmarshal(data, &input); err != nil {
		return StaleInput{}, fmt.Errorf("parse stale input: %w", err)
	}
	if len(input.Issues) == 0 && len(input.PullRequests) == 0 {
		return StaleInput{}, fmt.Errorf("parse stale input: expected issues or pull_requests in the root object")
	}
	return input, nil
}

func AnalyzeStale(input StaleInput, notes []string, opts staleScanOptions, lang string) StaleReport {
	lang = normalizeLang(lang)
	report := StaleReport{
		Repository: input.Repository,
		Source:     firstNonEmpty(strings.TrimSpace(input.Source), "local"),
		State:      opts.State,
		StaleDays:  maxInt(opts.StaleDays, 1),
		Top:        opts.Top,
		ByBucket: map[string]int{
			staleBucketFresh:  0,
			staleBucketWatch:  0,
			staleBucketStale:  0,
			staleBucketZombie: 0,
		},
	}

	candidates := make([]staleCandidate, 0, len(input.Issues)+len(input.PullRequests))
	if opts.IncludeIssues {
		for _, issue := range input.Issues {
			candidate, ok := staleCandidateFromIssue(issue)
			if !ok || !matchesState(candidate.State, opts.State) {
				continue
			}
			candidates = append(candidates, candidate)
			report.IssuesScanned++
		}
	}
	if opts.IncludePRs {
		for _, pr := range input.PullRequests {
			candidate, ok := staleCandidateFromPR(pr)
			if !ok || !matchesState(candidate.State, opts.State) {
				continue
			}
			candidates = append(candidates, candidate)
			report.PRsScanned++
		}
	}
	report.ScannedTotal = len(candidates)

	flagged := make([]StaleItem, 0, len(candidates))
	for _, candidate := range candidates {
		ageDays := apiAgeInDays(candidate.LastActivityAt)
		if ageDays > report.OldestAgeDays {
			report.OldestAgeDays = ageDays
		}
		bucket := staleBucketForAge(ageDays, report.StaleDays)
		report.ByBucket[bucket]++
		if bucket == staleBucketFresh {
			continue
		}

		item := buildStaleItem(candidate, bucket, ageDays, lang)
		flagged = append(flagged, item)
		report.FlaggedTotal++
		switch candidate.Kind {
		case staleKindIssue:
			report.IssuesFlagged++
		case staleKindPR:
			report.PRsFlagged++
		}
	}

	sort.Slice(flagged, func(i, j int) bool {
		if staleBucketRank(flagged[i].Bucket) != staleBucketRank(flagged[j].Bucket) {
			return staleBucketRank(flagged[i].Bucket) > staleBucketRank(flagged[j].Bucket)
		}
		if flagged[i].AgeDays != flagged[j].AgeDays {
			return flagged[i].AgeDays > flagged[j].AgeDays
		}
		if staleKindRank(flagged[i].Kind) != staleKindRank(flagged[j].Kind) {
			return staleKindRank(flagged[i].Kind) < staleKindRank(flagged[j].Kind)
		}
		return flagged[i].Number < flagged[j].Number
	})

	report.ShownTotal = len(flagged)
	if report.Top > 0 && report.ShownTotal > report.Top {
		report.OmittedTotal = report.ShownTotal - report.Top
		report.ShownTotal = report.Top
		report.Items = append([]StaleItem(nil), flagged[:report.Top]...)
	} else {
		report.Items = flagged
	}
	report.Recommendations = buildStaleRecommendations(report, lang)
	report.Notes = uniqueStrings(notes)
	if report.Repository == "" {
		report.Repository = "local"
	}
	return report
}

func staleCandidateFromIssue(issue IssueInput) (staleCandidate, bool) {
	title := strings.TrimSpace(issue.Title)
	if title == "" {
		return staleCandidate{}, false
	}
	lastActivity := apiLatestTime(issue.UpdatedAt, issue.CreatedAt)
	source := ""
	if !issue.UpdatedAt.IsZero() {
		source = "updated_at"
	} else if !issue.CreatedAt.IsZero() {
		source = "created_at"
	}
	return staleCandidate{
		Kind:           staleKindIssue,
		Number:         issue.Number,
		Title:          title,
		State:          issue.State,
		Author:         issue.Author,
		URL:            issue.URL,
		Labels:         append([]string(nil), issue.Labels...),
		CommentsCount:  issue.CommentsCount,
		LastActivityAt: lastActivity,
		ActivitySource: source,
	}, true
}

func staleCandidateFromPR(pr StalePullRequestInput) (staleCandidate, bool) {
	title := strings.TrimSpace(pr.Title)
	if title == "" {
		return staleCandidate{}, false
	}
	lastActivity := apiLatestTime(pr.LastActivityAt, pr.UpdatedAt, pr.CreatedAt)
	source := strings.TrimSpace(pr.ActivitySource)
	if source == "" {
		switch {
		case !pr.LastActivityAt.IsZero():
			source = "last_activity_at"
		case !pr.UpdatedAt.IsZero():
			source = "updated_at"
		case !pr.CreatedAt.IsZero():
			source = "created_at"
		}
	}
	return staleCandidate{
		Kind:           staleKindPR,
		Number:         pr.Number,
		Title:          title,
		State:          pr.State,
		Author:         pr.Author,
		URL:            pr.URL,
		BaseBranch:     pr.BaseBranch,
		HeadBranch:     pr.HeadBranch,
		CommentsCount:  pr.CommentsCount,
		LastActivityAt: lastActivity,
		ActivitySource: source,
	}, true
}

func buildStaleItem(candidate staleCandidate, bucket string, ageDays int, lang string) StaleItem {
	return StaleItem{
		Kind:             candidate.Kind,
		Number:           candidate.Number,
		Title:            candidate.Title,
		State:            firstNonEmpty(strings.TrimSpace(candidate.State), "open"),
		Author:           candidate.Author,
		URL:              candidate.URL,
		Labels:           append([]string(nil), candidate.Labels...),
		BaseBranch:       candidate.BaseBranch,
		HeadBranch:       candidate.HeadBranch,
		CommentsCount:    candidate.CommentsCount,
		LastActivityAt:   candidate.LastActivityAt,
		ActivitySource:   candidate.ActivitySource,
		AgeDays:          ageDays,
		Bucket:           bucket,
		SuggestedAction:  staleActionText(lang, candidate.Kind, bucket),
		SuggestedComment: staleCommentText(lang, candidate.Kind, bucket),
		Reasoning:        buildStaleReasoning(candidate, bucket, ageDays, lang),
	}
}

func buildStaleReasoning(candidate staleCandidate, bucket string, ageDays int, lang string) []string {
	reasons := []string{
		fmt.Sprintf(staleText(lang, "reason_age"), ageDays),
		fmt.Sprintf(staleText(lang, "reason_bucket"), bucket),
	}
	if candidate.CommentsCount > 0 {
		reasons = append(reasons, fmt.Sprintf(staleText(lang, "reason_comments"), candidate.CommentsCount))
	}
	if strings.TrimSpace(candidate.ActivitySource) != "" {
		reasons = append(reasons, fmt.Sprintf(staleText(lang, "reason_source"), candidate.ActivitySource))
	}
	if candidate.Kind == staleKindPR && candidate.HeadBranch != "" {
		reasons = append(reasons, fmt.Sprintf(staleText(lang, "reason_branches"), firstNonEmpty(candidate.BaseBranch, "?"), candidate.HeadBranch))
	}
	if candidate.Kind == staleKindIssue && len(candidate.Labels) > 0 {
		reasons = append(reasons, fmt.Sprintf(staleText(lang, "reason_labels"), strings.Join(candidate.Labels, ", ")))
	}
	return uniqueStrings(reasons)
}

func buildStaleRecommendations(report StaleReport, lang string) []string {
	recommendations := []string{}
	if report.ByBucket[staleBucketZombie] > 0 {
		recommendations = append(recommendations, staleText(lang, "rec_zombie"))
	}
	if report.PRsFlagged > 0 {
		recommendations = append(recommendations, staleText(lang, "rec_prs"))
	}
	if report.IssuesFlagged > 0 {
		recommendations = append(recommendations, staleText(lang, "rec_issues"))
	}
	if report.OmittedTotal > 0 {
		recommendations = append(recommendations, fmt.Sprintf(staleText(lang, "rec_omitted"), report.OmittedTotal))
	}
	if len(recommendations) == 0 {
		recommendations = append(recommendations, staleText(lang, "rec_clean"))
	}
	return uniqueStrings(recommendations)
}

func staleBucketForAge(ageDays, staleDays int) string {
	if staleDays <= 0 {
		staleDays = 30
	}
	switch {
	case ageDays < staleDays:
		return staleBucketFresh
	case ageDays < staleDays*2:
		return staleBucketWatch
	case ageDays < staleDays*4:
		return staleBucketStale
	default:
		return staleBucketZombie
	}
}

func staleBucketRank(bucket string) int {
	switch bucket {
	case staleBucketZombie:
		return 4
	case staleBucketStale:
		return 3
	case staleBucketWatch:
		return 2
	default:
		return 1
	}
}

func staleKindRank(kind string) int {
	switch kind {
	case staleKindPR:
		return 0
	case staleKindIssue:
		return 1
	default:
		return 2
	}
}

func matchesState(itemState, requested string) bool {
	requested = strings.TrimSpace(requested)
	if requested == "" || requested == "all" {
		return true
	}
	itemState = strings.TrimSpace(itemState)
	if itemState == "" {
		return true
	}
	return strings.EqualFold(itemState, requested)
}

func staleActionText(lang, kind, bucket string) string {
	lang = normalizeLang(lang)
	switch bucket {
	case staleBucketWatch:
		if kind == staleKindPR {
			if lang == langZH {
				return "确认贡献者是否仍在推进，并补齐待合并阻塞点。"
			}
			return "Confirm whether the contributor is still active and list the merge blockers."
		}
		if lang == langZH {
			return "补一个维护者跟进，明确下一步处理时间点。"
		}
		return "Leave a maintainer follow-up and define the next handling checkpoint."
	case staleBucketStale:
		if kind == staleKindPR {
			if lang == langZH {
				return "要求同步最新基线、重新验证测试，并确认是否继续维护。"
			}
			return "Request a rebase, rerun validation, and confirm whether the PR is still maintained."
		}
		if lang == langZH {
			return "要求补充进展或关闭条件，避免长期悬空。"
		}
		return "Ask for an update or a closing condition so the item does not stay open indefinitely."
	default:
		if kind == staleKindPR {
			if lang == langZH {
				return "优先清理长期无响应 PR，必要时建议关闭或拆分后重提。"
			}
			return "Prioritize long-idle PR cleanup and consider closing or asking for a smaller resubmission."
		}
		if lang == langZH {
			return "优先处理长期无人推进的 Issue，关闭前给出最后一次确认。"
		}
		return "Prioritize long-idle issue cleanup and give one final confirmation request before closing."
	}
}

func staleCommentText(lang, kind, bucket string) string {
	lang = normalizeLang(lang)
	if lang == langZH {
		switch bucket {
		case staleBucketWatch:
			if kind == staleKindPR {
				return "这条 PR 已经一段时间没有新的推进了。请确认当前是否还会继续维护，并说明还缺哪些合并前条件。"
			}
			return "这个条目已经一段时间没有新的进展了。请补充当前状态或下一步计划，方便维护者继续跟进。"
		case staleBucketStale:
			if kind == staleKindPR {
				return "这条 PR 已长期没有活动。请确认是否仍计划继续推进，并在回复中说明需要维护者协助的阻塞点。"
			}
			return "这个条目已长期没有活动。若问题仍然存在，请补充最新复现或处理进展；否则维护者可能会考虑关闭。"
		default:
			if kind == staleKindPR {
				return "这条 PR 已非常久没有活动。若近期没有恢复推进计划，建议关闭后在准备充分时重新提交。"
			}
			return "这个条目已非常久没有活动。若近期没有新的信息或推进计划，维护者可考虑在说明原因后关闭。"
		}
	}

	switch bucket {
	case staleBucketWatch:
		if kind == staleKindPR {
			return "This pull request has been idle for a while. Please confirm whether it is still active and list any remaining merge blockers."
		}
		return "This item has been quiet for a while. Please share the current status or next step so maintainers can continue triage."
	case staleBucketStale:
		if kind == staleKindPR {
			return "This pull request has been inactive for a long time. Please confirm whether you still plan to continue it and mention any blocker that needs maintainer help."
		}
		return "This item has been inactive for a long time. If it still needs work, please add the latest reproduction or progress details; otherwise maintainers may consider closing it."
	default:
		if kind == staleKindPR {
			return "This pull request has been inactive for a very long time. If there is no plan to continue it soon, please consider closing it and reopening with a smaller refreshed change later."
		}
		return "This item has been inactive for a very long time. If there is no new information or plan to continue it soon, maintainers may consider closing it with a short explanation."
	}
}

func staleText(lang, key string) string {
	lang = normalizeLang(lang)
	if lang == langZH {
		switch key {
		case "title":
			return "陈旧队列报告"
		case "overview":
			return "概览"
		case "items":
			return "待处理条目"
		case "recommendations":
			return "建议动作"
		case "notes":
			return "备注"
		case "no_items":
			return "没有达到陈旧阈值的条目。"
		case "no_notes":
			return "无额外备注。"
		case "reason_age":
			return "距离上次活动约 %d 天"
		case "reason_bucket":
			return "分桶：%s"
		case "reason_comments":
			return "评论/审查记录：%d"
		case "reason_source":
			return "活动时间来源：%s"
		case "reason_branches":
			return "目标分支：%s，来源分支：%s"
		case "reason_labels":
			return "标签：%s"
		case "rec_zombie":
			return "优先清理 zombie 桶中的条目，避免社区队列持续积压。"
		case "rec_prs":
			return "对陈旧 PR 优先给出继续推进或关闭建议，减少贡献者等待时间。"
		case "rec_issues":
			return "对陈旧 Issue 明确下一步动作、补充条件或关闭条件。"
		case "rec_omitted":
			return "当前结果省略了 %d 条已命中的陈旧条目，必要时可增大 --top。"
		case "rec_clean":
			return "当前扫描范围内没有命中陈旧阈值的条目，可维持现有跟进节奏。"
		}
	}

	switch key {
	case "title":
		return "Stale Queue Report"
	case "overview":
		return "Overview"
	case "items":
		return "Flagged Items"
	case "recommendations":
		return "Recommendations"
	case "notes":
		return "Notes"
	case "no_items":
		return "No items crossed the stale threshold."
	case "no_notes":
		return "No extra notes."
	case "reason_age":
		return "about %d days since the last activity"
	case "reason_bucket":
		return "bucket: %s"
	case "reason_comments":
		return "comments/reviews: %d"
	case "reason_source":
		return "activity source: %s"
	case "reason_branches":
		return "base branch: %s, head branch: %s"
	case "reason_labels":
		return "labels: %s"
	case "rec_zombie":
		return "Prioritize zombie-bucket cleanup so the community queue does not keep growing."
	case "rec_prs":
		return "Give stale PRs a clear continue-or-close decision to reduce contributor wait time."
	case "rec_issues":
		return "Define the next action, missing evidence, or closing condition for stale issues."
	case "rec_omitted":
		return "The report omitted %d flagged items; increase --top when maintainers need the full queue."
	case "rec_clean":
		return "No items crossed the stale threshold in the current scan scope."
	default:
		return key
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func maxInt(value, minValue int) int {
	if value < minValue {
		return minValue
	}
	return value
}
