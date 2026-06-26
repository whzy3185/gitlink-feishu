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

func BuildOwnerDigestCard(digest RoleDigest, title string, lang string) Card {
	return buildDigestCard(digest, firstNonEmpty(title, fmt.Sprintf(feishuLabel(lang, "owner_digest_title"), digest.Repository)), "owner", lang)
}

func BuildContributorDigestCard(digest RoleDigest, title string, lang string) Card {
	return buildDigestCard(digest, firstNonEmpty(title, fmt.Sprintf(feishuLabel(lang, "contributor_digest_title"), digest.Repository)), "contributor", lang)
}

func buildDigestCard(digest RoleDigest, title string, role string, lang string) Card {
	elements := []interface{}{
		div(fmt.Sprintf("**%s**\n%s", feishuLabel(lang, "repository"), escapeMD(digest.Repository))),
		fields([]fieldValue{
			{Label: feishuLabel(lang, "report_score"), Value: fmt.Sprintf("%d", digest.ReportScore)},
			{Label: feishuLabel(lang, "risk_level"), Value: digest.RiskLevel},
			{Label: feishuLabel(lang, "issues"), Value: fmt.Sprintf("%d", digest.IssueTotal)},
			{Label: feishuLabel(lang, "pull_requests"), Value: fmt.Sprintf("%d", digest.PRTotal)},
		}),
		fields([]fieldValue{
			{Label: feishuLabel(lang, "high_risk_issues"), Value: fmt.Sprintf("%d", digest.IssueHighRisk)},
			{Label: feishuLabel(lang, "missing_info_issues"), Value: fmt.Sprintf("%d", digest.IssueMissingInfo)},
			{Label: feishuLabel(lang, "high_risk_prs"), Value: fmt.Sprintf("%d", digest.PRHighRisk)},
			{Label: feishuLabel(lang, "review_focus"), Value: fmt.Sprintf("%d", len(digest.ReviewFocus))},
		}),
	}
	if digest.HealthScore != nil {
		elements = append(elements, fields([]fieldValue{
			{Label: feishuLabel(lang, "health_score"), Value: fmt.Sprintf("%d", *digest.HealthScore)},
			{Label: feishuLabel(lang, "health_risk"), Value: digest.HealthRisk},
		}))
	}
	if len(digest.AttentionItems) > 0 {
		elements = append(elements, div(fmt.Sprintf("**%s**\n%s", feishuLabel(lang, "attention"), bulletList(localizeFeishuLines(digest.AttentionItems, lang), 5))))
	}
	if len(digest.NextSteps) > 0 {
		elements = append(elements, div(fmt.Sprintf("**%s**\n%s", feishuLabel(lang, "suggested_next_steps"), bulletList(localizeFeishuLines(digest.NextSteps, lang), 5))))
	}
	if digest.RepositoryURL != "" {
		elements = append(elements, actionButton(feishuLabel(lang, "open_gitlink_repository"), digest.RepositoryURL))
	}
	if digest.DocURL != "" {
		elements = append(elements, actionButton(feishuLabel(lang, "open_feishu_report"), digest.DocURL))
	}
	elements = append(elements, note(localizedBoundary(digest, lang)))
	template := templateForRisk(digest.RiskLevel)
	if role == "contributor" && digest.PRSummaryNeedsAttention() {
		template = "yellow"
	}
	return baseCard(title, template, elements)
}

func localizedBoundary(digest RoleDigest, lang string) string {
	if !isChineseLang(lang) {
		return digest.BoundaryDescription
	}
	switch digest.Role {
	case "owner":
		return feishuLabel(lang, "boundary_owner")
	case "contributor":
		return feishuLabel(lang, "boundary_contributor")
	default:
		return localizeFeishuText(digest.BoundaryDescription, lang)
	}
}

func (d RoleDigest) PRSummaryNeedsAttention() bool {
	return d.PRHighRisk > 0 || len(d.ReviewFocus) > 0
}

func renderDigest(w io.Writer, digest RoleDigest, format string, lang string) error {
	switch normalizeFormat(format) {
	case "markdown":
		return writeDigestMarkdown(w, digest, lang)
	case "table":
		return writeDigestTable(w, digest, lang)
	default:
		return writeJSON(w, digest)
	}
}

