package cmd

import (
	"testing"
)

func TestInit(t *testing.T) {
	// init() runs automatically when package is imported.
	// Verify rootCmd has the expected settings.
	if rootCmd.Use != "gitlink-cli" {
		t.Fatalf("Use = %q", rootCmd.Use)
	}
	if rootCmd.SilenceUsage != true {
		t.Fatal("expected SilenceUsage=true")
	}
}

func TestExecute(t *testing.T) {
	rootCmd.SetArgs([]string{"--help"})
	if err := Execute(); err != nil {
		t.Fatalf("Execute error: %v", err)
	}
}

func TestVersionCmd(t *testing.T) {
	rootCmd.SetArgs([]string{"version"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("version command error: %v", err)
	}
}

func TestRootCmdHasSubcommands(t *testing.T) {
	names := map[string]bool{}
	for _, sub := range rootCmd.Commands() {
		names[sub.Use] = true
	}
	// At minimum, these core commands should exist
	for _, want := range []string{"auth", "config", "version"} {
		if !names[want] {
			t.Fatalf("missing subcommand: %s", want)
		}
	}
	if len(rootCmd.Commands()) < 4 {
		t.Fatalf("expected at least 4 subcommands, got %d", len(rootCmd.Commands()))
	}
}
