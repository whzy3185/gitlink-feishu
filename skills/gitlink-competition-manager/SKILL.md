---
name: gitlink-competition-manager
version: 1.0.0
description: "编程竞赛管理：批量创建队伍仓库、发布题目、追踪提交、生成排行榜、赛后归档。当用户需要管理编程竞赛、ACM 校内赛等竞赛时触发。"
metadata:
  requires:
    bins: ["gitlink-cli"]
  cliHelp: "gitlink-cli issue --help"
---

# gitlink-competition-manager（编程竞赛管理）

**CRITICAL — 开始前必须先阅读 [`../gitlink-shared/SKILL.md`](../gitlink-shared/SKILL.md)，其中包含认证、权限处理和 API 注意事项。**
**CRITICAL — GitLink 操作只能用 `gitlink-cli`。禁止用 `gh`（GitHub CLI）操作 GitLink 资源。**

> **前置条件：** 先阅读 [`../gitlink-shared/SKILL.md`](../gitlink-shared/SKILL.md) 了解认证和全局参数。

---

## 功能概述

本技能覆盖编程竞赛的完整管理流程：

1. **队伍管理** — 批量创建队伍仓库，配置参赛权限
2. **题目发布** — 创建题目 Issue 模板，设置截止时间
3. **提交追踪** — 监控各队伍提交记录，锁定最终版本
4. **排行榜生成** — 按通过率/用时/代码质量评分排序
5. **防作弊检测** — 跨队伍代码相似度比对 + 提交时间异常检测
6. **赛后归档** — 获奖队伍标记 + 优秀代码展示 + 仓库归档

---

## 一、赛前准备

### 1.1 获取组织信息

```bash
# 获取组织 ID（用于批量创建仓库），如果属于多个组织，需要用户手动选择使用哪个组织
gitlink-cli api GET /api/organizations --format json
# AI 匹配竞赛组织名称，获取组织 ID
```

### 1.2 批量创建队伍仓库

```bash
# 参见 gitlink-batch-repo-create Skill，本 Skill 复用其创建逻辑
# 输入格式：队伍清单 CSV 或者 Excel

# CSV 格式示例：（名字后面为参赛账号）
# team_id,team_name,leader,members,repo_name
# T001,算法之光,张三(zhangsan),"张三(zhangsan);李四(lisi);王五(wangwu)",algo-light
# T002,代码刺客,赵六(zhaoliu),"赵六(zhaoliu);钱七(qianqi)",code-assassin

# 创建仓库
gitlink-cli api POST /api/projects --body '{
  "user_id": <组织ID>,
  "name": "<队伍名>",
  "repository_name": "<仓库标识>",
  "description": "<竞赛名> - <队伍名> 参赛仓库",
  "private": true
}' --format json

# 初始化仓库（可选：竞赛模板）
gitlink-cli api POST /v1/:owner/:repo/contents --body '{
  "path": "README.md",
  "content": "<base64编码的模板内容>",
  "message": "Initialize competition repo",
  "branch": "master"
}' --format json
```

**仓库目录结构模板**：

```
<repo>/
├── README.md          # 队伍信息
├── problems/
│   ├── P001/          # 题目1解答
│   │   ├── solution.py
│   │   └── README.md  # 解题思路
│   ├── P002/          # 题目2解答
│   └── ...
├── tests/             # 自测用例
└── .gitignore
```

### 1.3 添加队伍成员为协作者

```bash

# 根据队员用户名获取对应的user_id
gitlink-cli user +info --login <member_login_name> --format json
# 为每个队伍仓库添加成员
gitlink-cli api POST /api/:owner/:repo/collaborators --body "{\"user_id\":\"<member_user_id>\"}" --format json
```

---

## 二、题目发布

### 2.1 创建题目标签

```bash
# 确保竞赛标签存在
gitlink-cli api GET /v1/:owner/:repo/issue_tags --format json

# 创建题目标签体系
gitlink-cli api POST /v1/:owner/:repo/issue_tags --body '{"name":"题目","description":"竞赛题目","color":"#0075ca"}' --format json
gitlink-cli api POST /v1/:owner/:repo/issue_tags --body '{"name":"已通过","description":"题目已通过","color":"#0e8a16"}' --format json
gitlink-cli api POST /v1/:owner/:repo/issue_tags --body '{"name":"未通过","description":"题目未通过","color":"#b60205"}' --format json
gitlink-cli api POST /v1/:owner/:repo/issue_tags --body '{"name":"待评测","description":"等待评测","color":"#fbca04"}' --format json
```

