package repo

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gitlink-org/gitlink-cli/internal/client"
	"github.com/gitlink-org/gitlink-cli/shortcuts/common"
)

func runShortcut(t *testing.T, server *httptest.Server, name string, args map[string]string) error {
	t.Helper()
	s := findShortcut(t, name)
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
	if ctx.Args == nil {
		ctx.Args = map[string]string{}
	}
	return s.Run(ctx)
}

func findShortcut(t *testing.T, name string) *common.Shortcut {
	t.Helper()
	for _, s := range Shortcuts() {
		if s.Name == name {
			return s
		}
	}
	t.Fatalf("shortcut %q not found", name)
	return nil
}

func writeJSON(t *testing.T, w http.ResponseWriter, v interface{}) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(v); err != nil {
		t.Fatalf("write response: %v", err)
	}
}

func writeText(t *testing.T, w http.ResponseWriter, code int, text string) {
	t.Helper()
	w.WriteHeader(code)
	if _, err := w.Write([]byte(text)); err != nil {
		t.Fatalf("write response: %v", err)
	}
}

func decodeRepoJSON(t *testing.T, r *http.Request) map[string]interface{} {
	t.Helper()
	var payload map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		t.Fatalf("decode request body: %v", err)
	}
	return payload
}

// --- list ---

func TestRepoListDefault(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Fatalf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/projects.json" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		writeJSON(t, w, map[string]interface{}{
			"total_count": float64(2),
			"data":        []interface{}{map[string]interface{}{"name": "repo1"}},
		})
	}))
	defer server.Close()

	err := runShortcut(t, server, "list", map[string]string{"page": "1", "limit": "20"})
	if err != nil {
		t.Fatalf("list failed: %v", err)
	}
}

func TestRepoListForUser(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/users/alice/projects.json" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		writeJSON(t, w, map[string]interface{}{"total_count": float64(0), "data": []interface{}{}})
	}))
	defer server.Close()

	err := runShortcut(t, server, "list", map[string]string{"user": "alice", "page": "1", "limit": "20"})
	if err != nil {
		t.Fatalf("list for user failed: %v", err)
	}
}

func TestRepoListWithCategory(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("category") != "mirror" {
			t.Fatalf("expected category=mirror, got %s", r.URL.Query().Get("category"))
		}
		writeJSON(t, w, map[string]interface{}{"data": []interface{}{}})
	}))
	defer server.Close()

	err := runShortcut(t, server, "list", map[string]string{"category": "mirror", "page": "1", "limit": "20"})
	if err != nil {
		t.Fatalf("list with category failed: %v", err)
	}
}

// --- info/readme/insights ---

func TestRepoInfo(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/owner/repo.json" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		writeJSON(t, w, map[string]interface{}{
			"name":        "repo",
			"description": "test repo",
		})
	}))
	defer server.Close()

	err := runShortcut(t, server, "info", nil)
	if err != nil {
		t.Fatalf("info failed: %v", err)
	}
}

func TestRepoReadmeUsesRepositoryReadmeEndpoint(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertRequest(t, r, "GET", "/owner/repo/readme.json")
		assertEqual(t, r.URL.Query().Get("ref"), "main")
		assertEqual(t, r.URL.Query().Get("filepath"), "docs")
		writeJSON(t, w, map[string]interface{}{
			"type":    "file",
			"name":    "README.md",
			"content": "# docs\n",
		})
	}))
	defer server.Close()

	err := runShortcut(t, server, "readme", map[string]string{
		"ref":  "main",
		"path": "docs",
	})
	if err != nil {
		t.Fatalf("readme shortcut failed: %v", err)
	}
}

func TestRepoTreeListsRootOnDefaultRef(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertRequest(t, r, "GET", "/owner/repo/sub_entries.json")
		if _, ok := r.URL.Query()["filepath"]; ok {
			t.Fatalf("did not expect filepath query for repository root, got %q", r.URL.Query().Get("filepath"))
		}
		assertEqual(t, r.URL.Query().Get("ref"), "master")
		writeJSON(t, w, map[string]interface{}{
			"entries": []map[string]interface{}{
				{"name": "README.md", "type": "file"},
			},
		})
	}))
	defer server.Close()

	if err := runShortcut(t, server, "tree", nil); err != nil {
		t.Fatalf("tree shortcut failed: %v", err)
	}
}

