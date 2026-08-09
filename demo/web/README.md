# GitLink CLI 智能化能力展示（演示网页）

一个**可交互的演示站**：点动词/敲命令 → 真跑 gitlink-cli → 显示真实输出，配合 25 域命令浏览器、48 Skill 卡片墙、pr-guard 流程、科研四维雷达，全面展示子任务一~四的成果。

> 位置：仓库内 `demo/web/`（`server.py` + `index.html`）。后端零依赖（仅 Python 标准库）。

## 架构（访客自带 token，零凭据上云）

```
浏览器 index.html  ──fetch──▶  server.py（Python 标准库）
  顶栏 token + owner/repo           │  GET /api/skill  读 SKILL.md
  命令域 / 终端 / Skill 墙           │  POST /api/run   真跑 CLI（token 透传给子进程）
  pr-guard / 科研雷达                │  POST /api/analyze  四维评分 + 巴士因子
                                   ▼
                              gitlink-cli（仓库根 ../../gitlink-cli[.exe]）
```

- 访客 token 仅存在**访客自己的浏览器**（localStorage），按请求传后端 → 注入子进程 `GITLINK_TOKEN` → 用完即弃，**不落服务端、不写日志**。
- 本地命令（`snippet`/`auth`）免 token 即可真跑；平台命令（`repo`/`issue`/`pr`…）需访客填自己的 token。

## 本地启动（3 步）

```bash
# 1. 在仓库根编译 CLI（已有可跳过）
cd gitlink-cli                # 仓库根（含 go.mod）
go build -o gitlink-cli .        # Windows 会生成 gitlink-cli.exe

# 2. 启动后端（零依赖）
cd demo/web
python server.py                 # → http://0.0.0.0:8000

# 3. 浏览器打开 http://localhost:8000
#    顶栏粘自己的 GitLink token（auth login --token 拿）→ 平台命令即可真跑
```

> 服务端会自动探测二进制：`GITLINK_BIN` 环境变量 > 仓库根 `gitlink-cli`/`gitlink-cli.exe` > PATH。
> 端口/主机可设：`PORT=9000 HOST=127.0.0.1 python server.py`。

## 展示区

| 区块 | 内容 |
|------|------|
| ① 命令全域浏览器 | 25 域 160+ 动词，按子任务分组 + 搜索；点动词填终端真跑 |
| ② Skill 全集 | 48 个 Skill 卡片（按 全部/新增/科研/质量 筛选），点开读 SKILL.md 全文 |
| ③ pr-guard | 5 步门禁动画 + 「用真实 PR 跑」（填 token） |
| ④ 科研画像 | 「实拉分析」目标仓库 → 四维雷达 + 协作网络 + 巴士因子 |
| ⑤ 验证 | 命令层 / 编排层 / 输出层 三层证据 |

## 安全

- 后端白名单（仅 30 个 gitlink-cli 顶层域）+ subprocess 列表参数（不经 shell）+ 30s 超时。
- 访客 token 不落服务端。公网部署也**不烘焙任何团队 token**。

## 云端部署

见上级 [`demo/README.md`](../README.md)（Dockerfile + `.devops` 流水线 + 服务器部署说明）。
