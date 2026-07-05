# Artifact handbook example

User request:

```text
请为 Gitconomy/Git4Research 生成一份科研成果沉淀手册，用于课题组交接和答辩展示。
```

Agent steps:

```bash
gitlink-cli repo +info --owner Gitconomy --repo Git4Research --format json
gitlink-cli repo +readme --owner Gitconomy --repo Git4Research --ref main --format json
gitlink-cli repo +tree --owner Gitconomy --repo Git4Research --ref main --format json
gitlink-cli issue +list --owner Gitconomy --repo Git4Research --state open --format json
gitlink-cli pr +list --owner Gitconomy --repo Git4Research --state merged --format json
gitlink-cli release +list --owner Gitconomy --repo Git4Research --format json
```

Expected answer:

- Produce a handbook using the template.
- Mark missing evidence as `待补充`.
- Confirm that no Wiki, Issue, PR, Release, or file write operation was performed.

