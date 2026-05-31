package workflow

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/gitlink-org/gitlink-cli/internal/client"
	"github.com/gitlink-org/gitlink-cli/shortcuts/common"
)

func TestAnalyzeRepoReportAggregatesHealthIssuesAndPRs(t *testing.T) {
	input := sampleRepoReportInput()
	result := AnalyzeRepoReport(input, "en")
	if result.ReportScore < 0 || result.ReportScore > 100 {
		t.Fatalf("ReportScore = %d, want 0..100", result.ReportScore)
	}
	if result.IssueSummary.Total != 3 {
		t.Fatalf("IssueSummary.Total = %d, want 3", result.IssueSummary.Total)
	}
	if result.IssueSummary.ByType[IssueTypeBug] == 0 {
		t.Fatalf("IssueSummary.ByType = %+v, want bug count", result.IssueSummary.ByType)
	}
	if result.PRSummary.Total != 3 {
		t.Fatalf("PRSummary.Total = %d, want 3", result.PRSummary.Total)
	}
	if result.PRSummary.ByRisk[PRRiskHigh] == 0 {
		t.Fatalf("PRSummary.ByRisk = %+v, want high risk count", result.PRSummary.ByRisk)
	}
	if len(result.Recommendations) == 0 {
		t.Fatal("Recommendations empty")
	}
}

func TestAnalyzeRepoReportWithSecurityIssueRaisesRisk(t *testing.T) {
	input := RepoReportInput{
		Repository: "owner/repo",
		Issues: []IssueInput{{
			Number: 1,
			Title:  "Token leaked in logs",
			Body:   "A secret token leaked from command output.",
			Labels: []string{"security"},
		}},
	}
	result := AnalyzeRepoReport(input, "en")
	if result.RiskLevel != "critical" {
		t.Fatalf("RiskLevel = %q, want critical", result.RiskLevel)
	}
}

func TestAnalyzeRepoReportPartialInput(t *testing.T) {
	input := RepoReportInput{
		Repository: "owner/repo",
		Health: &HealthInput{
			Repository:          "owner/repo",
			OpenIssues:          1,
			OpenPRs:             0,
			RecentActivityKnown: true,
			RecentActivityDays:  2,
			ReleaseKnown:        true,
			HasRecentRelease:    true,
			HasReadme:           true,
			HasLicense:          true,
			HasContributing:     true,
			AgentReadinessKnown: true,
			AgentReadinessScore: 9,
		},
	}
	result := AnalyzeRepoReport(input, "en")
	if result.Health == nil {
		t.Fatal("Health result nil")
	}
	if len(result.Recommendations) == 0 {
		t.Fatal("Recommendations empty")
	}
}

func TestAnalyzeRepoReportChinese(t *testing.T) {
	result := AnalyzeRepoReport(sampleRepoReportInput(), "zh-CN")
	if len(result.Recommendations) == 0 {
		t.Fatal("Recommendations empty")
	}
	rendered, err := RenderRepoReport(result, "markdown", "zh-CN")
	if err != nil {
		t.Fatalf("RenderRepoReport returned error: %v", err)
	}
	if !strings.Contains(rendered, "仓库工作流报告") {
		t.Fatalf("markdown output missing Chinese title:\n%s", rendered)
	}
}

func TestRenderRepoReportJSON(t *testing.T) {
	rendered, err := RenderRepoReport(AnalyzeRepoReport(sampleRepoReportInput(), "en"), "json", "en")
	if err != nil {
		t.Fatalf("RenderRepoReport returned error: %v", err)
	}
	var result RepoReportResult
	if err := json.Unmarshal([]byte(rendered), &result); err != nil {
		t.Fatalf("json.Unmarshal returned error: %v\noutput=%s", err, rendered)
	}
	if result.Repository != "owner/repo" {
		t.Fatalf("Repository = %q, want owner/repo", result.Repository)
	}
}

