package explore

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gitlink-org/gitlink-cli/shortcuts/common"
)

func TestExploreCategories(t *testing.T) {
	var path string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path = r.URL.Path
		common.WriteJSON(t, w, map[string]interface{}{
			"project_categories": []interface{}{
				map[string]interface{}{"id": 32, "name": "深度学习"},
			},
		})
	}))
	defer server.Close()

	ctx := common.NewTestContext(t, server, "", "", nil)
	if err := common.RunShortcut(t, Shortcuts(), "categories", ctx); err != nil {
		t.Fatalf("explore +categories failed: %v", err)
	}
	if !strings.Contains(path, "/project_categories") {
		t.Errorf("expected /project_categories path, got %s", path)
	}
}

func TestExplorePinnedByID(t *testing.T) {
	var query string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		query = r.URL.RawQuery
		common.WriteJSON(t, w, map[string]interface{}{"total_count": 1, "projects": []interface{}{}})
	}))
	defer server.Close()

	ctx := common.NewTestContext(t, server, "", "", map[string]string{"category": "32", "limit": "5"})
	if err := common.RunShortcut(t, Shortcuts(), "pinned", ctx); err != nil {
		t.Fatalf("explore +pinned failed: %v", err)
	}
	for _, want := range []string{"pinned=d", "category_id=32", "limit=5"} {
		if !strings.Contains(query, want) {
			t.Errorf("query missing %q; got %s", want, query)
		}
	}
}

func TestExplorePinnedByName(t *testing.T) {
	var pinnedQuery string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "project_categories") {
			common.WriteJSON(t, w, map[string]interface{}{
				"project_categories": []interface{}{
					map[string]interface{}{"id": 32, "name": "深度学习"},
				},
			})
			return
		}
		pinnedQuery = r.URL.RawQuery
		common.WriteJSON(t, w, map[string]interface{}{"total_count": 1, "projects": []interface{}{}})
	}))
	defer server.Close()

	ctx := common.NewTestContext(t, server, "", "", map[string]string{"category": "深度学习"})
	if err := common.RunShortcut(t, Shortcuts(), "pinned", ctx); err != nil {
		t.Fatalf("explore +pinned by name failed: %v", err)
	}
	if !strings.Contains(pinnedQuery, "category_id=32") {
		t.Errorf("expected name 深度学习 -> category_id=32; got %s", pinnedQuery)
	}
}
