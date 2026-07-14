package file

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/url"
	"os"

	"github.com/gitlink-org/gitlink-cli/internal/i18n"
	"github.com/gitlink-org/gitlink-cli/shortcuts/common"
)

// Shortcuts returns all file shortcuts.
func Shortcuts(translators ...*i18n.Translator) []*common.Shortcut {
	tr := shortcutTranslator(translators...)
	return []*common.Shortcut{
		viewShortcut(tr),
		searchShortcut(tr),
		writeShortcut(tr, "create"),
		writeShortcut(tr, "update"),
		deleteShortcut(tr),
		batchShortcut(tr),
	}
}

func batchShortcut(tr *i18n.Translator) *common.Shortcut {
	return &common.Shortcut{
		Name:        "batch",
		Description: tr.T("cmd.file.batch.short"),
		Flags: []common.Flag{
			{Name: "spec", Short: "s", Usage: tr.T("flag.file.batch_spec"), Required: true},
			{Name: "branch", Short: "b", Usage: tr.T("flag.file.branch"), Required: true},
			{Name: "new-branch", Usage: tr.T("flag.file.new_branch")},
			{Name: "message", Short: "m", Usage: tr.T("flag.file.message"), Required: true},
		},
		Run: func(ctx *common.RuntimeContext) error {
			if err := ctx.ResolveOwnerRepo(); err != nil {
				return err
			}
			specPath, err := ctx.RequireArg("spec")
			if err != nil {
				return err
			}
			branch, err := ctx.RequireArg("branch")
			if err != nil {
				return err
			}
			message, err := ctx.RequireArg("message")
			if err != nil {
				return err
			}
			data, err := os.ReadFile(specPath)
			if err != nil {
				return fmt.Errorf("read spec file: %w", err)
			}
			var files []map[string]interface{}
			if err := json.Unmarshal(data, &files); err != nil {
				return fmt.Errorf("spec must be a JSON array of file operations: %w", err)
			}
			if len(files) == 0 {
				return fmt.Errorf("spec contains no file operations")
			}
			for i, f := range files {
				action, _ := f["action_type"].(string)
				switch action {
				case "create", "update", "delete":
				default:
					return fmt.Errorf("files[%d]: action_type must be create, update, or delete; got %q", i, action)
				}
				if path, _ := f["file_path"].(string); path == "" {
					return fmt.Errorf("files[%d]: file_path is required", i)
				}
				if _, ok := f["content"]; !ok {
					f["content"] = ""
				}
				if _, ok := f["encoding"]; !ok {
					f["encoding"] = "text"
				}
			}
			payload := map[string]interface{}{
				"files":   files,
				"branch":  branch,
				"message": message,
			}
			if nb := ctx.Arg("new-branch"); nb != "" {
				payload["new_branch"] = nb
			}
			env, err := ctx.CallAPI("POST", "/v1"+ctx.RepoPath()+"/contents/batch", payload)
			if err != nil {
				return err
			}
			return ctx.Output(env)
		},
	}
}

func shortcutTranslator(translators ...*i18n.Translator) *i18n.Translator {
	if len(translators) > 0 && translators[0] != nil {
		return translators[0]
	}
	return i18n.Default()
}

func refFlag(tr *i18n.Translator) common.Flag {
	return common.Flag{Name: "ref", Usage: tr.T("flag.file.ref")}
}

func viewShortcut(tr *i18n.Translator) *common.Shortcut {
	return &common.Shortcut{
		Name:        "view",
		Description: tr.T("cmd.file.view.short"),
		Flags: []common.Flag{
			{Name: "path", Short: "p", Usage: tr.T("flag.file.path"), Required: true},
			refFlag(tr),
			{Name: "raw", Usage: tr.T("flag.file.raw"), Bool: true},
		},
		Run: func(ctx *common.RuntimeContext) error {
			if err := ctx.ResolveOwnerRepo(); err != nil {
				return err
			}
			path, err := ctx.RequireArg("path")
			if err != nil {
				return err
			}
			q := url.Values{}
			q.Set("filepath", path)
			if ref := ctx.Arg("ref"); ref != "" {
				q.Set("ref", ref)
			}
			env, err := ctx.CallAPIWithQuery("GET", ctx.RepoPath()+"/sub_entries", q)
			if err != nil {
				return err
			}
			if ctx.Arg("raw") == "true" {
				return printRawContent(env.Data)
			}
			return ctx.Output(env)
		},
	}
}

func searchShortcut(tr *i18n.Translator) *common.Shortcut {
	return &common.Shortcut{
		Name:        "search",
		Description: tr.T("cmd.file.search.short"),
		Flags: []common.Flag{
			{Name: "keyword", Short: "k", Usage: tr.T("flag.search.keyword"), Required: true},
			refFlag(tr),
		},
		Run: func(ctx *common.RuntimeContext) error {
			if err := ctx.ResolveOwnerRepo(); err != nil {
				return err
			}
			keyword, err := ctx.RequireArg("keyword")
			if err != nil {
				return err
			}
			q := url.Values{}
			q.Set("search", keyword)
			if ref := ctx.Arg("ref"); ref != "" {
				q.Set("ref", ref)
			}
			env, err := ctx.CallAPIWithQuery("GET", ctx.RepoPath()+"/files", q)
			if err != nil {
				return err
			}
			return ctx.Output(env)
		},
	}
}

