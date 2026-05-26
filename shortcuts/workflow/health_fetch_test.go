package workflow

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestFetchHealthInputCollectsSignals(t *testing.T) {
	now := time.Now().UTC()
	old := now.AddDate(0, 0, -45)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == "GET" && r.URL.Path == "/v1/owner/repo.json":
			writeWorkflowJSON(t, w, map[string]interface{}{
				"name":             "repo",
				"updated_at":       now.AddDate(0, 0, -2).Format(time.RFC3339),
				"has_readme":       true,
				"has_license":      true,
				"has_contributing": true,
			})
		case r.Method == "GET" && r.URL.Path == "/v1/owner/repo/issues.json":
			writeWorkflowJSON(t, w, map[string]interface{}{"issues": []map[string]interface{}{
				{"id": 1, "subject": "fresh issue", "updated_at": now.AddDate(0, 0, -1).Format(time.RFC3339)},
				{"id": 2, "subject": "stale issue", "updated_at": old.Format(time.RFC3339)},
			}})
		case r.Method == "GET" && r.URL.Path == "/v1/owner/repo/pulls.json":
			writeWorkflowJSON(t, w, map[string]interface{}{"pulls": []map[string]interface{}{
				{"id": 3, "title": "stale pr", "updated_at": old.Format(time.RFC3339)},
			}})
		case r.Method == "GET" && r.URL.Path == "/owner/repo/releases.json":
			writeWorkflowJSON(t, w, map[string]interface{}{"releases": []map[string]interface{}{
				{"id": 4, "name": "v1.0.0", "created_at": now.AddDate(0, 0, -3).Format(time.RFC3339)},
			}})
		case r.Method == "GET" && r.URL.Path == "/owner/repo/builds.json":
			writeWorkflowJSON(t, w, map[string]interface{}{"builds": []map[string]interface{}{
				{"id": 5, "status": "success"},
			}})
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	input, notes, err := FetchHealthInput(workflowTestContext(server), HealthFetchOptions{
		StaleDays:      30,
		IncludeCI:      true,
		IncludeRelease: true,
		IncludeDocs:    true,
	})
	if err != nil {
		t.Fatalf("FetchHealthInput returned error: %v", err)
	}
	if len(notes) != 0 {
		t.Fatalf("notes = %v, want empty", notes)
	}
	if input.Repository != "owner/repo" {
		t.Fatalf("Repository = %q, want owner/repo", input.Repository)
	}
	if input.OpenIssues != 2 || input.StaleIssues != 1 {
		t.Fatalf("issues = open %d stale %d, want open 2 stale 1", input.OpenIssues, input.StaleIssues)
	}
	if input.OpenPRs != 1 || input.StalePRs != 1 {
		t.Fatalf("prs = open %d stale %d, want open 1 stale 1", input.OpenPRs, input.StalePRs)
	}
	if !input.ReleaseKnown || !input.HasRecentRelease {
		t.Fatalf("release signals = known %v recent %v, want true true", input.ReleaseKnown, input.HasRecentRelease)
	}
	if !input.CIKnown || !input.CIPassing {
		t.Fatalf("ci signals = known %v passing %v, want true true", input.CIKnown, input.CIPassing)
	}
	if !input.HasReadme || !input.HasLicense || !input.HasContributing {
		t.Fatalf("doc signals = readme %v license %v contributing %v, want all true", input.HasReadme, input.HasLicense, input.HasContributing)
	}
	if !input.RecentActivityKnown || input.RecentActivityDays > 3 {
		t.Fatalf("recent activity = known %v days %d, want known and <= 3", input.RecentActivityKnown, input.RecentActivityDays)
	}
}

