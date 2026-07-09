package status

import (
	"strings"
	"testing"
)

func TestNewStatusCmd(t *testing.T) {
	cmd := NewStatusCmd()
	if cmd.Use != "status" {
		t.Errorf("expected Use 'status', got %s", cmd.Use)
	}
	if cmd.Short == "" {
		t.Error("Short description should not be empty")
	}
	if cmd.Long == "" {
		t.Error("Long description should not be empty")
	}
}

func TestNewStatusCmdExample(t *testing.T) {
	cmd := NewStatusCmd()
	if !strings.Contains(cmd.Example, "status") {
		t.Errorf("Example should contain 'status', got: %s", cmd.Example)
	}
}

func TestNewStatusCmdHasNoSubcommands(t *testing.T) {
	cmd := NewStatusCmd()
	if cmd.HasSubCommands() {
		t.Error("status should not have subcommands")
	}
}

func TestTokenSourceFromEnv(t *testing.T) {
	t.Setenv("GITLINK_TOKEN", "test-token-123")
	result := tokenSource("test-token-123")
	if result != "环境变量 GITLINK_TOKEN" {
		t.Errorf("expected env source, got: %s", result)
	}
}

func TestTokenSourceFromKeyring(t *testing.T) {
	// 不设置环境变量，或用不同的值
	t.Setenv("GITLINK_TOKEN", "")
	result := tokenSource("some-stored-token")
	if result != "keyring / 配置文件" {
		t.Errorf("expected keyring source, got: %s", result)
	}
}

func TestTokenSourceMismatch(t *testing.T) {
	t.Setenv("GITLINK_TOKEN", "env-token")
	result := tokenSource("different-token")
	if result != "keyring / 配置文件" {
		t.Errorf("should fallback to keyring when token differs from env, got: %s", result)
	}
}
