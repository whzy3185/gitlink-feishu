package workflow

import (
	"strings"
	"testing"
)

func TestScoreHealthLowRisk(t *testing.T) {
	result := ScoreHealth(HealthInput{
		Repository:          "Gitlink/gitlink-cli",
		OpenIssues:          4,
		OpenPRs:             2,
		StaleIssues:         1,
		StalePRs:            0,
		RecentActivityKnown: true,
		RecentActivityDays:  3,
		ReleaseKnown:        true,
		HasRecentRelease:    true,
		CIKnown:             true,
		CIPassing:           true,
		HasReadme:           true,
		HasLicense:          true,
		HasContributing:     true,
		AgentReadinessKnown: true,
		AgentReadinessScore: 9,
	}, "en")

	if result.HealthScore < 85 {
		t.Fatalf("HealthScore = %d, want >= 85", result.HealthScore)
	}
	if result.RiskLevel != "low" {
		t.Fatalf("RiskLevel = %q, want low", result.RiskLevel)
	}
}

func TestScoreHealthHighRisk(t *testing.T) {
	result := ScoreHealth(HealthInput{
		Repository:          "Gitlink/gitlink-cli",
		OpenIssues:          80,
		OpenPRs:             25,
		StaleIssues:         20,
		StalePRs:            10,
		RecentActivityKnown: true,
		RecentActivityDays:  120,
		ReleaseKnown:        true,
		HasRecentRelease:    false,
		CIKnown:             true,
		CIPassing:           false,
		HasReadme:           false,
		HasLicense:          false,
		HasContributing:     false,
		AgentReadinessKnown: true,
		AgentReadinessScore: 2,
	}, "en")

	if result.HealthScore >= 65 {
		t.Fatalf("HealthScore = %d, want < 65", result.HealthScore)
	}
	if result.RiskLevel != "high" && result.RiskLevel != "critical" {
		t.Fatalf("RiskLevel = %q, want high or critical", result.RiskLevel)
	}
	if len(result.Recommendations) == 0 {
		t.Fatal("Recommendations is empty")
	}
}

func TestScoreHealthUnknownMetrics(t *testing.T) {
	result := ScoreHealth(HealthInput{
		Repository:          "Gitlink/gitlink-cli",
		RecentActivityKnown: false,
		ReleaseKnown:        false,
		CIKnown:             false,
		HasReadme:           true,
		HasLicense:          true,
		HasContributing:     false,
		AgentReadinessKnown: false,
	}, "en")

	if len(result.ScoringNotes) == 0 {
		t.Fatal("ScoringNotes is empty")
	}
	if result.HealthScore < 0 || result.HealthScore > 100 {
		t.Fatalf("HealthScore = %d, want between 0 and 100", result.HealthScore)
	}
}

func TestScoreHealthChinese(t *testing.T) {
	result := ScoreHealth(HealthInput{
		Repository:          "Gitlink/gitlink-cli",
		OpenIssues:          40,
		OpenPRs:             12,
		StaleIssues:         8,
		StalePRs:            4,
		RecentActivityKnown: true,
		RecentActivityDays:  45,
		ReleaseKnown:        true,
		HasRecentRelease:    false,
		CIKnown:             false,
		HasReadme:           false,
		HasLicense:          false,
		HasContributing:     false,
		AgentReadinessKnown: true,
		AgentReadinessScore: 3,
	}, "zh-CN")

	if len(result.Recommendations) == 0 {
		t.Fatal("Recommendations is empty")
	}
	joined := strings.Join(result.Recommendations, "")
	if !strings.Contains(joined, "建议") && !strings.Contains(joined, "补充") && !strings.Contains(joined, "减少") {
		t.Fatalf("Recommendations = %v, want Chinese content", result.Recommendations)
	}
}
