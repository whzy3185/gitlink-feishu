# gitlink-notification-digest · 端到端示例

## 场景
汇总 GitLink 通知按类型分类，生成摘要（可批量标记已读）。

## 前置条件

- 已安装 gitlink-cli（`npm install -g @gitlink-ai/cli` 或 `go build`）
- 已登录：`gitlink-cli auth login`（平台命令需认证）
- 目标仓库：`--owner <owner> --repo <repo>`（git 仓库内可自动解析）

## 分步操作

该 Skill 为 AI 工作流型，在 Claude Code 里用自然语言触发效果最佳：

> 「读 skills/gitlink-notification-digest/SKILL.md，<具体任务描述>」

## 输出示例

命令返回统一 envelope：
```json
{"ok":true,"data":{ ... }}
```

## 命令速览

触发语 → AI 读 `skills/gitlink-notification-digest/SKILL.md` 编排

