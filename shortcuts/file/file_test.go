package file

import (
	"encoding/base64"
	"net/http"
	"testing"

	"github.com/gitlink-org/gitlink-cli/shortcuts/common"
)

func TestFileBrowse(t *testing.T) {
	server := common.NewTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" && r.URL.Path == "/owner/repo/sub_entries.json" {
			if r.URL.Query().Get("filepath") != "src/main.go" {
				t.Fatalf("expected filepath=src/main.go, got %s", r.URL.Query().Get("filepath"))
			}
			common.WriteJSON(t, w, map[string]interface{}{
				"entries": map[string]interface{}{
					"name": "main.go",
					"type": "file",
					"sha":  "abc123",
				},
			})
		} else {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	})
	defer server.Close()

	ctx := common.NewTestContext(t, server, "owner", "repo", map[string]string{
		"path": "src/main.go",
		"ref":  "master",
	})
	err := common.RunShortcut(t, Shortcuts(), "browse", ctx)
	if err != nil {
		t.Fatalf("browse failed: %v", err)
	}
}

func TestFileCreate(t *testing.T) {
	var createPayload map[string]interface{}
	server := common.NewTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" && r.URL.Path == "/owner/repo/create_file.json" {
			createPayload = common.DecodeJSON(t, r)
			common.WriteJSON(t, w, map[string]interface{}{
				"status":  0,
				"message": "创建成功",
			})
		} else {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	})
	defer server.Close()

	ctx := common.NewTestContext(t, server, "owner", "repo", map[string]string{
		"path":     "hello.txt",
		"content":  "Hello World",
		"branch":   "master",
		"messages": "",
	})
	err := common.RunShortcut(t, Shortcuts(), "create", ctx)
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}

	common.AssertEqual(t, createPayload["filepath"], "hello.txt")
	common.AssertEqual(t, createPayload["branch"], "master")

	decoded, err := base64.StdEncoding.DecodeString(createPayload["content"].(string))
	if err != nil {
		t.Fatalf("failed to decode base64 content: %v", err)
	}
	common.AssertEqual(t, string(decoded), "Hello World")
}

func TestFileDeleteFetchesSHA(t *testing.T) {
	var deletePayload map[string]interface{}
	server := common.NewTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == "GET" && r.URL.Path == "/owner/repo/sub_entries.json":
			common.WriteJSON(t, w, map[string]interface{}{
				"sha": "fetchedsha123",
			})
		case r.Method == "DELETE" && r.URL.Path == "/owner/repo/delete_file.json":
			deletePayload = common.DecodeJSON(t, r)
			common.WriteJSON(t, w, map[string]interface{}{
				"status":  0,
				"message": "删除成功",
			})
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	})
	defer server.Close()

	ctx := common.NewTestContext(t, server, "owner", "repo", map[string]string{
		"path":   "old-file.txt",
		"branch": "master",
	})
	err := common.RunShortcut(t, Shortcuts(), "delete", ctx)
	if err != nil {
		t.Fatalf("delete failed: %v", err)
	}

	common.AssertEqual(t, deletePayload["sha"], "fetchedsha123")
	common.AssertEqual(t, deletePayload["filepath"], "old-file.txt")
}
