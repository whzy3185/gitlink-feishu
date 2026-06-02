---
name: gitlink-license-compliance
version: 1.0.0
description: "许可证合规检查：扫描仓库的许可证兼容性、依赖项合规性与敏感信息泄露风险，生成结构化合规评估报告。当用户需要检查项目许可证合规性、排查敏感信息泄露、评估开源风险时触发。"
metadata:
  requires:
    bins: ["gitlink-cli"]
  cliHelp: "gitlink-cli repo --help"
---

# gitlink-license-compliance（许可证合规检查）

**CRITICAL — 开始前必须先阅读 [`../gitlink-shared/SKILL.md`](../gitlink-shared/SKILL.md)。**
**CRITICAL — GitLink 操作只能用 `gitlink-cli`。**

> **前置条件：** 先阅读 [`../gitlink-shared/SKILL.md`](../gitlink-shared/SKILL.md) 了解认证和全局参数。

## 功能概述

本技能为科研项目、开源项目提供全面的合规性扫描，包括：

1. **许可证识别** — 检测仓库根目录及子目录的许可证声明
2. **许可证兼容性分析** — 评估项目许可证与依赖项许可证的兼容性
3. **敏感信息扫描** — 检测代码中硬编码的密钥、密码、Token 等敏感数据
4. **合规报告生成** — 输出结构化的合规评估报告

---

## 一、许可证识别

### 获取仓库根目录文件列表

```bash
# 先克隆仓库到本地（如果尚未克隆）
git clone https://www.gitlink.org.cn/<owner>/<repo>.git
cd <repo>

# 获取根目录文件列表，查找许可证文件
git ls-tree --name-only HEAD
```

### 常见许可证文件名

```
LICENSE
LICENSE.md
LICENSE.txt
LICENSE-MIT
LICENSE-APACHE
COPYING
COPYING.md
```

### 读取许可证文件内容

```bash
# 读取 LICENSE 文件内容
git show HEAD:LICENSE
# 或查看其他常见许可证文件
git show HEAD:LICENSE.md
git show HEAD:LICENSE.txt
```

### 获取 GitLink 平台支持的许可证列表

```bash
# 获取仓库中的许可证文件
git ls-tree --name-only HEAD | grep -iE "(license|copying)"
# 如果仓库未声明许可证，结合下方的关键词识别规则 AI 直接判断
```

### 常见许可证识别关键词

| 许可证 | 识别关键词 |
|--------|-----------|
| MIT | `MIT License`, `Permission is hereby granted, free of charge` |
| Apache-2.0 | `Apache License, Version 2.0` |
| GPL-2.0 | `GNU GENERAL PUBLIC LICENSE`, `Version 2` |
| GPL-3.0 | `GNU GENERAL PUBLIC LICENSE`, `Version 3` |
| LGPL-2.1 | `GNU LESSER GENERAL PUBLIC LICENSE`, `Version 2.1` |
| BSD-2-Clause | `Redistribution and use in source and binary forms` |
| BSD-3-Clause | `Neither the name of`, `nor the names of its contributors` |
| MPL-2.0 | `Mozilla Public License, v. 2.0` |
| MulanPSL-2.0 | `木兰宽松许可证`, `Mulan Permissive Software License` |
| CC0-1.0 | `Creative Commons Legal Code`, `CC0` |

---

## 二、许可证兼容性分析

### 许可证分类（按限制程度）

```
宽松型（Permissive）—— 限制最少，可自由商用和闭源分发：
  MIT、Apache-2.0、BSD-2-Clause、BSD-3-Clause、ISC、MulanPSL-2.0

弱传染型（Weak Copyleft）—— 修改的库文件需开源，但不传染整体：
  LGPL-2.1、LGPL-3.0、MPL-2.0、EPL-2.0

强传染型（Strong Copyleft）—— 整个项目需以相同许可证开源：
  GPL-2.0、GPL-3.0、AGPL-3.0

网络服务传染型（Network Copyleft）—— 通过网络提供服务也需开源：
  AGPL-3.0
```

### 许可证兼容性矩阵

下表描述项目许可证（行）与依赖许可证（列）的兼容性：

