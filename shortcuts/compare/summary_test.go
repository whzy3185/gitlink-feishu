package compare

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func TestCompareFilesRejectsInvalidLimit(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
	}))
	defer server.Close()

	err := runCompareShortcut(t, server, "files", map[string]string{
		"head":  "feature/api",
		"base":  "master",
		"limit": "0",
	})
	if err == nil {
		t.Fatal("expected error for invalid --limit")
	}
}

func TestCompareCommitsFiltersAndLimitsResults(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" || r.URL.Path != "/owner/repo/compare/ZmVhdHVyZS9hcGk...bWFzdGVy.json" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		writeJSON(t, w, map[string]interface{}{
			"message":       "can merge",
			"commits_count": float64(3),
			"commits": []interface{}{
				map[string]interface{}{
					"sha":        "aaa111",
					"message":    "fix: normalize timestamps\n\nbody",
					"created_at": "2026-06-10 10:00",
					"author":     map[string]interface{}{"login": "alice", "name": "Alice"},
					"committer":  map[string]interface{}{"login": "alice", "name": "Alice"},
				},
				map[string]interface{}{
					"sha":        "bbb222",
					"message":    "feat: add compare summary",
					"created_at": "2026-06-10 11:00",
					"author":     map[string]interface{}{"login": "bob", "name": "Bob"},
					"committer":  map[string]interface{}{"login": "bob", "name": "Bob"},
				},
				map[string]interface{}{
					"sha":        "ccc333",
					"message":    "fix: docs cleanup",
					"created_at": "2026-06-10 12:00",
					"author":     map[string]interface{}{"login": "alice", "name": "Alice"},
					"committer":  map[string]interface{}{"login": "alice", "name": "Alice"},
				},
			},
		})
	}))
	defer server.Close()

	data := runCompareShortcutCapture(t, server, "commits", map[string]string{
		"head":    "feature/api",
		"base":    "master",
		"author":  "alice",
		"keyword": "fix",
		"limit":   "1",
	})

	assertJSONField(t, data, "repository", "owner/repo")
	assertJSONField(t, data, "head", "feature/api")
	assertJSONField(t, data, "base", "master")
	assertJSONField(t, data, "compare_message", "can merge")
	assertJSONField(t, data, "total_commits", float64(3))
	assertJSONField(t, data, "matched_commits", float64(2))
	assertJSONField(t, data, "returned_commits", float64(1))
	assertJSONField(t, data, "truncated", true)

	commits := data["commits"].([]interface{})
	if len(commits) != 1 {
		t.Fatalf("len(commits) = %d, want 1", len(commits))
	}
	commit := commits[0].(map[string]interface{})
	assertJSONField(t, commit, "sha", "aaa111")
	assertJSONField(t, commit, "subject", "fix: normalize timestamps")
	assertJSONField(t, commit, "author_login", "alice")
}

func TestCompareSummaryAggregatesResults(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == "GET" && r.URL.Path == "/owner/repo/compare/ZmVhdHVyZS9hcGk...bWFzdGVy.json":
			writeJSON(t, w, map[string]interface{}{
				"message":       "can merge",
				"commits_count": float64(2),
				"files_count":   float64(3),
				"commits": []interface{}{
					map[string]interface{}{
						"sha":        "aaa111",
						"message":    "feat: add compare summary",
						"created_at": "2026-06-10 10:00",
						"author":     map[string]interface{}{"login": "alice", "name": "Alice"},
						"committer":  map[string]interface{}{"login": "alice", "name": "Alice"},
					},
					map[string]interface{}{
						"sha":        "bbb222",
						"message":    "test: add compare summary tests",
						"created_at": "2026-06-10 11:00",
						"author":     map[string]interface{}{"login": "bob", "name": "Bob"},
						"committer":  map[string]interface{}{"login": "bob", "name": "Bob"},
					},
				},
			})
		case r.Method == "GET" && r.URL.Path == "/v1/owner/repo/compare/ZmVhdHVyZS9hcGk...bWFzdGVy/files.json":
			if got := r.URL.Query().Get("page"); got != "1" {
				t.Fatalf("page query = %q, want 1", got)
			}
			if got := r.URL.Query().Get("limit"); got != "10" {
				t.Fatalf("limit query = %q, want 10", got)
			}
			writeJSON(t, w, map[string]interface{}{
				"file_nums":      float64(3),
				"total_addition": float64(20),
				"total_deletion": float64(5),
				"message":        "can merge",
				"files": []interface{}{
					map[string]interface{}{
						"filename":   "README.md",
						"additions":  float64(1),
						"deletions":  float64(0),
						"changes":    float64(1),
						"is_created": false,
						"is_deleted": false,
						"is_renamed": false,
						"is_bin":     false,
					},
					map[string]interface{}{
						"filename":   "cmd/api/api.go",
						"additions":  float64(12),
						"deletions":  float64(5),
						"changes":    float64(17),
						"is_created": false,
						"is_deleted": false,
						"is_renamed": false,
						"is_bin":     false,
					},
					map[string]interface{}{
						"filename":   "docs/guide.md",
						"additions":  float64(7),
						"deletions":  float64(0),
						"changes":    float64(7),
						"is_created": true,
						"is_deleted": false,
						"is_renamed": false,
						"is_bin":     false,
					},
				},
			})
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	data := runCompareShortcutCapture(t, server, "summary", map[string]string{
		"head":         "feature/api",
		"base":         "master",
		"max-files":    "10",
		"top-files":    "2",
		"commit-limit": "1",
	})

	assertJSONField(t, data, "repository", "owner/repo")
	assertJSONField(t, data, "compare_message", "can merge")
	assertJSONField(t, data, "commits_count", float64(2))
	assertJSONField(t, data, "files_count", float64(3))
	assertJSONField(t, data, "files_analyzed", float64(3))
	assertJSONField(t, data, "truncated_files", false)

	authors := data["authors"].([]interface{})
	if len(authors) != 2 || authors[0] != "alice" || authors[1] != "bob" {
		t.Fatalf("authors = %v, want [alice bob]", authors)
	}

	commitsSample := data["commits_sample"].([]interface{})
	if len(commitsSample) != 1 {
		t.Fatalf("len(commits_sample) = %d, want 1", len(commitsSample))
	}
	changeTotals := data["change_totals"].(map[string]interface{})
	assertJSONField(t, changeTotals, "additions", float64(20))
	assertJSONField(t, changeTotals, "deletions", float64(5))
	assertJSONField(t, changeTotals, "changes", float64(25))

	fileTypes := data["file_types"].(map[string]interface{})
	assertJSONField(t, fileTypes, "created", float64(1))
	assertJSONField(t, fileTypes, "modified", float64(2))

	pathGroups := data["path_groups"].([]interface{})
	if len(pathGroups) == 0 {
		t.Fatal("expected path_groups to be populated")
	}
	firstPathGroup := pathGroups[0].(map[string]interface{})
	assertJSONField(t, firstPathGroup, "name", "cmd")

	extensions := data["extensions"].([]interface{})
	if len(extensions) == 0 {
		t.Fatal("expected extensions to be populated")
	}
	firstExtension := extensions[0].(map[string]interface{})
	assertJSONField(t, firstExtension, "name", ".go")

	topFiles := data["top_files"].([]interface{})
	if len(topFiles) != 2 {
		t.Fatalf("len(top_files) = %d, want 2", len(topFiles))
	}
	firstTopFile := topFiles[0].(map[string]interface{})
	assertJSONField(t, firstTopFile, "filename", "cmd/api/api.go")
	assertJSONField(t, firstTopFile, "status", "modified")
}

