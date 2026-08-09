package milestone

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gitlink-org/gitlink-cli/internal/client"
	"github.com/gitlink-org/gitlink-cli/shortcuts/common"
)

func TestGenerateMilestoneReportByID(t *testing.T) {
	oldNow := milestoneNow
	milestoneNow = func() time.Time {
		return time.Date(2026, 6, 11, 10, 0, 0, 0, time.Local)
	}
	defer func() { milestoneNow = oldNow }()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" || r.URL.Path != "/v1/owner/repo/milestones/7.json" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}

		page := r.URL.Query().Get("page")
		if page == "" {
			page = "1"
		}

		switch r.URL.Query().Get("category") {
		case "":
			if page != "1" {
				t.Fatalf("unexpected extra all-issues page request: %s", page)
			}
			writeJSON(t, w, map[string]interface{}{
				"milestone": map[string]interface{}{
					"id":                  7,
					"name":                "v1.0",
					"description":         "first release",
					"effective_date":      "2026-06-09",
					"status":              "open",
					"created_at":          "2026-05-01 09:00",
					"updated_on":          "2026-06-10 08:30",
					"percent":             0.6666667,
					"issues_count":        3,
					"opened_issues_count": 2,
					"close_issues_count":  1,
				},
				"total_issues_count":  3,
				"opened_issues_count": 2,
				"closed_issues_count": 1,
				"issues": []interface{}{
					map[string]interface{}{
						"id":                     101,
						"project_issues_index":   11,
						"subject":                "login failed",
						"status_name":            "Open",
						"priority_name":          "High",
						"author":                 map[string]interface{}{"name": "Alice"},
						"assigners":              []interface{}{map[string]interface{}{"name": "Bob"}},
						"tags":                   []interface{}{map[string]interface{}{"name": "bug"}},
						"comment_journals_count": 2,
						"updated_at":             "2026-06-10 09:30",
					},
					map[string]interface{}{
						"id":                     102,
						"project_issues_index":   12,
						"subject":                "docs cleanup",
						"status_name":            "Open",
						"priority_name":          "Normal",
						"author":                 map[string]interface{}{"login": "carol"},
						"assigners":              []interface{}{},
						"tags":                   []interface{}{},
						"comment_journals_count": 0,
						"updated_at":             "2026-06-08 08:00",
					},
					map[string]interface{}{
						"id":                     103,
						"project_issues_index":   13,
						"subject":                "closed bug",
						"status_name":            "Closed",
						"priority_name":          "Normal",
						"author":                 map[string]interface{}{"name": "Dave"},
						"assigners":              []interface{}{map[string]interface{}{"name": "Bob"}},
						"tags":                   []interface{}{map[string]interface{}{"name": "cleanup"}},
						"comment_journals_count": 1,
						"updated_at":             "2026-06-07 12:00",
					},
				},
			})
		case "opened":
			if page != "1" {
				t.Fatalf("opened issues should stop after first page, got page %s", page)
			}
			writeJSON(t, w, map[string]interface{}{
				"milestone": map[string]interface{}{
					"id":             7,
					"name":           "v1.0",
					"effective_date": "2026-06-09",
					"status":         "open",
					"percent":        0.6666667,
				},
				"total_issues_count":  3,
				"opened_issues_count": 2,
				"closed_issues_count": 1,
				"issues": []interface{}{
					map[string]interface{}{
						"id":                     101,
						"project_issues_index":   11,
						"subject":                "login failed",
						"status_name":            "Open",
						"priority_name":          "High",
						"author":                 map[string]interface{}{"name": "Alice"},
						"assigners":              []interface{}{map[string]interface{}{"name": "Bob"}},
						"tags":                   []interface{}{map[string]interface{}{"name": "bug"}},
						"comment_journals_count": 2,
						"updated_at":             "2026-06-10 09:30",
					},
					map[string]interface{}{
						"id":                     102,
						"project_issues_index":   12,
						"subject":                "docs cleanup",
						"status_name":            "Open",
						"priority_name":          "Normal",
						"author":                 map[string]interface{}{"login": "carol"},
						"assigners":              []interface{}{},
						"tags":                   []interface{}{},
						"comment_journals_count": 0,
						"updated_at":             "2026-06-08 08:00",
					},
				},
			})
		default:
			t.Fatalf("unexpected category: %q", r.URL.Query().Get("category"))
		}
	}))
	defer server.Close()

	ctx := &common.RuntimeContext{
		Client: &client.Client{HTTP: server.Client(), BaseURL: server.URL},
		Owner:  "owner",
		Repo:   "repo",
	}

	report, err := generateMilestoneReport(ctx, "7", "", "2")
	if err != nil {
		t.Fatalf("generateMilestoneReport failed: %v", err)
	}

	assertEqual(t, report.Repository, "owner/repo")
	assertEqual(t, report.Milestone.ID, 7)
	assertEqual(t, report.Milestone.Name, "v1.0")
	assertEqual(t, report.Milestone.CompletionPct, 66.67)
	assertEqual(t, report.Summary.TotalIssues, 3)
	assertEqual(t, report.Summary.OpenIssues, 2)
	assertEqual(t, report.Summary.ClosedIssues, 1)
	assertEqual(t, report.Summary.UnassignedOpenIssues, 1)
	assertEqual(t, report.Summary.UntaggedOpenIssues, 1)
	assertEqual(t, report.Summary.CommentedOpenIssues, 1)
	if report.Summary.DaysUntilDue == nil || *report.Summary.DaysUntilDue != -2 {
		t.Fatalf("unexpected days_until_due: %+v", report.Summary.DaysUntilDue)
	}
	if !report.Summary.Overdue {
		t.Fatal("expected overdue milestone")
	}
	if report.Readiness.ReadyToClose {
		t.Fatal("milestone should not be ready to close")
	}
	if len(report.Readiness.Blockers) != 1 || !strings.Contains(report.Readiness.Blockers[0], "2 open issues remain") {
		t.Fatalf("unexpected blockers: %+v", report.Readiness.Blockers)
	}
	if len(report.Readiness.Warnings) != 3 {
		t.Fatalf("unexpected warnings: %+v", report.Readiness.Warnings)
	}
	assertEqual(t, report.Breakdown.AllStatuses[0].Name, "Open")
	assertEqual(t, report.Breakdown.AllStatuses[0].Count, 2)
	assertEqual(t, report.Breakdown.OpenPriorities[0].Name, "High")
	assertEqual(t, report.Breakdown.OpenPriorities[0].Count, 1)
	if len(report.Samples.RecentOpenIssues) != 2 || report.Samples.RecentOpenIssues[0].Number != 11 {
		t.Fatalf("unexpected recent_open_issues: %+v", report.Samples.RecentOpenIssues)
	}
	if len(report.Samples.UnassignedOpenIssues) != 1 || report.Samples.UnassignedOpenIssues[0].Number != 12 {
		t.Fatalf("unexpected unassigned_open_issues: %+v", report.Samples.UnassignedOpenIssues)
	}
	if len(report.Samples.MostCommentedOpenIssues) != 1 || report.Samples.MostCommentedOpenIssues[0].Number != 11 {
		t.Fatalf("unexpected most_commented_open_issues: %+v", report.Samples.MostCommentedOpenIssues)
	}
}

