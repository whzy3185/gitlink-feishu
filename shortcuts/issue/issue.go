package issue

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/gitlink-org/gitlink-cli/internal/i18n"
	"github.com/gitlink-org/gitlink-cli/internal/output"
	"github.com/gitlink-org/gitlink-cli/shortcuts/common"
)

// v1RepoPath returns the v1 API path prefix: /v1/{owner}/{repo}
func v1RepoPath(ctx *common.RuntimeContext) string {
	return fmt.Sprintf("/v1/%s/%s", ctx.Owner, ctx.Repo)
}

type existingIssue struct {
	Subject     string
	Description string
}

func Shortcuts(translators ...*i18n.Translator) []*common.Shortcut {
	tr := shortcutTranslator(translators...)
	return []*common.Shortcut{
		newBatchCloseShortcut(),
		{
			Name:        "list",
			Description: tr.T("cmd.issue.list.short"),
			Flags: []common.Flag{
				{Name: "state", Short: "s", Usage: tr.T("flag.issue.state"), Default: "open"},
				{Name: "page", Short: "p", Usage: tr.T("flag.page"), Default: "1"},
				{Name: "limit", Short: "l", Usage: tr.T("flag.limit"), Default: "20"},
			},
			Run: func(ctx *common.RuntimeContext) error {
				if err := ctx.ResolveOwnerRepo(); err != nil {
					return err
				}
				q := url.Values{}
				q.Set("page", ctx.Arg("page"))
				q.Set("limit", ctx.Arg("limit"))
				if s := ctx.Arg("state"); s != "" {
					q.Set("state", s)
				}
				env, err := ctx.CallAPIWithQuery("GET", v1RepoPath(ctx)+"/issues", q)
				if err != nil {
					return err
				}
				normalizeIssueListIDs(env)
				return ctx.Output(env)
			},
		},
		{
			Name:        "create",
			Description: tr.T("cmd.issue.create.short"),
			Flags: []common.Flag{
				{Name: "title", Short: "t", Usage: tr.T("flag.issue.title"), Required: true},
				{Name: "body", Short: "b", Usage: tr.T("flag.issue.body")},
				{Name: "assignee", Short: "a", Usage: tr.T("flag.issue.assignee")},
				{Name: "milestone", Short: "m", Usage: tr.T("flag.issue.milestone")},
				{Name: "label", Usage: tr.T("flag.issue.label")},
			},
			Run: func(ctx *common.RuntimeContext) error {
				if err := ctx.ResolveOwnerRepo(); err != nil {
					return err
				}
				title, err := ctx.RequireArg("title")
				if err != nil {
					return err
				}
				body := map[string]interface{}{
					"subject":     title,
					"status_id":   1, // 1 = open (required by v1 API)
					"priority_id": 2, // 2 = normal
					"done_ratio":  0,
				}
				if desc := ctx.Arg("body"); desc != "" {
					body["description"] = desc
				}
				if a := ctx.Arg("assignee"); a != "" {
					body["assigned_to_id"] = a
				}
				if m := ctx.Arg("milestone"); m != "" {
					body["fixed_version_id"] = m
				}
				env, err := ctx.CallAPI("POST", v1RepoPath(ctx)+"/issues", body)
				if err != nil {
					return err
				}
				return ctx.Output(env)
			},
		},
		{
			Name:        "view",
			Description: tr.T("cmd.issue.view.short"),
			Flags: []common.Flag{
				{Name: "number", Short: "n", Usage: tr.T("flag.issue.number")},
				{Name: "id", Usage: "Alias for --number; uses the issue number from the web URL"},
			},
			Run: func(ctx *common.RuntimeContext) error {
				if err := ctx.ResolveOwnerRepo(); err != nil {
					return err
				}
				number, err := issueNumberArg(ctx)
				if err != nil {
					return err
				}
				env, err := ctx.CallAPI("GET", fmt.Sprintf("%s/issues/%s", v1RepoPath(ctx), number), nil)
				if err != nil {
					return err
				}
				return ctx.Output(env)
			},
		},
		{
			Name:        "close",
			Description: tr.T("cmd.issue.close.short"),
			Flags: []common.Flag{
				{Name: "number", Short: "n", Usage: tr.T("flag.issue.number"), Required: true},
			},
			Run: func(ctx *common.RuntimeContext) error {
				if err := ctx.ResolveOwnerRepo(); err != nil {
					return err
				}
				number, err := ctx.RequireArg("number")
				if err != nil {
					return err
				}
				current, err := fetchExistingIssue(ctx, number)
				if err != nil {
					return err
				}

				body := map[string]interface{}{
					"subject":     current.Subject,
					"description": current.Description,
					"status_id":   5, // 5 = closed
				}
				env, err := ctx.CallAPI("PATCH", fmt.Sprintf("%s/issues/%s", v1RepoPath(ctx), number), body)
				if err != nil {
					return err
				}
				return ctx.Output(env)
			},
		},
		{
			Name:        "update",
			Description: tr.T("cmd.issue.update.short"),
			Flags: []common.Flag{
				{Name: "number", Short: "n", Usage: tr.T("flag.issue.number"), Required: true},
				{Name: "title", Short: "t", Usage: tr.T("flag.issue.new_title")},
				{Name: "body", Short: "b", Usage: tr.T("flag.issue.new_body")},
				{Name: "state", Short: "s", Usage: tr.T("flag.issue.new_state")},
			},
			Run: func(ctx *common.RuntimeContext) error {
				if err := ctx.ResolveOwnerRepo(); err != nil {
					return err
				}
				number, err := ctx.RequireArg("number")
				if err != nil {
					return err
				}
				title := ctx.Arg("title")
				description := ctx.Arg("body")
				state := ctx.Arg("state")
				if title == "" && description == "" && state == "" {
					return fmt.Errorf("at least one of --title, --body, or --state is required")
				}

				current, err := fetchExistingIssue(ctx, number)
				if err != nil {
					return err
				}

				body := map[string]interface{}{
					"subject":     current.Subject,
					"description": current.Description,
				}
				if t := ctx.Arg("title"); t != "" {
					body["subject"] = t
				}
				if b := ctx.Arg("body"); b != "" {
					body["description"] = b
				}
				if s := ctx.Arg("state"); s != "" {
					statusID, err := normalizeIssueStatus(s)
					if err != nil {
						return err
					}
					body["status_id"] = statusID
				}
				env, err := ctx.CallAPI("PATCH", fmt.Sprintf("%s/issues/%s", v1RepoPath(ctx), number), body)
				if err != nil {
					return err
				}
				return ctx.Output(env)
			},
		},
		{
			Name:        "comment",
			Description: tr.T("cmd.issue.comment.short"),
			Flags: []common.Flag{
				{Name: "number", Short: "n", Usage: tr.T("flag.issue.number"), Required: true},
				{Name: "body", Short: "b", Usage: tr.T("flag.comment.body"), Required: true},
			},
			Run: func(ctx *common.RuntimeContext) error {
				if err := ctx.ResolveOwnerRepo(); err != nil {
					return err
				}
				number, err := ctx.RequireArg("number")
				if err != nil {
					return err
				}
				body, err := ctx.RequireArg("body")
				if err != nil {
					return err
				}
				payload := map[string]interface{}{
					"notes": body,
				}
				env, err := ctx.CallAPI("POST", fmt.Sprintf("%s/issues/%s/journals", v1RepoPath(ctx), number), payload)
				if err != nil {
					return err
				}
				return ctx.Output(env)
			},
		},
		{
			Name:        "assigners",
			Description: "List issue assigners",
			Flags: []common.Flag{
				{Name: "keyword", Short: "k", Usage: "Search keyword"},
			},
			Run: func(ctx *common.RuntimeContext) error {
				if err := ctx.ResolveOwnerRepo(); err != nil {
					return err
				}
				q := url.Values{}
				if keyword := ctx.Arg("keyword"); keyword != "" {
					q.Set("keyword", keyword)
				}
				env, err := ctx.CallAPIWithQuery("GET", v1RepoPath(ctx)+"/issue_assigners", q)
				if err != nil {
					return err
				}
				return ctx.Output(env)
			},
		},
		{
			Name:        "authors",
			Description: "List issue authors",
			Flags: []common.Flag{
				{Name: "keyword", Short: "k", Usage: "Search keyword"},
			},
			Run: func(ctx *common.RuntimeContext) error {
				if err := ctx.ResolveOwnerRepo(); err != nil {
					return err
				}
				q := url.Values{}
				if keyword := ctx.Arg("keyword"); keyword != "" {
					q.Set("keyword", keyword)
				}
				env, err := ctx.CallAPIWithQuery("GET", v1RepoPath(ctx)+"/issue_authors", q)
				if err != nil {
					return err
				}
				return ctx.Output(env)
			},
		},
	}
}

