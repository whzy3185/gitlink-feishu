package i18n

import (
	"fmt"
	"regexp"
	"sort"
)

var placeholderPattern = regexp.MustCompile(`\{([A-Za-z_][A-Za-z0-9_]*)\}`)

func renderTemplate(message string, args Args) string {
	if len(args) == 0 {
		return message
	}
	return placeholderPattern.ReplaceAllStringFunc(message, func(match string) string {
		name := match[1 : len(match)-1]
		value, ok := args[name]
		if !ok || value == nil {
			return match
		}
		if stringer, ok := value.(fmt.Stringer); ok {
			return stringer.String()
		}
		return fmt.Sprint(value)
	})
}

func extractTemplateArgs(message string) []string {
	matches := placeholderPattern.FindAllStringSubmatch(message, -1)
	seen := make(map[string]struct{}, len(matches))
	for _, match := range matches {
		seen[match[1]] = struct{}{}
	}

	args := make([]string, 0, len(seen))
	for arg := range seen {
		args = append(args, arg)
	}
	sort.Strings(args)
	return args
}
