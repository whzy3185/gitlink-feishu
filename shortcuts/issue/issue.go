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

func normalizeIssueListState(state string) string {
	switch strings.ToLower(strings.TrimSpace(state)) {
	case "open", "opened":
		return "opened"
	case "closed":
		return "closed"
	case "all", "":
		return "all"
	default:
		return state
	}
}

type existingIssue struct {
	Subject     string
	Description string
	StatusID    interface{}
	PriorityID  interface{}
	TagIDs      []interface{}
	AssignerIDs []interface{}
	BranchName  string
	StartDate   string
	DueDate     string
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
				{Name: "keyword", Short: "k", Usage: tr.T("flag.search.keyword")},
				{Name: "participant", Usage: tr.T("flag.issue.participant")},
				{Name: "author-id", Usage: tr.T("flag.issue.author_id")},
				{Name: "assignee-id", Usage: tr.T("flag.issue.assignee_id")},
				{Name: "milestone-id", Usage: tr.T("flag.issue.milestone")},
				{Name: "status-id", Usage: tr.T("flag.issue.status_id")},
				{Name: "tag-ids", Usage: tr.T("flag.issue.tag_ids")},
				{Name: "sort-by", Usage: tr.T("flag.sort_by")},
				{Name: "sort-direction", Usage: tr.T("flag.sort_direction")},
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
					q.Set("category", normalizeIssueListState(s))
				}
				if keyword := ctx.Arg("keyword"); keyword != "" {
					q.Set("keyword", keyword)
				}
				if participant := ctx.Arg("participant"); participant != "" {
					q.Set("participant_category", participant)
				}
				if authorID := ctx.Arg("author-id"); authorID != "" {
					q.Set("author_id", authorID)
				}
				if assigneeID := ctx.Arg("assignee-id"); assigneeID != "" {
					q.Set("assigner_id", assigneeID)
				}
				if milestoneID := ctx.Arg("milestone-id"); milestoneID != "" {
					q.Set("milestone_id", milestoneID)
				}
				if statusID := ctx.Arg("status-id"); statusID != "" {
					q.Set("status_id", statusID)
				}
				if tagIDs := ctx.Arg("tag-ids"); tagIDs != "" {
					q.Set("issue_tag_ids", tagIDs)
				}
				if sortBy := ctx.Arg("sort-by"); sortBy != "" {
					q.Set("sort_by", sortBy)
				}
				if sortDirection := ctx.Arg("sort-direction"); sortDirection != "" {
					q.Set("sort_direction", sortDirection)
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
				{Name: "priority-id", Usage: "Priority ID", Default: "2"},
				{Name: "tag-ids", Usage: "Comma-separated issue tag IDs"},
				{Name: "assigner-ids", Usage: "Comma-separated issue assigner IDs"},
				{Name: "branch", Usage: "Linked branch name"},
				{Name: "start-date", Usage: "Start date (YYYY-MM-DD)"},
				{Name: "due-date", Usage: "Due date (YYYY-MM-DD)"},
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
				if err := applyIssueMetadataArgs(ctx, body); err != nil {
					return err
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
			Flags:       issueNumberFlags(),
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
			Flags:       issueNumberFlags(),
			Run: func(ctx *common.RuntimeContext) error {
				if err := ctx.ResolveOwnerRepo(); err != nil {
					return err
				}
				number, err := issueNumberArg(ctx)
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
				}
				preserveIssueMetadata(body, current)
				body["status_id"] = 5 // 5 = closed
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
			Flags: appendIssueNumberFlags(
				common.Flag{Name: "title", Short: "t", Usage: tr.T("flag.issue.new_title")},
				common.Flag{Name: "body", Short: "b", Usage: tr.T("flag.issue.new_body")},
				common.Flag{Name: "state", Short: "s", Usage: tr.T("flag.issue.new_state")},
				common.Flag{Name: "priority-id", Usage: "New priority ID"},
				common.Flag{Name: "tag-ids", Usage: "Comma-separated issue tag IDs"},
				common.Flag{Name: "assigner-ids", Usage: "Comma-separated issue assigner IDs"},
				common.Flag{Name: "branch", Usage: "Linked branch name"},
				common.Flag{Name: "start-date", Usage: "Start date (YYYY-MM-DD)"},
				common.Flag{Name: "due-date", Usage: "Due date (YYYY-MM-DD)"},
			),
			Run: func(ctx *common.RuntimeContext) error {
				if err := ctx.ResolveOwnerRepo(); err != nil {
					return err
				}
				number, err := issueNumberArg(ctx)
				if err != nil {
					return err
				}
				title := ctx.Arg("title")
				description := ctx.Arg("body")
				state := ctx.Arg("state")
				if title == "" && description == "" && state == "" && !hasIssueMetadataArgs(ctx) {
					return fmt.Errorf("at least one update field is required")
				}

				current, err := fetchExistingIssue(ctx, number)
				if err != nil {
					return err
				}

				body := map[string]interface{}{
					"subject":     current.Subject,
					"description": current.Description,
				}
				preserveIssueMetadata(body, current)
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
				if err := applyIssueMetadataArgs(ctx, body); err != nil {
					return err
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
			Flags: appendIssueNumberFlags(
				common.Flag{Name: "body", Short: "b", Usage: tr.T("flag.comment.body"), Required: true},
				common.Flag{Name: "parent-id", Usage: "Parent comment ID when creating a child comment"},
				common.Flag{Name: "reply-id", Usage: "Reply target comment ID"},
				common.Flag{Name: "attachment-ids", Usage: "Comma-separated attachment IDs"},
				common.Flag{Name: "receivers", Short: "r", Usage: "Comma-separated @ receiver login names"},
				common.Flag{Name: "dry-run", Usage: "Preview the request body without creating the comment", Bool: true, Default: "false"},
			),
			Run: runIssueComment,
		},
		newIssueCommentsShortcut(),
		newIssueCommentUpdateShortcut(),
		newIssueCommentDeleteShortcut(),
		newIssueCommentChildrenShortcut(),
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
		{
			Name:        "priorities",
			Description: "List issue priorities",
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
				env, err := ctx.CallAPIWithQuery("GET", v1RepoPath(ctx)+"/issue_priorities", q)
				if err != nil {
					return err
				}
				return ctx.Output(env)
			},
		},
		{
			Name:        "tags",
			Description: "List issue tags",
			Flags: []common.Flag{
				{Name: "keyword", Short: "k", Usage: "Search keyword"},
				{Name: "only-name", Usage: "Only return tag names and IDs", Bool: true, Default: "false"},
				{Name: "order-by", Usage: "Order by: updated_on, created_on, issues_count"},
				{Name: "order-direction", Usage: "Order direction: asc or desc"},
			},
			Run: func(ctx *common.RuntimeContext) error {
				if err := ctx.ResolveOwnerRepo(); err != nil {
					return err
				}
				q := url.Values{}
				if keyword := ctx.Arg("keyword"); keyword != "" {
					q.Set("keyword", keyword)
				}
				if parseBool(ctx.Arg("only-name")) {
					q.Set("only_name", "true")
				}
				if orderBy := ctx.Arg("order-by"); orderBy != "" {
					q.Set("order_by", orderBy)
				}
				if orderDirection := ctx.Arg("order-direction"); orderDirection != "" {
					q.Set("order_direction", orderDirection)
				}
				env, err := ctx.CallAPIWithQuery("GET", v1RepoPath(ctx)+"/issue_tags", q)
				if err != nil {
					return err
				}
				return ctx.Output(env)
			},
		},
		{
			Name:        "statuses",
			Description: "List issue statuses",
			Flags: []common.Flag{
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
				env, err := ctx.CallAPIWithQuery("GET", v1RepoPath(ctx)+"/issue_statues", q)
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

func issueNumberFlags() []common.Flag {
	return []common.Flag{
		{Name: "number", Short: "n", Usage: "Issue number from the web URL (preferred)"},
		{Name: "id", Short: "i", Usage: "Compatibility alias for --number; this is not the database ID"},
	}
}

func appendIssueNumberFlags(flags ...common.Flag) []common.Flag {
	return append(issueNumberFlags(), flags...)
}

func issueNumberArg(ctx *common.RuntimeContext) (string, error) {
	if number := strings.TrimSpace(ctx.Arg("number")); number != "" {
		return number, nil
	}
	if id := strings.TrimSpace(ctx.Arg("id")); id != "" {
		return id, nil
	}
	return "", fmt.Errorf("required flag --number is missing (or use --id as a compatibility alias)")
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
		StatusID:    nestedIssueID(issueData, "status"),
		PriorityID:  nestedIssueID(issueData, "priority"),
		TagIDs:      issueObjectIDs(issueData, "tags", "issue_tags"),
		AssignerIDs: issueObjectIDs(issueData, "assigners"),
		BranchName:  stringField(issueData, "branch_name"),
		StartDate:   stringField(issueData, "start_date"),
		DueDate:     stringField(issueData, "due_date"),
	}, nil
}

func preserveIssueMetadata(body map[string]interface{}, issue *existingIssue) {
	if issue.StatusID != nil {
		body["status_id"] = issue.StatusID
	}
	if issue.PriorityID != nil {
		body["priority_id"] = issue.PriorityID
	}
	if len(issue.TagIDs) > 0 {
		body["issue_tag_ids"] = issue.TagIDs
	}
	if len(issue.AssignerIDs) > 0 {
		body["assigner_ids"] = issue.AssignerIDs
	}
	if issue.BranchName != "" {
		body["branch_name"] = issue.BranchName
	}
	if issue.StartDate != "" {
		body["start_date"] = issue.StartDate
	}
	if issue.DueDate != "" {
		body["due_date"] = issue.DueDate
	}
}

func nestedIssueID(data map[string]interface{}, key string) interface{} {
	item, ok := data[key].(map[string]interface{})
	if !ok {
		return nil
	}
	return item["id"]
}

func issueObjectIDs(data map[string]interface{}, keys ...string) []interface{} {
	for _, key := range keys {
		items, ok := data[key].([]interface{})
		if !ok {
			continue
		}
		ids := make([]interface{}, 0, len(items))
		for _, item := range items {
			obj, ok := item.(map[string]interface{})
			if !ok {
				continue
			}
			if id, ok := obj["id"]; ok {
				ids = append(ids, id)
			}
		}
		if len(ids) > 0 {
			return ids
		}
	}
	return nil
}

func stringField(data map[string]interface{}, key string) string {
	value, _ := data[key].(string)
	return value
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

func hasIssueMetadataArgs(ctx *common.RuntimeContext) bool {
	for _, name := range []string{"priority-id", "tag-ids", "label", "assigner-ids", "branch", "start-date", "due-date"} {
		if ctx.Arg(name) != "" {
			return true
		}
	}
	return false
}

func applyIssueMetadataArgs(ctx *common.RuntimeContext, body map[string]interface{}) error {
	if priority := ctx.Arg("priority-id"); priority != "" {
		priorityID, err := parseIssueID(priority, "priority-id")
		if err != nil {
			return err
		}
		body["priority_id"] = priorityID
	}
	tagIDs := ctx.Arg("tag-ids")
	if label := ctx.Arg("label"); label != "" {
		if tagIDs != "" {
			return fmt.Errorf("--label cannot be used with --tag-ids")
		}
		tagIDs = label
	}
	if tagIDs != "" {
		ids, err := parseIssueIDList(tagIDs, "tag-ids")
		if err != nil {
			return err
		}
		body["issue_tag_ids"] = ids
	}
	if assignerIDs := ctx.Arg("assigner-ids"); assignerIDs != "" {
		ids, err := parseIssueIDList(assignerIDs, "assigner-ids")
		if err != nil {
			return err
		}
		body["assigner_ids"] = ids
	}
	if branch := ctx.Arg("branch"); branch != "" {
		body["branch_name"] = branch
	}
	if startDate := ctx.Arg("start-date"); startDate != "" {
		body["start_date"] = startDate
	}
	if dueDate := ctx.Arg("due-date"); dueDate != "" {
		body["due_date"] = dueDate
	}
	return nil
}

func parseIssueIDList(value, flagName string) ([]int, error) {
	parts := strings.Split(value, ",")
	ids := make([]int, 0, len(parts))
	for _, part := range parts {
		id, err := parseIssueID(part, flagName)
		if err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, nil
}

func parseIssueID(value, flagName string) (int, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return 0, fmt.Errorf("--%s contains an empty ID", flagName)
	}
	id, err := strconv.Atoi(trimmed)
	if err != nil || id <= 0 {
		return 0, fmt.Errorf("--%s must contain positive numeric IDs", flagName)
	}
	return id, nil
}
