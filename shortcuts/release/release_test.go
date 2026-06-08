package release

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/gitlink-org/gitlink-cli/internal/client"
	"github.com/gitlink-org/gitlink-cli/shortcuts/common"
)

func TestReleaseList(t *testing.T) {
	server := newReleaseTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		assertReleaseRequest(t, r, "GET", "/owner/repo/releases.json")
		writeReleaseJSON(t, w, map[string]interface{}{"releases": []map[string]interface{}{{"tag_name": "v1.0"}}})
	})
	defer server.Close()

	if err := runReleaseShortcut(t, server, "list", map[string]string{"page": "1", "limit": "20"}); err != nil {
		t.Fatalf("list failed: %v", err)
	}
}

func TestReleaseCreatePayload(t *testing.T) {
	var payload map[string]interface{}
	server := newReleaseTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		assertReleaseRequest(t, r, "POST", "/owner/repo/releases.json")
		payload = decodeReleaseJSON(t, r)
		writeReleaseJSON(t, w, map[string]interface{}{"status": 0, "message": "created"})
	})
	defer server.Close()

	err := runReleaseShortcut(t, server, "create", map[string]string{
		"tag":            "v1.0.0",
		"name":           "v1.0.0",
		"body":           "Release notes",
		"target":         "main",
		"draft":          "true",
		"prerelease":     "true",
		"attachment-ids": "12,34,12",
	})
	if err != nil {
		t.Fatalf("create shortcut failed: %v", err)
	}

	assertReleaseEqual(t, payload["tag_name"], "v1.0.0")
	assertReleaseEqual(t, payload["name"], "v1.0.0")
	assertReleaseEqual(t, payload["body"], "Release notes")
	assertReleaseEqual(t, payload["target_commitish"], "main")
	assertReleaseEqual(t, payload["draft"], true)
	assertReleaseEqual(t, payload["prerelease"], true)
	assertReleaseStringSlice(t, payload["attachment_ids"], []string{"12", "34"})
}

func TestReleaseCreateWithBody(t *testing.T) {
	server := newReleaseTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		assertReleaseRequest(t, r, "POST", "/owner/repo/releases.json")
		writeReleaseJSON(t, w, map[string]interface{}{"tag_name": "v2.1"})
	})
	defer server.Close()

	err := runReleaseShortcut(t, server, "create", map[string]string{
		"tag":        "v2.1",
		"name":       "Version 2.1",
		"body":       "Release notes here",
		"target":     "develop",
		"prerelease": "true",
	})
	if err != nil {
		t.Fatalf("create with body failed: %v", err)
	}
}

func TestReleaseEdit(t *testing.T) {
	server := newReleaseTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		assertReleaseRequest(t, r, "GET", "/owner/repo/releases/7/edit.json")
		writeReleaseJSON(t, w, releaseEditFixture())
	})
	defer server.Close()

	if err := runReleaseShortcut(t, server, "edit", map[string]string{"id": "7"}); err != nil {
		t.Fatalf("edit shortcut failed: %v", err)
	}
}

func TestReleaseView(t *testing.T) {
	server := newReleaseTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		assertReleaseRequest(t, r, "GET", "/owner/repo/releases/v1.0.json")
		writeReleaseJSON(t, w, map[string]interface{}{"tag_name": "v1.0", "name": "Version 1.0"})
	})
	defer server.Close()

	if err := runReleaseShortcut(t, server, "view", map[string]string{"id": "v1.0"}); err != nil {
		t.Fatalf("view failed: %v", err)
	}
}

func TestReleaseUpdatePreservesExistingFields(t *testing.T) {
	var payload map[string]interface{}
	server := newReleaseTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == "GET" && r.URL.Path == "/owner/repo/releases/7/edit.json":
			writeReleaseJSON(t, w, releaseEditFixture())
		case r.Method == "PUT" && r.URL.Path == "/owner/repo/releases/7.json":
			payload = decodeReleaseJSON(t, r)
			writeReleaseJSON(t, w, map[string]interface{}{"status": 0, "message": "updated"})
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	})
	defer server.Close()

	err := runReleaseShortcut(t, server, "update", map[string]string{
		"id":         "7",
		"body":       "Updated notes",
		"prerelease": "true",
	})
	if err != nil {
		t.Fatalf("update shortcut failed: %v", err)
	}

	assertReleaseEqual(t, payload["tag_name"], "v1.0.0")
	assertReleaseEqual(t, payload["name"], "Old release")
	assertReleaseEqual(t, payload["body"], "Updated notes")
	assertReleaseEqual(t, payload["target_commitish"], "master")
	assertReleaseEqual(t, payload["draft"], false)
	assertReleaseEqual(t, payload["prerelease"], true)
	assertReleaseStringSlice(t, payload["attachment_ids"], []string{"12", "34"})
}

