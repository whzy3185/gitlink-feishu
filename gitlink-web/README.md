# GitLink Skills Web Service（科研智能分析服务）

> 子任务四「应用 GitLink 辅助科研」的在线服务。基于 **Flask + DeepSeek API + gitlink-cli**，通过 AI Agent 自动分析 GitLink 平台数据，覆盖科研项目分析、热点追踪、合规校验、协作匹配、进度预警等全链路科研辅助场景。
>
> 🌐 在线地址：<http://121.41.212.97/skills/> ｜ 归属：jiangtx/gitlink-cli 小组

---

## 一、它能做什么

输入一个研究主题或仓库地址，服务会自动调用 `gitlink-cli` 拉取 GitLink 平台数据，再由 DeepSeek 大模型按各 Skill 的工作流生成结构化分析报告。共 **14 个 Skill**，分两类：

### 🎯 核心 Skill（开发者场景）

| Skill | 入口 | 说明 | 典型输出 |
|---|---|---|---|
| 技术调研 `research-tracker` | `/research` | 输入主题→多关键词搜索→深度评估→调研报告 | 热点概览 / 项目排行 / 重点分析 / 趋势建议 |
| 贡献者分析 `contributor-insight` | `/contributor` | 仓库贡献者活跃度与团队健康度 | 团队概览 / 活跃排行 / 重点画像 |
| Issue 分拣 `issue-triage` | `/issue-triage` | 自动分类 Issue、评估紧急度 | 类型分类 / 紧急度 / 行动建议 |
| CI 健康巡检 `ci-health` | `/ci-health` | CI/CD 状态、构建成功率 | 健康度总览 / 构建趋势 / 故障分析 |
| 仓库健康巡检 `repo-health` | `/repo-health` | 综合活跃度、社区、代码产出 | 基本信息 / 活跃度 / 综合评分 |
| PR 效率分析 `pr-analytics` | `/pr-analytics` | PR 吞吐量、合并率 | 吞吐量 / 合并效率 / 贡献排行 |
| 跨维搜索 `cross-search` | `/cross-search` | 仓库+代码+Issue 三维搜索 | 各维命中 / 代码片段 / 讨论热点 |
| 用户分析 `user-analysis` | `/user-analysis` | 用户画像与项目参与 | 基本信息 / 活跃度 / 贡献列表 |
| 仓库对比 `repo-compare` | `/repo-compare` | 两仓库指标对比 | 基本/社区/活动对比 / 结论 |

### 🔬 科研实验室（科研场景，对应子任务四）

| Skill | 入口 | 科研场景 | 典型输出 |
|---|---|---|---|
| 热点追踪 `lab-hotspot` | `/lab-hotspot` | 多关键词搜索 + 领域知识图谱 | 热点概览 / 排行 / 知识图谱 / 趋势 |
| 项目洞悉 `lab-insight` | `/lab-insight` | 仓库+贡献者+PR/Issue 全息分析 | 项目概况 / 社区活跃 / 团队画像 |
| 合规检查 `lab-compliance` | `/lab-compliance` | License/CI/文档/复现性 | 合规评分 / 文档完整性 / 可复现性 |
| 协作匹配 `lab-match` | `/lab-match` | 新手友好度 + 社区健康度 | 项目概览 / 入门友好度 / 贡献方向 |
| 进度跟踪 `lab-track` | `/lab-track` | 多仓库批量巡检 + 三级告警 | 状态概览 / 指标表 / 预警详情 |

> 科研实验室 5 个 Skill 与《课程实践任务》子任务四场景一一对应：仓库级科研项目洞悉、科研热点追踪与知识图谱、科研项目合规与复现性检查、科研协作智能匹配、科研进度智能跟踪与预警。

---

## 二、架构

