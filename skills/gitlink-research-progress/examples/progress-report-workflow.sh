#!/usr/bin/env bash
# progress-report-workflow.sh — S5 科研进度周报 · 可复现执行脚本
# 子赛题四「应用 GitLink 辅助科研」交付物之一
#
# 用法： bash progress-report-workflow.sh <OWNER> <REPO> [OUT_DIR]
# 示例： bash progress-report-workflow.sh mindspore-Ecosystem mindspore ./out
set -euo pipefail

OWNER="${1:?用法: $0 <OWNER> <REPO> [OUT_DIR]}"
REPO="${2:?缺少 REPO}"
OUT_DIR="${3:-./progress-output}"

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../../.." && pwd)"
REPORT="$REPO_ROOT/scripts/research/report.py"

: "${GITLINK_CLI:=gitlink-cli}"

echo "==> 目标仓库: $OWNER/$REPO  (CLI: $GITLINK_CLI)"
mkdir -p "$OUT_DIR"
GITLINK_CLI="$GITLINK_CLI" python "$REPORT" --owner "$OWNER" --repo "$REPO" --out "$OUT_DIR"

echo
echo "==> 产物："; ls -1 "$OUT_DIR"
echo
echo "==> 摘要："
python -c "
import json
d=json.load(open('$OUT_DIR/report.json',encoding='utf-8'))
tw=d['week_stats']['this_week']; tr=d['trend']
print(f\"本周提交 {tw['commits']}，活跃贡献者 {tw['contributors_active']}，趋势 {tr['activity_level']}({tr['commit_delta_pct']}%)\")
print(f\"风险预警 {len(d['risk_warnings'])} 条:\")
for w in d['risk_warnings']:
    print(f\"  [{w['level']}] {w['type']}: {w['message']}\")
"
