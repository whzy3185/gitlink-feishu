# 示例：review 负载与责任停滞巡检

用户请求：

```text
Use $gitlink-maintainer-radar 看一下这个仓库最近哪些 PR 因 reviewer 负载失衡或责任人停滞而卡住，给我一个转派和提醒建议清单。
```

期望动作：

1. 读取 `../gitlink-shared/SKILL.md`。
2. 获取 open PR、相关 review 和未读提醒。
3. 识别 reviewer 高负载、低负载和停滞责任条目。
4. 输出按优先级排序的“转派建议”“提醒建议”“可继续观察”三部分结果。

输出要点：

- 不只统计 reviewer 数量，要指出具体的阻塞链路。
- 如果没有明确 reviewer 字段，也要基于 review 历史和等待关系做近似判断。
- 默认给建议，不直接修改分配关系。
