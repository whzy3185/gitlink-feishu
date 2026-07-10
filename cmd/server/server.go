/*
 * 子赛题四「网页终端」HTTP 演示服务
 *
 * 前端搜索驱动界面 → 后端串行运行 Python 算法脚本
 * 支持 7 个科研分析维度（热点/画像/启发/谱系/合规/报告/可视化）
 * Python 子进程继承 GITLINK_TOKEN / DEEPSEEK_API_KEY 环境变量
 *
 * API:
 *   GET  /api/dimensions  - 返回可用维度列表
 *   POST /api/chain       - 统一全链路分析入口
 *   POST /api/run         - 单场景执行(向后兼容)
 *   GET  /api/result/{key} - 查询缓存产物
 */
package server

import (
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/spf13/cobra"
)

//go:embed static/*
var staticFS embed.FS

const defaultPort = 8080

// 场景定义：前端按钮 ↔ 后端 Python 脚本。
type scenarioDef struct {
	Key     string   `json:"key"`
	Label   string   `json:"label"`
	Desc    string   `json:"desc"`
	Script  string   `json:"script"`
	Needs   []string `json:"needs"` // 需要的输入: "owner","repo","keyword"
	Timeout int      `json:"timeout_s"`
}

var scenarios = []scenarioDef{
	{Key: "s1", Label: "S1 仓库洞悉", Desc: "科研项目演进谱系 + 创新点", Script: "lineage.py", Needs: []string{"owner", "repo"}, Timeout: 180},
	{Key: "s2", Label: "S2 知识图谱", Desc: "科研领域知识图谱(networkx)", Script: "graph_build.py", Needs: []string{"keyword"}, Timeout: 240},
	{Key: "s3", Label: "S3 合规复现", Desc: "许可证/密钥/复现性检查", Script: "repro.py", Needs: []string{"owner", "repo"}, Timeout: 120},
	{Key: "s4", Label: "S4 协作匹配", Desc: "学者×缺口 智能匹配", Script: "match.py", Needs: []string{"owner", "repo"}, Timeout: 180},
	{Key: "s5", Label: "S5 进度预警", Desc: "周报 + 风险预警", Script: "report.py", Needs: []string{"owner", "repo"}, Timeout: 180},
	{Key: "s6", Label: "S6 成果可视化", Desc: "交互图表(plotly)", Script: "visual.py", Needs: []string{"owner", "repo"}, Timeout: 240},
	{Key: "hotspot", Label: "🔥 热点追踪（关键词）", Desc: "关键词搜索：飙升项目+活跃讨论+主题热度+学者团队", Script: "hotspot.py", Needs: []string{"keyword"}, Timeout: 300},
	{Key: "hotspot-cat", Label: "🔥 热点追踪（分类精选）", Desc: "GitLink 官方分类精选 → 领域热点（缩范围）", Script: "hotspot.py", Needs: []string{"category"}, Timeout: 300},
	{Key: "profile", Label: "🪪 主体画像", Desc: "项目画像：主题/语言/贡献者/研究维度评分", Script: "profile.py", Needs: []string{"owner", "repo"}, Timeout: 150},
	{Key: "inspire", Label: "💡 创新启发", Desc: "缺口挖掘 + 合作者匹配 + LLM 研究方向建议", Script: "inspire.py", Needs: []string{"owner", "repo"}, Timeout: 240},
	{Key: "chain", Label: "🔬 全链路", Desc: "分类→热点→画像→启发→合规→分析（一条命令打通）", Script: "research.py", Needs: []string{"category"}, Timeout: 600},
}

// dimensionDef 分析维度：前端展示用，对应一个 Python 脚本。
type dimensionDef struct {
	Key     string   `json:"key"`
	Label   string   `json:"label"`
	Icon    string   `json:"icon"`
	Desc    string   `json:"desc"`
	Script  string   `json:"script"`
	Needs   []string `json:"needs"` // 需要的参数: "keyword","category","owner","repo","owner_repo"
	Timeout int      `json:"timeout_s"`
}

