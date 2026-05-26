# GitLink 仓库目录结构

本作品以 `gitlink-cli` 工作流示例的形式托管在 GitLink 仓库中，目录与主项目源码保持隔离，避免改变主仓库既有命令、Skill 和设计文档结构。

## 作品路径

```text
examples/workflows/community-ops-automation/
```

## 目录内容

- `README.md`：项目说明与复现入口
- `LICENSE`：Apache 2.0 开源协议
- `.gitignore`：运行时产物忽略规则
- `docs/`：架构、运行、验证和交付说明
- `examples/`：配置文件和示例输出
- `scripts/`：工作流执行脚本
- `tests/`：单元测试

## 仓库内验证

进入作品目录后运行：

```powershell
python -m unittest discover -s tests
.\scripts\run_demo.ps1
```

生成的 `outputs/` 是运行时目录；固定示例产物位于 `examples/demo_outputs/`。
