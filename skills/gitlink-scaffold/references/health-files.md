# 社区健康文件清单与评分

本技能检测 8 类开源社区推荐文件，按权重计算 0-100 健康度分。

## 检测清单与权重

| 文件 | 权重 | 关键 | 候选文件名 | 查找目录 |
|------|:----:|:----:|------------|----------|
| README | 20 | 是 | readme.md / readme.rst / readme.txt / readme | 根目录 |
| LICENSE | 20 | 是 | license / license.md / license.txt / copying | 根目录 |
| CONTRIBUTING | 15 | 否 | contributing.md / contributing.rst | 根 / .gitlink / .github / docs |
| CODE_OF_CONDUCT | 10 | 否 | code_of_conduct.md / code-of-conduct.md | 根 / .gitlink / .github / docs |
| SECURITY | 10 | 否 | security.md / security | 根 / .gitlink / .github / docs |
| Issue 模板 | 10 | 否 | issue_template.md 等 | 根 / .gitlink / .github / ISSUE_TEMPLATE |
| PR 模板 | 10 | 否 | pull_request_template.md 等 | 根 / .gitlink / .github |
| CHANGELOG | 5 | 否 | changelog.md / changes.md / history.md | 根目录 |

健康度分 = 已具备文件的权重之和 / 总权重(100) × 100。

## 关键文件

README 与 LICENSE 标记为**关键文件**，缺失会在报告中以 ❗ 高亮，因为它们是开源项目最基本的要求（说明项目用途、明确授权）。

## 可生成模板的文件

| 文件 | 生成路径 | 模板语言 |
|------|----------|:--------:|
| CONTRIBUTING | CONTRIBUTING.md | 中文 |
| CODE_OF_CONDUCT | CODE_OF_CONDUCT.md | 中文 |
| SECURITY | SECURITY.md | 中文 |
| Issue 模板 | .gitlink/issue_template.md | 中文 |
| PR 模板 | .gitlink/pull_request_template.md | 中文 |
| CHANGELOG | CHANGELOG.md | 中文 |

README 与 LICENSE 不自动生成模板（README 需项目特定内容；LICENSE 应由作者选择许可证）。

## 多目录查找说明

不同项目把社区文件放在不同位置（GitHub 习惯 `.github/`，GitLink 习惯 `.gitlink/`，也有放根目录或 `docs/`）。本技能逐目录查找，命中任一即视为存在，避免误报缺失。