func writeShortcut(tr *i18n.Translator, action string) *common.Shortcut {
	return &common.Shortcut{
		Name:        action,
		Description: tr.T("cmd.file." + action + ".short"),
		Flags: []common.Flag{
			{Name: "path", Short: "p", Usage: tr.T("flag.file.path"), Required: true},
			{Name: "content", Short: "c", Usage: tr.T("flag.file.content")},
			{Name: "content-file", Usage: tr.T("flag.file.content_file")},
			{Name: "branch", Short: "b", Usage: tr.T("flag.file.branch"), Required: true},
			{Name: "new-branch", Usage: tr.T("flag.file.new_branch")},
			{Name: "message", Short: "m", Usage: tr.T("flag.file.message")},
		},
		Run: func(ctx *common.RuntimeContext) error {
			if err := ctx.ResolveOwnerRepo(); err != nil {
				return err
			}
			path, err := ctx.RequireArg("path")
			if err != nil {
				return err
			}
			branch, err := ctx.RequireArg("branch")
			if err != nil {
				return err
			}
			content, err := resolveContent(ctx)
			if err != nil {
				return err
			}
			message := ctx.Arg("message")
			if message == "" {
				message = fmt.Sprintf("%s %s", action, path)
			}
			payload := map[string]interface{}{
				"files": []map[string]interface{}{
					{
						"action_type": action,
						"file_path":   path,
						"content":     content,
						"encoding":    "text",
					},
				},
				"branch":  branch,
				"message": message,
			}
			if nb := ctx.Arg("new-branch"); nb != "" {
				payload["new_branch"] = nb
			}
			env, err := ctx.CallAPI("POST", "/v1"+ctx.RepoPath()+"/contents/batch", payload)
			if err != nil {
				return err
			}
			return ctx.Output(env)
		},
	}
}

func deleteShortcut(tr *i18n.Translator) *common.Shortcut {
	return &common.Shortcut{
		Name:        "delete",
		Description: tr.T("cmd.file.delete.short"),
		Flags: []common.Flag{
			{Name: "path", Short: "p", Usage: tr.T("flag.file.path"), Required: true},
			{Name: "branch", Short: "b", Usage: tr.T("flag.file.branch"), Required: true},
			{Name: "new-branch", Usage: tr.T("flag.file.new_branch")},
			{Name: "message", Short: "m", Usage: tr.T("flag.file.message")},
		},
		Run: func(ctx *common.RuntimeContext) error {
			if err := ctx.ResolveOwnerRepo(); err != nil {
				return err
			}
			path, err := ctx.RequireArg("path")
			if err != nil {
				return err
			}
			branch, err := ctx.RequireArg("branch")
			if err != nil {
				return err
			}
			message := ctx.Arg("message")
			if message == "" {
				message = fmt.Sprintf("delete %s", path)
			}
			payload := map[string]interface{}{
				"files": []map[string]interface{}{
					{
						"action_type": "delete",
						"file_path":   path,
						"content":     "",
						"encoding":    "text",
					},
				},
				"branch":  branch,
				"message": message,
			}
			if nb := ctx.Arg("new-branch"); nb != "" {
				payload["new_branch"] = nb
			}
			env, err := ctx.CallAPI("POST", "/v1"+ctx.RepoPath()+"/contents/batch", payload)
			if err != nil {
				return err
			}
			return ctx.Output(env)
		},
	}
}

// resolveContent reads file content from --content or --content-file.
func resolveContent(ctx *common.RuntimeContext) (string, error) {
	content := ctx.Arg("content")
	contentFile := ctx.Arg("content-file")
	if content != "" && contentFile != "" {
		return "", fmt.Errorf("use only one of --content or --content-file")
	}
	if contentFile != "" {
		data, err := os.ReadFile(contentFile)
		if err != nil {
			return "", fmt.Errorf("read content file: %w", err)
		}
		return string(data), nil
	}
	if content == "" {
		return "", fmt.Errorf("one of --content or --content-file is required")
	}
	return content, nil
}

// printRawContent extracts and prints the decoded file content from an API
// response (entries object, readme object, or a bare content field).
func printRawContent(data interface{}) error {
	content, encoding, ok := extractContent(data)
	if !ok {
		return fmt.Errorf("no file content in response (is the path a directory?)")
	}
	if encoding == "base64" {
		if decoded, err := base64.StdEncoding.DecodeString(content); err == nil {
			fmt.Print(string(decoded))
			return nil
		}
	}
	fmt.Print(content)
	return nil
}

func extractContent(data interface{}) (content, encoding string, ok bool) {
	m, isMap := data.(map[string]interface{})
	if !isMap {
		return "", "", false
	}
	if entries, has := m["entries"]; has {
		if em, isEM := entries.(map[string]interface{}); isEM {
			m = em
		}
	}
	c, has := m["content"].(string)
	if !has {
		return "", "", false
	}
	if t, hasType := m["type"].(string); hasType && t != "file" {
		return "", "", false
	}
	enc, _ := m["encoding"].(string)
	return c, enc, true
}
