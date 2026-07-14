# 跨平台兼容性验证报告

> 关联 Issue: #16 | PR: #13

## 测试矩阵

| 验证项 | Windows 11 | macOS | Ubuntu |
|--------|:---:|:---:|:---:|
| `git clone` + `go build ./...` | 待验证 | 待验证 | 待验证 |
| `go test -race ./...` 全部通过 | 待验证 | 待验证 | 待验证 |
| `gitlink-cli auth login` 登录 | 待验证 | 待验证 | 待验证 |
| `gitlink-cli repo +list` 可用 | 待验证 | 待验证 | 待验证 |
| `gitlink-cli pm +dashboards` 可用 | 待验证 | 待验证 | 待验证 |
| `gitlink-cli wiki +pages` 可用 | 待验证 | 待验证 | 待验证 |
| `gitlink-cli alias +list` 可用 | 待验证 | 待验证 | 待验证 |
| `gitlink-cli browse` 打开浏览器 | 待验证 | 待验证 | 待验证 |
| `gitlink-cli status` 显示状态 | 待验证 | 待验证 | 待验证 |
| `gitlink-cli export +issues` 导出 | 待验证 | 待验证 | 待验证 |
| Token 存储（keyring）正常 | 待验证 | 待验证 | 待验证 |

## 安装脚本

| 脚本 | 平台 | 路径 |
|------|------|------|
| install.sh | Linux / macOS | `scripts/install.sh` |
| install.ps1 | Windows | `scripts/install.ps1` |

## CI 配置

| 配置文件 | 说明 |
|---------|------|
| `.devops/ci.yml` | 建木流水线：push 到 wyx_branch 时自动触发构建+测试+格式化检查 |

### CI 检查内容

| 检查项 | 命令 | 说明 |
|--------|------|------|
| 构建 | `go build ./...` | 确保代码编译通过 |
| 静态分析 | `go vet ./...` | 检测常见代码问题 |
| 测试 | `go test -race ./...` | 运行全部测试，含竞态检测 |
| 格式化 | `gofmt -s -l .` | 确保代码格式符合 Go 标准 |

## 已知问题

1. **Windows keyring**: Windows Credential Manager 可能需要额外配置
2. **Linux keyring**: 需要 dbus 服务支持，无桌面环境时可能不可用
3. **browse 命令**: Linux 环境需要安装 xdg-utils 包
