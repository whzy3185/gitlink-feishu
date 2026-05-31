package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func tempConfigDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("GITLINK_CONFIG_DIR", dir)
	return dir
}

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.BaseURL != DefaultBaseURL {
		t.Fatalf("BaseURL = %q, want %q", cfg.BaseURL, DefaultBaseURL)
	}
	if cfg.Format != DefaultFormat {
		t.Fatalf("Format = %q, want %q", cfg.Format, DefaultFormat)
	}
}

func TestConfigDirEnv(t *testing.T) {
	dir := tempConfigDir(t)
	if got := ConfigDir(); got != dir {
		t.Fatalf("ConfigDir = %q, want %q", got, dir)
	}
}

func TestConfigDirDefault(t *testing.T) {
	// Without GITLINK_CONFIG_DIR set, should use $HOME/.config/gitlink-cli
	t.Setenv("GITLINK_CONFIG_DIR", "")
	got := ConfigDir()
	home, _ := os.UserHomeDir()
	if !strings.Contains(got, ".config") && !strings.Contains(got, "gitlink-cli") {
		t.Fatalf("ConfigDir = %q, expected path under home", got)
	}
	if home != "" && !strings.HasPrefix(got, home) {
		t.Fatalf("ConfigDir = %q, expected to start with home %q", got, home)
	}
}

func TestConfigPath(t *testing.T) {
	dir := tempConfigDir(t)
	got := ConfigPath()
	want := filepath.Join(dir, "config.yaml")
	if got != want {
		t.Fatalf("ConfigPath = %q, want %q", got, want)
	}
}

func TestLoadAndSave(t *testing.T) {
	tempConfigDir(t)

	cfg := DefaultConfig()
	cfg.BaseURL = "https://custom.example.com/api"
	cfg.Format = "json"
	cfg.Editor = "vim"
	cfg.Pager = "less"

	if err := Save(cfg); err != nil {
		t.Fatalf("Save error: %v", err)
	}

	loaded, err := Load()
	if err != nil {
		t.Fatalf("Load error: %v", err)
	}

	if loaded.BaseURL != "https://custom.example.com/api" {
		t.Fatalf("BaseURL = %q", loaded.BaseURL)
	}
	if loaded.Format != "json" {
		t.Fatalf("Format = %q", loaded.Format)
	}
	if loaded.Editor != "vim" {
		t.Fatalf("Editor = %q", loaded.Editor)
	}
	if loaded.Pager != "less" {
		t.Fatalf("Pager = %q", loaded.Pager)
	}
}

func TestLoadDefaultsWhenFileMissing(t *testing.T) {
	tempConfigDir(t)
	// No config file exists
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load error: %v", err)
	}
	if cfg.BaseURL != DefaultBaseURL {
		t.Fatalf("BaseURL = %q, want default", cfg.BaseURL)
	}
	if cfg.Format != DefaultFormat {
		t.Fatalf("Format = %q, want default", cfg.Format)
	}
}

func TestLoadEmptyValuesFallbackToDefaults(t *testing.T) {
	dir := tempConfigDir(t)
	// Write config with empty values
	if err := os.WriteFile(filepath.Join(dir, "config.yaml"), []byte("base_url: \"\"\ndefault_format: \"\"\n"), 0600); err != nil {
		t.Fatalf("write error: %v", err)
	}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load error: %v", err)
	}
	if cfg.BaseURL != DefaultBaseURL {
		t.Fatalf("BaseURL = %q, want default", cfg.BaseURL)
	}
	if cfg.Format != DefaultFormat {
		t.Fatalf("Format = %q, want default", cfg.Format)
	}
}

func TestGet(t *testing.T) {
	tempConfigDir(t)
	cfg := DefaultConfig()
	cfg.BaseURL = "https://get.example.com/api"
	if err := Save(cfg); err != nil {
		t.Fatalf("Save error: %v", err)
	}

	tests := []struct {
		key  string
		want string
	}{
		{"base_url", "https://get.example.com/api"},
		{"default_format", "table"},
		{"editor", ""},
		{"pager", ""},
		{"unknown_key", ""},
	}
	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			got, err := Get(tt.key)
			if err != nil {
				t.Fatalf("Get error: %v", err)
			}
			if got != tt.want {
				t.Fatalf("Get(%q) = %q, want %q", tt.key, got, tt.want)
			}
		})
	}
}

func TestSet(t *testing.T) {
	tempConfigDir(t)
	// First save defaults
	if err := Save(DefaultConfig()); err != nil {
		t.Fatalf("Save error: %v", err)
	}

	if err := Set("base_url", "https://set.example.com/api"); err != nil {
		t.Fatalf("Set base_url error: %v", err)
	}
	if err := Set("editor", "nano"); err != nil {
		t.Fatalf("Set editor error: %v", err)
	}

	// Verify Get reads updated values
	baseURL, _ := Get("base_url")
	if baseURL != "https://set.example.com/api" {
		t.Fatalf("Get base_url = %q", baseURL)
	}
	editor, _ := Get("editor")
	if editor != "nano" {
		t.Fatalf("Get editor = %q", editor)
	}
	// default_format should still be default
	format, _ := Get("default_format")
	if format != DefaultFormat {
		t.Fatalf("Get default_format = %q", format)
	}
}

func TestSetUnknownKey(t *testing.T) {
	tempConfigDir(t)
	if err := Save(DefaultConfig()); err != nil {
		t.Fatalf("Save error: %v", err)
	}
	// Setting unknown key should not error, just silently ignored
	if err := Set("nonexistent", "value"); err != nil {
		t.Fatalf("Set nonexistent error: %v", err)
	}
}

func TestSaveCreatesDir(t *testing.T) {
	// Use a subdirectory that doesn't exist yet
	dir := filepath.Join(t.TempDir(), "new", "subdir")
	t.Setenv("GITLINK_CONFIG_DIR", dir)

	cfg := DefaultConfig()
	cfg.BaseURL = "https://test.example.com/api"
	if err := Save(cfg); err != nil {
		t.Fatalf("Save error: %v", err)
	}

	// Verify it was actually saved
	loaded, err := Load()
	if err != nil {
		t.Fatalf("Load error: %v", err)
	}
	if loaded.BaseURL != "https://test.example.com/api" {
		t.Fatalf("BaseURL = %q", loaded.BaseURL)
	}
}

func TestLoadInvalidYAML(t *testing.T) {
	dir := tempConfigDir(t)
	if err := os.WriteFile(filepath.Join(dir, "config.yaml"), []byte("::: invalid yaml :::"), 0600); err != nil {
		t.Fatalf("write error: %v", err)
	}

	_, err := Load()
	if err == nil {
		t.Fatal("expected error for invalid YAML")
	}
}
