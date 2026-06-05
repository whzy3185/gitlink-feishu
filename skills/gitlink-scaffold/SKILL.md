---
name: gitlink-scaffold
version: 1.0.0
description: "社区健康文件体检与模板生成：检测仓库是否缺失 README、LICENSE、CONTRIBUTING、CODE_OF_CONDUCT、SECURITY、Issue/PR 模板、CHANGELOG 等开源社区推荐文件，给出健康度评分，并为缺失文件生成中文模板。当用户提到「社区健康文件」「CONTRIBUTING」「行为准则」「Issue 模板」「PR 模板」「开源规范」「仓库体检」「scaffold」时触发。"
metadata:
  requires:
    bins: ["gitlink-cli"]
    optional_bins: ["python"]
  cliHelp: "gitlink-cli repo --help"
---

# gitlink-scaffold（社区健康文件体检与模板生成）

**CRITICAL — 开始前先阅读 [`../gitlink-shared/SKILL.md`](../gitlink-shared/SKILL.md)，其中包含认证、权限处理和 API 注意事项。**
**CRITICAL — GitLink 操作只能用 `gitlink-cli`。禁止用 `gh`（GitHub CLI）操作 GitLink 资源。`gh` 仅适用于 GitHub 平台。**
**CRITICAL — 把生成的模板提交到仓库属于写操作，执行前必须征得用户确认。本技能默认只在本地生成模板，不自动提交。**

> **前置条件：** 先阅读 [`../gitlink-shared/SKILL.md`](../gitlink-shared/SKILL.md)。

## 何时使用本技能

- 维护者想检查仓库的开源社区规范是否齐全
- 用户问「我的项目还缺哪些社区文件」「帮我加个 CONTRIBUTING / 行为准则 / Issue 模板」
- 新建仓库后想快速补齐社区健康文件
- 准备开源发布前的合规体检

## 何时不使用

- 许可证兼容性 / 敏感信息扫描 → 用 `gitlink-compliance` / `gitlink-license-compliance`
- 仅查看仓库基本信息 → 用 `gitlink-repo`

## 能力概览

| 能力 | 说明 |
|------|------|
| 健康文件体检 | 检测 README/LICENSE/CONTRIBUTING/CODE_OF_CONDUCT/SECURITY/Issue 模板/PR 模板/CHANGELOG 是否存在 |
| 健康度评分 | 按权重计算 0-100 分，标记缺失的关键文件 |
| 模板生成 | 为缺失且支持的文件生成可直接使用的中文模板 |

## 工作流 1：仓库社区健康体检

### 方式 A：用配套脚本（推荐）

```bash
# 体检并输出 Markdown 报告
python scripts/scaffold.py --owner Gitlink --repo gitlink-cli

# JSON 输出
python scripts/scaffold.py --owner Gitlink --repo gitlink-cli --format json
```

参数说明：

| 参数 | 类型 | 必填 | 说明 |
|------|------|:----:|------|
| `--owner` | string | 是* | 仓库所有者（*或用 `--slug`） |
| `--repo` | string | 是* | 仓库名称 |
| `--slug` | string | 否 | `owner/repo` 或完整 URL |
| `--ref` | string | 否 | 分支或标签，默认 master |
| `--generate` | flag | 否 | 为缺失文件生成模板 |
| `--output-dir` | string | 否 | 模板输出目录，默认 scaffold_out |
| `--format` | string | 否 | `markdown`（默认）或 `json` |
| `--output` | string | 否 | 报告输出文件 |

### 方式 B：用 gitlink-cli 命令检查

```bash
# 列出仓库根目录文件，人工核对社区文件是否齐全
gitlink-cli api GET /:owner/:repo/sub_entries --query 'filepath=&ref=master' --format json

# 检查 .gitlink / .github 目录下是否有模板
gitlink-cli api GET /:owner/:repo/sub_entries --query 'filepath=.gitlink&ref=master' --format json
```

## 工作流 2：生成缺失的模板

```bash
# 体检并为缺失文件生成模板到 out/ 目录
python scripts/scaffold.py --owner Gitlink --repo gitlink-cli --generate --output-dir out

# 确认模板内容后，由用户决定是否提交到仓库（写操作，需确认）
# 例如通过 gitlink-cli 的文件创建接口提交（参考 gitlink-shared 的文件操作说明）
```

可生成模板的文件：CONTRIBUTING、CODE_OF_CONDUCT、SECURITY、Issue 模板、PR 模板、CHANGELOG。

## API 注意事项

- 检测依赖 `sub_entries` 接口列目录；不同项目把社区文件放在根目录、`.gitlink/`、`.github/` 或 `docs/`，本工具会逐个目录查找。
- 生成的模板仅写入本地，**提交到仓库是写操作**，需用户确认后再执行。
- 数据采集全程只读。

## 输出示例

参见 [`examples/`](examples/)：体检报告与生成的模板文件。

## References

- [api-reference.md](references/api-reference.md) — 采集接口、字段与文件提交说明
- [health-files.md](references/health-files.md) — 健康文件清单、权重与评分规则
- [gitlink-shared](../gitlink-shared/SKILL.md) — 认证、全局参数、安全规则
