---
name: gitlink-graduation-thesis-repo
version: 1.0.0
description: "毕业设计仓库管理：批量创建标准化毕设仓库、追踪毕设进度（开题→中期→初稿→终稿）、导师进度看板、答辩材料归档、毕业后仓库归档。当用户需要管理毕业设计仓库、追踪毕设进度时触发。"
metadata:
  requires:
    bins: ["gitlink-cli"]
  cliHelp: "gitlink-cli repo --help"
---

# gitlink-graduation-thesis-repo（毕业设计仓库管理）

**CRITICAL — 开始前必须先阅读 [`../gitlink-shared/SKILL.md`](../gitlink-shared/SKILL.md)，其中包含认证、权限处理和 API 注意事项。**
**CRITICAL — GitLink 操作只能用 `gitlink-cli`。禁止用 `gh`（GitHub CLI）操作 GitLink 资源。**

> **前置条件：** 先阅读 [`../gitlink-shared/SKILL.md`](../gitlink-shared/SKILL.md) 了解认证和全局参数。

---


## 功能概述

本技能覆盖毕业设计从仓库创建到归档的完整生命周期：

1. **批量创建** — 按学生名单批量创建标准化毕设仓库（统一目录结构）
2. **进度追踪** — 开题报告→中期检查→论文初稿→终稿，对应 Milestone 追踪
3. **导师看板** — 导师批量查看所有学生的毕设进度
4. **格式校验** — 自动检查论文文档格式合规性
5. **答辩归档** — 答辩 PPT + 评分表 + 录屏链接统一管理
6. **毕业归档** — 仓库转为只读 + 导出完整压缩包

---

## 一、批量创建毕设仓库

### 1.1 获取组织信息

```bash
# 获取当前用户所属的组织列表（获取组织 ID）， 如果属于多个组织，需要用户手动选择使用哪个组织
gitlink-cli api GET /api/organizations --format json
```

**AI 提取的关键字段**：
- `organizations[].id`：组织 ID（创建仓库时需要）
- `organizations[].name`：组织名称
- `organizations[].nickname`：组织显示名

### 1.2 读取学生名单

```csv
# students.csv 格式
student_id,student_name,advisor,thesis_title,gitlink_username
2021001,张三,李老师,基于深度学习的图像识别系统,zhangsan
2021002,李四,王老师,区块链数据隐私保护研究,lisi
2021003,王五,李老师,分布式系统一致性算法优化,wangwu
```

### 1.3 批量创建仓库

```bash
# 创建仓库（在组织下创建）
# POST /api/projects 的 user_id 参数指定组织 ID
gitlink-cli api POST /api/projects --body '{
  "user_id": <组织ID>,
  "name": "<repo_name>",
  "repository_name": "<仓库标识>",
  "description": "<student_name>的毕业设计:<thesis_title>"
}' --format json
```

**仓库命名规则**：

| 规则 | 格式 | 示例 |
|------|------|------|
| 默认 | ` graduation-<student_id>` | `graduation-2021001` |
| 带年份 | `graduation-<year>-<student_id>` | `graduation-2024-2021001` |
| 带姓名 | `graduation-<student_name>` | `graduation-zhangsan` |

### 1.4 初始化标准化目录结构



```bash
# 参考标准化结构如下
# 在每个仓库的 master 分支创建标准化目录
# 通过 Create File API 创建 .gitkeep 文件来建立目录

# 目录结构：
# /docs/        — 论文文档（开题报告、中期报告、论文初稿、终稿）
# /src/         — 源代码
# /slides/      — 答辩 PPT
# /data/        — 数据集（如适用）
# /results/     — 实验结果
# README.md     — 项目说明

# 创建 README.md
CONTENT=$(echo -n "# <student_name> 的毕业设计

## 论文题目
<thesis_title>

## 指导教师
<advisor>

## 目录结构
- \`/docs\` — 论文文档
- \`/src\` — 源代码
- \`/slides\` — 答辩 PPT
- \`/data\` — 数据集
- \`/results\` — 实验结果

## 进度
- [ ] 开题报告
- [ ] 中期检查
- [ ] 论文初稿
- [ ] 论文终稿
- [ ] 答辩材料" | base64 -w 0)
```

