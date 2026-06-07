package repo

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/gitlink-org/gitlink-cli/internal/client"
	"github.com/gitlink-org/gitlink-cli/shortcuts/common"
)

func runShortcut(t *testing.T, server *httptest.Server, name string, args map[string]string) error {
	t.Helper()
	s := findShortcut(t, name)
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
	return s.Run(ctx)
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

func writeJSON(t *testing.T, w http.ResponseWriter, v interface{}) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(v); err != nil {
		t.Fatalf("write response: %v", err)
	}
}

func writeText(t *testing.T, w http.ResponseWriter, code int, text string) {
	t.Helper()
	w.WriteHeader(code)
	if _, err := w.Write([]byte(text)); err != nil {
		t.Fatalf("write response: %v", err)
	}
}

// --- list ---

func TestRepoListDefault(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Fatalf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/projects.json" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		writeJSON(t, w, map[string]interface{}{
			"total_count": float64(2),
			"data":        []interface{}{map[string]interface{}{"name": "repo1"}},
		})
	}))
	defer server.Close()

	err := runShortcut(t, server, "list", map[string]string{"page": "1", "limit": "20"})
	if err != nil {
		t.Fatalf("list failed: %v", err)
	}
}

func TestRepoListForUser(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/users/alice/projects.json" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		writeJSON(t, w, map[string]interface{}{"total_count": float64(0), "data": []interface{}{}})
	}))
	defer server.Close()

	err := runShortcut(t, server, "list", map[string]string{"user": "alice", "page": "1", "limit": "20"})
	if err != nil {
		t.Fatalf("list for user failed: %v", err)
	}
}

func TestRepoListWithCategory(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("category") != "mirror" {
			t.Fatalf("expected category=mirror, got %s", r.URL.Query().Get("category"))
		}
		writeJSON(t, w, map[string]interface{}{"data": []interface{}{}})
	}))
	defer server.Close()

	err := runShortcut(t, server, "list", map[string]string{"category": "mirror", "page": "1", "limit": "20"})
	if err != nil {
		t.Fatalf("list with category failed: %v", err)
	}
}

// --- info/read metadata ---

func TestRepoInfo(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/owner/repo.json" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		writeJSON(t, w, map[string]interface{}{
			"name":        "repo",
			"description": "test repo",
		})
	}))
	defer server.Close()

	err := runShortcut(t, server, "info", nil)
	if err != nil {
		t.Fatalf("info failed: %v", err)
	}
}

func TestRepoMetadataShortcuts(t *testing.T) {
	tests := []struct {
		name string
		path string
	}{
		{name: "detail", path: "/owner/repo/detail.json"},
		{name: "simple", path: "/owner/repo/simple.json"},
		{name: "settings", path: "/owner/repo/edit.json"},
		{name: "units", path: "/owner/repo/project_units.json"},
		{name: "transfer-orgs", path: "/owner/repo/applied_transfer_projects/organizations.json"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assertRepoRequest(t, r, "GET", tc.path)
				writeJSON(t, w, map[string]interface{}{"status": 0})
			}))
			defer server.Close()

			if err := runShortcut(t, server, tc.name, nil); err != nil {
				t.Fatalf("%s shortcut failed: %v", tc.name, err)
			}
		})
	}
}

func TestRepoUnitsUpdatePayload(t *testing.T) {
	var payload map[string]interface{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertRepoRequest(t, r, "POST", "/owner/repo/project_units.json")
		payload = decodeRepoJSON(t, r)
		writeJSON(t, w, map[string]interface{}{"status": 0})
	}))
	defer server.Close()

	err := runShortcut(t, server, "units-update", map[string]string{"units": "code,issues,code,wiki"})
	if err != nil {
		t.Fatalf("units-update shortcut failed: %v", err)
	}
	assertRepoStringSlice(t, payload["unit_types"], []string{"code", "issues", "wiki"})
}

func TestRepoTopics(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertRepoRequest(t, r, "GET", "/v1/project_topics.json")
		assertRepoQuery(t, r, "keyword", "go")
		writeJSON(t, w, map[string]interface{}{"total_count": 0, "project_topics": []interface{}{}})
	}))
	defer server.Close()

	if err := runShortcut(t, server, "topics", map[string]string{"keyword": "go"}); err != nil {
		t.Fatalf("topics shortcut failed: %v", err)
	}
}

