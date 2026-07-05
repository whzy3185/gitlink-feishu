# Compliance audit example

User request:

```text
请检查 Gitconomy/Git4Research 是否具备开源发布合规基础。
```

Agent steps:

```bash
gitlink-cli repo +info --owner Gitconomy --repo Git4Research --format json
gitlink-cli repo +tree --owner Gitconomy --repo Git4Research --ref main --format json
```

Expected answer:

- Report license, security policy, contribution guide, dependency manifest, and sensitive file risks.
- Mention whether the audit is root-only or includes selected subdirectories.
- Avoid quoting or exposing sensitive file contents.

