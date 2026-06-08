package release

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/gitlink-org/gitlink-cli/internal/i18n"
	"github.com/gitlink-org/gitlink-cli/internal/output"
	"github.com/gitlink-org/gitlink-cli/shortcuts/common"
)

func Shortcuts(translators ...*i18n.Translator) []*common.Shortcut {
	tr := shortcutTranslator(translators...)
	return []*common.Shortcut{
		{
			Name:        "list",
			Description: tr.T("cmd.release.list.short"),
			Flags: []common.Flag{
				{Name: "page", Short: "p", Usage: tr.T("flag.page"), Default: "1"},
				{Name: "limit", Short: "l", Usage: tr.T("flag.limit"), Default: "20"},
			},
			Run: func(ctx *common.RuntimeContext) error {
				if err := ctx.ResolveOwnerRepo(); err != nil {
					return err
				}
				q := url.Values{}
				q.Set("page", ctx.Arg("page"))
				q.Set("limit", ctx.Arg("limit"))
				env, err := ctx.CallAPIWithQuery("GET", ctx.RepoPath()+"/releases", q)
				if err != nil {
					return err
				}
				return ctx.Output(env)
			},
		},
		{
			Name:        "create",
			Description: tr.T("cmd.release.create.short"),
			Flags: []common.Flag{
				{Name: "tag", Short: "t", Usage: tr.T("flag.release.tag"), Required: true},
				{Name: "name", Short: "n", Usage: tr.T("flag.release.name"), Required: true},
				{Name: "body", Short: "b", Usage: tr.T("flag.release.body")},
				{Name: "target", Usage: tr.T("flag.release.target"), Default: "master"},
				{Name: "prerelease", Usage: tr.T("flag.release.prerelease"), Default: "false"},
				{Name: "draft", Usage: "Mark as draft (true/false)", Default: "false"},
				{Name: "attachment-ids", Usage: "Comma-separated attachment IDs"},
			},
			Run: func(ctx *common.RuntimeContext) error {
				if err := ctx.ResolveOwnerRepo(); err != nil {
					return err
				}
				tag, err := ctx.RequireArg("tag")
				if err != nil {
					return err
				}
				name, err := ctx.RequireArg("name")
				if err != nil {
					return err
				}
				draft, err := releaseBoolArg(ctx, "draft", false)
				if err != nil {
					return err
				}
				prerelease, err := releaseBoolArg(ctx, "prerelease", false)
				if err != nil {
					return err
				}
				payload := map[string]interface{}{
					"tag_name":   tag,
					"name":       name,
					"draft":      draft,
					"prerelease": prerelease,
				}
				if b := ctx.Arg("body"); b != "" {
					payload["body"] = b
				}
				if t := ctx.Arg("target"); t != "" {
					payload["target_commitish"] = t
				}
				if attachmentIDs := ctx.Arg("attachment-ids"); attachmentIDs != "" {
					ids, err := parseReleaseAttachmentIDs(attachmentIDs)
					if err != nil {
						return err
					}
					payload["attachment_ids"] = ids
				}
				env, err := ctx.CallAPI("POST", ctx.RepoPath()+"/releases", payload)
				if err != nil {
					return err
				}
				return ctx.Output(env)
			},
		},
		{
			Name:        "edit",
			Description: "Get release edit data",
			Flags: []common.Flag{
				{Name: "id", Short: "i", Usage: "Release version ID", Required: true},
			},
			Run: func(ctx *common.RuntimeContext) error {
				if err := ctx.ResolveOwnerRepo(); err != nil {
					return err
				}
				id, err := ctx.RequireArg("id")
				if err != nil {
					return err
				}
				env, err := ctx.CallAPI("GET", fmt.Sprintf("%s/releases/%s/edit", ctx.RepoPath(), id), nil)
				if err != nil {
					return err
				}
				return ctx.Output(env)
			},
		},
		{
			Name:        "view",
			Description: tr.T("cmd.release.view.short"),
			Flags: []common.Flag{
				{Name: "id", Short: "i", Usage: tr.T("flag.release.id_or_tag"), Required: true},
			},
			Run: func(ctx *common.RuntimeContext) error {
				if err := ctx.ResolveOwnerRepo(); err != nil {
					return err
				}
				id, err := ctx.RequireArg("id")
				if err != nil {
					return err
				}
				env, err := ctx.CallAPI("GET", fmt.Sprintf("%s/releases/%s", ctx.RepoPath(), id), nil)
				if err != nil {
					return err
				}
				return ctx.Output(env)
			},
		},
		{
			Name:        "update",
			Description: "Update a release while preserving unspecified fields",
			Flags: []common.Flag{
				{Name: "id", Short: "i", Usage: "Release version ID", Required: true},
				{Name: "tag", Short: "t", Usage: "Tag name"},
				{Name: "name", Short: "n", Usage: "Release name"},
				{Name: "body", Short: "b", Usage: "Release notes"},
				{Name: "target", Usage: "Target branch"},
				{Name: "prerelease", Usage: "Mark as prerelease (true/false)"},
				{Name: "draft", Usage: "Mark as draft (true/false)"},
				{Name: "attachment-ids", Usage: "Comma-separated attachment IDs"},
				{Name: "dry-run", Usage: "Preview the update request without changing release state", Bool: true, Default: "false"},
			},
			Run: runUpdate,
		},
		{
			Name:        "delete",
			Description: tr.T("cmd.release.delete.short"),
			Flags: []common.Flag{
				{Name: "id", Short: "i", Usage: tr.T("flag.release.id"), Required: true},
				{Name: "dry-run", Usage: "Preview the delete request without changing release state", Bool: true, Default: "false"},
			},
			Run: func(ctx *common.RuntimeContext) error {
				if err := ctx.ResolveOwnerRepo(); err != nil {
					return err
				}
				id, err := ctx.RequireArg("id")
				if err != nil {
					return err
				}
				path := fmt.Sprintf("%s/releases/%s", ctx.RepoPath(), id)
				if ctx.Arg("dry-run") == "true" {
					return ctx.OutputData(map[string]interface{}{
						"repository": fmt.Sprintf("%s/%s", ctx.Owner, ctx.Repo),
						"dry_run":    true,
						"action":     "delete_release",
						"method":     "DELETE",
						"path":       path,
					})
				}
				_, delErr := ctx.CallAPI("DELETE", path, nil)
				if delErr != nil {
					// GitLink API bug: delete succeeds but returns error status.
					// Verify by checking if the release still exists.
					_, viewErr := ctx.CallAPI("GET", path, nil)
					if viewErr != nil {
						// Release no longer exists — delete actually succeeded
						return ctx.Output(output.SuccessEnvelope(map[string]interface{}{
							"message": "删除成功",
						}, nil))
					}
					// Release still exists — delete truly failed
					return delErr
				}
				return ctx.Output(output.SuccessEnvelope(map[string]interface{}{
					"message": "删除成功",
				}, nil))
			},
		},
	}
}

