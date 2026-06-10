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
	"time"

	"github.com/gitlink-org/gitlink-cli/cmd/cmdutil"
	"github.com/gitlink-org/gitlink-cli/shortcuts/common"
)

type ReleaseNotesInput struct {
	Repository   string           `json:"repository"`
	FromRef      string           `json:"from_ref,omitempty"`
	ToRef        string           `json:"to_ref,omitempty"`
	Version      string           `json:"version,omitempty"`
	PullRequests []PRSummaryInput `json:"pull_requests"`
	Source       string           `json:"source"`
}

type ReleaseNotesResult struct {
	Repository      string                `json:"repository"`
	Version         string                `json:"version,omitempty"`
	FromRef         string                `json:"from_ref,omitempty"`
	ToRef           string                `json:"to_ref,omitempty"`
	GeneratedAt     string                `json:"generated_at"`
	TotalPRs        int                   `json:"total_prs"`
	Sections        []ReleaseNotesSection `json:"sections"`
	Highlights      []string              `json:"highlights"`
	BreakingChanges []string              `json:"breaking_changes,omitempty"`
	TestingNotes    []string              `json:"testing_notes"`
	RiskSummary     map[string]int        `json:"risk_summary"`
	Source          string                `json:"source"`
}

type ReleaseNotesSection struct {
	Type  string              `json:"type"`
	Title string              `json:"title"`
	Items []ReleaseNotesEntry `json:"items"`
}

type ReleaseNotesEntry struct {
	Number     int      `json:"number,omitempty"`
	Title      string   `json:"title"`
	Author     string   `json:"author,omitempty"`
	ChangeType string   `json:"change_type"`
	RiskLevel  string   `json:"risk_level"`
	Files      int      `json:"files"`
	Commits    int      `json:"commits"`
	Notes      []string `json:"notes,omitempty"`
}

func newReleaseNotesShortcut() *common.Shortcut {
	return &common.Shortcut{
		Name:        "release-notes",
		Description: "Generate release notes from merged pull requests",
		Flags: []common.Flag{
			{Name: "from", Usage: "Read release notes input from a JSON file"},
			{Name: "version", Short: "v", Usage: "Release version or title"},
			{Name: "from-ref", Usage: "Previous release tag, branch, or commit for display"},
			{Name: "to-ref", Usage: "Target release tag, branch, or commit for display"},
			{Name: "state", Usage: "Remote pull request state to fetch", Default: "merged"},
			{Name: "page", Short: "p", Usage: "Remote pull request page", Default: "1"},
			{Name: "limit", Short: "l", Usage: "Maximum pull requests to include", Default: "30"},
			{Name: "lang", Usage: "Output language: en or zh-CN", Default: langEN},
		},
		Run: runReleaseNotes,
	}
}

func runReleaseNotes(ctx *common.RuntimeContext) error {
	lang := normalizeLang(ctx.Arg("lang"))
	input, err := collectReleaseNotesInput(ctx)
	if err != nil {
		return err
	}
	result := AnalyzeReleaseNotes(input, lang)
	format := ctx.Format
	if strings.TrimSpace(cmdutil.Format) == "" {
		format = "markdown"
	}
	rendered, err := RenderReleaseNotes(result, format, lang)
	if err != nil {
		return err
	}
	_, err = fmt.Fprint(os.Stdout, rendered)
	return err
}

func collectReleaseNotesInput(ctx *common.RuntimeContext) (ReleaseNotesInput, error) {
	if path := strings.TrimSpace(ctx.Arg("from")); path != "" {
		input, err := readReleaseNotesInput(path)
		if err != nil {
			return ReleaseNotesInput{}, err
		}
		overlayReleaseNotesFlags(ctx, &input)
		if strings.TrimSpace(input.Source) == "" {
			input.Source = "local-json"
		}
		return input, nil
	}

	limit, err := parseIntArg(ctx.Arg("limit"), 30, "limit")
	if err != nil {
		return ReleaseNotesInput{}, err
	}
	page, err := parseIntArg(ctx.Arg("page"), 1, "page")
	if err != nil {
		return ReleaseNotesInput{}, err
	}
	state := strings.TrimSpace(ctx.Arg("state"))
	if state == "" {
		state = "merged"
	}
	prs, owner, repo, err := fetchReleaseNotePullRequests(ctx, state, page, limit)
	if err != nil {
		return ReleaseNotesInput{}, err
	}
	input := ReleaseNotesInput{
		Repository:   fmt.Sprintf("%s/%s", owner, repo),
		PullRequests: prs,
		Source:       "remote-read-only-fetch",
	}
	overlayReleaseNotesFlags(ctx, &input)
	return input, nil
}

