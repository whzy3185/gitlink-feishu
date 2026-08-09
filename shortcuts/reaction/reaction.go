package reaction

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/gitlink-org/gitlink-cli/internal/i18n"
	"github.com/gitlink-org/gitlink-cli/shortcuts/common"
)

func Shortcuts(translators ...*i18n.Translator) []*common.Shortcut {
	tr := i18n.Default()
	if len(translators) > 0 && translators[0] != nil {
		tr = translators[0]
	}
	return []*common.Shortcut{
		{
			Name:        "watchers",
			Description: tr.T("cmd.reaction.watchers.short"),
			Flags:       timeRangeFlags(tr),
			Run:         runWatchers,
		},
		{
			Name:        "stargazers",
			Description: tr.T("cmd.reaction.stargazers.short"),
			Flags:       timeRangeFlags(tr),
			Run:         runStargazers,
		},
		{
			Name:        "follow",
			Description: tr.T("cmd.reaction.follow.short"),
			Flags:       projectIDFlags(tr),
			Run:         runFollow,
		},
		{
			Name:        "unfollow",
			Description: tr.T("cmd.reaction.unfollow.short"),
			Flags:       projectIDFlags(tr),
			Run:         runUnfollow,
		},
		{
			Name:        "like",
			Description: tr.T("cmd.reaction.like.short"),
			Flags:       projectIDFlags(tr),
			Run:         runLike,
		},
		{
			Name:        "unlike",
			Description: tr.T("cmd.reaction.unlike.short"),
			Flags:       projectIDFlags(tr),
			Run:         runUnlike,
		},
	}
}

func timeRangeFlags(tr *i18n.Translator) []common.Flag {
	return []common.Flag{
		{Name: "start-at", Usage: tr.T("flag.reaction.start_at")},
		{Name: "end-at", Usage: tr.T("flag.reaction.end_at")},
	}
}

func projectIDFlags(tr *i18n.Translator) []common.Flag {
	return []common.Flag{
		{Name: "project-id", Usage: tr.T("flag.reaction.project_id")},
	}
}

func runWatchers(ctx *common.RuntimeContext) error {
	return runUserList(ctx, "watchers")
}

func runStargazers(ctx *common.RuntimeContext) error {
	return runUserList(ctx, "stargazers")
}

func runUserList(ctx *common.RuntimeContext, kind string) error {
	if err := ctx.ResolveOwnerRepo(); err != nil {
		return err
	}
	query, err := timeRangeQuery(ctx)
	if err != nil {
		return err
	}
	env, err := ctx.CallAPIWithQuery("GET", ctx.RepoPath()+"/"+kind, query)
	if err != nil {
		return err
	}
	return ctx.Output(env)
}

func runFollow(ctx *common.RuntimeContext) error {
	return runFollowAction(ctx, "POST", "/watchers/follow")
}

func runUnfollow(ctx *common.RuntimeContext) error {
	return runFollowAction(ctx, "DELETE", "/watchers/unfollow")
}

func runFollowAction(ctx *common.RuntimeContext, method, path string) error {
	if err := ctx.ResolveOwnerRepo(); err != nil {
		return err
	}
	projectID, err := resolveProjectID(ctx)
	if err != nil {
		return err
	}
	query := url.Values{}
	query.Set("target_type", "project")
	query.Set("id", projectID)
	env, err := ctx.CallAPIWithQuery(method, path, query)
	if err != nil {
		return err
	}
	return ctx.Output(env)
}

func runLike(ctx *common.RuntimeContext) error {
	return runPraiseAction(ctx, "POST", "like")
}

func runUnlike(ctx *common.RuntimeContext) error {
	return runPraiseAction(ctx, "DELETE", "unlike")
}

func runPraiseAction(ctx *common.RuntimeContext, method, action string) error {
	if err := ctx.ResolveOwnerRepo(); err != nil {
		return err
	}
	projectID, err := resolveProjectID(ctx)
	if err != nil {
		return err
	}
	env, err := ctx.CallAPI(method, fmt.Sprintf("/projects/%s/praise_tread/%s", projectID, action), nil)
	if err != nil {
		return err
	}
	return ctx.Output(env)
}

func timeRangeQuery(ctx *common.RuntimeContext) (url.Values, error) {
	query := url.Values{}
	if start := strings.TrimSpace(ctx.Arg("start-at")); start != "" {
		if err := validateUnixTimestamp("start-at", start); err != nil {
			return nil, err
		}
		query.Set("start_at", start)
	}
	if end := strings.TrimSpace(ctx.Arg("end-at")); end != "" {
		if err := validateUnixTimestamp("end-at", end); err != nil {
			return nil, err
		}
		query.Set("end_at", end)
	}
	return query, nil
}

func validateUnixTimestamp(name, value string) error {
	parsed, err := strconv.ParseInt(value, 10, 64)
	if err != nil || parsed < 0 {
		return fmt.Errorf("invalid --%s %q: use a non-negative Unix timestamp", name, value)
	}
	return nil
}

func resolveProjectID(ctx *common.RuntimeContext) (string, error) {
	if raw := strings.TrimSpace(ctx.Arg("project-id")); raw != "" {
		return normalizeProjectID(raw)
	}
	env, err := ctx.CallAPI("GET", ctx.RepoPath(), nil)
	if err != nil {
		return "", fmt.Errorf("resolve project id: %w", err)
	}
	data, ok := env.Data.(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("resolve project id: unexpected repository response")
	}
	for _, key := range []string{"id", "project_id"} {
		if id := projectIDString(data[key]); id != "" {
			return id, nil
		}
	}
	return "", fmt.Errorf("resolve project id: repository response did not include id")
}

func normalizeProjectID(value string) (string, error) {
	parsed, err := strconv.ParseInt(value, 10, 64)
	if err != nil || parsed <= 0 {
		return "", fmt.Errorf("invalid --project-id %q: use a positive numeric project ID", value)
	}
	return value, nil
}

func projectIDString(value interface{}) string {
	switch v := value.(type) {
	case float64:
		return strconv.FormatInt(int64(v), 10)
	case int:
		return strconv.Itoa(v)
	case int64:
		return strconv.FormatInt(v, 10)
	case string:
		id, err := normalizeProjectID(strings.TrimSpace(v))
		if err == nil {
			return id
		}
	}
	return ""
}
