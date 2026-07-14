#!/usr/bin/env bash
# 一键构建 demo 所需的 Linux gitlink-cli 二进制（本地测试 Dockerfile 用）
# 用法：bash demo/build-demo.sh   → 产物 demo/bin/gitlink-cli
set -e
DIR="$(cd "$(dirname "$0")" && pwd)"
ROOT="$(cd "$DIR/.." && pwd)"          # 仓库根
OUT="$DIR/bin"
mkdir -p "$OUT"
echo "→ 在 $ROOT 编译 Linux amd64 二进制..."
( cd "$ROOT" && CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o "$OUT/gitlink-cli" . )
echo "✓ 产出：$OUT/gitlink-cli"
echo "  本地测容器：docker build -f $DIR/Dockerfile -t gitlink-cli-demo '$ROOT'"
echo "            docker run --rm -p 8000:8000 gitlink-cli-demo"