func TestFetchHealthInputToleratesOptionalProbeFailures(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == "GET" && r.URL.Path == "/v1/owner/repo.json":
			writeWorkflowJSON(t, w, map[string]interface{}{
				"name":        "repo",
				"has_readme":  true,
				"has_license": true,
			})
		case r.Method == "GET" && r.URL.Path == "/v1/owner/repo/issues.json":
			writeWorkflowJSON(t, w, map[string]interface{}{"issues": []map[string]interface{}{}})
		case r.Method == "GET" && r.URL.Path == "/v1/owner/repo/pulls.json":
			writeWorkflowJSON(t, w, map[string]interface{}{"pulls": []map[string]interface{}{}})
		case r.Method == "GET" && (r.URL.Path == "/owner/repo/releases.json" || r.URL.Path == "/owner/repo/builds.json"):
			http.Error(w, "temporary failure", http.StatusInternalServerError)
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	input, notes, err := FetchHealthInput(workflowTestContext(server), HealthFetchOptions{
		StaleDays:      30,
		IncludeCI:      true,
		IncludeRelease: true,
		IncludeDocs:    true,
	})
	if err != nil {
		t.Fatalf("FetchHealthInput returned error: %v", err)
	}
	if input.Repository != "owner/repo" {
		t.Fatalf("Repository = %q, want owner/repo", input.Repository)
	}
	if input.ReleaseKnown {
		t.Fatal("ReleaseKnown = true, want false after release probe failure")
	}
	if input.CIKnown {
		t.Fatal("CIKnown = true, want false after CI probe failure")
	}
	if len(notes) == 0 {
		t.Fatal("notes is empty, want scoring notes for failed optional probes")
	}
}

func TestFetchHealthInputHandlesMissingRepoActivityAndDocGaps(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == "GET" && r.URL.Path == "/v1/owner/repo.json":
			writeWorkflowJSON(t, w, map[string]interface{}{
				"name":        "repo",
				"has_readme":  true,
				"has_license": false,
			})
		case r.Method == "GET" && r.URL.Path == "/v1/owner/repo/issues.json":
			writeWorkflowJSON(t, w, map[string]interface{}{"issues": []map[string]interface{}{}})
		case r.Method == "GET" && r.URL.Path == "/v1/owner/repo/pulls.json":
			writeWorkflowJSON(t, w, map[string]interface{}{"pulls": []map[string]interface{}{}})
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	input, notes, err := FetchHealthInput(workflowTestContext(server), HealthFetchOptions{
		StaleDays:   30,
		IncludeCI:   false,
		IncludeDocs: true,
	})
	if err != nil {
		t.Fatalf("FetchHealthInput returned error: %v", err)
	}
	if input.RecentActivityKnown {
		t.Fatal("RecentActivityKnown = true, want false when updated_at is missing and no activity lists carry timestamps")
	}
	if len(notes) == 0 {
		t.Fatal("notes is empty, want scoring notes for missing repo signals")
	}
}

func TestFetchHealthInputUsesAlternativeActivityFieldsAndDefaultStaleDays(t *testing.T) {
	now := time.Now().UTC()
	fresh := now.AddDate(0, 0, -2)
	stale := now.AddDate(0, 0, -40)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == "GET" && r.URL.Path == "/v1/owner/repo.json":
			writeWorkflowJSON(t, w, map[string]interface{}{"name": "repo"})
		case r.Method == "GET" && r.URL.Path == "/v1/owner/repo/issues.json":
			writeWorkflowJSON(t, w, map[string]interface{}{"issues": []map[string]interface{}{
				{"id": 1, "updatedAt": fresh.Format(time.RFC3339)},
				{"id": 2, "last_activity_at": stale.Format(time.RFC3339)},
			}})
		case r.Method == "GET" && r.URL.Path == "/v1/owner/repo/pulls.json":
			writeWorkflowJSON(t, w, map[string]interface{}{"pulls": []map[string]interface{}{
				{"id": 3, "merged_at": fresh.Format(time.RFC3339)},
				{"id": 4, "closed_at": stale.Format(time.RFC3339)},
			}})
		case r.Method == "GET" && r.URL.Path == "/owner/repo/releases.json":
			writeWorkflowJSON(t, w, map[string]interface{}{"releases": []map[string]interface{}{}})
		case r.Method == "GET" && r.URL.Path == "/owner/repo/builds.json":
			writeWorkflowJSON(t, w, map[string]interface{}{"builds": []map[string]interface{}{
				{"id": 5, "status": "success"},
			}})
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	for _, staleDays := range []int{0, -5} {
		t.Run("stale-days", func(t *testing.T) {
			input, notes, err := FetchHealthInput(workflowTestContext(server), HealthFetchOptions{
				StaleDays:      staleDays,
				IncludeRelease: true,
				IncludeCI:      true,
			})
			if err != nil {
				t.Fatalf("FetchHealthInput returned error: %v", err)
			}
			if input.StaleIssues != 1 {
				t.Fatalf("StaleIssues = %d, want 1", input.StaleIssues)
			}
			if input.StalePRs != 1 {
				t.Fatalf("StalePRs = %d, want 1", input.StalePRs)
			}
			if !input.RecentActivityKnown {
				t.Fatal("RecentActivityKnown = false, want true")
			}
			if input.RecentActivityDays > 3 {
				t.Fatalf("RecentActivityDays = %d, want <= 3", input.RecentActivityDays)
			}
			if len(notes) != 0 {
				t.Fatalf("notes = %v, want empty when release and CI probes succeed", notes)
			}
		})
	}
}

