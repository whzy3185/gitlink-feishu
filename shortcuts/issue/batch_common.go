package issue

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gitlink-org/gitlink-cli/internal/i18n"
	"github.com/gitlink-org/gitlink-cli/shortcuts/common"
)

// batch 命令共享开关的默认值。
const (
	defaultBatchMaxItems = 100
	defaultBatchDelayMs  = 0
)

// BatchOptions 承载所有 batch_* 命令共享的运行时配置。
type BatchOptions struct {
	DryRun   bool
	Confirm  bool
	MaxItems int
	DelayMs  int
}

// parseBatchOptions 从 RuntimeContext 解析 batch 命令的共享开关。
func parseBatchOptions(ctx *common.RuntimeContext) BatchOptions {
	return BatchOptions{
		DryRun:   parseBool(ctx.Arg("dry-run")),
		Confirm:  parseBool(ctx.Arg("confirm")),
		MaxItems: parseIntArg(ctx, "max", defaultBatchMaxItems),
		DelayMs:  parseIntArg(ctx, "delay", defaultBatchDelayMs),
	}
}

// batchStateFlags 返回 batch 状态变更命令（close、open）共用的 flag 列表。
func batchStateFlags(tr *i18n.Translator) []common.Flag {
	return []common.Flag{
		{Name: "numbers", Short: "n", Usage: tr.T("flag.issue.batch.numbers")},
		{Name: "from", Usage: tr.T("flag.issue.batch.from")},
		{Name: "search", Usage: tr.T("flag.issue.batch.search")},
		{Name: "state", Usage: tr.T("flag.issue.batch.state")},
		{Name: "label", Usage: tr.T("flag.issue.batch.label")},
		{Name: "confirm", Usage: tr.T("flag.issue.batch.confirm"), Bool: true, Default: "false"},
		{Name: "max", Usage: tr.T("flag.issue.batch.max"), Default: strconv.Itoa(defaultBatchMaxItems)},
		{Name: "delay", Usage: tr.T("flag.issue.batch.delay"), Default: strconv.Itoa(defaultBatchDelayMs)},
		{Name: "dry-run", Usage: tr.T("flag.issue.batch.dry_run"), Bool: true, Default: "false"},
	}
}

// batchRuntimeFlags 返回各 batch 命令共用的运行时 flag（dry-run/confirm/max/delay）。
func batchRuntimeFlags(tr *i18n.Translator) []common.Flag {
	return []common.Flag{
		{Name: "dry-run", Usage: tr.T("flag.issue.batch.dry_run"), Bool: true, Default: "false"},
		{Name: "confirm", Usage: tr.T("flag.issue.batch.confirm"), Bool: true, Default: "false"},
		{Name: "max", Usage: tr.T("flag.issue.batch.max"), Default: strconv.Itoa(defaultBatchMaxItems)},
		{Name: "delay", Usage: tr.T("flag.issue.batch.delay"), Default: strconv.Itoa(defaultBatchDelayMs)},
	}
}

// runBatchStateChange 是 batch 状态变更命令（close、open）共享的 Run 实现。
// action 形参同时用作 RunBatch 的操作名与 patchIssue 的错误前缀。
func runBatchStateChange(ctx *common.RuntimeContext, action string, statusID int) error {
	if err := ctx.ResolveOwnerRepo(); err != nil {
		return err
	}
	numbers, err := ResolveIssueNumbers(ctx, ctx.Arg("numbers"), ctx.Arg("from"), ctx.Arg("search"))
	if err != nil {
		return err
	}
	fn := func(c *common.RuntimeContext, number string) error {
		return patchIssue(c, number, map[string]interface{}{"status_id": statusID}, action)
	}
	_, err = RunBatch(ctx, numbers, action, parseBatchOptions(ctx), fn)
	return err
}

// BatchResult 记录单条 issue 上一次 batch 操作的结果。
// ID 在 close/open/assign/label/update 中是 issue 编号，在 create 中是 "row-N"。
type BatchResult struct {
	ID     string `json:"id" yaml:"id"`
	Action string `json:"action" yaml:"action"`
	Status string `json:"status" yaml:"status"`
	Error  string `json:"error,omitempty" yaml:"error,omitempty"`
}