func TestRepoTreeUsesPathAndRef(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertRequest(t, r, "GET", "/owner/repo/sub_entries.json")
		assertEqual(t, r.URL.Query().Get("filepath"), "cmd")
		assertEqual(t, r.URL.Query().Get("ref"), "main")
		writeJSON(t, w, map[string]interface{}{
			"entries": []map[string]interface{}{
				{"name": "main.go", "type": "file"},
			},
		})
	}))
	defer server.Close()

	if err := runShortcut(t, server, "tree", map[string]string{"path": "cmd", "ref": "main"}); err != nil {
		t.Fatalf("tree shortcut failed: %v", err)
	}
}

func TestRepoTreeShortcutRegistersHelpFlags(t *testing.T) {
	tree := findShortcut(t, "tree")
	if tree.Description == "" {
		t.Fatal("tree shortcut description is empty")
	}

	flags := map[string]common.Flag{}
	for _, flag := range tree.Flags {
		flags[flag.Name] = flag
	}

	pathFlag, ok := flags["path"]
	if !ok {
		t.Fatal("tree shortcut missing path flag")
	}
	if pathFlag.Short != "p" || pathFlag.Usage == "" {
		t.Fatalf("unexpected path flag: %+v", pathFlag)
	}

	refFlag, ok := flags["ref"]
	if !ok {
		t.Fatal("tree shortcut missing ref flag")
	}
	if refFlag.Short != "r" || refFlag.Default != "master" || refFlag.Usage == "" {
		t.Fatalf("unexpected ref flag: %+v", refFlag)
	}
}

func TestRepoLanguagesUsesLanguagesEndpoint(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertRequest(t, r, "GET", "/owner/repo/languages.json")
		writeJSON(t, w, map[string]interface{}{"Go": "92.4%", "Shell": "7.6%"})
	}))
	defer server.Close()

	if err := runShortcut(t, server, "languages", nil); err != nil {
		t.Fatalf("languages shortcut failed: %v", err)
	}
}

func TestRepoContributorsUsesContributorsEndpoint(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertRequest(t, r, "GET", "/owner/repo/contributors.json")
		writeJSON(t, w, map[string]interface{}{"total_count": 1, "list": []interface{}{}})
	}))
	defer server.Close()

	if err := runShortcut(t, server, "contributors", nil); err != nil {
		t.Fatalf("contributors shortcut failed: %v", err)
	}
}

func TestRepoContributorStatsBuildsQuery(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertRequest(t, r, "GET", "/v1/owner/repo/contributors/stat.json")
		assertEqual(t, r.URL.Query().Get("ref"), "main")
		assertEqual(t, r.URL.Query().Get("pass_year"), "2")
		writeJSON(t, w, map[string]interface{}{"total_count": 1, "contributors": []interface{}{}})
	}))
	defer server.Close()

	err := runShortcut(t, server, "contributor-stats", map[string]string{
		"ref":       " main ",
		"pass-year": "2",
	})
	if err != nil {
		t.Fatalf("contributor-stats shortcut failed: %v", err)
	}
}

func TestRepoCodeStatsUsesRefQuery(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertRequest(t, r, "GET", "/v1/owner/repo/code_stats.json")
		assertEqual(t, r.URL.Query().Get("ref"), "release/v1")
		writeJSON(t, w, map[string]interface{}{"author_count": 1, "commit_count": 3})
	}))
	defer server.Close()

	if err := runShortcut(t, server, "code-stats", map[string]string{"ref": "release/v1"}); err != nil {
		t.Fatalf("code-stats shortcut failed: %v", err)
	}
}

func TestRepoFilesBuildsSearchAndRefQuery(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertRequest(t, r, "GET", "/owner/repo/files.json")
		assertEqual(t, r.URL.Query().Get("search"), "README")
		assertEqual(t, r.URL.Query().Get("ref"), "main")
		writeJSON(t, w, []interface{}{map[string]interface{}{"path": "README.md"}})
	}))
	defer server.Close()

	if err := runShortcut(t, server, "files", map[string]string{"search": "README", "ref": "main"}); err != nil {
		t.Fatalf("files shortcut failed: %v", err)
	}
}

func TestRepoCommitsBuildsRefPaginationQuery(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertRequest(t, r, "GET", "/v1/owner/repo/commits.json")
		assertEqual(t, r.URL.Query().Get("sha"), "main")
		assertEqual(t, r.URL.Query().Get("page"), "2")
		assertEqual(t, r.URL.Query().Get("limit"), "50")
		writeJSON(t, w, map[string]interface{}{"total_count": 1, "commits": []interface{}{}})
	}))
	defer server.Close()

	err := runShortcut(t, server, "commits", map[string]string{"ref": "main", "page": "2", "limit": "50"})
	if err != nil {
		t.Fatalf("commits shortcut failed: %v", err)
	}
}

