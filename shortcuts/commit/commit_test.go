package commit

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gitlink-org/gitlink-cli/internal/client"
	"github.com/gitlink-org/gitlink-cli/shortcuts/common"
)

func runShortcut(t *testing.T, server *httptest.Server, name string, args map[string]string) error {
	t.Helper()
	for _, s := range Shortcuts() {
		if s.Name == name {
			ctx := &common.RuntimeContext{
				Client: &client.Client{HTTP: server.Client(), BaseURL: server.URL},
				Owner:  "owner",
				Repo:   "repo",
				Format: "json",
				Args:   args,
			}
			if ctx.Args == nil {
				ctx.Args = map[string]string{}
			}
			return s.Run(ctx)
		}
	}
	t.Fatalf("shortcut %q not found", name)
	return nil
}

func TestCommitListForwardsShaAndPaging(t *testing.T) {
	var query string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" || r.URL.Path != "/v1/owner/repo/commits.json" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		query = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"total_count":1,"commits":[]}`))
	}))
	defer server.Close()

	if err := runShortcut(t, server, "list", map[string]string{"sha": "develop", "page": "2", "limit": "5"}); err != nil {
		t.Fatalf("list failed: %v", err)
	}
	for _, want := range []string{"sha=develop", "page=2", "limit=5"} {
		if !strings.Contains(query, want) {
			t.Fatalf("expected %q in query, got %q", want, query)
		}
	}
}

func TestCommitRecentForwardsKeyword(t *testing.T) {
	var query string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" || r.URL.Path != "/v1/owner/repo/commits/recent.json" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		query = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"commits":[]}`))
	}))
	defer server.Close()

	if err := runShortcut(t, server, "recent", map[string]string{"keyword": "fix", "page": "1", "limit": "20"}); err != nil {
		t.Fatalf("recent failed: %v", err)
	}
	if !strings.Contains(query, "keyword=fix") {
		t.Fatalf("expected keyword=fix in query, got %q", query)
	}
}

func TestCommitDiffAndFilesBuildShaPaths(t *testing.T) {
	var paths []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"ok":true}`))
	}))
	defer server.Close()

	if err := runShortcut(t, server, "diff", map[string]string{"sha": "abc123"}); err != nil {
		t.Fatalf("diff failed: %v", err)
	}
	if err := runShortcut(t, server, "files", map[string]string{"sha": "abc123", "filepath": "app/x.rb", "page": "1", "limit": "20"}); err != nil {
		t.Fatalf("files failed: %v", err)
	}
	if paths[0] != "/v1/owner/repo/commits/abc123/diff.json" {
		t.Fatalf("diff path = %s", paths[0])
	}
	if paths[1] != "/v1/owner/repo/commits/abc123/files.json" {
		t.Fatalf("files path = %s", paths[1])
	}
}
