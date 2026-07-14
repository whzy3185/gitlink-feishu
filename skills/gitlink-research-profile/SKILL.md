---
name: gitlink-research-profile
version: 1.0.0
description: "科研主体画像：基于 GitLink 平台原生统计接口，为科研工作者/课题组生成开发能力雷达、学科领域分布、角色定位、活跃节奏与语言栈画像，并支持团队画像与跨学科协作匹配。当用户需要了解某位研究者/团队的能力与方向、做人才评估、组队或寻找协作伙伴时触发。"
metadata:
  requires:
    bins: ["gitlink-cli"]
  cliHelp: "gitlink-cli profile --help"
---

# gitlink-research-profile（科研主体画像）

**CRITICAL — 开始前必须先阅读 [`../gitlink-shared/SKILL.md`](../gitlink-shared/SKILL.md)，其中包含认证、权限处理和 API 注意事项。**
**CRITICAL — 本 Skill 为只读操作，不会修改任何仓库或用户数据。无需用户额外确认即可执行。**
**CRITICAL — GitLink 操作只能用 `gitlink-cli`。禁止用 `gh`（GitHub CLI）操作 GitLink 资源。**

> **前置条件：** 先阅读 [`../gitlink-shared/SKILL.md`](../gitlink-shared/SKILL.md) 了解认证和全局参数。
> **依赖命令：** 本 Skill 依赖 `gitlink-cli profile`（开发能力/学科/角色/活动/贡献统计）。如提示命令不存在，请升级 gitlink-cli 至包含 `profile` 命令组的版本。

---

## 功能概述

面向**科研工作者、课题组、科研团队**的主体画像工具。不同于以"项目/主题"为中心的
[`gitlink-research-tracker`](../gitlink-research-tracker/SKILL.md)（技术调研）和以"PR 时间戳推算活跃度"
为主的 [`gitlink-contributor-insight`](../gitlink-contributor-insight/SKILL.md)，本 Skill 以**人/团队为中心**，
直接消费 GitLink **平台原生画像接口**（`/api/users/{owner}/statistics/*`），刻画其科研能力与方向：

1. **开发能力雷达** — 影响力 / 贡献度 / 活跃度 / 项目经验 / 语言能力 五维评分（含平台基线对照）
2. **学科领域画像** — 平台推断的研究方向标签（如人工智能、大数据、生物医药健康、天文地球物理…）
3. **角色定位** — 在所参与项目中作为 owner / manager / developer / reporter 的分布
4. **活跃节奏** — 近一周提交/疑修/合并请求趋势 + 一年贡献热力图与总量
5. **语言技术栈** — 各语言占比与单语言能力分
6. **团队画像与协作匹配** — 多人横向对比、学科互补度分析、组队/找人建议

---

## 适用场景

| 场景 | 说明 |
|------|------|
| 科研人才评估 | 评估候选人/合作者的能力结构、研究方向与活跃度 |
| 课题组盘点 | 汇总团队成员能力与学科覆盖，识别短板与重叠 |
| 跨学科协作匹配 | 按学科互补 + 能力互补，为某研究方向推荐协作伙伴 |
| 新成员定位 | 快速了解新人的技术栈与擅长方向，合理分工 |
| 个人科研复盘 | 研究者自查能力画像与活跃趋势，规划成长 |

---

## 工作流 1：单个科研主体画像

### Step 1：确认目标用户标识（login）

用户给出的可能是姓名或登录名。`profile` 系列命令需要 **login**（用户标识，非昵称）。
如不确定，先搜索或读取用户资料确认：

```bash
# 通过关键词搜索用户，确认 login
gitlink-cli search +users -k "<姓名或关键词>" --format json

# 读取用户资料（确认 login、机构、地区等）
gitlink-cli user +info --login <login> --format json
```

`user +info` 关键字段：`name`/`real_name`（姓名）、`custom_department`（所属机构/院校）、
`city`/`province`（地区）、`common_projects_count`（原创项目数）、`created_time`（注册时间）。

### Step 2：采集五类画像数据

对目标 `<login>` 依次执行（全部只读）：

```bash
gitlink-cli profile +ability      --user <login> --format json   # 开发能力 + 语言栈
gitlink-cli profile +major        --user <login> --format json   # 学科领域标签
gitlink-cli profile +role         --user <login> --format json   # 角色分布
gitlink-cli profile +activity     --user <login> --format json   # 近一周活跃
gitlink-cli profile +contribution --user <login> --format json   # 一年贡献热力图
```

