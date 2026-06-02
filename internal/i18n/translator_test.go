package i18n

import (
	"errors"
	"testing"
)

type mapLoader struct {
	messages map[string]map[string]string
}

func (l mapLoader) Load(locale string) (map[string]string, error) {
	messages, ok := l.messages[locale]
	if !ok {
		return nil, errors.New("missing locale")
	}
	return messages, nil
}

func (l mapLoader) AvailableLocales() ([]string, error) {
	locales := make([]string, 0, len(l.messages))
	for locale := range l.messages {
		locales = append(locales, locale)
	}
	return locales, nil
}

func TestTranslatorReturnsLocalizedMessage(t *testing.T) {
	tr, err := New(Options{
		Locale: "zh-CN",
		Loader: mapLoader{messages: map[string]map[string]string{
			"en-US": {"cmd.root.short": "GitLink CLI"},
			"zh-CN": {"cmd.root.short": "GitLink 命令行"},
		}},
	})
	if err != nil {
		t.Fatal(err)
	}

	if got := tr.T("cmd.root.short"); got != "GitLink 命令行" {
		t.Fatalf("expected localized message, got %q", got)
	}
}

func TestTranslatorFallsBackToBaseThenKey(t *testing.T) {
	tr, err := New(Options{
		Locale: "zh-CN",
		Loader: mapLoader{messages: map[string]map[string]string{
			"en-US": {"cmd.root.short": "GitLink CLI"},
			"zh-CN": {},
		}},
	})
	if err != nil {
		t.Fatal(err)
	}

	if got := tr.T("cmd.root.short"); got != "GitLink CLI" {
		t.Fatalf("expected fallback message, got %q", got)
	}
	if got := tr.T("cmd.missing.short"); got != "cmd.missing.short" {
		t.Fatalf("expected key fallback, got %q", got)
	}
}

func TestTranslatorRendersArgs(t *testing.T) {
	tr, err := New(Options{
		Locale: "en-US",
		Loader: mapLoader{messages: map[string]map[string]string{
			"en-US": {"output.version": "gitlink-cli {version}"},
		}},
	})
	if err != nil {
		t.Fatal(err)
	}

	if got := tr.Tf("output.version", Args{"version": "1.2.3"}); got != "gitlink-cli 1.2.3" {
		t.Fatalf("expected rendered message, got %q", got)
	}
}
