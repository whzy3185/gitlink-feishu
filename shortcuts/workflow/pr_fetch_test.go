package workflow

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestFetchPRSummaryInputNormalizesResponseShapes(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == "GET" && r.URL.Path == "/v1/owner/repo/pulls/13.json":
			writeWorkflowJSON(t, w, map[string]interface{}{
				"data": map[string]interface{}{
					"pull_request_number": 13,
					"title":               "feat: add workflow PR summary",
					"description":         "Summarize pull requests without writing remote data.",
					"status":              "open",
					"creator":             map[string]interface{}{"name": "carol"},
					"target_branch":       "master",
					"source_branch":       "feature/pr-summary",
					"additions":           100,
					"deletions":           4,
				},
			})
		case r.Method == "GET" && r.URL.Path == "/v1/owner/repo/pulls/13/files.json":
			if got := r.URL.Query().Get("limit"); got != "100" {
				t.Fatalf("files limit = %q, want 100", got)
			}
			writeWorkflowJSON(t, w, map[string]interface{}{
				"files": []map[string]interface{}{
					{"new_path": "shortcuts/workflow/pr_summary.go", "status": "added", "additions": 80, "deletions": 0},
					{"filename": "docs/workflow-agent-design.md", "status": "modified", "additions": 20, "deletions": 4},
				},
			})
		case r.Method == "GET" && r.URL.Path == "/v1/owner/repo/pulls/13/commits.json":
			writeWorkflowJSON(t, w, []map[string]interface{}{
				{"id": "abc123", "title": "feat: add workflow PR summary", "committer": map[string]interface{}{"login": "carol"}, "created_at": "2026-05-20T10:00:00Z"},
			})
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	input, notes, err := FetchPRSummaryInput(workflowTestContext(server), PRFetchOptions{
		Number:         13,
		IncludeFiles:   true,
		IncludeCommits: true,
		MaxFiles:       100,
		MaxCommits:     100,
	})
	if err != nil {
		t.Fatalf("FetchPRSummaryInput returned error: %v", err)
	}
	if len(notes) != 0 {
		t.Fatalf("notes = %v, want empty", notes)
	}
	if input.Repository != "owner/repo" || input.Number != 13 {
		t.Fatalf("input = %+v, want owner/repo #13", input)
	}
	if input.Author != "carol" || input.BaseBranch != "master" || input.HeadBranch != "feature/pr-summary" {
		t.Fatalf("author/branches = %q %q %q, want carol master feature/pr-summary", input.Author, input.BaseBranch, input.HeadBranch)
	}
	if len(input.ChangedFiles) != 2 {
		t.Fatalf("len(ChangedFiles) = %d, want 2", len(input.ChangedFiles))
	}
	if len(input.Commits) != 1 || input.Commits[0].SHA != "abc123" || input.Commits[0].Author != "carol" {
		t.Fatalf("Commits = %+v, want normalized commit", input.Commits)
	}
}

func TestFetchPRSummaryInputPartialFilesFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == "GET" && r.URL.Path == "/v1/owner/repo/pulls/2.json":
			writeWorkflowJSON(t, w, map[string]interface{}{
				"pull_request": map[string]interface{}{
					"number": 2,
					"title":  "fix: handle API errors",
					"user":   map[string]interface{}{"login": "bob"},
				},
			})
		case r.Method == "GET" && r.URL.Path == "/v1/owner/repo/pulls/2/files.json":
			http.Error(w, "files unavailable", http.StatusInternalServerError)
		case r.Method == "GET" && r.URL.Path == "/v1/owner/repo/pulls/2/commits.json":
			writeWorkflowJSON(t, w, []map[string]interface{}{{"sha": "abc", "message": "fix: handle API errors"}})
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	input, notes, err := FetchPRSummaryInput(workflowTestContext(server), PRFetchOptions{
		Number:         2,
		IncludeFiles:   true,
		IncludeCommits: true,
		MaxFiles:       10,
		MaxCommits:     10,
	})
	if err != nil {
		t.Fatalf("FetchPRSummaryInput returned error: %v", err)
	}
	if input.Title != "fix: handle API errors" {
		t.Fatalf("Title = %q, want base PR to remain", input.Title)
	}
	if len(notes) == 0 || !strings.Contains(notes[0].Metric, "pr_files") {
		t.Fatalf("notes = %v, want pr_files note", notes)
	}
}