// BatchSummary 汇总一次 batch 操作的总体结果。
type BatchSummary struct {
	Repository string        `json:"repository" yaml:"repository"`
	DryRun     bool          `json:"dry_run" yaml:"dry_run"`
	Total      int           `json:"total" yaml:"total"`
	Succeeded  int           `json:"succeeded" yaml:"succeeded"`
	Failed     int           `json:"failed" yaml:"failed"`
	Truncated  bool          `json:"truncated,omitempty" yaml:"truncated,omitempty"`
	Results    []BatchResult `json:"results" yaml:"results"`
}

// parseBool 把字符串解析为 bool。空串或解析失败时返回 false。
func parseBool(value string) bool {
	parsed, err := strconv.ParseBool(strings.TrimSpace(value))
	return err == nil && parsed
}

// parseIntArg 把指定 flag 解析为 int，空值或解析失败时回退到 defaultVal。
func parseIntArg(ctx *common.RuntimeContext, name string, defaultVal int) int {
	val := ctx.Arg(name)
	if val == "" {
		return defaultVal
	}
	v, err := strconv.Atoi(val)
	if err != nil {
		return defaultVal
	}
	return v
}

// ReadCSV 读取 CSV 文件并返回表头与数据行（不含表头）。
// 自动剥离首行首列单元格的 UTF-8 BOM。
func ReadCSV(path string) ([]string, [][]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, nil, fmt.Errorf("read CSV: %w", err)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	reader.TrimLeadingSpace = true
	records, err := reader.ReadAll()
	if err != nil {
		return nil, nil, fmt.Errorf("parse CSV: %w", err)
	}
	if len(records) == 0 {
		return nil, nil, fmt.Errorf("CSV file is empty or has no data rows")
	}

	// 去除表头首列单元格的 UTF-8 BOM
	records[0][0] = strings.TrimLeft(records[0][0], "\uFEFF")

	return records[0], records[1:], nil
}

// FindColumn 在 headers 中查找与任一 alias 忽略大小写、忽略首尾空格后匹配的列下标。
// 找不到时返回 -1。
func FindColumn(headers []string, aliases ...string) int {
	for i, header := range headers {
		normalized := strings.ToLower(strings.TrimSpace(header))
		for _, alias := range aliases {
			if normalized == strings.ToLower(strings.TrimSpace(alias)) {
				return i
			}
		}
	}
	return -1
}

// parseIssueNumbers 把逗号分隔的 issue 编号字符串拆分为列表并做归一化。
func parseIssueNumbers(value string) ([]string, error) {
	if strings.TrimSpace(value) == "" {
		return nil, nil
	}
	return normalizeIssueNumbers(strings.Split(value, ","))
}

// normalizeIssueNumbers 对编号列表做去空白、去重，并校验每项必须是合法整数。
func normalizeIssueNumbers(values []string) ([]string, error) {
	numbers := make([]string, 0, len(values))
	seen := map[string]bool{}
	for _, value := range values {
		number := strings.TrimSpace(value)
		if number == "" {
			continue
		}
		if _, err := strconv.ParseInt(number, 10, 64); err != nil {
			return nil, fmt.Errorf("invalid issue number %q: issue numbers must be integers", number)
		}
		if seen[number] {
			continue
		}
		seen[number] = true
		numbers = append(numbers, number)
	}
	return numbers, nil
}

// mergeIssueNumbers 把多组 issue 编号合并为单一列表，并做跨组去重，保持首次出现顺序。
func mergeIssueNumbers(values ...[]string) []string {
	merged := []string{}
	seen := map[string]bool{}
	for _, numbers := range values {
		for _, number := range numbers {
			if seen[number] {
				continue
			}
			seen[number] = true
			merged = append(merged, number)
		}
	}
	return merged
}

// readIssueNumbersFromCSV 读取 CSV 文件，定位 issue 编号所在列（number / issue_number，
// 大小写不敏感），返回该列中所有非空编号。文件不存在或解析失败时返回错误；
// 空文件（无数据行）返回 (nil, nil)，表示无来源而非错误。
func readIssueNumbersFromCSV(path string) ([]string, error) {
	headers, rows, err := ReadCSV(path)
	if err != nil {
		// 区分「文件不存在/读失败」（返回错误）与「无数据行」。
		// ReadCSV 对空文件返回 "CSV file is empty or has no data rows"，视作无来源。
		if strings.Contains(err.Error(), "empty or has no data rows") {
			return nil, nil
		}
		return nil, err
	}

	col := FindColumn(headers, "number", "issue_number")
	if col < 0 {
		return nil, fmt.Errorf("CSV missing issue number column (expected \"number\" or \"issue_number\")")
	}

	var numbers []string
	for _, row := range rows {
		if col >= len(row) {
			continue // 短行跳过
		}
		value := strings.TrimSpace(row[col])
		if value == "" {
			continue
		}
		numbers = append(numbers, value)
	}
	if len(numbers) == 0 {
		return nil, nil
	}
	return normalizeIssueNumbers(numbers)
}

