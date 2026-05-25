package member

import (
	"encoding/csv"
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"

	"github.com/gitlink-org/gitlink-cli/shortcuts/common"
)

var roleAliases = map[string]string{
	"manager":   "Manager",
	"developer": "Developer",
	"reporter":  "Reporter",
	"Manager":   "Manager",
	"Developer": "Developer",
	"Reporter":  "Reporter",
}

// Shortcuts returns repository member management shortcuts.
func Shortcuts() []*common.Shortcut {
	return []*common.Shortcut{
		{
			Name:        "list",
			Description: "List repository members",
			Run: func(ctx *common.RuntimeContext) error {
				if err := ctx.ResolveOwnerRepo(); err != nil {
					return err
				}
				env, err := ctx.CallAPI("GET", collaboratorsPath(ctx), nil)
				if err != nil {
					return err
				}
				return ctx.Output(env)
			},
		},
		{
			Name:        "add",
			Description: "Add a repository member by user ID",
			Flags: []common.Flag{
				{Name: "user-id", Short: "u", Usage: "GitLink user ID to add", Required: true},
			},
			Run: func(ctx *common.RuntimeContext) error {
				if err := ctx.ResolveOwnerRepo(); err != nil {
					return err
				}
				userID, err := parseUserID(ctx.Arg("user-id"))
				if err != nil {
					return err
				}
				env, err := ctx.CallAPI("POST", collaboratorsPath(ctx), map[string]interface{}{"user_id": userID})
				if err != nil {
					return err
				}
				return ctx.Output(env)
			},
		},
		{
			Name:        "batch-add",
			Description: "Add multiple repository members by user IDs or a CSV file",
			Flags: []common.Flag{
				{Name: "user-ids", Short: "u", Usage: "Comma-separated GitLink user IDs, for example: 101,102"},
				{Name: "from", Usage: "Read user IDs from a CSV file. Supports a user_id/id column or first column without header"},
				{Name: "dry-run", Usage: "Preview members that would be added without changing them", Bool: true, Default: "false"},
			},
			Run: runBatchAdd,
		},
		{
			Name:        "remove",
			Description: "Remove a repository member by user ID",
			Flags: []common.Flag{
				{Name: "user-id", Short: "u", Usage: "GitLink user ID to remove", Required: true},
			},
			Run: func(ctx *common.RuntimeContext) error {
				if err := ctx.ResolveOwnerRepo(); err != nil {
					return err
				}
				userID, err := parseUserID(ctx.Arg("user-id"))
				if err != nil {
					return err
				}
				env, err := ctx.CallAPI("DELETE", collaboratorsRemovePath(ctx), map[string]interface{}{"user_id": userID})
				if err != nil {
					return err
				}
				return ctx.Output(env)
			},
		},
		{
			Name:        "role",
			Description: "Change a repository member role",
			Flags: []common.Flag{
				{Name: "user-id", Short: "u", Usage: "GitLink user ID to update", Required: true},
				{Name: "role", Short: "r", Usage: "Member role: Manager, Developer, or Reporter", Required: true},
			},
			Run: func(ctx *common.RuntimeContext) error {
				if err := ctx.ResolveOwnerRepo(); err != nil {
					return err
				}
				userID, err := parseUserID(ctx.Arg("user-id"))
				if err != nil {
					return err
				}
				role, err := normalizeRole(ctx.Arg("role"))
				if err != nil {
					return err
				}
				env, err := ctx.CallAPI("PUT", collaboratorsRolePath(ctx), map[string]interface{}{
					"user_id": userID,
					"role":    role,
				})
				if err != nil {
					return err
				}
				return ctx.Output(env)
			},
		},
		{
			Name:        "invite-link",
			Description: "Get or create a repository invite link",
			Flags: []common.Flag{
				{Name: "role", Short: "r", Usage: "Invite role: manager, developer, or reporter", Default: "developer"},
				{Name: "apply", Usage: "Whether joining by invite requires approval: true or false", Default: "true"},
			},
			Run: func(ctx *common.RuntimeContext) error {
				if err := ctx.ResolveOwnerRepo(); err != nil {
					return err
				}
				role, err := normalizeInviteRole(ctx.Arg("role"))
				if err != nil {
					return err
				}
				apply, err := parseBoolArg("apply", ctx.Arg("apply"))
				if err != nil {
					return err
				}
				query := url.Values{}
				query.Set("role", role)
				query.Set("is_apply", strconv.FormatBool(apply))
				env, err := ctx.CallAPIWithQuery("GET", inviteLinkPath(ctx, "current_link"), query)
				if err != nil {
					return err
				}
				return ctx.Output(env)
			},
		},
		{
			Name:        "invite-info",
			Description: "Show repository invite link information",
			Flags: []common.Flag{
				{Name: "sign", Short: "s", Usage: "Invite link sign", Required: true},
			},
			Run: func(ctx *common.RuntimeContext) error {
				if err := ctx.ResolveOwnerRepo(); err != nil {
					return err
				}
				sign, err := ctx.RequireArg("sign")
				if err != nil {
					return err
				}
				query := url.Values{}
				query.Set("invite_sign", sign)
				env, err := ctx.CallAPIWithQuery("GET", inviteLinkPath(ctx, "show_link"), query)
				if err != nil {
					return err
				}
				return ctx.Output(env)
			},
		},
		{
			Name:        "accept-invite",
			Description: "Accept a repository invite link",
			Flags: []common.Flag{
				{Name: "sign", Short: "s", Usage: "Invite link sign", Required: true},
			},
			Run: func(ctx *common.RuntimeContext) error {
				if err := ctx.ResolveOwnerRepo(); err != nil {
					return err
				}
				sign, err := ctx.RequireArg("sign")
				if err != nil {
					return err
				}
				query := url.Values{}
				query.Set("invite_sign", sign)
				env, err := ctx.CallAPIWithQuery("POST", inviteLinkPath(ctx, "redirect_link"), query)
				if err != nil {
					return err
				}
				return ctx.Output(env)
			},
		},
	}
}

