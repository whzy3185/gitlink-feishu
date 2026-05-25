package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type ProjectConfig struct {
	Project    ProjectInfo    `json:"project"`
	Repository RepositoryInfo `json:"repository"`
	Branches   []BranchPlan   `json:"branches"`
	Issues     []IssuePlan    `json:"issues"`
	Publish    PublishConfig  `json:"publish"`
}

type ProjectInfo struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Language    string `json:"language"`
	License     string `json:"license"`
}

type RepositoryInfo struct {
	Owner string `json:"owner"`
	Name  string `json:"name"`
}

type BranchPlan struct {
	Name    string `json:"name"`
	From    string `json:"from"`
	Create  *bool  `json:"create"`
	Protect bool   `json:"protect"`
}

type IssuePlan struct {
	Title      string   `json:"title"`
	Type       string   `json:"type"`
	Priority   string   `json:"priority"`
	Tasks      []string `json:"tasks"`
	Acceptance string   `json:"acceptance"`
}

type PublishConfig struct {
	IssueNumber int `json:"issue_number"`
}

type FileManifestItem struct {
	Path  string `json:"path"`
	Bytes int    `json:"bytes"`
}

type OutputManifest struct {
	Repository  string             `json:"repository"`
	Project     ProjectInfo        `json:"project"`
	Files       []FileManifestItem `json:"files"`
	Branches    []BranchPlan       `json:"branches"`
	Issues      []IssuePlan        `json:"issues"`
	GeneratedAt string             `json:"generated_at"`
}

type CommandResult struct {
	Command    []string `json:"command"`
	Status     string   `json:"status"`
	ReturnCode *int     `json:"returncode"`
	Stdout     string   `json:"stdout"`
	Stderr     string   `json:"stderr"`
}

type CommandLog struct {
	Mode     string          `json:"mode"`
	Commands []CommandResult `json:"commands"`
}

type OutputPaths struct {
	Report   string
	Summary  string
	Manifest string
	Files    string
}

type Options struct {
	ConfigPath         string
	OutputDir          string
	CLIBin             string
	Apply              bool
	CreateRepo         bool
	PublishIssueNumber int
	Now                string
}

func parseFlags(args []string) Options {
	var opts Options
	fs := flag.NewFlagSet("bootstrap-project", flag.ExitOnError)
	fs.StringVar(&opts.ConfigPath, "config", filepath.FromSlash("examples/sample_project.json"), "配置文件路径")
	fs.StringVar(&opts.OutputDir, "output-dir", "outputs", "输出目录")
	fs.StringVar(&opts.CLIBin, "cli-bin", firstNonEmpty(os.Getenv("GITLINK_CLI_BIN"), "gitlink-cli"), "gitlink-cli 可执行文件路径")
	fs.BoolVar(&opts.Apply, "apply", false, "执行真实 GitLink 写操作")
	fs.BoolVar(&opts.CreateRepo, "create-repo", false, "仓库不存在时创建仓库")
	fs.IntVar(&opts.PublishIssueNumber, "publish-issue-number", 0, "把初始化摘要评论到指定 Issue")
	fs.StringVar(&opts.Now, "now", "", "固定当前时间，ISO8601 格式")
	_ = fs.Parse(args)
	return opts
}

func loadConfig(path string) (ProjectConfig, error) {
	var config ProjectConfig
	data, err := os.ReadFile(path)
	if err != nil {
		return config, fmt.Errorf("配置文件不存在: %s", path)
	}
	data = bytes.TrimPrefix(data, []byte{0xef, 0xbb, 0xbf})
	if err := json.Unmarshal(data, &config); err != nil {
		return config, err
	}
	return config, nil
}

func parseNow(value string) (time.Time, error) {
	if strings.TrimSpace(value) == "" {
		return time.Now().UTC(), nil
	}
	text := strings.ReplaceAll(strings.TrimSpace(value), "Z", "+00:00")
	dt, err := time.Parse(time.RFC3339, text)
	if err != nil {
		return time.Time{}, err
	}
	return dt.UTC(), nil
}

func isoTime(value time.Time) string {
	return value.UTC().Format(time.RFC3339)
}

func safeName(value string) string {
	replacer := strings.NewReplacer("/", "_", "\\", "_", " ", "_")
	return replacer.Replace(value)
}

