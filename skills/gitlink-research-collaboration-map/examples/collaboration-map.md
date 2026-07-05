# 协作画像示例

用户请求：

```text
请分析 Gitconomy/Git4Research 的科研协作结构，找出核心维护者和可跟进的问题方向。
```

Agent 步骤：

```bash
gitlink-cli repo +info --owner Gitconomy --repo Git4Research --format json
gitlink-cli repo +contributors --owner Gitconomy --repo Git4Research --format json
gitlink-cli issue +list --owner Gitconomy --repo Git4Research --state open --format json
gitlink-cli pr +list --owner Gitconomy --repo Git4Research --state open --format json
```

预期回答：

- 区分事实和建议。
- 如果仓库 Issue/PR 活动较少，需要说明数据局限。
- 不暴露私人联系方式。
