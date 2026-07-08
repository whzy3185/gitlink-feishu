# workflow 健康评分列表筛选修正

`workflow +health`、`+repo-report`、`+triage` 的远程抓取此前用 `state=open` 过滤 Issue/PR 列表，但 GitLink v1 列表接口对 PR 用 `status`（0 开启 / 1 合并 / 2 关闭），对 Issue 用 `category`（opened / closed / all），`state` 会被服务端静默忽略并返回全部状态。结果 open+merged+closed 全被计入，`OpenPRs`/`OpenIssues` 与由此推导的健康分被污染。

现在 PR 抓取发送 `status=0`，Issue 抓取发送 `category=opened`，与 `shortcuts/health`、`shortcuts/pr`、`shortcuts/issue` 已有的映射保持一致；`+triage` 的本地态客户端二次过滤保持不变。

同时修正 `updateRecentActivity`：`RecentActivityDays==0` 表示"今天有活动"（`apiAgeInDays` 对 24 小时内返回 0），旧逻辑里 `|| input.RecentActivityDays == 0` 会让更早的信号覆盖"今天"，使仓库显得更陈旧。移除该项后，首次赋值仍由 `!RecentActivityKnown` 分支处理，之后仅当发现更近的活动才更新。

测试断言列表请求发出的键为 `status`/`category`（不再是 `state`），并验证"今天"的活动信号不会被更早的信号覆盖。