var dimensions = []dimensionDef{
	{Key: "hotspot", Label: "科研热点分析", Icon: "🔥", Desc: "飙升项目 + 活跃讨论 + 主题热度 + 核心学者", Script: "hotspot.py", Needs: []string{"keyword_or_category"}, Timeout: 300},
{Key: "profile", Label: "主体画像", Icon: "🪪", Desc: "项目/学者画像：主题向量/语言/研究维度评分", Script: "profile.py", Needs: []string{"owner_repo_or_category"}, Timeout: 150},
	{Key: "inspire", Label: "创新启发", Icon: "💡", Desc: "缺口挖掘 + 合作者匹配 + LLM 研究方向建议", Script: "inspire.py", Needs: []string{"owner_repo_or_category"}, Timeout: 240},
	{Key: "lineage", Label: "演进谱系", Icon: "🌳", Desc: "创新点识别 + 项目演化分支 + 贡献者参与分析", Script: "lineage.py", Needs: []string{"owner", "repo"}, Timeout: 180},
	{Key: "repro", Label: "合规复现", Icon: "✅", Desc: "许可证/依赖锁定/容器化/密钥泄漏/复现性评分", Script: "repro.py", Needs: []string{"owner", "repo"}, Timeout: 120},
{Key: "report", Label: "进度报告", Icon: "📋", Desc: "周报 + 里程碑追踪 + 风险预警（交通灯系统）", Script: "report.py", Needs: []string{"owner", "repo"}, Timeout: 180},
	{Key: "visual", Label: "成果可视化", Icon: "📊", Desc: "交互式 Plotly 图表：时间线/热力图/语言饼图/Gantt", Script: "visual.py", Needs: []string{"owner", "repo"}, Timeout: 240},
}

// chainRequest 统一分析请求。
type chainRequest struct {
	Dimensions []string `json:"dimensions"` // 选中的维度 key 列表
	Keyword    string   `json:"keyword"`
	Owner      string   `json:"owner"`
	Repo       string   `json:"repo"`
	Category   string   `json:"category"`
}

// dimensionResult 单个维度的运行结果。
type dimensionResult struct {
	Key       string            `json:"key"`
	Label     string            `json:"label"`
	OK        bool              `json:"ok"`
	Error     string            `json:"error,omitempty"`
	Duration  string            `json:"duration"`
	Command   string            `json:"command"`
	Stdout    string            `json:"stdout"`
	OutDir    string            `json:"out_dir"`
	Artifacts map[string]string `json:"artifacts"`
}

// chainResponse 统一分析 API 返回。
type chainResponse struct {
	OK        bool                        `json:"ok"`
	SessionID string                      `json:"session_id"`
	Duration  string                      `json:"duration"`
	Results   map[string]*dimensionResult `json:"results"`
}

type Options struct {
	Port         int
	ResearchDir  string // scripts/research 目录
	WorkDir      string // 产物输出根目录
	Token        string // 可选鉴权 token
	LLMKey       string // LLM API Key（内置到服务端，非前端输入）
	LLMBase      string // LLM API Base URL
	LLMModel     string // LLM Model 名称
	GitLinkToken string // GitLink API Token（内置，Python 子进程通过 GITLINK_TOKEN 使用）
}

func NewServerCmd() *cobra.Command {
	opts := Options{Port: defaultPort, ResearchDir: "scripts/research", WorkDir: "research-output", LLMKey: "sk-52b7f7db19fe41118d3b931bded9403c", LLMBase: "https://api.deepseek.com/anthropic", LLMModel: "deepseek-v4-pro", GitLinkToken: "330e35fbb163da345df372b4cbe1cf973aae2b67"}
	cmd := &cobra.Command{
		Use:   "server",
		Short: "启动子赛题四网页终端（HTTP 演示服务）",
		RunE: func(cmd *cobra.Command, args []string) error {
			return Run(opts)
		},
	}
	cmd.Flags().IntVarP(&opts.Port, "port", "p", defaultPort, "监听端口")
	cmd.Flags().StringVar(&opts.ResearchDir, "research-dir", "scripts/research", "scripts/research 目录")
	cmd.Flags().StringVar(&opts.WorkDir, "work-dir", "research-output", "产物输出根目录")
	cmd.Flags().StringVar(&opts.Token, "token", "", "可选鉴权 token（亦可用 DEMO_TOKEN 环境变量）")
	cmd.Flags().StringVar(&opts.LLMKey, "llm-key", opts.LLMKey, "LLM API Key（默认内置）")
	cmd.Flags().StringVar(&opts.LLMBase, "llm-base", opts.LLMBase, "LLM API Base URL")
	cmd.Flags().StringVar(&opts.LLMModel, "llm-model", opts.LLMModel, "LLM Model 名称")
	cmd.Flags().StringVar(&opts.GitLinkToken, "gitlink-token", opts.GitLinkToken, "GitLink API Token（默认内置）")
	return cmd
}

