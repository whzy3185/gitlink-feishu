package org

import (
	"encoding/csv"
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gitlink-org/gitlink-cli/shortcuts/common"
)

type batchInviteResult struct {
	User   string `json:"user" yaml:"user"`
	Action string `json:"action" yaml:"action"`
	Status string `json:"status" yaml:"status"`
	Error  string `json:"error,omitempty" yaml:"error,omitempty"`
}

type batchInviteSummary struct {
	Org      string               `json:"org" yaml:"org"`
	DryRun   bool                 `json:"dry_run" yaml:"dry_run"`
	Total    int                  `json:"total" yaml:"total"`
	Succeeded int                 `json:"succeeded" yaml:"succeeded"`
	Failed   int                  `json:"failed" yaml:"failed"`
	Duration string               `json:"duration" yaml:"duration"`
	Results  []batchInviteResult  `json:"results" yaml:"results"`
}

func newBatchInviteShortcut() *common.Shortcut {
	return &common.Shortcut{
		Name:        "batch-invite",
		Description: "批量邀请成员加入组织，支持逗号分隔列表或 CSV 文件",
		Flags: []common.Flag{
			{Name: "id", Short: "i", Usage: "组织 ID 或 login", Required: true},
			{Name: "users", Short: "u", Usage: "逗号分隔的用户名或 ID，例如: alice,bob,charlie"},
			{Name: "from", Usage: "从 CSV 文件读取用户名。支持 user/login/user_id 列名或无表头首列"},
			{Name: "role", Short: "r", Usage: "成员角色: member 或 admin", Default: "member"},
			{Name: "dry-run", Usage: "仅预览将要邀请的成员，不实际执行", Bool: true, Default: "false"},
		},
		Run: runBatchInvite,
	}
}

func runBatchInvite(ctx *common.RuntimeContext) error {
	start := time.Now()

	orgID, err := ctx.RequireArg("id")
	if err != nil {
		return err
	}

	role := ctx.Arg("role")
	if role == "" {
		role = "member"
	}
	if role != "member" && role != "admin" {
		return fmt.Errorf("无效的角色 %q: 必须为 member 或 admin", role)
	}

	userInputs, err := collectUsers(ctx.Arg("users"), ctx.Arg("from"))
	if err != nil {
		return err
	}
	if len(userInputs) == 0 {
		return fmt.Errorf("未提供用户名，请使用 --users alice,bob 或 --from users.csv")
	}

	// 将用户名解析为数字 ID（如果传入的已经是数字则直接使用）
	resolvedUsers, resolveErrors := resolveUserIDs(ctx, userInputs)

	dryRun := parseBool(ctx.Arg("dry-run"))

	summary := batchInviteSummary{
		Org:     orgID,
		DryRun:  dryRun,
		Total:   len(userInputs),
		Results: make([]batchInviteResult, 0, len(userInputs)),
	}

	// 先记录解析失败的
	for input, errMsg := range resolveErrors {
		summary.Results = append(summary.Results, batchInviteResult{
			User:   input,
			Action: "invite",
			Status: "failed",
			Error:  errMsg,
		})
		summary.Failed++
	}

	for _, ru := range resolvedUsers {
		displayName := ru.Input
		if ru.Input != ru.UserID {
			displayName = fmt.Sprintf("%s (ID:%s)", ru.Input, ru.UserID)
		}
		result := batchInviteResult{User: displayName, Action: "invite"}
		if dryRun {
			result.Status = "planned"
			summary.Succeeded++
			summary.Results = append(summary.Results, result)
			continue
		}

		userIDInt, _ := strconv.ParseInt(ru.UserID, 10, 64)
		body := map[string]interface{}{
			"user_id": userIDInt,
			"role":    role,
		}
		path := fmt.Sprintf("/organizations/%s/organization_users", orgID)
		if _, err := ctx.CallAPI("POST", path, body); err != nil {
			result.Status = "failed"
			result.Error = err.Error()
			summary.Failed++
		} else {
			result.Status = "invited"
			summary.Succeeded++
		}
		summary.Results = append(summary.Results, result)
	}

	summary.Duration = time.Since(start).String()

	if err := ctx.OutputData(summary); err != nil {
		return err
	}
	if summary.Failed > 0 {
		return fmt.Errorf("%d / %d 个成员邀请失败", summary.Failed, summary.Total)
	}
	return nil
}

