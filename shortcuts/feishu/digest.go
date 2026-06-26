package feishu

import (
	"fmt"
	"io"
	"strings"
	"text/tabwriter"

	"github.com/gitlink-org/gitlink-cli/shortcuts/workflow"
)

type RoleDigest struct {
	Role                string   `json:"role"`
	Repository          string   `json:"repository"`
	RepositoryURL       string   `json:"repository_url,omitempty"`
	DocURL              string   `json:"doc_url,omitempty"`
	HealthScore         *int     `json:"health_score,omitempty"`
	HealthRisk          string   `json:"health_risk,omitempty"`
	RiskLevel           string   `json:"risk_level"`
	ReportScore         int      `json:"report_score"`
	IssueTotal          int      `json:"issue_total"`
	IssueHighRisk       int      `json:"issue_high_risk"`
	IssueMissingInfo    int      `json:"issue_missing_info"`
	PRTotal             int      `json:"pr_total"`
	PRHighRisk          int      `json:"pr_high_risk"`
	ReviewFocus         []string `json:"review_focus,omitempty"`
	Recommendations     []string `json:"recommendations,omitempty"`
	AttentionItems      []string `json:"attention_items,omitempty"`
	NextSteps           []string `json:"next_steps,omitempty"`
	BoundaryDescription string   `json:"boundary_description"`
}

func BuildOwnerDigest(report workflow.RepoReportResult, docURL string) RoleDigest {
	healthScore, healthRisk := digestHealth(report)
	attention := []string{}
	if report.IssueSummary.HighRisk > 0 {
		attention = append(attention, fmt.Sprintf("%d high-risk issues need maintainer triage", report.IssueSummary.HighRisk))
	}
	if report.IssueSummary.MissingInfo > 0 {
		attention = append(attention, fmt.Sprintf("%d issues are missing required information", report.IssueSummary.MissingInfo))
	}
	if report.PRSummary.HighRisk > 0 {
		attention = append(attention, fmt.Sprintf("%d high-risk pull requests need owner review", report.PRSummary.HighRisk))
	}
	if healthScore != nil && *healthScore < 65 {
		attention = append(attention, fmt.Sprintf("repository health score is %d", *healthScore))
	}
	if len(attention) == 0 {
		attention = append(attention, "No critical owner action was detected in the workflow report.")
	}
	nextSteps := []string{
		"Review high-risk issues and PRs first.",
		"Use the Feishu report document for full context when available.",
		"Keep GitLink write actions outside this digest; card buttons are navigation-only.",
	}
	if len(report.Recommendations) > 0 {
		nextSteps = append(report.Recommendations, nextSteps...)
	}
	return RoleDigest{
		Role:                "owner",
		Repository:          report.Repository,
		RepositoryURL:       gitlinkRepoURL(report.Repository),
		DocURL:              strings.TrimSpace(docURL),
		HealthScore:         healthScore,
		HealthRisk:          healthRisk,
		RiskLevel:           report.RiskLevel,
		ReportScore:         report.ReportScore,
		IssueTotal:          report.IssueSummary.Total,
		IssueHighRisk:       report.IssueSummary.HighRisk,
		IssueMissingInfo:    report.IssueSummary.MissingInfo,
		PRTotal:             report.PRSummary.Total,
		PRHighRisk:          report.PRSummary.HighRisk,
		ReviewFocus:         report.PRSummary.ReviewFocus,
		Recommendations:     report.Recommendations,
		AttentionItems:      uniqueDigestStrings(attention),
		NextSteps:           limitStrings(uniqueDigestStrings(nextSteps), 8),
		BoundaryDescription: "Owner digest is a read-only summary. It does not modify GitLink or Feishu resources.",
	}
}

