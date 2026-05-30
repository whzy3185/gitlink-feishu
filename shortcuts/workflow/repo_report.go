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

type RepoReportInput struct {
	Repository   string           `json:"repository"`
	Health       *HealthInput     `json:"health,omitempty"`
	Issues       []IssueInput     `json:"issues,omitempty"`
	PullRequests []PRSummaryInput `json:"pull_requests,omitempty"`
	Source       string           `json:"source"`
}

type RepoReportResult struct {
	Repository      string           `json:"repository"`
	Health          *HealthResult    `json:"health,omitempty"`
	IssueSummary    RepoIssueSummary `json:"issue_summary"`
	PRSummary       RepoPRSummary    `json:"pr_summary"`
	Recommendations []string         `json:"recommendations"`
	RiskLevel       string           `json:"risk_level"`
	ReportScore     int              `json:"report_score"`
	Sections        []string         `json:"sections"`
	Reasoning       []string         `json:"reasoning"`
	Source          string           `json:"source"`
}

type RepoIssueSummary struct {
	Total       int            `json:"total"`
	ByType      map[string]int `json:"by_type"`
	ByPriority  map[string]int `json:"by_priority"`
	HighRisk    int            `json:"high_risk"`
	MissingInfo int            `json:"missing_info"`
}

type RepoPRSummary struct {
	Total       int            `json:"total"`
	ByType      map[string]int `json:"by_type"`
	ByRisk      map[string]int `json:"by_risk"`
	HighRisk    int            `json:"high_risk"`
	ReviewFocus []string       `json:"review_focus"`
}

func newRepoReportShortcut() *common.Shortcut {
	return &common.Shortcut{
		Name:        "repo-report",
		Description: "Generate a read-only repository workflow report",
		Flags: []common.Flag{
			{Name: "from", Usage: "Read repository report input from a JSON file"},
			{Name: "issue-limit", Usage: "Maximum issues to fetch and analyze", Default: "20"},
			{Name: "pr-limit", Usage: "Maximum pull requests to fetch and summarize", Default: "10"},
			{Name: "stale-days", Usage: "Days before an issue or PR is considered stale", Default: "30"},
			{Name: "include-issues", Usage: "Include issue triage summary", Bool: true, Default: "true"},
			{Name: "include-prs", Usage: "Include pull request summary", Bool: true, Default: "true"},
			{Name: "include-health", Usage: "Include repository health summary", Bool: true, Default: "true"},
			{Name: "lang", Usage: "Output language: en or zh-CN", Default: langEN},
		},
		Run: runRepoReport,
	}
}

func runRepoReport(ctx *common.RuntimeContext) error {
	lang := normalizeLang(ctx.Arg("lang"))
	input, notes, err := collectRepoReportInput(ctx)
	if err != nil {
		return err
	}
	result := AnalyzeRepoReport(input, lang)
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
	rendered, err := RenderRepoReport(result, format, lang)
	if err != nil {
		return err
	}
	_, err = fmt.Fprint(os.Stdout, rendered)
	return err
}

func collectRepoReportInput(ctx *common.RuntimeContext) (RepoReportInput, []ScoringNote, error) {
	if path := strings.TrimSpace(ctx.Arg("from")); path != "" {
		input, err := readRepoReportInput(path)
		if err != nil {
			return RepoReportInput{}, nil, err
		}
		if strings.TrimSpace(input.Source) == "" {
			input.Source = "local-json"
		}
		return input, nil, nil
	}

	issueLimit, err := parseIntArg(ctx.Arg("issue-limit"), 20, "issue-limit")
	if err != nil {
		return RepoReportInput{}, nil, err
	}
	prLimit, err := parseIntArg(ctx.Arg("pr-limit"), 10, "pr-limit")
	if err != nil {
		return RepoReportInput{}, nil, err
	}
	staleDays, err := parseIntArg(ctx.Arg("stale-days"), 30, "stale-days")
	if err != nil {
		return RepoReportInput{}, nil, err
	}

	return FetchRepoReportInput(ctx, RepoReportFetchOptions{
		IssueLimit:    issueLimit,
		PRLimit:       prLimit,
		StaleDays:     staleDays,
		IncludeIssues: parseBoolArg(ctx.Arg("include-issues")),
		IncludePRs:    parseBoolArg(ctx.Arg("include-prs")),
		IncludeHealth: parseBoolArg(ctx.Arg("include-health")),
	})
}

