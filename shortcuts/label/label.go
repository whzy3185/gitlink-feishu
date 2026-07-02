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
				{Name: "dry-run", Usage: "Preview the delete request without removing the label", Bool: true, Default: "false"},
				{Name: "yes", Usage: "Confirm label deletion", Bool: true, Default: "false"},
			},
			Run: runDelete,
		},
		{
			Name:        "batch-create",
			Description: "Batch create issue labels from semicolon-separated specs",
			Flags: []common.Flag{
				{Name: "labels", Usage: "Semicolon-separated label specs: name:color:description", Required: true},
				{Name: "dry-run", Usage: "Preview labels without creating them", Bool: true, Default: "false"},
				{Name: "yes", Usage: "Confirm real batch label creation", Bool: true, Default: "false"},
			},
			Run: runBatchCreate,
		},
		{
			Name:        "batch-delete",
			Description: "Batch delete issue labels by IDs",
			Flags: []common.Flag{
				{Name: "ids", Usage: "Comma-separated label IDs", Required: true},
				{Name: "dry-run", Usage: "Preview labels without deleting them", Bool: true, Default: "false"},
				{Name: "yes", Usage: "Confirm real batch label deletion", Bool: true, Default: "false"},
			},
			Run: runBatchDelete,
		},
	}
}

type labelSpec struct {
	Name        string `json:"name" yaml:"name"`
	Description string `json:"description" yaml:"description"`
	Color       string `json:"color" yaml:"color"`
}

type labelBatchRequest struct {
	Method string                 `json:"method" yaml:"method"`
	Path   string                 `json:"path" yaml:"path"`
	Body   map[string]interface{} `json:"body,omitempty" yaml:"body,omitempty"`
}

type labelBatchPreview struct {
	Repository string              `json:"repository" yaml:"repository"`
	DryRun     bool                `json:"dry_run" yaml:"dry_run"`
	Action     string              `json:"action" yaml:"action"`
	Requests   []labelBatchRequest `json:"requests" yaml:"requests"`
}

