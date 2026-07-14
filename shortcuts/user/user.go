package user

import (
	"fmt"
	"net/url"
	"sort"
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
				env, err := ctx.CallAPI("GET", fmt.Sprintf("/users/%s", url.PathEscape(login)), nil)
				if err != nil {
					return err
				}
				return ctx.Output(env)
			},
		},
		{
			Name:        "headmap",
			Description: "Show user contribution heatmap and yearly totals",
			Flags: []common.Flag{
				{Name: "login", Short: "l", Usage: "User login name", Required: true},
				{Name: "year", Usage: "Contribution year, for example 2026"},
			},
			Run: func(ctx *common.RuntimeContext) error {
				login, err := ctx.RequireArg("login")
				if err != nil {
					return err
				}
				query, err := buildHeadmapQuery(ctx.Arg("year"))
				if err != nil {
					return err
				}
				env, err := ctx.CallAPIWithQuery("GET", fmt.Sprintf("/users/%s/headmaps", url.PathEscape(login)), query)
				if err != nil {
					return err
				}
				data, err := normalizeHeadmapData(login, ctx.Arg("year"), env.Data)
				if err != nil {
					return err
				}
				return ctx.OutputData(data)
			},
		},
		{
			Name:        "activity",
			Description: "Show recent user activity timeline across commits, issues, and pull requests",
			Flags: []common.Flag{
				{Name: "login", Short: "l", Usage: "User login name", Required: true},
			},
			Run: func(ctx *common.RuntimeContext) error {
				login, err := ctx.RequireArg("login")
				if err != nil {
					return err
				}
				env, err := ctx.CallAPI("GET", fmt.Sprintf("/users/%s/statistics/activity", url.PathEscape(login)), nil)
				if err != nil {
					return err
				}
				data, err := normalizeActivityData(login, env.Data)
				if err != nil {
					return err
				}
				return ctx.OutputData(data)
			},
		},
		{
			Name:        "develop",
			Description: "Show user development capability scores and language distribution",
			Flags: []common.Flag{
				{Name: "login", Short: "l", Usage: "User login name", Required: true},
				{Name: "start-time", Usage: "Start Unix timestamp"},
				{Name: "end-time", Usage: "End Unix timestamp"},
			},
			Run: func(ctx *common.RuntimeContext) error {
				login, err := ctx.RequireArg("login")
				if err != nil {
					return err
				}
				query, period, err := buildTimeRangeQuery(ctx.Arg("start-time"), ctx.Arg("end-time"))
				if err != nil {
					return err
				}
				env, err := ctx.CallAPIWithQuery("GET", fmt.Sprintf("/users/%s/statistics/develop", url.PathEscape(login)), query)
				if err != nil {
					return err
				}
				data, err := normalizeDevelopData(login, period, env.Data)
				if err != nil {
					return err
				}
				return ctx.OutputData(data)
			},
		},
		{
			Name:        "roles",
			Description: "Show user project role distribution",
			Flags: []common.Flag{
				{Name: "login", Short: "l", Usage: "User login name", Required: true},
				{Name: "start-time", Usage: "Start Unix timestamp"},
				{Name: "end-time", Usage: "End Unix timestamp"},
			},
			Run: func(ctx *common.RuntimeContext) error {
				login, err := ctx.RequireArg("login")
				if err != nil {
					return err
				}
				query, period, err := buildTimeRangeQuery(ctx.Arg("start-time"), ctx.Arg("end-time"))
				if err != nil {
					return err
				}
				env, err := ctx.CallAPIWithQuery("GET", fmt.Sprintf("/users/%s/statistics/role", url.PathEscape(login)), query)
				if err != nil {
					return err
				}
				data, err := normalizeRoleData(login, period, env.Data)
				if err != nil {
					return err
				}
				return ctx.OutputData(data)
			},
		},
		{
			Name:        "majors",
			Description: "Show user major domain categories",
			Flags: []common.Flag{
				{Name: "login", Short: "l", Usage: "User login name", Required: true},
				{Name: "start-time", Usage: "Start Unix timestamp"},
				{Name: "end-time", Usage: "End Unix timestamp"},
			},
			Run: func(ctx *common.RuntimeContext) error {
				login, err := ctx.RequireArg("login")
				if err != nil {
					return err
				}
				query, period, err := buildTimeRangeQuery(ctx.Arg("start-time"), ctx.Arg("end-time"))
				if err != nil {
					return err
				}
				env, err := ctx.CallAPIWithQuery("GET", fmt.Sprintf("/users/%s/statistics/major", url.PathEscape(login)), query)
				if err != nil {
					return err
				}
				data, err := normalizeMajorData(login, period, env.Data)
				if err != nil {
					return err
				}
				return ctx.OutputData(data)
			},
		},
		{
			Name:        "trends",
			Description: "Show user project trends with optional cross-page filtering",
			Flags: []common.Flag{
				{Name: "login", Short: "l", Usage: "User login name", Required: true},
				{Name: "page", Short: "p", Usage: "Page number", Default: "1"},
				{Name: "limit", Short: "L", Usage: "Items per page", Default: "20"},
				{Name: "all", Usage: "Fetch all pages before filtering", Bool: true, Default: "false"},
				{Name: "trend-type", Usage: "Trend type filter, for example PullRequest or CommitLog"},
				{Name: "project-owner", Usage: "Project owner login filter"},
				{Name: "project", Usage: "Project identifier filter"},
				{Name: "keyword", Usage: "Keyword filter applied to trend title and action type"},
			},
			Run: func(ctx *common.RuntimeContext) error {
				login, err := ctx.RequireArg("login")
				if err != nil {
					return err
				}
				page, err := parsePositiveInt("page", ctx.Arg("page"))
				if err != nil {
					return err
				}
				limit, err := parsePositiveInt("limit", ctx.Arg("limit"))
				if err != nil {
					return err
				}
				filters := trendFilters{
					TrendType:    strings.TrimSpace(ctx.Arg("trend-type")),
					ProjectOwner: strings.TrimSpace(ctx.Arg("project-owner")),
					Project:      strings.TrimSpace(ctx.Arg("project")),
					Keyword:      strings.TrimSpace(ctx.Arg("keyword")),
				}
				fetchAll := parseBoolArg(ctx.Arg("all")) || filters.enabled()
				items, totalCount, err := fetchUserTrends(ctx, login, page, limit, fetchAll)
				if err != nil {
					return err
				}
				data, err := normalizeTrendData(login, page, limit, totalCount, fetchAll, filters, items)
				if err != nil {
					return err
				}
				return ctx.OutputData(data)
			},
		},
	}
}