func shortcutTranslator(translators ...*i18n.Translator) *i18n.Translator {
	if len(translators) > 0 && translators[0] != nil {
		return translators[0]
	}
	return i18n.Default()
}

func runUpdate(ctx *common.RuntimeContext) error {
	if err := ctx.ResolveOwnerRepo(); err != nil {
		return err
	}
	id, err := ctx.RequireArg("id")
	if err != nil {
		return err
	}
	if !hasReleaseUpdateArgs(ctx) {
		return fmt.Errorf("at least one of --tag, --name, --body, --target, --draft, --prerelease, or --attachment-ids is required")
	}
	if err := validateReleaseUpdateArgs(ctx); err != nil {
		return err
	}
	current, err := fetchReleaseEdit(ctx, id)
	if err != nil {
		return fmt.Errorf("fetch release edit data: %w", err)
	}
	payload, err := releaseUpdatePayload(ctx, current)
	if err != nil {
		return err
	}
	path := fmt.Sprintf("%s/releases/%s", ctx.RepoPath(), id)
	if ctx.Arg("dry-run") == "true" {
		return ctx.OutputData(map[string]interface{}{
			"repository": fmt.Sprintf("%s/%s", ctx.Owner, ctx.Repo),
			"dry_run":    true,
			"action":     "update_release",
			"method":     "PUT",
			"path":       path,
			"payload":    payload,
		})
	}
	env, err := ctx.CallAPI("PUT", path, payload)
	if err != nil {
		return err
	}
	return ctx.Output(env)
}

func fetchReleaseEdit(ctx *common.RuntimeContext, id string) (map[string]interface{}, error) {
	env, err := ctx.CallAPI("GET", fmt.Sprintf("%s/releases/%s/edit", ctx.RepoPath(), id), nil)
	if err != nil {
		return nil, err
	}
	data, ok := env.Data.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("failed to parse release edit data")
	}
	return data, nil
}

