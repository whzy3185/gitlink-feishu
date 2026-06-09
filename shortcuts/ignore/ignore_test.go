package ignore

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gitlink-org/gitlink-cli/internal/client"
	"github.com/gitlink-org/gitlink-cli/shortcuts/common"
)

func runShortcut(t *testing.T, server *httptest.Server, name string, args map[string]string) error {
	t.Helper()
	shortcut := findShortcut(t, name)
	ctx := &common.RuntimeContext{
		Client: &client.Client{HTTP: server.Client(), BaseURL: server.URL},
		Owner:  "owner",
		Repo:   "repo",
		Format: "json",
		Args:   args,
	}
	if ctx.Args == nil {
		ctx.Args = map[string]string{}
	}
	return shortcut.Run(ctx)
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

func writeJSON(t *testing.T, w http.ResponseWriter, payload interface{}) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		t.Fatalf("failed to write response: %v", err)
	}
}

func TestIgnoreList(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Fatalf("got method %s, want GET", r.Method)
		}
		if r.URL.Path != "/ignores.json" {
			t.Fatalf("got path %s, want /ignores.json", r.URL.Path)
		}
		if got := r.URL.Query().Get("name"); got != "" {
			t.Fatalf("expected no name filter, got %q", got)
		}
		writeJSON(t, w, map[string]interface{}{
			"ignores": []interface{}{
				map[string]interface{}{"id": 1, "name": "Go"},
				map[string]interface{}{"id": 2, "name": "Ada"},
			},
		})
	}))
	defer server.Close()

	if err := runShortcut(t, server, "list", map[string]string{}); err != nil {
		t.Fatalf("ignore list failed: %v", err)
	}
}

func TestIgnoreListWithNameFilter(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Fatalf("got method %s, want GET", r.Method)
		}
		if r.URL.Path != "/ignores.json" {
			t.Fatalf("got path %s, want /ignores.json", r.URL.Path)
		}
		if got := r.URL.Query().Get("name"); got != "Ada" {
			t.Fatalf("expected name=Ada, got %q", got)
		}
		writeJSON(t, w, map[string]interface{}{
			"ignores": []interface{}{
				map[string]interface{}{"id": 2, "name": "Ada"},
			},
		})
	}))
	defer server.Close()

	if err := runShortcut(t, server, "list", map[string]string{"name": "Ada"}); err != nil {
		t.Fatalf("ignore list with name filter failed: %v", err)
	}
}

func TestIgnoreListHTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("server error"))
	}))
	defer server.Close()

	err := runShortcut(t, server, "list", map[string]string{})
	if err == nil {
		t.Fatal("expected error for HTTP 500")
	}
}

func TestIgnoreListEmptyResult(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/ignores.json" {
			t.Fatalf("got path %s, want /ignores.json", r.URL.Path)
		}
		writeJSON(t, w, map[string]interface{}{
			"ignores": []interface{}{},
		})
	}))
	defer server.Close()

	if err := runShortcut(t, server, "list", map[string]string{"name": "NONEXISTENT"}); err != nil {
		t.Fatalf("ignore list with empty result failed: %v", err)
	}
}
