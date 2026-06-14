// Command insight is an end-to-end research-team insight workflow (subtopic 3).
//
// It composes existing gitlink-cli commands to, for a set of members (or a
// research keyword), collect each member's GitLink platform statistics, aggregate
// them, and emit a team insight report (Markdown), a self-contained visual report
// (HTML with inline SVG: ability radar + collaboration network) and a command log.
//
// Chain: [search +users] -> profile statistics x N -> aggregate -> visualize -> archive.
// All operations are read-only.
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
	"sort"
	"strings"
	"time"
)

// ---------- data model ----------

type envelope struct {
	Data json.RawMessage `json:"data"`
}

type abilityScores struct {
	Influence    int `json:"influence"`
	Contribution int `json:"contribution"`
	Activity     int `json:"activity"`
	Experience   int `json:"experience"`
	Language     int `json:"language"`
}

func (a abilityScores) get(key string) int {
	switch key {
	case "influence":
		return a.Influence
	case "contribution":
		return a.Contribution
	case "activity":
		return a.Activity
	case "experience":
		return a.Experience
	case "language":
		return a.Language
	}
	return 0
}

type developData struct {
	Platform abilityScores `json:"platform"`
	User     struct {
		abilityScores
		LanguagesPercent  map[string]float64 `json:"languages_percent"`
		EachLanguageScore map[string]int     `json:"each_language_score"`
	} `json:"user"`
}

type majorData struct {
	Categories []string `json:"categories"`
}

type roleEntry struct {
	Count   int     `json:"count"`
	Percent float64 `json:"percent"`
}

type roleData struct {
	Role               map[string]roleEntry `json:"role"`
	TotalProjectsCount int                  `json:"total_projects_count"`
}

type activityData struct {
	Dates             []string `json:"dates"`
	CommitsCount      []int    `json:"commits_count"`
	IssuesCount       []int    `json:"issues_count"`
	PullRequestsCount []int    `json:"pull_requests_count"`
}

type headmapEntry struct {
	Date          string `json:"date"`
	Contributions int    `json:"contributions"`
}

type headmapsData struct {
	TotalContributions int            `json:"total_contributions"`
	Headmaps           []headmapEntry `json:"headmaps"`
}

type userInfo struct {
	Login               string `json:"login"`
	Name                string `json:"name"`
	RealName            string `json:"real_name"`
	CustomDepartment    string `json:"custom_department"`
	City                string `json:"city"`
	Province            string `json:"province"`
	CreatedTime         string `json:"created_time"`
	CommonProjectsCount int    `json:"common_projects_count"`
	MirrorProjectsCount int    `json:"mirror_projects_count"`
}

type subject struct {
	Login string
	Info  userInfo
	Dev   developData
	Major []string
	Role  roleData
	Act   activityData
	Heat  headmapsData
}

func (s subject) displayName() string {
	if s.Info.RealName != "" {
		return s.Info.RealName
	}
	if s.Info.Name != "" {
		return s.Info.Name
	}
	return s.Login
}

var abilityDims = []struct{ Key, ZH string }{
	{"influence", "影响力"},
	{"contribution", "贡献度"},
	{"activity", "活跃度"},
	{"experience", "项目经验"},
	{"language", "语言能力"},
}

// ---------- config / flags ----------

type options struct {
	Config   string
	Members  string
	Keyword  string
	Max      int
	TeamName string
	Out      string
	CLIBin   string
	NoFetch  bool
	DataRoot string
}

type teamConfig struct {
	TeamName string   `json:"team_name"`
	Members  []string `json:"members"`
}

// ---------- command execution + logging ----------

type commandResult struct {
	Command []string `json:"command"`
	Status  string   `json:"status"`
	Time    string   `json:"time"`
	Error   string   `json:"error,omitempty"`
}

var commandLog []commandResult

func runCLI(cliBin string, args []string, retries int) (string, error) {
	display := append([]string{"gitlink-cli"}, args...)
	var stdout, stderr bytes.Buffer
	var lastErr error
	for attempt := 0; attempt <= retries; attempt++ {
		stdout.Reset()
		stderr.Reset()
		cmd := exec.Command(cliBin, args...) // #nosec G204 - cliBin is operator-provided
		low := strings.ToLower(cliBin)
		if strings.HasSuffix(low, ".cmd") || strings.HasSuffix(low, ".bat") {
			cmd = exec.Command("cmd", append([]string{"/c", cliBin}, args...)...) // #nosec G204
		}
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr
		lastErr = cmd.Run()
		if lastErr == nil {
			commandLog = append(commandLog, commandResult{Command: display, Status: "ok", Time: now()})
			return stdout.String(), nil
		}
		if attempt < retries {
			time.Sleep(3 * time.Second)
		}
	}
	msg := strings.TrimSpace(stderr.String())
	commandLog = append(commandLog, commandResult{Command: display, Status: "failed", Time: now(), Error: truncate(msg, 300)})
	return "", fmt.Errorf("命令失败 gitlink-cli %s: %v: %s", strings.Join(args, " "), lastErr, msg)
}

