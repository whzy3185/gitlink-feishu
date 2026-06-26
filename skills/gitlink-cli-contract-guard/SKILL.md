---
name: gitlink-cli-contract-guard
description: "CLI 契约守卫：审查 GitLink CLI 改动是否破坏既有命令契约，重点检查 flags 与默认值、命令层级与帮助文本、`--format json` 输出结构、错误提示与编码质量、README/示例命令和实际行为是否漂移。用于用户需要判断某个 PR 或本地改动会不会破坏旧用法、引入不兼容输出、造成帮助文档失真，或在合并前补做兼容性审查时。"
---

# gitlink-cli-contract-guard

**CRITICAL - 如果需要拉取 GitLink 上的 PR 元数据、diff 或评论，先阅读 [`../gitlink-shared/SKILL.md`](../gitlink-shared/SKILL.md)。**
**CRITICAL - 这个 skill 默认只读分析，不直接修改远端评论、标签或分配关系。**
**CRITICAL - 这个 skill 只关注 CLI 对用户承诺的行为契约，不负责判断 PR 是否应当合并。**

这个 skill 的目标很窄，也很硬：**找出会把现有 CLI 用户用法搞坏的改动。**

它重点审查五类契约面：

1. **参数契约**：flag 名称、短别名、默认值、必填规则、参数语义。
2. **帮助契约**：命令层级、`--help` 内容、国际化文案、示例命令。
3. **输出契约**：`--format json` 结构、字段名、字段类型、包裹 envelope。
4. **错误契约**：错误提示、退出语义、编码质量、用户可理解性。
5. **文档契约**：README、示例、帮助文本与真实行为是否一致。

## 不覆盖的内容

下面这些不属于这个 skill 的职责：

- PR 是否值得合并：交给 `gitlink-pr-assessor`
- PR 是否适合集成主线：交给 `gitlink-pr-integrator`
- commit message、分支命名、PR 模板质量：交给 `gitlink-commit-quality`
- 维护者今日值班优先级：交给 `gitlink-maintainer-radar`

## 工作流

### Step 1：确定分析对象

优先区分两种输入：

- **本地改动**：当前工作区已有变更或已 checkout 到目标分支，直接看 `git diff`、文件改动和本地测试。
- **远端 PR**：用户只给出 GitLink PR 编号，需要先通过 `gitlink-cli` 拉 PR 元数据和 diff，再结合本地代码理解。

如果是本地改动，优先看：

```bash
git diff --name-only
git diff --stat
```

如果是 GitLink PR，优先看：

```bash
gitlink-cli pr +view --owner <owner> --repo <repo> -i <number> --format json
gitlink-cli pr +version-diff --owner <owner> --repo <repo> -i <number> --format json
```

### Step 2：把改动映射到契约面

根据改动文件，先判断它可能影响哪一类契约。常见映射见 [`references/contract-surfaces.md`](references/contract-surfaces.md)。

重点关注这些高风险位置：

- `cmd/`：根命令、全局 flag、命令层级、帮助文本
- `shortcuts/*/*.go`：shortcut 参数、默认值、必填规则、输出行为
- `shortcuts/common/`：通用运行时、输出封装、参数解析
- `internal/client/`、`internal/auth/`：API client 行为、header、错误处理
- `internal/i18n/`：国际化文案、语言切换、编码风险
- `README.md`、`README.zh-CN.md`、`examples/`：文档和真实行为漂移

### Step 3：逐类检查契约是否被破坏

#### 3.1 参数契约

检查：

- 是否删除或重命名了已有 flag
- 是否变更了短别名
- 是否改了默认值但没有迁移说明
- 是否把原本可选参数改成必填
- 是否改变了 flag 含义但名字未变

对 shortcut 代码重点查看 `Name`、`Short`、`Default`、`Required`、`Bool`、`Usage`。

#### 3.2 帮助契约

检查：

- 命令层级是否变了
- `--help` 内容是否仍然描述真实行为
- 示例命令是否还可运行
- 中英文帮助文本是否同步
- 本地化 key 是否丢失或回退异常

如果触及 `cmd/root.go`、`shortcuts/register.go` 或 i18n 文案，优先检查 help 相关测试。

#### 3.3 输出契约

检查：

- `--format json` 是否仍然返回既有结构
- 字段名是否发生破坏性变更
- 字段类型是否变化
- envelope 是否还保持稳定
- 机器可读消费者依赖的路径是否变化

如果字段是新增但非破坏性变更，要明确说明是“扩展”而不是“破坏”。

#### 3.4 错误契约

检查：

- 错误消息是否退化为难以理解的技术细节
- 中文或多语言提示是否出现乱码
- `suggestFix`、校验错误、缺参错误是否还可读
- 渲染后的 header、path 模板或其他用户输入是否可能导致非法请求

尤其要把 **中文 mojibake、编码损坏、格式化后非法 header/path** 当成高风险契约问题。

#### 3.5 文档契约

检查：

- README 中的示例命令是否与当前实现一致
- 文档新增内容是否引入乱码
- 文档说支持的参数/输出，代码是否真的支持
- 代码新增能力后，帮助或 README 是否漏更新

### Step 4：要求验证证据

只指出风险还不够，要同时判断“有没有证据证明它没坏”。

常用验证方式：

```bash
go build ./...
go test ./cmd/... ./shortcuts/...
go test ./...
```

需要更聚焦时，优先跑与改动最相关的包测试。典型情况：

- 改 `cmd/` 或帮助/i18n：优先看 `cmd/root_test.go`
- 改 shortcut 参数或输出：优先看对应 `shortcuts/<name>/*_test.go`
- 改 API client 或错误处理：优先看 `internal/...` 和相关 shortcut 测试

如果改动了契约面，但没有补测试或现有测试没覆盖到，直接把它列为缺口。

### Step 5：按严重性归类

用 [`references/severity-rubric.md`](references/severity-rubric.md) 把问题分成：

- `blocking`：明确破坏旧用法或输出契约
- `high`：高概率影响真实用户或自动化脚本
- `medium`：存在漂移或边界缺口，但不一定立即破坏
- `low`：文案、可读性或一致性问题

### Step 6：输出契约审查结论

推荐输出结构：

```markdown
# CLI 契约审查报告

## 高风险问题
- `--header` 模板渲染后未再次校验，可能生成非法 header。
- README.zh-CN 新增示例出现中文乱码，会污染用户可见文档。

## 契约面影响
- 参数契约：`--header` 新增并改变请求构造行为。
- 输出契约：无破坏性字段变更证据。
- 错误契约：中文错误提示存在编码退化风险。

## 缺失验证
- 缺少对 `Accept` 头覆盖行为的边界测试。
- 缺少对渲染后非法 header 的测试。

## 结论
- 需要修改后再合并。
```

## 典型触发语句

- “帮我看这个改动会不会破坏现有 CLI 用法。”
- “检查这个 PR 有没有 flag / help / JSON 输出兼容性问题。”
- “看看这个命令改动会不会影响脚本调用方。”
- “帮我做一轮 CLI 行为契约审查。”
