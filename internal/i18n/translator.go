package i18n

import "fmt"

// Translator resolves localized messages with fallback behavior suitable for CLI use.
type Translator struct {
	locale         string
	fallbackLocale string
	messages       map[string]string
	fallback       map[string]string
}

// Default returns an English translator for legacy migration only.
// New command code should receive *Translator explicitly.
func Default() *Translator {
	tr, err := New(Options{Locale: defaultFallbackLocale})
	if err != nil {
		return &Translator{
			locale:         defaultFallbackLocale,
			fallbackLocale: defaultFallbackLocale,
			messages:       map[string]string{},
			fallback:       map[string]string{},
		}
	}
	return tr
}

// New constructs a Translator. Missing messages fall back to FallbackLocale.
func New(opts Options) (*Translator, error) {
	loader := opts.Loader
	if loader == nil {
		loader = NewEmbedLoader()
	}

	available, err := loader.AvailableLocales()
	if err != nil {
		return nil, err
	}

	fallbackLocale := opts.FallbackLocale
	if fallbackLocale == "" {
		fallbackLocale = defaultFallbackLocale
	}
	fallbackLocale = MatchLocale(fallbackLocale, available, defaultFallbackLocale)
	locale := MatchLocale(opts.Locale, available, fallbackLocale)

	fallback, err := loader.Load(fallbackLocale)
	if err != nil {
		return nil, fmt.Errorf("load fallback locale: %w", err)
	}

	messages := fallback
	if locale != fallbackLocale {
		messages, err = loader.Load(locale)
		if err != nil {
			return nil, fmt.Errorf("load locale: %w", err)
		}
	}

	return &Translator{
		locale:         locale,
		fallbackLocale: fallbackLocale,
		messages:       messages,
		fallback:       fallback,
	}, nil
}

func (t *Translator) Locale() string {
	return t.locale
}

// T returns a localized message, falling back to en-US and then the key itself.
func (t *Translator) T(key string) string {
	if t == nil {
		return key
	}
	if value, ok := t.messages[key]; ok {
		return value
	}
	if value, ok := t.fallback[key]; ok {
		return value
	}
	return key
}

// Tf returns a localized message with {name} placeholders rendered from args.
func (t *Translator) Tf(key string, args Args) string {
	return renderTemplate(t.T(key), args)
}
