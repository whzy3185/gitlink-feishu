package explore

import (
	"fmt"
	"net/url"
	"strconv"

	"github.com/gitlink-org/gitlink-cli/internal/i18n"
	"github.com/gitlink-org/gitlink-cli/shortcuts/common"
)

// Shortcuts — gitlink-cli explore 域：GitLink 官方「分类精选 / 探索」数据源。
//
//	explore +categories              列出全部领域分类（id + name）
//	explore +pinned --category 深度学习 --limit 30   该分类下的精选项目（pinned=d）
//
// 数据源（公开、免鉴权）：
//   - GET /api/project_categories.json  -> {"project_categories":[{id,name},...]}
//   - GET /api/projects.json?pinned=d&category_id=N&limit=M
func Shortcuts(translators ...*i18n.Translator) []*common.Shortcut {
	_ = translators // reserved for future i18n (descriptions currently literal)
	return []*common.Shortcut{
		{
			Name:        "categories",
			Description: "List GitLink project categories (id + name)",
			Run: func(ctx *common.RuntimeContext) error {
				env, err := ctx.CallAPI("GET", "/project_categories", nil)
				if err != nil {
					return err
				}
				return ctx.Output(env)
			},
		},
		{
			Name:        "pinned",
			Description: "List pinned (curated) projects in a category",
			Flags: []common.Flag{
				{Name: "category", Short: "c", Usage: "category name or id (e.g. 深度学习 or 32)", Required: true},
				{Name: "limit", Short: "l", Usage: "number of results", Default: "20"},
				{Name: "page", Short: "p", Usage: "page number", Default: "1"},
			},
			Run: func(ctx *common.RuntimeContext) error {
				catID, err := resolveCategoryID(ctx, ctx.Arg("category"))
				if err != nil {
					return err
				}
				q := url.Values{}
				q.Set("pinned", "d")
				q.Set("category_id", catID)
				q.Set("limit", ctx.Arg("limit"))
				q.Set("page", ctx.Arg("page"))
				env, err := ctx.CallAPIWithQuery("GET", "/projects", q)
				if err != nil {
					return err
				}
				return ctx.Output(env)
			},
		},
	}
}

// resolveCategoryID：纯数字直接返回；否则拉分类表按 name 匹配出 id。
func resolveCategoryID(ctx *common.RuntimeContext, cat string) (string, error) {
	if cat == "" {
		return "", fmt.Errorf("category is required (name or id)")
	}
	if _, err := strconv.Atoi(cat); err == nil {
		return cat, nil
	}
	env, err := ctx.CallAPI("GET", "/project_categories", nil)
	if err != nil {
		return "", fmt.Errorf("resolve category %q: %w", cat, err)
	}
	data, _ := env.Data.(map[string]interface{})
	if data == nil {
		return "", fmt.Errorf("resolve category %q: unexpected response shape", cat)
	}
	cats, _ := data["project_categories"].([]interface{})
	for _, c := range cats {
		cm, ok := c.(map[string]interface{})
		if !ok {
			continue
		}
		if fmt.Sprint(cm["name"]) == cat {
			return fmt.Sprint(cm["id"]), nil
		}
	}
	return "", fmt.Errorf("category %q not found; run `gitlink-cli explore +categories` to list", cat)
}

func shortcutTranslator(translators ...*i18n.Translator) *i18n.Translator {
	if len(translators) > 0 && translators[0] != nil {
		return translators[0]
	}
	return i18n.Default()
}
