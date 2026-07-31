# 离线集成契约样例

这三个 JSON 文件是复赛策划阶段的离线契约尝试：

1. `collab-event.sample.json`：平台无关的 GitLink 协作事件。
2. `feishu-card.sample.json`：由同一事件渲染的飞书卡片草案。
3. `wecom-template-card.sample.json`：由同一事件渲染的企业微信模板卡片草案。

限制：

- 没有调用任何平台 API。
- 没有包含 webhook、token、app secret、open_id 或 userid。
- payload 只作为 renderer 契约和 mock 测试输入，尚未经过企业微信真实环境 smoke。
- 示例为群级 owner attention，不是个人定向通知。
- production 实现必须根据各平台最新官方 schema 做 contract test。

共同验收：

```text
event_id 一致
GitLink repository/source_url 一致
2 个高风险 Issue + 1 个高风险 PR 一致
CI unknown 一致
只提供 GitLink 跳转，不触发 GitLink 写操作
```
