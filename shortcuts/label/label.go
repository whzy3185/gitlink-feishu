package label

import (
	"encoding/json"
	"fmt"
	"net/url"
	"regexp"
	"strconv"
	"strings"

	"github.com/gitlink-org/gitlink-cli/shortcuts/common"
)

// defaultLabelColor is used when the caller does not provide a color.
const defaultLabelColor = "#1E90FF"

// hexColorPattern matches #RGB and #RRGGBB hex color values.
var hexColorPattern = regexp.MustCompile(`^#(?:[0-9a-fA-F]{3}|[0-9a-fA-F]{6})$`)

// Shortcuts returns issue label (项目标记) management shortcuts.
//
// Issue labels back the issue triage and PR gatekeeping workflows: until now
// they could only be managed through the raw API (issue_tags), so these
// shortcuts close that gap with first-class create/list/update/delete commands.
func Shortcuts() []*common.Shortcut {
	return []*common.Shortcut{
		{
			Name:        "list",
			Description: "List issue labels",
			Flags: []common.Flag{
				{Name: "keyword", Short: "k", Usage: "Filter labels by keyword"},
				{Name: "only-name", Usage: "Return only label id and name: true or false"},
				{Name: "sort-by", Usage: "Sort field: updated_on, created_on, issues_count"},
				{Name: "sort-direction", Usage: "Sort direction: asc or desc"},
			},
			Run: func(ctx *common.RuntimeContext) error {
				if err := ctx.ResolveOwnerRepo(); err != nil {
					return err
				}
				q := url.Values{}
				setQueryIfPresent(q, "keyword", ctx.Arg("keyword"))
				setQueryIfPresent(q, "only_name", ctx.Arg("only-name"))
				setQueryIfPresent(q, "order_by", ctx.Arg("sort-by"))
				setQueryIfPresent(q, "order_direction", ctx.Arg("sort-direction"))
				env, err := ctx.CallAPIWithQuery("GET", labelPath(ctx), q)
				if err != nil {
					return err
				}
				return ctx.Output(env)
			},
		},
		{
			Name:        "create",
			Description: "Create an issue label",
			Flags: []common.Flag{
				{Name: "name", Short: "n", Usage: "Label name", Required: true},
				{Name: "description", Short: "d", Usage: "Label description"},
				{Name: "color", Short: "c", Usage: "Label color in hex, for example: #1E90FF", Default: defaultLabelColor},
			},
			Run: runCreate,
		},
		{
			Name:        "update",
			Description: "Update an issue label while preserving unspecified fields",
			Flags: []common.Flag{
				{Name: "id", Short: "i", Usage: "Label ID", Required: true},
				{Name: "name", Short: "n", Usage: "Label name"},
				{Name: "description", Short: "d", Usage: "Label description"},
				{Name: "color", Short: "c", Usage: "Label color in hex, for example: #1E90FF"},
			},
			Run: runUpdate,
		},
		{
			Name:        "delete",
			Description: "Delete an issue label",
			Flags: []common.Flag{
				{Name: "id", Short: "i", Usage: "Label ID", Required: true},
			},
			Run: func(ctx *common.RuntimeContext) error {
				if err := ctx.ResolveOwnerRepo(); err != nil {
					return err
				}
				id, err := ctx.RequireArg("id")
				if err != nil {
					return err
				}
				env, err := ctx.CallAPI("DELETE", labelItemPath(ctx, id), nil)
				if err != nil {
					return err
				}
				return ctx.Output(env)
			},
		},
		{
			Name:        "clone",
			Description: "Clone all issue labels from a source repository into the current one",
			Flags: []common.Flag{
				{Name: "source", Short: "s", Usage: "Source repository as owner/repo", Required: true},
				{Name: "force", Short: "f", Usage: "Overwrite labels that already exist in the target", Bool: true},
			},
			Run: runClone,
		},
	}
}

