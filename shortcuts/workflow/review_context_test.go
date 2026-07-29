package workflow

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gitlink-org/gitlink-cli/internal/client"
	"github.com/gitlink-org/gitlink-cli/shortcuts/common"
)

func TestFetchReviewContextAllSections(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == "GET" && r.URL.Path == "/v1/owner/repo.json":
			writeWorkflowJSON(t, w, map[string]interface{}{"name": "repo", "default_branch": "master"})
		case r.Method == "GET" && r.URL.Path == "/owner/repo/pulls/7.json":
			writeWorkflowJSON(t, w, map[string]interface{}{
				"pull_request": map[string]interface{}{
					"number":          7,
					"title":           "feat: add workflow context",
					"state":           "open",
					"base_branch":     "master",
					"head_branch":     "feature/review-context",
					"head_commit_sha": "abc123def456",
					"user":            map[string]interface{}{"login": "alice"},
				},
			})
		case r.Method == "GET" && r.URL.Path == "/owner/repo/pulls/7/files.json":
			writeWorkflowJSON(t, w, map[string]interface{}{
				"files": []map[string]interface{}{
					{"filename": "shortcuts/workflow/review_context.go", "additions": 120},
				},
			})
		case r.Method == "GET" && r.URL.Path == "/v1/owner/repo/pulls/7/reviews.json":
			writeWorkflowJSON(t, w, map[string]interface{}{
				"reviews": []map[string]interface{}{
					{"id": 1, "status": "approved", "commit_id": "abc123def456", "user": map[string]interface{}{"login": "reviewer"}},
					{"id": 2, "status": "rejected", "commit_id": "old456", "user": map[string]interface{}{"login": "reviewer2"}},
				},
			})
		case r.Method == "GET" && r.URL.Path == "/v1/owner/repo/pulls/7/journals.json":
			if got := r.URL.Query().Get("is_full"); got != "true" {
				t.Fatalf("thread is_full = %q, want true", got)
			}
			if got := r.URL.Query().Get("limit"); got != "4" {
				t.Fatalf("thread limit = %q, want 4", got)
			}
			writeWorkflowJSON(t, w, map[string]interface{}{
				"journals": []map[string]interface{}{
					{"id": 10, "review_id": 1, "state": "opened", "type": "problem", "need_respond": true, "commit_id": "abc123def456", "path": "main.go"},
					{"id": 11, "review_id": 1, "state": "resolved", "commit_id": "old456", "path": "README.md"},
				},
			})
		case r.Method == "GET" && r.URL.Path == "/v1/owner/repo/issues.json":
			if got := r.URL.Query().Get("category"); got != "opened" {
				t.Fatalf("issue category = %q, want opened", got)
			}
			if got := r.URL.Query().Get("limit"); got != "2" {
				t.Fatalf("issue limit = %q, want 2", got)
			}
			writeWorkflowJSON(t, w, map[string]interface{}{
				"issues": []map[string]interface{}{
					{"number": 1, "title": "bug: install fails"},
				},
			})
		case r.Method == "GET" && r.URL.Path == "/v1/owner/repo/issue_tags.json":
			if got := r.URL.Query().Get("limit"); got != "3" {
				t.Fatalf("label limit = %q, want 3", got)
			}
			writeWorkflowJSON(t, w, map[string]interface{}{
				"issue_tags": []map[string]interface{}{
					{"id": 2, "name": "bug"},
				},
			})
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.String())
		}
	}))
	defer server.Close()

	ctx := workflowTestContext(server)
	got, err := FetchReviewContext(ctx, ReviewContextOptions{
		Number:         7,
		IssueLimit:     2,
		LabelLimit:     3,
		ThreadLimit:    4,
		IncludeRepo:    true,
		IncludePR:      true,
		IncludeFiles:   true,
		IncludeReviews: true,
		IncludeThreads: true,
		IncludeIssues:  true,
		IncludeLabels:  true,
	})
	if err != nil {
		t.Fatalf("FetchReviewContext returned error: %v", err)
	}
	if got.Repository != "owner/repo" || got.PullRequest != 7 {
		t.Fatalf("got repository=%q pr=%d", got.Repository, got.PullRequest)
	}
	if len(got.Sections) != 7 {
		t.Fatalf("sections = %v, want 7 sections", got.Sections)
	}
	if len(got.Files) != 1 || len(got.Reviews) != 2 || len(got.Threads) != 2 || len(got.OpenIssues) != 1 || len(got.Labels) != 1 {
		t.Fatalf("context lists not populated: files=%d reviews=%d threads=%d issues=%d labels=%d", len(got.Files), len(got.Reviews), len(got.Threads), len(got.OpenIssues), len(got.Labels))
	}
	if len(got.Notes) != 0 {
		t.Fatalf("notes = %+v, want empty", got.Notes)
	}
	if got.SchemaVersion != reviewContextSchemaVersion || got.CurrentHeadSHA != "abc123def456" {
		t.Fatalf("schema/head = %q/%q", got.SchemaVersion, got.CurrentHeadSHA)
	}
	if got.Summary.Decision != "changes_pending" || got.Summary.CurrentReviews != 1 || got.Summary.OutdatedReviews != 1 {
		t.Fatalf("review summary = %+v", got.Summary)
	}
	if !strings.HasPrefix(got.WorkItem.SourceFingerprint, "sha256:") {
		t.Fatalf("fingerprint = %q", got.WorkItem.SourceFingerprint)
	}
}