func TestGenerateMilestoneReportByNameResolvesExactMatch(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/owner/repo/milestones.json":
			writeJSON(t, w, map[string]interface{}{
				"total_count": 2,
				"milestones": []interface{}{
					map[string]interface{}{"id": 7, "name": "v1.0"},
					map[string]interface{}{"id": 8, "name": "v2.0"},
				},
			})
		case "/v1/owner/repo/milestones/7.json":
			writeJSON(t, w, map[string]interface{}{
				"milestone": map[string]interface{}{
					"id":             7,
					"name":           "v1.0",
					"effective_date": "2026-07-01",
					"status":         "open",
					"percent":        1,
				},
				"total_issues_count":  0,
				"opened_issues_count": 0,
				"closed_issues_count": 0,
				"issues":              []interface{}{},
			})
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	ctx := &common.RuntimeContext{
		Client: &client.Client{HTTP: server.Client(), BaseURL: server.URL},
		Owner:  "owner",
		Repo:   "repo",
	}

	report, err := generateMilestoneReport(ctx, "", "v1.0", "3")
	if err != nil {
		t.Fatalf("generateMilestoneReport by name failed: %v", err)
	}
	assertEqual(t, report.Milestone.ID, 7)
	assertEqual(t, report.Milestone.Name, "v1.0")
	if !report.Readiness.ReadyToClose {
		t.Fatal("milestone with zero open issues should be ready to close")
	}
}

