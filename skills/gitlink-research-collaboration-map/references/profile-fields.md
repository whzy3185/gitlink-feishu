# Collaboration profile fields

| 字段 | 来源 | 用途 |
|---|---|---|
| contributor login/name | `repo +contributors` | 识别代码贡献者 |
| commits/contributions | `repo +contributors` | 粗略衡量贡献量 |
| issue author | `issue +list` | 识别问题提出者 |
| issue assignee | `issue +list` | 识别责任分配 |
| issue labels | `issue +list` | 判断研究主题或任务类型 |
| PR author | `pr +list` | 识别代码贡献者和集成者 |
| PR status | `pr +list` | 判断协作流转状态 |
| language | `repo +languages` | 推断技术栈 |

Suggested role labels:

- 核心维护者：代码贡献和 Issue/PR 参与均较多。
- 问题提出者：Issue 活跃但代码贡献少，适合需求澄清和复现实验。
- 实验贡献者：PR 或实验相关 Issue 活跃。
- 潜在协作者：与主要语言或研究主题匹配，但当前参与较少。

