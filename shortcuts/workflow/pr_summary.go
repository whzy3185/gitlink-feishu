package workflow

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/gitlink-org/gitlink-cli/cmd/cmdutil"
	"github.com/gitlink-org/gitlink-cli/shortcuts/common"
)

const (
	PRChangeTypeDocs     = "docs"
	PRChangeTypeTest     = "test"
	PRChangeTypeFeature  = "feature"
	PRChangeTypeFix      = "fix"
	PRChangeTypeRefactor = "refactor"
	PRChangeTypeCI       = "ci"
	PRChangeTypeMixed    = "mixed"
	PRChangeTypeUnknown  = "unknown"
)

const (
	PRRiskLow      = "low"
	PRRiskMedium   = "medium"
	PRRiskHigh     = "high"
	PRRiskCritical = "critical"
)

type PRSummaryInput struct {
	Repository   string          `json:"repository"`
	Number       int             `json:"number"`
	Title        string          `json:"title"`
	Author       string          `json:"author"`
	State        string          `json:"state"`
	BaseBranch   string          `json:"base_branch"`
	HeadBranch   string          `json:"head_branch"`
	Body         string          `json:"body,omitempty"`
	ChangedFiles []PRChangedFile `json:"changed_files"`
	Commits      []PRCommit      `json:"commits"`
	Additions    int             `json:"additions"`
	Deletions    int             `json:"deletions"`
	Source       string          `json:"source"`
}

type PRChangedFile struct {
	Filename  string `json:"filename"`
	Status    string `json:"status"`
	Additions int    `json:"additions"`
	Deletions int    `json:"deletions"`
	Changes   int    `json:"changes"`
	Patch     string `json:"patch,omitempty"`
}

type PRCommit struct {
	SHA     string    `json:"sha"`
	Message string    `json:"message"`
	Author  string    `json:"author"`
	Date    time.Time `json:"date,omitempty"`
}

type PRSummaryResult struct {
	Repository        string   `json:"repository"`
	Number            int      `json:"number"`
	Title             string   `json:"title"`
	Author            string   `json:"author"`
	State             string   `json:"state"`
	BaseBranch        string   `json:"base_branch"`
	HeadBranch        string   `json:"head_branch"`
	ChangedFilesCount int      `json:"changed_files_count"`
	Additions         int      `json:"additions"`
	Deletions         int      `json:"deletions"`
	CommitCount       int      `json:"commit_count"`
	ChangeType        string   `json:"change_type"`
	RiskLevel         string   `json:"risk_level"`
	ReviewFocus       []string `json:"review_focus"`
	TestSuggestions   []string `json:"test_suggestions"`
	MergeChecklist    []string `json:"merge_checklist"`
	Reasoning         []string `json:"reasoning"`
	Source            string   `json:"source"`
}

func newPRSummaryShortcut() *common.Shortcut {
	return &common.Shortcut{
		Name:        "pr-summary",
		Description: "Generate a read-only pull request review summary",
		Flags: []common.Flag{
			{Name: "from", Usage: "Read PR summary input from a JSON file"},
			{Name: "number", Short: "n", Usage: "Pull request number for remote read-only analysis"},
			{Name: "include-commits", Usage: "Fetch commits in remote mode", Bool: true, Default: "true"},
			{Name: "include-files", Usage: "Fetch changed files in remote mode", Bool: true, Default: "true"},
			{Name: "max-files", Usage: "Maximum changed files to analyze", Default: "100"},
			{Name: "max-commits", Usage: "Maximum commits to analyze", Default: "100"},
			{Name: "lang", Usage: "Output language: en or zh-CN", Default: langEN},
		},
		Run: runPRSummary,
	}
}

