# 协作画像字段

| 字段 | 来源 | 用途 |
|---|---|---|
| 贡献者账号/名称 | `repo +contributors` | 识别代码贡献者 |
| 提交/贡献计数 | `repo +contributors` | 粗略衡量贡献量 |
| Issue 作者 | `issue +list` | 识别问题提出者 |
| Issue 负责人 | `issue +list` | 识别责任分配 |
| Issue 标签 | `issue +list` | 判断研究主题或任务类型 |
| PR 作者 | `pr +list` | 识别代码贡献者和集成者 |
| PR 状态 | `pr +list` | 判断协作流转状态 |
| 语言 | `repo +languages` | 推断技术栈 |

建议角色标签：

- 核心维护者：代码贡献和 Issue/PR 参与均较多。
- 问题提出者：Issue 活跃但代码贡献少，适合需求澄清和复现实验。
- 实验贡献者：PR 或实验相关 Issue 活跃。
- 潜在协作者：与主要语言或研究主题匹配，但当前参与较少。
