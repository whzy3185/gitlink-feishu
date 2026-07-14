---
name: gitlink-compliance
version: 1.0.0
description: "开源合规检查：扫描仓库许可证、版权声明、依赖合规性，生成合规报告与修复建议。当用户需要检查项目合规状态、许可证兼容性或准备开源发布时触发。"
metadata:
  requires:
    bins: ["gitlink-cli"]
  cliHelp: "gitlink-cli repo --help"
  scenario: "S3"
---

# gitlink-compliance（开源合规与复现性检查）

**CRITICAL — 开始前必须先阅读 [`../gitlink-shared/SKILL.md`](../gitlink-shared/SKILL.md)，其中包含认证、权限处理和 API 注意事项。**
**CRITICAL — GitLink 操作只能用 `gitlink-cli`。禁止用 `gh`（GitHub CLI）操作 GitLink 资源。`gh` 仅适用于 GitHub 平台。**

> **前置条件：** 先阅读 [`../gitlink-shared/SKILL.md`](../gitlink-shared/SKILL.md) 了解认证和全局参数。

子赛题四「应用 GitLink 辅助科研」· 场景 **S3 科研项目合规与复现性检查** 的自动化算法由
`scripts/research/repro.py`（Go 出数据 + Python 做算法）实现，见 **工作流 4**。

---

## 何时使用

- 科研项目准备开源发布前，做一次全面的合规与复现性审查。
- 想知道「这个仓库别人能不能复现」：CI、lockfile、README 复现说明、版本 tag、容器环境是否齐备。
- 想发现仓库里的合规风险与敏感信息泄露：缺 LICENSE / 版权头、数据目录入库、`.env` 泄露、硬编码密钥。

## 前置条件

1. 已 `gitlink-cli auth login`（Token 7 天有效）。
2. 目标仓库存在且可读取文件树（`repo +info` / `file +get` / `repo +tree`）。
3. 复现性自动化检查（工作流 4）只用 Python 标准库，无需第三方依赖。

---

## 工作流概览

本 Skill 提供开源项目的合规性与复现性自动化检查能力，帮助 Maintainer 在发布前发现并修复合规问题。

| 检查类型 | 覆盖范围 | 严重程度 |
|----------|---------|:--------:|
| 许可证检查 | LICENSE 文件存在性、许可证类型识别 | 🔴 / 🟡 |
| 版权声明 | 源文件头部版权注释 | 🟡 |
| 依赖合规 | 第三方依赖许可证兼容性 | 🔴 |
| 安全策略 | SECURITY.md、安全披露流程 | 🟡 |
| 贡献者协议 | CLA / DCO 要求 | 🔵 |
| 复现性 | CI / lockfile / README 复现说明 / 版本 tag / 容器环境 | 🟡 |
| 数据隐私 | data/ 入库、.env 泄露、密钥硬编码 | 🔴 |

---

## 工作流 1：完整合规检查

**场景**：项目准备开源发布前，进行全面的合规性审查。

### 采集数据

```bash
# 1. 获取仓库文件结构
gitlink-cli repo +tree --owner <owner> --repo <repo> --ref master --format json

# 2. 读取 LICENSE 文件
gitlink-cli repo +raw --owner <owner> --repo <repo> --path LICENSE --ref master --format json

# 3. 检查关键文档是否存在
# 检查以下文件是否存在：
# - LICENSE / LICENSE.txt / LICENSE.md
# - CONTRIBUTING / CONTRIBUTING.md
# - SECURITY / SECURITY.md
# - CODE_OF_CONDUCT / CODE_OF_CONDUCT.md
# - .gitignore
# - README.md

# 4. 获取依赖配置
gitlink-cli repo +manifest --owner <owner> --repo <repo> --kind node --ref master --format json
gitlink-cli repo +manifest --owner <owner> --repo <repo> --kind go --ref master --format json
gitlink-cli repo +manifest --owner <owner> --repo <repo> --kind python --ref master --format json
gitlink-cli repo +manifest --owner <owner> --repo <repo> --kind rust --ref master --format json
gitlink-cli repo +manifest --owner <owner> --repo <repo> --kind java --ref master --format json

# 5. 获取源文件检查（按语言采样）
gitlink-cli repo +tree --owner <owner> --repo <repo> --path src --ref master --format json

# 6. 获取仓库基本信息
gitlink-cli repo +info --owner <owner> --repo <repo> --format json
```

### 检查清单

#### 🔴 必须修复项

| 检查项 | 标准 | 判定方法 |
|--------|------|----------|
| LICENSE 文件 | 根目录存在 LICENSE 文件且内容有效 | 检查文件是否存在、内容是否为空 |
| 许可证类型 | 使用 OSI 批准的开放源码许可证 | 解析 LICENSE 内容，识别许可证类型 |
| 许可证兼容性 | 项目许可证与依赖许可证兼容 | 检查依赖许可证，对比兼容性矩阵 |
| 安全漏洞 | 依赖无已知 CVE | 检查依赖版本（需外部数据源） |