// collectIssueNumbers 从命令行编号（逗号分隔）与 CSV 文件两个来源汇总 issue 编号，
// 合并去重后返回。两者均可省略；任一来源出错（非法编号、CSV 读失败）立即返回错误。
func collectIssueNumbers(numbersValue, csvPath string) ([]string, error) {
	var sources [][]string

	if nums, err := parseIssueNumbers(numbersValue); err != nil {
		return nil, err
	} else {
		sources = append(sources, nums)
	}

	if csvPath != "" {
		csvNumbers, err := readIssueNumbersFromCSV(csvPath)
		if err != nil {
			return nil, err
		}
		sources = append(sources, csvNumbers)
	}

	merged := mergeIssueNumbers(sources...)
	if len(merged) == 0 {
		return nil, fmt.Errorf("no issue numbers provided: pass --numbers or --from")
	}
	return merged, nil
}

// ResolveIssueNumbers 从三个来源（--numbers、--from CSV、--search）汇总 issue 编号，
// 合并去重后返回。三者均可省略，但至少需有一个非空来源。
func ResolveIssueNumbers(ctx *common.RuntimeContext, numbersValue, csvPath, searchKeyword string) ([]string, error) {
	var allNumbers [][]string

	// 1. 来自 --numbers（逗号分隔字符串）
	nums, err := parseIssueNumbers(numbersValue)
	if err != nil {
		return nil, err
	}
	allNumbers = append(allNumbers, nums)

	// 2. 来自 --from 指定的 CSV
	if csvPath != "" {
		headers, rows, err := ReadCSV(csvPath)
		if err != nil {
			return nil, err
		}
		col := FindColumn(headers, "number", "issue_number", "project_issues_index")
		if col == -1 {
			return nil, fmt.Errorf("no matching column (number/issue_number/project_issues_index) in CSV: %s", csvPath)
		}
		csvNums := make([]string, 0, len(rows))
		for _, row := range rows {
			if col < len(row) {
				csvNums = append(csvNums, row[col])
			}
		}
		csvNums, err = normalizeIssueNumbers(csvNums)
		if err != nil {
			return nil, err
		}
		allNumbers = append(allNumbers, csvNums)
	}

	// 3. 来自 --search 关键词
	if searchKeyword != "" {
		searchNums, err := searchIssues(ctx, searchKeyword)
		if err != nil {
			return nil, err
		}
		allNumbers = append(allNumbers, searchNums)
	}

	result := mergeIssueNumbers(allNumbers...)
	if len(result) == 0 {
		return nil, fmt.Errorf("no issue numbers found")
	}
	return result, nil
}

// searchIssues 调用 v1 issues 搜索接口并提取匹配项的 issue 编号。
// API 单次最多返回 100 条；若响应中 total_count 表明匹配更多，会向 stderr 输出警告。
func searchIssues(ctx *common.RuntimeContext, keyword string) ([]string, error) {
	q := url.Values{}
	q.Set("search", keyword)
	q.Set("limit", "100")
	if state := ctx.Arg("state"); state != "" {
		q.Set("state", state)
	}
	if label := ctx.Arg("label"); label != "" {
		q.Set("label", label)
	}

	env, err := ctx.CallAPIWithQuery("GET", v1RepoPath(ctx)+"/issues", q)
	if err != nil {
		return nil, fmt.Errorf("search issues: %w", err)
	}

	rawMap, ok := env.Data.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("search issues: unexpected response format")
	}

	dataField, ok := rawMap["data"]
	if !ok {
		return nil, fmt.Errorf("search issues: no data in response")
	}

	issues, err := parseDataArray(dataField)
	if err != nil {
		return nil, fmt.Errorf("search issues: parse data: %w", err)
	}

	// 当响应声明的总数大于本页返回时给出警告
	if total, ok := rawMap["total_count"].(float64); ok && int(total) > len(issues) {
		fmt.Fprintf(os.Stderr, "警告：搜索 %q 匹配 %d 个 issue，但 API 一次最多返回 100 个，结果可能不完整。请用 --state/--label 缩小范围\n", keyword, int(total))
	}

	numbers := make([]string, 0, len(issues))
	for _, item := range issues {
		issue, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		// PATCH 接口要求项目内编号（project-local number），不接受全局 DB id。
		// 仅在 "number" 缺失时回退到 "iid"（Redmine 命名），不向 "id" 回退，
		// 否则会把全局 id 透传给 PATCH，接口必然 404。
		var id string
		if v, ok := issue["number"]; ok {
			id = fmt.Sprintf("%v", v)
		} else if v, ok := issue["iid"]; ok {
			id = fmt.Sprintf("%v", v)
		}
		if id != "" && id != "0" {
			numbers = append(numbers, id)
		}
	}

	return normalizeIssueNumbers(numbers)
}

