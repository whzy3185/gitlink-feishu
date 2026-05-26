package workflow

import (
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/gitlink-org/gitlink-cli/shortcuts/common"
)

type PRFetchOptions struct {
	Owner          string
	Repo           string
	Number         int
	IncludeFiles   bool
	IncludeCommits bool
	MaxFiles       int
	MaxCommits     int
}

func FetchPRSummaryInput(ctx *common.RuntimeContext, opts PRFetchOptions) (PRSummaryInput, []ScoringNote, error) {
	owner, repo, err := resolveFetchRepo(ctx, opts.Owner, opts.Repo)
	if err != nil {
		return PRSummaryInput{}, nil, err
	}
	if opts.Number <= 0 {
		return PRSummaryInput{}, nil, fmt.Errorf("pull request number is required")
	}
	if opts.MaxFiles <= 0 {
		opts.MaxFiles = 100
	}
	if opts.MaxCommits <= 0 {
		opts.MaxCommits = 100
	}

	input, err := fetchPRBase(ctx, owner, repo, opts.Number)
	if err != nil {
		return PRSummaryInput{}, nil, err
	}
	input.Repository = fmt.Sprintf("%s/%s", owner, repo)
	input.Number = opts.Number
	input.Source = "remote-read-only-fetch"

	notes := []ScoringNote{}
	if opts.IncludeFiles {
		files, err := fetchPRFiles(ctx, owner, repo, opts.Number, opts.MaxFiles)
		if err != nil {
			notes = append(notes, ScoringNote{Metric: "pr_files", Note: fmt.Sprintf("changed files probe failed: %v", err)})
		} else {
			input.ChangedFiles = files
			fillPRLineTotals(&input)
		}
	}
	if opts.IncludeCommits {
		commits, err := fetchPRCommits(ctx, owner, repo, opts.Number, opts.MaxCommits)
		if err != nil {
			notes = append(notes, ScoringNote{Metric: "pr_commits", Note: fmt.Sprintf("commits probe failed: %v", err)})
		} else {
			input.Commits = commits
		}
	}

	return input, notes, nil
}

func fetchPRBase(ctx *common.RuntimeContext, owner, repo string, number int) (PRSummaryInput, error) {
	env, err := ctx.CallAPI("GET", prPath(owner, repo, number), nil)
	if err != nil {
		return PRSummaryInput{}, fmt.Errorf("fetch pull request summary: %w", err)
	}
	item := prAPIObject(env.Data)
	if item == nil {
		return PRSummaryInput{}, fmt.Errorf("fetch pull request summary: PR response did not contain an object")
	}
	input, ok := normalizePRSummaryItem(item)
	if !ok {
		return PRSummaryInput{}, fmt.Errorf("fetch pull request summary: PR response missing title or number")
	}
	return input, nil
}

func fetchPRFiles(ctx *common.RuntimeContext, owner, repo string, number, limit int) ([]PRChangedFile, error) {
	query := url.Values{}
	query.Set("page", "1")
	query.Set("limit", fmt.Sprintf("%d", limit))
	env, err := ctx.CallAPIWithQuery("GET", prPath(owner, repo, number)+"/files", query)
	if err != nil {
		return nil, err
	}
	rawItems := apiList(env.Data)
	files := make([]PRChangedFile, 0, len(rawItems))
	for _, raw := range rawItems {
		file, ok := normalizePRChangedFile(raw)
		if !ok {
			continue
		}
		files = append(files, file)
		if len(files) >= limit {
			break
		}
	}
	return files, nil
}

func fetchPRCommits(ctx *common.RuntimeContext, owner, repo string, number, limit int) ([]PRCommit, error) {
	query := url.Values{}
	query.Set("page", "1")
	query.Set("limit", fmt.Sprintf("%d", limit))
	env, err := ctx.CallAPIWithQuery("GET", prPath(owner, repo, number)+"/commits", query)
	if err != nil {
		return nil, err
	}
	rawItems := apiList(env.Data)
	commits := make([]PRCommit, 0, len(rawItems))
	for _, raw := range rawItems {
		commit, ok := normalizePRCommit(raw)
		if !ok {
			continue
		}
		commits = append(commits, commit)
		if len(commits) >= limit {
			break
		}
	}
	return commits, nil
}

func prPath(owner, repo string, number int) string {
	return fmt.Sprintf("%s/pulls/%d", workflowRepoPath(owner, repo), number)
}

func prAPIObject(data interface{}) map[string]interface{} {
	normalized, err := normalizeAPIData(data)
	if err != nil {
		return nil
	}
	switch value := normalized.(type) {
	case map[string]interface{}:
		for _, key := range []string{"pull_request", "data", "pr"} {
			if raw, ok := value[key]; ok {
				if item := prAPIObject(raw); item != nil {
					return item
				}
			}
		}
		return value
	case []interface{}:
		if len(value) == 1 {
			if item, ok := value[0].(map[string]interface{}); ok {
				return item
			}
		}
	}
	return nil
}