// Run 启动 HTTP 服务（阻塞）。
func Run(opts Options) error {
	if t := os.Getenv("DEMO_TOKEN"); t != "" && opts.Token == "" {
		opts.Token = t
	}
	// LLM key 优先级：--llm-key > DEEPSEEK_API_KEY > LLM_API_KEY
	if opts.LLMKey == "" {
		if k := os.Getenv("DEEPSEEK_API_KEY"); k != "" {
			opts.LLMKey = k
		} else if k := os.Getenv("LLM_API_KEY"); k != "" {
			opts.LLMKey = k
		}
	}
	// GitLink token：内置默认值注入进程环境，子进程自动继承
	if opts.GitLinkToken != "" && os.Getenv("GITLINK_TOKEN") == "" {
		_ = os.Setenv("GITLINK_TOKEN", opts.GitLinkToken)
	}
	// 让 python 子进程复用本二进制（gitlink_data.cli_path 读 GITLINK_CLI），免去额外配置
	if os.Getenv("GITLINK_CLI") == "" {
		if exe, err := filepath.Abs(os.Args[0]); err == nil {
			_ = os.Setenv("GITLINK_CLI", exe)
		}
	}
	_ = os.MkdirAll(opts.WorkDir, 0o755)

	mux := http.NewServeMux()
	h := &handler{opts: opts}

	mux.HandleFunc("GET /api/scenarios", h.handleScenarios)
	mux.HandleFunc("POST /api/run", h.handleRun)
	mux.HandleFunc("GET /api/result/{key}", h.handleResult)
	mux.HandleFunc("GET /api/health", h.handleHealth)
	mux.HandleFunc("GET /api/dimensions", h.handleDimensions)
	mux.HandleFunc("POST /api/chain", h.handleChain)

	sub, err := fs.Sub(staticFS, "static")
	if err != nil {
		return fmt.Errorf("static fs: %w", err)
	}
	mux.Handle("GET /", http.FileServer(http.FS(sub)))

	addr := fmt.Sprintf(":%d", opts.Port)
	fmt.Fprintf(os.Stderr, "子赛题四 网页终端已启动: http://localhost%s\n", addr)
	fmt.Fprintf(os.Stderr, "  research-dir=%s work-dir=%s auth=%v\n", opts.ResearchDir, opts.WorkDir, opts.Token != "")
	if opts.LLMKey != "" {
		fmt.Fprintf(os.Stderr, "  LLM: enabled (model=%s)\n", opts.LLMModel)
	} else {
		fmt.Fprintf(os.Stderr, "  LLM: disabled (no key)\n")
	}
	srv := &http.Server{Addr: addr, Handler: mux, ReadHeaderTimeout: 10 * time.Second}
	return srv.ListenAndServe()
}

type handler struct {
	opts Options
	mu   sync.Mutex // 串行化场景执行，避免并发打爆 GitLink API
}

func (h *handler) authed(r *http.Request) bool {
	if h.opts.Token == "" {
		return true
	}
	return r.Header.Get("X-Demo-Token") == h.opts.Token
}

func (h *handler) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, map[string]any{"ok": true, "scenarios": len(scenarios), "dimensions": len(dimensions)})
}

func (h *handler) handleDimensions(w http.ResponseWriter, r *http.Request) {
	if !h.authed(r) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	writeJSON(w, map[string]any{"ok": true, "dimensions": dimensions})
}

func (h *handler) handleScenarios(w http.ResponseWriter, r *http.Request) {
	if !h.authed(r) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	writeJSON(w, map[string]any{"ok": true, "scenarios": scenarios})
}

