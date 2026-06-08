# 变更说明：新增 Shortcut 模块（wiki/commit/file/star/watch）+ 批量操作

## 概述

本 PR 新增 5 个 Shortcut 模块（共 23 个命令）和 3 个批量操作模块。

## 新增模块

### 1. Wiki 模块（5 个命令）

| 命令 | 说明 |
|------|------|
| `wiki +list` | 列出 Wiki 页面 |
| `wiki +view` | 查看 Wiki 页面内容 |
| `wiki +create` | 创建 Wiki 页面 |
| `wiki +update` | 更新 Wiki 页面 |
| `wiki +delete` | 删除 Wiki 页面（自动清理 Sidebar） |

**技术要点**：
- Wiki API 使用独立网关 `gateway.gitlink.org.cn`，不带 `.json` 后缀
- 使用 `client.DoRaw()` 方法处理非标准 API 响应
- delete 命令会自动清理 `_Sidebar` 中的残留链接

### 2. Commit 模块（4 个命令）

| 命令 | 说明 |
|------|------|
| `commit +list` | 提交历史列表 |
| `commit +view` | 查看提交详情 |
| `commit +diff` | 查看提交差异 |
| `commit +blame` | 代码追溯 |

### 3. File 模块（5 个命令）

| 命令 | 说明 |
|------|------|
| `file +list` | 列出目录文件 |
| `file +tree` | 文件树 |
| `file +get` | 获取文件内容 |
| `file +create` | 创建文件（自动 base64 编码） |
| `file +delete` | 删除文件 |

### 4. Star 模块（3 个命令）

| 命令 | 说明 |
|------|------|
| `star +star` | 点赞仓库 |
| `star +unstar` | 取消点赞 |
| `star +stars` | 查看点赞列表 |

### 5. Watch 模块（3 个命令）

| 命令 | 说明 |
|------|------|
| `watch +watch` | 关注仓库 |
| `watch +unwatch` | 取消关注 |
| `watch +watchers` | 查看关注者列表 |

## 新增批量操作

| 模块 | 命令 | 说明 |
|------|------|------|
| member | `batch-add` | 批量添加成员 |
| org | `batch-invite` | 批量邀请成员（自动解析用户名→ID） |
| repo | `batch-create` | 批量创建仓库 |
| repo | `batch-fork` | 批量 Fork 仓库 |
| repo | `batch-delete` | 批量删除仓库 |

所有批量操作支持 `--dry-run` 预览模式和 `--from CSV` 文件输入。

## 测试覆盖

| 模块 | 测试数 |
|------|--------|
| wiki | 12 |
| commit | 4+ |
| file | 5+ |
| star | 3+ |
| watch | 3+ |

## 注意事项

> ⚠️ 本 PR 的代码基于旧版 API 签名（`Shortcuts()` 无参数），需适配上游新版 i18n 翻译器接口（`Shortcuts(tr *i18n.Translator)`）后方可编译通过。
>
> Wiki 模块依赖 `client.DoRaw()` 方法（用于不带 `.json` 后缀的网关 API），需合入 client.go 的相关变更。
