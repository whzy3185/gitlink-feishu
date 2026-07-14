package workflow

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"text/tabwriter"

	"github.com/gitlink-org/gitlink-cli/cmd/cmdutil"
	"github.com/gitlink-org/gitlink-cli/shortcuts/common"
)

type DependencyAuditInput struct {
	Repository   string                  `json:"repository,omitempty"`
	Module       string                  `json:"module"`
	GoVersion    string                  `json:"go_version,omitempty"`
	Toolchain    string                  `json:"toolchain,omitempty"`
	Requirements []DependencyRequirement `json:"requirements"`
	Replacements []DependencyReplacement `json:"replacements,omitempty"`
	Source       string                  `json:"source,omitempty"`
}

type DependencyRequirement struct {
	Path     string `json:"path"`
	Version  string `json:"version"`
	Indirect bool   `json:"indirect,omitempty"`
}

type DependencyReplacement struct {
	OldPath    string `json:"old_path"`
	OldVersion string `json:"old_version,omitempty"`
	NewPath    string `json:"new_path"`
	NewVersion string `json:"new_version,omitempty"`
}

type DependencyAuditResult struct {
	Repository            string                   `json:"repository"`
	Module                string                   `json:"module"`
	GoVersion             string                   `json:"go_version,omitempty"`
	Toolchain             string                   `json:"toolchain,omitempty"`
	TotalDependencies     int                      `json:"total_dependencies"`
	DirectDependencies    int                      `json:"direct_dependencies"`
	IndirectDependencies  int                      `json:"indirect_dependencies"`
	ReplacementCount      int                      `json:"replacement_count"`
	PseudoVersionCount    int                      `json:"pseudo_version_count"`
	LocalReplacementCount int                      `json:"local_replacement_count"`
	MajorMismatchCount    int                      `json:"major_mismatch_count"`
	PreReleaseCount       int                      `json:"pre_release_count"`
	Score                 int                      `json:"score"`
	RiskLevel             string                   `json:"risk_level"`
	Findings              []DependencyAuditFinding `json:"findings"`
	Recommendations       []string                 `json:"recommendations"`
	Source                string                   `json:"source"`
}

type DependencyAuditFinding struct {
	Severity   string `json:"severity"`
	Code       string `json:"code"`
	Dependency string `json:"dependency,omitempty"`
	Message    string `json:"message"`
}

func newDependencyAuditShortcut() *common.Shortcut {
	return &common.Shortcut{
		Name:        "dependency-audit",
		Description: "Audit go.mod dependency risk signals without network access",
		Flags: []common.Flag{
			{Name: "from", Usage: "Read go.mod or DependencyAuditInput JSON from a local file", Default: "go.mod"},
			{Name: "repository", Usage: "Repository name, for example owner/repo"},
			{Name: "lang", Usage: "Output language: en or zh-CN", Default: langEN},
		},
		Run: runDependencyAudit,
	}
}

func runDependencyAudit(ctx *common.RuntimeContext) error {
	input, err := collectDependencyAuditInput(ctx)
	if err != nil {
		return err
	}
	lang := normalizeLang(ctx.Arg("lang"))
	result := AnalyzeDependencyAudit(input, lang)
	format := ctx.Format
	if strings.TrimSpace(cmdutil.Format) == "" {
		format = "table"
	}
	rendered, err := RenderDependencyAudit(result, format, lang)
	if err != nil {
		return err
	}
	_, err = fmt.Fprint(os.Stdout, rendered)
	return err
}

func collectDependencyAuditInput(ctx *common.RuntimeContext) (DependencyAuditInput, error) {
	path := strings.TrimSpace(ctx.Arg("from"))
	if path == "" {
		path = "go.mod"
	}
	input, err := readDependencyAuditInput(path)
	if err != nil {
		return DependencyAuditInput{}, err
	}
	if repo := strings.TrimSpace(ctx.Arg("repository")); repo != "" {
		input.Repository = repo
	}
	if input.Repository == "" {
		input.Repository = repositoryFromContext(ctx, "")
	}
	if input.Source == "" {
		input.Source = path
	}
	return input, nil
}