func BuildContributorDigest(report workflow.RepoReportResult, docURL string) RoleDigest {
	healthScore, healthRisk := digestHealth(report)
	attention := []string{}
	if len(report.PRSummary.ReviewFocus) > 0 {
		attention = append(attention, report.PRSummary.ReviewFocus...)
	}
	if report.PRSummary.HighRisk > 0 {
		attention = append(attention, fmt.Sprintf("%d high-risk pull requests may need contributor updates", report.PRSummary.HighRisk))
	}
	if report.IssueSummary.MissingInfo > 0 {
		attention = append(attention, fmt.Sprintf("%d issues need clearer reproduction details or missing information", report.IssueSummary.MissingInfo))
	}
	if report.IssueSummary.HighRisk > 0 {
		attention = append(attention, fmt.Sprintf("%d high-risk issues may need focused follow-up", report.IssueSummary.HighRisk))
	}
	if len(attention) == 0 {
		attention = append(attention, "No contributor-specific blocker was detected in the workflow report.")
	}
	nextSteps := []string{
		"Check PR review focus and update the related branch or description.",
		"Add missing reproduction steps, logs, or screenshots when requested.",
		"Open the GitLink repository or Feishu report link for details.",
	}
	if report.PRSummary.HighRisk > 0 {
		nextSteps = append([]string{"Prioritize high-risk pull request feedback before new work."}, nextSteps...)
	}
	return RoleDigest{
		Role:                "contributor",
		Repository:          report.Repository,
		RepositoryURL:       gitlinkRepoURL(report.Repository),
		DocURL:              strings.TrimSpace(docURL),
		HealthScore:         healthScore,
		HealthRisk:          healthRisk,
		RiskLevel:           report.RiskLevel,
		ReportScore:         report.ReportScore,
		IssueTotal:          report.IssueSummary.Total,
		IssueHighRisk:       report.IssueSummary.HighRisk,
		IssueMissingInfo:    report.IssueSummary.MissingInfo,
		PRTotal:             report.PRSummary.Total,
		PRHighRisk:          report.PRSummary.HighRisk,
		ReviewFocus:         report.PRSummary.ReviewFocus,
		Recommendations:     report.Recommendations,
		AttentionItems:      limitStrings(uniqueDigestStrings(attention), 8),
		NextSteps:           limitStrings(uniqueDigestStrings(nextSteps), 8),
		BoundaryDescription: "Contributor digest is role-oriented, not personalized. It does not use Feishu open_id or union_id routing.",
	}
}

func BuildOwnerDigestCard(digest RoleDigest, title string, _ string) Card {
	return buildDigestCard(digest, firstNonEmpty(title, "GitLink owner digest: "+digest.Repository), "owner")
}

func BuildContributorDigestCard(digest RoleDigest, title string, _ string) Card {
	return buildDigestCard(digest, firstNonEmpty(title, "GitLink contributor digest: "+digest.Repository), "contributor")
}

func buildDigestCard(digest RoleDigest, title string, role string) Card {
	elements := []interface{}{
		div(fmt.Sprintf("**Repository**\n%s", escapeMD(digest.Repository))),
		fields([]fieldValue{
			{Label: "Report score", Value: fmt.Sprintf("%d", digest.ReportScore)},
			{Label: "Risk level", Value: digest.RiskLevel},
			{Label: "Issues", Value: fmt.Sprintf("%d", digest.IssueTotal)},
			{Label: "Pull requests", Value: fmt.Sprintf("%d", digest.PRTotal)},
		}),
		fields([]fieldValue{
			{Label: "High-risk issues", Value: fmt.Sprintf("%d", digest.IssueHighRisk)},
			{Label: "Missing-info issues", Value: fmt.Sprintf("%d", digest.IssueMissingInfo)},
			{Label: "High-risk PRs", Value: fmt.Sprintf("%d", digest.PRHighRisk)},
			{Label: "Review focus", Value: fmt.Sprintf("%d", len(digest.ReviewFocus))},
		}),
	}
	if digest.HealthScore != nil {
		elements = append(elements, fields([]fieldValue{
			{Label: "Health score", Value: fmt.Sprintf("%d", *digest.HealthScore)},
			{Label: "Health risk", Value: digest.HealthRisk},
		}))
	}
	if len(digest.AttentionItems) > 0 {
		elements = append(elements, div("**Attention**\n"+bulletList(digest.AttentionItems, 5)))
	}
	if len(digest.NextSteps) > 0 {
		elements = append(elements, div("**Suggested next steps**\n"+bulletList(digest.NextSteps, 5)))
	}
	if digest.RepositoryURL != "" {
		elements = append(elements, actionButton("Open GitLink repository", digest.RepositoryURL))
	}
	if digest.DocURL != "" {
		elements = append(elements, actionButton("Open Feishu report", digest.DocURL))
	}
	elements = append(elements, note(digest.BoundaryDescription))
	template := templateForRisk(digest.RiskLevel)
	if role == "contributor" && digest.PRSummaryNeedsAttention() {
		template = "yellow"
	}
	return baseCard(title, template, elements)
}

