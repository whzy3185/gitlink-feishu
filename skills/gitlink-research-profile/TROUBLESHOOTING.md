# gitlink-research-profile — 故障排查

**CRITICAL — 开始前请先阅读 [`../gitlink-shared/SKILL.md`](../gitlink-shared/SKILL.md)（认证、全局参数）与 [`./SKILL.md`](./SKILL.md)（工作流）、[`./REFERENCE.md`](./REFERENCE.md)（字段参考）。**
**CRITICAL — 本 Skill 全部只读，不会修改任何数据。**
**CRITICAL — GitLink 资源只能用 `gitlink-cli` 操作，禁止用 `gh`/`glab`。**

本文档列出使用科研主体画像 Skill 时的常见问题，按「症状 / 原因 / 解决」三段式给出。

## 速查表

| # | 症状 | 根因 | 一句话解决 |
|---|------|------|-----------|
| 1 | `unknown command "profile"` | `feat/profile-shortcuts` 分支未合入 | 用 Raw API 降级，或等 profile 命令合入 |
| 2 | 某维度返回空（`categories: []`、活动全 0） | 用户数据稀疏或新注册 | 标注"数据稀疏"，不臆造 |
| 3 | Owner 数 / 学科数量异常巨大 | 镜像聚合账号导致虚高 | 结合 `user +info.mirror_projects_count` 标注 |
| 4 | `404` 用户不存在 | login 拼写错误或账号已注销 | 用 `search +users` 确认 login |
| 5 | 团队画像逐人调用超时 | 成员过多串行请求触发限流 | 控制人数 ≤10，串行逐人，间隔 5 秒 |
| 6 | `dataset +list` 命令不存在 | `feat/dataset-shortcuts` 分支未合入 | 该辅助命令非必须，可跳过 |
| 7 | `--start-time/--end-time` 无效 | API 不支持时间窗口过滤 | 忽略该参数，使用平台默认区间 |

---

## 1. `profile` 命令不存在

**症状**：运行 `gitlink-cli profile +ability --user xxx` 报 `unknown command "profile" for "gitlink-cli"`。

**原因**：`profile` 快捷命令在 `feat/profile-shortcuts` 分支中实现，尚未合入当前版本。

**解决**：

```bash
# 方案 A：使用 Raw API 降级（立即可用）
gitlink-cli api GET /users/<login>/statistics/develop.json
gitlink-cli api GET /users/<login>/statistics/major.json
gitlink-cli api GET /users/<login>/statistics/role.json
gitlink-cli api GET /users/<login>/statistics/activity.json
gitlink-cli api GET /users/<login>/headmaps.json

# 方案 B：等待 profile-shortcuts PR 合入后再使用
```

> Raw API 返回字段与 `profile` 命令完全一致，仅调用方式不同。

---

## 2. 某维度返回空数据

**症状**：`+major` 返回 `categories: []`，或 `+activity` 返回的 `commits_count`/`issues_count`/`pull_requests_count` 全为 0。

**原因**：
- 用户注册后尚未参与项目或提交代码
- 近一周无任何活动
- 平台统计接口对该用户无数据

**解决**：
- 在画像报告中标注"该维度数据稀疏"
- 不要臆造或补零，保持数据真实性
- 建议用户查看 `user +info` 的 `created_time` 判断是否为新用户

---

## 3. Owner 数 / 学科数量异常巨大

**症状**：`+role` 返回 `owner.count` 上万，或 `+major` 返回 20+ 个学科方向。

**原因**：该账号为平台管理型账号或持有大量镜像项目，镜像会自动计入 owner 角色和异质学科。

**解决**：

```bash
# 确认是否为镜像聚合账号
gitlink-cli user +info --login <login> --format json
# 关注 mirror_projects_count 字段
```

- `mirror_projects_count` 远大于 `common_projects_count` → 镜像聚合账号
- 在画像中标注"含镜像聚合，角色/方向指标虚高"
- 真实规模应以 `common_projects_count`（原创项目数）为参照

---

## 4. 404 用户不存在

**症状**：`profile +ability --user xxx` 返回 404。

**原因**：
- login 拼写错误（区分大小写）
- 账号已注销
- 给了昵称而非 login

**解决**：

```bash
# 通过搜索确认 login
gitlink-cli search +users -k "<姓名或关键词>" --format json

# 或直接读取用户资料验证
gitlink-cli user +info --login <login> --format json
```

> `profile` 命令的 `--user` 必须是 **login**（用户标识），不是昵称。

---

## 5. 团队画像逐人调用超时

**症状**：对 10+ 人团队逐人采集时，中途出现网络超时或 TLS 错误。

**原因**：串行逐人调用 5 个 API，总请求数 = 人数 × 5，并发或高频可能触发平台限流。

**解决**：
- 优先核心成员（≤10 人）
- 串行逐人调用，避免并发
- 每人之间间隔约 5 秒
- 对失败的用户标注"数据获取失败"，跳过继续
- 最少可只调用 `+ability` 和 `+major`，控制调用量

---

## 6. `dataset +list` 命令不存在

**症状**：REFERENCE.md 中引用了 `gitlink-cli dataset +list`，但 CLI 报 `unknown command "dataset"`。

**原因**：`dataset` 快捷命令在 `feat/dataset-shortcuts` 分支中实现，尚未合入。

**解决**：`dataset +list` 仅用于查询科研数据集的论文/许可证等元数据，**非画像必需**。可跳过或使用 Raw API 降级：

```bash
gitlink-cli api GET /v1/project_datasets.json?ids=<projectId>
```

---

## 7. `--start-time/--end-time` 参数无效

**症状**：传了 `--start-time` / `--end-time`，但返回数据与不传时相同。

**原因**：GitLink 平台 `/statistics/*` 接口当前可能忽略 `start_time`/`end_time` 查询参数，或仅对部分用户有数据。

**解决**：
- 这是平台侧限制，非 CLI 问题
- 忽略时间窗口参数，使用平台默认统计区间
- 如需特定时间段数据，可在画像报告中手动标注"数据为平台全量统计，非限定区间"

---

## 调试技巧

### 启用调试输出

```bash
gitlink-cli profile +ability --user <login> --debug
```

### 验证 API 连通性

```bash
# 最简单的验证（不需要 profile 命令）
gitlink-cli api GET /users/<login>/statistics/develop.json

# 验证用户是否存在
gitlink-cli user +info --login <login>
```

### 检查认证状态

```bash
gitlink-cli auth status
```
