# 科研团队洞察工作流（research-team-insight）

面向 GitLink 竞赛**子赛题三**的端到端自动化工作流示例（Go 实现，与主仓库技术栈一致）。

从"一组成员"或"一个研究方向"出发，用 `gitlink-cli` 串联**用户发现 → 多人画像采集 → 聚合分析 →
可视化 → 报告归档**的完整闭环，自动产出团队**可视化报告（HTML）**、**洞察报告（Markdown）**与
**命令执行日志**。可视化为内联 SVG 的自包含 HTML（能力雷达 + 协作网络图），浏览器直接打开。

## 命令链路（≥3 个命令）

```
search +users (可选，按方向发现成员)
      └─► profile +ability/+major/+role/+activity/+contribution + user +info   (每位成员 ×6)
              └─► 聚合分析（能力矩阵 / 学科覆盖 / 角色结构）
                      └─► 渲染团队 HTML（能力雷达 + 协作网络图）
                              └─► 洞察报告 team-insight.md + command_log.json
```

> 底层通过 `gitlink-cli api GET /users/{login}/statistics/*` 采集（无需 `profile` 命令组也可运行）。

## 交付物

- `scripts/insight.go` — 工作流主程序（编排 + 采集 + 聚合）
- `scripts/render.go` — Markdown / HTML+SVG 渲染
- `scripts/insight_test.go` — 单元测试
- `scripts/run_demo.ps1` — 一键复现脚本
- `config/team.example.json` — 示例团队配置
- `examples/demo_outputs/` — 真实跑通的固定示例输出（HTML / Markdown / 命令日志 / 原始数据）
- `docs/architecture.md` + `docs/assets/architecture.svg` — 架构与流程图
- `docs/runbook.md` — 运行手册与复现步骤

## 实现语言

主实现采用 **Go**，与 `gitlink-cli` 主仓库技术栈一致：复用根 `go.mod`，可被主仓库 CI
（`go build` / `golangci-lint` / `go test`）直接覆盖；仅用标准库生成内联 SVG，无第三方依赖。

## 快速开始

```bash
# 指定成员
go run ./scripts --members "yystopf,jiangtx,wangyue111" --team-name "课题组A" --out ./out

# 团队配置文件
go run ./scripts --config config/team.example.json --out ./out

# 按研究方向自动发现成员
go run ./scripts --keyword "联邦学习" --max 5 --out ./out

# 离线复现（用已采集数据，不访问网络）
go run ./scripts --config config/team.example.json --out ./out \
    --no-fetch --data-root ./examples/demo_outputs/data
```

> **Windows 提示**：`gitlink-cli` 为 npm 的 `.CMD` 包装，请用 `--cli-bin <路径>\gitlink-cli.CMD`
> 或设置环境变量 `GITLINK_CLI_BIN`；`scripts/run_demo.ps1` 已自动处理。

输出（`--out` 目录）：

| 文件 | 说明 |
|------|------|
| `team-report.html` | 团队可视化（能力雷达、能力矩阵、共有/独有方向、协作网络图）——浏览器直接打开 |
| `team-insight.md` | 团队洞察报告（能力矩阵、学科覆盖、技术栈、角色结构、协作建议） |
| `command_log.json` | 完整命令执行日志（可复现、可审计） |
| `data/<login>/*.json` | 每位成员的原始画像数据 |

## 依赖

- Go 1.21+（仅标准库；复用主仓库 `go.mod`）
- `gitlink-cli`（已登录；用 `api GET` 采集，无需 `profile` 命令组）

## 真实验证

`examples/demo_outputs/` 为对真实账号（@yystopf 何慧·国防科技大学、@jiangtx、@wangyue111）
跑通的固定输出，共 19 条命令调用。工作流对瞬时 TLS 超时自动重试，并对镜像聚合账号在报告中
自动标注"分布偏泛"。详见 [`docs/runbook.md`](docs/runbook.md)。

## 与子赛题二/四的关系

- 子赛题四的 [`gitlink-research-profile`](../../../skills/gitlink-research-profile/SKILL.md) Skill 定义
  "画像能力 + 单人/团队报告规范"。
- 本工作流（子赛题三）把该能力**串联成可复现的端到端解决方案**，加入用户发现、批量编排、聚合分析
  与归档，产出团队级可视化与洞察成果。