func readDependencyAuditInput(path string) (DependencyAuditInput, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return DependencyAuditInput{}, fmt.Errorf("read dependency audit input: %w", err)
	}
	trimmed := bytes.TrimSpace(data)
	if len(trimmed) > 0 && trimmed[0] == '{' {
		var input DependencyAuditInput
		if err := json.Unmarshal(trimmed, &input); err != nil {
			return DependencyAuditInput{}, fmt.Errorf("parse dependency audit JSON: %w", err)
		}
		if strings.TrimSpace(input.Module) == "" {
			return DependencyAuditInput{}, fmt.Errorf("parse dependency audit JSON: module is required")
		}
		if input.Source == "" {
			input.Source = path
		}
		return input, nil
	}
	input, err := ParseGoModForDependencyAudit(string(data))
	if err != nil {
		return DependencyAuditInput{}, err
	}
	input.Source = path
	return input, nil
}

func ParseGoModForDependencyAudit(content string) (DependencyAuditInput, error) {
	scanner := bufio.NewScanner(strings.NewReader(content))
	input := DependencyAuditInput{}
	inRequireBlock := false
	inReplaceBlock := false
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		if inRequireBlock {
			if line == ")" {
				inRequireBlock = false
				continue
			}
			if req, ok := parseGoModRequirement(line); ok {
				input.Requirements = append(input.Requirements, req)
			}
			continue
		}
		if inReplaceBlock {
			if line == ")" {
				inReplaceBlock = false
				continue
			}
			if repl, ok := parseGoModReplacement(line); ok {
				input.Replacements = append(input.Replacements, repl)
			}
			continue
		}
		switch {
		case line == "require (":
			inRequireBlock = true
		case line == "replace (":
			inReplaceBlock = true
		case strings.HasPrefix(line, "module "):
			input.Module = strings.TrimSpace(strings.TrimPrefix(line, "module "))
		case strings.HasPrefix(line, "go "):
			input.GoVersion = strings.TrimSpace(strings.TrimPrefix(line, "go "))
		case strings.HasPrefix(line, "toolchain "):
			input.Toolchain = strings.TrimSpace(strings.TrimPrefix(line, "toolchain "))
		case strings.HasPrefix(line, "require "):
			if req, ok := parseGoModRequirement(strings.TrimSpace(strings.TrimPrefix(line, "require "))); ok {
				input.Requirements = append(input.Requirements, req)
			}
		case strings.HasPrefix(line, "replace "):
			if repl, ok := parseGoModReplacement(strings.TrimSpace(strings.TrimPrefix(line, "replace "))); ok {
				input.Replacements = append(input.Replacements, repl)
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return DependencyAuditInput{}, fmt.Errorf("scan go.mod: %w", err)
	}
	if strings.TrimSpace(input.Module) == "" {
		return DependencyAuditInput{}, fmt.Errorf("parse go.mod: module directive is required")
	}
	return input, nil
}

func parseGoModRequirement(line string) (DependencyRequirement, bool) {
	withoutComment, comment := splitGoModComment(line)
	fields := strings.Fields(withoutComment)
	if len(fields) < 2 {
		return DependencyRequirement{}, false
	}
	return DependencyRequirement{
		Path:     fields[0],
		Version:  fields[1],
		Indirect: strings.Contains(comment, "indirect"),
	}, true
}

func parseGoModReplacement(line string) (DependencyReplacement, bool) {
	withoutComment, _ := splitGoModComment(line)
	parts := strings.Split(withoutComment, "=>")
	if len(parts) != 2 {
		return DependencyReplacement{}, false
	}
	oldFields := strings.Fields(strings.TrimSpace(parts[0]))
	newFields := strings.Fields(strings.TrimSpace(parts[1]))
	if len(oldFields) == 0 || len(newFields) == 0 {
		return DependencyReplacement{}, false
	}
	repl := DependencyReplacement{OldPath: oldFields[0], NewPath: newFields[0]}
	if len(oldFields) > 1 {
		repl.OldVersion = oldFields[1]
	}
	if len(newFields) > 1 {
		repl.NewVersion = newFields[1]
	}
	return repl, true
}

func splitGoModComment(line string) (string, string) {
	idx := strings.Index(line, "//")
	if idx < 0 {
		return strings.TrimSpace(line), ""
	}
	return strings.TrimSpace(line[:idx]), strings.TrimSpace(line[idx+2:])
}

func AnalyzeDependencyAudit(input DependencyAuditInput, lang string) DependencyAuditResult {
	lang = normalizeLang(lang)
	result := DependencyAuditResult{
		Repository:        strings.TrimSpace(input.Repository),
		Module:            strings.TrimSpace(input.Module),
		GoVersion:         strings.TrimSpace(input.GoVersion),
		Toolchain:         strings.TrimSpace(input.Toolchain),
		TotalDependencies: len(input.Requirements),
		ReplacementCount:  len(input.Replacements),
		Source:            strings.TrimSpace(input.Source),
	}
	if result.Repository == "" {
		result.Repository = "local"
	}
	if result.Source == "" {
		result.Source = "local"
	}

	for _, req := range input.Requirements {
		if req.Indirect {
			result.IndirectDependencies++
		} else {
			result.DirectDependencies++
		}
		if isPseudoVersion(req.Version) {
			result.PseudoVersionCount++
			result.Findings = append(result.Findings, dependencyFinding(lang, "medium", "pseudo_version", req.Path, "dependency uses a pseudo version; pin a tagged release when possible"))
		}
		if isPreReleaseVersion(req.Version) {
			result.PreReleaseCount++
			result.Findings = append(result.Findings, dependencyFinding(lang, "low", "pre_release", req.Path, "dependency uses a pre-release version; verify it is intentional"))
		}
		if hasMajorVersionMismatch(req.Path, req.Version) {
			result.MajorMismatchCount++
			result.Findings = append(result.Findings, dependencyFinding(lang, "high", "major_version_mismatch", req.Path, "module path and semantic major version do not match"))
		}
	}
	for _, repl := range input.Replacements {
		dep := repl.OldPath
		if dep == "" {
			dep = repl.NewPath
		}
		switch {
		case isLocalReplacement(repl.NewPath):
			result.LocalReplacementCount++
			result.Findings = append(result.Findings, dependencyFinding(lang, "high", "local_replace", dep, "local replace directive can break clean builds outside this checkout"))
		case repl.NewVersion == "" && !looksLikeModulePath(repl.NewPath):
			result.Findings = append(result.Findings, dependencyFinding(lang, "medium", "replace_without_version", dep, "replace directive has no target version"))
		default:
			result.Findings = append(result.Findings, dependencyFinding(lang, "low", "replace_directive", dep, "replace directive should be reviewed before release"))
		}
	}
	if result.GoVersion == "" {
		result.Findings = append(result.Findings, dependencyFinding(lang, "high", "missing_go_version", result.Module, "go directive is missing"))
	} else if isOldGoVersion(result.GoVersion) {
		result.Findings = append(result.Findings, dependencyFinding(lang, "medium", "old_go_version", result.Module, "go directive is older than 1.20; verify supported toolchains"))
	}
	sort.SliceStable(result.Findings, func(i, j int) bool {
		return dependencySeverityWeight(result.Findings[i].Severity) > dependencySeverityWeight(result.Findings[j].Severity)
	})
	result.Score = scoreDependencyAudit(result.Findings)
	result.RiskLevel = dependencyRiskLevel(result.Findings)
	result.Recommendations = dependencyAuditRecommendations(result, lang)
	return result
}

func dependencyFinding(lang, severity, code, dependency, message string) DependencyAuditFinding {
	if normalizeLang(lang) == langZH {
		switch code {
		case "pseudo_version":
			message = "依赖使用 pseudo version，建议在可行时固定到正式 tag。"
		case "pre_release":
			message = "依赖使用预发布版本，请确认这是有意选择。"
		case "major_version_mismatch":
			message = "模块路径和语义化主版本不匹配。"
		case "local_replace":
			message = "本地 replace 可能导致其他环境无法干净构建。"
		case "replace_without_version":
			message = "replace 指令缺少目标版本。"
		case "replace_directive":
			message = "发布前需要人工确认 replace 指令。"
		case "missing_go_version":
			message = "go.mod 缺少 go 指令。"
		case "old_go_version":
			message = "go 指令低于 1.20，请确认支持的构建工具链。"
		}
	}
	return DependencyAuditFinding{
		Severity:   severity,
		Code:       code,
		Dependency: dependency,
		Message:    message,
	}
}

var pseudoVersionPattern = regexp.MustCompile(`^v\d+\.\d+\.\d+-(?:\w+\.)?0\.\d{14}-[0-9a-f]{12}$|^v\d+\.\d+\.\d+-\d{14}-[0-9a-f]{12}$`)

func isPseudoVersion(version string) bool {
	return pseudoVersionPattern.MatchString(strings.TrimSpace(version))
}

func isPreReleaseVersion(version string) bool {
	version = strings.ToLower(strings.TrimSpace(version))
	return strings.Contains(version, "-alpha") || strings.Contains(version, "-beta") || strings.Contains(version, "-rc") || strings.Contains(version, "-dev")
}

func hasMajorVersionMismatch(path, version string) bool {
	major := semanticMajor(version)
	if major < 2 {
		return false
	}
	if strings.HasPrefix(path, "gopkg.in/") {
		return !strings.Contains(path, fmt.Sprintf(".v%d", major))
	}
	return !strings.HasSuffix(path, fmt.Sprintf("/v%d", major))
}

func semanticMajor(version string) int {
	version = strings.TrimPrefix(strings.TrimSpace(version), "v")
	parts := strings.Split(version, ".")
	if len(parts) == 0 {
		return 0
	}
	major, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0
	}
	return major
}

