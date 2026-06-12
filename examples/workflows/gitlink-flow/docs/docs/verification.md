# 验证记录

## 环境

- 操作系统：Windows，PowerShell
- Python：3.12
- 数据源：GitLink 公开 API（无需 token）

## 真实仓库验证

在真实活跃仓库 `Gitlink/gitlink-cli` 上运行完整工作流：

```bash
python src/flow.py --owner Gitlink --repo gitlink-cli
```

### 六个步骤的真实产出

| 步骤 | 结果 |
|------|------|
| ① Issue 自动分拣 | 17 个 Issue → bug 1 / feature 1 / 新手友好 5 / other 10 |
| ② PR Review 汇总 | 20 个开放 PR，列出待 Review 清单（含真实 fork PR） |
| ③ Release Notes | 从 127 条提交按 conventional commits 归类，覆盖 feat/fix/docs/test/ci 等 |
| ④ 社区健康体检 | 50/100（README、LICENSE 齐全，缺 CONTRIBUTING、行为准则） |
| ⑤ 贡献者致谢 | 20 位贡献者，致谢榜前列 wbtiger / wangyue789 / wauxing |
| ⑥ 社区运营周报 | 汇总以上为一份完整 Markdown 周报 |

完整周报见 [`../examples/demo_outputs/Gitlink_gitlink-cli_flow.md`](../examples/demo_outputs/Gitlink_gitlink-cli_flow.md)。

> 值得一提：工作流在 PR Review 步骤真实抓取到了本人为子赛题二提交的 5 个 Skill PR
> （gitlink-newcomer / scaffold / deps / contributor / kb），以及其他社区成员的 PR，
> 印证了工作流读取的是真实、实时的仓库数据。

## 单元测试

```bash
python -m pytest tests/ -q
# 16 passed
```

覆盖三个子工作流（triage / pr-review / release-notes）、复用 Skill 步骤（health / contributors）
与编排器 run_flow，全部使用合成数据 + FakeClient，不触网。

## Agent 平台验证

本工作流可由 AI Agent（如 Kiro CLI）按以下方式调用：

> 用户：「帮我给 Gitlink/gitlink-cli 生成一份社区运营周报」
> Agent：识别意图 → 执行 `python src/flow.py --owner Gitlink --repo gitlink-cli` → 解读周报

工作流输出支持 `--format json`，便于 Agent 解析后嵌入更大的自动化链路。

## 可复现性

```powershell
.\scripts\run_demo.ps1
```

所有数据来自 GitLink 平台公开接口实时采集，未做任何人工修改；分析全程只读，不向远程写入。
