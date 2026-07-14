---
name: gitlink-<你的-skill-名>
version: 1.0.0
description: "<一句话功能描述>：<能做什么>。当用户需要 <触发场景 1>、<触发场景 2>、<触发关键词> 时触发。"
metadata:
  requires:
    bins: ["gitlink-cli"]
  cliHelp: "gitlink-cli <相关命令组> --help"
---

# gitlink-<你的-skill-名>（<中文名>）

**CRITICAL — 开始前必须先阅读 [`../gitlink-shared/SKILL.md`](../gitlink-shared/SKILL.md)，其中包含认证、权限处理和 API 注意事项。**
**CRITICAL — <只读 Skill 写"本 Skill 为只读操作"；写操作 Skill 写"涉及写操作，执行前必须获得用户确认"。>**
**CRITICAL — GitLink 操作只能用 `gitlink-cli`。禁止用 `gh`（GitHub CLI）操作 GitLink 资源。**

> **前置条件：** 先阅读 [`../gitlink-shared/SKILL.md`](../gitlink-shared/SKILL.md) 了解认证和全局参数。

---

## 功能概述

<用 2–4 条编号列表说明 Skill 做什么、产出什么。>

1. **<能力一>** — <说明>
2. **<能力二>** — <说明>

## 使用场景

- <场景一（用户会怎么说）>
- <场景二>

## 执行步骤

### 第 1 步：<采集>

```bash
gitlink-cli <命令> --owner <owner> --repo <repo> --format json
```

<说明关键字段与分页处理（数据多时用 --all 或按 page 循环）。>

### 第 2 步：<分析/决策>

<写清确定性规则或判断标准，避免含糊表述，保证同输入同结论。>

### 第 3 步：<产出/回写>

<只读 Skill：输出报告的固定结构（建议给出 markdown 模板）。
写操作 Skill：列出将执行的每条写命令，并要求先向用户展示计划、确认后执行。>

## 输出格式

```markdown
# <报告标题>
- 结论：...
- 依据：...
```

## 注意事项

- <平台语义坑（如两代端点差异、ID 语义、分页上限），写明规避方法>
- <权限门槛（如需管理员/组织权限的接口）>
- <失败处理（命令报错时如何降级或提示用户）>

---

## 模板使用说明（提交前删除本节）

1. 复制本目录：`cp -r skills/_template skills/gitlink-<名字>`
2. 逐节填写并删除全部 `<尖括号占位符>`；frontmatter 的 `description` 决定 Agent 何时触发，务必写清触发场景与关键词
3. 在 [`skills/README.md`](../README.md) 的索引表中加一行
4. 用真实仓库走一遍执行步骤，把实测输出贴进「输出格式」示例
5. 惯例：三条 CRITICAL 声明必须保留并按读/写性质改写第二条
