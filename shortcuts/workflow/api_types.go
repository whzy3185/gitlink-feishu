package workflow

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/gitlink-org/gitlink-cli/shortcuts/common"
)

type TriageFetchOptions struct {
	Owner  string
	Repo   string
	State  string
	Limit  int
	Page   int
	Labels []string
	Since  string
}

type HealthFetchOptions struct {
	Owner          string
	Repo           string
	StaleDays      int
	IncludeCI      bool
	IncludeRelease bool
	IncludeDocs    bool
}

func workflowRepoPath(owner, repo string) string {
	return fmt.Sprintf("/v1/%s/%s", strings.TrimSpace(owner), strings.TrimSpace(repo))
}

func resolveFetchRepo(ctx *common.RuntimeContext, owner, repo string) (string, string, error) {
	if strings.TrimSpace(owner) != "" && strings.TrimSpace(repo) != "" {
		return strings.TrimSpace(owner), strings.TrimSpace(repo), nil
	}
	if strings.TrimSpace(ctx.Owner) != "" && strings.TrimSpace(ctx.Repo) != "" {
		return strings.TrimSpace(ctx.Owner), strings.TrimSpace(ctx.Repo), nil
	}
	if err := ctx.ResolveOwnerRepo(); err != nil {
		return "", "", err
	}
	if strings.TrimSpace(ctx.Owner) == "" || strings.TrimSpace(ctx.Repo) == "" {
		return "", "", fmt.Errorf("repository owner/repo is required; use --owner and --repo or run inside a GitLink repository")
	}
	return strings.TrimSpace(ctx.Owner), strings.TrimSpace(ctx.Repo), nil
}

func normalizeAPIData(data interface{}) (interface{}, error) {
	switch v := data.(type) {
	case nil:
		return nil, nil
	case string:
		trimmed := strings.TrimSpace(v)
		if trimmed == "" {
			return nil, nil
		}
		var decoded interface{}
		if json.Unmarshal([]byte(trimmed), &decoded) != nil {
			return v, nil
		}
		return decoded, nil
	case json.RawMessage:
		if len(v) == 0 {
			return nil, nil
		}
		var decoded interface{}
		if err := json.Unmarshal(v, &decoded); err != nil {
			return nil, err
		}
		return decoded, nil
	default:
		return v, nil
	}
}

func apiObject(data interface{}) map[string]interface{} {
	normalized, err := normalizeAPIData(data)
	if err != nil {
		return nil
	}
	switch v := normalized.(type) {
	case map[string]interface{}:
		return v
	case []interface{}:
		if len(v) == 1 {
			if item, ok := v[0].(map[string]interface{}); ok {
				return item
			}
		}
	}
	return nil
}

func apiList(data interface{}) []interface{} {
	normalized, err := normalizeAPIData(data)
	if err != nil {
		return nil
	}
	switch v := normalized.(type) {
	case []interface{}:
		return v
	case map[string]interface{}:
		for _, key := range []string{"issues", "pulls", "pull_requests", "files", "commits", "releases", "builds", "items", "records", "data"} {
			if raw, ok := v[key]; ok {
				if items := apiList(raw); len(items) > 0 {
					return items
				}
			}
		}
		if looksLikeIssueOrRepoItem(v) {
			return []interface{}{v}
		}
	}
	return nil
}

func looksLikeIssueOrRepoItem(v map[string]interface{}) bool {
	_, hasTitle := v["title"]
	_, hasSubject := v["subject"]
	_, hasNumber := v["number"]
	_, hasID := v["id"]
	_, hasIID := v["iid"]
	_, hasIssueNumber := v["issue_number"]
	_, hasProjectIndex := v["project_issues_index"]
	return hasTitle || hasSubject || hasNumber || hasID || hasIID || hasIssueNumber || hasProjectIndex
}

func apiString(v interface{}) string {
	switch value := v.(type) {
	case string:
		return value
	case json.Number:
		return value.String()
	case fmt.Stringer:
		return value.String()
	case float64:
		return trimTrailingZero(fmt.Sprintf("%f", value))
	case float32:
		return trimTrailingZero(fmt.Sprintf("%f", value))
	case int:
		return strconv.Itoa(value)
	case int64:
		return strconv.FormatInt(value, 10)
	case int32:
		return strconv.FormatInt(int64(value), 10)
	case uint64:
		return strconv.FormatUint(value, 10)
	case uint32:
		return strconv.FormatUint(uint64(value), 10)
	case bool:
		return strconv.FormatBool(value)
	default:
		return ""
	}
}