func TestReleaseUpdateOverridesAttachmentIDs(t *testing.T) {
	var payload map[string]interface{}
	server := newReleaseTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == "GET" && r.URL.Path == "/owner/repo/releases/7/edit.json":
			writeReleaseJSON(t, w, releaseEditFixture())
		case r.Method == "PUT" && r.URL.Path == "/owner/repo/releases/7.json":
			payload = decodeReleaseJSON(t, r)
			writeReleaseJSON(t, w, map[string]interface{}{"status": 0, "message": "updated"})
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	})
	defer server.Close()

	err := runReleaseShortcut(t, server, "update", map[string]string{
		"id":             "7",
		"name":           "New release",
		"attachment-ids": "99,100",
	})
	if err != nil {
		t.Fatalf("update shortcut failed: %v", err)
	}

	assertReleaseEqual(t, payload["name"], "New release")
	assertReleaseStringSlice(t, payload["attachment_ids"], []string{"99", "100"})
}

func TestReleaseUpdateDryRunDoesNotWrite(t *testing.T) {
	server := newReleaseTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "PUT" {
			t.Fatalf("dry-run should not update release, got %s %s", r.Method, r.URL.Path)
		}
		assertReleaseRequest(t, r, "GET", "/owner/repo/releases/7/edit.json")
		writeReleaseJSON(t, w, releaseEditFixture())
	})
	defer server.Close()

	err := runReleaseShortcut(t, server, "update", map[string]string{
		"id":      "7",
		"body":    "Preview notes",
		"dry-run": "true",
	})
	if err != nil {
		t.Fatalf("update dry-run failed: %v", err)
	}
}

func TestReleaseDeleteSuccess(t *testing.T) {
	server := newReleaseTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		assertReleaseRequest(t, r, "DELETE", "/owner/repo/releases/1.json")
		writeReleaseJSON(t, w, map[string]interface{}{"message": "deleted"})
	})
	defer server.Close()

	if err := runReleaseShortcut(t, server, "delete", map[string]string{"id": "1"}); err != nil {
		t.Fatalf("delete failed: %v", err)
	}
}

func TestReleaseDeleteDryRunDoesNotCallAPI(t *testing.T) {
	server := newReleaseTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		t.Fatalf("delete dry-run should not call API, got %s %s", r.Method, r.URL.Path)
	})
	defer server.Close()

	err := runReleaseShortcut(t, server, "delete", map[string]string{
		"id":      "7",
		"dry-run": "true",
	})
	if err != nil {
		t.Fatalf("delete dry-run failed: %v", err)
	}
}

func TestReleaseDeleteBugWorkaround(t *testing.T) {
	server := newReleaseTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "DELETE":
			w.WriteHeader(http.StatusInternalServerError)
			writeReleaseJSON(t, w, map[string]interface{}{"status": float64(500), "message": "server error"})
		case "GET":
			w.WriteHeader(http.StatusNotFound)
			writeReleaseJSON(t, w, map[string]interface{}{"status": float64(404), "message": "not found"})
		default:
			t.Fatalf("unexpected method: %s", r.Method)
		}
	})
	defer server.Close()

	if err := runReleaseShortcut(t, server, "delete", map[string]string{"id": "1"}); err != nil {
		t.Fatalf("delete bug workaround failed: %v", err)
	}
}

func TestReleaseDeleteTrulyFails(t *testing.T) {
	server := newReleaseTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "DELETE":
			w.WriteHeader(http.StatusInternalServerError)
			writeReleaseJSON(t, w, map[string]interface{}{"status": float64(500), "message": "server error"})
		case "GET":
			writeReleaseJSON(t, w, map[string]interface{}{"id": float64(1), "tag_name": "v1.0"})
		default:
			t.Fatalf("unexpected method: %s", r.Method)
		}
	})
	defer server.Close()

	if err := runReleaseShortcut(t, server, "delete", map[string]string{"id": "1"}); err == nil {
		t.Fatal("expected error when delete truly fails")
	}
}

func TestReleaseListHTTPError(t *testing.T) {
	server := newReleaseTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("server error"))
	})
	defer server.Close()

	if err := runReleaseShortcut(t, server, "list", map[string]string{"page": "1", "limit": "20"}); err == nil {
		t.Fatal("expected error for HTTP 500")
	}
}

func TestReleaseCreateHTTPError(t *testing.T) {
	server := newReleaseTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("server error"))
	})
	defer server.Close()

	if err := runReleaseShortcut(t, server, "create", map[string]string{"tag": "v1.0", "name": "v1.0"}); err == nil {
		t.Fatal("expected error for HTTP 500")
	}
}