### 2.2 发布全部题目（Issue 模板，从用户对话信息中提取，如果没有提到发布题目，则不发布题目issue）

```bash
# 在竞赛主仓库创建题目 Issue
gitlink-cli issue +create \
  --owner <owner> --repo <main_repo> \
  --title "【题目 P001】两数之和" \
  --body "## 题目描述

给定一个整数数组 nums 和一个整数目标值 target，请你在该数组中找出和为目标值的那两个整数，并返回它们的数组下标。

### 输入格式
第一行：n target
第二行：n 个整数

### 输出格式
两个下标（空格分隔）

### 样例输入
4 9
2 7 11 15

### 样例输出
0 1

### 数据范围
- 2 ≤ n ≤ 10^4
- -10^9 ≤ nums[i] ≤ 10^9
- 只有一个有效答案

### 分值
100 分

### 提交方式
在 problems/P001/ 目录下提交代码，向 master 发起 PR

### 截止时间
2024-07-15 15:00:00" \
  --format json

# 打"题目"标签
gitlink-cli api PATCH /v1/:owner/:repo/issues/:id --body '{"issue_tag_ids":[<tag_id>]}' --format json
```

### 2.3 题目发布清单

```bash
# 获取所有题目 Issue
gitlink-cli issue +list --state open --owner <owner> --repo <main_repo> --format json
# AI 过滤标题以"【题目"开头的 Issue
```

**题目清单格式**：

```
=== 竞赛题目清单 ===

竞赛：2024 校内算法竞赛
题目数：5
发布时间：2024-07-15 09:00

| 题号 | 标题 | 分值 | 截止时间 | 难度 |
|------|------|------|---------|------|
| P001 | 两数之和 | 100 | 15:00 | 🟢 简单 |
| P002 | 最长回文子串 | 150 | 15:00 | 🟡 中等 |
| P003 | 合并K个有序链表 | 200 | 15:00 | 🟠 较难 |
| P004 | 最短路径 | 200 | 15:00 | 🟠 较难 |
| P005 | 动态规划优化 | 350 | 15:00 | 🔴 困难 |

总分：1000 分
```

---

## 三、提交追踪

### 3.1 监控各队伍提交

```bash
# 获取某队伍仓库的所有 PR
gitlink-cli pr +list --state open --owner <org> --repo <team_repo> --format json
gitlink-cli pr +list --state merged --owner <org> --repo <team_repo> --format json

# 批量获取所有队伍仓库的 PR
# AI 遍历所有队伍仓库，汇总提交状态
```

### 3.2 提交记录汇总

```bash
# 对每个队伍的每个 PR 获取详情
gitlink-cli pr +view --id <pr_id> --owner <org> --repo <team_repo> --format json

# 获取 PR 评论（评测反馈）
gitlink-cli api GET /v1/:owner/:repo/issues/:issue_id/journals --format json
```

**提交记录汇总格式**：

```
=== 竞赛提交记录汇总 ===

| 队伍 | P001 | P002 | P003 | P004 | P005 | 总提交数 | 最后提交时间 |
|------|------|------|------|------|------|---------|-------------|
| 算法之光 | ✅ 通过 | ✅ 通过 | ❌ 未通过 | ✅ 通过 | — | 8 | 14:52 |
| 代码刺客 | ✅ 通过 | ✅ 通过 | ✅ 通过 | ❌ 未通过 | ❌ 未通过 | 12 | 14:58 |
| AC之王 | ✅ 通过 | ✅ 通过 | ✅ 通过 | ✅ 通过 | ✅ 通过 | 15 | 14:45 |
| 菜鸟队 | ✅ 通过 | ❌ 未通过 | — | — | — | 3 | 13:20 |

✅ = 已通过  ❌ = 未通过  — = 未提交
```

### 3.3 最终版本锁定

```bash
# 竞赛截止后，锁定各队伍最终提交
# 方式：关闭截止后的新 PR + 标记最终版本

# 获取截止时间后的 PR
gitlink-cli pr +list --state open --owner <org> --repo <team_repo> --format json
# AI 筛选 pr_created_unix > 截止时间 的 PR

# 关闭迟到的提交
gitlink-cli issue +comment --id <issue_id> --owner <org> --repo <team_repo> \
  --body "⚠️ 此 PR 提交于截止时间之后，不予评测。" --format json
gitlink-cli issue +close --id <issue_id> --owner <org> --repo <team_repo>
```

