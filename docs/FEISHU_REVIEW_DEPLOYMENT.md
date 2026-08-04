# GitLink 飞书 Review 服务部署

## Linux systemd

材料位于 `deploy/systemd/`。Unit 使用 `Type=simple`、SIGTERM、45 秒停止超时、失败重启和 systemd 文件系统限制。仅状态、备份、日志目录可写；Home 和其余系统目录受保护。

Secret 从权限为 0600 的 `EnvironmentFile` 加载，不进入 Unit 的 `ExecStart` 参数。安装脚本建立专用系统用户和目录；卸载脚本只移除 Unit，保留数据库、备份、绑定和环境文件。

## Windows Service

材料位于 `deploy/windows/`，只使用 PowerShell、`New-Service`、`sc.exe` 和注册表，不依赖 NSSM。安装、卸载、启动、停止和重启脚本支持 `-WhatIf`；可配置服务名、工作目录、Bindings 和状态数据库。

环境文件内容写入服务专用注册表 Environment 值，环境文件 ACL 限制为 SYSTEM/Administrators。Secret 不进入 BinaryPathName。卸载只删除服务注册，不删除数据库或备份。

## Reverse Proxy

Caddy 与 Nginx 示例只代理：

```text
/webhooks/gitlink/*
/healthz
/readyz
```

管理 API 和 Metrics 保持 Loopback，不经公共反向代理。Webhook 保留原始 Body 与 `X-GitLink-Signature`、`X-GitLink-Timestamp`、`X-GitLink-Delivery`，设置 2MB Body 上限和短超时。生产环境需自行配置 TLS 证书、可信代理来源和网络防火墙。

## 上线前门禁

1. 使用受限账户和独立状态/备份目录。
2. 写入受限环境文件和 Bindings，禁止提交真实 Secret。
3. 先执行七个离线验证脚本和 Evidence Export。
4. 用 `review-service doctor` 检查 SQLite、Schema、配置、锁和 WAL。
5. 在测试群进行真实 Webhook/GET、双群、Card/Reply/Base/Doc/Task 验收。
6. 保存真实平台证据后才能宣称正式环境完成。

当前部署材料没有完成多实例高可用、完整 GitHub Actions、Linux Race、Windows CI 或真实平台验收。