func TestRenderRepoReportMarkdown(t *testing.T) {
	result := AnalyzeRepoReport(sampleRepoReportInput(), "en")
	rendered, err := RenderRepoReport(result, "markdown", "en")
	if err != nil {
		t.Fatalf("RenderRepoReport returned error: %v", err)
	}
	for _, want := range []string{"Repository Workflow Report", "Report score", "Recommendations"} {
		if !strings.Contains(rendered, want) {
			t.Fatalf("markdown output missing %q:\n%s", want, rendered)
		}
	}
}

func TestRenderRepoReportTable(t *testing.T) {
	rendered, err := RenderRepoReport(AnalyzeRepoReport(sampleRepoReportInput(), "en"), "table", "en")
	if err != nil {
		t.Fatalf("RenderRepoReport returned error: %v", err)
	}
	if !strings.Contains(rendered, "REPORT_SCORE") || !strings.Contains(rendered, "owner/repo") {
		t.Fatalf("table output = %q, want report score and repository", rendered)
	}
}

func TestRenderRepoReportUnknownFormat(t *testing.T) {
	_, err := RenderRepoReport(AnalyzeRepoReport(sampleRepoReportInput(), "en"), "xml", "en")
	if err == nil {
		t.Fatal("RenderRepoReport returned nil error for unknown format")
	}
}

func TestReadRepoReportInput(t *testing.T) {
	input, err := readRepoReportInput("testdata/repo_report.json")
	if err != nil {
		t.Fatalf("readRepoReportInput returned error: %v", err)
	}
	if input.Repository == "" || len(input.Issues) == 0 || len(input.PullRequests) == 0 {
		t.Fatalf("input = %+v, want populated fixture", input)
	}
}

func TestRunRepoReportFromFile(t *testing.T) {
	ctx := &common.RuntimeContext{
		Client: &client.Client{HTTP: http.DefaultClient, BaseURL: "http://localhost"},
		Args: map[string]string{
			"from": "testdata/repo_report.json",
			"lang": "en",
		},
		Format: "json",
	}
	err := runRepoReport(ctx)
	if err != nil {
		t.Fatalf("runRepoReport error: %v", err)
	}
}

func TestRunRepoReportFromFileMarkdown(t *testing.T) {
	ctx := &common.RuntimeContext{
		Client: &client.Client{HTTP: http.DefaultClient, BaseURL: "http://localhost"},
		Args: map[string]string{
			"from": "testdata/repo_report.json",
			"lang": "en",
		},
		Format: "markdown",
	}
	err := runRepoReport(ctx)
	if err != nil {
		t.Fatalf("runRepoReport error: %v", err)
	}
}

func TestRunRepoReportFromFileChinese(t *testing.T) {
	ctx := &common.RuntimeContext{
		Client: &client.Client{HTTP: http.DefaultClient, BaseURL: "http://localhost"},
		Args: map[string]string{
			"from": "testdata/repo_report.json",
			"lang": "zh-CN",
		},
		Format: "markdown",
	}
	err := runRepoReport(ctx)
	if err != nil {
		t.Fatalf("runRepoReport error: %v", err)
	}
}

