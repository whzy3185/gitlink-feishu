# gitlink-research-graph · 科研协作知识图谱（使用说明）

> 任务四（应用 GitLink 辅助科研）加分 Skill · 覆盖「知识图谱构建」场景
> 作者：ylly

---

## 一、这是什么

**gitlink-research-graph** 是一个科研辅助 Skill：采集 GitLink 仓库的协作关系（贡献者-PR-Issue），构建**协作知识图谱**，AI 分析**核心贡献者、协作社区、知识流动、团队结构**，帮科研工作者洞察开源团队的协作模式。

## 二、解决什么真实问题

研究一个开源项目时，科研工作者常想了解：
- **谁是核心人物**？（团队依赖谁）
- **团队怎么分工**？（分成几个协作小组）
- **问题如何被解决**？（需求→代码的路径）
- **协作健康吗**？（集中还是分散）

本 Skill **自动构建协作图谱并回答**，适用于协作生态研究、团队模式分析、开源治理参考。

## 三、图谱模型

### 节点（3 类）
- 👤 贡献者（PR/Issue/commit 作者）
- 🔀 PR（代码贡献）
- 🐛 Issue（需求/讨论）

### 边（关系）
- 贡献者 →提交→ PR
- 贡献者 →创建→ Issue
- PR →关联→ Issue（fix #N）
- 贡献者 →review→ PR

### 4 维分析
| 维度 | 分析 |
|------|------|
| 🎯 核心贡献者 | 度中心性（谁的 PR/Issue 最多）|
| 🤝 协作社区 | 聚类（共同协作的人）|
| 🔄 知识流动 | Issue→PR→merge 路径 |
| 🏗 团队结构 | 角色/分层分布 |

## 四、怎么用

### Claude Code 一句话触发
```
请阅读 examples/workflows/gitlink-research/skills/gitlink-research-graph/SKILL.md，
对 Gitlink/gitlink-cli 构建协作知识图谱并分析。
```

### 手动采集
```bash
git log --format="%an" | sort | uniq -c | sort -rn    # 贡献者 commit 排行
gitlink-cli issue +list --owner <o> --repo <r> --format json   # Issue 作者
gitlink-cli pr +list --owner <o> --repo <r> --format json      # PR（注意默认open）
```

## 五、验证案例

**Gitlink/gitlink-cli**（29 贡献者 / 322 PR / 19 Issue）：

| 发现 | 结果 |
|------|------|
| 🎯 核心枢纽 | **wbtiger**（42 commits + 9 issue，统筹合并）|
| 🤝 协作分层 | 核心层(wbtiger) / 活跃层(ylly/ZxR/zhangqing/whzy/wbavon) / 边缘层(20+) |
| 🔄 知识流动 | Issue→PR→review→merge 典型开源闭环 |
| 🏗 团队结构 | 多小组并行 + 集中审核 |

详见 [`verification.md`](./verification.md)。

## 六、适用场景

| 场景 | 用法 |
|------|------|
| **协作生态研究** | 分析开源项目的协作网络结构 |
| **核心人物识别** | 找出项目枢纽（依赖风险/关键人）|
| **团队模式研究** | 了解分工方式（模块化/集中式/混合）|
| **开源治理参考** | 为科研团队组织开源项目提供范式 |

## 七、文件清单

```
gitlink-research-graph/
├── SKILL.md          ← Skill 本体（图谱模型 + 4维分析 + 工作流）
├── README.md         ← 本文件（中文使用说明）
└── verification.md   ← 真实验证（Gitlink/gitlink-cli 协作图谱）
```

## 八、注意事项

- `pr +list` 默认只返回 open PR，已合并的要 `--state merged` 或用 `git log` 降级
- `contributors` API 可能返回 HTML，降级用 `git log` 聚合作者
- 本 Skill 输出**图谱的文字分析**；若需可视化（力导向图），导出数据给 ECharts/D3 渲染
- 纯只读采集，不写入仓库
