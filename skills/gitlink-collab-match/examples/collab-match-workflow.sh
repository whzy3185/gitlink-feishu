#!/usr/bin/env bash
# collab-match-workflow.sh — S4 科研协作智能匹配 · 可复现执行脚本
# 子赛题四「应用 GitLink 辅助科研」交付物之一
#
# 用法： bash collab-match-workflow.sh <OWNER> <REPO> [OUT_DIR] [POOL] [TOP] [ISSUE_SAMPLE]
# 示例： bash collab-match-workflow.sh mindspore-Ecosystem mindspore ./out 15 10 100
set -euo pipefail

OWNER="${1:?用法: $0 <OWNER> <REPO> [OUT_DIR] [POOL] [TOP] [ISSUE_SAMPLE]}"
REPO="${2:?缺少 REPO}"
OUT_DIR="${3:-./collab-match-output}"
POOL="${4:-15}"
TOP="${5:-10}"
ISSUE_SAMPLE="${6:-100}"

# 定位仓库根（脚本位于 skills/gitlink-collab-match/examples/）
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../../.." && pwd)"
MATCH="$REPO_ROOT/scripts/research/match.py"

# 本地 Windows 开发可设 GITLINK_CLI=./gitlink-cli.exe；Linux/容器走 PATH 默认值
: "${GITLINK_CLI:=gitlink-cli}"

echo "==> 目标仓库: $OWNER/$REPO"
echo "==> CLI: $GITLINK_CLI"
echo "==> 候选池=$POOL  Top=$TOP  Issue采样=$ISSUE_SAMPLE"

mkdir -p "$OUT_DIR"
GITLINK_CLI="$GITLINK_CLI" python "$MATCH" \
  --owner "$OWNER" --repo "$REPO" \
  --pool "$POOL" --top "$TOP" --issue-sample "$ISSUE_SAMPLE" \
  --out "$OUT_DIR"

echo
echo "==> 产物："
ls -1 "$OUT_DIR"
echo
echo "==> Top 推荐预览："
python -c "
import json,sys
d=json.load(open('$OUT_DIR/match.json',encoding='utf-8'))
print('缺口主题:', ', '.join(d['gap_topics']))
for i,m in enumerate(d['candidates'],1):
    print(f\"  {i}. {m['login']} ({m['score']}分) — {'; '.join(m['reasons'][:2])}\")
"
