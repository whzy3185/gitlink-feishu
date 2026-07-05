# Reproducibility audit example

User request:

```text
请用 GitLink 分析 songhui18/ICCV2021 的论文复现性，输出缺失项和整改优先级。
```

Agent steps:

```bash
gitlink-cli repo +info --owner songhui18 --repo ICCV2021 --format json
gitlink-cli repo +tree --owner songhui18 --repo ICCV2021 --ref master --format json
gitlink-cli api GET /songhui18/ICCV2021/sub_entries --query 'filepath=&ref=master' --format json
```

Expected answer:

- Quote the collected command names.
- Score README, LICENSE, dependencies, tests, CI, examples/docs, data notes.
- List the top three fixes before any optional polishing.
- State that no write action was performed.

