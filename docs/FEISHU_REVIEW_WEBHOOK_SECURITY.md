# GitLink Review Webhook 安全策略

## 入口

```text
POST /webhooks/gitlink/{installation_id}
```

路径只选择一个 Installation 和一个 Secret，不遍历所有 Secret。Handler 顺序为 Method、Content-Type、Installation、1 MiB Body 上限、Timestamp、HMAC、JSON、Delivery、标准化、Inbox 原子插入、唤醒 Processor、返回 202。Handler 不查询 GitLink、不访问飞书、不遍历 Subscription、不创建 Job。

## 签名与时间戳

- `body_sha256`：`HMAC-SHA256(secret, body)`。
- `timestamp_body_sha256`：`HMAC-SHA256(secret, timestamp + "." + body)`。
- 比较使用 `hmac.Equal`。
- Timestamp 支持 Unix 秒、RFC3339 和 RFC3339Nano。
- Timestamp Mode 支持 `required`、`optional`、`disabled`，默认 optional。
- 最大时钟偏差默认 300 秒，允许 30—600 秒。

兼容读取 GitLink、Gitea、GitHub 风格 Header，但优先 GitLink Header；兼容路径不代表相应平台 Payload 已完成真实验收。

Delivery Required 开启时缺少 Header 返回 400；关闭时使用 Installation + Body Fingerprint 的哈希作为 fallback。相同 Installation + Delivery 返回 202 duplicate，不覆盖原 Event。

## 当前验收边界

离线合同测试覆盖签名、时间戳、Body 上限、重复 Delivery、fallback、Fast ACK 与 Inbox-first。尚未完成真实 GitLink Webhook 平台验证，也不宣称强实时或绝不丢事件。
