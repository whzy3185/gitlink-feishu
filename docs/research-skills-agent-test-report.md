# 科研 Skills Agent 测试报告

## 覆盖范围

本报告覆盖本次为 GitLink 竞赛新增的 7 个只读科研 Skills：

- `gitlink-research-reproducibility`
- `gitlink-research-compliance`
- `gitlink-research-progress-tracker`
- `gitlink-research-collaboration-map`
- `gitlink-research-knowledge-graph`
- `gitlink-research-data-provenance`
- `gitlink-research-artifact-handbook`

这些 Skills 均为独立的 `gitlink-cli` Agent Skills，不依赖外部项目代码，可作为独立 Skill PR 提交。

## 自动验证

执行：

```bash
make validate-research-skills
git diff --check
```

验证脚本检查以下内容：

- 每个 Skill 的 `SKILL.md` 均具备合法 frontmatter、预期名称和可触发描述；
- 每个 Skill 均要求 Agent 先读取 `gitlink-shared`；
- 每个 Skill 默认只读，并要求使用 `gitlink-cli --format json`；
- 每个 Skill 覆盖对应场景所需命令；
- 每个 Skill 至少包含一个参考文件和一个示例文件；
- 示例包含用户请求、Agent 步骤、预期回答和具体 `gitlink-cli` 命令；
- `skills/README.md` 和 `doc/changes/research-skills-suite.md` 已索引全部 7 个 Skills。

## 离线 Agent 测试提示词

安装 Skills 后，可在 Claude Code、Cursor、Codex 或 OpenClaw 中使用以下提示词测试。

### 复现性审计

```text
请使用 gitlink-research-reproducibility 审计 songhui18/ICCV2021，输出复现性评分、缺失项和前三个整改动作。不要向 GitLink 写回任何内容。
```

预期行为：

- 读取 `gitlink-shared`；
- 执行 `repo +info` 和 `repo +tree`，必要时回退到 `api GET /sub_entries`；
- 标注命令数据来源；
- 输出清单式报告。

### 合规预审

```text
请使用 gitlink-research-compliance 检查 Gitconomy/Git4Research 的开源发布准备情况，报告许可证、SECURITY、CONTRIBUTING、依赖和敏感文件风险。
```

预期行为：

- 执行只读文件树检查；
- 避免打印敏感内容；
- 说明输出是工程预检查，不构成法律意见。

### 进度跟踪

```text
请使用 gitlink-research-progress-tracker 为 songhui18/ICCV2021 生成科研项目周报，包含 Issue/PR 队列和风险预警。
```

预期行为：

- 采集仓库、Issue、PR、Release 和贡献者数据；
- 区分事实和建议；
- 除非用户明确要求，不发布评论。

### 协作画像

```text
请使用 gitlink-research-collaboration-map 分析 Gitconomy/Git4Research 的协作结构，识别维护者、贡献者、协作缺口和下一步建议。
```

预期行为：

- 使用公开贡献者、Issue、PR、语言和可选用户数据；
- 避免输出私人联系方式；
- 建议必须有证据支撑。

### 知识图谱

```text
请使用 gitlink-research-knowledge-graph 在 GitLink 搜索“论文复现”和“open research”，输出 Markdown 趋势摘要和图谱 JSON。
```

预期行为：

- 执行有边界的关键词搜索；
- 深度分析不超过 8 个仓库；
- 按图谱 schema 输出 `nodes` 和 `edges`；
- 标注平台数据局限。

### 数据来源审计

```text
请使用 gitlink-research-data-provenance 审计 songhui18/ICCV2021 的数据集来源、引用、许可证和隐私风险。不要打印原始数据内容。
```

预期行为：

- 采集仓库信息和选定文件树；
- 只报告路径和风险类型；
- 区分证据缺失和已确认风险。

### 成果手册

```text
请使用 gitlink-research-artifact-handbook 为 Gitconomy/Git4Research 生成科研成果沉淀手册，用于项目交接和竞赛演示。
```

预期行为：

- 在可用时采集仓库、README、文件树、Issue/PR、Release、语言和贡献者信号；
- 输出结构化手册，缺失字段标注为 `待补充`；
- 除非用户明确要求，不写入 Wiki、Issue、Release 或仓库文件。

## 真实 Agent 验证

竞赛提交时，建议至少录制一次 Agent 运行过程截图或视频：

1. 安装或暴露这些 Skills 给 Agent。
2. 选择一个公开 GitLink 仓库运行本报告中的提示词。
3. 展示 Agent 读取 Skill、执行只读 `gitlink-cli` 命令并生成报告。
4. 展示未执行任何写操作。

推荐演示目标：

```text
songhui18/ICCV2021
```

该仓库具备较丰富的 Issue 和 PR 数据，便于展示进度跟踪和协作画像。

## 已知限制

- 自动脚本验证 Skill 结构和工作流说明，不证明大模型生成报告的语义质量。
- 真实 GitLink API 验证需要网络访问；私有仓库可能需要认证。
- 竞赛提交材料仍需在选定 Agent 界面中补充截图或视频。
