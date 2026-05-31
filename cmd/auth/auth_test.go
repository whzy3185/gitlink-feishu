package auth

import (
	"errors"
	"os"
	"testing"

	"github.com/spf13/cobra"
	"github.com/zalando/go-keyring"

	internalAuth "github.com/gitlink-org/gitlink-cli/internal/auth"
)

func TestEnvTokenVar(t *testing.T) {
	if envTokenVar != "GITLINK_TOKEN" {
		t.Fatalf("envTokenVar = %q, want GITLINK_TOKEN", envTokenVar)
	}
}

func TestNewAuthCmd(t *testing.T) {
	cmd := NewAuthCmd()
	if cmd.Use != "auth" {
		t.Fatalf("Use = %q, want auth", cmd.Use)
	}
	if cmd.Short == "" {
		t.Fatal("Short is empty")
	}

	expectedSubs := map[string]bool{
		"login": false, "logout": false, "status": false,
	}
	for _, sub := range cmd.Commands() {
		if _, ok := expectedSubs[sub.Use]; !ok {
			t.Fatalf("unexpected subcommand: %q", sub.Use)
		}
		if expectedSubs[sub.Use] {
			t.Fatalf("duplicate subcommand: %q", sub.Use)
		}
		expectedSubs[sub.Use] = true
		if sub.Short == "" {
			t.Fatalf("subcommand %q has empty Short", sub.Use)
		}
	}
	for name, found := range expectedSubs {
		if !found {
			t.Fatalf("missing subcommand: %q", name)
		}
	}
}

func TestLoginTokenFlag(t *testing.T) {
	cmd := NewAuthCmd()
	loginCmd := findSub(cmd, "login")
	if loginCmd == nil {
		t.Fatal("login subcommand not found")
	}
	if f := loginCmd.Flags().Lookup("token"); f == nil {
		t.Fatal("login command missing --token flag")
	}
}

func TestStatusCmdNotLoggedIn(t *testing.T) {
	keyring.MockInitWithError(errors.New("keychain unavailable"))
	t.Setenv("HOME", t.TempDir())
	t.Setenv("GITLINK_TOKEN", "")
	_ = internalAuth.DeleteToken()

	cmd := findSub(NewAuthCmd(), "status")
	if cmd == nil {
		t.Fatal("status subcommand not found")
	}
	if err := cmd.RunE(cmd, nil); err != nil {
		t.Fatalf("status error: %v", err)
	}
}

func TestStatusCmdEnvToken(t *testing.T) {
	keyring.MockInitWithError(errors.New("keychain unavailable"))
	t.Setenv("HOME", t.TempDir())
	t.Setenv("GITLINK_TOKEN", "env-token-123")
	_ = internalAuth.DeleteToken()

	cmd := findSub(NewAuthCmd(), "status")
	cmd.RunE(cmd, nil)
}

func TestStatusCmdStoredToken(t *testing.T) {
	keyring.MockInitWithError(errors.New("keychain unavailable"))
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	t.Setenv("GITLINK_TOKEN", "")

	os.MkdirAll(dir+"/.config/gitlink-cli", 0700)
	os.WriteFile(dir+"/.config/gitlink-cli/credentials", []byte("cookie:test=abc"), 0600)

	cmd := findSub(NewAuthCmd(), "status")
	cmd.RunE(cmd, nil)
}

func TestStatusCmdEnvAndStoredToken(t *testing.T) {
	keyring.MockInitWithError(errors.New("keychain unavailable"))
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	t.Setenv("GITLINK_TOKEN", "env-token")

	os.MkdirAll(dir+"/.config/gitlink-cli", 0700)
	os.WriteFile(dir+"/.config/gitlink-cli/credentials", []byte("stored-token"), 0600)

	cmd := findSub(NewAuthCmd(), "status")
	if err := cmd.RunE(cmd, nil); err != nil {
		t.Fatalf("status error: %v", err)
	}
}

