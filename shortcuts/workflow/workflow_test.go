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

func TestReadIssueInputsSingleObject(t *testing.T) {
	path := filepath.Join(t.TempDir(), "issue.json")
	writeJSONFixture(t, path, IssueInput{Number: 5, Title: "single issue", State: "open"})

	issues, err := readIssueInputs(path)
	if err != nil {
		t.Fatalf("readIssueInputs returned error: %v", err)
	}
	if len(issues) != 1 {
		t.Fatalf("len(issues) = %d, want 1", len(issues))
	}
	if issues[0].Title != "single issue" {
		t.Fatalf("Title = %q", issues[0].Title)
	}
}

func TestReadIssueInputsSingleObjectNoTitle(t *testing.T) {
	path := filepath.Join(t.TempDir(), "issue.json")
	writeJSONFixture(t, path, map[string]interface{}{"number": 1})

	_, err := readIssueInputs(path)
	if err == nil {
		t.Fatal("expected error for single object without title")
	}
}

func TestReadIssueInputsBadJSON(t *testing.T) {
	path := filepath.Join(t.TempDir(), "bad.json")
	os.WriteFile(path, []byte("not json"), 0600)

	_, err := readIssueInputs(path)
	if err == nil {
		t.Fatal("expected error for bad JSON")
	}
}

func TestReadIssueInputsMissingFile(t *testing.T) {
	_, err := readIssueInputs("/nonexistent/file.json")
	if err == nil {
		t.Fatal("expected error for missing file")
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

func TestCollectHealthFromArgsFromFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "health.json")
	writeJSONFixture(t, path, HealthInput{
		Repository:          "owner/repo",
		RecentActivityKnown: true,
		RecentActivityDays:  5,
		HasReadme:           true,
	})

	ctx := &common.RuntimeContext{Args: map[string]string{"from": path}}
	input, err := collectHealthFromArgs(ctx)
	if err != nil {
		t.Fatalf("collectHealthFromArgs error: %v", err)
	}
	if input.Repository != "owner/repo" {
		t.Fatalf("Repository = %q", input.Repository)
	}
	if input.RecentActivityDays != 5 {
		t.Fatalf("RecentActivityDays = %d", input.RecentActivityDays)
	}
}

func TestCollectHealthFromArgsDirect(t *testing.T) {
	ctx := &common.RuntimeContext{Args: map[string]string{
		"repository":            "owner/repo",
		"open-issues":           "5",
		"open-prs":              "3",
		"recent-activity-known": "true",
		"recent-activity-days":  "7",
		"has-readme":            "true",
	}}

	input, err := collectHealthFromArgs(ctx)
	if err != nil {
		t.Fatalf("collectHealthFromArgs error: %v", err)
	}
	if input.OpenIssues != 5 {
		t.Fatalf("OpenIssues = %d", input.OpenIssues)
	}
	if input.OpenPRs != 3 {
		t.Fatalf("OpenPRs = %d", input.OpenPRs)
	}
	if !input.RecentActivityKnown {
		t.Fatal("expected RecentActivityKnown=true")
	}
}

func TestReadHealthInput(t *testing.T) {
	path := filepath.Join(t.TempDir(), "health.json")
	writeJSONFixture(t, path, HealthInput{
		Repository:          "owner/repo",
		RecentActivityKnown: true,
		RecentActivityDays:  3,
	})

	input, err := readHealthInput(path)
	if err != nil {
		t.Fatalf("readHealthInput error: %v", err)
	}
	if input.Repository != "owner/repo" {
		t.Fatalf("Repository = %q", input.Repository)
	}
}