// handleResult 返回某场景最近一次运行的产物（供 result.html 独立结果页按 key 读取，
// URL 可刷新/分享，便于演示讲解）。无需鉴权串行锁——只读已落盘产物。
func (h *handler) handleResult(w http.ResponseWriter, r *http.Request) {
	if !h.authed(r) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	key := r.PathValue("key")
	sc, ok := findScenario(key)
	if !ok {
		writeJSON(w, map[string]any{"ok": false, "error": "unknown scenario: " + key})
		return
	}
	outDir := filepath.Join(h.opts.WorkDir, sc.Key)
	writeJSON(w, map[string]any{
		"ok":        true,
		"scenario":  sc.Key,
		"label":     sc.Label,
		"desc":      sc.Desc,
		"artifacts": readArtifacts(outDir),
	})
}

type runRequest struct {
	Scenario string `json:"scenario"`
	Owner    string `json:"owner"`
	Repo     string `json:"repo"`
	Keyword  string `json:"keyword"`
	Category string `json:"category"`
}

func (h *handler) handleRun(w http.ResponseWriter, r *http.Request) {
	if !h.authed(r) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	var req runRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "bad request: " + err.Error()})
		return
	}
	sc, ok := findScenario(req.Scenario)
	if !ok {
		writeJSON(w, map[string]any{"ok": false, "error": "unknown scenario: " + req.Scenario})
		return
	}
	for _, need := range sc.Needs {
		if (need == "keyword" && req.Keyword == "") ||
			(need == "owner" && req.Owner == "") ||
			(need == "repo" && req.Repo == "") ||
			(need == "category" && req.Category == "") {
			writeJSON(w, map[string]any{"ok": false, "error": "missing parameter: " + need})
			return
		}
	}

	// 串行执行：一次只跑一个场景，保护 GitLink API。
	h.mu.Lock()
	defer h.mu.Unlock()

	scriptPath := filepath.Join(h.opts.ResearchDir, sc.Script)
	outDir := filepath.Join(h.opts.WorkDir, sc.Key)
	_ = os.MkdirAll(outDir, 0o755)

	argv := []string{scriptPath, "--out", outDir}
	if contains(sc.Needs, "owner") {
		argv = append(argv, "--owner", req.Owner, "--repo", req.Repo)
	}
	if contains(sc.Needs, "keyword") {
		argv = append(argv, "--keywords", req.Keyword)
	}
	if contains(sc.Needs, "category") {
		argv = append(argv, "--category", req.Category)
	}
	// chain 支持可选焦点仓（省略则自动取热点榜 top-1）
	if sc.Key == "chain" && req.Owner != "" && req.Repo != "" {
		argv = append(argv, "--repo", req.Owner+"/"+req.Repo)
	}

	// python3 优先，回退 python
	py, err := pythonBin()
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": err.Error()})
		return
	}
	cmd := exec.Command(py, argv...)
	// 子进程复用本二进制（python 经 GITLINK_CLI 找 gitlink-cli）；直接注入子进程 env，最稳。
	env := os.Environ()
	if !envHas(env, "GITLINK_CLI") {
		if exe, err := filepath.Abs(os.Args[0]); err == nil {
			env = append(env, "GITLINK_CLI="+exe)
		}
	}
	cmd.Env = env
	start := time.Now()
	out, err := cmd.CombinedOutput()
	dur := time.Since(start)
	resp := map[string]any{
		"ok":       err == nil,
		"scenario": sc.Key,
		"command":  py + " " + strings.Join(argv, " "),
		"duration": dur.Truncate(time.Millisecond).String(),
		"stdout":   string(out),
		"out_dir":  outDir,
	}
	if err != nil {
		resp["error"] = err.Error()
	}
	// 附带读取关键产物（json + 第一个 mmd + report.md），便于前端直接渲染
	resp["artifacts"] = readArtifacts(outDir)
	writeJSON(w, resp)
}

