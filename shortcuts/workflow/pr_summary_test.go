package workflow

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gitlink-org/gitlink-cli/cmd/cmdutil"
	"github.com/gitlink-org/gitlink-cli/internal/client"
	"github.com/gitlink-org/gitlink-cli/shortcuts/common"
)

func TestAnalyzePRSummaryDocsOnlyLowRisk(t *testing.T) {
	result := AnalyzePRSummary(PRSummaryInput{
		Repository: "owner/repo",
		Number:     1,
		Title:      "docs: update README examples",
		ChangedFiles: []PRChangedFile{
			{Filename: "README.md", Status: "modified", Additions: 12},
			{Filename: "docs/workflow-agent-design.md", Status: "modified", Additions: 8},
		},
		Additions: 20,
	}, "en")

	if result.ChangeType != PRChangeTypeDocs {
		t.Fatalf("ChangeType = %q, want %q", result.ChangeType, PRChangeTypeDocs)
	}
	if result.RiskLevel != PRRiskLow {
		t.Fatalf("RiskLevel = %q, want %q", result.RiskLevel, PRRiskLow)
	}
}

func TestAnalyzePRSummaryWorkflowCodeMediumRisk(t *testing.T) {
	result := AnalyzePRSummary(PRSummaryInput{
		Repository: "owner/repo",
		Number:     2,
		Title:      "feat: add PR summary workflow",
		ChangedFiles: []PRChangedFile{
			{Filename: "shortcuts/workflow/pr_summary.go", Status: "added", Additions: 80},
		},
		Additions: 80,
	}, "en")

	if result.RiskLevel != PRRiskMedium {
		t.Fatalf("RiskLevel = %q, want %q", result.RiskLevel, PRRiskMedium)
	}
}

func TestAnalyzePRSummaryInternalClientHighRisk(t *testing.T) {
	result := AnalyzePRSummary(PRSummaryInput{
		Repository: "owner/repo",
		Number:     3,
		Title:      "fix: normalize API errors",
		ChangedFiles: []PRChangedFile{
			{Filename: "internal/client/client.go", Status: "modified", Additions: 20, Deletions: 4},
		},
		Additions: 20,
		Deletions: 4,
	}, "en")

	if result.RiskLevel != PRRiskHigh {
		t.Fatalf("RiskLevel = %q, want %q", result.RiskLevel, PRRiskHigh)
	}
}

func TestAnalyzePRSummaryAuthTokenCriticalRisk(t *testing.T) {
	result := AnalyzePRSummary(PRSummaryInput{
		Repository: "owner/repo",
		Number:     4,
		Title:      "fix: prevent token permission leak",
		Body:       "Avoid auth bypass and secret exposure.",
		ChangedFiles: []PRChangedFile{
			{Filename: "internal/auth/auth.go", Status: "modified", Additions: 20},
		},
	}, "en")

	if result.RiskLevel != PRRiskCritical {
		t.Fatalf("RiskLevel = %q, want %q", result.RiskLevel, PRRiskCritical)
	}
}

func TestAnalyzePRSummaryMixedFiles(t *testing.T) {
	result := AnalyzePRSummary(PRSummaryInput{
		Repository: "owner/repo",
		Number:     5,
		Title:      "feat: add workflow command and docs",
		ChangedFiles: []PRChangedFile{
			{Filename: "shortcuts/workflow/pr_summary.go", Status: "added", Additions: 80},
			{Filename: "docs/workflow-agent-design.md", Status: "modified", Additions: 10},
		},
	}, "en")

	if result.ChangeType != PRChangeTypeMixed {
		t.Fatalf("ChangeType = %q, want %q", result.ChangeType, PRChangeTypeMixed)
	}
}

func TestAnalyzePRSummaryChineseText(t *testing.T) {
	result := AnalyzePRSummary(PRSummaryInput{
		Repository: "owner/repo",
		Number:     6,
		Title:      "feat: add workflow command",
		ChangedFiles: []PRChangedFile{
			{Filename: "shortcuts/workflow/pr_summary.go", Status: "added", Additions: 80},
		},
	}, "zh-CN")

	if len(result.ReviewFocus) == 0 || len(result.TestSuggestions) == 0 || len(result.MergeChecklist) == 0 {
		t.Fatalf("expected non-empty zh-CN recommendations, got focus=%v suggestions=%v checklist=%v", result.ReviewFocus, result.TestSuggestions, result.MergeChecklist)
	}
	joined := strings.Join(append(append(result.ReviewFocus, result.TestSuggestions...), result.MergeChecklist...), "")
	if !strings.Contains(joined, "检查") && !strings.Contains(joined, "运行") {
		t.Fatalf("expected zh-CN text, got %q", joined)
	}
}

