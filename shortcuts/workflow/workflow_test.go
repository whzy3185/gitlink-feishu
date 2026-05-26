package workflow

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/gitlink-org/gitlink-cli/internal/client"
	"github.com/gitlink-org/gitlink-cli/shortcuts/common"
)

func TestShortcutsExposesWorkflowCommands(t *testing.T) {
	shortcuts := Shortcuts()
	names := map[string]bool{}
	for _, shortcut := range shortcuts {
		names[shortcut.Name] = true
	}
	if !names["triage"] {
		t.Fatal("Shortcuts missing triage")
	}
	if !names["health"] {
		t.Fatal("Shortcuts missing health")
	}
	if !names["pr-summary"] {
		t.Fatal("Shortcuts missing pr-summary")
	}
	if !names["repo-report"] {
		t.Fatal("Shortcuts missing repo-report")
	}
}

func TestRunTriageWithSingleIssueArgs(t *testing.T) {
	ctx := &common.RuntimeContext{
		Format: "json",
		Args: map[string]string{
			"title":   "Token leaked in output",
			"body":    "A secret token leaked from logs.",
			"number":  "7",
			"state":   "open",
			"limit":   "30",
			"dry-run": "true",
			"lang":    "en",
		},
	}

	issues, err := collectIssuesFromArgs(ctx)
	if err != nil {
		t.Fatalf("collectIssuesFromArgs returned error: %v", err)
	}
	if len(issues) != 1 {
		t.Fatalf("len(issues) = %d, want 1", len(issues))
	}

	result := AnalyzeIssue(issues[0], ctx.Arg("lang"))
	if result.DetectedType != IssueTypeSecurity {
		t.Fatalf("DetectedType = %q, want %q", result.DetectedType, IssueTypeSecurity)
	}
}

func TestReadIssueInputsFromJSONFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "issues.json")
	data := []IssueInput{{
		Number: 1,
		Title:  "README typo",
		State:  "open",
	}}
	writeJSONFixture(t, path, map[string]interface{}{"issues": data})

	issues, err := readIssueInputs(path)
	if err != nil {
		t.Fatalf("readIssueInputs returned error: %v", err)
	}
	if len(issues) != 1 {
		t.Fatalf("len(issues) = %d, want 1", len(issues))
	}
	if issues[0].Title != "README typo" {
		t.Fatalf("Title = %q, want README typo", issues[0].Title)
	}
}

func TestRenderHealthMarkdown(t *testing.T) {
	result := ScoreHealth(HealthInput{
		Repository:          "owner/repo",
		OpenIssues:          1,
		OpenPRs:             1,
		RecentActivityKnown: true,
		RecentActivityDays:  1,
		ReleaseKnown:        true,
		HasRecentRelease:    true,
		HasReadme:           true,
		HasLicense:          true,
		HasContributing:     true,
		AgentReadinessKnown: true,
		AgentReadinessScore: 9,
	}, "en")

	var buf bytes.Buffer
	if err := renderHealthResult(&buf, result, "markdown"); err != nil {
		t.Fatalf("renderHealthResult returned error: %v", err)
	}
	if !bytes.Contains(buf.Bytes(), []byte("# Repository Health Report")) {
		t.Fatalf("markdown output missing title: %s", buf.String())
	}
}

func TestRunTriageRemoteModeUsesFetch(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" || r.URL.Path != "/v1/owner/repo/issues.json" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		writeWorkflowJSON(t, w, map[string]interface{}{
			"issues": []map[string]interface{}{
				{
					"number":     7,
					"title":      "Token leaked in logs",
					"body":       "The access token appears in command output.",
					"state":      "open",
					"author":     map[string]interface{}{"login": "bob"},
					"labels":     []map[string]interface{}{{"name": "security"}},
					"created_at": "2026-05-01T00:00:00Z",
					"updated_at": "2026-05-02T00:00:00Z",
					"html_url":   "https://example.com/issues/7",
					"comments":   1,
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
			"limit":   "10",
			"state":   "open",
			"dry-run": "true",
			"lang":    "en",
		},
	}

	if err := runTriage(ctx); err != nil {
		t.Fatalf("runTriage returned error: %v", err)
	}
}

func TestRunHealthRemoteModeUsesFetch(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == "GET" && r.URL.Path == "/v1/owner/repo.json":
			writeWorkflowJSON(t, w, map[string]interface{}{
				"updated_at":       "2026-05-19T00:00:00Z",
				"has_readme":       true,
				"has_license":      true,
				"has_contributing": true,
			})
		case r.Method == "GET" && r.URL.Path == "/v1/owner/repo/issues.json":
			writeWorkflowJSON(t, w, map[string]interface{}{"issues": []map[string]interface{}{}})
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

	ctx := &common.RuntimeContext{
		Client: &client.Client{
			HTTP:    server.Client(),
			BaseURL: server.URL,
		},
		Owner:  "owner",
		Repo:   "repo",
		Format: "json",
		Args: map[string]string{
			"lang": "en",
		},
	}

	if err := runHealth(ctx); err != nil {
		t.Fatalf("runHealth returned error: %v", err)
	}
}

func TestCollectRepoReportFromJSONFile(t *testing.T) {
	ctx := &common.RuntimeContext{
		Args: map[string]string{
			"from": filepath.Join("testdata", "repo_report.json"),
		},
	}
	input, notes, err := collectRepoReportInput(ctx)
	if err != nil {
		t.Fatalf("collectRepoReportInput returned error: %v", err)
	}
	if len(notes) != 0 {
		t.Fatalf("notes = %+v, want empty", notes)
	}
	if input.Repository == "" || len(input.Issues) == 0 || len(input.PullRequests) == 0 {
		t.Fatalf("input = %+v, want populated report fixture", input)
	}
}

func TestCollectRepoReportMissingInputs(t *testing.T) {
	ctx := &common.RuntimeContext{Args: map[string]string{}}
	_, _, err := collectRepoReportInput(ctx)
	if err == nil {
		t.Fatal("collectRepoReportInput returned nil error without --from or owner/repo")
	}
}

func writeJSONFixture(t *testing.T, path string, data interface{}) {
	t.Helper()
	encoded, err := json.Marshal(data)
	if err != nil {
		t.Fatalf("json.Marshal returned error: %v", err)
	}
	if err := os.WriteFile(path, encoded, 0600); err != nil {
		t.Fatalf("write fixture returned error: %v", err)
	}
}
