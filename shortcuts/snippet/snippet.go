package snippet

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/gitlink-org/gitlink-cli/internal/config"
	"github.com/gitlink-org/gitlink-cli/shortcuts/common"
)

// Snippet 是一条本地代码片段（Gist-like，存储在 ~/.config/gitlink-cli/snippets.json）
type Snippet struct {
	Name      string    `json:"name"`
	Content   string    `json:"content"`
	Language  string    `json:"language,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

func snippetsPath() string {
	return filepath.Join(config.ConfigDir(), "snippets.json")
}

func loadSnippets() (map[string]Snippet, error) {
	snippets := map[string]Snippet{}
	data, err := os.ReadFile(snippetsPath())
	if err != nil {
		if os.IsNotExist(err) {
			return snippets, nil
		}
		return nil, err
	}
	if err := json.Unmarshal(data, &snippets); err != nil {
		return nil, err
	}
	return snippets, nil
}

func saveSnippets(snippets map[string]Snippet) error {
	if err := os.MkdirAll(config.ConfigDir(), 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(snippets, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(snippetsPath(), data, 0644)
}

// Shortcuts 返回代码片段管理命令：+list/+create/+view/+delete
func Shortcuts() []*common.Shortcut {
	return []*common.Shortcut{
		{
			Name:        "list",
			Description: "List all local code snippets",
			Run: func(ctx *common.RuntimeContext) error {
				snippets, err := loadSnippets()
				if err != nil {
					return err
				}
				names := make([]string, 0, len(snippets))
				for n := range snippets {
					names = append(names, n)
				}
				sort.Strings(names)
				list := make([]map[string]interface{}, 0, len(names))
				for _, n := range names {
					s := snippets[n]
					list = append(list, map[string]interface{}{
						"name":       s.Name,
						"language":   s.Language,
						"created_at": s.CreatedAt.Format("2006-01-02"),
						"length":     len(s.Content),
					})
				}
				return ctx.OutputData(map[string]interface{}{
					"count":    len(list),
					"snippets": list,
				})
			},
		},
		{
			Name:        "create",
			Description: "Create a local code snippet",
			Flags: []common.Flag{
				{Name: "name", Short: "n", Usage: "Snippet name", Required: true},
				{Name: "content", Short: "c", Usage: "Snippet content", Required: true},
				{Name: "language", Short: "l", Usage: "Language (e.g. go, python)"},
			},
			Run: func(ctx *common.RuntimeContext) error {
				name, err := ctx.RequireArg("name")
				if err != nil {
					return err
				}
				content, err := ctx.RequireArg("content")
				if err != nil {
					return err
				}
				language := ctx.Arg("language")
				snippets, err := loadSnippets()
				if err != nil {
					return err
				}
				snippets[name] = Snippet{
					Name:      name,
					Content:   content,
					Language:  language,
					CreatedAt: time.Now(),
				}
				if err := saveSnippets(snippets); err != nil {
					return err
				}
				return ctx.OutputData(map[string]interface{}{
					"ok":      true,
					"name":    name,
					"message": "snippet created",
				})
			},
		},
		{
			Name:        "view",
			Description: "View a code snippet",
			Flags: []common.Flag{
				{Name: "name", Short: "n", Usage: "Snippet name", Required: true},
			},
			Run: func(ctx *common.RuntimeContext) error {
				name, err := ctx.RequireArg("name")
				if err != nil {
					return err
				}
				snippets, err := loadSnippets()
				if err != nil {
					return err
				}
				s, ok := snippets[name]
				if !ok {
					return fmt.Errorf("snippet '%s' not found", name)
				}
				return ctx.OutputData(s)
			},
		},
		{
			Name:        "delete",
			Description: "Delete a code snippet",
			Flags: []common.Flag{
				{Name: "name", Short: "n", Usage: "Snippet name", Required: true},
			},
			Run: func(ctx *common.RuntimeContext) error {
				name, err := ctx.RequireArg("name")
				if err != nil {
					return err
				}
				snippets, err := loadSnippets()
				if err != nil {
					return err
				}
				if _, ok := snippets[name]; !ok {
					return fmt.Errorf("snippet '%s' not found", name)
				}
				delete(snippets, name)
				if err := saveSnippets(snippets); err != nil {
					return err
				}
				return ctx.OutputData(map[string]interface{}{
					"ok":      true,
					"name":    name,
					"message": "snippet deleted",
				})
			},
		},
	}
}
