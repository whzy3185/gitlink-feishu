package workflow

import (
	"encoding/json"
	"strings"
	"testing"
)

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
