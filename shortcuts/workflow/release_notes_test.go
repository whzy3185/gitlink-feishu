package workflow

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gitlink-org/gitlink-cli/internal/client"
	"github.com/gitlink-org/gitlink-cli/shortcuts/common"
)

func TestAnalyzeReleaseNotesGroupsByChangeType(t *testing.T) {
	result := AnalyzeReleaseNotes(ReleaseNotesInput{
		Repository: "owner/repo",
		Version:    "v1.2.0",
		FromRef:    "v1.1.0",
		ToRef:      "v1.2.0",
		PullRequests: []PRSummaryInput{
			{
				Number: 1,
				Title:  "feat: add project export",
				Author: "alice",
				ChangedFiles: []PRChangedFile{
					{Filename: "shortcuts/export/export.go", Additions: 120},
				},
				Commits: []PRCommit{{SHA: "abc", Message: "feat: add project export"}},
			},
			{
				Number: 2,
				Title:  "fix: avoid nil panic",
				Author: "bob",
				Body:   "Fixes a crash without breaking change.",
				ChangedFiles: []PRChangedFile{
					{Filename: "internal/client/client.go", Additions: 5, Deletions: 2},
				},
			},
			{
				Number: 3,
				Title:  "docs: update README",
				ChangedFiles: []PRChangedFile{
					{Filename: "README.md", Additions: 8},
				},
			},
		},
		Source: "local-json",
	}, "en")

	if result.TotalPRs != 3 {
		t.Fatalf("TotalPRs = %d, want 3", result.TotalPRs)
	}
	if result.Repository != "owner/repo" || result.Version != "v1.2.0" {
		t.Fatalf("unexpected metadata: %+v", result)
	}
	if len(result.Sections) < 3 {
		t.Fatalf("expected feature/fix/docs sections, got %+v", result.Sections)
	}
	if result.Sections[0].Type != PRChangeTypeFeature {
		t.Fatalf("first section = %q, want feature", result.Sections[0].Type)
	}
	if len(result.Highlights) == 0 || !strings.Contains(result.Highlights[0], "#1") {
		t.Fatalf("expected feature highlight, got %#v", result.Highlights)
	}
}

func TestReadReleaseNotesInputObjectAndArray(t *testing.T) {
	objectPath := filepath.Join(t.TempDir(), "release.json")
	writeJSONFixture(t, objectPath, ReleaseNotesInput{
		Repository:   "owner/repo",
		PullRequests: []PRSummaryInput{{Number: 1, Title: "feat: one"}},
	})
	objectInput, err := readReleaseNotesInput(objectPath)
	if err != nil {
		t.Fatalf("read object input: %v", err)
	}
	if objectInput.Repository != "owner/repo" || len(objectInput.PullRequests) != 1 {
		t.Fatalf("unexpected object input: %+v", objectInput)
	}

	arrayPath := filepath.Join(t.TempDir(), "prs.json")
	writeJSONFixture(t, arrayPath, []PRSummaryInput{{Number: 2, Title: "fix: two"}})
	arrayInput, err := readReleaseNotesInput(arrayPath)
	if err != nil {
		t.Fatalf("read array input: %v", err)
	}
	if len(arrayInput.PullRequests) != 1 || arrayInput.PullRequests[0].Number != 2 {
		t.Fatalf("unexpected array input: %+v", arrayInput)
	}
}

func TestRunReleaseNotesRemoteModeFetchesMergedPRs(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" || r.URL.Path != "/v1/owner/repo/pulls.json" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		if r.URL.Query().Get("state") != "merged" {
			t.Fatalf("state = %q, want merged", r.URL.Query().Get("state"))
		}
		if r.URL.Query().Get("page") != "2" || r.URL.Query().Get("limit") != "5" {
			t.Fatalf("unexpected pagination: %s", r.URL.RawQuery)
		}
		writeWorkflowJSON(t, w, map[string]interface{}{
			"pulls": []map[string]interface{}{
				{
					"number":      9,
					"title":       "feat: release notes",
					"description": "Add release notes generator",
					"state":       "merged",
					"author":      map[string]interface{}{"login": "alice"},
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
		Owner:  "owner",
		Repo:   "repo",
		Format: "json",
		Args: map[string]string{
			"state": "merged",
			"page":  "2",
			"limit": "5",
			"lang":  "en",
		},
	}
	if err := runReleaseNotes(ctx); err != nil {
		t.Fatalf("runReleaseNotes returned error: %v", err)
	}
}

func TestRenderReleaseNotesMarkdownAndTable(t *testing.T) {
	result := AnalyzeReleaseNotes(ReleaseNotesInput{
		Repository: "owner/repo",
		Version:    "v1.0.0",
		PullRequests: []PRSummaryInput{{
			Number: 1,
			Title:  "feat: export reports",
			Author: "alice",
		}},
	}, "en")

	markdown, err := RenderReleaseNotes(result, "markdown", "en")
	if err != nil {
		t.Fatalf("RenderReleaseNotes markdown: %v", err)
	}
	if !strings.Contains(markdown, "# Release Notes v1.0.0") || !strings.Contains(markdown, "Features") {
		t.Fatalf("unexpected markdown output:\n%s", markdown)
	}

	table, err := RenderReleaseNotes(result, "table", "en")
	if err != nil {
		t.Fatalf("RenderReleaseNotes table: %v", err)
	}
	if !bytes.Contains([]byte(table), []byte("TYPE")) || !strings.Contains(table, "#1") {
		t.Fatalf("unexpected table output:\n%s", table)
	}
}
