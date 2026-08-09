# 提交规范检查 · 验证记录 — gitlink-commit-check

**验证仓库**：ylly/gitlink-cli
**验证日期**：2026-07-04

## 检查近 20 条 commit

### ✅ 规范（约 85%）
- `feat: add wiki +list shortcut`
- `fix: correct label API path to /{owner}/{repo}/labels`
- `feat(skills): 新增 Issue 智能分拣 Skill`
- `chore: 清理远端测试污染`

### ⚠️ 不规范（约 15%）+ 修复建议
| commit | 原 message | 问题 | 建议 |
|--------|-----------|------|------|
| 测试流水线 | "测试流水线" | 缺 type | → `chore: 测试流水线触发` |
| 1 | "1" | 无意义 | → 补充描述（如 `test: 触发流水线`）|
| 增加label标签管理 | "增加label标签管理" | 缺 type 前缀 | → `feat: 增加 label 标签管理` |

## 规范率：85%（17/20）

## 验证结论
- 检查模型成功识别不规范 commit（缺 type / 无意义）
- 修复建议具体可操作（给出改写后的 message）
- 规范率 85%，主要问题是早期"测试流水线"等测试提交，后续已规范

**价值**：规范 commit 提升 Release Notes 自动生成质量（配合 gitlink-release-auto）、PR 审查效率、历史可读性。
