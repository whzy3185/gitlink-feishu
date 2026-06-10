package workflow

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gitlink-org/gitlink-cli/internal/client"
	"github.com/gitlink-org/gitlink-cli/shortcuts/common"
)

func TestAnalyzeReviewQueuePrioritizesRiskAndSize(t *testing.T) {
	result := AnalyzeReviewQueue(ReviewQueueInput{
		Repository: "owner/repo",
		Source:     "local-json",
		PullRequests: []PRSummaryInput{
			{
				Repository: "owner/repo",
				Number:     1,
				Title:      "docs: update README",
				State:      "open",
				ChangedFiles: []PRChangedFile{
					{Filename: "README.md", Status: "modified", Additions: 8, Deletions: 1, Changes: 9},
				},
				Additions: 8,
				Deletions: 1,
			},
			{
				Repository: "owner/repo",
				Number:     2,
				Title:      "fix: avoid token leak in auth logging",
				Body:       "The current error path may expose an access token in logs.",
				State:      "open",
				ChangedFiles: []PRChangedFile{
					{Filename: "internal/client/client.go", Status: "modified", Additions: 220, Deletions: 20, Changes: 240},
				},
				Commits: []PRCommit{
					{SHA: "abc", Message: "fix: avoid token leak"},
				},
				Additions: 220,
				Deletions: 20,
			},
			{
				Repository: "owner/repo",
				Number:     3,
				Title:      "feat: add workflow review queue",
				State:      "open",
				ChangedFiles: []PRChangedFile{
					{Filename: "shortcuts/workflow/review_queue.go", Status: "added", Additions: 210, Deletions: 0, Changes: 210},
					{Filename: "shortcuts/workflow/review_queue_test.go", Status: "added", Additions: 80, Deletions: 0, Changes: 80},
				},
				Commits: []PRCommit{
					{SHA: "def", Message: "feat: add review queue"},
				},
				Additions: 290,
			},
		},
	}, "en")

	if result.TotalPRs != 3 {
		t.Fatalf("TotalPRs = %d, want 3", result.TotalPRs)
	}
	if result.Items[0].Number != 2 {
		t.Fatalf("top PR = #%d, want #2: %+v", result.Items[0].Number, result.Items)
	}
	if result.HighPriority != 1 || result.MediumPriority != 1 || result.LowPriority != 1 {
		t.Fatalf("priority counts = high:%d medium:%d low:%d, want 1/1/1", result.HighPriority, result.MediumPriority, result.LowPriority)
	}
	if len(result.TopFocus) == 0 {
		t.Fatal("expected repeated review focus hints")
	}
	if !strings.Contains(result.Items[0].SuggestedAction, "Prioritize maintainer review") {
		t.Fatalf("SuggestedAction = %q, want high-priority action", result.Items[0].SuggestedAction)
	}
}

func TestReadReviewQueueInputSupportsObjectAndArray(t *testing.T) {
	dir := t.TempDir()
	objectPath := filepath.Join(dir, "queue_object.json")
	arrayPath := filepath.Join(dir, "queue_array.json")

	writeJSONFixture(t, objectPath, ReviewQueueInput{
		Repository: "owner/repo",
		PullRequests: []PRSummaryInput{
			{Number: 7, Title: "feat: object input"},
		},
	})
	writeJSONFixture(t, arrayPath, []PRSummaryInput{
		{Number: 8, Title: "fix: array input"},
	})

	objectInput, err := readReviewQueueInput(objectPath)
	if err != nil {
		t.Fatalf("readReviewQueueInput(object) returned error: %v", err)
	}
	if objectInput.Repository != "owner/repo" || len(objectInput.PullRequests) != 1 || objectInput.PullRequests[0].Number != 7 {
		t.Fatalf("object input = %+v, want repository and PR #7", objectInput)
	}

	arrayInput, err := readReviewQueueInput(arrayPath)
	if err != nil {
		t.Fatalf("readReviewQueueInput(array) returned error: %v", err)
	}
	if arrayInput.Source != "local-json" || len(arrayInput.PullRequests) != 1 || arrayInput.PullRequests[0].Number != 8 {
		t.Fatalf("array input = %+v, want local-json PR #8", arrayInput)
	}
}

func TestFetchReviewQueuePullRequestsUsesReadOnlyQuery(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" || r.URL.Path != "/v1/owner/repo/pulls.json" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		if got := r.URL.Query().Get("state"); got != "open" {
			t.Fatalf("state query = %q, want open", got)
		}
		if got := r.URL.Query().Get("page"); got != "2" {
			t.Fatalf("page query = %q, want 2", got)
		}
		if got := r.URL.Query().Get("limit"); got != "5" {
			t.Fatalf("limit query = %q, want 5", got)
		}
		writeWorkflowJSON(t, w, map[string]interface{}{
			"pulls": []map[string]interface{}{
				{
					"pull_request_number": 11,
					"title":               "feat: remote review queue",
					"description":         "Adds queue analysis.",
					"status":              "open",
					"creator":             map[string]interface{}{"login": "alice"},
					"additions":           130,
					"deletions":           5,
				},
			},
		})
	}))
	defer server.Close()

	ctx := &common.RuntimeContext{
		Client: &client.Client{
			HTTP:    server.Client(),
			BaseURL: server.URL,
		},
		Owner: "owner",
		Repo:  "repo",
	}

	prs, owner, repo, err := fetchReviewQueuePullRequests(ctx, "open", 2, 5)
	if err != nil {
		t.Fatalf("fetchReviewQueuePullRequests returned error: %v", err)
	}
	if owner != "owner" || repo != "repo" || len(prs) != 1 {
		t.Fatalf("owner/repo/prs = %s/%s/%d, want owner/repo/1", owner, repo, len(prs))
	}
	if prs[0].Number != 11 || prs[0].Repository != "owner/repo" || prs[0].State != "open" {
		t.Fatalf("normalized PR = %+v, want owner/repo #11 open", prs[0])
	}
}

func TestRenderReviewQueueMarkdownAndTable(t *testing.T) {
	result := AnalyzeReviewQueue(ReviewQueueInput{
		Repository: "owner/repo",
		Source:     "local-json",
		PullRequests: []PRSummaryInput{
			{
				Number: 9,
				Title:  "feat: add queue command",
				State:  "open",
				ChangedFiles: []PRChangedFile{
					{Filename: "shortcuts/workflow/review_queue.go", Additions: 180, Changes: 180},
				},
				Additions: 180,
			},
		},
	}, "zh-CN")

	markdown, err := RenderReviewQueue(result, "markdown", "zh-CN")
	if err != nil {
		t.Fatalf("RenderReviewQueue markdown returned error: %v", err)
	}
	if !strings.Contains(markdown, "# PR 审查队列") || !strings.Contains(markdown, "feat: add queue command") {
		t.Fatalf("markdown output missing expected content:\n%s", markdown)
	}

	table, err := RenderReviewQueue(result, "table", "en")
	if err != nil {
		t.Fatalf("RenderReviewQueue table returned error: %v", err)
	}
	if !strings.Contains(table, "RANK") || !strings.Contains(table, "#9") {
		t.Fatalf("table output missing expected content:\n%s", table)
	}
}
