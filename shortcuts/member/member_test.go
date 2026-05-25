package member

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/gitlink-org/gitlink-cli/internal/client"
	"github.com/gitlink-org/gitlink-cli/shortcuts/common"
)

func TestMemberList(t *testing.T) {
	server := newMemberTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		assertRequest(t, r, "GET", "/owner/repo/collaborators.json")
		writeJSON(t, w, map[string]interface{}{"total_count": 1, "members": []interface{}{}})
	})
	defer server.Close()

	if err := runMemberShortcut(t, server, "list", nil); err != nil {
		t.Fatalf("list shortcut failed: %v", err)
	}
}

func TestMemberAdd(t *testing.T) {
	var payload map[string]interface{}
	server := newMemberTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		assertRequest(t, r, "POST", "/owner/repo/collaborators.json")
		payload = decodeJSON(t, r)
		writeJSON(t, w, map[string]interface{}{"status": 0, "message": "success"})
	})
	defer server.Close()

	if err := runMemberShortcut(t, server, "add", map[string]string{"user-id": "101"}); err != nil {
		t.Fatalf("add shortcut failed: %v", err)
	}
	assertNumber(t, payload["user_id"], 101)
}

func TestMemberBatchAdd(t *testing.T) {
	var seen []int
	server := newMemberTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		assertRequest(t, r, "POST", "/owner/repo/collaborators.json")
		payload := decodeJSON(t, r)
		seen = append(seen, int(payload["user_id"].(float64)))
		writeJSON(t, w, map[string]interface{}{"status": 0, "message": "success"})
	})
	defer server.Close()

	csvPath := writeTempCSV(t, "user_id\n102\n103\n")
	err := runMemberShortcut(t, server, "batch-add", map[string]string{
		"user-ids": "101,102",
		"from":     csvPath,
	})
	if err != nil {
		t.Fatalf("batch-add shortcut failed: %v", err)
	}
	want := []int{101, 102, 103}
	if !reflect.DeepEqual(seen, want) {
		t.Fatalf("batch-add user IDs = %v, want %v", seen, want)
	}
}

func TestMemberBatchAddDryRunDoesNotCallAPI(t *testing.T) {
	server := newMemberTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		t.Fatalf("dry-run should not call API, got: %s %s", r.Method, r.URL.Path)
	})
	defer server.Close()

	err := runMemberShortcut(t, server, "batch-add", map[string]string{
		"user-ids": "101,102",
		"dry-run":  "true",
	})
	if err != nil {
		t.Fatalf("batch-add dry-run failed: %v", err)
	}
}

func TestMemberBatchAddReturnsErrorWhenAnyRequestFails(t *testing.T) {
	server := newMemberTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		assertRequest(t, r, "POST", "/owner/repo/collaborators.json")
		payload := decodeJSON(t, r)
		if int(payload["user_id"].(float64)) == 102 {
			http.Error(w, "member add failed", http.StatusBadRequest)
			return
		}
		writeJSON(t, w, map[string]interface{}{"status": 0, "message": "success"})
	})
	defer server.Close()

	err := runMemberShortcut(t, server, "batch-add", map[string]string{
		"user-ids": "101,102",
	})
	if err == nil {
		t.Fatal("expected batch-add to return an error when one request fails")
	}
}

func TestMemberRemove(t *testing.T) {
	var payload map[string]interface{}
	server := newMemberTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		assertRequest(t, r, "DELETE", "/owner/repo/collaborators/remove.json")
		payload = decodeJSON(t, r)
		writeJSON(t, w, map[string]interface{}{"status": 0, "message": "success"})
	})
	defer server.Close()

	if err := runMemberShortcut(t, server, "remove", map[string]string{"user-id": "101"}); err != nil {
		t.Fatalf("remove shortcut failed: %v", err)
	}
	assertNumber(t, payload["user_id"], 101)
}

func TestMemberRole(t *testing.T) {
	var payload map[string]interface{}
	server := newMemberTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		assertRequest(t, r, "PUT", "/owner/repo/collaborators/change_role.json")
		payload = decodeJSON(t, r)
		writeJSON(t, w, map[string]interface{}{"status": 0, "message": "success"})
	})
	defer server.Close()

	err := runMemberShortcut(t, server, "role", map[string]string{
		"user-id": "101",
		"role":    "developer",
	})
	if err != nil {
		t.Fatalf("role shortcut failed: %v", err)
	}
	assertNumber(t, payload["user_id"], 101)
	if payload["role"] != "Developer" {
		t.Fatalf("role = %v, want Developer", payload["role"])
	}
}

