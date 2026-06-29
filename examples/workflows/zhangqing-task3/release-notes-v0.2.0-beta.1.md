> 本版本为**任务三阶段性预发布**。正式版 v0.2.0 将在任务三全部 7 个端到端工作流完工后发布。

## v0.2.0 Beta 1 — 智能运营能力升级版（预发布）

本版本集中呈现任务二「AI Skill 套件」与任务三「端到端自动化工作流」的阶段成果，为 GitLink CLI 引入完整的智能运营能力。

### ✨ 新功能 · Skills 智能套件

任务二交付的 AI Skill，让 Agent 能直接驱动 GitLink 完成高级运营场景：

- feat(skills): 新增 gitlink-issue-triage — Issue 智能分拣（自动打标签 / 识别 good first）
- feat(skills): 新增 gitlink-release-auto — 自动发版（语义化版本推荐 + Release Notes 生成）
- feat(skills): 新增 gitlink-commit-quality — 提交信息规范检查
- feat(skills): 新增 gitlink-code-review — 代码审查
- feat(skills): 新增 gitlink-insight — 仓库健康洞察
- feat(skills): 新增 gitlink-compliance — 合规检查
- feat(skills): 新增文档智能维护 + 新人引导 Skill

### ✨ 新功能 · Shortcut 命令扩展

- feat(wiki): 新增 wiki +list / +view / +create / +update / +delete
- feat(label): 新增 label 标签管理命令
- feat(notification): 新增 notification 通知模块
- feat(member): 新增 member 仓库成员管理命令
- feat(milestone): 新增 milestone 里程碑命令
- feat(compare): 新增 compare 对比命令
- feat(pr): 新增 pr reopen / review / patchset version 命令
- feat(webhook): 新增 webhook 快捷命令组
- feat(repo): 新增 repo readme 快捷命令
- feat(issue): 新增 issue authors / assigners 查询命令
- feat(api): 支持 --body-file 从文件读取请求体（解决 Windows 中文编码问题）

### 📚 工作流示例

- feat(workflows): 新增 community-ops-automation 社区运营自动化端到端示例

### 🐛 Bug 修复

- fix(notification): 修正通知读取 / 删除的 API endpoint 与请求体
- fix(issue): 修正 --label 语义，新增 --label clear 清除标签
- fix(wiki): 修正 wiki 网关 API（gateway.gitlink.org.cn）与请求方法
- fix(label): 修正 label API path 至 /{owner}/{repo}/labels
- fix(pr): view 输出补充 closed time 字段
- fix(member): 报告批量添加中的部分失败

### 📦 其他变更

- docs: 3 个 Skill 的 REFERENCE.md 参考文档
- docs: 工作流示例的提交清单 / 验证文档
- test: webhook / label / notification 测试用例对齐
- refactor: 自动部署配置精简

### 🤝 贡献者

感谢以下贡献者参与本版本开发：@ylly、@zhangqing、@ZxR、@yangsai01、@Leo77、@Mengz、@wangyue789、@puygob236、@dtwdtw、@muel、@Jiachen Li、@Tiger、@wbtiger、@ljc0426、@whzy
