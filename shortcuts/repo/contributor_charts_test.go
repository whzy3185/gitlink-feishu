package repo

import (
	"strings"
	"testing"

	"github.com/gitlink-org/gitlink-cli/internal/i18n"
)

func TestRenderContributorsChart(t *testing.T) {
	tests := []struct {
		name     string
		data     *ContributorsResponse
		config   ChartConfig
		contains []string
		empty    bool
	}{
		{
			name: "nil data",
			data: nil,
			config: ChartConfig{
				Width:    60,
				MaxItems: 10,
			},
			empty: true,
		},
		{
			name: "empty list",
			data: &ContributorsResponse{
				List:       []ContributorData{},
				TotalCount: 0,
			},
			config: ChartConfig{
				Width:    60,
				MaxItems: 10,
			},
			empty: true,
		},
		{
			name: "single contributor",
			data: &ContributorsResponse{
				List: []ContributorData{
					{Login: "user1", Name: "User One", Contributions: 100, Type: "User"},
				},
				TotalCount: 1,
			},
			config: ChartConfig{
				Width:    60,
				MaxItems: 10,
			},
			contains: []string{"Contributors", "User One", "100"},
		},
		{
			name: "multiple contributors",
			data: &ContributorsResponse{
				List: []ContributorData{
					{Login: "user1", Name: "User One", Contributions: 100, Type: "User"},
					{Login: "user2", Name: "User Two", Contributions: 50, Type: "User"},
					{Login: "user3", Name: "User Three", Contributions: 25, Type: "User"},
				},
				TotalCount: 3,
			},
			config: ChartConfig{
				Width:    60,
				MaxItems: 10,
			},
			contains: []string{"Contributors", "User One", "User Two", "User Three"},
		},
		{
			name: "limit max items",
			data: &ContributorsResponse{
				List: []ContributorData{
					{Login: "user1", Contributions: 100},
					{Login: "user2", Contributions: 90},
					{Login: "user3", Contributions: 80},
					{Login: "user4", Contributions: 70},
					{Login: "user5", Contributions: 60},
				},
				TotalCount: 5,
			},
			config: ChartConfig{
				Width:    60,
				MaxItems: 2,
			},
			contains: []string{"user1", "user2"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := RenderContributorsChart(tt.data, tt.config, i18n.Default())

			if tt.empty {
				if !strings.Contains(result, "No contributors found") {
					t.Errorf("expected result to contain 'No contributors found', got %q", result)
				}
				return
			}

			for _, s := range tt.contains {
				if !strings.Contains(result, s) {
					t.Errorf("expected result to contain %q, but it didn't\nResult:\n%s", s, result)
				}
			}
		})
	}
}

func TestRenderPieChart(t *testing.T) {
	tests := []struct {
		name     string
		data     []ContributorData
		contains []string
		empty    bool
	}{
		{
			name:  "empty list",
			data:  []ContributorData{},
			empty: true,
		},
		{
			name: "single contributor",
			data: []ContributorData{
				{Login: "user1", Contributions: 100},
			},
			contains: []string{"Contribution Distribution", "user1", "100.0%"},
		},
		{
			name: "multiple contributors",
			data: []ContributorData{
				{Login: "user1", Contributions: 100},
				{Login: "user2", Contributions: 50},
				{Login: "user3", Contributions: 25},
			},
			contains: []string{"user1", "user2", "user3"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := RenderPieChart(tt.data, 60, i18n.Default())

			if tt.empty {
				if result != "" {
					t.Errorf("expected empty result, got %q", result)
				}
				return
			}

			for _, s := range tt.contains {
				if !strings.Contains(result, s) {
					t.Errorf("expected result to contain %q, but it didn't\nResult:\n%s", s, result)
				}
			}
		})
	}
}

func TestRenderContributorsTable(t *testing.T) {
	tests := []struct {
		name     string
		data     []ContributorData
		contains []string
		empty    bool
	}{
		{
			name:  "empty list",
			data:  []ContributorData{},
			empty: true,
		},
		{
			name: "single contributor",
			data: []ContributorData{
				{Login: "user1", Name: "User One", Contributions: 100, Type: "User"},
			},
			contains: []string{"Contributors List", "User One", "100"},
		},
		{
			name: "multiple contributors",
			data: []ContributorData{
				{Login: "user1", Contributions: 100, Type: "User"},
				{Login: "user2", Contributions: 50, Type: "User"},
			},
			contains: []string{"Rank", "Name", "Contributions", "Type"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := RenderContributorsTable(tt.data, i18n.Default())

			if tt.empty {
				if !strings.Contains(result, "No contributors found") {
					t.Errorf("expected result to contain 'No contributors found', got %q", result)
				}
				return
			}

			for _, s := range tt.contains {
				if !strings.Contains(result, s) {
					t.Errorf("expected result to contain %q, but it didn't\nResult:\n%s", s, result)
				}
			}
		})
	}
}

func TestRenderDonutChart(t *testing.T) {
	tests := []struct {
		name     string
		data     []ContributorData
		contains []string
		empty    bool
	}{
		{
			name:  "empty list",
			data:  []ContributorData{},
			empty: true,
		},
		{
			name: "with contributors",
			data: []ContributorData{
				{Login: "user1", Contributions: 100},
				{Login: "user2", Contributions: 50},
			},
			contains: []string{"Contributions", "150"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := RenderDonutChart(tt.data, i18n.Default())

			if tt.empty {
				if result != "" {
					t.Errorf("expected empty result, got %q", result)
				}
				return
			}

			for _, s := range tt.contains {
				if !strings.Contains(result, s) {
					t.Errorf("expected result to contain %q, but it didn't\nResult:\n%s", s, result)
				}
			}
		})
	}
}

func TestRenderSparkline(t *testing.T) {
	tests := []struct {
		name     string
		data     []ContributorData
		empty    bool
		minChars int
	}{
		{
			name:  "empty list",
			data:  []ContributorData{},
			empty: true,
		},
		{
			name: "single contributor",
			data: []ContributorData{
				{Login: "user1", Contributions: 100},
			},
			minChars: 1,
		},
		{
			name: "multiple contributors",
			data: []ContributorData{
				{Login: "user1", Contributions: 100},
				{Login: "user2", Contributions: 80},
				{Login: "user3", Contributions: 60},
				{Login: "user4", Contributions: 40},
				{Login: "user5", Contributions: 20},
			},
			minChars: 5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := RenderSparkline(tt.data)

			if tt.empty {
				if result != "" {
					t.Errorf("expected empty result, got %q", result)
				}
				return
			}

			if len(result) < tt.minChars {
				t.Errorf("expected at least %d sparkline characters, got %d", tt.minChars, len(result))
			}
		})
	}
}

func TestGetDisplayName(t *testing.T) {
	tests := []struct {
		name     string
		input    ContributorData
		expected string
	}{
		{
			name: "with name",
			input: ContributorData{
				Login: "user1",
				Name:  "User One",
			},
			expected: "User One",
		},
		{
			name: "without name",
			input: ContributorData{
				Login: "user1",
				Name:  "",
			},
			expected: "user1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getDisplayName(tt.input)
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}
