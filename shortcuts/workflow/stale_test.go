package workflow

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gitlink-org/gitlink-cli/internal/client"
	"github.com/gitlink-org/gitlink-cli/shortcuts/common"
)

func TestReadStaleInput(t *testing.T) {
	path := filepath.Join(t.TempDir(), "stale.json")
	writeJSONFixture(t, path, StaleInput{
		Repository: "owner/repo",
		Issues: []IssueInput{{
			Number:    1,
			Title:     "stale issue",
			State:     "open",
			UpdatedAt: time.Now().AddDate(0, 0, -45),
		}},
	})

	input, err := readStaleInput(path)
	if err != nil {
		t.Fatalf("readStaleInput returned error: %v", err)
	}
	if input.Repository != "owner/repo" {
		t.Fatalf("Repository = %q", input.Repository)
	}
	if len(input.Issues) != 1 {
		t.Fatalf("len(Issues) = %d, want 1", len(input.Issues))
	}
}

func TestAnalyzeStaleBuildsBucketsAndTopSlice(t *testing.T) {
	now := time.Now()
	report := AnalyzeStale(StaleInput{
		Repository: "owner/repo",
		Source:     "local-json",
		Issues: []IssueInput{
			{Number: 1, Title: "fresh issue", State: "open", UpdatedAt: now.AddDate(0, 0, -5)},
			{Number: 2, Title: "watch issue", State: "open", UpdatedAt: now.AddDate(0, 0, -35), Labels: []string{"bug"}},
			{Number: 3, Title: "zombie issue", State: "open", UpdatedAt: now.AddDate(0, 0, -140)},
		},
		PullRequests: []StalePullRequestInput{
			{Number: 7, Title: "stale pr", State: "open", LastActivityAt: now.AddDate(0, 0, -75), HeadBranch: "feat/stale", BaseBranch: "master"},
		},
	}, []string{"journal fallback failed"}, staleScanOptions{
		State:         "open",
		StaleDays:     30,
		Top:           2,
		IncludeIssues: true,
		IncludePRs:    true,
	}, "en")

	if report.ScannedTotal != 4 {
		t.Fatalf("ScannedTotal = %d, want 4", report.ScannedTotal)
	}
	if report.ByBucket[staleBucketFresh] != 1 || report.ByBucket[staleBucketWatch] != 1 || report.ByBucket[staleBucketStale] != 1 || report.ByBucket[staleBucketZombie] != 1 {
		t.Fatalf("ByBucket = %+v", report.ByBucket)
	}
	if report.FlaggedTotal != 3 {
		t.Fatalf("FlaggedTotal = %d, want 3", report.FlaggedTotal)
	}
	if report.ShownTotal != 2 || len(report.Items) != 2 {
		t.Fatalf("ShownTotal = %d len(items) = %d, want 2", report.ShownTotal, len(report.Items))
	}
	if report.OmittedTotal != 1 {
		t.Fatalf("OmittedTotal = %d, want 1", report.OmittedTotal)
	}
	if report.Items[0].Bucket != staleBucketZombie {
		t.Fatalf("first bucket = %q, want zombie", report.Items[0].Bucket)
	}
	if len(report.Recommendations) == 0 {
		t.Fatal("expected recommendations")
	}
	if len(report.Notes) != 1 || report.Notes[0] != "journal fallback failed" {
		t.Fatalf("Notes = %+v", report.Notes)
	}
}

func TestNormalizeIssueItemSupportsGitLinkFields(t *testing.T) {
	issue, ok := normalizeIssueItem(map[string]interface{}{
		"project_issues_index":   11,
		"subject":                "subject title",
		"description":            "body",
		"status_name":            "新增",
		"author":                 map[string]interface{}{"login": "alice"},
		"issue_tags":             []map[string]interface{}{{"name": "bug"}},
		"comment_journals_count": 3,
		"updated_at":             "2026-06-01T10:00:00Z",
	})
	if !ok {
		t.Fatal("normalizeIssueItem returned ok=false")
	}
	if issue.Number != 11 {
		t.Fatalf("Number = %d, want 11", issue.Number)
	}
	if issue.State != "新增" {
		t.Fatalf("State = %q, want 新增", issue.State)
	}
	if issue.CommentsCount != 3 {
		t.Fatalf("CommentsCount = %d, want 3", issue.CommentsCount)
	}
	if len(issue.Labels) != 1 || issue.Labels[0] != "bug" {
		t.Fatalf("Labels = %+v", issue.Labels)
	}
}

