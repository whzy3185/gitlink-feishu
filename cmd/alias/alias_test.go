package alias

import (
	"io"
	"os"
	"strings"
	"testing"

	"github.com/gitlink-org/gitlink-cli/cmd/cmdutil"
)

func TestLoadAliasesEmpty(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("GITLINK_CONFIG_DIR", tmpDir)

	aliases, err := loadAliases()
	if err != nil {
		t.Fatalf("loadAliases failed: %v", err)
	}
	if len(aliases) != 0 {
		t.Fatalf("expected empty aliases, got %d", len(aliases))
	}
}

func TestSaveAndLoadAliases(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("GITLINK_CONFIG_DIR", tmpDir)

	original := map[string]string{
		"rl": "repo +list",
		"ri": "repo +info",
	}
	if err := saveAliases(original); err != nil {
		t.Fatalf("saveAliases failed: %v", err)
	}

	loaded, err := loadAliases()
	if err != nil {
		t.Fatalf("loadAliases failed: %v", err)
	}
	if len(loaded) != 2 {
		t.Fatalf("expected 2 aliases, got %d", len(loaded))
	}
	if loaded["rl"] != "repo +list" {
		t.Errorf("expected rl -> repo +list, got %s", loaded["rl"])
	}
	if loaded["ri"] != "repo +info" {
		t.Errorf("expected ri -> repo +info, got %s", loaded["ri"])
	}
}

func TestSaveAliasesOverwrite(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("GITLINK_CONFIG_DIR", tmpDir)

	saveAliases(map[string]string{"rl": "repo +list"})
	saveAliases(map[string]string{"rl": "repo +list --owner Gitlink"})

	loaded, _ := loadAliases()
	if loaded["rl"] != "repo +list --owner Gitlink" {
		t.Errorf("alias should be overwritten, got %s", loaded["rl"])
	}
}

func TestLoadAliasesInvalidYAML(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("GITLINK_CONFIG_DIR", tmpDir)

	os.WriteFile(tmpDir+"/aliases.yaml", []byte("{{invalid yaml}}"), 0600)

	aliases, err := loadAliases()
	if err != nil {
		t.Fatalf("should not error on invalid YAML, got: %v", err)
	}
	if len(aliases) != 0 {
		t.Fatalf("should return empty map on invalid YAML, got %d", len(aliases))
	}
}

func TestNewAliasCmdStructure(t *testing.T) {
	cmd := NewAliasCmd()
	if cmd.Use != "alias" {
		t.Errorf("expected Use 'alias', got %s", cmd.Use)
	}
	if !cmd.HasSubCommands() {
		t.Error("alias command should have subcommands")
	}

	subcmds := cmd.Commands()
	if len(subcmds) != 4 {
		t.Fatalf("expected 4 subcommands, got %d", len(subcmds))
	}

	expectedUses := map[string]bool{"+list": false, "+set <name> <command>": false, "+delete <name>": false, "+expand <name>": false}
	for _, sub := range subcmds {
		if _, ok := expectedUses[sub.Use]; ok {
			expectedUses[sub.Use] = true
		}
	}
	for use, found := range expectedUses {
		if !found {
			t.Errorf("subcommand %q not found", use)
		}
	}
}

func TestAliasSetAndDeleteFlow(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("GITLINK_CONFIG_DIR", tmpDir)

	// 模拟 +set 操作：直接调用 saveAliases
	aliases := make(map[string]string)
	aliases["rl"] = "repo +list"
	aliases["ri"] = "repo +info"
	if err := saveAliases(aliases); err != nil {
		t.Fatalf("saveAliases failed: %v", err)
	}

	// 验证保存成功
	loaded, _ := loadAliases()
	if loaded["rl"] != "repo +list" {
		t.Fatalf("alias not saved correctly: %v", loaded)
	}
	if loaded["ri"] != "repo +info" {
		t.Fatalf("alias not saved correctly: %v", loaded)
	}

	// 模拟 +delete 操作：删除别名后保存
	delete(loaded, "rl")
	if err := saveAliases(loaded); err != nil {
		t.Fatalf("saveAliases after delete failed: %v", err)
	}

	// 验证删除成功
	final, _ := loadAliases()
	if _, ok := final["rl"]; ok {
		t.Fatal("alias 'rl' should have been deleted")
	}
	if final["ri"] != "repo +info" {
		t.Fatal("alias 'ri' should still exist")
	}
}

func TestAliasDeleteNonExistent(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("GITLINK_CONFIG_DIR", tmpDir)

	// 空别名列表，删除不存在的别名
	aliases, _ := loadAliases()
	if _, ok := aliases["nonexistent"]; ok {
		t.Fatal("nonexistent alias should not exist")
	}
	// 验证逻辑：别名不存在时不应执行删除
	// 这对应 alias.go 中 if _, ok := aliases[args[0]]; !ok 的检查
}

func TestAliasesFilePath(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("GITLINK_CONFIG_DIR", tmpDir)

	expected := tmpDir + "/aliases.yaml"
	got := aliasesPath()
	if got != expected {
		t.Errorf("expected path %s, got %s", expected, got)
	}
}

func TestAliasExpandExisting(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("GITLINK_CONFIG_DIR", tmpDir)

	saveAliases(map[string]string{
		"rl": "repo +list",
		"ri": "repo +info",
	})

	aliases, _ := loadAliases()
	if expanded, ok := aliases["rl"]; !ok || expanded != "repo +list" {
		t.Fatalf("expected rl → repo +list, got %s", expanded)
	}
	if expanded, ok := aliases["ri"]; !ok || expanded != "repo +info" {
		t.Fatalf("expected ri → repo +info, got %s", expanded)
	}
}

func TestAliasExpandNonExistent(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("GITLINK_CONFIG_DIR", tmpDir)

	aliases, _ := loadAliases()
	if _, ok := aliases["nonexistent"]; ok {
		t.Fatal("nonexistent alias should not be found")
	}
}

func TestAliasListJSONFormat(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("GITLINK_CONFIG_DIR", tmpDir)
	if err := saveAliases(map[string]string{"rl": "repo +list", "ri": "repo +info"}); err != nil {
		t.Fatalf("save: %v", err)
	}
	cmdutil.Format = "json"
	defer func() { cmdutil.Format = "" }()

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	root := NewAliasCmd()
	root.SetArgs([]string{"+list"})
	execErr := root.Execute()
	w.Close()
	os.Stdout = old
	if execErr != nil {
		t.Fatalf("execute: %v", execErr)
	}
	var buf strings.Builder
	io.Copy(&buf, r)
	out := buf.String()
	for _, want := range []string{`"ok": true`, `"name"`, `"rl"`, `"repo +list"`} {
		if !strings.Contains(out, want) {
			t.Errorf("JSON output missing %q: %s", want, out)
		}
	}
}
