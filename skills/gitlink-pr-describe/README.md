# gitlink-pr-describe · PR 自动描述生成（使用说明）

> 任务二**创新增强** Skill · 作者 ylly

## 是什么
获取 PR 的 diff/变更文件，AI 按规范结构（背景/改动/测试/影响/类型）**自动生成 PR 描述**，提升 PR 质量和评审效率。

## 解决的痛点
开发者提 PR 描述写不全/不规范 → 评审难理解；手写费时。

## 怎么用
```
请阅读 skills/gitlink-pr-describe/SKILL.md，为 ylly/gitlink-cli 的 PR #<id> 生成规范描述。
```

## 验证案例
gitlink/gitlink-cli wiki shortcut PR → AI 生成：背景(补 wiki 命令)/改动(5 命令+测试+注册)/测试(go test)/影响(新模块)/类型(feat)。详见 verification.md。

## 创新点
结合 commit message + diff **双源**生成，结构规范（Conventional Commits），可写入 PR body。

## 文件清单
SKILL.md（生成工作流）/ README.md / verification.md