func runPRSummary(ctx *common.RuntimeContext) error {
	lang := normalizeLang(ctx.Arg("lang"))

	input, notes, err := collectPRSummaryInput(ctx)
	if err != nil {
		return err
	}

	result := AnalyzePRSummary(input, lang)
	for _, note := range notes {
		if note.Metric == "" && note.Note == "" {
			continue
		}
		result.Reasoning = append(result.Reasoning, fmt.Sprintf("%s: %s", note.Metric, note.Note))
	}

	format := ctx.Format
	if strings.TrimSpace(cmdutil.Format) == "" {
		format = "table"
	}
	rendered, err := RenderPRSummary(result, format, lang)
	if err != nil {
		return err
	}
	_, err = fmt.Fprint(os.Stdout, rendered)
	return err
}

func collectPRSummaryInput(ctx *common.RuntimeContext) (PRSummaryInput, []ScoringNote, error) {
	if path := strings.TrimSpace(ctx.Arg("from")); path != "" {
		input, err := readPRSummaryInput(path)
		if err != nil {
			return PRSummaryInput{}, nil, err
		}
		if strings.TrimSpace(input.Source) == "" {
			input.Source = "local-json"
		}
		return input, nil, nil
	}

	number, err := parseIntArg(ctx.Arg("number"), 0, "number")
	if err != nil {
		return PRSummaryInput{}, nil, err
	}
	if number <= 0 {
		return PRSummaryInput{}, nil, fmt.Errorf("workflow +pr-summary requires --from pr_summary.json or --number with --owner and --repo for read-only fetch")
	}
	maxFiles, err := parseIntArg(ctx.Arg("max-files"), 100, "max-files")
	if err != nil {
		return PRSummaryInput{}, nil, err
	}
	maxCommits, err := parseIntArg(ctx.Arg("max-commits"), 100, "max-commits")
	if err != nil {
		return PRSummaryInput{}, nil, err
	}

	return FetchPRSummaryInput(ctx, PRFetchOptions{
		Number:         number,
		IncludeFiles:   parseBoolArg(ctx.Arg("include-files")),
		IncludeCommits: parseBoolArg(ctx.Arg("include-commits")),
		MaxFiles:       maxFiles,
		MaxCommits:     maxCommits,
	})
}

func readPRSummaryInput(path string) (PRSummaryInput, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return PRSummaryInput{}, fmt.Errorf("read PR summary input: %w", err)
	}
	var input PRSummaryInput
	if err := json.Unmarshal(data, &input); err != nil {
		return PRSummaryInput{}, fmt.Errorf("parse PR summary input: %w", err)
	}
	if strings.TrimSpace(input.Title) == "" && input.Number == 0 {
		return PRSummaryInput{}, fmt.Errorf("parse PR summary input: expected PRSummaryInput root object")
	}
	return input, nil
}

func AnalyzePRSummary(input PRSummaryInput, lang string) PRSummaryResult {
	lang = normalizeLang(lang)
	scores, scoreReasons := scorePRChangeTypes(input)
	changeType := choosePRChangeType(scores)
	riskLevel, riskReasons := scorePRRisk(input, changeType)
	reviewFocus := buildPRReviewFocus(input, lang, riskLevel)
	testSuggestions := buildPRTestSuggestions(input, lang)
	mergeChecklist := buildPRMergeChecklist(input, lang, riskLevel)
	reasoning := buildPRReasoning(lang, changeType, riskLevel, scores, scoreReasons, riskReasons)

	source := strings.TrimSpace(input.Source)
	if source == "" {
		source = "local"
	}

	return PRSummaryResult{
		Repository:        input.Repository,
		Number:            input.Number,
		Title:             input.Title,
		Author:            input.Author,
		State:             input.State,
		BaseBranch:        input.BaseBranch,
		HeadBranch:        input.HeadBranch,
		ChangedFilesCount: len(input.ChangedFiles),
		Additions:         input.Additions,
		Deletions:         input.Deletions,
		CommitCount:       len(input.Commits),
		ChangeType:        changeType,
		RiskLevel:         riskLevel,
		ReviewFocus:       reviewFocus,
		TestSuggestions:   testSuggestions,
		MergeChecklist:    mergeChecklist,
		Reasoning:         reasoning,
		Source:            source,
	}
}

