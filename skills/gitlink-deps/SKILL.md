---
name: gitlink-deps
version: 1.0.0
description: "依赖追踪：扫描仓库的依赖声明文件（go.mod、package.json、requirements.txt、pom.xml、Cargo.toml 等），解析依赖清单、统计数量、识别技术栈、提示版本锁定与供应链风险，生成依赖报告。当用户提到「项目依赖」「用了哪些库」「依赖清单」「技术栈」「go.mod」「package.json」「依赖风险」「dependencies」时触发。"
metadata:
  requires:
    bins: ["gitlink-cli"]
    optional_bins: ["python"]
  cliHelp: "gitlink-cli repo --help"
---

# gitlink-deps（项目依赖追踪）

**CRITICAL — 开始前先阅读 [`../gitlink-shared/SKILL.md`](../gitlink-shared/SKILL.md)，其中包含认证、权限处理和 API 注意事项。**
**CRITICAL — GitLink 操作只能用 `gitlink-cli`。禁止用 `gh`（GitHub CLI）操作 GitLink 资源。`gh` 仅适用于 GitHub 平台。**
**CRITICAL — 本技能全程只读，不修改任何远程数据。**

> **前置条件：** 先阅读 [`../gitlink-shared/SKILL.md`](../gitlink-shared/SKILL.md)。

## 何时使用本技能

- 用户问「这个项目依赖了哪些库 / 用了什么技术栈」
- 接手项目前想快速了解依赖规模与构成
- 想检查依赖是否锁定版本、是否存在供应链风险
- 需要一份依赖清单用于审计或文档

## 何时不使用

- 许可证合规 / 敏感信息扫描 → 用 `gitlink-compliance` / `gitlink-license-compliance`
- 仅查看仓库文件结构 → 用 `gitlink-repo`

## 支持的依赖文件

| 文件 | 生态 | 是否解析 |
|------|------|:--------:|
| `go.mod` | Go | ✅ |
| `package.json` | Node.js | ✅ |
| `requirements.txt` | Python | ✅ |
| `Cargo.toml` | Rust | ✅ |
| `pom.xml` | Java (Maven) | ✅ |
| `pyproject.toml` / `Pipfile` / `build.gradle` / `composer.json` / `Gemfile` | 多语言 | 识别存在性 |

## 工作流：扫描项目依赖

### 方式 A：用配套脚本（推荐）

```bash
# 扫描并输出依赖报告（Markdown）
python scripts/deps.py --owner Gitlink --repo gitlink-cli

# JSON 输出，供 Agent 进一步处理
python scripts/deps.py --owner Gitlink --repo gitlink-cli --format json
```

参数说明：

| 参数 | 类型 | 必填 | 说明 |
|------|------|:----:|------|
| `--owner` | string | 是* | 仓库所有者（*或用 `--slug`） |
| `--repo` | string | 是* | 仓库名称 |
| `--slug` | string | 否 | `owner/repo` 或完整 URL |
| `--ref` | string | 否 | 分支或标签，默认 master |
| `--format` | string | 否 | `markdown`（默认）或 `json` |
| `--output` | string | 否 | 报告输出文件 |

### 方式 B：用 gitlink-cli 读取依赖文件

```bash
# 读取根目录文件列表，确认有哪些依赖清单
gitlink-cli api GET /:owner/:repo/sub_entries --query 'filepath=&ref=master' --format json

# 读取具体依赖文件内容（如 go.mod）
gitlink-cli api GET /:owner/:repo/sub_entries --query 'filepath=go.mod&ref=master' --format json
```

## 报告内容

- 技术栈识别（依据存在的清单文件）
- 依赖总数、直接/间接依赖区分
- 直接依赖清单（名称 + 版本 + 生态）
- 风险提示：未锁定版本、依赖数量过多等

## API 注意事项

- 依赖文件内容通过 `sub_entries` 接口读取（单文件查询会在 `entries` 中返回明文 `content`）。
- 仅扫描仓库根目录的依赖文件；子目录/多模块项目可能需指定具体路径。
- 数据采集全程只读。

## 输出示例

参见 [`examples/`](examples/) 的真实依赖报告。

## References

- [api-reference.md](references/api-reference.md) — 采集接口、字段与输出结构
- [parsing.md](references/parsing.md) — 各生态解析规则与风险评估规则
- [gitlink-shared](../gitlink-shared/SKILL.md) — 认证、全局参数、安全规则
