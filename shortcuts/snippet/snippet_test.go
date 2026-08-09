package snippet

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/gitlink-org/gitlink-cli/internal/snippet"
	"github.com/gitlink-org/gitlink-cli/shortcuts/common"
)

// setupTestStore creates a temp dir and overrides the package-level testStorePath.
// Returns a cleanup function to restore the original value.
func setupTestStore(t *testing.T) (storePath string) {
	t.Helper()
	dir := t.TempDir()
	storePath = filepath.Join(dir, "snippets.json")
	original := testStorePath
	testStorePath = storePath
	t.Cleanup(func() { testStorePath = original })
	return storePath
}

func newCtx(args map[string]string) *common.RuntimeContext {
	return &common.RuntimeContext{
		Format: "json",
		Args:   args,
	}
}

// --- Create tests ---

func TestSnippetCreate(t *testing.T) {
	storePath := setupTestStore(t)

	ctx := newCtx(map[string]string{
		"title":    "Hello World",
		"language": "go",
		"tags":     "test,example",
		"content":  `fmt.Println("hello")`,
	})
	err := common.RunShortcut(t, Shortcuts(), "create", ctx)
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}

	store := snippet.NewSnippetStoreWithPath(storePath)
	snippets, _ := store.Load()
	if len(snippets) != 1 {
		t.Fatalf("expected 1 snippet, got %d", len(snippets))
	}
	if snippets[0].Title != "Hello World" {
		t.Errorf("title mismatch: got %s", snippets[0].Title)
	}
	if snippets[0].Language != "go" {
		t.Errorf("language mismatch: got %s", snippets[0].Language)
	}
	if len(snippets[0].Tags) != 2 {
		t.Errorf("expected 2 tags, got %d", len(snippets[0].Tags))
	}
	if snippets[0].Content != `fmt.Println("hello")` {
		t.Errorf("content mismatch")
	}
}

func TestSnippetCreateRequiresTitle(t *testing.T) {
	setupTestStore(t)
	ctx := newCtx(map[string]string{
		"content": "some code",
	})
	err := common.RunShortcut(t, Shortcuts(), "create", ctx)
	if err == nil {
		t.Fatal("expected error for missing --title")
	}
}

// --- List tests ---

func TestSnippetList(t *testing.T) {
	storePath := setupTestStore(t)

	// Pre-populate
	store := snippet.NewSnippetStoreWithPath(storePath)
	store.Save([]snippet.Snippet{
		{ID: "a1", Title: "Alpha", Language: "go", Tags: []string{"test"}},
		{ID: "b2", Title: "Beta", Language: "python", Tags: []string{"example"}},
		{ID: "c3", Title: "Gamma", Language: "go", Tags: []string{"test", "http"}},
	})

	ctx := newCtx(map[string]string{})
	err := common.RunShortcut(t, Shortcuts(), "list", ctx)
	if err != nil {
		t.Fatalf("list failed: %v", err)
	}
}

func TestSnippetListFilterByTag(t *testing.T) {
	storePath := setupTestStore(t)
	store := snippet.NewSnippetStoreWithPath(storePath)
	store.Save([]snippet.Snippet{
		{ID: "a1", Title: "Alpha", Language: "go", Tags: []string{"test"}},
		{ID: "b2", Title: "Beta", Language: "python", Tags: []string{"example"}},
		{ID: "c3", Title: "Gamma", Language: "go", Tags: []string{"test", "http"}},
	})

	ctx := newCtx(map[string]string{"tag": "test"})
	err := common.RunShortcut(t, Shortcuts(), "list", ctx)
	if err != nil {
		t.Fatalf("list --tag test failed: %v", err)
	}
}

func TestSnippetListFilterByLanguage(t *testing.T) {
	storePath := setupTestStore(t)
	store := snippet.NewSnippetStoreWithPath(storePath)
	store.Save([]snippet.Snippet{
		{ID: "a1", Title: "Alpha", Language: "go", Tags: []string{"test"}},
		{ID: "b2", Title: "Beta", Language: "python", Tags: []string{"example"}},
	})

	ctx := newCtx(map[string]string{"language": "go"})
	err := common.RunShortcut(t, Shortcuts(), "list", ctx)
	if err != nil {
		t.Fatalf("list --language go failed: %v", err)
	}
}

// --- View tests ---

func TestSnippetView(t *testing.T) {
	storePath := setupTestStore(t)
	store := snippet.NewSnippetStoreWithPath(storePath)
	store.Save([]snippet.Snippet{
		{ID: "abc12345", Title: "Hello", Language: "go", Content: "code"},
	})

	ctx := newCtx(map[string]string{"id": "abc12345"})
	err := common.RunShortcut(t, Shortcuts(), "view", ctx)
	if err != nil {
		t.Fatalf("view failed: %v", err)
	}
}

