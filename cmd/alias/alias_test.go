package alias

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestLoadAliasesEmpty(t *testing.T) {
	// 设置临时配置目录
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

	// 保存
	original := map[string]string{
		"rl": "repo +list",
		"ri": "repo +info",
	}
	if err := saveAliases(original); err != nil {
		t.Fatalf("saveAliases failed: %v", err)
	}

	// 加载
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

	// 第一次保存
	saveAliases(map[string]string{"rl": "repo +list"})

	// 覆盖保存
	saveAliases(map[string]string{"rl": "repo +list --owner Gitlink"})

	loaded, _ := loadAliases()
	if loaded["rl"] != "repo +list --owner Gitlink" {
		t.Errorf("alias should be overwritten, got %s", loaded["rl"])
	}
}

func TestLoadAliasesInvalidYAML(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("GITLINK_CONFIG_DIR", tmpDir)

	// 写入无效 YAML
	os.WriteFile(tmpDir+"/aliases.yaml", []byte("{{invalid yaml}}"), 0600)

	aliases, err := loadAliases()
	if err != nil {
		t.Fatalf("should not error on invalid YAML, got: %v", err)
	}
	if len(aliases) != 0 {
		t.Fatalf("should return empty map on invalid YAML, got %d", len(aliases))
	}
}

func TestNewAliasCmd(t *testing.T) {
	cmd := NewAliasCmd()
	if cmd.Use != "alias" {
		t.Errorf("expected Use 'alias', got %s", cmd.Use)
	}
	if !cmd.HasSubCommands() {
		t.Error("alias command should have subcommands")
	}
	subcmds := cmd.Commands()
	if len(subcmds) != 3 {
		t.Fatalf("expected 3 subcommands, got %d", len(subcmds))
	}
}

func TestAliasListSubcommand(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("GITLINK_CONFIG_DIR", tmpDir)

	cmd := NewAliasCmd()
	// 找到 +list 子命令
	var listCmd *cobra.Command
	for _, sub := range cmd.Commands() {
		if sub.Use == "+list" {
			listCmd = sub
			break
		}
	}
	if listCmd == nil {
		t.Fatal("+list subcommand not found")
	}

	// 无别名时运行
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetArgs([]string{"+list"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("list failed: %v", err)
	}
	if !strings.Contains(buf.String(), "未定义任何别名") {
		t.Errorf("expected hint for no aliases, got: %s", buf.String())
	}
}

func TestAliasSetAndDeleteSubcommands(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("GITLINK_CONFIG_DIR", tmpDir)

	cmd := NewAliasCmd()

	// 找到 +set 子命令
	var setCmd, deleteCmd *cobra.Command
	for _, sub := range cmd.Commands() {
		if strings.HasPrefix(sub.Use, "+set") {
			setCmd = sub
		}
		if strings.HasPrefix(sub.Use, "+delete") {
			deleteCmd = sub
		}
	}
	if setCmd == nil || deleteCmd == nil {
		t.Fatal("alias set/delete subcommands not found")
	}

	// +set
	cmd.SetArgs([]string{"+set", "rl", "repo +list"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("set failed: %v", err)
	}

	// 验证文件写入
	aliases, _ := loadAliases()
	if aliases["rl"] != "repo +list" {
		t.Fatalf("alias not saved correctly: %v", aliases)
	}

	// +delete
	cmd.SetArgs([]string{"+delete", "rl"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("delete failed: %v", err)
	}

	// 验证已删除
	aliases, _ = loadAliases()
	if _, ok := aliases["rl"]; ok {
		t.Fatal("alias should have been deleted")
	}
}

func TestAliasDeleteNonExistent(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("GITLINK_CONFIG_DIR", tmpDir)

	cmd := NewAliasCmd()
	var deleteCmd *cobra.Command
	for _, sub := range cmd.Commands() {
		if strings.HasPrefix(sub.Use, "+delete") {
			deleteCmd = sub
			break
		}
	}
	if deleteCmd == nil {
		t.Fatal("alias delete subcommand not found")
	}

	cmd.SetArgs([]string{"+delete", "nonexistent"})
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error when deleting nonexistent alias")
	}
	if !strings.Contains(err.Error(), "不存在") {
		t.Errorf("error should mention alias does not exist: %v", err)
	}
}
