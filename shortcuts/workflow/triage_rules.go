package workflow

import (
	"fmt"
	"sort"
	"strings"
)

type keywordRule struct {
	issueType string
	keywords  []string
	weight    int
}

var triageKeywordRules = []keywordRule{
	{IssueTypeBug, []string{"crash", "error", "panic", "exception", "fail", "failed", "failure", "broken", "cannot", "bug", "报错", "崩溃", "异常", "失败", "无法", "不能", "问题"}, 10},
	{IssueTypeFeature, []string{"feature", "request", "support", "add", "implement", "enhancement", "功能", "支持", "增加", "建议", "增强"}, 8},
	{IssueTypeQuestion, []string{"how", "why", "help", "usage", "question", "如何", "怎么", "为什么", "请问", "求助"}, 7},
	{IssueTypeDocs, []string{"doc", "docs", "documentation", "readme", "typo", "example", "guide", "文档", "说明", "错别字", "示例", "教程"}, 8},
	{IssueTypeCI, []string{"ci", "build", "workflow", "action", "test failed", "pipeline", "构建", "测试失败", "流水线"}, 10},
	{IssueTypeSecurity, []string{"token", "leak", "leaked", "secret", "auth", "permission", "vulnerability", "cve", "泄露", "密钥", "权限", "漏洞", "安全"}, 12},
	{IssueTypePerformance, []string{"slow", "timeout", "latency", "memory", "cpu", "performance", "慢", "超时", "性能", "内存"}, 8},
	{IssueTypeRefactor, []string{"refactor", "cleanup", "simplify", "restructure", "重构", "清理", "简化", "结构调整"}, 6},
}

var typeTieOrder = []string{
	IssueTypeSecurity,
	IssueTypeBug,
	IssueTypeCI,
	IssueTypePerformance,
	IssueTypeFeature,
	IssueTypeDocs,
	IssueTypeQuestion,
	IssueTypeRefactor,
}

func AnalyzeIssue(input IssueInput, lang string) TriageResult {
	lang = normalizeLang(lang)
	text := normalizeIssueText(input)

	scores, matchedRules, reasoning := scoreIssueTypes(text, input.Labels)
	detectedType := chooseDetectedType(scores)
	riskFlags := detectRiskFlags(text, detectedType)
	priority, priorityReason := determinePriority(text, detectedType)
	if priorityReason != "" {
		matchedRules = append(matchedRules, priorityReason)
		reasoning = append(reasoning, priorityReason)
	}

	missingInformation := detectMissingInformation(text, detectedType, lang)
	if len(missingInformation) > 0 {
		riskFlags = append(riskFlags, RiskInsufficientInfo)
		matchedRules = append(matchedRules, "missing information detected")
		reasoning = append(reasoning, "missing information detected")
	}

	confidence := calculateConfidence(detectedType, scores, matchedRules, input.Labels)
	return TriageResult{
		Issue: IssueRef{
			ID:     input.ID,
			Number: input.Number,
			Title:  input.Title,
			URL:    input.URL,
			Author: input.Author,
			State:  input.State,
		},
		DetectedType:       detectedType,
		Priority:           priority,
		Confidence:         confidence,
		SuggestedLabels:    suggestedLabels(detectedType, priority, riskFlags),
		MissingInformation: missingInformation,
		RiskFlags:          uniqueStrings(riskFlags),
		RecommendedAction:  recommendedAction(detectedType, priority, riskFlags, missingInformation),
		SuggestedComment:   suggestedComment(lang, detectedType, missingInformation),
		Reasoning:          uniqueStrings(reasoning),
		MatchedRules:       uniqueStrings(matchedRules),
	}
}

func normalizeIssueText(input IssueInput) string {
	parts := []string{input.Title, input.Body}
	parts = append(parts, input.Labels...)
	return strings.ToLower(strings.Join(parts, "\n"))
}

func scoreIssueTypes(text string, labels []string) (map[string]int, []string, []string) {
	scores := make(map[string]int)
	matchedRules := []string{}
	reasoning := []string{}

	for _, rule := range triageKeywordRules {
		for _, keyword := range rule.keywords {
			if strings.Contains(text, strings.ToLower(keyword)) {
				scores[rule.issueType] += rule.weight
				ruleText := "matched keyword: " + keyword
				matchedRules = append(matchedRules, ruleText)
				reasoning = append(reasoning, fmt.Sprintf("%s -> %s", ruleText, rule.issueType))
			}
		}
	}

	for _, label := range labels {
		normalized := strings.ToLower(strings.TrimSpace(label))
		for _, issueType := range typeTieOrder {
			if normalized == issueType || strings.Contains(normalized, issueType) {
				scores[issueType] += 14
				ruleText := "label matched: " + normalized
				matchedRules = append(matchedRules, ruleText)
				reasoning = append(reasoning, fmt.Sprintf("%s -> %s", ruleText, issueType))
			}
		}
	}

	return scores, matchedRules, reasoning
}

func chooseDetectedType(scores map[string]int) string {
	bestType := IssueTypeUnknown
	bestScore := 0
	for _, issueType := range typeTieOrder {
		score := scores[issueType]
		if score > bestScore {
			bestType = issueType
			bestScore = score
		}
	}
	return bestType
}

func determinePriority(text string, detectedType string) (string, string) {
	if detectedType == IssueTypeSecurity || containsAny(text, []string{"token leak", "secret leak", "leaked token", "leaked secret", "auth bypass", "permission bypass", "vulnerability", "cve", "密钥泄露", "漏洞", "认证绕过", "权限绕过"}) {
		return PriorityP0, "priority rule: security sensitive token leak"
	}
	if containsAny(text, []string{"cannot login", "install failed", "installation failed", "core command unavailable", "command unavailable", "login failed", "crash", "panic", "无法登录", "安装失败", "核心命令不可用", "崩溃"}) {
		return PriorityP1, "priority rule: core blocker"
	}
	if detectedType == IssueTypeBug || detectedType == IssueTypeCI || detectedType == IssueTypePerformance {
		return PriorityP2, "priority rule: normal bug or operational failure"
	}
	if detectedType == IssueTypeFeature {
		return PriorityP2, "priority rule: feature request"
	}
	return PriorityP3, "priority rule: low risk request"
}