func TestRepoTopicAddPayload(t *testing.T) {
	var payload map[string]interface{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertRepoRequest(t, r, "POST", "/v1/project_topics.json")
		payload = decodeRepoJSON(t, r)
		writeJSON(t, w, map[string]interface{}{"status": 0})
	}))
	defer server.Close()

	err := runShortcut(t, server, "topic-add", map[string]string{
		"project-id": "17",
		"name":       "go",
	})
	if err != nil {
		t.Fatalf("topic-add shortcut failed: %v", err)
	}
	assertRepoEqual(t, payload["project_id"], float64(17))
	assertRepoEqual(t, payload["name"], "go")
}

func TestRepoTopicDelete(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertRepoRequest(t, r, "DELETE", "/v1/project_topics/8.json")
		assertRepoQuery(t, r, "project_id", "17")
		writeJSON(t, w, map[string]interface{}{"status": 0})
	}))
	defer server.Close()

	err := runShortcut(t, server, "topic-delete", map[string]string{
		"id":         "8",
		"project-id": "17",
	})
	if err != nil {
		t.Fatalf("topic-delete shortcut failed: %v", err)
	}
}

func TestRepoTransferPayload(t *testing.T) {
	var payload map[string]interface{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertRepoRequest(t, r, "POST", "/owner/repo/applied_transfer_projects.json")
		payload = decodeRepoJSON(t, r)
		writeJSON(t, w, map[string]interface{}{"id": 1})
	}))
	defer server.Close()

	if err := runShortcut(t, server, "transfer", map[string]string{"owner-name": "target-org"}); err != nil {
		t.Fatalf("transfer shortcut failed: %v", err)
	}
	assertRepoEqual(t, payload["owner_name"], "target-org")
}

func TestRepoTransferCancel(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertRepoRequest(t, r, "POST", "/owner/repo/applied_transfer_projects/cancel.json")
		writeJSON(t, w, map[string]interface{}{"status": 0})
	}))
	defer server.Close()

	if err := runShortcut(t, server, "transfer-cancel", nil); err != nil {
		t.Fatalf("transfer-cancel shortcut failed: %v", err)
	}
}

// --- fork/delete/create ---

func TestRepoFork(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Fatalf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/owner/repo/forks.json" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		writeJSON(t, w, map[string]interface{}{"message": "forked"})
	}))
	defer server.Close()

	err := runShortcut(t, server, "fork", nil)
	if err != nil {
		t.Fatalf("fork failed: %v", err)
	}
}

func TestRepoDelete(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "DELETE" {
			t.Fatalf("expected DELETE, got %s", r.Method)
		}
		if r.URL.Path != "/owner/repo.json" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		writeJSON(t, w, map[string]interface{}{"message": "deleted"})
	}))
	defer server.Close()

	err := runShortcut(t, server, "delete", nil)
	if err != nil {
		t.Fatalf("delete failed: %v", err)
	}
}

func TestRepoCreate(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/users/me.json" && r.Method == "GET":
			writeJSON(t, w, map[string]interface{}{
				"login":   "creator",
				"user_id": float64(42),
			})
		case r.URL.Path == "/creator/new-repo.json" && r.Method == "POST":
			writeJSON(t, w, map[string]interface{}{"name": "new-repo"})
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	err := runShortcut(t, server, "create", map[string]string{"name": "new-repo"})
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}
}

func TestRepoCreateWithOptions(t *testing.T) {
	var body map[string]interface{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/users/me.json":
			writeJSON(t, w, map[string]interface{}{
				"login":   "creator",
				"user_id": float64(42),
			})
		case "/creator/my-repo.json":
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode request body: %v", err)
			}
			writeJSON(t, w, map[string]interface{}{"name": "my-repo"})
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	err := runShortcut(t, server, "create", map[string]string{
		"name":        "my-repo",
		"description": "a test repo",
		"private":     "true",
	})
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}
	if body["description"] != "a test repo" {
		t.Fatalf("description = %v", body["description"])
	}
	if body["private"] != true {
		t.Fatalf("private = %v", body["private"])
	}
}

// --- validation/dry-run/error paths ---

