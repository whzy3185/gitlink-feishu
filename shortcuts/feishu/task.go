package feishu

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/gitlink-org/gitlink-cli/shortcuts/common"
	"github.com/gitlink-org/gitlink-cli/shortcuts/workflow"
)

type TaskCandidate struct {
	UniqueKey        string `json:"unique_key"`
	Title            string `json:"title"`
	Description      string `json:"description"`
	SourceType       string `json:"source_type"`
	SourceKey        string `json:"source_key"`
	Repository       string `json:"repository"`
	Priority         string `json:"priority"`
	TaskType         string `json:"task_type"`
	RecommendedOwner string `json:"recommended_owner,omitempty"`
	Status           string `json:"status"`
	DueHint          string `json:"due_hint,omitempty"`
	GitLinkURL       string `json:"gitlink_url,omitempty"`
	DocURL           string `json:"doc_url,omitempty"`
}

type TaskCreateOptions struct {
	AppID         string `json:"-"`
	AppSecret     string `json:"-"`
	TaskProjectID string `json:"task_project_id,omitempty"`
	TaskSectionID string `json:"task_section_id,omitempty"`
	Send          bool   `json:"send"`
	DryRun        bool   `json:"dry_run"`
}

type TaskOutput struct {
	Mode          string             `json:"mode"`
	Send          bool               `json:"send"`
	DryRun        bool               `json:"dry_run"`
	TaskProjectID string             `json:"task_project_id,omitempty"`
	TaskSectionID string             `json:"task_section_id,omitempty"`
	TaskCount     int                `json:"task_count"`
	Tasks         []TaskCandidate    `json:"tasks"`
	Results       []TaskCreateResult `json:"results,omitempty"`
	Warnings      []string           `json:"warnings,omitempty"`
}

type TaskCreateResult struct {
	UniqueKey string `json:"unique_key"`
	Title     string `json:"title"`
	TaskID    string `json:"task_id,omitempty"`
	Created   bool   `json:"created"`
	Error     string `json:"error,omitempty"`
}

