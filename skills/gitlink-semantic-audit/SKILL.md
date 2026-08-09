---
name: gitlink-semantic-audit
version: 1.0.0
description: "CLI/平台语义审计：系统性比对 gitlink-cli 命令行为与 GitLink 生产 API 真实语义，发现伪成功、参数错配、端点失效等缺陷并产出审计报告。当用户需要验证 CLI 行为正确性、排查命令异常、或对新命令做上线前体检时触发。"
metadata:
  requires:
    bins: ["gitlink-cli"]
  cliHelp: "gitlink-cli api --help"
---

# gitlink-semantic-audit（CLI/平台语义审计）

**CRITICAL — 开始前必须先阅读 [`../gitlink-shared/SKILL.md`](../gitlink-shared/SKILL.md)，其中包含认证、权限处理和 API 注意事项。**
**CRITICAL — 审计过程只做读取与可控探针写入；任何写入探针必须在自己拥有的仓库中进行，结束后立即清理。**

> 本 Skill 把「生产反证法」沉淀为 Agent 可复用的工作流：CLI 的每个命令都隐含着对平台 API 的假设，
> 用 `gitlink-cli api` 直接访问生产端点获得 ground truth，再与 shortcut 输出比对，任何差异即缺陷。

## 适用场景

1. 排查"命令返回成功但平台上没生效"（伪成功）
2. 新增/修改 shortcut 后的上线前体检
3. 平台 API 升级后批量回归 CLI 命令语义
4. 给上游报 bug 前收集可复现证据链

## 审计工作流

### 第 1 步：确定审计对象与 ground truth 端点

```bash
# 列出目标命令组的全部 shortcut 与 flag
gitlink-cli <group> --help

# 用 api 命令直接访问同语义生产端点（认证自动注入）
gitlink-cli api GET /:owner/:repo/issues --query 'page=1&limit=5'
```

### 第 2 步：五类典型缺陷检查清单

| 缺陷类型 | 检查方法 | 判定标准 |
|----------|----------|----------|
| 伪成功 | 用不存在的资源调用命令 | 必须报错（4xx），返回 `ok:true` 或 HTML 即缺陷 |
| 参数错配 | 省略 CLI 标记为"必填"的字段直接调 API | API 接受则 CLI 过度约束；反之为欠约束 |
| 端点失效 | 对比 shortcut 使用的路径与平台当前路由 | 微服务下线/迁移（如旧版 CI builds 端点）即缺陷 |
| 分页丢数据 | 大列表下对比 `--all` 合并数与 API total_count | 数量不一致即缺陷 |
| ID 语义混淆 | 分别用网页编号与全局数据库 ID 调用 | 文档声明与实际接受的 ID 类型不一致即缺陷 |

### 第 3 步：可控探针验证（写路径）

```bash
# 在自己的仓库创建探针资源 → 用被审计命令操作 → 用 api 直接读取验证 → 清理
gitlink-cli issue +create --owner <me> --repo <probe-repo> --title "probe"
gitlink-cli issue +close --owner <me> --repo <probe-repo> --number <n>
gitlink-cli api GET /:owner/:repo/issues/<n>   # ground truth 确认状态
```

### 第 4 步：产出审计报告

对每个发现输出四元组，可直接作为上游 issue/PR 素材：

```
命令: <group> +<shortcut>
预期: <CLI 文档/help 声明的行为>
实际: <生产 API 真实行为，附原始响应>
复现: <最小 curl / gitlink-cli 命令序列>
```

## 使用示例

```bash
# 审计 release 组按 tag 引用是否真实生效
gitlink-cli release +view --owner <me> --repo <probe-repo> --tag v0.0.1
gitlink-cli api GET /:owner/:repo/releases   # 对照 ground truth

# 审计列表分页完整性
gitlink-cli pr +list --owner Gitlink --repo forgeplus --all --jq total_count
gitlink-cli api GET /:owner/:repo/pulls --query 'page=1&limit=1'

# 审计不存在路径是否被判伪成功
gitlink-cli api GET /:owner/:repo/nonexistent-endpoint
```

## 注意事项

- 免登录读取仅适用于公开仓库；权限相关缺陷需分别用无权限/有权限账号双重验证
- 平台存在 `/api` 与 `/api/v1` 两代端点，同名资源语义可能不同，审计时需分别取证
- 报告中的复现步骤必须最小化且可独立执行，不依赖审计者本地状态
