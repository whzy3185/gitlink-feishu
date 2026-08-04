---
name: gitlink-research-trust
version: 1.0.0
metadata:
  requires:
    bins: ["gitlink-cli"]
  cliHelp: "gitlink-cli repo --help"
description: "科研开源可信度评估：面向论文代码、数据集、实验复现仓库，使用 gitlink-cli 采集仓库信息、目录树、README 和协作指标，生成复现性、可追踪性、协作健康、合规风险和文档可用性报告。当用户需要评估科研仓库可信度、复现准备度或开源治理风险时触发。"
---

# gitlink-research-trust（科研开源可信度评估）

## 目标

本 Skill 帮助 AI Agent 使用 `gitlink-cli` 对 GitLink 上的科研类开源仓库做只读可信度评估，回答：

- 这个科研仓库是否容易复现？
- 论文、数据、代码、实验之间是否可追踪？
- 协作活跃度和维护健康度如何？
- 许可证、敏感信息和镜像仓库风险是否清晰？
- README 是否足以支撑评审、复现和二次开发？

## 安全边界

- 默认只读，不向第三方仓库创建 Issue、评论、PR、Release 或 Wiki。
- 不输出真实 Token、Cookie、密码、私钥。
- 证据日志只记录命令、摘要和脱敏后的结构化结果。
- 对疑似敏感信息只给出文件路径、行号和脱敏片段，不复述完整密钥。

## 推荐采集命令

### 1. 搜索候选科研仓库

```bash
gitlink-cli search +repos -k "research" --format json
gitlink-cli search +repos -k "paper" --format json
gitlink-cli search +repos -k "dataset" --format json
gitlink-cli search +repos -k "科研" --format json
```

合并候选时优先选择命中多个关键词、近期更新、非空、公开、非纯镜像或镜像状态正常的仓库。

### 2. 获取仓库元数据

```bash
gitlink-cli repo +info --owner <owner> --repo <repo> --format json
gitlink-cli repo +contributors --owner <owner> --repo <repo> --format json
gitlink-cli repo +tree --owner <owner> --repo <repo> --ref <branch> --format json
gitlink-cli repo +readme --owner <owner> --repo <repo> --ref <branch> --format json
```

分支优先使用 `repo +info` 返回的 `default_branch`；不存在时再尝试 `master`、`main`。

### 3. 可选补充数据

```bash
gitlink-cli issue +list --owner <owner> --repo <repo> --format json
gitlink-cli pr +list --owner <owner> --repo <repo> --format json
gitlink-cli release +list --owner <owner> --repo <repo> --format json
```

若接口失败，不要终止整体评估；记录失败原因并降低对应置信度。

## 评分模型

总分 100：

| 维度 | 分值 | 核心信号 |
|---|---:|---|
| 复现性 | 25 | README、依赖声明、测试/CI、Release、近期活跃、最小运行命令 |
| 论文/数据/代码可追踪性 | 20 | DOI/arXiv/BibTeX、数据集说明、实验脚本、引用文件、示例 |
| 协作健康 | 20 | 贡献者、Issue/PR、关注/Fork、维护活跃度 |
| 合规与敏感信息 | 20 | 许可证、敏感信息扫描、依赖声明、安全/贡献文档、镜像风险 |
| 文档与可用性 | 15 | README 完整度、安装说明、运行命令、示例、社区文档 |

等级建议：

- A：85-100，可信度高，可作为复现或标杆仓库。
- B：70-84，主体可信，有少量治理或复现缺口。
- C：55-69，可参考但需要补足关键证据。
- D：0-54，缺口较大，不建议直接作为复现基线。

## 输出格式

建议生成以下文件：

```text
reports/<run_id>/
├── index.md
├── research_trust_report.md
├── summary.csv
├── graph.mmd
├── analysis.json
└── evidence.jsonl
```

`summary.csv` 字段建议：

```csv
repo,total_score,grade,reproducibility,traceability,collaboration,compliance,documentation,language,license,risks,recommendations
```

`evidence.jsonl` 每行建议包含：

```json
{"step":"repo-info:<owner>/<repo>","command":["gitlink-cli","repo","+info"],"ok":true,"summary":"ok","data":{"data_keys":["default_branch","issues_count"]}}
```

## 报告结构

```markdown
# GitLink 科研开源可信度报告：<topic>

## 一、结论总览

| 排名 | 仓库 | 总分 | 等级 | 复现性 | 可追踪性 | 协作 | 合规 | 文档 |

## 二、仓库深度分析

### <owner>/<repo>：<score>/100（<grade>）

- 主要优势
- 主要风险
- 优先整改建议

## 三、科研赋能价值

- 复现基线
- 论文/数据/代码证据链
- 开源治理整改

## 四、数据来源与限制

- GitLink CLI 命令列表
- 失败或跳过的接口
- 脱敏策略
```

## 触发示例

- “帮我评估 GitLink 上 AI 论文代码仓库的科研可信度。”
- “对这个 GitLink 仓库做复现性、许可证和敏感信息风险检查。”
- “搜索 5 个数据集相关仓库，输出科研可信度排行榜和 Mermaid 图谱。”