| 项目 \ 依赖 | MIT | Apache-2.0 | GPL-2.0 | GPL-3.0 | LGPL-2.1 | AGPL-3.0 | MulanPSL-2.0 |
|-------------|-----|-----------|---------|---------|----------|----------|--------------|
| MIT | ✅ | ✅ | ❌ | ❌ | ⚠️ | ❌ | ✅ |
| Apache-2.0 | ✅ | ✅ | ❌ | ⚠️ | ⚠️ | ❌ | ✅ |
| GPL-2.0 | ✅ | ❌ | ✅ | ❌ | ✅ | ❌ | ✅ |
| GPL-3.0 | ✅ | ✅ | ⚠️ | ✅ | ✅ | ⚠️ | ✅ |
| MulanPSL-2.0 | ✅ | ✅ | ❌ | ❌ | ⚠️ | ❌ | ✅ |

> ✅ 兼容 | ⚠️ 需谨慎（存在条件兼容场景）| ❌ 不兼容（引入此依赖可能违反许可证）

### 依赖项许可证扫描

```bash
# 获取仓库文件列表，定位依赖声明文件
git ls-tree --name-only HEAD

# 常见依赖文件：
# Go:     go.mod
# Python: requirements.txt, setup.py, pyproject.toml
# Node:   package.json
# Java:   pom.xml, build.gradle
# Rust:   Cargo.toml

# 读取依赖文件内容
git show HEAD:go.mod
```

### AI分析依赖许可证的步骤

1. 读取依赖声明文件，提取所有依赖包名和版本
2. 对每个依赖包，根据已知知识库判断其许可证类型
3. 对照兼容性矩阵，评估与项目主许可证的兼容性
4. 对不确定的依赖，标记为"需人工确认"

---

## 三、敏感信息扫描

### 扫描目标文件类型

```bash
# 获取仓库文件树，确定扫描范围
git ls-tree --name-only HEAD

# 递归列出子目录文件
git ls-tree -r --name-only HEAD
```

**高风险文件（优先扫描）：**

```
配置文件：.env, .env.local, .env.production, config.yaml, config.json, settings.py
证书文件：*.pem, *.key, *.p12, *.pfx, *.crt, *.cer
SSH 密钥：id_rsa, id_dsa, id_ecdsa, id_ed25519
数据库配置：database.yml, db.conf, datasource.properties
CI 配置：.travis.yml, .github/workflows/*.yml, .gitlink-ci.yml
```

### 敏感信息特征模式

AI扫描文件内容时，重点识别以下模式：

```
密钥/Token 特征（正则）：
  AWS:        AKIA[0-9A-Z]{16}
  GitHub:     ghp_[A-Za-z0-9]{36}
  GitLink:    glpat-[A-Za-z0-9\-_]{20}
  私钥头:     -----BEGIN (RSA |EC |DSA |OPENSSH )?PRIVATE KEY-----
  通用密钥:   (password|passwd|pwd|secret|token|apikey|api_key)\s*[=:]\s*['"]?[^\s'"]{8,}
  数据库连接: (mysql|postgres|mongodb)://[^@]+:[^@]+@

IP/内部地址：
  内网 IP:    (192\.168\.|10\.\d+\.|172\.(1[6-9]|2[0-9]|3[0-1])\.)
  localhost:  localhost:\d{4,5}（需结合上下文判断是否为敏感配置）
```

### 读取特定文件内容

```bash
# 读取配置文件内容（用于敏感信息检测）
git show HEAD:.env
# 文件不存在时会报错，可用以下方式判断文件是否存在
git ls-tree --name-only HEAD | grep "^\.env$"
```

### 敏感信息风险等级

| 风险等级 | 描述 | 处理建议 |
|---------|------|---------|
| 🔴 严重 | 真实密钥/Token 暴露在代码中 | 立即撤销密钥，从 Git 历史清除 |
| 🟠 高 | 密码/数据库连接串硬编码 | 替换为环境变量，清理历史提交 |
| 🟡 中 | 内网地址/测试账号泄露 | 评估影响范围，按需处理 |
| 🟢 低 | 疑似敏感但可能是示例数据 | 人工确认后决定是否处理 |

---

## 四、合规报告生成

### 完整合规报告模板

