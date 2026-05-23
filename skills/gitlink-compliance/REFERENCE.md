# gitlink-compliance API 参考

## 文件检查 API

合规检查的核心是读取仓库中的文件内容，判断是否存在、内容是否正确。

### 读取文件内容

```bash
gitlink-cli api GET /:owner/:repo/raw/<branch>/<filepath>
```

**说明：** 直接返回文件原始内容，用于检查 LICENSE、README、CONTRIBUTING 等文件。

### 获取文件列表

```bash
gitlink-cli api GET /:owner/:repo/sub_entries --query 'filepath=<path>&ref=<branch>' --format json
```

| 字段 | 类型 | 说明 |
|------|------|------|
| `data.entries[].name` | string | 文件名 |
| `data.entries[].type` | string | "file" 或 "dir" |
| `data.entries[].size` | int | 文件大小（字节） |
| `data.entries[].sha` | string | 文件 SHA |
| `data.entries[].commit.message` | string | 最后提交信息 |
| `data.entries[].commit.created_at` | string | 最后修改时间 |
| `data.entries[].is_readme_file` | boolean | 是否为 README 文件 |
| `data.entries[].direct_download` | boolean | 是否可直接下载 |

### 获取仓库信息

```bash
gitlink-cli repo +info --owner <owner> --repo <repo> --format json
```

| 字段 | 类型 | 用途 |
|------|------|------|
| `data.private` | boolean | 判断仓库可见性 |
| `data.default_branch` | string | 默认分支 |
| `data.license` | string | 许可证（如平台有返回） |
| `data.author.login` | string | 所有者 |

### 获取贡献者列表

```bash
gitlink-cli api GET /:owner/:repo/contributors --format json
```

用于检查贡献者是否签署了 CLA/DCO。

---

## 常见许可证识别

通过读取 LICENSE 文件内容，关键字匹配识别许可证类型：

| 许可证 | 关键字 | 兼容性 |
|--------|--------|--------|
| MIT | "MIT License", "Permission is hereby granted" | Apache-2.0, BSD, ISC |
| Apache-2.0 | "Apache License", "Version 2.0" | MIT, BSD, ISC |
| GPL-2.0 | "GNU GENERAL PUBLIC LICENSE", "Version 2" | 严格 copyleft |
| GPL-3.0 | "GNU GENERAL PUBLIC LICENSE", "Version 3" | 严格 copyleft |
| BSD-2/3 | "BSD", "Redistribution and use" | MIT, Apache-2.0 |
| MulanPSL-2 | "木兰", "Mulan Permissive Software License" | MIT, Apache-2.0, BSD |
| LGPL | "GNU LESSER GENERAL PUBLIC LICENSE" | 弱 copyleft |
| AGPL | "GNU AFFERO GENERAL PUBLIC LICENSE" | 严格 copyleft（网络传播） |
| Unlicense | "Unlicense", "public domain" | 所有许可证 |
| CC0 | "CC0", "Creative Commons Zero" | 公共领域 |

---

## 许可证兼容性矩阵

| 项目许可证 | 兼容的依赖许可证 | 不兼容的依赖许可证 |
|-----------|-----------------|-------------------|
| MIT | MIT, Apache-2.0, BSD-2/3, Unlicense, ISC, CC0, MulanPSL-2 | GPL-2/3 (仅分发时), AGPL |
| Apache-2.0 | Apache-2.0, MIT, BSD-2/3, ISC, Unlicense | GPL-2/3 |
| GPL-3.0 | GPL-3.0, MIT, Apache-2.0, BSD | —（兼容大多数） |
| BSD-3 | MIT, BSD-2/3, Apache-2.0, ISC | GPL-2/3, AGPL |
| MulanPSL-2 | MulanPSL-2, MIT, Apache-2.0, BSD | GPL-3 (需确认) |

---

## 合规检查清单

### 🔴 必须项检查

| 检查项 | 检查方法 | 通过标准 |
|--------|----------|----------|
| LICENSE 存在 | `raw/HEAD/LICENSE` != 404 | 文件存在且非空 |
| 许可证有效 | 内容匹配已知许可证关键字 | 可识别为 OSI 批准的许可证 |
| README 存在 | `sub_entries` 中 `is_readme_file=true` | 文件存在 |
| .gitignore 存在 | `sub_entries` 中 name=".gitignore" | 文件存在 |

### 🟡 建议项检查

| 检查项 | 检查方法 | 通过标准 |
|--------|----------|----------|
| CONTRIBUTING.md | `sub_entries` 或 `raw` | 文件存在 |
| SECURITY.md | `sub_entries` 或 `raw` | 文件存在 |
| CODE_OF_CONDUCT.md | `sub_entries` 或 `raw` | 文件存在 |
| CHANGELOG 或 Release | `sub_entries` 或 `release +list` | 任一存在 |

### 🔵 优化项检查

| 检查项 | 检查方法 | 通过标准 |
|--------|----------|----------|
| CI 配置 | 检查 `.trustie-pipeline.yml` / `.github/workflows` 等 | 存在即通过 |
| 源文件版权头 | 采样检查源码文件前 5 行 | 50% 以上文件有版权声明 |
| 依赖配置文件 | 检查 package.json / go.mod / Cargo.toml 等 | 存在即通过 |

---

## 输出格式规范

### 合规报告 Markdown 模板

```markdown
## ⚖️ 合规检查报告 — <owner>/<repo>

📅 检查时间：<YYYY-MM-DD>
📋 项目许可证：<识别到的许可证>

### 🔴 必须修复
| # | 问题 | 文件 | 建议 |
|---|------|------|------|

### 🟡 建议修复
| # | 问题 | 文件 | 建议 |
|---|------|------|------|

### 🔵 可选优化
| # | 建议 | 说明 |
|---|------|------|

### 📊 合规评分
| 维度 | 状态 | 评分 |
|------|:----:|:----:|
| 📜 许可证 | ✅/⚠️/❌ | ☆☆☆☆☆ |
| 🏷️ 版权声明 | ✅/⚠️/❌ | ☆☆☆☆☆ |
| 📦 依赖合规 | ✅/⚠️/❌ | ☆☆☆☆☆ |
| 🔒 安全策略 | ✅/⚠️/❌ | ☆☆☆☆☆ |
| 📖 项目文档 | ✅/⚠️/❌ | ☆☆☆☆☆ |

**总体合规评分：<分数>/100**
```

---

## 注意事项

1. 合规检查结果仅基于仓库中可读取的信息，不构成法律建议
2. 许可证兼容性问题建议咨询法务或 OSPO
3. 依赖分析需要根据项目语言选择对应的依赖文件
4. 不同许可证的兼容性规则可能因 Jurisdiction 而异
5. MulanPSL-2.0 是中国广泛使用的开源许可证，在 GitLink 平台上常见
6. 版权声明检查为采样性质，100% 覆盖需专业扫描工具
7. RAW API 返回的某些文件可能包含完整内容（`replace_content` 字段）
