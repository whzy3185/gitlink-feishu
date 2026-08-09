package cmd

import (
	"bytes"
	"strings"
	"testing"
)

func TestCompletionCmdGeneratesSupportedShells(t *testing.T) {
	cases := []struct {
		shell string
		want  string
	}{
		{shell: "bash", want: "__gitlink-cli"},
		{shell: "zsh", want: "#compdef gitlink-cli"},
		{shell: "fish", want: "complete -c gitlink-cli"},
		{shell: "powershell", want: "Register-ArgumentCompleter"},
	}

	for _, tc := range cases {
		root, err := NewRootCmd(RootOptions{Version: "test", Args: []string{"completion", tc.shell}}, nil)
		if err != nil {
			t.Fatal(err)
		}
		var out bytes.Buffer
		root.SetOut(&out)
		root.SetErr(&out)
		if err := root.Execute(); err != nil {
			t.Fatalf("%s completion error: %v", tc.shell, err)
		}
		if !strings.Contains(out.String(), tc.want) {
			t.Fatalf("%s completion missing %q, got:\n%s", tc.shell, tc.want, out.String()[:min(len(out.String()), 400)])
		}
	}
}

func TestCompletionCmdNoDescriptions(t *testing.T) {
	root, err := NewRootCmd(RootOptions{
		Version: "test",
		Args:    []string{"completion", "bash", "--no-descriptions"},
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
	if strings.Contains(out.String(), "GitLink CLI - command-line tool for GitLink") {
		t.Fatalf("expected descriptions to be omitted")
	}
}

func TestCompletionCmdRejectsUnsupportedShell(t *testing.T) {
	root, err := NewRootCmd(RootOptions{Version: "test", Args: []string{"completion", "xonsh"}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	err = root.Execute()
	if err == nil {
		t.Fatal("expected unsupported shell error")
	}
	if !strings.Contains(err.Error(), "invalid argument") {
		t.Fatalf("expected invalid argument error, got %q", err.Error())
	}
}
