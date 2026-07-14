---
name: research-mirror-analyzer
version: 1.0.0
description: "自动化分析 GitLink 科研仓库活跃度、贡献者生态，调用 gitlink-cli 拉取仓库代码统计、贡献者原始数据，自动计算指标并输出结构化科研项目洞察报告；支持 OpenClaw、Claude Code、Cursor 等多类 AI Agent 自动触发执行。"
license: MulanPSL-2.0
metadata:
  requires:
    bins_any: ["gitlink-cli"]
    bins_note: "仅依赖官方 gitlink-cli，用于拉取仓库代码统计、贡献者列表 JSON 数据，需提前配置 GITLINK_TOKEN 环境鉴权变量"
  cliHelp: "gitlink-cli repo --help"
  platforms:
    agents:
      - openclaw
      - claude-code
      - cursor
      - generic-agent
    backends:
      - id: gitlink
        cli: gitlink-cli
        url_template: "https://www.gitlink.org.cn/{owner}/{repo}"
        raw_template: "https://www.gitlink.org.cn/{owner}/{repo}/raw/master/{path}"
        api_base: "https://www.gitlink.org.cn/api/v1"
        auth_env: GITLINK_TOKEN
        default: true
---
# research-mirror-analyzer 科研仓库活跃度分析 Skill
## **CRITICAL — 所有仓库查询操作执行前必须校验 gitlink-cli 鉴权状态，GITLINK_TOKEN 缺失/失效直接终止流程并提示用户配置环境变量。**
## **CRITICAL — 开始前必须先阅读 [`../gitlink-shared/SKILL.md`](https://www.gitlink.org.cn/Gitlink/gitlink-cli/tree/master/skills/gitlink-shared/SKILL.md)（仅 GitLink 后端）或对应平台的 CLI 文档。所有 GitHub 操作必须使用 `gh`；所有 GitLab 操作必须使用 `glab`；所有 GitLink 操作必须使用 `gitlink-cli`。禁止混用或替代。**
## **CRITICAL — 调用 gitlink-cli 拉取数据后必须校验返回 JSON 结构，关键字段缺失、空数据时终止分析并返回异常提示。**
## **CRITICAL — 本技能仅读取仓库统计数据，无任何 Issue/代码/仓库写入操作，无需 dry-run 预览、无需用户确认步骤。**

## 功能概述
本 Skill 为 C1 纯文档模式，规则与执行逻辑全部内嵌本文档，无需外部配置文件，Agent 可直接按内置流程执行完整分析链路：
1. 解析用户自然语言指令，提取仓库 `owner` / `repo` 标识；
2. 调用 gitlink-cli 批量拉取仓库代码统计、贡献者原始 JSON 数据；
3. 解析代码行数、总提交、贡献者分布指标；
4. 计算贡献集中度、活跃度分级；
5. 输出标准化 Markdown 科研仓库评估报告。

## 触发场景
用户输入包含以下关键词/意图时自动激活本 Skill：
- "分析仓库XX的科研活跃度"
- "评估XX仓库贡献者集中度"
- "查看科研仓库代码统计"
- "拉取仓库贡献者数据并生成报告"
- 携带 `owner/repo` 仓库路径并要求做活跃度评估

## 输入提取规则
从用户语句中拆分仓库标识，格式固定为 `owner/repo`：
示例：`NSCCN/AlphaFold3`
拆分结果：owner = NSCCN, repo = AlphaFold3
未识别到合法 `owner/repo` 格式时，提示用户补充完整仓库地址。

## 平台与鉴权
本 Skill 默认针对 GitLink 平台，Agent 执行时直接使用 `gitlink-cli`，无需检测其他后端。
鉴权检查：若 gitlink-cli 命令返回 “authentication required” 或类似鉴权错误，则提示用户配置 `GITLINK_TOKEN` 环境变量后重试；否则继续执行。

## 执行流程
1. 从用户输入提取 owner 和 repo（如 “NSCCN/AlphaFold3” 拆分为 owner=NSCCN, repo=AlphaFold3）。
2. 依次执行以下命令（agent 需模拟 shell 执行，在无法真实执行的环境中可要求用户提供命令输出）：
   ```bash
   gitlink-cli repo +code-stats --owner <owner> --repo <repo> --format json
   gitlink-cli repo +contributors --owner <owner> --repo <repo> --format json
   ```
3. 解析返回的 JSON，提取 additions, deletions, commit_count, author_count, authors 数组。
4. 若解析失败或关键字段缺失，终止分析并返回错误信息。
5. 按照下方输出模板生成科研项目洞察报告。

## 输出模板
分析完成后，必须严格按以下 Markdown 格式输出，不得省略任何小节：

### 项目总览
仓库全名：[full_name 或用户提供的仓库标识]
总新增代码行数：[additions] 行
总删除代码行数：[deletions] 行
总提交次数：[commit_count]
贡献者总数：[author_count]

### 核心贡献者 Top3
| 排名 | 贡献者 | 提交次数 | 占总提交比例 |
|------|--------|----------|------------|
| 1 | [name] | [commits] | [百分比]% | 
| 2 | [name] | [commits] | [百分比]% |
| 3 | [name] | [commits] | [百分比]% |

### 活跃度评估
等级：[高/中/低]
高：commit_count > 500
中：100 ≤ commit_count ≤ 500
低：commit_count < 100

### 科研协作建议
若 Top1 提交占比 > 80%：
项目高度集中于单一开发者，代码质量可能较高，但外部贡献门槛高。建议作为算法学习与复现的参考，参与时优先通过 Issue 讨论。

若贡献者分布较均匀：
社区开放度较高，适合研究者参与代码贡献和多团队协作。

## 使用示例
用户：分析 NSCCN/AlphaFold3 的科研活跃度

Agent 行为：
提取 owner=NSCCN, repo=AlphaFold3
尝试执行 gitlink-cli 命令（若无执行环境，则提示用户提供数据，或在用户提供 JSON 后继续）
假设获得以下数据（模拟）：
additions: 83809, deletions: 16005, commit_count: 244, author_count: 14
top authors: Augustin Zidek (206 commits), Josh Abramson (11), James Spencer (9)

生成报告：
### 项目总览
仓库全名：NSCCN/AlphaFold3
总新增代码行数：83809 行
总删除代码行数：16005 行
总提交次数：244
贡献者总数：14

### 核心贡献者 Top3
| 排名 | 贡献者 | 提交次数 | 占总提交比例 |
|-----|---------|---------|------------|
| 1 | Augustin Zidek | 206 | 84.4% |
| 2 | Josh Abramson | 11 | 4.5% |
| 3 | James Spencer | 9 | 3.7% |

### 活跃度评估
等级：中等

### 科研协作建议
该项目高度集中于 Augustin Zidek（84.4% 提交），适合作为核心算法参考，建议在 GitLink 源仓库参与讨论或提交 Issue。
