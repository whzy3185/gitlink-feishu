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

func TestReadPRSummaryInputMissingFile(t *testing.T) {
	_, err := readPRSummaryInput("/nonexistent/pr_summary.json")
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestReadPRSummaryInputBadJSON(t *testing.T) {
	path := filepath.Join(t.TempDir(), "bad.json")
	if err := os.WriteFile(path, []byte("{invalid json}"), 0600); err != nil {
		t.Fatalf("os.WriteFile error: %v", err)
	}
	_, err := readPRSummaryInput(path)
	if err == nil {
		t.Fatal("expected error for bad JSON")
	}
}

func TestReadPRSummaryInputEmpty(t *testing.T) {
	path := filepath.Join(t.TempDir(), "empty.json")
	if err := os.WriteFile(path, []byte("{}"), 0600); err != nil {
		t.Fatalf("os.WriteFile error: %v", err)
	}
	_, err := readPRSummaryInput(path)
	if err == nil {
		t.Fatal("expected error for empty input (no title, no number)")
	}
}

func TestPrFocusText(t *testing.T) {
	tests := []struct {
		lang, key, want string
	}{
		{"en", "shortcuts", "Check shortcut command compatibility and flag behavior."},
		{"zh-CN", "shortcuts", "检查 shortcuts 命令兼容性和参数行为。"},
		{"en", "registration", "Confirm command registration and shortcut mounting compatibility."},
		{"zh-CN", "registration", "确认命令注册和 shortcut 挂载兼容。"},
		{"en", "client", "Check API error handling and response normalization."},
		{"zh-CN", "client", "检查 API 错误处理和响应归一化。"},
		{"en", "output", "Check output format compatibility and stability."},
		{"zh-CN", "output", "检查输出格式兼容性和稳定性。"},
		{"en", "auth", "Check credential handling and security boundaries."},
		{"zh-CN", "auth", "检查凭据处理和安全边界。"},
		{"en", "docs", "Check that documentation examples match implementation."},
		{"zh-CN", "docs", "检查文档示例是否与实现一致。"},
		{"en", "tests", "Check that tests reflect behavior and failure paths."},
		{"zh-CN", "tests", "检查测试是否真实覆盖行为和失败路径。"},
		{"en", "api", "Check fetch/API failure fallback and normalization."},
		{"zh-CN", "api", "检查 fetch/API 失败时的降级和归一化。"},
		{"en", "security", "Confirm no credential leakage or unsafe remote write operation."},
		{"zh-CN", "security", "确认没有凭据泄露或不安全的远端写操作。"},
		{"en", "unknown_key", "unknown_key"},
	}
	for _, tt := range tests {
		t.Run(tt.lang+"/"+tt.key, func(t *testing.T) {
			if got := prFocusText(tt.lang, tt.key); got != tt.want {
				t.Fatalf("prFocusText(%q, %q) = %q, want %q", tt.lang, tt.key, got, tt.want)
			}
		})
	}
}

func TestPrTestText(t *testing.T) {
	tests := []struct {
		lang, key, want string
	}{
		{"en", "go_all", "Run `go test ./...`."},
		{"zh-CN", "go_all", "运行 `go test ./...`。"},
		{"en", "workflow", "Run `go test ./shortcuts/workflow`."},
		{"zh-CN", "workflow", "运行 `go test ./shortcuts/workflow`。"},
		{"en", "docs", "Manually check README and documentation examples."},
		{"zh-CN", "docs", "手动检查 README 和文档示例命令。"},
		{"en", "fetch", "Run httptest mocks and a read-only remote smoke check if needed."},
		{"zh-CN", "fetch", "运行 httptest mock，必要时执行只读远端 smoke。"},
		{"en", "render", "Verify json/table/markdown output structures."},
		{"zh-CN", "render", "验证 json/table/markdown 输出结构。"},
		{"en", "unknown", "unknown"},
	}
	for _, tt := range tests {
		t.Run(tt.lang+"/"+tt.key, func(t *testing.T) {
			if got := prTestText(tt.lang, tt.key); got != tt.want {
				t.Fatalf("prTestText(%q, %q) = %q, want %q", tt.lang, tt.key, got, tt.want)
			}
		})
	}
}

func TestPrChecklistText(t *testing.T) {
	tests := []struct {
		lang, key, want string
	}{
		{"en", "tests", "Tests pass."},
		{"zh-CN", "tests", "测试通过。"},
		{"en", "readme", "README updated if command behavior changed."},
		{"zh-CN", "readme", "命令行为变化已更新 README。"},
		{"en", "no_write", "No remote write operation introduced."},
		{"zh-CN", "no_write", "未引入远端写操作。"},
		{"en", "json_stable", "JSON output remains stable."},
		{"zh-CN", "json_stable", "JSON 输出字段保持稳定。"},
		{"en", "errors", "Error handling is covered."},
		{"zh-CN", "errors", "错误处理已覆盖。"},
		{"en", "credentials", "Confirm no credential leakage."},
		{"zh-CN", "credentials", "确认没有凭据泄露。"},
		{"en", "api_fallback", "Verify API failure fallback."},
		{"zh-CN", "api_fallback", "验证 API 失败时的降级路径。"},
		{"en", "registration", "Confirm command registration compatibility."},
		{"zh-CN", "registration", "确认命令注册兼容。"},
		{"en", "contract", "Review output contract for Agent consumers."},
		{"zh-CN", "contract", "复核 Agent 消费的输出协议。"},
		{"en", "unknown", "unknown"},
	}
	for _, tt := range tests {
		t.Run(tt.lang+"/"+tt.key, func(t *testing.T) {
			if got := prChecklistText(tt.lang, tt.key); got != tt.want {
				t.Fatalf("prChecklistText(%q, %q) = %q, want %q", tt.lang, tt.key, got, tt.want)
			}
		})
	}
}
