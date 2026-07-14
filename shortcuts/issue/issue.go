package issue

import (
	"errors"
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
	Subject        string
	Description    string
	StatusID       interface{}
	PriorityID     interface{}
	TagIDs         []interface{}
	AssignerIDs    []interface{}
	AssignedToID   interface{}
	FixedVersionID interface{}
	TrackerID      interface{}
	IssueType      interface{}
	BranchName     string
	StartDate      string
	DueDate        string
}

func legacyIssuePath(ctx *common.RuntimeContext, number string) string {
	return fmt.Sprintf("%s/issues/%s", ctx.RepoPath(), number)
}

func legacyIssueEditPath(ctx *common.RuntimeContext, number string) string {
	return legacyIssuePath(ctx, number) + "/edit"
}

func Shortcuts(translators ...*i18n.Translator) []*common.Shortcut {
	tr := shortcutTranslator(translators...)
	return []*common.Shortcut{
		newBatchCloseShortcut(),
		newBatchReopenShortcut(),
		newBatchCommentShortcut(),
		newBatchUpdateShortcut(),
		newBatchDeleteShortcut(),
		newExportShortcut(tr),
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
				enrichIssueView(ctx, number, env)
				return ctx.Output(env)
			},
		},
		{
			Name:        "close",
			Description: tr.T("cmd.issue.close.short"),
			Flags:       issueNumberFlags(),
			Run: func(ctx *common.RuntimeContext) error {
				return setIssueStatus(ctx, 5) // 5 = closed
			},
		},
		{
			Name:        "reopen",
			Description: tr.T("cmd.issue.reopen.short"),
			Flags:       issueNumberFlags(),
			Run: func(ctx *common.RuntimeContext) error {
				return setIssueStatus(ctx, 1) // 1 = open
			},
		},
		{
			Name:        "delete",
			Description: tr.T("cmd.issue.delete.short"),
			Flags: appendIssueNumberFlags(
				common.Flag{Name: "yes", Usage: tr.T("flag.issue.delete.yes"), Bool: true, Default: "false"},
			),
			Run: func(ctx *common.RuntimeContext) error {
				if err := ctx.ResolveOwnerRepo(); err != nil {
					return err
				}
				number, err := issueNumberArg(ctx)
				if err != nil {
					return err
				}
				if !parseBool(ctx.Arg("yes")) {
					return fmt.Errorf("delete is destructive; pass --yes to confirm deleting issue #%s", number)
				}
				env, err := ctx.CallAPI("DELETE", fmt.Sprintf("%s/issues/%s", v1RepoPath(ctx), number), nil)
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
				common.Flag{Name: "parent-id", Usage: "Parent comment ID for a threaded reply"},
				common.Flag{Name: "reply-id", Usage: "Comment ID being replied to"},
				common.Flag{Name: "attachment-ids", Usage: "Comma-separated attachment IDs"},
				common.Flag{Name: "receivers", Usage: "Comma-separated user logins to mention"},
			),
			Run: runIssueComment,
		},
		{
			Name:        "comments",
			Description: "List issue comments and operation records",
			Flags: appendIssueNumberFlags(
				common.Flag{Name: "category", Short: "c", Usage: "Filter by all, comment, or operate", Default: "comment"},
				common.Flag{Name: "keyword", Short: "k", Usage: "Search comment content"},
				common.Flag{Name: "sort-by", Usage: "Sort field: created_on or updated_on"},
				common.Flag{Name: "sort-direction", Usage: "Sort direction: asc or desc"},
				common.Flag{Name: "page", Short: "p", Usage: "Page number", Default: "1"},
				common.Flag{Name: "limit", Short: "l", Usage: "Items per page", Default: "20"},
			),
			Run: runIssueComments,
		},
		{
			Name:        "comment-update",
			Description: "Update an issue comment",
			Flags: appendIssueNumberFlags(
				common.Flag{Name: "comment-id", Usage: "Comment ID", Required: true},
				common.Flag{Name: "body", Short: "b", Usage: tr.T("flag.comment.body"), Required: true},
				common.Flag{Name: "attachment-ids", Usage: "Comma-separated attachment IDs"},
				common.Flag{Name: "receivers", Usage: "Comma-separated user logins to mention"},
			),
			Run: runIssueCommentUpdate,
		},
		{
			Name:        "comment-delete",
			Description: "Delete an issue comment",
			Flags: appendIssueNumberFlags(
				common.Flag{Name: "comment-id", Usage: "Comment ID", Required: true},
			),
			Run: runIssueCommentDelete,
		},
		{
			Name:        "comment-replies",
			Description: "List replies under an issue comment",
			Flags: appendIssueNumberFlags(
				common.Flag{Name: "comment-id", Usage: "Parent comment ID", Required: true},
				common.Flag{Name: "keyword", Short: "k", Usage: "Search reply content"},
				common.Flag{Name: "page", Short: "p", Usage: "Page number", Default: "1"},
				common.Flag{Name: "limit", Short: "l", Usage: "Items per page", Default: "20"},
			),
			Run: runIssueCommentReplies,
		},
		{
			Name:        "comments",
			Description: tr.T("cmd.issue.comments.short"),
			Flags: appendIssueNumberFlags(
				common.Flag{Name: "keyword", Short: "k", Usage: tr.T("flag.issue.comments_keyword")},
				common.Flag{Name: "category", Usage: tr.T("flag.issue.comments_category")},
				common.Flag{Name: "page", Short: "p", Usage: tr.T("flag.page"), Default: "1"},
				common.Flag{Name: "limit", Short: "l", Usage: tr.T("flag.limit"), Default: "20"},
			),
			Run: func(ctx *common.RuntimeContext) error {
				if err := ctx.ResolveOwnerRepo(); err != nil {
					return err
				}
				number, err := issueNumberArg(ctx)
				if err != nil {
					return err
				}
				q := url.Values{}
				q.Set("page", ctx.Arg("page"))
				q.Set("limit", ctx.Arg("limit"))
				if keyword := ctx.Arg("keyword"); keyword != "" {
					q.Set("keyword", keyword)
				}
				if category := ctx.Arg("category"); category != "" {
					q.Set("category", category)
				}
				env, err := ctx.CallAPIWithQuery("GET", fmt.Sprintf("%s/issues/%s/journals", v1RepoPath(ctx), number), q)
				if err != nil {
					return err
				}
				return ctx.Output(env)
			},
		},
		{
			Name:        "comment-edit",
			Description: tr.T("cmd.issue.comment_edit.short"),
			Flags: appendIssueNumberFlags(
				common.Flag{Name: "comment-id", Short: "c", Usage: tr.T("flag.issue.comment_id"), Required: true},
				common.Flag{Name: "body", Short: "b", Usage: tr.T("flag.comment.body"), Required: true},
			),
			Run: func(ctx *common.RuntimeContext) error {
				if err := ctx.ResolveOwnerRepo(); err != nil {
					return err
				}
				number, err := issueNumberArg(ctx)
				if err != nil {
					return err
				}
				commentID, err := parseIssueID(ctx.Arg("comment-id"), "comment-id")
				if err != nil {
					return err
				}
				body, err := ctx.RequireArg("body")
				if err != nil {
					return err
				}
				env, err := ctx.CallAPI("PATCH", fmt.Sprintf("%s/issues/%s/journals/%d", v1RepoPath(ctx), number, commentID), map[string]interface{}{"notes": body})
				if err != nil {
					return err
				}
				return ctx.Output(env)
			},
		},
		{
			Name:        "comment-delete",
			Description: tr.T("cmd.issue.comment_delete.short"),
			Flags: appendIssueNumberFlags(
				common.Flag{Name: "comment-id", Short: "c", Usage: tr.T("flag.issue.comment_id"), Required: true},
			),
			Run: func(ctx *common.RuntimeContext) error {
				if err := ctx.ResolveOwnerRepo(); err != nil {
					return err
				}
				number, err := issueNumberArg(ctx)
				if err != nil {
					return err
				}
				commentID, err := parseIssueID(ctx.Arg("comment-id"), "comment-id")
				if err != nil {
					return err
				}
				env, err := ctx.CallAPI("DELETE", fmt.Sprintf("%s/issues/%s/journals/%d", v1RepoPath(ctx), number, commentID), nil)
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

func runIssueComment(ctx *common.RuntimeContext) error {
	if err := ctx.ResolveOwnerRepo(); err != nil {
		return err
	}
	number, err := issueNumberArg(ctx)
	if err != nil {
		return err
	}
	body, err := ctx.RequireArg("body")
	if err != nil {
		return err
	}
	payload, err := issueCommentPayload(ctx, body, true)
	if err != nil {
		return err
	}
	env, err := ctx.CallAPI("POST", issueJournalPath(ctx, number), payload)
	if err != nil {
		return err
	}
	return ctx.Output(env)
}

func runIssueComments(ctx *common.RuntimeContext) error {
	if err := ctx.ResolveOwnerRepo(); err != nil {
		return err
	}
	number, err := issueNumberArg(ctx)
	if err != nil {
		return err
	}
	q := url.Values{}
	setIssueQueryIfPresent(q, "category", ctx.Arg("category"))
	setIssueQueryIfPresent(q, "keyword", ctx.Arg("keyword"))
	setIssueQueryIfPresent(q, "sort_by", ctx.Arg("sort-by"))
	setIssueQueryIfPresent(q, "sort_direction", ctx.Arg("sort-direction"))
	setIssueQueryIfPresent(q, "page", ctx.Arg("page"))
	setIssueQueryIfPresent(q, "limit", ctx.Arg("limit"))
	env, err := ctx.CallAPIWithQuery("GET", issueJournalPath(ctx, number), q)
	if err != nil {
		return err
	}
	return ctx.Output(env)
}

func runIssueCommentUpdate(ctx *common.RuntimeContext) error {
	if err := ctx.ResolveOwnerRepo(); err != nil {
		return err
	}
	number, commentID, err := issueCommentTarget(ctx)
	if err != nil {
		return err
	}
	body, err := ctx.RequireArg("body")
	if err != nil {
		return err
	}
	payload, err := issueCommentPayload(ctx, body, false)
	if err != nil {
		return err
	}
	env, err := ctx.CallAPI("PATCH", issueJournalItemPath(ctx, number, commentID), payload)
	if err != nil {
		return err
	}
	return ctx.Output(env)
}

func runIssueCommentDelete(ctx *common.RuntimeContext) error {
	if err := ctx.ResolveOwnerRepo(); err != nil {
		return err
	}
	number, commentID, err := issueCommentTarget(ctx)
	if err != nil {
		return err
	}
	env, err := ctx.CallAPI("DELETE", issueJournalItemPath(ctx, number, commentID), nil)
	if err != nil {
		return err
	}
	return ctx.Output(env)
}

func runIssueCommentReplies(ctx *common.RuntimeContext) error {
	if err := ctx.ResolveOwnerRepo(); err != nil {
		return err
	}
	number, commentID, err := issueCommentTarget(ctx)
	if err != nil {
		return err
	}
	q := url.Values{}
	setIssueQueryIfPresent(q, "keyword", ctx.Arg("keyword"))
	setIssueQueryIfPresent(q, "page", ctx.Arg("page"))
	setIssueQueryIfPresent(q, "limit", ctx.Arg("limit"))
	env, err := ctx.CallAPIWithQuery("GET", issueJournalItemPath(ctx, number, commentID)+"/children_journals", q)
	if err != nil {
		return err
	}
	return ctx.Output(env)
}

func issueJournalPath(ctx *common.RuntimeContext, number string) string {
	return fmt.Sprintf("%s/issues/%s/journals", v1RepoPath(ctx), url.PathEscape(number))
}

func issueJournalItemPath(ctx *common.RuntimeContext, number, commentID string) string {
	return fmt.Sprintf("%s/%s", issueJournalPath(ctx, number), url.PathEscape(commentID))
}

func issueCommentTarget(ctx *common.RuntimeContext) (string, string, error) {
	number, err := issueNumberArg(ctx)
	if err != nil {
		return "", "", err
	}
	commentID, err := ctx.RequireArg("comment-id")
	if err != nil {
		return "", "", err
	}
	if _, err := parseIssueID(commentID, "comment-id"); err != nil {
		return "", "", err
	}
	return number, strings.TrimSpace(commentID), nil
}

func issueCommentPayload(ctx *common.RuntimeContext, body string, includeThreading bool) (map[string]interface{}, error) {
	payload := map[string]interface{}{"notes": body}
	if includeThreading {
		if parentID := ctx.Arg("parent-id"); parentID != "" {
			id, err := parseIssueID(parentID, "parent-id")
			if err != nil {
				return nil, err
			}
			payload["parent_id"] = id
		}
		if replyID := ctx.Arg("reply-id"); replyID != "" {
			id, err := parseIssueID(replyID, "reply-id")
			if err != nil {
				return nil, err
			}
			payload["reply_id"] = id
		}
	}
	if attachmentIDs := ctx.Arg("attachment-ids"); attachmentIDs != "" {
		ids, err := parseIssueIDList(attachmentIDs, "attachment-ids")
		if err != nil {
			return nil, err
		}
		payload["attachment_ids"] = ids
	}
	if receivers := parseIssueStringList(ctx.Arg("receivers")); len(receivers) > 0 {
		payload["receivers_login"] = receivers
	}
	return payload, nil
}

func setIssueQueryIfPresent(q url.Values, name, value string) {
	if strings.TrimSpace(value) != "" {
		q.Set(name, strings.TrimSpace(value))
	}
}

func parseIssueStringList(value string) []string {
	parts := strings.Split(value, ",")
	result := make([]string, 0, len(parts))
	seen := map[string]bool{}
	for _, part := range parts {
		item := strings.TrimSpace(part)
		if item == "" || seen[item] {
			continue
		}
		seen[item] = true
		result = append(result, item)
	}
	return result
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

// setIssueStatus flips an issue to statusID. The v1 PATCH is read-modify-write,
// so the current issue is fetched and its metadata replayed to avoid clearing
// fields that were not part of the status change.
func setIssueStatus(ctx *common.RuntimeContext, statusID int) error {
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
	body["status_id"] = statusID
	env, err := ctx.CallAPI("PATCH", fmt.Sprintf("%s/issues/%s", v1RepoPath(ctx), number), body)
	if err != nil {
		return err
	}
	return ctx.Output(env)
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
	editData, err := fetchLegacyIssueEdit(ctx, number)
	if err != nil {
		return nil, fmt.Errorf("fetch issue edit metadata: %w", err)
	}
	return &existingIssue{
		Subject:        subject,
		Description:    description,
		StatusID:       firstNonNil(nestedIssueID(issueData, "status"), editData["status_id"]),
		PriorityID:     firstNonNil(nestedIssueID(issueData, "priority"), editData["priority_id"]),
		TagIDs:         firstNonEmptyIDs(issueObjectIDs(issueData, "tags", "issue_tags"), issueValueIDs(editData, "issue_tags")),
		AssignerIDs:    issueObjectIDs(issueData, "assigners"),
		AssignedToID:   firstNonNil(issueData["assigned_to_id"], editData["assigned_to_id"]),
		FixedVersionID: firstNonNil(issueData["fixed_version_id"], editData["fixed_version_id"]),
		TrackerID:      firstNonNil(issueData["tracker_id"], editData["tracker_id"], nestedIssueID(issueData, "tracker")),
		IssueType:      firstNonNil(issueData["issue_type"], editData["issue_type"]),
		BranchName:     firstNonEmptyString(stringField(issueData, "branch_name"), stringField(editData, "branch_name")),
		StartDate:      firstNonEmptyString(stringField(issueData, "start_date"), stringField(editData, "start_date")),
		DueDate:        firstNonEmptyString(stringField(issueData, "due_date"), stringField(editData, "due_date")),
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
	if issue.AssignedToID != nil {
		body["assigned_to_id"] = issue.AssignedToID
	}
	if issue.FixedVersionID != nil {
		body["fixed_version_id"] = issue.FixedVersionID
	}
	if issue.TrackerID != nil {
		body["tracker_id"] = issue.TrackerID
	}
	if issue.IssueType != nil {
		body["issue_type"] = issue.IssueType
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
		items, ok := interfaceSlice(data[key])
		if !ok {
			continue
		}
		ids := make([]interface{}, 0, len(items))
		for _, item := range items {
			switch value := item.(type) {
			case float64, int, int64, string:
				ids = append(ids, value)
				continue
			}
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

func issueObjectNames(data map[string]interface{}, key string) []string {
	items, ok := interfaceSlice(data[key])
	if !ok {
		return nil
	}
	names := make([]string, 0, len(items))
	for _, item := range items {
		obj, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		if name, ok := obj["name"].(string); ok && name != "" {
			names = append(names, name)
		}
	}
	if len(names) == 0 {
		return nil
	}
	return names
}

func stringField(data map[string]interface{}, key string) string {
	value, _ := data[key].(string)
	return value
}

func mapField(data map[string]interface{}, key string) map[string]interface{} {
	item, _ := data[key].(map[string]interface{})
	return item
}

func interfaceSlice(value interface{}) ([]interface{}, bool) {
	items, ok := value.([]interface{})
	if ok {
		return items, true
	}
	switch typed := value.(type) {
	case []map[string]interface{}:
		items = make([]interface{}, 0, len(typed))
		for _, item := range typed {
			items = append(items, item)
		}
		return items, true
	}
	return nil, false
}

func issueValueIDs(data map[string]interface{}, keys ...string) []interface{} {
	return issueObjectIDs(data, keys...)
}

func firstNonNil(values ...interface{}) interface{} {
	for _, value := range values {
		if !isNilValue(value) {
			return value
		}
	}
	return nil
}

func firstNonEmptyString(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func firstNonEmptyIDs(values ...[]interface{}) []interface{} {
	for _, ids := range values {
		if len(ids) > 0 {
			return ids
		}
	}
	return nil
}

func isNilValue(value interface{}) bool {
	if value == nil {
		return true
	}
	switch typed := value.(type) {
	case map[string]interface{}:
		return len(typed) == 0
	}
	return false
}

func fetchLegacyIssueDetail(ctx *common.RuntimeContext, number string) (map[string]interface{}, error) {
	return fetchIssueMap(ctx, legacyIssuePath(ctx, number), "failed to parse legacy issue detail")
}

func fetchLegacyIssueEdit(ctx *common.RuntimeContext, number string) (map[string]interface{}, error) {
	return fetchIssueMap(ctx, legacyIssueEditPath(ctx, number), "failed to parse legacy issue edit data")
}

func fetchIssueMap(ctx *common.RuntimeContext, path, parseErr string) (map[string]interface{}, error) {
	env, err := ctx.CallAPI("GET", path, nil)
	if err != nil {
		return nil, err
	}
	data, ok := env.Data.(map[string]interface{})
	if !ok {
		return nil, errors.New(parseErr)
	}
	return data, nil
}

func enrichIssueView(ctx *common.RuntimeContext, number string, env *output.Envelope) {
	if env == nil {
		return
	}
	issueData, ok := env.Data.(map[string]interface{})
	if !ok {
		return
	}
	legacyDetail, _ := fetchLegacyIssueDetail(ctx, number)
	legacyEdit, _ := fetchLegacyIssueEdit(ctx, number)
	env.Data = mergeIssueViewData(issueData, legacyDetail, legacyEdit)
}

func mergeIssueViewData(v1Data, legacyDetail, legacyEdit map[string]interface{}) map[string]interface{} {
	issue := cloneIssueMap(v1Data)
	if issue == nil {
		return v1Data
	}

	if number := firstNonNil(issue["number"], issue["project_issues_index"], legacyDetail["project_issues_index"]); number != nil {
		issue["number"] = number
	}
	if databaseID := firstNonNil(issue["id"], legacyDetail["id"]); databaseID != nil {
		issue["database_id"] = databaseID
		delete(issue, "id")
	}

	status := firstNonNil(mapField(issue, "status"), mapField(legacyDetail, "issue_status"))
	if status != nil {
		issue["status"] = status
		if statusMap, ok := status.(map[string]interface{}); ok {
			if name := stringField(statusMap, "name"); name != "" {
				issue["status_name"] = name
			}
		}
	}
	if priority := firstNonNil(mapField(issue, "priority"), mapField(legacyDetail, "priority")); priority != nil {
		issue["priority"] = priority
		if priorityMap, ok := priority.(map[string]interface{}); ok {
			if name := stringField(priorityMap, "name"); name != "" {
				issue["priority_name"] = name
			}
		}
	}
	if tracker := firstNonNil(mapField(issue, "tracker"), mapField(legacyDetail, "tracker")); tracker != nil {
		issue["tracker"] = tracker
	}
	if trackerID := firstNonNil(issue["tracker_id"], nestedIssueID(issue, "tracker"), nestedIssueID(legacyDetail, "tracker"), legacyEdit["tracker_id"]); trackerID != nil {
		issue["tracker_id"] = trackerID
	}
	if issueType := firstNonNil(issue["issue_type"], legacyDetail["issue_type"], legacyEdit["issue_type"]); issueType != nil {
		issue["issue_type"] = issueType
	}
	if assignedToID := firstNonNil(issue["assigned_to_id"], legacyDetail["assigned_to_id"], legacyEdit["assigned_to_id"]); assignedToID != nil {
		issue["assigned_to_id"] = assignedToID
	}
	if fixedVersionID := firstNonNil(issue["fixed_version_id"], legacyDetail["fixed_version_id"], legacyDetail["version_id"], legacyEdit["fixed_version_id"]); fixedVersionID != nil {
		issue["fixed_version_id"] = fixedVersionID
	}
	if versionID := firstNonNil(issue["version_id"], legacyDetail["version_id"], legacyEdit["fixed_version_id"]); versionID != nil {
		issue["version_id"] = versionID
	}

	if tagIDs := firstNonEmptyIDs(issueObjectIDs(issue, "tags", "issue_tags"), issueObjectIDs(legacyDetail, "issue_tags"), issueValueIDs(legacyEdit, "issue_tags")); len(tagIDs) > 0 {
		issue["issue_tag_ids"] = tagIDs
	}
	if tagNames := issueObjectNames(legacyDetail, "issue_tags"); len(tagNames) > 0 {
		issue["issue_tag_names"] = tagNames
	}

	return issue
}

func cloneIssueMap(data map[string]interface{}) map[string]interface{} {
	if data == nil {
		return nil
	}
	cloned := make(map[string]interface{}, len(data))
	for key, value := range data {
		cloned[key] = value
	}
	return cloned
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
		if len(ids) == 1 {
			body["assigned_to_id"] = ids[0]
		}
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
