#!/usr/bin/env bash
# research-visual-workflow.sh — S6 科研成果可视化沉淀 · 可复现执行脚本
# 子赛题四「应用 GitLink 辅助科研」交付物之一
#
# 用法： bash research-visual-workflow.sh <OWNER> <REPO> [OUT_DIR] [WEEKS]
# 示例： bash research-visual-workflow.sh mindspore-Ecosystem mindspore ./out 26
set -euo pipefail

OWNER="${1:?用法: $0 <OWNER> <REPO> [OUT_DIR] [WEEKS]}"
REPO="${2:?缺少 REPO}"
OUT_DIR="${3:-./research-visual-output}"
WEEKS="${4:-26}"

# 定位仓库根（脚本位于 skills/gitlink-research-visual/examples/）
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../../.." && pwd)"
VISUAL="$REPO_ROOT/scripts/research/visual.py"

# 本地 Windows 开发可设 GITLINK_CLI=./gitlink-cli.exe；Linux/容器走 PATH 默认值
: "${GITLINK_CLI:=gitlink-cli}"

echo "==> 目标仓库: $OWNER/$REPO"
echo "==> CLI: $GITLINK_CLI"
echo "==> 回溯周数: $WEEKS"

mkdir -p "$OUT_DIR"
GITLINK_CLI="$GITLINK_CLI" python "$VISUAL" \
  --owner "$OWNER" --repo "$REPO" \
  --weeks "$WEEKS" \
  --out "$OUT_DIR"

echo
echo "==> 产物："
ls -1 "$OUT_DIR"
echo
echo "==> 成果概览："
python -c "
import json
d=json.load(open('$OUT_DIR/visual.json',encoding='utf-8'))
m=d['meta']; a=d['artifact_summary']
print(f\"仓库: {d['repo']}  回溯 {d['weeks']} 周\")
print(f\"活跃度: commits={m['commit_count']} issues={m['issue_count']} prs={m['pr_count']} 贡献者={m['contributor_count']}\")
print(f\"产物分类: paper={a['paper']} dataset={a['dataset']} model={a['model']} benchmark={a['benchmark']}\")
print(f\"抽取论文引用: {len(d['paper_links'])} 条\")
tl=d['timeline']
peak=max(tl['commits']) if tl['commits'] else 0
print(f\"开发节奏(最近窗口): 峰值 {peak} 提交/周\")
"
