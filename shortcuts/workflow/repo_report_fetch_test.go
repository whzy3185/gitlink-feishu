package workflow

import (
	"fmt"
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

func TestFetchRepoReportInputPaginatesAllOpenItemsByDefault(t *testing.T) {
	prPages := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/owner/repo/issues.json":
			if got := r.URL.Query().Get("category"); got != "opened" {
				t.Fatalf("issue category = %q, want opened", got)
			}
			writeWorkflowJSON(t, w, map[string]interface{}{
				"total_count": 2,
				"issues": []map[string]interface{}{
					{"id": 1, "number": 1, "title": "First open issue"},
					{"id": 2, "number": 2, "title": "Second open issue"},
				},
			})
		case "/v1/owner/repo/pulls.json":
			if got := r.URL.Query().Get("status"); got != "0" {
				t.Fatalf("PR status = %q, want 0", got)
			}
			page := mustParseInt(r.URL.Query().Get("page"), 1)
			prPages++
			start := (page - 1) * 50
			end := start + 50
			if end > 120 {
				end = 120
			}
			pulls := make([]map[string]interface{}, 0, end-start)
			for index := start + 1; index <= end; index++ {
				pulls = append(pulls, map[string]interface{}{
					"id":    1000 + index,
					"index": index,
					"title": fmt.Sprintf("PR %d", index),
				})
			}
			writeWorkflowJSON(t, w, map[string]interface{}{
				"total_count": 120,
				"pulls":       pulls,
			})
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	input, _, err := FetchRepoReportInput(workflowTestContext(server), RepoReportFetchOptions{
		IncludeHealth: false,
		IncludeIssues: true,
		IncludePRs:    true,
	})
	if err != nil {
		t.Fatalf("FetchRepoReportInput returned error: %v", err)
	}
	if len(input.Issues) != 2 {
		t.Fatalf("len(Issues) = %d, want 2", len(input.Issues))
	}
	if len(input.PullRequests) != 120 {
		t.Fatalf("len(PullRequests) = %d, want 120", len(input.PullRequests))
	}
	if prPages != 3 {
		t.Fatalf("PR pages = %d, want 3", prPages)
	}
	if input.PullRequests[119].Number != 120 {
		t.Fatalf("last PR number = %d, want 120", input.PullRequests[119].Number)
	}
}

func TestFetchRepoReportInputIncludesPRLifecycleTotals(t *testing.T) {
	openCalls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" || r.URL.Path != "/v1/owner/repo/pulls.json" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		if got := r.URL.Query().Get("limit"); got != "1" {
			t.Fatalf("PR limit = %q, want 1", got)
		}
		switch r.URL.Query().Get("status") {
		case "0":
			openCalls++
			writeWorkflowJSON(t, w, map[string]interface{}{
				"total_count": 12,
				"pulls": []map[string]interface{}{
					{"id": 1001, "index": 1, "title": "Open PR"},
				},
			})
		case "1":
			writeWorkflowJSON(t, w, map[string]interface{}{
				"total_count": 5,
				"pulls":       []map[string]interface{}{},
			})
		case "2":
			writeWorkflowJSON(t, w, map[string]interface{}{
				"total_count": 7,
				"pulls":       []map[string]interface{}{},
			})
		default:
			t.Fatalf("unexpected PR status query: %q", r.URL.Query().Get("status"))
		}
	}))
	defer server.Close()

	input, notes, err := FetchRepoReportInput(workflowTestContext(server), RepoReportFetchOptions{
		PRLimit:            1,
		IncludeHealth:      false,
		IncludeIssues:      false,
		IncludePRs:         true,
		IncludePRLifecycle: true,
	})
	if err != nil {
		t.Fatalf("FetchRepoReportInput returned error: %v", err)
	}
	if len(input.PullRequests) != 1 {
		t.Fatalf("len(PullRequests) = %d, want 1", len(input.PullRequests))
	}
	if input.PRLifecycle == nil {
		t.Fatal("PRLifecycle is nil, want totals")
	}
	if input.PRLifecycle.Open != 12 || input.PRLifecycle.Merged != 5 || input.PRLifecycle.ClosedOrRejected != 7 || input.PRLifecycle.Total != 24 {
		t.Fatalf("PRLifecycle = %+v, want open=12 merged=5 closed=7 total=24", input.PRLifecycle)
	}
	if openCalls != 2 {
		t.Fatalf("open status calls = %d, want 2 for list fetch plus lifecycle total", openCalls)
	}
	if !hasNote(notes, "repo_report_prs") {
		t.Fatalf("notes = %+v, want repo_report_prs note", notes)
	}
}

