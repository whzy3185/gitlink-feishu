package tag

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gitlink-org/gitlink-cli/internal/client"
	"github.com/gitlink-org/gitlink-cli/shortcuts/common"
)

func TestTagListUsesV1Pagination(t *testing.T) {
	server := newTagTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		assertRequest(t, r, "GET", "/v1/owner/repo/tags.json")
		assertEqual(t, r.URL.Query().Get("page"), "2")
		assertEqual(t, r.URL.Query().Get("limit"), "50")
		writeJSON(t, w, map[string]interface{}{"total_count": 1, "tags": []interface{}{}})
	})
	defer server.Close()

	err := runTagShortcut(t, server, "list", map[string]string{
		"page":  "2",
		"limit": "50",
	})
	if err != nil {
		t.Fatalf("list shortcut failed: %v", err)
	}
}

func TestTagListUsesDefaultPagination(t *testing.T) {
	server := newTagTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		assertRequest(t, r, "GET", "/v1/owner/repo/tags.json")
		assertEqual(t, r.URL.Query().Get("page"), "1")
		assertEqual(t, r.URL.Query().Get("limit"), "20")
		writeJSON(t, w, map[string]interface{}{"total_count": 0, "tags": []interface{}{}})
	})
	defer server.Close()

	if err := runTagShortcut(t, server, "list", nil); err != nil {
		t.Fatalf("list shortcut failed: %v", err)
	}
}

func TestTagNamesUsesNameSearch(t *testing.T) {
	server := newTagTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		assertRequest(t, r, "GET", "/owner/repo/tags.json")
		assertEqual(t, r.URL.Query().Get("only_name"), "true")
		assertEqual(t, r.URL.Query().Get("name"), "v1")
		writeJSON(t, w, map[string]interface{}{
			"total_count": 1,
			"tags":        []interface{}{map[string]interface{}{"name": "v1.0"}},
		})
	})
	defer server.Close()

	if err := runTagShortcut(t, server, "names", map[string]string{"keyword": " v1 "}); err != nil {
		t.Fatalf("names shortcut failed: %v", err)
	}
}

func TestTagViewEscapesTagName(t *testing.T) {
	server := newTagTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		assertRequest(t, r, "GET", "/v1/owner/repo/tags/release%2Fv1.0.json")
		writeJSON(t, w, map[string]interface{}{"name": "release/v1.0"})
	})
	defer server.Close()

	if err := runTagShortcut(t, server, "view", map[string]string{"name": "release/v1.0"}); err != nil {
		t.Fatalf("view shortcut failed: %v", err)
	}
}

func TestTagDeleteEscapesTagName(t *testing.T) {
	server := newTagTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		assertRequest(t, r, "DELETE", "/v1/owner/repo/tags/release%2Fv1.0.json")
		writeJSON(t, w, map[string]interface{}{"status": 0, "message": "success"})
	})
	defer server.Close()

	if err := runTagShortcut(t, server, "delete", map[string]string{"name": "release/v1.0"}); err != nil {
		t.Fatalf("delete shortcut failed: %v", err)
	}
}

func TestTagValidation(t *testing.T) {
	server := newTagTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		t.Fatalf("invalid input should not call API, got: %s %s", r.Method, r.URL.Path)
	})
	defer server.Close()

	cases := []struct {
		name     string
		shortcut string
		args     map[string]string
	}{
		{
			name:     "invalid page",
			shortcut: "list",
			args:     map[string]string{"page": "0", "limit": "20"},
		},
		{
			name:     "invalid limit",
			shortcut: "list",
			args:     map[string]string{"page": "1", "limit": "many"},
		},
		{
			name:     "empty tag name",
			shortcut: "view",
			args:     map[string]string{"name": "  "},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if err := runTagShortcut(t, server, tc.shortcut, tc.args); err == nil {
				t.Fatal("expected validation error")
			}
		})
	}
}

func runTagShortcut(t *testing.T, server *httptest.Server, name string, args map[string]string) error {
	t.Helper()
	shortcut := findTagShortcut(t, name)
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
	return shortcut.Run(ctx)
}

func findTagShortcut(t *testing.T, name string) *common.Shortcut {
	t.Helper()
	for _, shortcut := range Shortcuts() {
		if shortcut.Name == name {
			return shortcut
		}
	}
	t.Fatalf("shortcut %q not found", name)
	return nil
}

func newTagTestServer(t *testing.T, handler http.HandlerFunc) *httptest.Server {
	t.Helper()
	return httptest.NewServer(handler)
}

func assertRequest(t *testing.T, r *http.Request, method, path string) {
	t.Helper()
	if r.Method != method || r.URL.EscapedPath() != path {
		t.Fatalf("unexpected request: %s %s", r.Method, r.URL.EscapedPath())
	}
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
