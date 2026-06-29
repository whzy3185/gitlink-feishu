package health

import (
	"github.com/gitlink-org/gitlink-cli/internal/i18n"
)

// generateSuggestions generates improvement suggestions based on the health report.
func generateSuggestions(report *HealthReport, data *DiagnosisData, tr *i18n.Translator) []Suggestion {
	suggestions := make([]Suggestion, 0)

	// Generate suggestions for each category
	for _, cat := range report.Categories {
		switch cat.Name {
		case tr.T("output.category_documentation"):
			suggestions = append(suggestions, generateDocSuggestions(cat, data, tr)...)
		case tr.T("output.category_license"):
			suggestions = append(suggestions, generateLicenseSuggestions(cat, data, tr)...)
		case tr.T("output.category_community"):
			suggestions = append(suggestions, generateCommunitySuggestions(cat, data, tr)...)
		case tr.T("output.category_maturity"):
			suggestions = append(suggestions, generateMaturitySuggestions(cat, data, tr)...)
		case tr.T("output.category_ci"):
			suggestions = append(suggestions, generateCISuggestions(cat, data, tr)...)
		}
	}

	// Limit to top 5 most important suggestions
	if len(suggestions) > 5 {
		suggestions = suggestions[:5]
	}

	return suggestions
}

// generateDocSuggestions generates documentation improvement suggestions.
func generateDocSuggestions(cat CategoryScore, data *DiagnosisData, tr *i18n.Translator) []Suggestion {
	suggestions := make([]Suggestion, 0)

	// Check if README is missing or empty
	if data.Readme == nil || getReadmeContent(data.Readme) == "" {
		suggestions = append(suggestions, Suggestion{
			Category:    cat.Name,
			Title:       tr.T("output.suggest_readme_add"),
			Description: tr.T("output.suggest_readme_add_desc"),
			Actions: []string{
				tr.T("output.suggest_readme_action1"),
				tr.T("output.suggest_readme_action2"),
				tr.T("output.suggest_readme_action3"),
			},
		})
	} else {
		// Check README quality
		content := getReadmeContent(data.Readme)
		contentLower := content // Will be used for quality check

		// Check for missing sections
		missingSections := checkMissingReadmeSections(contentLower, tr)
		if len(missingSections) > 0 {
			suggestions = append(suggestions, Suggestion{
				Category:    cat.Name,
				Title:       tr.T("output.suggest_readme_improve"),
				Description: tr.T("output.suggest_readme_improve_desc"),
				Actions:     missingSections,
			})
		}
	}

	// Check project description
	if data.Detail != nil {
		if desc, ok := data.Detail["description"].(string); !ok || desc == "" {
			suggestions = append(suggestions, Suggestion{
				Category:    cat.Name,
				Title:       tr.T("output.suggest_description_add"),
				Description: tr.T("output.suggest_description_add_desc"),
				Actions: []string{
					tr.T("output.suggest_description_action1"),
				},
			})
		}
	}

	return suggestions
}

// checkMissingReadmeSections checks for missing important README sections.
func checkMissingReadmeSections(content string, tr *i18n.Translator) []string {
	missing := make([]string, 0)
	contentLower := content

	// Check for installation instructions
	if !containsAny(contentLower, "install", "installation", "getting started", "quick start") {
		missing = append(missing, tr.T("output.suggest_section_install"))
	}

	// Check for usage examples
	if !containsAny(contentLower, "usage", "example", "how to use") {
		missing = append(missing, tr.T("output.suggest_section_usage"))
	}

	// Check for contributing guidelines
	if !containsAny(contentLower, "contributing", "contribute", "development") {
		missing = append(missing, tr.T("output.suggest_section_contrib"))
	}

	// Check for license information
	if !containsAny(contentLower, "license") {
		missing = append(missing, tr.T("output.suggest_section_license"))
	}

	return missing
}

// generateLicenseSuggestions generates license improvement suggestions.
func generateLicenseSuggestions(cat CategoryScore, data *DiagnosisData, tr *i18n.Translator) []Suggestion {
	suggestions := make([]Suggestion, 0)

	if data.Detail != nil {
		if licenseName, ok := data.Detail["license_name"].(string); !ok || licenseName == "" {
			suggestions = append(suggestions, Suggestion{
				Category:    cat.Name,
				Title:       tr.T("output.suggest_license_add"),
				Description: tr.T("output.suggest_license_add_desc"),
				Actions: []string{
					tr.T("output.suggest_license_action1"),
					tr.T("output.suggest_license_action2"),
					tr.T("output.suggest_license_action3"),
				},
			})
		} else if !isWellKnownLicense(licenseName) {
			suggestions = append(suggestions, Suggestion{
				Category:    cat.Name,
				Title:       tr.T("output.suggest_license_standard"),
				Description: tr.T("output.suggest_license_standard_desc"),
				Actions: []string{
					tr.T("output.suggest_license_standard_action"),
				},
			})
		}
	}

	return suggestions
}