// 名称→ID 解析器的进程级缓存。
// labelCache / milestoneCache 以 "{owner}/{repo}" 为键做仓库级隔离，
// 避免在同一进程内切换仓库时产生脏数据。所有 map 的读写都在 resolverCacheMu 保护下。
var (
	resolverCacheMu sync.Mutex
	userCache       map[string]int
	labelCache      map[string]map[string]int // 仓库路径 → label 名称 → label ID
	milestoneCache  map[string]map[string]int // 仓库路径 → milestone 名称 → milestone ID
)

// parseDataArray 把 API 响应中的 Data 字段统一解析为 []interface{}，
// 兼容 client.Do 返回的 []interface{}、json.RawMessage、JSON 字符串三种形态。
func parseDataArray(data interface{}) ([]interface{}, error) {
	switch d := data.(type) {
	case []interface{}:
		return d, nil
	case json.RawMessage:
		var items []interface{}
		if err := json.Unmarshal([]byte(d), &items); err != nil {
			return nil, err
		}
		return items, nil
	case string:
		var items []interface{}
		if err := json.Unmarshal([]byte(d), &items); err != nil {
			return nil, err
		}
		return items, nil
	default:
		return nil, fmt.Errorf("unexpected data type %T", data)
	}
}

// ResolveUserID 把用户登录名解析为数字 user ID。
// 若 name 本身是数字则直接返回；否则调用 GET /users/search?q={name}，
// 把返回的全部用户按 login→id 缓存，并返回匹配的 ID。
func ResolveUserID(ctx *common.RuntimeContext, name string) (int, error) {
	name = strings.TrimSpace(name)
	if id, err := strconv.Atoi(name); err == nil {
		return id, nil
	}

	// 命中缓存直接返回
	resolverCacheMu.Lock()
	if id, ok := userCache[name]; ok {
		resolverCacheMu.Unlock()
		return id, nil
	}
	resolverCacheMu.Unlock()

	// 缓存未命中，调用用户搜索 API
	q := url.Values{}
	q.Set("q", name)
	env, err := ctx.CallAPIWithQuery("GET", "/users/search", q)
	if err != nil {
		return 0, fmt.Errorf("resolve user: %w", err)
	}

	users, err := parseDataArray(env.Data)
	if err != nil {
		return 0, fmt.Errorf("resolve user: parse data: %w", err)
	}

	// 把搜索返回的全部用户写进缓存，便于后续按 login 命中
	resolverCacheMu.Lock()
	if userCache == nil {
		userCache = make(map[string]int, len(users))
	}
	for _, item := range users {
		u, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		login, _ := u["login"].(string)
		id := getMapInt(u, "id")
		if login != "" && id > 0 {
			userCache[login] = id
		}
	}
	id, ok := userCache[name]
	resolverCacheMu.Unlock()

	if !ok {
		return 0, fmt.Errorf("user %q not found", name)
	}
	return id, nil
}

