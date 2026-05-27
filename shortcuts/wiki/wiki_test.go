package wiki

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gitlink-org/gitlink-cli/internal/client"
	"github.com/gitlink-org/gitlink-cli/internal/i18n"
	"github.com/gitlink-org/gitlink-cli/shortcuts/common"
)

func TestWikiShortcutsRouteToOpenAPIEndpoints(t *testing.T) {
	cases := []struct {
		name    string
		args    map[string]string
		method  string
		path    string
		query   map[string]string
		payload map[string]interface{}
	}{
		{
			name:   "pages",
			args:   map[string]string{"project-id": "123"},
			method: "GET",
			path:   "/wiki/wikiPages.json",
			query:  map[string]string{"owner": "owner", "repo": "repo", "projectId": "123"},
		},
		{
			name:   "view",
			args:   map[string]string{"project-id": "123", "page": "Home"},
			method: "GET",
			path:   "/wiki/getWiki.json",
			query:  map[string]string{"owner": "owner", "repo": "repo", "projectId": "123", "pageName": "Home"},
		},
		{
			name:   "create",
			args:   map[string]string{"project-id": "123", "page": "Home", "title": "Home", "message": "Add Home", "content": "hello"},
			method: "POST",
			path:   "/wiki/createWiki.json",
			payload: map[string]interface{}{
				"owner":          "owner",
				"repo":           "repo",
				"projectId":      float64(123),
				"pageName":       "Home",
				"title":          "Home",
				"message":        "Add Home",
				"content_base64": "aGVsbG8=",
			},
		},
		{
			name:   "update",
			args:   map[string]string{"project-id": "123", "page": "Home", "title": "Home", "content-base64": "dXBkYXRlZA=="},
			method: "PUT",
			path:   "/wiki/updateWiki.json",
			payload: map[string]interface{}{
				"owner":          "owner",
				"repo":           "repo",
				"projectId":      float64(123),
				"pageName":       "Home",
				"title":          "Home",
				"content_base64": "dXBkYXRlZA==",
			},
		},
		{
			name:   "delete",
			args:   map[string]string{"project-id": "123", "page": "Home"},
			method: "DELETE",
			path:   "/wiki/deleteWiki.json",
			payload: map[string]interface{}{
				"owner":     "owner",
				"repo":      "repo",
				"projectId": float64(123),
				"pageName":  "Home",
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			called := false
			server := newWikiTestServer(t, func(w http.ResponseWriter, r *http.Request) {
				if r.Method != tc.method || r.URL.Path != tc.path {
					t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
				}
				assertWikiQuery(t, r, tc.query)
				if tc.payload != nil {
					assertWikiPayload(t, r, tc.payload)
				}
				called = true
				writeWikiJSON(t, w, map[string]interface{}{"message": "success", "data": map[string]interface{}{}})
			})
			defer server.Close()

			err := runWikiShortcut(server, tc.name, tc.args)
			if err != nil {
				t.Fatalf("%s shortcut failed: %v", tc.name, err)
			}
			if !called {
				t.Fatal("expected API request")
			}
		})
	}
}

func TestWikiDeleteDryRunDoesNotCallAPI(t *testing.T) {
	server := newWikiTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
	})
	defer server.Close()

	err := runWikiShortcut(server, "delete", map[string]string{
		"project-id": "123",
		"page":       "Home",
		"dry-run":    "true",
	})
	if err != nil {
		t.Fatalf("delete dry-run failed: %v", err)
	}
}

func TestWikiShortcutsValidateRequiredArgs(t *testing.T) {
	server := newWikiTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
	})
	defer server.Close()

	cases := []struct {
		name string
		args map[string]string
		want string
	}{
		{name: "pages", args: map[string]string{}, want: "--project-id"},
		{name: "pages", args: map[string]string{"project-id": "abc"}, want: "--project-id must be a positive integer"},
		{name: "view", args: map[string]string{"project-id": "123"}, want: "--page"},
		{name: "create", args: map[string]string{"project-id": "123", "page": "Home", "title": "Home"}, want: "--content"},
		{name: "create", args: map[string]string{"project-id": "123", "page": "Home", "title": "Home", "content": "x", "content-base64": "eA=="}, want: "--content cannot be used with --content-base64"},
		{name: "update", args: map[string]string{"project-id": "123", "page": "Home"}, want: "--title"},
		{name: "delete", args: map[string]string{"project-id": "123"}, want: "--page"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := runWikiShortcut(server, tc.name, tc.args)
			if err == nil {
				t.Fatal("expected validation error")
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error = %q, want it to mention %s", err.Error(), tc.want)
			}
		})
	}
}

func runWikiShortcut(server *httptest.Server, name string, args map[string]string) error {
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
			Tr:     i18n.Default(),
		}
		return shortcut.Run(ctx)
	}
	return fmt.Errorf("shortcut %q not found", name)
}

func newWikiTestServer(t *testing.T, handler http.HandlerFunc) *httptest.Server {
	t.Helper()
	return httptest.NewServer(handler)
}

func assertWikiQuery(t *testing.T, r *http.Request, want map[string]string) {
	t.Helper()
	query := r.URL.Query()
	if len(query) != len(want) {
		t.Fatalf("query = %v, want %v", query, want)
	}
	for key, value := range want {
		if got := query.Get(key); got != value {
			t.Fatalf("query %s = %q, want %q", key, got, value)
		}
	}
}

func assertWikiPayload(t *testing.T, r *http.Request, want map[string]interface{}) {
	t.Helper()
	var got map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
		t.Fatalf("decode request body: %v", err)
	}
	if len(got) != len(want) {
		t.Fatalf("payload = %v, want %v", got, want)
	}
	for key, value := range want {
		if got[key] != value {
			t.Fatalf("payload %s = %v, want %v", key, got[key], value)
		}
	}
}

func writeWikiJSON(t *testing.T, w http.ResponseWriter, payload interface{}) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		t.Fatalf("failed to write response: %v", err)
	}
}
