---
name: gitlink-research-reproducibility
description: "科研仓库复现性审计：检查 GitLink 科研、论文复现、实验代码仓库是否具备 README、LICENSE、依赖清单、测试入口、CI、示例和数据/实验说明。用于生成复现性评分、缺失项和整改建议。"
---

# gitlink-research-reproducibility

开始前必须先阅读 `../gitlink-shared/SKILL.md`，确认认证、权限、安全规则和 GitLink API 注意事项。

## 安全规则

- 只读执行，不创建 Issue、不评论、不修改仓库。
- 所有 GitLink 操作必须使用 `gitlink-cli`。
- 所有命令使用 `--format json`。
- 不输出 Token、Cookie 或认证 Header。
- 如果用户要求回写整改清单，先输出 dry-run 文本并等待确认。

## 工作流

1. 确认目标仓库：

```bash
gitlink-cli repo +info --owner <owner> --repo <repo> --format json
```

2. 识别默认分支。优先使用 `repo +info` 返回的 `default_branch` 或 `default_branch_name`。没有字段时再询问用户，或使用 `master` 作为 GitLink 常见默认值。

3. 采集根目录文件树：

```bash
gitlink-cli repo +tree --owner <owner> --repo <repo> --ref <branch> --format json
```

如果当前 CLI 版本没有 `repo +tree`，回退 Raw API：

```bash
gitlink-cli api GET /<owner>/<repo>/sub_entries --query 'filepath=&ref=<branch>' --format json
```

4. 如根目录存在 `docs`、`examples`、`data`、`experiments`、`notebooks`、`tests`、`.github`、`.gitlink`，按需采集一层子目录用于证据补强：

```bash
gitlink-cli repo +tree --owner <owner> --repo <repo> --path docs --ref <branch> --format json
```

5. 按 `references/checklist.md` 的评分表生成报告。没有读取子目录证据时，报告必须标注“仅基于根目录文件树”。

## 输出格式

```markdown
# 科研复现性审计：<owner>/<repo>

## 总分

<score>/100

## 检查明细

| 项目 | 分值 | 结果 | 证据 | 建议 |
|---|---:|---|---|---|
| README | 20 | 通过/缺失 | root_entries | ... |

## 优先整改

1. ...
2. ...
3. ...

## 数据来源

- `repo +info`
- `repo +tree` 或 `api GET /sub_entries`
```

## 判定规则

- README：`README.md`、`README.zh-CN.md`、`readme`。
- LICENSE：`LICENSE`、`LICENSE.md`、`LICENSE.txt`。
- 依赖清单：`requirements.txt`、`pyproject.toml`、`environment.yml`、`package.json`、`go.mod`、`Cargo.toml`、`pom.xml` 等。
- 测试入口：`tests`、`test`、`__tests__`。
- CI 配置：`.github`、`.gitlink`、`.gitlab-ci.yml`、`Jenkinsfile`、`.circleci`。
- 示例或文档：`docs`、`examples`、`demo`。
- 数据或实验说明：`data`、`datasets`、`experiments`、`notebooks`。

## 注意事项

- 复现性评分是工程辅助判断，不等同于论文质量评价。
- GitLink 平台 API 可能返回字段差异，缺失字段时用“未知”而不是编造。
- 如果仓库是镜像仓库，需标注 GitLink 内协作数据可能不代表原平台活跃度。