---

## 四、排行榜生成

### 4.1 评分计算

```
评分规则（通过规则自动计算，或者通过CSV、Excel导入得分）：

方式 A：通过率排序（默认）
  总分 = Σ 各题通过分值
  排序：总分降序 → 最后通过时间升序

方式 B：用时排序（ACM 赛制）
  总罚时 = Σ (通过题目的提交时间 + 未通过提交次数 × 20分钟)
  排序：通过题数降序 → 总罚时升序

方式 C：代码质量评分
  总分 = 通过分 × 80% + 代码质量分 × 20%
  代码质量分 = 代码规范(30) + 可读性(30) + 复杂度(20) + 测试覆盖(20)
```

### 4.2 排行榜格式

```markdown
## 🏆 2024 校内算法竞赛 — 排行榜

**竞赛时间：** 2024-07-15 09:00 - 15:00
**参赛队伍：** 24 支
**题目数量：** 5 题

---

### 🥇 最终排名

| 排名 | 队伍 | 队长 | 通过题数 | 总分 | 最后通过 | 罚时 |
|------|------|------|---------|------|---------|------|
| 🥇 1 | AC之王 | 张三 | 5/5 | 1000 | 14:45 | 325min |
| 🥈 2 | 代码刺客 | 赵六 | 4/5 | 650 | 14:58 | 412min |
| 🥉 3 | 算法之光 | 李四 | 3/5 | 450 | 14:52 | 298min |
| 4 | 冲冲冲 | 王五 | 3/5 | 450 | 14:30 | 356min |
| 5 | 菜鸟队 | 钱七 | 1/5 | 100 | 13:20 | 145min |
| ... | ... | ... | ... | ... | ... | ... |

---

### 📊 题目通过统计

| 题号 | 标题 | 分值 | 通过数 | 通过率 | 平均提交次数 |
|------|------|------|--------|--------|------------|
| P001 | 两数之和 | 100 | 22/24 | 91.7% | 1.3 |
| P002 | 最长回文子串 | 150 | 18/24 | 75.0% | 2.1 |
| P003 | 合并K个有序链表 | 200 | 8/24 | 33.3% | 3.5 |
| P004 | 最短路径 | 200 | 5/24 | 20.8% | 4.2 |
| P005 | 动态规划优化 | 350 | 1/24 | 4.2% | 5.0 |

---

### ⏱️ 提交时间线

| 时间段 | 提交数 | 通过数 | 高峰说明 |
|--------|--------|--------|---------|
| 09:00-10:00 | 45 | 28 | 开局快速通过 P001 |
| 10:00-11:00 | 38 | 15 | P002 攻坚阶段 |
| 11:00-12:00 | 22 | 5 | P003/P004 难度提升 |
| 12:00-13:00 | 8 | 2 | 午休低谷 |
| 13:00-14:00 | 35 | 8 | 下午冲刺 |
| 14:00-15:00 | 52 | 12 | 最后冲刺（含多次未通过） |

---
*由 gitlink-competition-manager Skill 自动生成*
```

---

## 五、防作弊检测

### 5.1 跨队伍代码相似度检测（用户可选）

```
检测流程：
1. 提取所有队伍对同一题目的最终提交代码
2. 两两比对代码相似度
   a. 预处理：去除注释、空白、变量名重命名归一化
   b. Token 序列比对：计算编辑距离相似度
   c. AST 结构比对：比较语法树结构相似度
3. 综合相似度 = Token 相似度 × 60% + AST 相似度 × 40%
4. 标记高相似度对（> 80%）为可疑
```

**检测报告格式**：

