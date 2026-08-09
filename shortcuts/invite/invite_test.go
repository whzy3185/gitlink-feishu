package invite

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

// --- generate ---

func TestInviteGenerateDefault(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Fatalf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/owner/repo/project_invite_links/current_link.json" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if r.URL.Query().Get("role") != "developer" {
			t.Fatalf("expected role=developer, got %s", r.URL.Query().Get("role"))
		}
		if r.URL.Query().Get("is_apply") != "true" {
			t.Fatalf("expected is_apply=true, got %s", r.URL.Query().Get("is_apply"))
		}
		writeJSON(t, w, map[string]interface{}{
			"invite_sign": "abc123",
			"role":        "developer",
		})
	}))
	defer server.Close()

	err := runShortcut(t, server, "generate", nil)
	if err != nil {
		t.Fatalf("generate failed: %v", err)
	}
}

func TestInviteGenerateWithRole(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("role") != "manager" {
			t.Fatalf("expected role=manager, got %s", r.URL.Query().Get("role"))
		}
		writeJSON(t, w, map[string]interface{}{
			"invite_sign": "xyz789",
			"role":        "manager",
		})
	}))
	defer server.Close()

	err := runShortcut(t, server, "generate", map[string]string{"role": "manager"})
	if err != nil {
		t.Fatalf("generate with role failed: %v", err)
	}
}

func TestInviteGenerateWithIsApply(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("is_apply") != "false" {
			t.Fatalf("expected is_apply=false, got %s", r.URL.Query().Get("is_apply"))
		}
		writeJSON(t, w, map[string]interface{}{
			"invite_sign": "test456",
			"role":        "developer",
		})
	}))
	defer server.Close()

	err := runShortcut(t, server, "generate", map[string]string{"is-apply": "false"})
	if err != nil {
		t.Fatalf("generate with is-apply failed: %v", err)
	}
}

// --- show ---

func TestInviteShow(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Fatalf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/owner/repo/project_invite_links/show_link.json" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if r.URL.Query().Get("invite_sign") != "abc123" {
			t.Fatalf("expected invite_sign=abc123, got %s", r.URL.Query().Get("invite_sign"))
		}
		writeJSON(t, w, map[string]interface{}{
			"invite_sign": "abc123",
			"project":     "owner/repo",
			"role":        "developer",
		})
	}))
	defer server.Close()

	err := runShortcut(t, server, "show", map[string]string{"invite-sign": "abc123"})
	if err != nil {
		t.Fatalf("show failed: %v", err)
	}
}

func TestInviteShowMissingSign(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("no API call should be made without invite-sign")
	}))
	defer server.Close()

	err := runShortcut(t, server, "show", nil)
	if err == nil {
		t.Fatal("expected error for missing invite-sign")
	}
}

// --- accept ---

func TestInviteAccept(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Fatalf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/owner/repo/project_invite_links/redirect_link.json" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if r.URL.Query().Get("invite_sign") != "abc123" {
			t.Fatalf("expected invite_sign=abc123, got %s", r.URL.Query().Get("invite_sign"))
		}
		writeJSON(t, w, map[string]interface{}{
			"status":  0,
			"message": "success",
		})
	}))
	defer server.Close()

	err := runShortcut(t, server, "accept", map[string]string{"invite-sign": "abc123"})
	if err != nil {
		t.Fatalf("accept failed: %v", err)
	}
}

func TestInviteAcceptMissingSign(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("no API call should be made without invite-sign")
	}))
	defer server.Close()

	err := runShortcut(t, server, "accept", nil)
	if err == nil {
		t.Fatal("expected error for missing invite-sign")
	}
}

// --- join ---

func TestInviteJoin(t *testing.T) {
	var body map[string]interface{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Fatalf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/applied_projects.json" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		writeJSON(t, w, map[string]interface{}{
			"status":  0,
			"message": "success",
		})
	}))
	defer server.Close()

	err := runShortcut(t, server, "join", map[string]string{"code": "PROJECT123"})
	if err != nil {
		t.Fatalf("join failed: %v", err)
	}

	appliedProject, ok := body["applied_project"].(map[string]interface{})
	if !ok {
		t.Fatal("expected applied_project in body")
	}
	if appliedProject["code"] != "PROJECT123" {
		t.Fatalf("expected code=PROJECT123, got %v", appliedProject["code"])
	}
	if appliedProject["role"] != "developer" {
		t.Fatalf("expected role=developer, got %v", appliedProject["role"])
	}
}

func TestInviteJoinWithRole(t *testing.T) {
	var body map[string]interface{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		writeJSON(t, w, map[string]interface{}{"status": 0})
	}))
	defer server.Close()

	err := runShortcut(t, server, "join", map[string]string{"code": "PROJECT123", "role": "manager"})
	if err != nil {
		t.Fatalf("join with role failed: %v", err)
	}

	appliedProject, ok := body["applied_project"].(map[string]interface{})
	if !ok {
		t.Fatal("expected applied_project in body")
	}
	if appliedProject["role"] != "manager" {
		t.Fatalf("expected role=manager, got %v", appliedProject["role"])
	}
}

func TestInviteJoinMissingCode(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("no API call should be made without code")
	}))
	defer server.Close()

	err := runShortcut(t, server, "join", nil)
	if err == nil {
		t.Fatal("expected error for missing code")
	}
}

// --- quit ---

func TestInviteQuit(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Fatalf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/owner/repo/quit.json" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		writeJSON(t, w, map[string]interface{}{
			"status":  0,
			"message": "success",
		})
	}))
	defer server.Close()

	err := runShortcut(t, server, "quit", nil)
	if err != nil {
		t.Fatalf("quit failed: %v", err)
	}
}

// --- HTTP error handling ---

func TestInviteGenerateHTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("server error"))
	}))
	defer server.Close()

	err := runShortcut(t, server, "generate", nil)
	if err == nil {
		t.Fatal("expected error for HTTP 500")
	}
}

func TestInviteShowHTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("server error"))
	}))
	defer server.Close()

	err := runShortcut(t, server, "show", map[string]string{"invite-sign": "abc123"})
	if err == nil {
		t.Fatal("expected error for HTTP 500")
	}
}

func TestInviteAcceptHTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("server error"))
	}))
	defer server.Close()

	err := runShortcut(t, server, "accept", map[string]string{"invite-sign": "abc123"})
	if err == nil {
		t.Fatal("expected error for HTTP 500")
	}
}

func TestInviteJoinHTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("server error"))
	}))
	defer server.Close()

	err := runShortcut(t, server, "join", map[string]string{"code": "PROJECT123"})
	if err == nil {
		t.Fatal("expected error for HTTP 500")
	}
}

func TestInviteQuitHTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("server error"))
	}))
	defer server.Close()

	err := runShortcut(t, server, "quit", nil)
	if err == nil {
		t.Fatal("expected error for HTTP 500")
	}
}

// --- shortcut registration ---

func TestInviteShortcutsRegistered(t *testing.T) {
	shortcuts := Shortcuts()
	names := make(map[string]bool)
	for _, s := range shortcuts {
		names[s.Name] = true
	}

	expected := []string{"generate", "show", "accept", "join", "quit"}
	for _, name := range expected {
		if !names[name] {
			t.Fatalf("shortcut %q not registered", name)
		}
	}
}

func TestInviteShortcutsHaveDescription(t *testing.T) {
	for _, s := range Shortcuts() {
		if s.Description == "" {
			t.Fatalf("shortcut %q has no description", s.Name)
		}
	}
}
