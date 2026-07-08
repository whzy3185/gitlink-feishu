package workflow

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/gitlink-org/gitlink-cli/shortcuts/common"
)

type ReleaseNotesFetchOptions struct {
	Owner      string
	Repo       string
	FromRef    string
	ToRef      string
	Version    string
	MaxCommits int
	IncludePRs bool
}

func FetchReleaseNotesInput(ctx *common.RuntimeContext, opts ReleaseNotesFetchOptions) (ReleaseNotesInput, []ScoringNote, error) {
	owner, repo, err := resolveFetchRepo(ctx, opts.Owner, opts.Repo)
	if err != nil {
		return ReleaseNotesInput{}, nil, fmt.Errorf("workflow +release-notes remote mode requires --owner and --repo or a Git remote: %w", err)
	}
	fromRef := strings.TrimSpace(opts.FromRef)
	if fromRef == "" {
		return ReleaseNotesInput{}, nil, fmt.Errorf("workflow +release-notes remote mode requires --from-ref")
	}
	toRef := strings.TrimSpace(opts.ToRef)
	if toRef == "" {
		toRef = "master"
	}
	if opts.MaxCommits <= 0 {
		opts.MaxCommits = 200
	}

	query := url.Values{}
	query.Set("from", fromRef)
	query.Set("to", toRef)
	env, err := ctx.CallAPIWithQuery("GET", workflowRepoPath(owner, repo)+"/compare", query)
	if err != nil {
		return ReleaseNotesInput{}, nil, fmt.Errorf("fetch release notes compare: %w\nhint: use --from release_notes.json for local release notes generation", err)
	}

	commits := releaseNotesCommitsFromData(env.Data, opts.MaxCommits)
	prs := []ReleaseNotesPR{}
	if opts.IncludePRs {
		prs = releaseNotesPRsFromData(env.Data)
	}
	input := ReleaseNotesInput{
		Repository:   fmt.Sprintf("%s/%s", owner, repo),
		Version:      strings.TrimSpace(opts.Version),
		FromRef:      fromRef,
		ToRef:        toRef,
		PullRequests: prs,
		Commits:      commits,
		Source:       "remote-read-only-fetch",
	}
	notes := []ScoringNote{}
	if len(commits) == opts.MaxCommits {
		notes = append(notes, ScoringNote{Metric: "release_notes_commits", Note: fmt.Sprintf("commit list truncated to %d entries", opts.MaxCommits)})
	}
	if len(commits) == 0 && len(prs) == 0 {
		notes = append(notes, ScoringNote{Metric: "release_notes_compare", Note: "compare response contained no commits or pull requests"})
	}
	return input, uniqueScoringNotes(notes), nil
}

func releaseNotesCommitsFromData(data interface{}, limit int) []ReleaseNotesCommit {
	items := releaseNotesListFromData(data, []string{"commits", "commit_list"})
	commits := make([]ReleaseNotesCommit, 0, len(items))
	for _, raw := range items {
		commit, ok := normalizeReleaseNotesCommit(raw)
		if !ok {
			continue
		}
		commits = append(commits, commit)
		if limit > 0 && len(commits) >= limit {
			break
		}
	}
	return commits
}

func releaseNotesPRsFromData(data interface{}) []ReleaseNotesPR {
	items := releaseNotesListFromData(data, []string{"pull_requests", "pulls", "prs", "merge_requests"})
	prs := make([]ReleaseNotesPR, 0, len(items))
	for _, raw := range items {
		pr, ok := normalizeReleaseNotesPR(raw)
		if !ok {
			continue
		}
		prs = append(prs, pr)
	}
	return prs
}

func releaseNotesListFromData(data interface{}, keys []string) []interface{} {
	normalized, err := normalizeAPIData(data)
	if err != nil {
		return nil
	}
	switch value := normalized.(type) {
	case []interface{}:
		return value
	case map[string]interface{}:
		for _, key := range keys {
			if raw, ok := value[key]; ok {
				if items := apiList(raw); len(items) > 0 {
					return items
				}
			}
		}
		for _, key := range []string{"data", "compare", "result"} {
			if raw, ok := value[key]; ok {
				if items := releaseNotesListFromData(raw, keys); len(items) > 0 {
					return items
				}
			}
		}
	}
	return nil
}

func normalizeReleaseNotesCommit(raw interface{}) (ReleaseNotesCommit, bool) {
	item, ok := raw.(map[string]interface{})
	if !ok {
		return ReleaseNotesCommit{}, false
	}
	sha := firstPRString(item, "sha", "id")
	message := firstPRString(item, "message", "title", "subject")
	if strings.TrimSpace(sha) == "" && strings.TrimSpace(message) == "" {
		return ReleaseNotesCommit{}, false
	}
	return ReleaseNotesCommit{
		SHA:     sha,
		Message: firstLine(message),
		Author:  firstPRCommitAuthor(item),
		URL:     firstPRString(item, "html_url", "url"),
		Files:   releaseNotesFiles(item),
	}, true
}

func normalizeReleaseNotesPR(raw interface{}) (ReleaseNotesPR, bool) {
	item, ok := raw.(map[string]interface{})
	if !ok {
		return ReleaseNotesPR{}, false
	}
	number := firstPRInt(item, "number", "iid", "pull_request_number")
	title := firstPRString(item, "title", "subject")
	if number == 0 && strings.TrimSpace(title) == "" {
		return ReleaseNotesPR{}, false
	}
	return ReleaseNotesPR{
		Number: number,
		Title:  title,
		Author: firstPRAuthor(item),
		URL:    firstPRString(item, "html_url", "url"),
		Files:  releaseNotesFiles(item),
	}, true
}

func releaseNotesFiles(item map[string]interface{}) []string {
	rawItems := apiList(item["files"])
	files := make([]string, 0, len(rawItems))
	for _, raw := range rawItems {
		switch typed := raw.(type) {
		case string:
			if strings.TrimSpace(typed) != "" {
				files = append(files, typed)
			}
		case map[string]interface{}:
			if file := firstPRString(typed, "filename", "file", "path", "new_path"); file != "" {
				files = append(files, file)
			}
		}
	}
	return files
}
