# 科研复现性审计报告：steph1/Awesome-Paper-Crawler（ref: 默认分支）

**总分：8/100 — 等级 D（复现困难）**

| 检查项 | 得分 | 证据 | 建议 |
|--------|------|------|------|
| README 与运行说明 | 8/15 | README=README.md；运行/复现章节=未检出 | 补充 README 并加入 How to run / 复现步骤章节 |
| 开源许可证 | 0/15 | 许可证文件=缺失 | 添加 LICENSE 文件（科研代码推荐 MIT/Apache-2.0/BSD） |
| 引用信息 | 0/10 | CITATION 文件=无；README 引用章节=无 | 添加 CITATION.cff 或在 README 中给出 BibTeX |
| 依赖清单（环境固化） | 0/15 | 依赖文件=缺失 | 添加 requirements.txt / environment.yml 等依赖清单并固定版本 |
| 运行入口 | 0/10 | 入口=未检出 | 提供 Makefile / run.sh / main.py 等一键运行入口 |
| 数据可得性说明 | 0/10 | 数据目录=无；README 数据说明=无 | 在 README 中说明数据集来源、获取方式与预处理步骤 |
| 测试/验证代码 | 0/10 | 测试目录=无 | 添加 tests/ 验证关键结果可复算 |
| 版本发布（成果固化） | 0/15 | release 数=0 | 为论文对应的代码状态打 tag 并创建 release |

---
*由 repro-audit 生成（确定性检查，同输入同输出）。*