func TestFetchStaleInputRemoteUsesIssueAndPRJournalData(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == "GET" && r.URL.Path == "/v1/owner/repo/issues.json":
			writeWorkflowJSON(t, w, map[string]interface{}{
				"issues": []map[string]interface{}{
					{
						"project_issues_index":   3,
						"subject":                "old issue",
						"description":            "body",
						"status_name":            "open",
						"author":                 map[string]interface{}{"login": "alice"},
						"issue_tags":             []map[string]interface{}{{"name": "bug"}},
						"comment_journals_count": 2,
						"updated_at":             "2026-05-01T00:00:00Z",
					},
				},
			})
		case r.Method == "GET" && r.URL.Path == "/v1/owner/repo/pulls.json":
			writeWorkflowJSON(t, w, map[string]interface{}{
				"pulls": []map[string]interface{}{
					{
						"index":           7,
						"title":           "feat: old pr",
						"status":          "open",
						"base":            "master",
						"head":            "feat/old-pr",
						"pr_created_unix": 1714608000,
						"issue": map[string]interface{}{
							"id":             42,
							"author":         map[string]interface{}{"login": "bob"},
							"journals_count": 4,
						},
					},
				},
			})
		case r.Method == "GET" && r.URL.Path == "/issues/42/journals.json":
			writeWorkflowJSON(t, w, map[string]interface{}{
				"issue_journals": []map[string]interface{}{
					{"format_time": "2026-06-01 10:00"},
				},
			})
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	input, notes, err := FetchStaleInput(workflowTestContext(server), StaleFetchOptions{
		State:         "open",
		IssueLimit:    5,
		PRLimit:       5,
		IncludeIssues: true,
		IncludePRs:    true,
	})
	if err != nil {
		t.Fatalf("FetchStaleInput returned error: %v", err)
	}
	if len(notes) != 0 {
		t.Fatalf("notes = %+v, want empty", notes)
	}
	if len(input.Issues) != 1 || len(input.PullRequests) != 1 {
		t.Fatalf("input = %+v", input)
	}
	if input.Issues[0].State != "open" || input.Issues[0].CommentsCount != 2 {
		t.Fatalf("issue = %+v", input.Issues[0])
	}
	pr := input.PullRequests[0]
	if pr.Number != 7 || pr.Author != "bob" {
		t.Fatalf("pr = %+v", pr)
	}
	if pr.ActivitySource != "issue_journal" {
		t.Fatalf("ActivitySource = %q, want issue_journal", pr.ActivitySource)
	}
	if pr.LastActivityAt.IsZero() {
		t.Fatal("expected LastActivityAt from journal fallback")
	}
}

func TestFetchStaleInputAddsJournalFallbackNote(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == "GET" && r.URL.Path == "/v1/owner/repo/pulls.json":
			writeWorkflowJSON(t, w, map[string]interface{}{
				"pulls": []map[string]interface{}{
					{
						"index":           9,
						"title":           "feat: old pr",
						"status":          "open",
						"pr_created_unix": 1714608000,
						"issue": map[string]interface{}{
							"id": 51,
						},
					},
				},
			})
		case r.Method == "GET" && r.URL.Path == "/issues/51/journals.json":
			http.Error(w, "journals unavailable", http.StatusServiceUnavailable)
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	input, notes, err := FetchStaleInput(workflowTestContext(server), StaleFetchOptions{
		State:         "open",
		IssueLimit:    5,
		PRLimit:       5,
		IncludeIssues: false,
		IncludePRs:    true,
	})
	if err != nil {
		t.Fatalf("FetchStaleInput returned error: %v", err)
	}
	if len(input.PullRequests) != 1 {
		t.Fatalf("len(PullRequests) = %d, want 1", len(input.PullRequests))
	}
	if len(notes) == 0 || !strings.Contains(notes[0], "PR #9 journal fallback failed") {
		t.Fatalf("notes = %+v", notes)
	}
	if input.PullRequests[0].ActivitySource != "pr_created_unix" {
		t.Fatalf("ActivitySource = %q, want pr_created_unix", input.PullRequests[0].ActivitySource)
	}
}

func TestRunStaleFromFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "stale.json")
	writeJSONFixture(t, path, StaleInput{
		Repository: "owner/repo",
		Issues: []IssueInput{{
			Number:    1,
			Title:     "stale issue",
			State:     "open",
			UpdatedAt: time.Now().AddDate(0, 0, -45),
		}},
	})

	ctx := &common.RuntimeContext{
		Client: &client.Client{HTTP: http.DefaultClient, BaseURL: "http://localhost"},
		Format: "markdown",
		Args: map[string]string{
			"from": path,
			"lang": "en",
		},
	}
	if err := runStale(ctx); err != nil {
		t.Fatalf("runStale returned error: %v", err)
	}
}
