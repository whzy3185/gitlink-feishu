# 成果手册示例

用户请求：

```text
请为 Gitconomy/Git4Research 生成一份科研成果沉淀手册，用于课题组交接和答辩展示。
```

Agent 步骤：

```bash
gitlink-cli repo +info --owner Gitconomy --repo Git4Research --format json
gitlink-cli repo +readme --owner Gitconomy --repo Git4Research --ref main --format json
gitlink-cli repo +tree --owner Gitconomy --repo Git4Research --ref main --format json
gitlink-cli issue +list --owner Gitconomy --repo Git4Research --state open --format json
gitlink-cli pr +list --owner Gitconomy --repo Git4Research --state merged --format json
gitlink-cli release +list --owner Gitconomy --repo Git4Research --format json
```

预期回答：

- 按模板生成科研成果沉淀手册。
- 将缺失证据标注为 `待补充`。
- 确认未执行 Wiki、Issue、PR、Release 或文件写操作。