func TestFetchRepoReportInputIncludesPRReviewAudit(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == "GET" && r.URL.Path == "/v1/owner/repo/pulls.json":
			writeWorkflowJSON(t, w, map[string]interface{}{
				"total_count": 2,
				"pulls": []map[string]interface{}{
					{
						"id":         1001,
						"index":      1,
						"title":      "feat: reviewed change",
						"author":     map[string]interface{}{"login": "alice"},
						"issue":      map[string]interface{}{"id": 501},
						"updated_at": "2026-06-02T12:00:00Z",
					},
					{"id": 1002, "index": 2, "title": "docs: unreviewed change", "author": map[string]interface{}{"login": "dana"}, "issue": map[string]interface{}{"id": 502}},
				},
			})
		case r.Method == "GET" && r.URL.Path == "/v1/owner/repo/pulls/1/reviews.json":
			writeWorkflowJSON(t, w, map[string]interface{}{"reviews": []map[string]interface{}{
				{"reviewer": map[string]interface{}{"login": "bob"}, "status": "approved", "content": "looks good", "created_at": "2026-06-01T10:00:00Z"},
			}})
		case r.Method == "GET" && r.URL.Path == "/v1/owner/repo/pulls/2/reviews.json":
			writeWorkflowJSON(t, w, map[string]interface{}{"reviews": []map[string]interface{}{}})
		case r.Method == "GET" && r.URL.Path == "/v1/owner/repo/issues/501/journals.json":
			writeWorkflowJSON(t, w, map[string]interface{}{"journals": []map[string]interface{}{
				{"user": map[string]interface{}{"login": "alice"}, "notes": "I updated the branch", "operate_category": "comment", "created_at": "2026-06-02T11:00:00Z"},
				{"user": map[string]interface{}{"login": "bob"}, "notes": "Please keep this test", "operate_category": "comment", "created_at": "2026-06-01T09:30:00Z"},
				{"user": map[string]interface{}{"login": "carol"}, "notes": "I can reproduce this", "operate_category": "comment", "created_at": "2026-06-01T12:00:00Z"},
				{"user": map[string]interface{}{"login": "system"}, "operate_content": "status changed", "operate_category": "status"},
			}})
		case r.Method == "GET" && r.URL.Path == "/v1/owner/repo/issues/502/journals.json":
			writeWorkflowJSON(t, w, map[string]interface{}{"journals": []map[string]interface{}{
				{"user": map[string]interface{}{"login": "dana"}, "notes": "Initial description", "operate_category": "comment"},
			}})
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	input, notes, err := FetchRepoReportInput(workflowTestContext(server), RepoReportFetchOptions{
		IncludeHealth:        false,
		IncludeIssues:        false,
		IncludePRs:           true,
		IncludePRReviewAudit: true,
	})
	if err != nil {
		t.Fatalf("FetchRepoReportInput returned error: %v", err)
	}
	if len(notes) != 1 || notes[0].Metric != "repo_report_prs" {
		t.Fatalf("notes = %+v, want only list metadata note", notes)
	}
	audit := input.PRReviewAudit
	if audit == nil {
		t.Fatal("PRReviewAudit is nil")
	}
	if audit.Audited != 2 || audit.Reviewed != 1 || audit.Unreviewed != 1 || audit.NeedsReReview != 1 || audit.FormalReviews != 1 {
		t.Fatalf("audit summary = %+v, want audited=2 reviewed=1 unreviewed=1 formal=1", audit)
	}
	if audit.SubmitterComments != 2 || audit.ReviewerComments != 1 || audit.ParticipantComments != 1 || audit.SystemEvents != 1 {
		t.Fatalf("actor counts = submitter:%d reviewer:%d participant:%d system:%d",
			audit.SubmitterComments, audit.ReviewerComments, audit.ParticipantComments, audit.SystemEvents)
	}
	first := audit.PullRequests[0]
	if !first.Reviewed || first.ReviewStandard != reviewStandardFormal || first.FormalReviewStatus != "approved" {
		t.Fatalf("first audit = %+v, want formal approved review", first)
	}
	if !first.NeedsReReview {
		t.Fatalf("first audit = %+v, want needs_re_review after submitter update", first)
	}
	second := audit.PullRequests[1]
	if second.Reviewed || second.ReviewStandard != reviewStandardUnreviewed {
		t.Fatalf("second audit = %+v, want unreviewed despite submitter comment", second)
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