func TestFetchHealthInputSupportsReleaseShapeVariants(t *testing.T) {
	now := time.Now().UTC()
	releasePayloads := []map[string]interface{}{
		{"releases": []map[string]interface{}{{"id": 1, "name": "v1.0.0", "created_at": now.AddDate(0, 0, -1).Format(time.RFC3339)}}},
		{"data": []map[string]interface{}{{"id": 2, "name": "v1.0.1", "updated_at": now.AddDate(0, 0, -1).Format(time.RFC3339)}}},
	}

	for i, payload := range releasePayloads {
		t.Run("shape", func(t *testing.T) {
			payload := payload
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				switch {
				case r.Method == "GET" && r.URL.Path == "/v1/owner/repo.json":
					writeWorkflowJSON(t, w, map[string]interface{}{
						"name":        "repo",
						"updated_at":  now.Format(time.RFC3339),
						"has_readme":  true,
						"has_license": true,
					})
				case r.Method == "GET" && r.URL.Path == "/v1/owner/repo/issues.json":
					writeWorkflowJSON(t, w, map[string]interface{}{"issues": []map[string]interface{}{}})
				case r.Method == "GET" && r.URL.Path == "/v1/owner/repo/pulls.json":
					writeWorkflowJSON(t, w, map[string]interface{}{"pulls": []map[string]interface{}{}})
				case r.Method == "GET" && r.URL.Path == "/owner/repo/releases.json":
					writeWorkflowJSON(t, w, payload)
				case r.Method == "GET" && r.URL.Path == "/owner/repo/builds.json":
					writeWorkflowJSON(t, w, map[string]interface{}{"builds": []map[string]interface{}{
						{"id": 3, "status": "success"},
					}})
				default:
					t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
				}
			}))
			defer server.Close()

			input, notes, err := FetchHealthInput(workflowTestContext(server), HealthFetchOptions{
				StaleDays:      30,
				IncludeRelease: true,
				IncludeCI:      true,
				IncludeDocs:    false,
			})
			if err != nil {
				t.Fatalf("FetchHealthInput returned error: %v", err)
			}
			if !input.ReleaseKnown || !input.HasRecentRelease {
				t.Fatalf("release signals = known %v recent %v, want true true", input.ReleaseKnown, input.HasRecentRelease)
			}
			if input.RecentActivityDays > 1 {
				t.Fatalf("RecentActivityDays = %d, want <= 1", input.RecentActivityDays)
			}
			if len(notes) != 0 {
				t.Fatalf("notes = %v, want empty for supported release shape %d", notes, i)
			}
		})
	}
}

func TestFetchHealthInputReportsCIUnavailable(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == "GET" && r.URL.Path == "/v1/owner/repo.json":
			writeWorkflowJSON(t, w, map[string]interface{}{
				"name":        "repo",
				"has_readme":  true,
				"has_license": true,
			})
		case r.Method == "GET" && r.URL.Path == "/v1/owner/repo/issues.json":
			writeWorkflowJSON(t, w, map[string]interface{}{"issues": []map[string]interface{}{}})
		case r.Method == "GET" && r.URL.Path == "/v1/owner/repo/pulls.json":
			writeWorkflowJSON(t, w, map[string]interface{}{"pulls": []map[string]interface{}{}})
		case r.Method == "GET" && r.URL.Path == "/owner/repo/builds.json":
			http.Error(w, "not found", http.StatusNotFound)
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	input, notes, err := FetchHealthInput(workflowTestContext(server), HealthFetchOptions{
		StaleDays:      30,
		IncludeCI:      true,
		IncludeRelease: false,
		IncludeDocs:    false,
	})
	if err != nil {
		t.Fatalf("FetchHealthInput returned error: %v", err)
	}
	if input.CIKnown {
		t.Fatal("CIKnown = true, want false for CI probe failure")
	}
	if len(notes) == 0 {
		t.Fatal("notes is empty, want note for unavailable CI")
	}
	joined := ""
	for _, note := range notes {
		joined += note.Metric + " " + note.Note + "\n"
	}
	if !strings.Contains(joined, "ci_status") {
		t.Fatalf("notes = %v, want ci_status note", notes)
	}
}
