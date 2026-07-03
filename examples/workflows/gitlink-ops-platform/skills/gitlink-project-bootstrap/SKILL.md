---
name: gitlink-project-bootstrap
version: 3.0.0
description: "项目初始化编排：依次调度 gitlink-repo → gitlink-issue → gitlink-milestone 三个子 Skill。当用户需要快速初始化一个新项目时触发。"
metadata:
  requires:
    bins: ["gitlink-cli"]
  cliHelp: "gitlink-cli repo --help"
---

# gitlink-project-bootstrap（项目初始化 · 编排 Skill）

> **你是编排者（Orchestrator），不是执行者。每一步都通过 Skill 工具调用对应的子 Skill 来完成。不要自己直接跑命令。**

---

## 编排架构

```
gitlink-project-bootstrap
  ├── Step 1 → Skill("gitlink-repo")     建仓库 + 生成标配文件
  ├── Step 2 → Skill("gitlink-issue")    创建初始 Issue
  └── Step 3 → Skill("gitlink-milestone")  创建里程碑
```

## 子 Skill 依赖

| 子 Skill | 用途 | 写入 |
|----------|------|:----:|
| gitlink-repo | 创建仓库 + README/LICENSE/.gitignore/CI 配置 | 是 |
| gitlink-issue | 创建初始开发 Issue | 是 |
| gitlink-milestone | 设定版本里程碑 | 是 |

---

## 前置：收集参数

开始前确认以下信息（未提供则询问用户）：

| 参数 | 说明 | 示例 |
|------|------|------|
| 项目名称 | kebab-case | `hello-demo` |
| 项目描述 | 一句话 | `Python 演示项目` |
| owner | 归属用户/组织 | `ZxR123-Z` |
| 项目类型 | 决定 .gitignore 和 CI 模板 | `Python` |
| 可见性 | 公开/私有 | 公开 |

---

## 工作流

### Step 1：建仓库 + 生成文件

**→ 调用 Skill 工具：`Skill("gitlink-repo", args="在 <owner> 下创建仓库 <project-name>，描述：<描述>。然后生成 README.md、LICENSE(MIT)、.gitignore(<项目类型>)、.gitlink-ci.yml 文件。注意 create_file 需要 base64 编码 content，Windows 下需要 MSYS_NO_PATHCONV=1。")`**

调用后记录 owner/repo 名称，传递给后续步骤。

### Step 2：创建初始 Issue

**→ 调用 Skill 工具：`Skill("gitlink-issue", args="在 <owner>/<repo> 下创建 3 个初始 Issue：(1) 项目初始化：搭建基础架构，(2) feat: 实现核心功能 MVP，(3) test: 补充单元测试和集成测试。每个 Issue 的 body 包含目标描述和任务清单。")`**

### Step 3：创建里程碑

**→ 调用 Skill 工具：`Skill("gitlink-milestone", args="在 <owner>/<repo> 下创建两个里程碑：v0.1.0 MVP（截止 2 周后）和 v1.0.0 正式版（截止 2 个月后）。")`**

---

## 最终输出

三个子 Skill 执行完毕后，汇总输出：

```markdown
## 🚀 项目初始化报告 — <owner>/<repo>
- URL: https://www.gitlink.org.cn/<owner>/<repo>
- 生成文件：README.md / LICENSE / .gitignore / .gitlink-ci.yml
- 初始 Issue：#1 基础架构 / #2 MVP / #3 测试
- 里程碑：v0.1.0 MVP / v1.0.0 正式版
```
