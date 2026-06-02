package i18n

import (
	"strings"
)

// NormalizeLocale converts common locale spellings to a stable BCP-47-like form.
func NormalizeLocale(locale string) string {
	locale = strings.TrimSpace(locale)
	if locale == "" {
		return ""
	}
	if idx := strings.IndexByte(locale, '.'); idx >= 0 {
		locale = locale[:idx]
	}
	locale = strings.ReplaceAll(locale, "_", "-")

	parts := strings.Split(locale, "-")
	normalized := make([]string, 0, len(parts))
	for i, part := range parts {
		if part == "" {
			continue
		}
		switch {
		case i == 0:
			normalized = append(normalized, strings.ToLower(part))
		case len(part) == 2:
			normalized = append(normalized, strings.ToUpper(part))
		case len(part) == 4:
			normalized = append(normalized, strings.ToUpper(part[:1])+strings.ToLower(part[1:]))
		default:
			normalized = append(normalized, part)
		}
	}
	return strings.Join(normalized, "-")
}

// MatchLocale resolves requested to one of available using exact, safe alias,
// then fallback matching.
func MatchLocale(requested string, available []string, fallback string) string {
	return matchLocale(requested, available, fallback).Locale
}

type localeMatch struct {
	Locale     string
	Requested  string
	Fallbacked bool
	Supported  bool
}

func matchLocale(requested string, available []string, fallback string) localeMatch {
	fallback = NormalizeLocale(fallback)
	if fallback == "" {
		fallback = defaultFallbackLocale
	}
	if len(available) == 0 {
		return localeMatch{Locale: fallback, Requested: NormalizeLocale(requested), Fallbacked: true}
	}

	byLocale := make(map[string]string, len(available))
	for _, locale := range available {
		normalized := NormalizeLocale(locale)
		byLocale[normalized] = normalized
	}

	candidate := NormalizeLocale(requested)
	if candidate != "" {
		if matched, ok := byLocale[candidate]; ok {
			return localeMatch{Locale: matched, Requested: candidate, Supported: true}
		}
		if alias := localeAlias(candidate); alias != "" {
			if matched, ok := byLocale[alias]; ok {
				return localeMatch{Locale: matched, Requested: candidate, Supported: true}
			}
		}
	}

	if matched, ok := byLocale[fallback]; ok {
		return localeMatch{Locale: matched, Requested: candidate, Fallbacked: candidate != "", Supported: false}
	}
	return localeMatch{Locale: NormalizeLocale(available[0]), Requested: candidate, Fallbacked: candidate != "", Supported: false}
}

func primaryLanguage(locale string) string {
	if idx := strings.IndexByte(locale, '-'); idx >= 0 {
		return locale[:idx]
	}
	return locale
}

func localeAlias(locale string) string {
	switch {
	case locale == "zh" || locale == "zh-CN" || locale == "zh-Hans" || locale == "zh-Hans-CN":
		return "zh-CN"
	case primaryLanguage(locale) == "en":
		return "en-US"
	default:
		return ""
	}
}