func trimTrailingZero(value string) string {
	value = strings.TrimSuffix(value, "000000")
	value = strings.TrimSuffix(value, ".000000")
	value = strings.TrimSuffix(value, ".0")
	value = strings.TrimSuffix(value, ".")
	return value
}

func apiInt(v interface{}) int {
	switch value := v.(type) {
	case int:
		return value
	case int8:
		return int(value)
	case int16:
		return int(value)
	case int32:
		return int(value)
	case int64:
		return int(value)
	case uint:
		return int(value)
	case uint8:
		return int(value)
	case uint16:
		return int(value)
	case uint32:
		return int(value)
	case uint64:
		return int(value)
	case float32:
		return int(value)
	case float64:
		return int(value)
	case json.Number:
		n, _ := value.Int64()
		return int(n)
	case string:
		if value == "" {
			return 0
		}
		if n, err := strconv.Atoi(value); err == nil {
			return n
		}
		if n, err := strconv.ParseFloat(value, 64); err == nil {
			return int(n)
		}
	}
	return 0
}

func apiBool(v interface{}) bool {
	switch value := v.(type) {
	case bool:
		return value
	case string:
		parsed, err := strconv.ParseBool(strings.TrimSpace(value))
		return err == nil && parsed
	case float64:
		return value != 0
	case int:
		return value != 0
	case json.Number:
		n, err := value.Int64()
		return err == nil && n != 0
	default:
		return false
	}
}

func apiTime(v interface{}) time.Time {
	switch value := v.(type) {
	case time.Time:
		return value
	case string:
		return parseAPIStringTime(value)
	case float64:
		return parseAPINumericTime(int64(value))
	case float32:
		return parseAPINumericTime(int64(value))
	case int:
		return parseAPINumericTime(int64(value))
	case int64:
		return parseAPINumericTime(value)
	case json.Number:
		if n, err := value.Int64(); err == nil {
			return parseAPINumericTime(n)
		}
	}
	return time.Time{}
}

func parseAPIStringTime(value string) time.Time {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return time.Time{}
	}
	layouts := []string{
		time.RFC3339,
		time.RFC3339Nano,
		"2006-01-02 15:04:05",
		"2006-01-02T15:04:05",
		"2006-01-02",
	}
	for _, layout := range layouts {
		if parsed, err := time.Parse(layout, trimmed); err == nil {
			return parsed
		}
	}
	if n, err := strconv.ParseInt(trimmed, 10, 64); err == nil {
		return parseAPINumericTime(n)
	}
	return time.Time{}
}

func parseAPINumericTime(n int64) time.Time {
	if n <= 0 {
		return time.Time{}
	}
	if n > 1_000_000_000_000 {
		return time.Unix(0, n*int64(time.Millisecond))
	}
	return time.Unix(n, 0)
}

func apiStringSlice(v interface{}) []string {
	switch value := v.(type) {
	case nil:
		return nil
	case []string:
		return append([]string(nil), value...)
	case []interface{}:
		out := make([]string, 0, len(value))
		for _, item := range value {
			if s := apiStringValue(item); s != "" {
				out = append(out, s)
			}
		}
		return out
	case string:
		if strings.TrimSpace(value) == "" {
			return nil
		}
		parts := strings.Split(value, ",")
		out := make([]string, 0, len(parts))
		for _, part := range parts {
			part = strings.TrimSpace(part)
			if part != "" {
				out = append(out, part)
			}
		}
		return out
	}
	return nil
}

func apiStringValue(v interface{}) string {
	switch value := v.(type) {
	case map[string]interface{}:
		for _, key := range []string{"name", "title", "login", "label", "text"} {
			if s := apiString(value[key]); s != "" {
				return s
			}
		}
		return ""
	default:
		return apiString(v)
	}
}

func apiAuthor(v interface{}) string {
	switch value := v.(type) {
	case map[string]interface{}:
		for _, key := range []string{"login", "name", "username", "full_name", "display_name"} {
			if s := apiString(value[key]); s != "" {
				return s
			}
		}
		return ""
	default:
		return apiString(v)
	}
}

func apiLatestTime(values ...time.Time) time.Time {
	var latest time.Time
	for _, value := range values {
		if value.IsZero() {
			continue
		}
		if latest.IsZero() || value.After(latest) {
			latest = value
		}
	}
	return latest
}

func apiAgeInDays(value time.Time) int {
	if value.IsZero() {
		return -1
	}
	return int(time.Since(value).Hours() / 24)
}
