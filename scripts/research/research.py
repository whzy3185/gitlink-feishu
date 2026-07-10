"""research.py — 科研情报全链路编排器（打通五主线）。

一条命令跑通：选分类(explore) → ③ 热点追踪 → ② 主体画像 → ④ 创新启发 → ⑤ 合规校验 → ① 项目分析。
in-process 复用各 pillar 的函数（非 subprocess），共享数据；产物按 pillar 分目录 + 顶层 chain.json/chain_report.md。

用法：
  python research.py --category 深度学习 --repo owner/repo --out ./out/chain [--limit 20]
  python research.py --category 深度学习            # 焦点仓自动取热点榜 top-1
"""
from __future__ import annotations

import argparse
import json
import os
import sys

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
import collect as c          # noqa: F401  (各 pillar 依赖共享)
import hotspot as H
import profile as P
import inspire as I
import repro as R
import lineage as L


def _write(path: str, text: str) -> None:
    with open(path, "w", encoding="utf-8") as f:
        f.write(text)


def _save(out_root: str, pillar: str, result: dict, report_md: str | None = None) -> None:
    if not out_root:
        return
    d = os.path.join(out_root, pillar)
    os.makedirs(d, exist_ok=True)
    _write(os.path.join(d, f"{pillar}.json"), json.dumps(result, ensure_ascii=False, indent=2))
    if report_md:
        _write(os.path.join(d, "report.md"), report_md)


def _focal_from_trending(trending: list[dict]) -> tuple[str, str]:
    for r in trending:
        full = r.get("repo") or ""
        if "/" in full:
            o, rr = full.split("/", 1)
            if o and rr and o.lower() != "default":
                return o, rr
    # 回退：允许 default/<repo>
    for r in trending:
        full = r.get("repo") or ""
        if "/" in full:
            o, rr = full.split("/", 1)
            if o and rr:
                return o, rr
    return "", ""


def run_chain(category: str, repo: str | None = None, limit: int = 20, out: str = "") -> dict:
    summary: dict = {"scenario": "chain", "category": category, "pillars": {}}

    # ① 热点追踪
    sys.stderr.write(f"[chain] ① 热点追踪：分类 {category}（上限 {limit}）…\n"); sys.stderr.flush()
    raw = H.collect(category=category, limit=limit)
    hot = H.compute(raw)
    trending = hot.get("trending_repos") or []
    _save(out, "hotspot", hot, getattr(H, "_report_md", lambda x: "")(hot))

    # ② 焦点仓
    if repo and "/" in repo:
        owner, repo_name = repo.split("/", 1)
    else:
        owner, repo_name = _focal_from_trending(trending)
    summary["focal_repo"] = f"{owner}/{repo_name}" if owner else "(未解析到)"
    sys.stderr.write(f"[chain] 焦点仓：{owner}/{repo_name}\n"); sys.stderr.flush()
    summary["hotspot"] = {
        "repo_count": (hot.get("meta") or {}).get("repo_count"),
        "trending_top": trending[0]["repo"] if trending else "",
        "topic_heat": [t["topic"] for t in (hot.get("topic_heat") or [])[:5]],
    }

    if not (owner and repo_name):
        sys.stderr.write("[chain] ⚠ 未解析到焦点仓，后续画像/启发/合规/分析跳过\n")
        _finalize(out, summary, hot)
        return summary

    ctx = {"category": category,
           "topic_heat": [t["topic"] for t in (hot.get("topic_heat") or [])[:6]]}

    # ③ 主体画像
    sys.stderr.write("[chain] ② 主体画像…\n"); sys.stderr.flush()
    proj = P.project_profile(owner, repo_name)
    top_login = ((hot.get("core_scholars") or [{}])[0]).get("login") or ""
    scholar = P.scholar_profile(top_login) if top_login else None
    prof_result = {"scenario": "profile", "mode": "chain",
                   "project": proj, "scholar": scholar}
    prof_md = []
    prof_md.append(P.render_report({"mode": "project", "profiles": [proj]}))
    if scholar:
        prof_md.append(P.render_report({"mode": "scholar", "profiles": [scholar]}))
    _save(out, "profile", prof_result, "\n".join(prof_md))
    summary["pillars"]["profile"] = {
        "repo": proj.get("repo"), "score_total": proj.get("score_total"),
        "scholar": top_login or None,
    }

    # ④ 创新启发
    sys.stderr.write("[chain] ③ 创新启发…\n"); sys.stderr.flush()
    ins = I.analyze_repo(owner, repo_name, context=ctx)
    _save(out, "inspire", ins, I.render_report(ins))
    summary["pillars"]["inspire"] = {
        "gap_topics": (ins.get("gap_topics") or [])[:5],
        "candidates": len(ins.get("candidates") or []),
        "innovations": len(ins.get("innovation_points") or []),
        "llm_used": ins.get("llm_used"),
    }

    # ⑤ 合规校验
    sys.stderr.write("[chain] ④ 合规校验…\n"); sys.stderr.flush()
    try:
        rep = R.run(owner, repo_name)
        _save(out, "repro", rep, R.render_report(rep))
        summary["pillars"]["repro"] = {
            "license": rep.get("license"),
            "repro_score": rep.get("repro_score"),
            "compliance_score": rep.get("compliance_score"),
        }
    except Exception as e:
        sys.stderr.write(f"[chain] repro 失败: {e!r}\n")

    # ⑥ 项目分析
    sys.stderr.write("[chain] ⑤ 项目分析…\n"); sys.stderr.flush()
    try:
        lin = L.lineage(owner, repo_name)
        _save(out, "lineage", lin, L.render_report(lin))
        summary["pillars"]["lineage"] = {
            "innovations": len(lin.get("innovation_points") or []),
        }
    except Exception as e:
        sys.stderr.write(f"[chain] lineage 失败: {e!r}\n")

    _finalize(out, summary, hot)
    return summary


