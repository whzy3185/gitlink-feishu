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

func TestFetchIssuesForTriageNormalizesAPIResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" || r.URL.Path != "/v1/owner/repo/issues.json" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		if got := r.URL.Query().Get("state"); got != "open" {
			t.Fatalf("state query = %q, want open", got)
		}
		if got := r.URL.Query().Get("limit"); got != "30" {
			t.Fatalf("limit query = %q, want 30", got)
		}
		writeWorkflowJSON(t, w, map[string]interface{}{
			"issues": []map[string]interface{}{
				{
					"id":                   12345,
					"project_issues_index": 7,
					"subject":              "Crash on login",
					"description":          "panic error when running login",
					"status":               "open",
					"author":               map[string]interface{}{"login": "alice"},
					"labels":               []map[string]interface{}{{"name": "bug"}, {"name": "login"}},
					"created_at":           "2026-05-01T10:00:00Z",
					"updated_at":           "2026-05-02T10:00:00Z",
					"comments_count":       2,
					"html_url":             "https://www.gitlink.org.cn/owner/repo/issues/7",
				},
			},
		})
	}))
	defer server.Close()

	ctx := workflowTestContext(server)
	issues, err := FetchIssuesForTriage(ctx, TriageFetchOptions{State: "open", Limit: 30, Page: 1})
	if err != nil {
		t.Fatalf("FetchIssuesForTriage returned error: %v", err)
	}
	if len(issues) != 1 {
		t.Fatalf("len(issues) = %d, want 1", len(issues))
	}
	issue := issues[0]
	if issue.ID != "12345" {
		t.Fatalf("ID = %q, want 12345", issue.ID)
	}
	if issue.Number != 7 {
		t.Fatalf("Number = %d, want 7", issue.Number)
	}
	if issue.Title != "Crash on login" {
		t.Fatalf("Title = %q, want Crash on login", issue.Title)
	}
	if issue.Author != "alice" {
		t.Fatalf("Author = %q, want alice", issue.Author)
	}
	if len(issue.Labels) != 2 || issue.Labels[0] != "bug" || issue.Labels[1] != "login" {
		t.Fatalf("Labels = %v, want [bug login]", issue.Labels)
	}
	if issue.CreatedAt.IsZero() || issue.UpdatedAt.IsZero() {
		t.Fatalf("expected parsed timestamps, got created=%v updated=%v", issue.CreatedAt, issue.UpdatedAt)
	}
}

func TestFetchIssuesForTriageSupportsDataString(t *testing.T) {
	payload, err := json.Marshal([]map[string]interface{}{
		{
			"number": 3,
			"title":  "README typo",
			"body":   "documentation example typo",
			"state":  "open",
		},
	})
	if err != nil {
		t.Fatalf("json.Marshal returned error: %v", err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" || r.URL.Path != "/v1/owner/repo/issues.json" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		writeWorkflowJSON(t, w, map[string]interface{}{"data": string(payload)})
	}))
	defer server.Close()

	issues, err := FetchIssuesForTriage(workflowTestContext(server), TriageFetchOptions{Limit: 10})
	if err != nil {
		t.Fatalf("FetchIssuesForTriage returned error: %v", err)
	}
	if len(issues) != 1 {
		t.Fatalf("len(issues) = %d, want 1", len(issues))
	}
	if issues[0].Number != 3 || issues[0].Title != "README typo" {
		t.Fatalf("issue = %+v, want number 3 title README typo", issues[0])
	}
}

func TestFetchIssuesForTriageEmptyResponseReturnsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeWorkflowJSON(t, w, map[string]interface{}{"issues": []map[string]interface{}{}})
	}))
	defer server.Close()

	_, err := FetchIssuesForTriage(workflowTestContext(server), TriageFetchOptions{Limit: 10})
	if err == nil {
		t.Fatal("FetchIssuesForTriage returned nil error for empty response")
	}
	if !strings.Contains(err.Error(), "no issues found") {
		t.Fatalf("error = %v, want empty-response message", err)
	}
}