func TestFetchReviewContextPartialFailureKeepsNotes(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/owner/repo.json":
			writeWorkflowJSON(t, w, map[string]interface{}{"name": "repo"})
		case "/owner/repo/pulls/9/files.json":
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte("files unavailable"))
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.String())
		}
	}))
	defer server.Close()

	got, err := FetchReviewContext(workflowTestContext(server), ReviewContextOptions{
		Number:       9,
		IncludeRepo:  true,
		IncludeFiles: true,
	})
	if err != nil {
		t.Fatalf("FetchReviewContext returned error: %v", err)
	}
	if got.RepositoryInfo == nil {
		t.Fatal("expected repository info to be populated")
	}
	if len(got.Notes) != 1 || got.Notes[0].Metric != "pr_files" {
		t.Fatalf("notes = %+v, want one pr_files note", got.Notes)
	}
}

func TestFetchReviewContextAllSectionsFail(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("server error"))
	}))
	defer server.Close()

	_, err := FetchReviewContext(workflowTestContext(server), ReviewContextOptions{
		Number:      9,
		IncludeRepo: true,
	})
	if err == nil {
		t.Fatal("expected error when all enabled sections fail")
	}
}

func TestReviewContextShortcutRemoteFetchJSON(t *testing.T) {
	restoreFormat := setCommandFormatForTest(t, "json")
	defer restoreFormat()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/owner/repo.json":
			writeWorkflowJSON(t, w, map[string]interface{}{"name": "repo"})
		case "/owner/repo/pulls/3.json":
			writeWorkflowJSON(t, w, map[string]interface{}{"number": 3, "title": "fix: bug"})
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.String())
		}
	}))
	defer server.Close()

	ctx := &common.RuntimeContext{
		Client: &client.Client{
			HTTP:    server.Client(),
			BaseURL: server.URL,
		},
		Owner:  "owner",
		Repo:   "repo",
		Format: "json",
		Args: map[string]string{
			"number":          "3",
			"issue-limit":     "20",
			"label-limit":     "50",
			"include-repo":    "true",
			"include-pr":      "true",
			"include-files":   "false",
			"include-reviews": "false",
			"include-threads": "false",
			"include-issues":  "false",
			"include-labels":  "false",
		},
	}

	output := captureStdout(t, func() error {
		return findWorkflowShortcut(t, "review-context").Run(ctx)
	})
	var result ReviewContext
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatalf("json.Unmarshal returned error: %v\noutput=%s", err, output)
	}
	if result.PullRequest != 3 || result.Repository != "owner/repo" {
		t.Fatalf("result = %+v, want PR 3 owner/repo", result)
	}
	if len(result.Sections) != 2 {
		t.Fatalf("sections = %v, want repo and pr", result.Sections)
	}
}

func TestReviewContextShortcutMissingNumber(t *testing.T) {
	ctx := &common.RuntimeContext{Args: map[string]string{}}
	err := findWorkflowShortcut(t, "review-context").Run(ctx)
	if err == nil {
		t.Fatal("expected error for missing number")
	}
	if !strings.Contains(err.Error(), "requires --number") {
		t.Fatalf("error = %v, want missing number hint", err)
	}
}

func TestRenderReviewContextFormats(t *testing.T) {
	context := ReviewContext{
		Repository:  "owner/repo",
		PullRequest: 4,
		Source:      "shortcut-backed-read-only-fetch",
		Sections:    []string{"repo_info", "pr", "reviews", "threads"},
		PR: map[string]interface{}{
			"number":          4,
			"title":           "feat: stable review context",
			"state":           "open",
			"head_commit_sha": "abc123def456",
		},
		Files:   []map[string]interface{}{{"filename": "README.md"}},
		Reviews: []map[string]interface{}{{"id": 1, "status": "approved", "commit_id": "abc123def456"}},
		Threads: []ReviewContextThread{},
		Notes:   []ScoringNote{{Metric: "labels", Note: "label +list equivalent failed"}},
	}
	finalizeReviewContext(&context, time.Date(2026, 7, 29, 12, 0, 0, 0, time.UTC))

	table, err := RenderReviewContext(context, "table")
	if err != nil {
		t.Fatalf("RenderReviewContext table returned error: %v", err)
	}
	if !strings.Contains(table, "REPOSITORY") || !strings.Contains(table, "DECISION") || !strings.Contains(table, "owner/repo") {
		t.Fatalf("table output = %q", table)
	}

	markdown, err := RenderReviewContext(context, "markdown")
	if err != nil {
		t.Fatalf("RenderReviewContext markdown returned error: %v", err)
	}
	if !strings.Contains(markdown, "# PR Review Context") || !strings.Contains(markdown, "Review freshness") || !strings.Contains(markdown, "label +list") {
		t.Fatalf("markdown output = %q", markdown)
	}
}

func TestParseBoolDefault(t *testing.T) {
	if !parseBoolDefault("", true) {
		t.Fatal("empty value should use true default")
	}
	if parseBoolDefault("false", true) {
		t.Fatal("false value should override true default")
	}
}
