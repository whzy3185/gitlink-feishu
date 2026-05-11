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