func now() string { return time.Now().Format(time.RFC3339) }

func truncate(s string, n int) string {
	if len(s) > n {
		return s[:n]
	}
	return s
}

// ---------- fetch + load ----------

var statEndpoints = [][2]string{
	{"develop.json", "statistics/develop"},
	{"major.json", "statistics/major"},
	{"role.json", "statistics/role"},
	{"activity.json", "statistics/activity"},
	{"contribution.json", "headmaps"},
}

func fetchMember(cliBin, login, dir string) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	for _, ep := range statEndpoints {
		out, err := runCLI(cliBin, []string{"api", "GET", fmt.Sprintf("/users/%s/%s", login, ep[1]), "--format", "json"}, 2)
		if err != nil {
			return err
		}
		if err := os.WriteFile(filepath.Join(dir, ep[0]), []byte(out), 0o644); err != nil {
			return err
		}
	}
	out, err := runCLI(cliBin, []string{"user", "+info", "--login", login, "--format", "json"}, 2)
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, "user.json"), []byte(out), 0o644)
}

func parseInto(path string, v interface{}) error {
	raw, err := os.ReadFile(path) // #nosec G304 - path derived from operator inputs
	if err != nil {
		return err
	}
	var env envelope
	if err := json.Unmarshal(raw, &env); err != nil {
		return fmt.Errorf("%s: %w", path, err)
	}
	payload := env.Data
	if len(payload) == 0 {
		payload = raw // tolerate already-unwrapped payloads
	}
	if err := json.Unmarshal(payload, v); err != nil {
		return fmt.Errorf("%s: %w", path, err)
	}
	return nil
}

func loadSubject(dir string) (subject, error) {
	s := subject{Login: filepath.Base(dir)}
	if err := parseInto(filepath.Join(dir, "develop.json"), &s.Dev); err != nil {
		return s, err
	}
	var mj majorData
	if err := parseInto(filepath.Join(dir, "major.json"), &mj); err == nil {
		s.Major = mj.Categories
	}
	_ = parseInto(filepath.Join(dir, "role.json"), &s.Role)
	_ = parseInto(filepath.Join(dir, "activity.json"), &s.Act)
	_ = parseInto(filepath.Join(dir, "contribution.json"), &s.Heat)
	if err := parseInto(filepath.Join(dir, "user.json"), &s.Info); err == nil && s.Info.Login != "" {
		s.Login = s.Info.Login
	}
	return s, nil
}

// ---------- aggregation helpers ----------

func teamAverages(subs []subject) map[string]int {
	avg := map[string]int{}
	if len(subs) == 0 {
		return avg
	}
	for _, d := range abilityDims {
		sum := 0
		for _, s := range subs {
			sum += s.Dev.User.get(d.Key)
		}
		avg[d.Key] = sum / len(subs)
	}
	return avg
}

func grade(v int) string {
	switch {
	case v >= 90:
		return "卓越"
	case v >= 75:
		return "优秀"
	case v >= 60:
		return "良好"
	case v >= 40:
		return "一般"
	default:
		return "较弱"
	}
}

// disciplineCoverage returns shared (count>1) and unique categories.
func disciplineCoverage(subs []subject) (shared map[string][]string, unique []string) {
	all := map[string][]string{}
	for _, s := range subs {
		for _, c := range s.Major {
			all[c] = append(all[c], s.Login)
		}
	}
	shared = map[string][]string{}
	for c, who := range all {
		if len(who) > 1 {
			shared[c] = who
		} else {
			unique = append(unique, c)
		}
	}
	sort.Strings(unique)
	return shared, unique
}

func topLanguages(subs []subject, n int) []string {
	agg := map[string]float64{}
	for _, s := range subs {
		for lang, pct := range s.Dev.User.LanguagesPercent {
			agg[lang] += pct
		}
	}
	type kv struct {
		k string
		v float64
	}
	var list []kv
	for k, v := range agg {
		list = append(list, kv{k, v})
	}
	sort.Slice(list, func(i, j int) bool { return list[i].v > list[j].v })
	var out []string
	for i, e := range list {
		if i >= n {
			break
		}
		out = append(out, e.k)
	}
	return out
}

func activitySum(a activityData) (commits, issues, prs int) {
	for _, v := range a.CommitsCount {
		commits += v
	}
	for _, v := range a.IssuesCount {
		issues += v
	}
	for _, v := range a.PullRequestsCount {
		prs += v
	}
	return
}

func strongest(subs []subject) subject {
	best := subs[0]
	bestScore := -1
	for _, s := range subs {
		total := 0
		for _, d := range abilityDims {
			total += s.Dev.User.get(d.Key)
		}
		if total > bestScore {
			bestScore = total
			best = s
		}
	}
	return best
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "错误:", err)
		os.Exit(1)
	}
}