type trendFilters struct {
	TrendType    string
	ProjectOwner string
	Project      string
	Keyword      string
}

func (f trendFilters) enabled() bool {
	return f.TrendType != "" || f.ProjectOwner != "" || f.Project != "" || f.Keyword != ""
}

func buildHeadmapQuery(year string) (url.Values, error) {
	q := url.Values{}
	if strings.TrimSpace(year) == "" {
		return q, nil
	}
	parsed, err := parsePositiveInt("year", year)
	if err != nil {
		return nil, err
	}
	q.Set("year", strconv.Itoa(parsed))
	return q, nil
}

func buildTimeRangeQuery(start, end string) (url.Values, map[string]interface{}, error) {
	q := url.Values{}
	period := map[string]interface{}{}

	if strings.TrimSpace(start) != "" {
		parsed, err := parsePositiveInt("start-time", start)
		if err != nil {
			return nil, nil, err
		}
		q.Set("start_time", strconv.Itoa(parsed))
		period["start_time"] = parsed
	}
	if strings.TrimSpace(end) != "" {
		parsed, err := parsePositiveInt("end-time", end)
		if err != nil {
			return nil, nil, err
		}
		q.Set("end_time", strconv.Itoa(parsed))
		period["end_time"] = parsed
	}
	if startValue, ok := period["start_time"].(int); ok {
		if endValue, ok := period["end_time"].(int); ok && startValue > endValue {
			return nil, nil, fmt.Errorf("--start-time must be less than or equal to --end-time")
		}
	}
	if len(period) == 0 {
		return q, nil, nil
	}
	return q, period, nil
}

