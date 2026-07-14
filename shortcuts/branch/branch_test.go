package branch

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

func newBranchTestServer(t *testing.T, handler http.HandlerFunc) *httptest.Server {
	t.Helper()
	return httptest.NewServer(handler)
}

func assertBranchRequest(t *testing.T, r *http.Request, method, path string) {
	t.Helper()
	if r.Method != method || r.URL.Path != path {
		t.Fatalf("got request %s %s, want %s %s", r.Method, r.URL.Path, method, path)
	}
}

func decodeBranchJSON(t *testing.T, r *http.Request) map[string]interface{} {
	t.Helper()
	var payload map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		t.Fatalf("failed to decode request body: %v", err)
	}
	return payload
}

func writeBranchJSON(t *testing.T, w http.ResponseWriter, payload interface{}) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		t.Fatalf("failed to write response: %v", err)
	}
}

func writeBranchText(t *testing.T, w http.ResponseWriter, code int, text string) {
	t.Helper()
	w.WriteHeader(code)
	if _, err := w.Write([]byte(text)); err != nil {
		t.Fatalf("failed to write response: %v", err)
	}
}

func assertBranchEqual(t *testing.T, got interface{}, want interface{}) {
	t.Helper()
	if got != want {
		t.Fatalf("got %v (%T), want %v (%T)", got, got, want, want)
	}
}

func TestBranchListBuildsQuery(t *testing.T) {
	server := newBranchTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		assertBranchRequest(t, r, "GET", "/v1/owner/repo/branches.json")
		query := r.URL.Query()
		assertBranchEqual(t, query.Get("keyword"), "feature")
		assertBranchEqual(t, query.Get("state"), "deleted")
		assertBranchEqual(t, query.Get("page"), "2")
		assertBranchEqual(t, query.Get("limit"), "5")
		writeBranchJSON(t, w, map[string]interface{}{"total_count": 0, "branches": []interface{}{}})
	})
	defer server.Close()

	err := runShortcut(t, server, "list", map[string]string{
		"keyword": "feature",
		"state":   "deleted",
		"page":    "2",
		"limit":   "5",
	})
	if err != nil {
		t.Fatalf("list shortcut failed: %v", err)
	}
}

func TestBranchAllUsesAllEndpoint(t *testing.T) {
	server := newBranchTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		assertBranchRequest(t, r, "GET", "/v1/owner/repo/branches/all.json")
		writeBranchJSON(t, w, []map[string]interface{}{{"name": "master"}})
	})
	defer server.Close()

	if err := runShortcut(t, server, "all", nil); err != nil {
		t.Fatalf("all shortcut failed: %v", err)
	}
}

func TestBranchCreatePayload(t *testing.T) {
	var payload map[string]interface{}
	server := newBranchTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		assertBranchRequest(t, r, "POST", "/v1/owner/repo/branches.json")
		payload = decodeBranchJSON(t, r)
		writeBranchJSON(t, w, map[string]interface{}{"name": "feature/a"})
	})
	defer server.Close()

	err := runShortcut(t, server, "create", map[string]string{
		"name": "feature/a",
		"from": "master",
	})
	if err != nil {
		t.Fatalf("create shortcut failed: %v", err)
	}
	assertBranchEqual(t, payload["new_branch_name"], "feature/a")
	assertBranchEqual(t, payload["old_branch_name"], "master")
}

func TestBranchCreateDefaultFrom(t *testing.T) {
	var payload map[string]interface{}
	server := newBranchTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		assertBranchRequest(t, r, "POST", "/v1/owner/repo/branches.json")
		payload = decodeBranchJSON(t, r)
		writeBranchJSON(t, w, map[string]interface{}{"name": "feature-y"})
	})
	defer server.Close()

	err := runShortcut(t, server, "create", map[string]string{"name": "feature-y"})
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}
	assertBranchEqual(t, payload["old_branch_name"], "master")
}

func TestBranchCreateDryRunDoesNotCallAPI(t *testing.T) {
	server := newBranchTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		t.Fatalf("dry-run should not call API, got: %s %s", r.Method, r.URL.Path)
	})
	defer server.Close()

	err := runShortcut(t, server, "create", map[string]string{
		"name":    "feature/a",
		"from":    "master",
		"dry-run": "true",
	})
	if err != nil {
		t.Fatalf("create dry-run failed: %v", err)
	}
}

func TestBranchDeleteUsesV1EndpointAndEscapesName(t *testing.T) {
	server := newBranchTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "DELETE" || r.URL.EscapedPath() != "/v1/owner/repo/branches/feature%2Fold.json" {
			t.Fatalf("got request %s %s, want DELETE /v1/owner/repo/branches/feature%%2Fold.json", r.Method, r.URL.EscapedPath())
		}
		writeBranchJSON(t, w, map[string]interface{}{"status": 0, "message": "success"})
	})
	defer server.Close()

	err := runShortcut(t, server, "delete", map[string]string{"name": "feature/old"})
	if err != nil {
		t.Fatalf("delete shortcut failed: %v", err)
	}
}

