package snippet

import (
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/gitlink-org/gitlink-cli/internal/output"
	"github.com/gitlink-org/gitlink-cli/internal/snippet"
	"github.com/gitlink-org/gitlink-cli/shortcuts/common"
)

// testStorePath overrides the snippet store file path. Empty means use default.
// This variable exists for testing only.
var testStorePath string

func getStore() *snippet.SnippetStore {
	if testStorePath != "" {
		return snippet.NewSnippetStoreWithPath(testStorePath)
	}
	return snippet.NewSnippetStore()
}

func Shortcuts() []*common.Shortcut {
	return []*common.Shortcut{
		{
			Name:        "create",
			Description: "Create a new code snippet",
			Flags: []common.Flag{
				{Name: "title", Short: "t", Usage: "Snippet title", Required: true},
				{Name: "language", Short: "l", Usage: "Programming language"},
				{Name: "tags", Short: "g", Usage: "Tags (comma-separated)"},
				{Name: "content", Short: "c", Usage: "Snippet content (- for stdin)"},
			},
			Run: func(ctx *common.RuntimeContext) error {
				title, err := ctx.RequireArg("title")
				if err != nil {
					return err
				}
				content, err := readContent(ctx)
				if err != nil {
					return err
				}

				now := time.Now()
				s := snippet.Snippet{
					ID:        snippet.GenerateID(),
					Title:     title,
					Language:  ctx.Arg("language"),
					Tags:      parseTags(ctx.Arg("tags")),
					Content:   content,
					CreatedAt: now,
					UpdatedAt: now,
				}

				store := getStore()
				snippets, err := store.Load()
				if err != nil {
					return fmt.Errorf("读取代码片段失败: %w", err)
				}
				snippets = append(snippets, s)
				if err := store.Save(snippets); err != nil {
					return fmt.Errorf("保存代码片段失败: %w", err)
				}
				return ctx.OutputData(s)
			},
		},
		{
			Name:        "list",
			Description: "List all saved code snippets",
			Flags: []common.Flag{
				{Name: "tag", Short: "t", Usage: "Filter by tag"},
				{Name: "language", Short: "l", Usage: "Filter by language"},
				{Name: "keyword", Short: "k", Usage: "Filter by keyword in title"},
			},
			Run: func(ctx *common.RuntimeContext) error {
				store := getStore()
				snippets, err := store.Load()
				if err != nil {
					return fmt.Errorf("读取代码片段失败: %w", err)
				}

				filtered := filterSnippets(snippets, ctx)

				var summaries []map[string]interface{}
				for _, s := range filtered {
					summaries = append(summaries, toSummary(s))
				}
				if summaries == nil {
					summaries = []map[string]interface{}{}
				}
				return ctx.OutputData(summaries)
			},
		},
		{
			Name:        "view",
			Description: "View a saved code snippet",
			Flags: []common.Flag{
				{Name: "id", Short: "i", Usage: "Snippet ID", Required: true},
			},
			Run: func(ctx *common.RuntimeContext) error {
				id, err := ctx.RequireArg("id")
					if err != nil {
						return err
					}
				store := getStore()
				snippets, err := store.Load()
				if err != nil {
					return fmt.Errorf("读取代码片段失败: %w", err)
				}
				s, _ := findByID(snippets, id)
				if s == nil {
					return fmt.Errorf("代码片段 %s 不存在", id)
				}
				return ctx.OutputData(s)
			},
		},
		{
			Name:        "search",
			Description: "Full-text search across snippets",
			Flags: []common.Flag{
				{Name: "query", Short: "q", Usage: "Search query", Required: true},
			},
			Run: func(ctx *common.RuntimeContext) error {
				query, err := ctx.RequireArg("query")
				if err != nil {
					return err
				}
				store := getStore()
				snippets, err := store.Load()
				if err != nil {
					return fmt.Errorf("读取代码片段失败: %w", err)
				}

				lowerQuery := strings.ToLower(query)
				var results []map[string]interface{}
				for _, s := range snippets {
					if matchesQuery(s, lowerQuery) {
						results = append(results, toSummary(s))
					}
				}
				if results == nil {
					results = []map[string]interface{}{}
				}
				return ctx.OutputData(results)
			},
		},
		{
			Name:        "update",
			Description: "Update an existing code snippet",
			Flags: []common.Flag{
				{Name: "id", Short: "i", Usage: "Snippet ID", Required: true},
				{Name: "title", Short: "t", Usage: "New title"},
				{Name: "language", Short: "l", Usage: "New language"},
				{Name: "tags", Short: "g", Usage: "New tags (comma-separated)"},
				{Name: "content", Short: "c", Usage: "New content"},
			},
			Run: func(ctx *common.RuntimeContext) error {
				id, err := ctx.RequireArg("id")
					if err != nil {
						return err
					}

				title := ctx.Arg("title")
				language := ctx.Arg("language")
				tags := ctx.Arg("tags")
				content := ctx.Arg("content")
				if title == "" && language == "" && tags == "" && content == "" {
					return fmt.Errorf("至少需要指定 --title、--language、--tags 或 --content 之一")
				}

				store := getStore()
				snippets, err := store.Load()
				if err != nil {
					return fmt.Errorf("读取代码片段失败: %w", err)
				}

				s, idx := findByID(snippets, id)
				if s == nil {
					return fmt.Errorf("代码片段 %s 不存在", id)
				}

				if title != "" {
					s.Title = title
				}
				if language != "" {
					s.Language = language
				}
				if tags != "" {
					s.Tags = parseTags(tags)
				}
				if content != "" {
					s.Content = content
				}
				s.UpdatedAt = time.Now()
				snippets[idx] = *s

				if err := store.Save(snippets); err != nil {
					return fmt.Errorf("保存代码片段失败: %w", err)
				}
				return ctx.OutputData(s)
			},
		},
		{
			Name:        "delete",
			Description: "Delete a saved code snippet",
			Flags: []common.Flag{
				{Name: "id", Short: "i", Usage: "Snippet ID", Required: true},
			},
			Run: func(ctx *common.RuntimeContext) error {
				id, err := ctx.RequireArg("id")
					if err != nil {
						return err
					}
				store := getStore()
				snippets, err := store.Load()
				if err != nil {
					return fmt.Errorf("读取代码片段失败: %w", err)
				}

				_, idx := findByID(snippets, id)
				if idx == -1 {
					return fmt.Errorf("代码片段 %s 不存在", id)
				}

				remaining := make([]snippet.Snippet, 0, len(snippets)-1)
				remaining = append(remaining, snippets[:idx]...)
				remaining = append(remaining, snippets[idx+1:]...)

				if err := store.Save(remaining); err != nil {
					return fmt.Errorf("保存代码片段失败: %w", err)
				}
				return ctx.OutputData(map[string]interface{}{
					"message": "代码片段已删除",
					"id":      id,
				})
			},
		},
		{
			Name:        "export",
			Description: "Export a snippet to a file",
			Flags: []common.Flag{
				{Name: "id", Short: "i", Usage: "Snippet ID", Required: true},
				{Name: "output", Short: "o", Usage: "Output file path (default: stdout)"},
			},
			Run: func(ctx *common.RuntimeContext) error {
				id, err := ctx.RequireArg("id")
					if err != nil {
						return err
					}
				store := getStore()
				snippets, err := store.Load()
				if err != nil {
					return fmt.Errorf("读取代码片段失败: %w", err)
				}

				s, _ := findByID(snippets, id)
				if s == nil {
					return fmt.Errorf("代码片段 %s 不存在", id)
				}

				outputPath := ctx.Arg("output")
				if outputPath != "" {
					if err := os.WriteFile(outputPath, []byte(s.Content), 0o644); err != nil {
						return fmt.Errorf("导出文件失败: %w", err)
					}
					return ctx.OutputData(map[string]interface{}{
						"message": "导出成功",
						"file":    outputPath,
						"id":      id,
					})
				}
				// No output file — print content to stdout
				fmt.Fprint(os.Stdout, s.Content)
				return nil
			},
		},
	}
}

