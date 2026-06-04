package catalog

import (
	"net/url"

	"github.com/gitlink-org/gitlink-cli/internal/i18n"
	"github.com/gitlink-org/gitlink-cli/shortcuts/common"
)

// Shortcuts returns GitLink platform catalog lookup shortcuts.
func Shortcuts(translators ...*i18n.Translator) []*common.Shortcut {
	tr := shortcutTranslator(translators...)
	return []*common.Shortcut{
		{
			Name:        "licenses",
			Description: tr.T("cmd.catalog.licenses.short"),
			Flags: []common.Flag{
				{Name: "name", Short: "n", Usage: tr.T("flag.catalog.name")},
			},
			Run: func(ctx *common.RuntimeContext) error {
				return listCatalog(ctx, "/licenses")
			},
		},
		{
			Name:        "ignores",
			Description: tr.T("cmd.catalog.ignores.short"),
			Flags: []common.Flag{
				{Name: "name", Short: "n", Usage: tr.T("flag.catalog.name")},
			},
			Run: func(ctx *common.RuntimeContext) error {
				return listCatalog(ctx, "/ignores")
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

func listCatalog(ctx *common.RuntimeContext, path string) error {
	query := url.Values{}
	if name := ctx.Arg("name"); name != "" {
		query.Set("name", name)
	}
	env, err := ctx.CallAPIWithQuery("GET", path, query)
	if err != nil {
		return err
	}
	return ctx.Output(env)
}