// ResolveLabelID 把 label 名称解析为数字 label ID。
// 若 name 本身是数字则直接返回；否则首次按当前仓库拉取全部 label
// （GET /{owner}/{repo}/labels，v0 前缀）并按仓库维度缓存，后续直接走缓存。
func ResolveLabelID(ctx *common.RuntimeContext, name string) (int, error) {
	if id, err := strconv.Atoi(name); err == nil {
		return id, nil
	}

	repoKey := fmt.Sprintf("%s/%s", ctx.Owner, ctx.Repo)

	// 命中当前仓库的 label 缓存
	resolverCacheMu.Lock()
	if repoCache, ok := labelCache[repoKey]; ok {
		id, found := repoCache[name]
		resolverCacheMu.Unlock()
		if !found {
			return 0, fmt.Errorf("label %q not found", name)
		}
		return id, nil
	}
	resolverCacheMu.Unlock()

	// 缓存未命中，从 API 拉取该仓库的全部 label
	env, err := ctx.CallAPI("GET", fmt.Sprintf("/%s/%s/labels", ctx.Owner, ctx.Repo), nil)
	if err != nil {
		return 0, fmt.Errorf("resolve label: %w", err)
	}

	// API 返回 {"status":0, "issue_tags":[...], ...}，client.Do 把整个响应包在 envelope 里，
	// 所以 env.Data 是包含 issue_tags 键的 map，需要先提取 issue_tags 再解析数组。
	rawMap, ok := env.Data.(map[string]interface{})
	if !ok {
		return 0, fmt.Errorf("resolve label: unexpected response type %T", env.Data)
	}
	itemsRaw, ok := rawMap["issue_tags"]
	if !ok {
		return 0, fmt.Errorf("resolve label: response missing issue_tags field")
	}
	items, err := parseDataArray(itemsRaw)
	if err != nil {
		return 0, fmt.Errorf("resolve label: parse issue_tags: %w", err)
	}

	// 在锁内把当前仓库的 label 全量写入按 repoKey 隔离的缓存
	resolverCacheMu.Lock()
	if labelCache == nil {
		labelCache = make(map[string]map[string]int)
	}
	repoCache := make(map[string]int, len(items))
	for _, item := range items {
		l, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		labelName, _ := l["name"].(string)
		id := getMapInt(l, "id")
		if labelName != "" && id > 0 {
			repoCache[labelName] = id
		}
	}
	labelCache[repoKey] = repoCache
	id, ok := repoCache[name]
	resolverCacheMu.Unlock()

	if !ok {
		return 0, fmt.Errorf("label %q not found", name)
	}
	return id, nil
}

// ResolveMilestoneID 把 milestone 名称解析为数字 milestone ID。
// 若 name 本身是数字则直接返回；否则首次按当前仓库拉取全部 milestone
// （GET /v1/{owner}/{repo}/milestones）并按仓库维度缓存，后续直接走缓存。
func ResolveMilestoneID(ctx *common.RuntimeContext, name string) (int, error) {
	if id, err := strconv.Atoi(name); err == nil {
		return id, nil
	}

	repoKey := fmt.Sprintf("%s/%s", ctx.Owner, ctx.Repo)

	// 命中当前仓库的 milestone 缓存
	resolverCacheMu.Lock()
	if repoCache, ok := milestoneCache[repoKey]; ok {
		id, found := repoCache[name]
		resolverCacheMu.Unlock()
		if !found {
			return 0, fmt.Errorf("milestone %q not found", name)
		}
		return id, nil
	}
	resolverCacheMu.Unlock()

	// 缓存未命中，从 API 拉取该仓库的全部 milestone
	env, err := ctx.CallAPI("GET", v1RepoPath(ctx)+"/milestones", nil)
	if err != nil {
		return 0, fmt.Errorf("resolve milestone: %w", err)
	}

	// API 返回 {"closed_milestone_count":0, "opening_milestone_count":0, "total_count":0, "milestones":[...]}，
	// client.Do 把整个响应包在 envelope 里，所以 env.Data 是包含 milestones 键的 map，
	// 需要先提取 milestones 再解析数组。
	rawMap, ok := env.Data.(map[string]interface{})
	if !ok {
		return 0, fmt.Errorf("resolve milestone: unexpected response type %T", env.Data)
	}
	itemsRaw, ok := rawMap["milestones"]
	if !ok {
		return 0, fmt.Errorf("resolve milestone: response missing milestones field")
	}
	items, err := parseDataArray(itemsRaw)
	if err != nil {
		return 0, fmt.Errorf("resolve milestone: parse milestones: %w", err)
	}

	// 在锁内把当前仓库的 milestone 全量写入按 repoKey 隔离的缓存
	resolverCacheMu.Lock()
	if milestoneCache == nil {
		milestoneCache = make(map[string]map[string]int)
	}
	repoCache := make(map[string]int, len(items))
	for _, item := range items {
		m, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		milestoneName, _ := m["name"].(string)
		id := getMapInt(m, "id")
		if milestoneName != "" && id > 0 {
			repoCache[milestoneName] = id
		}
	}
	milestoneCache[repoKey] = repoCache
	id, ok := repoCache[name]
	resolverCacheMu.Unlock()

	if !ok {
		return 0, fmt.Errorf("milestone %q not found", name)
	}
	return id, nil
}

