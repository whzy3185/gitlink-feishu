package repo

import (
	"fmt"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/gitlink-org/gitlink-cli/internal/i18n"
)

// ContributorData represents a single contributor's information
type ContributorData struct {
	Login         string `json:"login"`
	Name          string `json:"name"`
	Contributions int    `json:"contributions"`
	Type          string `json:"type"`
	ImageURL      string `json:"image_url"`
}

// ContributorsResponse represents the API response for contributors
type ContributorsResponse struct {
	List       []ContributorData `json:"list"`
	TotalCount int               `json:"total_count"`
}

// ChartConfig holds configuration for chart rendering
type ChartConfig struct {
	Width      int
	ShowLegend bool
	MaxItems   int
}

// displayWidth calculates the display width of a string, accounting for CJK characters
func displayWidth(s string) int {
	width := 0
	for _, r := range s {
		// CJK characters and fullwidth characters have width 2
		if r >= 0x4E00 && r <= 0x9FFF || // CJK Unified Ideographs
			r >= 0x3000 && r <= 0x303F || // CJK Symbols and Punctuation
			r >= 0xFF00 && r <= 0xFFEF { // Halfwidth and Fullwidth Forms
			width += 2
		} else {
			width += 1
		}
	}
	return width
}

// padRight pads a string to the specified display width
func padRight(s string, width int) string {
	currentWidth := displayWidth(s)
	if currentWidth >= width {
		return s
	}
	return s + strings.Repeat(" ", width-currentWidth)
}

// truncateString truncates a string to fit within the specified display width
func truncateString(s string, maxDisplayWidth int) string {
	if displayWidth(s) <= maxDisplayWidth {
		return s
	}

	result := ""
	currentWidth := 0
	for _, r := range s {
		rWidth := 1
		if r >= 0x4E00 && r <= 0x9FFF || r >= 0x3000 && r <= 0x303F || r >= 0xFF00 && r <= 0xFFEF {
			rWidth = 2
		}

		if currentWidth+rWidth > maxDisplayWidth-3 {
			break
		}
		result += string(r)
		currentWidth += rWidth
	}
	return result + "..."
}

// RenderContributorsChart renders an ASCII chart for contributors
func RenderContributorsChart(data *ContributorsResponse, config ChartConfig, tr *i18n.Translator) string {
	if data == nil || len(data.List) == 0 {
		return "  " + tr.T("output.contributors.chart.no_data")
	}

	// Limit the number of items to display
	items := data.List
	if config.MaxItems > 0 && len(items) > config.MaxItems {
		items = items[:config.MaxItems]
	}

	// Calculate total contributions
	totalContributions := 0
	for _, c := range items {
		totalContributions += c.Contributions
	}

	var sections []string

	// Header
	sections = append(sections, "")
	sections = append(sections, "  "+tr.T("output.contributors.chart.title"))
	sections = append(sections, "  "+strings.Repeat("-", 60))

	// Summary line
	sections = append(sections, "  "+tr.Tf("output.contributors.chart.summary", i18n.Args{"total": data.TotalCount, "count": totalContributions}))
	sections = append(sections, "")

	// Bar chart
	barChart := renderBarChart(items, config.Width, totalContributions)
	sections = append(sections, barChart)

	return strings.Join(sections, "\n")
}

// renderBarChart renders a simple horizontal bar chart
func renderBarChart(contributors []ContributorData, width int, total int) string {
	if len(contributors) == 0 {
		return ""
	}

	// Sort by contributions (descending)
	sorted := make([]ContributorData, len(contributors))
	copy(sorted, contributors)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Contributions > sorted[j].Contributions
	})

	// Find max contribution for scaling
	maxContrib := sorted[0].Contributions

	// Calculate max name display width for alignment
	maxNameWidth := 0
	for _, c := range sorted {
		name := getDisplayName(c)
		w := displayWidth(name)
		if w > maxNameWidth {
			maxNameWidth = w
		}
	}

	// Limit name width
	if maxNameWidth > 20 {
		maxNameWidth = 20
	}

	barWidth := 30

	var lines []string

	// Render each contributor
	for i, c := range sorted {
		name := getDisplayName(c)
		name = truncateString(name, maxNameWidth)
		name = padRight(name, maxNameWidth)

		// Calculate bar length
		barLen := 0
		if maxContrib > 0 {
			barLen = int(float64(c.Contributions) / float64(maxContrib) * float64(barWidth))
		}
		if barLen < 1 && c.Contributions > 0 {
			barLen = 1
		}

		// Calculate percentage
		percentage := 0.0
		if total > 0 {
			percentage = float64(c.Contributions) / float64(total) * 100
		}

		// Create bar
		bar := strings.Repeat("#", barLen)
		bar = padRight(bar, barWidth)

		// Format: rank. name | bar | count (percentage)
		line := fmt.Sprintf("  %2d. %s | %s | %6d (%5.1f%%)",
			i+1,
			name,
			bar,
			c.Contributions,
			percentage,
		)
		lines = append(lines, line)
	}

	return strings.Join(lines, "\n")
}

