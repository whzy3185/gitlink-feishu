package cmd

import (
	"bytes"
	"strings"
	"testing"
)

func TestNewRootCmdDefaults(t *testing.T) {
	root, err := NewRootCmd(RootOptions{Version: "test"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if root.Use != "gitlink-cli" {
		t.Fatalf("Use = %q", root.Use)
	}
	if !root.SilenceUsage {
		t.Fatal("expected SilenceUsage=true")
	}
}

func TestRootHelp(t *testing.T) {
	root, err := NewRootCmd(RootOptions{Version: "test", Args: []string{"--help"}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	if err := root.Execute(); err != nil {
		t.Fatalf("help command error: %v", err)
	}
}

func TestVersionCmd(t *testing.T) {
	root, err := NewRootCmd(RootOptions{Version: "test", Args: []string{"version"}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	if err := root.Execute(); err != nil {
		t.Fatalf("version command error: %v", err)
	}
	if got := strings.TrimSpace(out.String()); got != "gitlink-cli test" {
		t.Fatalf("version output = %q", got)
	}
}

func TestRootCmdHasSubcommands(t *testing.T) {
	root, err := NewRootCmd(RootOptions{Version: "test"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	names := map[string]bool{}
	for _, sub := range root.Commands() {
		names[sub.Use] = true
	}
	for _, want := range []string{"auth", "config", "doctor", "version"} {
		if !names[want] {
			t.Fatalf("missing subcommand: %s", want)
		}
	}
	if len(root.Commands()) < 4 {
		t.Fatalf("expected at least 4 subcommands, got %d", len(root.Commands()))
	}
}