func renderReadme(config ProjectConfig) string {
	language := firstNonEmpty(config.Project.Language, "未指定")
	return fmt.Sprintf("# %s\n\n%s\n\n## 项目信息\n\n- GitLink 仓库: `%s/%s`\n- 技术方向: %s\n- 初始化来源: GitLink 项目一键初始化工作流\n\n## 快速开始\n\n```bash\ngit clone https://gitlink.org.cn/%s/%s.git\ncd %s\n```\n\n## 协作约定\n\n- 使用 Issue 跟踪需求、缺陷和文档任务。\n- 使用 Pull Request 合并代码变更。\n- 重要里程碑通过 Release Notes 记录。\n",
		config.Project.Name,
		config.Project.Description,
		config.Repository.Owner,
		config.Repository.Name,
		language,
		config.Repository.Owner,
		config.Repository.Name,
		config.Repository.Name,
	)
}

func renderLicense(config ProjectConfig) string {
	licenseName := firstNonEmpty(config.Project.License, "MulanPSL-2.0")
	return fmt.Sprintf("# License\n\nThis project is initialized with the `%s` license.\n\nThe final repository should keep the complete license text that matches the selected open-source license.\n", licenseName)
}

func renderCI(config ProjectConfig) string {
	if strings.Contains(strings.ToLower(config.Project.Language), "go") {
		return `name: Go CI

on:
  push:
  pull_request:

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: "1.23"
      - run: go test ./...
`
	}
	return `name: Basic CI

on:
  push:
  pull_request:

jobs:
  check:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - run: echo "Add project-specific checks here."
`
}

func plannedFiles(config ProjectConfig) map[string]string {
	return map[string]string{
		"README.md":                renderReadme(config),
		"LICENSE":                  renderLicense(config),
		".github/workflows/ci.yml": renderCI(config),
		"docs/CONTRIBUTING.md":     "# 贡献指南\n\n请通过 Issue 讨论需求，通过 Pull Request 提交变更。\n",
		"docs/ROADMAP.md":          "# Roadmap\n\n- [ ] 完成项目初始化\n- [ ] 建立基础测试\n- [ ] 发布第一个版本\n",
	}
}

func issueBody(item IssuePlan, config ProjectConfig) string {
	repo := fmt.Sprintf("%s/%s", config.Repository.Owner, config.Repository.Name)
	tasks := "- [ ] 待补充"
	if len(item.Tasks) > 0 {
		lines := make([]string, 0, len(item.Tasks))
		for _, task := range item.Tasks {
			lines = append(lines, "- [ ] "+task)
		}
		tasks = strings.Join(lines, "\n")
	}
	return fmt.Sprintf("仓库: `%s`\n\n类型: %s\n优先级: %s\n\n## 任务清单\n\n%s\n\n## 验收标准\n\n%s\n",
		repo,
		firstNonEmpty(item.Type, "task"),
		firstNonEmpty(item.Priority, "normal"),
		tasks,
		firstNonEmpty(item.Acceptance, "完成后在本 Issue 中说明验证结果。"),
	)
}

func shouldCreateBranch(branch BranchPlan) bool {
	return branch.Create == nil || *branch.Create
}

func branchFrom(branch BranchPlan) string {
	return firstNonEmpty(branch.From, "master")
}

func buildCLIPlan(config ProjectConfig, summary string, createRepo bool) [][]string {
	owner := config.Repository.Owner
	repo := config.Repository.Name
	commands := [][]string{}
	if createRepo {
		commands = append(commands, []string{"repo", "+create", "--name", repo, "--description", config.Project.Description, "--format", "json"})
	}
	commands = append(commands,
		[]string{"repo", "+info", "--owner", owner, "--repo", repo, "--format", "json"},
		[]string{"branch", "+list", "--owner", owner, "--repo", repo, "--format", "json"},
	)
	for _, branch := range config.Branches {
		if shouldCreateBranch(branch) {
			commands = append(commands, []string{"branch", "+create", "--owner", owner, "--repo", repo, "--name", branch.Name, "--from", branchFrom(branch), "--format", "json"})
		}
		if branch.Protect {
			commands = append(commands, []string{"branch", "+protect", "--owner", owner, "--repo", repo, "--name", branch.Name, "--format", "json"})
		}
	}
	for _, issue := range config.Issues {
		commands = append(commands, []string{"issue", "+create", "--owner", owner, "--repo", repo, "--title", issue.Title, "--body", issueBody(issue, config), "--format", "json"})
	}
	if summary != "" && config.Publish.IssueNumber > 0 {
		commands = append(commands, []string{"issue", "+comment", "--owner", owner, "--repo", repo, "--number", strconv.Itoa(config.Publish.IssueNumber), "--body", summary, "--format", "json"})
	}
	return commands
}

