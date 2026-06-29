package invite

import (
	"fmt"
	"net/url"

	"github.com/gitlink-org/gitlink-cli/internal/i18n"
	"github.com/gitlink-org/gitlink-cli/shortcuts/common"
)

// Shortcuts returns all invite-related shortcuts.
func Shortcuts(translators ...*i18n.Translator) []*common.Shortcut {
	tr := i18n.Default()
	if len(translators) > 0 && translators[0] != nil {
		tr = translators[0]
	}
	return []*common.Shortcut{
		{
			Name:        "generate",
			Description: tr.T("cmd.invite.generate.short"),
			Long:        tr.T("cmd.invite.generate.long"),
			Flags: []common.Flag{
				{Name: "role", Short: "r", Usage: tr.T("flag.invite.role"), Default: "developer"},
				{Name: "is-apply", Usage: tr.T("flag.invite.is-apply"), Default: "true"},
			},
			Run: runGenerateInviteLink,
		},
		{
			Name:        "show",
			Description: tr.T("cmd.invite.show.short"),
			Long:        tr.T("cmd.invite.show.long"),
			Flags: []common.Flag{
				{Name: "invite-sign", Short: "s", Usage: tr.T("flag.invite.sign"), Required: true},
			},
			Run: runShowInviteLink,
		},
		{
			Name:        "accept",
			Description: tr.T("cmd.invite.accept.short"),
			Long:        tr.T("cmd.invite.accept.long"),
			Flags: []common.Flag{
				{Name: "invite-sign", Short: "s", Usage: tr.T("flag.invite.sign"), Required: true},
			},
			Run: runAcceptInvite,
		},
		{
			Name:        "join",
			Description: tr.T("cmd.invite.join.short"),
			Long:        tr.T("cmd.invite.join.long"),
			Flags: []common.Flag{
				{Name: "code", Short: "c", Usage: tr.T("flag.invite.code"), Required: true},
				{Name: "role", Short: "r", Usage: tr.T("flag.invite.role"), Default: "developer"},
			},
			Run: runJoinProject,
		},
		{
			Name:        "quit",
			Description: tr.T("cmd.invite.quit.short"),
			Long:        tr.T("cmd.invite.quit.long"),
			Run:         runQuitProject,
		},
	}
}

// runGenerateInviteLink generates or retrieves the project invite link.
// GET /api/{owner}/{repo}/project_invite_links/current_link.json
func runGenerateInviteLink(ctx *common.RuntimeContext) error {
	if err := ctx.ResolveOwnerRepo(); err != nil {
		return err
	}

	role := ctx.Arg("role")
	if role == "" {
		role = "developer"
	}
	isApply := ctx.Arg("is-apply")
	if isApply == "" {
		isApply = "true"
	}

	q := url.Values{}
	q.Set("role", role)
	q.Set("is_apply", isApply)

	path := fmt.Sprintf("/%s/%s/project_invite_links/current_link.json", ctx.Owner, ctx.Repo)
	env, err := ctx.CallAPIWithQuery("GET", path, q)
	if err != nil {
		return err
	}
	return ctx.Output(env)
}

// runShowInviteLink shows the invite link information.
// GET /api/{owner}/{repo}/project_invite_links/show_link.json
func runShowInviteLink(ctx *common.RuntimeContext) error {
	if err := ctx.ResolveOwnerRepo(); err != nil {
		return err
	}

	inviteSign, err := ctx.RequireArg("invite-sign")
	if err != nil {
		return err
	}

	q := url.Values{}
	q.Set("invite_sign", inviteSign)

	path := fmt.Sprintf("/%s/%s/project_invite_links/show_link.json", ctx.Owner, ctx.Repo)
	env, err := ctx.CallAPIWithQuery("GET", path, q)
	if err != nil {
		return err
	}
	return ctx.Output(env)
}

// runAcceptInvite accepts an invite via the invite link.
// POST /api/{owner}/{repo}/project_invite_links/redirect_link.json
func runAcceptInvite(ctx *common.RuntimeContext) error {
	if err := ctx.ResolveOwnerRepo(); err != nil {
		return err
	}

	inviteSign, err := ctx.RequireArg("invite-sign")
	if err != nil {
		return err
	}

	q := url.Values{}
	q.Set("invite_sign", inviteSign)

	path := fmt.Sprintf("/%s/%s/project_invite_links/redirect_link.json", ctx.Owner, ctx.Repo)
	env, err := ctx.CallAPIWithQuery("POST", path, q)
	if err != nil {
		return err
	}
	return ctx.Output(env)
}

// runJoinProject joins a project by code.
// POST /api/applied_projects.json
func runJoinProject(ctx *common.RuntimeContext) error {
	code, err := ctx.RequireArg("code")
	if err != nil {
		return err
	}

	role := ctx.Arg("role")
	if role == "" {
		role = "developer"
	}

	body := map[string]interface{}{
		"applied_project": map[string]interface{}{
			"code": code,
			"role": role,
		},
	}

	env, err := ctx.CallAPI("POST", "/applied_projects.json", body)
	if err != nil {
		return err
	}
	return ctx.Output(env)
}

// runQuitProject quits from a project.
// POST /api/{owner}/{repo}/quit.json
func runQuitProject(ctx *common.RuntimeContext) error {
	if err := ctx.ResolveOwnerRepo(); err != nil {
		return err
	}

	path := fmt.Sprintf("/%s/%s/quit.json", ctx.Owner, ctx.Repo)
	env, err := ctx.CallAPI("POST", path, nil)
	if err != nil {
		return err
	}
	return ctx.Output(env)
}
