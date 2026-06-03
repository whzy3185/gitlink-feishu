# gitlink-ci-health 使用样例

## 样例 1：CI 未激活的仓库

**日期**：2026-06-03
**仓库**：jiangtx/gitlink-cli（Fork from Gitlink/gitlink-cli）
**CLI 版本**：gitlink-cli 0.1.18

### 执行流程

```bash
# Step 1: 检查 CI 状态（方法 1 — repo +info）
gitlink-cli repo +info --owner jiangtx --repo gitlink-cli --format json
# → "open_devops": false  ← CI 未激活

# Step 1 补充（方法 2 — ci +builds）
gitlink-cli ci +builds --owner jiangtx --repo gitlink-cli --format json
# → {"status":-1,"message":"接口数据异常"}  ← 确认 CI 未激活

# 此时终止后续步骤，生成"CI 未激活"报告
```

### 关键发现

| 项目 | 值 |
|------|-----|
| `open_devops` | `false` |
| `ci +builds` 返回 | `{"status": -1, "message": "接口数据异常"}` |
| 可用 CI 命令 | `+builds`、`+logs`、`+restart`、`+stop` |
| 不存在的命令 | `ci +authorize`、`ci +activate`、`ci +deactivate` |

### 诊断结论

CI 完全未启用，需通过 GitLink Web 界面开启（仓库设置 → DevOps）。用户拥有 Manager 权限，可以操作。

### 生成的报告

```markdown
# 🔧 CI 健康巡检报告：jiangtx/gitlink-cli

> 巡检时间：2026-06-03
> 仓库：jiangtx/gitlink-cli（Fork from Gitlink/gitlink-cli）
> CI 状态：❌ 未激活

## 一、健康度总览

| 指标 | 数值 | 评分 |
|------|------|------|
| CI 激活状态 | ❌ 未激活（open_devops: false） | 0/4 |
| 整体成功率 | N/A | —/5 |
| 近期稳定性 | N/A | —/5 |
| 构建频率 | N/A | —/3 |
| 修复速度 | N/A | —/3 |
| **总分** | | **0/20** |

## 二、诊断详情

API 调用 ci +builds 返回：
{"status": -1, "message": "接口数据异常"}

仓库元数据显示 open_devops: false，确认该仓库尚未启用 GitLink 平台的 CI/CD（DevOps）服务。

## 三、改进建议

- 🔴 立即激活 CI：前往 GitLink Web 界面 → 仓库设置 → DevOps 开启 CI/CD 服务
- 🟡 配置 CI Pipeline：建议添加 .gitlink-ci.yml 配置编译和测试流水线

## 四、仓库基本信息

| 项目 | 值 |
|------|-----|
| 默认分支 | master |
| 仓库大小 | 13.4 MB |
| 贡献者 | 2 |
| PR 数量 | 9 |
| 权限 | Manager |
```

---

## 异常场景速查

| 场景 | 检测方式 | `ci +builds` 返回值 | 处理 |
|------|----------|---------------------|------|
| CI 未激活 | `repo +info` 的 `open_devops: false` | `{"status":-1,"message":"接口数据异常"}` | 建议 Web 界面激活，终止巡检 |
| CI 已激活但无构建 | `repo +info` 的 `open_devops: true` + builds 为空 | `[]` 或空列表 | 标注"暂无构建记录" |
| 构建样本不足（<5） | builds 列表长度 < 5 | 正常 JSON 数组 | 标注"数据有限，不具代表性" |

---

## 版本兼容性说明

本 skill 基于 `gitlink-cli 0.1.18` 编写。不同版本的 CI 子命令可能有差异：

| CLI 版本 | 可用 CI 命令 |
|----------|-------------|
| 0.1.18 | `+builds`、`+logs`、`+restart`、`+stop` |
| 未来版本 | 可能新增 `+activate`、`+deactivate` 等 |

当 CLI 版本更新后，重新验证可用命令：
```bash
gitlink-cli ci --help
```

---

## 样例 2：通过 Agent 调用 Skill（自动巡检）

**日期**：2026-06-03
**仓库**：jiangtx/gitlink-cli
**调用方式**：`Agent(subagent_type="general-purpose", prompt="调用 gitlink-ci-health skill，检查 jiangtx/gitlink-cli 的 CI 状态。严格按照 skill 的工作流步骤执行。")`

### Agent 自主执行的命令序列

```
工具调用 1: gitlink-cli repo +info --owner jiangtx --repo gitlink-cli --format json
           → open_devops: false  ← 发现 CI 未激活

工具调用 2: gitlink-cli ci +builds --owner jiangtx --repo gitlink-cli --format json
           → {"status": -1, "message": "接口数据异常"}  ← 二次确认
```

### Agent 决策过程

Agent 读取到 `open_devops: false` 后，**正确遵循了 skill 的异常场景处理规则**：

> "CI 未激活 → 报告 CI 状态为'未激活'，建议通过 Web 界面开启，**不再继续后续步骤**"

Agent 没有继续尝试 `ci +logs` 或统计分析，直接生成了完整报告。共消耗 **5 次工具调用**，**32,113 tokens**，耗时 **35.8 秒**。

### Agent 生成的报告

```markdown
# CI 健康巡检报告：gitlink-cli

> 巡检时间：2026-06-03
> 仓库：jiangtx/gitlink-cli
> CI 状态：未激活

## 一、健康度总览

| 指标 | 数值 | 评分 |
|------|------|------|
| CI 激活状态 | 未激活 | 0/4 |
| 整体成功率 | N/A | -/5 |
| 近期稳定性 | N/A | -/5 |
| 构建频率 | N/A | -/3 |
| 修复速度 | N/A | -/3 |
| **总分** | | **0/20** |

## 二、构建趋势

无构建记录。仓库 CI 未启用。

## 三、故障分析

CI 未激活，无构建数据可供分析。

## 四、改进建议

- **立即激活 CI**：前往 GitLink Web 界面 → 仓库设置 → DevOps，开启 CI/CD 服务。
  （CLI 暂不支持 ci +activate 命令，需要通过 Web 界面操作。）

---

总结：仓库 jiangtx/gitlink-cli（项目 ID: 1547588，默认分支: master，权限: Manager）
当前 open_devops 字段为 false，CI/CD 服务未开启。其他仓库信息：2 位贡献者、
9 个 PR、0 个 Issue，Fork 自 Gitlink/gitlink-cli。
```

### 验证结论

✅ skill v1.1.0 修复验证通过：
- Agent 正确使用了 `repo +info` 的 `open_devops` 字段判断 CI 状态
- Agent 在 CI 未激活时正确终止，没有执行后续无效步骤
- Agent 没有尝试调用不存在的 `ci +authorize` 或 `ci +activate`
- Agent 正确建议通过 Web 界面激活
- 报告结构完整，包含了仓库基本信息