// generateCommunitySuggestions generates community improvement suggestions.
func generateCommunitySuggestions(cat CategoryScore, data *DiagnosisData, tr *i18n.Translator) []Suggestion {
	suggestions := make([]Suggestion, 0)

	// Check contributors
	contributorCount := len(data.Contributors)
	if contributorCount < 3 {
		suggestions = append(suggestions, Suggestion{
			Category:    cat.Name,
			Title:       tr.T("output.suggest_community_grow"),
			Description: tr.T("output.suggest_community_grow_desc"),
			Actions: []string{
				tr.T("output.suggest_community_action1"),
				tr.T("output.suggest_community_action2"),
				tr.T("output.suggest_community_action3"),
			},
		})
	}

	// Check issues
	if data.Detail != nil {
		issuesCount := getIntFromMap(data.Detail, "issues_count")
		if issuesCount == 0 {
			suggestions = append(suggestions, Suggestion{
				Category:    cat.Name,
				Title:       tr.T("output.suggest_issues_encourage"),
				Description: tr.T("output.suggest_issues_encourage_desc"),
				Actions: []string{
					tr.T("output.suggest_issues_action1"),
					tr.T("output.suggest_issues_action2"),
				},
			})
		}
	}

	return suggestions
}

// generateMaturitySuggestions generates maturity improvement suggestions.
func generateMaturitySuggestions(cat CategoryScore, data *DiagnosisData, tr *i18n.Translator) []Suggestion {
	suggestions := make([]Suggestion, 0)

	if data.Detail != nil {
		// Check branches
		branchesCount := getIntFromMap(data.Detail, "branches_count")
		if branchesCount < 2 {
			suggestions = append(suggestions, Suggestion{
				Category:    cat.Name,
				Title:       tr.T("output.suggest_branches_setup"),
				Description: tr.T("output.suggest_branches_setup_desc"),
				Actions: []string{
					tr.T("output.suggest_branches_action1"),
					tr.T("output.suggest_branches_action2"),
				},
			})
		}

		// Check releases
		tagsCount := getIntFromMap(data.Detail, "tags_count")
		if tagsCount == 0 {
			suggestions = append(suggestions, Suggestion{
				Category:    cat.Name,
				Title:       tr.T("output.suggest_release_create"),
				Description: tr.T("output.suggest_release_create_desc"),
				Actions: []string{
					tr.T("output.suggest_release_action1"),
					tr.T("output.suggest_release_action2"),
				},
			})
		}
	}

	return suggestions
}

// generateCISuggestions generates CI/CD improvement suggestions.
func generateCISuggestions(cat CategoryScore, data *DiagnosisData, tr *i18n.Translator) []Suggestion {
	suggestions := make([]Suggestion, 0)

	if len(data.Builds) == 0 {
		suggestions = append(suggestions, Suggestion{
			Category:    cat.Name,
			Title:       tr.T("output.suggest_ci_setup"),
			Description: tr.T("output.suggest_ci_setup_desc"),
			Actions: []string{
				tr.T("output.suggest_ci_action1"),
				tr.T("output.suggest_ci_action2"),
				tr.T("output.suggest_ci_action3"),
			},
		})
	} else {
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

		successRate := float64(successCount) / float64(len(data.Builds))
		if successRate < 0.5 {
			suggestions = append(suggestions, Suggestion{
				Category:    cat.Name,
				Title:       tr.T("output.suggest_ci_improve"),
				Description: tr.T("output.suggest_ci_improve_desc"),
				Actions: []string{
					tr.T("output.suggest_ci_improve_action1"),
					tr.T("output.suggest_ci_improve_action2"),
				},
			})
		}
	}

	return suggestions
}

// containsAny checks if content contains any of the given substrings.
func containsAny(content string, substrings ...string) bool {
	for _, s := range substrings {
		if contains(content, s) {
			return true
		}
	}
	return false
}

// contains checks if content contains a substring (case-insensitive).
func contains(content, substr string) bool {
	return len(content) >= len(substr) &&
		(content == substr || len(content) > 0 && containsHelper(content, substr))
}

// containsHelper is a helper function for case-insensitive substring check.
func containsHelper(content, substr string) bool {
	// Simple case-insensitive check
	for i := 0; i <= len(content)-len(substr); i++ {
		match := true
		for j := 0; j < len(substr); j++ {
			c1 := content[i+j]
			c2 := substr[j]
			// Convert to lowercase for comparison
			if c1 >= 'A' && c1 <= 'Z' {
				c1 += 32
			}
			if c2 >= 'A' && c2 <= 'Z' {
				c2 += 32
			}
			if c1 != c2 {
				match = false
				break
			}
		}
		if match {
			return true
		}
	}
	return false
}
