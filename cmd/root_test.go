package cmd

import (
	"bytes"
	"strings"
	"testing"

	"github.com/gitlink-org/gitlink-cli/internal/i18n"
)

func TestRootHelpUsesSelectedLocale(t *testing.T) {
	tr, err := i18n.New(i18n.Options{Locale: "zh-CN"})
	if err != nil {
		t.Fatal(err)
	}
	root, err := NewRootCmd(RootOptions{Version: "test", Args: []string{"--help"}}, tr)
	if err != nil {
		t.Fatal(err)
	}

	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}

	help := out.String()
	if !strings.Contains(help, "用于管理 GitLink 上的仓库") {
		t.Fatalf("expected Chinese root long help, got:\n%s", help)
	}
	if !strings.Contains(help, "仓库操作") {
		t.Fatalf("expected Chinese shortcut group help, got:\n%s", help)
	}
	if !strings.Contains(help, "认证命令") || !strings.Contains(help, "管理 gitlink-cli 配置") {
		t.Fatalf("expected Chinese core command help, got:\n%s", help)
	}
	if !strings.Contains(help, "--lang") || !strings.Contains(help, "显示语言") {
		t.Fatalf("expected localized lang flag help, got:\n%s", help)
	}
}

func TestRootHelpUsesExplicitLang(t *testing.T) {
	root, err := NewRootCmd(RootOptions{Version: "test", Args: []string{"--lang", "zh-CN", "--help"}, Env: map[string]string{}}, nil)
	if err != nil {
		t.Fatal(err)
	}

	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}

	help := out.String()
	for _, want := range []string{"用于管理 GitLink", "显示语言", "仓库"} {
		if !strings.Contains(help, want) {
			t.Fatalf("expected %q in help, got:\n%s", want, help)
		}
	}
}

func TestRootHelpUsesEnvLang(t *testing.T) {
	root, err := NewRootCmd(RootOptions{
		Version: "test",
		Args:    []string{"repo", "--help"},
		Env:     map[string]string{"GITLINK_LANG": "zh-CN"},
	}, nil)
	if err != nil {
		t.Fatal(err)
	}

	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}

	help := out.String()
	for _, want := range []string{"仓库操作", "仓库所有者", "仓库名称"} {
		if !strings.Contains(help, want) {
			t.Fatalf("expected %q in help, got:\n%s", want, help)
		}
	}
}

func TestExplicitLangOverridesConfigLang(t *testing.T) {
	root, err := NewRootCmd(RootOptions{
		Version:    "test",
		Args:       []string{"--lang", "en-US", "--help"},
		Env:        map[string]string{},
		ConfigLang: "zh-CN",
	}, nil)
	if err != nil {
		t.Fatal(err)
	}

	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}

	help := out.String()
	if !strings.Contains(help, "Repository operations") {
		t.Fatalf("expected English help, got:\n%s", help)
	}
	if strings.Contains(help, "仓库操作") {
		t.Fatalf("expected explicit en-US to override config zh-CN, got:\n%s", help)
	}
}

func TestUnsupportedExplicitLangReturnsError(t *testing.T) {
	_, err := NewRootCmd(RootOptions{
		Version: "test",
		Args:    []string{"--lang", "fr-FR", "--help"},
		Env:     map[string]string{},
	}, nil)
	if err == nil {
		t.Fatal("expected unsupported language error")
	}
	if !strings.Contains(err.Error(), "unsupported language") {
		t.Fatalf("expected unsupported language error, got %q", err.Error())
	}
}

func TestRequireArgUsesLocalizedError(t *testing.T) {
	root, err := NewRootCmd(RootOptions{
		Version: "test",
		Args:    []string{"--lang", "zh-CN", "repo", "+create"},
		Env:     map[string]string{},
	}, nil)
	if err != nil {
		t.Fatal(err)
	}

	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	err = root.Execute()
	if err == nil {
		t.Fatal("expected missing required flag error")
	}
	if !strings.Contains(err.Error(), "缺少必需参数") {
		t.Fatalf("expected localized missing flag error, got %q", err.Error())
	}
}