func TestCompareSummaryMarksTruncatedWhenMaxFilesIsSmallerThanTotal(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == "GET" && r.URL.Path == "/owner/repo/compare/ZmVhdHVyZS9hcGk...bWFzdGVy.json":
			writeJSON(t, w, map[string]interface{}{
				"message":       "can merge",
				"commits_count": float64(1),
				"files_count":   float64(3),
				"commits": []interface{}{
					map[string]interface{}{
						"sha":     "aaa111",
						"message": "fix: compare summary",
						"author":  map[string]interface{}{"login": "alice", "name": "Alice"},
					},
				},
			})
		case r.Method == "GET" && r.URL.Path == "/v1/owner/repo/compare/ZmVhdHVyZS9hcGk...bWFzdGVy/files.json":
			if got := r.URL.Query().Get("limit"); got != "2" {
				t.Fatalf("limit query = %q, want 2", got)
			}
			writeJSON(t, w, map[string]interface{}{
				"file_nums":      float64(3),
				"total_addition": float64(10),
				"total_deletion": float64(1),
				"files": []interface{}{
					map[string]interface{}{
						"filename":   "README.md",
						"additions":  float64(1),
						"deletions":  float64(0),
						"changes":    float64(1),
						"is_created": false,
						"is_deleted": false,
						"is_renamed": false,
						"is_bin":     false,
					},
					map[string]interface{}{
						"filename":   "shortcuts/compare/summary.go",
						"additions":  float64(9),
						"deletions":  float64(1),
						"changes":    float64(10),
						"is_created": true,
						"is_deleted": false,
						"is_renamed": false,
						"is_bin":     false,
					},
				},
			})
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	data := runCompareShortcutCapture(t, server, "summary", map[string]string{
		"head":      "feature/api",
		"base":      "master",
		"max-files": "2",
	})

	assertJSONField(t, data, "files_count", float64(3))
	assertJSONField(t, data, "files_analyzed", float64(2))
	assertJSONField(t, data, "truncated_files", true)
}

func runCompareShortcutCapture(t *testing.T, server *httptest.Server, name string, args map[string]string) map[string]interface{} {
	t.Helper()

	oldStdout := os.Stdout
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe error: %v", err)
	}
	defer reader.Close()
	defer writer.Close()
	defer func() {
		os.Stdout = oldStdout
	}()
	os.Stdout = writer
	runErr := runCompareShortcut(t, server, name, args)
	writer.Close()

	if runErr != nil {
		t.Fatalf("shortcut %q returned error: %v", name, runErr)
	}

	outputBytes, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("ReadAll error: %v", err)
	}
	var envelope map[string]interface{}
	if err := json.Unmarshal(outputBytes, &envelope); err != nil {
		t.Fatalf("failed to unmarshal command output %q: %v", string(outputBytes), err)
	}
	data, ok := envelope["data"].(map[string]interface{})
	if !ok {
		t.Fatalf("command output missing data object: %v", envelope)
	}
	return data
}

func assertJSONField(t *testing.T, data map[string]interface{}, key string, want interface{}) {
	t.Helper()
	if got := data[key]; got != want {
		t.Fatalf("%s = %v (%T), want %v (%T)", key, got, got, want, want)
	}
}
