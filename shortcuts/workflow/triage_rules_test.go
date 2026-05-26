package workflow

import "testing"

func TestAnalyzeIssueDetectsSecurityP0(t *testing.T) {
	result := AnalyzeIssue(IssueInput{
		Title:  "Token leaked in command output",
		Body:   "A secret token leaked and may allow permission escalation.",
		Labels: []string{"security"},
	}, "en")

	if result.DetectedType != IssueTypeSecurity {
		t.Fatalf("DetectedType = %q, want %q", result.DetectedType, IssueTypeSecurity)
	}
	if result.Priority != PriorityP0 {
		t.Fatalf("Priority = %q, want %q", result.Priority, PriorityP0)
	}
	if !containsString(result.RiskFlags, RiskPossibleSecretLeak) && !containsString(result.RiskFlags, RiskSecuritySensitive) {
		t.Fatalf("RiskFlags = %v, want security or secret leak flag", result.RiskFlags)
	}
	if result.Confidence < 70 {
		t.Fatalf("Confidence = %d, want >= 70", result.Confidence)
	}
}

func TestAnalyzeIssueDetectsBugAndMissingInfo(t *testing.T) {
	result := AnalyzeIssue(IssueInput{
		Title: "CLI crash with error",
		Body:  "It crashes.",
	}, "en")

	if result.DetectedType != IssueTypeBug {
		t.Fatalf("DetectedType = %q, want %q", result.DetectedType, IssueTypeBug)
	}
	if result.Priority != PriorityP1 && result.Priority != PriorityP2 {
		t.Fatalf("Priority = %q, want P1 or P2", result.Priority)
	}
	if len(result.MissingInformation) == 0 {
		t.Fatal("MissingInformation is empty, want bug info requirements")
	}
	if !containsString(result.RiskFlags, RiskInsufficientInfo) {
		t.Fatalf("RiskFlags = %v, want %q", result.RiskFlags, RiskInsufficientInfo)
	}
}

func TestAnalyzeIssueDetectsDocs(t *testing.T) {
	result := AnalyzeIssue(IssueInput{
		Title: "README typo in documentation example",
		Body:  "The docs guide has a typo.",
	}, "en")

	if result.DetectedType != IssueTypeDocs {
		t.Fatalf("DetectedType = %q, want %q", result.DetectedType, IssueTypeDocs)
	}
	if result.Priority != PriorityP3 {
		t.Fatalf("Priority = %q, want %q", result.Priority, PriorityP3)
	}
	if result.RecommendedAction != ActionUpdateDocs {
		t.Fatalf("RecommendedAction = %q, want %q", result.RecommendedAction, ActionUpdateDocs)
	}
}

func TestAnalyzeIssueChinese(t *testing.T) {
	result := AnalyzeIssue(IssueInput{
		Title: "安装失败并且报错，无法登录",
		Body:  "执行登录命令后失败。",
	}, "zh-CN")

	if result.DetectedType != IssueTypeBug {
		t.Fatalf("DetectedType = %q, want %q", result.DetectedType, IssueTypeBug)
	}
	if result.Priority != PriorityP1 {
		t.Fatalf("Priority = %q, want %q", result.Priority, PriorityP1)
	}
	if result.SuggestedComment == "" {
		t.Fatal("SuggestedComment is empty")
	}
}

func TestAnalyzeIssueUnknownLowConfidence(t *testing.T) {
	result := AnalyzeIssue(IssueInput{
		Title: "General repository note",
		Body:  "This is a neutral note without clear maintenance signal.",
	}, "en")

	if result.DetectedType != IssueTypeUnknown {
		t.Fatalf("DetectedType = %q, want %q", result.DetectedType, IssueTypeUnknown)
	}
	if result.Confidence > 40 {
		t.Fatalf("Confidence = %d, want <= 40", result.Confidence)
	}
}
