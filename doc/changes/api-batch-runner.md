# 新增 Raw API 批处理执行器

`gitlink-cli api` 现在支持通过 `--batch-file` 读取 JSON 批处理计划，把多个尚未封装为 shortcut 的 GitLink API 请求组织成一次可审计的自动化执行。计划文件支持 `vars` 模板变量，`--var key=value` 可以在执行时覆盖变量，便于同一批处理流程复用到不同仓库、Issue 或分支。

批处理模式提供 `--dry-run` 预览渲染后的 method、path、query 和 body，不会访问远端；实际执行时会输出每一步的成功/失败、响应数据和汇总计数。默认遇到失败立即停止，传入 `--continue-on-error` 后会继续执行后续请求，适合批量巡检、批量评论、批量元数据修复等场景。

本次变更包含计划文件解析、模板渲染、query/body 递归替换、失败控制、结构化汇总输出、中英文帮助文案、README 示例、Skill reference 和单元测试。测试覆盖 dry-run 不发请求、变量覆盖、模板缺失报错、失败默认中断以及失败继续执行等关键行为。
