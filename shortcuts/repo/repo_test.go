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

func TestRepoReadmeUsesRepositoryReadmeEndpoint(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" || r.URL.Path != "/owner/repo/readme.json" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		if got := r.URL.Query().Get("ref"); got != "main" {
			t.Fatalf("ref query = %q, want main", got)
		}
		if got := r.URL.Query().Get("filepath"); got != "docs" {
			t.Fatalf("filepath query = %q, want docs", got)
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(map[string]interface{}{
			"type":    "file",
			"name":    "README.md",
			"content": "# docs\n",
		}); err != nil {
			t.Fatalf("write response: %v", err)
		}
	}))
	defer server.Close()

	err := runRepoShortcut(server, "readme", map[string]string{
		"ref":  "main",
		"path": "docs",
	})
	if err != nil {
		t.Fatalf("readme shortcut failed: %v", err)
	}
}

func runRepoShortcut(server *httptest.Server, name string, args map[string]string) error {
	for _, shortcut := range Shortcuts() {
		if shortcut.Name != name {
			continue
		}
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
	return fmt.Errorf("shortcut %q not found", name)
}