func TestRepoCommitFilesUsesPaginationWhenNoFileFilter(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertRequest(t, r, "GET", "/v1/owner/repo/commits/abc123/files.json")
		assertEqual(t, r.URL.Query().Get("page"), "3")
		assertEqual(t, r.URL.Query().Get("limit"), "10")
		if got := r.URL.Query().Get("filepath"); got != "" {
			t.Fatalf("did not expect filepath query, got %q", got)
		}
		writeJSON(t, w, map[string]interface{}{"files": []interface{}{}})
	}))
	defer server.Close()

	err := runShortcut(t, server, "commit-files", map[string]string{"sha": "abc123", "page": "3", "limit": "10"})
	if err != nil {
		t.Fatalf("commit-files shortcut failed: %v", err)
	}
}

func TestRepoCommitFilesUsesFileFilterWithoutPagination(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertRequest(t, r, "GET", "/v1/owner/repo/commits/abc123/files.json")
		assertEqual(t, r.URL.Query().Get("filepath"), "src/main.go")
		if got := r.URL.Query().Get("page"); got != "" {
			t.Fatalf("did not expect page query with filepath filter, got %q", got)
		}
		writeJSON(t, w, map[string]interface{}{"files": []interface{}{}})
	}))
	defer server.Close()

	err := runShortcut(t, server, "commit-files", map[string]string{"sha": "abc123", "file": "src/main.go"})
	if err != nil {
		t.Fatalf("commit-files shortcut failed: %v", err)
	}
}

func TestRepoCommitDiffUsesCommitDiffEndpoint(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertRequest(t, r, "GET", "/v1/owner/repo/commits/abc123/diff.json")
		writeJSON(t, w, map[string]interface{}{"diff": "@@ -1 +1 @@"})
	}))
	defer server.Close()

	if err := runShortcut(t, server, "commit-diff", map[string]string{"sha": "abc123"}); err != nil {
		t.Fatalf("commit-diff shortcut failed: %v", err)
	}
}

func TestRepoTagsBuildsPaginationQuery(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertRequest(t, r, "GET", "/v1/owner/repo/tags.json")
		assertEqual(t, r.URL.Query().Get("page"), "2")
		assertEqual(t, r.URL.Query().Get("limit"), "30")
		writeJSON(t, w, map[string]interface{}{"total_count": 1, "tags": []interface{}{}})
	}))
	defer server.Close()

	err := runShortcut(t, server, "tags", map[string]string{"page": "2", "limit": "30"})
	if err != nil {
		t.Fatalf("tags shortcut failed: %v", err)
	}
}

func TestRepoTagsUsesSearchEndpointForNameFilter(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertRequest(t, r, "GET", "/owner/repo/tags.json")
		assertEqual(t, r.URL.Query().Get("name"), "v1")
		assertEqual(t, r.URL.Query().Get("only_name"), "true")
		writeJSON(t, w, map[string]interface{}{"total_count": 1, "tags": []interface{}{map[string]interface{}{"name": "v1.0.0"}}})
	}))
	defer server.Close()

	err := runShortcut(t, server, "tags", map[string]string{"name": "v1", "only-name": "true"})
	if err != nil {
		t.Fatalf("tags shortcut with name filter failed: %v", err)
	}
}

func TestRepoTagUsesTagDetailEndpoint(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertRequest(t, r, "GET", "/v1/owner/repo/tags/v1.0.0.json")
		writeJSON(t, w, map[string]interface{}{"name": "v1.0.0"})
	}))
	defer server.Close()

	if err := runShortcut(t, server, "tag", map[string]string{"name": "v1.0.0"}); err != nil {
		t.Fatalf("tag shortcut failed: %v", err)
	}
}

func TestRepoDeleteTagDryRunDoesNotCallAPI(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatalf("delete-tag dry-run should not call remote API: %s %s", r.Method, r.URL.Path)
	}))
	defer server.Close()

	err := runShortcut(t, server, "delete-tag", map[string]string{"name": "v1.0.0", "dry-run": "true"})
	if err != nil {
		t.Fatalf("delete-tag dry-run failed: %v", err)
	}
}