func parsePositiveInt(flagName, value string) (int, error) {
	parsed, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil || parsed <= 0 {
		return 0, fmt.Errorf("--%s must be a positive integer", flagName)
	}
	return parsed, nil
}

func parseBoolArg(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "1", "true", "yes", "y", "on":
		return true
	default:
		return false
	}
}

func fetchUserTrends(ctx *common.RuntimeContext, login string, page, limit int, fetchAll bool) ([]interface{}, int, error) {
	if !fetchAll {
		env, err := ctx.CallAPIWithQuery("GET", fmt.Sprintf("/users/%s/project_trends", url.PathEscape(login)), url.Values{
			"page":  []string{strconv.Itoa(page)},
			"limit": []string{strconv.Itoa(limit)},
		})
		if err != nil {
			return nil, 0, err
		}
		return unwrapTrendResponse(env.Data)
	}

	var all []interface{}
	totalCount := 0
	currentPage := 1

	for {
		env, err := ctx.CallAPIWithQuery("GET", fmt.Sprintf("/users/%s/project_trends", url.PathEscape(login)), url.Values{
			"page":  []string{strconv.Itoa(currentPage)},
			"limit": []string{strconv.Itoa(limit)},
		})
		if err != nil {
			return nil, 0, err
		}
		items, count, err := unwrapTrendResponse(env.Data)
		if err != nil {
			return nil, 0, err
		}
		if totalCount == 0 {
			totalCount = count
		}
		if len(items) == 0 {
			break
		}
		all = append(all, items...)
		if len(items) < limit || (totalCount > 0 && len(all) >= totalCount) {
			break
		}
		currentPage++
	}

	return all, totalCount, nil
}

func unwrapTrendResponse(data interface{}) ([]interface{}, int, error) {
	m, err := asMap(data)
	if err != nil {
		return nil, 0, err
	}
	items, err := asSlice(m["project_trends"])
	if err != nil {
		return nil, 0, err
	}
	return items, asInt(m["total_count"]), nil
}

func normalizeHeadmapData(login, year string, data interface{}) (map[string]interface{}, error) {
	m, err := asMap(data)
	if err != nil {
		return nil, err
	}
	items, err := asSlice(m["headmaps"])
	if err != nil {
		return nil, err
	}

	headmaps := make([]map[string]interface{}, 0, len(items))
	activeDays := 0
	busiest := map[string]interface{}{}
	maxContributions := -1

	for _, item := range items {
		entry, err := asMap(item)
		if err != nil {
			return nil, err
		}
		normalized := map[string]interface{}{
			"date":          strings.TrimSpace(asString(entry["date"])),
			"contributions": asInt(entry["contributions"]),
		}
		headmaps = append(headmaps, normalized)
		if normalized["contributions"].(int) > 0 {
			activeDays++
		}
		if normalized["contributions"].(int) > maxContributions {
			maxContributions = normalized["contributions"].(int)
			busiest = map[string]interface{}{
				"date":          normalized["date"],
				"contributions": normalized["contributions"],
			}
		}
	}

	result := map[string]interface{}{
		"login":               login,
		"total_contributions": asInt(m["total_contributions"]),
		"active_days":         activeDays,
		"headmaps":            headmaps,
	}
	if strings.TrimSpace(year) != "" {
		result["year"] = asInt(year)
	}
	if len(busiest) > 0 {
		result["busiest_day"] = busiest
	}
	return result, nil
}

