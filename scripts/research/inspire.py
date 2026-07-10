"""
创新启发（Pillar 4）：缺口挖掘 + 合作者匹配 + 创新点 + LLM 研究方向建议

模式:
  --owner O --repo R        单仓：缺口/匹配/创新点 + LLM idea
  --category <域>           领域级：聚合分类下 top 仓的缺口 + 领域主题 + LLM idea

LLM 可选：配置 DEEPSEEK_API_KEY 才生成 idea；无 key 确定性产出照常。
产物：inspire.json + report.md。
"""
from __future__ import annotations

import argparse
import json
import os
import sys
from collections import Counter

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
import collect as c
import topics as T
import match as M
import lineage as L
import hotspot as H
import llm

SYS_PROMPT = (
    "你是资深科研协作顾问。基于给定的 GitLink 仓库/领域缺口信号、可合作学者、近期创新点，"
    "提出 3-5 条具体、可执行的研究方向或合作建议。要求：紧扣缺口主题、指出可切入的角度、"
    "点名潜在合作者类型。用中文，分条输出（1. 2. 3.），每条不超过 80 字。"
)


# ---------------------------------------------------------------------------
# 单仓创新启发
# ---------------------------------------------------------------------------

def analyze_repo(owner: str, repo: str, top: int = 8, pool_cap: int = 12,
                 context: dict | None = None) -> dict:
    """单仓：缺口 + 匹配 + 创新点 + LLM idea。"""
    m = M.match(owner, repo, top=top, pool_cap=pool_cap)  # gap_topics/signals/candidates
    innovations: list[dict] = []
    try:
        innovations = (L.lineage(owner, repo) or {}).get("innovation_points") or []
    except Exception as e:
        sys.stderr.write(f"[inspire] lineage 取创新点失败: {e!r}\n")

    idea = _llm_idea_repo(owner, repo, m, innovations, context)

    return {
        "scenario": "inspire",
        "mode": "repo",
        "repo": f"{owner}/{repo}",
        "gap_topics": m.get("gap_topics") or [],
        "needed_languages": m.get("needed_languages") or [],
        "gap_signals": (m.get("gap_signals") or [])[:12],
        "candidates": m.get("candidates") or [],
        "innovation_points": innovations[:6],
        "idea": idea,
        "llm_used": idea is not None,
    }


def _llm_idea_repo(owner: str, repo: str, m: dict, innovations: list[dict],
                   context: dict | None) -> str | None:
    if not llm.available():
        return None
    gaps = ", ".join(f"{t}" for t in (m.get("gap_topics") or [])[:6]) or "(无明显缺口主题)"
    cands = "; ".join(
        f"{x.get('login')}(契合{x.get('score')}, 重叠{x.get('topic_overlap')})"
        for x in (m.get("candidates") or [])[:3]) or "(暂无候选)"
    inno = "; ".join((x.get("description") or "")[:50] for x in innovations[:3]) or "(暂无)"
    cat_line = ""
    if context and context.get("category"):
        cat_line = f"\n所属领域热点分类：{context['category']}；领域热门主题：{', '.join(context.get('topic_heat', [])[:5])}"
    prompt = (
        f"仓库：{owner}/{repo}{cat_line}\n"
        f"缺口主题（按权重）：{gaps}\n"
        f"可合作学者：{cands}\n"
        f"近期创新点：{inno}\n\n"
        f"请基于以上给出 3-5 条研究方向/合作建议。"
    )
    return llm.chat(prompt, system=SYS_PROMPT)


# ---------------------------------------------------------------------------
# 领域级创新启发
# ---------------------------------------------------------------------------

def analyze_category(category: str, repos_limit: int = 12, top_repos_for_gap: int = 3) -> dict:
    """领域级：聚合分类下 top 仓的缺口 + 领域主题热度 + LLM idea。"""
    raw = H.collect(category=category, limit=repos_limit)
    hot = H.compute(raw)
    topic_heat = [t["topic"] for t in (hot.get("topic_heat") or [])[:8]]
    scholars = [s["login"] for s in (hot.get("core_scholars") or [])[:5]]

    # 聚合 top-N 仓的缺口（轻量：每仓 build_gap_signals，累加 gap_topics）
    agg: Counter = Counter()
    gap_signals_agg: list[dict] = []
    repos = hot.get("trending_repos") or []
    for r in repos[:top_repos_for_gap]:
        full = r.get("repo") or ""
        if "/" not in full:
            continue
        o, rr = full.split("/", 1)
        try:
            info = c.repo_info(o, rr) or {}
            gt, _langs, sigs = M.build_gap_signals(o, rr, info, issue_sample=40)
            agg.update({k: v for k, v in gt.items()})
            for s in sigs[:3]:
                s2 = dict(s); s2["repo"] = full; gap_signals_agg.append(s2)
        except Exception as e:
            sys.stderr.write(f"[inspire] {full} 缺口采集失败: {e!r}\n")

    idea = _llm_idea_category(category, topic_heat, agg, scholars)

    return {
        "scenario": "inspire",
        "mode": "category",
        "category": category,
        "topic_heat": topic_heat,
        "gap_topics": [t for t, _ in agg.most_common(10)],
        "gap_signals": gap_signals_agg[:12],
        "core_scholars": scholars,
        "idea": idea,
        "llm_used": idea is not None,
    }


