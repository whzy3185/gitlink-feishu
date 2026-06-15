# 新需求构思报告：文档智能维护 Skill

**Skill 名称：** gitlink-docs-assistant  
**作者：** ZxR  
**日期：** 2026-06-15  

---

## 1. 背景与痛点

开源项目中存在普遍的"文档漂移"问题：代码迭代频繁，而项目文档（CONTRIBUTING、CHANGELOG、API 说明等）往往缺失或长期无人维护。具体表现为：

- 新贡献者找不到 CONTRIBUTING，不知道如何参与
- 没有 CHANGELOG，用户无法了解版本变更
- API 文档缺失，使用者只能读源码
- Maintainer 无法快速判断仓库文档是否完整

以 ylly/gitlink-cli 为例：实验一新增了 wiki、label、notification 三个模块共 13 个命令，但仓库 Wiki 中无对应文档，CONTRIBUTING 和 CHANGELOG 均缺失。

---

## 2. 需求定义

### 核心问题

> 如何让 AI Agent 自动扫描仓库文档状态，识别缺失项，并读取代码自动生成缺失文档写入 Wiki？

### 用户需求

| 角色 | 需求 |
|------|------|
| 项目 Maintainer | 一键获得"文档体检报告"，知道哪些文档缺失 |
| 开发者 | 新增功能后，AI 自动补全对应 Wiki 文档，无需手动写 |
| 新贡献者 | CONTRIBUTING 始终存在且有效，快速了解如何参与 |

---

## 3. 方案设计

### 设计原则

1. **先体检后补全**：只读模式生成报告，用户确认后再执行写入
2. **读代码生成文档**：AI 读取 README 和目录结构，生成符合项目实际的文档（非通用模板）
3. **复用实验一成果**：直接调用实验一开发的 `wiki +create/+update` shortcuts

### 复用命令

| 命令 | 来源 | 用途 |
|------|------|------|
| `gitlink-cli repo +info` | 原有 shortcut | 获取仓库基本信息 |
| `gitlink-cli api GET /:owner/:repo/sub_entries` | Raw API | 扫描根目录文件 |
| `gitlink-cli api GET /:owner/:repo/readme` | Raw API | 读取 README 作为文档素材 |
| `gitlink-cli wiki +list` | **实验一新增** | 列出现有 Wiki 页面 |
| `gitlink-cli wiki +view` | **实验一新增** | 读取页面内容 |
| `gitlink-cli wiki +create` | **实验一新增** | 创建缺失文档 |
| `gitlink-cli wiki +update` | **实验一新增** | 更新过时文档 |

### 原创性说明

与任务书中列出的场景对比：

| 已有场景 | gitlink-docs-assistant 的差异 |
|---------|------------------------------|
| 智能代码审查（PR diff → Review 评论） | 本 Skill 输出写入 Wiki，不是 PR 评论 |
| Release Notes 生成（commit → 版本说明） | 本 Skill 面向持续文档维护，不是一次性发布 |
| 项目健康度报告（统计指标） | 本 Skill 聚焦文档覆盖度并闭环修复，产生实际写入 |

**核心原创点：** 将"文档完整性体检"与"AI 自动生成文档写入 Wiki"串成完整闭环，复用实验一 wiki shortcuts，是现有 Skills 中未覆盖的场景。

---

## 4. 预期价值

- 新仓库 5 分钟完成文档初始化，不再依赖人工
- 文档覆盖度可量化，可纳入项目健康度指标（与 gitlink-insight 联动）
- 充分展示实验一 wiki 模块的实用价值
