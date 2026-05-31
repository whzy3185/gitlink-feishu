package config

import (
	"os"
	"testing"

	"github.com/spf13/cobra"
)

func TestNewConfigCmd(t *testing.T) {
	cmd := NewConfigCmd()
	if cmd.Use != "config" {
		t.Fatalf("Use = %q, want config", cmd.Use)
	}
	if cmd.Short == "" {
		t.Fatal("Short is empty")
	}

	expectedSubs := map[string]bool{
		"init": false, "set <key> <value>": false, "get <key>": false, "list": false,
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

func TestSetCmdArgs(t *testing.T) {
	cmd := findSub(NewConfigCmd(), "set <key> <value>")
	if cmd == nil {
		t.Fatal("set subcommand not found")
	}
	if cmd.Args == nil {
		t.Fatal("set should require exact args")
	}
}

func TestGetCmdArgs(t *testing.T) {
	cmd := findSub(NewConfigCmd(), "get <key>")
	if cmd == nil {
		t.Fatal("get subcommand not found")
	}
	if cmd.Args == nil {
		t.Fatal("get should require exact args")
	}
}

func TestConfigInitRun(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("GITLINK_CONFIG_DIR", dir)

	cmd := findSub(NewConfigCmd(), "init")
	cmd.SetArgs([]string{})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("init error: %v", err)
	}
}

func TestConfigSetAndGet(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("GITLINK_CONFIG_DIR", dir)

	// Init first
	initCmd := findSub(NewConfigCmd(), "init")
	initCmd.SetArgs([]string{})
	if err := initCmd.Execute(); err != nil {
		t.Fatalf("init error: %v", err)
	}

	// Set a value
	setCmd := findSub(NewConfigCmd(), "set <key> <value>")
	setCmd.SetArgs([]string{"base_url", "https://example.com"})
	if err := setCmd.Execute(); err != nil {
		t.Fatalf("set error: %v", err)
	}

	// Get it back
	getCmd := findSub(NewConfigCmd(), "get <key>")
	getCmd.SetArgs([]string{"base_url"})
	if err := getCmd.Execute(); err != nil {
		t.Fatalf("get error: %v", err)
	}
}

func TestConfigGetNotSet(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("GITLINK_CONFIG_DIR", dir)
	os.WriteFile(dir+"/config.yaml", []byte("base_url: https://example.com\n"), 0644)

	getCmd := findSub(NewConfigCmd(), "get <key>")
	getCmd.SetArgs([]string{"editor"})
	if err := getCmd.Execute(); err != nil {
		t.Fatalf("get not-set error: %v", err)
	}
}

func TestConfigList(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("GITLINK_CONFIG_DIR", dir)

	initCmd := findSub(NewConfigCmd(), "init")
	initCmd.SetArgs([]string{})
	if err := initCmd.Execute(); err != nil {
		t.Fatalf("init error: %v", err)
	}

	listCmd := findSub(NewConfigCmd(), "list")
	listCmd.SetArgs([]string{})
	if err := listCmd.Execute(); err != nil {
		t.Fatalf("list error: %v", err)
	}
}

func TestConfigInitRunE(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("GITLINK_CONFIG_DIR", dir)

	cmd := newInitCmd()
	if err := cmd.RunE(cmd, nil); err != nil {
		t.Fatalf("init RunE error: %v", err)
	}
}

func TestConfigSetRunE(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("GITLINK_CONFIG_DIR", dir)

	// Init first so config file exists
	initCmd := newInitCmd()
	if err := initCmd.RunE(initCmd, nil); err != nil {
		t.Fatalf("init error: %v", err)
	}

	cmd := newSetCmd()
	if err := cmd.RunE(cmd, []string{"base_url", "https://example.com"}); err != nil {
		t.Fatalf("set RunE error: %v", err)
	}
}

func TestConfigSetRunENoConfig(t *testing.T) {
	// Set without init should still work — Load returns defaults, Save creates dir
	dir := t.TempDir()
	t.Setenv("GITLINK_CONFIG_DIR", dir)
	cmd := newSetCmd()
	if err := cmd.RunE(cmd, []string{"base_url", "https://example.com"}); err != nil {
		t.Fatalf("set RunE error: %v", err)
	}
}

func TestConfigGetRunE(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("GITLINK_CONFIG_DIR", dir)

	initCmd := newInitCmd()
	initCmd.RunE(initCmd, nil)

	cmd := newGetCmd()
	if err := cmd.RunE(cmd, []string{"base_url"}); err != nil {
		t.Fatalf("get RunE error: %v", err)
	}
}

func TestConfigGetRunENotSet(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("GITLINK_CONFIG_DIR", dir)
	os.WriteFile(dir+"/config.yaml", []byte("base_url: https://example.com\n"), 0644)

	cmd := newGetCmd()
	if err := cmd.RunE(cmd, []string{"editor"}); err != nil {
		t.Fatalf("get RunE not-set error: %v", err)
	}
}

func TestConfigListRunE(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("GITLINK_CONFIG_DIR", dir)

	initCmd := newInitCmd()
	initCmd.RunE(initCmd, nil)

	cmd := newListCmd()
	if err := cmd.RunE(cmd, nil); err != nil {
		t.Fatalf("list RunE error: %v", err)
	}
}

func TestConfigListRunENoConfig(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("GITLINK_CONFIG_DIR", dir)
	// Don't init — Load returns defaults for missing file, so this should work
	cmd := newListCmd()
	if err := cmd.RunE(cmd, nil); err != nil {
		t.Fatalf("list RunE error: %v", err)
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
