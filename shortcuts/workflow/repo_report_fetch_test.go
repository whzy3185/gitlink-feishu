package workflow

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestFetchRepoReportInputPartialPRUnavailable(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == "GET" && r.URL.Path == "/v1/owner/repo.json":
			writeWorkflowJSON(t, w, map[string]interface{}{
				"updated_at":       "2026-05-20T00:00:00Z",
				"has_readme":       true,
				"has_license":      true,
				"has_contributing": true,
			})
		case r.Method == "GET" && r.URL.Path == "/v1/owner/repo/issues.json":
			writeWorkflowJSON(t, w, map[string]interface{}{"issues": []map[string]interface{}{
				{"number": 1, "title": "Install failed", "body": "error on install", "updated_at": "2026-05-20T00:00:00Z"},
			}})
		case r.Method == "GET" && r.URL.Path == "/v1/owner/repo/pulls.json":
			http.Error(w, "pulls unavailable", http.StatusServiceUnavailable)
		case r.Method == "GET" && r.URL.Path == "/owner/repo/releases.json":
			writeWorkflowJSON(t, w, map[string]interface{}{"releases": []map[string]interface{}{{"created_at": "2026-05-01T00:00:00Z"}}})
		case r.Method == "GET" && r.URL.Path == "/owner/repo/builds.json":
			writeWorkflowJSON(t, w, map[string]interface{}{"builds": []map[string]interface{}{{"status": "success"}}})
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	input, notes, err := FetchRepoReportInput(workflowTestContext(server), RepoReportFetchOptions{
		IssueLimit:    10,
		PRLimit:       10,
		StaleDays:     30,
		IncludeHealth: true,
		IncludeIssues: true,
		IncludePRs:    true,
	})
	if err != nil {
		t.Fatalf("FetchRepoReportInput returned error: %v", err)
	}
	if input.Health == nil || len(input.Issues) != 1 {
		t.Fatalf("input = %+v, want health and issues", input)
	}
	if len(input.PullRequests) != 0 {
		t.Fatalf("len(PullRequests) = %d, want 0", len(input.PullRequests))
	}
	if !hasNote(notes, "repo_report_prs") {
		t.Fatalf("notes = %+v, want repo_report_prs note", notes)
	}
}

func TestFetchRepoReportInputHealthFailureIssuesSuccess(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == "GET" && r.URL.Path == "/v1/owner/repo.json":
			http.Error(w, "repo unavailable", http.StatusInternalServerError)
		case r.Method == "GET" && r.URL.Path == "/v1/owner/repo/issues.json":
			writeWorkflowJSON(t, w, map[string]interface{}{"issues": []map[string]interface{}{
				{"number": 1, "title": "README typo", "body": "docs typo"},
			}})
		case r.Method == "GET" && r.URL.Path == "/v1/owner/repo/pulls.json":
			writeWorkflowJSON(t, w, map[string]interface{}{"pulls": []map[string]interface{}{}})
		case r.Method == "GET" && r.URL.Path == "/owner/repo/releases.json":
			writeWorkflowJSON(t, w, map[string]interface{}{"releases": []map[string]interface{}{}})
		case r.Method == "GET" && r.URL.Path == "/owner/repo/builds.json":
			writeWorkflowJSON(t, w, map[string]interface{}{"builds": []map[string]interface{}{}})
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	input, notes, err := FetchRepoReportInput(workflowTestContext(server), RepoReportFetchOptions{
		IssueLimit:    10,
		IncludeHealth: true,
		IncludeIssues: true,
		IncludePRs:    false,
	})
	if err != nil {
		t.Fatalf("FetchRepoReportInput returned error: %v", err)
	}
	if input.Health == nil || len(input.Issues) != 1 {
		t.Fatalf("input = %+v, want degraded health and one issue", input)
	}
	if !hasNote(notes, "repository") {
		t.Fatalf("notes = %+v, want repository note", notes)
	}
}

func TestFetchRepoReportInputAllSectionsFail(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "unavailable", http.StatusInternalServerError)
	}))
	defer server.Close()

	_, _, err := FetchRepoReportInput(workflowTestContext(server), RepoReportFetchOptions{
		IncludeHealth: false,
		IncludeIssues: true,
		IncludePRs:    true,
	})
	if err == nil {
		t.Fatal("FetchRepoReportInput returned nil error when all sections failed")
	}
}

func TestFetchRepoReportInputRespectsIssueLimitAndIncludeFlags(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" || r.URL.Path != "/v1/owner/repo/issues.json" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		if got := r.URL.Query().Get("limit"); got != "1" {
			t.Fatalf("issue limit = %q, want 1", got)
		}
		writeWorkflowJSON(t, w, map[string]interface{}{"issues": []map[string]interface{}{
			{"number": 1, "title": "First bug", "body": "error"},
			{"number": 2, "title": "Second bug", "body": "error"},
		}})
	}))
	defer server.Close()

	input, notes, err := FetchRepoReportInput(workflowTestContext(server), RepoReportFetchOptions{
		IssueLimit:    1,
		IncludeHealth: false,
		IncludeIssues: true,
		IncludePRs:    false,
	})
	if err != nil {
		t.Fatalf("FetchRepoReportInput returned error: %v", err)
	}
	if len(notes) != 0 {
		t.Fatalf("notes = %+v, want empty", notes)
	}
	if len(input.Issues) != 1 {
		t.Fatalf("len(Issues) = %d, want 1", len(input.Issues))
	}
}

func TestFetchRepoReportInputPRListMetadata(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" || r.URL.Path != "/v1/owner/repo/pulls.json" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		if got := r.URL.Query().Get("limit"); got != "1" {
			t.Fatalf("PR limit = %q, want 1", got)
		}
		writeWorkflowJSON(t, w, map[string]interface{}{"pulls": []map[string]interface{}{
			{"number": 1, "title": "feat: add report", "user": map[string]interface{}{"login": "alice"}},
			{"number": 2, "title": "docs: update guide"},
		}})
	}))
	defer server.Close()

	input, notes, err := FetchRepoReportInput(workflowTestContext(server), RepoReportFetchOptions{
		PRLimit:       1,
		IncludeHealth: false,
		IncludeIssues: false,
		IncludePRs:    true,
	})
	if err != nil {
		t.Fatalf("FetchRepoReportInput returned error: %v", err)
	}
	if len(input.PullRequests) != 1 {
		t.Fatalf("len(PullRequests) = %d, want 1", len(input.PullRequests))
	}
	if !hasNote(notes, "repo_report_prs") || !strings.Contains(notes[0].Note, "list metadata") {
		t.Fatalf("notes = %+v, want list metadata note", notes)
	}
}

func hasNote(notes []ScoringNote, metric string) bool {
	for _, note := range notes {
		if note.Metric == metric {
			return true
		}
	}
	return false
}