func scorePRChangeTypes(input PRSummaryInput) (map[string]int, []string) {
	scores := map[string]int{
		PRChangeTypeDocs:     0,
		PRChangeTypeTest:     0,
		PRChangeTypeFeature:  0,
		PRChangeTypeFix:      0,
		PRChangeTypeRefactor: 0,
		PRChangeTypeCI:       0,
	}
	reasons := []string{}

	for _, file := range input.ChangedFiles {
		path := normalizedPath(file.Filename)
		switch {
		case isDocsPath(path):
			scores[PRChangeTypeDocs] += 2
			reasons = append(reasons, "file:docs")
		case isTestPath(path):
			scores[PRChangeTypeTest] += 2
			reasons = append(reasons, "file:test")
		case isCIPath(path):
			scores[PRChangeTypeCI] += 2
			reasons = append(reasons, "file:ci")
		}
	}

	text := strings.ToLower(prTextCorpus(input, false))
	addKeywordScore(scores, &reasons, text, PRChangeTypeDocs, []string{"doc", "docs", "documentation", "readme", "typo", "example", "guide"})
	addKeywordScore(scores, &reasons, text, PRChangeTypeTest, []string{"test", "tests", "coverage"})
	addKeywordScore(scores, &reasons, text, PRChangeTypeFeature, []string{"feat", "feature", "add", "support", "implement"})
	addKeywordScore(scores, &reasons, text, PRChangeTypeFix, []string{"fix", "bug", "error", "crash", "resolve"})
	addKeywordScore(scores, &reasons, text, PRChangeTypeRefactor, []string{"refactor", "cleanup", "simplify", "restructure"})
	addKeywordScore(scores, &reasons, text, PRChangeTypeCI, []string{"ci", "workflow", "action", "build", "pipeline"})

	return scores, uniqueStrings(reasons)
}

func choosePRChangeType(scores map[string]int) string {
	topType := PRChangeTypeUnknown
	topScore := 0
	secondScore := 0
	hits := []string{}
	for _, kind := range []string{PRChangeTypeDocs, PRChangeTypeTest, PRChangeTypeFeature, PRChangeTypeFix, PRChangeTypeRefactor, PRChangeTypeCI} {
		score := scores[kind]
		if score <= 0 {
			continue
		}
		hits = append(hits, kind)
		if score > topScore {
			secondScore = topScore
			topScore = score
			topType = kind
		} else if score > secondScore {
			secondScore = score
		}
	}
	if len(hits) == 0 {
		return PRChangeTypeUnknown
	}
	if len(hits) == 1 {
		return topType
	}
	if len(hits) == 2 && containsString(hits, PRChangeTypeDocs) && containsString(hits, PRChangeTypeTest) {
		return topType
	}
	if topScore >= secondScore+2 {
		return topType
	}
	return PRChangeTypeMixed
}

func addKeywordScore(scores map[string]int, reasons *[]string, text, kind string, keywords []string) {
	for _, keyword := range keywords {
		if strings.Contains(text, keyword) {
			scores[kind]++
			*reasons = append(*reasons, "keyword:"+kind+":"+keyword)
		}
	}
}