func TestRepoDeleteTagRequiresYesForRemoteDelete(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatalf("delete-tag without --yes should not call remote API: %s %s", r.Method, r.URL.Path)
	}))
	defer server.Close()

	err := runShortcut(t, server, "delete-tag", map[string]string{"name": "v1.0.0"})
	if err == nil {
		t.Fatal("expected error when delete-tag is missing --yes")
	}
}

func TestRepoDeleteTagWithYesCallsDeleteEndpoint(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertRequest(t, r, "DELETE", "/v1/owner/repo/tags/v1.0.0.json")
		writeJSON(t, w, map[string]interface{}{"status": 0, "message": "success"})
	}))
	defer server.Close()

	err := runShortcut(t, server, "delete-tag", map[string]string{"name": "v1.0.0", "yes": "true"})
	if err != nil {
		t.Fatalf("delete-tag failed: %v", err)
	}
}

func TestRepoBatchCommitDryRunDoesNotCallAPI(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatalf("dry-run should not call remote API: %s %s", r.Method, r.URL.Path)
	}))
	defer server.Close()

	err := runShortcut(t, server, "batch-commit", map[string]string{
		"branch":     "master",
		"message":    "docs: update",
		"files":      "update:README.md:# hello;delete:old.txt",
		"new-branch": "docs/update-readme",
		"dry-run":    "true",
	})
	if err != nil {
		t.Fatalf("batch-commit dry-run failed: %v", err)
	}
}

func TestRepoBatchCommitRequiresYesForRemoteWrite(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatalf("batch-commit without --yes should not call remote API: %s %s", r.Method, r.URL.Path)
	}))
	defer server.Close()

	err := runShortcut(t, server, "batch-commit", map[string]string{
		"branch":  "master",
		"message": "docs: update",
		"files":   "update:README.md:# hello",
	})
	if err == nil {
		t.Fatal("expected error when batch-commit is missing --yes")
	}
}

func TestRepoBatchCommitPostsPayloadWithYes(t *testing.T) {
	var payload map[string]interface{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertRequest(t, r, "POST", "/v1/owner/repo/contents/batch.json")
		payload = decodeRepoJSON(t, r)
		writeJSON(t, w, map[string]interface{}{"commit": map[string]interface{}{"sha": "abc123"}})
	}))
	defer server.Close()

	err := runShortcut(t, server, "batch-commit", map[string]string{
		"branch":          "master",
		"message":         "docs: update",
		"files":           "create:docs/new.md:# new;update:README.md:# hello;delete:old.txt",
		"encoding":        "text",
		"author-name":     "Alice",
		"author-email":    "alice@example.com",
		"committer-name":  "Bot",
		"committer-email": "bot@example.com",
		"yes":             "true",
	})
	if err != nil {
		t.Fatalf("batch-commit failed: %v", err)
	}
	assertEqual(t, payload["branch"], "master")
	assertEqual(t, payload["message"], "docs: update")
	assertEqual(t, payload["author_name"], "Alice")
	files, ok := payload["files"].([]interface{})
	if !ok || len(files) != 3 {
		t.Fatalf("expected three file operations, got %#v", payload["files"])
	}
	first, _ := files[0].(map[string]interface{})
	assertEqual(t, first["action_type"], "create")
	assertEqual(t, first["file_path"], "docs/new.md")
	assertEqual(t, first["content"], "# new")
	assertEqual(t, first["encoding"], "text")
}

func TestRepoBatchCommitRejectsInvalidFileSpec(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		t.Fatal("invalid file spec should fail before calling remote API")
	}))
	defer server.Close()

	err := runShortcut(t, server, "batch-commit", map[string]string{
		"branch":  "master",
		"message": "docs: update",
		"files":   "move:README.md:# hello",
		"dry-run": "true",
	})
	if err == nil {
		t.Fatal("expected invalid file action error")
	}
}

func TestRepoWatchersBuildsTimeRangeQuery(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertRequest(t, r, "GET", "/owner/repo/watchers.json")
		assertEqual(t, r.URL.Query().Get("start_at"), "1714521600")
		assertEqual(t, r.URL.Query().Get("end_at"), "1717200000")
		writeJSON(t, w, map[string]interface{}{"count": 1, "users": []interface{}{}})
	}))
	defer server.Close()

	err := runShortcut(t, server, "watchers", map[string]string{
		"start-at": "1714521600",
		"end-at":   "1717200000",
	})
	if err != nil {
		t.Fatalf("watchers shortcut failed: %v", err)
	}
}

