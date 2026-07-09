# notification — 通知管理

## 概述

`notification` 模块封装 GitLink 通知相关 OpenAPI：列出通知、标记已读、关注/取消关注仓库通知。

## 命令列表

### notification +list — 列出通知
- **参数**：
  - `-p, --page` 页码，默认 `1`
  - `-l, --limit` 每页数量，默认 `20`
- **示例**：
  - `gitlink-cli notification +list`
  - `gitlink-cli notification +list -l 50`

### notification +read — 标记单条已读
- **参数**：
  - `-i, --id`（必填）通知 ID
- **示例**：
  - `gitlink-cli notification +read -i 12345`

### notification +read-all — 全部标记已读
- **参数**：无
- **示例**：
  - `gitlink-cli notification +read-all`

### notification +watch — 关注/取消关注仓库通知
- **参数**：
  - `-o, --owner`（必填）仓库所有者
  - `-r, --repo`（必填）仓库名称
- **示例**：
  - `gitlink-cli notification +watch -o Gitlink -r gitlink-cli`

## 输出

支持 `--format json|table|yaml`。

## 备注

该模块当前**尚无单元测试**（见 `开发日志.md` 待办），调用前建议用 `capability +check` 确认后端通知接口可用。