func calculateConfidence(detectedType string, scores map[string]int, matchedRules []string, labels []string) int {
	if detectedType == IssueTypeUnknown {
		return 20
	}

	confidence := 35 + scores[detectedType]
	if len(matchedRules) > 1 {
		confidence += minInt(len(matchedRules)*3, 18)
	}
	if len(labels) > 0 {
		confidence += 8
	}
	if detectedType == IssueTypeSecurity || detectedType == IssueTypeBug || detectedType == IssueTypeCI {
		confidence += 10
	}
	return clampInt(confidence, 0, 100)
}

func detectMissingInformation(text string, detectedType string, lang string) []string {
	if detectedType != IssueTypeBug {
		return nil
	}

	checks := []struct {
		key      string
		present  []string
		messageK string
	}{
		{"reproduction_steps", []string{"reproduction", "reproduce", "steps", "复现", "步骤"}, "missing_reproduction_steps"},
		{"expected_behavior", []string{"expected", "expect", "期望", "预期"}, "missing_expected_behavior"},
		{"actual_behavior", []string{"actual", "实际"}, "missing_actual_behavior"},
		{"version", []string{"version", "版本"}, "missing_version"},
		{"os_or_platform", []string{"os", "platform", "windows", "linux", "macos", "darwin", "系统", "平台"}, "missing_os_or_platform"},
		{"command_output_or_logs", []string{"output", "log", "trace", "stdout", "stderr", "输出", "日志"}, "missing_command_output_or_logs"},
	}

	missing := []string{}
	for _, check := range checks {
		if !containsAny(text, check.present) {
			if normalizeLang(lang) == langZH {
				missing = append(missing, message(lang, check.messageK))
			} else {
				missing = append(missing, check.key)
			}
		}
	}
	return missing
}

func detectRiskFlags(text string, detectedType string) []string {
	flags := []string{}
	if detectedType == IssueTypeSecurity || containsAny(text, []string{"vulnerability", "cve", "漏洞", "安全", "auth bypass", "permission bypass", "认证绕过", "权限绕过"}) {
		flags = append(flags, RiskSecuritySensitive)
	}
	if containsAny(text, []string{"token leak", "secret leak", "leaked token", "leaked secret", "token leaked", "secret leaked", "密钥泄露", "泄露"}) {
		flags = append(flags, RiskPossibleSecretLeak)
	}
	if containsAny(text, []string{"install failed", "installation failed", "安装失败"}) {
		flags = append(flags, RiskInstallationBlocker)
	}
	if containsAny(text, []string{"cannot login", "login failed", "auth failed", "无法登录", "登录失败"}) {
		flags = append(flags, RiskAuthenticationBlocker)
	}
	if detectedType == IssueTypeCI || containsAny(text, []string{"test failed", "pipeline failed", "build failed", "测试失败", "构建失败"}) {
		flags = append(flags, RiskCIBlocker)
	}
	return flags
}

func suggestedLabels(detectedType string, priority string, riskFlags []string) []string {
	labels := []string{}
	if detectedType != IssueTypeUnknown {
		labels = append(labels, detectedType)
	}
	labels = append(labels, strings.ToLower(priority))
	for _, flag := range riskFlags {
		switch flag {
		case RiskSecuritySensitive, RiskPossibleSecretLeak:
			labels = append(labels, "security")
		case RiskCIBlocker:
			labels = append(labels, "ci")
		}
	}
	return uniqueStrings(labels)
}

func recommendedAction(detectedType string, priority string, riskFlags []string, missingInformation []string) string {
	if containsString(riskFlags, RiskSecuritySensitive) || containsString(riskFlags, RiskPossibleSecretLeak) || detectedType == IssueTypeSecurity {
		return ActionReviewSecurity
	}
	if priority == PriorityP0 {
		return ActionPrioritizeImmediate
	}
	if len(missingInformation) > 0 {
		return ActionRequestMoreInfo
	}
	switch detectedType {
	case IssueTypeQuestion:
		return ActionConvertToDiscussion
	case IssueTypeDocs:
		return ActionUpdateDocs
	case IssueTypeCI:
		return ActionInvestigateCI
	default:
		return ActionScheduleFix
	}
}

func suggestedComment(lang string, detectedType string, missingInformation []string) string {
	lang = normalizeLang(lang)
	if detectedType == IssueTypeSecurity {
		return message(lang, "comment_security")
	}
	if len(missingInformation) > 0 {
		return fmt.Sprintf(message(lang, "comment_more_info"), strings.Join(missingInformation, ", "))
	}
	if detectedType == IssueTypeDocs {
		return message(lang, "comment_docs")
	}
	return message(lang, "comment_default")
}

func containsAny(text string, keywords []string) bool {
	for _, keyword := range keywords {
		if strings.Contains(text, strings.ToLower(keyword)) {
			return true
		}
	}
	return false
}

func containsString(values []string, needle string) bool {
	for _, value := range values {
		if value == needle {
			return true
		}
	}
	return false
}

func uniqueStrings(values []string) []string {
	seen := map[string]struct{}{}
	unique := []string{}
	for _, value := range values {
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		unique = append(unique, value)
	}
	return unique
}

func sortedStrings(values []string) []string {
	copied := append([]string(nil), values...)
	sort.Strings(copied)
	return copied
}