GitLink 创建仓库时会**自动生成**一个默认的 README.md。因此：
- 不能直接用 `create_file` 创建 README.md（会报"文件已存在"）
- 必须先用 GET 获取文件的 `sha`，再用 `update_file` 更新内容

#### ⚠️ 关键编码差异

| API | `content` 字段格式 | 说明 |
|-----|-------------------|------|
| `POST /:owner/:repo/create_file` | **base64 编码** | 二进制安全，任何内容都 base64 |
| `PUT /:owner/:repo/update_file` | **原始文本（UTF-8 字符串）** | **不要 base64 编码！** 否则 GitLink 页面直接显示 base64 乱码 |

#### 完整执行流程

**Step 1：获取 README.md 的 sha**

```bash
# 方法 A：通过 sub_entries API（优先尝试）
gitlink-cli api GET /:owner/:repo/sub_entries --query 'filepath=README.md&ref=master' --format json
# 从返回结果中提取：data.entries.sha

# 方法 B：通过 files API（如果方法 A 返回 404，尝试此路径）
gitlink-cli api GET /:owner/:repo/files/README.md --query 'ref=master' --format json
# 从返回结果中提取：data.sha 或 data.file.sha

# 方法 C：列出根目录确认文件路径
gitlink-cli api GET /:owner/:repo/tree --query 'ref=master' --format json
```

> **注意**：如果上述路径均返回 404，先用 `tree` API 确认仓库文件列表，再确定正确路径。

**Step 2：用 update_file 更新 README.md**

```bash
# ⚠️ content 传原始文本，不要 base64 编码！
gitlink-cli api PUT /:owner/:repo/update_file --body '{
  "filepath": "README.md",
  "content": "# <student_name> 的毕业设计\n\n## 论文题目\n<thesis_title>\n\n## 指导教师\n<advisor>\n\n## 目录结构\n- `/docs` — 论文文档\n- `/src` — 源代码\n- `/slides` — 答辩PPT\n- `/data` — 数据集\n- `/results` — 实验结果\n\n## 进度\n- [ ] 开题报告\n- [ ] 中期检查\n- [ ] 论文初稿\n- [ ] 论文终稿\n- [ ] 答辩材料\n",
  "sha": "<从 Step1 获取的 sha>",
  "branch": "master",
  "message": "init: 初始化毕设仓库README"
}' --format json
```


### 1.5 设置 Milestone（毕设阶段）

```bash
# 获取现有 Milestone
gitlink-cli milestone +list --owner <owner> --repo <repo> --format json

# 创建毕设阶段 Milestone（4个阶段）
# 注意：通过 API 创建 Milestone
gitlink-cli api POST /v1/:owner/:repo/milestones --body '{
  "title": "开题报告",
  "description": "完成开题报告撰写与提交",
  "due_date": "2024-03-15"
}' --format json

gitlink-cli api POST /v1/:owner/:repo/milestones --body '{
  "title": "中期检查",
  "description": "完成中期检查报告与进度展示",
  "due_date": "2024-04-30"
}' --format json

gitlink-cli api POST /v1/:owner/:repo/milestones --body '{
  "title": "论文初稿",
  "description": "完成论文初稿撰写",
  "due_date": "2024-05-20"
}' --format json

gitlink-cli api POST /v1/:owner/:repo/milestones --body '{
  "title": "论文终稿",
  "description": "完成论文终稿修改与答辩准备",
  "due_date": "2024-06-10"
}' --format json
```

### 1.6 添加学生为协作者

```bash
# 根据学生用户名获取对应的user_id
gitlink-cli user +info --login <gitlink_username> --format json
# 为每个队伍仓库添加成员
gitlink-cli api POST /api/:owner/:repo/collaborators --body "{\"user_id\":\"<member_user_id>\"}" --format json --format json
```

