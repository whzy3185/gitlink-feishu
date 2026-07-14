package user

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
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

func writeJSON(t *testing.T, w http.ResponseWriter, v interface{}) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(v); err != nil {
		t.Fatalf("write response: %v", err)
	}
}

// --- me ---

func TestUserMe(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/users/me.json" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		writeJSON(t, w, map[string]interface{}{
			"login": "currentuser",
			"name":  "Current User",
			"id":    float64(1),
		})
	}))
	defer server.Close()

	err := runShortcut(t, server, "me", nil)
	if err != nil {
		t.Fatalf("me failed: %v", err)
	}
}

// --- info ---

func TestUserInfo(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/users/alice.json" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		writeJSON(t, w, map[string]interface{}{
			"login": "alice",
			"name":  "Alice",
		})
	}))
	defer server.Close()

	err := runShortcut(t, server, "info", map[string]string{"login": "alice"})
	if err != nil {
		t.Fatalf("info failed: %v", err)
	}
}

func TestUserInfoMissingLogin(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("no API call expected")
	}))
	defer server.Close()

	err := runShortcut(t, server, "info", map[string]string{})
	if err == nil {
		t.Fatal("expected error for missing login")
	}
}

// --- SSH public keys ---

func TestUserKeysUsesPublicKeysEndpoint(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" || r.URL.Path != "/public_keys.json" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		if got := r.URL.Query().Get("page"); got != "2" {
			t.Fatalf("page query = %q, want 2", got)
		}
		if got := r.URL.Query().Get("limit"); got != "50" {
			t.Fatalf("limit query = %q, want 50", got)
		}
		writeJSON(t, w, map[string]interface{}{
			"total_count": 1,
			"public_keys": []interface{}{
				map[string]interface{}{"id": 1, "name": "laptop"},
			},
		})
	}))
	defer server.Close()

	err := runShortcut(t, server, "keys", map[string]string{
		"page":  "2",
		"limit": "50",
	})
	if err != nil {
		t.Fatalf("keys shortcut failed: %v", err)
	}
}

func TestUserAddKeySendsTitleAndInlineKey(t *testing.T) {
	const key = "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIDemoKey user@example.com"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" || r.URL.Path != "/public_keys.json" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		var body map[string]string
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if body["title"] != "laptop" {
			t.Fatalf("title = %q, want laptop", body["title"])
		}
		if body["key"] != key {
			t.Fatalf("key = %q, want %q", body["key"], key)
		}
		writeJSON(t, w, map[string]interface{}{
			"id":          2,
			"name":        "laptop",
			"fingerprint": "SHA256:demo",
		})
	}))
	defer server.Close()

	err := runShortcut(t, server, "add-key", map[string]string{
		"title": "laptop",
		"key":   key,
	})
	if err != nil {
		t.Fatalf("add-key shortcut failed: %v", err)
	}
}

func TestUserAddKeyReadsKeyFromFileAndDefaultsTitle(t *testing.T) {
	const key = "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQDemo user@example.com"
	path := filepath.Join(t.TempDir(), "id_rsa.pub")
	if err := os.WriteFile(path, []byte("  "+key+"\n"), 0o600); err != nil {
		t.Fatalf("write key file: %v", err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" || r.URL.Path != "/public_keys.json" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		var body map[string]string
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if body["title"] != "id_rsa.pub" {
			t.Fatalf("title = %q, want id_rsa.pub", body["title"])
		}
		if body["key"] != key {
			t.Fatalf("key = %q, want %q", body["key"], key)
		}
		writeJSON(t, w, map[string]interface{}{"id": 3})
	}))
	defer server.Close()

	err := runShortcut(t, server, "add-key", map[string]string{"from": path})
	if err != nil {
		t.Fatalf("add-key from file failed: %v", err)
	}
}

func TestUserAddKeyRejectsAmbiguousKeySourcesBeforeRequest(t *testing.T) {
	called := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	}))
	defer server.Close()

	err := runShortcut(t, server, "add-key", map[string]string{
		"title": "laptop",
		"key":   "ssh-ed25519 AAAA",
		"from":  "id_ed25519.pub",
	})
	if err == nil {
		t.Fatalf("expected ambiguous key source error")
	}
	if called {
		t.Fatalf("server was called for invalid key sources")
	}
}

func TestUserAddKeyRejectsInlineKeyWithoutTitleBeforeRequest(t *testing.T) {
	called := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	}))
	defer server.Close()

	err := runShortcut(t, server, "add-key", map[string]string{"key": "ssh-rsa AAAA"})
	if err == nil {
		t.Fatalf("expected missing title error")
	}
	if called {
		t.Fatalf("server was called without title")
	}
}

func TestUserDeleteKeyUsesPublicKeyID(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "DELETE" || r.URL.Path != "/public_keys/12.json" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		writeJSON(t, w, map[string]interface{}{
			"status":  0,
			"message": "success",
		})
	}))
	defer server.Close()

	if err := runShortcut(t, server, "delete-key", map[string]string{"id": "12"}); err != nil {
		t.Fatalf("delete-key shortcut failed: %v", err)
	}
}

func TestUserDeleteKeyRejectsNonNumericIDBeforeRequest(t *testing.T) {
	called := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	}))
	defer server.Close()

	err := runShortcut(t, server, "delete-key", map[string]string{"id": "12/34"})
	if err == nil {
		t.Fatalf("expected invalid key id error")
	}
	if called {
		t.Fatalf("server was called for invalid key id")
	}
}

func TestPublicKeyContentAcceptsCommonOpenSSHPrefixes(t *testing.T) {
	for _, key := range []string{
		"ssh-rsa AAAA",
		"ssh-dss AAAA",
		"ssh-ed25519 AAAA",
		"ecdsa-sha2-nistp256 AAAA",
		"sk-ssh-ed25519@openssh.com AAAA",
	} {
		t.Run(key, func(t *testing.T) {
			if _, err := publicKeyContent(key, ""); err != nil {
				t.Fatalf("publicKeyContent(%q) returned error: %v", key, err)
			}
		})
	}
}

func TestPublicKeyContentRejectsNonPublicKeyPrefix(t *testing.T) {
	if _, err := publicKeyContent("not-a-key", ""); err == nil {
		t.Fatalf("expected invalid public key prefix error")
	}
}

// --- HTTP error paths ---

func TestUserMeHTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("server error"))
	}))
	defer server.Close()

	err := runShortcut(t, server, "me", nil)
	if err == nil {
		t.Fatal("expected error for HTTP 500")
	}
}

func TestUserInfoHTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("server error"))
	}))
	defer server.Close()

	err := runShortcut(t, server, "info", map[string]string{"login": "alice"})
	if err == nil {
		t.Fatal("expected error for HTTP 500")
	}
}
