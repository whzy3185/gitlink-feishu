# gitlink-onboard API 参考

> **前置条件：** 先阅读 [`../../gitlink-shared/SKILL.md`](../../gitlink-shared/SKILL.md)。

本技能全程只读。

## 采集的接口

| 接口 | 用途 |
|------|------|
| `GET /:owner/:repo.json` | 项目简介、默认分支、Star/Fork |
| `GET /:owner/:repo/sub_entries.json?filepath=&ref=<branch>` | 根目录条目（技术栈识别 + 目录导航） |
| `GET /:owner/:repo/issues.json` | 识别 good-first-issue |
| `GET /:owner/:repo/contributors.json` | 核心贡献者 |

## 使用的字段

- 仓库：`description` / `default_branch` / `praises_count` / `forked_count`
- 目录条目：`name` / `type`(file|dir)
- Issue：`name`/`subject` / `description` / `issue_status`
- 贡献者：`login`/`name` / `contributions`

## 输出字段（JSON）

```json
{
  "owner","repo","description","default_branch","stars","forks",
  "stacks": ["Go","Make"],
  "navigation": [{"name":"cmd","hint":"命令行入口"}],
  "good_first": [{"id","title"}],
  "health": {"README":true,"CONTRIBUTING":false,"行为准则":false},
  "core_contributors": [{"name","contributions"}]
}
```

## 错误处理

目录采集失败时降级为空导航，不中断其他部分。沿用 gitlink-shared 错误码。
