package repo

import (
	"encoding/csv"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/gitlink-org/gitlink-cli/shortcuts/common"
)

type batchRepoResult struct {
	Repo   string `json:"repo" yaml:"repo"`
	Action string `json:"action" yaml:"action"`
	Status string `json:"status" yaml:"status"`
	Error  string `json:"error,omitempty" yaml:"error,omitempty"`
}

type batchRepoSummary struct {
	Total     int               `json:"total" yaml:"total"`
	Succeeded int               `json:"succeeded" yaml:"succeeded"`
	Failed    int               `json:"failed" yaml:"failed"`
	Duration  string            `json:"duration" yaml:"duration"`
	Results   []batchRepoResult `json:"results" yaml:"results"`
}

func newBatchCreateShortcut() *common.Shortcut {
	return &common.Shortcut{
		Name:        "batch-create",
		Description: "批量创建仓库，支持逗号分隔列表或 CSV 文件",
		Flags: []common.Flag{
			{Name: "repos", Short: "r", Usage: "逗号分隔的仓库名称，例如: repo1,repo2,repo3"},
			{Name: "from", Usage: "从 CSV 文件读取仓库名称。支持 name/repo/repository 列名或无表头首列"},
			{Name: "description", Short: "d", Usage: "仓库描述（所有仓库共用一个描述）"},
			{Name: "private", Usage: "设为私有仓库 (true/false)", Default: "false"},
			{Name: "dry-run", Usage: "仅预览将要创建的仓库，不实际执行", Bool: true, Default: "false"},
		},
		Run: runBatchCreate,
	}
}

func runBatchCreate(ctx *common.RuntimeContext) error {
	start := time.Now()

	repos, err := collectRepos(ctx.Arg("repos"), ctx.Arg("from"))
	if err != nil {
		return err
	}
	if len(repos) == 0 {
		return fmt.Errorf("未提供仓库名称，请使用 --repos repo1,repo2 或 --from repos.csv")
	}

	// 仅取仓库名，不需要 owner/repo 格式
	names := make([]string, len(repos))
	for i, r := range repos {
		parts := strings.SplitN(r, "/", 2)
		names[i] = parts[len(parts)-1]
	}

	dryRun := parseBool(ctx.Arg("dry-run"))
	summary := batchRepoSummary{
		Total:   len(names),
		Results: make([]batchRepoResult, 0, len(names)),
	}

	var userLogin string
	var userID int
	if !dryRun {
		userEnv, err := ctx.CallAPI("GET", "/users/me", nil)
		if err != nil {
			return fmt.Errorf("获取当前用户信息失败: %w", err)
		}
		userData, _ := userEnv.Data.(map[string]interface{})
		login, _ := userData["login"].(string)
		if login == "" {
			return fmt.Errorf("无法获取当前用户名")
		}
		userLogin = login
		uid, _ := userData["user_id"].(float64)
		userID = int(uid)
	}

	private := ctx.Arg("private") == "true"
	desc := ctx.Arg("description")

	for _, name := range names {
		result := batchRepoResult{Repo: name, Action: "create"}
		if dryRun {
			result.Status = "planned"
			summary.Succeeded++
			summary.Results = append(summary.Results, result)
			continue
		}

		body := map[string]interface{}{
			"name":            name,
			"repository_name": name,
			"user_id":         userID,
		}
		if desc != "" {
			body["description"] = desc
		}
		if private {
			body["private"] = true
		}

		path := fmt.Sprintf("/%s/%s", userLogin, name)
		if _, err := ctx.CallAPI("POST", path, body); err != nil {
			result.Status = "failed"
			result.Error = err.Error()
			summary.Failed++
		} else {
			result.Status = "created"
			summary.Succeeded++
		}
		summary.Results = append(summary.Results, result)
	}

	summary.Duration = time.Since(start).String()

	if err := ctx.OutputData(summary); err != nil {
		return err
	}
	if summary.Failed > 0 {
		return fmt.Errorf("%d / %d 个仓库创建失败", summary.Failed, summary.Total)
	}
	return nil
}

func newBatchForkShortcut() *common.Shortcut {
	return &common.Shortcut{
		Name:        "batch-fork",
		Description: "批量 Fork 仓库，支持逗号分隔列表或 CSV 文件",
		Flags: []common.Flag{
			{Name: "repos", Short: "r", Usage: "逗号分隔的仓库标识，格式为 owner/repo，例如: alice/proj1,bob/proj2"},
			{Name: "from", Usage: "从 CSV 文件读取仓库标识。支持 owner/repo 单列或 owner、repo 双列格式"},
			{Name: "dry-run", Usage: "仅预览将要 Fork 的仓库，不实际执行", Bool: true, Default: "false"},
		},
		Run: runBatchFork,
	}
}

func runBatchFork(ctx *common.RuntimeContext) error {
	start := time.Now()

	repos, err := collectRepos(ctx.Arg("repos"), ctx.Arg("from"))
	if err != nil {
		return err
	}
	if len(repos) == 0 {
		return fmt.Errorf("未提供仓库标识，请使用 --repos owner/repo1,owner/repo2 或 --from repos.csv")
	}

	dryRun := parseBool(ctx.Arg("dry-run"))
	summary := batchRepoSummary{
		Total:   len(repos),
		Results: make([]batchRepoResult, 0, len(repos)),
	}

	for _, repoID := range repos {
		parts := strings.SplitN(repoID, "/", 2)
		result := batchRepoResult{Repo: repoID, Action: "fork"}
		if dryRun {
			result.Status = "planned"
			summary.Succeeded++
			summary.Results = append(summary.Results, result)
			continue
		}

		path := fmt.Sprintf("/%s/%s/forks", parts[0], parts[1])
		if _, err := ctx.CallAPI("POST", path, nil); err != nil {
			result.Status = "failed"
			result.Error = err.Error()
			summary.Failed++
		} else {
			result.Status = "forked"
			summary.Succeeded++
		}
		summary.Results = append(summary.Results, result)
	}

	summary.Duration = time.Since(start).String()

	if err := ctx.OutputData(summary); err != nil {
		return err
	}
	if summary.Failed > 0 {
		return fmt.Errorf("%d / %d 个仓库 Fork 失败", summary.Failed, summary.Total)
	}
	return nil
}

