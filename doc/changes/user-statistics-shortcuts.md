# User Statistics Shortcuts

## Background

GitLink OpenAPI exposes user analytics endpoints, including activity, contribution heatmap, development capability, role distribution, and professional categories. gitlink-cli previously only exposed `user +me` and `user +info`.

## What Changed

Extended the `user` shortcut group with read-only statistics commands:

- `user +activity`
- `user +headmap`
- `user +develop`
- `user +role`
- `user +major`

The commands support `--login`, with fallback to `--owner` or `/users/me`. Range-based endpoints support `--start-time` and `--end-time`; heatmap supports `--year`.

## OpenAPI Coverage

- `GET /users/{owner}/statistics/activity`
- `GET /users/{owner}/headmaps`
- `GET /users/{owner}/statistics/develop`
- `GET /users/{owner}/statistics/role`
- `GET /users/{owner}/statistics/major`

## Validation

```bash
git diff --check
GOPROXY=https://goproxy.cn,direct go test ./shortcuts/user ./shortcuts
go vet ./shortcuts/user ./shortcuts
go run . user --help
GOPROXY=https://goproxy.cn,direct go test ./...
go vet ./...
```