func TestReadHealthInputMissingFile(t *testing.T) {
	_, err := readHealthInput("/nonexistent/health.json")
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestFilterIssueInputs(t *testing.T) {
	issues := []IssueInput{
		{Number: 1, State: "open", Title: "bug"},
		{Number: 2, State: "closed", Title: "done"},
		{Number: 3, State: "open", Title: "feat"},
	}

	filtered := filterIssueInputs(issues, "open", 0)
	if len(filtered) != 2 {
		t.Fatalf("expected 2 open issues, got %d", len(filtered))
	}

	filtered = filterIssueInputs(issues, "all", 0)
	if len(filtered) != 3 {
		t.Fatalf("expected 3 (all) issues, got %d", len(filtered))
	}

	filtered = filterIssueInputs(issues, "", 0)
	if len(filtered) != 3 {
		t.Fatalf("expected 3 (no filter) issues, got %d", len(filtered))
	}

	filtered = filterIssueInputs(issues, "", 1)
	if len(filtered) != 1 {
		t.Fatalf("expected 1 (limit) issue, got %d", len(filtered))
	}
}

func TestRepositoryFromContext(t *testing.T) {
	ctx := &common.RuntimeContext{Owner: "owner", Repo: "repo"}
	if got := repositoryFromContext(ctx, "fallback"); got != "owner/repo" {
		t.Fatalf("expected owner/repo, got %s", got)
	}

	ctx2 := &common.RuntimeContext{}
	if got := repositoryFromContext(ctx2, "fallback"); got != "fallback" {
		t.Fatalf("expected fallback, got %s", got)
	}

	if got := repositoryFromContext(ctx2, ""); got != "local" {
		t.Fatalf("expected local, got %s", got)
	}
}

func TestParseCSV(t *testing.T) {
	parts := parseCSV("a, b ,c")
	if len(parts) != 3 {
		t.Fatalf("expected 3 parts, got %d", len(parts))
	}
	if parts[0] != "a" || parts[1] != "b" || parts[2] != "c" {
		t.Fatalf("parts = %v", parts)
	}

	if parts := parseCSV(""); parts != nil {
		t.Fatalf("expected nil for empty string, got %v", parts)
	}

	if parts := parseCSV("  "); parts != nil {
		t.Fatalf("expected nil for whitespace, got %v", parts)
	}
}

func TestParseIntArg(t *testing.T) {
	v, err := parseIntArg("5", 0, "count")
	if err != nil {
		t.Fatalf("parseIntArg error: %v", err)
	}
	if v != 5 {
		t.Fatalf("= %d", v)
	}

	v, err = parseIntArg("", 10, "count")
	if err != nil {
		t.Fatalf("parseIntArg empty error: %v", err)
	}
	if v != 10 {
		t.Fatalf("default = %d", v)
	}

	_, err = parseIntArg("abc", 0, "count")
	if err == nil {
		t.Fatal("expected error for non-int")
	}

	_, err = parseIntArg("-1", 0, "count")
	if err == nil {
		t.Fatal("expected error for negative")
	}
}

func TestMustParseInt(t *testing.T) {
	if v := mustParseInt("5", 10); v != 5 {
		t.Fatalf("= %d", v)
	}
	if v := mustParseInt("abc", 10); v != 10 {
		t.Fatalf("default = %d", v)
	}
}

func TestRunTriageFromFile(t *testing.T) {
	// Write a temp JSON file with issues
	path := filepath.Join(t.TempDir(), "issues.json")
	writeJSONFixture(t, path, map[string]interface{}{
		"issues": []map[string]interface{}{
			{
				"number": 1,
				"title":  "CLI crash on login",
				"body":   "Panic when running login command.",
				"state":  "open",
				"labels": []string{"bug"},
			},
		},
	})

	ctx := &common.RuntimeContext{
		Client: &client.Client{HTTP: http.DefaultClient, BaseURL: "http://localhost"},
		Format: "json",
		Args: map[string]string{
			"from":    path,
			"lang":    "en",
			"dry-run": "true",
		},
	}

	if err := runTriage(ctx); err != nil {
		t.Fatalf("runTriage from file error: %v", err)
	}
}

func TestRunTriageFromFileMarkdown(t *testing.T) {
	path := filepath.Join(t.TempDir(), "issues.json")
	writeJSONFixture(t, path, map[string]interface{}{
		"issues": []map[string]interface{}{
			{
				"number": 2,
				"title":  "README typo",
				"body":   "Docs have a minor typo.",
				"state":  "open",
				"labels": []string{"docs"},
			},
		},
	})

	ctx := &common.RuntimeContext{
		Client: &client.Client{HTTP: http.DefaultClient, BaseURL: "http://localhost"},
		Format: "markdown",
		Args: map[string]string{
			"from":    path,
			"lang":    "en",
			"dry-run": "true",
		},
	}

	if err := runTriage(ctx); err != nil {
		t.Fatalf("runTriage from file markdown error: %v", err)
	}
}

func TestRunHealthFromFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "health.json")
	writeJSONFixture(t, path, HealthInput{
		Repository:          "owner/repo",
		OpenIssues:          2,
		OpenPRs:             1,
		RecentActivityKnown: true,
		RecentActivityDays:  3,
		HasReadme:           true,
		HasLicense:          true,
		HasContributing:     true,
		AgentReadinessKnown: true,
		AgentReadinessScore: 8,
	})

	ctx := &common.RuntimeContext{
		Client: &client.Client{HTTP: http.DefaultClient, BaseURL: "http://localhost"},
		Format: "json",
		Args: map[string]string{
			"from": path,
			"lang": "en",
		},
	}

	if err := runHealth(ctx); err != nil {
		t.Fatalf("runHealth from file error: %v", err)
	}
}

