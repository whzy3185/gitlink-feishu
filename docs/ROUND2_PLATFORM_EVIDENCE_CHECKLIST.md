# GitLink 协作平台真实验收与证据清单

日期：2026-08-01

本清单只用于测试租户、测试群和专用 GitLink 测试 PR。任何真实写入必须事先明确目标、操作者和回滚方式。证据只保存哈希、状态、时间、commit SHA 和请求类型，不保存 Secret、Token、Cookie、Webhook key、消息正文或原始用户 ID。

## 0. 通用证据头

每个场景保存一份 JSON 或 Markdown 记录：

```text
scenario_id
tested_at
commit_sha
configuration_revision
operator_id_hash
installation_id
repository
pr_number
before_state
action
after_state
remote_id_hash
gitlink_mutation_status
result
notes
```

## 1. 多仓库飞书查询

1. 一个测试群绑定两个测试仓库，不设置默认仓库。
2. 发送未限定仓库的 `查看 PR #编号`，应返回需要选择仓库且不创建 Job。
3. 分别发送两个限定仓库命令，均应读取正确仓库。
4. 请求第三个未授权仓库，应受控拒绝。
5. 重复同一 message_id，不得产生第二个 Job 或最终回复。

保存：message_id 哈希、Job ID、解析后的 installation/repository、HTTP GET 计数、GitLink 写入状态 `none`。

## 2. 固定卡片

1. 首次同步一个 WorkItem，记录飞书 message_id 哈希。
2. 不改变 WorkItem 再同步，应为 skipped。
3. 改变 Review 事实后同步，应 PATCH 同一 message_id。
4. 领取和修改截止日期后同步，应仍为同一 message_id。

禁止以“重新发送一张新卡片”代替 PATCH 验收。

## 3. Base

1. 在专用 Base 首次同步一个 PR，记录 record_id 哈希。
2. 重复同步，不得创建第二条 `pr_key` 业务记录。
3. 人工修改负责人、协作状态和截止日期。
4. GitLink 事实变化后同步，人工字段必须保留。
5. 制造两条相同 `pr_key`，同步必须拒绝并提示对账。
6. 删除远端记录后同步，记录恢复策略与新 remote_id 哈希。

## 4. Doc

1. 使用测试文件夹为一个 PR 创建文档，记录 document_id 哈希。
2. 相同指纹重复同步，不追加内容。
3. 新 patchset 后追加一次快照。
4. 重启 Gateway 后再次同步，仍使用同一 document_id。
5. 删除或撤销文档权限时，应进入失败/对账状态，不新建同名文档掩盖问题。

## 5. Task

1. 首次创建 Task，记录 client_token 哈希和 Task GUID 哈希。
2. 立即重复请求，不得创建第二个 Task。
3. 修改负责人和全天截止日期，更新原 GUID。
4. PR merged/closed 后完成原 Task。
5. 模拟远端成功但本地保存失败，状态必须为 unknown/possible，禁止自动重试。

## 6. GitLink Webhook

1. 使用测试 Secret 发送签名正确的 PR 更新事件。
2. 记录 delivery ID 哈希、仓库、PR、路由群和 Job ID。
3. 重放同一 delivery，不得创建第二个 Job。
4. 错误签名、超大 body、未授权仓库均应拒绝。
5. 经 TLS 反向代理再做一次真实投递，确认原始 body 和签名头未被改写。

## 7. 受控 common Review

1. 只使用专用测试 PR 和测试账号。
2. 保存写前 Review 列表摘要、head SHA 和 SourceFingerprint。
3. 生成 ActionPlan，确认一次。
4. 保存写后 Review ID 哈希、Review 列表摘要和 mutation status `confirmed`。
5. 分别改变 head、fingerprint 和身份绑定，确认 POST 计数为 0。
6. 在可控代理中制造网络不确定结果，确认状态为 `possible` 且不会自动重试。

批准、拒绝、行级评论、Reviewer 管理和合并不属于本验收。

## 8. Agent

1. 固定一个完整 Context 和 Agent plan。
2. 运行 correctness、tests、security 等角色。
3. 保存 endpoint 主机哈希、task ID、耗时和 assessment 摘要，不保存 bearer token。
4. 让一个 Agent 返回空 coverage/evidence，应被拒绝并使 synthesis incomplete。
5. 让一个 Agent 返回 stale head，应被拒绝。
6. 正常结果仍必须显示 `human_decision_required`，GitLink 写入为 0。

## 9. 企业微信

1. 启动唯一 Sidecar 与回环 Review Core。
2. 在 allowlist 测试群发送限定仓库查询。
3. 保存 req_id/msgid 哈希、policy observation、Core action/repository 和最终流式回复截图。
4. 重投同一 msgid，重启前后均不得再次调用 Core。
5. 未授权群、用户、仓库和写操作命令必须拒绝。
6. 证据中确认 Sidecar 环境没有 GitLink Token，Core 环境没有企业微信 Secret。

## 10. 退出标准

只有场景 1—9 中与比赛演示范围对应的项目全部有脱敏证据，才能把状态从“代码合同完成”改为“真实平台验收完成”。CI 绿灯不能替代任何外部平台证据。