// runClone copies every label from a source repository into the current one.
//
// It is a pure composition of the existing list and create/update endpoints:
// the target labels are listed first so that name collisions follow gh's
// semantics — skipped by default, and overwritten (updated in place, which
// preserves the label id and its issue associations) only under --force.
func runClone(ctx *common.RuntimeContext) error {
	if err := ctx.ResolveOwnerRepo(); err != nil {
		return err
	}
	source, err := ctx.RequireArg("source")
	if err != nil {
		return err
	}
	srcOwner, srcRepo, err := splitOwnerRepo(source)
	if err != nil {
		return err
	}
	force := ctx.Arg("force") == "true"

	srcLabels, err := fetchLabelsForRepo(ctx, srcOwner, srcRepo)
	if err != nil {
		return err
	}
	dstLabels, err := fetchLabelsForRepo(ctx, ctx.Owner, ctx.Repo)
	if err != nil {
		return err
	}
	existing := make(map[string]map[string]interface{}, len(dstLabels))
	for _, tag := range dstLabels {
		existing[stringFromMap(tag, "name")] = tag
	}

	created := []string{}
	updated := []string{}
	skipped := []string{}
	for _, tag := range srcLabels {
		name := stringFromMap(tag, "name")
		if name == "" {
			continue
		}
		payload := map[string]interface{}{
			"name":        name,
			"description": stringFromMap(tag, "description"),
			"color":       firstNonEmpty(stringFromMap(tag, "color"), defaultLabelColor),
		}
		if dst, ok := existing[name]; ok {
			if !force {
				skipped = append(skipped, name)
				continue
			}
			id := labelIDString(dst["id"])
			if _, err := ctx.CallAPI("PATCH", repoLabelItemPath(ctx.Owner, ctx.Repo, id), payload); err != nil {
				return err
			}
			updated = append(updated, name)
			continue
		}
		if _, err := ctx.CallAPI("POST", labelPath(ctx), payload); err != nil {
			return err
		}
		created = append(created, name)
	}

	return ctx.OutputData(map[string]interface{}{
		"source":  fmt.Sprintf("%s/%s", srcOwner, srcRepo),
		"target":  fmt.Sprintf("%s/%s", ctx.Owner, ctx.Repo),
		"created": created,
		"updated": updated,
		"skipped": skipped,
	})
}

func runCreate(ctx *common.RuntimeContext) error {
	if err := ctx.ResolveOwnerRepo(); err != nil {
		return err
	}
	name, err := ctx.RequireArg("name")
	if err != nil {
		return err
	}
	color := firstNonEmpty(ctx.Arg("color"), defaultLabelColor)
	if err := validateColor(color); err != nil {
		return err
	}
	payload := map[string]interface{}{
		"name":        name,
		"description": ctx.Arg("description"),
		"color":       color,
	}
	env, err := ctx.CallAPI("POST", labelPath(ctx), payload)
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
	if ctx.Arg("name") == "" && ctx.Arg("description") == "" && ctx.Arg("color") == "" {
		return fmt.Errorf("at least one of --name, --description, or --color is required")
	}

	// The update endpoint requires name, description and color together, so we
	// merge the requested changes onto the label's current values to avoid
	// clobbering fields the caller did not pass.
	current, err := fetchLabel(ctx, id)
	if err != nil {
		return err
	}

	name := firstNonEmpty(ctx.Arg("name"), stringFromMap(current, "name"))
	if name == "" {
		return fmt.Errorf("could not resolve label name for id %s; pass --name explicitly", id)
	}
	color := firstNonEmpty(ctx.Arg("color"), stringFromMap(current, "color"), defaultLabelColor)
	if err := validateColor(color); err != nil {
		return err
	}
	description := ctx.Arg("description")
	if description == "" {
		description = stringFromMap(current, "description")
	}

	payload := map[string]interface{}{
		"name":        name,
		"description": description,
		"color":       color,
	}
	env, err := ctx.CallAPI("PATCH", labelItemPath(ctx, id), payload)
	if err != nil {
		return err
	}
	return ctx.Output(env)
}