func TestPRSummaryShortcutFromJSONFile(t *testing.T) {
	restoreFormat := setCommandFormatForTest(t, "json")
	defer restoreFormat()

	ctx := &common.RuntimeContext{
		Format: "json",
		Args: map[string]string{
			"from": "testdata/pr_summary.json",
			"lang": "en",
		},
	}

	output := captureStdout(t, func() error {
		return findWorkflowShortcut(t, "pr-summary").Run(ctx)
	})
	var result PRSummaryResult
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatalf("json.Unmarshal returned error: %v\noutput=%s", err, output)
	}
	if result.Number != 1 {
		t.Fatalf("Number = %d, want 1", result.Number)
	}
}

func TestPRSummaryShortcutRemoteFetch(t *testing.T) {
	restoreFormat := setCommandFormatForTest(t, "json")
	defer restoreFormat()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == "GET" && r.URL.Path == "/v1/owner/repo/pulls/5.json":
			writeWorkflowJSON(t, w, map[string]interface{}{
				"data": map[string]interface{}{
					"number":      5,
					"title":       "feat: add remote summary",
					"state":       "open",
					"user":        map[string]interface{}{"login": "alice"},
					"base_branch": "master",
					"head_branch": "feature/pr-summary",
				},
			})
		case r.Method == "GET" && r.URL.Path == "/v1/owner/repo/pulls/5/files.json":
			writeWorkflowJSON(t, w, map[string]interface{}{
				"files": []map[string]interface{}{
					{"filename": "shortcuts/workflow/pr_summary.go", "status": "added", "additions": 50},
				},
			})
		case r.Method == "GET" && r.URL.Path == "/v1/owner/repo/pulls/5/commits.json":
			writeWorkflowJSON(t, w, []map[string]interface{}{
				{"sha": "abc123", "message": "feat: add remote summary", "author": map[string]interface{}{"name": "alice"}},
			})
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	ctx := &common.RuntimeContext{
		Client: &client.Client{
			HTTP:    server.Client(),
			BaseURL: server.URL,
		},
		Owner:  "owner",
		Repo:   "repo",
		Format: "json",
		Args: map[string]string{
			"number":          "5",
			"include-files":   "true",
			"include-commits": "true",
			"max-files":       "100",
			"max-commits":     "100",
			"lang":            "en",
		},
	}

	output := captureStdout(t, func() error {
		return findWorkflowShortcut(t, "pr-summary").Run(ctx)
	})
	var result PRSummaryResult
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatalf("json.Unmarshal returned error: %v\noutput=%s", err, output)
	}
	if result.Number != 5 || result.Source != "remote-read-only-fetch" {
		t.Fatalf("result = %+v, want number 5 remote source", result)
	}
}

func TestPRSummaryShortcutMissingParameters(t *testing.T) {
	ctx := &common.RuntimeContext{
		Format: "json",
		Args:   map[string]string{"lang": "en"},
	}

	err := findWorkflowShortcut(t, "pr-summary").Run(ctx)
	if err == nil {
		t.Fatal("Run returned nil error for missing parameters")
	}
	if !strings.Contains(err.Error(), "requires --from") {
		t.Fatalf("error = %v, want clear missing input error", err)
	}
}

func findWorkflowShortcut(t *testing.T, name string) *common.Shortcut {
	t.Helper()
	for _, shortcut := range Shortcuts() {
		if shortcut.Name == name {
			return shortcut
		}
	}
	t.Fatalf("shortcut %q not found", name)
	return nil
}

func setCommandFormatForTest(t *testing.T, format string) func() {
	t.Helper()
	old := cmdutil.Format
	cmdutil.Format = format
	return func() {
		cmdutil.Format = old
	}
}

func captureStdout(t *testing.T, fn func() error) string {
	t.Helper()
	old := os.Stdout
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe returned error: %v", err)
	}
	os.Stdout = writer
	runErr := fn()
	closeErr := writer.Close()
	os.Stdout = old
	if runErr != nil {
		t.Fatalf("function returned error: %v", runErr)
	}
	if closeErr != nil {
		t.Fatalf("writer.Close returned error: %v", closeErr)
	}
	data, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("io.ReadAll returned error: %v", err)
	}
	return string(data)
}

func TestReadPRSummaryInput(t *testing.T) {
	path := filepath.Join(t.TempDir(), "pr_summary.json")
	input := PRSummaryInput{Number: 9, Title: "docs: update README", Repository: "owner/repo"}
	encoded, err := json.Marshal(input)
	if err != nil {
		t.Fatalf("json.Marshal returned error: %v", err)
	}
	if err := os.WriteFile(path, encoded, 0600); err != nil {
		t.Fatalf("os.WriteFile returned error: %v", err)
	}

	got, err := readPRSummaryInput(path)
	if err != nil {
		t.Fatalf("readPRSummaryInput returned error: %v", err)
	}
	if got.Number != 9 || got.Title != "docs: update README" {
		t.Fatalf("got = %+v, want input fields", got)
	}
}