func TestRunRepoReportMissingFile(t *testing.T) {
	ctx := &common.RuntimeContext{
		Client: &client.Client{HTTP: http.DefaultClient, BaseURL: "http://localhost"},
		Args: map[string]string{
			"from": "testdata/does_not_exist.json",
		},
	}
	err := runRepoReport(ctx)
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestRunRepoReportBadJSON(t *testing.T) {
	ctx := &common.RuntimeContext{
		Client: &client.Client{HTTP: http.DefaultClient, BaseURL: "http://localhost"},
		Args: map[string]string{
			"from": "testdata/../workflow_test.go",
		},
	}
	err := runRepoReport(ctx)
	if err == nil {
		t.Fatal("expected error for non-JSON file")
	}
}

func TestCollectRepoReportInputWithFlags(t *testing.T) {
	ctx := &common.RuntimeContext{
		Client: &client.Client{HTTP: http.DefaultClient, BaseURL: "http://localhost"},
		Args: map[string]string{
			"issue-limit":    "5",
			"pr-limit":       "3",
			"stale-days":     "14",
			"include-issues": "false",
			"include-prs":    "true",
			"include-health": "false",
			"lang":           "en",
		},
		Owner: "owner",
		Repo:  "repo",
	}
	// This will try to make HTTP calls; verify it fails cleanly (no panic)
	_, _, err := collectRepoReportInput(ctx)
	if err != nil {
		t.Logf("expected network error: %v", err)
	}
}

func TestRepoReportText(t *testing.T) {
	keys := []string{
		"title", "overview", "health", "issues", "prs", "recommendations",
		"reasoning", "not_available", "health_missing", "critical_signal",
		"rec_security_issues", "rec_missing_info", "rec_high_risk_prs",
		"rec_health", "rec_maintain_report", "rec_review_report",
	}
	for _, lang := range []string{"en", "zh-CN"} {
		for _, key := range keys {
			got := repoReportText(lang, key)
			if got == "" {
				t.Fatalf("repoReportText(%q, %q) returned empty", lang, key)
			}
			if got == key {
				t.Logf("repoReportText(%q, %q) returned key itself: %q", lang, key, got)
			}
		}
	}
	// Default case: unknown key returns the key itself
	if got := repoReportText("en", "nonexistent_key"); got != "nonexistent_key" {
		t.Fatalf("default case: got %q, want nonexistent_key", got)
	}
}

func TestRepoReportTextChineseDistinct(t *testing.T) {
	// Verify Chinese text is actually different from English
	for _, key := range []string{"title", "health", "issues", "prs", "recommendations", "not_available"} {
		en := repoReportText("en", key)
		zh := repoReportText("zh-CN", key)
		if en == zh {
			t.Fatalf("repoReportText: key %q has same text for en and zh-CN: %q", key, en)
		}
	}
}

func sampleRepoReportInput() RepoReportInput {
	return RepoReportInput{
		Repository: "owner/repo",
		Health: &HealthInput{
			Repository:          "owner/repo",
			OpenIssues:          5,
			OpenPRs:             3,
			StaleIssues:         1,
			StalePRs:            1,
			RecentActivityKnown: true,
			RecentActivityDays:  2,
			ReleaseKnown:        true,
			HasRecentRelease:    true,
			HasReadme:           true,
			HasLicense:          true,
			HasContributing:     true,
			AgentReadinessKnown: true,
			AgentReadinessScore: 9,
		},
		Issues: []IssueInput{
			{Number: 1, Title: "Install failed", Body: "error on install", Labels: []string{"bug"}},
			{Number: 2, Title: "README typo", Body: "docs typo", Labels: []string{"docs"}},
			{Number: 3, Title: "Crash on login", Body: "panic", Labels: []string{"bug"}},
		},
		PullRequests: []PRSummaryInput{
			{
				Repository: "owner/repo",
				Number:     1,
				Title:      "docs: update guide",
				ChangedFiles: []PRChangedFile{
					{Filename: "README.md", Additions: 10, Deletions: 1, Changes: 11},
				},
			},
			{
				Repository: "owner/repo",
				Number:     2,
				Title:      "feat: add workflow command",
				ChangedFiles: []PRChangedFile{
					{Filename: "shortcuts/workflow/workflow.go", Additions: 40, Deletions: 4, Changes: 44},
				},
			},
			{
				Repository: "owner/repo",
				Number:     3,
				Title:      "fix: normalize API client errors",
				ChangedFiles: []PRChangedFile{
					{Filename: "internal/client/client.go", Additions: 30, Deletions: 10, Changes: 40},
				},
			},
		},
		Source: "local-json",
	}
}
