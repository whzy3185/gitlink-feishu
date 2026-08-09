# 示例：日常值班扫描

用户请求：

```text
Use $gitlink-maintainer-radar 扫描 Gitlink/gitlink-cli 当前 open PR 和 open Issue，告诉我今天维护者先处理什么。
```

期望动作：

1. 读取 `../gitlink-shared/SKILL.md`。
2. 获取当前登录用户、仓库信息、未读消息、open PR、open Issue。
3. 对命中的高风险 PR 补拉详情和 review。
4. 输出一份中文报告，至少包含“SLA 超时”“Review 负载”“责任停滞”“今日建议动作”。

输出要点：

- 不要把所有通知原样抄出来。
- 优先指出卡在维护者侧的条目。
- 优先发现超时未响应、review 失衡和责任失效。
- 给出 3 到 5 条能直接执行的建议。