type labelBatchResult struct {
	Request labelBatchRequest `json:"request" yaml:"request"`
	OK      bool              `json:"ok" yaml:"ok"`
	Data    interface{}       `json:"data,omitempty" yaml:"data,omitempty"`
	Error   string            `json:"error,omitempty" yaml:"error,omitempty"`
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

func runDelete(ctx *common.RuntimeContext) error {
	if err := ctx.ResolveOwnerRepo(); err != nil {
		return err
	}
	id, err := ctx.RequireArg("id")
	if err != nil {
		return err
	}
	path := labelItemPath(ctx, id)
	if ctx.Arg("dry-run") == "true" {
		return ctx.OutputData(labelBatchPreview{
			Repository: repositoryName(ctx),
			DryRun:     true,
			Action:     "delete_label",
			Requests: []labelBatchRequest{{
				Method: "DELETE",
				Path:   path,
			}},
		})
	}
	if ctx.Arg("yes") != "true" {
		return fmt.Errorf("label delete is destructive; run with --dry-run first, then pass --yes to confirm")
	}
	env, err := ctx.CallAPI("DELETE", path, nil)
	if err != nil {
		return err
	}
	return ctx.Output(env)
}

func runBatchCreate(ctx *common.RuntimeContext) error {
	if err := ctx.ResolveOwnerRepo(); err != nil {
		return err
	}
	specs, err := parseLabelSpecs(ctx.Arg("labels"))
	if err != nil {
		return err
	}
	requests := make([]labelBatchRequest, 0, len(specs))
	for _, spec := range specs {
		requests = append(requests, labelBatchRequest{
			Method: "POST",
			Path:   labelPath(ctx),
			Body: map[string]interface{}{
				"name":        spec.Name,
				"description": spec.Description,
				"color":       spec.Color,
			},
		})
	}
	if ctx.Arg("dry-run") == "true" || ctx.Arg("yes") != "true" {
		return ctx.OutputData(labelBatchPreview{
			Repository: repositoryName(ctx),
			DryRun:     true,
			Action:     "batch_create_labels",
			Requests:   requests,
		})
	}
	return runLabelBatchRequests(ctx, "batch_create_labels", requests)
}

func runBatchDelete(ctx *common.RuntimeContext) error {
	if err := ctx.ResolveOwnerRepo(); err != nil {
		return err
	}
	ids, err := parseLabelIDList(ctx.Arg("ids"))
	if err != nil {
		return err
	}
	requests := make([]labelBatchRequest, 0, len(ids))
	for _, id := range ids {
		requests = append(requests, labelBatchRequest{
			Method: "DELETE",
			Path:   labelItemPath(ctx, id),
		})
	}
	if ctx.Arg("dry-run") == "true" || ctx.Arg("yes") != "true" {
		return ctx.OutputData(labelBatchPreview{
			Repository: repositoryName(ctx),
			DryRun:     true,
			Action:     "batch_delete_labels",
			Requests:   requests,
		})
	}
	return runLabelBatchRequests(ctx, "batch_delete_labels", requests)
}

func runLabelBatchRequests(ctx *common.RuntimeContext, action string, requests []labelBatchRequest) error {
	results := make([]labelBatchResult, 0, len(requests))
	succeeded := 0
	failed := 0
	for _, request := range requests {
		env, err := ctx.CallAPI(request.Method, request.Path, request.Body)
		result := labelBatchResult{Request: request}
		if err != nil {
			result.OK = false
			result.Error = err.Error()
			failed++
		} else {
			result.OK = env.OK
			result.Data = env.Data
			if env.OK {
				succeeded++
			} else {
				failed++
			}
		}
		results = append(results, result)
	}
	if err := ctx.OutputData(map[string]interface{}{
		"repository": repositoryName(ctx),
		"action":     action,
		"count":      len(requests),
		"succeeded":  succeeded,
		"failed":     failed,
		"results":    results,
	}); err != nil {
		return err
	}
	if failed > 0 {
		return fmt.Errorf("%d of %d label request(s) failed", failed, len(requests))
	}
	return nil
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

func labelPath(ctx *common.RuntimeContext) string {
	return fmt.Sprintf("/v1/%s/%s/issue_tags", ctx.Owner, ctx.Repo)
}

func labelItemPath(ctx *common.RuntimeContext, id string) string {
	return fmt.Sprintf("%s/%s", labelPath(ctx), url.PathEscape(id))
}

func parseLabelSpecs(raw string) ([]labelSpec, error) {
	if strings.TrimSpace(raw) == "" {
		return nil, fmt.Errorf("--labels is required")
	}
	parts := strings.Split(raw, ";")
	specs := make([]labelSpec, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		fields := strings.SplitN(part, ":", 3)
		name := strings.TrimSpace(fields[0])
		if name == "" {
			return nil, fmt.Errorf("invalid label spec %q: name is required", part)
		}
		color := defaultLabelColor
		if len(fields) > 1 && strings.TrimSpace(fields[1]) != "" {
			color = strings.TrimSpace(fields[1])
		}
		if err := validateColor(color); err != nil {
			return nil, err
		}
		description := ""
		if len(fields) > 2 {
			description = strings.TrimSpace(fields[2])
		}
		specs = append(specs, labelSpec{
			Name:        name,
			Description: description,
			Color:       color,
		})
	}
	if len(specs) == 0 {
		return nil, fmt.Errorf("--labels must contain at least one label spec")
	}
	return specs, nil
}

func parseLabelIDList(raw string) ([]string, error) {
	if strings.TrimSpace(raw) == "" {
		return nil, fmt.Errorf("--ids is required")
	}
	parts := strings.Split(raw, ",")
	ids := make([]string, 0, len(parts))
	seen := map[string]bool{}
	for _, part := range parts {
		id := strings.TrimSpace(part)
		if id == "" {
			continue
		}
		n, err := strconv.Atoi(id)
		if err != nil || n <= 0 {
			return nil, fmt.Errorf("invalid label id %q: use positive integer IDs", id)
		}
		if seen[id] {
			continue
		}
		seen[id] = true
		ids = append(ids, id)
	}
	if len(ids) == 0 {
		return nil, fmt.Errorf("--ids must contain at least one label ID")
	}
	return ids, nil
}

func repositoryName(ctx *common.RuntimeContext) string {
	return fmt.Sprintf("%s/%s", ctx.Owner, ctx.Repo)
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