```markdown
### 🔍 防作弊检测报告

**检测方法：** Token 序列比对 + AST 结构比对
**检测范围：** 24 支队伍 × 5 道题目 = 120 份代码

---

#### ⚠️ 可疑相似度对（> 80%）

| 队伍A | 队伍B | 题目 | Token相似度 | AST相似度 | 综合相似度 | 判定 |
|-------|-------|------|-----------|----------|-----------|------|
| 算法之光 | 冲冲冲 | P001 | 92% | 88% | 90.4% | 🔴 高度可疑 |
| 菜鸟队 | 摸鱼队 | P001 | 85% | 82% | 83.8% | 🟡 轻度可疑 |

#### ✅ 正常范围

| 统计项 | 数值 |
|--------|------|
| 比对总数 | 7140 对 |
| 可疑对数 | 2 对 |
| 可疑率 | 0.03% |
| 平均相似度 | 23.5% |

---

#### 🔴 高度可疑详情：算法之光 vs 冲冲冲（P001）

**相似代码片段**：
```python
# 算法之光
def two_sum(nums, target):
    seen = {}
    for i, num in enumerate(nums):
        diff = target - num
        if diff in seen:
            return [seen[diff], i]
        seen[num] = i

# 冲冲冲
def twoSum(nums, target):
    visited = {}
    for i, num in enumerate(nums):
        remain = target - num
        if remain in visited:
            return [visited[remain], i]
        visited[num] = i
```

**分析**：变量名不同（seen→visited, diff→remain），但代码结构和逻辑完全一致，仅做了变量重命名。

**建议**：约谈两队了解情况，要求解释解题思路。
```

### 5.2 提交时间异常检测

```
异常检测规则：
1. 短时间大量提交：同一队伍 5 分钟内提交 > 5 次
2. 提交时间高度重合：两支队伍提交时间差 < 30 秒（多次）

标记为异常的提交需人工复查。
```

---

## 六、赛后归档

### 6.1 获奖队伍标记

```bash
# 为获奖队伍仓库打标签
# 先确保标签存在
gitlink-cli api GET /v1/:owner/:repo/issue_tags --format json
gitlink-cli api POST /v1/:owner/:repo/issue_tags --body '{"name":"冠军","description":"竞赛冠军","color":"#ffd700"}' --format json
gitlink-cli api POST /v1/:owner/:repo/issue_tags --body '{"name":"亚军","description":"竞赛亚军","color":"#c0c0c0"}' --format json
gitlink-cli api POST /v1/:owner/:repo/issue_tags --body '{"name":"季军","description":"竞赛季军","color":"#cd7f32"}' --format json

# 在主仓库创建获奖公告 Issue
gitlink-cli issue +create \
  --owner <owner> --repo <main_repo> \
  --title "【公告】2024 校内算法竞赛获奖名单" \
  --body "## 🏆 获奖名单

### 🥇 冠军
- 队伍：AC之王
- 队长：张三
- 成员：张三、李四、王五
- 通过题数：5/5
- 总分：1000

### 🥈 亚军
- 队伍：代码刺客
- 队长：赵六
- 成员：赵六、钱七
- 通过题数：4/5

### 🥉 季军
- 队伍：算法之光
- 队长：李四
- 成员：李四、王五
- 通过题数：3/5

### 仓库链接
- [AC之王仓库](https://www.gitlink.org.cn/<org>/ac-kings)
- [代码刺客仓库](https://www.gitlink.org.cn/<org>/code-assassin)
- [算法之光仓库](https://www.gitlink.org.cn/<org>/algo-light)

恭喜以上队伍！" \
  --format json
```

### 6.2 仓库归档

```bash
# 将所有队伍仓库设为只读（通过移除写权限）
# 保留仓库但将协作者权限降为 报告者
gitlink-cli api PUT /api/:owner/:repo/collaborators/change_role --body "{\"user_id\":\"<member_user_id>\",\"role\":\"Reporter\"}" --format json

# 在仓库 README 中追加归档说明（可选）
gitlink-cli issue +comment \
  --id <issue_id> --owner <org> --repo <team_repo> \
  --body "📦 本仓库已归档。竞赛已结束，仓库转为只读。如有需要请联系组织者。" --format json
```

---

## 七、执行步骤总览

### 7.1 赛前准备

```bash
# Step 1：获取组织 ID
gitlink-cli api GET /api/organizations --format json

# Step 2：批量创建队伍仓库（复用 gitlink-batch-repo-create 逻辑）
for 每个队伍:
  gitlink-cli api POST /api/projects --body '{...}' --format json
  gitlink-cli api POST /v1/:owner/:repo/contents --body '{...}' --format json  # 初始化模板
  for 每个队员:
    gitlink-cli api POST /api/:owner/:repo/collaborators --body "{...}" --format json

# Step 3：创建题目标签体系
gitlink-cli api POST /v1/:owner/:repo/issue_tags --body '{...}' --format json

# Step 4：发布题目 Issue
for 每道题:
  gitlink-cli issue +create --owner <owner> --repo <main_repo> --title "【题目 Pxxx】<标题>" --body "<题目内容>" --format json
  gitlink-cli api PATCH /v1/:owner/:repo/issues/:id --body '{"issue_tag_ids":[<tag_id>]}' --format json

# Step 5：输出竞赛信息总览（队伍列表 + 题目清单 + 仓库链接）
```

