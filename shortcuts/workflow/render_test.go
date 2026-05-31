package workflow

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func TestWriteTriageTable(t *testing.T) {
	report := TriageReport{
		Repository: "owner/repo",
		Results: []TriageResult{
			{
				Issue:              IssueRef{Number: 7, Title: "bug"},
				DetectedType:       IssueTypeBug,
				Priority:           PriorityP1,
				Confidence:         85,
				RecommendedAction:  ActionScheduleFix,
				MissingInformation: []string{"steps to reproduce"},
			},
		},
	}
	var buf bytes.Buffer
	if err := writeTriageTable(&buf, report); err != nil {
		t.Fatalf("writeTriageTable error: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "NUMBER") {
		t.Fatalf("missing header: %s", out)
	}
	if !strings.Contains(out, "7") {
		t.Fatalf("missing issue number: %s", out)
	}
	if !strings.Contains(out, "steps to reproduce") {
		t.Fatalf("missing missing info: %s", out)
	}
}

func TestWriteTriageTableNoMissing(t *testing.T) {
	report := TriageReport{
		Repository: "owner/repo",
		Results: []TriageResult{
			{
				Issue:              IssueRef{Number: 1, Title: "feat"},
				DetectedType:       IssueTypeFeature,
				Priority:           PriorityP2,
				Confidence:         60,
				RecommendedAction:  ActionScheduleFix,
				MissingInformation: nil,
			},
		},
	}
	var buf bytes.Buffer
	if err := writeTriageTable(&buf, report); err != nil {
		t.Fatalf("writeTriageTable error: %v", err)
	}
	if !strings.Contains(buf.String(), "-") {
		t.Fatalf("expected - for no missing info: %s", buf.String())
	}
}

func TestWriteTriageMarkdown(t *testing.T) {
	report := TriageReport{
		Repository: "owner/repo",
		Results: []TriageResult{
			{
				Issue:              IssueRef{Number: 7, Title: "security bug"},
				DetectedType:       IssueTypeSecurity,
				Priority:           PriorityP0,
				Confidence:         95,
				RecommendedAction:  ActionReviewSecurity,
				MissingInformation: []string{"impact"},
			},
		},
	}
	var buf bytes.Buffer
	if err := writeTriageMarkdown(&buf, report); err != nil {
		t.Fatalf("writeTriageMarkdown error: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "owner/repo") {
		t.Fatalf("missing repo: %s", out)
	}
	if !strings.Contains(out, "#7") {
		t.Fatalf("missing issue ref: %s", out)
	}
	if !strings.Contains(out, "security") {
		t.Fatalf("missing type: %s", out)
	}
}

func TestWriteTriageMarkdownEmpty(t *testing.T) {
	report := TriageReport{Repository: "owner/repo"}
	var buf bytes.Buffer
	if err := writeTriageMarkdown(&buf, report); err != nil {
		t.Fatalf("writeTriageMarkdown error: %v", err)
	}
	if !strings.Contains(buf.String(), "owner/repo") {
		t.Fatalf("missing repo in empty report: %s", buf.String())
	}
}

func TestWriteHealthTable(t *testing.T) {
	result := HealthResult{
		Repository:  "owner/repo",
		HealthScore: 85,
		RiskLevel:   "low",
		Metrics: []HealthMetric{
			{Name: "issues", Status: "good", Score: 20, MaxScore: 20, Reason: "few open issues"},
			{Name: "activity", Status: "warning", Score: 10, MaxScore: 20, Reason: "last activity 7 days ago"},
		},
	}
	var buf bytes.Buffer
	if err := writeHealthTable(&buf, result); err != nil {
		t.Fatalf("writeHealthTable error: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "owner/repo") {
		t.Fatalf("missing repo: %s", out)
	}
	if !strings.Contains(out, "85") {
		t.Fatalf("missing score: %s", out)
	}
	if !strings.Contains(out, "issues") {
		t.Fatalf("missing metric: %s", out)
	}
	if !strings.Contains(out, "few open issues") {
		t.Fatalf("missing reason: %s", out)
	}
}

func TestRenderTriageReportTable(t *testing.T) {
	report := TriageReport{
		Repository: "owner/repo",
		Results: []TriageResult{
			{
				Issue:             IssueRef{Number: 1, Title: "test"},
				DetectedType:      IssueTypeBug,
				Priority:          PriorityP1,
				Confidence:        80,
				RecommendedAction: ActionScheduleFix,
			},
		},
	}
	var buf bytes.Buffer
	if err := renderTriageReport(&buf, report, "table"); err != nil {
		t.Fatalf("renderTriageReport table error: %v", err)
	}
	if !strings.Contains(buf.String(), "NUMBER") {
		t.Fatalf("missing table header: %s", buf.String())
	}
}

func TestRenderTriageReportMarkdown(t *testing.T) {
	report := TriageReport{
		Repository: "owner/repo",
		Results:    []TriageResult{},
	}
	var buf bytes.Buffer
	if err := renderTriageReport(&buf, report, "markdown"); err != nil {
		t.Fatalf("renderTriageReport markdown error: %v", err)
	}
	if !strings.Contains(buf.String(), "owner/repo") {
		t.Fatalf("missing repo: %s", buf.String())
	}
}

func TestRenderTriageReportJSON(t *testing.T) {
	report := TriageReport{
		Repository: "owner/repo",
		Results:    []TriageResult{},
	}
	var buf bytes.Buffer
	if err := renderTriageReport(&buf, report, "json"); err != nil {
		t.Fatalf("renderTriageReport json error: %v", err)
	}
	var parsed TriageReport
	if err := json.Unmarshal(buf.Bytes(), &parsed); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	if parsed.Repository != "owner/repo" {
		t.Fatalf("Repository = %q", parsed.Repository)
	}
}

func TestRenderTriageReportBadFormat(t *testing.T) {
	report := TriageReport{Repository: "owner/repo"}
	var buf bytes.Buffer
	err := renderTriageReport(&buf, report, "xml")
	if err == nil {
		t.Fatal("expected error for bad format")
	}
}

func TestRenderHealthResultTable(t *testing.T) {
	result := ScoreHealth(HealthInput{
		Repository:          "owner/repo",
		RecentActivityKnown: true,
		RecentActivityDays:  3,
		ReleaseKnown:        true,
		HasRecentRelease:    true,
		HasReadme:           true,
		HasLicense:          true,
		HasContributing:     true,
		AgentReadinessKnown: true,
		AgentReadinessScore: 9,
	}, "en")

	var buf bytes.Buffer
	if err := renderHealthResult(&buf, result, "table"); err != nil {
		t.Fatalf("renderHealthResult table error: %v", err)
	}
	if !strings.Contains(buf.String(), "REPOSITORY") {
		t.Fatalf("missing table header: %s", buf.String())
	}
}

func TestRenderHealthResultBadFormat(t *testing.T) {
	var buf bytes.Buffer
	err := renderHealthResult(&buf, HealthResult{}, "xml")
	if err == nil {
		t.Fatal("expected error for bad format")
	}
}

func TestNormalizeFormat(t *testing.T) {
	tests := []struct {
		input, want string
	}{
		{"", "json"},
		{"  ", "json"},
		{"JSON", "json"},
		{"yaml", "yaml"},
		{"  TABLE  ", "table"},
	}
	for _, tt := range tests {
		got := normalizeFormat(tt.input)
		if got != tt.want {
			t.Errorf("normalizeFormat(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestTruncateTableText(t *testing.T) {
	tests := []struct {
		value string
		max   int
		want  string
	}{
		{"", 10, ""},
		{"short", 10, "short"},
		{"hello world", 5, "he..."},
		{"hello world", 3, "hel"},
		{"hello world", 0, "hello world"},
		{"hello world", -1, "hello world"},
		{"hello   world", 20, "hello world"},
		{"a very long string that needs truncation", 15, "a very long ..."},
	}
	for _, tt := range tests {
		got := truncateTableText(tt.value, tt.max)
		if got != tt.want {
			t.Errorf("truncateTableText(%q, %d) = %q, want %q", tt.value, tt.max, got, tt.want)
		}
	}
}

func TestWriteHealthMarkdownWithRecommendations(t *testing.T) {
	result := HealthResult{
		Repository:  "owner/repo",
		HealthScore: 90,
		RiskLevel:   "low",
		Metrics: []HealthMetric{
			{Name: "issues", Status: "good", Score: 20, MaxScore: 20, Reason: "few issues"},
		},
		Recommendations: []string{"Fix bugs", "Add docs"},
	}
	var buf bytes.Buffer
	if err := writeHealthMarkdown(&buf, result); err != nil {
		t.Fatalf("writeHealthMarkdown error: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "Recommendations") {
		t.Fatalf("missing recommendations: %s", out)
	}
	if !strings.Contains(out, "Fix bugs") {
		t.Fatalf("missing first recommendation: %s", out)
	}
}

func TestWriteRepoReportMarkdownNoHealth(t *testing.T) {
	result := RepoReportResult{
		Repository: "owner/repo",
		Source:     "local-json",
	}
	var buf bytes.Buffer
	if err := writeRepoReportMarkdown(&buf, result, "en"); err != nil {
		t.Fatalf("writeRepoReportMarkdown error: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "owner/repo") {
		t.Fatalf("missing repository: %s", out)
	}
}

func TestWriteRepoReportTableNoHealth(t *testing.T) {
	result := RepoReportResult{
		Repository:  "owner/repo",
		ReportScore: 50,
		RiskLevel:   "medium",
		IssueSummary: RepoIssueSummary{
			Total:    3,
			HighRisk: 1,
		},
		PRSummary: RepoPRSummary{
			Total:    2,
			HighRisk: 0,
		},
		Source: "local-json",
	}
	var buf bytes.Buffer
	if err := writeRepoReportTable(&buf, result, "en"); err != nil {
		t.Fatalf("writeRepoReportTable error: %v", err)
	}
	if !strings.Contains(buf.String(), "owner/repo") {
		t.Fatalf("missing repository: %s", buf.String())
	}
}

func TestWriteRepoReportTableWithTopRecommendation(t *testing.T) {
	result := RepoReportResult{
		Repository:  "owner/repo",
		ReportScore: 80,
		RiskLevel:   "low",
		Health:      &HealthResult{HealthScore: 90, RiskLevel: "low", Metrics: nil},
		IssueSummary: RepoIssueSummary{
			Total:      5,
			HighRisk:   2,
			ByType:     map[string]int{"bug": 3},
			ByPriority: map[string]int{"P0": 1, "P1": 2},
		},
		PRSummary: RepoPRSummary{
			Total:    3,
			HighRisk: 1,
		},
		Recommendations: []string{"Address security issues immediately"},
		Source:          "local-json",
	}
	var buf bytes.Buffer
	if err := writeRepoReportTable(&buf, result, "en"); err != nil {
		t.Fatalf("writeRepoReportTable error: %v", err)
	}
	if !strings.Contains(buf.String(), "Address") {
		t.Fatalf("missing recommendation: %s", buf.String())
	}
}

func TestWriteCountMapMarkdownEmpty(t *testing.T) {
	var buf bytes.Buffer
	if err := writeCountMapMarkdown(&buf, "Test", nil); err != nil {
		t.Fatalf("writeCountMapMarkdown error: %v", err)
	}
	if buf.Len() != 0 {
		t.Fatalf("expected empty output for nil map, got: %s", buf.String())
	}
}

func TestWriteRepoReportMarkdownListEmpty(t *testing.T) {
	var buf bytes.Buffer
	if err := writeRepoReportMarkdownList(&buf, "Empty", nil, "none"); err != nil {
		t.Fatalf("writeRepoReportMarkdownList error: %v", err)
	}
	if !strings.Contains(buf.String(), "none") {
		t.Fatalf("missing fallback: %s", buf.String())
	}
}

func TestWritePRSummaryMarkdownListEmpty(t *testing.T) {
	var buf bytes.Buffer
	if err := writePRSummaryMarkdownList(&buf, "Empty", nil, "fallback text"); err != nil {
		t.Fatalf("writePRSummaryMarkdownList error: %v", err)
	}
	if !strings.Contains(buf.String(), "fallback text") {
		t.Fatalf("missing fallback: %s", buf.String())
	}
}

func TestWriteRepoReportMarkdownChinese(t *testing.T) {
	result := AnalyzeRepoReport(sampleRepoReportInput(), "zh-CN")
	var buf bytes.Buffer
	if err := writeRepoReportMarkdown(&buf, result, "zh-CN"); err != nil {
		t.Fatalf("writeRepoReportMarkdown zh-CN error: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "仓库工作流报告") {
		t.Fatalf("missing Chinese title: %s", out)
	}
}

func TestWritePRSummaryMarkdownChinese(t *testing.T) {
	result := samplePRSummaryResult()
	var buf bytes.Buffer
	if err := writePRSummaryMarkdown(&buf, result, "zh-CN"); err != nil {
		t.Fatalf("writePRSummaryMarkdown zh-CN error: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "PR 审阅摘要") {
		t.Fatalf("missing Chinese title: %s", out)
	}
}

func TestWritePRSummaryTableChinese(t *testing.T) {
	result := samplePRSummaryResult()
	var buf bytes.Buffer
	// table format ignores language
	if err := writePRSummaryTable(&buf, result); err != nil {
		t.Fatalf("writePRSummaryTable error: %v", err)
	}
	if !strings.Contains(buf.String(), "#42") {
		t.Fatalf("missing PR number: %s", buf.String())
	}
}

func TestWriteTriageMarkdownMultipleIssues(t *testing.T) {
	report := TriageReport{
		Repository: "owner/repo",
		Results: []TriageResult{
			{
				Issue:              IssueRef{Number: 1, Title: "bug"},
				DetectedType:       IssueTypeBug,
				Priority:           PriorityP1,
				Confidence:         80,
				RecommendedAction:  ActionScheduleFix,
				MissingInformation: []string{"steps", "version"},
			},
			{
				Issue:             IssueRef{Number: 2, Title: "feat"},
				DetectedType:      IssueTypeFeature,
				Priority:          PriorityP2,
				Confidence:        60,
				RecommendedAction: ActionScheduleFix,
			},
		},
	}
	var buf bytes.Buffer
	if err := writeTriageMarkdown(&buf, report); err != nil {
		t.Fatalf("writeTriageMarkdown error: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "#1") || !strings.Contains(out, "#2") {
		t.Fatalf("missing issue refs: %s", out)
	}
	if !strings.Contains(out, "steps, version") {
		t.Fatalf("missing multiple missing info: %s", out)
	}
}

func TestWriteRepoReportMarkdownFull(t *testing.T) {
	result := AnalyzeRepoReport(sampleRepoReportInput(), "en")
	var buf bytes.Buffer
	if err := writeRepoReportMarkdown(&buf, result, "en"); err != nil {
		t.Fatalf("writeRepoReportMarkdown error: %v", err)
	}
	out := buf.String()
	for _, want := range []string{"Repository Workflow Report", "Overview", "Health Summary", "Issue Triage Summary", "PR Review Summary", "Recommendations", "Reasoning"} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing section %q: %s", want, out)
		}
	}
}

func TestWriteRepoReportMarkdownNoPRReviewFocus(t *testing.T) {
	result := AnalyzeRepoReport(sampleRepoReportInput(), "en")
	result.PRSummary.ReviewFocus = nil
	var buf bytes.Buffer
	if err := writeRepoReportMarkdown(&buf, result, "en"); err != nil {
		t.Fatalf("writeRepoReportMarkdown error: %v", err)
	}
	if !strings.Contains(buf.String(), "owner/repo") {
		t.Fatalf("missing repo: %s", buf.String())
	}
}

func TestWriteJSON(t *testing.T) {
	var buf bytes.Buffer
	if err := writeJSON(&buf, map[string]string{"key": "val"}); err != nil {
		t.Fatalf("writeJSON error: %v", err)
	}
	if !strings.Contains(buf.String(), `"key"`) {
		t.Fatalf("missing key: %s", buf.String())
	}
}

func TestRenderPRSummaryJSON(t *testing.T) {
	rendered, err := RenderPRSummary(samplePRSummaryResult(), "json", "en")
	if err != nil {
		t.Fatalf("RenderPRSummary returned error: %v", err)
	}
	var result PRSummaryResult
	if err := json.Unmarshal([]byte(rendered), &result); err != nil {
		t.Fatalf("json.Unmarshal returned error: %v\noutput=%s", err, rendered)
	}
	if result.Number != 42 {
		t.Fatalf("Number = %d, want 42", result.Number)
	}
}

func TestRenderPRSummaryMarkdown(t *testing.T) {
	result := samplePRSummaryResult()
	rendered, err := RenderPRSummary(result, "markdown", "en")
	if err != nil {
		t.Fatalf("RenderPRSummary returned error: %v", err)
	}
	for _, want := range []string{result.Title, "Risk level", "Review Focus"} {
		if !strings.Contains(rendered, want) {
			t.Fatalf("markdown output missing %q:\n%s", want, rendered)
		}
	}
}

func TestRenderPRSummaryTable(t *testing.T) {
	rendered, err := RenderPRSummary(samplePRSummaryResult(), "table", "en")
	if err != nil {
		t.Fatalf("RenderPRSummary returned error: %v", err)
	}
	if !strings.Contains(rendered, "#42") || !strings.Contains(rendered, "medium") {
		t.Fatalf("table output = %q, want PR number and risk", rendered)
	}
}

func TestRenderPRSummaryUnknownFormat(t *testing.T) {
	_, err := RenderPRSummary(samplePRSummaryResult(), "xml", "en")
	if err == nil {
		t.Fatal("RenderPRSummary returned nil error for unknown format")
	}
}

func TestRenderPRSummaryChineseMarkdown(t *testing.T) {
	rendered, err := RenderPRSummary(samplePRSummaryResult(), "markdown", "zh-CN")
	if err != nil {
		t.Fatalf("RenderPRSummary returned error: %v", err)
	}
	if !strings.Contains(rendered, "PR 审阅摘要") {
		t.Fatalf("zh-CN markdown output missing Chinese title:\n%s", rendered)
	}
}

func TestRenderPRSummaryChineseTable(t *testing.T) {
	rendered, err := RenderPRSummary(samplePRSummaryResult(), "table", "zh-CN")
	if err != nil {
		t.Fatalf("RenderPRSummary error: %v", err)
	}
	if !strings.Contains(rendered, "#42") {
		t.Fatalf("table output missing PR number: %s", rendered)
	}
}

func TestRenderPRSummaryEmptyReviewFocus(t *testing.T) {
	result := samplePRSummaryResult()
	result.ReviewFocus = nil
	result.TestSuggestions = nil
	result.MergeChecklist = nil
	result.Reasoning = nil
	rendered, err := RenderPRSummary(result, "markdown", "en")
	if err != nil {
		t.Fatalf("RenderPRSummary error: %v", err)
	}
	if !strings.Contains(rendered, "no focus areas") && !strings.Contains(rendered, "No review") {
		// Falls back to fallback text - should not be empty
		if !strings.Contains(rendered, result.Title) {
			t.Fatalf("markdown output missing title: %s", rendered)
		}
	}
}

func TestRenderHealthResultChineseMarkdown(t *testing.T) {
	result := ScoreHealth(HealthInput{
		Repository:          "owner/repo",
		RecentActivityKnown: true,
		RecentActivityDays:  3,
		HasReadme:           true,
	}, "zh-CN")

	var buf bytes.Buffer
	if err := renderHealthResult(&buf, result, "markdown"); err != nil {
		t.Fatalf("renderHealthResult error: %v", err)
	}
	if !strings.Contains(buf.String(), "Issue 积压处于可控状态") {
		t.Fatalf("missing Chinese metric text: %s", buf.String())
	}
}

func TestRenderHealthResultJSON(t *testing.T) {
	result := ScoreHealth(HealthInput{
		Repository:          "owner/repo",
		RecentActivityKnown: true,
		RecentActivityDays:  2,
	}, "en")

	var buf bytes.Buffer
	if err := renderHealthResult(&buf, result, "json"); err != nil {
		t.Fatalf("renderHealthResult error: %v", err)
	}
	var parsed HealthResult
	if err := json.Unmarshal(buf.Bytes(), &parsed); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
}

func samplePRSummaryResult() PRSummaryResult {
	return PRSummaryResult{
		Repository:        "owner/repo",
		Number:            42,
		Title:             "feat: add workflow PR summary",
		Author:            "alice",
		State:             "open",
		BaseBranch:        "master",
		HeadBranch:        "feature/pr-summary",
		ChangedFilesCount: 2,
		Additions:         100,
		Deletions:         4,
		CommitCount:       2,
		ChangeType:        PRChangeTypeFeature,
		RiskLevel:         PRRiskMedium,
		ReviewFocus:       []string{"Check shortcut command compatibility and flag behavior."},
		TestSuggestions:   []string{"Run `go test ./shortcuts/workflow`."},
		MergeChecklist:    []string{"Tests pass."},
		Reasoning:         []string{"change type: feature", "risk level: medium"},
		Source:            "local-json",
	}
}
