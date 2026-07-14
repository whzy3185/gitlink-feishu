package user

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

func TestUserMe(t *testing.T) {
	server := newUserTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		assertUserRequest(t, r, "GET", "/users/me.json")
		writeUserJSON(t, w, map[string]interface{}{
			"login": "currentuser",
			"name":  "Current User",
			"id":    float64(1),
		})
	})
	defer server.Close()

	if err := runUserShortcut(t, server, "me", nil); err != nil {
		t.Fatalf("me shortcut failed: %v", err)
	}
}

func TestUserCurrent(t *testing.T) {
	server := newUserTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		assertUserRequest(t, r, "GET", "/users/get_user_info.json")
		writeUserJSON(t, w, map[string]interface{}{"login": "alice"})
	})
	defer server.Close()

	if err := runUserShortcut(t, server, "current", nil); err != nil {
		t.Fatalf("current shortcut failed: %v", err)
	}
}

func TestUserInfo(t *testing.T) {
	server := newUserTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		assertUserRequest(t, r, "GET", "/users/alice.json")
		writeUserJSON(t, w, map[string]interface{}{
			"login": "alice",
			"name":  "Alice",
		})
	})
	defer server.Close()

	if err := runUserShortcut(t, server, "info", map[string]string{"login": "alice"}); err != nil {
		t.Fatalf("info shortcut failed: %v", err)
	}
}

func TestUserKeys(t *testing.T) {
	server := newUserTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		assertUserRequest(t, r, "GET", "/public_keys.json")
		assertUserQuery(t, r, "page", "2")
		assertUserQuery(t, r, "limit", "50")
		writeUserJSON(t, w, map[string]interface{}{"total_count": 0, "public_keys": []interface{}{}})
	})
	defer server.Close()

	err := runUserShortcut(t, server, "keys", map[string]string{
		"page":  "2",
		"limit": "50",
	})
	if err != nil {
		t.Fatalf("keys shortcut failed: %v", err)
	}
}

func TestUserKeyCreatePayload(t *testing.T) {
	var payload map[string]interface{}
	server := newUserTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		assertUserRequest(t, r, "POST", "/public_keys.json")
		payload = decodeUserJSON(t, r)
		writeUserJSON(t, w, map[string]interface{}{"id": 1})
	})
	defer server.Close()

	err := runUserShortcut(t, server, "key-create", map[string]string{
		"title": "work laptop",
		"key":   "ssh-rsa AAAA test@example.com",
	})
	if err != nil {
		t.Fatalf("key-create shortcut failed: %v", err)
	}

	assertUserEqual(t, payload["title"], "work laptop")
	assertUserEqual(t, payload["key"], "ssh-rsa AAAA test@example.com")
}

func TestUserKeyDelete(t *testing.T) {
	server := newUserTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		assertUserRequest(t, r, "DELETE", "/public_keys/7.json")
		writeUserJSON(t, w, map[string]interface{}{"status": 0, "message": "success"})
	})
	defer server.Close()

	if err := runUserShortcut(t, server, "key-delete", map[string]string{"id": "7"}); err != nil {
		t.Fatalf("key-delete shortcut failed: %v", err)
	}
}

func TestUserKeyDryRunDoesNotCallAPI(t *testing.T) {
	dryRunCases := []struct {
		name string
		args map[string]string
	}{
		{name: "key-create", args: map[string]string{"title": "work laptop", "key": "ssh-rsa AAAA", "dry-run": "true"}},
		{name: "key-delete", args: map[string]string{"id": "7", "dry-run": "true"}},
	}

	for _, tc := range dryRunCases {
		t.Run(tc.name, func(t *testing.T) {
			server := newUserTestServer(t, func(w http.ResponseWriter, r *http.Request) {
				t.Fatalf("dry-run should not call API, got %s %s", r.Method, r.URL.Path)
			})
			defer server.Close()

			if err := runUserShortcut(t, server, tc.name, tc.args); err != nil {
				t.Fatalf("%s dry-run failed: %v", tc.name, err)
			}
		})
	}
}