func scorePRRisk(input PRSummaryInput, changeType string) (string, []string) {
	corpus := strings.ToLower(prTextCorpus(input, true))
	reasons := []string{}
	if containsAny(corpus, []string{"token", "secret", "credential", "permission", "vulnerability", "auth bypass", "permission bypass", "leak"}) {
		return PRRiskCritical, []string{"security-sensitive keyword"}
	}
	if touchesCodePath(input, []string{"shortcuts/", "cmd/", "internal/client"}) &&
		containsAny(corpus, []string{"merge", "approve", "comment", "label", "close", "refuse", "journal"}) {
		return PRRiskCritical, []string{"possible remote write operation"}
	}

	if touchesCodePath(input, []string{"internal/client", "internal/auth", "internal/config", "cmd/"}) {
		reasons = append(reasons, "high-risk core path")
	}
	if touchesExactPath(input, "shortcuts/register.go") {
		reasons = append(reasons, "command registration changed")
	}
	if input.Deletions >= 200 || deletedFiles(input) >= 5 {
		reasons = append(reasons, "large deletion")
	}
	if len(reasons) > 0 {
		return PRRiskHigh, reasons
	}

	if changeType == PRChangeTypeDocs || changeType == PRChangeTypeTest {
		if len(input.ChangedFiles) <= 8 && input.Deletions <= 80 {
			return PRRiskLow, []string{"docs-or-tests only"}
		}
	}
	if touchesGoCode(input) || touchesCodePath(input, []string{"shortcuts/"}) {
		return PRRiskMedium, []string{"go or shortcut code changed"}
	}
	return PRRiskLow, []string{"low-risk file scope"}
}

func buildPRReviewFocus(input PRSummaryInput, lang, riskLevel string) []string {
	focus := []string{}
	if touchesCodePath(input, []string{"shortcuts/"}) {
		focus = append(focus, prFocusText(lang, "shortcuts"))
	}
	if touchesExactPath(input, "shortcuts/register.go") {
		focus = append(focus, prFocusText(lang, "registration"))
	}
	if touchesCodePath(input, []string{"internal/client"}) {
		focus = append(focus, prFocusText(lang, "client"))
	}
	if touchesCodePath(input, []string{"internal/output"}) {
		focus = append(focus, prFocusText(lang, "output"))
	}
	if touchesCodePath(input, []string{"internal/auth", "internal/config"}) {
		focus = append(focus, prFocusText(lang, "auth"))
	}
	if touchesDocs(input) {
		focus = append(focus, prFocusText(lang, "docs"))
	}
	if touchesTests(input) {
		focus = append(focus, prFocusText(lang, "tests"))
	}
	if touchesFetchOrAPI(input) {
		focus = append(focus, prFocusText(lang, "api"))
	}
	if riskLevel == PRRiskCritical {
		focus = append(focus, prFocusText(lang, "security"))
	}
	return uniqueStrings(focus)
}

func buildPRTestSuggestions(input PRSummaryInput, lang string) []string {
	suggestions := []string{}
	if touchesGoCode(input) {
		suggestions = append(suggestions, prTestText(lang, "go_all"))
	}
	if touchesCodePath(input, []string{"shortcuts/workflow"}) {
		suggestions = append(suggestions, prTestText(lang, "workflow"))
	}
	if touchesDocs(input) {
		suggestions = append(suggestions, prTestText(lang, "docs"))
	}
	if touchesFetchOrAPI(input) {
		suggestions = append(suggestions, prTestText(lang, "fetch"))
	}
	if touchesCodePath(input, []string{"render", "internal/output"}) {
		suggestions = append(suggestions, prTestText(lang, "render"))
	}
	if len(suggestions) == 0 {
		suggestions = append(suggestions, prTestText(lang, "go_all"))
	}
	return uniqueStrings(suggestions)
}

func buildPRMergeChecklist(input PRSummaryInput, lang, riskLevel string) []string {
	checklist := []string{
		prChecklistText(lang, "tests"),
		prChecklistText(lang, "readme"),
		prChecklistText(lang, "no_write"),
		prChecklistText(lang, "json_stable"),
		prChecklistText(lang, "errors"),
	}
	if riskLevel == PRRiskCritical || riskLevel == PRRiskHigh {
		checklist = append(checklist, prChecklistText(lang, "credentials"))
	}
	if touchesFetchOrAPI(input) || riskLevel == PRRiskHigh {
		checklist = append(checklist, prChecklistText(lang, "api_fallback"))
	}
	if touchesExactPath(input, "shortcuts/register.go") {
		checklist = append(checklist, prChecklistText(lang, "registration"))
	}
	if touchesCodePath(input, []string{"internal/output", "render"}) {
		checklist = append(checklist, prChecklistText(lang, "contract"))
	}
	return uniqueStrings(checklist)
}