func BuildTaskCandidates(report workflow.RepoReportResult, docURL string) []TaskCandidate {
	tasks := []TaskCandidate{}
	repoURL := gitlinkRepoURL(report.Repository)
	for i, recommendation := range report.Recommendations {
		title := firstNonEmpty(recommendation, "Review workflow recommendation")
		tasks = append(tasks, TaskCandidate{
			UniqueKey:   stableKey("task", report.Repository, "recommendation", fmt.Sprintf("%d", i+1)),
			Title:       title,
			Description: "Workflow recommendation from gitlink-cli repo report.",
			SourceType:  "recommendation",
			SourceKey:   fmt.Sprintf("recommendation-%d", i+1),
			Repository:  report.Repository,
			Priority:    priorityForRisk(report.RiskLevel),
			TaskType:    "workflow_recommendation",
			Status:      "todo",
			DueHint:     "next review cycle",
			GitLinkURL:  repoURL,
			DocURL:      strings.TrimSpace(docURL),
		})
	}
	if report.IssueSummary.HighRisk > 0 {
		tasks = append(tasks, TaskCandidate{
			UniqueKey:   stableKey("task", report.Repository, "issues", "high-risk"),
			Title:       fmt.Sprintf("Triage %d high-risk GitLink issues", report.IssueSummary.HighRisk),
			Description: "High-risk issue bucket from workflow report. Review GitLink issues before routine work.",
			SourceType:  "issues",
			SourceKey:   "issues-high-risk",
			Repository:  report.Repository,
			Priority:    "high",
			TaskType:    "issue_triage",
			Status:      "todo",
			DueHint:     "as soon as possible",
			GitLinkURL:  appendPath(repoURL, "issues"),
			DocURL:      strings.TrimSpace(docURL),
		})
	}
	if report.IssueSummary.MissingInfo > 0 {
		tasks = append(tasks, TaskCandidate{
			UniqueKey:   stableKey("task", report.Repository, "issues", "missing-info"),
			Title:       fmt.Sprintf("Request missing information for %d issues", report.IssueSummary.MissingInfo),
			Description: "Some issues need reproduction steps, logs, version details, or command output.",
			SourceType:  "issues",
			SourceKey:   "issues-missing-info",
			Repository:  report.Repository,
			Priority:    "medium",
			TaskType:    "issue_followup",
			Status:      "todo",
			DueHint:     "this week",
			GitLinkURL:  appendPath(repoURL, "issues"),
			DocURL:      strings.TrimSpace(docURL),
		})
	}
	if report.PRSummary.HighRisk > 0 {
		tasks = append(tasks, TaskCandidate{
			UniqueKey:   stableKey("task", report.Repository, "prs", "high-risk"),
			Title:       fmt.Sprintf("Review %d high-risk pull requests", report.PRSummary.HighRisk),
			Description: "High-risk PR bucket from workflow report. Check review focus and merge readiness.",
			SourceType:  "prs",
			SourceKey:   "prs-high-risk",
			Repository:  report.Repository,
			Priority:    "high",
			TaskType:    "pr_review",
			Status:      "todo",
			DueHint:     "before next merge window",
			GitLinkURL:  appendPath(repoURL, "pulls"),
			DocURL:      strings.TrimSpace(docURL),
		})
	}
	if len(report.PRSummary.ReviewFocus) > 0 {
		tasks = append(tasks, TaskCandidate{
			UniqueKey:   stableKey("task", report.Repository, "prs", "review-focus"),
			Title:       "Review PR focus areas",
			Description: strings.Join(limitStrings(report.PRSummary.ReviewFocus, 8), "\n"),
			SourceType:  "prs",
			SourceKey:   "prs-review-focus",
			Repository:  report.Repository,
			Priority:    "medium",
			TaskType:    "review_focus",
			Status:      "todo",
			DueHint:     "this week",
			GitLinkURL:  appendPath(repoURL, "pulls"),
			DocURL:      strings.TrimSpace(docURL),
		})
	}
	if len(tasks) == 0 {
		tasks = append(tasks, TaskCandidate{
			UniqueKey:   stableKey("task", report.Repository, "report", "review"),
			Title:       "Review GitLink workflow report",
			Description: "No high-risk task candidates were detected. Keep a regular owner review cadence.",
			SourceType:  "report",
			SourceKey:   "report-review",
			Repository:  report.Repository,
			Priority:    "low",
			TaskType:    "report_review",
			Status:      "todo",
			DueHint:     "next review cycle",
			GitLinkURL:  repoURL,
			DocURL:      strings.TrimSpace(docURL),
		})
	}
	return dedupeTasks(tasks)
}

func taskCreateOptionsFromContext(ctx *common.RuntimeContext) (TaskCreateOptions, error) {
	opts := TaskCreateOptions{
		AppID:         firstNonEmpty(ctx.Arg("app-id"), os.Getenv("FEISHU_APP_ID")),
		AppSecret:     firstNonEmpty(ctx.Arg("app-secret"), os.Getenv("FEISHU_APP_SECRET")),
		TaskProjectID: firstNonEmpty(ctx.Arg("task-project-id"), os.Getenv("FEISHU_TASK_PROJECT_ID")),
		TaskSectionID: firstNonEmpty(ctx.Arg("task-section-id"), os.Getenv("FEISHU_TASK_SECTION_ID")),
		Send:          parseBool(ctx.Arg("send")),
		DryRun:        parseBool(ctx.Arg("dry-run")),
	}
	if opts.Send && opts.DryRun {
		return TaskCreateOptions{}, fmt.Errorf("--send and --dry-run cannot be used together")
	}
	if opts.Send {
		if strings.TrimSpace(opts.AppID) == "" {
			return TaskCreateOptions{}, fmt.Errorf("--send requires --app-id or FEISHU_APP_ID")
		}
		if strings.TrimSpace(opts.AppSecret) == "" {
			return TaskCreateOptions{}, fmt.Errorf("--send requires --app-secret or FEISHU_APP_SECRET")
		}
	}
	return opts, nil
}

