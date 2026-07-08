package workflow

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/gitlink-org/gitlink-cli/cmd/cmdutil"
	"github.com/gitlink-org/gitlink-cli/shortcuts/common"
)

const (
	ReleaseNotesBreaking      = "breaking_changes"
	ReleaseNotesFeatures      = "features"
	ReleaseNotesBugFixes      = "bug_fixes"
	ReleaseNotesDocumentation = "documentation"
	ReleaseNotesTests         = "tests"
	ReleaseNotesRefactoring   = "refactoring"
	ReleaseNotesChores        = "chores"
)

type ReleaseNotesInput struct {
	Repository   string               `json:"repository"`
	Version      string               `json:"version"`
	FromRef      string               `json:"from_ref"`
	ToRef        string               `json:"to_ref"`
	PullRequests []ReleaseNotesPR     `json:"pull_requests"`
	Commits      []ReleaseNotesCommit `json:"commits"`
	Source       string               `json:"source"`
}

type ReleaseNotesPR struct {
	Number int      `json:"number"`
	Title  string   `json:"title"`
	Author string   `json:"author"`
	URL    string   `json:"url,omitempty"`
	Files  []string `json:"files,omitempty"`
}

type ReleaseNotesCommit struct {
	SHA     string   `json:"sha"`
	Message string   `json:"message"`
	Author  string   `json:"author"`
	URL     string   `json:"url,omitempty"`
	Files   []string `json:"files,omitempty"`
}

type ReleaseNotesResult struct {
	Repository        string                `json:"repository"`
	Version           string                `json:"version"`
	FromRef           string                `json:"from_ref"`
	ToRef             string                `json:"to_ref"`
	Summary           string                `json:"summary"`
	Sections          []ReleaseNotesSection `json:"sections"`
	Contributors      []string              `json:"contributors"`
	CommitsCount      int                   `json:"commits_count"`
	PullRequestsCount int                   `json:"pull_requests_count"`
	BreakingChanges   []string              `json:"breaking_changes"`
	Reasoning         []string              `json:"reasoning"`
	Source            string                `json:"source"`
}

type ReleaseNotesSection struct {
	Key   string             `json:"key"`
	Title string             `json:"title"`
	Items []ReleaseNotesItem `json:"items"`
}

type ReleaseNotesItem struct {
	Title    string   `json:"title"`
	Author   string   `json:"author,omitempty"`
	PRNumber int      `json:"pr_number,omitempty"`
	SHA      string   `json:"sha,omitempty"`
	URL      string   `json:"url,omitempty"`
	Reasons  []string `json:"reasons,omitempty"`
}

func newReleaseNotesShortcut() *common.Shortcut {
	return &common.Shortcut{
		Name:        "release-notes",
		Description: "Generate deterministic release notes from commits and pull requests",
		Flags: []common.Flag{
			{Name: "from", Usage: "Read release notes input from a JSON file"},
			{Name: "from-ref", Usage: "Start ref for remote read-only compare fetch"},
			{Name: "to-ref", Usage: "End ref for remote read-only compare fetch", Default: "master"},
			{Name: "version", Usage: "Release version label. Defaults to --to-ref when omitted"},
			{Name: "max-commits", Usage: "Maximum commits to analyze in remote mode", Default: "200"},
			{Name: "include-prs", Usage: "Include pull request signals when available", Bool: true, Default: "true"},
			{Name: "lang", Usage: "Output language: en or zh-CN", Default: langEN},
		},
		Run: runReleaseNotes,
	}
}

