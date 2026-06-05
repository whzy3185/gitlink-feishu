package health

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gitlink-org/gitlink-cli/internal/client"
	"github.com/gitlink-org/gitlink-cli/shortcuts/common"
)

func TestFetchPRListPageUsesV1StatusFilter(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Fatalf("method=%s, want GET", r.Method)
		}
		if r.URL.Path != "/v1/owner/repo/pulls.json" {
			t.Fatalf("path=%s, want /v1/owner/repo/pulls.json", r.URL.Path)
		}
		if got := r.URL.Query().Get("status"); got != "1" {
			t.Fatalf("status=%q, want 1", got)
		}
		if got := r.URL.Query().Get("state"); got != "" {
			t.Fatalf("state should not be sent, got %q", got)
		}
		writeHealthJSON(t, w, map[string]interface{}{
			"total_count": 1,
			"pulls": []map[string]interface{}{
				{"id": 15414, "index": 109, "status": "merged"},
			},
		})
	}))
	defer server.Close()

	pulls, err := fetchPRListPage(healthTestContext(server), "merged", 2, 5)
	if err != nil {
		t.Fatalf("fetchPRListPage failed: %v", err)
	}
	if len(pulls) != 1 {
		t.Fatalf("len(pulls)=%d, want 1", len(pulls))
	}
}

func TestFetchIssueListPageUsesCategoryFilter(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Fatalf("method=%s, want GET", r.Method)
		}
		if r.URL.Path != "/v1/owner/repo/issues.json" {
			t.Fatalf("path=%s, want /v1/owner/repo/issues.json", r.URL.Path)
		}
		if got := r.URL.Query().Get("category"); got != "closed" {
			t.Fatalf("category=%q, want closed", got)
		}
		if got := r.URL.Query().Get("state"); got != "" {
			t.Fatalf("state should not be sent, got %q", got)
		}
		writeHealthJSON(t, w, map[string]interface{}{
			"total_count": 1,
			"issues": []map[string]interface{}{
				{"id": 140801, "project_issues_index": 1},
			},
		})
	}))
	defer server.Close()

	issues, err := fetchIssueListPage(healthTestContext(server), "owner", "repo", "closed", 1, 20)
	if err != nil {
		t.Fatalf("fetchIssueListPage failed: %v", err)
	}
	if len(issues) != 1 {
		t.Fatalf("len(issues)=%d, want 1", len(issues))
	}
}

func TestNormalizeListFilters(t *testing.T) {
	prCases := map[string]string{"open": "0", "opened": "0", "merged": "1", "closed": "2", "all": ""}
	for input, want := range prCases {
		if got := normalizePRListStatus(input); got != want {
			t.Fatalf("normalizePRListStatus(%q)=%q, want %q", input, got, want)
		}
	}
	issueCases := map[string]string{"open": "opened", "opened": "opened", "closed": "closed", "all": "all"}
	for input, want := range issueCases {
		if got := normalizeIssueListCategory(input); got != want {
			t.Fatalf("normalizeIssueListCategory(%q)=%q, want %q", input, got, want)
		}
	}
}

func healthTestContext(server *httptest.Server) *common.RuntimeContext {
	return &common.RuntimeContext{
		Client: &client.Client{HTTP: server.Client(), BaseURL: server.URL},
		Owner:  "owner",
		Repo:   "repo",
		Format: "json",
		Args:   map[string]string{},
	}
}

func writeHealthJSON(t *testing.T, w http.ResponseWriter, payload interface{}) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		t.Fatalf("write json: %v", err)
	}
}
