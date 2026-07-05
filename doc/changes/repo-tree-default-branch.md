# repo +tree 遵循仓库默认分支

## 背景

`repo +tree` 未指定 `--ref` 时把 ref 硬编码为 `master`，对默认分支是
`main`（或其他名称）的仓库直接返回 `[-2] 你访问的文件不存在`
（生产实测 `datawhalechina/paper-chart-tutorial`，默认分支 `main`）。
平台 `sub_entries` API 在不带 `ref` 参数时会自动使用仓库默认分支。

## 变更

- `--ref` 未指定时不再发送 `ref` 参数（交由平台落到默认分支），flag 也不再
  声明 `master` 默认值。

## 验证

- `go test ./shortcuts/repo/`（默认 ref 断言改为「不携带 ref 参数」）
- 生产 gitlink.org.cn 实测：默认分支为 `main` 与 `master` 的仓库均正常列出根目录。

## 追加：branch/pr/release 同类问题一并修复

- 新增 `RuntimeContext.DefaultBranch()`：读取仓库详情的 `default_branch`（缺失时回退 master）
- `branch +create --from` / `pr +create --base` 未指定时回退到仓库默认分支（不再硬编码 master）
- `release +create --target` 未指定时省略 `target_commitish`（平台自动落默认分支）
- 生产实测（默认分支为 main 的仓库）：`repo +tree` / `branch +create` / `release +create` 均成功
