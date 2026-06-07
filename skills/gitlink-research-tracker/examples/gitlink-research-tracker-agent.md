# gitlink-research-tracker 使用示例

> 触发方式：显式 `/` 命令调用（自然语言触发冲突已知问题）
> 日期：2026-05-28
> 验证状态：✅ 通过（显式调用 + 全部命令执行成功 + 报告按模板输出）

---

## 用户输入

```
/gitlink-research-tracker 帮我调研一下 GitLink 上 AI Agent 相关的项目，做个技术分析
```

## Skill 触发

显式斜杠命令调用，100% 命中，同时自动加载前置依赖 `gitlink-shared`。

> **已知限制**：自然语言调用时 `gitlink-search` 的语义覆盖过宽（"在 GitLink 上搜索资源"），会抢走 research-tracker 的触发。显式 `/` 命令不受影响。

---

## 执行过程摘要

### Step 1：关键词拆解

6 个搜索关键词：`agent`、`智能体`、`LLM`、`LangChain`、`AutoGPT`、`copilot`

### Step 2：多关键词并行搜索

```bash
gitlink-cli search +repos -k "agent" --format json
gitlink-cli search +repos -k "智能体" --format json
gitlink-cli search +repos -k "LLM" --format json
gitlink-cli search +repos -k "LangChain" --format json
gitlink-cli search +repos -k "AutoGPT" --format json
gitlink-cli search +repos -k "copilot" --format json
```

命中 347 个仓库，去重后筛选 12 个核心项目。

### Step 3：深度评估

对全部 12 个项目执行 `repo +info`，提取贡献者数/watch/fork/Release/镜像状态。

### Step 4：评分与分类

- 镜像项目正确标注 `[镜像]`，评分加 `*` 并单独说明
- 原创项目按 5 维度评分（满分 25）
- 镜像占比 58%，原创项目平均评分 9.8/25

### 验证要点

| 检查项 | 状态 |
|--------|------|
| 镜像仓库正确标注 `[镜像]` 和 `*` 评分 | ✅ |
| 镜像修正声明"不参与排名比较" | ✅ |
| 5 维度评分符合 SKILL.md 标准 | ✅ |
| 报告六段式模板完整 | ✅ |
| 趋势洞察有数据支撑 | ✅ |
| 生态空白分析有实际价值 | ✅ |
| 项目去重（author.login/identifier） | ✅ |
| `language` 为 null 时标注正确 | ✅ |

---

## 最终报告摘要

```
# 技术调研报告：GitLink 平台 AI Agent 项目

## 核心发现：
- GitLink 上 AI Agent 项目以镜像仓库为主（58%）
- 原创项目平均评分仅 9.8/25，处于早期探索阶段
- 原创 Top 1：datawhalechina/动手学Agent应用开发（12/25）
- 镜像 Top 1：Gitconomy/Git4GenThinking（22/25*）

## 生态空白：
- 无原创可嵌入 Agent SDK
- 缺少中文 Agent 评测基准
- Agent 运维工具链完全空白
```

---

## 验证结论

| 检查项 | 状态 |
|--------|------|
| 显式 `/` 命令正确触发 | ✅ |
| 自动加载 gitlink-shared 前置依赖 | ✅ |
| 6 关键词并行搜索 | ✅ |
| `repo +info` 深度评估 | ✅ |
| 镜像/原创/Fork 分类正确 | ✅ |
| 5 维度评分正确应用 | ✅ |
| 六段式报告模板完整 | ✅ |
| 镜像修正逻辑正确 | ✅ |
| `language` null → "未知" | ✅ |
| Release=0 → 2 分（非镜像） | ✅ |
| 生态空白分析 | ✅ |
| 自然语言触发被 gitlink-search 拦截 | ⚠️ 已知限制 |