// --- Helper functions ---

func findByID(snippets []snippet.Snippet, id string) (*snippet.Snippet, int) {
	for i, s := range snippets {
		if s.ID == id {
			return &snippets[i], i
		}
	}
	return nil, -1
}

func toSummary(s snippet.Snippet) map[string]interface{} {
	return map[string]interface{}{
		"id":         s.ID,
		"title":      s.Title,
		"language":   s.Language,
		"tags":       s.Tags,
		"updated_at": s.UpdatedAt,
	}
}

func parseTags(raw string) []string {
	if raw == "" {
		return nil
	}
	var tags []string
	for _, t := range strings.Split(raw, ",") {
		t = strings.TrimSpace(t)
		if t != "" {
			tags = append(tags, t)
		}
	}
	return tags
}

func readContent(ctx *common.RuntimeContext) (string, error) {
	content := ctx.Arg("content")
	if content == "-" {
		data, err := io.ReadAll(os.Stdin)
		if err != nil {
			return "", fmt.Errorf("读取标准输入失败: %w", err)
		}
		return string(data), nil
	}
	if content == "" {
		// Check if stdin has data (piped)
		info, err := os.Stdin.Stat()
		if err == nil && info.Mode()&os.ModeCharDevice == 0 {
			data, err := io.ReadAll(os.Stdin)
			if err != nil {
				return "", fmt.Errorf("读取标准输入失败: %w", err)
			}
			return string(data), nil
		}
	}
	return content, nil
}

func filterSnippets(snippets []snippet.Snippet, ctx *common.RuntimeContext) []snippet.Snippet {
	tag := ctx.Arg("tag")
	lang := ctx.Arg("language")
	keyword := ctx.Arg("keyword")

	var filtered []snippet.Snippet
	for _, s := range snippets {
		if tag != "" && !hasTag(s, tag) {
			continue
		}
		if lang != "" && !strings.EqualFold(s.Language, lang) {
			continue
		}
		if keyword != "" && !strings.Contains(strings.ToLower(s.Title), strings.ToLower(keyword)) {
			continue
		}
		filtered = append(filtered, s)
	}
	return filtered
}

func hasTag(s snippet.Snippet, tag string) bool {
	lower := strings.ToLower(tag)
	for _, t := range s.Tags {
		if strings.ToLower(t) == lower {
			return true
		}
	}
	return false
}

func matchesQuery(s snippet.Snippet, lowerQuery string) bool {
	if strings.Contains(strings.ToLower(s.Title), lowerQuery) {
		return true
	}
	if strings.Contains(strings.ToLower(s.Language), lowerQuery) {
		return true
	}
	if strings.Contains(strings.ToLower(s.Content), lowerQuery) {
		return true
	}
	for _, t := range s.Tags {
		if strings.Contains(strings.ToLower(t), lowerQuery) {
			return true
		}
	}
	return false
}

// ensure output package is referenced (used in export stdout fallback)
var _ = (*output.Envelope)(nil)
