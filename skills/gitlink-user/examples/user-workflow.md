# 用户信息完整工作流示例

**场景**：开发者需要查看 GitLink 用户信息。

## 工作流步骤

### Step 1：查看当前用户信息

```bash
# 查看当前登录用户
gitlink-cli auth status
```

### Step 2：查看其他用户信息

```bash
# 通过 Raw API 获取用户信息
gitlink-cli api GET /users/me --format json

# 获取指定用户信息
gitlink-cli api GET /users/<username> --format json
```

### Step 3：搜索用户

```bash
# 通过搜索命令查找用户
gitlink-cli search +users --keyword "zhangsan"
```

---

## 完整命令速览

```bash
gitlink-cli auth status
gitlink-cli api GET /users/me --format json
gitlink-cli search +users --keyword "<keyword>"
```
