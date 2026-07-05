# 中英文档一致性守护工作流（doc-sync-automation）

把 [`gitlink-doc-sync` Skill](../../../skills/gitlink-doc-sync/SKILL.md)（中英文档一致性守护）包成**可直接运行的端到端工作流**：

> **采集 → 检测 → 报告 → 回写**：用 `repo +tree` 按命名约定自动发现双语文档对，用 `file +view --raw` 拉取两版内容，做**确定性结构比对**（章节大纲 / 代码块 / 表格行数 / 版本号），输出分级（🔴严重 / 🟡中等 / 🟢轻微）漂移报告，并（仅在 `--apply` 时）用 `issue +create` 把报告作为 tracking issue 真实回写到 GitLink。

与仓库内已有能力的关系：`file` 命令组（文件读写）→ `gitlink-doc-sync` Skill（AI 语义比对与翻译同步知识）→ **本工作流（确定性可复现闭环）**，三层互为支撑而非重复：Skill 负责需要语义理解的翻译同步，本工作流负责可进 CI 的确定性漂移检测。

## 交付物

- `scripts/doc_sync_workflow.py`：文档对发现 + 漂移检测 + 报告 + tracking issue 回写（纯标准库，Python ≥3.9，零第三方依赖）
- `tests/test_drift.py`：确定性回归护栏（同输入 → 同发现 → 同退出语义）
- `docs/verification.md`：真实平台验证证据
- `examples/demo-outputs/`：对生产环境真实仓库运行的漂移报告

## 快速运行（默认 dry-run，不写远端）

```bash
npm install -g @gitlink-ai/cli
gitlink-cli auth login

python3 scripts/doc_sync_workflow.py --owner <owner> --repo <repo> --output-dir outputs
```

- 自动发现失败时手动指定文档对：`--pair README.md:README.zh-CN.md`（可多次）
- 指定分支：`--ref develop`
- 真实回写 tracking issue：加 `--apply`（请先在自有仓库演练）
- 退出码：`0` 无严重漂移，`2` 存在严重漂移（可直接作为 CI 门禁）

## 已在真实平台验证

全部证据见 [`docs/verification.md`](docs/verification.md)，要点：

| 验证 | 对象 | 结果 |
|------|------|------|
| 文档对自动发现 | 生产 gitlink.org.cn 真实仓库 | ✅ 自动识别 README.md ⇄ README.zh-CN.md |
| 漂移检测 | 本仓库 README 双语版本 | ✅ 检出真实漂移：代码块 32 vs 29、表格 43 vs 40 行 |
| `--apply` 真实回写 | 自有 fork | ✅ tracking issue 创建成功（issue #1，API 回执确认） |
| 单测 | `tests/test_drift.py` | ✅ 9/9 全绿 |

## 设计要点

- **确定性**：漂移检测只做结构比对（标题大纲、代码块数/行数、表格行数、版本号），同输入必同输出，可进 CI；需要语义理解的翻译同步交给 `gitlink-doc-sync` Skill。
- **围栏内解析**：代码块内的 `#` 不会被误判为标题；表格分隔行不计入行数。
- **安全默认**：dry-run 为默认行为，`--apply` 才写远端，且只创建 tracking issue（可追溯、可关闭），不直接改文档。
- **CLI 为唯一依赖**：所有平台交互都通过 `gitlink-cli`（`repo +tree` / `file +view` / `issue +create`），无直接 HTTP 调用。

> 依赖 `file` 快捷命令组（PR #330）；在其合并前可用 `--cli` 指向包含该命令的本地构建。
