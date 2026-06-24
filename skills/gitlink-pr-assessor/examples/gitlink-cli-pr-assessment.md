# gitlink-cli PR 队列评估示例

这个示例展示如何用 `gitlink-pr-assessor` 扫描 `gitlink-cli` 仓库里的 open PR，并筛出“还没有形成维护者结论”的 PR，给每条 PR 生成一份报告。

## 示例目标

维护者希望每天或每隔一段时间得到一份待审队列报告，帮助回答：

- 哪些 open PR 还没有被真正处理
- 哪些 PR 值得优先投入人工评审
- 哪些 PR 只是证据不足，需要作者先补测试或补说明

## 推荐流程

### 1. 拉 open PR 列表

```bash
gitlink-cli pr +list --owner Gitlink --repo gitlink-cli --state open --page 1 --limit 50 --format json
```

注意：

- 不要只信 `--state open`
- 必须再按 `pull_request_status == 0` 过滤

### 2. 对每条 PR 判断是否进入待评估队列

```bash
gitlink-cli pr +view --owner Gitlink --repo gitlink-cli --id <pr_number> --format json
gitlink-cli pr +reviews --owner Gitlink --repo gitlink-cli --id <pr_number> --format json
```

建议跳过：

- 已有 `approved`
- 已有 `rejected`
- 已经带有 assessor 报告标记 `<!-- gitlink-pr-assessor:report v1 -->`

### 3. 对纳入队列的 PR 拉上下文

```bash
gitlink-cli pr +files --id <pr_number> --format json
gitlink-cli pr +diff --id <pr_number> --format json
gitlink-cli api GET /Gitlink/gitlink-cli/pulls/<pr_number>/commits --format json
gitlink-cli repo +info --owner Gitlink --repo gitlink-cli --format json
gitlink-cli ci +builds --owner Gitlink --repo gitlink-cli --format json
```

### 4. 提炼待验证声明

例如 PR 描述写了：

- 新增某个 shortcut
- 修复某个复杂边界输入问题
- 补充帮助文档和测试

那么可以整理出声明清单：

1. 命令或参数是否真的存在。
2. 行为是否与 PR 描述一致。
3. 测试是否覆盖关键边界值。
4. 构建和相关模块测试是否通过。

### 5. 按 `gitlink-cli` 的仓库习惯做执行验证

对这个仓库，通常优先验证：

```bash
go build ./...
go test ./...
```

如果 PR 只影响局部模块，再补局部测试，例如：

```bash
go test ./shortcuts/pr/...
go test ./shortcuts/attachment/...
go test ./shortcuts/workflow/...
```

### 6. 输出两层结果

#### 队列总览

给维护者快速看：

- 今天扫描到多少 open PR
- 其中多少条需要优先看
- 哪些被跳过，为什么跳过

#### 单条 PR 报告

给维护者细看：

- 这条 PR 值不值得继续投精力
- 当前最关键的风险是什么
- 作者下一步该补什么
- 维护者适合给出什么 review 建议

## 一个典型结论示例

如果某条 PR 的情况是：

- 功能方向合理
- 局部测试通过
- 全量构建通过
- 但复杂边界输入没有测试

那么更合适的报告结论通常是：

`建议补测试或补验证说明后再进入人工评审`

如果某条 PR 的情况是：

- 涉及鉴权、路径、编码或发布逻辑
- 影响面大
- 证据还不够完整

那么应提高优先级，并明确写出：

- `risk_level: high`
- `priority: P1`
- 需要哪类维护者继续接手，例如 CLI、安全或架构方向
