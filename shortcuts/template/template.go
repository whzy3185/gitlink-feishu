package template

import (
	"errors"
	"fmt"
	"net/url"

	"github.com/gitlink-org/gitlink-cli/internal/i18n"
	"github.com/gitlink-org/gitlink-cli/shortcuts/common"
)

// templateTypes enumerates the supported project template types.
var templateTypes = []string{
	"ProjectTemplates::Issue",
	"ProjectTemplates::PullRequest",
	"ProjectTemplates::Commit",
}

// Shortcuts returns project template management shortcuts.
//
// Project templates provide reusable content scaffolds for issues, pull
// requests and commits. Until now they could only be managed through the raw
// REST API; these shortcuts expose list/get/create/update/delete as
// first-class commands.
func Shortcuts(translators ...*i18n.Translator) []*common.Shortcut {
	tr := shortcutTranslator(translators...)
	return []*common.Shortcut{
		{
			Name:        "list",
			Description: tr.T("cmd.template.list.short"),
			Run: func(ctx *common.RuntimeContext) error {
				if err := ctx.ResolveOwnerRepo(); err != nil {
					return err
				}
				env, err := ctx.CallAPI("GET", templatePath(ctx), nil)
				if err != nil {
					return err
				}
				return ctx.Output(env)
			},
		},
		{
			Name:        "get",
			Description: tr.T("cmd.template.get.short"),
			Flags: []common.Flag{
				{Name: "id", Short: "i", Usage: tr.T("flag.template.id"), Required: true},
			},
			Run: func(ctx *common.RuntimeContext) error {
				if err := ctx.ResolveOwnerRepo(); err != nil {
					return err
				}
				id, err := ctx.RequireArg("id")
				if err != nil {
					return err
				}
				env, err := ctx.CallAPI("GET", templateItemPath(ctx, id), nil)
				if err != nil {
					return err
				}
				return ctx.Output(env)
			},
		},
		{
			Name:        "create",
			Description: tr.T("cmd.template.create.short"),
			Flags: []common.Flag{
				{Name: "type", Short: "t", Usage: tr.T("flag.template.type"), Required: true},
				{Name: "name", Short: "n", Usage: tr.T("flag.template.name"), Required: true},
				{Name: "content", Short: "c", Usage: tr.T("flag.template.content"), Required: true},
			},
			Run: runCreate,
		},
		{
			Name:        "update",
			Description: tr.T("cmd.template.update.short"),
			Flags: []common.Flag{
				{Name: "id", Short: "i", Usage: tr.T("flag.template.id"), Required: true},
				{Name: "type", Short: "t", Usage: tr.T("flag.template.type"), Required: true},
				{Name: "name", Short: "n", Usage: tr.T("flag.template.name"), Required: true},
				{Name: "content", Short: "c", Usage: tr.T("flag.template.content"), Required: true},
			},
			Run: runUpdate,
		},
		{
			Name:        "delete",
			Description: tr.T("cmd.template.delete.short"),
			Flags: []common.Flag{
				{Name: "id", Short: "i", Usage: tr.T("flag.template.id"), Required: true},
			},
			Run: func(ctx *common.RuntimeContext) error {
				if err := ctx.ResolveOwnerRepo(); err != nil {
					return err
				}
				id, err := ctx.RequireArg("id")
				if err != nil {
					return err
				}
				env, err := ctx.CallAPI("DELETE", templateItemPath(ctx, id), nil)
				if err != nil {
					return err
				}
				return ctx.Output(env)
			},
		},
	}
}

func runCreate(ctx *common.RuntimeContext) error {
	if err := ctx.ResolveOwnerRepo(); err != nil {
		return err
	}
	payload, err := buildPayload(ctx)
	if err != nil {
		return err
	}
	env, err := ctx.CallAPI("POST", templatePath(ctx), payload)
	if err != nil {
		return err
	}
	return ctx.Output(env)
}

func runUpdate(ctx *common.RuntimeContext) error {
	if err := ctx.ResolveOwnerRepo(); err != nil {
		return err
	}
	id, err := ctx.RequireArg("id")
	if err != nil {
		return err
	}
	payload, err := buildPayload(ctx)
	if err != nil {
		return err
	}
	env, err := ctx.CallAPI("PUT", templateItemPath(ctx, id), payload)
	if err != nil {
		return err
	}
	return ctx.Output(env)
}

// buildPayload collects the required type/name/content fields and validates
// the template type against the known set.
func buildPayload(ctx *common.RuntimeContext) (map[string]interface{}, error) {
	typ, err := ctx.RequireArg("type")
	if err != nil {
		return nil, err
	}
	if !isValidTemplateType(typ) {
		var msg string
		if ctx.Tr != nil {
			msg = ctx.Tr.Tf("error.template.invalid_type", i18n.Args{
				"type":  typ,
				"types": fmt.Sprintf("%v", templateTypes),
			})
		} else {
			msg = fmt.Sprintf("invalid --type %q; expected one of %v", typ, templateTypes)
		}
		return nil, errors.New(msg)
	}
	name, err := ctx.RequireArg("name")
	if err != nil {
		return nil, err
	}
	content, err := ctx.RequireArg("content")
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{
		"type":    typ,
		"name":    name,
		"content": content,
	}, nil
}

func isValidTemplateType(value string) bool {
	for _, candidate := range templateTypes {
		if candidate == value {
			return true
		}
	}
	return false
}

func templatePath(ctx *common.RuntimeContext) string {
	return fmt.Sprintf("/v1/%s/%s/project_templates", ctx.Owner, ctx.Repo)
}

func templateItemPath(ctx *common.RuntimeContext, id string) string {
	return fmt.Sprintf("%s/%s", templatePath(ctx), url.PathEscape(id))
}

// setQueryIfPresent is a small helper kept for future list-filter extensions.
func setQueryIfPresent(q url.Values, name, value string) {
	if value != "" {
		q.Set(name, value)
	}
}

func shortcutTranslator(translators ...*i18n.Translator) *i18n.Translator {
	if len(translators) > 0 && translators[0] != nil {
		return translators[0]
	}
	return i18n.Default()
}