### 7.2 赛中监控

```bash
# Step 1：获取所有队伍仓库的 PR
for 每个队伍仓库:
  gitlink-cli pr +list --state open --owner <org> --repo <team_repo> --format json
  gitlink-cli pr +list --state merged --owner <org> --repo <team_repo> --format json

# Step 2：汇总提交记录
# Step 3：实时排行榜更新
# Step 4：输出当前排名 + 提交统计
```

### 7.3 赛后处理

```bash
# Step 1：锁定最终提交（关闭截止后的 PR）
gitlink-cli issue +close --id <issue_id> --owner <org> --repo <team_repo>

# Step 2：防作弊检测（代码相似度 + 提交时间异常）

# Step 3：生成最终排行榜

# Step 4：发布获奖公告 Issue
gitlink-cli issue +create --owner <owner> --repo <main_repo> --title "【公告】获奖名单" --body "<获奖信息>" --format json

# Step 5：归档队伍仓库（降权为报告者）
```

---

## 八、可配置参数

| 参数 | 默认值 | 说明 |
|------|--------|------|
| `scoring_mode` | pass_rate | 评分方式（pass_rate / acm / quality） |
| `late_penalty` | 0 | 迟交扣分 |
| `plagiarism_threshold` | 80 | 抄袭相似度阈值（%） |
| `time_anomaly_threshold` | 5 | 短时间提交异常阈值（5分钟内N次） |
| `repo_private` | true | 队伍仓库是否私有 |
| `lock_after_deadline` | true | 截止后是否锁定提交 |
| `archive_after_contest` | true | 赛后是否归档仓库 |
| `team_template` | default | 仓库初始化模板名称 |

---

## 九、常见场景示例

### 场景 A：赛前批量准备

```
用户："帮我准备校内算法竞赛，24支队伍，5道题，7月15日9点开始"

AI 执行：
1. 获取组织 ID
2. 读取队伍清单 CSV → 批量创建 24 个队伍仓库
3. 初始化仓库模板（problems/ 目录结构）
4. 添加队员为协作者
5. 创建题目标签体系
6. 逐题创建 Issue（含题目描述/分值/截止时间）
7. 输出竞赛信息总览（队伍列表 + 题目清单 + 仓库链接）
```

### 场景 B：赛中实时排行

```
用户："看看现在排行榜什么情况"

AI 执行：
1. 遍历所有队伍仓库获取 PR 数据
2. 统计各队通过/未通过情况
3. 计算排名（按评分模式）
4. 输出当前排行榜 + 提交统计 + 时间线
```

### 场景 C：赛后完整处理

```
用户："竞赛结束了，帮我做赛后处理"

AI 执行：
1. 锁定最终提交（关闭截止后的 PR）
2. 防作弊检测（代码相似度 + 提交时间异常）
3. 生成最终排行榜
4. 创建获奖公告 Issue
5. 归档队伍仓库（降权为只读）
6. 输出赛后报告（排行榜 + 防作弊报告 + 归档清单）
```

---

## 十、注意事项

- ✅ **标签预创建**：打标签前必须先查询标签列表，确认目标标签存在，不存在则先创建
- ✅ **issue_tag_ids 完整替换**：PATCH 的 issue_tag_ids 是完整替换，需包含已有标签 ID
- ✅ **仓库隐私**：竞赛仓库建议设为 private，避免队伍间互相查看代码
- ⚠️ **防作弊局限性**：Token+AST 相似度检测不能替代人工审查，仅作为辅助参考
- ⚠️ **API 频率**：24 支队伍 × 5 题目 = 120 个仓库的 PR 查询，需分批处理
- ⚠️ **协作者权限**：赛后归档时需逐个修改协作者权限，工作量与队伍数成正比
- ⚠️ **时区**：截止时间以服务器时区为准，建议明确标注 GMT+8
