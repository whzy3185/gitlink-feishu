# Research weekly report example

User request:

```text
请给 songhui18/ICCV2021 生成一份科研项目进度周报和风险预警。
```

Agent steps:

```bash
gitlink-cli repo +info --owner songhui18 --repo ICCV2021 --format json
gitlink-cli issue +list --owner songhui18 --repo ICCV2021 --state open --format json
gitlink-cli issue +list --owner songhui18 --repo ICCV2021 --state closed --format json
gitlink-cli pr +list --owner songhui18 --repo ICCV2021 --state open --format json
gitlink-cli pr +list --owner songhui18 --repo ICCV2021 --state merged --format json
gitlink-cli release +list --owner songhui18 --repo ICCV2021 --format json
```

Expected answer:

- Provide a concise weekly report.
- Identify the top 3 risks with evidence.
- Do not write comments back unless explicitly requested.