func TestRepoStargazersUsesStargazersEndpoint(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertRequest(t, r, "GET", "/owner/repo/stargazers.json")
		writeJSON(t, w, map[string]interface{}{"count": 0, "users": []interface{}{}})
	}))
	defer server.Close()

	if err := runShortcut(t, server, "stargazers", nil); err != nil {
		t.Fatalf("stargazers shortcut failed: %v", err)
	}
}

func TestRepoFollowResolvesProjectID(t *testing.T) {
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		switch requests {
		case 1:
			assertRequest(t, r, "GET", "/owner/repo.json")
			writeJSON(t, w, map[string]interface{}{"id": float64(123)})
		case 2:
			assertRequest(t, r, "POST", "/watchers/follow.json")
			assertEqual(t, r.URL.Query().Get("target_type"), "project")
			assertEqual(t, r.URL.Query().Get("id"), "123")
			writeJSON(t, w, map[string]interface{}{"status": 0, "message": "success", "watched": true})
		default:
			t.Fatalf("unexpected extra request: %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	if err := runShortcut(t, server, "follow", nil); err != nil {
		t.Fatalf("follow shortcut failed: %v", err)
	}
	assertEqual(t, requests, 2)
}

func TestRepoUnfollowUsesExplicitProjectID(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertRequest(t, r, "DELETE", "/watchers/unfollow.json")
		assertEqual(t, r.URL.Query().Get("target_type"), "project")
		assertEqual(t, r.URL.Query().Get("id"), "456")
		writeJSON(t, w, map[string]interface{}{"status": 0, "message": "success", "watched": false})
	}))
	defer server.Close()

	if err := runShortcut(t, server, "unfollow", map[string]string{"project-id": "456"}); err != nil {
		t.Fatalf("unfollow shortcut failed: %v", err)
	}
}

func TestRepoLikeUsesExplicitProjectID(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertRequest(t, r, "POST", "/projects/456/praise_tread/like.json")
		writeJSON(t, w, map[string]interface{}{"status": 0, "message": "success"})
	}))
	defer server.Close()

	if err := runShortcut(t, server, "like", map[string]string{"project-id": "456"}); err != nil {
		t.Fatalf("like shortcut failed: %v", err)
	}
}

func TestRepoUnlikeResolvesStringProjectID(t *testing.T) {
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		switch requests {
		case 1:
			assertRequest(t, r, "GET", "/owner/repo.json")
			writeJSON(t, w, map[string]interface{}{"project_id": "789"})
		case 2:
			assertRequest(t, r, "DELETE", "/projects/789/praise_tread/unlike.json")
			writeJSON(t, w, map[string]interface{}{"status": 0, "message": "success"})
		default:
			t.Fatalf("unexpected extra request: %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	if err := runShortcut(t, server, "unlike", nil); err != nil {
		t.Fatalf("unlike shortcut failed: %v", err)
	}
	assertEqual(t, requests, 2)
}

func TestRepoInteractionDryRunDoesNotCallAPIWithExplicitProjectID(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatalf("dry-run with explicit project-id should not call API, got: %s %s", r.Method, r.URL.Path)
	}))
	defer server.Close()

	err := runShortcut(t, server, "like", map[string]string{
		"project-id": "456",
		"dry-run":    "true",
	})
	if err != nil {
		t.Fatalf("dry-run shortcut failed: %v", err)
	}
}

// --- fork/delete/create ---

func TestRepoFork(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Fatalf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/owner/repo/forks.json" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		writeJSON(t, w, map[string]interface{}{"message": "forked"})
	}))
	defer server.Close()

	err := runShortcut(t, server, "fork", nil)
	if err != nil {
		t.Fatalf("fork failed: %v", err)
	}
}

func TestRepoDelete(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "DELETE" {
			t.Fatalf("expected DELETE, got %s", r.Method)
		}
		if r.URL.Path != "/owner/repo.json" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		writeJSON(t, w, map[string]interface{}{"message": "deleted"})
	}))
	defer server.Close()

	err := runShortcut(t, server, "delete", nil)
	if err != nil {
		t.Fatalf("delete failed: %v", err)
	}
}

