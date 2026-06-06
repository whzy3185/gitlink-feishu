# 架构说明

gitlink-flow 采用「采集 → 步骤分析 → 汇总生成」的分层管线，编排器只负责串联，
每个步骤是独立可测的纯函数。

## 数据流

```
GitLink 平台（公开 API，只读）
        │
        ▼
┌─────────────┐
│  glapi.py   │  采集：repo_info / issues / pulls / commits /
│  （采集层）  │       contributors / releases / 文件树
└─────────────┘
        │  原始数据
        ▼
┌─────────────┐
│  steps.py   │  6 个步骤（纯函数，可独立测试）：
│  （能力层）  │  ① triage_issues        Issue 自动分拣
│             │  ② pr_review_summary    PR Review 汇总
│             │  ③ release_notes        Release Notes 生成
│             │  ④ health_check         社区健康体检（复用 scaffold）
│             │  ⑤ contributor_highlights 贡献者致谢（复用 contributor）
└─────────────┘
        │  各步骤结构化结果
        ▼
┌─────────────┐
│  flow.py    │  编排器：run_flow() 按顺序调用 6 步，
│  （编排层）  │           组装为统一结果字典
└─────────────┘
        │
        ▼
┌─────────────┐
│  report.py  │  ⑥ render_weekly() 把 6 步结果汇总为
│  （生成层）  │      一份社区运营周报（Markdown）
└─────────────┘
        │
        ▼
   outputs/<owner>_<repo>_flow.md  或  JSON
```

## 分层职责

| 层 | 文件 | 职责 |
|----|------|------|
| 采集层 | `glapi.py` | 调用 GitLink 公开 API，带缓存，只读 |
| 能力层 | `steps.py` | 各步骤分析逻辑，纯函数，不触网 |
| 编排层 | `flow.py` | 串联步骤、命令行参数、批量处理 |
| 生成层 | `report.py` | 汇总为社区运营周报 |

## 为什么这样设计

**编排与能力分离**：`flow.py` 只管"按什么顺序调用哪些步骤"，`steps.py` 只管"每一步算什么"。
新增/调整步骤只需改对应层，互不影响。

**纯函数步骤**：每个步骤接收数据、返回结果，不触网、无副作用，因此能用合成数据完整单元测试
（16 个测试，不依赖网络）。

**对标官方参考**：①②③ 三个子工作流对齐官方 `examples/workflows` 的三个参考场景
（Issue 分拣 / PR Review / Release Notes），降低理解与收录成本。

## 与三个官方参考工作流的对应

| 官方参考场景 | 本作品对应步骤 | 实现 |
|--------------|----------------|------|
| Issue 自动分拣 | ① triage_issues | 按关键词/标签分类 bug/feature/question/新手友好 |
| PR Review | ② pr_review_summary | 统计 PR 状态、识别待 Review、标注 fork 来源 |
| Release Notes 生成 | ③ release_notes | 按 conventional commits 归类生成 Markdown |

## 可扩展点

- **新增步骤**：在 `steps.py` 加一个纯函数，在 `flow.py` 的 `run_flow` 接入，在 `report.py` 增加对应章节。
- **接入写操作**：当前全程只读；如需自动发布周报到 Issue，可在编排末尾增加一步调用 `gitlink-cli issue +comment`（写操作，需用户确认）。
- **更多数据源**：`glapi.py` 的客户端接口可替换为其他平台实现。