func buildPRReasoning(lang, changeType, riskLevel string, scores map[string]int, scoreReasons, riskReasons []string) []string {
	reasoning := []string{}
	if lang == langZH {
		reasoning = append(reasoning, fmt.Sprintf("变更类型判定：%s", changeType))
		reasoning = append(reasoning, fmt.Sprintf("风险等级判定：%s", riskLevel))
	} else {
		reasoning = append(reasoning, fmt.Sprintf("change type: %s", changeType))
		reasoning = append(reasoning, fmt.Sprintf("risk level: %s", riskLevel))
	}
	for _, kind := range []string{PRChangeTypeDocs, PRChangeTypeTest, PRChangeTypeFeature, PRChangeTypeFix, PRChangeTypeRefactor, PRChangeTypeCI} {
		if scores[kind] > 0 {
			if lang == langZH {
				reasoning = append(reasoning, fmt.Sprintf("规则得分 %s=%d", kind, scores[kind]))
			} else {
				reasoning = append(reasoning, fmt.Sprintf("rule score %s=%d", kind, scores[kind]))
			}
		}
	}
	for _, reason := range append(scoreReasons, riskReasons...) {
		if lang == langZH {
			reasoning = append(reasoning, "命中规则："+reason)
		} else {
			reasoning = append(reasoning, "matched rule: "+reason)
		}
	}
	return uniqueStrings(reasoning)
}

func prTextCorpus(input PRSummaryInput, includeFiles bool) string {
	parts := []string{input.Title, input.Body}
	for _, commit := range input.Commits {
		parts = append(parts, commit.Message)
	}
	if includeFiles {
		for _, file := range input.ChangedFiles {
			parts = append(parts, file.Filename, file.Status, file.Patch)
		}
	}
	return strings.Join(parts, "\n")
}

func prFocusText(lang, key string) string {
	zh := lang == langZH
	switch key {
	case "shortcuts":
		if zh {
			return "检查 shortcuts 命令兼容性和参数行为。"
		}
		return "Check shortcut command compatibility and flag behavior."
	case "registration":
		if zh {
			return "确认命令注册和 shortcut 挂载兼容。"
		}
		return "Confirm command registration and shortcut mounting compatibility."
	case "client":
		if zh {
			return "检查 API 错误处理和响应归一化。"
		}
		return "Check API error handling and response normalization."
	case "output":
		if zh {
			return "检查输出格式兼容性和稳定性。"
		}
		return "Check output format compatibility and stability."
	case "auth":
		if zh {
			return "检查凭据处理和安全边界。"
		}
		return "Check credential handling and security boundaries."
	case "docs":
		if zh {
			return "检查文档示例是否与实现一致。"
		}
		return "Check that documentation examples match implementation."
	case "tests":
		if zh {
			return "检查测试是否真实覆盖行为和失败路径。"
		}
		return "Check that tests reflect behavior and failure paths."
	case "api":
		if zh {
			return "检查 fetch/API 失败时的降级和归一化。"
		}
		return "Check fetch/API failure fallback and normalization."
	case "security":
		if zh {
			return "确认没有凭据泄露或不安全的远端写操作。"
		}
		return "Confirm no credential leakage or unsafe remote write operation."
	default:
		return key
	}
}