#### 🟡 建议修复项

| 检查项 | 标准 | 判定方法 |
|--------|------|----------|
| 版权声明 | 源文件头部包含版权和许可证信息 | 采样检查源文件头部注释 |
| SECURITY.md | 存在安全策略披露流程 | 检查文件是否存在 |
| CONTRIBUTING.md | 存在贡献指南 | 检查文件是否存在 |
| 商标声明 | 正确使用项目/组织商标 | 检查 README 和 LICENSE 中的商标声明 |

#### 🔵 可选优化项

| 检查项 | 标准 | 判定方法 |
|--------|------|----------|
| CODE_OF_CONDUCT | 存在行为准则 | 检查文件是否存在 |
| .gitignore | 存在且配置完整 | 检查文件是否存在、内容是否覆盖常见模式 |
| README 质量 | README 包含安装、使用、贡献说明 | 检查内容长度和关键章节 |

### 输出格式

```markdown
## ⚖️ 合规检查报告 — <owner>/<repo>

📅 检查时间：<YYYY-MM-DD>
📋 项目许可证：<许可证类型>

### 🔴 必须修复

| # | 问题 | 文件 | 建议 |
|---|------|------|------|
| 1 | 缺少 LICENSE 文件 | — | 添加 LICENSE 文件，建议使用 <推荐许可证> |
| 2 | 依赖 xxx 使用 GPL 许可证，与项目 MIT 许可证不兼容 | package.json | 替换为兼容许可证的替代库 |
| 3 | 源文件缺少版权声明头 | src/*.py | 添加标准版权注释头 |

### 🟡 建议修复

| # | 问题 | 文件 | 建议 |
|---|------|------|------|
| 1 | 缺少 SECURITY.md | — | 添加安全漏洞披露流程文档 |
| 2 | CONTRIBUTING.md 不存在 | — | 添加贡献指南 |

### 🔵 可选优化

| # | 建议 | 说明 |
|---|------|------|
| 1 | 添加 CODE_OF_CONDUCT.md | 规范社区行为准则 |
| 2 | 完善 README 安装说明 | 当前缺少环境要求部分 |

### 📊 合规评分

| 维度 | 状态 | 评分 |
|------|:----:|:----:|
| 📜 许可证 | ✅ / ⚠️ / ❌ | ☆☆☆☆☆ |
| 🏷️ 版权声明 | ✅ / ⚠️ / ❌ | ☆☆☆☆☆ |
| 📦 依赖合规 | ✅ / ⚠️ / ❌ | ☆☆☆☆☆ |
| 🔒 安全策略 | ✅ / ⚠️ / ❌ | ☆☆☆☆☆ |
| 📖 项目文档 | ✅ / ⚠️ / ❌ | ☆☆☆☆☆ |

**总体合规评分：<分数>/100**
```

---

## 工作流 2：许可证兼容性检查

**场景**：检查项目使用的第三方依赖是否与项目许可证兼容。

```bash
# 1. 获取依赖配置文件
gitlink-cli repo +manifest --owner <owner> --repo <repo> --kind node --ref master --format json
```

### 许可证兼容性参考

| 项目许可证 | 兼容的依赖许可证 | 不兼容的依赖许可证 |
|-----------|-----------------|-------------------|
| MIT | MIT, Apache-2.0, BSD-2/3, Unlicense, ISC, CC0 | GPL-2/3, AGPL |
| Apache-2.0 | Apache-2.0, MIT, BSD-2/3, ISC, Unlicense | GPL-2/3 |
| GPL-3.0 | GPL-3.0, MIT, Apache-2.0, BSD | — |
| BSD-3 | MIT, BSD-2/3, Apache-2.0, ISC | GPL-2/3, AGPL |
| MulanPSL-2 | MulanPSL-2, MIT, Apache-2.0, BSD | GPL-3, AGPL (需确认) |

### 输出格式

```bash
# 依赖合规分析（示例输出结构）

## 📦 依赖合规分析

| 依赖 | 版本 | 许可证 | 兼容性 | 建议 |
|------|:----:|:------:|:------:|------|
| express | 4.18.2 | MIT | ✅ 兼容 | — |
| lodash | 4.17.21 | MIT | ✅ 兼容 | — |
| anticonflict-lib | 1.0.0 | GPL-3.0 | ❌ 不兼容 | 替换为兼容替代库 |
```

---

## 工作流 3：版权声明批量检查

**场景**：检查项目中所有源文件是否包含正确的版权声明头。

```bash
# 1. 遍历源文件目录
gitlink-cli repo +tree --owner <owner> --repo <repo> --path src --ref master --format json

# 2. 采样检查源文件头部（取前 5-10 行）
gitlink-cli repo +raw --owner <owner> --repo <repo> --path src/main.py --ref master --format json
```

### 标准版权声明模板

```python
# Copyright (c) <年份> <组织/作者>
# Licensed under the <许可证> License.
# See LICENSE file in the project root for full license information.
```

### 常见问题

