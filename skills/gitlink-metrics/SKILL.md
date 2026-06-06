---
name: gitlink-metrics
version: 1.0.0
description: "仓库量化指标看板：计算 PR 合并率、按月活跃趋势、基尼系数、CR3/CR5、巴士因子与多维评分卡（活跃度/协作/社区/可维护性），生成数据驱动的运营看板。当用户提到「指标」「数据看板」「合并率」「活跃趋势」「巴士因子」「评分卡」「metrics」「dashboard」时触发。"
metadata:
  requires:
    bins: ["gitlink-cli"]
    optional_bins: ["python"]
  cliHelp: "gitlink-cli repo --help"
---

# gitlink-metrics（仓库量化指标看板）

**CRITICAL — 开始前先阅读 [`../gitlink-shared/SKILL.md`](../gitlink-shared/SKILL.md)，其中包含认证、权限处理和 API 注意事项。**
**CRITICAL — GitLink 操作只能用 `gitlink-cli`。禁止用 `gh`（GitHub CLI）操作 GitLink 资源。**
**CRITICAL — 本技能全程只读，不修改任何远程数据。**

> **前置条件：** 先阅读 [`../gitlink-shared/SKILL.md`](../gitlink-shared/SKILL.md)。

## 何时使用本技能

- 需要用量化指标评估仓库状态（合并率、活跃趋势、集中度）
- 用户问「这个项目的数据指标怎么样」「给我一个指标看板」「巴士因子是多少」
- 多仓库横向对比、数据驱动的社区运营

## 与 gitlink-insight 的区别

`gitlink-insight` 偏定性的健康度评估与周报；本技能聚焦**量化指标与评分卡**——
PR 合并率、按月活跃趋势、基尼系数、CR3/CR5、巴士因子、四维评分卡，用可计算、
可对比的数字刻画仓库，功能更偏数据分析。

## 指标说明

| 指标 | 含义 |
|------|------|
| PR 合并率 | 已合并 PR / 总 PR |
| 按月活跃趋势 | 各月提交量分布 |
| 基尼系数 | 贡献分布不平等程度（0 均衡，1 集中） |
| CR3 / CR5 | 前 3 / 前 5 贡献者占比 |
| 巴士因子 | 累计贡献达 50% 所需最少人数 |
| 多维评分卡 | 活跃度 / 协作 / 社区 / 可维护性 / 综合（各 0-100） |

## 工作流：生成指标看板

### 方式 A：配套脚本（推荐）

```bash
python scripts/metrics.py --owner Gitlink --repo gitlink-cli
python scripts/metrics.py --owner Gitlink --repo gitlink-cli --format json
```

参数说明：

| 参数 | 类型 | 必填 | 说明 |
|------|------|:----:|------|
| `--owner` / `--repo` | string | 是* | 仓库（*或用 `--slug`） |
| `--slug` | string | 否 | `owner/repo` 或完整 URL |
| `--commit-pages` | int | 否 | 提交采集页数（每页 50），默认 6 |
| `--format` | string | 否 | `markdown`（默认）或 `json` |
| `--output` | string | 否 | 输出文件 |

### 方式 B：用 gitlink-cli 命令

```bash
gitlink-cli repo +info --owner Gitlink --repo gitlink-cli --format json
gitlink-cli pr +list --owner Gitlink --repo gitlink-cli --format json
gitlink-cli api GET /:owner/:repo/commits --query 'page=1&limit=50' --format json
gitlink-cli api GET /:owner/:repo/contributors --format json
```

## API 注意事项

- 活跃趋势依据提交 `timestamp`（unix）按月统计。
- PR 合并率依据 `pull_request_status`（1=merged）。
- 评分卡为启发式综合评分，用于横向参考而非绝对评价。
- 数据采集全程只读。

## References

- [api-reference.md](references/api-reference.md) — 采集接口与字段
- [metrics-formula.md](references/metrics-formula.md) — 指标计算公式
- [gitlink-shared](../gitlink-shared/SKILL.md) — 认证、全局参数、安全规则