> `+ability/+major/+role` 支持 `--start-time/--end-time`（Unix 时间戳）限定区间；
> `+contribution` 支持 `--year`。不带则为平台默认窗口。

### Step 3：字段解析

详见 [REFERENCE.md](REFERENCE.md)。核心映射：

| 画像维度 | 命令 | 字段 | 解读 |
|----------|------|------|------|
| 能力五维（用户） | `+ability` | `data.user.{influence,contribution,activity,experience,language}` | 0–100，越高越强 |
| 能力五维（平台基线） | `+ability` | `data.platform.{...}` | 平台 Top 水平参照，用于对照而非简单比高低 |
| 语言占比 | `+ability` | `data.user.languages_percent{}` | 各语言代码量占比（0–1） |
| 单语言能力分 | `+ability` | `data.user.each_language_score{}` | 各语言 0–100 |
| 学科领域 | `+major` | `data.categories[]` | 平台推断的研究/技术方向标签 |
| 角色分布 | `+role` | `data.role.{owner,manager,developer,reporter}.{count,percent}` | 项目参与角色构成 |
| 总项目数 | `+role` | `data.total_projects_count` | 参与项目总量 |
| 近周活跃 | `+activity` | `data.dates[]` 对齐 `commits_count[]/issues_count[]/pull_requests_count[]` | 三条等长时间序列 |
| 贡献热力 | `+contribution` | `data.total_contributions`、`data.headmaps[].{date,contributions}` | 年度贡献总量与每日分布 |

### Step 4：能力分级与解读规则

| 维度分值 | 等级 | 解读 |
|----------|------|------|
| ≥90 | 卓越 | 平台头部水平 |
| 75–89 | 优秀 | 明显高于平均 |
| 60–74 | 良好 | 稳定贡献者 |
| 40–59 | 一般 | 参与度有限 |
| <40 | 较弱 | 数据稀疏或新用户 |

- **语言栈**：取 `languages_percent` 降序前 3–5 种作为"主力语言"，结合 `each_language_score` 标注熟练度。
- **学科聚焦度**：`categories` 数量少而集中 → 方向专精；数量多而分散 → 通才/平台维护者（需结合
  `user +info` 的 `mirror_projects_count` 判断是否为镜像聚合导致的虚高）。
- **角色画像**：`owner` 占比高 → 主导者/项目发起人；`developer`/`reporter` 占比高 → 参与贡献型。

### Step 5：生成单主体画像报告