```
┌──────────────┐   HTTP    ┌──────────────────────────────┐
│  浏览器/评委  │ ───────► │  Flask (app.py, gunicorn)     │
└──────────────┘           │  路由 / /<slug> /api/run      │
                           └──────────────┬───────────────┘
                                          │ 按 Skill 选 agent_*()
                           ┌──────────────▼───────────────┐
                           │  DeepSeek API (Chat)          │
                           │  注入 SKILL.md prompt + 日期  │
                           └──────────────┬───────────────┘
                                          │ 工具调用 gitlink-cli
                           ┌──────────────▼───────────────┐
                           │  gitlink-cli (npm 全局)        │
                           │  repo/issue/pr/release/...    │
                           └──────────────┬───────────────┘
                                          │
                           ┌──────────────▼───────────────┐
                           │  GitLink 平台 OpenAPI         │
                           └──────────────────────────────┘
```

- `app.py`：Flask 服务，14 个 `agent_*` / `lab_*` 函数分别对应 14 个 Skill，每个严格遵循 `skills/*.txt` 里的工作流。
- `skills/*.txt`：各 Skill 的 prompt 工作流（与仓库 `skills/gitlink-*/SKILL.md` 同源）。
- `templates/`：`base.html` / `index.html` / `skill.html` 三套页面。
- `reports/`：运行时生成的历史报告（运行时创建）。

---

## 三、本地运行

```bash
# 1. 依赖
pip install -r requirements.txt          # flask requests gunicorn

# 2. 安装 gitlink-cli（数据源）
npm install -g gitlink-cli
gitlink-cli auth login                    # 登录 GitLink

# 3. 配置大模型 Key（DeepSeek 或兼容 OpenAI 的接口）
export API_KEY="sk-xxxxxxxx"
export API_BASE="https://api.deepseek.com/v1"
export API_MODEL="deepseek-chat"

# 4. 启动
python app.py                             # 默认 5000 端口；--port=80 指定
# 访问 http://localhost:5000/
```

> `API_KEY` 也可不设环境变量，`app.py` 会回退到内置占位（仅本地测试）。生产环境务必用环境变量，不要把 Key 写进代码。

---

## 四、部署到服务器

### 方式 A：DevOps 流水线（推荐，可复现）

仓库根目录 `.devops/gitlink-web.yml` 是建木流水线：master 合并后自动 `git clone` → 同步 `gitlink-web/` 到 `/opt/gitlink-web` → 装 Python 依赖 → 写 systemd 单元 → 重启 → 健康检查。需在建木密钥组 `gitlink_cli` 中配置：
- `wyx_ssh_pass`：服务器 SSH 密码
- `deepseek_api_key`：DeepSeek API Key

### 方式 B：手动脚本

```bash
cd gitlink-web
# 编辑 deploy.py 顶部的 PASSWORD 与 API_KEY（或改成从环境变量读）
python deploy.py
```

`deploy.py` 用 paramiko 走 SSH：装系统依赖 → 装 gitlink-cli → 上传文件 → 装 Python 依赖 → 写 systemd 服务 → 启动。服务以 `gitlink-web.service` 常驻，监听 80 端口。

---

## 五、API 端点

| 端点 | 方法 | 说明 |
|---|---|---|
| `/` | GET | 首页（Skill 列表） |
| `/<slug>` | GET | 单个 Skill 表单页（如 `/research`、`/lab-hotspot`） |
| `/api/run` | POST | 执行 Skill，body：`{"skill":"research-tracker","input":"大模型"}`，返回生成报告 |

---

## 六、目录结构

```
gitlink-web/
├── app.py              # Flask 服务 + 14 个 Skill 处理函数（865 行）
├── deploy.py           # paramiko 一键部署脚本
├── requirements.txt    # flask / requests / gunicorn
├── skills/             # 各 Skill 的 prompt 工作流
│   ├── research-tracker.txt
│   ├── contributor-insight.txt
│   ├── issue-triage.txt
│   └── ci-health.txt
└── templates/
    ├── base.html
    ├── index.html
    └── skill.html
```

---

## 七、与 CLI Skills 的关系

本服务的 `skills/*.txt` 与仓库 `skills/gitlink-*/SKILL.md` 同源、可互换：
- **CLI Skill**（`skills/gitlink-research-tracker/SKILL.md`）：供 Claude Code / Cursor 等 Agent 平台直接调用，离线、可复现。
- **Web Skill**（本服务）：封装成 Web 界面 + 后端 DeepSeek，评委无需装 Agent 即可在浏览器体验。

两者覆盖相同的科研场景，互为印证。
