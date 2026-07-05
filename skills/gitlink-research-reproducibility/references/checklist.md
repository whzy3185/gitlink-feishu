# Reproducibility checklist

| 项目 | 分值 | 通过条件 | 缺失时建议 |
|---|---:|---|---|
| README | 20 | 根目录存在 README，且能说明项目目标、安装或运行方式 | 补充项目背景、环境准备、最小运行命令 |
| LICENSE | 15 | 根目录存在 LICENSE 文件 | 选择合适开源许可证并在 README 中声明 |
| 依赖清单 | 15 | 存在语言对应依赖文件 | 补充 requirements.txt、pyproject.toml、environment.yml、go.mod 等 |
| 测试入口 | 20 | 存在 tests/test/__tests__ 或可运行验证脚本 | 提供最小单元测试或 smoke test |
| CI 配置 | 10 | 存在 CI 配置目录或文件 | 增加基础 lint/test CI |
| 示例或文档 | 10 | 存在 docs/examples/demo | 补充复现实验步骤或示例输入输出 |
| 数据或实验说明 | 10 | 存在 data/datasets/experiments/notebooks 或 README 中明确说明 | 写明数据来源、下载方式、实验配置 |

