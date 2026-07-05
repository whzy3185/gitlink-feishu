# Data provenance audit example

User request:

```text
请检查 songhui18/ICCV2021 是否有科研数据来源、数据许可证或隐私风险问题。
```

Agent steps:

```bash
gitlink-cli repo +info --owner songhui18 --repo ICCV2021 --format json
gitlink-cli repo +tree --owner songhui18 --repo ICCV2021 --ref master --format json
gitlink-cli repo +tree --owner songhui18 --repo ICCV2021 --path data --ref master --format json
```

Expected answer:

- Report only paths and risk types, not raw data contents.
- Separate data source, license, citation, privacy, and large-file risks.
- State whether the assessment is root-only or includes selected subdirectories.