func normalizeActivityData(login string, data interface{}) (map[string]interface{}, error) {
	m, err := asMap(data)
	if err != nil {
		return nil, err
	}
	dates := asStringSlice(m["dates"])
	commits := asIntSlice(m["commits_count"])
	issues := asIntSlice(m["issues_count"])
	prs := asIntSlice(m["pull_requests_count"])

	seriesLength := len(dates)
	if len(commits) > seriesLength {
		seriesLength = len(commits)
	}
	if len(issues) > seriesLength {
		seriesLength = len(issues)
	}
	if len(prs) > seriesLength {
		seriesLength = len(prs)
	}

	timeline := make([]map[string]interface{}, 0, seriesLength)
	totalCommits := 0
	totalIssues := 0
	totalPRs := 0
	peakDay := map[string]interface{}{}
	peakTotal := -1

	for i := 0; i < seriesLength; i++ {
		entry := map[string]interface{}{
			"date":          stringAt(dates, i),
			"commits":       intAt(commits, i),
			"issues":        intAt(issues, i),
			"pull_requests": intAt(prs, i),
		}
		entry["total"] = entry["commits"].(int) + entry["issues"].(int) + entry["pull_requests"].(int)
		totalCommits += entry["commits"].(int)
		totalIssues += entry["issues"].(int)
		totalPRs += entry["pull_requests"].(int)
		if entry["total"].(int) > peakTotal {
			peakTotal = entry["total"].(int)
			peakDay = entry
		}
		timeline = append(timeline, entry)
	}

	return map[string]interface{}{
		"login":       login,
		"period_days": len(timeline),
		"totals": map[string]interface{}{
			"commits":       totalCommits,
			"issues":        totalIssues,
			"pull_requests": totalPRs,
			"all":           totalCommits + totalIssues + totalPRs,
		},
		"peak_day": peakDay,
		"timeline": timeline,
	}, nil
}

func normalizeDevelopData(login string, period map[string]interface{}, data interface{}) (map[string]interface{}, error) {
	m, err := asMap(data)
	if err != nil {
		return nil, err
	}
	platform, err := asMap(m["platform"])
	if err != nil {
		return nil, err
	}
	userScores, err := asMap(m["user"])
	if err != nil {
		return nil, err
	}
	languages := mergeLanguageStats(userScores["languages_percent"], userScores["each_language_score"])

	result := map[string]interface{}{
		"login":           login,
		"platform_scores": normalizeScoreMap(platform),
		"user_scores":     normalizeScoreMap(userScores),
		"languages":       languages,
	}
	if len(languages) > 0 {
		result["top_language"] = languages[0]
	}
	if period != nil {
		result["period"] = period
	}
	return result, nil
}

func normalizeRoleData(login string, period map[string]interface{}, data interface{}) (map[string]interface{}, error) {
	m, err := asMap(data)
	if err != nil {
		return nil, err
	}
	roleMap, err := asMap(m["role"])
	if err != nil {
		return nil, err
	}
	roles := make([]map[string]interface{}, 0, len(roleMap))
	for name, value := range roleMap {
		entry, err := asMap(value)
		if err != nil {
			return nil, err
		}
		roles = append(roles, map[string]interface{}{
			"name":    name,
			"count":   asInt(entry["count"]),
			"percent": asFloat(entry["percent"]),
		})
	}
	sort.Slice(roles, func(i, j int) bool {
		if roles[i]["count"].(int) == roles[j]["count"].(int) {
			return roles[i]["name"].(string) < roles[j]["name"].(string)
		}
		return roles[i]["count"].(int) > roles[j]["count"].(int)
	})

	result := map[string]interface{}{
		"login":                login,
		"total_projects_count": asInt(m["total_projects_count"]),
		"roles":                roles,
	}
	if len(roles) > 0 {
		result["primary_role"] = roles[0]
	}
	if period != nil {
		result["period"] = period
	}
	return result, nil
}

