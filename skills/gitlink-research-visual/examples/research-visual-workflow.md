# gitlink-research-visual · 端到端示例

## 场景
把科研项目时间线/热力/甘特/语言占比沉淀成交互可视化（S6）。

## 前置条件

- 已安装 gitlink-cli（`npm install -g @gitlink-ai/cli` 或 `go build`）
- 已登录：`gitlink-cli auth login`（平台命令需认证）
- 目标仓库：`--owner <owner> --repo <repo>`（git 仓库内可自动解析）

## 分步操作

该 Skill 为 AI 工作流型，在 Claude Code 里用自然语言触发效果最佳：

> 「读 skills/gitlink-research-visual/SKILL.md，<具体任务描述>」

## 输出示例

命令返回统一 envelope：
```json
{"ok":true,"data":{ ... }}
```

工作流型 Skill 的输出符合其 SKILL.md「输出模板」（含评分/判定/报告关键字段）。

## 命令速览

触发语 → AI 读 `skills/gitlink-research-visual/SKILL.md` 编排