func TestUserStatsShortcuts(t *testing.T) {
	tests := []struct {
		name        string
		args        map[string]string
		path        string
		queryKey    string
		queryValue  string
		queryKey2   string
		queryValue2 string
	}{
		{name: "activity", args: map[string]string{"login": "alice"}, path: "/users/alice/statistics/activity.json"},
		{name: "headmap", args: map[string]string{"login": "alice", "year": "2026"}, path: "/users/alice/headmaps.json", queryKey: "year", queryValue: "2026"},
		{name: "develop", args: map[string]string{"login": "alice", "start-time": "100", "end-time": "200"}, path: "/users/alice/statistics/develop.json", queryKey: "start_time", queryValue: "100", queryKey2: "end_time", queryValue2: "200"},
		{name: "role", args: map[string]string{"login": "alice", "start-time": "100", "end-time": "200"}, path: "/users/alice/statistics/role.json", queryKey: "start_time", queryValue: "100", queryKey2: "end_time", queryValue2: "200"},
		{name: "major", args: map[string]string{"login": "alice", "start-time": "100", "end-time": "200"}, path: "/users/alice/statistics/major.json", queryKey: "start_time", queryValue: "100", queryKey2: "end_time", queryValue2: "200"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			server := newUserTestServer(t, func(w http.ResponseWriter, r *http.Request) {
				assertUserRequest(t, r, "GET", tc.path)
				if tc.queryKey != "" {
					assertUserQuery(t, r, tc.queryKey, tc.queryValue)
				}
				if tc.queryKey2 != "" {
					assertUserQuery(t, r, tc.queryKey2, tc.queryValue2)
				}
				writeUserJSON(t, w, map[string]interface{}{"status": 0})
			})
			defer server.Close()

			if err := runUserShortcut(t, server, tc.name, tc.args); err != nil {
				t.Fatalf("%s shortcut failed: %v", tc.name, err)
			}
		})
	}
}

func TestUserInfoMissingLogin(t *testing.T) {
	server := newUserTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("missing login should not call API")
	})
	defer server.Close()

	err := runUserShortcut(t, server, "info", map[string]string{})
	if err == nil {
		t.Fatal("expected error for missing login")
	}
}

func TestUserKeyCreateRejectsMissingKey(t *testing.T) {
	server := newUserTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		t.Fatalf("missing key should not call API, got %s %s", r.Method, r.URL.Path)
	})
	defer server.Close()

	err := runUserShortcut(t, server, "key-create", map[string]string{"title": "work laptop"})
	if err == nil {
		t.Fatal("expected missing key to return an error")
	}
}

func TestUserMeHTTPError(t *testing.T) {
	server := newUserTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		if _, err := w.Write([]byte("server error")); err != nil {
			t.Fatalf("write response: %v", err)
		}
	})
	defer server.Close()

	err := runUserShortcut(t, server, "me", nil)
	if err == nil {
		t.Fatal("expected error for HTTP 500")
	}
}

func TestUserInfoHTTPError(t *testing.T) {
	server := newUserTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		if _, err := w.Write([]byte("server error")); err != nil {
			t.Fatalf("write response: %v", err)
		}
	})
	defer server.Close()

	err := runUserShortcut(t, server, "info", map[string]string{"login": "alice"})
	if err == nil {
		t.Fatal("expected error for HTTP 500")
	}
}

func TestUserShortcutNames(t *testing.T) {
	got := map[string]bool{}
	for _, shortcut := range Shortcuts() {
		got[shortcut.Name] = true
	}
	want := []string{"me", "current", "info", "keys", "key-create", "key-delete", "activity", "headmap", "develop", "role", "major"}
	for _, name := range want {
		if !got[name] {
			t.Fatalf("missing shortcut %q in %v", name, got)
		}
	}
	if len(got) != len(want) {
		t.Fatalf("shortcut count = %d, want %d: %v", len(got), len(want), got)
	}
}

func runUserShortcut(t *testing.T, server *httptest.Server, name string, args map[string]string) error {
	t.Helper()
	shortcut := findUserShortcut(t, name)
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

func findUserShortcut(t *testing.T, name string) *common.Shortcut {
	t.Helper()
	for _, shortcut := range Shortcuts() {
		if shortcut.Name == name {
			return shortcut
		}
	}
	t.Fatalf("shortcut %q not found", name)
	return nil
}

func newUserTestServer(t *testing.T, handler http.HandlerFunc) *httptest.Server {
	t.Helper()
	return httptest.NewServer(handler)
}

func assertUserRequest(t *testing.T, r *http.Request, method, path string) {
	t.Helper()
	if r.Method != method || r.URL.Path != path {
		t.Fatalf("got request %s %s, want %s %s", r.Method, r.URL.Path, method, path)
	}
}

func assertUserQuery(t *testing.T, r *http.Request, key, want string) {
	t.Helper()
	if got := r.URL.Query().Get(key); got != want {
		t.Fatalf("query %s = %q, want %q", key, got, want)
	}
}

func decodeUserJSON(t *testing.T, r *http.Request) map[string]interface{} {
	t.Helper()
	var payload map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		t.Fatalf("failed to decode request body: %v", err)
	}
	return payload
}

func writeUserJSON(t *testing.T, w http.ResponseWriter, payload interface{}) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		t.Fatalf("failed to write response: %v", err)
	}
}

func assertUserEqual(t *testing.T, got interface{}, want interface{}) {
	t.Helper()
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v (%T), want %v (%T)", got, got, want, want)
	}
}

func ExampleShortcuts() {
	for _, shortcut := range Shortcuts() {
		fmt.Println(shortcut.Name)
	}
	// Output:
	// me
	// current
	// info
	// keys
	// key-create
	// key-delete
	// activity
	// headmap
	// develop
	// role
	// major
}
