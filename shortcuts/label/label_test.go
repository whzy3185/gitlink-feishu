package label

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gitlink-org/gitlink-cli/internal/client"
	"github.com/gitlink-org/gitlink-cli/shortcuts/common"
)

func TestLabelList(t *testing.T) {
	server := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		assertRequest(t, r, "GET", "/owner/repo/labels.json")
		writeJSON(t, w, []interface{}{})
	})
	defer server.Close()

	if err := runShortcut(t, server, "list", nil); err != nil {
		t.Fatalf("list failed: %v", err)
	}
}

func TestLabelCreate(t *testing.T) {
	var payload map[string]interface{}
	server := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		assertRequest(t, r, "POST", "/owner/repo/labels.json")
		payload = decodeJSON(t, r)
		writeJSON(t, w, map[string]interface{}{"id": 1})
	})
	defer server.Close()

	if err := runShortcut(t, server, "create", map[string]string{"name": "bug", "color": "#ff0000"}); err != nil {
		t.Fatalf("create failed: %v", err)
	}
	assertEqual(t, payload["name"], "bug")
	assertEqual(t, payload["color"], "#ff0000")
}

func TestLabelUpdate(t *testing.T) {
	var payload map[string]interface{}
	server := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		assertRequest(t, r, "PATCH", "/owner/repo/labels/3.json")
		payload = decodeJSON(t, r)
		writeJSON(t, w, map[string]interface{}{"id": 3})
	})
	defer server.Close()

	if err := runShortcut(t, server, "update", map[string]string{"id": "3", "name": "enhancement", "color": "#00ff00"}); err != nil {
		t.Fatalf("update failed: %v", err)
	}
	assertEqual(t, payload["name"], "enhancement")
	assertEqual(t, payload["color"], "#00ff00")
}

func TestLabelDelete(t *testing.T) {
	server := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		assertRequest(t, r, "DELETE", "/owner/repo/labels/3.json")
		writeJSON(t, w, map[string]interface{}{"status": 0})
	})
	defer server.Close()

	if err := runShortcut(t, server, "delete", map[string]string{"id": "3"}); err != nil {
		t.Fatalf("delete failed: %v", err)
	}
}

func runShortcut(t *testing.T, server *httptest.Server, name string, args map[string]string) error {
	t.Helper()
	var shortcut *common.Shortcut
	for _, s := range Shortcuts() {
		if s.Name == name {
			shortcut = s
			break
		}
	}
	if shortcut == nil {
		t.Fatalf("shortcut %q not found", name)
	}
	if args == nil {
		args = map[string]string{}
	}
	ctx := &common.RuntimeContext{
		Client: &client.Client{HTTP: server.Client(), BaseURL: server.URL},
		Owner:  "owner",
		Repo:   "repo",
		Format: "json",
		Args:   args,
	}
	return shortcut.Run(ctx)
}

func newTestServer(t *testing.T, handler http.HandlerFunc) *httptest.Server {
	t.Helper()
	return httptest.NewServer(handler)
}

func assertRequest(t *testing.T, r *http.Request, method, path string) {
	t.Helper()
	if r.Method != method || r.URL.Path != path {
		t.Fatalf("got %s %s, want %s %s", r.Method, r.URL.Path, method, path)
	}
}

func decodeJSON(t *testing.T, r *http.Request) map[string]interface{} {
	t.Helper()
	var payload map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	return payload
}

func writeJSON(t *testing.T, w http.ResponseWriter, v interface{}) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(v); err != nil {
		t.Fatalf("write response: %v", err)
	}
}

func assertEqual(t *testing.T, got, want interface{}) {
	t.Helper()
	if got != want {
		t.Fatalf("got %v, want %v", got, want)
	}
}
