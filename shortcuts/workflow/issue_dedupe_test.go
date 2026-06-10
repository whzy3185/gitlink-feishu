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

func TestAnalyzeIssueDedupeFindsDuplicatePairs(t *testing.T) {
	result := AnalyzeIssueDedupe(IssueDedupeInput{
		Repository: "owner/repo",
		Source:     "local-json",
		Issues: []IssueInput{
			{Number: 1, Title: "Login fails on Windows with token auth", Body: "The token login command fails on Windows credential manager.", State: "open", Labels: []string{"bug", "auth"}},
			{Number: 2, Title: "Token login failure in Windows credential manager", Body: "Auth token login cannot complete on Windows.", State: "open", Labels: []string{"bug", "auth"}},
			{Number: 3, Title: "Add docs for release upload", Body: "Document release asset upload examples.", State: "open", Labels: []string{"docs"}},
		},
	}, 35, 10, "en")

	if result.CandidatePairs != 1 {
		t.Fatalf("CandidatePairs = %d, want 1: %+v", result.CandidatePairs, result.Pairs)
	}
	pair := result.Pairs[0]
	if pair.Primary.Number != 1 || pair.Duplicate.Number != 2 {
		t.Fatalf("pair = #%d/#%d, want #1/#2", pair.Primary.Number, pair.Duplicate.Number)
	}
	if pair.Score < 35 || len(pair.SharedTerms) == 0 {
		t.Fatalf("score/shared = %d/%v, want duplicate signal", pair.Score, pair.SharedTerms)
	}
	if len(result.Recommendations) == 0 {
		t.Fatal("expected recommendations")
	}
}

func TestCollectIssueDedupeInputFromFileFiltersStateAndLimit(t *testing.T) {
	path := filepath.Join(t.TempDir(), "issues.json")
	writeJSONFixture(t, path, map[string]interface{}{
		"issues": []IssueInput{
			{Number: 1, Title: "open issue", State: "open"},
			{Number: 2, Title: "closed issue", State: "closed"},
			{Number: 3, Title: "second open issue", State: "open"},
		},
	})
	ctx := &common.RuntimeContext{
		Args: map[string]string{
			"from":      path,
			"state":     "open",
			"limit":     "1",
			"threshold": "55",
			"max-pairs": "20",
		},
	}

	input, threshold, maxPairs, err := collectIssueDedupeInput(ctx)
	if err != nil {
		t.Fatalf("collectIssueDedupeInput returned error: %v", err)
	}
	if threshold != 55 || maxPairs != 20 {
		t.Fatalf("threshold/maxPairs = %d/%d, want 55/20", threshold, maxPairs)
	}
	if len(input.Issues) != 1 || input.Issues[0].Number != 1 {
		t.Fatalf("issues = %+v, want only first open issue", input.Issues)
	}
}

func TestCollectIssueDedupeInputRemoteFetch(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" || r.URL.Path != "/v1/owner/repo/issues.json" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		if got := r.URL.Query().Get("state"); got != "open" {
			t.Fatalf("state = %q, want open", got)
		}
		if got := r.URL.Query().Get("page"); got != "2" {
			t.Fatalf("page = %q, want 2", got)
		}
		if got := r.URL.Query().Get("limit"); got != "5" {
			t.Fatalf("limit = %q, want 5", got)
		}
		writeWorkflowJSON(t, w, map[string]interface{}{
			"issues": []map[string]interface{}{
				{"number": 7, "title": "Login fails", "description": "Token login fails", "state": "open"},
			},
		})
	}))
	defer server.Close()
	ctx := &common.RuntimeContext{
		Client: &client.Client{HTTP: server.Client(), BaseURL: server.URL},
		Owner:  "owner",
		Repo:   "repo",
		Args: map[string]string{
			"state":     "open",
			"page":      "2",
			"limit":     "5",
			"threshold": "55",
			"max-pairs": "20",
		},
	}

	input, _, _, err := collectIssueDedupeInput(ctx)
	if err != nil {
		t.Fatalf("collectIssueDedupeInput remote returned error: %v", err)
	}
	if input.Source != "remote-read-only-fetch" || len(input.Issues) != 1 || input.Issues[0].Number != 7 {
		t.Fatalf("input = %+v, want remote issue #7", input)
	}
}

func TestRenderIssueDedupeMarkdownAndTable(t *testing.T) {
	result := AnalyzeIssueDedupe(IssueDedupeInput{
		Repository: "owner/repo",
		Issues: []IssueInput{
			{Number: 1, Title: "Login token fails on Windows", Body: "credential manager auth failure", Labels: []string{"auth"}},
			{Number: 2, Title: "Windows token login auth failure", Body: "credential manager login fails", Labels: []string{"auth"}},
		},
	}, 30, 10, "zh-CN")

	markdown, err := RenderIssueDedupe(result, "markdown", "zh-CN")
	if err != nil {
		t.Fatalf("RenderIssueDedupe markdown returned error: %v", err)
	}
	if !strings.Contains(markdown, "# Issue 重复候选") || !strings.Contains(markdown, "共享关键词") {
		t.Fatalf("markdown output missing expected content:\n%s", markdown)
	}

	table, err := RenderIssueDedupe(result, "table", "en")
	if err != nil {
		t.Fatalf("RenderIssueDedupe table returned error: %v", err)
	}
	if !strings.Contains(table, "SCORE") || !strings.Contains(table, "#1") {
		t.Fatalf("table output missing expected content:\n%s", table)
	}
}

func TestShortcutsExposeIssueDedupe(t *testing.T) {
	names := map[string]bool{}
	for _, shortcut := range Shortcuts() {
		names[shortcut.Name] = true
	}
	if !names["issue-dedupe"] {
		t.Fatal("Shortcuts missing issue-dedupe")
	}
}