```markdown
# 📋 仓库合规检查报告

**仓库：** owner/repo
**检查时间：** 2026-05-07
**检查人：** AI Agent（gitlink-license-compliance v1.0.0）

---

## 一、许可证概览

| 项目 | 结果 |
|------|------|
| 主许可证 | ✅ MIT License |
| 许可证文件位置 | `LICENSE` |
| 商业使用 | ✅ 允许 |
| 闭源分发 | ✅ 允许 |
| 专利授权 | ⚠️ 无明确专利授权（建议升级为 Apache-2.0） |

---

## 二、依赖项许可证分析

| 依赖包 | 版本 | 许可证 | 兼容性 |
|--------|------|--------|--------|
| github.com/spf13/cobra | v1.8.0 | Apache-2.0 | ✅ 兼容 |
| github.com/stretchr/testify | v1.9.0 | MIT | ✅ 兼容 |
| gopkg.in/yaml.v3 | v3.0.1 | MIT / Apache-2.0 | ✅ 兼容 |
| golang.org/x/sys | v0.21.0 | BSD-3-Clause | ✅ 兼容 |
| example.com/gpl-lib | v1.0.0 | GPL-3.0 | ❌ 不兼容 |

### 不兼容项详情

**[1] example.com/gpl-lib（GPL-3.0）**
- 问题：GPL-3.0 要求整个项目以 GPL-3.0 发布，与 MIT 主许可证冲突
- 影响：若分发包含此依赖的产品，需将整个项目改为 GPL-3.0
- 建议：寻找 MIT/Apache-2.0 许可证的替代库，或与法务确认使用场景

---

## 三、敏感信息扫描结果

### 风险汇总

| 风险等级 | 数量 |
|---------|------|
| 🔴 严重 | 0 |
| 🟠 高 | 1 |
| 🟡 中 | 0 |
| 🟢 低 | 2 |

### 详细发现

**[高风险] config/database.yml**
- 位置：第 12 行
- 内容特征：`password: prod_db_pass_123`（疑似真实密码硬编码）
- 建议：替换为环境变量 `${DB_PASSWORD}`，并检查该密码是否已在生产环境使用

**[低风险] examples/demo.go**
- 位置：第 45 行
- 内容特征：`token: "example_token_for_demo"`（疑似示例 Token）
- 建议：添加注释说明这是示例数据，如 `// TODO: replace with real token`

**[低风险] README.md**
- 位置：第 78 行
- 内容特征：内网地址 `192.168.1.100:8080`
- 建议：确认是否为文档示例，若是则无需处理

---

## 四、合规评分

| 评分维度 | 得分 | 满分 |
|---------|------|------|
| 许可证完整性 | 10 | 10 |
| 许可证兼容性 | 7 | 10 |
| 敏感信息管控 | 7 | 10 |
| **综合评分** | **24** | **30** |

**评级：B — 基本合规，存在需整改项**

---

## 五、整改建议（按优先级）

### 必须处理（P0）
1. **移除 config/database.yml 中的硬编码密码**（高风险）
   - 操作：将密码替换为环境变量
   - 检查 Git 历史：`git log --all --follow -p config/database.yml | grep password`
   - 如已有历史提交包含真实密码，需联系 GitLink 平台管理员清理

### 建议处理（P1）
2. **替换 GPL-3.0 依赖库** example.com/gpl-lib
   - 评估是否有功能等价的宽松许可证替代库
   - 若无替代，需在项目文档中说明 GPL-3.0 传染影响

### 可选优化（P2）
3. 将许可证从 MIT 升级为 Apache-2.0（获得专利保护）
4. 为 examples/ 目录添加 .gitignore 排除示例密钥文件
```

---

## 五、执行步骤总览

```bash
# Step 1：获取仓库基本信息
gitlink-cli repo +info --owner <owner> --repo <repo> --format json

# Step 2：克隆仓库到本地并获取根目录文件列表，定位许可证和依赖文件
git clone https://www.gitlink.org.cn/<owner>/<repo>.git
cd <repo>
git ls-tree --name-only HEAD

# Step 3：读取许可证文件内容
git show HEAD:LICENSE

# Step 4：读取依赖声明文件
git show HEAD:go.mod
# 根据项目类型选择对应文件（如 requirements.txt / package.json / pom.xml）

# Step 5：递归列出所有文件，读取高风险配置文件（.env, config.yaml 等）
git ls-tree -r --name-only HEAD
git show HEAD:.env

# Step 6：AI 综合分析所有数据，生成合规报告
```

---

## 注意事项

- ✅ **文件内容直接读取**：通过 `git show` 读取文件内容为明文，可直接分析
- ✅ **历史提交也需检查**：敏感信息即使已被删除，仍可能存在于 Git 历史中
- ⚠️ **依赖许可证以官方声明为准**：AI推断可能不准确，对不确定的依赖标注"需人工确认"
- ✅ **报告仅供参考**：许可证合规最终应由法务或专业人员确认
- ⚠️ **不自动修改代码**：本技能只做扫描和报告，不自动修改任何文件
- ⚠️ **不删除任何文件**：本技能不允许执行任何删除命令