| 问题 | 说明 |
|------|------|
| 缺少头部注释 | 源文件没有版权/许可证信息 |
| 年份过时 | 版权年份未更新到当前年份 |
| 许可证不匹配 | 声明的许可证与实际 LICENSE 文件不一致 |
| 组织名称错误 | 版权声明的组织名称与项目所属不一致 |

---

## 工作流 4：合规与复现性自动化检查（repro.py）

**场景**：子赛题四·S3 科研项目合规与复现性检查 —— 对一个科研仓库同时给出「合规分」与「复现分」，并产出检查清单、风险项与中文报告。

本工作流的算法由 `scripts/research/repro.py` 实现，数据全部经 gitlink-cli 获取（Go 出数据 + Python 做算法）。

### 数据采集

`repro.py` 内部调用以下 gitlink-cli 命令（已封装在 `collect.py` 中）：

```bash
# 仓库信息（默认分支、版本 tag）
gitlink-cli --owner <owner> --repo <repo> repo +info --format json

# 关键文件文本（LICENSE / README / go.mod / requirements.txt / package.json / .gitignore / SECURITY.md / ...）
gitlink-cli --owner <owner> --repo <repo> file +get --path LICENSE --ref master

# 根文件树（扫 data/、.env、config、.gitea/.github workflows 等是否存在）
gitlink-cli --owner <owner> --repo <repo> repo +tree --ref master

# 语言占比（仅作为元信息记录）
gitlink-cli --owner <owner> --repo <repo> repo +languages --format json
```

### 算法（纯函数，可单测）

| 函数 | 作用 |
|------|------|
| `identify_license(text)` | 关键词匹配 MulanPSL / Apache / MIT / GPL / LGPL / BSD / ISC / MPL / 无 |
| `scan_secrets(text, file)` | 正则找 private key / AWS token / API key / Slack / GitHub token / JWT / 邮箱 / 手机号 → `[{level,category,file,line,detail}]`（脱敏） |
| `repro_checks(file_texts, tree, repo_info)` | CI 配置、lockfile、README 复现说明、版本 tag、容器化，每项 `{name,pass,score(0-2),evidence}` |
| `compliance_items(license_info, file_texts, tree)` | LICENSE 声明、SECURITY.md、版权头、依赖合规、CONTRIBUTING.md |
| `data_privacy(tree, gitignore_text)` | data/ 入库、.env 入库、.gitignore 是否忽略 .env |

打分：`repro_score` / `compliance_score` 均为 0-10（各项 0-2 分聚合归一）。

### 命令

```bash
# 默认输出到 stdout（JSON）
python scripts/research/repro.py --owner mindspore-Ecosystem --repo mindspore

# 输出两件产物到目录（repro.json + compliance_report.md）
python scripts/research/repro.py --owner <OWNER> --repo <REPO> --out ./out

# 可复现脚本（封装了上述流程）
bash skills/gitlink-compliance/examples/compliance-repro-workflow.sh <OWNER> <REPO> [OUT_DIR]
```

### 输出结构（repro.json）

```json
{
  "scenario": "S3_compliance_reproducibility",
  "repo": "owner/repo",
  "default_branch": "master",
  "license": "MIT",
  "repro_items": [{"name": "CI 配置", "pass": true, "score": 2, "evidence": "..."}],
  "compliance_items": [{"name": "LICENSE 文件", "pass": true, "score": 2, "evidence": "..."}],
  "privacy_items": [{"name": ".env 入库", "pass": true, "score": 2, "evidence": "..."}],
  "secrets": [{"level": "critical", "category": "private_key", "file": "config.env", "line": 5, "detail": "..."}],
  "risks": [{"area": "secret", "name": "private_key", "file": "...", "level": "critical", "evidence": "..."}],
  "repro_score": 8.0,
  "compliance_score": 6.0,
  "meta": {"key_files_found": ["LICENSE", "README.md"], "tree_size": 42, "languages": {"Python": "99%"}}
}
```

`compliance_report.md` 包含：复现性检查清单表、合规性检查清单表、数据隐私检查表、风险项表（按严重程度排序）与打分。

---

## Raw API 参考

```bash
# 获取文件内容
gitlink-cli repo +raw --owner <owner> --repo <repo> --path <path> --ref <branch> --format json

# 获取文件列表（遍历目录）
gitlink-cli repo +tree --owner <owner> --repo <repo> --path <path> --ref <branch> --format json

# 获取仓库信息
gitlink-cli api GET /:owner/:repo --format json

# 获取贡献者列表
gitlink-cli api GET /:owner/:repo/contributors --format json
```

## 注意事项

- 合规检查结果仅基于仓库中可读取的文件，不构成法律建议
- 对于许可证兼容性问题，建议在实际发布前咨询法务或开源办公室（OSPO）
- 不同语言的依赖管理文件格式不同，需要根据项目主语言选择对应的依赖文件分析
- 版权声明检查是采样性的，100% 覆盖需要运行专门的扫描工具
- MulanPSL-2（木兰许可证）是 GitLink 平台上常用的许可证，需注意其与 GPL 的兼容性