func (d RoleDigest) PRSummaryNeedsAttention() bool {
	return d.PRHighRisk > 0 || len(d.ReviewFocus) > 0
}

func renderDigest(w io.Writer, digest RoleDigest, format string) error {
	switch normalizeFormat(format) {
	case "markdown":
		return writeDigestMarkdown(w, digest)
	case "table":
		return writeDigestTable(w, digest)
	default:
		return writeJSON(w, digest)
	}
}

func writeDigestMarkdown(w io.Writer, digest RoleDigest) error {
	if _, err := fmt.Fprintf(w, "# GitLink %s digest: %s\n\n", digest.Role, digest.Repository); err != nil {
		return err
	}
	lines := []string{
		fmt.Sprintf("- Report score: `%d`", digest.ReportScore),
		fmt.Sprintf("- Risk level: `%s`", firstNonEmpty(digest.RiskLevel, "unknown")),
		fmt.Sprintf("- Issues: `%d` total, `%d` high risk, `%d` missing info", digest.IssueTotal, digest.IssueHighRisk, digest.IssueMissingInfo),
		fmt.Sprintf("- Pull requests: `%d` total, `%d` high risk", digest.PRTotal, digest.PRHighRisk),
	}
	if digest.HealthScore != nil {
		lines = append(lines, fmt.Sprintf("- Health score: `%d`; health risk: `%s`", *digest.HealthScore, firstNonEmpty(digest.HealthRisk, "unknown")))
	}
	if digest.RepositoryURL != "" {
		lines = append(lines, "- GitLink repository: "+digest.RepositoryURL)
	}
	if digest.DocURL != "" {
		lines = append(lines, "- Feishu report: "+digest.DocURL)
	}
	if _, err := fmt.Fprintln(w, strings.Join(lines, "\n")); err != nil {
		return err
	}
	if len(digest.AttentionItems) > 0 {
		if _, err := fmt.Fprint(w, "\n## Attention\n\n"+bulletList(digest.AttentionItems, 8)+"\n"); err != nil {
			return err
		}
	}
	if len(digest.NextSteps) > 0 {
		if _, err := fmt.Fprint(w, "\n## Suggested next steps\n\n"+bulletList(digest.NextSteps, 8)+"\n"); err != nil {
			return err
		}
	}
	_, err := fmt.Fprintf(w, "\n> %s\n", digest.BoundaryDescription)
	return err
}

func writeDigestTable(w io.Writer, digest RoleDigest) error {
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	if _, err := fmt.Fprintln(tw, "ROLE\tREPOSITORY\tRISK\tSCORE\tISSUES\tHIGH_RISK_ISSUES\tPRS\tHIGH_RISK_PRS\tATTENTION"); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(tw, "%s\t%s\t%s\t%d\t%d\t%d\t%d\t%d\t%d\n",
		digest.Role,
		digest.Repository,
		digest.RiskLevel,
		digest.ReportScore,
		digest.IssueTotal,
		digest.IssueHighRisk,
		digest.PRTotal,
		digest.PRHighRisk,
		len(digest.AttentionItems),
	); err != nil {
		return err
	}
	return tw.Flush()
}

func digestHealth(report workflow.RepoReportResult) (*int, string) {
	if report.Health == nil {
		return nil, ""
	}
	score := report.Health.HealthScore
	return &score, report.Health.RiskLevel
}

func gitlinkRepoURL(repository string) string {
	repository = strings.Trim(strings.TrimSpace(repository), "/")
	if repository == "" || repository == "local" {
		return ""
	}
	if strings.Contains(repository, "://") {
		return repository
	}
	if !strings.Contains(repository, "/") {
		return ""
	}
	return "https://www.gitlink.org.cn/" + repository
}

func limitStrings(values []string, limit int) []string {
	if limit <= 0 || len(values) <= limit {
		return values
	}
	return values[:limit]
}

func uniqueDigestStrings(values []string) []string {
	seen := map[string]bool{}
	result := []string{}
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		result = append(result, value)
	}
	return result
}
