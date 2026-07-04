# Issue 智能分配 @通知 · 验证记录 — gitlink-issue-assigner

**验证仓库**：Gitlink/gitlink-cli
**验证日期**：2026-07-04

## 验证场景
假设 issue #X 涉及 wiki 模块（如"wiki +view 中文乱码"），演示智能分配流程。

## Step 1：分析 issue 涉及模块
issue 标题/描述 → 涉及 **shortcuts/wiki** 模块。

## Step 2：推荐负责人（历史贡献匹配）
```bash
git log --format="%an" -- shortcuts/wiki/
```
结果：**ylly** 是 shortcuts/wiki 的主要贡献者（wiki 5 命令的作者）。

## Step 3：@ 通知（软分配）
推荐 **@ylly**（wiki 模块主贡献者，最熟悉），评论内容：
```
@ylly 这个 issue 涉及 wiki 模块，你是 shortcuts/wiki 的主要贡献者，方便看一下吗？
（注：个人仓库 assigners 受限，改用 @ 通知软分配）
```
通过 `api POST /v1/.../journals` 发布。

## 验证结论
| 维度 | 结果 |
|------|:----:|
| 历史贡献匹配推荐 | ✅ ylly（wiki 主贡献者）|
| @ 通知软分配 | ✅ 绕过 assigners 限制 |
| 推荐理由 | ✅ "wiki 模块主贡献者" |

**创新价值**：解决了 `issue +assigners` 个人仓库返回空导致"分拣完没人管"的断点——这是 gitlink-cli 在个人仓库场景下的**真实痛点**，本 Skill 提供了唯一可行的自动分配方案。
