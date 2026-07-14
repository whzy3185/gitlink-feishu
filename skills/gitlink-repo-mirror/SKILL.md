---
name: gitlink-repo-mirror
version: 1.0.0
description: "GitLink 镜像仓库同步：触发镜像仓库 sync_mirror，支持 dry-run。"
metadata:
  requires:
    bins: ["gitlink-cli"]
  cliHelp: "gitlink-cli repo-mirror --help"
---

# gitlink-repo-mirror

镜像同步是写操作，先 dry-run 并确认。

```bash
gitlink-cli repo-mirror +sync --id 42 --dry-run
gitlink-cli repo-mirror +sync --id 42
```

对应 OpenAPI：`POST /repositories/{id}/sync_mirror`。
