package workflow

import (
	"testing"
)

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

func TestDeterminePriority(t *testing.T) {
	tests := []struct {
		name         string
		text         string
		detectedType string
		wantPriority string
	}{
		{"security type", "something", IssueTypeSecurity, PriorityP0},
		{"token leak in text", "token leak found", IssueTypeBug, PriorityP0},
		{"secret leak", "a secret leak happened", IssueTypeBug, PriorityP0},
		{"vulnerability", "vulnerability in parse", IssueTypeBug, PriorityP0},
		{"chinese auth bypass", "认证绕过", IssueTypeBug, PriorityP0},
		{"crash", "the cli crash on start", IssueTypeBug, PriorityP1},
		{"panic", "panic at runtime", IssueTypeBug, PriorityP1},
		{"install failed", "install failed on windows", IssueTypeBug, PriorityP1},
		{"cannot login", "cannot login with token", IssueTypeBug, PriorityP1},
		{"chinese install failed", "安装失败", IssueTypeBug, PriorityP1},
		{"chinese login fail", "无法登录", IssueTypeBug, PriorityP1},
		{"chinese crash", "崩溃", IssueTypeBug, PriorityP1},
		{"bug type", "some text", IssueTypeBug, PriorityP2},
		{"ci type", "ci failure", IssueTypeCI, PriorityP2},
		{"performance", "slow response", IssueTypePerformance, PriorityP2},
		{"feature", "new feature request", IssueTypeFeature, PriorityP2},
		{"default", "random note", IssueTypeQuestion, PriorityP3},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, _ := determinePriority(tt.text, tt.detectedType)
			if got != tt.wantPriority {
				t.Fatalf("determinePriority = %q, want %q", got, tt.wantPriority)
			}
		})
	}
}

func TestRecommendedAction(t *testing.T) {
	tests := []struct {
		name               string
		detectedType       string
		priority           string
		riskFlags          []string
		missingInformation []string
		want               string
	}{
		{"security flag", IssueTypeBug, PriorityP2, []string{RiskSecuritySensitive}, nil, ActionReviewSecurity},
		{"secret leak flag", IssueTypeBug, PriorityP2, []string{RiskPossibleSecretLeak}, nil, ActionReviewSecurity},
		{"security type", IssueTypeSecurity, PriorityP2, nil, nil, ActionReviewSecurity},
		{"p0 immediate", IssueTypeBug, PriorityP0, nil, nil, ActionPrioritizeImmediate},
		{"request more info", IssueTypeBug, PriorityP1, nil, []string{"version"}, ActionRequestMoreInfo},
		{"question convert", IssueTypeQuestion, PriorityP3, nil, nil, ActionConvertToDiscussion},
		{"docs update", IssueTypeDocs, PriorityP3, nil, nil, ActionUpdateDocs},
		{"ci investigate", IssueTypeCI, PriorityP2, nil, nil, ActionInvestigateCI},
		{"default schedule fix", IssueTypeBug, PriorityP2, nil, nil, ActionScheduleFix},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := recommendedAction(tt.detectedType, tt.priority, tt.riskFlags, tt.missingInformation)
			if got != tt.want {
				t.Fatalf("recommendedAction = %q, want %q", got, tt.want)
			}
		})
	}
}

