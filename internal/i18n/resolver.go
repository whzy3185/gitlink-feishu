package i18n

import (
	"os"
	"strings"
)

// ResolveOptions contains locale inputs ordered by caller intent.
type ResolveOptions struct {
	ExplicitLang string
	Env          map[string]string
	ConfigLang   string
}

type ResolvedLocale struct {
	Locale     string
	Source     string
	Requested  string
	Fallbacked bool
	Supported  bool
}

// ResolveLocale resolves a locale using CLI flag, env, config, system env, fallback.
func ResolveLocale(opts ResolveOptions, available []string) string {
	return ResolveLocaleDetailed(opts, available).Locale
}

func ResolveLocaleDetailed(opts ResolveOptions, available []string) ResolvedLocale {
	candidates := []struct {
		source string
		value  string
	}{
		{source: "flag", value: opts.ExplicitLang},
		{source: "env", value: envValue(opts.Env, "GITLINK_LANG")},
		{source: "config", value: opts.ConfigLang},
		{source: "lc_all", value: envValue(opts.Env, "LC_ALL")},
		{source: "lang", value: envValue(opts.Env, "LANG")},
	}
	for _, candidate := range candidates {
		if strings.TrimSpace(candidate.value) == "" {
			continue
		}
		match := matchLocale(candidate.value, available, defaultFallbackLocale)
		return ResolvedLocale{
			Locale:     match.Locale,
			Source:     candidate.source,
			Requested:  match.Requested,
			Fallbacked: match.Fallbacked,
			Supported:  match.Supported,
		}
	}
	match := matchLocale(defaultFallbackLocale, available, defaultFallbackLocale)
	return ResolvedLocale{
		Locale:     match.Locale,
		Source:     "default",
		Requested:  match.Requested,
		Fallbacked: match.Fallbacked,
		Supported:  match.Supported,
	}
}

// PreScanLang reads --lang before Cobra constructs localized help text.
func PreScanLang(args []string) string {
	for i, arg := range args {
		if arg == "--lang" {
			if i+1 < len(args) {
				return args[i+1]
			}
			return ""
		}
		if strings.HasPrefix(arg, "--lang=") {
			return strings.TrimPrefix(arg, "--lang=")
		}
	}
	return ""
}

// EnvMap returns process environment as a string map.
func EnvMap() map[string]string {
	env := make(map[string]string)
	for _, item := range os.Environ() {
		key, value, ok := strings.Cut(item, "=")
		if ok {
			env[key] = value
		}
	}
	return env
}

func envValue(env map[string]string, key string) string {
	if env == nil {
		return os.Getenv(key)
	}
	return env[key]
}
