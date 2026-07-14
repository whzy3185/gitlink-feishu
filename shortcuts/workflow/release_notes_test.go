package workflow

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gitlink-org/gitlink-cli/internal/client"
	"github.com/gitlink-org/gitlink-cli/shortcuts/common"
)

func TestAnalyzeReleaseNotesCategorizesEntries(t *testing.T) {
	result := AnalyzeReleaseNotes(sampleReleaseNotesInput(), "en")
	if result.Version != "v1.2.0" {
		t.Fatalf("Version = %q, want v1.2.0", result.Version)
	}
	if result.CommitsCount != 2 || result.PullRequestsCount != 2 {
		t.Fatalf("counts = commits %d prs %d, want 2 and 2", result.CommitsCount, result.PullRequestsCount)
	}
	if result.Summary != "Analyzed 2 commits and 2 pull requests." {
		t.Fatalf("Summary = %q, want pluralized English summary", result.Summary)
	}
	if sectionItemCount(result, "features") == 0 {
		t.Fatalf("features section empty: %+v", result.Sections)
	}
	if sectionItemCount(result, "bug_fixes") == 0 {
		t.Fatalf("bug fixes section empty: %+v", result.Sections)
	}
	if sectionItemCount(result, "documentation") == 0 {
		t.Fatalf("documentation section empty: %+v", result.Sections)
	}
	if len(result.BreakingChanges) == 0 {
		t.Fatalf("BreakingChanges empty")
	}
}

func TestAnalyzeReleaseNotesEnglishSummarySingular(t *testing.T) {
	result := AnalyzeReleaseNotes(ReleaseNotesInput{
		Repository: "owner/repo",
		Version:    "v1.2.0",
		ToRef:      "master",
		Commits: []ReleaseNotesCommit{
			{SHA: "abc123", Message: "fix: normalize compare response", Author: "bob"},
		},
		PullRequests: []ReleaseNotesPR{
			{Number: 10, Title: "feat: add workflow release notes", Author: "alice"},
		},
	}, "en")

	if result.Summary != "Analyzed 1 commit and 1 pull request." {
		t.Fatalf("Summary = %q, want singular English summary", result.Summary)
	}
}

func sampleReleaseNotesInput() ReleaseNotesInput {
	return ReleaseNotesInput{
		Repository: "owner/repo",
		Version:    "v1.2.0",
		FromRef:    "v1.1.0",
		ToRef:      "master",
		PullRequests: []ReleaseNotesPR{
			{Number: 10, Title: "feat: add workflow release notes", Author: "alice"},
			{Number: 11, Title: "fix: normalize compare response", Author: "bob"},
		},
		Commits: []ReleaseNotesCommit{
			{SHA: "abc123", Message: "docs: update workflow guide", Author: "carol"},
			{SHA: "def456", Message: "BREAKING CHANGE: rename release flag", Author: "dave"},
		},
		Source: "local-json",
	}
}

func sectionItemCount(result ReleaseNotesResult, key string) int {
	for _, section := range result.Sections {
		if section.Key == key {
			return len(section.Items)
		}
	}
	return 0
}

func TestReadReleaseNotesInput(t *testing.T) {
	input, err := readReleaseNotesInput("testdata/release_notes.json")
	if err != nil {
		t.Fatalf("readReleaseNotesInput returned error: %v", err)
	}
	if input.Repository != "owner/repo" || len(input.Commits) == 0 {
		t.Fatalf("input = %+v, want populated fixture", input)
	}
}

func TestReleaseNotesShortcutFromJSONFile(t *testing.T) {
	restoreFormat := setCommandFormatForTest(t, "json")
	defer restoreFormat()

	ctx := &common.RuntimeContext{
		Format: "json",
		Args: map[string]string{
			"from": "testdata/release_notes.json",
			"lang": "en",
		},
	}

	output := captureStdout(t, func() error {
		return findWorkflowShortcut(t, "release-notes").Run(ctx)
	})
	var result ReleaseNotesResult
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatalf("json.Unmarshal returned error: %v\noutput=%s", err, output)
	}
	if result.Repository != "owner/repo" {
		t.Fatalf("Repository = %q, want owner/repo", result.Repository)
	}
}

func TestReleaseNotesShortcutRemoteFetch(t *testing.T) {
	restoreFormat := setCommandFormatForTest(t, "json")
	defer restoreFormat()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" || r.URL.Path != "/v1/owner/repo/compare.json" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		if r.URL.Query().Get("from") != "v1.1.0" || r.URL.Query().Get("to") != "master" {
			t.Fatalf("query = %s, want from/to refs", r.URL.RawQuery)
		}
		writeWorkflowJSON(t, w, map[string]interface{}{
			"commits": []map[string]interface{}{
				{"sha": "abc123", "message": "feat: add release notes", "author": map[string]interface{}{"name": "alice"}},
				{"sha": "def456", "message": "fix: compare fallback", "author": map[string]interface{}{"name": "bob"}},
			},
		})
	}))
	defer server.Close()

	ctx := &common.RuntimeContext{
		Client: &client.Client{HTTP: server.Client(), BaseURL: server.URL},
		Owner:  "owner",
		Repo:   "repo",
		Format: "json",
		Args: map[string]string{
			"from-ref":    "v1.1.0",
			"to-ref":      "master",
			"version":     "v1.2.0",
			"max-commits": "200",
			"include-prs": "true",
			"lang":        "en",
		},
	}

	output := captureStdout(t, func() error {
		return findWorkflowShortcut(t, "release-notes").Run(ctx)
	})
	var result ReleaseNotesResult
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatalf("json.Unmarshal returned error: %v\noutput=%s", err, output)
	}
	if result.Source != "remote-read-only-fetch" || result.CommitsCount != 2 {
		t.Fatalf("result = %+v, want remote source with 2 commits", result)
	}
}

func TestReleaseNotesShortcutRemoteFetchDefaultsToIncludingPullRequests(t *testing.T) {
	restoreFormat := setCommandFormatForTest(t, "json")
	defer restoreFormat()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeWorkflowJSON(t, w, map[string]interface{}{
			"commits": []map[string]interface{}{
				{"sha": "abc123", "message": "fix: compare fallback", "author": map[string]interface{}{"name": "bob"}},
			},
			"pull_requests": []map[string]interface{}{
				{"number": 12, "title": "feat: add release note workflow", "author": map[string]interface{}{"login": "alice"}},
			},
		})
	}))
	defer server.Close()

	ctx := &common.RuntimeContext{
		Client: &client.Client{HTTP: server.Client(), BaseURL: server.URL},
		Owner:  "owner",
		Repo:   "repo",
		Format: "json",
		Args: map[string]string{
			"from-ref":    "v1.1.0",
			"to-ref":      "master",
			"max-commits": "200",
			"lang":        "en",
		},
	}

	output := captureStdout(t, func() error {
		return findWorkflowShortcut(t, "release-notes").Run(ctx)
	})
	var result ReleaseNotesResult
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatalf("json.Unmarshal returned error: %v\noutput=%s", err, output)
	}
	if result.PullRequestsCount != 1 {
		t.Fatalf("PullRequestsCount = %d, want default include-prs to include one PR", result.PullRequestsCount)
	}
}