---

## 二、进度追踪

### 2.1 毕设阶段定义

| 阶段 | Milestone | 截止时间（示例） | 交付物 | 对应目录 |
|------|-----------|----------------|--------|---------|
| 阶段一 | 开题报告 | 2024-03-15 | 开题报告文档 | `/docs/proposal/` |
| 阶段二 | 中期检查 | 2024-04-30 | 中期报告 + 进度演示 | `/docs/midterm/` |
| 阶段三 | 论文初稿 | 2024-05-20 | 论文初稿 | `/docs/draft/` |
| 阶段四 | 论文终稿 | 2024-06-10 | 论文终稿 + 答辩 PPT | `/docs/final/` + `/slides/` |

### 2.2 追踪每个学生的进度

```bash
# Step 1：获取仓库列表（所有毕设仓库）
gitlink-cli api GET /api/organizations --format json
# 获取组织下所有仓库
gitlink-cli api GET /v1/:owner/projects --format json

# Step 2：对每个仓库检查进度
# 获取 Milestone 完成情况
gitlink-cli milestone +list --owner <owner> --repo <repo> --format json
# 检查各 Milestone 的 issue 完成率

# Step 3：检查文件提交情况（各阶段目录是否有文件）
gitlink-cli api GET /:owner/:repo/sub_entries --query 'filepath=docs/proposal&ref=master' --format json
gitlink-cli api GET /:owner/:repo/sub_entries --query 'filepath=docs/midterm&ref=master' --format json
gitlink-cli api GET /:owner/:repo/sub_entries --query 'filepath=docs/draft&ref=master' --format json
gitlink-cli api GET /:owner/:repo/sub_entries --query 'filepath=docs/final&ref=master' --format json

# Step 4：检查最近提交时间（判断是否在持续推进）
gitlink-cli api GET /v1/:owner/:repo/commits --query 'sha=master&limit=1' --format json
```

### 2.3 进度状态判定

```
对每个学生的每个阶段，判定状态：

├─ ✅ 已完成：对应目录有文件 + Milestone 已关闭
├─ 🟡 已提交：对应目录有文件但 Milestone 未关闭
├─ 🔴 未提交：对应目录为空
├─ ⏰ 超时：未提交且当前时间 > 截止时间
└─ ⬜ 未开始：尚未到该阶段截止时间
```

---

## 三、导师进度看板

### 3.1 数据采集

```bash
# Step 1：获取所有毕设仓库列表
gitlink-cli api GET /v1/:owner/projects --format json

# Step 2：对每个仓库采集进度数据
gitlink-cli milestone +list --owner <owner> --repo <repo> --format json
gitlink-cli api GET /:owner/:repo/sub_entries --query 'filepath=docs&ref=master' --format json
gitlink-cli api GET /v1/:owner/:repo/commits --query 'sha=master&limit=1' --format json
```

### 3.2 导师看板格式

