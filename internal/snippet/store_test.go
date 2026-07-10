package snippet

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLoadReturnsEmptyOnMissingFile(t *testing.T) {
	dir := t.TempDir()
	store := NewSnippetStoreWithPath(filepath.Join(dir, "snippets.json"))

	snippets, err := store.Load()
	if err != nil {
		t.Fatalf("Load on missing file should not error: %v", err)
	}
	if len(snippets) != 0 {
		t.Fatalf("expected empty slice, got %d items", len(snippets))
	}
}

func TestSaveAndLoad(t *testing.T) {
	dir := t.TempDir()
	store := NewSnippetStoreWithPath(filepath.Join(dir, "snippets.json"))

	now := time.Now().Truncate(time.Second)
	original := []Snippet{
		{
			ID:        "abc12345",
			Title:     "Hello World",
			Language:  "go",
			Tags:      []string{"test", "example"},
			Content:   `fmt.Println("hello")`,
			CreatedAt: now,
			UpdatedAt: now,
		},
		{
			ID:        "def67890",
			Title:     "HTTP Handler",
			Language:  "go",
			Tags:      []string{"http"},
			Content:   `func handler(w http.ResponseWriter, r *http.Request) {}`,
			CreatedAt: now,
			UpdatedAt: now,
		},
	}

	if err := store.Save(original); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	loaded, err := store.Load()
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if len(loaded) != 2 {
		t.Fatalf("expected 2 snippets, got %d", len(loaded))
	}
	if loaded[0].ID != "abc12345" {
		t.Errorf("ID mismatch: got %s", loaded[0].ID)
	}
	if loaded[0].Title != "Hello World" {
		t.Errorf("Title mismatch: got %s", loaded[0].Title)
	}
	if loaded[0].Language != "go" {
		t.Errorf("Language mismatch: got %s", loaded[0].Language)
	}
	if len(loaded[0].Tags) != 2 || loaded[0].Tags[0] != "test" {
		t.Errorf("Tags mismatch: got %v", loaded[0].Tags)
	}
	if loaded[0].Content != `fmt.Println("hello")` {
		t.Errorf("Content mismatch: got %s", loaded[0].Content)
	}
	if !loaded[0].CreatedAt.Equal(now) {
		t.Errorf("CreatedAt mismatch: got %v, want %v", loaded[0].CreatedAt, now)
	}
}

func TestSaveCreatesDirectory(t *testing.T) {
	dir := t.TempDir()
	nestedPath := filepath.Join(dir, "a", "b", "c", "snippets.json")
	store := NewSnippetStoreWithPath(nestedPath)

	err := store.Save([]Snippet{})
	if err != nil {
		t.Fatalf("Save to nested path failed: %v", err)
	}
	if _, err := os.Stat(nestedPath); os.IsNotExist(err) {
		t.Fatal("file was not created")
	}
}

func TestGenerateID(t *testing.T) {
	ids := make(map[string]bool)
	for i := 0; i < 100; i++ {
		id := GenerateID()
		if len(id) != 8 {
			t.Errorf("ID length should be 8, got %d: %s", len(id), id)
		}
		if ids[id] {
			t.Errorf("duplicate ID generated: %s", id)
		}
		ids[id] = true
	}
}

func TestLoadEmptyFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "snippets.json")
	os.WriteFile(path, []byte(""), 0o644)

	store := NewSnippetStoreWithPath(path)
	snippets, err := store.Load()
	if err != nil {
		t.Fatalf("Load empty file should not error: %v", err)
	}
	if len(snippets) != 0 {
		t.Fatalf("expected empty slice, got %d", len(snippets))
	}
}