func normalizeMajorData(login string, period map[string]interface{}, data interface{}) (map[string]interface{}, error) {
	m, err := asMap(data)
	if err != nil {
		return nil, err
	}
	categories := asStringSlice(m["categories"])
	result := map[string]interface{}{
		"login":          login,
		"category_count": len(categories),
		"categories":     categories,
	}
	if len(categories) > 0 {
		result["primary_category"] = categories[0]
	}
	if period != nil {
		result["period"] = period
	}
	return result, nil
}

func normalizeTrendData(login string, page, limit, totalCount int, fetchedAll bool, filters trendFilters, items []interface{}) (map[string]interface{}, error) {
	normalized := make([]map[string]interface{}, 0, len(items))
	countsByType := map[string]int{}

	for _, item := range items {
		entry, err := normalizeTrendItem(item)
		if err != nil {
			return nil, err
		}
		if !matchesTrendFilters(entry, filters) {
			continue
		}
		countsByType[entry["trend_type"].(string)]++
		normalized = append(normalized, entry)
	}

	result := map[string]interface{}{
		"login":          login,
		"page":           page,
		"limit":          limit,
		"fetched_all":    fetchedAll,
		"total_count":    totalCount,
		"matched_count":  len(normalized),
		"counts_by_type": countsByType,
		"items":          normalized,
	}
	if filters.enabled() {
		filterMap := map[string]interface{}{}
		if filters.TrendType != "" {
			filterMap["trend_type"] = filters.TrendType
		}
		if filters.ProjectOwner != "" {
			filterMap["project_owner"] = filters.ProjectOwner
		}
		if filters.Project != "" {
			filterMap["project"] = filters.Project
		}
		if filters.Keyword != "" {
			filterMap["keyword"] = filters.Keyword
		}
		result["filters"] = filterMap
	}
	return result, nil
}

func normalizeTrendItem(item interface{}) (map[string]interface{}, error) {
	m, err := asMap(item)
	if err != nil {
		return nil, err
	}
	project, _ := asMap(m["project"])
	projectOwner, _ := asMap(project["owner"])
	commitLog, _ := asMap(m["commit_log"])

	entry := map[string]interface{}{
		"id":                 asInt(m["id"]),
		"trend_id":           asInt(m["trend_id"]),
		"trend_type":         strings.TrimSpace(asString(m["trend_type"])),
		"name":               strings.TrimSpace(asString(m["name"])),
		"action_type":        strings.TrimSpace(asString(m["action_type"])),
		"action_time":        strings.TrimSpace(asString(m["action_time"])),
		"created_at":         strings.TrimSpace(asString(m["created_at"])),
		"user_login":         strings.TrimSpace(asString(m["user_login"])),
		"user_name":          strings.TrimSpace(asString(m["user_name"])),
		"project_identifier": strings.TrimSpace(asString(project["identifier"])),
		"project_owner":      strings.TrimSpace(asString(projectOwner["login"])),
	}
	if description := strings.TrimSpace(asString(project["description"])); description != "" {
		entry["project_description"] = description
	}
	if ref := strings.TrimSpace(asString(commitLog["ref"])); ref != "" {
		entry["commit_ref"] = ref
	}
	if commitID := strings.TrimSpace(asString(commitLog["commit_id"])); commitID != "" {
		entry["commit_id"] = commitID
	}
	return entry, nil
}

func matchesTrendFilters(entry map[string]interface{}, filters trendFilters) bool {
	if filters.TrendType != "" && !strings.EqualFold(entry["trend_type"].(string), filters.TrendType) {
		return false
	}
	if filters.ProjectOwner != "" && !strings.EqualFold(entry["project_owner"].(string), filters.ProjectOwner) {
		return false
	}
	if filters.Project != "" && !strings.EqualFold(entry["project_identifier"].(string), filters.Project) {
		return false
	}
	if filters.Keyword != "" {
		haystack := strings.ToLower(strings.Join([]string{
			entry["name"].(string),
			entry["action_type"].(string),
			entry["project_identifier"].(string),
		}, " "))
		if !strings.Contains(haystack, strings.ToLower(filters.Keyword)) {
			return false
		}
	}
	return true
}