```markdown
## 📊 毕业设计进度看板 — 2024 届

**导师：** 李老师
**学生总数：** 8 人
**当前阶段：** 中期检查阶段

---

### 📈 总体进度

| 阶段 | 已完成 | 进行中 | 未开始 | 超时 | 完成率 |
|------|--------|--------|--------|------|--------|
| 开题报告 | 8 | 0 | 0 | 0 | 100% ✅ |
| 中期检查 | 5 | 2 | 1 | 0 | 62.5% 🟡 |
| 论文初稿 | 0 | 0 | 8 | 0 | 0% ⬜ |
| 论文终稿 | 0 | 0 | 8 | 0 | 0% ⬜ |

---

### 👨‍🎓 学生进度明细

| 学号 | 姓名 | 课题 | 开题 | 中期 | 初稿 | 终稿 | 最近提交 | 状态 |
|------|------|------|------|------|------|------|---------|------|
| 2021001 | 张三 | 基于深度学习的图像识别系统 | ✅ | ✅ | ⬜ | ⬜ | 2天前 | 🟢 正常 |
| 2021002 | 李四 | 区块链数据隐私保护研究 | ✅ | ✅ | ⬜ | ⬜ | 1天前 | 🟢 正常 |
| 2021003 | 王五 | 分布式系统一致性算法优化 | ✅ | 🟡 | ⬜ | ⬜ | 5天前 | 🟡 关注 |
| 2021004 | 赵六 | 微服务架构性能优化研究 | ✅ | ✅ | ⬜ | ⬜ | 3天前 | 🟢 正常 |
| 2021005 | 钱七 | 自然语言处理在客服中的应用 | ✅ | 🔴 | ⬜ | ⬜ | 12天前 | 🔴 落后 |
| 2021006 | 孙八 | 机器学习推荐系统设计 | ✅ | 🟡 | ⬜ | ⬜ | 4天前 | 🟡 关注 |
| 2021007 | 周九 | 物联网安全协议研究 | ✅ | ✅ | ⬜ | ⬜ | 1天前 | 🟢 正常 |
| 2021008 | 吴十 | 云计算资源调度算法 | ✅ | ✅ | ⬜ | ⬜ | 2天前 | 🟢 正常 |

---

### ⚠️ 需关注学生

#### 🔴 钱七（2021005）
- **课题：** 自然语言处理在客服中的应用
- **问题：** 中期检查未提交，最近 12 天无提交
- **建议：** 立即约谈，确认是否遇到困难

#### 🟡 王五（2021003）
- **课题：** 分布式系统一致性算法优化
- **问题：** 中期检查已提交但未完成，最近 5 天无更新
- **建议：** 关注中期报告质量

#### 🟡 孙八（2021006）
- **课题：** 机器学习推荐系统设计
- **问题：** 中期检查进行中，进度稍慢
- **建议：** 提醒加快进度

---

### 📊 按导师分组（如果多名导师）

| 导师 | 学生数 | 平均完成率 | 超时学生 | 正常学生 |
|------|--------|-----------|---------|---------|
| 李老师 | 5 | 70% | 1 | 4 |
| 王老师 | 3 | 83% | 0 | 3 |
| **合计** | **8** | **75%** | **1** | **7** |
```

---

## 四、格式校验

### 4.1 论文格式检查项

```bash
# 获取论文文件列表
gitlink-cli api GET /:owner/:repo/sub_entries --query 'filepath=docs/final&ref=master' --format json
# AI 检查文件命名和格式
```

**AI 自动校验清单**：

| 检查项 | 规则 | 不合规处理 |
|--------|------|-----------|
| 文件格式 | 论文为 .docx 或 .pdf | 评论提醒转换格式 |
| 文件命名 | `学号_姓名_论文.pdf` | 评论提醒重命名 |
| 文件大小 | 1MB-20MB（合理范围） | 评论提醒检查 |
| 目录结构 | 论文在 `/docs/final/` 下 | 评论提醒整理 |
| 答辩 PPT | 在 `/slides/` 下有 .pptx 文件 | 评论提醒提交 |
| README 更新 | README 中进度项已勾选 | 评论提醒更新 |

### 4.2 发布格式校验结果

```bash
# 在学生仓库创建 Issue 反馈格式问题
gitlink-cli issue +create \
  --owner <owner> --repo <repo> \
  --title "【格式检查】论文终稿格式校验结果" \
  --body "## 📋 格式校验报告

**检查时间：** 2024-06-05
**校验结果：** ❌ 有 2 项不合规

---

### 检查明细

| 检查项 | 结果 | 详情 |
|--------|------|------|
| 文件格式 | ✅ 通过 | PDF 格式 |
| 文件命名 | ❌ 不合规 | 当前：`论文终稿.pdf`，应为：`2021001_张三_论文.pdf` |
| 文件大小 | ✅ 通过 | 8.5MB |
| 目录结构 | ✅ 通过 | 在 `/docs/final/` 下 |
| 答辩 PPT | ❌ 缺失 | `/slides/` 目录下无 PPT 文件 |
| README 更新 | ✅ 通过 | 进度项已更新 |

---

### 需修正项

1. **重命名论文文件**：`论文终稿.pdf` → `2021001_张三_论文.pdf`
2. **提交答辩 PPT**：将 PPT 文件上传到 `/slides/` 目录

请在 2024-06-08 前完成修正。" \
  --format json
```

