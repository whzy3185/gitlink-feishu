package workflow

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/gitlink-org/gitlink-cli/cmd/cmdutil"
	"github.com/gitlink-org/gitlink-cli/shortcuts/common"
)

const (
	ReleaseGatePass   = "pass"
	ReleaseGateReview = "review"
	ReleaseGateBlock  = "block"
)

type ReleaseReadinessInput struct {
	Repository      string             `json:"repository,omitempty"`
	Version         string             `json:"version"`
	TargetDate      string             `json:"target_date,omitempty"`
	Changes         []string           `json:"changes,omitempty"`
	BreakingChanges bool               `json:"breaking_changes,omitempty"`
	KnownBlockers   []string           `json:"known_blockers,omitempty"`
	Tests           []ReleaseCheckItem `json:"tests,omitempty"`
	Artifacts       []ReleaseCheckItem `json:"artifacts,omitempty"`
	RollbackPlan    string             `json:"rollback_plan,omitempty"`
	DependencyRisk  string             `json:"dependency_risk,omitempty"`
	Source          string             `json:"source,omitempty"`
}

type ReleaseCheckItem struct {
	Name    string `json:"name"`
	Status  string `json:"status"`
	Details string `json:"details,omitempty"`
}

type ReleaseReadinessResult struct {
	Repository      string                    `json:"repository"`
	Version         string                    `json:"version"`
	TargetDate      string                    `json:"target_date,omitempty"`
	Gate            string                    `json:"gate"`
	Score           int                       `json:"score"`
	PassedChecks    int                       `json:"passed_checks"`
	WarningChecks   int                       `json:"warning_checks"`
	FailedChecks    int                       `json:"failed_checks"`
	MissingChecks   int                       `json:"missing_checks"`
	Findings        []ReleaseReadinessFinding `json:"findings"`
	Recommendations []string                  `json:"recommendations"`
	Source          string                    `json:"source"`
}

type ReleaseReadinessFinding struct {
	Severity string `json:"severity"`
	Code     string `json:"code"`
	Message  string `json:"message"`
}

func newReleaseReadinessShortcut() *common.Shortcut {
	return &common.Shortcut{
		Name:        "release-readiness",
		Description: "Evaluate release readiness from local gate signals",
		Flags: []common.Flag{
			{Name: "from", Usage: "Read release readiness input from a JSON file"},
			{Name: "repository", Usage: "Repository name, for example owner/repo"},
			{Name: "version", Usage: "Release version or tag"},
			{Name: "target-date", Usage: "Planned release date"},
			{Name: "changes", Usage: "Comma-separated release change highlights"},
			{Name: "breaking-changes", Usage: "Whether the release contains breaking changes", Bool: true, Default: "false"},
			{Name: "known-blockers", Usage: "Comma-separated known release blockers"},
			{Name: "tests", Usage: "Comma-separated checks, for example 'go test ./...=passed,go build ./...=passed'"},
			{Name: "artifacts", Usage: "Comma-separated artifact checks, for example 'windows zip=passed,linux tar=missing'"},
			{Name: "rollback-plan", Usage: "Rollback or mitigation plan summary"},
			{Name: "dependency-risk", Usage: "Dependency risk level: clean, low, medium, high"},
			{Name: "lang", Usage: "Output language: en or zh-CN", Default: langEN},
		},
		Run: runReleaseReadiness,
	}
}

func runReleaseReadiness(ctx *common.RuntimeContext) error {
	lang := normalizeLang(ctx.Arg("lang"))
	input, err := collectReleaseReadinessInput(ctx)
	if err != nil {
		return err
	}
	result := AnalyzeReleaseReadiness(input, lang)
	format := ctx.Format
	if strings.TrimSpace(cmdutil.Format) == "" {
		format = "table"
	}
	rendered, err := RenderReleaseReadiness(result, format, lang)
	if err != nil {
		return err
	}
	_, err = fmt.Fprint(os.Stdout, rendered)
	return err
}

