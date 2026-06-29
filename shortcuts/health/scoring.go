package health

import (
	"encoding/base64"
	"strconv"
	"strings"

	"github.com/gitlink-org/gitlink-cli/internal/i18n"
)

// calculateHealthScore calculates the overall health score for a project.
func calculateHealthScore(owner, repo string, data *DiagnosisData, tr *i18n.Translator) *HealthReport {
	report := &HealthReport{
		Owner:      owner,
		Repo:       repo,
		MaxScore:   100,
		Categories: make([]CategoryScore, 0, 5),
	}

	// 1. Documentation (20 points)
	docScore := scoreDocumentation(data, tr)
	report.Categories = append(report.Categories, docScore)

	// 2. License (15 points)
	licenseScore := scoreLicense(data, tr)
	report.Categories = append(report.Categories, licenseScore)

	// 3. Community (25 points)
	communityScore := scoreCommunity(data, tr)
	report.Categories = append(report.Categories, communityScore)

	// 4. Maturity (20 points)
	maturityScore := scoreMaturity(data, tr)
	report.Categories = append(report.Categories, maturityScore)

	// 5. CI/CD (20 points)
	ciScore := scoreCI(data, tr)
	report.Categories = append(report.Categories, ciScore)

	// Calculate total score
	for _, cat := range report.Categories {
		report.TotalScore += cat.Score
	}

	return report
}

// scoreDocumentation evaluates documentation completeness (20 points).
func scoreDocumentation(data *DiagnosisData, tr *i18n.Translator) CategoryScore {
	cat := CategoryScore{
		Name:     tr.T("output.category_documentation"),
		MaxScore: 20,
		Details:  make([]string, 0),
	}

	score := 0

	// Check README existence and content (15 points)
	if data.Readme != nil {
		content := getReadmeContent(data.Readme)
		if content != "" {
			// README exists
			score += 5
			cat.Details = append(cat.Details, tr.T("output.doc_readme_found"))

			// Check README quality (length and sections)
			readmeScore := evaluateReadmeQuality(content, tr)
			score += readmeScore

			if readmeScore >= 8 {
				cat.Details = append(cat.Details, tr.T("output.doc_readme_comprehensive"))
			} else if readmeScore >= 5 {
				cat.Details = append(cat.Details, tr.T("output.doc_readme_basic"))
			} else {
				cat.Details = append(cat.Details, tr.T("output.doc_readme_minimal"))
			}
		} else {
			cat.Details = append(cat.Details, tr.T("output.doc_readme_empty"))
		}
	} else {
		cat.Details = append(cat.Details, tr.T("output.doc_readme_missing"))
	}

	// Check project description (5 points)
	if data.Detail != nil {
		if desc, ok := data.Detail["description"].(string); ok && desc != "" {
			score += 5
			cat.Details = append(cat.Details, tr.T("output.doc_description_found"))
		} else {
			cat.Details = append(cat.Details, tr.T("output.doc_description_missing"))
		}
	}

	cat.Score = min(score, 20)
	cat.Status = getCategoryStatus(cat.Score, cat.MaxScore)

	return cat
}

// evaluateReadmeQuality evaluates README content quality.
func evaluateReadmeQuality(content string, tr *i18n.Translator) int {
	score := 0
	contentLower := strings.ToLower(content)

	// Check for common sections (2 points each, max 10)
	sections := []string{
		"installation", "install", "usage", "getting started", "quick start",
		"contributing", "contribute", "license", "api", "documentation",
		"features", "requirements", "dependencies", "build", "test",
	}

	foundSections := 0
	for _, section := range sections {
		if strings.Contains(contentLower, section) {
			foundSections++
		}
	}

	// Score based on number of sections found
	if foundSections >= 5 {
		score = 10
	} else if foundSections >= 3 {
		score = 7
	} else if foundSections >= 1 {
		score = 4
	}

	return score
}

