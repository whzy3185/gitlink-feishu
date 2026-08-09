---
name: project-health-report
description: 自动分析指定GitLink仓库的健康状况，生成包含Issue响应时间、PR合并效率、社区活跃度等关键指标的综合报告。
version: 1.1.0
author: HANXIAO
triggers:
  - "生成健康报告"
  - "项目状态分析"
  - "团队效能报告"
---

# 项目健康度报告生成技能

## 1. 技能目标
自动化采集和分析GitLink仓库数据，生成结构化的健康度报告，帮助项目维护者快速掌握项目状态。

## 2. 适用场景
- 项目月度/季度复盘
- 新贡献者了解项目活跃度
- 团队负责人评估项目健康度

## 3. 前置条件
- `gitlink-cli` 已安装并完成登录配置 (`gitlink-cli config init`)
- 拥有目标仓库的**读取权限**
- Python 3.6+ 环境（用于执行指标计算脚本）

## 4. 执行步骤

**步骤 1：获取并验证用户输入**
- 询问用户目标仓库，格式为 `<OWNER>/<REPO>`，例如 `Gitlink/gitlink-cli`。
- 如果用户未提供，使用 `gitlink-cli repo +list` 列出其仓库，并请用户选择。

**步骤 2：使用本地数据文件**
本 Skill 使用预置的干净数据文件进行演示，位于项目目录下的 `tmp/` 文件夹中：
- `tmp/clean_issues.json`
- `tmp/clean_prs.json`
- `tmp/clean_contributors.json`

**步骤 3：调用分析脚本**
```bash
python scripts/calc_health.py --issues tmp/clean_issues.json --prs tmp/clean_prs.json --contributors tmp/clean_contributors.json --output tmp/final_report.md
```

**步骤 4：生成并输出智能报告**
1. 读取 `tmp/final_report.md` 的内容。
2. 将报告呈现给用户。

**步骤 5：发送通知到飞书/钉钉（可选）**
1. **询问用户**："报告已生成，是否需要将摘要发送到飞书/钉钉群？"
2. 如果用户同意，**询问用户的 Webhook 地址**。
3. 使用 `curl` 命令发送包含报告摘要的POST请求。

## 5. 预期输出格式
```markdown
# 📊 项目健康度报告
> 报告生成时间：2026-07-03 14:30:00

## 📈 核心指标概览
| 指标 | 数值 | 状态 |
|------|------|------|
| 总 Issue 数 | 1 | - |
| Issue 关闭率 | 0% | 🟡 |

## 💡 智能改进建议
> 项目各项指标表现良好，请继续保持！
```

## 6. 错误处理
- 如果 Python 脚本执行失败，显示错误信息并引导用户检查 Python 环境。