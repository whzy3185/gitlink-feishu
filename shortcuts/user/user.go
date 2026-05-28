package user

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
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
			Description: tr.T("cmd.user.keys.short"),
			Flags: []common.Flag{
				{Name: "page", Short: "p", Usage: tr.T("flag.page"), Default: "1"},
				{Name: "limit", Short: "l", Usage: tr.T("flag.limit"), Default: "20"},
			},
			Run: func(ctx *common.RuntimeContext) error {
				q := url.Values{}
				q.Set("page", ctx.Arg("page"))
				q.Set("limit", ctx.Arg("limit"))
				env, err := ctx.CallAPIWithQuery("GET", publicKeysPath(), q)
				if err != nil {
					return err
				}
				return ctx.Output(env)
			},
		},
		{
			Name:        "add-key",
			Description: tr.T("cmd.user.add_key.short"),
			Flags: []common.Flag{
				{Name: "title", Short: "t", Usage: tr.T("flag.user.key.title")},
				{Name: "key", Short: "k", Usage: tr.T("flag.user.key.content")},
				{Name: "from", Short: "f", Usage: tr.T("flag.user.key.from")},
			},
			Run: func(ctx *common.RuntimeContext) error {
				key, err := publicKeyContent(ctx.Arg("key"), ctx.Arg("from"))
				if err != nil {
					return err
				}
				title, err := publicKeyTitle(ctx.Arg("title"), ctx.Arg("from"))
				if err != nil {
					return err
				}
				env, err := ctx.CallAPI("POST", publicKeysPath(), map[string]interface{}{
					"title": title,
					"key":   key,
				})
				if err != nil {
					return err
				}
				return ctx.Output(env)
			},
		},
		{
			Name:        "delete-key",
			Description: tr.T("cmd.user.delete_key.short"),
			Flags: []common.Flag{
				{Name: "id", Short: "i", Usage: tr.T("flag.user.key.id"), Required: true},
			},
			Run: func(ctx *common.RuntimeContext) error {
				id, err := ctx.RequireArg("id")
				if err != nil {
					return err
				}
				id = strings.TrimSpace(id)
				if id == "" {
					return fmt.Errorf("required flag --id is missing")
				}
				keyID, err := parsePublicKeyID(id)
				if err != nil {
					return err
				}
				env, err := ctx.CallAPI("DELETE", fmt.Sprintf("%s/%d", publicKeysPath(), keyID), nil)
				if err != nil {
					return err
				}
				return ctx.Output(env)
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

func parsePublicKeyID(raw string) (int, error) {
	id, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil || id <= 0 {
		return 0, fmt.Errorf("SSH key ID must be a positive integer")
	}
	return id, nil
}

func publicKeysPath() string {
	return "/public_keys"
}

func publicKeyTitle(title, from string) (string, error) {
	title = strings.TrimSpace(title)
	if title != "" {
		return title, nil
	}
	from = strings.TrimSpace(from)
	if from != "" {
		if base := filepath.Base(from); base != "." && base != string(filepath.Separator) {
			return base, nil
		}
	}
	return "", fmt.Errorf("required SSH key title is missing; use --title or provide --from")
}

func publicKeyContent(inline, from string) (string, error) {
	inline = strings.TrimSpace(inline)
	from = strings.TrimSpace(from)
	if inline != "" && from != "" {
		return "", fmt.Errorf("use either --key or --from, not both")
	}
	if from != "" {
		content, err := os.ReadFile(from)
		if err != nil {
			return "", fmt.Errorf("read SSH public key file: %w", err)
		}
		inline = strings.TrimSpace(string(content))
	}
	if inline == "" {
		return "", fmt.Errorf("required SSH public key content is missing; use --key or --from")
	}
	if !hasPublicKeyPrefix(inline) {
		return "", fmt.Errorf("SSH public key content should start with ssh-rsa, ssh-dss, ssh-ed25519, ecdsa-sha2-, or sk-")
	}
	return inline, nil
}

func hasPublicKeyPrefix(key string) bool {
	for _, prefix := range []string{"ssh-rsa", "ssh-dss", "ssh-ed25519", "ecdsa-sha2-", "sk-"} {
		if strings.HasPrefix(key, prefix) {
			return true
		}
	}
	return false
}
