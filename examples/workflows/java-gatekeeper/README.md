这是子赛题三的端到端 Java 看门人工作流示例。



GitLink Gatekeeper - 基于 Java 的 AI 智能代码质量门禁系统



项目定位与核心价值）

本项目专为 2026 GitLink 自动化工作流大赛-子赛题三打造。

在开源社区中，维护者往往需要花费大量时间评审 Pull Request (PR)。`GitLink Gatekeeper` 是一个端到端的自动化解决方案，它将 `gitlink-cli` 的核心能力与 Java 强大的后端控制逻辑相结合，充当社区的\*\*“代码质量看门人”。



核心价值：

1\. 全自动审计：无需人工干预，一键自动拉取 PR 元数据并调用核心 AI 能力生成评审报告。

2\. 高鲁棒性分流：具备状态感知能力，针对已关闭/已合并的 PR 自动执行安全降级策略（转为日志留痕），防止接口报错或污染历史代码。

3\. 环境自适应：内置底层环境变量清洗与字符流重映射，彻底解决 Windows/Linux 跨平台执行时的网络代理死锁及控制台乱码问题。



\---



工作流架构设计（3个及以上命令串联）



本工作流通过 Java 进程控制，完美串联了以下能力链条：

1\. 步骤一：元数据侦测 —— 调起 `gitlink-cli pr +view` 抓取目标 PR 的实时状态与技术描述文本。

2\. 步骤二：AI 智能评审 —— 驱动 `gitlink-cli workflow +pr-summary` 核心 Skill 对代码 Diff 进行深度分析，生成 Markdown 格式的摘要。

3\. 步骤三：动态决策回写 —— 解析步骤一状态，若 PR 开放则调用 `gitlink-cli pr +review` 自动回写 Approved 评论；若 PR 已关闭，则优雅回退至本地日志记录。



\---



快速开始与复现指南



为了保证评委与维护者能够 \*\*100% 完美复现\*\* 本工作流，请遵循以下步骤：



1\. 前置环境准备

Java 环境：JDK 17 或更高版本

构建工具：Apache Maven 3.6+

官方工具：已全局安装 `gitlink-cli` 并且已通过 `gitlink-cli auth login` 成功登录您的账户。



2\. 搬运与配置

确保本项目放置在官方主仓库的指定路径下：

`gitlink-cli/examples/workflows/java-gatekeeper/`



3\. 一键运行命令

打开终端，切换到当前工作流根目录，直接执行以下 Maven 命令即可拉起工作流：

```cmd

mvn spring-boot:run