func runBatchAdd(ctx *common.RuntimeContext) error {
	if err := ctx.ResolveOwnerRepo(); err != nil {
		return err
	}
	userIDs, err := collectUserIDs(ctx.Arg("user-ids"), ctx.Arg("from"))
	if err != nil {
		return err
	}
	if len(userIDs) == 0 {
		return fmt.Errorf("provide --user-ids or --from")
	}
	if parseDryRun(ctx.Arg("dry-run")) {
		return ctx.OutputData(map[string]interface{}{
			"dry_run":  true,
			"user_ids": userIDs,
			"count":    len(userIDs),
		})
	}

	results := make([]map[string]interface{}, 0, len(userIDs))
	succeeded := 0
	failed := 0
	for _, userID := range userIDs {
		env, err := ctx.CallAPI("POST", collaboratorsPath(ctx), map[string]interface{}{"user_id": userID})
		result := map[string]interface{}{"user_id": userID}
		if err != nil {
			result["ok"] = false
			result["error"] = err.Error()
			failed++
		} else {
			result["ok"] = env.OK
			result["data"] = env.Data
			if env.OK {
				succeeded++
			} else {
				failed++
			}
		}
		results = append(results, result)
	}
	if err := ctx.OutputData(map[string]interface{}{
		"count":     len(userIDs),
		"succeeded": succeeded,
		"failed":    failed,
		"results":   results,
	}); err != nil {
		return err
	}
	if failed > 0 {
		return fmt.Errorf("%d of %d member(s) failed to add", failed, len(userIDs))
	}
	return nil
}

func collaboratorsPath(ctx *common.RuntimeContext) string {
	return fmt.Sprintf("/%s/%s/collaborators", ctx.Owner, ctx.Repo)
}

func collaboratorsRemovePath(ctx *common.RuntimeContext) string {
	return fmt.Sprintf("%s/remove", collaboratorsPath(ctx))
}

func collaboratorsRolePath(ctx *common.RuntimeContext) string {
	return fmt.Sprintf("%s/change_role", collaboratorsPath(ctx))
}

func inviteLinkPath(ctx *common.RuntimeContext, action string) string {
	return fmt.Sprintf("/%s/%s/project_invite_links/%s", ctx.Owner, ctx.Repo, action)
}

func parseUserID(value string) (int, error) {
	value = strings.TrimSpace(value)
	userID, err := strconv.Atoi(value)
	if err != nil || userID <= 0 {
		return 0, fmt.Errorf("invalid user ID %q", value)
	}
	return userID, nil
}

func normalizeRole(value string) (string, error) {
	role, ok := roleAliases[strings.TrimSpace(value)]
	if !ok {
		return "", fmt.Errorf("invalid --role value %q: use Manager, Developer, or Reporter", value)
	}
	return role, nil
}

func normalizeInviteRole(value string) (string, error) {
	role, err := normalizeRole(value)
	if err != nil {
		return "", fmt.Errorf("invalid --role value %q: use manager, developer, or reporter", value)
	}
	return strings.ToLower(role), nil
}

func parseBoolArg(name, value string) (bool, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", "true":
		return true, nil
	case "false":
		return false, nil
	default:
		return false, fmt.Errorf("invalid --%s value %q: use true or false", name, value)
	}
}

func parseDryRun(value string) bool {
	ok, _ := parseBoolArg("dry-run", value)
	return ok && strings.TrimSpace(value) != ""
}

func collectUserIDs(inline, csvPath string) ([]int, error) {
	seen := map[int]bool{}
	var ids []int
	add := func(raw string) error {
		if strings.TrimSpace(raw) == "" {
			return nil
		}
		userID, err := parseUserID(raw)
		if err != nil {
			return err
		}
		if !seen[userID] {
			seen[userID] = true
			ids = append(ids, userID)
		}
		return nil
	}

	for _, part := range strings.Split(inline, ",") {
		if err := add(part); err != nil {
			return nil, err
		}
	}
	if csvPath != "" {
		csvIDs, err := readUserIDsFromCSV(csvPath)
		if err != nil {
			return nil, err
		}
		for _, userID := range csvIDs {
			if !seen[userID] {
				seen[userID] = true
				ids = append(ids, userID)
			}
		}
	}
	return ids, nil
}

func readUserIDsFromCSV(path string) ([]int, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	rows, err := csv.NewReader(file).ReadAll()
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, nil
	}

	column := 0
	start := 0
	if idx := userIDColumn(rows[0]); idx >= 0 {
		column = idx
		start = 1
	}

	var ids []int
	for _, row := range rows[start:] {
		if column >= len(row) {
			continue
		}
		userID, err := parseUserID(row[column])
		if err != nil {
			return nil, err
		}
		ids = append(ids, userID)
	}
	return ids, nil
}

func userIDColumn(header []string) int {
	for i, name := range header {
		switch strings.ToLower(strings.TrimSpace(name)) {
		case "user_id", "userid", "id":
			return i
		}
	}
	return -1
}
