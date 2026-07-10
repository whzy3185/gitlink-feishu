#!/usr/bin/env bash
# ============================================================
# 4 个 Skill 登录实演 · 辅助采集脚本
# 作用：把每个 Skill 的「只读采集命令」串起来自动跑，展示真实数据
#       分析（评估/聚合/排序）部分由 AI 在 Claude Code 里做——那才是亮点
# 用法：bash demo/live-demo.sh <owner> <repo> [your-username]
#   例：bash demo/live-demo.sh myorg myproject zhangsan
# 前置：先 gitlink-cli auth login
# ============================================================

_DIR="$(cd "$(dirname "$0")" && pwd)"
CLI="$_DIR/../gitlink-cli.exe"; [ -f "$CLI" ] || CLI="$_DIR/../gitlink-cli"   # Win→.exe，Linux→无后缀
OWNER="${1:-}"; REPO="${2:-}"; ME="${3:-$OWNER}"

G='\033[0;32m'; Y='\033[1;33m'; C='\033[0;36m'; R='\033[0;31m'; B='\033[1m'; N='\033[0m'
banner() { echo -e "\n${B}══════════════════════════════════════════════════════${N}"; echo -e "${B}  $1${N}"; echo -e "${B}══════════════════════════════════════════════════════${N}"; }
section() { echo -e "\n${C}━━━ $1 ━━━${N}"; }
run() { echo -e "${Y}❯ $1${N}"; eval "$1" 2>&1 | head -16; echo; }

# ---------- 前置检查 ----------
banner "4 Skill 登录实演 · 辅助采集"
[ -f "$CLI" ] || { echo -e "${R}✗ 找不到 gitlink-cli${N}"; exit 1; }
if [ -z "$OWNER" ] || [ -z "$REPO" ]; then
  echo -e "${R}用法: bash $0 <owner> <repo> [your-username]${N}"
  echo -e "${R}例  : bash $0 myorg myproject zhangsan${N}"; exit 1
fi
echo -e "${G}✓${N} 目标仓库: ${B}$OWNER/$REPO${N}，当前用户: ${B}$ME${N}"
echo -e "${C}提示：本脚本只跑只读采集命令；分析（评估/聚合/排序）请在 Claude Code 里让 AI 做${N}"

# ---------- 0. auth：登录状态 ----------
section "auth · 登录状态"
run "\"$CLI\" auth status"

# ---------- 1. onboarding：找新手任务 ----------
section "onboarding · 新手 Issue + 项目概览（供 AI 做 5 维度评估）"
run "\"$CLI\" search +issues --owner $OWNER --repo $REPO --keyword 'good first issue' --category opened --format json"
run "\"$CLI\" repo +info --owner $OWNER --repo $REPO --format json"

# ---------- 2. digest：多源数据（供 AI 聚合成简报）----------
section "digest · Issue / PR / CI / 通知（供 AI 跨源聚合）"
run "\"$CLI\" issue +list --owner $OWNER --repo $REPO --state open --format json"
run "\"$CLI\" pr +list --owner $OWNER --repo $REPO --format json"
run "\"$CLI\" ci +builds --owner $OWNER --repo $REPO --format json"
run "\"$CLI\" api GET \"users/$ME/messages.json\""

# ---------- 3. todo：个人待办数据（供 AI 排序）----------
section "todo · 分配给我的 Issue + @我消息（供 AI 排序成待办）"
run "\"$CLI\" api GET \"users/me\" --format json"
run "\"$CLI\" search +issues --assignee $ME --category opened --format json"
run "\"$CLI\" api GET \"users/$ME/messages.json\""

# ---------- 总结 ----------
banner "采集完成"
echo -e "${B}接下来${N}：在 Claude Code 里用自然语言触发，让 AI 读对应 SKILL.md 分析以上数据："
echo -e "  • ${C}「找适合新手的任务」${N} → onboarding 的 5 维度评估"
echo -e "  • ${C}「给我项目简报」${N}     → digest 的跨源聚合"
echo -e "  • ${C}「我的待办有哪些」${N}   → todo 的紧急度排序"
echo -e "\n详见 ${Y}live-demo-guide.md${N}"
read -p "按回车键继续..."
