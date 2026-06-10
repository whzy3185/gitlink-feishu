package org

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
			Name:        "list",
			Description: tr.T("cmd.org.list.short"),
			Flags: []common.Flag{
				{Name: "page", Short: "p", Usage: tr.T("flag.page"), Default: "1"},
				{Name: "limit", Short: "l", Usage: tr.T("flag.limit"), Default: "20"},
			},
			Run: func(ctx *common.RuntimeContext) error {
				page, err := parsePositiveInt("page", ctx.Arg("page"))
				if err != nil {
					return err
				}
				limit, err := parsePositiveInt("limit", ctx.Arg("limit"))
				if err != nil {
					return err
				}
				q := url.Values{}
				q.Set("page", strconv.Itoa(page))
				q.Set("limit", strconv.Itoa(limit))
				env, err := ctx.CallAPIWithQuery("GET", "/organizations", q)
				if err != nil {
					return err
				}
				return ctx.Output(env)
			},
		},
		{
			Name:        "info",
			Description: tr.T("cmd.org.info.short"),
			Flags: []common.Flag{
				{Name: "id", Short: "i", Usage: tr.T("flag.org.id_or_login"), Required: true},
			},
			Run: func(ctx *common.RuntimeContext) error {
				id, err := ctx.RequireArg("id")
				if err != nil {
					return err
				}
				env, err := ctx.CallAPI("GET", fmt.Sprintf("/organizations/%s", url.PathEscape(id)), nil)
				if err != nil {
					return err
				}
				return ctx.Output(env)
			},
		},
		{
			Name:        "members",
			Description: "List organization members with team and keyword filters",
			Flags: []common.Flag{
				{Name: "id", Short: "i", Usage: tr.T("flag.org.id"), Required: true},
				{Name: "page", Short: "p", Usage: tr.T("flag.page"), Default: "1"},
				{Name: "limit", Short: "l", Usage: tr.T("flag.limit"), Default: "20"},
				{Name: "all", Usage: "Fetch all pages before filtering", Bool: true, Default: "false"},
				{Name: "team", Usage: "Filter by team name"},
				{Name: "login", Usage: "Filter by member login"},
				{Name: "keyword", Usage: "Keyword filter against login, name, mail, and teams"},
			},
			Run: func(ctx *common.RuntimeContext) error {
				id, err := ctx.RequireArg("id")
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
				filters := memberFilters{
					Team:    strings.TrimSpace(ctx.Arg("team")),
					Login:   strings.TrimSpace(ctx.Arg("login")),
					Keyword: strings.TrimSpace(ctx.Arg("keyword")),
				}
				fetchAll := parseBoolArg(ctx.Arg("all")) || filters.enabled()
				items, totalCount, err := fetchOrganizationMembers(ctx, id, page, limit, fetchAll)
				if err != nil {
					return err
				}
				data, err := normalizeOrganizationMembers(id, page, limit, totalCount, fetchAll, filters, items)
				if err != nil {
					return err
				}
				return ctx.OutputData(data)
			},
		},
		{
			Name:        "teams",
			Description: "List organization teams with permission and keyword filters",
			Flags: []common.Flag{
				{Name: "id", Short: "i", Usage: tr.T("flag.org.id"), Required: true},
				{Name: "authorize", Usage: "Filter by authorize level: owner, admin, write, read"},
				{Name: "unit", Usage: "Filter teams containing a specific unit"},
				{Name: "keyword", Usage: "Keyword filter against team name, nickname, and description"},
				{Name: "include-users", Usage: "Include normalized member details for each team", Bool: true, Default: "false"},
			},
			Run: func(ctx *common.RuntimeContext) error {
				id, err := ctx.RequireArg("id")
				if err != nil {
					return err
				}
				env, err := ctx.CallAPI("GET", fmt.Sprintf("/organizations/%s/teams", url.PathEscape(id)), nil)
				if err != nil {
					return err
				}
				filters := teamFilters{
					Authorize: strings.TrimSpace(ctx.Arg("authorize")),
					Unit:      strings.TrimSpace(ctx.Arg("unit")),
					Keyword:   strings.TrimSpace(ctx.Arg("keyword")),
				}
				data, err := normalizeTeams(id, parseBoolArg(ctx.Arg("include-users")), filters, env.Data)
				if err != nil {
					return err
				}
				return ctx.OutputData(data)
			},
		},
		{
			Name:        "create",
			Description: tr.T("cmd.org.create.short"),
			Flags: []common.Flag{
				{Name: "name", Short: "n", Usage: tr.T("flag.org.name"), Required: true},
				{Name: "description", Short: "d", Usage: tr.T("flag.description")},
			},
			Run: func(ctx *common.RuntimeContext) error {
				name, err := ctx.RequireArg("name")
				if err != nil {
					return err
				}
				payload := map[string]interface{}{
					"name": name,
				}
				if d := ctx.Arg("description"); d != "" {
					payload["description"] = d
				}
				env, err := ctx.CallAPI("POST", "/organizations", payload)
				if err != nil {
					return err
				}
				return ctx.Output(env)
			},
		},
		{
			Name:        "team-create",
			Description: "Create an organization team with dry-run preview",
			Flags: []common.Flag{
				{Name: "id", Short: "i", Usage: tr.T("flag.org.id"), Required: true},
				{Name: "name", Short: "n", Usage: "Team name", Required: true},
				{Name: "nickname", Usage: "Team display name"},
				{Name: "description", Short: "d", Usage: tr.T("flag.description")},
				{Name: "dry-run", Usage: "Preview the request without creating the team", Bool: true, Default: "false"},
			},
			Run: func(ctx *common.RuntimeContext) error {
				id, err := ctx.RequireArg("id")
				if err != nil {
					return err
				}
				name, err := ctx.RequireArg("name")
				if err != nil {
					return err
				}
				payload := map[string]interface{}{
					"name": name,
				}
				if nickname := strings.TrimSpace(ctx.Arg("nickname")); nickname != "" {
					payload["nickname"] = nickname
				}
				if description := strings.TrimSpace(ctx.Arg("description")); description != "" {
					payload["description"] = description
				}
				path := fmt.Sprintf("/organizations/%s/teams", url.PathEscape(id))
				if parseBoolArg(ctx.Arg("dry-run")) {
					return ctx.OutputData(map[string]interface{}{
						"dry_run": true,
						"method":  "POST",
						"path":    path,
						"payload": payload,
					})
				}
				env, err := ctx.CallAPI("POST", path, payload)
				if err != nil {
					return err
				}
				return ctx.Output(env)
			},
		},
		{
			Name:        "member-remove",
			Description: "Remove an organization member by user ID or login with dry-run preview",
			Flags: []common.Flag{
				{Name: "id", Short: "i", Usage: tr.T("flag.org.id"), Required: true},
				{Name: "user-id", Usage: "Organization member user ID"},
				{Name: "login", Usage: "Organization member login name"},
				{Name: "dry-run", Usage: "Preview the request without removing the member", Bool: true, Default: "false"},
			},
			Run: func(ctx *common.RuntimeContext) error {
				id, err := ctx.RequireArg("id")
				if err != nil {
					return err
				}
				userID, member, err := resolveOrganizationMember(ctx, id, ctx.Arg("user-id"), ctx.Arg("login"))
				if err != nil {
					return err
				}
				path := fmt.Sprintf("/organizations/%s/organization_users/%d", url.PathEscape(id), userID)
				if parseBoolArg(ctx.Arg("dry-run")) {
					return ctx.OutputData(map[string]interface{}{
						"dry_run": true,
						"method":  "DELETE",
						"path":    path,
						"member":  member,
					})
				}
				env, err := ctx.CallAPI("DELETE", path, nil)
				if err != nil {
					return err
				}
				return ctx.Output(env)
			},
		},
	}
}