def _finalize(out: str, summary: dict, hot: dict) -> None:
    if not out:
        return
    _write(os.path.join(out, "chain.json"), json.dumps(summary, ensure_ascii=False, indent=2))
    _write(os.path.join(out, "chain_report.md"), _chain_report(summary))
    sys.stderr.write(f"[chain] ✓ 全链路完成，产物落 {out}\n"); sys.stderr.flush()


def _chain_report(s: dict) -> str:
    lines = [
        "# 🔬 科研情报全链路报告",
        "",
        f"> 领域分类：**{s.get('category')}** · 焦点仓：`{s.get('focal_repo')}`",
        "",
        "## 链路",
        "选分类(explore) → ① 热点追踪 → ② 主体画像 → ③ 创新启发 → ④ 合规校验 → ⑤ 项目分析",
        "",
    ]
    h = s.get("hotspot") or {}
    lines.append("### ① 热点追踪")
    lines.append(f"- 榜首：`{h.get('trending_top')}`；领域主题：" +
                 "、".join(f"`{t}`" for t in h.get("topic_heat", [])))
    p = s.get("pillars") or {}
    if p.get("profile"):
        lines.append("\n### ② 主体画像")
        pr = p["profile"]
        lines.append(f"- `{pr.get('repo')}` 研究维度评分 **{pr.get('score_total')}/40**；核心学者 `{pr.get('scholar')}`")
    if p.get("inspire"):
        lines.append("\n### ③ 创新启发")
        ii = p["inspire"]
        lines.append(f"- 缺口 {len(ii.get('gap_topics', []))} 主题、可合作候选 {ii.get('candidates')} 位、创新点 {ii.get('innovations')} 个；LLM 建议 {'✅' if ii.get('llm_used') else '⏭ 未启用'}")
    if p.get("repro"):
        lines.append("\n### ④ 合规校验")
        rr = p["repro"]
        lines.append(f"- 许可证 `{rr.get('license')}`；复现性 {rr.get('repro_score')}/10 · 合规 {rr.get('compliance_score')}/10")
    if p.get("lineage"):
        lines.append("\n### ⑤ 项目分析")
        lines.append(f"- 识别 {p['lineage'].get('innovations')} 个创新点（详见 lineage/report.md）")
    lines += ["", "各 pillar 完整产物见子目录 `{pillar}/`。", "", "---", "*由 gitlink-research chain 生成*"]
    return "\n".join(lines)


def main() -> None:
    ap = argparse.ArgumentParser(description="科研情报全链路编排器")
    ap.add_argument("--category", "-c", required=True, help="领域分类（中文名或 id，如 深度学习/32）")
    ap.add_argument("--repo", default="", help="焦点仓 owner/repo（省略则取热点榜 top-1）")
    ap.add_argument("--limit", type=int, default=20, help="热点榜上限（默认 20）")
    ap.add_argument("--out", "-o", default="", help="输出目录")
    args = ap.parse_args()
    run_chain(args.category, repo=args.repo or None, limit=args.limit, out=args.out)


if __name__ == "__main__":
    main()