func TestFetchIssuesForTriageNormalizesLabelAndAuthorShapes(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("page"); got != "1" {
			t.Fatalf("page query = %q, want 1", got)
		}
		if got := r.URL.Query().Get("limit"); got != "3" {
			t.Fatalf("limit query = %q, want 3", got)
		}
		writeWorkflowJSON(t, w, map[string]interface{}{
			"issues": []map[string]interface{}{
				{
					"number": 1,
					"body":   "panic on install",
					"labels": []string{"bug", "help wanted"},
					"author": "alice",
				},
				{
					"number": 2,
					"title":  "token leak",
					"labels": []map[string]interface{}{{"name": "bug"}, {"name": "security"}},
					"user":   map[string]interface{}{"login": "bob"},
				},
				{
					"number":  3,
					"title":   "README typo",
					"labels":  []map[string]interface{}{{"title": "docs"}},
					"creator": map[string]interface{}{"name": "carol"},
				},
			},
		})
	}))
	defer server.Close()

	issues, err := FetchIssuesForTriage(workflowTestContext(server), TriageFetchOptions{State: "open", Limit: 3, Page: 1})
	if err != nil {
		t.Fatalf("FetchIssuesForTriage returned error: %v", err)
	}
	if len(issues) != 3 {
		t.Fatalf("len(issues) = %d, want 3", len(issues))
	}
	if issues[0].Title != "" || issues[0].Body != "panic on install" {
		t.Fatalf("issue[0] = %+v, want body-only item with empty title", issues[0])
	}
	if got := strings.Join(issues[0].Labels, ","); got != "bug,help wanted" {
		t.Fatalf("issue[0].Labels = %q, want bug,help wanted", got)
	}
	if issues[0].Author != "alice" {
		t.Fatalf("issue[0].Author = %q, want alice", issues[0].Author)
	}
	if got := strings.Join(issues[1].Labels, ","); got != "bug,security" {
		t.Fatalf("issue[1].Labels = %q, want bug,security", got)
	}
	if issues[1].Author != "bob" {
		t.Fatalf("issue[1].Author = %q, want bob", issues[1].Author)
	}
	if got := strings.Join(issues[2].Labels, ","); got != "docs" {
		t.Fatalf("issue[2].Labels = %q, want docs", got)
	}
	if issues[2].Author != "carol" {
		t.Fatalf("issue[2].Author = %q, want carol", issues[2].Author)
	}
}

func TestFetchIssuesForTriageRespectsLimitAndReportsRequestErrors(t *testing.T) {
	requestCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		if got := r.URL.Query().Get("page"); got != "2" {
			t.Fatalf("page query = %q, want 2", got)
		}
		if got := r.URL.Query().Get("limit"); got != "1" {
			t.Fatalf("limit query = %q, want 1", got)
		}
		writeWorkflowJSON(t, w, map[string]interface{}{
			"issues": []map[string]interface{}{
				{"number": 1, "title": "first"},
				{"number": 2, "title": "second"},
			},
		})
	}))
	defer server.Close()

	issues, err := FetchIssuesForTriage(workflowTestContext(server), TriageFetchOptions{State: "open", Limit: 1, Page: 2})
	if err != nil {
		t.Fatalf("FetchIssuesForTriage returned error: %v", err)
	}
	if requestCount != 1 {
		t.Fatalf("requestCount = %d, want 1", requestCount)
	}
	if len(issues) != 1 {
		t.Fatalf("len(issues) = %d, want 1", len(issues))
	}
	if issues[0].Number != 1 {
		t.Fatalf("issues[0].Number = %d, want 1", issues[0].Number)
	}
}

func TestFetchIssuesForTriageReportsGitLinkErrorInBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		writeWorkflowJSON(t, w, map[string]interface{}{
			"status":  403,
			"message": "permission denied",
		})
	}))
	defer server.Close()

	_, err := FetchIssuesForTriage(workflowTestContext(server), TriageFetchOptions{Limit: 10})
	if err == nil {
		t.Fatal("FetchIssuesForTriage returned nil error for error-in-body response")
	}
	if !strings.Contains(err.Error(), "permission denied") {
		t.Fatalf("error = %v, want permission denied", err)
	}
}

func workflowTestContext(server *httptest.Server) *common.RuntimeContext {
	return &common.RuntimeContext{
		Client: &client.Client{
			HTTP:    server.Client(),
			BaseURL: server.URL,
		},
		Owner:  "owner",
		Repo:   "repo",
		Format: "json",
		Args:   map[string]string{},
	}
}

func writeWorkflowJSON(t *testing.T, w http.ResponseWriter, payload interface{}) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		t.Fatalf("failed to write response: %v", err)
	}
}
