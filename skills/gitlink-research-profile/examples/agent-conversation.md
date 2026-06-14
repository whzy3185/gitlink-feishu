# 示例：在 AI Agent 中使用 gitlink-research-profile

下面是一段在 Claude Code 中调用本 Skill 的真实交互示意（命令均为只读）。

---

## 场景 A：单个科研主体画像

**用户：** 帮我看看 GitLink 上 yystopf 这个人的科研画像。

**Agent：**（读取 `gitlink-research-profile/SKILL.md` 与 `gitlink-shared/SKILL.md`）
先确认用户资料，再采集五类画像数据：

```bash
gitlink-cli user +info --login yystopf --format json
gitlink-cli profile +ability      --user yystopf --format json
gitlink-cli profile +major        --user yystopf --format json
gitlink-cli profile +role         --user yystopf --format json
gitlink-cli profile +activity     --user yystopf --format json
gitlink-cli profile +contribution --user yystopf --format json
```

**Agent（产出）：** 生成《科研主体画像：何慧（@yystopf）》报告——
能力雷达（贡献/经验/活跃均达卓越）、主力语言 Go/Java、并**识别出该账号含 11453 个镜像项目**，
据此标注其学科（22 个）与角色（Owner 2 万）为镜像虚高、给出修正解读。
完整样例见 [`researcher-profile-report.md`](researcher-profile-report.md)。

---

## 场景 B：跨学科协作匹配

**用户：** 我在做"联邦学习 + 医疗影像"的课题，想在 GitLink 上找能补强的协作者。

**Agent：** 先按方向找候选，再对候选取画像并按"学科互补 + 能力互补"排序：

```bash
# 1) 候选发现（多关键词）
gitlink-cli search +users -k "联邦学习" --format json
gitlink-cli search +users -k "federated" --format json
gitlink-cli search +users -k "医疗影像" --format json

# 2) 对每个候选取学科与能力（示例一位候选）
gitlink-cli profile +major   --user <candidate> --format json
gitlink-cli profile +ability --user <candidate> --format json
```

**Agent（产出）：** 输出协作匹配表——

| 候选 | 匹配度 | 学科互补 | 能力互补 | 推荐理由 |
|------|--------|----------|----------|----------|
| @alice | 高 | 覆盖"生物医药健康"空白 | 经验 88 补强 | 医疗方向 + 高经验，契合影像子课题 |
| @bob | 中 | 覆盖"人工智能/大数据" | 语言 85（Python） | 联邦学习实现栈匹配 |

并说明：候选的学科 `categories` 覆盖了课题缺口、能力维度补强了团队短板。

---

## 要点

- 全程只读，不修改任何数据，无需额外确认。
- 命令均使用 `--format json` 以便 Agent 解析。
- 对镜像/管理员账号，Agent 会结合 `user +info.mirror_projects_count` 自动标注"指标虚高"，避免误读。