func normalizePRSummaryItem(item map[string]interface{}) (PRSummaryInput, bool) {
	number := firstPRInt(item, "number", "iid", "pull_request_number")
	title := firstPRString(item, "title", "subject")
	if number == 0 && strings.TrimSpace(title) == "" {
		return PRSummaryInput{}, false
	}
	body := firstPRString(item, "body", "description", "content")
	state := firstPRString(item, "state", "status")
	author := firstPRAuthor(item)
	base := firstPRBranch(item, "base_branch", "target_branch", "base")
	head := firstPRBranch(item, "head_branch", "source_branch", "head")
	additions := firstPRInt(item, "additions", "additions_count")
	deletions := firstPRInt(item, "deletions", "deletions_count")

	return PRSummaryInput{
		Number:     number,
		Title:      title,
		Author:     author,
		State:      state,
		BaseBranch: base,
		HeadBranch: head,
		Body:       body,
		Additions:  additions,
		Deletions:  deletions,
	}, true
}

func normalizePRChangedFile(raw interface{}) (PRChangedFile, bool) {
	item, ok := raw.(map[string]interface{})
	if !ok {
		return PRChangedFile{}, false
	}
	filename := firstPRString(item, "filename", "file", "path", "new_path")
	if strings.TrimSpace(filename) == "" {
		return PRChangedFile{}, false
	}
	additions := firstPRInt(item, "additions", "additions_count")
	deletions := firstPRInt(item, "deletions", "deletions_count")
	changes := firstPRInt(item, "changes", "total_changes")
	if changes == 0 {
		changes = additions + deletions
	}
	return PRChangedFile{
		Filename:  filename,
		Status:    firstPRString(item, "status", "state"),
		Additions: additions,
		Deletions: deletions,
		Changes:   changes,
		Patch:     firstPRString(item, "patch", "diff"),
	}, true
}

func normalizePRCommit(raw interface{}) (PRCommit, bool) {
	item, ok := raw.(map[string]interface{})
	if !ok {
		return PRCommit{}, false
	}
	sha := firstPRString(item, "sha", "id")
	message := firstPRString(item, "message", "title")
	if strings.TrimSpace(sha) == "" && strings.TrimSpace(message) == "" {
		return PRCommit{}, false
	}
	return PRCommit{
		SHA:     sha,
		Message: firstLine(message),
		Author:  firstPRCommitAuthor(item),
		Date:    firstPRTime(item, "date", "committed_at", "created_at"),
	}, true
}

func fillPRLineTotals(input *PRSummaryInput) {
	if input == nil || len(input.ChangedFiles) == 0 {
		return
	}
	additions := 0
	deletions := 0
	for _, file := range input.ChangedFiles {
		additions += file.Additions
		deletions += file.Deletions
	}
	if input.Additions == 0 {
		input.Additions = additions
	}
	if input.Deletions == 0 {
		input.Deletions = deletions
	}
}

func firstPRString(item map[string]interface{}, keys ...string) string {
	for _, key := range keys {
		if value, ok := item[key]; ok {
			if s := apiString(value); s != "" {
				return s
			}
		}
	}
	return ""
}

func firstPRInt(item map[string]interface{}, keys ...string) int {
	for _, key := range keys {
		if value, ok := item[key]; ok {
			if n := apiInt(value); n != 0 {
				return n
			}
		}
	}
	return 0
}

func firstPRTime(item map[string]interface{}, keys ...string) time.Time {
	for _, key := range keys {
		if value, ok := item[key]; ok {
			if t := apiTime(value); !t.IsZero() {
				return t
			}
		}
	}
	return time.Time{}
}

func firstPRAuthor(item map[string]interface{}) string {
	for _, key := range []string{"author", "user", "creator"} {
		if value, ok := item[key]; ok {
			if s := apiAuthor(value); s != "" {
				return s
			}
		}
	}
	return ""
}

func firstPRCommitAuthor(item map[string]interface{}) string {
	for _, key := range []string{"author", "committer", "user", "creator"} {
		if value, ok := item[key]; ok {
			if s := apiAuthor(value); s != "" {
				return s
			}
		}
	}
	return ""
}

func firstPRBranch(item map[string]interface{}, keys ...string) string {
	for _, key := range keys {
		if value, ok := item[key]; ok {
			if s := prBranchString(value); s != "" {
				return s
			}
		}
	}
	return ""
}

func prBranchString(value interface{}) string {
	switch typed := value.(type) {
	case map[string]interface{}:
		for _, key := range []string{"ref", "name", "branch", "title"} {
			if s := apiString(typed[key]); s != "" {
				return s
			}
		}
		return ""
	default:
		return apiString(value)
	}
}

func firstLine(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	lines := strings.Split(value, "\n")
	return strings.TrimSpace(lines[0])
}
