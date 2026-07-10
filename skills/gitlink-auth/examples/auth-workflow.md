# 认证工作流示例

**场景**：首次使用 gitlink-cli 的用户需要完成登录认证。

## 工作流步骤

### Step 1：交互式登录

```bash
# 方式 1：用户名密码登录
gitlink-cli auth login
# 按提示输入用户名和密码

# 方式 2：使用已有 Token
gitlink-cli auth login --token
# 粘贴 GitLink 个人访问令牌
```

### Step 2：验证登录状态

```bash
gitlink-cli auth status
```

**输出示例：**
```
已登录为: zhangsan
Token 有效期至: 2026-06-19
存储位置: OS Keychain
```

### Step 3：Token 过期后重新登录

```bash
# 遇到 401 错误时，重新登录
gitlink-cli auth login

# 退出登录
gitlink-cli auth logout
```

---

## 注意事项

- GitLink Token 有效期 7 天，过期需重新登录
- Token 存储在系统密钥管理器中，安全可靠
- 非交互环境可设置环境变量 `GITLINK_TOKEN`