func mergeLanguageStats(percentValue interface{}, scoreValue interface{}) []map[string]interface{} {
	percentMap := asFloatMap(percentValue)
	scoreMap := asIntMap(scoreValue)
	keys := map[string]struct{}{}
	for key := range percentMap {
		keys[key] = struct{}{}
	}
	for key := range scoreMap {
		keys[key] = struct{}{}
	}

	languages := make([]map[string]interface{}, 0, len(keys))
	for key := range keys {
		languages = append(languages, map[string]interface{}{
			"name":    key,
			"percent": percentMap[key],
			"score":   scoreMap[key],
		})
	}
	sort.Slice(languages, func(i, j int) bool {
		if languages[i]["percent"].(float64) == languages[j]["percent"].(float64) {
			return languages[i]["name"].(string) < languages[j]["name"].(string)
		}
		return languages[i]["percent"].(float64) > languages[j]["percent"].(float64)
	})
	return languages
}

func normalizeScoreMap(values map[string]interface{}) map[string]interface{} {
	scores := map[string]interface{}{}
	for key, value := range values {
		if key == "languages_percent" || key == "each_language_score" {
			continue
		}
		scores[key] = asInt(value)
	}
	return scores
}

func asMap(value interface{}) (map[string]interface{}, error) {
	m, ok := value.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("expected object response, got %T", value)
	}
	return m, nil
}

func asSlice(value interface{}) ([]interface{}, error) {
	s, ok := value.([]interface{})
	if !ok {
		return nil, fmt.Errorf("expected array response, got %T", value)
	}
	return s, nil
}

func asString(value interface{}) string {
	if value == nil {
		return ""
	}
	switch v := value.(type) {
	case string:
		return v
	default:
		return fmt.Sprintf("%v", value)
	}
}

func asInt(value interface{}) int {
	switch v := value.(type) {
	case int:
		return v
	case int32:
		return int(v)
	case int64:
		return int(v)
	case float32:
		return int(v)
	case float64:
		return int(v)
	case string:
		parsed, _ := strconv.Atoi(strings.TrimSpace(v))
		return parsed
	default:
		return 0
	}
}

func asFloat(value interface{}) float64 {
	switch v := value.(type) {
	case float64:
		return v
	case float32:
		return float64(v)
	case int:
		return float64(v)
	case int32:
		return float64(v)
	case int64:
		return float64(v)
	case string:
		parsed, _ := strconv.ParseFloat(strings.TrimSpace(v), 64)
		return parsed
	default:
		return 0
	}
}

func asFloatMap(value interface{}) map[string]float64 {
	m, ok := value.(map[string]interface{})
	if !ok {
		return map[string]float64{}
	}
	result := make(map[string]float64, len(m))
	for key, item := range m {
		result[key] = asFloat(item)
	}
	return result
}

func asIntMap(value interface{}) map[string]int {
	m, ok := value.(map[string]interface{})
	if !ok {
		return map[string]int{}
	}
	result := make(map[string]int, len(m))
	for key, item := range m {
		result[key] = asInt(item)
	}
	return result
}

func asStringSlice(value interface{}) []string {
	items, ok := value.([]interface{})
	if !ok {
		return nil
	}
	result := make([]string, 0, len(items))
	for _, item := range items {
		result = append(result, strings.TrimSpace(asString(item)))
	}
	return result
}

func asIntSlice(value interface{}) []int {
	items, ok := value.([]interface{})
	if !ok {
		return nil
	}
	result := make([]int, 0, len(items))
	for _, item := range items {
		result = append(result, asInt(item))
	}
	return result
}

func stringAt(values []string, index int) string {
	if index < len(values) {
		return values[index]
	}
	return ""
}

func intAt(values []int, index int) int {
	if index < len(values) {
		return values[index]
	}
	return 0
}

func shortcutTranslator(translators ...*i18n.Translator) *i18n.Translator {
	if len(translators) > 0 && translators[0] != nil {
		return translators[0]
	}
	return i18n.Default()
}
