package workflow

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"
	"text/tabwriter"
)

func renderTriageReport(w io.Writer, report TriageReport, format string) error {
	switch normalizeFormat(format) {
	case "json":
		return writeJSON(w, report)
	case "markdown":
		return writeTriageMarkdown(w, report)
	case "table":
		return writeTriageTable(w, report)
	default:
		return fmt.Errorf("unsupported workflow output format %q", format)
	}
}

func renderHealthResult(w io.Writer, result HealthResult, format string) error {
	switch normalizeFormat(format) {
	case "json":
		return writeJSON(w, result)
	case "markdown":
		return writeHealthMarkdown(w, result)
	case "table":
		return writeHealthTable(w, result)
	default:
		return fmt.Errorf("unsupported workflow output format %q", format)
	}
}

func RenderPRSummary(result PRSummaryResult, format string, lang string) (string, error) {
	var buf bytes.Buffer
	switch normalizeFormat(format) {
	case "json":
		if err := writeJSON(&buf, result); err != nil {
			return "", err
		}
	case "markdown":
		if err := writePRSummaryMarkdown(&buf, result, lang); err != nil {
			return "", err
		}
	case "table":
		if err := writePRSummaryTable(&buf, result); err != nil {
			return "", err
		}
	default:
		return "", fmt.Errorf("unsupported workflow output format %q", format)
	}
	return buf.String(), nil
}

func RenderRepoReport(result RepoReportResult, format string, lang string) (string, error) {
	var buf bytes.Buffer
	switch normalizeFormat(format) {
	case "json":
		if err := writeJSON(&buf, result); err != nil {
			return "", err
		}
	case "markdown":
		if err := writeRepoReportMarkdown(&buf, result, lang); err != nil {
			return "", err
		}
	case "table":
		if err := writeRepoReportTable(&buf, result, lang); err != nil {
			return "", err
		}
	default:
		return "", fmt.Errorf("unsupported workflow output format %q", format)
	}
	return buf.String(), nil
}

func normalizeFormat(format string) string {
	format = strings.ToLower(strings.TrimSpace(format))
	if format == "" {
		return "json"
	}
	return format
}

func writeJSON(w io.Writer, data interface{}) error {
	encoded, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(w, string(encoded))
	return err
}

func writeTriageTable(w io.Writer, report TriageReport) error {
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	if _, err := fmt.Fprintln(tw, "NUMBER\tTYPE\tPRIORITY\tCONFIDENCE\tMISSING\tACTION"); err != nil {
		return err
	}
	for _, result := range report.Results {
		missing := "-"
		if len(result.MissingInformation) > 0 {
			missing = strings.Join(result.MissingInformation, ",")
		}
		if _, err := fmt.Fprintf(tw, "%d\t%s\t%s\t%d\t%s\t%s\n",
			result.Issue.Number,
			result.DetectedType,
			result.Priority,
			result.Confidence,
			missing,
			result.RecommendedAction,
		); err != nil {
			return err
		}
	}
	return tw.Flush()
}

func writePRSummaryTable(w io.Writer, result PRSummaryResult) error {
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	if _, err := fmt.Fprintln(tw, "PR\tTITLE\tTYPE\tRISK\tFILES\tCOMMITS\tSOURCE"); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(tw, "#%d\t%s\t%s\t%s\t%d\t%d\t%s\n",
		result.Number,
		truncateTableText(result.Title, 72),
		result.ChangeType,
		result.RiskLevel,
		result.ChangedFilesCount,
		result.CommitCount,
		result.Source,
	); err != nil {
		return err
	}
	return tw.Flush()
}

func writeHealthTable(w io.Writer, result HealthResult) error {
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	if _, err := fmt.Fprintf(tw, "REPOSITORY\tSCORE\tRISK\n%s\t%d\t%s\n\n", result.Repository, result.HealthScore, result.RiskLevel); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(tw, "METRIC\tSTATUS\tSCORE\tMAX\tREASON"); err != nil {
		return err
	}
	for _, metric := range result.Metrics {
		if _, err := fmt.Fprintf(tw, "%s\t%s\t%d\t%d\t%s\n", metric.Name, metric.Status, metric.Score, metric.MaxScore, metric.Reason); err != nil {
			return err
		}
	}
	return tw.Flush()
}

