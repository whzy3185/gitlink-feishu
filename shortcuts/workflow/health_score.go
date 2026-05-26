package workflow

func ScoreHealth(input HealthInput, lang string) HealthResult {
	lang = normalizeLang(lang)

	metrics := []HealthMetric{}
	notes := []ScoringNote{}
	recommendations := []string{}

	issueMetric := scoreIssueBacklog(input, lang)
	metrics = append(metrics, issueMetric)
	if issueMetric.Score < issueMetric.MaxScore {
		recommendations = append(recommendations, message(lang, "rec_reduce_issues"))
	}

	prMetric := scorePRBacklog(input, lang)
	metrics = append(metrics, prMetric)
	if prMetric.Score < prMetric.MaxScore {
		recommendations = append(recommendations, message(lang, "rec_reduce_prs"))
	}

	recentMetric, recentNote := scoreRecentActivity(input, lang)
	metrics = append(metrics, recentMetric)
	if recentNote.Note != "" {
		notes = append(notes, recentNote)
	}
	if recentMetric.Score < recentMetric.MaxScore/2 {
		recommendations = append(recommendations, message(lang, "rec_restore_activity"))
	}

	releaseMetric, releaseNote := scoreReleaseStatus(input, lang)
	metrics = append(metrics, releaseMetric)
	if releaseNote.Note != "" {
		notes = append(notes, releaseNote)
	}
	if releaseMetric.Score < releaseMetric.MaxScore/2 {
		recommendations = append(recommendations, message(lang, "rec_release"))
	}

	docMetric := scoreDocumentation(input, lang)
	metrics = append(metrics, docMetric)
	if docMetric.Score < docMetric.MaxScore {
		recommendations = append(recommendations, message(lang, "rec_docs"))
	}

	licenseMetric := scoreLicenseAndContributing(input, lang)
	metrics = append(metrics, licenseMetric)
	if licenseMetric.Score < licenseMetric.MaxScore {
		recommendations = append(recommendations, message(lang, "rec_license"))
	}

	agentMetric, agentNote := scoreAgentReadiness(input, lang)
	metrics = append(metrics, agentMetric)
	if agentNote.Note != "" {
		notes = append(notes, agentNote)
	}
	if agentMetric.Score < agentMetric.MaxScore/2 {
		recommendations = append(recommendations, message(lang, "rec_agent"))
	}

	ciMetric, ciNote := scoreCIStatus(input, lang)
	metrics = append(metrics, ciMetric)
	if ciNote.Note != "" {
		notes = append(notes, ciNote)
	}

	total := 0
	maxTotal := 0
	for _, metric := range metrics {
		total += metric.Score
		maxTotal += metric.MaxScore
	}

	healthScore := 0
	if maxTotal > 0 {
		healthScore = clampInt(total*100/maxTotal, 0, 100)
	}
	if len(recommendations) == 0 {
		recommendations = append(recommendations, message(lang, "rec_maintain"))
	}

	return HealthResult{
		Repository:      input.Repository,
		HealthScore:     healthScore,
		RiskLevel:       riskLevel(healthScore),
		Metrics:         metrics,
		Recommendations: uniqueStrings(recommendations),
		ScoringNotes:    notes,
	}
}

func scoreIssueBacklog(input HealthInput, lang string) HealthMetric {
	score := 20
	score -= minInt(input.StaleIssues*3, 14)
	if input.OpenIssues > 50 {
		score -= 8
	} else if input.OpenIssues > 20 {
		score -= 5
	} else if input.OpenIssues > 10 {
		score -= 2
	}
	score = clampInt(score, 0, 20)

	status := "good"
	reason := message(lang, "health_issue_backlog_good")
	if score < 14 {
		status = "attention"
		reason = message(lang, "health_issue_backlog_attention")
	}
	return HealthMetric{Name: "issue_backlog_and_response", Status: status, Score: score, MaxScore: 20, Reason: reason}
}

func scorePRBacklog(input HealthInput, lang string) HealthMetric {
	score := 20
	score -= minInt(input.StalePRs*5, 15)
	if input.OpenPRs > 20 {
		score -= 8
	} else if input.OpenPRs > 10 {
		score -= 5
	} else if input.OpenPRs > 5 {
		score -= 2
	}
	score = clampInt(score, 0, 20)

	status := "good"
	reason := message(lang, "health_pr_backlog_good")
	if score < 14 {
		status = "attention"
		reason = message(lang, "health_pr_backlog_attention")
	}
	return HealthMetric{Name: "pr_backlog_and_merge_state", Status: status, Score: score, MaxScore: 20, Reason: reason}
}