func newBatchDeleteShortcut() *common.Shortcut {
	return &common.Shortcut{
		Name:        "batch-delete",
		Description: "批量删除仓库，支持逗号分隔列表或 CSV 文件",
		Flags: []common.Flag{
			{Name: "repos", Short: "r", Usage: "逗号分隔的仓库标识，格式为 owner/repo，例如: alice/proj1,bob/proj2"},
			{Name: "from", Usage: "从 CSV 文件读取仓库标识。支持 owner/repo 单列或 owner、repo 双列格式"},
			{Name: "dry-run", Usage: "仅预览将要删除的仓库，不实际执行", Bool: true, Default: "false"},
		},
		Run: runBatchDelete,
	}
}

func runBatchDelete(ctx *common.RuntimeContext) error {
	start := time.Now()

	repos, err := collectRepos(ctx.Arg("repos"), ctx.Arg("from"))
	if err != nil {
		return err
	}
	if len(repos) == 0 {
		return fmt.Errorf("未提供仓库标识，请使用 --repos owner/repo1,owner/repo2 或 --from repos.csv")
	}

	dryRun := parseBool(ctx.Arg("dry-run"))
	summary := batchRepoSummary{
		Total:   len(repos),
		Results: make([]batchRepoResult, 0, len(repos)),
	}

	for _, repoID := range repos {
		parts := strings.SplitN(repoID, "/", 2)
		result := batchRepoResult{Repo: repoID, Action: "delete"}
		if dryRun {
			result.Status = "planned"
			summary.Succeeded++
			summary.Results = append(summary.Results, result)
			continue
		}

		path := fmt.Sprintf("/%s/%s", parts[0], parts[1])
		if _, err := ctx.CallAPI("DELETE", path, nil); err != nil {
			result.Status = "failed"
			result.Error = err.Error()
			summary.Failed++
		} else {
			result.Status = "deleted"
			summary.Succeeded++
		}
		summary.Results = append(summary.Results, result)
	}

	summary.Duration = time.Since(start).String()

	if err := ctx.OutputData(summary); err != nil {
		return err
	}
	if summary.Failed > 0 {
		return fmt.Errorf("%d / %d 个仓库删除失败", summary.Failed, summary.Total)
	}
	return nil
}

// --- CSV / list helpers ---

func collectRepos(reposValue, csvPath string) ([]string, error) {
	repos, err := parseRepoList(reposValue)
	if err != nil {
		return nil, err
	}
	if csvPath == "" {
		return repos, nil
	}

	csvRepos, err := readReposFromCSV(csvPath)
	if err != nil {
		return nil, err
	}
	return mergeRepoStrings(repos, csvRepos), nil
}

func parseRepoList(value string) ([]string, error) {
	if strings.TrimSpace(value) == "" {
		return nil, nil
	}
	return normalizeRepoIDs(strings.Split(value, ","))
}

func readReposFromCSV(path string) ([]string, error) {
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

	singleCol, ownerCol, repoCol := -1, -1, -1
	startRow := 0
	for i, cell := range records[0] {
		switch strings.ToLower(strings.TrimSpace(cell)) {
		case "owner/repo", "full_name":
			singleCol = i
			startRow = 1
		case "owner":
			ownerCol = i
			startRow = 1
		case "repo", "repository", "name":
			if repoCol == -1 {
				repoCol = i
			}
			startRow = 1
		}
	}

	if ownerCol == -1 || repoCol == -1 {
		// Not dual-column: use single-column mode
		if singleCol == -1 {
			singleCol = 0
		}
		ownerCol = -1
		repoCol = -1
	}

	values := make([]string, 0, len(records)-startRow)
	for _, record := range records[startRow:] {
		var repoID string
		if singleCol >= 0 && singleCol < len(record) {
			repoID = record[singleCol]
		} else if ownerCol >= 0 && repoCol >= 0 && ownerCol < len(record) && repoCol < len(record) {
			repoID = record[ownerCol] + "/" + record[repoCol]
		} else {
			continue
		}
		values = append(values, repoID)
	}
	return normalizeRepoIDs(values)
}

func normalizeRepoIDs(values []string) ([]string, error) {
	repos := make([]string, 0, len(values))
	seen := map[string]bool{}
	for _, value := range values {
		repoID := strings.TrimSpace(value)
		if repoID == "" {
			continue
		}
		if seen[repoID] {
			continue
		}
		seen[repoID] = true
		repos = append(repos, repoID)
	}
	return repos, nil
}

func mergeRepoStrings(values ...[]string) []string {
	merged := []string{}
	seen := map[string]bool{}
	for _, repos := range values {
		for _, r := range repos {
			if seen[r] {
				continue
			}
			seen[r] = true
			merged = append(merged, r)
		}
	}
	return merged
}

func parseBool(value string) bool {
	if strings.EqualFold(strings.TrimSpace(value), "true") {
		return true
	}
	return false
}