// RunBatch 在一组 issue 编号上执行批量操作，集成 dry-run、节流、确认门、--max 截断。
// fn 是逐条执行的操作回调，dry-run 模式下不会被调用。
// 返回值同时包含汇总和错误：完全成功时 error 为 nil；存在失败或截断时附带描述性错误。
func RunBatch(ctx *common.RuntimeContext, numbers []string, action string, opts BatchOptions, fn func(ctx *common.RuntimeContext, number string) error) (*BatchSummary, error) {
	// 确认门：dry-run 直接放行；非 dry-run 必须显式 --confirm 或环境变量
	if !opts.DryRun && !opts.Confirm && os.Getenv("GITLINK_CONFIRM_BATCH") != "true" {
		return nil, fmt.Errorf("请添加 --confirm 确认执行，或使用 --dry-run 预览。也可设置 GITLINK_CONFIRM_BATCH=true 环境变量跳过此检查")
	}

	// --max 截断：超过上限时取前 N 条并标记 truncated
	truncated := false
	if opts.MaxItems > 0 && len(numbers) > opts.MaxItems {
		fmt.Fprintf(os.Stderr, "警告：已按 --max=%d 截断，从 %d 个减少到 %d 个\n", opts.MaxItems, len(numbers), opts.MaxItems)
		numbers = numbers[:opts.MaxItems]
		truncated = true
	}

	summary := &BatchSummary{
		Repository: fmt.Sprintf("%s/%s", ctx.Owner, ctx.Repo),
		DryRun:     opts.DryRun,
		Total:      len(numbers),
		Results:    make([]BatchResult, 0, len(numbers)),
		Truncated:  truncated,
	}

	for i, number := range numbers {
		// 仅在两次请求之间节流，跳过第一条
		if opts.DelayMs > 0 && i > 0 {
			time.Sleep(time.Duration(opts.DelayMs) * time.Millisecond)
		}

		result := BatchResult{ID: number, Action: action}

		if opts.DryRun {
			result.Status = "dry_run"
			summary.Succeeded++
		} else {
			if err := fn(ctx, number); err != nil {
				result.Status = "failed"
				result.Error = err.Error()
				summary.Failed++
			} else {
				result.Status = "success"
				summary.Succeeded++
			}
		}
		summary.Results = append(summary.Results, result)
	}

	// 先输出汇总，再根据失败/截断状态决定是否返回错误
	if err := ctx.OutputData(summary); err != nil {
		return summary, err
	}

	// 失败与截断同时出现时，错误信息合并提示
	if summary.Failed > 0 && summary.Truncated {
		return summary, fmt.Errorf("%d of %d issue(s) failed to %s (results truncated to %d)", summary.Failed, summary.Total, action, opts.MaxItems)
	}
	if summary.Failed > 0 {
		return summary, fmt.Errorf("%d of %d issue(s) failed to %s", summary.Failed, summary.Total, action)
	}
	if summary.Truncated {
		return summary, fmt.Errorf("results truncated to %d issues", opts.MaxItems)
	}

	return summary, nil
}

// patchIssue 先读取 issue 当前数据，再以 subject/description 为基础合并 extraFields 后发送 PATCH。
// action 用于包裹 PATCH 阶段错误（形如 "close issue: %w"），便于定位失败操作。
func patchIssue(ctx *common.RuntimeContext, number string, extraFields map[string]interface{}, action string) error {
	current, err := fetchIssueData(ctx, number)
	if err != nil {
		return fmt.Errorf("fetch issue: %w", err)
	}
	body := map[string]interface{}{
		"subject":     current.Subject,
		"description": current.Description,
	}
	for k, v := range extraFields {
		body[k] = v
	}
	_, err = ctx.CallAPI("PATCH", fmt.Sprintf("%s/issues/%s", v1RepoPath(ctx), number), body)
	if err != nil {
		return fmt.Errorf("%s issue: %w", action, err)
	}
	return nil
}