func TestMemberInviteLink(t *testing.T) {
	server := newMemberTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		assertRequest(t, r, "GET", "/owner/repo/project_invite_links/current_link.json")
		if r.URL.Query().Get("role") != "developer" {
			t.Fatalf("role query = %q, want developer", r.URL.Query().Get("role"))
		}
		if r.URL.Query().Get("is_apply") != "false" {
			t.Fatalf("is_apply query = %q, want false", r.URL.Query().Get("is_apply"))
		}
		writeJSON(t, w, map[string]interface{}{"sign": "abc"})
	})
	defer server.Close()

	err := runMemberShortcut(t, server, "invite-link", map[string]string{
		"role":  "developer",
		"apply": "false",
	})
	if err != nil {
		t.Fatalf("invite-link shortcut failed: %v", err)
	}
}

func TestMemberInviteInfo(t *testing.T) {
	server := newMemberTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		assertRequest(t, r, "GET", "/owner/repo/project_invite_links/show_link.json")
		if r.URL.Query().Get("invite_sign") != "abc" {
			t.Fatalf("invite_sign query = %q, want abc", r.URL.Query().Get("invite_sign"))
		}
		writeJSON(t, w, map[string]interface{}{"sign": "abc"})
	})
	defer server.Close()

	if err := runMemberShortcut(t, server, "invite-info", map[string]string{"sign": "abc"}); err != nil {
		t.Fatalf("invite-info shortcut failed: %v", err)
	}
}

func TestMemberAcceptInvite(t *testing.T) {
	server := newMemberTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		assertRequest(t, r, "POST", "/owner/repo/project_invite_links/redirect_link.json")
		if r.URL.Query().Get("invite_sign") != "abc" {
			t.Fatalf("invite_sign query = %q, want abc", r.URL.Query().Get("invite_sign"))
		}
		writeJSON(t, w, map[string]interface{}{"status": 0, "message": "success"})
	})
	defer server.Close()

	if err := runMemberShortcut(t, server, "accept-invite", map[string]string{"sign": "abc"}); err != nil {
		t.Fatalf("accept-invite shortcut failed: %v", err)
	}
}

func TestCollectUserIDs(t *testing.T) {
	csvPath := writeTempCSV(t, "name,id\nfirst,102\nsecond,103\n")
	got, err := collectUserIDs("101,102", csvPath)
	if err != nil {
		t.Fatalf("collectUserIDs returned error: %v", err)
	}
	want := []int{101, 102, 103}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("collectUserIDs() = %v, want %v", got, want)
	}
}

func TestNormalizeRoleRejectsInvalidRole(t *testing.T) {
	if _, err := normalizeRole("owner"); err == nil {
		t.Fatal("expected invalid role to return an error")
	}
}

func runMemberShortcut(t *testing.T, server *httptest.Server, name string, args map[string]string) error {
	t.Helper()
	shortcut := findMemberShortcut(t, name)
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

func findMemberShortcut(t *testing.T, name string) *common.Shortcut {
	t.Helper()
	for _, shortcut := range Shortcuts() {
		if shortcut.Name == name {
			return shortcut
		}
	}
	t.Fatalf("shortcut %q not found", name)
	return nil
}

func newMemberTestServer(t *testing.T, handler http.HandlerFunc) *httptest.Server {
	t.Helper()
	return httptest.NewServer(handler)
}

func assertRequest(t *testing.T, r *http.Request, method, path string) {
	t.Helper()
	if r.Method != method || r.URL.Path != path {
		t.Fatalf("got request %s %s, want %s %s", r.Method, r.URL.Path, method, path)
	}
}

func decodeJSON(t *testing.T, r *http.Request) map[string]interface{} {
	t.Helper()
	var payload map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		t.Fatalf("failed to decode request body: %v", err)
	}
	return payload
}

func writeJSON(t *testing.T, w http.ResponseWriter, payload interface{}) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		t.Fatalf("failed to write response: %v", err)
	}
}

func assertNumber(t *testing.T, got interface{}, want int) {
	t.Helper()
	value, ok := got.(float64)
	if !ok {
		t.Fatalf("got %v (%T), want JSON number", got, got)
	}
	if int(value) != want {
		t.Fatalf("got %v, want %d", got, want)
	}
}

func writeTempCSV(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "members.csv")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write temp csv: %v", err)
	}
	return path
}
