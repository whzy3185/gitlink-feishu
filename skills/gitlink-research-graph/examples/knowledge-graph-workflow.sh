#!/usr/bin/env bash
# knowledge-graph-workflow.sh — S2 科研热点追踪与知识图谱 · 可复现执行脚本
# 子赛题四「应用 GitLink 辅助科研」交付物之一
#
# 用法： bash knowledge-graph-workflow.sh <KEYWORDS> [OUT_DIR] [REPOS_LIMIT]
# 示例： bash knowledge-graph-workflow.sh "deep learning,nlp" ./out 20
set -euo pipefail

KEYWORDS="${1:?用法: $0 <KEYWORDS> [OUT_DIR] [REPOS_LIMIT] 例如: $0 \"deep learning,nlp\" ./out 20}"
OUT_DIR="${2:-./knowledge-graph-output}"
REPOS_LIMIT="${3:-20}"

# 定位仓库根（脚本位于 skills/gitlink-research-graph/examples/）
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../../.." && pwd)"
GRAPH="$REPO_ROOT/scripts/research/graph_build.py"

# 本地 Windows 开发可设 GITLINK_CLI=./gitlink-cli.exe；Linux/容器走 PATH 默认值
: "${GITLINK_CLI:=gitlink-cli}"

echo "==> 关键词: $KEYWORDS"
echo "==> CLI: $GITLINK_CLI"
echo "==> 仓库上限: $REPOS_LIMIT"

mkdir -p "$OUT_DIR"
GITLINK_CLI="$GITLINK_CLI" python "$GRAPH" \
  --keywords "$KEYWORDS" \
  --repos-limit "$REPOS_LIMIT" \
  --out "$OUT_DIR"

echo
echo "==> 产物："
ls -1 "$OUT_DIR"
echo
echo "==> 图谱概览："
python -c "
import json
d=json.load(open('$OUT_DIR/graph.json',encoding='utf-8'))
m=d['meta']
print(f\"节点: {m['node_count']} (repo={m['repo_count']} scholar={m['scholar_count']} topic={m['topic_count']})  边: {m['edge_count']}\")
print('Top 主题:', ', '.join(h['topic'] for h in d['topic_heat'][:5]))
print('核心学者:', ', '.join(f\"{s['login']}({s['repo_count']})\" for s in d['core_scholars'][:5]))
"
