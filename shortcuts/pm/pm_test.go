package pm

import (
	"net/http"
	"testing"

	"github.com/gitlink-org/gitlink-cli/shortcuts/common"
)

func TestFetchProjectID(t *testing.T) {
	cases := []struct {
		name     string
		response map[string]interface{}
		wantID   int
	}{
		{"repo_id", map[string]interface{}{"repo_id": float64(100)}, 100},
		{"project_id", map[string]interface{}{"project_id": float64(200)}, 200},
		{"id", map[string]interface{}{"id": float64(300)}, 300},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			resp := tc.response
			server := common.NewTestServer(t, func(w http.ResponseWriter, r *http.Request) {
				if r.Method == "GET" && r.URL.Path == "/owner/repo.json" {
					common.WriteJSON(t, w, resp)
					return
				}
				t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
			})
			defer server.Close()

			ctx := common.NewTestContext(t, server, "owner", "repo", map[string]string{})
			id, err := fetchProjectID(ctx)
			if err != nil {
				t.Fatalf("fetchProjectID failed: %v", err)
			}
			if id != tc.wantID {
				t.Fatalf("got %d, want %d", id, tc.wantID)
			}
		})
	}
}

func TestFetchProjectIDNotFound(t *testing.T) {
	server := common.NewTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		common.WriteJSON(t, w, map[string]interface{}{"name": "repo"})
	})
	defer server.Close()

	ctx := common.NewTestContext(t, server, "owner", "repo", map[string]string{})
	_, err := fetchProjectID(ctx)
	if err == nil {
		t.Fatal("expected error for missing project ID")
	}
}

func TestPMBoards(t *testing.T) {
	server := common.NewTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == "GET" && r.URL.Path == "/owner/repo.json":
			common.WriteJSON(t, w, map[string]interface{}{"id": float64(123)})
		case r.Method == "GET" && r.URL.Path == "/pm/dashboards":
			common.WriteJSON(t, w, map[string]interface{}{
				"boards": []interface{}{
					map[string]interface{}{"id": 1, "name": "Sprint 1"},
					map[string]interface{}{"id": 2, "name": "Sprint 2"},
				},
			})
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	})
	defer server.Close()

	ctx := common.NewTestContext(t, server, "owner", "repo", map[string]string{})
	err := common.RunShortcut(t, Shortcuts(), "boards", ctx)
	if err != nil {
		t.Fatalf("boards failed: %v", err)
	}
}

func TestPMSprints(t *testing.T) {
	server := common.NewTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == "GET" && r.URL.Path == "/owner/repo.json":
			common.WriteJSON(t, w, map[string]interface{}{"id": float64(123)})
		case r.Method == "GET" && r.URL.Path == "/pm/sprint_issues":
			common.WriteJSON(t, w, map[string]interface{}{
				"issues": []interface{}{
					map[string]interface{}{"id": 10, "subject": "Task A"},
				},
			})
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	})
	defer server.Close()

	ctx := common.NewTestContext(t, server, "owner", "repo", map[string]string{})
	err := common.RunShortcut(t, Shortcuts(), "sprints", ctx)
	if err != nil {
		t.Fatalf("sprints failed: %v", err)
	}
}

func TestPMWeekly(t *testing.T) {
	server := common.NewTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == "GET" && r.URL.Path == "/owner/repo.json":
			common.WriteJSON(t, w, map[string]interface{}{"id": float64(123)})
		case r.Method == "GET" && r.URL.Path == "/pm/weekly_issues":
			common.WriteJSON(t, w, map[string]interface{}{
				"reports": []interface{}{
					map[string]interface{}{"id": 1, "title": "Week 21"},
				},
			})
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	})
	defer server.Close()

	ctx := common.NewTestContext(t, server, "owner", "repo", map[string]string{})
	err := common.RunShortcut(t, Shortcuts(), "weekly", ctx)
	if err != nil {
		t.Fatalf("weekly failed: %v", err)
	}
}

func TestPMTags(t *testing.T) {
	server := common.NewTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == "GET" && r.URL.Path == "/owner/repo.json":
			common.WriteJSON(t, w, map[string]interface{}{"id": float64(123)})
		case r.Method == "GET" && r.URL.Path == "/pm/issue_tags":
			common.WriteJSON(t, w, map[string]interface{}{
				"tags": []interface{}{
					map[string]interface{}{"id": 1, "name": "bug"},
				},
			})
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	})
	defer server.Close()

	ctx := common.NewTestContext(t, server, "owner", "repo", map[string]string{})
	err := common.RunShortcut(t, Shortcuts(), "tags", ctx)
	if err != nil {
		t.Fatalf("tags failed: %v", err)
	}
}

func TestPMPipelines(t *testing.T) {
	server := common.NewTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == "GET" && r.URL.Path == "/owner/repo.json":
			common.WriteJSON(t, w, map[string]interface{}{"id": float64(123)})
		case r.Method == "GET" && r.URL.Path == "/pm/pipelines":
			common.WriteJSON(t, w, map[string]interface{}{
				"pipelines": []interface{}{
					map[string]interface{}{"id": 1, "name": "CI"},
				},
			})
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	})
	defer server.Close()

	ctx := common.NewTestContext(t, server, "owner", "repo", map[string]string{})
	err := common.RunShortcut(t, Shortcuts(), "pipelines", ctx)
	if err != nil {
		t.Fatalf("pipelines failed: %v", err)
	}
}

func TestPMActions(t *testing.T) {
	server := common.NewTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == "GET" && r.URL.Path == "/owner/repo.json":
			common.WriteJSON(t, w, map[string]interface{}{"id": float64(123)})
		case r.Method == "GET" && r.URL.Path == "/pm/action_runs":
			common.WriteJSON(t, w, map[string]interface{}{
				"runs": []interface{}{
					map[string]interface{}{"id": 1, "status": "success"},
				},
			})
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	})
	defer server.Close()

	ctx := common.NewTestContext(t, server, "owner", "repo", map[string]string{})
	err := common.RunShortcut(t, Shortcuts(), "actions", ctx)
	if err != nil {
		t.Fatalf("actions failed: %v", err)
	}
}
