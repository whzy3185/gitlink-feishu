package health

import (
	"testing"

	"github.com/gitlink-org/gitlink-cli/internal/i18n"
)

// TestDiagnoseShortcutExists tests that diagnose shortcut is registered
func TestDiagnoseShortcutExists(t *testing.T) {
	tr := i18n.Default()
	shortcuts := Shortcuts(tr)

	if len(shortcuts) < 2 {
		t.Errorf("Expected at least 2 shortcuts (fetch + diagnose), got %d", len(shortcuts))
	}

	// Find diagnose shortcut
	var diagnoseFound bool
	for _, s := range shortcuts {
		if s.Name == "diagnose" {
			diagnoseFound = true
			// Check that diagnose has expected flags
			expectedFlags := []string{"format", "verbose", "db"}
			for _, expectedFlag := range expectedFlags {
				found := false
				for _, f := range s.Flags {
					if f.Name == expectedFlag {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("diagnose shortcut missing expected flag: %s", expectedFlag)
				}
			}
			break
		}
	}

	if !diagnoseFound {
		t.Error("diagnose shortcut not found in Shortcuts()")
	}
}

// TestDiagnosisDataStructure tests that DiagnosisData can be created
func TestDiagnosisDataStructure(t *testing.T) {
	data := &DiagnosisData{
		Detail:       map[string]interface{}{"name": "test"},
		Readme:       map[string]interface{}{"content": "test readme"},
		Contributors: []interface{}{map[string]interface{}{"id": 1}},
		Builds:       []interface{}{map[string]interface{}{"id": "build-1"}},
	}

	if data.Detail == nil {
		t.Error("Detail should not be nil")
	}
	if data.Readme == nil {
		t.Error("Readme should not be nil")
	}
	if len(data.Contributors) == 0 {
		t.Error("Contributors should not be empty")
	}
	if len(data.Builds) == 0 {
		t.Error("Builds should not be empty")
	}
}

// TestHealthReportStructure tests that HealthReport can be created
func TestHealthReportStructure(t *testing.T) {
	report := &HealthReport{
		Owner:      "test-owner",
		Repo:       "test-repo",
		TotalScore: 75,
		MaxScore:   100,
		Categories: []CategoryScore{
			{
				Name:     "Documentation",
				Score:    15,
				MaxScore: 20,
				Status:   "good",
			},
		},
	}

	if report.Owner != "test-owner" {
		t.Errorf("Expected owner 'test-owner', got '%s'", report.Owner)
	}
	if report.TotalScore != 75 {
		t.Errorf("Expected total score 75, got %d", report.TotalScore)
	}
	if len(report.Categories) != 1 {
		t.Errorf("Expected 1 category, got %d", len(report.Categories))
	}
}

// TestSuggestionStructure tests that Suggestion can be created
func TestSuggestionStructure(t *testing.T) {
	suggestion := Suggestion{
		Category:    "Documentation",
		Title:       "Add README",
		Description: "Project lacks a README file",
		Actions: []string{
			"Create README.md",
			"Add project description",
		},
	}

	if suggestion.Category != "Documentation" {
		t.Errorf("Expected category 'Documentation', got '%s'", suggestion.Category)
	}
	if len(suggestion.Actions) != 2 {
		t.Errorf("Expected 2 actions, got %d", len(suggestion.Actions))
	}
}

// TestCalculateHealthScoreWithEmptyData tests scoring with empty data
func TestCalculateHealthScoreWithEmptyData(t *testing.T) {
	tr := i18n.Default()
	data := &DiagnosisData{
		Detail:       nil,
		Readme:       nil,
		Contributors: nil,
		Builds:       nil,
	}

	report := calculateHealthScore("owner", "repo", data, tr)

	if report == nil {
		t.Fatal("Report should not be nil")
	}

	// With empty data, score should be low
	if report.TotalScore > 30 {
		t.Errorf("Expected low score with empty data, got %d", report.TotalScore)
	}

	// Should have 5 categories
	if len(report.Categories) != 5 {
		t.Errorf("Expected 5 categories, got %d", len(report.Categories))
	}
}

// TestCalculateHealthScoreWithGoodData tests scoring with good data
func TestCalculateHealthScoreWithGoodData(t *testing.T) {
	tr := i18n.Default()

	// Create good data
	readmeContent := "# Test Project\n\n## Introduction\n\nThis is a test project.\n\n## Installation\n\nRun `go install`.\n\n## Usage\n\nSee examples.\n\n## API\n\nDocumentation here.\n\n## Contributing\n\nPlease contribute.\n\n## License\n\nMIT License."
	encodedContent := encodeBase64(readmeContent)

	data := &DiagnosisData{
		Detail: map[string]interface{}{
			"name":         "test-repo",
			"description":  "A test repository for health check",
			"license_name": "MIT",
		},
		Readme: map[string]interface{}{
			"content": encodedContent,
		},
		Contributors: []interface{}{
			map[string]interface{}{"id": 1, "login": "user1"},
			map[string]interface{}{"id": 2, "login": "user2"},
			map[string]interface{}{"id": 3, "login": "user3"},
			map[string]interface{}{"id": 4, "login": "user4"},
			map[string]interface{}{"id": 5, "login": "user5"},
		},
		Builds: []interface{}{
			map[string]interface{}{"id": "build-1", "status": "success"},
			map[string]interface{}{"id": "build-2", "status": "success"},
			map[string]interface{}{"id": "build-3", "status": "success"},
			map[string]interface{}{"id": "build-4", "status": "success"},
			map[string]interface{}{"id": "build-5", "status": "success"},
		},
	}

	report := calculateHealthScore("owner", "repo", data, tr)

	if report == nil {
		t.Fatal("Report should not be nil")
	}

	// With good data, score should be high
	if report.TotalScore < 50 {
		t.Errorf("Expected higher score with good data, got %d", report.TotalScore)
	}

	// Should have 5 categories
	if len(report.Categories) != 5 {
		t.Errorf("Expected 5 categories, got %d", len(report.Categories))
	}
}

// TestGenerateSuggestionsWithLowScores tests suggestion generation
func TestGenerateSuggestionsWithLowScores(t *testing.T) {
	tr := i18n.Default()

	// Create data that will result in low scores
	data := &DiagnosisData{
		Detail:       nil,
		Readme:       nil,
		Contributors: nil,
		Builds:       nil,
	}

	report := calculateHealthScore("owner", "repo", data, tr)
	suggestions := generateSuggestions(report, data, tr)

	if len(suggestions) == 0 {
		t.Error("Expected suggestions for low-scoring report")
	}

	// Check that suggestions have required fields
	for _, s := range suggestions {
		if s.Title == "" {
			t.Error("Suggestion should have a title")
		}
		if s.Category == "" {
			t.Error("Suggestion should have a category")
		}
	}
}

// TestGetCategoryStatus tests status determination based on score
func TestGetCategoryStatus(t *testing.T) {
	tests := []struct {
		score    int
		maxScore int
		expected string
	}{
		{18, 20, "good"},    // 90%
		{16, 20, "good"},    // 80%
		{14, 20, "good"},    // 70%
		{12, 20, "warning"}, // 60%
		{8, 20, "warning"},  // 40%
		{6, 20, "critical"}, // 30%
		{4, 20, "critical"}, // 20%
	}

	for _, tt := range tests {
		status := getCategoryStatus(tt.score, tt.maxScore)
		if status != tt.expected {
			t.Errorf("getCategoryStatus(%d, %d) = %s, want %s", tt.score, tt.maxScore, status, tt.expected)
		}
	}
}

// Helper function to encode base64
func encodeBase64(s string) string {
	return s // The actual implementation in scoring.go handles this
}