// scoreLicense evaluates license compliance (15 points).
func scoreLicense(data *DiagnosisData, tr *i18n.Translator) CategoryScore {
	cat := CategoryScore{
		Name:     tr.T("output.category_license"),
		MaxScore: 15,
		Details:  make([]string, 0),
	}

	score := 0

	if data.Detail != nil {
		if licenseName, ok := data.Detail["license_name"].(string); ok && licenseName != "" {
			// Has a license
			score += 10
			cat.Details = append(cat.Details, tr.T("output.license_found")+": "+licenseName)

			// Check if it's a well-known license
			if isWellKnownLicense(licenseName) {
				score += 5
				cat.Details = append(cat.Details, tr.T("output.license_well_known"))
			} else {
				cat.Details = append(cat.Details, tr.T("output.license_custom"))
			}
		} else {
			cat.Details = append(cat.Details, tr.T("output.license_missing"))
		}
	} else {
		cat.Details = append(cat.Details, tr.T("output.license_unknown"))
	}

	cat.Score = min(score, 15)
	cat.Status = getCategoryStatus(cat.Score, cat.MaxScore)

	return cat
}

// isWellKnownLicense checks if the license is a well-known open source license.
func isWellKnownLicense(name string) bool {
	nameLower := strings.ToLower(name)
	wellKnown := []string{
		"mit", "apache", "gpl", "lgpl", "bsd", "mpl",
		"creative commons", "cc0", "isc", "eclipse public",
	}

	for _, license := range wellKnown {
		if strings.Contains(nameLower, license) {
			return true
		}
	}
	return false
}

// scoreCommunity evaluates community activity (25 points).
func scoreCommunity(data *DiagnosisData, tr *i18n.Translator) CategoryScore {
	cat := CategoryScore{
		Name:     tr.T("output.category_community"),
		MaxScore: 25,
		Details:  make([]string, 0),
	}

	score := 0

	if data.Detail != nil {
		// Contributors (10 points)
		contributorCount := len(data.Contributors)
		if contributorCount >= 10 {
			score += 10
			cat.Details = append(cat.Details, tr.T("output.community_contributors_many"))
		} else if contributorCount >= 5 {
			score += 7
			cat.Details = append(cat.Details, tr.T("output.community_contributors_good"))
		} else if contributorCount >= 2 {
			score += 4
			cat.Details = append(cat.Details, tr.T("output.community_contributors_few"))
		} else {
			cat.Details = append(cat.Details, tr.T("output.community_contributors_solo"))
		}

		// Stars/Forks (8 points)
		praisesCount := getIntFromMap(data.Detail, "praises_count")
		forkedCount := getIntFromMap(data.Detail, "forked_count")

		if praisesCount >= 100 || forkedCount >= 50 {
			score += 8
			cat.Details = append(cat.Details, tr.T("output.community_popular"))
		} else if praisesCount >= 20 || forkedCount >= 10 {
			score += 5
			cat.Details = append(cat.Details, tr.T("output.community_moderate"))
		} else {
			cat.Details = append(cat.Details, tr.T("output.community_growing"))
		}

		// Issues (7 points)
		issuesCount := getIntFromMap(data.Detail, "issues_count")
		if issuesCount > 0 {
			// Having issues indicates community engagement
			if issuesCount >= 50 {
				score += 7
				cat.Details = append(cat.Details, tr.T("output.community_issues_active"))
			} else if issuesCount >= 10 {
				score += 5
				cat.Details = append(cat.Details, tr.T("output.community_issues_moderate"))
			} else {
				score += 3
				cat.Details = append(cat.Details, tr.T("output.community_issues_few"))
			}
		} else {
			cat.Details = append(cat.Details, tr.T("output.community_issues_none"))
		}
	}

	cat.Score = min(score, 25)
	cat.Status = getCategoryStatus(cat.Score, cat.MaxScore)

	return cat
}

