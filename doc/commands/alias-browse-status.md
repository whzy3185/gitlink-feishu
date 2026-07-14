# alias — 命令别名管理

> 关联 Issue: #14 | PR: #11

## 概述

alias 命令用于管理 gitlink-cli 的命令别名，将常用长命令缩短为简短别名，提升日常使用效率。对标 `gh alias`。

## 命令列表

### alias +list
- **用途**: 列出所有已定义的命令别名
- **示例**: `gitlink-cli alias +list`

### alias +set \<name\> \<command\>
- **用途**: 设置命令别名
- **参数**: name（别名名称）和 command（实际命令）
- **示例**:
  - `gitlink-cli alias +set rl "repo +list"`
  - `gitlink-cli alias +set ri "repo +info --owner Gitlink --repo gitlink-cli"`

### alias +delete \<name\>
- **用途**: 删除已定义的命令别名
- **参数**: name（要删除的别名名称）
- **示例**: `gitlink-cli alias +delete rl`

## 存储

别名存储在 `~/.config/gitlink-cli/aliases.yaml` 文件中，格式为 YAML。

---

# browse — 浏览器打开 GitLink 页面

> 关联 Issue: #14 | PR: #11

## 概述

browse 命令用于在浏览器中快速打开当前仓库或指定资源的 GitLink 页面。对标 `gh browse`。

## 用法

```
gitlink-cli browse [resource]
```

- 不带参数：打开当前仓库主页
- 带参数：打开指定资源页面

## 示例

```bash
# 打开当前仓库主页
gitlink-cli browse

# 打开指定 Issue
gitlink-cli browse issues/42

# 打开指定 PR
gitlink-cli browse pulls/128

# 打开 Wiki 页面
gitlink-cli browse wiki
```

## 跨平台支持

- macOS: 使用 `open` 命令
- Windows: 使用 `start` 命令
- Linux: 使用 `xdg-open` 命令

---

# status — 显示当前状态

> 关联 Issue: #14 | PR: #11

## 概述

status 命令显示 gitlink-cli 的当前状态信息，包括认证状态、API 地址、当前目录和自动推断的仓库信息。对标 `gh auth status`。

## 用法

```bash
gitlink-cli status
```

## 输出示例

```
GitLink CLI 状态
───────────────
  认证状态:    已登录
  Token 来源:  keyring / 配置文件
  API 地址:    https://www.gitlink.org.cn/api
  当前目录:    /home/user/projects/gitlink-cli
  推断仓库:    Gitlink/gitlink-cli
```

## 功能说明

- 自动检测 Token 来源（环境变量 GITLINK_TOKEN / keyring / 配置文件）
- 自动从 git remote 推断 owner/repo
- 未登录时给出 `gitlink-cli auth login` 提示