func TestStatusCmdStoredTokenButLoadFails(t *testing.T) {
	keyring.MockInitWithError(errors.New("keychain unavailable"))
	t.Setenv("HOME", t.TempDir())
	t.Setenv("GITLINK_TOKEN", "")
	// Don't create credentials file — LoadToken returns empty

	cmd := findSub(NewAuthCmd(), "status")
	if err := cmd.RunE(cmd, nil); err != nil {
		t.Fatalf("status error: %v", err)
	}
}

func TestLogoutCmdError(t *testing.T) {
	keyring.MockInitWithError(errors.New("keychain unavailable"))
	t.Setenv("HOME", t.TempDir())
	t.Setenv("GITLINK_TOKEN", "")
	// Don't create credentials dir — DeleteToken will fail

	cmd := findSub(NewAuthCmd(), "logout")
	err := cmd.RunE(cmd, nil)
	if err == nil {
		t.Fatal("expected error when DeleteToken fails")
	}
}

func TestLogoutCmd(t *testing.T) {
	keyring.MockInitWithError(errors.New("keychain unavailable"))
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("GITLINK_TOKEN", "")

	// Store a token first so DeleteToken has something to delete
	credDir := home + "/.config/gitlink-cli"
	os.MkdirAll(credDir, 0700)
	os.WriteFile(credDir+"/credentials", []byte("some-token"), 0600)

	cmd := findSub(NewAuthCmd(), "logout")
	if cmd == nil {
		t.Fatal("logout subcommand not found")
	}
	if err := cmd.RunE(cmd, nil); err != nil {
		t.Fatalf("logout error: %v", err)
	}
}

func TestLoginWithToken(t *testing.T) {
	keyring.MockInitWithError(errors.New("keychain unavailable"))
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("GITLINK_TOKEN", "")

	// Mock stdin
	oldStdin := os.Stdin
	r, w, _ := os.Pipe()
	os.Stdin = r
	defer func() { os.Stdin = oldStdin }()

	go func() {
		w.Write([]byte("test-token-123\n"))
		w.Close()
	}()

	cmd := findSub(NewAuthCmd(), "login")
	if cmd == nil {
		t.Fatal("login subcommand not found")
	}
	cmd.Flags().Set("token", "true")
	err := cmd.RunE(cmd, nil)
	if err != nil {
		t.Fatalf("login --token error: %v", err)
	}

	// Verify token was saved to file
	data, err := os.ReadFile(home + "/.config/gitlink-cli/credentials")
	if err != nil {
		t.Fatalf("read credentials: %v", err)
	}
	if string(data) != "test-token-123" {
		t.Fatalf("token = %q, want test-token-123", string(data))
	}
}

func TestLoginWithTokenEmpty(t *testing.T) {
	keyring.MockInitWithError(errors.New("keychain unavailable"))
	t.Setenv("HOME", t.TempDir())
	t.Setenv("GITLINK_TOKEN", "")

	oldStdin := os.Stdin
	r, w, _ := os.Pipe()
	os.Stdin = r
	defer func() { os.Stdin = oldStdin }()

	go func() {
		w.Write([]byte("\n"))
		w.Close()
	}()

	cmd := findSub(NewAuthCmd(), "login")
	cmd.Flags().Set("token", "true")
	err := cmd.RunE(cmd, nil)
	if err == nil {
		t.Fatal("expected error for empty token")
	}
}

func TestLoginWithPasswordNoTerminal(t *testing.T) {
	keyring.MockInitWithError(errors.New("keychain unavailable"))
	t.Setenv("HOME", t.TempDir())
	t.Setenv("GITLINK_TOKEN", "")

	// term.ReadPassword will fail because test has no terminal
	cmd := findSub(NewAuthCmd(), "login")
	// Don't set --token, so it goes to loginWithPassword
	err := cmd.RunE(cmd, nil)
	if err == nil {
		t.Fatal("expected error when terminal unavailable (ReadPassword fails)")
	}
}

func findSub(cmd *cobra.Command, name string) *cobra.Command {
	for _, sub := range cmd.Commands() {
		if sub.Use == name {
			return sub
		}
	}
	return nil
}
