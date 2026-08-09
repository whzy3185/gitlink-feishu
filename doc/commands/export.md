# export — 数据导出命令

> 关联 Issue: #15 | PR: #12

## 概述

export 模块提供将仓库数据（Issue、PR、贡献者）导出为 CSV 或 JSON 文件的能力，支持离线分析和科研用途。

## 命令列表

### export +issues
- **用途**: 导出仓库 Issue 列表为 CSV 或 JSON 文件
- **API**: GET /v1/:owner/:repo/issues
- **参数**:
  - --format, -f (可选) 输出格式: csv / json，默认 csv
  - --output, -o (可选) 输出文件路径，默认 issues.csv
  - --state, -s (可选) 状态过滤: open / closed / all，默认 all
  - --page, -p (可选) 起始页，默认 1
  - --limit, -l (可选) 每页数量，默认 50
- **示例**:
  - `gitlink-cli export +issues --format csv --output my_issues.csv`
  - `gitlink-cli export +issues --format json --state open`

### export +prs
- **用途**: 导出仓库 PR 列表为 CSV 或 JSON 文件
- **API**: GET /v1/:owner/:repo/pulls
- **参数**:
  - --format, -f (可选) 输出格式: csv / json，默认 csv
  - --output, -o (可选) 输出文件路径，默认 prs.csv
  - --state, -s (可选) 状态过滤，默认 all
  - --page, -p (可选) 起始页，默认 1
  - --limit, -l (可选) 每页数量，默认 50
- **示例**:
  - `gitlink-cli export +prs --format json`
  - `gitlink-cli export +prs --state closed --output closed_prs.csv`

### export +contributors
- **用途**: 导出贡献者统计为 CSV 或 JSON 文件
- **API**: GET /:owner/:repo/contributors
- **参数**:
  - --format, -f (可选) 输出格式: csv / json，默认 csv
  - --output, -o (可选) 输出文件路径，默认 contributors.csv
- **示例**:
  - `gitlink-cli export +contributors --format json`

## CSV 输出格式

### issues.csv
```csv
id,title,state,created_at
1,Bug fix,1,2026-01-01
```

### prs.csv
```csv
id,title,state,created_at
2,Feature PR,0,2026-02-01
```

### contributors.csv
```csv
id,login,contributions
1,dev1,42
```

## 向后兼容性

无破坏性变更。所有命令通过 export 域组 + 前缀添加。
