# 快速开始

## 1. 环境

Python 3.10+。工作流的采集与编排仅用 Python 标准库，**无需安装依赖、无需登录**即可分析公开仓库。

（仅运行单元测试需要 pytest：`pip install pytest`）

## 2. 一键运行

```powershell
# 默认分析 Gitlink/gitlink-cli，生成社区运营周报
.\scripts\run_demo.ps1
```

产物输出到 `outputs/Gitlink_gitlink-cli_flow.md`。

## 3. 分析指定仓库

```powershell
.\scripts\run_demo.ps1 -Owner <owner> -Repo <repo>
```

或直接用 Python：

```bash
python src/flow.py --owner <owner> --repo <repo>
```

## 4. 其他用法

```bash
# JSON 输出（供 Agent 或脚本消费）
python src/flow.py --owner Gitlink --repo gitlink-cli --format json

# 用 owner/repo 形式或完整 URL
python src/flow.py --slug Gitlink/gitlink-cli

# 批量分析多个仓库
python src/flow.py --config examples/config.json --output-dir outputs

# 控制提交采集量（每页 50 条，默认 4 页）
python src/flow.py --owner Gitlink --repo gitlink-cli --commit-pages 2
```

## 5. 周报包含什么

生成的社区运营周报有 6 个部分：

1. 仓库概览（Star/Fork/Issue/PR）
2. Issue 自动分拣（分类 + 新手友好任务建议）
3. PR Review 汇总（状态统计 + 待 Review 清单）
4. 社区健康体检（健康度评分 + 缺失文件）
5. 贡献者致谢（致谢榜）
6. Release Notes（按 conventional commits 自动归类）

## 运行测试

```bash
python -m pytest tests/ -q
```

## 常见问题

**首次运行慢？** 大仓库提交多，采集需要时间。可用 `--commit-pages 2` 减少采集量加速。

**想发布周报到仓库？** 工作流默认只生成本地文件。如需发布，可把周报内容通过
`gitlink-cli issue +comment` 发到指定 Issue（写操作，请先确认）。
