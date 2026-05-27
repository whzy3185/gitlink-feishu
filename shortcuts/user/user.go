package user

import (
	"fmt"
	"net/url"
	"os"
	"strings"

	"github.com/gitlink-org/gitlink-cli/internal/i18n"
	"github.com/gitlink-org/gitlink-cli/shortcuts/common"
)

func Shortcuts(translators ...*i18n.Translator) []*common.Shortcut {
	tr := shortcutTranslator(translators...)
	return []*common.Shortcut{
		{
			Name:        "me",
			Description: tr.T("cmd.user.me.short"),
			Run: func(ctx *common.RuntimeContext) error {
				env, err := ctx.CallAPI("GET", "/users/me", nil)
				if err != nil {
					return err
				}
				return ctx.Output(env)
			},
		},
		{
			Name:        "current",
			Description: "Show current user profile details",
			Run: func(ctx *common.RuntimeContext) error {
				env, err := ctx.CallAPI("GET", "/users/get_user_info", nil)
				if err != nil {
					return err
				}
				return ctx.Output(env)
			},
		},
		{
			Name:        "info",
			Description: tr.T("cmd.user.info.short"),
			Flags: []common.Flag{
				{Name: "login", Short: "l", Usage: tr.T("flag.user.login"), Required: true},
			},
			Run: func(ctx *common.RuntimeContext) error {
				login, err := ctx.RequireArg("login")
				if err != nil {
					return err
				}
				env, err := ctx.CallAPI("GET", fmt.Sprintf("/users/%s", login), nil)
				if err != nil {
					return err
				}
				return ctx.Output(env)
			},
		},
		{
			Name:        "keys",
			Description: "List current user public keys",
			Flags: []common.Flag{
				{Name: "page", Short: "p", Usage: "Page number", Default: "1"},
				{Name: "limit", Usage: "Items per page", Default: "20"},
			},
			Run: func(ctx *common.RuntimeContext) error {
				q := url.Values{}
				setQueryIfPresent(q, ctx, "page", "page")
				setQueryIfPresent(q, ctx, "limit", "limit")
				env, err := ctx.CallAPIWithQuery("GET", "/public_keys", q)
				if err != nil {
					return err
				}
				return ctx.Output(env)
			},
		},
		{
			Name:        "key-create",
			Description: "Create a public key",
			Flags: []common.Flag{
				{Name: "title", Short: "t", Usage: "Public key title", Required: true},
				{Name: "key", Short: "k", Usage: "Public key content"},
				{Name: "key-file", Usage: "Path to a public key file"},
				{Name: "dry-run", Usage: "Preview the create request without adding a key", Bool: true, Default: "false"},
			},
			Run: func(ctx *common.RuntimeContext) error {
				title, err := ctx.RequireArg("title")
				if err != nil {
					return err
				}
				key, err := publicKeyFromArgs(ctx)
				if err != nil {
					return err
				}
				payload := map[string]interface{}{
					"title": title,
					"key":   key,
				}
				path := "/public_keys"
				if ctx.Arg("dry-run") == "true" {
					return ctx.OutputData(map[string]interface{}{
						"dry_run": true,
						"action":  "create_public_key",
						"method":  "POST",
						"path":    path,
						"payload": payload,
					})
				}
				env, err := ctx.CallAPI("POST", path, payload)
				if err != nil {
					return err
				}
				return ctx.Output(env)
			},
		},
		{
			Name:        "key-delete",
			Description: "Delete a public key",
			Flags: []common.Flag{
				{Name: "id", Short: "i", Usage: "Public key ID", Required: true},
				{Name: "dry-run", Usage: "Preview the delete request without removing a key", Bool: true, Default: "false"},
			},
			Run: func(ctx *common.RuntimeContext) error {
				id, err := ctx.RequireArg("id")
				if err != nil {
					return err
				}
				path := fmt.Sprintf("/public_keys/%s", url.PathEscape(id))
				if ctx.Arg("dry-run") == "true" {
					return ctx.OutputData(map[string]interface{}{
						"dry_run": true,
						"action":  "delete_public_key",
						"method":  "DELETE",
						"path":    path,
					})
				}
				env, err := ctx.CallAPI("DELETE", path, nil)
				if err != nil {
					return err
				}
				return ctx.Output(env)
			},
		},
		userStatsShortcut("activity", "Show user recent activity statistics", "statistics/activity", false, false),
		userStatsShortcut("headmap", "Show user contribution heatmap", "headmaps", true, false),
		userStatsShortcut("develop", "Show user development ability statistics", "statistics/develop", false, true),
		userStatsShortcut("role", "Show user role statistics", "statistics/role", false, true),
		userStatsShortcut("major", "Show user major statistics", "statistics/major", false, true),
	}
}

func userStatsShortcut(name, description, endpoint string, withYear, withTimeRange bool) *common.Shortcut {
	flags := []common.Flag{
		{Name: "login", Short: "l", Usage: "User login name", Required: true},
	}
	if withYear {
		flags = append(flags, common.Flag{Name: "year", Usage: "Contribution year"})
	}
	if withTimeRange {
		flags = append(flags,
			common.Flag{Name: "start-time", Usage: "Start timestamp"},
			common.Flag{Name: "end-time", Usage: "End timestamp"},
		)
	}
	return &common.Shortcut{
		Name:        name,
		Description: description,
		Flags:       flags,
		Run: func(ctx *common.RuntimeContext) error {
			login, err := ctx.RequireArg("login")
			if err != nil {
				return err
			}
			q := url.Values{}
			setQueryIfPresent(q, ctx, "year", "year")
			setQueryIfPresent(q, ctx, "start-time", "start_time")
			setQueryIfPresent(q, ctx, "end-time", "end_time")
			path := fmt.Sprintf("/users/%s/%s", url.PathEscape(login), endpoint)
			env, err := ctx.CallAPIWithQuery("GET", path, q)
			if err != nil {
				return err
			}
			return ctx.Output(env)
		},
	}
}

func publicKeyFromArgs(ctx *common.RuntimeContext) (string, error) {
	key := strings.TrimSpace(ctx.Arg("key"))
	keyFile := strings.TrimSpace(ctx.Arg("key-file"))
	if key != "" && keyFile != "" {
		return "", fmt.Errorf("use only one of --key or --key-file")
	}
	if key == "" && keyFile == "" {
		return "", fmt.Errorf("required flag --key or --key-file is missing")
	}
	if keyFile != "" {
		data, err := os.ReadFile(keyFile)
		if err != nil {
			return "", fmt.Errorf("read --key-file: %w", err)
		}
		key = strings.TrimSpace(string(data))
	}
	if key == "" {
		return "", fmt.Errorf("public key content is empty")
	}
	return key, nil
}

func setQueryIfPresent(q url.Values, ctx *common.RuntimeContext, flagName, queryName string) {
	if value := ctx.Arg(flagName); value != "" {
		q.Set(queryName, value)
	}
}

func shortcutTranslator(translators ...*i18n.Translator) *i18n.Translator {
	if len(translators) > 0 && translators[0] != nil {
		return translators[0]
	}
	return i18n.Default()
}
