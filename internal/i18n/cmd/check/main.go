package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/gitlink-org/gitlink-cli/internal/i18n"
)

func main() {
	fix := flag.Bool("fix", false, "format locale JSON files")
	scanCode := flag.Bool("scan-code", false, "scan Go source for referenced i18n keys")
	flag.Parse()

	problems, err := i18n.Validate(i18n.NewEmbedLoader(), "en-US")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if len(problems) > 0 {
		for _, problem := range problems {
			fmt.Fprintln(os.Stderr, problem.String())
		}
		os.Exit(1)
	}
	if err := checkLocaleFormat(*fix); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if *scanCode {
		if err := checkCodeReferences(); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	}
	fmt.Println("i18n messages are valid")
}

func checkLocaleFormat(fix bool) error {
	files, err := filepath.Glob(filepath.Join("internal", "i18n", "locales", "*.json"))
	if err != nil {
		return err
	}
	for _, path := range files {
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		formatted, err := formatJSON(data)
		if err != nil {
			return fmt.Errorf("%s: %w", path, err)
		}
		if string(data) == string(formatted) {
			continue
		}
		if fix {
			if err := os.WriteFile(path, formatted, 0600); err != nil {
				return err
			}
			continue
		}
		return fmt.Errorf("%s: locale JSON is not formatted; run go run ./internal/i18n/cmd/check --fix", path)
	}
	return nil
}

func formatJSON(data []byte) ([]byte, error) {
	var messages map[string]string
	if err := json.Unmarshal(data, &messages); err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	if err := enc.Encode(messages); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func checkCodeReferences() error {
	loader := i18n.NewEmbedLoader()
	base, err := loader.Load("en-US")
	if err != nil {
		return err
	}
	used, defaultUses, err := scanCodeKeys([]string{"cmd", "shortcuts", "internal"})
	if err != nil {
		return err
	}
	var missing []string
	for key := range used {
		if _, ok := base[key]; !ok {
			missing = append(missing, key)
		}
	}
	sort.Strings(missing)
	if len(missing) > 0 {
		return fmt.Errorf("missing i18n key references: %s", strings.Join(missing, ", "))
	}
	for _, item := range defaultUses {
		fmt.Fprintf(os.Stderr, "warning: avoid new i18n.Default() usage at %s\n", item)
	}
	return nil
}

func scanCodeKeys(roots []string) (map[string]struct{}, []string, error) {
	keyPattern := regexp.MustCompile(`(?:tr|ctx\.Tr|i18n\.Default\(\))\.T(?:f)?\("([^"]+)"`)
	defaultPattern := regexp.MustCompile(`i18n\.Default\(\)\.T(?:f)?\("([^"]+)"`)
	used := map[string]struct{}{}
	var defaultUses []string
	for _, root := range roots {
		err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if entry.IsDir() {
				if strings.Contains(filepath.ToSlash(path), "internal/i18n/locales") {
					return filepath.SkipDir
				}
				return nil
			}
			if entry.Type()&os.ModeSymlink != 0 {
				return nil
			}
			if filepath.Ext(path) != ".go" {
				return nil
			}
			if strings.HasSuffix(path, "_test.go") {
				return nil
			}
			if !entry.Type().IsRegular() {
				return nil
			}
			data, err := os.ReadFile(path) // #nosec G122 -- dev-only scan over repo roots; symlinks are skipped above.
			if err != nil {
				return err
			}
			text := string(data)
			for _, match := range keyPattern.FindAllStringSubmatch(text, -1) {
				used[match[1]] = struct{}{}
			}
			for _, match := range defaultPattern.FindAllStringSubmatchIndex(text, -1) {
				line := 1 + strings.Count(text[:match[0]], "\n")
				defaultUses = append(defaultUses, fmt.Sprintf("%s:%d", filepath.ToSlash(path), line))
			}
			return nil
		})
		if err != nil {
			return nil, nil, err
		}
	}
	sort.Strings(defaultUses)
	return used, defaultUses, nil
}
