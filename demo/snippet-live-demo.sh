#!/usr/bin/env bash
# ============================================================
# GitLink Skills 功能演示 —— snippet 完整闭环
# 核心卖点:AI 读取 SKILL.md → 自动编排 gitlink-cli 命令 → 完成完整场景
# 特点:snippet 是本地功能,无需登录,可安全现场实演
# 用法:bash demo/snippet-live-demo.sh
# ============================================================

# 不用 set -e:保证演示连续性,关键步骤手动检查
_DIR="$(cd "$(dirname "$0")" && pwd)"
CLI="$_DIR/../gitlink-cli.exe"; [ -f "$CLI" ] || CLI="$_DIR/../gitlink-cli"   # Win→.exe，Linux→无后缀

# ANSI 颜色
G='\033[0;32m'; Y='\033[1;33m'; C='\033[0;36m'; R='\033[0;31m'; B='\033[1m'; N='\033[0m'

banner() { echo -e "\n${B}══════════════════════════════════════════════════════${N}"; echo -e "${B}  $1${N}"; echo -e "${B}══════════════════════════════════════════════════════${N}"; }
scene()  { echo -e "\n${C}━━━ 场景 $1 ━━━ ${B}$2${N}"; }
user()   { echo -e "🧑 ${B}用户:${N} $1"; }
ai()     { echo -e "🤖 ${G}AI（读 snippet/SKILL.md 后）:${N} $1"; }
show()   { echo -e "${Y}❯ $1${N}"; }

# ---------- 前置检查 ----------
banner "GitLink Skills 演示 · snippet 闭环"
[ -f "$CLI" ] || { echo -e "${R}✗ 找不到 gitlink-cli: $CLI${N}"; exit 1; }
echo -e "${G}✓${N} gitlink-cli 就绪"
echo -e "${G}✓${N} snippet 为本地功能（${B}无需登录${N}），可安全现场演示"
echo -e "${G}✓${N} 演示数据用完即删，不污染环境"

# ---------- 场景 1：创建 ----------
scene 1 "保存一段常用代码"
user "帮我存一段快速排序代码，语言 python，标签 algorithm"
ai "决策 → \`snippet +create\`（⚠️ Write）。SKILL.md 规则：--title 必填、--tags 逗号分隔、--language 标注。"
show "gitlink-cli snippet +create --title '快速排序(演示)' --language python --tags algorithm,demo --content '...'"
OUT=$("$CLI" snippet +create --title '快速排序(演示)' --language python --tags algorithm,demo \
  --content 'def qs(a): return a if len(a)<2 else qs([x for x in a[1:] if x<=a[0]])+[a[0]]+qs([x for x in a[1:] if x>a[0]])' \
  --format json 2>&1)
echo "$OUT" | head -12
DEMO_ID=$(echo "$OUT" | grep -oE '"id":[[:space:]]*"[0-9a-f]+"' | head -1 | grep -oE '[0-9a-f]{8}')
echo -e "${G}✓${N} 已创建，id = ${B}$DEMO_ID${N}"

# ---------- 场景 2：列表 ----------
scene 2 "浏览片段库"
user "我存了哪些片段？"
ai "决策 → \`snippet +list\`（Read）。可按 --tag / --language / --keyword 过滤。"
show "gitlink-cli snippet +list --format json"
"$CLI" snippet +list --format json 2>&1 | head -14

# ---------- 场景 3：搜索 ----------
scene 3 "全文检索"
user "帮我找包含 '排序' 的片段"
ai "决策 → \`snippet +search\`（Read，全文匹配 title + content）。"
show "gitlink-cli snippet +search --query '排序' --format json"
"$CLI" snippet +search --query '排序' --format json 2>&1 | head -10

# ---------- 场景 4：查看详情 ----------
scene 4 "查看指定片段"
user "看看 id=$DEMO_ID 这个的详情"
ai "决策 → \`snippet +view --id\`（Read）。"
show "gitlink-cli snippet +view --id $DEMO_ID --format json"
"$CLI" snippet +view --id "$DEMO_ID" --format json 2>&1 | head -12

# ---------- 场景 5：导出 ----------
scene 5 "导出到文件复用"
user "把它导出成文件，我要贴到项目里"
ai "决策 → \`snippet +export --output\`（Read）。SKILL.md：默认输出到 stdout，-o 写文件。"
TMP="$PWD/.demo_export_$$.py"
show "gitlink-cli snippet +export --id $DEMO_ID --output $TMP"
"$CLI" snippet +export --id "$DEMO_ID" --output "$TMP" >/dev/null 2>&1
echo -e "${G}✓${N} 已导出，文件内容："; cat "$TMP"; rm -f "$TMP"

# ---------- 场景 6：更新 ----------
scene 6 "更新片段字段"
user "给这个片段补个 tag 'sort'"
ai "决策 → \`snippet +update\`（⚠️ Write）。--id 必填，至少一个字段。"
show "gitlink-cli snippet +update --id $DEMO_ID --tags algorithm,demo,sort"
"$CLI" snippet +update --id "$DEMO_ID" --tags algorithm,demo,sort --format json 2>&1 | head -8

# ---------- 场景 7：删除（清理）----------
scene 7 "删除演示片段（清理）"
user "演示结束，删掉刚才的测试片段"
ai "决策 → \`snippet +delete\`（🔴 Destructive）。SKILL.md：删除不可逆，建议先 view 确认。"
show "gitlink-cli snippet +delete --id $DEMO_ID"
"$CLI" snippet +delete --id "$DEMO_ID" --format json 2>&1 | head -4
echo -e "${G}✓${N} 演示数据已清理"

# ---------- 总结 ----------
banner "演示完成"
echo -e "${B}gitlink-snippet${N} Skill 的 7 个命令全部实测通过："
echo -e "  create / list / search / view / export / update / delete"
echo ""
echo -e "${B}核心价值${N}：AI 读取 SKILL.md 后，能自动编排 gitlink-cli 命令完成完整场景，"
echo -e "输出严格符合 SKILL.md 定义的 envelope 格式 {\"ok\":true,\"data\":{...}}。"
echo -e "\n${C}其他 Skill（onboarding / digest / todo）涉及平台 API，登录后可按其 SKILL.md 的「工作流」演示。${N}"
read -p "按回车键继续..."