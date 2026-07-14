package catalog

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gitlink-org/gitlink-cli/internal/client"
	"github.com/gitlink-org/gitlink-cli/shortcuts/common"
)

func TestCatalogLicenses(t *testing.T) {
	server := newCatalogTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		assertRequest(t, r, "GET", "/licenses.json")
		if got := r.URL.Query().Get("name"); got != "MIT" {
			t.Fatalf("name query = %q, want MIT", got)
		}
		writeJSON(t, w, map[string]interface{}{
			"licenses": []interface{}{
				map[string]interface{}{"id": 1, "name": "MIT"},
			},
		})
	})
	defer server.Close()

	if err := runCatalogShortcut(t, server, "licenses", map[string]string{"name": "MIT"}); err != nil {
		t.Fatalf("licenses shortcut failed: %v", err)
	}
}

func TestCatalogIgnores(t *testing.T) {
	server := newCatalogTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		assertRequest(t, r, "GET", "/ignores.json")
		if got := r.URL.Query().Get("name"); got != "Go" {
			t.Fatalf("name query = %q, want Go", got)
		}
		writeJSON(t, w, map[string]interface{}{
			"ignores": []interface{}{
				map[string]interface{}{"id": 2, "name": "Go"},
			},
		})
	})
	defer server.Close()

	if err := runCatalogShortcut(t, server, "ignores", map[string]string{"name": "Go"}); err != nil {
		t.Fatalf("ignores shortcut failed: %v", err)
	}
}

func TestCatalogListWithoutFilter(t *testing.T) {
	server := newCatalogTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		assertRequest(t, r, "GET", "/licenses.json")
		if got := r.URL.RawQuery; got != "" {
			t.Fatalf("raw query = %q, want empty", got)
		}
		writeJSON(t, w, map[string]interface{}{"licenses": []interface{}{}})
	})
	defer server.Close()

	if err := runCatalogShortcut(t, server, "licenses", nil); err != nil {
		t.Fatalf("licenses shortcut failed: %v", err)
	}
}

func runCatalogShortcut(t *testing.T, server *httptest.Server, name string, args map[string]string) error {
	t.Helper()
	shortcut := findCatalogShortcut(t, name)
	ctx := &common.RuntimeContext{
		Client: &client.Client{
			HTTP:    server.Client(),
			BaseURL: server.URL,
		},
		Format: "json",
		Args:   args,
	}
	if ctx.Args == nil {
		ctx.Args = map[string]string{}
	}
	return shortcut.Run(ctx)
}

func findCatalogShortcut(t *testing.T, name string) *common.Shortcut {
	t.Helper()
	for _, shortcut := range Shortcuts() {
		if shortcut.Name == name {
			return shortcut
		}
	}
	t.Fatalf("shortcut %q not found", name)
	return nil
}

func newCatalogTestServer(t *testing.T, handler http.HandlerFunc) *httptest.Server {
	t.Helper()
	return httptest.NewServer(handler)
}

func assertRequest(t *testing.T, r *http.Request, method, path string) {
	t.Helper()
	if r.Method != method || r.URL.Path != path {
		t.Fatalf("got request %s %s, want %s %s", r.Method, r.URL.Path, method, path)
	}
}

func writeJSON(t *testing.T, w http.ResponseWriter, payload interface{}) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		t.Fatalf("failed to write response: %v", err)
	}
}
