package cmd

import (
	"strings"

	"github.com/spf13/cobra"

	internalConfig "github.com/gitlink-org/gitlink-cli/internal/config"
)

// expandAlias rewrites args when the first positional token names a saved alias.
// Built-in commands always take precedence, so an alias can never shadow a real
// command; a colliding alias simply never expands.
func expandAlias(root *cobra.Command, args []string) ([]string, bool) {
	if len(args) == 0 {
		return args, false
	}
	name := args[0]
	if name == "" || strings.HasPrefix(name, "-") {
		return args, false
	}
	if isBuiltinCommand(root, name) {
		return args, false
	}

	cfg, err := internalConfig.Load()
	if err != nil {
		return args, false
	}
	expansion, ok := cfg.Aliases[name]
	if !ok {
		return args, false
	}
	parts := splitArgs(expansion)
	if len(parts) == 0 {
		return args, false
	}
	return append(parts, args[1:]...), true
}

func isBuiltinCommand(root *cobra.Command, name string) bool {
	for _, c := range root.Commands() {
		if c.Name() == name {
			return true
		}
		for _, alias := range c.Aliases {
			if alias == name {
				return true
			}
		}
	}
	return false
}

// splitArgs breaks an alias expansion into argv tokens, honoring single and
// double quotes so expansions can carry multi-word flag values.
func splitArgs(s string) []string {
	var args []string
	var buf strings.Builder
	inWord := false
	var quote rune

	for _, r := range s {
		switch {
		case quote != 0:
			if r == quote {
				quote = 0
			} else {
				buf.WriteRune(r)
			}
			inWord = true
		case r == '\'' || r == '"':
			quote = r
			inWord = true
		case r == ' ' || r == '\t' || r == '\n':
			if inWord {
				args = append(args, buf.String())
				buf.Reset()
				inWord = false
			}
		default:
			buf.WriteRune(r)
			inWord = true
		}
	}
	if inWord {
		args = append(args, buf.String())
	}
	return args
}