func prTestText(lang, key string) string {
	zh := lang == langZH
	switch key {
	case "go_all":
		if zh {
			return "运行 `go test ./...`。"
		}
		return "Run `go test ./...`."
	case "workflow":
		if zh {
			return "运行 `go test ./shortcuts/workflow`。"
		}
		return "Run `go test ./shortcuts/workflow`."
	case "docs":
		if zh {
			return "手动检查 README 和文档示例命令。"
		}
		return "Manually check README and documentation examples."
	case "fetch":
		if zh {
			return "运行 httptest mock，必要时执行只读远端 smoke。"
		}
		return "Run httptest mocks and a read-only remote smoke check if needed."
	case "render":
		if zh {
			return "验证 json/table/markdown 输出结构。"
		}
		return "Verify json/table/markdown output structures."
	default:
		return key
	}
}

func prChecklistText(lang, key string) string {
	zh := lang == langZH
	switch key {
	case "tests":
		if zh {
			return "测试通过。"
		}
		return "Tests pass."
	case "readme":
		if zh {
			return "命令行为变化已更新 README。"
		}
		return "README updated if command behavior changed."
	case "no_write":
		if zh {
			return "未引入远端写操作。"
		}
		return "No remote write operation introduced."
	case "json_stable":
		if zh {
			return "JSON 输出字段保持稳定。"
		}
		return "JSON output remains stable."
	case "errors":
		if zh {
			return "错误处理已覆盖。"
		}
		return "Error handling is covered."
	case "credentials":
		if zh {
			return "确认没有凭据泄露。"
		}
		return "Confirm no credential leakage."
	case "api_fallback":
		if zh {
			return "验证 API 失败时的降级路径。"
		}
		return "Verify API failure fallback."
	case "registration":
		if zh {
			return "确认命令注册兼容。"
		}
		return "Confirm command registration compatibility."
	case "contract":
		if zh {
			return "复核 Agent 消费的输出协议。"
		}
		return "Review output contract for Agent consumers."
	default:
		return key
	}
}

func normalizedPath(path string) string {
	path = strings.ReplaceAll(path, "\\", "/")
	return strings.ToLower(strings.TrimSpace(path))
}

func isDocsPath(path string) bool {
	return strings.Contains(path, "docs/") ||
		strings.Contains(path, "/docs/") ||
		strings.Contains(path, "readme") ||
		strings.HasSuffix(path, ".md")
}

func isTestPath(path string) bool {
	return strings.Contains(path, "test") || strings.HasSuffix(path, "_test.go")
}

func isCIPath(path string) bool {
	return strings.Contains(path, ".github") ||
		strings.Contains(path, "workflow") ||
		strings.Contains(path, "ci") ||
		strings.Contains(path, "build")
}

func touchesDocs(input PRSummaryInput) bool {
	for _, file := range input.ChangedFiles {
		if isDocsPath(normalizedPath(file.Filename)) {
			return true
		}
	}
	return false
}

func touchesTests(input PRSummaryInput) bool {
	for _, file := range input.ChangedFiles {
		if isTestPath(normalizedPath(file.Filename)) {
			return true
		}
	}
	return false
}

func touchesGoCode(input PRSummaryInput) bool {
	for _, file := range input.ChangedFiles {
		if strings.HasSuffix(normalizedPath(file.Filename), ".go") {
			return true
		}
	}
	return false
}

func touchesFetchOrAPI(input PRSummaryInput) bool {
	return touchesCodePath(input, []string{"fetch", "api", "internal/client"})
}

func touchesCodePath(input PRSummaryInput, fragments []string) bool {
	for _, file := range input.ChangedFiles {
		path := normalizedPath(file.Filename)
		for _, fragment := range fragments {
			if strings.Contains(path, strings.ToLower(fragment)) {
				return true
			}
		}
	}
	return false
}

func touchesExactPath(input PRSummaryInput, target string) bool {
	target = normalizedPath(target)
	for _, file := range input.ChangedFiles {
		if normalizedPath(file.Filename) == target {
			return true
		}
	}
	return false
}

func deletedFiles(input PRSummaryInput) int {
	count := 0
	for _, file := range input.ChangedFiles {
		if strings.EqualFold(file.Status, "removed") || strings.EqualFold(file.Status, "deleted") {
			count++
		}
	}
	return count
}
