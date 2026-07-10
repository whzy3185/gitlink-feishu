#!/usr/bin/env bash
# compliance-repro-workflow.sh — S3 科研项目合规与复现性检查 · 可复现执行脚本
# 子赛题四「应用 GitLink 辅助科研」交付物之一
#
# 用法： bash compliance-repro-workflow.sh <OWNER> <REPO> [OUT_DIR]
# 示例： bash compliance-repro-workflow.sh mindspore-Ecosystem mindspore ./out
set -euo pipefail

OWNER="${1:?用法: $0 <OWNER> <REPO> [OUT_DIR]}"
REPO="${2:?缺少 REPO}"
OUT_DIR="${3:-./compliance-repro-output}"

# 定位仓库根（脚本位于 skills/gitlink-compliance/examples/）
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../../.." && pwd)"
REPRO="$REPO_ROOT/scripts/research/repro.py"

# 本地 Windows 开发可设 GITLINK_CLI=./gitlink-cli.exe；Linux/容器走 PATH 默认值
: "${GITLINK_CLI:=gitlink-cli}"

echo "==> 目标仓库: $OWNER/$REPO"
echo "==> CLI: $GITLINK_CLI"
echo "==> 输出目录: $OUT_DIR"

mkdir -p "$OUT_DIR"
GITLINK_CLI="$GITLINK_CLI" python "$REPRO" \
  --owner "$OWNER" --repo "$REPO" \
  --out "$OUT_DIR"

echo
echo "==> 产物："
ls -1 "$OUT_DIR"
echo
echo "==> 合规/复现检查摘要："
python -c "
import json
d=json.load(open('$OUT_DIR/repro.json',encoding='utf-8'))
print('许可证:', d['license'])
print('复现分: %s/10' % d['repro_score'])
print('合规分: %s/10' % d['compliance_score'])
print('风险项: %d 处' % len(d['risks']))
for r in d['risks'][:5]:
    loc = (r.get('file','') + ':' + str(r.get('line',''))) if r.get('file') else '-'
    print('  [%s] %s (%s) %s' % (r.get('level','medium'), r.get('name',''), loc, r.get('evidence') or r.get('detail','')))
"
