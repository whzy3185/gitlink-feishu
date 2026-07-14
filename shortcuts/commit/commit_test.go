package commit

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gitlink-org/gitlink-cli/internal/client"
	"github.com/gitlink-org/gitlink-cli/shortcuts/common"
)

func TestCommitList(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertRequest(t, r, "GET", "/v1/owner/repo/commits.json")
		if got := r.URL.Query().Get("sha"); got != "develop" {
			t.Fatalf("sha query = %q, want develop", got)
		}
		writeJSON(t, w, map[string]interface{}{"total_count": 1, "commits": []interface{}{map[string]interface{}{"sha": "abc"}}})
	}))
	defer server.Close()

	if err := runCommitShortcut(t, server, "list", map[string]string{"ref": "develop", "page": "1", "limit": "20"}); err != nil {
		t.Fatalf("list shortcut failed: %v", err)
	}
}

func TestCommitListAllMergesPages(t *testing.T) {
	pages := map[string][]interface{}{
		"1": {map[string]interface{}{"sha": "a"}, map[string]interface{}{"sha": "b"}},
		"2": {map[string]interface{}{"sha": "c"}},
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertRequest(t, r, "GET", "/v1/owner/repo/commits.json")
		writeJSON(t, w, map[string]interface{}{"total_count": 3, "commits": pages[r.URL.Query().Get("page")]})
	}))
	defer server.Close()

	if err := runCommitShortcut(t, server, "list", map[string]string{"all": "true", "page": "1", "limit": "2"}); err != nil {
		t.Fatalf("list --all shortcut failed: %v", err)
	}
}

func TestCommitView(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertRequest(t, r, "GET", "/owner/repo/commits/abc123.json")
		writeJSON(t, w, map[string]interface{}{"commit": map[string]interface{}{"sha": "abc123"}})
	}))
	defer server.Close()

	if err := runCommitShortcut(t, server, "view", map[string]string{"sha": "abc123"}); err != nil {
		t.Fatalf("view shortcut failed: %v", err)
	}
}

func runCommitShortcut(t *testing.T, server *httptest.Server, name string, args map[string]string) error {
	t.Helper()
	for _, shortcut := range Shortcuts() {
		if shortcut.Name == name {
			ctx := &common.RuntimeContext{
				Client: &client.Client{
					HTTP:    server.Client(),
					BaseURL: server.URL,
				},
				Owner:  "owner",
				Repo:   "repo",
				Format: "json",
				Args:   args,
			}
			return shortcut.Run(ctx)
		}
	}
	t.Fatalf("shortcut %q not found", name)
	return nil
}

func assertRequest(t *testing.T, r *http.Request, method, path string) {
	t.Helper()
	if r.Method != method || r.URL.Path != path {
		t.Fatalf("got request %s %s, want %s %s", r.Method, r.URL.Path, method, path)
	}
}

func writeJSON(t *testing.T, w http.ResponseWriter, payload interface{}) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		t.Fatalf("failed to write response: %v", err)
	}
}