func TestGenerateMilestoneReportByNameAmbiguous(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/owner/repo/milestones.json" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		writeJSON(t, w, map[string]interface{}{
			"total_count": 2,
			"milestones": []interface{}{
				map[string]interface{}{"id": 7, "name": "release-v1"},
				map[string]interface{}{"id": 8, "name": "release-v1-hotfix"},
			},
		})
	}))
	defer server.Close()

	ctx := &common.RuntimeContext{
		Client: &client.Client{HTTP: server.Client(), BaseURL: server.URL},
		Owner:  "owner",
		Repo:   "repo",
	}

	_, err := generateMilestoneReport(ctx, "", "release", "5")
	if err == nil || !strings.Contains(err.Error(), "matched multiple") {
		t.Fatalf("expected ambiguous milestone error, got %v", err)
	}
}

func TestFetchAllMilestoneIssuesFollowsReportedTotalAcrossPages(t *testing.T) {
	pages := []string{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" || r.URL.Path != "/v1/owner/repo/milestones/7.json" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		if r.URL.Query().Get("category") != "" {
			t.Fatalf("unexpected category: %q", r.URL.Query().Get("category"))
		}

		page := r.URL.Query().Get("page")
		pages = append(pages, page)
		switch page {
		case "1":
			writeJSON(t, w, map[string]interface{}{
				"milestone":           map[string]interface{}{"id": 7, "name": "v1.0"},
				"total_issues_count":  3,
				"opened_issues_count": 2,
				"closed_issues_count": 1,
				"issues": []interface{}{
					map[string]interface{}{"id": 101, "subject": "one"},
					map[string]interface{}{"id": 102, "subject": "two"},
				},
			})
		case "2":
			writeJSON(t, w, map[string]interface{}{
				"milestone":           map[string]interface{}{"id": 7, "name": "v1.0"},
				"total_issues_count":  3,
				"opened_issues_count": 2,
				"closed_issues_count": 1,
				"issues": []interface{}{
					map[string]interface{}{"id": 103, "subject": "three"},
				},
			})
		default:
			t.Fatalf("unexpected page request: %s", page)
		}
	}))
	defer server.Close()

	ctx := &common.RuntimeContext{
		Client: &client.Client{HTTP: server.Client(), BaseURL: server.URL},
		Owner:  "owner",
		Repo:   "repo",
	}

	issues, err := fetchAllMilestoneIssues(ctx, "7", "")
	if err != nil {
		t.Fatalf("fetchAllMilestoneIssues failed: %v", err)
	}
	if len(issues) != 3 {
		t.Fatalf("expected 3 issues, got %d", len(issues))
	}
	if strings.Join(pages, ",") != "1,2" {
		t.Fatalf("unexpected pages fetched: %v", pages)
	}
}

func TestMilestoneReportShortcutRequiresSelector(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatalf("report shortcut should fail before API call: %s %s", r.Method, r.URL.Path)
	}))
	defer server.Close()

	err := runMilestoneShortcut(t, server, "report", map[string]string{})
	if err == nil {
		t.Fatal("expected missing selector error")
	}
}

func TestMilestoneReportShortcutRuns(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" || r.URL.Path != "/v1/owner/repo/milestones/7.json" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		writeJSON(t, w, map[string]interface{}{
			"milestone": map[string]interface{}{
				"id":             7,
				"name":           "v1.0",
				"effective_date": "2026-07-01",
				"status":         "open",
				"percent":        1,
			},
			"total_issues_count":  0,
			"opened_issues_count": 0,
			"closed_issues_count": 0,
			"issues":              []interface{}{},
		})
	}))
	defer server.Close()

	if err := runMilestoneShortcut(t, server, "report", map[string]string{"id": "7", "sample-limit": "2"}); err != nil {
		t.Fatalf("report shortcut failed: %v", err)
	}
}

func TestParseMilestoneReportSampleLimit(t *testing.T) {
	if _, err := parseMilestoneReportSampleLimit("0"); err == nil {
		t.Fatal("expected validation error for zero sample-limit")
	}
	if got, err := parseMilestoneReportSampleLimit("3"); err != nil || got != 3 {
		t.Fatalf("parseMilestoneReportSampleLimit = %d, %v; want 3, nil", got, err)
	}
}