func runCommand(cliBin string, args []string, apply bool) CommandResult {
	command := append([]string{cliBin}, args...)
	if !apply {
		return CommandResult{Command: command, Status: "planned"}
	}
	cmd := exec.Command(cliBin, args...)
	if strings.HasSuffix(strings.ToLower(cliBin), ".cmd") || strings.HasSuffix(strings.ToLower(cliBin), ".bat") {
		cmd = exec.Command("cmd", append([]string{"/c", cliBin}, args...)...)
	}
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	returnCode := 0
	status := "ok"
	if err != nil {
		status = "failed"
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			returnCode = exitErr.ExitCode()
		} else {
			returnCode = 1
		}
		if isIdempotentSkip(stderr.String()) {
			status = "skipped"
		}
	}
	return CommandResult{
		Command:    command,
		Status:     status,
		ReturnCode: &returnCode,
		Stdout:     strings.TrimSpace(stdout.String()),
		Stderr:     strings.TrimSpace(stderr.String()),
	}
}

func isIdempotentSkip(stderr string) bool {
	knownMessages := []string{
		"新分支已存在",
		"branch already exists",
		"repository already exists",
		"仓库已存在",
	}
	for _, message := range knownMessages {
		if strings.Contains(stderr, message) {
			return true
		}
	}
	return false
}

func writeOutputs(config ProjectConfig, outputDir string, now time.Time) (OutputPaths, error) {
	owner := config.Repository.Owner
	repo := config.Repository.Name
	prefix := fmt.Sprintf("%s_%s_%s", safeName(owner), safeName(repo), now.UTC().Format("20060102_150405"))
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return OutputPaths{}, err
	}
	files := plannedFiles(config)
	fileManifest := make([]FileManifestItem, 0, len(files))
	for _, path := range []string{"README.md", "LICENSE", ".github/workflows/ci.yml", "docs/CONTRIBUTING.md", "docs/ROADMAP.md"} {
		if content, ok := files[path]; ok {
			fileManifest = append(fileManifest, FileManifestItem{Path: path, Bytes: len([]byte(content))})
		}
	}
	summary := fmt.Sprintf("# GitLink 项目初始化摘要\n\n- 目标仓库: `%s/%s`\n- 项目名称: %s\n- 生成时间: %s\n- 初始化文件: %d 个\n- 初始 Issue: %d 个\n- 分支动作: %d 个\n",
		owner,
		repo,
		config.Project.Name,
		isoTime(now),
		len(files),
		len(config.Issues),
		len(config.Branches),
	)
	report := renderReport(config, fileManifest, now)
	manifest := OutputManifest{
		Repository:  fmt.Sprintf("%s/%s", owner, repo),
		Project:     config.Project,
		Files:       fileManifest,
		Branches:    config.Branches,
		Issues:      config.Issues,
		GeneratedAt: isoTime(now),
	}
	manifestJSON, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return OutputPaths{}, err
	}
	filesJSON, err := json.MarshalIndent(files, "", "  ")
	if err != nil {
		return OutputPaths{}, err
	}
	paths := OutputPaths{
		Report:   filepath.Join(outputDir, prefix+"_bootstrap_report.md"),
		Summary:  filepath.Join(outputDir, prefix+"_summary.md"),
		Manifest: filepath.Join(outputDir, prefix+"_manifest.json"),
		Files:    filepath.Join(outputDir, prefix+"_files.json"),
	}
	writes := map[string][]byte{
		paths.Report:   []byte(report),
		paths.Summary:  []byte(summary),
		paths.Manifest: manifestJSON,
		paths.Files:    filesJSON,
	}
	for path, data := range writes {
		if err := os.WriteFile(path, data, 0o644); err != nil {
			return OutputPaths{}, err
		}
	}
	return paths, nil
}