func runReleaseNotes(ctx *common.RuntimeContext) error {
	lang := normalizeLang(ctx.Arg("lang"))
	input, notes, err := collectReleaseNotesInput(ctx)
	if err != nil {
		return err
	}
	result := AnalyzeReleaseNotes(input, lang)
	for _, note := range notes {
		if note.Metric == "" && note.Note == "" {
			continue
		}
		result.Reasoning = append(result.Reasoning, fmt.Sprintf("%s: %s", note.Metric, note.Note))
	}

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

func collectReleaseNotesInput(ctx *common.RuntimeContext) (ReleaseNotesInput, []ScoringNote, error) {
	if path := strings.TrimSpace(ctx.Arg("from")); path != "" {
		input, err := readReleaseNotesInput(path)
		if err != nil {
			return ReleaseNotesInput{}, nil, err
		}
		if strings.TrimSpace(input.Source) == "" {
			input.Source = "local-json"
		}
		return input, nil, nil
	}

	maxCommits, err := parseIntArg(ctx.Arg("max-commits"), 200, "max-commits")
	if err != nil {
		return ReleaseNotesInput{}, nil, err
	}
	return FetchReleaseNotesInput(ctx, ReleaseNotesFetchOptions{
		FromRef:    ctx.Arg("from-ref"),
		ToRef:      ctx.Arg("to-ref"),
		Version:    ctx.Arg("version"),
		MaxCommits: maxCommits,
		IncludePRs: parseBoolArgDefault(ctx.Arg("include-prs"), true),
	})
}

func readReleaseNotesInput(path string) (ReleaseNotesInput, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return ReleaseNotesInput{}, fmt.Errorf("read release notes input: %w", err)
	}
	var input ReleaseNotesInput
	if err := json.Unmarshal(data, &input); err != nil {
		return ReleaseNotesInput{}, fmt.Errorf("parse release notes input: %w", err)
	}
	if strings.TrimSpace(input.Repository) == "" && len(input.Commits) == 0 && len(input.PullRequests) == 0 {
		return ReleaseNotesInput{}, fmt.Errorf("parse release notes input: expected ReleaseNotesInput root object")
	}
	return input, nil
}

func AnalyzeReleaseNotes(input ReleaseNotesInput, lang string) ReleaseNotesResult {
	lang = normalizeLang(lang)
	version := strings.TrimSpace(input.Version)
	if version == "" {
		version = strings.TrimSpace(input.ToRef)
	}
	if version == "" {
		version = "unreleased"
	}

	sectionsByKey := map[string][]ReleaseNotesItem{}
	breakingChanges := []string{}
	contributors := []string{}
	reasoning := []string{}

	for _, pr := range input.PullRequests {
		title := firstLine(pr.Title)
		key, reasons := classifyReleaseNotesItem(title, pr.Files)
		item := ReleaseNotesItem{
			Title:    title,
			Author:   strings.TrimSpace(pr.Author),
			PRNumber: pr.Number,
			URL:      strings.TrimSpace(pr.URL),
			Reasons:  reasons,
		}
		sectionsByKey[key] = append(sectionsByKey[key], item)
		if key == ReleaseNotesBreaking {
			breakingChanges = append(breakingChanges, title)
		}
		if item.Author != "" {
			contributors = append(contributors, item.Author)
		}
		reasoning = append(reasoning, fmt.Sprintf("pr #%d classified as %s", pr.Number, key))
	}

	for _, commit := range input.Commits {
		title := firstLine(commit.Message)
		key, reasons := classifyReleaseNotesItem(title, commit.Files)
		item := ReleaseNotesItem{
			Title:   title,
			Author:  strings.TrimSpace(commit.Author),
			SHA:     shortSHA(commit.SHA),
			URL:     strings.TrimSpace(commit.URL),
			Reasons: reasons,
		}
		sectionsByKey[key] = append(sectionsByKey[key], item)
		if key == ReleaseNotesBreaking {
			breakingChanges = append(breakingChanges, title)
		}
		if item.Author != "" {
			contributors = append(contributors, item.Author)
		}
		if item.SHA != "" {
			reasoning = append(reasoning, fmt.Sprintf("commit %s classified as %s", item.SHA, key))
		}
	}

	source := strings.TrimSpace(input.Source)
	if source == "" {
		source = "local"
	}

	sections := make([]ReleaseNotesSection, 0, len(sectionsByKey))
	for _, key := range releaseNotesSectionOrder() {
		items := sectionsByKey[key]
		if len(items) == 0 {
			continue
		}
		sections = append(sections, ReleaseNotesSection{
			Key:   key,
			Title: releaseNotesSectionTitle(lang, key),
			Items: items,
		})
	}

	return ReleaseNotesResult{
		Repository:        input.Repository,
		Version:           version,
		FromRef:           input.FromRef,
		ToRef:             input.ToRef,
		Summary:           buildReleaseNotesSummary(lang, len(input.Commits), len(input.PullRequests)),
		Sections:          sections,
		Contributors:      sortedUniqueStrings(contributors),
		CommitsCount:      len(input.Commits),
		PullRequestsCount: len(input.PullRequests),
		BreakingChanges:   uniqueStrings(breakingChanges),
		Reasoning:         uniqueStrings(reasoning),
		Source:            source,
	}
}

