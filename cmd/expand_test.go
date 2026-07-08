package cmd

import (
	"reflect"
	"testing"

	"github.com/spf13/cobra"

	internalConfig "github.com/gitlink-org/gitlink-cli/internal/config"
)

func TestSplitArgs(t *testing.T) {
	cases := []struct {
		in   string
		want []string
	}{
		{"", nil},
		{"pr +view", []string{"pr", "+view"}},
		{"  issue   +list  ", []string{"issue", "+list"}},
		{`pr +view --title "needs review"`, []string{"pr", "+view", "--title", "needs review"}},
		{`repo +create --name 'my repo'`, []string{"repo", "+create", "--name", "my repo"}},
	}
	for _, tc := range cases {
		if got := splitArgs(tc.in); !reflect.DeepEqual(got, tc.want) {
			t.Fatalf("splitArgs(%q) = %#v, want %#v", tc.in, got, tc.want)
		}
	}
}

func TestExpandAlias(t *testing.T) {
	t.Setenv("GITLINK_CONFIG_DIR", t.TempDir())

	cfg := internalConfig.DefaultConfig()
	cfg.Aliases = map[string]string{
		"co":     "pr +view",
		"config": "should never expand",
	}
	if err := internalConfig.Save(cfg); err != nil {
		t.Fatalf("Save error: %v", err)
	}

	root := &cobra.Command{Use: "gitlink-cli"}
	root.AddCommand(&cobra.Command{Use: "config"})

	cases := []struct {
		name   string
		args   []string
		want   []string
		expand bool
	}{
		{"alias match", []string{"co", "42"}, []string{"pr", "+view", "42"}, true},
		{"builtin wins", []string{"config", "list"}, nil, false},
		{"flag first", []string{"--lang", "zh-CN"}, nil, false},
		{"unknown token", []string{"nope"}, nil, false},
		{"empty args", nil, nil, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := expandAlias(root, tc.args)
			if ok != tc.expand {
				t.Fatalf("expandAlias ok = %v, want %v", ok, tc.expand)
			}
			if tc.expand && !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("expandAlias = %#v, want %#v", got, tc.want)
			}
		})
	}
}
