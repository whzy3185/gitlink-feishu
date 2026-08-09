# 运行手册（Runbook）

## 1. 前置条件

```bash
# 安装并登录 gitlink-cli
npm install -g @gitlink-ai/cli
gitlink-cli auth login          # 或 export GITLINK_TOKEN=...

# Go 1.21+（复用主仓库 go.mod，无需额外依赖）
go version
```

## 2. 最短复现路径（离线，使用已归档数据）

无需网络，基于仓库内固定示例数据复跑，产物应与 `examples/demo_outputs/` 一致：

```bash
cd examples/workflows/research-team-insight
go run ./scripts --config config/team.example.json \
    --out ./_repro --no-fetch --data-root ./examples/demo_outputs/data
# 打开 ./_repro/team-report.html 查看
```

## 3. 真实运行（在线采集）

```bash
cd examples/workflows/research-team-insight

# Linux / macOS
go run ./scripts --members "yystopf,jiangtx,wangyue111" \
    --team-name "gitlink-cli 贡献组" --out ./out

# Windows（gitlink-cli 为 .CMD 包装）
go run ./scripts --members "yystopf,jiangtx,wangyue111" --out ./out `
    --cli-bin "C:\nvm4w\nodejs\gitlink-cli.CMD"
# 或直接用一键脚本：
.\scripts\run_demo.ps1
```

按方向自动发现成员：

```bash
go run ./scripts --keyword "联邦学习" --max 5 --out ./out
```

## 4. 真实验证记录

- 验证对象：`@yystopf`（何慧·国防科技大学）、`@jiangtx`、`@wangyue111`
- 运行结果：退出码 0；采集 3 人 × 6 接口 + 1 次渲染 = **19 条命令**（见 `examples/demo_outputs/command_log.json`）
- 产物：
  - `team-report.html`（能力雷达叠加 + 能力矩阵 + 协作网络图）
  - `team-insight.md`（识别团队最强项=贡献度、短板=影响力；标注 @yystopf 为镜像聚合账号、分布偏泛）
  - `command_log.json`

## 5. 测试

```bash
# 在主仓库根目录
go test ./examples/workflows/research-team-insight/scripts/
```

## 6. 常见问题

| 现象 | 处理 |
|------|------|
| Windows 上 `exec: "gitlink-cli": file not found` | `gitlink-cli` 为 npm `.CMD` 包装；用 `--cli-bin ...\gitlink-cli.CMD` 或设 `GITLINK_CLI_BIN`。程序对 `.cmd/.bat` 会自动用 `cmd /c` 包装 |
| `TLS handshake timeout` | 瞬时网络问题，自动重试 2 次（间隔 3s）；仍失败会记录到 `command_log.json` 并退出非零 |
| 某成员数据稀疏 / 学科为空 | 报告中如实标注，不臆造 |
| 某账号项目/学科数异常巨大 | 已结合 `user +info.mirror_projects_count` 自动标注"含镜像聚合，分布偏泛" |