func collectReleaseReadinessInput(ctx *common.RuntimeContext) (ReleaseReadinessInput, error) {
	if path := strings.TrimSpace(ctx.Arg("from")); path != "" {
		input, err := readReleaseReadinessInput(path)
		if err != nil {
			return ReleaseReadinessInput{}, err
		}
		if repo := strings.TrimSpace(ctx.Arg("repository")); repo != "" {
			input.Repository = repo
		}
		if input.Repository == "" {
			input.Repository = repositoryFromContext(ctx, "")
		}
		if input.Source == "" {
			input.Source = "local-json"
		}
		return input, nil
	}
	return ReleaseReadinessInput{
		Repository:      repositoryFromContext(ctx, strings.TrimSpace(ctx.Arg("repository"))),
		Version:         strings.TrimSpace(ctx.Arg("version")),
		TargetDate:      strings.TrimSpace(ctx.Arg("target-date")),
		Changes:         parseCSV(ctx.Arg("changes")),
		BreakingChanges: parseBoolArg(ctx.Arg("breaking-changes")),
		KnownBlockers:   parseCSV(ctx.Arg("known-blockers")),
		Tests:           parseReleaseCheckItems(ctx.Arg("tests")),
		Artifacts:       parseReleaseCheckItems(ctx.Arg("artifacts")),
		RollbackPlan:    strings.TrimSpace(ctx.Arg("rollback-plan")),
		DependencyRisk:  strings.TrimSpace(ctx.Arg("dependency-risk")),
		Source:          "local-flags",
	}, nil
}

func readReleaseReadinessInput(path string) (ReleaseReadinessInput, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return ReleaseReadinessInput{}, fmt.Errorf("read release readiness input: %w", err)
	}
	var input ReleaseReadinessInput
	if err := json.Unmarshal(data, &input); err != nil {
		return ReleaseReadinessInput{}, fmt.Errorf("parse release readiness input: %w", err)
	}
	return input, nil
}

func parseReleaseCheckItems(value string) []ReleaseCheckItem {
	parts := parseCSV(value)
	items := make([]ReleaseCheckItem, 0, len(parts))
	for _, part := range parts {
		name := part
		status := "unknown"
		if before, after, ok := strings.Cut(part, "="); ok {
			name = strings.TrimSpace(before)
			status = normalizeReleaseCheckStatus(after)
		}
		if name == "" {
			continue
		}
		items = append(items, ReleaseCheckItem{Name: name, Status: status})
	}
	return items
}

func normalizeReleaseCheckStatus(status string) string {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "pass", "passed", "ok", "success", "green":
		return "passed"
	case "fail", "failed", "error", "red":
		return "failed"
	case "missing", "todo", "none":
		return "missing"
	case "skip", "skipped":
		return "skipped"
	default:
		return "unknown"
	}
}

func AnalyzeReleaseReadiness(input ReleaseReadinessInput, lang string) ReleaseReadinessResult {
	lang = normalizeLang(lang)
	result := ReleaseReadinessResult{
		Repository: strings.TrimSpace(input.Repository),
		Version:    strings.TrimSpace(input.Version),
		TargetDate: strings.TrimSpace(input.TargetDate),
		Score:      100,
		Source:     strings.TrimSpace(input.Source),
	}
	if result.Repository == "" {
		result.Repository = "local"
	}
	if result.Source == "" {
		result.Source = "local"
	}
	if result.Version == "" {
		result.addReleaseFinding(lang, "high", "missing_version", "release version is missing")
		result.Score -= 20
	}
	if len(input.Changes) == 0 {
		result.addReleaseFinding(lang, "medium", "missing_changes", "release changes are not summarized")
		result.Score -= 12
	}
	if len(input.KnownBlockers) > 0 {
		for _, blocker := range input.KnownBlockers {
			result.addReleaseFinding(lang, "critical", "known_blocker", fmt.Sprintf("known blocker: %s", blocker))
		}
		result.Score -= 30
	}
	result.scoreReleaseChecks(lang, "test", input.Tests)
	result.scoreReleaseChecks(lang, "artifact", input.Artifacts)
	if input.BreakingChanges && strings.TrimSpace(input.RollbackPlan) == "" {
		result.addReleaseFinding(lang, "high", "missing_rollback_plan", "breaking changes require a rollback or mitigation plan")
		result.Score -= 15
	}
	switch strings.ToLower(strings.TrimSpace(input.DependencyRisk)) {
	case "high", "critical":
		result.addReleaseFinding(lang, "high", "dependency_risk", "dependency risk is high")
		result.Score -= 15
	case "medium":
		result.addReleaseFinding(lang, "medium", "dependency_risk", "dependency risk needs review")
		result.Score -= 8
	case "low":
		result.addReleaseFinding(lang, "low", "dependency_risk", "dependency risk is low")
		result.Score -= 3
	}
	if result.Score < 0 {
		result.Score = 0
	}
	result.Gate = releaseGate(result)
	result.Recommendations = releaseReadinessRecommendations(result, lang)
	return result
}

