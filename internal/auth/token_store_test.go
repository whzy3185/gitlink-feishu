package auth

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/zalando/go-keyring"
)

func tempHome(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	t.Setenv("USERPROFILE", dir)
	return dir
}

func tempConfigDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("GITLINK_CONFIG_DIR", dir)
	return dir
}

func TestStoreLoadDeleteTokenFile(t *testing.T) {
	tempConfigDir(t)

	// First, delete any existing token
	_ = deleteTokenFile()

	// Initially, loading should fail
	_, err := loadTokenFile()
	if err == nil {
		t.Fatal("expected error loading non-existent token file")
	}

	// Store a token
	if err := storeTokenFile("test-token-123"); err != nil {
		t.Fatalf("storeTokenFile error: %v", err)
	}

	// Load it back
	token, err := loadTokenFile()
	if err != nil {
		t.Fatalf("loadTokenFile error: %v", err)
	}
	if token != "test-token-123" {
		t.Fatalf("token = %q, want test-token-123", token)
	}

	// Delete it
	if err := deleteTokenFile(); err != nil {
		t.Fatalf("deleteTokenFile error: %v", err)
	}

	// Now loading should fail again
	_, err = loadTokenFile()
	if err == nil {
		t.Fatal("expected error after delete")
	}
}

func TestCredentialPath(t *testing.T) {
	dir := tempConfigDir(t)
	got := credentialPath()
	expected := filepath.Join(dir, "credentials")
	if got != expected {
		t.Fatalf("credentialPath = %q, want %q", got, expected)
	}
}

func TestCredentialPathDefaultsToConfigDir(t *testing.T) {
	home := tempHome(t)
	t.Setenv("GITLINK_CONFIG_DIR", "")

	got := credentialPath()
	expected := filepath.Join(home, ".config", "gitlink-cli", "credentials")
	if got != expected {
		t.Fatalf("credentialPath = %q, want %q", got, expected)
	}
}

func TestStoreTokenFileCreatesDir(t *testing.T) {
	dir := tempConfigDir(t)
	_ = deleteTokenFile()

	// Config dir shouldn't exist yet
	credDir := dir
	os.RemoveAll(credDir)

	if err := storeTokenFile("new-token"); err != nil {
		t.Fatalf("storeTokenFile error: %v", err)
	}

	// Verify file exists and has content
	data, err := os.ReadFile(filepath.Join(credDir, "credentials"))
	if err != nil {
		t.Fatalf("read error: %v", err)
	}
	if string(data) != "new-token" {
		t.Fatalf("file content = %q, want new-token", string(data))
	}
}

func TestDeleteTokenFileNonExistent(t *testing.T) {
	tempConfigDir(t)
	_ = deleteTokenFile()
	// Logout-style cleanup should be idempotent when fallback credentials do not exist.
	if err := deleteTokenFile(); err != nil {
		t.Fatalf("deleteTokenFile error: %v", err)
	}
}

func TestStoreLoadTokenFileEmpty(t *testing.T) {
	tempConfigDir(t)
	_ = deleteTokenFile()

	if err := storeTokenFile(""); err != nil {
		t.Fatalf("storeTokenFile empty: %v", err)
	}

	token, err := loadTokenFile()
	if err != nil {
		t.Fatalf("loadTokenFile error: %v", err)
	}
	if token != "" {
		t.Fatalf("token = %q, want empty", token)
	}
}

func TestStoreTokenFallback(t *testing.T) {
	keyring.MockInitWithError(errors.New("keychain unavailable"))
	dir := tempConfigDir(t)
	_ = deleteTokenFile()

	if err := StoreToken("keychain-fallback-token"); err != nil {
		t.Fatalf("StoreToken error: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, "credentials"))
	if err != nil {
		t.Fatalf("read error: %v", err)
	}
	if string(data) != "keychain-fallback-token" {
		t.Fatalf("file content = %q, want keychain-fallback-token", string(data))
	}
}

func TestDeleteTokenFallback(t *testing.T) {
	keyring.MockInitWithError(errors.New("keychain unavailable"))
	dir := tempConfigDir(t)
	_ = deleteTokenFile()

	p := filepath.Join(dir, "credentials")
	os.MkdirAll(filepath.Dir(p), 0700)
	os.WriteFile(p, []byte("delete-me"), 0600)

	if err := DeleteToken(); err != nil {
		t.Fatalf("DeleteToken error: %v", err)
	}

	_, err := os.Stat(p)
	if !os.IsNotExist(err) {
		t.Fatal("file should be deleted")
	}
}
