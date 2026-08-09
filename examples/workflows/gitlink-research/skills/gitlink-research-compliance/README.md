# gitlink-research-compliance · 科研合规与复现性检查（使用说明）

> 任务四科研辅助 Skill · 覆盖「合规与复现性检查」场景 · 作者 ylly

## 是什么
检查 GitLink 仓库的**开源合规性**（能否合法引用）+ **科研复现性**（能否稳定复现），输出合规与复现性报告。复用 `gitlink-compliance` 并增强科研复现维度。

## 检查模型（合规 3 + 复现 5 = 8 项）
- 📜 合规：LICENSE / 版权声明 / 依赖兼容
- 🔬 复现：数据可获取 / 环境说明 / 依赖锁定 / 复现步骤 / 版本稳定

## 怎么用
```
请阅读 .../gitlink-research-compliance/SKILL.md，检查 Gitlink/gitlink-cli 的合规与复现性。
```

## 验证案例
Gitlink/gitlink-cli：合规 ✅（MulanPSL-2.0）、复现 ⭐4.5（Go 环境+go.sum+12 Release+README 复现步骤）。详见 verification.md。

## 文件清单
SKILL.md（8项检查模型）/ README.md / verification.md