func writeRepoReportTable(w io.Writer, result RepoReportResult, lang string) error {
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	healthScore := repoReportText(lang, "not_available")
	if result.Health != nil {
		healthScore = fmt.Sprintf("%d", result.Health.HealthScore)
	}
	topRecommendation := repoReportText(lang, "not_available")
	if len(result.Recommendations) > 0 {
		topRecommendation = truncateTableText(result.Recommendations[0], 96)
	}
	openPRs, mergedPRs, closedPRs := "N/A", "N/A", "N/A"
	if result.PRLifecycle != nil {
		openPRs = fmt.Sprintf("%d", result.PRLifecycle.Open)
		mergedPRs = fmt.Sprintf("%d", result.PRLifecycle.Merged)
		closedPRs = fmt.Sprintf("%d", result.PRLifecycle.ClosedOrRejected)
	}
	reviewedPRs, unreviewedPRs, needsReReviewPRs := "N/A", "N/A", "N/A"
	if result.PRReviewAudit != nil {
		reviewedPRs = fmt.Sprintf("%d", result.PRReviewAudit.Reviewed)
		unreviewedPRs = fmt.Sprintf("%d", result.PRReviewAudit.Unreviewed)
		needsReReviewPRs = fmt.Sprintf("%d", result.PRReviewAudit.NeedsReReview)
	}
	if _, err := fmt.Fprintln(tw, "REPOSITORY\tREPORT_SCORE\tRISK\tHEALTH_SCORE\tISSUES_ANALYZED\tHIGH_RISK_ISSUES\tPRS_ANALYZED\tHIGH_RISK_PRS\tOPEN_PRS\tMERGED_PRS\tCLOSED_PRS\tREVIEWED_PRS\tUNREVIEWED_PRS\tNEEDS_RE_REVIEW\tTOP_RECOMMENDATION"); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(tw, "%s\t%d\t%s\t%s\t%d\t%d\t%d\t%d\t%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
		result.Repository,
		result.ReportScore,
		result.RiskLevel,
		healthScore,
		result.IssueSummary.Total,
		result.IssueSummary.HighRisk,
		result.PRSummary.Total,
		result.PRSummary.HighRisk,
		openPRs,
		mergedPRs,
		closedPRs,
		reviewedPRs,
		unreviewedPRs,
		needsReReviewPRs,
		topRecommendation,
	); err != nil {
		return err
	}
	return tw.Flush()
}

func writeTriageMarkdown(w io.Writer, report TriageReport) error {
	if _, err := fmt.Fprintf(w, "# Issue Triage Report\n\nRepository: `%s`\n\n", report.Repository); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(w, "| Issue | Type | Priority | Confidence | Action | Missing Information |"); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(w, "| --- | --- | --- | ---: | --- | --- |"); err != nil {
		return err
	}
	for _, result := range report.Results {
		missing := "-"
		if len(result.MissingInformation) > 0 {
			missing = strings.Join(result.MissingInformation, ", ")
		}
		if _, err := fmt.Fprintf(w, "| #%d | %s | %s | %d | %s | %s |\n",
			result.Issue.Number,
			result.DetectedType,
			result.Priority,
			result.Confidence,
			result.RecommendedAction,
			missing,
		); err != nil {
			return err
		}
	}
	return nil
}