// getDisplayName returns the display name for a contributor
func getDisplayName(c ContributorData) string {
	if c.Name != "" {
		return c.Name
	}
	return c.Login
}

// RenderPieChart renders a simple percentage distribution chart
func RenderPieChart(contributors []ContributorData, width int, tr *i18n.Translator) string {
	if len(contributors) == 0 {
		return ""
	}

	// Calculate total
	total := 0
	for _, c := range contributors {
		total += c.Contributions
	}

	if total == 0 {
		return ""
	}

	// Sort by contributions
	sorted := make([]ContributorData, len(contributors))
	copy(sorted, contributors)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Contributions > sorted[j].Contributions
	})

	var lines []string

	// Header
	lines = append(lines, "")
	lines = append(lines, "  "+tr.T("output.contributors.chart.distribution"))
	lines = append(lines, "  "+strings.Repeat("-", 60))
	lines = append(lines, "")

	barWidth := 25

	// Calculate max name display width
	maxNameWidth := 0
	for _, c := range sorted {
		name := getDisplayName(c)
		w := displayWidth(name)
		if w > maxNameWidth {
			maxNameWidth = w
		}
	}
	if maxNameWidth > 20 {
		maxNameWidth = 20
	}

	// Render percentage bars
	for _, c := range sorted {
		name := getDisplayName(c)
		name = truncateString(name, maxNameWidth)
		name = padRight(name, maxNameWidth)

		percentage := float64(c.Contributions) / float64(total) * 100

		// Create percentage bar
		filled := int(percentage / 100 * float64(barWidth))
		if filled < 1 && c.Contributions > 0 {
			filled = 1
		}

		bar := strings.Repeat("#", filled) + strings.Repeat("-", barWidth-filled)

		line := fmt.Sprintf("  %s | %s | %5.1f%%",
			name,
			bar,
			percentage,
		)
		lines = append(lines, line)
	}

	return strings.Join(lines, "\n")
}

// RenderContributorsTable renders a simple table view of contributors
func RenderContributorsTable(contributors []ContributorData, tr *i18n.Translator) string {
	if len(contributors) == 0 {
		return "  " + tr.T("output.contributors.chart.no_data")
	}

	// Sort by contributions
	sorted := make([]ContributorData, len(contributors))
	copy(sorted, contributors)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Contributions > sorted[j].Contributions
	})

	var lines []string

	// Header
	lines = append(lines, "")
	lines = append(lines, "  "+tr.T("output.contributors.chart.list"))
	lines = append(lines, "  "+strings.Repeat("-", 60))
	lines = append(lines, "")

	// Table header
	lines = append(lines, "  "+tr.T("output.contributors.chart.table_header"))
	lines = append(lines, "  "+strings.Repeat("-", 60))

	// Table rows
	for i, c := range sorted {
		name := getDisplayName(c)
		name = truncateString(name, 20)
		name = padRight(name, 20)

		userType := c.Type
		if userType == "" || userType == "User" {
			userType = tr.T("output.contributors.chart.type_user")
		} else if userType == "Organization" {
			userType = tr.T("output.contributors.chart.type_organization")
		}

		line := fmt.Sprintf("  %-4d  %s  %13d  %s",
			i+1,
			name,
			c.Contributions,
			userType,
		)
		lines = append(lines, line)
	}

	return strings.Join(lines, "\n")
}

// RenderDonutChart renders a simple donut chart (placeholder)
func RenderDonutChart(contributors []ContributorData, tr *i18n.Translator) string {
	if len(contributors) == 0 {
		return ""
	}

	// Calculate total
	total := 0
	for _, c := range contributors {
		total += c.Contributions
	}

	if total == 0 {
		return ""
	}

	return "  " + tr.Tf("output.contributors.chart.total_contributions", i18n.Args{"count": total})
}

// RenderSparkline renders a sparkline for contribution trends
func RenderSparkline(contributors []ContributorData) string {
	if len(contributors) == 0 {
		return ""
	}

	// Sort by contributions
	sorted := make([]ContributorData, len(contributors))
	copy(sorted, contributors)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Contributions > sorted[j].Contributions
	})

	// Get top 10 contributions
	values := []int{}
	for i, c := range sorted {
		if i >= 10 {
			break
		}
		values = append(values, c.Contributions)
	}

	if len(values) == 0 {
		return ""
	}

	max := float64(values[0])
	if max == 0 {
		max = 1
	}

	// Sparkline characters
	sparkChars := []string{"_", ".", ":", "|", "!", "#", "$", "@"}

	sparkline := ""
	for _, v := range values {
		normalized := int(float64(v) / max * 7)
		if normalized < 0 {
			normalized = 0
		}
		if normalized > 7 {
			normalized = 7
		}
		sparkline += sparkChars[normalized] + " "
	}

	return sparkline
}

// stripANSI removes ANSI escape codes from a string
func stripANSI(s string) string {
	result := ""
	inEscape := false
	for _, c := range s {
		if c == '\033' {
			inEscape = true
			continue
		}
		if inEscape {
			if c == 'm' {
				inEscape = false
			}
			continue
		}
		result += string(c)
	}
	return result
}

// Unused import guard
var _ = utf8.RuneLen