func (result *ReleaseReadinessResult) scoreReleaseChecks(lang, prefix string, checks []ReleaseCheckItem) {
	if len(checks) == 0 {
		result.addReleaseFinding(lang, "medium", "missing_"+prefix+"_checks", prefix+" checks are missing")
		result.MissingChecks++
		result.Score -= 10
		return
	}
	for _, check := range checks {
		switch normalizeReleaseCheckStatus(check.Status) {
		case "passed":
			result.PassedChecks++
		case "failed":
			result.FailedChecks++
			result.addReleaseFinding(lang, "critical", prefix+"_failed", fmt.Sprintf("%s failed: %s", prefix, check.Name))
			result.Score -= 25
		case "missing":
			result.MissingChecks++
			result.addReleaseFinding(lang, "high", prefix+"_missing", fmt.Sprintf("%s missing: %s", prefix, check.Name))
			result.Score -= 12
		case "skipped":
			result.WarningChecks++
			result.addReleaseFinding(lang, "medium", prefix+"_skipped", fmt.Sprintf("%s skipped: %s", prefix, check.Name))
			result.Score -= 8
		default:
			result.WarningChecks++
			result.addReleaseFinding(lang, "medium", prefix+"_unknown", fmt.Sprintf("%s status unknown: %s", prefix, check.Name))
			result.Score -= 8
		}
	}
}

func (result *ReleaseReadinessResult) addReleaseFinding(lang, severity, code, message string) {
	if normalizeLang(lang) == langZH {
		switch code {
		case "missing_version":
			message = "缺少发布版本号。"
		case "missing_changes":
			message = "缺少发布变更摘要。"
		case "known_blocker":
			message = strings.Replace(message, "known blocker:", "已知阻塞项：", 1)
		case "missing_rollback_plan":
			message = "包含破坏性变更时需要回滚或缓解方案。"
		case "dependency_risk":
			message = "依赖风险需要在发布前确认。"
		default:
			message = strings.ReplaceAll(message, "test", "测试")
			message = strings.ReplaceAll(message, "artifact", "制品")
			message = strings.ReplaceAll(message, "failed", "失败")
			message = strings.ReplaceAll(message, "missing", "缺失")
			message = strings.ReplaceAll(message, "skipped", "已跳过")
			message = strings.ReplaceAll(message, "status unknown", "状态未知")
		}
	}
	result.Findings = append(result.Findings, ReleaseReadinessFinding{
		Severity: severity,
		Code:     code,
		Message:  message,
	})
}

func releaseGate(result ReleaseReadinessResult) string {
	if result.FailedChecks > 0 || result.Score < 60 {
		return ReleaseGateBlock
	}
	for _, finding := range result.Findings {
		if finding.Severity == "critical" {
			return ReleaseGateBlock
		}
	}
	if result.Score < 85 || result.WarningChecks > 0 || result.MissingChecks > 0 {
		return ReleaseGateReview
	}
	return ReleaseGatePass
}

