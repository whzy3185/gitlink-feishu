package template

import (
	"testing"

	"github.com/gitlink-org/gitlink-cli/shortcuts/common"
)

func TestShortcutsReturnsAllCommands(t *testing.T) {
	shortcuts := Shortcuts()

	expectedCommands := []string{"list", "get", "create", "update", "delete"}
	if len(shortcuts) != len(expectedCommands) {
		t.Errorf("Expected %d shortcuts, got %d", len(expectedCommands), len(shortcuts))
	}

	for i, cmd := range expectedCommands {
		if shortcuts[i].Name != cmd {
			t.Errorf("Expected shortcut %q at index %d, got %q", cmd, i, shortcuts[i].Name)
		}
	}
}

func TestShortcutDescriptions(t *testing.T) {
	shortcuts := Shortcuts()

	for _, s := range shortcuts {
		if s.Description == "" {
			t.Errorf("Shortcut %q has empty description", s.Name)
		}
	}
}

func TestRequiredFlags(t *testing.T) {
	shortcuts := Shortcuts()

	// Test create command required flags
	createCmd := findShortcut(shortcuts, "create")
	if createCmd == nil {
		t.Fatal("create command not found")
	}
	assertRequiredFlags(t, createCmd, []string{"type", "name", "content"})

	// Test get command required flags
	getCmd := findShortcut(shortcuts, "get")
	if getCmd == nil {
		t.Fatal("get command not found")
	}
	assertRequiredFlags(t, getCmd, []string{"id"})

	// Test update command required flags
	updateCmd := findShortcut(shortcuts, "update")
	if updateCmd == nil {
		t.Fatal("update command not found")
	}
	assertRequiredFlags(t, updateCmd, []string{"id"})

	// Test delete command required flags
	deleteCmd := findShortcut(shortcuts, "delete")
	if deleteCmd == nil {
		t.Fatal("delete command not found")
	}
	assertRequiredFlags(t, deleteCmd, []string{"id"})
}

func findShortcut(shortcuts []*common.Shortcut, name string) *common.Shortcut {
	for _, s := range shortcuts {
		if s.Name == name {
			return s
		}
	}
	return nil
}

func assertRequiredFlags(t *testing.T, shortcut *common.Shortcut, expectedFlags []string) {
	t.Helper()
	for _, expectedFlag := range expectedFlags {
		found := false
		for _, flag := range shortcut.Flags {
			if flag.Name == expectedFlag && flag.Required {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Shortcut %q: expected required flag %q not found", shortcut.Name, expectedFlag)
		}
	}
}