func writeRepoReportMarkdown(w io.Writer, result RepoReportResult, lang string) error {
	lang = normalizeLang(lang)
	if _, err := fmt.Fprintf(w, "# %s\n\n", repoReportText(lang, "title")); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "## %s\n\n", repoReportText(lang, "overview")); err != nil {
		return err
	}
	healthScore := repoReportText(lang, "not_available")
	if result.Health != nil {
		healthScore = fmt.Sprintf("%d", result.Health.HealthScore)
	}
	overview := []string{
		fmt.Sprintf("- Repository: `%s`", result.Repository),
		fmt.Sprintf("- Report score: `%d`", result.ReportScore),
		fmt.Sprintf("- Risk level: `%s`", result.RiskLevel),
		fmt.Sprintf("- Health score: `%s`", healthScore),
		fmt.Sprintf("- Issues analyzed: `%d`", result.IssueSummary.Total),
		fmt.Sprintf("- Pull requests analyzed: `%d`", result.PRSummary.Total),
		fmt.Sprintf("- Source: `%s`", result.Source),
	}
	for _, line := range overview {
		if _, err := fmt.Fprintln(w, line); err != nil {
			return err
		}
	}

	if _, err := fmt.Fprintf(w, "\n## %s\n\n", repoReportText(lang, "health")); err != nil {
		return err
	}
	if result.Health == nil {
		if _, err := fmt.Fprintf(w, "- %s\n", repoReportText(lang, "not_available")); err != nil {
			return err
		}
	} else {
		if _, err := fmt.Fprintf(w, "- Score: `%d`\n- Risk: `%s`\n", result.Health.HealthScore, result.Health.RiskLevel); err != nil {
			return err
		}
	}

	if _, err := fmt.Fprintf(w, "\n## %s\n\n", repoReportText(lang, "issues")); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "- Total: `%d`\n- High risk: `%d`\n- Missing information: `%d`\n", result.IssueSummary.Total, result.IssueSummary.HighRisk, result.IssueSummary.MissingInfo); err != nil {
		return err
	}
	writeCountMapMarkdown(w, "By type", result.IssueSummary.ByType)
	writeCountMapMarkdown(w, "By priority", result.IssueSummary.ByPriority)

	if _, err := fmt.Fprintf(w, "\n## %s\n\n", repoReportText(lang, "prs")); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "- Total: `%d`\n- High risk: `%d`\n", result.PRSummary.Total, result.PRSummary.HighRisk); err != nil {
		return err
	}
	writeCountMapMarkdown(w, "By type", result.PRSummary.ByType)
	writeCountMapMarkdown(w, "By risk", result.PRSummary.ByRisk)
	writeCountMapMarkdown(w, "Risk rule sources", result.PRSummary.RiskSources)
	if result.PRLifecycle != nil {
		if _, err := fmt.Fprintf(w, "- Lifecycle totals: open `%d`, merged `%d`, closed/rejected `%d`, all states `%d`\n",
			result.PRLifecycle.Open,
			result.PRLifecycle.Merged,
			result.PRLifecycle.ClosedOrRejected,
			result.PRLifecycle.Total,
		); err != nil {
			return err
		}
	}
	if result.PRReviewAudit != nil {
		if _, err := fmt.Fprintf(w, "- Review audit: audited `%d`, reviewed `%d`, unreviewed `%d`, needs re-review `%d`, formal reviews `%d`\n",
			result.PRReviewAudit.Audited,
			result.PRReviewAudit.Reviewed,
			result.PRReviewAudit.Unreviewed,
			result.PRReviewAudit.NeedsReReview,
			result.PRReviewAudit.FormalReviews,
		); err != nil {
			return err
		}
		if _, err := fmt.Fprintf(w, "- Review actor attribution: reviewer comments `%d`, submitter comments `%d`, participant comments `%d`, system events `%d`\n",
			result.PRReviewAudit.ReviewerComments,
			result.PRReviewAudit.SubmitterComments,
			result.PRReviewAudit.ParticipantComments,
			result.PRReviewAudit.SystemEvents,
		); err != nil {
			return err
		}
	}
	if len(result.PRSummary.ReviewFocus) > 0 {
		if _, err := fmt.Fprintln(w, "- Review focus:"); err != nil {
			return err
		}
		for _, focus := range result.PRSummary.ReviewFocus {
			if _, err := fmt.Fprintf(w, "  - %s\n", focus); err != nil {
				return err
			}
		}
	}

	if err := writeRepoReportMarkdownList(w, repoReportText(lang, "recommendations"), result.Recommendations, repoReportText(lang, "not_available")); err != nil {
		return err
	}
	return writeRepoReportMarkdownList(w, repoReportText(lang, "reasoning"), result.Reasoning, repoReportText(lang, "not_available"))
}

func writeCountMapMarkdown(w io.Writer, title string, values map[string]int) error {
	if len(values) == 0 {
		return nil
	}
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	if _, err := fmt.Fprintf(w, "- %s:\n", title); err != nil {
		return err
	}
	for _, key := range keys {
		if _, err := fmt.Fprintf(w, "  - `%s`: `%d`\n", key, values[key]); err != nil {
			return err
		}
	}
	return nil
}

