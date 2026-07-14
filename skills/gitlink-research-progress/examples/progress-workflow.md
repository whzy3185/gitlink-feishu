# 示例：为课题组科研仓库生成进度周报与风险预警

本示例演示用 `gitlink-research-progress` 技能跟踪一个科研仓库的进度，并产出《周报 + 风险预警》。命令均为 gitlink-cli 真实命令，可复现。

> 实际使用时把 `someorg/algo-core` 替换为目标仓库；数值以实际采集为准。

---

## 一、Agent 对话片段

```
用户：帮我看看 someorg/algo-core 这周进展怎么样，有没有要预警的？

Agent：好的，我用 gitlink-research-progress 技能采集近 8 周的提交/Issue/里程碑/PR/贡献数据，
       打风险灯并生成周报，全程只读。开始……
```

---

## 二、采集命令序列

```bash
gitlink-cli repo +info --owner someorg --repo algo-core --format json

# 提交节奏
gitlink-cli api GET "/someorg/algo-core/commits?page=1&limit=50" --format json

# Issue 进度
gitlink-cli issue +list --owner someorg --repo algo-core --state open   --format json
gitlink-cli issue +list --owner someorg --repo algo-core --state closed --format json

# 里程碑
gitlink-cli milestone +list --owner someorg --repo algo-core --format json

# PR 吞吐
gitlink-cli pr +list --owner someorg --repo algo-core --state open   --format json
gitlink-cli pr +list --owner someorg --repo algo-core --state merged --format json

# 团队 / Bus Factor
gitlink-cli repo +contributors      --owner someorg --repo algo-core --format json
gitlink-cli repo +contributor-stats --owner someorg --repo algo-core --format json
```

---

## 三、信号归集（Agent 内部记录）

| 信号 | 采集结果 | 灯 |
|------|----------|----|
| 提交节奏 | 近 2 周 1 次提交，最近提交 11 天前，环比 ↓78% | 🔴 |
| Issue | 开放 14 / 已闭 36，近 4 周净增 +3，5 个 >30 天未动 | 🟡 |
| 里程碑 | M2 完成率 55%，距到期 10 天 | 🟡 |
| PR | 开放 2（最久 9 天），近 4 周合并 5 | 🟢 |
| Bus Factor | 活跃 3 人，Top1 占比 64% | 🟡 |

→ **项目总灯：🔴 红灯。** 依据 SKILL.md 第三节规则「任一信号为 🔴 即红灯」：提交节奏命中「近 2 周提交为 0~1 次」判 🔴，故总灯为红。

---

## 四、生成的周报（样例输出）

```markdown
## 📈 科研进度周报与风险预警

**项目：** someorg/algo-core
**统计周期：** 2026-06-09 ~ 2026-06-15（近 8 周趋势）
**项目总灯：** 🔴 红灯（高风险）

### 一、一句话结论
> 主仓库近 2 周近乎停滞（最近提交 11 天前），M2 里程碑有延期苗头，存在单点依赖，建议本周组会重点处理。

### 二、五维信号面板
| 信号 | 指标 | 灯 |
|------|------|----|
| 提交节奏 | 近 2 周 1 次，环比 ↓78%，最近 11 天前 | 🔴 |
| Issue 进度 | 开放 14 / 已闭 36，近 4 周净增 +3 | 🟡 |
| 里程碑 | M2 完成率 55%，距到期 10 天 | 🟡 |
| PR 吞吐 | 开放 2（最久 9 天），近 4 周合并 5 | 🟢 |
| 团队 / Bus Factor | 活跃 3 人，Top1 占比 64% | 🟡 |

### 三、提交节奏趋势
近 8 周周提交量：
W-7 ████████        12
W-6 ██████          9
W-5 █████████       14
W-4 ███             4
W-3 ██              3
W-2 █               1
W-1 ▏               0   ← 🔴
W-0 ▏               1

### 四、风险预警清单
| 等级 | 风险 | 建议 |
|------|------|------|
| 🔴 | 主仓库近 2 周停滞 | 组会确认卡点/人力 |
| 🟡 | M2 延期苗头（55%/10天） | 砍范围或顺延 |
| 🟡 | 单点依赖（Top1 64%） | 安排第二负责人 |
| 🟡 | 5 个陈旧 Issue | 分诊：关闭/重排 |

### 五、本周建议动作（Top 3）
1. 排查主仓库停滞原因，必要时拆任务。
2. 重新评估 M2 范围与到期日。
3. 指定核心模块第二负责人，降低 Bus Factor。
```

---

## 五、可选：定期化

把上述命令封装为脚本，配合外部定时器（cron / CI 定时任务）每周一运行，即可得到可纵向对比的进度趋势序列。
