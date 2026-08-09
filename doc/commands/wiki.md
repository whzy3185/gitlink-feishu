# wiki — Wiki 管理命令

> 关联 Issue: #13 | PR: #10

## 概述

wiki 模块提供 GitLink 仓库 Wiki 页面的管理命令，支持列出、查看、创建、更新和删除 Wiki 页面。

## 命令列表

### wiki +pages
- **用途**: 列出仓库所有 Wiki 页面
- **API**: GET /api/wiki/wikiPages
- **参数**: 无（自动从 git remote 推断 owner/repo）
- **示例**: `gitlink-cli wiki +pages`

### wiki +get
- **用途**: 获取指定 Wiki 页面内容
- **API**: GET /api/wiki/getWiki?id=\<id\>
- **参数**: --id, -i (必填) Wiki 页面 ID
- **示例**: `gitlink-cli wiki +get --id 42`

### wiki +create
- **用途**: 创建新的 Wiki 页面
- **API**: POST /api/wiki/createWiki
- **参数**:
  - --title, -t (必填) 页面标题
  - --content, -c (必填) 页面内容（Markdown）
  - --project (可选) 项目 ID
- **示例**: `gitlink-cli wiki +create --title "Getting Started" --content "# Welcome"`

### wiki +update
- **用途**: 更新已有 Wiki 页面
- **API**: PUT /api/wiki/updateWiki
- **参数**:
  - --id, -i (必填) Wiki 页面 ID
  - --title, -t (可选) 新标题
  - --content, -c (可选) 新内容（Markdown）
- **示例**: `gitlink-cli wiki +update --id 42 --title "Updated Title"`

### wiki +delete
- **用途**: 删除 Wiki 页面
- **API**: POST /api/wiki/deleteWiki
- **参数**: --id, -i (必填) Wiki 页面 ID
- **示例**: `gitlink-cli wiki +delete --id 42`

## 向后兼容性

无破坏性变更。所有命令通过 wiki 域组 + 前缀添加。
