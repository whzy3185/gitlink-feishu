---
name: gitlink-insight
version: 1.0.0
description: "项目健康度与协作洞察：分析 Issue/PR 指标、贡献者活跃度、生成项目周报和健康度报告。当用户需要了解项目进展、团队协作状况或生成报告时触发。"
metadata:
  requires:
    bins: ["gitlink-cli"]
  cliHelp: "gitlink-cli issue --help"
---

# gitlink-insight（项目健康度与协作洞察）

**CRITICAL — 开始前必须先阅读 [`../gitlink-shared/SKILL.md`](../gitlink-shared/SKILL.md)，其中包含认证、权限处理和 API 注意事项。**
**CRITICAL — GitLink 操作只能用 `gitlink-cli`。禁止用 `gh`（GitHub CLI）操作 GitLink 资源。`gh` 仅适用于 GitHub 平台。**

> **前置条件：** 先阅读 [`../gitlink-shared/SKILL.md`](../gitlink-shared/SKILL.md) 了解认证和全局参数。

---

## 工作流概览

本 Skill 通过聚合 GitLink API 数据，生成多维度的项目协作洞察报告。

| 报告类型 | 数据来源 | 适用场景 |
|----------|---------|----------|
| 项目健康度 | 仓库元数据 + 代码规范 + 测试 + 文档 | 项目 Maintainer 评估仓库状态 |
| Sprint 周报 | Issue + PR + Release | 团队每周同步 |
| 贡献者洞察 | 贡献者列表 + 活跃度 + PR 数据 | 管理者了解团队贡献分布 |
| Issue 分析 | Issue 列表 + 标签 + 状态 | 项目管理跟踪进度 |

---

## 工作流 1：项目健康度报告

**场景**：评估一个仓库的整体健康状况。

### 采集数据

```bash
# 1. 获取仓库基本信息
gitlink-cli repo +info --owner <owner> --repo <repo> --format json

# 2. 获取 Issue 统计
gitlink-cli issue +list --state open --format json
gitlink-cli issue +list --state closed --format json

# 3. 获取 PR 统计
gitlink-cli pr +list --state open --format json
gitlink-cli pr +list --state merged --format json

# 4. 获取仓库文件结构（检查文档、CI 配置）
gitlink-cli api GET /:owner/:repo/sub_entries --query 'filepath=&ref=master'

# 5. 获取语言统计
gitlink-cli api GET /:owner/:repo/languages --format json

# 6. 获取贡献者列表
gitlink-cli api GET /:owner/:repo/contributors --format json
```

### 分析指标

| 维度 | 指标 | 计算方法 | 权重 |
|------|------|----------|:----:|
| 📖 文档 | README 质量、CONTRIBUTING 存在性、CHANGELOG | 检查关键文档文件是否存在 | 15% |
| 📜 合规 | LICENSE 文件、版权声明 | 检查 LICENSE 文件 | 10% |
| 🔧 工程化 | CI 配置、linter 配置、.gitignore | 检查 CI/linter 配置文件 | 15% |
| 🧪 质量 | 测试文件比例、代码规范 | 测试文件占源码比例 | 20% |
| 🐛 协作 | Issue 响应时间、PR 合并率 | 分析 Issue/PR 时间序列 | 20% |
| 👥 社区 | 贡献者数量、Fork 数、Star 数 | 从仓库元数据获取 | 10% |
| 📦 依赖 | 依赖配置文件、依赖数量 | 检查 package.json / go.mod 等 | 10% |

### 输出格式

```markdown
## 🏥 项目健康度报告 — <owner>/<repo>

📅 报告时间：<YYYY-MM-DD>

### 总体评分：<⭐x5>（<分数>/100）

| 维度 | 评分 | 详情 |
|------|:----:|------|
| 📖 文档 | ☆☆☆☆☆ / 5 | <详情> |
| 📜 合规 | ☆☆☆☆☆ / 5 | <详情> |
| 🔧 工程化 | ☆☆☆☆☆ / 5 | <详情> |
| 🧪 质量 | ☆☆☆☆☆ / 5 | <详情> |
| 🐛 协作 | ☆☆☆☆☆ / 5 | <详情> |
| 👥 社区 | ☆☆☆☆☆ / 5 | <详情> |
| 📦 依赖 | ☆☆☆☆☆ / 5 | <详情> |

### 📊 关键指标

| 指标 | 数值 |
|------|:----:|
| 总 Issue（开放/关闭） | <n> / <n> |
| 总 PR（开放/已合并） | <n> / <n> |
| 贡献者 | <n> 人 |
| 主要语言 | <语言> |
| Star / Fork | <n> / <n> |

### ✅ 亮点
- <做得好的地方>

### ⚠️ 待改进
- <需要改进的地方>

### 🎯 建议行动
1. <优先级最高的改进建议>
2. <次要建议>
3. <可选的优化建议>
```

---

## 工作流 2：Sprint 周报

**场景**：每周自动生成团队协作周报。