func TestRunHealthFromFileMarkdown(t *testing.T) {
	path := filepath.Join(t.TempDir(), "health.json")
	writeJSONFixture(t, path, HealthInput{
		Repository:          "owner/repo",
		RecentActivityKnown: true,
		RecentActivityDays:  1,
		HasReadme:           true,
	})

	ctx := &common.RuntimeContext{
		Client: &client.Client{HTTP: http.DefaultClient, BaseURL: "http://localhost"},
		Format: "markdown",
		Args: map[string]string{
			"from": path,
			"lang": "en",
		},
	}

	if err := runHealth(ctx); err != nil {
		t.Fatalf("runHealth from file markdown error: %v", err)
	}
}

func TestRunTriageMissingInput(t *testing.T) {
	ctx := &common.RuntimeContext{
		Client: &client.Client{HTTP: http.DefaultClient, BaseURL: "http://localhost"},
		Args:   map[string]string{},
	}
	err := runTriage(ctx)
	if err == nil {
		t.Fatal("expected error with no title, from, or owner/repo")
	}
}

func TestCollectIssuesFromArgsFromFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "issues.json")
	writeJSONFixture(t, path, map[string]interface{}{
		"issues": []map[string]interface{}{
			{"number": 1, "title": "Test", "state": "open"},
		},
	})

	ctx := &common.RuntimeContext{
		Args: map[string]string{"from": path, "state": "open"},
	}
	issues, err := collectIssuesFromArgs(ctx)
	if err != nil {
		t.Fatalf("collectIssuesFromArgs error: %v", err)
	}
	if len(issues) != 1 {
		t.Fatalf("expected 1 issue, got %d", len(issues))
	}
}

func TestCollectIssuesFromArgsMissingTitle(t *testing.T) {
	ctx := &common.RuntimeContext{
		Args: map[string]string{"number": "1", "state": "open"},
	}
	_, err := collectIssuesFromArgs(ctx)
	if err == nil {
		t.Fatal("expected error when title is missing from single issue args")
	}
}

func TestCollectIssuesFromArgsRequiresFromOrTitle(t *testing.T) {
	ctx := &common.RuntimeContext{
		Args: map[string]string{"state": "open"},
	}
	_, err := collectIssuesFromArgs(ctx)
	if err == nil {
		t.Fatal("expected error with neither --from nor --title")
	}
}

func TestCollectHealthFromArgsRemoteMode(t *testing.T) {
	ctx := &common.RuntimeContext{
		Client: &client.Client{HTTP: http.DefaultClient, BaseURL: "http://localhost"},
		Owner:  "owner",
		Repo:   "repo",
		Args:   map[string]string{},
	}
	// Will try HTTP and fail, but shouldn't panic
	_, err := collectHealthFromArgs(ctx)
	if err != nil {
		t.Logf("expected network error: %v", err)
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