---

## 五、答辩材料归档

### 5.1 答辩材料清单

| 材料 | 存放路径 | 格式 | 必填 |
|------|---------|------|------|
| 答辩 PPT | `/slides/` | .pptx | ✅ |
| 论文终稿 | `/docs/final/` | .pdf | ✅ |
| 评分表 | `/docs/defense/` | .pdf | ✅ |
| 答辩录屏链接 | README.md | URL | ❌ |

### 5.2 检查答辩材料完整性

```bash
# 检查各路径下的文件
gitlink-cli api GET /:owner/:repo/sub_entries --query 'filepath=slides&ref=master' --format json
gitlink-cli api GET /:owner/:repo/sub_entries --query 'filepath=docs/final&ref=master' --format json
gitlink-cli api GET /:owner/:repo/sub_entries --query 'filepath=docs/defense&ref=master' --format json

# AI 检查是否所有必填材料都已提交
```

### 5.3 答辩结果记录

```bash
# 在仓库创建答辩结果 Issue
gitlink-cli issue +create \
  --owner <owner> --repo <repo> \
  --title "【答辩结果】2024届毕业设计答辩" \
  --body "## 🎓 答辩结果

**学生：** 张三（2021001）
**课题：** 基于深度学习的图像识别系统
**答辩时间：** 2024-06-15 14:00
**答辩地点：** 计算机学院 A301

---

### 评分

| 评分项 | 得分 | 满分 |
|--------|------|------|
| 选题与意义 | 18 | 20 |
| 文献综述 | 17 | 20 |
| 研究方法 | 18 | 20 |
| 成果与创新 | 19 | 20 |
| 答辩表现 | 18 | 20 |
| **总分** | **90** | **100** |

**等级：** 优秀

---

### 答辩委员会意见

论文选题具有实际应用价值，研究方法合理，实验结果充分。答辩过程表达清晰，回答问题准确。

### 建议

进一步完善实验数据的统计分析。" \
  --format json
```

---

## 六、毕业后归档

### 6.1 仓库归档操作

```bash
# 保留仓库但将学生权限降为 报告者
gitlink-cli api PUT /api/:owner/:repo/collaborators/change_role --body "{\"user_id\":\"<member_user_id>\",\"role\":\"Reporter\"}" --format json
```

### 6.2 打归档标签

```bash
# 确保标签存在
gitlink-cli api GET /v1/:owner/:repo/issue_tags --format json
# 创建归档标签
gitlink-cli api POST /v1/:owner/:repo/issue_tags --body '{"name":"已归档","description":"毕业设计已完成并归档","color":"#1a1918"}' --format json

# 创建归档 Issue（标记仓库状态）
gitlink-cli issue +create \
  --owner <owner> --repo <repo> \
  --title "【归档】2024届毕业设计已归档" \
  --body "本仓库已完成 2024 届毕业设计全部流程，现已归档。

**归档时间：** 2024-07-01
**学生：** 张三（2021001）
**最终成绩：** 优秀（90/100）
**答辩日期：** 2024-06-15

仓库已转为只读模式。如需修改，请联系导师重新开放权限。" \
  --format json

# 打"已归档"标签
gitlink-cli api PATCH /v1/:owner/:repo/issues/:id --body '{"issue_tag_ids":[<tag_id>]}' --format json
```

### 6.3 归档报告

