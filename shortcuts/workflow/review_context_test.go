package workflow

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

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
					"number": 7,
					"title":  "feat: add workflow context",
					"user":   map[string]interface{}{"login": "alice"},
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
					{"id": 1, "status": "approved", "user": map[string]interface{}{"login": "reviewer"}},
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
		IncludeRepo:    true,
		IncludePR:      true,
		IncludeFiles:   true,
		IncludeReviews: true,
		IncludeIssues:  true,
		IncludeLabels:  true,
	})
	if err != nil {
		t.Fatalf("FetchReviewContext returned error: %v", err)
	}
	if got.Repository != "owner/repo" || got.PullRequest != 7 {
		t.Fatalf("got repository=%q pr=%d", got.Repository, got.PullRequest)
	}
	if len(got.Sections) != 6 {
		t.Fatalf("sections = %v, want 6 sections", got.Sections)
	}
	if len(got.Files) != 1 || len(got.Reviews) != 1 || len(got.OpenIssues) != 1 || len(got.Labels) != 1 {
		t.Fatalf("context lists not populated: files=%d reviews=%d issues=%d labels=%d", len(got.Files), len(got.Reviews), len(got.OpenIssues), len(got.Labels))
	}
	if len(got.Notes) != 0 {
		t.Fatalf("notes = %+v, want empty", got.Notes)
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
		Sections:    []string{"repo_info", "pr"},
		Files:       []map[string]interface{}{{"filename": "README.md"}},
		Notes:       []ScoringNote{{Metric: "labels", Note: "label +list equivalent failed"}},
	}

	table, err := RenderReviewContext(context, "table")
	if err != nil {
		t.Fatalf("RenderReviewContext table returned error: %v", err)
	}
	if !strings.Contains(table, "REPOSITORY") || !strings.Contains(table, "owner/repo") {
		t.Fatalf("table output = %q", table)
	}

	markdown, err := RenderReviewContext(context, "markdown")
	if err != nil {
		t.Fatalf("RenderReviewContext markdown returned error: %v", err)
	}
	if !strings.Contains(markdown, "# PR Review Context") || !strings.Contains(markdown, "label +list") {
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
