# gitlink-research-graph · 端到端示例

## 场景
按关键词抓取科研仓库，构建 仓库-学者-主题 知识图谱（S2）。

## 前置条件

- 已安装 gitlink-cli（`npm install -g @gitlink-ai/cli` 或 `go build`）
- 已登录：`gitlink-cli auth login`（平台命令需认证）
- 目标仓库：`--owner <owner> --repo <repo>`（git 仓库内可自动解析）

## 分步操作

```bash
gitlink-cli search +repos --help"
```

## 输出示例

命令返回统一 envelope：
```json
{"ok":true,"data":{ ... }}
```

工作流型 Skill 的输出符合其 SKILL.md「输出模板」（含评分/判定/报告关键字段）。

## 命令速览

`gitlink-cli search +repos --help"`

