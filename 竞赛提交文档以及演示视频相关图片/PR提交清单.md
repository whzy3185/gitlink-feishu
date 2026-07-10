# 子赛题一 · PR 提交说明文档

**项目**：gitlink-cli · AI 可驱动的智能协作平台
**主仓库**：`https://gitlink.org.cn/whale_hihihi/gitlink-cli`（团队 fork，作为竞赛主仓库接收 PR）
**团队**：崔佳祥（whale）/ 包尔俊（wauxing）/ 王嘉奇（Surponess）
**说明**：本文档为子赛题一的 PR 提交说明，列出三人组向主仓库提交并合并的全部 PR，每个 PR 含功能代码、单元测试、命令帮助文档及变更说明。

## 一、提交统计

- 合并 PR 总数：**20** 个（Surponess 12 / whale 2 / wauxing 6)
- 其中子赛题一（CLI 能力）核心 PR：**11** 个
- 全部 PR 累计变更：功能代码 .go 文件 77、单元测试 _test.go 文件 45、文档 .md 文件 147。

## 二、子赛题一（CLI 能力）PR 明细

每个 PR 均包含：**功能代码 + 单元测试 + 命令帮助文档 + 变更说明**（符合竞赛交付要求）。

| PR | 标题 | 作者 | 合并日期 | 代码 | 测试 | 文档 | 主要内容 |
|---|---|---|---|:--:|:--:|:--:|---|
| [#18](https://gitlink.org.cn/whale_hihihi/gitlink-cli/pulls/18) | feat: 合并 upstream/master 并对齐风格 | wauxing | 2026-06-15 | 31 | 18 | 79 | 合并 upstream/master + 行为对齐 + i18n 化 |
| [#15](https://gitlink.org.cn/whale_hihihi/gitlink-cli/pulls/15) | 修改编译 | Surponess | 2026-06-04 | 5 | 1 | 1 | 编译修复 + pm 域注册（25 域全可用） |
| [#14](https://gitlink.org.cn/whale_hihihi/gitlink-cli/pulls/14) | feat: 补全 21 个 Shortcut + 修正批量操 | wauxing | 2026-06-03 | 8 | 8 | 13 | 补全 21 个 Shortcut + 批量修正 + 文档清理 |
| [#12](https://gitlink.org.cn/whale_hihihi/gitlink-cli/pulls/12) | feat: 实现 batch issue 批量操作命令 | wauxing | 2026-06-01 | 9 | 4 | 0 | issue batch 批量操作命令族（CSV 驱动） |
| [#8](https://gitlink.org.cn/whale_hihihi/gitlink-cli/pulls/8) | 文件描述符泄漏等修改 | Surponess | 2026-06-01 | 5 | 0 | 0 | 文件描述符泄漏等 6 类代码质量修复 |
| [#7](https://gitlink.org.cn/whale_hihihi/gitlink-cli/pulls/7) | 代码片段管理功能 | Surponess | 2026-06-01 | 3 | 2 | 0 | snippet 本地代码片段管理 7 命令 |
| [#6](https://gitlink.org.cn/whale_hihihi/gitlink-cli/pulls/6) | release download 命令 新增 downloa | Surponess | 2026-05-30 | 7 | 1 | 0 | release download 完善 + 错误消息统一化 |
| [#5](https://gitlink.org.cn/whale_hihihi/gitlink-cli/pulls/5) | release download 功能（新增）  在 rel | Surponess | 2026-05-29 | 1 | 6 | 0 | release +download 流式下载二进制资源 |
| [#4](https://gitlink.org.cn/whale_hihihi/gitlink-cli/pulls/4) | label 领域原来只有 3 个命令（list/create | Surponess | 2026-05-29 | 2 | 1 | 0 | label 领域新增 +update 命令 |
| [#3](https://gitlink.org.cn/whale_hihihi/gitlink-cli/pulls/3) | 创建 webhook 领域 + 实现 `+update` | Surponess | 2026-05-28 | 1 | 1 | 0 | 创建 webhook 领域 + +update（GET-then-PUT） |
| [#2](https://gitlink.org.cn/whale_hihihi/gitlink-cli/pulls/2) | 本次实现：search +issues — 搜索 Issue | Surponess | 2026-05-25 | 1 | 1 | 2 | search +issues 搜索 Issue（10 筛选参数） |

## 三、其他子赛题相关 PR（备查）

| PR | 标题 | 作者 | 归属 |
|---|---|---|---|
| #29 | feat: 统一全链路科研分析页面 + Dockerfile 修复 | whale | 四·科研 |
| #28 | feat(demo): 调整 demo 前端展示（合并 surpones | wauxing | 三/四·Demo |
| #27 | feat(demo): 优化 demo 前端展示（合并 surpones | wauxing | 三/四·Demo |
| #26 | chore: move community-ops-sweep work | wauxing | 三·工作流 |
| #24 | 新增demo | Surponess | 三/四·Demo |
| #22 | 修改了两个skills的命令使用 | Surponess | 二·Skills |
| #19 | 新增skills | Surponess | 二·Skills |
| #144 | feat(skills): 新增 3 个 Agent Skill — w | whale | 二·Skills |
| #16 | 新增skills | Surponess | 二·Skills |

## 四、PR 提交规范说明

- **分支模型**：三人各自从 master 切特性分支（`surponess_br` / `cuijixiang` / `baoerjun_branch`），开发完成后向 `master` 发 PR，经 review 合并。
- **每个 PR 内容**：`shortcuts/<域>/` 功能代码 + `<域>_test.go` 单元测试 + `skills/gitlink-<域>/SKILL.md` 命令帮助文档 + commit message/PR 描述作为变更说明。
- **CI**：PR 触发 `.gitea/workflows/ci.yml`（go build + go test + lint）；通过后方可合并。
- **PR 链接**：`https://gitlink.org.cn/whale_hihihi/gitlink-cli/pulls/<PR号>`。
