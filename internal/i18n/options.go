package i18n

const defaultFallbackLocale = "en-US"

// Options controls Translator construction.
type Options struct {
	Locale         string
	FallbackLocale string
	Loader         Loader
}