```bash
# 1. 获取时间范围内关闭的 Issue
gitlink-cli issue +list --state closed --format json

# 2. 获取时间范围内合并的 PR
gitlink-cli pr +list --state merged --format json

# 3. 获取新的 Release
gitlink-cli release +list --format json

# 4. 获取开放中的 Issue（正在进行的任务）
gitlink-cli issue +list --state open --format json

# 5. 获取项目动态
gitlink-cli api GET /:owner/:repo/activity --format json
```

### 输出格式

```markdown
## 📅 Sprint 周报 — <owner>/<repo>

📆 周期：<YYYY-MM-DD> ~ <YYYY-MM-DD>

### ✅ 本周完成
| 类型 | 数量 | 详情 |
|------|:----:|------|
| Issue 关闭 | <n> | <关键 Issue 列表> |
| PR 合并 | <n> | <关键 PR 列表> |
| Release | <n> | <版本号列表> |

### 🚧 进行中
| Issue | 负责人 | 状态 |
|-------|:------:|:----:|
| <标题> | <assignee> | 进行中 / Review 中 / 阻塞 |

### 📊 统计
| 指标 | 本周 | 上周 | 环比 |
|------|:----:|:----:|:----:|
| 关闭 Issue | <n> | <n> | ±<n>% |
| 合并 PR | <n> | <n> | ±<n>% |
| 新增 Issue | <n> | <n> | ±<n>% |
| 新增贡献者 | <n> | <n> | ±<n> |

### ⚠️ 风险与阻塞
- <需要关注的问题>

---

## 工作流 3：贡献者洞察

**场景**：分析团队成员的贡献分布和活跃度。

```bash
# 1. 获取贡献者列表
gitlink-cli api GET /:owner/:repo/contributors --format json

# 2. 获取每个贡献者的 PR
# 通过 PR 列表按 author 过滤
gitlink-cli pr +list --state merged --format json

# 3. 获取用户信息
gitlink-cli user +info --login <username> --format json
```

### 输出格式

```markdown
## 👥 贡献者洞察 — <owner>/<repo>

| 贡献者 | 合并 PR | 关闭 Issue | 最近活跃 | 角色 |
|--------|:-------:|:----------:|:---------:|:----:|
| <user> | <n> | <n> | <日期> | Maintainer / Contributor |

### 活跃度分布
- 核心贡献者（本月 5+ PR）：<n> 人
- 活跃贡献者（本月 1-4 PR）：<n> 人
- 新增贡献者（本月首次贡献）：<n> 人

### 贡献趋势
- 本月 PR 合并数：<n>（环比 <±n%>）
- 本月 Issue 响应中位数：<n> 小时
- 本月 PR Review 中位数：<n> 小时
```

---

## 工作流 4：Issue 积压分析

**场景**：分析未处理的 Issue 积压情况，帮助排期。

```bash
# 1. 获取所有开放 Issue
gitlink-cli issue +list --state open --format json

# 2. 按标签分析分布
# 遍历每个 Issue，统计标签分布

# 3. 获取最老的 Issue（积压时间最长）
# 按 created_at 排序，取前 10
```

### 输出格式

```markdown
## 🐛 Issue 积压分析 — <owner>/<repo>

### 总览
- 开放 Issue：<n>
- 最老 Issue 已存在：<n> 天
- 平均存留时间：<n> 天

### 标签分布
| 标签 | 数量 | 占比 |
|------|:----:|:----:|
| bug | <n> | <n>% |
| enhancement | <n> | <n>% |
| question | <n> | <n>% |
| <其他> | <n> | <n>% |

### 需要关注的积压 Issue
1. ⏰ <标题>（已开放 <n> 天）— <标签>
2. ⏰ <标题>（已开放 <n> 天）— <标签>
3. ⏰ <标题>（已开放 <n> 天）— <标签>

### 建议
- 本周应优先处理的 Issue：<n> 个
- 可关闭的过期 Issue：<n> 个
```

---

## Raw API 参考

```bash
# 仓库信息
gitlink-cli api GET /:owner/:repo --format json

# 仓库语言统计
gitlink-cli api GET /:owner/:repo/languages --format json

# 贡献者列表
gitlink-cli api GET /:owner/:repo/contributors --format json

# 仓库动态
gitlink-cli api GET /:owner/:repo/activity --format json

# 文件列表（检查文档/配置完整性）
gitlink-cli api GET /:owner/:repo/sub_entries --query 'filepath=&ref=master'

# 获取用户信息
gitlink-cli api GET /users/:user_id --format json

# 用户贡献热力图
gitlink-cli api GET /users/:user_id/headmaps --format json
```

## 注意事项

- 所有报告输出为 **Markdown 格式**，可直接粘贴到 Issue、Wiki 或 PR 描述中
- 数据分析基于 API 返回的实时数据，不依赖本地缓存
- 对于大型仓库（100+ Issue/PR），使用分页参数 `page=1&limit=50` 逐页获取
- 趋势分析建议每周运行一次，形成历史对比基线
- `pr +list` 的 `--state` 参数可能不精确过滤——需客户端通过 `pull_request_status` 字段判断：0=open, 1=merged, 2=closed