def _llm_idea_category(category: str, topic_heat: list[str], gaps: Counter,
                       scholars: list[str]) -> str | None:
    if not llm.available():
        return None
    gap_str = ", ".join(f"{t}({n})" for t, n in gaps.most_common(8)) or "(无明显缺口)"
    prompt = (
        f"GitLink 领域：{category}\n"
        f"热门主题：{', '.join(topic_heat[:6])}\n"
        f"缺口主题（主题词频）：{gap_str}\n"
        f"核心活跃学者：{', '.join(scholars)}\n\n"
        f"请基于该领域的热点与缺口，给出 3-5 条具切入价值的研究方向建议。"
    )
    return llm.chat(prompt, system=SYS_PROMPT)


# ---------------------------------------------------------------------------
# 报告
# ---------------------------------------------------------------------------

def render_report(result: dict) -> str:
    lines = ["# 💡 创新启发报告", ""]
    if result.get("mode") == "category":
        lines.append(f"> 领域：**{result.get('category')}**（领域级缺口 + LLM 建议）")
    else:
        lines.append(f"> 焦点仓库：`{result.get('repo')}`")
    lines.append(f"> LLM 建议：{'✅ 已生成' if result.get('llm_used') else '⏭ 未启用（无 DEEPSEEK_API_KEY）'}")
    lines.append("")

    gaps = result.get("gap_topics") or []
    if gaps:
        lines.append("## 🎯 缺口主题")
        lines.append(", ".join(f"`{g}`" for g in gaps[:10]))
        lines.append("")

    sigs = result.get("gap_signals") or []
    if sigs:
        lines.append("## 🚩 缺口信号（未解决 Issue）")
        for s in sigs[:6]:
            lines.append(f"- [{s.get('priority') or '—'}] {s.get('topic')}：{(s.get('evidence') or '')[:70]}  · `{s.get('repo','')}`")
        lines.append("")

    cands = result.get("candidates") or []
    if cands:
        lines.append("## 🤝 可合作学者 Top 5")
        lines.append("| 学者 | 契合度 | 主题重叠 | 语言匹配 | 活跃 | 理由 |")
        lines.append("|---|---|---|---|---|---|")
        for x in cands[:5]:
            lines.append(f"| `{x.get('login')}` | {x.get('score')} | {x.get('topic_overlap')} | "
                         f"{x.get('language_match')} | {x.get('activity_level')} | {';'.join(x.get('reasons', [])[:2])} |")
        lines.append("")

    inno = result.get("innovation_points") or []
    if inno:
        lines.append("## 🌟 近期创新点")
        for i in inno[:5]:
            lines.append(f"- {i.get('description')} _({i.get('category','')})_")
        lines.append("")

    idea = result.get("idea")
    if idea:
        lines.append("## 🧠 LLM 研究方向建议")
        lines.append("")
        lines.append(idea.strip())
        lines.append("")

    lines.append("---\n*由 gitlink-research-inspire 生成*")
    return "\n".join(lines)


# ---------------------------------------------------------------------------
# CLI
# ---------------------------------------------------------------------------

def main() -> None:
    ap = argparse.ArgumentParser(description="创新启发 — 缺口/匹配/创新点 + LLM idea")
    ap.add_argument("--owner", default="")
    ap.add_argument("--repo", default="")
    ap.add_argument("--category", "-c", default="", help="领域分类（领域级模式）")
    ap.add_argument("--top", type=int, default=8)
    ap.add_argument("--pool", type=int, default=12)
    ap.add_argument("--out", "-o", default="")
    args = ap.parse_args()

    if args.category:
        result = analyze_category(args.category)
    elif args.owner and args.repo:
        result = analyze_repo(args.owner, args.repo, top=args.top, pool_cap=args.pool)
    else:
        print(json.dumps({"ok": False, "error": "need --owner/--repo OR --category"}, ensure_ascii=False))
        sys.exit(1)

    text = json.dumps(result, ensure_ascii=False, indent=2)
    if args.out:
        os.makedirs(args.out, exist_ok=True)
        with open(os.path.join(args.out, "inspire.json"), "w", encoding="utf-8") as f:
            f.write(text)
        with open(os.path.join(args.out, "report.md"), "w", encoding="utf-8") as f:
            f.write(render_report(result))
        sys.stderr.write(f"[inspire] ✓ mode={result['mode']} llm={result['llm_used']} 产物落 {args.out}\n")
    else:
        print(text)


if __name__ == "__main__":
    main()