func classifyReleaseNotesItem(text string, files []string) (string, []string) {
	corpus := strings.ToLower(strings.TrimSpace(text))
	normalizedFiles := make([]string, 0, len(files))
	for _, file := range files {
		normalizedFiles = append(normalizedFiles, normalizedPath(file))
	}

	if containsAny(corpus, []string{"breaking change", "breaking:"}) || strings.Contains(corpus, "!:") {
		return ReleaseNotesBreaking, []string{"keyword:breaking"}
	}
	if containsAny(corpus, []string{"feat", "feature", "add", "support", "implement"}) {
		return ReleaseNotesFeatures, []string{"keyword:feature"}
	}
	if containsAny(corpus, []string{"fix", "bug", "resolve", "crash", "error"}) {
		return ReleaseNotesBugFixes, []string{"keyword:fix"}
	}
	if containsAny(corpus, []string{"docs", "doc", "readme", "guide", "example"}) || releaseNotesTouchesDocs(normalizedFiles) {
		return ReleaseNotesDocumentation, []string{"keyword:docs"}
	}
	if containsAny(corpus, []string{"test", "tests", "coverage"}) || releaseNotesTouchesTests(normalizedFiles) {
		return ReleaseNotesTests, []string{"keyword:test"}
	}
	if containsAny(corpus, []string{"refactor", "cleanup", "simplify", "restructure"}) {
		return ReleaseNotesRefactoring, []string{"keyword:refactor"}
	}
	return ReleaseNotesChores, []string{"fallback:chore"}
}

func releaseNotesSectionTitle(lang, key string) string {
	zh := normalizeLang(lang) == langZH
	switch key {
	case ReleaseNotesBreaking:
		if zh {
			return "破坏性变更"
		}
		return "Breaking Changes"
	case ReleaseNotesFeatures:
		if zh {
			return "新功能"
		}
		return "Features"
	case ReleaseNotesBugFixes:
		if zh {
			return "问题修复"
		}
		return "Bug Fixes"
	case ReleaseNotesDocumentation:
		if zh {
			return "文档"
		}
		return "Documentation"
	case ReleaseNotesTests:
		if zh {
			return "测试"
		}
		return "Tests"
	case ReleaseNotesRefactoring:
		if zh {
			return "重构"
		}
		return "Refactoring"
	case ReleaseNotesChores:
		if zh {
			return "维护"
		}
		return "Chores"
	default:
		return key
	}
}

func releaseNotesSectionOrder() []string {
	return []string{
		ReleaseNotesBreaking,
		ReleaseNotesFeatures,
		ReleaseNotesBugFixes,
		ReleaseNotesDocumentation,
		ReleaseNotesTests,
		ReleaseNotesRefactoring,
		ReleaseNotesChores,
	}
}

func buildReleaseNotesSummary(lang string, commits int, prs int) string {
	if normalizeLang(lang) == langZH {
		return fmt.Sprintf("共分析 %d 个 commit、%d 个 PR。", commits, prs)
	}
	return fmt.Sprintf("Analyzed %d %s and %d %s.", commits, pluralizeReleaseNotesNoun(commits, "commit"), prs, pluralizeReleaseNotesNoun(prs, "pull request"))
}

func pluralizeReleaseNotesNoun(count int, singular string) string {
	if count == 1 {
		return singular
	}
	return singular + "s"
}

func releaseNotesTouchesDocs(files []string) bool {
	for _, file := range files {
		if isDocsPath(file) {
			return true
		}
	}
	return false
}

func releaseNotesTouchesTests(files []string) bool {
	for _, file := range files {
		if isTestPath(file) {
			return true
		}
	}
	return false
}

func shortSHA(sha string) string {
	sha = strings.TrimSpace(sha)
	if len(sha) <= 7 {
		return sha
	}
	return sha[:7]
}

func sortedUniqueStrings(values []string) []string {
	result := uniqueStrings(values)
	sort.Strings(result)
	return result
}
