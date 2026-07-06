package license

import (
	"net/url"

	"github.com/gitlink-org/gitlink-cli/internal/i18n"
	"github.com/gitlink-org/gitlink-cli/shortcuts/common"
)

// Shortcuts returns license management shortcuts.
//
// These shortcuts provide access to the GitLink license registry,
// which lists all available open-source licenses supported by the platform.
func Shortcuts(translators ...*i18n.Translator) []*common.Shortcut {
	tr := shortcutTranslator(translators...)
	return []*common.Shortcut{
		{
			Name:        "list",
			Description: tr.T("cmd.license.list.short"),
			Flags: []common.Flag{
				{Name: "name", Short: "n", Usage: tr.T("flag.license.name")},
			},
			Run: runList,
		},
	}
}

func runList(ctx *common.RuntimeContext) error {
	q := url.Values{}
	if name := ctx.Arg("name"); name != "" {
		q.Set("name", name)
	}
	env, err := ctx.CallAPIWithQuery("GET", "/licenses", q)
	if err != nil {
		return err
	}
	return ctx.Output(env)
}

func shortcutTranslator(translators ...*i18n.Translator) *i18n.Translator {
	if len(translators) > 0 && translators[0] != nil {
		return translators[0]
	}
	return i18n.Default()
}
