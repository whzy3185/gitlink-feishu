# gitlink-pr-integrator 参考命令

## 1. 命令入口选择

优先级如下：

1. 当前仓库源码构建出的 CLI
2. `go run .`
3. 全局安装的 `gitlink-cli.cmd`

在 Windows PowerShell 中，如果 `gitlink-cli` 被执行策略拦截，改用：

```powershell
& "$env:APPDATA\npm\gitlink-cli.cmd" auth status
```

如果需要确保使用的是当前仓库源码能力，直接在仓库根目录执行：

```powershell
go run . pr --help
go run . pr +list --owner Gitlink --repo gitlink-cli --state open --format json
```

## 2. 采集 PR 元信息

```bash
gitlink-cli pr +view --owner <owner> --repo <repo> --id <pr_number> --format json
gitlink-cli pr +files --owner <owner> --repo <repo> --id <pr_number> --format json
gitlink-cli pr +diff --owner <owner> --repo <repo> --id <pr_number> --format json
gitlink-cli pr +reviews --owner <owner> --repo <repo> --id <pr_number> --format json
gitlink-cli repo +info --owner <owner> --repo <repo> --format json
gitlink-cli ci +builds --owner <owner> --repo <repo> --format json
```

关注字段：

- `pull_request.base` / `pull_request.head`
- `pull_request_status`
- `pull_request_number`
- `issue.id`
- 变更文件路径、增删行、diff 片段
- review 状态：`common` / `approved` / `rejected`
- `default_branch`
- `open_devops`

## 3. 列出候选 open PR

```bash
gitlink-cli pr +list --owner <owner> --repo <repo> --state open --page 1 --limit 50 --format json
```

批量扫描时优先抓取：

- PR 号
- 标题
- 作者
- 更新时间
- base / head
- 状态

再对高风险或高优先级项补拉 `+view`、`+files`、`+reviews`。

## 4. 远端回写命令

只有用户明确要求时才使用：

```bash
gitlink-cli pr +comment --owner <owner> --repo <repo> --id <pr_number> --body "<markdown>"
gitlink-cli pr +review --owner <owner> --repo <repo> --id <pr_number> --status common --content "<markdown>" --dry-run
gitlink-cli pr +review --owner <owner> --repo <repo> --id <pr_number> --status common --content "<markdown>"
```

规则：

- 默认先 `--dry-run`
- 默认用 `common`
- 不默认 `approved`
- 不默认触发 `merge`

## 5. 独立 worktree 验证

推荐在仓库根目录执行：

```bash
git fetch origin <base_branch>
git worktree add <temp_dir> origin/<base_branch>
cd <temp_dir>
git switch -c pr-integration-check
```

如果 PR head 来自 fork：

```bash
git remote add pr-source <head_repo_url>
git fetch pr-source <head_branch>
git merge --no-ff --no-commit FETCH_HEAD
```

如果 PR head 来自同仓库：

```bash
git fetch origin <head_branch>
git merge --no-ff --no-commit origin/<head_branch>
```

注意 GitLink PR 的 `head` 常常已经是 `login/branch` 形式。对 fork PR，不要默认只取最后一段 branch 名。若 remote 名恰好也是这个 login，实际可用 ref 可能是 `refs/remotes/<remote>/<head>`，例如：

```bash
refs/remotes/mengz/mengz/api-single-call-templates
```

如果本地已经存在同名本地分支，也可以直接 merge 那个本地分支，但报告里要写清楚你实际使用的是哪个 ref。

记录四类结果：

1. 是否发生冲突
2. 是否需要 rebase
3. 官方构建是否通过
4. 官方测试是否通过

验证结束后，如果这个 worktree 只是一次性检查环境，及时清理：

```bash
git worktree remove <temp_dir>
```

不要在用户当前工作树清理或覆盖任何未提交改动。

## 6. 官方验证命令选择顺序

按以下顺序选命令：

1. PR 描述里作者写的验证步骤
2. 仓库 `README` / `CONTRIBUTING`
3. CI 配置或 `Makefile`
4. 语言惯例

常见命令：

```bash
go build ./...
go test ./...
npm test
pnpm test
pytest
cargo test
```

如果仓库没有明确写测试命令，不要假装“全部通过”；应标注“未找到项目定义的官方验证命令”。

## 7. 冲突雷达的最小比对法

没有必要把每个 open PR 都完整 clone 一遍。先用文件级和目录级比对做第一轮筛查：

- 同文件重叠：高风险
- 同目录或同模块重叠：中风险
- 同一 CLI 命令、flag、帮助文本或 API 包装层：至少中风险
- 文档或测试只轻微重叠：低到中风险，按实际耦合上调

只有当前两条 PR 都处于高优先级、而且重叠严重时，才进入更深的本地顺序合并验证。

## 8. 报告字段约定

推荐结构化字段：

```json
{
  "verdict": "ready_after_followups",
  "merge_readiness": "medium",
  "integration_risk": "medium",
  "conflict_risk": "high",
  "release_impact": "minor",
  "merge_validation": {
    "base_branch": "master",
    "merge_result": "clean",
    "build": "passed",
    "tests": "passed"
  },
  "conflicts": [
    {
      "pr": 123,
      "risk": "high",
      "reason": "same file overlap: shortcuts/pr/pr.go",
      "suggested_order": "merge #123 first"
    }
  ],
  "post_merge_actions": [
    "update README example",
    "add regression test for fork PR head parsing"
  ]
}
```

Markdown 报告和 JSON 结论保持一致，不要出现“结构化字段说可合并，正文却写暂缓”的相互矛盾。

## 9. 中文报告落盘

Windows PowerShell 中保存中文报告时显式指定 UTF-8：

```powershell
$report | Set-Content -Path .\pr-integration-report.md -Encoding utf8
```

如果输出里已经出现 `?`，先停下来修正编码链路，不要带着乱码继续演示或提交。
