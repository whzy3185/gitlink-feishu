package i18n

import (
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"path/filepath"
	"sort"
	"strings"
)

//go:embed locales/*.json
var embeddedLocales embed.FS

// Loader loads locale messages from a backing store.
type Loader interface {
	Load(locale string) (map[string]string, error)
	AvailableLocales() ([]string, error)
}

type embedLoader struct {
	fs fs.FS
}

// NewEmbedLoader returns the default loader backed by embedded locale files.
func NewEmbedLoader() Loader {
	return embedLoader{fs: embeddedLocales}
}

func (l embedLoader) Load(locale string) (map[string]string, error) {
	locale = NormalizeLocale(locale)
	path := filepath.ToSlash(filepath.Join("locales", locale+".json"))
	data, err := fs.ReadFile(l.fs, path)
	if err != nil {
		return nil, fmt.Errorf("load locale %s: %w", locale, err)
	}

	var messages map[string]string
	if err := json.Unmarshal(data, &messages); err != nil {
		return nil, fmt.Errorf("parse locale %s: %w", locale, err)
	}
	return messages, nil
}

func (l embedLoader) AvailableLocales() ([]string, error) {
	entries, err := fs.ReadDir(l.fs, "locales")
	if err != nil {
		return nil, err
	}

	locales := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		locales = append(locales, strings.TrimSuffix(entry.Name(), ".json"))
	}
	sort.Strings(locales)
	return locales, nil
}

// AvailableLocales returns locales available from the embedded loader.
func AvailableLocales() ([]string, error) {
	return NewEmbedLoader().AvailableLocales()
}