func renderReport(config ProjectConfig, fileManifest []FileManifestItem, now time.Time) string {
	fileRows := []string{}
	for _, item := range fileManifest {
		fileRows = append(fileRows, fmt.Sprintf("| `%s` | %d |", item.Path, item.Bytes))
	}
	branchRows := []string{}
	for _, item := range config.Branches {
		branchRows = append(branchRows, fmt.Sprintf("| `%s` | `%s` | %t |", item.Name, branchFrom(item), item.Protect))
	}
	if len(branchRows) == 0 {
		branchRows = append(branchRows, "| 无 | 无 | false |")
	}
	issueRows := []string{}
	for idx, item := range config.Issues {
		issueRows = append(issueRows, fmt.Sprintf("| %d | %s | %s |", idx+1, item.Title, firstNonEmpty(item.Priority, "normal")))
	}
	if len(issueRows) == 0 {
		issueRows = append(issueRows, "| 0 | 无 | normal |")
	}
	return fmt.Sprintf("# GitLink 项目初始化工作流报告\n\n## 目标项目\n\n- 仓库: `%s/%s`\n- 项目名称: %s\n- 描述: %s\n- 生成时间: %s\n\n## 初始化文件\n\n| 文件 | 字节数 |\n| --- | ---: |\n%s\n\n## 分支计划\n\n| 分支 | 来源 | 保护 |\n| --- | --- | --- |\n%s\n\n## 初始 Issue 计划\n\n| 序号 | 标题 | 优先级 |\n| ---: | --- | --- |\n%s\n\n## 工作流闭环\n\n1. 读取项目配置。\n2. 生成 README、LICENSE、CI 和协作文档。\n3. 调用 gitlink-cli 检查仓库和分支状态。\n4. 调用 gitlink-cli 创建初始化 Issue。\n5. 输出报告、摘要和结构化 manifest，必要时回写到 GitLink Issue。\n",
		config.Repository.Owner,
		config.Repository.Name,
		config.Project.Name,
		config.Project.Description,
		isoTime(now),
		strings.Join(fileRows, "\n"),
		strings.Join(branchRows, "\n"),
		strings.Join(issueRows, "\n"),
	)
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func main() {
	opts := parseFlags(os.Args[1:])
	config, err := loadConfig(opts.ConfigPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	now, err := parseNow(opts.Now)
	if err != nil {
		fmt.Fprintf(os.Stderr, "无法解析 --now 的值: %v\n", err)
		os.Exit(1)
	}
	if opts.PublishIssueNumber > 0 {
		config.Publish.IssueNumber = opts.PublishIssueNumber
	}
	outputPaths, err := writeOutputs(config, opts.OutputDir, now)
	if err != nil {
		fmt.Fprintf(os.Stderr, "写入输出失败: %v\n", err)
		os.Exit(1)
	}
	summaryBytes, err := os.ReadFile(outputPaths.Summary)
	if err != nil {
		fmt.Fprintf(os.Stderr, "读取摘要失败: %v\n", err)
		os.Exit(1)
	}
	plan := buildCLIPlan(config, string(summaryBytes), opts.CreateRepo)
	results := make([]CommandResult, 0, len(plan))
	for _, command := range plan {
		results = append(results, runCommand(opts.CLIBin, command, opts.Apply))
	}
	mode := "dry-run"
	if opts.Apply {
		mode = "apply"
	}
	commandLog := CommandLog{Mode: mode, Commands: results}
	commandLogJSON, err := json.MarshalIndent(commandLog, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "生成命令日志失败: %v\n", err)
		os.Exit(1)
	}
	commandLogPath := filepath.Join(opts.OutputDir, fmt.Sprintf("command_log_%s.json", now.UTC().Format("20060102_150405")))
	if err := os.WriteFile(commandLogPath, commandLogJSON, 0o644); err != nil {
		fmt.Fprintf(os.Stderr, "写入命令日志失败: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("已生成初始化报告: %s\n", outputPaths.Report)
	fmt.Printf("已生成初始化摘要: %s\n", outputPaths.Summary)
	fmt.Printf("已生成文件清单: %s\n", outputPaths.Manifest)
	fmt.Printf("已生成命令日志: %s\n", commandLogPath)
	fmt.Printf("模式: %s\n", mode)
	fmt.Printf("计划/执行 gitlink-cli 调用: %d 个\n", len(results))
	failed := 0
	for _, result := range results {
		if result.Status == "failed" {
			failed++
		}
	}
	if failed > 0 {
		fmt.Printf("失败命令: %d 个\n", failed)
		os.Exit(1)
	}
}