```markdown
## 📦 毕业设计归档报告 — 2024 届

**归档时间：** 2024-07-01
**学生总数：** 8 人
**归档完成：** 8/8（100%）

---

### 归档清单

| 学号 | 姓名 | 仓库 | 成绩 | 答辩日期 | 归档状态 |
|------|------|------|------|---------|---------|
| 2021001 | 张三 | graduation-2021001 | 优秀(90) | 2024-06-15 | ✅ 已归档 |
| 2021002 | 李四 | graduation-2021002 | 良好(82) | 2024-06-15 | ✅ 已归档 |
| 2021003 | 王五 | graduation-2021003 | 良好(85) | 2024-06-15 | ✅ 已归档 |
| 2021004 | 赵六 | graduation-2021004 | 及格(65) | 2024-06-15 | ✅ 已归档 |
| 2021005 | 钱七 | graduation-2021005 | 良好(80) | 2024-06-15 | ✅ 已归档 |
| 2021006 | 孙八 | graduation-2021006 | 优秀(92) | 2024-06-15 | ✅ 已归档 |
| 2021007 | 周九 | graduation-2021007 | 良好(78) | 2024-06-15 | ✅ 已归档 |
| 2021008 | 吴十 | graduation-2021008 | 及格(68) | 2024-06-15 | ✅ 已归档 |

---

### 成绩分布

| 等级 | 人数 | 占比 |
|------|------|------|
| 优秀（≥90） | 2 | 25% |
| 良好（80-89） | 4 | 50% |
| 及格（60-79） | 2 | 25% |
| 不及格（<60） | 0 | 0% |

---

### 优秀论文推荐

| 学号 | 姓名 | 课题 | 推荐理由 |
|------|------|------|---------|
| 2021006 | 孙八 | 机器学习推荐系统设计 | 研究方法创新，实验充分 |
| 2021001 | 张三 | 基于深度学习的图像识别系统 | 工程实现完整，有实际应用价值 |
```

---

## 七、执行步骤总览

### 7.1 批量创建毕设仓库

```bash
# Step 1：获取组织 ID (多个组织时需要用户确认)
gitlink-cli api GET /api/organizations --format json

# Step 2：读取学生名单（CSV/JSON）

# Step 3：干运行预览（输出创建清单，等待确认）

# Step 4（用户确认后）：批量创建仓库
gitlink-cli api POST /api/projects --body '{"user_id":<org_id>,"name":"<repo_name>",...}' --format json

# Step 5：初始化标准化目录结构（README + 5 个目录）
gitlink-cli api POST /:owner/:repo/create_file --body '...' --format json

# Step 6：创建 4 个 Milestone（开题/中期/初稿/终稿）
gitlink-cli api POST /v1/:owner/:repo/milestones --body '...' --format json

# Step 7：添加学生为协作者
gitlink-cli api POST /api/:owner/:repo/collaborators --body "{\"user_id\":\"<member_user_id>\"}" --format json

# Step 8：确保标签存在（开题/中期/初稿/终稿/已归档）
gitlink-cli api GET /v1/:owner/:repo/issue_tags --format json

# Step 9：输出创建报告 + 仓库链接列表
```

### 7.2 查看进度看板

```bash
# Step 1：获取所有毕设仓库列表
gitlink-cli api GET /v1/:owner/projects --format json

# Step 2：对每个仓库采集进度
gitlink-cli milestone +list --owner <owner> --repo <repo> --format json
gitlink-cli api GET /:owner/:repo/sub_entries --query 'filepath=docs&ref=master' --format json
gitlink-cli api GET /v1/:owner/:repo/commits --query 'sha=master&limit=1' --format json

# Step 3：AI 判定每个学生各阶段状态
# Step 4：输出导师进度看板
# Step 5：标记需关注学生
```

### 7.3 格式校验

```bash
# Step 1：获取论文文件列表
gitlink-cli api GET /:owner/:repo/sub_entries --query 'filepath=docs/final&ref=master' --format json
gitlink-cli api GET /:owner/:repo/sub_entries --query 'filepath=slides&ref=master' --format json

# Step 2：AI 校验文件格式/命名/大小/目录
# Step 3：对不合规项创建 Issue 反馈
gitlink-cli issue +create --owner <owner> --repo <repo> \
  --title "【格式检查】..." --body "<校验报告>" --format json
```