// handleChain 统一全链路科研分析入口
//
// 接受维度 key 列表 + 关键词/分类/仓库参数，串行执行各 Python 脚本，
// 聚合结果返回给前端 Tab 面板渲染。每个维度独立计时并记录成功/失败状态。
// 复用全局 mutex 防止并发打爆 GitLink API。
//
// 参数:
//   w - HTTP ResponseWriter
//   r - HTTP Request（JSON body 为 chainRequest）
//
// 返回:
//   JSON chainResponse，包含 session_id、总耗时、各维度结果映射
func (h *handler) handleChain(w http.ResponseWriter, r *http.Request) {
	if !h.authed(r) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	var req chainRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, chainResponse{OK: false, Results: map[string]*dimensionResult{"_error": {Key: "_error", Label: "parse error", OK: false, Error: "bad request: " + err.Error()}}})
		return
	}
	if len(req.Dimensions) == 0 {
		writeJSON(w, chainResponse{OK: false, Results: map[string]*dimensionResult{"_error": {Key: "_error", Label: "no dimensions", OK: false, Error: "至少选择一个分析维度"}}})
		return
	}

	// 校验所有维度 key 合法
	for _, dk := range req.Dimensions {
		if _, ok := findDimension(dk); !ok {
			writeJSON(w, chainResponse{OK: false, Results: map[string]*dimensionResult{"_error": {Key: dk, Label: dk, OK: false, Error: "未知分析维度: " + dk}}})
			return
		}
	}

	// 串行执行（复用 mutex 保护 GitLink API）
	h.mu.Lock()
	defer h.mu.Unlock()

	sessionID := fmt.Sprintf("session_%s", time.Now().Format("20060102_150405"))
	results := make(map[string]*dimensionResult)
	totalStart := time.Now()

	for _, dk := range req.Dimensions {
		dim, _ := findDimension(dk)
		dimArgs, err := buildDimensionArgs(dim, req)
		outDir := filepath.Join(h.opts.WorkDir, sessionID, dim.Key)
		_ = os.MkdirAll(outDir, 0o755)

		dr := &dimensionResult{Key: dim.Key, Label: dim.Label, OutDir: outDir}

		if err != nil {
			dr.OK = false
			dr.Error = err.Error()
			results[dim.Key] = dr
			continue
		}

		py, err := pythonBin()
		if err != nil {
			dr.OK = false
			dr.Error = err.Error()
			results[dim.Key] = dr
			continue
		}

		scriptPath := filepath.Join(h.opts.ResearchDir, dim.Script)
		argv := append([]string{scriptPath, "--out", outDir}, dimArgs...)
		cmd := exec.Command(py, argv...)
		env := os.Environ()
		if !envHas(env, "GITLINK_CLI") {
			if exe, e2 := filepath.Abs(os.Args[0]); e2 == nil {
				env = append(env, "GITLINK_CLI="+exe)
			}
		}
		// LLM key 注入子进程
		if h.opts.LLMKey != "" {
			if !envHas(env, "DEEPSEEK_API_KEY") && !envHas(env, "LLM_API_KEY") {
				env = append(env, "DEEPSEEK_API_KEY="+h.opts.LLMKey)
			}
			if !envHas(env, "DEEPSEEK_BASE_URL") && !envHas(env, "LLM_BASE_URL") {
				env = append(env, "DEEPSEEK_BASE_URL="+h.opts.LLMBase)
			}
			if !envHas(env, "DEEPSEEK_MODEL") && !envHas(env, "LLM_MODEL") {
				env = append(env, "DEEPSEEK_MODEL="+h.opts.LLMModel)
			}
		}
		cmd.Env = env

		dr.Command = py + " " + strings.Join(argv, " ")
		dimStart := time.Now()
		out, runErr := cmd.CombinedOutput()
		dr.Duration = time.Since(dimStart).Truncate(time.Millisecond).String()
		dr.Stdout = string(out)
		dr.OK = runErr == nil
		if runErr != nil {
			dr.Error = runErr.Error()
		}
		dr.Artifacts = readArtifacts(outDir)
		results[dim.Key] = dr
	}

	resp := chainResponse{
		OK:        true,
		SessionID: sessionID,
		Duration:  time.Since(totalStart).Truncate(time.Millisecond).String(),
		Results:   results,
	}
	writeJSON(w, resp)
}

func findScenario(key string) (scenarioDef, bool) {
	for _, s := range scenarios {
		if s.Key == key || strings.EqualFold(s.Key, key) {
			return s, true
		}
	}
	return scenarioDef{}, false
}

