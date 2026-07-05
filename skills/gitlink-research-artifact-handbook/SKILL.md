---
name: gitlink-research-artifact-handbook
description: "科研成果沉淀手册生成：基于 GitLink 仓库 README、目录结构、Release、Issue、PR、贡献者和语言信息，生成面向答辩、开源发布或课题组交接的科研成果手册。用于梳理代码、实验、论文、数据、贡献和复现入口。"
---

# gitlink-research-artifact-handbook

开始前必须先阅读 `../gitlink-shared/SKILL.md`，确认认证、权限、安全规则和 GitLink API 注意事项。

## 安全规则

- 默认只读执行，不创建 Wiki、不发布 Release、不提交文件。
- 所有 GitLink 操作必须使用 `gitlink-cli`。
- 所有命令使用 `--format json`。
- 不输出 Token、Cookie 或认证 Header。
- 如果用户要求把手册写回 Wiki、Issue 或仓库文件，必须先输出 dry-run 草稿并等待确认。

## 工作流

1. 采集仓库概览、语言和贡献者：

```bash
gitlink-cli repo +info --owner <owner> --repo <repo> --format json
gitlink-cli repo +languages --owner <owner> --repo <repo> --format json
gitlink-cli repo +contributors --owner <owner> --repo <repo> --format json
```

2. 采集根目录和关键目录：

```bash
gitlink-cli repo +tree --owner <owner> --repo <repo> --ref <branch> --format json
gitlink-cli repo +readme --owner <owner> --repo <repo> --ref <branch> --format json
```

如果 `repo +tree` 不可用，回退：

```bash
gitlink-cli api GET /<owner>/<repo>/sub_entries --query 'filepath=&ref=<branch>' --format json
```

3. 采集协作和发布信息：

```bash
gitlink-cli issue +list --owner <owner> --repo <repo> --state open --format json
gitlink-cli pr +list --owner <owner> --repo <repo> --state merged --format json
gitlink-cli release +list --owner <owner> --repo <repo> --format json
```

4. 按 `references/handbook-template.md` 生成科研成果沉淀手册。

## 输出格式

```markdown
# 科研成果沉淀手册：<owner>/<repo>

## 1. 项目概览
## 2. 研究问题与成果定位
## 3. 代码与目录导览
## 4. 实验和数据入口
## 5. 复现步骤
## 6. 协作与贡献记录
## 7. 发布与引用信息
## 8. 交接清单
```

## 分析原则

- 手册是“可交接材料”，不是营销稿；优先列入口、证据和缺失项。
- 不要编造论文标题、数据来源或实验结果；缺失时写“待补充”。
- 对 Issue/PR 只做摘要，不贴完整讨论。
- 如果仓库没有 Release，建议使用 Release 固化阶段成果。
- 如果根目录 README 已完整，保留其结构并补交接清单；如果 README 很弱，按模板重建手册。

