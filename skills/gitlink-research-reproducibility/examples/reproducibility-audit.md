# 复现性审计示例

用户请求：

```text
请用 GitLink 分析 songhui18/ICCV2021 的论文复现性，输出缺失项和整改优先级。
```

Agent 步骤：

```bash
gitlink-cli repo +info --owner songhui18 --repo ICCV2021 --format json
gitlink-cli repo +tree --owner songhui18 --repo ICCV2021 --ref master --format json
gitlink-cli api GET /songhui18/ICCV2021/sub_entries --query 'filepath=&ref=master' --format json
```

预期回答：

- 列出已采集的命令名称。
- 对 README、LICENSE、依赖、测试、CI、示例/文档和数据说明评分。
- 优先列出前三个整改项，再给可选优化建议。
- 声明未执行写操作。
