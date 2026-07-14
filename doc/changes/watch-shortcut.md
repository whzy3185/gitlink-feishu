# Watch shortcut

新增 `watch` Shortcut 组，封装 GitLink 仓库关注（订阅）操作：

- `watch +watch`     关注指定仓库
- `watch +unwatch`   取消关注
- `watch +watchers`  列出仓库关注者

实现要点：

- `+watch` 调用 `POST /{owner}/{repo}/watchers/follow`；`+unwatch` 调用 `DELETE /{owner}/{repo}/watchers/unfollow`。
- `+watchers` 调用 `GET /{owner}/{repo}/watchers` 列出关注者。
- 复用 RuntimeContext，自动注入 owner/repo/auth，与现有 Shortcut 组风格一致。

含单元测试 `shortcuts/watch/watch_test.go`。

## Examples

```bash
gitlink-cli watch +watch    --owner Gitlink --repo gitlink-cli
gitlink-cli watch +unwatch  --owner Gitlink --repo gitlink-cli
gitlink-cli watch +watchers --owner Gitlink --repo gitlink-cli
```

## Tests

```bash
go test ./shortcuts/watch/...
```