// scoreMaturity evaluates project maturity (20 points).
func scoreMaturity(data *DiagnosisData, tr *i18n.Translator) CategoryScore {
	cat := CategoryScore{
		Name:     tr.T("output.category_maturity"),
		MaxScore: 20,
		Details:  make([]string, 0),
	}

	score := 0

	if data.Detail != nil {
		// Branches (8 points)
		branchesCount := getIntFromMap(data.Detail, "branches_count")
		if branchesCount >= 3 {
			score += 8
			cat.Details = append(cat.Details, tr.T("output.maturity_branches_good"))
		} else if branchesCount >= 2 {
			score += 5
			cat.Details = append(cat.Details, tr.T("output.maturity_branches_basic"))
		} else {
			cat.Details = append(cat.Details, tr.T("output.maturity_branches_minimal"))
		}

		// Tags/Releases (7 points)
		tagsCount := getIntFromMap(data.Detail, "tags_count")
		if tagsCount >= 5 {
			score += 7
			cat.Details = append(cat.Details, tr.T("output.maturity_releases_many"))
		} else if tagsCount >= 2 {
			score += 4
			cat.Details = append(cat.Details, tr.T("output.maturity_releases_some"))
		} else if tagsCount == 1 {
			score += 2
			cat.Details = append(cat.Details, tr.T("output.maturity_releases_initial"))
		} else {
			cat.Details = append(cat.Details, tr.T("output.maturity_releases_none"))
		}

		// Pull Requests (5 points)
		prCount := getIntFromMap(data.Detail, "pull_requests_count")
		if prCount >= 10 {
			score += 5
			cat.Details = append(cat.Details, tr.T("output.maturity_pr_active"))
		} else if prCount >= 3 {
			score += 3
			cat.Details = append(cat.Details, tr.T("output.maturity_pr_some"))
		} else {
			cat.Details = append(cat.Details, tr.T("output.maturity_pr_few"))
		}
	}

	cat.Score = min(score, 20)
	cat.Status = getCategoryStatus(cat.Score, cat.MaxScore)

	return cat
}

// scoreCI evaluates CI/CD configuration (20 points).
func scoreCI(data *DiagnosisData, tr *i18n.Translator) CategoryScore {
	cat := CategoryScore{
		Name:     tr.T("output.category_ci"),
		MaxScore: 20,
		Details:  make([]string, 0),
	}

	score := 0

	if len(data.Builds) > 0 {
		// CI is configured
		score += 10
		cat.Details = append(cat.Details, tr.T("output.ci_configured"))

		// Check build success rate
		successCount := 0
		for _, build := range data.Builds {
			if b, ok := build.(map[string]interface{}); ok {
				if status, ok := b["status"].(string); ok {
					if status == "success" || status == "completed" {
						successCount++
					}
				}
			}
		}

		totalBuilds := len(data.Builds)
		if totalBuilds > 0 {
			successRate := float64(successCount) / float64(totalBuilds)
			if successRate >= 0.8 {
				score += 10
				cat.Details = append(cat.Details, tr.T("output.ci_stable"))
			} else if successRate >= 0.5 {
				score += 5
				cat.Details = append(cat.Details, tr.T("output.ci_moderate"))
			} else {
				cat.Details = append(cat.Details, tr.T("output.ci_unstable"))
			}
		}
	} else {
		cat.Details = append(cat.Details, tr.T("output.ci_not_configured"))
	}

	cat.Score = min(score, 20)
	cat.Status = getCategoryStatus(cat.Score, cat.MaxScore)

	return cat
}

// getReadmeContent extracts README content from API response.
func getReadmeContent(readme map[string]interface{}) string {
	if content, ok := readme["content"].(string); ok {
		// Try to decode base64
		decoded, err := base64.StdEncoding.DecodeString(content)
		if err == nil {
			return string(decoded)
		}
		// If not base64, return as is
		return content
	}
	return ""
}

// getIntFromMap safely extracts an integer from a map.
func getIntFromMap(m map[string]interface{}, key string) int {
	if val, ok := m[key]; ok {
		switch v := val.(type) {
		case int:
			return v
		case int64:
			return int(v)
		case float64:
			return int(v)
		case string:
			// Try to parse string as int
			if result, err := strconv.Atoi(v); err == nil {
				return result
			}
		}
	}
	return 0
}

// getCategoryStatus returns status based on score percentage.
func getCategoryStatus(score, maxScore int) string {
	percentage := float64(score) / float64(maxScore) * 100
	if percentage >= 70 {
		return "good"
	} else if percentage >= 40 {
		return "warning"
	}
	return "critical"
}

// min returns the minimum of two integers.
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
