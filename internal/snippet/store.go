package snippet

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

// Snippet represents a locally stored code snippet.
type Snippet struct {
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	Language  string    `json:"language"`
	Tags      []string  `json:"tags"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// SnippetStore manages snippet persistence in a JSON file.
type SnippetStore struct {
	FilePath string
}

// NewSnippetStore creates a store pointing at the default path:
// ~/.config/gitlink-cli/snippets.json
// Respects GITLINK_CONFIG_DIR env var.
func NewSnippetStore() *SnippetStore {
	dir := os.Getenv("GITLINK_CONFIG_DIR")
	if dir == "" {
		home, _ := os.UserHomeDir()
		dir = filepath.Join(home, ".config", "gitlink-cli")
	}
	return &SnippetStore{
		FilePath: filepath.Join(dir, "snippets.json"),
	}
}

// NewSnippetStoreWithPath creates a store with an explicit file path.
// Used in tests to point at temp directories.
func NewSnippetStoreWithPath(path string) *SnippetStore {
	return &SnippetStore{FilePath: path}
}

// Load reads all snippets from the JSON file.
// Returns an empty slice (not error) if the file does not exist.
func (s *SnippetStore) Load() ([]Snippet, error) {
	data, err := os.ReadFile(s.FilePath)
	if err != nil {
		if os.IsNotExist(err) {
			return []Snippet{}, nil
		}
		return nil, err
	}
	if len(data) == 0 {
		return []Snippet{}, nil
	}
	var snippets []Snippet
	if err := json.Unmarshal(data, &snippets); err != nil {
		return nil, err
	}
	if snippets == nil {
		return []Snippet{}, nil
	}
	return snippets, nil
}

// Save writes all snippets to the JSON file.
// Creates parent directories if needed.
func (s *SnippetStore) Save(snippets []Snippet) error {
	if err := os.MkdirAll(filepath.Dir(s.FilePath), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(snippets, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.FilePath, data, 0o644)
}

// GenerateID creates a random 8-character hex ID.
func GenerateID() string {
	b := make([]byte, 4)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
