package common

import (
	"testing"

	"github.com/spf13/cobra"
)

func TestMountShortcutSupportsBoolFlags(t *testing.T) {
	var got string
	root := &cobra.Command{Use: "root"}
	MountShortcut(root, &Shortcut{
		Name:        "preview",
		Description: "preview command",
		Flags: []Flag{
			{Name: "dry-run", Usage: "preview only", Bool: true, Default: "false"},
		},
		Run: func(ctx *RuntimeContext) error {
			got = ctx.Arg("dry-run")
			return nil
		},
	})

	root.SetArgs([]string{"+preview", "--dry-run"})
	if err := root.Execute(); err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	if got != "true" {
		t.Fatalf("dry-run flag = %q, want true", got)
	}
}

func TestMountShortcuts(t *testing.T) {
	var called []string
	root := &cobra.Command{Use: "root"}
	shortcuts := []*Shortcut{
		{
			Name: "first",
			Flags: []Flag{
				{Name: "name", Default: "default1"},
			},
			Run: func(ctx *RuntimeContext) error {
				called = append(called, "first")
				return nil
			},
		},
		{
			Name: "second",
			Flags: []Flag{
				{Name: "name", Default: "default2"},
			},
			Run: func(ctx *RuntimeContext) error {
				called = append(called, "second")
				return nil
			},
		},
	}
	MountShortcuts(root, shortcuts)

	if len(root.Commands()) != 2 {
		t.Fatalf("expected 2 subcommands, got %d", len(root.Commands()))
	}

	root.SetArgs([]string{"+first"})
	if err := root.Execute(); err != nil {
		t.Fatalf("Execute +first: %v", err)
	}
	if len(called) != 1 || called[0] != "first" {
		t.Fatalf("called = %v, want [first]", called)
	}

	root.SetArgs([]string{"+second"})
	if err := root.Execute(); err != nil {
		t.Fatalf("Execute +second: %v", err)
	}
	if len(called) != 2 || called[1] != "second" {
		t.Fatalf("called = %v, want [first second]", called)
	}
}
