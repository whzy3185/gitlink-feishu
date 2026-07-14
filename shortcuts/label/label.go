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

// labelListPageSize is the page size fetchLabel requests while paging the list
// endpoint. A short page that returns fewer rows than this marks the last page.
const labelListPageSize = 50

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
	}
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
	color := firstNonEmpty(ctx.Arg("color"), stringFromMap(current, "color"))
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
// not expose a single-label GET, so we page through the list and match by id. A
// label beyond the first page must still be found, otherwise update would PATCH
// the server's real name/description/color away with defaults, so an id that is
// absent after the whole list is exhausted is reported as an error.
func fetchLabel(ctx *common.RuntimeContext, id string) (map[string]interface{}, error) {
	for page := 1; ; page++ {
		q := url.Values{}
		q.Set("page", strconv.Itoa(page))
		q.Set("limit", strconv.Itoa(labelListPageSize))
		env, err := ctx.CallAPIWithQuery("GET", labelPath(ctx), q)
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
		for _, raw := range rawTags {
			tag, ok := raw.(map[string]interface{})
			if !ok {
				continue
			}
			if labelIDString(tag["id"]) == id {
				return tag, nil
			}
		}
		if len(rawTags) < labelListPageSize {
			break
		}
	}
	return nil, fmt.Errorf("label id %s not found in this repository's issue labels", id)
}

func labelPath(ctx *common.RuntimeContext) string {
	return fmt.Sprintf("/v1/%s/%s/issue_tags", ctx.Owner, ctx.Repo)
}

func labelItemPath(ctx *common.RuntimeContext, id string) string {
	return fmt.Sprintf("%s/%s", labelPath(ctx), url.PathEscape(id))
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