func TestReleaseViewHTTPError(t *testing.T) {
	server := newReleaseTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("server error"))
	})
	defer server.Close()

	if err := runReleaseShortcut(t, server, "view", map[string]string{"id": "v1.0"}); err == nil {
		t.Fatal("expected error for HTTP 500")
	}
}

func TestReleaseRejectsInvalidCreateBool(t *testing.T) {
	server := newReleaseTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		t.Fatalf("invalid bool should not call API, got %s %s", r.Method, r.URL.Path)
	})
	defer server.Close()

	err := runReleaseShortcut(t, server, "create", map[string]string{
		"tag":        "v1.0.0",
		"name":       "v1.0.0",
		"prerelease": "maybe",
	})
	if err == nil {
		t.Fatal("expected invalid prerelease value to return an error")
	}
}

func TestReleaseUpdateRejectsNoFields(t *testing.T) {
	server := newReleaseTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		t.Fatalf("empty update should not call API, got %s %s", r.Method, r.URL.Path)
	})
	defer server.Close()

	err := runReleaseShortcut(t, server, "update", map[string]string{"id": "7"})
	if err == nil {
		t.Fatal("expected empty update to return an error")
	}
}

func TestReleaseUpdateRejectsInvalidBoolBeforeFetch(t *testing.T) {
	server := newReleaseTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		t.Fatalf("invalid update bool should not call API, got %s %s", r.Method, r.URL.Path)
	})
	defer server.Close()

	err := runReleaseShortcut(t, server, "update", map[string]string{
		"id":    "7",
		"draft": "maybe",
	})
	if err == nil {
		t.Fatal("expected invalid draft value to return an error")
	}
}

func TestReleaseShortcutNames(t *testing.T) {
	got := map[string]bool{}
	for _, shortcut := range Shortcuts() {
		got[shortcut.Name] = true
	}
	want := []string{"list", "create", "edit", "view", "update", "delete"}
	for _, name := range want {
		if !got[name] {
			t.Fatalf("missing shortcut %q in %v", name, got)
		}
	}
	if len(got) != len(want) {
		t.Fatalf("shortcut count = %d, want %d: %v", len(got), len(want), got)
	}
}

func runReleaseShortcut(t *testing.T, server *httptest.Server, name string, args map[string]string) error {
	t.Helper()
	shortcut := findReleaseShortcut(t, name)
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

func findReleaseShortcut(t *testing.T, name string) *common.Shortcut {
	t.Helper()
	for _, shortcut := range Shortcuts() {
		if shortcut.Name == name {
			return shortcut
		}
	}
	t.Fatalf("shortcut %q not found", name)
	return nil
}

func newReleaseTestServer(t *testing.T, handler http.HandlerFunc) *httptest.Server {
	t.Helper()
	return httptest.NewServer(handler)
}

func releaseEditFixture() map[string]interface{} {
	return map[string]interface{}{
		"id":               7,
		"name":             "Old release",
		"body":             "Old notes",
		"tag_name":         "v1.0.0",
		"target_commitish": "master",
		"draft":            false,
		"prerelease":       false,
		"attachments": []map[string]interface{}{
			{"id": 12, "title": "a.zip"},
			{"id": "34", "title": "b.zip"},
		},
	}
}

func assertReleaseRequest(t *testing.T, r *http.Request, method, path string) {
	t.Helper()
	if r.Method != method || r.URL.Path != path {
		t.Fatalf("got request %s %s, want %s %s", r.Method, r.URL.Path, method, path)
	}
}

func decodeReleaseJSON(t *testing.T, r *http.Request) map[string]interface{} {
	t.Helper()
	var payload map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		t.Fatalf("failed to decode request body: %v", err)
	}
	return payload
}

func writeReleaseJSON(t *testing.T, w http.ResponseWriter, payload interface{}) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		t.Fatalf("failed to write response: %v", err)
	}
}

func assertReleaseEqual(t *testing.T, got interface{}, want interface{}) {
	t.Helper()
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v (%T), want %v (%T)", got, got, want, want)
	}
}

func assertReleaseStringSlice(t *testing.T, got interface{}, want []string) {
	t.Helper()
	values, ok := got.([]interface{})
	if !ok {
		t.Fatalf("got %T, want []interface{}", got)
	}
	result := make([]string, 0, len(values))
	for _, value := range values {
		text, ok := value.(string)
		if !ok {
			t.Fatalf("got value %v (%T), want string", value, value)
		}
		result = append(result, text)
	}
	if !reflect.DeepEqual(result, want) {
		t.Fatalf("got %v, want %v", result, want)
	}
}

func ExampleShortcuts() {
	for _, shortcut := range Shortcuts() {
		fmt.Println(shortcut.Name)
	}
	// Output:
	// list
	// create
	// edit
	// view
	// update
	// delete
}
