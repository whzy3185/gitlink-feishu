package health

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sync"

	"github.com/gitlink-org/gitlink-cli/internal/i18n"
	"github.com/gitlink-org/gitlink-cli/shortcuts/common"
	"golang.org/x/sync/errgroup"
)

// DiagnosisData holds all data needed for health diagnosis.
type DiagnosisData struct {
	Detail       map[string]interface{}
	Readme       map[string]interface{}
	Contributors []interface{}
	Builds       []interface{}
}

// HealthReport represents the complete health diagnosis report.
type HealthReport struct {
	Owner       string
	Repo        string
	TotalScore  int
	MaxScore    int
	Categories  []CategoryScore
	Suggestions []Suggestion
}

// CategoryScore represents score for a single category.
type CategoryScore struct {
	Name     string
	Score    int
	MaxScore int
	Status   string // "good", "warning", "critical"
	Details  []string
}

// Suggestion represents an improvement suggestion.
type Suggestion struct {
	Category    string
	Title       string
	Description string
	Actions     []string
}

// diagnoseShortcut returns the diagnose command shortcut.
func diagnoseShortcut(tr *i18n.Translator) *common.Shortcut {
	return &common.Shortcut{
		Name:        "diagnose",
		Description: tr.T("cmd.health.diagnose.short"),
		Long:        tr.T("cmd.health.diagnose.long"),
		Flags: []common.Flag{
			{Name: "format", Short: "f", Usage: tr.T("cmd.health.diagnose.format"), Default: "text"},
			{Name: "verbose", Short: "v", Usage: tr.T("cmd.health.diagnose.verbose"), Bool: true},
			{Name: "db", Short: "d", Usage: tr.T("cmd.health.diagnose.db")},
		},
		Run: func(ctx *common.RuntimeContext) error {
			return runDiagnose(ctx, tr)
		},
	}
}

// runDiagnose executes the health diagnosis.
func runDiagnose(ctx *common.RuntimeContext, tr *i18n.Translator) error {
	// 1. Resolve owner/repo
	if err := ctx.ResolveOwnerRepo(); err != nil {
		return err
	}

	fmt.Fprintf(os.Stderr, "\n🔍 %s\n", tr.T("prompt.diagnosing"))
	fmt.Fprintf(os.Stderr, "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
	fmt.Fprintf(os.Stderr, "📦 %s/%s\n\n", ctx.Owner, ctx.Repo)

	// 2. Fetch data concurrently
	data, err := fetchDiagnosisData(ctx)
	if err != nil {
		return fmt.Errorf("fetch diagnosis data: %w", err)
	}

	// 3. Calculate scores
	report := calculateHealthScore(ctx.Owner, ctx.Repo, data, tr)

	// 4. Generate suggestions
	report.Suggestions = generateSuggestions(report, data, tr)

	// 5. Output report
	format := ctx.Arg("format")
	verbose := ctx.Arg("verbose") == "true"
	return outputReport(report, format, verbose, tr)
}

// fetchDiagnosisData fetches all required data concurrently.
func fetchDiagnosisData(ctx *common.RuntimeContext) (*DiagnosisData, error) {
	data := &DiagnosisData{}
	var mu sync.Mutex

	g, _ := errgroup.WithContext(context.Background())

	// Fetch project detail
	g.Go(func() error {
		env, err := ctx.CallAPI("GET", ctx.RepoPath()+"/detail", nil)
		if err != nil {
			return fmt.Errorf("fetch detail: %w", err)
		}
		if env.OK && env.Data != nil {
			mu.Lock()
			data.Detail = env.Data.(map[string]interface{})
			mu.Unlock()
		}
		return nil
	})

	// Fetch README
	g.Go(func() error {
		env, err := ctx.CallAPI("GET", ctx.RepoPath()+"/readme", nil)
		if err != nil {
			// README might not exist, don't fail
			return nil
		}
		if env.OK && env.Data != nil {
			mu.Lock()
			data.Readme = env.Data.(map[string]interface{})
			mu.Unlock()
		}
		return nil
	})

	// Fetch contributors
	g.Go(func() error {
		env, err := ctx.CallAPI("GET", ctx.RepoPath()+"/contributors", nil)
		if err != nil {
			return fmt.Errorf("fetch contributors: %w", err)
		}
		if env.OK && env.Data != nil {
			if resp, ok := env.Data.(map[string]interface{}); ok {
				if list, ok := resp["list"].([]interface{}); ok {
					mu.Lock()
					data.Contributors = list
					mu.Unlock()
				}
			}
		}
		return nil
	})

	// Fetch CI builds
	g.Go(func() error {
		q := map[string]string{"page": "1", "limit": "20"}
		queryParams := make(map[string][]string)
		for k, v := range q {
			queryParams[k] = []string{v}
		}
		env, err := ctx.CallAPIWithQuery("GET", ctx.RepoPath()+"/builds", nil)
		if err != nil {
			// CI might not be configured, don't fail
			return nil
		}
		if env.OK && env.Data != nil {
			if resp, ok := env.Data.(map[string]interface{}); ok {
				if builds, ok := resp["builds"].([]interface{}); ok {
					mu.Lock()
					data.Builds = builds
					mu.Unlock()
				} else if builds, ok := resp["builds"].([]map[string]interface{}); ok {
					// Handle alternative response format
					var buildList []interface{}
					for _, b := range builds {
						buildList = append(buildList, b)
					}
					mu.Lock()
					data.Builds = buildList
					mu.Unlock()
				}
			}
		}
		return nil
	})

	if err := g.Wait(); err != nil {
		return nil, err
	}

	return data, nil
}

// outputReport outputs the health report in the specified format.
func outputReport(report *HealthReport, format string, verbose bool, tr *i18n.Translator) error {
	switch format {
	case "json":
		return outputJSON(report)
	case "markdown", "md":
		return outputMarkdown(report, tr)
	default:
		return outputText(report, verbose, tr)
	}
}

// outputText outputs the report in text format.
func outputText(report *HealthReport, verbose bool, tr *i18n.Translator) error {
	// Header
	fmt.Printf("\n🔍 %s\n", tr.T("output.report_title"))
	fmt.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
	fmt.Printf("📦 %s: %s/%s\n", tr.T("output.project"), report.Owner, report.Repo)

	// Total score with stars
	stars := getStars(report.TotalScore, report.MaxScore)
	fmt.Printf("📊 %s: %d/%d %s\n\n", tr.T("output.total_score"), report.TotalScore, report.MaxScore, stars)

	// Category scores
	fmt.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
	fmt.Printf("📋 %s\n", tr.T("output.diagnosis_details"))
	fmt.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n\n")

	for _, cat := range report.Categories {
		icon := getCategoryIcon(cat.Status)
		fmt.Printf("%s %s (%d/%d)\n", icon, cat.Name, cat.Score, cat.MaxScore)
		for _, detail := range cat.Details {
			fmt.Printf("   %s\n", detail)
		}
		fmt.Println()
	}

	// Suggestions
	if len(report.Suggestions) > 0 {
		fmt.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
		fmt.Printf("💡 %s\n", tr.T("output.suggestions"))
		fmt.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n\n")

		for i, sug := range report.Suggestions {
			fmt.Printf("%d. %s\n", i+1, sug.Title)
			for _, action := range sug.Actions {
				fmt.Printf("   - %s\n", action)
			}
			fmt.Println()
		}
	}

	fmt.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
	return nil
}

// outputJSON outputs the report in JSON format.
func outputJSON(report *HealthReport) error {
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(data))
	return nil
}

