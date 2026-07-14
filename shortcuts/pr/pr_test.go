package pr

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/gitlink-org/gitlink-cli/internal/client"
	"github.com/gitlink-org/gitlink-cli/internal/output"
	"github.com/gitlink-org/gitlink-cli/shortcuts/common"
)

func TestPRCommentPostsToCorrectIssueJournal(t *testing.T) {
	var journalPayload map[string]interface{}
	var journalPath string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == "GET" && r.URL.Path == "/owner/repo/pulls/13.json":
			writeJSON(t, w, map[string]interface{}{
				"issue": map[string]interface{}{
					"id":      float64(142301),
					"subject": "test PR",
				},
				"pull_request": map[string]interface{}{
					"id": float64(14791),
				},
			})
		case r.Method == "POST" && r.URL.Path == "/v1/owner/repo/issues/142301/journals.json":
			journalPath = r.URL.Path
			journalPayload = decodeJSON(t, r)
			writeJSON(t, w, map[string]interface{}{
				"id":      float64(12345),
				"message": "评论成功",
			})
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	err := runPRShortcut(t, server, "comment", map[string]string{
		"id":   "13",
		"body": "LGTM, looks good!",
	})
	if err != nil {
		t.Fatalf("comment shortcut failed: %v", err)
	}

	if journalPath == "" {
		t.Fatal("journal endpoint was not called")
	}
	assertEqual(t, journalPayload["notes"], "LGTM, looks good!")
}

func TestPRCommentFailsWhenPRNotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		writeJSON(t, w, map[string]interface{}{
			"status": 404,
			"error":  "Not Found",
		})
	}))
	defer server.Close()

	err := runPRShortcut(t, server, "comment", map[string]string{
		"id":   "999",
		"body": "test",
	})
	if err == nil {
		t.Fatal("expected error for non-existent PR, got nil")
	}
}

func TestPRCommentFailsWhenIssueFieldMissing(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(t, w, map[string]interface{}{
			"pull_request": map[string]interface{}{
				"id": float64(14791),
			},
		})
	}))
	defer server.Close()

	err := runPRShortcut(t, server, "comment", map[string]string{
		"id":   "13",
		"body": "test",
	})
	if err == nil {
		t.Fatal("expected error when issue field is missing, got nil")
	}
}

func TestPRReviewCommentsListWithFilters(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Fatalf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/v1/owner/repo/pulls/42/journals.json" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		query := r.URL.Query()
		assertEqual(t, query.Get("review_id"), "12")
		assertEqual(t, query.Get("need_respond"), "true")
		assertEqual(t, query.Get("state"), "resolved")
		assertEqual(t, query.Get("parent_id"), "3")
		assertEqual(t, query.Get("path"), "cmd/api/api.go")
		assertEqual(t, query.Get("is_full"), "true")
		assertEqual(t, query.Get("sort_by"), "updated_on")
		assertEqual(t, query.Get("sort_direction"), "desc")
		writeJSON(t, w, map[string]interface{}{"total_count": float64(0), "journals": []interface{}{}})
	}))
	defer server.Close()

	err := runPRShortcut(t, server, "review-comments", map[string]string{
		"id":             "42",
		"review-id":      "12",
		"need-respond":   "true",
		"state":          "resolved",
		"parent-id":      "3",
		"path":           "cmd/api/api.go",
		"is-full":        "true",
		"sort-by":        "updated_on",
		"sort-direction": "desc",
	})
	if err != nil {
		t.Fatalf("review-comments failed: %v", err)
	}
}

