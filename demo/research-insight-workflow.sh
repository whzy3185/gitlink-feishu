#!/usr/bin/env bash
# ============================================================
# 科研仓库画像 · 端到端工作流脚本（子任务四）
# 4 步：采集数据 → 四维评分 → 协作图谱 → 科研画像报告
# 用法：bash demo/research-insight-workflow.sh <owner> <repo>
#   例：bash demo/research-insight-workflow.sh someresearch awesome-paper-code
# 前置：gitlink-cli auth login（只读分析，不改数据）
# 说明：采集命令真实执行；四维评分 + 协作图谱由 AI（读 research-insight/SKILL.md）完成
# ============================================================

_DIR="$(cd "$(dirname "$0")" && pwd)"
CLI="$_DIR/../gitlink-cli.exe"; [ -f "$CLI" ] || CLI="$_DIR/../gitlink-cli"   # Win→.exe，Linux→无后缀
OWNER="${1:-}"; REPO="${2:-}"

G='\033[0;32m'; Y='\033[1;33m'; C='\033[0;36m'; R='\033[0;31m'; B='\033[1m'; N='\033[0m'
banner() { echo -e "\n${B}══════════════════════════════════════════════════════${N}"; echo -e "${B}  $1${N}"; echo -e "${B}══════════════════════════════════════════════════════${N}"; }
step()  { echo -e "\n${C}━━━ Step $1 ━━━ ${B}$2${N}"; }
ai()    { echo -e "🤖 ${G}AI（读 research-insight/SKILL.md 后）:${N} $1"; }
run()   { echo -e "${Y}❯ $1${N}"; eval "$1" 2>&1 | head -14; echo; }

# ---------- 前置检查 ----------
banner "🔬 科研仓库画像 · $OWNER/${REPO:-?}"
[ -f "$CLI" ] || { echo -e "${R}✗ 找不到 gitlink-cli${N}"; exit 1; }
if [ -z "$OWNER" ] || [ -z "$REPO" ]; then
  echo -e "${R}用法: bash $0 <owner> <repo>${N}"
  echo -e "${R}例  : bash $0 someresearch awesome-paper-code${N}"; exit 1
fi
echo -e "${G}✓${N} 分析对象: ${B}$OWNER/$REPO${N}（只读，不改数据）"

