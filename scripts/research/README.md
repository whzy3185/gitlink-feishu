# scripts/research — 子赛题四·科研辅助算法层

本目录是子赛题四「应用 GitLink 辅助科研」的 **Python 工具代码**（赛题交付物之一）。
采用 **Go 出数据 + Python 做算法** 的分工：所有原始数据经现有 gitlink-cli（25 个域）获取，
Python 负责知识图谱 / 协作匹配 / 可视化 / 复现性 / 报告等算法与产物生成。

## 文件

| 文件 | 场景 | 说明 |
|------|------|------|
| `gitlink_data.py` | 共享 | 调 gitlink-cli、解析 envelope、限速、分页、只读 health SQLite |
| `collect.py` | 共享 | 各命令的采集器 + 字段归一化（login/repo 全名等） |
| `lineage.py` | S1 | 仓库级科研项目洞悉：提交/分支/PR/文档/实验代码演进 + 创新点 |
| `graph_build.py` | S2 | 科研知识图谱（networkx）：节点/边模型 + 主题词典 + mermaid/DOT |
| `repro.py` | S3 | 合规与复现性：license/密钥/依赖/复现清单 + 评分 |
| `match.py` | S4 | 科研协作智能匹配：学者画像 × 仓库缺口 TF-IDF + 余弦 |
| `report.py` | S5 | 科研进度智能跟踪与预警：周统计 + 里程碑 + 风险阈值 |
| `visual.py` | S6 | 科研成果可视化：plotly 时间线/热力/饼/甘特/论文关联 |
| `templates/*.j2` | 全部 | jinja2 中文报告模板 |
| `test_*.py` | — | 单元测试（`pytest scripts/research/`） |

## 安装

```bash
pip install -r scripts/research/requirements.txt
```

## 环境变量

| 变量 | 默认 | 说明 |
|------|------|------|
| `GITLINK_CLI` | `gitlink-cli` | CLI 可执行路径；本地 Windows 开发可设 `./gitlink-cli.exe` |
| `GITLINK_CLI_INTERVAL` | `0.6` | CLI 调用最小间隔（秒），防 API 限流 |
| `GITLINK_HEALTH_DB` | `~/.agents/skills/gitlink-health/data/gitlink_health.db` | health SQLite 路径（图谱/匹配读历史协作数据） |

## 数据来源

- **结构化数据**：`repo/issue/pr/search/user/org/milestone/file/license` 等域，输出统一 envelope `{ok,data,meta}`。
- **提交历史**：Raw API `gitlink-cli api GET /{owner}/{repo}/commits`（shortcuts 无 repo +commits，路径已在 mindspore 仓库核验通过）。
- **历史协作**：直接只读 `health` SQLite（`users/repos/issues/pulls/tags` 表）。

## 验证仓库

**主验证仓库：`mindspore-Ecosystem/mindspore`**（华为 MindSpore 深度学习框架镜像，真实科研级 AI 框架）：
issues≈20346 / PR=9 / 贡献者=6 / star=31，协作数据充足，适合 S2/S4/S5；同时是 Python 研究代码仓库，适合 S1/S3/S6。
S2 知识图谱为多仓库场景，按关键词 `search +repos` 跨仓库构建。

### 已核验响应形状（mindspore 实测，供各场景脚本参考）
- `repo +info` → `default_branch`/`issues_count`/`pull_requests_count`/`contributor_users_count`/`size`/`clone_url`
- `repo +contributors[]` → `login`/`name`/`contributions`/`contribution_perc`/`email`
- `issue +list[]` → 标题 `subject`、时间 `created_at`、`status`、`priority_name`、`milestone_name`、`author`、`number`/`project_issues_index`
- `pr +list[]` → 标题 `title`、创建时间 `pr_created_unix`(秒)、`status`(0=open/1=merged/2=closed)、`reviewers`、`index`
- `api GET /commits[]` → `sha`/`message`/`timestamp`/`author.login`/`committer.login`
- `repo +languages` → `{"Python":"99.7%", ...}`

## 复现

每个场景配一个 `skills/<skill>/examples/<scenario>-workflow.sh`，串联 CLI → Python → 报告产物，
即赛题要求的「可复现执行脚本」。