func TestRepoCreate(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/users/me.json" && r.Method == "GET":
			writeJSON(t, w, map[string]interface{}{
				"login":   "creator",
				"user_id": float64(42),
			})
		case r.URL.Path == "/creator/new-repo.json" && r.Method == "POST":
			writeJSON(t, w, map[string]interface{}{"name": "new-repo"})
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	err := runShortcut(t, server, "create", map[string]string{"name": "new-repo"})
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}
}

func TestRepoCreateWithOptions(t *testing.T) {
	var body map[string]interface{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/users/me.json":
			writeJSON(t, w, map[string]interface{}{
				"login":   "creator",
				"user_id": float64(42),
			})
		case "/creator/my-repo.json":
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode request body: %v", err)
			}
			writeJSON(t, w, map[string]interface{}{"name": "my-repo"})
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	err := runShortcut(t, server, "create", map[string]string{
		"name":        "my-repo",
		"description": "a test repo",
		"private":     "true",
	})
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}
	if body["description"] != "a test repo" {
		t.Fatalf("description = %v", body["description"])
	}
	if body["private"] != true {
		t.Fatalf("private = %v", body["private"])
	}
}

// --- validation/error paths ---

func TestRepoInsightValidation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatalf("invalid input should not call API, got: %s %s", r.Method, r.URL.Path)
	}))
	defer server.Close()

	cases := []struct {
		name     string
		shortcut string
		args     map[string]string
	}{
		{
			name:     "invalid pass year",
			shortcut: "contributor-stats",
			args:     map[string]string{"pass-year": "0"},
		},
		{
			name:     "invalid start timestamp",
			shortcut: "watchers",
			args:     map[string]string{"start-at": "abc"},
		},
		{
			name:     "start after end",
			shortcut: "stargazers",
			args:     map[string]string{"start-at": "20", "end-at": "10"},
		},
		{
			name:     "invalid project id",
			shortcut: "follow",
			args:     map[string]string{"project-id": "repo"},
		},
		{
			name:     "negative project id",
			shortcut: "unlike",
			args:     map[string]string{"project-id": "-1"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if err := runShortcut(t, server, tc.shortcut, tc.args); err == nil {
				t.Fatal("expected validation error")
			}
		})
	}
}

func TestRepoCreateFailsWithoutName(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("no API call should be made")
	}))
	defer server.Close()

	err := runShortcut(t, server, "create", map[string]string{})
	if err == nil {
		t.Fatal("expected error for missing name")
	}
}

func TestRepoListHTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeText(t, w, http.StatusInternalServerError, "server error")
	}))
	defer server.Close()

	err := runShortcut(t, server, "list", map[string]string{"page": "1", "limit": "20"})
	if err == nil {
		t.Fatal("expected error for HTTP 500")
	}
}

func TestRepoInfoHTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeText(t, w, http.StatusInternalServerError, "server error")
	}))
	defer server.Close()

	err := runShortcut(t, server, "info", nil)
	if err == nil {
		t.Fatal("expected error for HTTP 500")
	}
}

func TestRepoForkHTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeText(t, w, http.StatusInternalServerError, "server error")
	}))
	defer server.Close()

	err := runShortcut(t, server, "fork", nil)
	if err == nil {
		t.Fatal("expected error for HTTP 500")
	}
}

func TestRepoDeleteHTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeText(t, w, http.StatusInternalServerError, "server error")
	}))
	defer server.Close()

	err := runShortcut(t, server, "delete", nil)
	if err == nil {
		t.Fatal("expected error for HTTP 500")
	}
}

func TestRepoCreateGetUserHTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeText(t, w, http.StatusInternalServerError, "server error")
	}))
	defer server.Close()

	err := runShortcut(t, server, "create", map[string]string{"name": "new-repo"})
	if err == nil {
		t.Fatal("expected error when get user fails")
	}
}

func TestRepoCreateUserNoLogin(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(t, w, map[string]interface{}{"user_id": float64(42)})
	}))
	defer server.Close()

	err := runShortcut(t, server, "create", map[string]string{"name": "new-repo"})
	if err == nil {
		t.Fatal("expected error when user response has no login")
	}
}

func assertRequest(t *testing.T, r *http.Request, method, path string) {
	t.Helper()
	if r.Method != method || r.URL.Path != path {
		t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
	}
}

func assertEqual(t *testing.T, got interface{}, want interface{}) {
	t.Helper()
	if fmt.Sprintf("%v", got) != fmt.Sprintf("%v", want) {
		t.Fatalf("got %v (%T), want %v (%T)", got, got, want, want)
	}
}