// fetchLabel looks up a single label by id from the list endpoint. GitLink does
// not expose a single-label GET, so we page through the list and match by id.
// A nil result (label not found) is not an error: the caller falls back to the
// flags it was given.
func fetchLabel(ctx *common.RuntimeContext, id string) (map[string]interface{}, error) {
	env, err := ctx.CallAPI("GET", labelPath(ctx), nil)
	if err != nil {
		return nil, err
	}
	data, ok := env.Data.(map[string]interface{})
	if !ok {
		return nil, nil
	}
	rawTags, ok := data["issue_tags"].([]interface{})
	if !ok {
		return nil, nil
	}
	for _, raw := range rawTags {
		tag, ok := raw.(map[string]interface{})
		if !ok {
			continue
		}
		if labelIDString(tag["id"]) == id {
			return tag, nil
		}
	}
	return nil, nil
}

// labelPageSize bounds each page of the issue_tags list walk. It mirrors the
// workflow fetchers so a repo with many labels is still copied in full.
const labelPageSize = 100

// fetchLabelsForRepo returns every label of an arbitrary owner/repo, walking the
// paginated issue_tags list so a source or target with more than one page of
// labels is still mirrored completely. A page without an issue_tags array ends
// the walk rather than erroring, so an empty or unrecognized repo reads as "no
// labels".
func fetchLabelsForRepo(ctx *common.RuntimeContext, owner, repo string) ([]map[string]interface{}, error) {
	path := repoLabelPath(owner, repo)
	labels := []map[string]interface{}{}
	// Track ids across pages so the walk terminates even if the endpoint were
	// to ignore the page/limit params and re-serve the full list every time.
	seen := map[string]bool{}
	for page := 1; ; page++ {
		q := url.Values{}
		q.Set("page", strconv.Itoa(page))
		q.Set("limit", strconv.Itoa(labelPageSize))
		env, err := ctx.CallAPIWithQuery("GET", path, q)
		if err != nil {
			return nil, err
		}
		data, ok := env.Data.(map[string]interface{})
		if !ok {
			break
		}
		rawTags, ok := data["issue_tags"].([]interface{})
		if !ok {
			break
		}
		added := 0
		for _, raw := range rawTags {
			tag, ok := raw.(map[string]interface{})
			if !ok {
				continue
			}
			id := labelIDString(tag["id"])
			if id != "" && seen[id] {
				continue
			}
			if id != "" {
				seen[id] = true
			}
			labels = append(labels, tag)
			added++
		}
		if added < labelPageSize {
			break
		}
	}
	return labels, nil
}

func labelPath(ctx *common.RuntimeContext) string {
	return repoLabelPath(ctx.Owner, ctx.Repo)
}

func labelItemPath(ctx *common.RuntimeContext, id string) string {
	return repoLabelItemPath(ctx.Owner, ctx.Repo, id)
}

func repoLabelPath(owner, repo string) string {
	return fmt.Sprintf("/v1/%s/%s/issue_tags", owner, repo)
}

func repoLabelItemPath(owner, repo, id string) string {
	return fmt.Sprintf("%s/%s", repoLabelPath(owner, repo), url.PathEscape(id))
}

// splitOwnerRepo parses an "owner/repo" reference, tolerating a leading slash
// and an extra trailing path so that a full repo URL path still resolves.
func splitOwnerRepo(source string) (string, string, error) {
	trimmed := strings.Trim(strings.TrimSpace(source), "/")
	parts := strings.SplitN(trimmed, "/", 3)
	if len(parts) < 2 || parts[0] == "" || parts[1] == "" {
		return "", "", fmt.Errorf("invalid --source %q: expected owner/repo", source)
	}
	return parts[0], parts[1], nil
}

func validateColor(color string) error {
	if !hexColorPattern.MatchString(color) {
		return fmt.Errorf("invalid --color value %q: use a hex color like #1E90FF or #abc", color)
	}
	return nil
}

func labelIDString(v interface{}) string {
	switch id := v.(type) {
	case string:
		return id
	case float64:
		return strconv.FormatInt(int64(id), 10)
	case json.Number:
		return id.String()
	default:
		return ""
	}
}

func setQueryIfPresent(q url.Values, name, value string) {
	if value != "" {
		q.Set(name, value)
	}
}

func stringFromMap(values map[string]interface{}, key string) string {
	if values == nil {
		return ""
	}
	value, _ := values[key].(string)
	return value
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}
