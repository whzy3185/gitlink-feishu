package i18n

import (
	"fmt"
	"reflect"
	"regexp"
	"sort"
	"strings"
)

var keyPattern = regexp.MustCompile(`^(cmd|flag|error|prompt|success|warning|confirm|table|output)\.[a-z0-9_.-]+$`)

// Problem describes a locale validation issue.
type Problem struct {
	Locale  string
	Key     string
	Message string
}

func (p Problem) String() string {
	if p.Key == "" {
		return fmt.Sprintf("%s: %s", p.Locale, p.Message)
	}
	return fmt.Sprintf("%s:%s: %s", p.Locale, p.Key, p.Message)
}

// Validate checks all locales against baseLocale.
func Validate(loader Loader, baseLocale string) ([]Problem, error) {
	if loader == nil {
		loader = NewEmbedLoader()
	}
	baseLocale = NormalizeLocale(baseLocale)
	if baseLocale == "" {
		baseLocale = defaultFallbackLocale
	}

	locales, err := loader.AvailableLocales()
	if err != nil {
		return nil, err
	}
	sort.Strings(locales)

	allMessages := make(map[string]map[string]string, len(locales))
	for _, locale := range locales {
		normalized := NormalizeLocale(locale)
		if normalized != locale {
			return []Problem{{Locale: locale, Message: "locale filename is not normalized"}}, nil
		}
		messages, err := loader.Load(locale)
		if err != nil {
			return nil, err
		}
		allMessages[locale] = messages
	}

	base, ok := allMessages[baseLocale]
	if !ok {
		return []Problem{{Locale: baseLocale, Message: "base locale is missing"}}, nil
	}

	var problems []Problem
	for key, value := range base {
		problems = append(problems, validateMessage(baseLocale, key, value)...)
	}

	for _, locale := range locales {
		messages := allMessages[locale]
		for key, value := range messages {
			problems = append(problems, validateMessage(locale, key, value)...)
			if _, ok := base[key]; !ok {
				problems = append(problems, Problem{Locale: locale, Key: key, Message: "key is not present in base locale"})
			}
		}
		for key, baseValue := range base {
			value, ok := messages[key]
			if !ok {
				problems = append(problems, Problem{Locale: locale, Key: key, Message: "missing key"})
				continue
			}
			baseArgs := extractTemplateArgs(baseValue)
			args := extractTemplateArgs(value)
			if !reflect.DeepEqual(baseArgs, args) {
				problems = append(problems, Problem{
					Locale:  locale,
					Key:     key,
					Message: fmt.Sprintf("template args mismatch: expected {%s}, got {%s}", strings.Join(baseArgs, ","), strings.Join(args, ",")),
				})
			}
		}
	}

	return problems, nil
}

func validateMessage(locale, key, value string) []Problem {
	var problems []Problem
	if !keyPattern.MatchString(key) {
		problems = append(problems, Problem{Locale: locale, Key: key, Message: "key does not match naming rules"})
	}
	if strings.TrimSpace(value) == "" {
		problems = append(problems, Problem{Locale: locale, Key: key, Message: "message is empty"})
	}
	return problems
}