func isLocalReplacement(path string) bool {
	path = strings.TrimSpace(path)
	if path == "" {
		return false
	}
	if filepath.IsAbs(path) || strings.HasPrefix(path, ".") || strings.HasPrefix(path, `..\`) || strings.HasPrefix(path, "../") {
		return true
	}
	return len(path) >= 2 && path[1] == ':'
}

func looksLikeModulePath(path string) bool {
	return strings.Contains(path, ".") && strings.Contains(path, "/")
}

func isOldGoVersion(version string) bool {
	parts := strings.Split(strings.TrimSpace(version), ".")
	if len(parts) < 2 {
		return false
	}
	major, errMajor := strconv.Atoi(parts[0])
	minor, errMinor := strconv.Atoi(parts[1])
	if errMajor != nil || errMinor != nil {
		return false
	}
	return major < 1 || (major == 1 && minor < 20)
}

func scoreDependencyAudit(findings []DependencyAuditFinding) int {
	score := 100
	for _, finding := range findings {
		switch finding.Severity {
		case "critical":
			score -= 25
		case "high":
			score -= 15
		case "medium":
			score -= 8
		case "low":
			score -= 3
		}
	}
	if score < 0 {
		return 0
	}
	return score
}

func dependencyRiskLevel(findings []DependencyAuditFinding) string {
	for _, finding := range findings {
		if finding.Severity == "critical" || finding.Severity == "high" {
			return "high"
		}
	}
	for _, finding := range findings {
		if finding.Severity == "medium" {
			return "medium"
		}
	}
	if len(findings) > 0 {
		return "low"
	}
	return "clean"
}

func dependencySeverityWeight(severity string) int {
	switch severity {
	case "critical":
		return 4
	case "high":
		return 3
	case "medium":
		return 2
	case "low":
		return 1
	default:
		return 0
	}
}

func dependencyAuditRecommendations(result DependencyAuditResult, lang string) []string {
	zh := normalizeLang(lang) == langZH
	recs := []string{}
	if result.LocalReplacementCount > 0 {
		if zh {
			recs = append(recs, "合并或发布前移除本地 replace，或改为可复现的远端模块版本。")
		} else {
			recs = append(recs, "Remove local replace directives before merge or release, or point them at reproducible module versions.")
		}
	}
	if result.MajorMismatchCount > 0 {
		if zh {
			recs = append(recs, "修正主版本路径不匹配的依赖，避免 Go module 解析和升级出现异常。")
		} else {
			recs = append(recs, "Fix major-version path mismatches to avoid Go module resolution and upgrade surprises.")
		}
	}
	if result.PseudoVersionCount > 0 {
		if zh {
			recs = append(recs, "将 pseudo version 升级到正式 tag，降低不可读提交依赖带来的维护成本。")
		} else {
			recs = append(recs, "Prefer tagged releases over pseudo versions to reduce maintenance risk.")
		}
	}
	if len(recs) == 0 {
		if zh {
			recs = append(recs, "当前未发现明显依赖风险，保持 go.mod 简洁并在合并前运行 go mod tidy。")
		} else {
			recs = append(recs, "No obvious dependency risks found; keep go.mod tidy and run go mod tidy before merge.")
		}
	}
	return recs
}

func RenderDependencyAudit(result DependencyAuditResult, format string, lang string) (string, error) {
	var buf bytes.Buffer
	switch normalizeFormat(format) {
	case "json":
		if err := writeJSON(&buf, result); err != nil {
			return "", err
		}
	case "markdown":
		if err := writeDependencyAuditMarkdown(&buf, result, lang); err != nil {
			return "", err
		}
	case "table":
		if err := writeDependencyAuditTable(&buf, result); err != nil {
			return "", err
		}
	default:
		return "", fmt.Errorf("unsupported workflow output format %q", format)
	}
	return buf.String(), nil
}

func writeDependencyAuditMarkdown(buf *bytes.Buffer, result DependencyAuditResult, lang string) error {
	title := "Dependency Audit"
	if normalizeLang(lang) == langZH {
		title = "依赖风险审计"
	}
	if _, err := fmt.Fprintf(buf, "# %s\n\n", title); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(buf, "- Repository: `%s`\n- Module: `%s`\n- Go version: `%s`\n- Score: `%d`\n- Risk: `%s`\n- Dependencies: `%d` direct / `%d` indirect\n- Replacements: `%d`\n- Source: `%s`\n\n",
		result.Repository, result.Module, emptyDash(result.GoVersion), result.Score, result.RiskLevel, result.DirectDependencies, result.IndirectDependencies, result.ReplacementCount, result.Source); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(buf, "## Findings"); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(buf); err != nil {
		return err
	}
	if len(result.Findings) == 0 {
		if _, err := fmt.Fprintln(buf, "- No dependency risk findings."); err != nil {
			return err
		}
	} else {
		for _, finding := range result.Findings {
			dep := finding.Dependency
			if dep == "" {
				dep = "-"
			}
			if _, err := fmt.Fprintf(buf, "- `%s` `%s` %s: %s\n", finding.Severity, finding.Code, dep, finding.Message); err != nil {
				return err
			}
		}
	}
	if _, err := fmt.Fprintln(buf); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(buf, "## Recommendations"); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(buf); err != nil {
		return err
	}
	for _, rec := range result.Recommendations {
		if _, err := fmt.Fprintf(buf, "- %s\n", rec); err != nil {
			return err
		}
	}
	return nil
}

func writeDependencyAuditTable(buf *bytes.Buffer, result DependencyAuditResult) error {
	tw := tabwriter.NewWriter(buf, 0, 0, 2, ' ', 0)
	if _, err := fmt.Fprintln(tw, "MODULE\tSCORE\tRISK\tDIRECT\tINDIRECT\tREPLACE\tPSEUDO\tLOCAL_REPLACE\tMAJOR_MISMATCH"); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(tw, "%s\t%d\t%s\t%d\t%d\t%d\t%d\t%d\t%d\n",
		result.Module,
		result.Score,
		result.RiskLevel,
		result.DirectDependencies,
		result.IndirectDependencies,
		result.ReplacementCount,
		result.PseudoVersionCount,
		result.LocalReplacementCount,
		result.MajorMismatchCount,
	); err != nil {
		return err
	}
	if len(result.Findings) > 0 {
		if _, err := fmt.Fprintln(tw, "\nSEVERITY\tCODE\tDEPENDENCY\tMESSAGE"); err != nil {
			return err
		}
		for _, finding := range result.Findings {
			if _, err := fmt.Fprintf(tw, "%s\t%s\t%s\t%s\n", finding.Severity, finding.Code, emptyDash(finding.Dependency), finding.Message); err != nil {
				return err
			}
		}
	}
	return tw.Flush()
}

func emptyDash(value string) string {
	if strings.TrimSpace(value) == "" {
		return "-"
	}
	return value
}