# ---------- Step 0：fork 检测（避免给 fork 错评） ----------
step 0 "fork 检测（引用价值要改评 upstream）"
ai "先看 repo +info 的 fork_info。是 fork 则引用价值/活跃度改评 upstream。"
INFO="$("$CLI" repo +info --owner "$OWNER" --repo "$REPO" --format json 2>/dev/null)"
UPSTREAM="$(printf '%s' "$INFO" | grep -o '"fork_project_user_login": *"[^"]*"' | head -1 | sed 's/.*: *"//;s/"$//')"
# fork_project_user_login 缺失或为 null → 非空才算 fork
[ "$UPSTREAM" = "null" ] && UPSTREAM=""
if [ -n "$UPSTREAM" ]; then
  echo -e "  ${R}⚠️ $OWNER/$REPO 是 ${B}$UPSTREAM/$REPO${N}${R} 的 fork —— 引用价值应改评 upstream ${B}$UPSTREAM/$REPO${N}"
else
  echo -e "  ${G}✓${N} 独立仓库（非 fork），正常评估"
fi

# ---------- Step 1：采集科研仓库数据 ----------
step 1 "采集科研仓库数据（repo / file / issue / pr + 本地 git 兜底）"
ai "拉基础画像、活跃度、合规复现性数据，作为科研评估输入。"
echo -e "${Y}❯ 基础画像（repo +info / +languages / +contributors）${N}"
"$CLI" repo +info --owner "$OWNER" --repo "$REPO" --format json 2>&1 | head -14
"$CLI" repo +languages --owner "$OWNER" --repo "$REPO" --format json 2>&1 | head -8
"$CLI" repo +contributors --owner "$OWNER" --repo "$REPO" --format json 2>&1 | head -12
echo -e "\n${Y}❯ 复现性文件（file +get 读 LICENSE / CI —— 替代不存在的 repo +raw）${N}"
"$CLI" file +get --owner "$OWNER" --repo "$REPO" --path LICENSE --format json 2>&1 | head -3
"$CLI" repo +tree --owner "$OWNER" --repo "$REPO" --path .gitea/workflows --format json 2>&1 | head -6
echo -e "\n${Y}❯ 版本归档（release +list —— 替代不存在的 repo +tags）${N}"
"$CLI" release +list --owner "$OWNER" --repo "$REPO" --format json 2>&1 | head -6
echo -e "\n${Y}❯ 活跃度（issue/pr + git 兜底 —— repo +commits 不存在）${N}"
"$CLI" issue +list --owner "$OWNER" --repo "$REPO" --format json 2>&1 | head -8
"$CLI" pr +list --owner "$OWNER" --repo "$REPO" --format json 2>&1 | head -8
echo -e "  ${C}repo +commits 不存在 → 用 git clone 兜底读提交时间线${N}"
TMP="/tmp/${OWNER}-${REPO}-analyze"; rm -rf "$TMP"
if git clone --quiet --depth 100 "https://gitlink.org.cn/$OWNER/$REPO.git" "$TMP" 2>/dev/null; then
  echo -e "  近 3 月提交: $(git -C "$TMP" log --oneline --since='3 months ago' 2>/dev/null | wc -l) 次"
  echo -e "  最近提交  : $(git -C "$TMP" log -1 --format='%ci %an' 2>/dev/null)"
  echo -e "  tag 列表  : $(git -C "$TMP" tag 2>/dev/null | tr '\n' ' ')"
  echo -e "  PR 合并数 : $(git -C "$TMP" log --merges --oneline 2>/dev/null | wc -l)"
else
  echo -e "  ${R}✗ git clone 失败（无 git 或无网络）→ 活跃度改用 repo +info 计数近似${N}"
fi

# ---------- Step 2：四维科研评分 ----------
step 2 "四维科研评分（AI 按指标体系打分）"
ai "对采集数据按科研四维评分。这一步由 AI 完成（指标体系见 SKILL.md）。"
echo -e "  ${C}🔁 可复现性${N}（科研核心，满分10）：CI(+2) / 依赖锁定(+2) / 数据说明(+2) / 运行文档(+2) / 版本归档(+2)"
echo -e "  ${C}📈 活跃度${N}：近3月提交频率 + Issue/PR 活跃 + 贡献者趋势"
echo -e "  ${C}📑 引用价值${N}：LICENSE + 版本归档 + 文档完整 + 星标"
echo -e "  ${C}🤝 协作健康${N}：Issue响应 + PR合并率 + 巴士因子（核心贡献者占比）"
echo -e "  ${C}Agent 在此输出各维度得分 + 判定依据${N}"

# ---------- Step 3：协作知识图谱 ----------
step 3 "协作知识图谱（贡献者协作网络）"
ai "从贡献者 + PR 协作数据生成 mermaid 协作网络，呼应『知识图谱』要求。"
cat <<'MERMAID'
  graph LR
    A[核心贡献者1] -->|主提交| P((项目))
    B[核心贡献者2] -->|主提交| P
    C[偶发贡献者] -->|贡献| P
    A -.评审.-> C
    B -.评审.-> C
MERMAID
echo -e "  ${C}巴士因子${N}：核心贡献者提交占比 → <健康 / 单点风险>"

# ---------- Step 4：科研画像报告 ----------
step 4 "生成科研画像报告"
ai "组装成《科研仓库画像报告》：一句话定性 + 综合评分 + 四维详情 + 协作图 + 引用/复现/合作建议。"
echo -e "  ${C}报告含${N}：🔬综合评分 / 📋基础信息 / 🔁可复现性详情 / 🤝协作网络图 / 💡给科研工作者建议"
echo -e "  ${C}模板见 SKILL.md「输出模板」+ research-insight-guide.md${N}"

# ---------- 总结 ----------
banner "分析完成"
echo -e "${B}科研仓库画像${N} 串联 ${B}5 步${N}（fork 检测 + 采集 + 评分 + 图谱 + 报告），覆盖 CLI 域（只读）："
echo -e "  repo / file / release / issue / pr + 本地 git（提交时间线兜底，repo +commits 不存在）"
echo -e "\n${C}科研视角创新${N}：可复现性评分 + 引用价值 + 协作知识图谱（区别于普通 health 工程视角）"
echo -e "${C}真实验证${N}：登录后对 GitLink 一个科研类仓库跑此脚本，由 AI 完成评分 → 产出报告 + 截图"
echo -e "详见 ${Y}research-insight-guide.md${N}（完整中文使用文档 + 报告样例）"
read -p "按回车键继续..."