func createTasksOrPreview(ctx *common.RuntimeContext, opts TaskCreateOptions, tasks []TaskCandidate) error {
	output := TaskOutput{
		Mode:          "preview",
		Send:          opts.Send,
		DryRun:        !opts.Send,
		TaskProjectID: redactToken(opts.TaskProjectID),
		TaskSectionID: redactToken(opts.TaskSectionID),
		TaskCount:     len(tasks),
		Tasks:         tasks,
		Warnings: []string{
			"Experimental: Feishu task creation requires self-built app task scopes.",
			"Deduplication is local unique_key generation only; Feishu Task API search/linking is not implemented in this pass.",
		},
	}
	if !opts.Send {
		return renderTaskOutput(os.Stdout, output, formatOrDefault(ctx, "markdown"))
	}

	client := NewOpenAPIClient(nil)
	token, err := client.TenantAccessToken(context.Background(), opts.AppID, opts.AppSecret)
	if err != nil {
		return err
	}
	output.Mode = "sent"
	output.DryRun = false
	for _, task := range tasks {
		result := TaskCreateResult{UniqueKey: task.UniqueKey, Title: task.Title}
		created, err := client.CreateTask(context.Background(), token.Value, task)
		if err != nil {
			result.Error = diagnoseOpenAPIError(err, "task create", "task")
			output.Results = append(output.Results, result)
			_ = renderTaskOutput(os.Stdout, output, formatOrDefault(ctx, "json"))
			return err
		}
		result.TaskID = created.TaskID
		result.Created = true
		output.Results = append(output.Results, result)
	}
	return renderTaskOutput(os.Stdout, output, formatOrDefault(ctx, "json"))
}

func renderTaskOutput(w io.Writer, output TaskOutput, format string) error {
	switch normalizeFormat(format) {
	case "markdown":
		return writeTaskMarkdown(w, output)
	case "table":
		return writeTaskTable(w, output)
	default:
		return writeJSON(w, output)
	}
}

func writeTaskMarkdown(w io.Writer, output TaskOutput) error {
	if _, err := fmt.Fprintf(w, "# Feishu Task %s\n\n", titleWord(output.Mode)); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "- Send: `%t`\n- Dry run: `%t`\n- Tasks: `%d`\n\n", output.Send, output.DryRun, output.TaskCount); err != nil {
		return err
	}
	for _, warning := range output.Warnings {
		if _, err := fmt.Fprintf(w, "- %s\n", warning); err != nil {
			return err
		}
	}
	if len(output.Warnings) > 0 {
		if _, err := fmt.Fprintln(w); err != nil {
			return err
		}
	}
	for _, task := range output.Tasks {
		if _, err := fmt.Fprintf(w, "## %s\n\n- Key: `%s`\n- Priority: `%s`\n- Source: `%s/%s`\n\n%s\n\n", task.Title, task.UniqueKey, task.Priority, task.SourceType, task.SourceKey, task.Description); err != nil {
			return err
		}
	}
	return nil
}

func writeTaskTable(w io.Writer, output TaskOutput) error {
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	if _, err := fmt.Fprintln(tw, "KEY\tPRIORITY\tSOURCE\tTITLE"); err != nil {
		return err
	}
	for _, task := range output.Tasks {
		if _, err := fmt.Fprintf(tw, "%s\t%s\t%s/%s\t%s\n", task.UniqueKey, task.Priority, task.SourceType, task.SourceKey, task.Title); err != nil {
			return err
		}
	}
	return tw.Flush()
}

func titleWord(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	return strings.ToUpper(value[:1]) + value[1:]
}

func priorityForRisk(risk string) string {
	switch strings.ToLower(strings.TrimSpace(risk)) {
	case "critical", "high":
		return "high"
	case "medium":
		return "medium"
	default:
		return "low"
	}
}

func appendPath(base string, path string) string {
	if strings.TrimSpace(base) == "" {
		return ""
	}
	return strings.TrimRight(base, "/") + "/" + strings.Trim(path, "/")
}

func dedupeTasks(tasks []TaskCandidate) []TaskCandidate {
	seen := map[string]bool{}
	result := []TaskCandidate{}
	for _, task := range tasks {
		if task.UniqueKey == "" || seen[task.UniqueKey] {
			continue
		}
		seen[task.UniqueKey] = true
		result = append(result, task)
	}
	return result
}