func TestFetchPRSummaryInputPartialCommitsFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == "GET" && r.URL.Path == "/v1/owner/repo/pulls/3.json":
			writeWorkflowJSON(t, w, map[string]interface{}{
				"number": 3,
				"title":  "docs: update README",
			})
		case r.Method == "GET" && r.URL.Path == "/v1/owner/repo/pulls/3/files.json":
			writeWorkflowJSON(t, w, map[string]interface{}{"files": []map[string]interface{}{{"filename": "README.md"}}})
		case r.Method == "GET" && r.URL.Path == "/v1/owner/repo/pulls/3/commits.json":
			http.Error(w, "commits unavailable", http.StatusServiceUnavailable)
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	input, notes, err := FetchPRSummaryInput(workflowTestContext(server), PRFetchOptions{
		Number:         3,
		IncludeFiles:   true,
		IncludeCommits: true,
		MaxFiles:       10,
		MaxCommits:     10,
	})
	if err != nil {
		t.Fatalf("FetchPRSummaryInput returned error: %v", err)
	}
	if len(input.ChangedFiles) != 1 {
		t.Fatalf("len(ChangedFiles) = %d, want 1", len(input.ChangedFiles))
	}
	if len(notes) == 0 || !strings.Contains(notes[0].Metric, "pr_commits") {
		t.Fatalf("notes = %v, want pr_commits note", notes)
	}
}

func TestFetchPRSummaryInputReportsErrorInBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeWorkflowJSON(t, w, map[string]interface{}{
			"status":  403,
			"message": "permission denied",
		})
	}))
	defer server.Close()

	_, _, err := FetchPRSummaryInput(workflowTestContext(server), PRFetchOptions{Number: 9, IncludeFiles: true})
	if err == nil {
		t.Fatal("FetchPRSummaryInput returned nil error for error-in-body")
	}
	if !strings.Contains(err.Error(), "permission denied") {
		t.Fatalf("error = %v, want permission denied", err)
	}
}

func TestFetchPRSummaryInputRespectsLimits(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == "GET" && r.URL.Path == "/v1/owner/repo/pulls/4.json":
			writeWorkflowJSON(t, w, map[string]interface{}{
				"data": map[string]interface{}{"number": 4, "title": "feat: limit lists"},
			})
		case r.Method == "GET" && r.URL.Path == "/v1/owner/repo/pulls/4/files.json":
			if got := r.URL.Query().Get("limit"); got != "1" {
				t.Fatalf("files limit = %q, want 1", got)
			}
			writeWorkflowJSON(t, w, map[string]interface{}{
				"files": []map[string]interface{}{
					{"filename": "one.go"},
					{"filename": "two.go"},
				},
			})
		case r.Method == "GET" && r.URL.Path == "/v1/owner/repo/pulls/4/commits.json":
			if got := r.URL.Query().Get("limit"); got != "1" {
				t.Fatalf("commits limit = %q, want 1", got)
			}
			writeWorkflowJSON(t, w, map[string]interface{}{
				"commits": []map[string]interface{}{
					{"sha": "one", "message": "feat: first"},
					{"sha": "two", "message": "feat: second"},
				},
			})
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	input, notes, err := FetchPRSummaryInput(workflowTestContext(server), PRFetchOptions{
		Number:         4,
		IncludeFiles:   true,
		IncludeCommits: true,
		MaxFiles:       1,
		MaxCommits:     1,
	})
	if err != nil {
		t.Fatalf("FetchPRSummaryInput returned error: %v", err)
	}
	if len(notes) != 0 {
		t.Fatalf("notes = %v, want empty", notes)
	}
	if len(input.ChangedFiles) != 1 || len(input.Commits) != 1 {
		t.Fatalf("files/commits lengths = %d/%d, want 1/1", len(input.ChangedFiles), len(input.Commits))
	}
}