func TestPRReviewCommentCreateAutoDiff(t *testing.T) {
	var payload map[string]interface{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == "GET" && r.URL.Path == "/owner/repo/pulls/42/files.json":
			writeJSON(t, w, map[string]interface{}{
				"files": []interface{}{
					map[string]interface{}{
						"name":        "cmd/api/api.go",
						"old_name":    "cmd/api/api.go",
						"addition":    float64(1),
						"deletion":    float64(0),
						"type":        float64(2),
						"isCreated":   false,
						"isDeleted":   false,
						"isBin":       false,
						"isLFSFile":   false,
						"isRenamed":   false,
						"isSubmodule": false,
						"sections": []interface{}{
							map[string]interface{}{
								"fileName": "cmd/api/api.go",
								"name":     "",
								"lines": []interface{}{
									map[string]interface{}{
										"leftIdx":  float64(0),
										"rightIdx": float64(0),
										"type":     float64(4),
										"content":  "@@ -1 +1 @@",
										"sectionInfo": map[string]interface{}{
											"path":          "cmd/api/api.go",
											"lastLeftIdx":   float64(0),
											"lastRightIdx":  float64(0),
											"leftIdx":       float64(1),
											"rightIdx":      float64(1),
											"leftHunkSize":  float64(1),
											"rightHunkSize": float64(1),
										},
									},
									map[string]interface{}{
										"leftIdx":     float64(0),
										"rightIdx":    float64(1),
										"type":        float64(2),
										"content":     "+package api",
										"sectionInfo": nil,
									},
								},
							},
						},
					},
				},
			})
		case r.Method == "POST" && r.URL.Path == "/v1/owner/repo/pulls/42/journals.json":
			payload = decodeJSON(t, r)
			writeJSON(t, w, map[string]interface{}{"id": float64(301), "note": "needs work"})
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	err := runPRShortcut(t, server, "review-comment", map[string]string{
		"id":        "42",
		"review-id": "12",
		"path":      "cmd/api/api.go",
		"line-code": "abc_0_1",
		"note":      "needs work",
		"type":      "problem",
		"commit":    "deadbeef",
	})
	if err != nil {
		t.Fatalf("review-comment failed: %v", err)
	}

	assertEqual(t, payload["type"], "problem")
	assertEqual(t, payload["review_id"], "12")
	assertEqual(t, payload["line_code"], "abc_0_1")
	assertEqual(t, payload["commit_id"], "deadbeef")
	diff, ok := payload["diff"].(map[string]interface{})
	if !ok {
		t.Fatalf("diff missing or wrong type: %#v", payload["diff"])
	}
	assertEqual(t, diff["name"], "cmd/api/api.go")
	assertEqual(t, diff["oldname"], "cmd/api/api.go")
	assertEqual(t, diff["is_created"], false)
	sections, ok := diff["sections"].([]interface{})
	if !ok || len(sections) != 1 {
		t.Fatalf("sections = %#v", diff["sections"])
	}
	section := sections[0].(map[string]interface{})
	assertEqual(t, section["file_name"], "cmd/api/api.go")
	lines := section["lines"].([]interface{})
	firstLine := lines[0].(map[string]interface{})
	assertEqual(t, firstLine["left_index"], float64(0))
	assertEqual(t, firstLine["right_index"], float64(0))
	assertEqual(t, firstLine["section_path"], "cmd/api/api.go")
	secondLine := lines[1].(map[string]interface{})
	assertEqual(t, secondLine["match"], float64(1))
}

func TestPRReviewCommentCreateDryRunWithDiffFile(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatalf("dry-run with diff file should not call API, got %s %s", r.Method, r.URL.Path)
	}))
	defer server.Close()

	diffPath := filepath.Join(t.TempDir(), "diff.json")
	if err := os.WriteFile(diffPath, []byte(`{
		"name":"cmd/api/api.go",
		"old_name":"cmd/api/api.go",
		"addition":1,
		"deletion":0,
		"type":2,
		"sections":[{"fileName":"cmd/api/api.go","name":"","lines":[{"leftIdx":0,"rightIdx":1,"type":2,"content":"+package api","sectionInfo":null}]}]
	}`), 0o600); err != nil {
		t.Fatalf("write diff file: %v", err)
	}

	err := runPRShortcut(t, server, "review-comment", map[string]string{
		"id":        "42",
		"review-id": "12",
		"path":      "cmd/api/api.go",
		"line-code": "abc_0_1",
		"note":      "needs work",
		"diff-file": diffPath,
		"dry-run":   "true",
	})
	if err != nil {
		t.Fatalf("review-comment dry-run failed: %v", err)
	}
}

func TestPRReviewCommentCreateFailsWhenDiffMissing(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" || r.URL.Path != "/owner/repo/pulls/42/files.json" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		writeJSON(t, w, map[string]interface{}{
			"files": []interface{}{
				map[string]interface{}{
					"name":     "README.md",
					"sections": []interface{}{},
					"addition": float64(1),
				},
			},
		})
	}))
	defer server.Close()

	err := runPRShortcut(t, server, "review-comment", map[string]string{
		"id":        "42",
		"review-id": "12",
		"path":      "cmd/api/api.go",
		"line-code": "abc_0_1",
		"note":      "needs work",
	})
	if err == nil {
		t.Fatal("expected missing diff error")
	}
}

