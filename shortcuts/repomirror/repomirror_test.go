package repomirror

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gitlink-org/gitlink-cli/internal/client"
	"github.com/gitlink-org/gitlink-cli/shortcuts/common"
)

func TestMirrorSync(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" || r.URL.Path != "/repositories/42/sync_mirror.json" {
			t.Fatalf("got %s %s", r.Method, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"status": 0})
	}))
	defer server.Close()
	if err := runShortcut(server, map[string]string{"id": "42"}); err != nil {
		t.Fatalf("sync failed: %v", err)
	}
}

func TestMirrorSyncDryRun(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatalf("dry-run should not call API")
	}))
	defer server.Close()
	if err := runShortcut(server, map[string]string{"id": "42", "dry-run": "true"}); err != nil {
		t.Fatalf("dry-run failed: %v", err)
	}
}

func TestMirrorSyncRejectsInvalidID(t *testing.T) {
	if err := runShortcut(httptest.NewServer(http.NewServeMux()), map[string]string{"id": "abc"}); err == nil {
		t.Fatal("expected invalid id error")
	}
}

func runShortcut(server *httptest.Server, args map[string]string) error {
	return Shortcuts()[0].Run(&common.RuntimeContext{Client: &client.Client{HTTP: server.Client(), BaseURL: server.URL}, Format: "json", Args: args})
}
