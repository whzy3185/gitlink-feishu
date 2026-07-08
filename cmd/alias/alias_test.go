package alias

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	internalConfig "github.com/gitlink-org/gitlink-cli/internal/config"
)

func tempConfigDir(t *testing.T) {
	t.Helper()
	t.Setenv("GITLINK_CONFIG_DIR", t.TempDir())
}

func run(t *testing.T, cmd *cobra.Command, args ...string) string {
	t.Helper()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs(args)
	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute %v: %v", args, err)
	}
	return out.String()
}

func TestNewAliasCmd(t *testing.T) {
	cmd := NewAliasCmd()
	if cmd.Use != "alias" {
		t.Fatalf("Use = %q, want alias", cmd.Use)
	}
	expected := map[string]bool{
		"set <name> <expansion>...": false,
		"list":                      false,
		"delete <name>":             false,
		"import":                    false,
	}
	for _, sub := range cmd.Commands() {
		if _, ok := expected[sub.Use]; !ok {
			t.Fatalf("unexpected subcommand: %q", sub.Use)
		}
		expected[sub.Use] = true
		if sub.Short == "" {
			t.Fatalf("subcommand %q has empty Short", sub.Use)
		}
	}
	for name, found := range expected {
		if !found {
			t.Fatalf("missing subcommand: %q", name)
		}
	}
}

func TestAliasSetListDelete(t *testing.T) {
	tempConfigDir(t)

	sets := []struct {
		args []string
		want string
	}{
		{[]string{"co", "pr", "+view"}, "pr +view"},
		{[]string{"bugs", "issue", "+list", "--label", "bug"}, "issue +list --label bug"},
	}
	for _, s := range sets {
		run(t, newSetCmd(), s.args...)
	}

	// set persists the joined expansion
	cfg, err := internalConfig.Load()
	if err != nil {
		t.Fatalf("Load error: %v", err)
	}
	for _, s := range sets {
		if cfg.Aliases[s.args[0]] != s.want {
			t.Fatalf("Aliases[%q] = %q, want %q", s.args[0], cfg.Aliases[s.args[0]], s.want)
		}
	}

	// list shows both
	listOut := run(t, newListCmd())
	for _, s := range sets {
		if !strings.Contains(listOut, s.args[0]+": "+s.want) {
			t.Fatalf("list missing %q, got:\n%s", s.args[0], listOut)
		}
	}

	// delete removes one, keeps the other
	run(t, newDeleteCmd(), "co")
	listOut = run(t, newListCmd())
	if strings.Contains(listOut, "co: ") {
		t.Fatalf("expected co deleted, got:\n%s", listOut)
	}
	if !strings.Contains(listOut, "bugs: ") {
		t.Fatalf("expected bugs retained, got:\n%s", listOut)
	}

	// deleting a missing alias errors
	del := newDeleteCmd()
	del.SetOut(&bytes.Buffer{})
	del.SetErr(&bytes.Buffer{})
	del.SetArgs([]string{"nope"})
	if err := del.Execute(); err == nil {
		t.Fatal("expected error deleting missing alias")
	}
}

func TestAliasSetRejectsEmptyExpansion(t *testing.T) {
	tempConfigDir(t)
	cmd := newSetCmd()
	// MinimumNArgs(2) is satisfied, but a whitespace-only expansion is rejected.
	if err := cmd.RunE(cmd, []string{"x", "   "}); err == nil {
		t.Fatal("expected error for empty expansion")
	}
}

func TestAliasListEmpty(t *testing.T) {
	tempConfigDir(t)
	out := run(t, newListCmd())
	if strings.TrimSpace(out) == "" {
		t.Fatal("expected a message for empty alias list")
	}
}

func TestAliasImportFile(t *testing.T) {
	tempConfigDir(t)

	dir := t.TempDir()
	path := filepath.Join(dir, "aliases.yaml")
	content := "co: pr +view\nbugs: issue +list --label bug\n"
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatalf("write error: %v", err)
	}

	cmd := newImportCmd()
	run(t, cmd, "--file", path)

	cfg, err := internalConfig.Load()
	if err != nil {
		t.Fatalf("Load error: %v", err)
	}
	if cfg.Aliases["co"] != "pr +view" || cfg.Aliases["bugs"] != "issue +list --label bug" {
		t.Fatalf("imported aliases = %#v", cfg.Aliases)
	}
}

func TestAliasImportStdin(t *testing.T) {
	tempConfigDir(t)

	cmd := newImportCmd()
	cmd.SetIn(strings.NewReader("ci: ci +status\n"))
	run(t, cmd)

	cfg, err := internalConfig.Load()
	if err != nil {
		t.Fatalf("Load error: %v", err)
	}
	if cfg.Aliases["ci"] != "ci +status" {
		t.Fatalf("imported alias = %q", cfg.Aliases["ci"])
	}
}

func TestAliasImportInvalidYAML(t *testing.T) {
	tempConfigDir(t)
	cmd := newImportCmd()
	cmd.SetIn(strings.NewReader("::: not yaml :::"))
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs(nil)
	if err := cmd.Execute(); err == nil {
		t.Fatal("expected error for invalid YAML")
	}
}
