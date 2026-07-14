package member

import (
	"encoding/csv"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gitlink-org/gitlink-cli/shortcuts/common"
)

type batchAddResult struct {
	User   string `json:"user" yaml:"user"`
	Action string `json:"action" yaml:"action"`
	Status string `json:"status" yaml:"status"`
	Error  string `json:"error,omitempty" yaml:"error,omitempty"`
}

type batchAddSummary struct {
	Owner     string            `json:"owner" yaml:"owner"`
	Repo      string            `json:"repo" yaml:"repo"`
	DryRun    bool              `json:"dry_run" yaml:"dry_run"`
	Total     int               `json:"total" yaml:"total"`
	Succeeded int               `json:"succeeded" yaml:"succeeded"`
	Failed    int               `json:"failed" yaml:"failed"`
	Duration  string            `json:"duration" yaml:"duration"`
	Results   []batchAddResult  `json:"results" yaml:"results"`
}

func batchAddShortcut() *common.Shortcut {
	return &common.Shortcut{
		Name:        "batch-add",
		Description: "批量添加成员到项目，支持逗号分隔列表或 CSV 文件",
		Flags: []common.Flag{
			{Name: "users", Short: "u", Usage: "逗号分隔的用户数字 ID，例如: 42,99,105"},
			{Name: "from", Usage: "从 CSV 文件读取用户 ID。支持 user_id/id/user 列名或无表头首列"},
			{Name: "dry-run", Usage: "仅预览将要添加的成员，不实际执行", Bool: true, Default: "false"},
		},
		Run: runBatchAdd,
	}
}

func runBatchAdd(ctx *common.RuntimeContext) error {
	start := time.Now()

	if err := ctx.ResolveOwnerRepo(); err != nil {
		return err
	}

	userIDs, err := collectUserIDs(ctx.Arg("users"), ctx.Arg("from"))
	if err != nil {
		return err
	}
	if len(userIDs) == 0 {
		return fmt.Errorf("未提供用户 ID，请使用 --users 42,99 或 --from users.csv")
	}

	dryRun := parseMemberBool(ctx.Arg("dry-run"))

	summary := batchAddSummary{
		Owner:   ctx.Owner,
		Repo:    ctx.Repo,
		DryRun:  dryRun,
		Total:   len(userIDs),
		Results: make([]batchAddResult, 0, len(userIDs)),
	}

	for _, uid := range userIDs {
		result := batchAddResult{User: uid, Action: "add"}
		if dryRun {
			result.Status = "planned"
			summary.Succeeded++
			summary.Results = append(summary.Results, result)
			continue
		}

		id, _ := strconv.ParseInt(uid, 10, 64)
		body := map[string]interface{}{"user_id": id}
		if _, err := ctx.CallAPI("POST", ctx.RepoPath()+"/collaborators", body); err != nil {
			result.Status = "failed"
			result.Error = err.Error()
			summary.Failed++
		} else {
			result.Status = "added"
			summary.Succeeded++
		}
		summary.Results = append(summary.Results, result)
	}

	summary.Duration = time.Since(start).String()

	if err := ctx.OutputData(summary); err != nil {
		return err
	}
	if summary.Failed > 0 {
		return fmt.Errorf("%d / %d 个成员添加失败", summary.Failed, summary.Total)
	}
	return nil
}

func collectUserIDs(usersValue, csvPath string) ([]string, error) {
	ids, err := parseUserIDList(usersValue)
	if err != nil {
		return nil, err
	}
	if csvPath == "" {
		return ids, nil
	}

	csvIDs, err := readUserIDsFromCSV(csvPath)
	if err != nil {
		return nil, err
	}
	return mergeUserIDLists(ids, csvIDs), nil
}

func parseUserIDList(value string) ([]string, error) {
	if strings.TrimSpace(value) == "" {
		return nil, nil
	}
	return normalizeUserIDs(strings.Split(value, ","))
}

func readUserIDsFromCSV(path string) ([]string, error) {
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

	idCol := -1
	startRow := 0
	for i, cell := range records[0] {
		switch strings.ToLower(strings.TrimSpace(cell)) {
		case "user_id", "id", "user", "uid":
			idCol = i
			startRow = 1
		}
	}
	if idCol == -1 {
		idCol = 0
	}

	values := make([]string, 0, len(records)-startRow)
	for _, record := range records[startRow:] {
		if idCol >= len(record) {
			continue
		}
		values = append(values, record[idCol])
	}
	return normalizeUserIDs(values)
}

func normalizeUserIDs(values []string) ([]string, error) {
	ids := make([]string, 0, len(values))
	seen := map[string]bool{}
	for _, value := range values {
		id := strings.TrimSpace(value)
		if id == "" {
			continue
		}
		if _, err := strconv.ParseInt(id, 10, 64); err != nil {
			return nil, fmt.Errorf("无效的用户 ID %q: 必须是整数", id)
		}
		if seen[id] {
			continue
		}
		seen[id] = true
		ids = append(ids, id)
	}
	return ids, nil
}

func mergeUserIDLists(values ...[]string) []string {
	merged := []string{}
	seen := map[string]bool{}
	for _, ids := range values {
		for _, id := range ids {
			if seen[id] {
				continue
			}
			seen[id] = true
			merged = append(merged, id)
		}
	}
	return merged
}

func parseMemberBool(value string) bool {
	parsed, err := strconv.ParseBool(strings.TrimSpace(value))
	return err == nil && parsed
}