func overlayReleaseNotesFlags(ctx *common.RuntimeContext, input *ReleaseNotesInput) {
	if version := strings.TrimSpace(ctx.Arg("version")); version != "" {
		input.Version = version
	}
	if fromRef := strings.TrimSpace(ctx.Arg("from-ref")); fromRef != "" {
		input.FromRef = fromRef
	}
	if toRef := strings.TrimSpace(ctx.Arg("to-ref")); toRef != "" {
		input.ToRef = toRef
	}
}

func readReleaseNotesInput(path string) (ReleaseNotesInput, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return ReleaseNotesInput{}, fmt.Errorf("read release notes input: %w", err)
	}
	var input ReleaseNotesInput
	if err := json.Unmarshal(data, &input); err == nil && (len(input.PullRequests) > 0 || strings.TrimSpace(input.Repository) != "") {
		return input, nil
	}
	var prs []PRSummaryInput
	if err := json.Unmarshal(data, &prs); err != nil {
		return ReleaseNotesInput{}, fmt.Errorf("parse release notes input: expected ReleaseNotesInput or []PRSummaryInput: %w", err)
	}
	return ReleaseNotesInput{PullRequests: prs, Source: "local-json"}, nil
}

func fetchReleaseNotePullRequests(ctx *common.RuntimeContext, state string, page, limit int) ([]PRSummaryInput, string, string, error) {
	owner, repo, err := resolveFetchRepo(ctx, "", "")
	if err != nil {
		return nil, "", "", fmt.Errorf("workflow +release-notes remote mode requires --owner and --repo or a Git remote: %w", err)
	}
	if limit <= 0 {
		limit = 30
	}
	if page <= 0 {
		page = 1
	}
	query := url.Values{}
	query.Set("state", state)
	query.Set("page", fmt.Sprintf("%d", page))
	query.Set("limit", fmt.Sprintf("%d", limit))
	env, err := ctx.CallAPIWithQuery("GET", workflowRepoPath(owner, repo)+"/pulls", query)
	if err != nil {
		return nil, "", "", fmt.Errorf("fetch pull requests for release notes: %w", err)
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
	return prs, owner, repo, nil
}

func AnalyzeReleaseNotes(input ReleaseNotesInput, lang string) ReleaseNotesResult {
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

	riskSummary := map[string]int{}
	grouped := map[string][]ReleaseNotesEntry{}
	highlights := []string{}
	breaking := []string{}
	testing := []string{}

	for _, pr := range input.PullRequests {
		summary := AnalyzePRSummary(pr, lang)
		entry := releaseNotesEntryFromSummary(summary)
		grouped[entry.ChangeType] = append(grouped[entry.ChangeType], entry)
		riskSummary[entry.RiskLevel]++
		if entry.ChangeType == PRChangeTypeFeature || entry.RiskLevel == PRRiskHigh || entry.RiskLevel == PRRiskCritical {
			highlights = append(highlights, releaseNotesEntryLine(entry))
		}
		if isBreakingChange(pr) {
			breaking = append(breaking, releaseNotesEntryLine(entry))
		}
		testing = append(testing, summary.TestSuggestions...)
	}

	sections := buildReleaseNotesSections(grouped, lang)
	return ReleaseNotesResult{
		Repository:      repository,
		Version:         strings.TrimSpace(input.Version),
		FromRef:         strings.TrimSpace(input.FromRef),
		ToRef:           strings.TrimSpace(input.ToRef),
		GeneratedAt:     time.Now().UTC().Format(time.RFC3339),
		TotalPRs:        len(input.PullRequests),
		Sections:        sections,
		Highlights:      limitStrings(uniqueStrings(highlights), 10),
		BreakingChanges: uniqueStrings(breaking),
		TestingNotes:    limitStrings(uniqueStrings(testing), 12),
		RiskSummary:     riskSummary,
		Source:          source,
	}
}

func releaseNotesEntryFromSummary(summary PRSummaryResult) ReleaseNotesEntry {
	notes := []string{}
	if summary.RiskLevel == PRRiskHigh || summary.RiskLevel == PRRiskCritical {
		notes = append(notes, fmt.Sprintf("risk:%s", summary.RiskLevel))
	}
	if len(summary.ReviewFocus) > 0 {
		notes = append(notes, summary.ReviewFocus[0])
	}
	return ReleaseNotesEntry{
		Number:     summary.Number,
		Title:      summary.Title,
		Author:     summary.Author,
		ChangeType: summary.ChangeType,
		RiskLevel:  summary.RiskLevel,
		Files:      summary.ChangedFilesCount,
		Commits:    summary.CommitCount,
		Notes:      uniqueStrings(notes),
	}
}

func buildReleaseNotesSections(grouped map[string][]ReleaseNotesEntry, lang string) []ReleaseNotesSection {
	order := []string{
		PRChangeTypeFeature,
		PRChangeTypeFix,
		PRChangeTypeDocs,
		PRChangeTypeTest,
		PRChangeTypeRefactor,
		PRChangeTypeCI,
		PRChangeTypeMixed,
		PRChangeTypeUnknown,
	}
	sections := []ReleaseNotesSection{}
	for _, changeType := range order {
		items := grouped[changeType]
		if len(items) == 0 {
			continue
		}
		sort.SliceStable(items, func(i, j int) bool {
			if items[i].RiskLevel != items[j].RiskLevel {
				return releaseNotesRiskWeight(items[i].RiskLevel) > releaseNotesRiskWeight(items[j].RiskLevel)
			}
			return items[i].Number < items[j].Number
		})
		sections = append(sections, ReleaseNotesSection{
			Type:  changeType,
			Title: releaseNotesSectionTitle(changeType, lang),
			Items: items,
		})
	}
	return sections
}

func releaseNotesRiskWeight(risk string) int {
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

func releaseNotesSectionTitle(changeType, lang string) string {
	if normalizeLang(lang) == langZH {
		switch changeType {
		case PRChangeTypeFeature:
			return "新增能力"
		case PRChangeTypeFix:
			return "问题修复"
		case PRChangeTypeDocs:
			return "文档"
		case PRChangeTypeTest:
			return "测试"
		case PRChangeTypeRefactor:
			return "重构"
		case PRChangeTypeCI:
			return "工程与 CI"
		case PRChangeTypeMixed:
			return "综合变更"
		default:
			return "其他变更"
		}
	}
	switch changeType {
	case PRChangeTypeFeature:
		return "Features"
	case PRChangeTypeFix:
		return "Fixes"
	case PRChangeTypeDocs:
		return "Documentation"
	case PRChangeTypeTest:
		return "Tests"
	case PRChangeTypeRefactor:
		return "Refactoring"
	case PRChangeTypeCI:
		return "Build and CI"
	case PRChangeTypeMixed:
		return "Mixed Changes"
	default:
		return "Other Changes"
	}
}

func isBreakingChange(pr PRSummaryInput) bool {
	text := strings.ToLower(prTextCorpus(pr, true))
	for _, keyword := range []string{"breaking change", "breaking-change", "incompatible", "migration required", "remove deprecated", "drop support"} {
		if strings.Contains(text, keyword) {
			return true
		}
	}
	return false
}

func releaseNotesEntryLine(entry ReleaseNotesEntry) string {
	prefix := ""
	if entry.Number > 0 {
		prefix = fmt.Sprintf("#%d ", entry.Number)
	}
	if strings.TrimSpace(entry.Author) != "" {
		return fmt.Sprintf("%s%s (@%s)", prefix, entry.Title, entry.Author)
	}
	return prefix + entry.Title
}

func limitStrings(values []string, limit int) []string {
	if limit <= 0 || len(values) <= limit {
		return values
	}
	return values[:limit]
}

func RenderReleaseNotes(result ReleaseNotesResult, format string, lang string) (string, error) {
	var buf bytes.Buffer
	switch normalizeFormat(format) {
	case "json":
		if err := writeJSON(&buf, result); err != nil {
			return "", err
		}
	case "markdown":
		if err := writeReleaseNotesMarkdown(&buf, result, lang); err != nil {
			return "", err
		}
	case "table":
		if err := writeReleaseNotesTable(&buf, result); err != nil {
			return "", err
		}
	default:
		return "", fmt.Errorf("unsupported workflow output format %q", format)
	}
	return buf.String(), nil
}

func writeReleaseNotesMarkdown(buf *bytes.Buffer, result ReleaseNotesResult, lang string) error {
	title := "Release Notes"
	if normalizeLang(lang) == langZH {
		title = "发布说明"
	}
	if strings.TrimSpace(result.Version) != "" {
		title = fmt.Sprintf("%s %s", title, result.Version)
	}
	if _, err := fmt.Fprintf(buf, "# %s\n\n", title); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(buf, "- Repository: `%s`\n- Pull requests: `%d`\n- Source: `%s`\n", result.Repository, result.TotalPRs, result.Source); err != nil {
		return err
	}
	if result.FromRef != "" || result.ToRef != "" {
		if _, err := fmt.Fprintf(buf, "- Range: `%s` -> `%s`\n", result.FromRef, result.ToRef); err != nil {
			return err
		}
	}
	if _, err := fmt.Fprintln(buf); err != nil {
		return err
	}
	if len(result.Highlights) > 0 {
		if _, err := fmt.Fprintln(buf, "## Highlights"); err != nil {
			return err
		}
		if _, err := fmt.Fprintln(buf); err != nil {
			return err
		}
		for _, item := range result.Highlights {
			if _, err := fmt.Fprintf(buf, "- %s\n", item); err != nil {
				return err
			}
		}
		if _, err := fmt.Fprintln(buf); err != nil {
			return err
		}
	}
	if len(result.BreakingChanges) > 0 {
		if _, err := fmt.Fprintln(buf, "## Breaking Changes"); err != nil {
			return err
		}
		if _, err := fmt.Fprintln(buf); err != nil {
			return err
		}
		for _, item := range result.BreakingChanges {
			if _, err := fmt.Fprintf(buf, "- %s\n", item); err != nil {
				return err
			}
		}
		if _, err := fmt.Fprintln(buf); err != nil {
			return err
		}
	}
	for _, section := range result.Sections {
		if _, err := fmt.Fprintf(buf, "## %s\n\n", section.Title); err != nil {
			return err
		}
		for _, item := range section.Items {
			line := releaseNotesEntryLine(item)
			meta := []string{item.RiskLevel}
			if item.Files > 0 {
				meta = append(meta, fmt.Sprintf("%d files", item.Files))
			}
			if item.Commits > 0 {
				meta = append(meta, fmt.Sprintf("%d commits", item.Commits))
			}
			if _, err := fmt.Fprintf(buf, "- %s `%s`\n", line, strings.Join(meta, ", ")); err != nil {
				return err
			}
		}
		if _, err := fmt.Fprintln(buf); err != nil {
			return err
		}
	}
	if len(result.TestingNotes) > 0 {
		if _, err := fmt.Fprintln(buf, "## Suggested Verification"); err != nil {
			return err
		}
		if _, err := fmt.Fprintln(buf); err != nil {
			return err
		}
		for _, item := range result.TestingNotes {
			if _, err := fmt.Fprintf(buf, "- %s\n", item); err != nil {
				return err
			}
		}
	}
	return nil
}

func writeReleaseNotesTable(buf *bytes.Buffer, result ReleaseNotesResult) error {
	tw := tabwriter.NewWriter(buf, 0, 0, 2, ' ', 0)
	if _, err := fmt.Fprintln(tw, "TYPE\tPR\tTITLE\tRISK\tFILES\tCOMMITS"); err != nil {
		return err
	}
	for _, section := range result.Sections {
		for _, item := range section.Items {
			number := "-"
			if item.Number > 0 {
				number = fmt.Sprintf("#%d", item.Number)
			}
			if _, err := fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%d\t%d\n",
				section.Type,
				number,
				truncateTableText(item.Title, 80),
				item.RiskLevel,
				item.Files,
				item.Commits,
			); err != nil {
				return err
			}
		}
	}
	return tw.Flush()
}