func shortcutTranslator(translators ...*i18n.Translator) *i18n.Translator {
	if len(translators) > 0 && translators[0] != nil {
		return translators[0]
	}
	return i18n.Default()
}

// normalizeIssueListIDs adds "number" (project_issues_index) and renames
// "id" to "database_id" so the user-facing output uses the project-level
// issue number, not the global database primary key.
func normalizeIssueListIDs(env *output.Envelope) {
	data, ok := env.Data.(map[string]interface{})
	if !ok {
		return
	}
	issues, ok := data["issues"].([]interface{})
	if !ok {
		return
	}
	for i, item := range issues {
		issue, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		// Copy project_issues_index to top-level "number"
		if num, ok := issue["project_issues_index"]; ok {
			issue["number"] = num
		}
		// Rename "id" (global database PK) to "database_id"
		if id, ok := issue["id"]; ok {
			issue["database_id"] = id
			delete(issue, "id")
		}
		issues[i] = issue
	}
}

func fetchExistingIssue(ctx *common.RuntimeContext, number string) (*existingIssue, error) {
	getEnv, err := ctx.CallAPI("GET", fmt.Sprintf("%s/issues/%s", v1RepoPath(ctx), number), nil)
	if err != nil {
		return nil, err
	}
	issueData, ok := getEnv.Data.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("failed to parse issue data")
	}
	subject, _ := issueData["subject"].(string)
	if subject == "" {
		return nil, fmt.Errorf("failed to parse issue subject")
	}
	description, _ := issueData["description"].(string)
	return &existingIssue{
		Subject:     subject,
		Description: description,
	}, nil
}

func normalizeIssueStatus(state string) (interface{}, error) {
	switch strings.ToLower(strings.TrimSpace(state)) {
	case "open":
		return 1, nil
	case "closed":
		return 5, nil
	default:
		if id, err := strconv.Atoi(state); err == nil {
			return id, nil
		}
		return nil, fmt.Errorf("invalid --state %q: use open, closed, or a numeric status_id", state)
	}
}

func issueNumberArg(ctx *common.RuntimeContext) (string, error) {
	if number := strings.TrimSpace(ctx.Arg("number")); number != "" {
		return number, nil
	}
	if id := strings.TrimSpace(ctx.Arg("id")); id != "" {
		return id, nil
	}
	return "", fmt.Errorf("required flag --number (or --id alias) not set")
}
