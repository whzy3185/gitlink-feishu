package issue

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"

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
	Metadata    map[string]interface{}
}

func Shortcuts() []*common.Shortcut {
	return []*common.Shortcut{
		newBatchCloseShortcut(),
		{
			Name:        "list",
			Description: "List issues",
			Flags: []common.Flag{
				{Name: "state", Short: "s", Usage: "Filter by state: open, closed, all", Default: "open"},
				{Name: "page", Short: "p", Usage: "Page number", Default: "1"},
				{Name: "limit", Short: "l", Usage: "Items per page", Default: "20"},
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
			Description: "Create a new issue",
			Flags: []common.Flag{
				{Name: "title", Short: "t", Usage: "Issue title", Required: true},
				{Name: "body", Short: "b", Usage: "Issue description"},
				{Name: "assignee", Short: "a", Usage: "Assignee login"},
				{Name: "milestone", Short: "m", Usage: "Milestone ID"},
				{Name: "label", Usage: "Label ID"},
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
			Description: "View issue details",
			Flags: []common.Flag{
				{Name: "number", Short: "n", Usage: "Issue number (as shown in the web URL)", Required: true},
			},
			Run: func(ctx *common.RuntimeContext) error {
				if err := ctx.ResolveOwnerRepo(); err != nil {
					return err
				}
				number, err := ctx.RequireArg("number")
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
			Description: "Close an issue",
			Flags: []common.Flag{
				{Name: "number", Short: "n", Usage: "Issue number (as shown in the web URL)", Required: true},
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
				copyIssueMetadata(body, current.Metadata)
				env, err := ctx.CallAPI("PATCH", fmt.Sprintf("%s/issues/%s", v1RepoPath(ctx), number), body)
				if err != nil {
					return err
				}
				return ctx.Output(env)
			},
		},
		{
			Name:        "update",
			Description: "Update an issue",
			Flags: []common.Flag{
				{Name: "number", Short: "n", Usage: "Issue number (as shown in the web URL)", Required: true},
				{Name: "title", Short: "t", Usage: "New title"},
				{Name: "body", Short: "b", Usage: "New description"},
				{Name: "state", Short: "s", Usage: "New state: open, closed, or numeric status_id"},
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
				copyIssueMetadata(body, current.Metadata)
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
			Description: "Add a comment to an issue",
			Flags: []common.Flag{
				{Name: "number", Short: "n", Usage: "Issue number (as shown in the web URL)", Required: true},
				{Name: "body", Short: "b", Usage: "Comment body", Required: true},
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
		Metadata:    existingIssueMetadata(issueData),
	}, nil
}

func existingIssueMetadata(issueData map[string]interface{}) map[string]interface{} {
	metadata := map[string]interface{}{}
	copyIDValue(metadata, "priority_id", issueData["priority_id"])
	copyNestedIDValue(metadata, "priority_id", issueData["priority"])
	copyIDValue(metadata, "tracker_id", issueData["tracker_id"])
	copyNestedIDValue(metadata, "tracker_id", issueData["tracker"])
	copyIDValue(metadata, "fixed_version_id", issueData["fixed_version_id"])
	copyNestedIDValue(metadata, "fixed_version_id", issueData["fixed_version"])
	copyIDValue(metadata, "assigned_to_id", issueData["assigned_to_id"])
	copyNestedIDValue(metadata, "assigned_to_id", issueData["assigned_to"])

	if ids := issueTagIDs(issueData["issue_tags"]); len(ids) > 0 {
		metadata["issue_tag_ids"] = ids
	}
	return metadata
}

func copyIssueMetadata(body map[string]interface{}, metadata map[string]interface{}) {
	for key, value := range metadata {
		body[key] = value
	}
}

func copyNestedIDValue(dst map[string]interface{}, dstKey string, value interface{}) {
	object, ok := value.(map[string]interface{})
	if !ok {
		return
	}
	copyIDValue(dst, dstKey, object["id"])
}

func copyIDValue(dst map[string]interface{}, dstKey string, value interface{}) {
	switch v := value.(type) {
	case int:
		dst[dstKey] = v
	case int64:
		dst[dstKey] = v
	case float64:
		dst[dstKey] = int(v)
	case string:
		if strings.TrimSpace(v) != "" {
			dst[dstKey] = v
		}
	}
}

func issueTagIDs(value interface{}) []interface{} {
	tags, ok := value.([]interface{})
	if !ok {
		return nil
	}
	ids := make([]interface{}, 0, len(tags))
	for _, tag := range tags {
		tagData, ok := tag.(map[string]interface{})
		if !ok {
			continue
		}
		if id, ok := normalizedIDValue(tagData["id"]); ok {
			ids = append(ids, id)
		}
	}
	return ids
}

func normalizedIDValue(value interface{}) (interface{}, bool) {
	switch v := value.(type) {
	case int:
		return v, true
	case int64:
		return v, true
	case float64:
		return int(v), true
	case string:
		if strings.TrimSpace(v) != "" {
			return v, true
		}
	}
	return nil, false
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