// outputMarkdown outputs the report in Markdown format.
func outputMarkdown(report *HealthReport, tr *i18n.Translator) error {
	fmt.Printf("# 🔍 %s\n\n", tr.T("output.report_title"))
	fmt.Printf("**%s**: `%s/%s`\n\n", tr.T("output.project"), report.Owner, report.Repo)
	fmt.Printf("**%s**: %d/%d %s\n\n", tr.T("output.total_score"), report.TotalScore, report.MaxScore, getStars(report.TotalScore, report.MaxScore))

	fmt.Printf("## 📋 %s\n\n", tr.T("output.diagnosis_details"))
	fmt.Printf("| %s | %s | %s |\n", tr.T("table.category"), tr.T("table.score"), tr.T("table.status"))
	fmt.Printf("|------|------|--------|\n")

	for _, cat := range report.Categories {
		fmt.Printf("| %s | %d/%d | %s |\n", cat.Name, cat.Score, cat.MaxScore, cat.Status)
	}

	if len(report.Suggestions) > 0 {
		fmt.Printf("\n## 💡 %s\n\n", tr.T("output.suggestions"))
		for i, sug := range report.Suggestions {
			fmt.Printf("%d. **%s**\n", i+1, sug.Title)
			for _, action := range sug.Actions {
				fmt.Printf("   - %s\n", action)
			}
		}
	}

	return nil
}

// getStars returns star rating based on score percentage.
func getStars(score, maxScore int) string {
	percentage := float64(score) / float64(maxScore) * 100
	var stars string
	if percentage >= 80 {
		stars = "⭐⭐⭐⭐⭐"
	} else if percentage >= 60 {
		stars = "⭐⭐⭐⭐"
	} else if percentage >= 40 {
		stars = "⭐⭐⭐"
	} else if percentage >= 20 {
		stars = "⭐⭐"
	} else {
		stars = "⭐"
	}
	return stars
}

// getCategoryIcon returns icon based on category status.
func getCategoryIcon(status string) string {
	switch status {
	case "good":
		return "✅"
	case "warning":
		return "⚠️"
	case "critical":
		return "❌"
	default:
		return "ℹ️"
	}
}