使用下方[输出模板](#输出模板单主体)。示例见
[`examples/researcher-profile-report.md`](examples/researcher-profile-report.md)。

---

## 工作流 2：团队画像与协作匹配

### Step 1：确定成员名单

用户给出一组 `login`（课题组成员），或给出研究方向由你先 `search +users` 找候选。

### Step 2：逐人采集

对每个成员执行工作流 1 的 Step 2（建议每人至少 `+ability` 和 `+major`，控制调用量）。

### Step 3：团队聚合分析

- **能力矩阵**：成员 × 能力五维的表格，算团队均值与最强项/短板。
- **学科覆盖**：合并所有成员 `categories`，统计覆盖的学科与重叠度（多人共有 vs 独有方向）。
- **语言栈覆盖**：合并 `languages_percent` 主力语言，识别团队技术栈与缺口。
- **角色结构**：统计 owner/manager/developer 分布，判断团队是"多主导"还是"主导+执行"。

### Step 4：协作匹配（可选）

给定一个目标研究方向或一位核心研究者：

1. 候选发现：`search +users -k <方向关键词>`，对候选取 `+major` 和 `+ability`。
2. **学科互补度**：候选的 `categories` 与目标方向/团队缺口的契合度（覆盖空白方向加分）。
3. **能力互补度**：候选在团队短板维度（如 `language`/`experience`）上的得分（补强加分）。
4. 输出推荐列表，每位候选给出匹配理由（覆盖了哪个学科空白 / 补强了哪项能力）。

### Step 5：生成团队/匹配报告

使用[团队输出模板](#输出模板团队与匹配)。

---

## 输出模板（单主体）

```markdown
# 🔬 科研主体画像：{{name}}（@{{login}}）

> 生成时间：{{now}} ｜ 数据来源：GitLink 平台统计接口（`gitlink-cli profile`）
> 机构：{{custom_department}} ｜ 地区：{{province}}{{city}} ｜ 注册：{{created_time}}

## 一、综合摘要
{{2–3 句话总结：擅长方向、主力语言、能力亮点与活跃状态}}

## 二、开发能力雷达（0–100）

| 维度 | 用户 | 平台基线 | 等级 |
|------|------|----------|------|
| 影响力 influence | {{u.influence}} | {{p.influence}} | {{等级}} |
| 贡献度 contribution | {{u.contribution}} | {{p.contribution}} | {{等级}} |
| 活跃度 activity | {{u.activity}} | {{p.activity}} | {{等级}} |
| 项目经验 experience | {{u.experience}} | {{p.experience}} | {{等级}} |
| 语言能力 language | {{u.language}} | {{p.language}} | {{等级}} |

## 三、学科领域
{{categories 列表，聚焦方向加粗；附"方向聚焦度"判断}}

## 四、角色定位
参与项目 {{total_projects_count}} 个 — Owner {{owner.count}}（{{owner.percent}}）/ Manager {{...}} / Developer {{...}} / Reporter {{...}}
{{一句话角色画像}}

## 五、语言技术栈

| 语言 | 占比 | 能力分 |
|------|------|--------|
| {{lang}} | {{percent}} | {{score}} |
（取前 5）

## 六、活跃节奏
- 近一周：提交 {{sum_commits}}、疑修 {{sum_issues}}、PR {{sum_prs}}
- 年度贡献总量：{{total_contributions}}；活跃峰值日：{{peak_date}}（{{peak_value}}）

## 七、画像结论与建议
{{适合的角色/方向、可补强的能力、协作建议}}
```

## 输出模板（团队与匹配）

```markdown
# 🔬 课题组科研画像：{{team_name}}（{{n}} 人）

## 一、能力矩阵（0–100）
| 成员 | 影响力 | 贡献度 | 活跃度 | 经验 | 语言 |
|------|--------|--------|--------|------|------|
| {{login}} | ... | ... | ... | ... | ... |
| **团队均值** | ... | ... | ... | ... | ... |

## 二、学科覆盖
- 共有方向：{{多人共有}}
- 独有方向：{{各自独有}}
- 覆盖空白：{{团队未覆盖但相关的方向}}

## 三、技术栈与角色结构
{{主力语言覆盖与缺口；owner/developer 结构}}

## 四、协作匹配推荐（如适用）
| 候选 | 匹配度 | 学科互补 | 能力互补 | 推荐理由 |
|------|--------|----------|----------|----------|
| @{{login}} | 高/中 | {{补的方向}} | {{补的能力}} | {{一句话}} |

## 五、结论
{{团队优势、短板、补人/分工建议}}
```

---

## 异常场景处理

| 场景 | 处理方式 |
|------|----------|
| 用户给的是姓名不是 login | 先 `search +users -k` 或 `user +info` 确认 login，再调用 profile |
| `profile` 命令不存在 | 提示升级 gitlink-cli 到含 `profile` 命令组的版本；或临时用 `gitlink-cli api GET /users/<login>/statistics/develop` 等价获取 |
| 某接口返回空（如 `categories: []`、活动全 0） | 标注"该维度数据稀疏"，不臆造；说明可能是新用户或近期不活跃 |
| 用户 `owner.count`/学科数量异常巨大 | 结合 `user +info` 的 `mirror_projects_count` 判断是否为平台管理员/镜像聚合账号，画像中标注"含镜像聚合，方向分布偏泛" |
| 404 用户不存在 | 确认 login 拼写（区分大小写），或该账号已注销 |
| 团队成员过多（>10） | 优先核心成员；逐人串行调用，避免并发触发限流/TLS 超时 |
| 网络超时/TLS 错误 | 等待 5 秒重试一次；仍失败则跳过该用户并在报告中标注"数据获取失败" |

---

## 注意事项

- ✅ **全部命令使用 `--format json`**，便于解析。
- ✅ **本 Skill 为纯只读画像**，不修改任何数据，无需确认即可执行。
- ✅ **平台基线对照**：`platform.*` 是平台头部参照值（常接近满分），用于说明"距头部的差距"，不要简单地把 user 与 platform 相减当作排名。
- ⚠️ **login ≠ 昵称**：`profile` 命令的 `--user` 必须是 login（用户标识）。
- ⚠️ **镜像/管理员账号会让 `role`/`major` 虚高**：参与项目数上万、学科 20+ 往往是平台账号或大量镜像导致，需结合 `user +info` 的 `mirror_projects_count` 标注，避免误读为"全能科研牛人"。
- ⚠️ **数据为 GitLink 平台内画像**，反映其在 GitLink 上的行为，不代表其在其他平台或线下的全部科研产出。
- ⚠️ **`+activity` 三个计数数组与 `dates` 等长且按下标对齐**，解读时务必按同一下标取值。
```