func findDimension(key string) (dimensionDef, bool) {
	for _, d := range dimensions {
		if d.Key == key || strings.EqualFold(d.Key, key) {
			return d, true
		}
	}
	return dimensionDef{}, false
}

func readArtifacts(dir string) map[string]string {
	out := map[string]string{}
	// json 产物（取第一个 *.json）
	if entries, err := os.ReadDir(dir); err == nil {
		for _, e := range entries {
			if e.IsDir() {
				continue
			}
			name := e.Name()
			switch {
			case strings.HasSuffix(name, ".json"):
				b, _ := os.ReadFile(filepath.Join(dir, name))
				out["json"] = string(b)
			case strings.HasSuffix(name, ".mmd"):
				b, _ := os.ReadFile(filepath.Join(dir, name))
				out["mermaid"] = string(b)
			case name == "visual.html":
				b, _ := os.ReadFile(filepath.Join(dir, name))
				out["html"] = string(b)
			case strings.HasSuffix(name, ".md"):
				// 第一份 .md 报告（report.md / weekly_report.md / compliance_report.md）
				if _, ok := out["report"]; !ok {
					b, _ := os.ReadFile(filepath.Join(dir, name))
					out["report"] = string(b)
				}
			}
		}
	}
	return out
}

func pythonBin() (string, error) {
	// 候选按 Linux 习惯 python3 优先，再 python / py(Windows)。
	// 必须实测能产出：Windows 的 WindowsApps\python3.exe 是 Store 桩，对 -c 也可能 exit 0 但不真正执行，
	// 故用「stdout 必须含 PYOK」来拦截桩。
	for _, name := range []string{"python3", "python", "py"} {
		path, err := exec.LookPath(name)
		if err != nil {
			continue
		}
		if out, err := exec.Command(path, "-c", "print('PYOK')").Output(); err == nil &&
			strings.Contains(string(out), "PYOK") {
			return path, nil
		}
	}
	return "", fmt.Errorf("python 未安装；容器需内置 python3 并 pip install -r scripts/research/requirements.txt")
}

func contains(xs []string, s string) bool {
	for _, x := range xs {
		if x == s {
			return true
		}
	}
	return false
}

// buildDimensionArgs 根据维度定义和请求参数构建 Python 脚本 CLI 参数。
func buildDimensionArgs(dim dimensionDef, req chainRequest) ([]string, error) {
	argv := []string{} // script 在调用方追加
	hasKeyword := req.Keyword != ""
	hasCategory := req.Category != ""
	hasRepo := req.Owner != "" && req.Repo != ""

	// 检查每个 need
	for _, need := range dim.Needs {
		switch need {
		case "keyword_or_category":
			if !hasKeyword && !hasCategory {
				return nil, fmt.Errorf("维度 %s 需要 --keywords 或 --category", dim.Key)
			}
			if hasCategory {
				argv = append(argv, "--category", req.Category)
			} else {
				argv = append(argv, "--keywords", req.Keyword)
			}
		case "keyword":
			if !hasKeyword {
				return nil, fmt.Errorf("维度 %s 需要 --keywords", dim.Key)
			}
			argv = append(argv, "--keywords", req.Keyword)
		case "owner_repo_or_category":
			if !hasRepo && !hasCategory {
				return nil, fmt.Errorf("维度 %s 需要 --owner/--repo 或 --category", dim.Key)
			}
			if hasCategory {
				argv = append(argv, "--category", req.Category)
			} else {
				argv = append(argv, "--owner", req.Owner, "--repo", req.Repo)
			}
		case "owner":
			if !hasRepo {
				return nil, fmt.Errorf("维度 %s 需要 --owner 和 --repo", dim.Key)
			}
			argv = append(argv, "--owner", req.Owner)
		case "repo":
			if !hasRepo {
				return nil, fmt.Errorf("维度 %s 需要 --repo", dim.Key)
			}
			argv = append(argv, "--repo", req.Repo)
		}
	}
	return argv, nil
}

// envHas 报告环境变量切片里是否已含某 KEY（形如 "KEY=..."）。
func envHas(env []string, key string) bool {
	prefix := key + "="
	for _, e := range env {
		if strings.HasPrefix(e, prefix) {
			return true
		}
	}
	return false
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(v)
}