func TestCoreCommandHelpUsesSelectedLocale(t *testing.T) {
	tr, err := i18n.New(i18n.Options{Locale: "zh-CN"})
	if err != nil {
		t.Fatal(err)
	}

	cases := []struct {
		args []string
		want []string
	}{
		{
			args: []string{"api", "--help"},
			want: []string{"向 GitLink API 发送任意 HTTP 请求", "--body", "请求体（JSON 字符串）"},
		},
		{
			args: []string{"auth", "login", "--help"},
			want: []string{"登录 GitLink", "--token", "通过粘贴已有 Token 登录"},
		},
		{
			args: []string{"config", "--help"},
			want: []string{"管理 gitlink-cli 配置", "初始化配置文件", "列出所有配置项"},
		},
	}

	for _, tc := range cases {
		root, err := NewRootCmd(RootOptions{Version: "test", Args: tc.args}, tr)
		if err != nil {
			t.Fatal(err)
		}

		var out bytes.Buffer
		root.SetOut(&out)
		root.SetErr(&out)
		if err := root.Execute(); err != nil {
			t.Fatalf("%v: %v", tc.args, err)
		}

		help := out.String()
		for _, want := range tc.want {
			if !strings.Contains(help, want) {
				t.Fatalf("%v: expected %q in help, got:\n%s", tc.args, want, help)
			}
		}
	}
}

func TestShortcutHelpUsesSelectedLocale(t *testing.T) {
	tr, err := i18n.New(i18n.Options{Locale: "zh-CN"})
	if err != nil {
		t.Fatal(err)
	}

	cases := []struct {
		args []string
		want []string
	}{
		{
			args: []string{"repo", "+create", "--help"},
			want: []string{"创建新仓库", "--name", "仓库名称", "--private", "设为私有仓库"},
		},
		{
			args: []string{"pr", "+review", "--help"},
			want: []string{"创建拉取请求评审", "--content", "评审内容", "--dry-run"},
		},
	}

	for _, tc := range cases {
		root, err := NewRootCmd(RootOptions{Version: "test", Args: tc.args}, tr)
		if err != nil {
			t.Fatal(err)
		}

		var out bytes.Buffer
		root.SetOut(&out)
		root.SetErr(&out)
		if err := root.Execute(); err != nil {
			t.Fatalf("%v: %v", tc.args, err)
		}

		help := out.String()
		for _, want := range tc.want {
			if !strings.Contains(help, want) {
				t.Fatalf("%v: expected %q in help, got:\n%s", tc.args, want, help)
			}
		}
	}
}

func TestRemainingShortcutHelpUsesSelectedLocale(t *testing.T) {
	tr, err := i18n.New(i18n.Options{Locale: "zh-CN"})
	if err != nil {
		t.Fatal(err)
	}

	cases := []struct {
		args []string
		want []string
	}{
		{
			args: []string{"branch", "+create", "--help"},
			want: []string{"创建分支", "--from", "源分支或 Commit"},
		},
		{
			args: []string{"release", "+create", "--help"},
			want: []string{"创建发布", "--prerelease", "标记为预发布"},
		},
		{
			args: []string{"webhook", "+create", "--help"},
			want: []string{"创建仓库 Webhook", "--events", "逗号分隔的事件"},
		},
		{
			args: []string{"ci", "+logs", "--help"},
			want: []string{"查看构建日志", "--build", "构建编号"},
		},
	}

	for _, tc := range cases {
		root, err := NewRootCmd(RootOptions{Version: "test", Args: tc.args}, tr)
		if err != nil {
			t.Fatal(err)
		}

		var out bytes.Buffer
		root.SetOut(&out)
		root.SetErr(&out)
		if err := root.Execute(); err != nil {
			t.Fatalf("%v: %v", tc.args, err)
		}

		help := out.String()
		for _, want := range tc.want {
			if !strings.Contains(help, want) {
				t.Fatalf("%v: expected %q in help, got:\n%s", tc.args, want, help)
			}
		}
	}
}

func TestVersionUsesInjectedVersion(t *testing.T) {
	tr, err := i18n.New(i18n.Options{Locale: "en-US"})
	if err != nil {
		t.Fatal(err)
	}
	root, err := NewRootCmd(RootOptions{Version: "1.2.3", Args: []string{"version"}}, tr)
	if err != nil {
		t.Fatal(err)
	}

	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}

	if got := strings.TrimSpace(out.String()); got != "gitlink-cli 1.2.3" {
		t.Fatalf("version output = %q", got)
	}
}
