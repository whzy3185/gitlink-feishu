---
name: gitlink-research-compliance
version: 1.0.0
description: "科研合规与复现性检查：检查 GitLink 仓库的开源合规性（LICENSE/版权/依赖）+ 科研复现性（数据/环境/依赖锁定/复现步骤），输出合规与复现性报告。当科研工作者需要评估一个仓库能否合规引用、能否稳定复现时触发。覆盖任务四「科研项目合规与复现性检查」场景。"
metadata:
  requires:
    bins: ["gitlink-cli"]
  cliHelp: "gitlink-cli repo --help"
---

# gitlink-research-compliance（科研合规与复现性检查 · 科研辅助 Skill）

**CRITICAL — 开始前先阅读 [`../../../skills/gitlink-shared/SKILL.md`](../../../skills/gitlink-shared/SKILL.md)。**
**CRITICAL — 本 Skill 为只读检查，不写入任何仓库。**
**CRITICAL — GitLink 操作只能用 `gitlink-cli`。禁止用 gh。**

> **定位**：任务四科研辅助 Skill（第 3 个）。科研复现是学术研究的基石——一个仓库能否被合规引用、能否稳定复现，直接决定其科研价值。本 Skill 检查**开源合规性**（LICENSE/版权/依赖兼容）+ **科研复现性**（数据可获取/环境说明/依赖锁定/复现步骤），输出合规与复现性报告。复用平台已有的 `gitlink-compliance` 并增强科研复现维度。

---

## 检查模型（两大类 8 项）

### 📜 开源合规性（能否合法引用）
| 检查项 | 标准 | 命令 |
|--------|------|------|
| LICENSE | 存在且为 OSI/木兰合规许可 | `repo +info` license 字段 |
| 版权声明 | 源文件头部/README 版权 | `repo +readme` 检查 |
| 依赖兼容 | 依赖许可证与项目兼容 | 读 go.mod/package.json |

### 🔬 科研复现性（能否稳定复现）⭐ 科研特色
| 检查项 | 标准 | 命令 |
|--------|------|------|
| 数据可获取 | 数据集有链接/公开存储 | `repo +readme` 找数据说明 |
| 环境说明 | 有 Dockerfile/requirements/go.mod | `repo +readme` + 文件列表 |
| 依赖锁定 | 有 lock 文件（go.sum/package-lock）| `repo +info` 或文件检查 |
| 复现步骤 | README 含 install/run 说明 | `repo +readme` 含 install/run |
| 版本稳定 | 有 Release tag（可锁定版本）| `release +list` |

---

## 工作流

### Step 1：采集合规数据
```bash
gitlink-cli repo +info --owner <owner> --repo <repo> --format json   # license/fork
gitlink-cli repo +readme --owner <owner> --repo <repo>               # 版权/数据/环境/复现步骤
```

### Step 2：采集复现性数据
```bash
gitlink-cli release +list --owner <owner> --repo <repo> --format json  # 版本稳定性
MSYS_NO_PATHCONV=1 gitlink-cli api GET /<owner>/<repo>/languages.json  # 技术栈→推断依赖文件
```

### Step 3：AI 双类 8 项评估
AI 按"开源合规(3) + 科研复现(5)"逐项判断 ✅/⚠️/❌，给依据。

### Step 4：输出合规与复现性报告
```markdown
## ⚖️ 科研合规与复现性报告 — <owner>/<repo>

### 📜 开源合规性（能否合法引用）
| 检查项 | 状态 | 详情 |
|--------|:----:|------|
| LICENSE | ✅ | MulanPSL-2.0 |
| 版权声明 | ... | |
| 依赖兼容 | ... | |

### 🔬 科研复现性（能否稳定复现）
| 检查项 | 状态 | 详情 |
|--------|:----:|------|
| 数据可获取 | ... | |
| 环境说明 | ... | |
| 依赖锁定 | ... | |
| 复现步骤 | ... | |
| 版本稳定 | ... | |

### 总评：合规 ✅/⚠️ | 复现性 ⭐x/5
### 科研使用建议：可合规引用 / 复现风险点 / 建议
```

---

## 关键避坑

| 坑 | 解决 |
|----|------|
| README 太长难解析 | AI 按关键词检索（license/install/data/docker/seed）|
| 依赖文件类型按语言不同 | 按主语言（Go→go.mod/Python→requirements/JS→package.json）|
| 中文仓库名编码 | 优先英文 repo 名 |
| 无 LICENSE 字段但根目录有文件 | `repo +readme` 检查 + 人工判断 |

---

## 实测落地参考

**验证仓库**：`Gitlink/gitlink-cli`
- 📜 合规：✅ LICENSE=MulanPSL-2.0（合规）、版权声明有、依赖(go.sum)完整
- 🔬 复现：✅ 环境说明(Go 1.26+)、依赖锁定(go.sum)、复现步骤(go build)、版本稳定(12 Release)
- **总评**：合规 ✅、复现性 ⭐4.5/5（科研复现门槛低，适合引用）

详见 `verification.md`。
