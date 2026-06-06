# 上手指南结构与识别规则

## 指南六段结构

1. **项目是什么**：简介 + 技术栈 + 社区指标
2. **代码在哪**：核心目录导航
3. **社区文件是否齐全**：README/CONTRIBUTING/行为准则
4. **从哪个 Issue 开始**：good-first-issue
5. **遇到问题找谁**：核心贡献者
6. **上手步骤**：Fork → 改 → PR

## 技术栈识别

依据根目录依赖/配置文件推断：

| 文件 | 技术栈 |
|------|--------|
| go.mod | Go |
| package.json | Node.js / JavaScript |
| requirements.txt / pyproject.toml | Python |
| Cargo.toml | Rust |
| pom.xml / build.gradle | Java |
| composer.json | PHP |
| Gemfile | Ruby |
| Dockerfile | Docker |
| Makefile | Make |

## 核心目录语义提示

| 目录 | 含义 |
|------|------|
| src / lib | 源码 / 库 |
| cmd | 命令行入口 |
| internal / pkg | 内部包 / 公共包 |
| app / core | 应用 / 核心模块 |
| docs | 文档 |
| test(s) | 测试 |
| examples | 示例 |
| skills | Agent Skills |

带语义提示的目录优先展示，帮助新人快速定位代码入口。

## good-first-issue 识别

开放 Issue 中，标题/正文含 typo/docs/readme/test/翻译/示例 等关键词且描述较短（<800 字）
的，判定为新手友好，最多取 8 个。
