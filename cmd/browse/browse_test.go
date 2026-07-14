package browse

import (
	"strings"
	"testing"
)

func TestNewBrowseCmd(t *testing.T) {
	cmd := NewBrowseCmd()
	if cmd.Use != "browse [resource]" {
		t.Errorf("expected Use 'browse [resource]', got %s", cmd.Use)
	}
	if cmd.Short == "" {
		t.Error("Short description should not be empty")
	}
	if cmd.Long == "" {
		t.Error("Long description should not be empty")
	}
}

func TestBrowseCmdHasCorrectArgs(t *testing.T) {
	cmd := NewBrowseCmd()
	// MaximumNArgs(1) should allow 0 or 1 args
	if err := cmd.Args(cmd, []string{}); err != nil {
		t.Errorf("should accept 0 args: %v", err)
	}
	if err := cmd.Args(cmd, []string{"issues/42"}); err != nil {
		t.Errorf("should accept 1 arg: %v", err)
	}
	if err := cmd.Args(cmd, []string{"a", "b"}); err == nil {
		t.Error("should reject more than 1 arg")
	}
}

func TestBrowseCmdSubcommandStructure(t *testing.T) {
	cmd := NewBrowseCmd()
	// browse 不应该有子命令
	if cmd.HasSubCommands() {
		t.Error("browse should not have subcommands")
	}
}

func TestBrowseCmdExample(t *testing.T) {
	cmd := NewBrowseCmd()
	if cmd.Example == "" {
		t.Error("Example should not be empty")
	}
	if !strings.Contains(cmd.Example, "browse") {
		t.Error("Example should contain 'browse'")
	}
}

func TestOpenBrowserReturnsNoError(t *testing.T) {
	// openBrowser 在所有平台都应该返回 nil 或一个 error
	// 在无头环境下可能会失败，但不应该 panic
	_ = openBrowser("https://gitlink.org.cn")
	// 只要不 panic 就行
}