### 7.4 毕业归档

```bash
# Step 1：检查所有学生答辩材料完整性
gitlink-cli api GET /:owner/:repo/sub_entries --query 'filepath=slides&ref=master' --format json
gitlink-cli api GET /:owner/:repo/sub_entries --query 'filepath=docs/defense&ref=master' --format json

# Step 2：移除学生写权限（设置角色为报告者）
gitlink-cli api PUT /api/:owner/:repo/collaborators/change_role --body "{\"user_id\":\"<member_user_id>\",\"role\":\"Reporter\"}" --format json

# Step 3：创建归档 Issue + 打"已归档"标签
gitlink-cli issue +create --owner <owner> --repo <repo> \
  --title "【归档】..." --body "<归档信息...>
  <归档信息...>
  " --format json
gitlink-cli api PATCH /v1/:owner/:repo/issues/:id --body '{"issue_tag_ids":[<tag_id>]}' --format json

# Step 4：输出归档报告
```

---

## 八、可配置参数

| 参数 | 默认值 | 说明 |
|------|--------|------|
| `repo_naming` | graduation-<student_id> | 仓库命名规则 |
| `org_name` | — | 组织名称（必填） |
| `milestone_dates` | 见阶段表 | 各阶段截止日期 |
| `create_milestones` | true | 是否自动创建 Milestone |
| `init_directories` | true | 是否初始化标准化目录 |
| `add_collaborator` | true | 是否自动添加学生为协作者 |
| `format_check_enabled` | true | 是否启用格式校验 |
| `archive_after_defense` | true | 答辩后是否自动归档 |
| `min_thesis_size_mb` | 1 | 论文最小文件大小（MB） |
| `max_thesis_size_mb` | 20 | 论文最大文件大小（MB） |

---

## 九、常见场景示例

### 场景 A：新学期批量创建毕设仓库

```
用户："给 cs-grad-2024 组织创建 8 个毕设仓库，学生清单在 students.csv"

AI 执行：
1. 获取组织 ID
2. 读取学生清单
3. 干运行预览（8 个仓库的创建清单）
4. 用户确认后批量创建
5. 初始化标准化目录 + 创建 Milestone + 添加协作者
6. 输出创建报告 + 仓库链接
```

### 场景 B：查看所有学生进度

```
用户："看看所有学生的毕设进度"

AI 执行：
1. 获取所有毕设仓库列表
2. 逐个采集进度数据（Milestone + 文件 + 最近提交）
3. 输出导师进度看板
4. 标记需关注学生（超时/落后）
```

### 场景 C：论文格式校验

```
用户："检查所有学生的论文终稿格式"

AI 执行：
1. 获取所有仓库的 /docs/final/ 和 /slides/ 目录
2. 逐个校验格式/命名/大小/目录
3. 对不合规项创建 Issue 反馈
4. 输出校验汇总报告
```

### 场景 D：毕业后归档

```
用户："答辩结束了，归档所有毕设仓库"

AI 执行：
1. 检查答辩材料完整性
2. 移除学生写权限
3. 创建归档 Issue + 打标签
4. 输出归档报告 + 成绩分布
```

---

## 十、注意事项

- ✅ **标签预创建**：打标签前必须先查询标签列表，确认目标标签存在，不存在则先通过 `POST /v1/:owner/:repo/issue_tags` 创建
- ✅ **issue_tag_ids 完整替换**：`PATCH /v1/:owner/:repo/issues/:id` 的 `issue_tag_ids` 是完整替换，需包含已有标签 ID
- ⚠️ **权限要求**：当前登录用户必须有在组织下创建仓库的权限
- ⚠️ **批量限制**：建议每次不超过 50 个仓库，避免频率限制链接
- ⚠️ **报错重试**：由于请求数量过大，可能偶发性出现接口报错，建议重试最多3次

---