func releaseReadinessRecommendations(result ReleaseReadinessResult, lang string) []string {
	zh := normalizeLang(lang) == langZH
	switch result.Gate {
	case ReleaseGatePass:
		if zh {
			return []string{"发布门禁已通过，发布前保留测试日志和制品校验记录。"}
		}
		return []string{"Release gate passed; keep test logs and artifact verification records before publishing."}
	case ReleaseGateBlock:
		if zh {
			return []string{"先解决阻塞项或失败检查，再重新生成发布就绪度报告。"}
		}
		return []string{"Resolve blockers or failed checks first, then regenerate the readiness report."}
	default:
		if zh {
			return []string{"需要维护者复核缺失或未知检查项，确认后再推进发布。"}
		}
		return []string{"Maintainer review is required for missing or unknown checks before release."}
	}
}

func RenderReleaseReadiness(result ReleaseReadinessResult, format string, lang string) (string, error) {
	var buf bytes.Buffer
	switch normalizeFormat(format) {
	case "json":
		if err := writeJSON(&buf, result); err != nil {
			return "", err
		}
	case "markdown":
		if err := writeReleaseReadinessMarkdown(&buf, result, lang); err != nil {
			return "", err
		}
	case "table":
		if err := writeReleaseReadinessTable(&buf, result); err != nil {
			return "", err
		}
	default:
		return "", fmt.Errorf("unsupported workflow output format %q", format)
	}
	return buf.String(), nil
}

func writeReleaseReadinessMarkdown(buf *bytes.Buffer, result ReleaseReadinessResult, lang string) error {
	title := "Release Readiness"
	if normalizeLang(lang) == langZH {
		title = "发布就绪度检查"
	}
	if _, err := fmt.Fprintf(buf, "# %s\n\n", title); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(buf, "- Repository: `%s`\n- Version: `%s`\n- Gate: `%s`\n- Score: `%d`\n- Source: `%s`\n\n",
		result.Repository, releaseTextOrDash(result.Version), result.Gate, result.Score, result.Source); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(buf, "## Findings"); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(buf); err != nil {
		return err
	}
	if len(result.Findings) == 0 {
		if _, err := fmt.Fprintln(buf, "- No readiness findings."); err != nil {
			return err
		}
	} else {
		for _, finding := range result.Findings {
			if _, err := fmt.Fprintf(buf, "- `%s` `%s`: %s\n", finding.Severity, finding.Code, finding.Message); err != nil {
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
	for _, rec := range result.Recommendations {
		if _, err := fmt.Fprintf(buf, "- %s\n", rec); err != nil {
			return err
		}
	}
	return nil
}

func writeReleaseReadinessTable(buf *bytes.Buffer, result ReleaseReadinessResult) error {
	tw := tabwriter.NewWriter(buf, 0, 0, 2, ' ', 0)
	if _, err := fmt.Fprintln(tw, "VERSION\tGATE\tSCORE\tPASSED\tWARNINGS\tFAILED\tMISSING\tREPOSITORY"); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(tw, "%s\t%s\t%d\t%d\t%d\t%d\t%d\t%s\n",
		releaseTextOrDash(result.Version),
		result.Gate,
		result.Score,
		result.PassedChecks,
		result.WarningChecks,
		result.FailedChecks,
		result.MissingChecks,
		result.Repository,
	); err != nil {
		return err
	}
	if len(result.Findings) > 0 {
		if _, err := fmt.Fprintln(tw); err != nil {
			return err
		}
		if _, err := fmt.Fprintln(tw, "SEVERITY\tCODE\tMESSAGE"); err != nil {
			return err
		}
		for _, finding := range result.Findings {
			if _, err := fmt.Fprintf(tw, "%s\t%s\t%s\n", finding.Severity, finding.Code, finding.Message); err != nil {
				return err
			}
		}
	}
	return tw.Flush()
}

func releaseTextOrDash(value string) string {
	if strings.TrimSpace(value) == "" {
		return "-"
	}
	return value
}
