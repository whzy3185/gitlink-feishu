# pr +checkout

将指定 PR 的源分支拉取到本地，并切换到一个本地 review 分支。

```bash
gitlink-cli pr +checkout --id 3
gitlink-cli pr +checkout -i 3 --branch review/pr-3
gitlink-cli pr +checkout -i 3 --force
gitlink-cli pr +checkout -i 3 --dry-run
```

## 参数

| 参数 | 必需 | 说明 |
|------|------|------|
| `--id` / `-i` | 是 | PR 序号，即网页 URL `/pulls/N` 中的数字 |
| `--branch` / `-b` | 否 | 本地分支名，默认 `pr-<id>` |
| `--force` / `-f` | 否 | 本地分支已存在时使用 `git checkout -B` 重置 |
| `--dry-run` | 否 | 只输出将执行的 git 命令，不修改本地仓库 |

## 行为

命令会先读取 PR 详情，解析源 fork、源仓库和源分支，然后执行：

```bash
git fetch --no-tags <source-url> <source-branch>
git checkout -b <local-branch> FETCH_HEAD
```

如果传入 `--force`，第二步会改为 `git checkout -B <local-branch> FETCH_HEAD`。

## 使用建议

- 审查 PR 前可以先执行 `pr +view` 和 `pr +files` 获取上下文，再用 `pr +checkout` 拉到本地跑测试。
- fork PR 不需要手动添加 contributor remote，命令会从 PR 详情里解析源仓库地址。
- 不确定会执行哪些本地 git 操作时，先加 `--dry-run`。