func TestSnippetViewNotFound(t *testing.T) {
	setupTestStore(t)
	ctx := newCtx(map[string]string{"id": "nonexistent"})
	err := common.RunShortcut(t, Shortcuts(), "view", ctx)
	if err == nil {
		t.Fatal("expected error for nonexistent ID")
	}
}

// --- Search tests ---

func TestSnippetSearch(t *testing.T) {
	storePath := setupTestStore(t)
	store := snippet.NewSnippetStoreWithPath(storePath)
	store.Save([]snippet.Snippet{
		{ID: "a1", Title: "HTTP Handler", Language: "go", Content: "func handler()"},
		{ID: "b2", Title: "Sort Algorithm", Language: "python", Content: "def sort(arr)"},
	})

	ctx := newCtx(map[string]string{"query": "handler"})
	err := common.RunShortcut(t, Shortcuts(), "search", ctx)
	if err != nil {
		t.Fatalf("search failed: %v", err)
	}
}

// --- Update tests ---

func TestSnippetUpdate(t *testing.T) {
	storePath := setupTestStore(t)
	store := snippet.NewSnippetStoreWithPath(storePath)
	store.Save([]snippet.Snippet{
		{ID: "abc12345", Title: "Old Title", Language: "go", Tags: []string{"old"}, Content: "old code"},
	})

	ctx := newCtx(map[string]string{
		"id":    "abc12345",
		"title": "New Title",
	})
	err := common.RunShortcut(t, Shortcuts(), "update", ctx)
	if err != nil {
		t.Fatalf("update failed: %v", err)
	}

	loaded, _ := store.Load()
	if loaded[0].Title != "New Title" {
		t.Errorf("title not updated: got %s", loaded[0].Title)
	}
	if loaded[0].Content != "old code" {
		t.Errorf("content should not change: got %s", loaded[0].Content)
	}
}

func TestSnippetUpdateRequiresField(t *testing.T) {
	setupTestStore(t)
	ctx := newCtx(map[string]string{"id": "abc12345"})
	err := common.RunShortcut(t, Shortcuts(), "update", ctx)
	if err == nil {
		t.Fatal("expected error when no fields provided")
	}
}

func TestSnippetUpdateNotFound(t *testing.T) {
	setupTestStore(t)
	ctx := newCtx(map[string]string{"id": "nonexistent", "title": "X"})
	err := common.RunShortcut(t, Shortcuts(), "update", ctx)
	if err == nil {
		t.Fatal("expected error for nonexistent ID")
	}
}

// --- Delete tests ---

func TestSnippetDelete(t *testing.T) {
	storePath := setupTestStore(t)
	store := snippet.NewSnippetStoreWithPath(storePath)
	store.Save([]snippet.Snippet{
		{ID: "a1", Title: "Keep"},
		{ID: "b2", Title: "Delete Me"},
	})

	ctx := newCtx(map[string]string{"id": "b2"})
	err := common.RunShortcut(t, Shortcuts(), "delete", ctx)
	if err != nil {
		t.Fatalf("delete failed: %v", err)
	}

	loaded, _ := store.Load()
	if len(loaded) != 1 {
		t.Fatalf("expected 1 snippet after delete, got %d", len(loaded))
	}
	if loaded[0].ID != "a1" {
		t.Errorf("wrong snippet remained: got %s", loaded[0].ID)
	}
}

func TestSnippetDeleteNotFound(t *testing.T) {
	setupTestStore(t)
	ctx := newCtx(map[string]string{"id": "nonexistent"})
	err := common.RunShortcut(t, Shortcuts(), "delete", ctx)
	if err == nil {
		t.Fatal("expected error for nonexistent ID")
	}
}

// --- Export tests ---

func TestSnippetExportToFile(t *testing.T) {
	storePath := setupTestStore(t)
	store := snippet.NewSnippetStoreWithPath(storePath)
	store.Save([]snippet.Snippet{
		{ID: "abc12345", Title: "Hello", Content: "package main\nfunc main() {}"},
	})

	outFile := filepath.Join(t.TempDir(), "main.go")
	ctx := newCtx(map[string]string{
		"id":     "abc12345",
		"output": outFile,
	})
	err := common.RunShortcut(t, Shortcuts(), "export", ctx)
	if err != nil {
		t.Fatalf("export failed: %v", err)
	}

	data, err := os.ReadFile(outFile)
	if err != nil {
		t.Fatalf("failed to read exported file: %v", err)
	}
	if string(data) != "package main\nfunc main() {}" {
		t.Errorf("export content mismatch: got %q", string(data))
	}
}

func TestSnippetExportNotFound(t *testing.T) {
	setupTestStore(t)
	ctx := newCtx(map[string]string{"id": "nonexistent"})
	err := common.RunShortcut(t, Shortcuts(), "export", ctx)
	if err == nil {
		t.Fatal("expected error for nonexistent ID")
	}
}
