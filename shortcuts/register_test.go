package shortcuts

import (
	"testing"

	"github.com/spf13/cobra"
)

func TestRegisterAll(t *testing.T) {
	root := &cobra.Command{Use: "gitlink-cli"}
	RegisterAll(root)

	expectedGroups := []string{
		"repo", "issue", "label", "license", "pr", "release", "branch",
		"org", "user", "search", "ci", "workflow",
		"compare", "member", "milestone", "pipeline", "webhook",
		"health", "attachment", "meta",
	}

	groupSet := map[string]bool{}
	for _, name := range expectedGroups {
		groupSet[name] = false
	}

	if len(root.Commands()) != len(expectedGroups) {
		t.Fatalf("expected %d group commands, got %d", len(expectedGroups), len(root.Commands()))
	}

	for _, cmd := range root.Commands() {
		if _, ok := groupSet[cmd.Use]; !ok {
			t.Fatalf("unexpected group command: %q", cmd.Use)
		}
		if groupSet[cmd.Use] {
			t.Fatalf("duplicate group command: %q", cmd.Use)
		}
		groupSet[cmd.Use] = true

		if cmd.Short == "" {
			t.Fatalf("group %q has empty Short description", cmd.Use)
		}

		if len(cmd.Commands()) == 0 {
			t.Fatalf("group %q has no shortcuts mounted", cmd.Use)
		}
	}

	for name, found := range groupSet {
		if !found {
			t.Fatalf("missing group command: %q", name)
		}
	}
}

func TestRegisterAllGroupDescriptions(t *testing.T) {
	root := &cobra.Command{Use: "gitlink-cli"}
	RegisterAll(root)

	for _, cmd := range root.Commands() {
		t.Run(cmd.Use, func(t *testing.T) {
			if cmd.Short == "" {
				t.Fatal("Short description is empty")
			}
		})
	}
}