func releaseUpdatePayload(ctx *common.RuntimeContext, current map[string]interface{}) (map[string]interface{}, error) {
	name := firstReleaseValue(ctx.Arg("name"), releaseString(current, "name"))
	if name == "" {
		return nil, fmt.Errorf("required release name is missing; pass --name")
	}
	tag := firstReleaseValue(ctx.Arg("tag"), releaseString(current, "tag_name"))
	if tag == "" {
		return nil, fmt.Errorf("required release tag is missing; pass --tag")
	}
	body := firstReleaseValue(ctx.Arg("body"), releaseString(current, "body"))
	target := firstReleaseValue(ctx.Arg("target"), releaseString(current, "target_commitish"))

	draft, err := releaseBoolFromArgsOrMap(ctx, "draft", current, false)
	if err != nil {
		return nil, err
	}
	prerelease, err := releaseBoolFromArgsOrMap(ctx, "prerelease", current, false)
	if err != nil {
		return nil, err
	}

	payload := map[string]interface{}{
		"name":             name,
		"tag_name":         tag,
		"body":             body,
		"target_commitish": target,
		"draft":            draft,
		"prerelease":       prerelease,
	}
	if attachmentIDs := ctx.Arg("attachment-ids"); attachmentIDs != "" {
		ids, err := parseReleaseAttachmentIDs(attachmentIDs)
		if err != nil {
			return nil, err
		}
		payload["attachment_ids"] = ids
	} else if ids := releaseAttachmentIDs(current); len(ids) > 0 {
		payload["attachment_ids"] = ids
	}
	return payload, nil
}

func hasReleaseUpdateArgs(ctx *common.RuntimeContext) bool {
	for _, name := range []string{"tag", "name", "body", "target", "draft", "prerelease", "attachment-ids"} {
		if ctx.Arg(name) != "" {
			return true
		}
	}
	return false
}

func validateReleaseUpdateArgs(ctx *common.RuntimeContext) error {
	for _, name := range []string{"draft", "prerelease"} {
		if ctx.Arg(name) == "" {
			continue
		}
		if _, err := releaseBoolArg(ctx, name, false); err != nil {
			return err
		}
	}
	if ctx.Arg("attachment-ids") != "" {
		_, err := parseReleaseAttachmentIDs(ctx.Arg("attachment-ids"))
		return err
	}
	return nil
}

func releaseBoolArg(ctx *common.RuntimeContext, name string, defaultValue bool) (bool, error) {
	value := strings.TrimSpace(ctx.Arg(name))
	if value == "" {
		return defaultValue, nil
	}
	switch strings.ToLower(value) {
	case "true":
		return true, nil
	case "false":
		return false, nil
	default:
		return false, fmt.Errorf("invalid --%s value %q: use true or false", name, value)
	}
}

func releaseBoolFromArgsOrMap(ctx *common.RuntimeContext, name string, current map[string]interface{}, defaultValue bool) (bool, error) {
	if ctx.Arg(name) != "" {
		return releaseBoolArg(ctx, name, defaultValue)
	}
	if current != nil {
		if value, ok := current[name].(bool); ok {
			return value, nil
		}
	}
	return defaultValue, nil
}

func parseReleaseAttachmentIDs(value string) ([]string, error) {
	parts := strings.Split(value, ",")
	ids := make([]string, 0, len(parts))
	seen := map[string]bool{}
	for _, part := range parts {
		id := strings.TrimSpace(part)
		if id == "" {
			continue
		}
		if seen[id] {
			continue
		}
		seen[id] = true
		ids = append(ids, id)
	}
	if len(ids) == 0 {
		return nil, fmt.Errorf("--attachment-ids must include at least one ID")
	}
	return ids, nil
}

func releaseAttachmentIDs(current map[string]interface{}) []string {
	if current == nil {
		return nil
	}
	attachments, ok := current["attachments"].([]interface{})
	if !ok {
		return nil
	}
	ids := make([]string, 0, len(attachments))
	for _, attachment := range attachments {
		item, ok := attachment.(map[string]interface{})
		if !ok {
			continue
		}
		if id := releaseIDString(item["id"]); id != "" {
			ids = append(ids, id)
		}
	}
	return ids
}

func releaseIDString(value interface{}) string {
	switch id := value.(type) {
	case string:
		return strings.TrimSpace(id)
	case float64:
		if id == float64(int64(id)) {
			return strconv.FormatInt(int64(id), 10)
		}
		return fmt.Sprintf("%v", id)
	case int:
		return strconv.Itoa(id)
	default:
		return ""
	}
}

func releaseString(values map[string]interface{}, key string) string {
	if values == nil {
		return ""
	}
	value, _ := values[key].(string)
	return strings.TrimSpace(value)
}

func firstReleaseValue(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}
