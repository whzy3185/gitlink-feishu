# 数据来源审计示例

用户请求：

```text
请检查 songhui18/ICCV2021 是否有科研数据来源、数据许可证或隐私风险问题。
```

Agent 步骤：

```bash
gitlink-cli repo +info --owner songhui18 --repo ICCV2021 --format json
gitlink-cli repo +tree --owner songhui18 --repo ICCV2021 --ref master --format json
gitlink-cli repo +tree --owner songhui18 --repo ICCV2021 --path data --ref master --format json
```

预期回答：

- 只报告路径和风险类型，不输出原始数据内容。
- 区分数据来源、许可证、引用、隐私和大文件风险。
- 说明评估范围是仅根目录，还是包含选定子目录。