func TestBranchSetDefaultUsesQueryName(t *testing.T) {
	server := newBranchTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		assertBranchRequest(t, r, "PATCH", "/v1/owner/repo/branches/update_default_branch.json")
		assertBranchEqual(t, r.URL.Query().Get("name"), "develop")
		writeBranchJSON(t, w, map[string]interface{}{"status": 0, "message": "success"})
	})
	defer server.Close()

	err := runShortcut(t, server, "set-default", map[string]string{"name": "develop"})
	if err != nil {
		t.Fatalf("set-default shortcut failed: %v", err)
	}
}

func TestBranchRestorePayload(t *testing.T) {
	var payload map[string]interface{}
	server := newBranchTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		assertBranchRequest(t, r, "POST", "/v1/owner/repo/branches/restore.json")
		payload = decodeBranchJSON(t, r)
		writeBranchJSON(t, w, map[string]interface{}{"status": 0, "message": "success"})
	})
	defer server.Close()

	err := runShortcut(t, server, "restore", map[string]string{
		"branch-id": "7",
		"name":      "feature/deleted",
	})
	if err != nil {
		t.Fatalf("restore shortcut failed: %v", err)
	}
	assertBranchEqual(t, payload["branch_id"], float64(7))
	assertBranchEqual(t, payload["branch_name"], "feature/deleted")
}

func TestBranchRejectsInvalidInputs(t *testing.T) {
	server := newBranchTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		t.Fatalf("server should not be called for invalid input: %s %s", r.Method, r.URL.Path)
	})
	defer server.Close()

	if err := runShortcut(t, server, "list", map[string]string{"state": "open"}); err == nil {
		t.Fatal("expected invalid state to fail")
	}
	if err := runShortcut(t, server, "restore", map[string]string{"branch-id": "abc", "name": "deleted"}); err == nil {
		t.Fatal("expected invalid branch-id to fail")
	}
}

func TestBranchProtect(t *testing.T) {
	server := newBranchTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		assertBranchRequest(t, r, "POST", "/owner/repo/protected_branches.json")
		writeBranchJSON(t, w, map[string]interface{}{"message": "protected"})
	})
	defer server.Close()

	err := runShortcut(t, server, "protect", map[string]string{"name": "master"})
	if err != nil {
		t.Fatalf("protect failed: %v", err)
	}
}

func TestBranchUnprotect(t *testing.T) {
	server := newBranchTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		assertBranchRequest(t, r, "DELETE", "/owner/repo/protected_branches/master.json")
		writeBranchJSON(t, w, map[string]interface{}{"message": "unprotected"})
	})
	defer server.Close()

	err := runShortcut(t, server, "unprotect", map[string]string{"name": "master"})
	if err != nil {
		t.Fatalf("unprotect failed: %v", err)
	}
}

func TestBranchListHTTPError(t *testing.T) {
	server := newBranchTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		writeBranchText(t, w, http.StatusInternalServerError, "server error")
	})
	defer server.Close()

	err := runShortcut(t, server, "list", map[string]string{"page": "1", "limit": "20"})
	if err == nil {
		t.Fatal("expected error for HTTP 500")
	}
}

func TestBranchCreateHTTPError(t *testing.T) {
	server := newBranchTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		writeBranchText(t, w, http.StatusInternalServerError, "server error")
	})
	defer server.Close()

	err := runShortcut(t, server, "create", map[string]string{"name": "feature-x"})
	if err == nil {
		t.Fatal("expected error for HTTP 500")
	}
}

func TestBranchDeleteHTTPError(t *testing.T) {
	server := newBranchTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		writeBranchText(t, w, http.StatusInternalServerError, "server error")
	})
	defer server.Close()

	err := runShortcut(t, server, "delete", map[string]string{"name": "old-branch"})
	if err == nil {
		t.Fatal("expected error for HTTP 500")
	}
}

func TestBranchProtectHTTPError(t *testing.T) {
	server := newBranchTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		writeBranchText(t, w, http.StatusInternalServerError, "server error")
	})
	defer server.Close()

	err := runShortcut(t, server, "protect", map[string]string{"name": "master"})
	if err == nil {
		t.Fatal("expected error for HTTP 500")
	}
}

func TestBranchUnprotectHTTPError(t *testing.T) {
	server := newBranchTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		writeBranchText(t, w, http.StatusInternalServerError, "server error")
	})
	defer server.Close()

	err := runShortcut(t, server, "unprotect", map[string]string{"name": "master"})
	if err == nil {
		t.Fatal("expected error for HTTP 500")
	}
}
