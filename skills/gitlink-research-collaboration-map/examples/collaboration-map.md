# Collaboration map example

User request:

```text
请分析 Gitconomy/Git4Research 的科研协作结构，找出核心维护者和可跟进的问题方向。
```

Agent steps:

```bash
gitlink-cli repo +info --owner Gitconomy --repo Git4Research --format json
gitlink-cli repo +contributors --owner Gitconomy --repo Git4Research --format json
gitlink-cli issue +list --owner Gitconomy --repo Git4Research --state open --format json
gitlink-cli pr +list --owner Gitconomy --repo Git4Research --state open --format json
```

Expected answer:

- Separate facts from recommendations.
- Mention data limits if the repository has little Issue/PR activity.
- Do not expose private contact information.

