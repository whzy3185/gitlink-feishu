---
name: gitlink-research-data-provenance
description: "科研数据来源与隐私审计：检查 GitLink 科研仓库中的数据集说明、下载来源、引用链、隐私风险、数据许可证和大文件痕迹。用于论文代码发布前的数据治理、复现实验数据说明和科研合规预审。"
---

# gitlink-research-data-provenance

开始前必须先阅读 `../gitlink-shared/SKILL.md`，确认认证、权限、安全规则和 GitLink API 注意事项。

## 安全规则

- 只读执行，不下载大文件、不打开疑似敏感数据内容。
- 所有 GitLink 操作必须使用 `gitlink-cli`。
- 所有命令使用 `--format json`。
- 不输出 Token、Cookie 或认证 Header。
- 不输出个人隐私、样本原文、密钥或数据内容；只报告路径、类型和风险。
- 报告必须注明“科研数据治理预审，不构成法律意见或伦理审查结论”。

## 工作流

1. 采集仓库概览和默认分支：

```bash
gitlink-cli repo +info --owner <owner> --repo <repo> --format json
```

2. 采集根目录文件树：

```bash
gitlink-cli repo +tree --owner <owner> --repo <repo> --ref <branch> --format json
```

如果 `repo +tree` 不可用，回退：

```bash
gitlink-cli api GET /<owner>/<repo>/sub_entries --query 'filepath=&ref=<branch>' --format json
```

3. 如果根目录存在 `data`、`datasets`、`benchmark`、`experiments`、`notebooks`、`docs`、`README`、`paper`、`assets`，只采集一层目录结构：

```bash
gitlink-cli repo +tree --owner <owner> --repo <repo> --path data --ref <branch> --format json
gitlink-cli repo +tree --owner <owner> --repo <repo> --path docs --ref <branch> --format json
```

4. 按 `references/provenance-rules.md` 生成数据来源和隐私风险报告。

## 输出格式

```markdown
# 科研数据来源与隐私审计：<owner>/<repo>

## 数据治理总览

| 检查项 | 结果 | 风险 | 证据 |
|---|---|---|---|

## 数据来源链

| 数据/目录 | 来源说明 | 许可证/引用 | 复现状态 |
|---|---|---|---|

## 风险与整改

1. ...

## 说明

本报告是科研数据治理预审，不构成法律意见或伦理审查结论。
```

## 分析原则

- 有数据目录但无来源说明时，风险高于“未发现数据目录”。
- 发现 `.csv`、`.jsonl`、`.parquet`、`.h5`、`.npy`、`.zip`、`.tar` 等文件名时，只报告文件名和路径，不打印内容。
- 发现 `patient`、`student`、`email`、`phone`、`idcard`、`address`、`face`、`medical` 等词时，标注隐私复核风险。
- 如果仓库只包含数据下载脚本，检查脚本是否说明下载地址、数据许可证和引用方式。
- 无法确认的事项标注“未确认”，不要编造数据来源。

