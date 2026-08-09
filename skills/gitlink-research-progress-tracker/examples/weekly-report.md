# 科研周报示例

用户请求：

```text
请给 songhui18/ICCV2021 生成一份科研项目进度周报和风险预警。
```

Agent 步骤：

```bash
gitlink-cli repo +info --owner songhui18 --repo ICCV2021 --format json
gitlink-cli issue +list --owner songhui18 --repo ICCV2021 --state open --format json
gitlink-cli issue +list --owner songhui18 --repo ICCV2021 --state closed --format json
gitlink-cli pr +list --owner songhui18 --repo ICCV2021 --state open --format json
gitlink-cli pr +list --owner songhui18 --repo ICCV2021 --state merged --format json
gitlink-cli release +list --owner songhui18 --repo ICCV2021 --format json
```

预期回答：

- 提供简洁的科研项目周报。
- 识别前三个风险并给出证据。
- 除非用户明确要求，不写回评论。