func TestPrAPIObjectRecursiveUnwrap(t *testing.T) {
	got := prAPIObject(map[string]interface{}{
		"data": map[string]interface{}{"number": float64(1), "title": "test"},
	})
	if got == nil || got["number"] != float64(1) {
		t.Fatalf("recursive unwrap via data: got %v", got)
	}

	got = prAPIObject(map[string]interface{}{
		"pull_request": map[string]interface{}{"number": float64(2)},
	})
	if got == nil || got["number"] != float64(2) {
		t.Fatalf("recursive unwrap via pull_request: got %v", got)
	}

	got = prAPIObject(map[string]interface{}{
		"pr": map[string]interface{}{"number": float64(3)},
	})
	if got == nil || got["number"] != float64(3) {
		t.Fatalf("recursive unwrap via pr: got %v", got)
	}
}

func TestPrAPIObjectSlicePaths(t *testing.T) {
	got := prAPIObject([]interface{}{
		map[string]interface{}{"number": float64(42), "title": "single"},
	})
	if got == nil || got["number"] != float64(42) {
		t.Fatalf("single-element slice: got %v", got)
	}

	if got := prAPIObject([]interface{}{}); got != nil {
		t.Fatalf("empty slice: expected nil, got %v", got)
	}
	if got := prAPIObject([]interface{}{map[string]interface{}{"a": "b"}, map[string]interface{}{"c": "d"}}); got != nil {
		t.Fatalf("multi-element slice: expected nil, got %v", got)
	}
	if got := prAPIObject([]interface{}{"not-a-map"}); got != nil {
		t.Fatalf("non-map element: expected nil, got %v", got)
	}
}

func TestPrAPIObjectNonMapNonSlice(t *testing.T) {
	if got := prAPIObject(42); got != nil {
		t.Fatalf("int: expected nil, got %v", got)
	}
	if got := prAPIObject("plain string"); got != nil {
		t.Fatalf("string: expected nil, got %v", got)
	}
}

func TestPrBranchStringMapKeys(t *testing.T) {
	got := prBranchString(map[string]interface{}{"ref": "refs/heads/main", "branch": "main"})
	if got != "refs/heads/main" {
		t.Fatalf("ref key: got %q, want refs/heads/main", got)
	}

	got = prBranchString(map[string]interface{}{"name": "feature-branch"})
	if got != "feature-branch" {
		t.Fatalf("name key: got %q, want feature-branch", got)
	}

	got = prBranchString(map[string]interface{}{"branch": "dev"})
	if got != "dev" {
		t.Fatalf("branch key: got %q, want dev", got)
	}

	got = prBranchString(map[string]interface{}{"title": "My Title"})
	if got != "My Title" {
		t.Fatalf("title key: got %q, want My Title", got)
	}

	got = prBranchString(map[string]interface{}{"other": "value"})
	if got != "" {
		t.Fatalf("no matching keys: got %q, want empty", got)
	}

	got = prBranchString(map[string]interface{}{})
	if got != "" {
		t.Fatalf("empty map: got %q, want empty", got)
	}
}

func TestPrBranchStringNonMap(t *testing.T) {
	if got := prBranchString("direct-string"); got != "direct-string" {
		t.Fatalf("string: got %q, want direct-string", got)
	}
	if got := prBranchString(42); got != "42" {
		t.Fatalf("int: got %q, want 42", got)
	}
	if got := prBranchString(true); got != "true" {
		t.Fatalf("bool: got %q, want true", got)
	}
	if got := prBranchString(nil); got != "" {
		t.Fatalf("nil: got %q, want empty", got)
	}
}

func TestPrAPIObjectDoubleWrap(t *testing.T) {
	got := prAPIObject(map[string]interface{}{
		"data": map[string]interface{}{
			"pull_request": map[string]interface{}{"id": float64(99), "title": "nested"},
		},
	})
	if got == nil || got["id"] != float64(99) {
		t.Fatalf("double wrap: got %v", got)
	}
}
