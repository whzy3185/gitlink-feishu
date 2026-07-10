#!/usr/bin/env bash
# ============================================================
# 代码质量看门人 · 端到端工作流脚本（子任务三）
# 串联 5 步：采集 PR → AI Review → CI 检查 → 汇总评论 → 质量判定
# 用法：bash demo/pr-guard-workflow.sh <owner> <repo> <pr_id>
#   例：bash demo/pr-guard-workflow.sh myorg myproject 42
# 前置：gitlink-cli auth login（涉及平台 API）
# 说明：采集命令真实执行；Review 分析由 AI Agent（读 pr-guard/SKILL.md）完成
# ============================================================

_DIR="$(cd "$(dirname "$0")" && pwd)"
CLI="$_DIR/../gitlink-cli.exe"; [ -f "$CLI" ] || CLI="$_DIR/../gitlink-cli"   # Win→.exe，Linux→无后缀
OWNER="${1:-}"; REPO="${2:-}"; PR_ID="${3:-}"

G='\033[0;32m'; Y='\033[1;33m'; C='\033[0;36m'; R='\033[0;31m'; B='\033[1m'; N='\033[0m'
banner() { echo -e "\n${B}══════════════════════════════════════════════════════${N}"; echo -e "${B}  $1${N}"; echo -e "${B}══════════════════════════════════════════════════════${N}"; }
step()  { echo -e "\n${C}━━━ Step $1 ━━━ ${B}$2${N}"; }
ai()    { echo -e "🤖 ${G}AI（读 pr-guard/SKILL.md 后）:${N} $1"; }
run()   { echo -e "${Y}❯ $1${N}"; eval "$1" 2>&1 | head -16; echo; }

# ---------- 前置检查 ----------
banner "代码质量看门人 · PR #${PR_ID:-?} 质量流水线"
[ -f "$CLI" ] || { echo -e "${R}✗ 找不到 gitlink-cli${N}"; exit 1; }
if [ -z "$OWNER" ] || [ -z "$REPO" ] || [ -z "$PR_ID" ]; then
  echo -e "${R}用法: bash $0 <owner> <repo> <pr_id>${N}"
  echo -e "${R}例  : bash $0 myorg myproject 42${N}"; exit 1
fi
echo -e "${G}✓${N} 目标: ${B}$OWNER/$REPO${N} PR #${B}$PR_ID${N}"

# ---------- Step 1：采集 PR 变更 ----------
step 1 "采集 PR 变更（pr +view / +files / +diff）"
ai "先拉 PR 详情、变更文件、diff 统计，作为审查输入。"
run "\"$CLI\" pr +view --id $PR_ID --format json"
run "\"$CLI\" pr +files --id $PR_ID --format json"
run "\"$CLI\" pr +diff --id $PR_ID --stat"

# ---------- Step 2：AI Review（复用 code-review 逻辑）----------
step 2 "AI Review（按 code-review 分级找问题）"
ai "逐文件分析 diff，按安全红线/错误处理/规范分级。这一步由 AI Agent 完成（读 code-review/SKILL.md）。"
echo -e "  ${C}分级框架${N}："
echo -e "    🔴 Critical：硬编码密钥 / SQL·命令注入 / 路径遍历（安全红线，阻断合并）"
echo -e "    🟡 Warning ：错误处理缺失 / 边界条件 / 明文敏感信息"
echo -e "    🔵 Suggestion：命名 / 性能 / 可配置化"
echo -e "  ${C}Agent 在此输出分级清单（示例见 SKILL.md 输出模板）${N}"

# ---------- Step 3：CI 检查 ----------
step 3 "CI 检查（ci +builds）"
ai "查 PR 对应分支的最新构建状态，作为门禁第二维。"
run "\"$CLI\" ci +builds --owner $OWNER --repo $REPO --format json"
echo -e "  ${C}判定：从返回按 source_branch 匹配最新构建 → success / failure / pending${N}"

# ---------- Step 4：发布汇总评论 ----------
step 4 "发布质量看门人报告（api POST .../reviews）"
ai "把 Review 意见 + CI 状态 + 质量判定组装成报告，评论到 PR。"
echo -e "${Y}❯ gitlink-cli api POST /$OWNER/$REPO/pulls/$PR_ID/reviews --body '<报告>'${N}"
echo -e "  ${C}报告含${N}：质量判定 + Critical/Warning 清单 + CI 状态 + 处置建议"
echo -e "  ${C}[实演时此处真实发送；脚本演示仅展示结构]${N}"

# ---------- Step 5：质量判定 ----------
step 5 "质量判定（门禁规则 → 合并 / 请求修改）"
ai "按门禁规则决策。注意：合并是写操作，默认只建议，确认后才执行。"
echo -e "  ${C}门禁规则${N}："
echo -e "    0 Critical + CI success        → ✅ 通过，建议合并"
echo -e "    有 Critical 任一                → 🔴 拒绝，请求修改"
echo -e "    CI failure                      → 🔴 拒绝，附 CI 日志"
echo -e "    仅 Warning/Suggestion           → 🟡 通过(带建议)"
echo ""
echo -e "  ${G}若判定通过 + 用户确认 →${N} ${Y}gitlink-cli pr +merge --id $PR_ID --method squash${N}"

# ---------- 总结 ----------
banner "流水线完成"
echo -e "${B}代码质量看门人${N} 串联了 ${B}5 步${N}，覆盖 ${B}4 个 CLI 域${N}："
echo -e "  pr（采集/合并）+ code-review（Review）+ ci（构建）+ api（评论）"
echo -e "\n${C}真实演示${N}：登录后对本仓库一个真实 PR 跑此脚本，由 AI 完成 Step 2 分析。"
echo -e "详见 ${Y}pr-guard-architecture.md${N}（工作流说明 + 架构图）"
read -p "按回车键继续..."
