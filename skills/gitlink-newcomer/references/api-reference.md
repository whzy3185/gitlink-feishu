# gitlink-newcomer API 参考

> **前置条件：** 先阅读 [`../../gitlink-shared/SKILL.md`](../../gitlink-shared/SKILL.md) 了解认证、全局参数和安全规则。

本技能识别 good-first-issue 所依赖的 GitLink 接口与字段。数据采集全程只读。

## 采集的接口

### Issue 列表

```
GET /:owner/:repo/issues.json?page={page}&limit={limit}
# 或经 gitlink-cli：
gitlink-cli issue +list --owner <owner> --repo <repo> --state open --format json
```

返回的 `issues[]` 中本技能使用的字段：

| 字段 | 说明 | 用途 |
|------|------|------|
| `id` | 全局数据库 ID | 去重、展示（**非** web 序号） |
| `name` / `subject` | Issue 标题 | 关键词分析、展示 |
| `description` | 正文 | 关键词分析、长度评估 |
| `issue_tags` / `labels` | 标签 | good-first-issue 标签信号 |
| `comment_journals_count` / `journals_count` | 评论数 | 讨论热度评估 |
| `author_login` / `author_name` | 作者 | 展示 |

### Issue 详情（单个）

```
GET /:owner/:repo/issues/{number}.json
# 或：
gitlink-cli issue +view --owner <owner> --repo <repo> --number {number} --format json
```

`{number}` 为 web 序号（URL 中的编号）。详情接口可拿到更完整的字段。

## ID 混淆（重要）

GitLink 存在两种 ID：

| 名称 | 来源 | 用途 |
|------|------|------|
| 全局数据库 `id` | Issue 列表接口 | 仅去重 / 内部引用 |
| web 序号（`project_issues_index` / `number`） | 单 Issue 详情、web URL | 发评论、PR 关联、`issue +comment --number` |

本技能的处理：
- 列表接口只返回全局 `id` 时，看板以 `id:xxx` 形式标注，**不**伪装成 web 序号；
- 生成引导评论时，若无可靠 web 序号则用 Issue 标题引用，避免误导；
- 发布评论用 `gitlink-cli issue +comment --number <web序号>`。

## 写操作

发布引导评论是写操作：

```
gitlink-cli issue +comment --owner <owner> --repo <repo> --number <web序号> -b "<引导文案>"
```

执行前必须征得用户确认；本技能默认只生成文案，不自动发布。

## 错误处理

沿用 gitlink-shared 的错误码（401 重新登录 / 403 权限 / 404 检查 owner/repo）。采集失败时脚本返回非零退出码并打印原因。