func run() error {
	opts := parseFlags()

	members, teamName, err := resolveMembers(opts)
	if err != nil {
		return err
	}
	fmt.Printf("成员：%s\n", strings.Join(members, ", "))

	dataRoot := opts.DataRoot
	if dataRoot == "" {
		dataRoot = filepath.Join(opts.Out, "data")
	}
	if err := os.MkdirAll(opts.Out, 0o755); err != nil {
		return err
	}

	var subs []subject
	for i, login := range members {
		dir := filepath.Join(dataRoot, login)
		if !opts.NoFetch {
			fmt.Printf("[2/4] 采集画像 %d/%d：%s\n", i+1, len(members), login)
			if err := fetchMember(opts.CLIBin, login, dir); err != nil {
				return err
			}
		}
		s, err := loadSubject(dir)
		if err != nil {
			fmt.Printf("      ⚠️ 跳过 %s：%v\n", login, err)
			continue
		}
		subs = append(subs, s)
	}
	if len(subs) == 0 {
		return errors.New("无可用成员数据")
	}

	fmt.Println("[3/4] 生成团队可视化报告与洞察报告…")
	html := buildHTML(teamName, subs)
	if err := os.WriteFile(filepath.Join(opts.Out, "team-report.html"), []byte(html), 0o644); err != nil {
		return err
	}
	md := buildInsightMarkdown(teamName, subs)
	if err := os.WriteFile(filepath.Join(opts.Out, "team-insight.md"), []byte(md), 0o644); err != nil {
		return err
	}

	fmt.Println("[4/4] 写入命令日志…")
	commandLog = append(commandLog, commandResult{Command: []string{"insight", "render", "team-report.html"}, Status: "ok", Time: now()})
	logJSON, err := json.MarshalIndent(commandLog, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(opts.Out, "command_log.json"), logJSON, 0o644); err != nil {
		return err
	}

	fmt.Printf("\n完成 输出目录：%s\n  - team-report.html\n  - team-insight.md\n  - command_log.json（%d 条）\n", opts.Out, len(commandLog))
	return nil
}

func parseFlags() options {
	var opts options
	fs := flag.CommandLine
	fs.StringVar(&opts.Config, "config", "", "团队配置 JSON：{\"team_name\":..,\"members\":[..]}")
	fs.StringVar(&opts.Members, "members", "", "成员 login，逗号分隔")
	fs.StringVar(&opts.Keyword, "keyword", "", "按研究方向发现成员（search +users）")
	fs.IntVar(&opts.Max, "max", 5, "--keyword 模式下最多成员数")
	fs.StringVar(&opts.TeamName, "team-name", "科研团队", "团队名称")
	fs.StringVar(&opts.Out, "out", "./out", "输出目录")
	fs.StringVar(&opts.CLIBin, "cli-bin", firstNonEmpty(os.Getenv("GITLINK_CLI_BIN"), "gitlink-cli"), "gitlink-cli 可执行文件路径")
	fs.BoolVar(&opts.NoFetch, "no-fetch", false, "离线复现：不调用网络")
	fs.StringVar(&opts.DataRoot, "data-root", "", "--no-fetch 时成员数据根目录（默认 <out>/data）")
	flag.Parse()
	return opts
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}

func resolveMembers(opts options) ([]string, string, error) {
	teamName := opts.TeamName
	var members []string
	if opts.Members != "" {
		for _, m := range strings.Split(opts.Members, ",") {
			if m = strings.TrimSpace(m); m != "" {
				members = append(members, m)
			}
		}
	}
	if opts.Config != "" {
		raw, err := os.ReadFile(opts.Config) // #nosec G304
		if err != nil {
			return nil, "", err
		}
		var cfg teamConfig
		if err := json.Unmarshal(raw, &cfg); err != nil {
			return nil, "", err
		}
		if len(members) == 0 {
			members = cfg.Members
		}
		if cfg.TeamName != "" && opts.TeamName == "科研团队" {
			teamName = cfg.TeamName
		}
	}
	if len(members) == 0 && opts.Keyword != "" {
		fmt.Printf("[1/4] 按方向「%s」发现成员…\n", opts.Keyword)
		found, err := discoverMembers(opts.CLIBin, opts.Keyword, opts.Max)
		if err != nil {
			return nil, "", err
		}
		members = found
	}
	if len(members) == 0 {
		return nil, "", errors.New("需要 --members、--config 或 --keyword")
	}
	return members, teamName, nil
}

func discoverMembers(cliBin, keyword string, limit int) ([]string, error) {
	out, err := runCLI(cliBin, []string{"search", "+users", "-k", keyword, "--format", "json"}, 2)
	if err != nil {
		return nil, err
	}
	var env envelope
	if err := json.Unmarshal([]byte(out), &env); err != nil {
		return nil, err
	}
	var sd struct {
		Users []userInfo `json:"users"`
	}
	_ = json.Unmarshal(env.Data, &sd)
	var logins []string
	for _, u := range sd.Users {
		if u.Login != "" {
			logins = append(logins, u.Login)
		}
		if len(logins) >= limit {
			break
		}
	}
	if len(logins) == 0 {
		return nil, fmt.Errorf("方向「%s」未发现可用成员", keyword)
	}
	return logins, nil
}
