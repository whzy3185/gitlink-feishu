package compare

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gitlink-org/gitlink-cli/internal/client"
	"github.com/gitlink-org/gitlink-cli/shortcuts/common"
)

func TestCompareViewEncodesRefs(t *testing.T) {
	var calledPath string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calledPath = r.URL.Path
		if r.Method != "GET" || r.URL.Path != "/owner/repo/compare/ZmVhdHVyZS9hcGk...bWFzdGVy.json" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		writeJSON(t, w, map[string]interface{}{"commits_count": 1})
	}))
	defer server.Close()

	err := runCompareShortcut(t, server, "view", map[string]string{
		"head": "feature/api",
		"base": "master",
	})
	if err != nil {
		t.Fatalf("view shortcut failed: %v", err)
	}
	if calledPath == "" {
		t.Fatal("server was not called")
	}
}

func TestCompareFilesUsesV1EndpointWithFilters(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" || r.URL.Path != "/v1/owner/repo/compare/YnVnZml4...cmVsZWFzZS92MQ/files.json" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		if got := r.URL.Query().Get("filepath"); got != "cmd/api/api.go" {
			t.Fatalf("filepath query = %q, want cmd/api/api.go", got)
		}
		if got := r.URL.Query().Get("page"); got != "2" {
			t.Fatalf("page query = %q, want 2", got)
		}
		if got := r.URL.Query().Get("limit"); got != "50" {
			t.Fatalf("limit query = %q, want 50", got)
		}
		writeJSON(t, w, map[string]interface{}{"files": []interface{}{}})
	}))
	defer server.Close()

	err := runCompareShortcut(t, server, "files", map[string]string{
		"head":  "bugfix",
		"base":  "release/v1",
		"file":  "cmd/api/api.go",
		"page":  "2",
		"limit": "50",
	})
	if err != nil {
		t.Fatalf("files shortcut failed: %v", err)
	}
}

func TestEncodeRefUsesRawURLBase64(t *testing.T) {
	got := encodeRef("feature/api")
	want := "ZmVhdHVyZS9hcGk"
	if got != want {
		t.Fatalf("encodeRef() = %q, want %q", got, want)
	}
}

func runCompareShortcut(t *testing.T, server *httptest.Server, name string, args map[string]string) error {
	t.Helper()
	shortcut := findCompareShortcut(t, name)
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

func findCompareShortcut(t *testing.T, name string) *common.Shortcut {
	t.Helper()
	for _, shortcut := range Shortcuts() {
		if shortcut.Name == name {
			return shortcut
		}
	}
	t.Fatalf("shortcut %q not found", name)
	return nil
}

func writeJSON(t *testing.T, w http.ResponseWriter, payload interface{}) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		t.Fatalf("failed to write response: %v", err)
	}
}