func readRepoReportInput(path string) (RepoReportInput, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return RepoReportInput{}, fmt.Errorf("read repo report input: %w", err)
	}
	var input RepoReportInput
	if err := json.Unmarshal(data, &input); err != nil {
		return RepoReportInput{}, fmt.Errorf("parse repo report input: %w", err)
	}
	if strings.TrimSpace(input.Repository) == "" && input.Health == nil && len(input.Issues) == 0 && len(input.PullRequests) == 0 {
		return RepoReportInput{}, fmt.Errorf("parse repo report input: expected RepoReportInput root object")
	}
	return input, nil
}

func AnalyzeRepoReport(input RepoReportInput, lang string) RepoReportResult {
	lang = normalizeLang(lang)
	source := strings.TrimSpace(input.Source)
	if source == "" {
		source = "local"
	}
	repository := strings.TrimSpace(input.Repository)
	sections := []string{}
	reasoning := []string{}

	var healthResult *HealthResult
	baseScore := 70
	if input.Health != nil {
		health := *input.Health
		if repository == "" {
			repository = health.Repository
		}
		scored := ScoreHealth(health, lang)
		healthResult = &scored
		baseScore = scored.HealthScore
		sections = append(sections, "health")
		reasoning = append(reasoning, fmt.Sprintf("health score: %d", scored.HealthScore))
	} else {
		reasoning = append(reasoning, repoReportText(lang, "health_missing"))
	}

	issueSummary, issueResults := summarizeRepoIssues(input.Issues, lang)
	if len(input.Issues) > 0 {
		sections = append(sections, "issues")
		reasoning = append(reasoning, fmt.Sprintf("issues analyzed: %d", len(input.Issues)))
	}

	prSummary, prResults := summarizeRepoPRs(input.PullRequests, lang)
	if len(input.PullRequests) > 0 {
		sections = append(sections, "pull_requests")
		reasoning = append(reasoning, fmt.Sprintf("pull requests analyzed: %d", len(input.PullRequests)))
	}

	reportScore := computeRepoReportScore(baseScore, input.Health != nil, issueSummary, prSummary)
	risk := riskLevel(reportScore)
	if hasSecurityP0(issueResults) || hasCriticalPR(prResults) {
		risk = "critical"
		reportScore = minInt(reportScore, 39)
		reasoning = append(reasoning, repoReportText(lang, "critical_signal"))
	}

	recommendations := buildRepoReportRecommendations(lang, healthResult, issueSummary, prSummary, risk)
	if repository == "" {
		repository = "local"
	}

	return RepoReportResult{
		Repository:      repository,
		Health:          healthResult,
		IssueSummary:    issueSummary,
		PRSummary:       prSummary,
		Recommendations: recommendations,
		RiskLevel:       risk,
		ReportScore:     reportScore,
		Sections:        sections,
		Reasoning:       uniqueStrings(reasoning),
		Source:          source,
	}
}

func summarizeRepoIssues(issues []IssueInput, lang string) (RepoIssueSummary, []TriageResult) {
	summary := RepoIssueSummary{
		ByType:     map[string]int{},
		ByPriority: map[string]int{},
	}
	results := make([]TriageResult, 0, len(issues))
	for _, issue := range issues {
		result := AnalyzeIssue(issue, lang)
		results = append(results, result)
		summary.Total++
		summary.ByType[result.DetectedType]++
		summary.ByPriority[result.Priority]++
		if result.Priority == PriorityP0 || result.Priority == PriorityP1 || containsString(result.RiskFlags, RiskSecuritySensitive) {
			summary.HighRisk++
		}
		if len(result.MissingInformation) > 0 {
			summary.MissingInfo++
		}
	}
	return summary, results
}

func summarizeRepoPRs(inputs []PRSummaryInput, lang string) (RepoPRSummary, []PRSummaryResult) {
	summary := RepoPRSummary{
		ByType: map[string]int{},
		ByRisk: map[string]int{},
	}
	results := make([]PRSummaryResult, 0, len(inputs))
	focus := []string{}
	for _, input := range inputs {
		result := AnalyzePRSummary(input, lang)
		results = append(results, result)
		summary.Total++
		summary.ByType[result.ChangeType]++
		summary.ByRisk[result.RiskLevel]++
		if result.RiskLevel == PRRiskHigh || result.RiskLevel == PRRiskCritical {
			summary.HighRisk++
		}
		focus = append(focus, result.ReviewFocus...)
	}
	summary.ReviewFocus = uniqueStrings(focus)
	sort.Strings(summary.ReviewFocus)
	return summary, results
}

