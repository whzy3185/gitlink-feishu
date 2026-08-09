package search

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gitlink-org/gitlink-cli/shortcuts/common"
)

func TestSearchIssuesWithKeyword(t *testing.T) {
	var requestQuery string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestQuery = r.URL.RawQuery
		common.WriteJSON(t, w, map[string]interface{}{
			"total_count":  float64(2),
			"opened_count": float64(1),
			"closed_count": float64(1),
			"issues": []interface{}{
				map[string]interface{}{
					"id":                   float64(1),
					"subject":              "Fix login bug",
					"project_issues_index": float64(10),
					"status_name":          "新增",
				},
				map[string]interface{}{
					"id":                   float64(2),
					"subject":              "Update login page",
					"project_issues_index": float64(11),
					"status_name":          "关闭",
				},
			},
		})
	}))
	defer server.Close()

	ctx := common.NewTestContext(t, server, "owner", "repo", map[string]string{
		"keyword": "login",
	})
	err := common.RunShortcut(t, Shortcuts(), "issues", ctx)
	if err != nil {
		t.Fatalf("search issues failed: %v", err)
	}

	if !strings.Contains(requestQuery, "keyword=login") {
		t.Errorf("expected keyword param, got: %s", requestQuery)
	}
}

func TestSearchIssuesWithAllFilters(t *testing.T) {
	var requestQuery string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestQuery = r.URL.RawQuery
		common.WriteJSON(t, w, map[string]interface{}{
			"total_count":  float64(1),
			"opened_count": float64(1),
			"closed_count": float64(0),
			"issues":       []interface{}{},
		})
	}))
	defer server.Close()

	ctx := common.NewTestContext(t, server, "owner", "repo", map[string]string{
		"keyword":   "bug",
		"category":  "opened",
		"assignee":  "42",
		"author":    "10",
		"milestone": "5",
		"tag":       "1,2",
		"sort-by":   "created_on",
		"sort-dir":  "asc",
	})
	err := common.RunShortcut(t, Shortcuts(), "issues", ctx)
	if err != nil {
		t.Fatalf("search issues with filters failed: %v", err)
	}

	checks := []string{
		"keyword=bug",
		"category=opened",
		"assigner_id=42",
		"author_id=10",
		"milestone_id=5",
		"issue_tag_ids=1%2C2",
		"sort_by=issues.created_on",
		"sort_direction=asc",
	}
	for _, want := range checks {
		if !strings.Contains(requestQuery, want) {
			t.Errorf("missing query param %q in: %s", want, requestQuery)
		}
	}
}

func TestSearchIssuesRequiresKeyword(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("no request should be made without keyword")
	}))
	defer server.Close()

	ctx := common.NewTestContext(t, server, "owner", "repo", map[string]string{})
	err := common.RunShortcut(t, Shortcuts(), "issues", ctx)
	if err == nil {
		t.Fatal("expected error when keyword is missing, got nil")
	}
}
