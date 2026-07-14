#!/usr/bin/env bash
# research-insight-workflow.sh — S1 仓库级科研项目洞悉 · 可复现执行脚本
# 子赛题四「应用 GitLink 辅助科研」交付物之一（Go 出数据 + Python 做算法）
#
# 用法： bash research-insight-workflow.sh <OWNER> <REPO> [OUT_DIR] [BRANCHES_LIMIT]
# 示例： bash research-insight-workflow.sh mindspore-Ecosystem mindspore ./out 5
set -euo pipefail

OWNER="${1:?用法: $0 <OWNER> <REPO> [OUT_DIR] [BRANCHES_LIMIT]}"
REPO="${2:?缺少 REPO}"
OUT_DIR="${3:-./research-insight-output}"
BRANCHES_LIMIT="${4:-5}"

# 定位仓库根（脚本位于 skills/gitlink-research-insight/examples/）
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../../.." && pwd)"
LINEAGE="$REPO_ROOT/scripts/research/lineage.py"

# 本地 Windows 开发可设 GITLINK_CLI=./gitlink-cli.exe；Linux/容器走 PATH 默认值
: "${GITLINK_CLI:=gitlink-cli}"

echo "==> 目标仓库: $OWNER/$REPO"
echo "==> CLI: $GITLINK_CLI"
echo "==> 分支地图上限=$BRANCHES_LIMIT"

mkdir -p "$OUT_DIR"
GITLINK_CLI="$GITLINK_CLI" python "$LINEAGE" \
  --owner "$OWNER" --repo "$REPO" \
  --branches-limit "$BRANCHES_LIMIT" \
  --out "$OUT_DIR"

echo
echo "==> 产物："
ls -1 "$OUT_DIR"
echo
echo "==> 洞悉摘要预览："
python -c "
import json
d=json.load(open('$OUT_DIR/lineage.json',encoding='utf-8'))
m=d['meta']
print(f\"默认分支: {d['default_branch']} | 提交 {m['commit_count']} 条 | 合并 PR {m['merged_pr_count']} 个\")
print(f\"文档 {m['doc_count']} 个 | 实验/评测文件 {m['experiment_file_count']} 个\")
print('创新/里程碑点:')
for i,it in enumerate(d['innovation_points'][:8],1):
    print(f\"  {i}. [{it['category']}] {it['description']}\")
"