func scoreRecentActivity(input HealthInput, lang string) (HealthMetric, ScoringNote) {
	if !input.RecentActivityKnown {
		return HealthMetric{Name: "recent_activity", Status: "unknown", Score: 8, MaxScore: 15, Reason: message(lang, "health_recent_unknown")}, ScoringNote{Metric: "recent_activity", Note: message(lang, "health_recent_unknown")}
	}
	if input.RecentActivityDays <= 7 {
		return HealthMetric{Name: "recent_activity", Status: "good", Score: 15, MaxScore: 15, Reason: "recent activity within 7 days"}, ScoringNote{}
	}
	if input.RecentActivityDays <= 30 {
		return HealthMetric{Name: "recent_activity", Status: "attention", Score: 10, MaxScore: 15, Reason: "recent activity within 30 days"}, ScoringNote{}
	}
	if input.RecentActivityDays <= 90 {
		return HealthMetric{Name: "recent_activity", Status: "attention", Score: 6, MaxScore: 15, Reason: "recent activity older than 30 days"}, ScoringNote{}
	}
	return HealthMetric{Name: "recent_activity", Status: "risk", Score: 2, MaxScore: 15, Reason: "recent activity older than 90 days"}, ScoringNote{}
}

func scoreReleaseStatus(input HealthInput, lang string) (HealthMetric, ScoringNote) {
	if !input.ReleaseKnown {
		return HealthMetric{Name: "release_status", Status: "unknown", Score: 8, MaxScore: 15, Reason: message(lang, "health_release_unknown")}, ScoringNote{Metric: "release_status", Note: message(lang, "health_release_unknown")}
	}
	if input.HasRecentRelease {
		return HealthMetric{Name: "release_status", Status: "good", Score: 15, MaxScore: 15, Reason: "recent release found"}, ScoringNote{}
	}
	return HealthMetric{Name: "release_status", Status: "risk", Score: 4, MaxScore: 15, Reason: "no recent release found"}, ScoringNote{}
}

func scoreDocumentation(input HealthInput, lang string) HealthMetric {
	score := 0
	if input.HasReadme {
		score += 7
	}
	if input.HasContributing {
		score += 3
	}
	status := "attention"
	reason := message(lang, "rec_docs")
	if score == 10 {
		status = "good"
		reason = "README and contribution guidance are present"
	}
	return HealthMetric{Name: "documentation", Status: status, Score: score, MaxScore: 10, Reason: reason}
}

func scoreLicenseAndContributing(input HealthInput, lang string) HealthMetric {
	score := 0
	if input.HasLicense {
		score += 6
	}
	if input.HasContributing {
		score += 4
	}
	status := "attention"
	reason := message(lang, "rec_license")
	if score == 10 {
		status = "good"
		reason = "LICENSE and CONTRIBUTING are present"
	}
	return HealthMetric{Name: "license_and_contributing", Status: status, Score: score, MaxScore: 10, Reason: reason}
}

func scoreAgentReadiness(input HealthInput, lang string) (HealthMetric, ScoringNote) {
	if !input.AgentReadinessKnown {
		return HealthMetric{Name: "agent_readiness", Status: "unknown", Score: 5, MaxScore: 10, Reason: message(lang, "health_agent_unknown")}, ScoringNote{Metric: "agent_readiness", Note: message(lang, "health_agent_unknown")}
	}
	score := clampInt(input.AgentReadinessScore, 0, 10)
	status := "attention"
	if score >= 8 {
		status = "good"
	} else if score < 4 {
		status = "risk"
	}
	return HealthMetric{Name: "agent_readiness", Status: status, Score: score, MaxScore: 10, Reason: "agent readiness score provided"}, ScoringNote{}
}

func scoreCIStatus(input HealthInput, lang string) (HealthMetric, ScoringNote) {
	if !input.CIKnown {
		return HealthMetric{Name: "ci_status", Status: "unknown", Score: 0, MaxScore: 0, Reason: message(lang, "health_ci_unknown")}, ScoringNote{Metric: "ci_status", Note: message(lang, "health_ci_unknown")}
	}
	if input.CIPassing {
		return HealthMetric{Name: "ci_status", Status: "good", Score: 0, MaxScore: 0, Reason: "CI status is passing"}, ScoringNote{}
	}
	return HealthMetric{Name: "ci_status", Status: "risk", Score: 0, MaxScore: 0, Reason: "CI status is failing"}, ScoringNote{}
}

func riskLevel(score int) string {
	switch {
	case score >= 85:
		return "low"
	case score >= 65:
		return "medium"
	case score >= 40:
		return "high"
	default:
		return "critical"
	}
}

func clampInt(value int, minValue int, maxValue int) int {
	if value < minValue {
		return minValue
	}
	if value > maxValue {
		return maxValue
	}
	return value
}

func minInt(a int, b int) int {
	if a < b {
		return a
	}
	return b
}