func TestRepoDryRunDoesNotCallAPI(t *testing.T) {
	dryRunCases := []struct {
		name string
		args map[string]string
	}{
		{name: "units-update", args: map[string]string{"units": "code,issues", "dry-run": "true"}},
		{name: "topic-add", args: map[string]string{"project-id": "17", "name": "go", "dry-run": "true"}},
		{name: "topic-delete", args: map[string]string{"id": "8", "project-id": "17", "dry-run": "true"}},
		{name: "transfer", args: map[string]string{"owner-name": "target-org", "dry-run": "true"}},
		{name: "transfer-cancel", args: map[string]string{"dry-run": "true"}},
	}

	for _, tc := range dryRunCases {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				t.Fatalf("dry-run should not call API, got %s %s", r.Method, r.URL.Path)
			}))
			defer server.Close()

			if err := runShortcut(t, server, tc.name, tc.args); err != nil {
				t.Fatalf("%s dry-run failed: %v", tc.name, err)
			}
		})
	}
}

func TestRepoTopicAddRejectsInvalidProjectID(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatalf("invalid project id should not call API, got %s %s", r.Method, r.URL.Path)
	}))
	defer server.Close()

	err := runShortcut(t, server, "topic-add", map[string]string{
		"project-id": "abc",
		"name":       "go",
	})
	if err == nil {
		t.Fatal("expected invalid project id to return an error")
	}
}

func TestRepoCreateFailsWithoutName(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("no API call should be made")
	}))
	defer server.Close()

	err := runShortcut(t, server, "create", map[string]string{})
	if err == nil {
		t.Fatal("expected error for missing name")
	}
}

func TestRepoListHTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeText(t, w, http.StatusInternalServerError, "server error")
	}))
	defer server.Close()

	err := runShortcut(t, server, "list", map[string]string{"page": "1", "limit": "20"})
	if err == nil {
		t.Fatal("expected error for HTTP 500")
	}
}

func TestRepoInfoHTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeText(t, w, http.StatusInternalServerError, "server error")
	}))
	defer server.Close()

	err := runShortcut(t, server, "info", nil)
	if err == nil {
		t.Fatal("expected error for HTTP 500")
	}
}

func TestRepoForkHTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeText(t, w, http.StatusInternalServerError, "server error")
	}))
	defer server.Close()

	err := runShortcut(t, server, "fork", nil)
	if err == nil {
		t.Fatal("expected error for HTTP 500")
	}
}

func TestRepoDeleteHTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeText(t, w, http.StatusInternalServerError, "server error")
	}))
	defer server.Close()

	err := runShortcut(t, server, "delete", nil)
	if err == nil {
		t.Fatal("expected error for HTTP 500")
	}
}

func TestRepoCreateGetUserHTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeText(t, w, http.StatusInternalServerError, "server error")
	}))
	defer server.Close()

	err := runShortcut(t, server, "create", map[string]string{"name": "new-repo"})
	if err == nil {
		t.Fatal("expected error when get user fails")
	}
}

func TestRepoCreateUserNoLogin(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(t, w, map[string]interface{}{"user_id": float64(42)})
	}))
	defer server.Close()

	err := runShortcut(t, server, "create", map[string]string{"name": "new-repo"})
	if err == nil {
		t.Fatal("expected error when user response has no login")
	}
}

func assertRepoRequest(t *testing.T, r *http.Request, method, path string) {
	t.Helper()
	if r.Method != method || r.URL.Path != path {
		t.Fatalf("got request %s %s, want %s %s", r.Method, r.URL.Path, method, path)
	}
}

func assertRepoQuery(t *testing.T, r *http.Request, key, want string) {
	t.Helper()
	if got := r.URL.Query().Get(key); got != want {
		t.Fatalf("query %s = %q, want %q", key, got, want)
	}
}

func decodeRepoJSON(t *testing.T, r *http.Request) map[string]interface{} {
	t.Helper()
	var payload map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		t.Fatalf("failed to decode request body: %v", err)
	}
	return payload
}

func assertRepoEqual(t *testing.T, got interface{}, want interface{}) {
	t.Helper()
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v (%T), want %v (%T)", got, got, want, want)
	}
}

func assertRepoStringSlice(t *testing.T, got interface{}, want []string) {
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