func collectUsers(usersValue, csvPath string) ([]string, error) {
	users, err := parseUserList(usersValue)
	if err != nil {
		return nil, err
	}
	if csvPath == "" {
		return users, nil
	}

	csvUsers, err := readUsersFromCSV(csvPath)
	if err != nil {
		return nil, err
	}
	return mergeUserLists(users, csvUsers), nil
}

func parseUserList(value string) ([]string, error) {
	if strings.TrimSpace(value) == "" {
		return nil, nil
	}
	return normalizeUserIDs(strings.Split(value, ","))
}

func readUsersFromCSV(path string) ([]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("读取 CSV 文件失败: %w", err)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	reader.TrimLeadingSpace = true
	records, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("解析 CSV 文件失败: %w", err)
	}
	if len(records) == 0 {
		return nil, nil
	}

	userCol := -1
	startRow := 0
	for i, cell := range records[0] {
		switch strings.ToLower(strings.TrimSpace(cell)) {
		case "user", "login", "user_id", "username":
			userCol = i
			startRow = 1
		}
	}
	if userCol == -1 {
		userCol = 0
	}

	values := make([]string, 0, len(records)-startRow)
	for _, record := range records[startRow:] {
		if userCol >= len(record) {
			continue
		}
		values = append(values, record[userCol])
	}
	return normalizeUserIDs(values)
}

func normalizeUserIDs(values []string) ([]string, error) {
	users := make([]string, 0, len(values))
	seen := map[string]bool{}
	for _, value := range values {
		user := strings.TrimSpace(value)
		if user == "" {
			continue
		}
		if seen[user] {
			continue
		}
		seen[user] = true
		users = append(users, user)
	}
	return users, nil
}

func parseBool(value string) bool {
	if strings.EqualFold(strings.TrimSpace(value), "true") {
		return true
	}
	return false
}

func mergeUserLists(values ...[]string) []string {
	merged := []string{}
	seen := map[string]bool{}
	for _, users := range values {
		for _, u := range users {
			if seen[u] {
				continue
			}
			seen[u] = true
			merged = append(merged, u)
		}
	}
	return merged
}

// resolvedUser holds the mapping from user input to numeric ID.
type resolvedUser struct {
	Input  string // original input (username or numeric string)
	UserID string // resolved numeric user ID
}

// resolveUserIDs converts usernames to numeric user IDs via the search API.
// If an input is already numeric, it is used directly.
func resolveUserIDs(ctx *common.RuntimeContext, inputs []string) ([]resolvedUser, map[string]string) {
	results := make([]resolvedUser, 0, len(inputs))
	errors := make(map[string]string)

	for _, input := range inputs {
		// 如果已经是纯数字，直接使用
		if _, err := strconv.Atoi(input); err == nil {
			results = append(results, resolvedUser{Input: input, UserID: input})
			continue
		}

		// 通过搜索 API 查找用户名对应的数字 ID
		q := url.Values{}
		q.Set("search", input)
		q.Set("limit", "5")
		env, err := ctx.CallAPIWithQuery("GET", "/users/list", q)
		if err != nil {
			errors[input] = fmt.Sprintf("查找用户失败: %v", err)
			continue
		}

		// env.Data 是 {"total_count":N, "users":[...]} 的 map 结构
		users := extractUsers(env.Data)
		if len(users) == 0 {
			errors[input] = fmt.Sprintf("未找到用户 %q", input)
			continue
		}

		// 精确匹配用户名
		matched := users[0]
		for _, u := range users {
			if u.Login == input {
				matched = u
				break
			}
		}

		results = append(results, resolvedUser{
			Input:  input,
			UserID: strconv.Itoa(matched.UserID),
		})
	}

	return results, errors
}

// searchUser holds a parsed user from search results.
type searchUser struct {
	Login  string
	UserID int
}

// extractUsers extracts the user list from the search API response data.
// data is expected to be map[string]interface{} with a "users" key containing a slice.
func extractUsers(data interface{}) []searchUser {
	if data == nil {
		return nil
	}
	m, ok := data.(map[string]interface{})
	if !ok {
		return nil
	}
	rawUsers, ok := m["users"]
	if !ok {
		return nil
	}
	usersSlice, ok := rawUsers.([]interface{})
	if !ok {
		return nil
	}
	var result []searchUser
	for _, item := range usersSlice {
		um, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		login, _ := um["login"].(string)
		userID := 0
		switch v := um["user_id"].(type) {
		case float64:
			userID = int(v)
		case int:
			userID = v
		case string:
			userID, _ = strconv.Atoi(v)
		}
		if login != "" && userID > 0 {
			result = append(result, searchUser{Login: login, UserID: userID})
		}
	}
	return result
}
