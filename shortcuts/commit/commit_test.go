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
	server := newCommitServer(t, func(w http.ResponseWriter, r *http.Request) {
		assertRequest(t, r, "GET", "/v1/owner/repo/commits.json")
		assertEqual(t, r.URL.Query().Get("sha"), "master")
		assertEqual(t, r.URL.Query().Get("page"), "2")
		writeJSON(t, w, map[string]interface{}{"total_count": 0, "commits": []interface{}{}})
	})
	defer server.Close()
	if err := runCommitShortcut(t, server, "list", map[string]string{"sha": "master", "page": "2", "limit": "20"}); err != nil {
		t.Fatalf("list failed: %v", err)
	}
}

func TestCommitFiles(t *testing.T) {
	server := newCommitServer(t, func(w http.ResponseWriter, r *http.Request) {
		assertRequest(t, r, "GET", "/v1/owner/repo/commits/abc/files.json")
		assertEqual(t, r.URL.Query().Get("filepath"), "README.md")
		writeJSON(t, w, map[string]interface{}{"files": []interface{}{}})
	})
	defer server.Close()
	if err := runCommitShortcut(t, server, "files", map[string]string{"sha": "abc", "filepath": "README.md"}); err != nil {
		t.Fatalf("files failed: %v", err)
	}
}

func TestCommitDiff(t *testing.T) {
	server := newCommitServer(t, func(w http.ResponseWriter, r *http.Request) {
		assertRequest(t, r, "GET", "/v1/owner/repo/commits/abc/diff.json")
		writeJSON(t, w, map[string]interface{}{"files": []interface{}{}})
	})
	defer server.Close()
	if err := runCommitShortcut(t, server, "diff", map[string]string{"sha": "abc"}); err != nil {
		t.Fatalf("diff failed: %v", err)
	}
}

func TestCommitBlame(t *testing.T) {
	server := newCommitServer(t, func(w http.ResponseWriter, r *http.Request) {
		assertRequest(t, r, "GET", "/v1/owner/repo/blame.json")
		assertEqual(t, r.URL.Query().Get("sha"), "master")
		assertEqual(t, r.URL.Query().Get("filepath"), "README.md")
		writeJSON(t, w, map[string]interface{}{"file_name": "README.md"})
	})
	defer server.Close()
	if err := runCommitShortcut(t, server, "blame", map[string]string{"sha": "master", "filepath": "README.md"}); err != nil {
		t.Fatalf("blame failed: %v", err)
	}
}

func runCommitShortcut(t *testing.T, server *httptest.Server, name string, args map[string]string) error {
	t.Helper()
	var shortcut *common.Shortcut
	for _, s := range Shortcuts() {
		if s.Name == name {
			shortcut = s
		}
	}
	if shortcut == nil {
		t.Fatalf("shortcut %q not found", name)
	}
	return shortcut.Run(&common.RuntimeContext{Client: &client.Client{HTTP: server.Client(), BaseURL: server.URL}, Owner: "owner", Repo: "repo", Format: "json", Args: args})
}

func newCommitServer(t *testing.T, handler http.HandlerFunc) *httptest.Server {
	t.Helper()
	return httptest.NewServer(handler)
}

func assertRequest(t *testing.T, r *http.Request, method, path string) {
	t.Helper()
	if r.Method != method || r.URL.Path != path {
		t.Fatalf("got %s %s, want %s %s", r.Method, r.URL.Path, method, path)
	}
}

func writeJSON(t *testing.T, w http.ResponseWriter, payload interface{}) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		t.Fatalf("write json: %v", err)
	}
}

func assertEqual(t *testing.T, got, want interface{}) {
	t.Helper()
	if got != want {
		t.Fatalf("got %v (%T), want %v (%T)", got, got, want, want)
	}
}