func writeDigestMarkdown(w io.Writer, digest RoleDigest, lang string) error {
	title := fmt.Sprintf("# GitLink %s digest: %s\n\n", digest.Role, digest.Repository)
	if isChineseLang(lang) {
		role := "角色"
		if digest.Role == "owner" {
			role = "Owner"
		}
		if digest.Role == "contributor" {
			role = "贡献者"
		}
		title = fmt.Sprintf("# GitLink %s摘要：%s\n\n", role, digest.Repository)
	}
	if _, err := fmt.Fprint(w, title); err != nil {
		return err
	}
	lines := digestMarkdownLines(digest, lang)
	if digest.HealthScore != nil {
		if isChineseLang(lang) {
			lines = append(lines, fmt.Sprintf("- 健康分：`%d`；健康风险：`%s`", *digest.HealthScore, firstNonEmpty(digest.HealthRisk, "unknown")))
		} else {
			lines = append(lines, fmt.Sprintf("- Health score: `%d`; health risk: `%s`", *digest.HealthScore, firstNonEmpty(digest.HealthRisk, "unknown")))
		}
	}
	if digest.RepositoryURL != "" {
		if isChineseLang(lang) {
			lines = append(lines, "- GitLink 仓库："+digest.RepositoryURL)
		} else {
			lines = append(lines, "- GitLink repository: "+digest.RepositoryURL)
		}
	}
	if digest.DocURL != "" {
		if isChineseLang(lang) {
			lines = append(lines, "- 飞书报告："+digest.DocURL)
		} else {
			lines = append(lines, "- Feishu report: "+digest.DocURL)
		}
	}
	if _, err := fmt.Fprintln(w, strings.Join(lines, "\n")); err != nil {
		return err
	}
	if len(digest.AttentionItems) > 0 {
		heading := "Attention"
		if isChineseLang(lang) {
			heading = "需要关注"
		}
		if _, err := fmt.Fprintf(w, "\n## %s\n\n%s\n", heading, bulletList(localizeFeishuLines(digest.AttentionItems, lang), 8)); err != nil {
			return err
		}
	}
	if len(digest.NextSteps) > 0 {
		heading := "Suggested next steps"
		if isChineseLang(lang) {
			heading = "建议下一步"
		}
		if _, err := fmt.Fprintf(w, "\n## %s\n\n%s\n", heading, bulletList(localizeFeishuLines(digest.NextSteps, lang), 8)); err != nil {
			return err
		}
	}
	_, err := fmt.Fprintf(w, "\n> %s\n", localizedBoundary(digest, lang))
	return err
}

func digestMarkdownLines(digest RoleDigest, lang string) []string {
	if isChineseLang(lang) {
		return []string{
			fmt.Sprintf("- 报告分数：`%d`", digest.ReportScore),
			fmt.Sprintf("- 风险等级：`%s`", firstNonEmpty(digest.RiskLevel, "unknown")),
			fmt.Sprintf("- Issue：总数 `%d`，高风险 `%d`，信息缺失 `%d`", digest.IssueTotal, digest.IssueHighRisk, digest.IssueMissingInfo),
			fmt.Sprintf("- PR：总数 `%d`，高风险 `%d`", digest.PRTotal, digest.PRHighRisk),
		}
	}
	return []string{
		fmt.Sprintf("- Report score: `%d`", digest.ReportScore),
		fmt.Sprintf("- Risk level: `%s`", firstNonEmpty(digest.RiskLevel, "unknown")),
		fmt.Sprintf("- Issues: `%d` total, `%d` high risk, `%d` missing info", digest.IssueTotal, digest.IssueHighRisk, digest.IssueMissingInfo),
		fmt.Sprintf("- Pull requests: `%d` total, `%d` high risk", digest.PRTotal, digest.PRHighRisk),
	}
}

func writeDigestTable(w io.Writer, digest RoleDigest, lang string) error {
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	header := "ROLE\tREPOSITORY\tRISK\tSCORE\tISSUES\tHIGH_RISK_ISSUES\tPRS\tHIGH_RISK_PRS\tATTENTION"
	if isChineseLang(lang) {
		header = "角色\t仓库\t风险\t分数\tIssue\t高风险Issue\tPR\t高风险PR\t关注项"
	}
	if _, err := fmt.Fprintln(tw, header); err != nil {
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
