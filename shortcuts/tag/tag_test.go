package tag

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gitlink-org/gitlink-cli/internal/client"
	"github.com/gitlink-org/gitlink-cli/shortcuts/common"
)

func TestTagList(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" || r.URL.Path != "/v1/owner/repo/tags.json" {
			t.Fatalf("got request %s %s, want GET /v1/owner/repo/tags.json", r.Method, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(map[string]interface{}{"total_count": 1, "tags": []interface{}{map[string]interface{}{"name": "v1.0.0"}}}); err != nil {
			t.Fatalf("failed to write response: %v", err)
		}
	}))
	defer server.Close()

	if err := runTagShortcut(t, server, "list", map[string]string{"page": "1", "limit": "20"}); err != nil {
		t.Fatalf("list shortcut failed: %v", err)
	}
}

func TestTagListAllMergesPages(t *testing.T) {
	pages := map[string][]interface{}{
		"1": {map[string]interface{}{"name": "v1"}, map[string]interface{}{"name": "v2"}},
		"2": {map[string]interface{}{"name": "v3"}},
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(map[string]interface{}{"total_count": 3, "tags": pages[r.URL.Query().Get("page")]}); err != nil {
			t.Fatalf("failed to write response: %v", err)
		}
	}))
	defer server.Close()

	if err := runTagShortcut(t, server, "list", map[string]string{"all": "true", "page": "1", "limit": "2"}); err != nil {
		t.Fatalf("list --all shortcut failed: %v", err)
	}
}

func runTagShortcut(t *testing.T, server *httptest.Server, name string, args map[string]string) error {
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

func TestTagViewDirectShow(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/owner/repo/tags/v1.0.json" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"name":"v1.0"}`))
	}))
	defer server.Close()

	if err := runTagShortcut(t, server, "view", map[string]string{"name": "v1.0"}); err != nil {
		t.Fatalf("view failed: %v", err)
	}
}

func TestTagViewFallsBackToListScan(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/v1/owner/repo/tags/mytag.json" {
			w.Write([]byte(`{"status":-1,"message":"标签不存在！"}`))
			return
		}
		if r.URL.Path == "/v1/owner/repo/tags.json" {
			w.Write([]byte(`{"total_count":1,"tags":[{"name":"mytag","id":"abc"}]}`))
			return
		}
		t.Fatalf("unexpected path: %s", r.URL.Path)
	}))
	defer server.Close()

	if err := runTagShortcut(t, server, "view", map[string]string{"name": "mytag"}); err != nil {
		t.Fatalf("view fallback failed: %v", err)
	}
}
