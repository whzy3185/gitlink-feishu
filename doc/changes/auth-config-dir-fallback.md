# 认证凭据 fallback 配置目录一致性修复

`gitlink-cli` 的主配置文件已经支持通过 `GITLINK_CONFIG_DIR` 指定配置目录，但认证模块在系统 Keychain 不可用时仍然把 fallback 凭据写到用户 home 下的 `~/.config/gitlink-cli/credentials`。这会让 CI、Windows 测试、Agent 沙箱和多账号隔离场景出现配置目录与凭据目录不一致的问题，也会导致测试中设置临时 HOME 后仍读写真实用户目录。

本次修复让文件凭据路径统一复用 `internal/config.ConfigDir()`：设置 `GITLINK_CONFIG_DIR` 时，fallback 凭据保存到 `$GITLINK_CONFIG_DIR/credentials`；未设置时仍保持原有默认路径。`auth logout` 在 fallback 文件不存在时也改为幂等成功，避免用户已经没有本地凭据时退出登录反而报错。

测试同步改为使用 `GITLINK_CONFIG_DIR` 隔离凭据目录，覆盖默认配置目录、文件创建、保存/读取/删除、Keychain 不可用 fallback、无凭据登出等场景。该修复提升了跨平台稳定性，也让本地全量测试不再因为 Windows `HOME`/`USERPROFILE` 解析差异污染真实用户凭据目录。