func writeRepoReportMarkdownList(w io.Writer, title string, values []string, fallback string) error {
	if _, err := fmt.Fprintf(w, "\n## %s\n\n", title); err != nil {
		return err
	}
	if len(values) == 0 {
		_, err := fmt.Fprintf(w, "- %s\n", fallback)
		return err
	}
	for _, value := range values {
		if _, err := fmt.Fprintf(w, "- %s\n", value); err != nil {
			return err
		}
	}
	return nil
}

func writePRSummaryMarkdown(w io.Writer, result PRSummaryResult, lang string) error {
	lang = normalizeLang(lang)
	if _, err := fmt.Fprintf(w, "# %s\n\n", message(lang, "pr_summary_title")); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "## %s\n\n", message(lang, "pr_summary_overview")); err != nil {
		return err
	}
	lines := []string{
		fmt.Sprintf("- Repository: `%s`", result.Repository),
		fmt.Sprintf("- PR: `#%d` %s", result.Number, result.Title),
		fmt.Sprintf("- Author: `%s`", result.Author),
		fmt.Sprintf("- State: `%s`", result.State),
		fmt.Sprintf("- Base branch: `%s`", result.BaseBranch),
		fmt.Sprintf("- Head branch: `%s`", result.HeadBranch),
		fmt.Sprintf("- Change type: `%s`", result.ChangeType),
		fmt.Sprintf("- Risk level: `%s`", result.RiskLevel),
		fmt.Sprintf("- Changed files: `%d`", result.ChangedFilesCount),
		fmt.Sprintf("- Commits: `%d`", result.CommitCount),
		fmt.Sprintf("- Source: `%s`", result.Source),
	}
	for _, line := range lines {
		if _, err := fmt.Fprintln(w, line); err != nil {
			return err
		}
	}

	if err := writePRSummaryMarkdownList(w, message(lang, "pr_summary_review_focus"), result.ReviewFocus, message(lang, "pr_summary_no_focus")); err != nil {
		return err
	}
	if err := writePRSummaryMarkdownList(w, message(lang, "pr_summary_test_suggestions"), result.TestSuggestions, message(lang, "pr_summary_no_suggestions")); err != nil {
		return err
	}
	if err := writePRSummaryMarkdownList(w, message(lang, "pr_summary_merge_checklist"), result.MergeChecklist, message(lang, "pr_summary_no_checklist")); err != nil {
		return err
	}
	return writePRSummaryMarkdownList(w, message(lang, "pr_summary_reasoning"), result.Reasoning, message(lang, "pr_summary_no_reasoning"))
}

func writePRSummaryMarkdownList(w io.Writer, title string, values []string, fallback string) error {
	if _, err := fmt.Fprintf(w, "\n## %s\n\n", title); err != nil {
		return err
	}
	if len(values) == 0 {
		_, err := fmt.Fprintf(w, "- %s\n", fallback)
		return err
	}
	for _, value := range values {
		if _, err := fmt.Fprintf(w, "- %s\n", value); err != nil {
			return err
		}
	}
	return nil
}

func writeHealthMarkdown(w io.Writer, result HealthResult) error {
	if _, err := fmt.Fprintf(w, "# Repository Health Report\n\nRepository: `%s`\n\nHealth score: **%d**\n\nRisk level: **%s**\n\n", result.Repository, result.HealthScore, result.RiskLevel); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(w, "| Metric | Status | Score | Max | Reason |"); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(w, "| --- | --- | ---: | ---: | --- |"); err != nil {
		return err
	}
	for _, metric := range result.Metrics {
		if _, err := fmt.Fprintf(w, "| %s | %s | %d | %d | %s |\n", metric.Name, metric.Status, metric.Score, metric.MaxScore, metric.Reason); err != nil {
			return err
		}
	}
	if len(result.Recommendations) == 0 {
		return nil
	}
	if _, err := fmt.Fprintln(w, "\n## Recommendations"); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(w); err != nil {
		return err
	}
	for _, recommendation := range result.Recommendations {
		if _, err := fmt.Fprintf(w, "- %s\n", recommendation); err != nil {
			return err
		}
	}
	return nil
}

func truncateTableText(value string, max int) string {
	value = strings.Join(strings.Fields(value), " ")
	runes := []rune(value)
	if max <= 0 || len(runes) <= max {
		return value
	}
	if max <= 3 {
		return string(runes[:max])
	}
	return string(runes[:max-3]) + "..."
}