type memberFilters struct {
	Team    string
	Login   string
	Keyword string
}

func (f memberFilters) enabled() bool {
	return f.Team != "" || f.Login != "" || f.Keyword != ""
}

type teamFilters struct {
	Authorize string
	Unit      string
	Keyword   string
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

func fetchOrganizationMembers(ctx *common.RuntimeContext, orgID string, page, limit int, fetchAll bool) ([]interface{}, int, error) {
	if !fetchAll {
		env, err := ctx.CallAPIWithQuery("GET", fmt.Sprintf("/organizations/%s/organization_users", url.PathEscape(orgID)), url.Values{
			"page":  []string{strconv.Itoa(page)},
			"limit": []string{strconv.Itoa(limit)},
		})
		if err != nil {
			return nil, 0, err
		}
		return unwrapOrganizationUsers(env.Data)
	}

	var all []interface{}
	totalCount := 0
	currentPage := 1

	for {
		env, err := ctx.CallAPIWithQuery("GET", fmt.Sprintf("/organizations/%s/organization_users", url.PathEscape(orgID)), url.Values{
			"page":  []string{strconv.Itoa(currentPage)},
			"limit": []string{strconv.Itoa(limit)},
		})
		if err != nil {
			return nil, 0, err
		}
		items, count, err := unwrapOrganizationUsers(env.Data)
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

func unwrapOrganizationUsers(data interface{}) ([]interface{}, int, error) {
	m, err := asMap(data)
	if err != nil {
		return nil, 0, err
	}
	items, err := asSlice(m["organization_users"])
	if err != nil {
		return nil, 0, err
	}
	return items, asInt(m["total_count"]), nil
}

func normalizeOrganizationMembers(orgID string, page, limit, totalCount int, fetchedAll bool, filters memberFilters, items []interface{}) (map[string]interface{}, error) {
	members := make([]map[string]interface{}, 0, len(items))
	countsByTeam := map[string]int{}

	for _, item := range items {
		member, err := normalizeMember(item)
		if err != nil {
			return nil, err
		}
		if !matchesMemberFilters(member, filters) {
			continue
		}
		for _, teamName := range member["team_names"].([]string) {
			countsByTeam[teamName]++
		}
		members = append(members, member)
	}

	sort.Slice(members, func(i, j int) bool {
		return members[i]["login"].(string) < members[j]["login"].(string)
	})

	result := map[string]interface{}{
		"organization":       orgID,
		"page":               page,
		"limit":              limit,
		"fetched_all":        fetchedAll,
		"total_count":        totalCount,
		"matched_count":      len(members),
		"counts_by_team":     countsByTeam,
		"organization_users": members,
	}
	if filters.enabled() {
		filterMap := map[string]interface{}{}
		if filters.Team != "" {
			filterMap["team"] = filters.Team
		}
		if filters.Login != "" {
			filterMap["login"] = filters.Login
		}
		if filters.Keyword != "" {
			filterMap["keyword"] = filters.Keyword
		}
		result["filters"] = filterMap
	}
	return result, nil
}

func normalizeTeams(orgID string, includeUsers bool, filters teamFilters, data interface{}) (map[string]interface{}, error) {
	m, err := asMap(data)
	if err != nil {
		return nil, err
	}
	items, err := asSlice(m["teams"])
	if err != nil {
		return nil, err
	}

	teams := make([]map[string]interface{}, 0, len(items))
	countsByAuthorize := map[string]int{}

	for _, item := range items {
		team, err := normalizeTeam(item, includeUsers)
		if err != nil {
			return nil, err
		}
		if !matchesTeamFilters(team, filters) {
			continue
		}
		countsByAuthorize[team["authorize"].(string)]++
		teams = append(teams, team)
	}

	sort.Slice(teams, func(i, j int) bool {
		return teams[i]["name"].(string) < teams[j]["name"].(string)
	})

	result := map[string]interface{}{
		"organization":        orgID,
		"total_count":         asInt(m["total_count"]),
		"matched_count":       len(teams),
		"include_users":       includeUsers,
		"counts_by_authorize": countsByAuthorize,
		"teams":               teams,
	}
	if filters.Authorize != "" || filters.Unit != "" || filters.Keyword != "" {
		filterMap := map[string]interface{}{}
		if filters.Authorize != "" {
			filterMap["authorize"] = filters.Authorize
		}
		if filters.Unit != "" {
			filterMap["unit"] = filters.Unit
		}
		if filters.Keyword != "" {
			filterMap["keyword"] = filters.Keyword
		}
		result["filters"] = filterMap
	}
	return result, nil
}

func normalizeTeam(value interface{}, includeUsers bool) (map[string]interface{}, error) {
	m, err := asMap(value)
	if err != nil {
		return nil, err
	}
	units := asStringSlice(m["units"])
	users, err := asSlice(m["users"])
	if err != nil && m["users"] != nil {
		return nil, err
	}

	memberLogins := make([]string, 0, len(users))
	normalizedUsers := make([]map[string]interface{}, 0, len(users))

	for _, user := range users {
		userMap, err := normalizeTeamUser(user)
		if err != nil {
			return nil, err
		}
		memberLogins = append(memberLogins, userMap["login"].(string))
		if includeUsers {
			normalizedUsers = append(normalizedUsers, userMap)
		}
	}
	sort.Strings(memberLogins)

	result := map[string]interface{}{
		"id":                     asInt(m["id"]),
		"name":                   strings.TrimSpace(asString(m["name"])),
		"nickname":               strings.TrimSpace(asString(m["nickname"])),
		"description":            strings.TrimSpace(asString(m["description"])),
		"authorize":              strings.TrimSpace(asString(m["authorize"])),
		"includes_all_projects":  asBool(m["includes_all_project"]),
		"can_create_org_project": asBool(m["can_create_org_project"]),
		"num_projects":           asInt(m["num_projects"]),
		"num_users":              asInt(m["num_users"]),
		"units":                  units,
		"member_logins":          memberLogins,
	}
	if includeUsers {
		sort.Slice(normalizedUsers, func(i, j int) bool {
			return normalizedUsers[i]["login"].(string) < normalizedUsers[j]["login"].(string)
		})
		result["users"] = normalizedUsers
	}
	return result, nil
}

func normalizeMember(value interface{}) (map[string]interface{}, error) {
	m, err := asMap(value)
	if err != nil {
		return nil, err
	}
	user, err := asMap(m["user"])
	if err != nil {
		return nil, err
	}
	teamNames := asStringSlice(m["team_names"])
	sort.Strings(teamNames)
	return map[string]interface{}{
		"id":         asInt(m["id"]),
		"login":      strings.TrimSpace(asString(user["login"])),
		"name":       strings.TrimSpace(asString(user["name"])),
		"identity":   strings.TrimSpace(asString(user["identity"])),
		"mail":       strings.TrimSpace(asString(user["mail"])),
		"joined_at":  strings.TrimSpace(asString(m["created_at"])),
		"team_names": teamNames,
		"team_count": len(teamNames),
	}, nil
}

func normalizeTeamUser(value interface{}) (map[string]interface{}, error) {
	m, err := asMap(value)
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{
		"user_id":  asInt(m["user_id"]),
		"login":    strings.TrimSpace(asString(m["login"])),
		"name":     strings.TrimSpace(asString(m["name"])),
		"identity": strings.TrimSpace(asString(m["identity"])),
		"mail":     strings.TrimSpace(asString(m["mail"])),
	}, nil
}

func matchesMemberFilters(member map[string]interface{}, filters memberFilters) bool {
	if filters.Team != "" {
		matchedTeam := false
		for _, teamName := range member["team_names"].([]string) {
			if strings.EqualFold(teamName, filters.Team) {
				matchedTeam = true
				break
			}
		}
		if !matchedTeam {
			return false
		}
	}
	if filters.Login != "" && !strings.EqualFold(member["login"].(string), filters.Login) {
		return false
	}
	if filters.Keyword != "" {
		haystack := strings.ToLower(strings.Join([]string{
			member["login"].(string),
			member["name"].(string),
			member["mail"].(string),
			strings.Join(member["team_names"].([]string), " "),
		}, " "))
		if !strings.Contains(haystack, strings.ToLower(filters.Keyword)) {
			return false
		}
	}
	return true
}

func matchesTeamFilters(team map[string]interface{}, filters teamFilters) bool {
	if filters.Authorize != "" && !strings.EqualFold(team["authorize"].(string), filters.Authorize) {
		return false
	}
	if filters.Unit != "" {
		found := false
		for _, unit := range team["units"].([]string) {
			if strings.EqualFold(unit, filters.Unit) {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	if filters.Keyword != "" {
		haystack := strings.ToLower(strings.Join([]string{
			team["name"].(string),
			team["nickname"].(string),
			team["description"].(string),
		}, " "))
		if !strings.Contains(haystack, strings.ToLower(filters.Keyword)) {
			return false
		}
	}
	return true
}

func resolveOrganizationMember(ctx *common.RuntimeContext, orgID, userIDValue, login string) (int, map[string]interface{}, error) {
	if strings.TrimSpace(userIDValue) != "" {
		userID, err := parsePositiveInt("user-id", userIDValue)
		if err != nil {
			return 0, nil, err
		}
		member := map[string]interface{}{
			"id":    userID,
			"login": strings.TrimSpace(login),
		}
		return userID, member, nil
	}
	if strings.TrimSpace(login) == "" {
		return 0, nil, fmt.Errorf("one of --user-id or --login is required")
	}
	items, _, err := fetchOrganizationMembers(ctx, orgID, 1, 100, true)
	if err != nil {
		return 0, nil, err
	}
	for _, item := range items {
		member, err := normalizeMember(item)
		if err != nil {
			return 0, nil, err
		}
		if strings.EqualFold(member["login"].(string), strings.TrimSpace(login)) {
			return member["id"].(int), member, nil
		}
	}
	return 0, nil, fmt.Errorf("organization member %q was not found", login)
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

func asBool(value interface{}) bool {
	switch v := value.(type) {
	case bool:
		return v
	case string:
		return parseBoolArg(v)
	default:
		return false
	}
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

func shortcutTranslator(translators ...*i18n.Translator) *i18n.Translator {
	if len(translators) > 0 && translators[0] != nil {
		return translators[0]
	}
	return i18n.Default()
}
