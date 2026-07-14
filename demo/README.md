# GitLink CLI · 演示网页（demo/）

> 一个**可交互演示站**，展示子任务一~四全部成果：25 域命令浏览器、48 Skill 卡片墙、pr-guard 工作流、科研四维画像。
> 后端 `web/server.py`（Python 标准库，零依赖）能真跑 `gitlink-cli`；**访客自带 token，零凭据上云端**。

## 目录结构

```
demo/
├── web/                          # 演示网页（核心）
│   ├── server.py                 # Python 后端（/api/run /api/skill /api/analyze）
│   ├── index.html                # 前端单页（Tailwind+Chart.js CDN）
│   └── README.md                 # 网页本地启动说明
├── Dockerfile                    # demo 部署镜像（Go 编译 + Python 运行时）
├── build-demo.sh                 # 本地一键产 Linux 二进制（测 Dockerfile 用）
├── research-insight-workflow.sh  # 任务四：科研画像端到端脚本
├── pr-guard-workflow.sh          # 任务三：质量看门人脚本
├── live-demo.sh / snippet-live-demo.sh
└── *.md                          # 各任务指南 + 验证记录 + 报告原件
```

---

## 一、本地跑（Windows / Linux / macOS）

```bash
# 1. 编译 CLI（仓库根）
cd gitlink-cli                    # 含 go.mod 的仓库根
go build -o gitlink-cli .           # Windows 产出 gitlink-cli.exe

# 2. 启动后端
cd demo/web
python server.py                   # → http://0.0.0.0:8000

# 3. 浏览器开 http://localhost:8000
#    顶栏粘自己的 GitLink token → 平台命令真跑；snippet 等本地命令免 token
```

> 二进制自动探测：`GITLINK_BIN` > 仓库根 `gitlink-cli[.exe]` > PATH。
> 端口/主机：`PORT=9000 HOST=127.0.0.1 python server.py`。

---

## 二、云端部署（用你们已有服务器 121.41.222.73）

已配置好「**push 即上线**」：`.devops/自动构建部署.yml` 在 push 到 master 后，SSH 到服务器增量拉取并**自动构建启动两个服务**：

| 端口 | 服务 | 镜像 | 入口 |
|:---:|------|------|------|
| **:8080** | 任务四科研网页终端 | 根 `Dockerfile`（Go + Python） | `gitlink-cli server --port 8080`（调 `scripts/research/` 跑 S1–S6） |
| **:8000** | 综合能力展示站（本 demo） | `demo/Dockerfile` | `python3 demo/web/server.py` |

```
# ssh_cmd_0：根 Dockerfile → :8080（带 --env-file /root/.gitlink-env 注入 token）
docker build -t gitlink-cli:latest . && docker run -d --name gitlink-cli -p 8080:8080 --env-file /root/.gitlink-env gitlink-cli:latest
# ssh_cmd_1：demo Dockerfile → :8000（非阻塞，失败不影响 :8080）
docker build -f demo/Dockerfile -t gitlink-cli-demo . && docker run -d --name gitlink-cli-demo -p 8000:8000 --restart unless-stopped gitlink-cli-demo
```

**你只需（一次性服务器侧准备）**：
1. 阿里云安全组/防火墙**开放 8080 + 8000** 两条入方向 TCP 规则。
2. 在服务器建 `/root/.gitlink-env`，内容 `GITLINK_TOKEN=你的令牌`（供 :8080 科研终端调平台 API；:8000 展示站不需要，访客自带 token）。
3. 之后每次 `git push origin master` → CI 自动重建双服务 → 直接打开网址：
   - 科研终端 `http://121.41.222.73:8080`
   - 综合展示 `http://121.41.222.73:8000`

> 手动部署（不走 CI）：SSH 到服务器，`cd /root/gitlink-cli` 后分别跑上面两条 docker 命令。

### 镜像里有什么（demo/Dockerfile 多阶段）
- Stage1 `golang:1.26-alpine`：`CGO_ENABLED=0 GOOS=linux go build` 产 Linux 二进制。
- Stage2 `python:3.11-alpine`：装 `git`/`ca-certificates`，放二进制到 `/usr/local/bin/gitlink-cli`，拷 `demo/` 和 `skills/`，`ENV PORT=8000`，`CMD python3 demo/web/server.py`。

---

## 三、安全模型（为什么能放心公网开放）

| 点 | 做法 |
|----|------|
| 团队 token | **不烘焙**进镜像/代码。镜像里没有任何 GitLink 凭据。 |
| 访客 token | 只存在访客自己的浏览器 localStorage，按请求传后端 → 注入子进程 `GITLINK_TOKEN` → 用完即弃，**不落盘、不写日志**。 |
| 命令注入 | 后端白名单（仅 30 个 gitlink-cli 顶层域）+ subprocess 列表参数（不经 shell）+ 30s 超时。 |
| 写操作 | CLI 写操作本就要 `--dry-run`/确认；演示页默认只点只读命令。 |

→ 公网开放的安全风险≈0：泄露的至多是访客自己输错的那一次请求。

---

## 四、访客怎么用（写进 PPT/答辩）

1. 打开 `http://121.41.222.73:8000`。
2. 顶栏粘自己的 GitLink 个人访问令牌（GitLink → 个人中心 → 个人令牌）。
3. 点「命令域」里任意动词 → 终端真跑；或点 Skill 卡片读 SKILL.md；或科研区「实拉分析」任一仓库。

---

## 五、其它 PaaS 部署（可选，不占你们服务器）

也可部署到 Render / Railway / Koyeb 等（需能跑 Docker）：
- 用 `demo/Dockerfile`，暴露端口环境变量 `PORT`（已支持）。
- 这些平台默认按其给的端口注入 `PORT`，server.py 已读 `PORT` 环境变量，无需改。
