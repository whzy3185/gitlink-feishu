package auth

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/zalando/go-keyring"
)

func setupConfigDir(t *testing.T, baseURL string) string {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("GITLINK_CONFIG_DIR", dir)
	// Write a minimal config
	cfgDir := filepath.Join(dir)
	os.MkdirAll(cfgDir, 0700)
	os.WriteFile(filepath.Join(cfgDir, "config.yaml"), []byte("base_url: "+baseURL+"\ndefault_format: table\n"), 0600)
	return dir
}

func TestGetCurrentUserSuccess(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/users/me.json" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"login": "testuser",
			"name":  "Test User",
			"id":    42,
		})
	}))
	defer server.Close()

	setupConfigDir(t, server.URL)
	// Need to prevent any cookie/token auth from interfering
	os.Unsetenv("GITLINK_TOKEN")

	user, err := GetCurrentUser()
	if err != nil {
		t.Fatalf("GetCurrentUser error: %v", err)
	}
	if user["login"] != "testuser" {
		t.Fatalf("login = %q, want testuser", user["login"])
	}
	if user["name"] != "Test User" {
		t.Fatalf("name = %q", user["name"])
	}
}

func TestGetCurrentUserHTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte("unauthorized"))
	}))
	defer server.Close()

	setupConfigDir(t, server.URL)
	os.Unsetenv("GITLINK_TOKEN")

	_, err := GetCurrentUser()
	if err == nil {
		t.Fatal("expected error for 401")
	}
}

func TestGetCurrentUserStatusError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  float64(-1),
			"message": "Token invalid",
		})
	}))
	defer server.Close()

	setupConfigDir(t, server.URL)
	os.Unsetenv("GITLINK_TOKEN")

	_, err := GetCurrentUser()
	if err == nil {
		t.Fatal("expected error for status=-1")
	}
}

func TestGetCurrentUserMissingLogin(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id": 42,
		})
	}))
	defer server.Close()

	setupConfigDir(t, server.URL)
	os.Unsetenv("GITLINK_TOKEN")

	_, err := GetCurrentUser()
	if err == nil {
		t.Fatal("expected error when login field is missing")
	}
}

func TestLoginSuccess(t *testing.T) {
	keyring.MockInitWithError(errors.New("keychain unavailable"))
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	t.Setenv("GITLINK_TOKEN", "")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == "POST" && r.URL.Path == "/accounts/login.json":
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("Set-Cookie", "autologin_trustie=sess123; Path=/")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"username": "testuser",
				"login":    "testuser",
				"user_id":  42,
				"token":    "tok123",
			})
		case r.Method == "GET" && r.URL.Path == "/users/me.json":
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"login": "testuser",
				"name":  "Test User",
				"id":    float64(42),
			})
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	setupConfigDir(t, server.URL)

	result, err := Login("testuser", "password")
	if err != nil {
		t.Fatalf("Login error: %v", err)
	}
	if result.Username != "testuser" {
		t.Fatalf("Username = %q, want testuser", result.Username)
	}
}

func TestLoginStatusError(t *testing.T) {
	keyring.MockInitWithError(errors.New("keychain unavailable"))

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  float64(-1),
			"message": "Invalid credentials",
		})
	}))
	defer server.Close()

	setupConfigDir(t, server.URL)

	_, err := Login("testuser", "wrongpass")
	if err == nil {
		t.Fatal("expected error for invalid credentials")
	}
}

func TestLoginNoCookies(t *testing.T) {
	keyring.MockInitWithError(errors.New("keychain unavailable"))
	dir := t.TempDir()
	t.Setenv("HOME", dir)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"username": "testuser",
			"login":    "testuser",
			"user_id":  42,
		})
	}))
	defer server.Close()

	setupConfigDir(t, server.URL)

	_, err := Login("testuser", "password")
	if err == nil {
		t.Fatal("expected error when no auth cookies")
	}
}
