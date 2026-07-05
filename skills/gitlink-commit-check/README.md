# gitlink-commit-check · 提交规范检查（使用说明）

> 任务二**创新增强** Skill · 作者 ylly

## 是什么
检查 commit message 是否符合 **Conventional Commits**（feat/fix/docs 等），识别不规范提交并给**修复建议**，提升提交质量和 Release Notes 可生成性。

## 解决的痛点
commit 不规范（"测试流水线""1"）→ Release Notes 难生成、历史难读、协作混乱。

## 规范模型
`type(scope): subject`，type 必须合法（feat/fix/docs/refactor/test/chore/ci/perf/style）。

## 怎么用
```
请阅读 skills/gitlink-commit-check/SKILL.md，检查 ylly/gitlink-cli 近 20 条 commit 的规范性。
```

## 验证案例
gitlink/gitlink-cli：规范率 ~85%。不规范的（"测试流水线"→建议 chore:、"1"→补充、"增加label"→feat:）均给出修复建议。详见 verification.md。

## 文件清单
SKILL.md（规范模型+检查工作流）/ README.md / verification.md