func TestPRUpdateReviewComment(t *testing.T) {
	var payload map[string]interface{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "PUT" {
			t.Fatalf("expected PUT, got %s", r.Method)
		}
		if r.URL.Path != "/v1/owner/repo/pulls/42/journals/301.json" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		payload = decodeJSON(t, r)
		writeJSON(t, w, map[string]interface{}{"id": float64(301), "state": "resolved"})
	}))
	defer server.Close()

	err := runPRShortcut(t, server, "update-review-comment", map[string]string{
		"id":         "42",
		"comment-id": "301",
		"note":       "fixed",
		"state":      "resolved",
		"commit":     "deadbeef",
	})
	if err != nil {
		t.Fatalf("update-review-comment failed: %v", err)
	}
	assertEqual(t, payload["note"], "fixed")
	assertEqual(t, payload["state"], "resolved")
	assertEqual(t, payload["commit_id"], "deadbeef")
}

func TestPRUpdateReviewCommentRequiresChanges(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatalf("should not call API when no update fields are provided")
	}))
	defer server.Close()

	err := runPRShortcut(t, server, "update-review-comment", map[string]string{
		"id":         "42",
		"comment-id": "301",
	})
	if err == nil {
		t.Fatal("expected validation error")
	}
}

func TestPRDeleteReviewComment(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "DELETE" {
			t.Fatalf("expected DELETE, got %s", r.Method)
		}
		if r.URL.Path != "/v1/owner/repo/pulls/42/journals/301.json" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		writeJSON(t, w, map[string]interface{}{"message": "deleted"})
	}))
	defer server.Close()

	err := runPRShortcut(t, server, "delete-review-comment", map[string]string{
		"id":         "42",
		"comment-id": "301",
	})
	if err != nil {
		t.Fatalf("delete-review-comment failed: %v", err)
	}
}

// --- list ---

func TestPRList(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Fatalf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/v1/owner/repo/pulls.json" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if r.URL.Query().Get("status") != "0" {
			t.Fatalf("expected status=0, got %s", r.URL.Query().Get("status"))
		}
		if r.URL.Query().Get("page") != "1" {
			t.Fatalf("expected page=1, got %s", r.URL.Query().Get("page"))
		}
		writeJSON(t, w, []interface{}{
			map[string]interface{}{"id": float64(1), "title": "PR 1"},
		})
	}))
	defer server.Close()

	err := runPRShortcut(t, server, "list", map[string]string{"state": "open", "page": "1", "limit": "20"})
	if err != nil {
		t.Fatalf("list failed: %v", err)
	}
}

func TestPRListWithFilters(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/owner/repo/pulls.json" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		query := r.URL.Query()
		assertEqual(t, query.Get("status"), "1")
		assertEqual(t, query.Get("keyword"), "release")
		assertEqual(t, query.Get("priority_id"), "2")
		assertEqual(t, query.Get("issue_tag_id"), "3")
		assertEqual(t, query.Get("version_id"), "4")
		assertEqual(t, query.Get("reviewer_id"), "5")
		assertEqual(t, query.Get("assign_user_id"), "6")
		assertEqual(t, query.Get("sort_by"), "updated_at")
		assertEqual(t, query.Get("sort_direction"), "desc")
		writeJSON(t, w, map[string]interface{}{"pulls": []interface{}{}})
	}))
	defer server.Close()

	err := runPRShortcut(t, server, "list", map[string]string{
		"state":          "merged",
		"keyword":        "release",
		"priority-id":    "2",
		"tag-id":         "3",
		"milestone-id":   "4",
		"reviewer-id":    "5",
		"assignee-id":    "6",
		"sort-by":        "updated_at",
		"sort-direction": "desc",
		"page":           "2",
		"limit":          "50",
	})
	if err != nil {
		t.Fatalf("list with filters failed: %v", err)
	}
}

func TestPRListWithLoginFilter(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/owner/repo/pulls.json" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		writeJSON(t, w, map[string]interface{}{
			"pulls": []interface{}{
				map[string]interface{}{
					"id":    float64(1),
					"title": "PR by alice",
					"issue": map[string]interface{}{
						"author": map[string]interface{}{"login": "alice"},
					},
				},
				map[string]interface{}{
					"id":    float64(2),
					"title": "PR by bob",
					"issue": map[string]interface{}{
						"author": map[string]interface{}{"login": "bob"},
					},
				},
				map[string]interface{}{
					"id":    float64(3),
					"title": "PR by Alice (uppercase)",
					"issue": map[string]interface{}{
						"author": map[string]interface{}{"login": "Alice"},
					},
				},
			},
			"total_count": float64(3),
		})
	}))
	defer server.Close()

	err := runPRShortcut(t, server, "list", map[string]string{
		"state": "open",
		"login": "alice",
		"page":  "1",
		"limit": "20",
	})
	if err != nil {
		t.Fatalf("list with login filter failed: %v", err)
	}
}

func TestPRListStateAllOmitsStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/owner/repo/pulls.json" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if got := r.URL.Query().Get("status"); got != "" {
			t.Fatalf("status should be omitted for all, got %q", got)
		}
		writeJSON(t, w, map[string]interface{}{"pulls": []interface{}{}})
	}))
	defer server.Close()

	err := runPRShortcut(t, server, "list", map[string]string{"state": "all", "page": "1", "limit": "20"})
	if err != nil {
		t.Fatalf("list all failed: %v", err)
	}
}

func TestNormalizePullRequestListNumbersCopiesIndex(t *testing.T) {
	env := &output.Envelope{
		OK: true,
		Data: map[string]interface{}{
			"pulls": []interface{}{
				map[string]interface{}{
					"id":    float64(11),
					"index": float64(7),
					"title": "feat: show number",
				},
			},
		},
	}

	normalizePullRequestListNumbers(env)

	data := env.Data.(map[string]interface{})
	pulls := data["pulls"].([]interface{})
	pr := pulls[0].(map[string]interface{})
	assertEqual(t, pr["number"], float64(7))
}

func TestNormalizePullRequestListNumbersKeepsExistingNumber(t *testing.T) {
	env := &output.Envelope{
		OK: true,
		Data: map[string]interface{}{
			"pulls": []interface{}{
				map[string]interface{}{
					"number": float64(9),
					"index":  float64(7),
					"title":  "feat: keep number",
				},
			},
		},
	}

	normalizePullRequestListNumbers(env)

	data := env.Data.(map[string]interface{})
	pulls := data["pulls"].([]interface{})
	pr := pulls[0].(map[string]interface{})
	assertEqual(t, pr["number"], float64(9))
}

// --- create ---

func TestPRCreate(t *testing.T) {
	var payload map[string]interface{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Fatalf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/owner/repo/pulls.json" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		payload = decodeJSON(t, r)
		writeJSON(t, w, map[string]interface{}{"id": float64(42), "title": "feat: new"})
	}))
	defer server.Close()

	err := runPRShortcut(t, server, "create", map[string]string{
		"title": "feat: new",
		"head":  "feature/x",
		"base":  "master",
		"body":  "description",
	})
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}
	assertEqual(t, payload["title"], "feat: new")
	assertEqual(t, payload["head"], "feature/x")
	assertEqual(t, payload["base"], "master")
	assertEqual(t, payload["body"], "description")
}

func TestPRCreateNoBody(t *testing.T) {
	var payload map[string]interface{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/owner/repo.json" {
			writeJSON(t, w, map[string]interface{}{"default_branch": "main"})
			return
		}
		payload = decodeJSON(t, r)
		writeJSON(t, w, map[string]interface{}{"id": float64(43), "title": "feat: nob"})
	}))
	defer server.Close()

	err := runPRShortcut(t, server, "create", map[string]string{
		"title": "feat: nob",
		"head":  "feature/y",
	})
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}
	if _, ok := payload["body"]; ok {
		t.Fatal("body should not be in payload when not provided")
	}
	if payload["base"] != "main" {
		t.Fatalf("expected base to fall back to default branch main, got %v", payload["base"])
	}
}

// --- view ---

func TestPRView(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Fatalf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/owner/repo/pulls/42.json" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		writeJSON(t, w, map[string]interface{}{
			"id":    float64(42),
			"title": "feat: new",
		})
	}))
	defer server.Close()

	err := runPRShortcut(t, server, "view", map[string]string{"id": "42"})
	if err != nil {
		t.Fatalf("view failed: %v", err)
	}
}

