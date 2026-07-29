package shortcuts

import (
	"testing"

	"github.com/spf13/cobra"
)

func TestRegisterAll(t *testing.T) {
	root := &cobra.Command{Use: "gitlink-cli"}
	RegisterAll(root)

	requiredGroups := []string{
		"repo", "issue", "label", "pr", "release", "branch",
		"org", "user", "search", "ci", "workflow", "feishu",
		"compare", "member", "milestone", "notification", "pipeline",
		"pm", "export", "webhook", "wiki",
	}

	groupSet := make(map[string]bool, len(root.Commands()))
	for _, cmd := range root.Commands() {
		if groupSet[cmd.Use] {
			t.Fatalf("duplicate group command: %q", cmd.Use)
		}
		groupSet[cmd.Use] = true
	}

	for _, name := range requiredGroups {
		if !groupSet[name] {
			t.Errorf("missing required group command: %q", name)
		}
	}

	for _, cmd := range root.Commands() {
		if cmd.Short == "" {
			t.Fatalf("group %q has empty Short description", cmd.Use)
		}

		if len(cmd.Commands()) == 0 {
			t.Fatalf("group %q has no shortcuts mounted", cmd.Use)
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
