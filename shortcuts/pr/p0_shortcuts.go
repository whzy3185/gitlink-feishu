package pr

import (
	"fmt"
	"strconv"

	"github.com/gitlink-org/gitlink-cli/internal/i18n"
	"github.com/gitlink-org/gitlink-cli/shortcuts/common"
)

// restoredPRShortcuts contains commands whose implementations or tests were
// merged previously but were dropped from the command registry during later
// batch conflict resolution. Review-comment writes stay intentionally
// unregistered until their production API contract is verified.
func restoredPRShortcuts(tr *i18n.Translator) []*common.Shortcut {
	return []*common.Shortcut{
		newPRReviewCommentsShortcut(),
		{
			Name:        "commits",
			Description: tr.T("cmd.pr.commits.short"),
			Flags: []common.Flag{
				{Name: "id", Short: "i", Usage: tr.T("flag.pr.id"), Required: true},
			},
			Run: func(ctx *common.RuntimeContext) error {
				if err := ctx.ResolveOwnerRepo(); err != nil {
					return err
				}
				id, err := ctx.RequireArg("id")
				if err != nil {
					return err
				}
				env, err := ctx.CallAPI("GET", prV1Path(ctx, id)+"/commits", nil)
				if err != nil {
					return err
				}
				return ctx.Output(env)
			},
		},
		{
			Name:        "branches",
			Description: tr.T("cmd.pr.branches.short"),
			Run: func(ctx *common.RuntimeContext) error {
				if err := ctx.ResolveOwnerRepo(); err != nil {
					return err
				}
				env, err := ctx.CallAPI("GET", ctx.RepoPath()+"/pulls/get_branches", nil)
				if err != nil {
					return err
				}
				return ctx.Output(env)
			},
		},
		{
			Name:        "check-merge",
			Description: tr.T("cmd.pr.check_merge.short"),
			Flags: []common.Flag{
				{Name: "head", Usage: tr.T("flag.pr.head"), Required: true},
				{Name: "base", Usage: tr.T("flag.pr.base"), Required: true},
				{Name: "fork-project-id", Usage: tr.T("flag.pr.fork_project_id")},
			},
			Run: func(ctx *common.RuntimeContext) error {
				if err := ctx.ResolveOwnerRepo(); err != nil {
					return err
				}
				head, err := ctx.RequireArg("head")
				if err != nil {
					return err
				}
				base, err := ctx.RequireArg("base")
				if err != nil {
					return err
				}
				payload := map[string]interface{}{
					"head": head,
					"base": base,
				}
				if forkID := ctx.Arg("fork-project-id"); forkID != "" {
					id, err := strconv.Atoi(forkID)
					if err != nil {
						return fmt.Errorf("--fork-project-id must be an integer, got %q", forkID)
					}
					payload["fork_project_id"] = id
					payload["is_original"] = true
				}
				env, err := ctx.CallAPI("POST", ctx.RepoPath()+"/pulls/check_can_merge", payload)
				if err != nil {
					return err
				}
				return ctx.Output(env)
			},
		},
		{
			Name:        "checks",
			Description: tr.T("cmd.pr.checks.short"),
			Long:        tr.T("cmd.pr.checks.long"),
			Flags: []common.Flag{
				{Name: "id", Short: "i", Usage: tr.T("flag.pr.id"), Required: true},
			},
			Run: func(ctx *common.RuntimeContext) error {
				if err := ctx.ResolveOwnerRepo(); err != nil {
					return err
				}
				id, err := ctx.RequireArg("id")
				if err != nil {
					return err
				}
				prEnv, err := ctx.CallAPI("GET", fmt.Sprintf("%s/pulls/%s", ctx.RepoPath(), id), nil)
				if err != nil {
					return err
				}
				headBranch, headSHA, err := extractPullRequestHead(prEnv)
				if err != nil {
					return err
				}
				buildsEnv, err := ctx.CallAPI("GET", ctx.RepoPath()+"/builds", nil)
				if err != nil {
					return err
				}
				result := selectPullRequestChecks(tr, id, headBranch, headSHA, buildsFromEnvelope(buildsEnv))
				return ctx.OutputData(result)
			},
		},
		{
			Name:        "checkout",
			Description: tr.T("cmd.pr.checkout.short"),
			Long:        tr.T("cmd.pr.checkout.long"),
			Flags: []common.Flag{
				{Name: "id", Short: "i", Usage: tr.T("flag.pr.id"), Required: true},
				{Name: "branch", Short: "b", Usage: tr.T("flag.pr.checkout_branch")},
				{Name: "force", Short: "f", Usage: tr.T("flag.pr.checkout_force"), Bool: true, Default: "false"},
				{Name: "dry-run", Usage: tr.T("flag.pr.checkout_dry_run"), Bool: true, Default: "false"},
			},
			Run: runCheckout,
		},
	}
}