func TestPRViewSurfacesMergedAt(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/owner/repo/pulls/42.json" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		writeJSON(t, w, map[string]interface{}{
			"id": float64(42),
			"pull_request": map[string]interface{}{
				"merged_at":          "2026-07-05T12:52:05+08:00",
				"merged":             true,
				"pull_request_staus": "merged",
			},
		})
	}))
	defer server.Close()

	ctx := &common.RuntimeContext{
		Client: &client.Client{HTTP: server.Client(), BaseURL: server.URL},
		Owner:  "owner",
		Repo:   "repo",
		Format: "json",
	}
	env, err := ctx.CallAPI("GET", "/owner/repo/pulls/42", nil)
	if err != nil {
		t.Fatalf("CallAPI error: %v", err)
	}
	if err := enrichPullRequestTimestamps(ctx, env); err != nil {
		t.Fatalf("enrich error: %v", err)
	}

	data, ok := env.Data.(map[string]interface{})
	if !ok {
		t.Fatalf("unexpected data type: %T", env.Data)
	}
	assertEqual(t, data["merged_at"], "2026-07-05T12:52:05+08:00")
	assertEqual(t, data["merged"], true)
}

// --- merge ---

func TestPRMerge(t *testing.T) {
	var payload map[string]interface{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Fatalf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/owner/repo/pulls/42/pr_merge.json" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		payload = decodeJSON(t, r)
		writeJSON(t, w, map[string]interface{}{"message": "merged"})
	}))
	defer server.Close()

	err := runPRShortcut(t, server, "merge", map[string]string{"id": "42"})
	if err != nil {
		t.Fatalf("merge failed: %v", err)
	}
	assertEqual(t, payload["do"], "merge")
}

func TestPRMergeSquash(t *testing.T) {
	var payload map[string]interface{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		payload = decodeJSON(t, r)
		writeJSON(t, w, map[string]interface{}{"message": "squashed"})
	}))
	defer server.Close()

	err := runPRShortcut(t, server, "merge", map[string]string{"id": "42", "method": "squash"})
	if err != nil {
		t.Fatalf("merge squash failed: %v", err)
	}
	assertEqual(t, payload["do"], "squash")
}

// --- refuse ---

func TestPRRefuse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Fatalf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/owner/repo/pulls/42/refuse_merge.json" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		writeJSON(t, w, map[string]interface{}{"message": "closed"})
	}))
	defer server.Close()

	err := runPRShortcut(t, server, "refuse", map[string]string{"id": "42"})
	if err != nil {
		t.Fatalf("refuse failed: %v", err)
	}
}

// --- files ---

func TestPRFiles(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Fatalf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/owner/repo/pulls/42/files.json" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		writeJSON(t, w, []interface{}{
			map[string]interface{}{"filename": "main.go", "status": "modified"},
		})
	}))
	defer server.Close()

	err := runPRShortcut(t, server, "files", map[string]string{"id": "42"})
	if err != nil {
		t.Fatalf("files failed: %v", err)
	}
}

// --- diff ---

func TestPRDiff(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Fatalf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/owner/repo/pulls/42/files.json" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		writeJSON(t, w, []interface{}{
			map[string]interface{}{"filename": "main.go", "patch": "@@ -1 +1 @@"},
		})
	}))
	defer server.Close()

	err := runPRShortcut(t, server, "diff", map[string]string{"id": "42"})
	if err != nil {
		t.Fatalf("diff failed: %v", err)
	}
}

// --- edit ---

func editServer(t *testing.T, current map[string]interface{}, put *map[string]interface{}) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/owner/repo/pulls/42.json" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		switch r.Method {
		case "GET":
			writeJSON(t, w, current)
		case "PUT":
			*put = decodeJSON(t, r)
			writeJSON(t, w, map[string]interface{}{})
		default:
			t.Fatalf("unexpected method: %s", r.Method)
		}
	}))
}

func TestPREditPreservesUnspecifiedFields(t *testing.T) {
	var put map[string]interface{}
	server := editServer(t, map[string]interface{}{
		"id":    float64(42),
		"title": "original title",
		"body":  "original body",
		"head":  "feature/x",
		"base":  "master",
		"issue": map[string]interface{}{
			"id": float64(105),
			"issue_tags": []interface{}{
				map[string]interface{}{"id": float64(7), "name": "bug"},
				map[string]interface{}{"id": float64(9), "name": "urgent"},
			},
		},
	}, &put)
	defer server.Close()

	err := runPRShortcut(t, server, "edit", map[string]string{
		"id":   "42",
		"body": "updated body",
		"base": "develop",
	})
	if err != nil {
		t.Fatalf("edit failed: %v", err)
	}
	if put == nil {
		t.Fatal("PUT was not called")
	}
	assertEqual(t, put["title"], "original title")
	assertEqual(t, put["body"], "updated body")
	assertEqual(t, put["head"], "feature/x")
	assertEqual(t, put["base"], "develop")

	tags, ok := put["issue_tag_ids"].([]interface{})
	if !ok || len(tags) != 2 || fmt.Sprintf("%v", tags[0]) != "7" || fmt.Sprintf("%v", tags[1]) != "9" {
		t.Fatalf("expected preserved tag ids [7 9], got %v", put["issue_tag_ids"])
	}
	if _, ok := put["receivers_login"].([]interface{}); !ok {
		t.Fatalf("expected receivers_login array, got %v", put["receivers_login"])
	}
}