func computeRepoReportScore(baseScore int, hasHealth bool, issueSummary RepoIssueSummary, prSummary RepoPRSummary) int {
	score := baseScore
	if !hasHealth {
		score = 70
	}
	score -= issueSummary.HighRisk * 8
	score -= issueSummary.MissingInfo * 3
	score -= prSummary.HighRisk * 8
	score -= prSummary.ByRisk[PRRiskCritical] * 10
	if issueSummary.Total == 0 && prSummary.Total == 0 && !hasHealth {
		score = 50
	}
	return clampInt(score, 100)
}

func hasSecurityP0(results []TriageResult) bool {
	for _, result := range results {
		if result.Priority == PriorityP0 || result.DetectedType == IssueTypeSecurity || containsString(result.RiskFlags, RiskSecuritySensitive) {
			return true
		}
	}
	return false
}

func hasCriticalPR(results []PRSummaryResult) bool {
	for _, result := range results {
		if result.RiskLevel == PRRiskCritical {
			return true
		}
	}
	return false
}

func buildRepoReportRecommendations(lang string, health *HealthResult, issueSummary RepoIssueSummary, prSummary RepoPRSummary, risk string) []string {
	recommendations := []string{}
	if issueSummary.ByPriority[PriorityP0] > 0 || issueSummary.ByType[IssueTypeSecurity] > 0 {
		recommendations = append(recommendations, repoReportText(lang, "rec_security_issues"))
	}
	if issueSummary.MissingInfo > 0 {
		recommendations = append(recommendations, repoReportText(lang, "rec_missing_info"))
	}
	if prSummary.HighRisk > 0 {
		recommendations = append(recommendations, repoReportText(lang, "rec_high_risk_prs"))
	}
	if health != nil {
		recommendations = append(recommendations, health.Recommendations...)
	}
	if health != nil && health.HealthScore < 65 {
		recommendations = append(recommendations, repoReportText(lang, "rec_health"))
	}
	if risk == "low" && len(recommendations) == 0 {
		recommendations = append(recommendations, repoReportText(lang, "rec_maintain_report"))
	}
	if len(recommendations) == 0 {
		recommendations = append(recommendations, repoReportText(lang, "rec_review_report"))
	}
	recommendations = uniqueStrings(recommendations)
	if len(recommendations) > 8 {
		recommendations = recommendations[:8]
	}
	return recommendations
}

func repoReportText(lang, key string) string {
	zh := normalizeLang(lang) == langZH
	switch key {
	case "title":
		if zh {
			return "仓库工作流报告"
		}
		return "Repository Workflow Report"
	case "overview":
		if zh {
			return "总览"
		}
		return "Overview"
	case "health":
		if zh {
			return "健康度摘要"
		}
		return "Health Summary"
	case "issues":
		if zh {
			return "Issue 分诊摘要"
		}
		return "Issue Triage Summary"
	case "prs":
		if zh {
			return "PR 审阅摘要"
		}
		return "PR Review Summary"
	case "recommendations":
		if zh {
			return "建议操作"
		}
		return "Recommendations"
	case "reasoning":
		if zh {
			return "判断依据"
		}
		return "Reasoning"
	case "not_available":
		if zh {
			return "不可用"
		}
		return "Not available"
	case "health_missing":
		if zh {
			return "未提供健康度输入。"
		}
		return "health input not provided"
	case "critical_signal":
		if zh {
			return "发现安全 P0 Issue 或 critical PR，整体风险上调。"
		}
		return "security P0 issue or critical PR raised overall risk"
	case "rec_security_issues":
		if zh {
			return "优先处理安全相关或 P0 Issue。"
		}
		return "Prioritize security-related or P0 issues."
	case "rec_missing_info":
		if zh {
			return "要求补充复现步骤、版本、命令输出或日志。"
		}
		return "Request missing reproduction steps, version, command output, or logs."
	case "rec_high_risk_prs":
		if zh {
			return "优先审阅 high / critical 风险 PR。"
		}
		return "Prioritize high or critical risk pull requests."
	case "rec_health":
		if zh {
			return "根据健康度建议降低仓库治理风险。"
		}
		return "Use the health recommendations to reduce repository governance risk."
	case "rec_maintain_report":
		if zh {
			return "保持当前维护节奏，并定期复查仓库工作流报告。"
		}
		return "Maintain the current workflow and review the repository report regularly."
	case "rec_review_report":
		if zh {
			return "复查报告中的风险项并安排下一步维护动作。"
		}
		return "Review report risks and schedule the next maintenance actions."
	default:
		return key
	}
}
