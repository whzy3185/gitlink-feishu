package workflow

import (
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/gitlink-org/gitlink-cli/shortcuts/common"
)

func FetchIssuesForTriage(ctx *common.RuntimeContext, opts TriageFetchOptions) ([]IssueInput, error) {
	owner, repo, err := resolveFetchRepo(ctx, opts.Owner, opts.Repo)
	if err != nil {
		return nil, err
	}

	limit := opts.Limit
	if limit <= 0 {
		limit = 30
	}
	page := opts.Page
	if page <= 0 {
		page = 1
	}
	state := strings.TrimSpace(opts.State)
	if state == "" {
		state = "open"
	}

	query := url.Values{}
	query.Set("state", state)
	query.Set("limit", fmt.Sprintf("%d", limit))
	query.Set("page", fmt.Sprintf("%d", page))
	if len(opts.Labels) > 0 {
		query.Set("labels", strings.Join(opts.Labels, ","))
	}
	if strings.TrimSpace(opts.Since) != "" {
		query.Set("since", strings.TrimSpace(opts.Since))
	}

	env, err := ctx.CallAPIWithQuery("GET", workflowRepoPath(owner, repo)+"/issues", query)
	if err != nil {
		return nil, fmt.Errorf("fetch issues for triage: %w", err)
	}

	items := apiList(env.Data)
	issues := make([]IssueInput, 0, len(items))
	for _, raw := range items {
		issue, ok := normalizeIssueItem(raw)
		if !ok {
			continue
		}
		issues = append(issues, issue)
		if len(issues) >= limit {
			break
		}
	}

	if len(issues) == 0 {
		return nil, fmt.Errorf("fetch issues for triage: no issues found in API response")
	}

	return issues, nil
}

func normalizeIssueItem(raw interface{}) (IssueInput, bool) {
	item, ok := raw.(map[string]interface{})
	if !ok {
		return IssueInput{}, false
	}

	title := firstIssueString(item, "title", "subject")
	body := firstIssueString(item, "body", "description", "content")
	if strings.TrimSpace(title) == "" && strings.TrimSpace(body) == "" {
		return IssueInput{}, false
	}

	number := firstIssueInt(item, "number", "iid", "issue_number", "project_issues_index", "id")
	id := firstIssueString(item, "id")
	if id == "" {
		id = fmt.Sprintf("%d", number)
	}
	state := firstIssueString(item, "state", "status")
	author := firstIssueString(item, "author", "user", "creator")
	urlValue := firstIssueString(item, "html_url", "url", "web_url")
	labels := firstIssueLabels(item["labels"])
	createdAt := firstIssueTime(item, "created_at", "created")
	updatedAt := firstIssueTime(item, "updated_at", "updated", "last_updated_at")
	comments := firstIssueInt(item, "comments_count", "comments")

	return IssueInput{
		ID:            id,
		Number:        number,
		Title:         title,
		Body:          body,
		State:         state,
		Author:        apiAuthor(authorValue(item, author)),
		URL:           urlValue,
		Labels:        labels,
		CreatedAt:     createdAt,
		UpdatedAt:     updatedAt,
		CommentsCount: comments,
	}, true
}

func authorValue(item map[string]interface{}, fallback string) interface{} {
	if raw, ok := item["author"]; ok {
		return raw
	}
	if raw, ok := item["user"]; ok {
		return raw
	}
	if raw, ok := item["creator"]; ok {
		return raw
	}
	return fallback
}

func firstIssueString(item map[string]interface{}, keys ...string) string {
	for _, key := range keys {
		if value, ok := item[key]; ok {
			if s := apiString(value); s != "" {
				return s
			}
		}
	}
	return ""
}

func firstIssueInt(item map[string]interface{}, keys ...string) int {
	for _, key := range keys {
		if value, ok := item[key]; ok {
			if n := apiInt(value); n != 0 {
				return n
			}
		}
	}
	return 0
}

func firstIssueTime(item map[string]interface{}, keys ...string) time.Time {
	for _, key := range keys {
		if value, ok := item[key]; ok {
			if t := apiTime(value); !t.IsZero() {
				return t
			}
		}
	}
	return time.Time{}
}

func firstIssueLabels(value interface{}) []string {
	switch labels := value.(type) {
	case []interface{}:
		out := make([]string, 0, len(labels))
		for _, label := range labels {
			if s := apiStringValue(label); s != "" {
				out = append(out, s)
			}
		}
		return out
	case []string:
		return append([]string(nil), labels...)
	case string:
		return apiStringSlice(labels)
	default:
		return nil
	}
}