func TestPREditTagIDsOverride(t *testing.T) {
	var put map[string]interface{}
	server := editServer(t, map[string]interface{}{
		"title": "keep",
		"body":  "keep",
		"head":  "src",
		"base":  "master",
		"issue": map[string]interface{}{
			"issue_tags": []interface{}{
				map[string]interface{}{"id": float64(7)},
			},
		},
	}, &put)
	defer server.Close()

	err := runPRShortcut(t, server, "edit", map[string]string{
		"id":      "42",
		"tag-ids": "3, 5",
	})
	if err != nil {
		t.Fatalf("edit failed: %v", err)
	}
	tags, ok := put["issue_tag_ids"].([]interface{})
	if !ok || len(tags) != 2 || fmt.Sprintf("%v", tags[0]) != "3" || fmt.Sprintf("%v", tags[1]) != "5" {
		t.Fatalf("expected tag ids [3 5], got %v", put["issue_tag_ids"])
	}
}

// The non-v1 detail endpoint nests the branch names under pull_request.head /
// pull_request.base (branch-name strings), which is the shape the live API
// actually returns; editing title-only must still preserve them.
func TestPREditReadsNestedPullRequestFields(t *testing.T) {
	var put map[string]interface{}
	server := editServer(t, map[string]interface{}{
		"pull_request": map[string]interface{}{
			"title": "nested title",
			"body":  "nested body",
			"head":  "topic",
			"base":  "main",
		},
		"issue": map[string]interface{}{"id": float64(1)},
	}, &put)
	defer server.Close()

	err := runPRShortcut(t, server, "edit", map[string]string{
		"id":    "42",
		"title": "new title",
	})
	if err != nil {
		t.Fatalf("edit failed: %v", err)
	}
	assertEqual(t, put["title"], "new title")
	assertEqual(t, put["body"], "nested body")
	assertEqual(t, put["head"], "topic")
	assertEqual(t, put["base"], "main")
}

func TestPREditRequiresAtLeastOneField(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatalf("no request expected, got %s %s", r.Method, r.URL.Path)
	}))
	defer server.Close()

	err := runPRShortcut(t, server, "edit", map[string]string{"id": "42"})
	if err == nil {
		t.Fatal("expected error when no edit fields are provided")
	}
}

func TestPREditHTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("server error"))
	}))
	defer server.Close()

	err := runPRShortcut(t, server, "edit", map[string]string{"id": "42", "title": "x"})
	if err == nil {
		t.Fatal("expected error for HTTP 500")
	}
}

// --- extractIssueID ---

func TestExtractIssueID(t *testing.T) {
	id, err := extractIssueID(&output.Envelope{Data: map[string]interface{}{
		"issue": map[string]interface{}{"id": float64(42)},
	}})
	if err != nil {
		t.Fatalf("extractIssueID error: %v", err)
	}
	if id != 42 {
		t.Fatalf("= %d, want 42", id)
	}
}

func TestExtractIssueIDNotMap(t *testing.T) {
	_, err := extractIssueID(&output.Envelope{Data: "not a map"})
	if err == nil {
		t.Fatal("expected error for non-map data")
	}
}

func TestExtractIssueIDMissingIssue(t *testing.T) {
	_, err := extractIssueID(&output.Envelope{Data: map[string]interface{}{"pr": map[string]interface{}{}}})
	if err == nil {
		t.Fatal("expected error for missing issue field")
	}
}

func TestExtractIssueIDMissingID(t *testing.T) {
	_, err := extractIssueID(&output.Envelope{Data: map[string]interface{}{
		"issue": map[string]interface{}{"subject": "test"},
	}})
	if err == nil {
		t.Fatal("expected error for missing issue.id")
	}
}

// --- HTTP error paths ---

func TestPRListHTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("server error"))
	}))
	defer server.Close()

	err := runPRShortcut(t, server, "list", map[string]string{"page": "1", "limit": "20"})
	if err == nil {
		t.Fatal("expected error for HTTP 500")
	}
}

func TestPRCreateHTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("server error"))
	}))
	defer server.Close()

	err := runPRShortcut(t, server, "create", map[string]string{"title": "test", "head": "feature/x"})
	if err == nil {
		t.Fatal("expected error for HTTP 500")
	}
}

func TestPRViewHTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("server error"))
	}))
	defer server.Close()

	err := runPRShortcut(t, server, "view", map[string]string{"id": "42"})
	if err == nil {
		t.Fatal("expected error for HTTP 500")
	}
}

func TestPRMergeHTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("server error"))
	}))
	defer server.Close()

	err := runPRShortcut(t, server, "merge", map[string]string{"id": "42"})
	if err == nil {
		t.Fatal("expected error for HTTP 500")
	}
}

func TestPRRefuseHTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("server error"))
	}))
	defer server.Close()

	err := runPRShortcut(t, server, "refuse", map[string]string{"id": "42"})
	if err == nil {
		t.Fatal("expected error for HTTP 500")
	}
}

func TestPRFilesHTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("server error"))
	}))
	defer server.Close()

	err := runPRShortcut(t, server, "files", map[string]string{"id": "42"})
	if err == nil {
		t.Fatal("expected error for HTTP 500")
	}
}

func TestPRDiffHTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("server error"))
	}))
	defer server.Close()

	err := runPRShortcut(t, server, "diff", map[string]string{"id": "42"})
	if err == nil {
		t.Fatal("expected error for HTTP 500")
	}
}

// --- status ---

func TestPRStatusGrouping(t *testing.T) {
	tests := []struct {
		name        string
		openPulls   []interface{}
		reviewPulls []interface{}
		wantCreated []string
		wantReview  []string
	}{
		{
			name: "groups created and review-requested",
			openPulls: []interface{}{
				makePull(1, "mine one", "currentuser"),
				makePull(2, "theirs", "someoneelse"),
				makePull(3, "mine two", "currentuser"),
			},
			reviewPulls: []interface{}{
				makePull(4, "review me", "author4"),
			},
			wantCreated: []string{"mine one", "mine two"},
			wantReview:  []string{"review me"},
		},
		{
			name: "author filter excludes other people",
			openPulls: []interface{}{
				makePull(2, "theirs", "someoneelse"),
			},
			reviewPulls: []interface{}{},
			wantCreated: []string{},
			wantReview:  []string{},
		},
		{
			name:        "empty repository yields empty groups",
			openPulls:   []interface{}{},
			reviewPulls: []interface{}{},
			wantCreated: []string{},
			wantReview:  []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var reviewerID string
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				switch {
				case r.URL.Path == "/users/me.json":
					writeJSON(t, w, map[string]interface{}{"login": "currentuser", "id": float64(7)})
				case r.URL.Path == "/v1/owner/repo/pulls.json":
					if got := r.URL.Query().Get("status"); got != "0" {
						t.Fatalf("expected status=0, got %q", got)
					}
					if rid := r.URL.Query().Get("reviewer_id"); rid != "" {
						reviewerID = rid
						writeJSON(t, w, map[string]interface{}{"pulls": tt.reviewPulls})
						return
					}
					writeJSON(t, w, map[string]interface{}{"pulls": tt.openPulls})
				default:
					t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
				}
			}))
			defer server.Close()

			result, err := collectPullStatus(statusContext(server))
			if err != nil {
				t.Fatalf("collectPullStatus failed: %v", err)
			}

			assertEqual(t, result["login"], "currentuser")
			assertPullTitles(t, result["created"], tt.wantCreated)
			assertPullTitles(t, result["review_requested"], tt.wantReview)
			if reviewerID != "7" {
				t.Fatalf("expected reviewer_id=7 sent to API, got %q", reviewerID)
			}
		})
	}
}

func TestPRStatusPropagatesUserError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("server error"))
	}))
	defer server.Close()

	if _, err := collectPullStatus(statusContext(server)); err == nil {
		t.Fatal("expected error when /users/me fails")
	}
}

func TestPRStatusMissingLogin(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(t, w, map[string]interface{}{"id": float64(7)})
	}))
	defer server.Close()

	if _, err := collectPullStatus(statusContext(server)); err == nil {
		t.Fatal("expected error when login is missing")
	}
}

func TestPRStatusRun(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/users/me.json":
			writeJSON(t, w, map[string]interface{}{"login": "currentuser", "id": float64(7)})
		case "/v1/owner/repo/pulls.json":
			writeJSON(t, w, map[string]interface{}{"pulls": []interface{}{}})
		default:
			t.Fatalf("unexpected request: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	if err := runPRShortcut(t, server, "status", nil); err != nil {
		t.Fatalf("status run failed: %v", err)
	}
}

func statusContext(server *httptest.Server) *common.RuntimeContext {
	return &common.RuntimeContext{
		Client: &client.Client{HTTP: server.Client(), BaseURL: server.URL},
		Owner:  "owner",
		Repo:   "repo",
		Format: "json",
	}
}

func makePull(id int, title, authorLogin string) map[string]interface{} {
	return map[string]interface{}{
		"id":    float64(id),
		"title": title,
		"issue": map[string]interface{}{
			"author": map[string]interface{}{"login": authorLogin},
		},
	}
}

func assertPullTitles(t *testing.T, raw interface{}, want []string) {
	t.Helper()
	items, ok := raw.([]interface{})
	if !ok {
		t.Fatalf("expected []interface{}, got %T", raw)
	}
	if len(items) != len(want) {
		t.Fatalf("expected %d pulls, got %d", len(want), len(items))
	}
	for i, it := range items {
		m, ok := it.(map[string]interface{})
		if !ok {
			t.Fatalf("pull %d not a map: %T", i, it)
		}
		if got := stringField(m, "title"); got != want[i] {
			t.Fatalf("pull %d title = %q, want %q", i, got, want[i])
		}
	}
}

func runPRShortcut(t *testing.T, server *httptest.Server, name string, args map[string]string) error {
	t.Helper()
	shortcut := findPRShortcut(t, name)
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

func findPRShortcut(t *testing.T, name string) *common.Shortcut {
	t.Helper()
	for _, shortcut := range Shortcuts() {
		if shortcut.Name == name {
			return shortcut
		}
	}
	t.Fatalf("shortcut %q not found", name)
	return nil
}

func decodeJSON(t *testing.T, r *http.Request) map[string]interface{} {
	t.Helper()
	var payload map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		t.Fatalf("failed to decode request body: %v", err)
	}
	return payload
}

func writeJSON(t *testing.T, w http.ResponseWriter, payload interface{}) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		t.Fatalf("failed to write response: %v", err)
	}
}

func assertEqual(t *testing.T, got interface{}, want interface{}) {
	t.Helper()
	if fmt.Sprintf("%v", got) != fmt.Sprintf("%v", want) {
		t.Fatalf("got %v (%T), want %v (%T)", got, got, want, want)
	}
}

func TestPRCommentsUsesPullJournalsEndpoint(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" || r.URL.Path != "/v1/owner/repo/pulls/382/journals.json" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		writeJSON(t, w, map[string]interface{}{"journals": []interface{}{}, "total_count": float64(0)})
	}))
	defer server.Close()

	if err := runPRShortcut(t, server, "comments", map[string]string{"id": "382"}); err != nil {
		t.Fatalf("comments failed: %v", err)
	}
}

func TestPRCommentEditSendsNoteAndState(t *testing.T) {
	var payload map[string]interface{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "PATCH" || r.URL.Path != "/v1/owner/repo/pulls/382/journals/484052.json" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		payload = decodeJSON(t, r)
		writeJSON(t, w, map[string]interface{}{"id": float64(484052)})
	}))
	defer server.Close()

	args := map[string]string{"id": "382", "comment-id": "484052", "body": "edited", "state": "resolved"}
	if err := runPRShortcut(t, server, "comment-edit", args); err != nil {
		t.Fatalf("comment-edit failed: %v", err)
	}
	assertEqual(t, payload["note"], "edited")
	assertEqual(t, payload["state"], "resolved")

	args["state"] = "bogus"
	if err := runPRShortcut(t, server, "comment-edit", args); err == nil {
		t.Fatal("expected error for invalid --state")
	}
	args["state"] = "opened"
	args["comment-id"] = "abc"
	if err := runPRShortcut(t, server, "comment-edit", args); err == nil {
		t.Fatal("expected error for non-integer --comment-id")
	}
}

func TestPRCommentDeleteUsesPullJournalsEndpoint(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "DELETE" || r.URL.Path != "/v1/owner/repo/pulls/382/journals/484052.json" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		writeJSON(t, w, map[string]interface{}{"status": float64(0)})
	}))
	defer server.Close()

	if err := runPRShortcut(t, server, "comment-delete", map[string]string{"id": "382", "comment-id": "484052"}); err != nil {
		t.Fatalf("comment-delete failed: %v", err)
	}
}
