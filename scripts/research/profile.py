"""profile.py — 主体画像（Pillar 2）。

模式（三选一）：
  --login <user>            学者画像（主题向量/语言/活跃度/协作开放度 + 基本信息）
  --owner O --repo R        项目画像（仓库元数据/贡献者/主题/语言/研究维度评分）
  --category <域> --top N   领域核心学者（从热点榜 core_scholars 取 top-N 逐个画像）

产物：profile.json + report.md（画像卡片）。
复用：match.profile_candidate / collect.user_info/user_repos/contributors/repo_info / topics。
"""
from __future__ import annotations

import argparse
import json
import os
import sys

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
import collect as c
import topics as T
import match as M          # 复用 profile_candidate
import hotspot as H        # --category 模式取 core_scholars

_BOT_HINTS = ("bot", "i-robot", "dependabot", "renovate", "semantic-release-bot")


def _is_bot(login: str) -> bool:
    low = login.lower()
    return any(h in low for h in _BOT_HINTS)


# ---------------------------------------------------------------------------
# 学者画像
# ---------------------------------------------------------------------------

def scholar_profile(login: str) -> dict:
    """学者画像：复用 match.profile_candidate（topic_vec/langs/activity/collab）+ user_info。"""
    info = c.user_info(login) or {}
    prof = M.profile_candidate(login, {})
    topic_vec = prof.get("topic_vec", {}) or {}
    topics_top = sorted(topic_vec.items(), key=lambda x: -x[1])[:8]
    return {
        "type": "scholar",
        "login": login,
        "name": info.get("name") or info.get("username") or login,
        "bio": info.get("bio") or info.get("description") or "",
        "topic_vec": topic_vec,
        "top_topics": [{"topic": t, "count": n} for t, n in topics_top],
        "languages": prof.get("langs", []),
        "activity": round(prof.get("activity", 0), 2),
        "collab": round(prof.get("collab", 0), 2),
        "repo_count": prof.get("repo_count", 0),
    }


# ---------------------------------------------------------------------------
# 项目画像
# ---------------------------------------------------------------------------

def project_profile(owner: str, repo: str) -> dict:
    """项目画像：仓库元数据 + 贡献者 + 主题/语言 + 研究维度评分（0–40）。"""
    info = c.repo_info(owner, repo) or {}
    contribs = c.contributors(owner, repo) or []
    readme = (c.readme(owner, repo) or "")[:4000]
    langs = c.languages(owner, repo) or {}
    desc = info.get("description") or ""

    tc = T.topic_counter([desc, readme])
    topics_top = [{"topic": t, "count": n} for t, n in tc.most_common(8)]

    stars = c.as_int(info.get("praises_count") or info.get("watchers_count") or 0)
    forks = c.as_int(info.get("forked_count") or 0)
    visits = c.as_int(info.get("visits") or 0)
    score = {
        "doc": 5 if readme else 0,
        "license": 5 if info.get("license") else 0,
        "collab": min(len(contribs), 10) * 1,      # 贡献者（封顶10）
        "impact": min(stars + forks, 10) * 2,       # star+fork（封顶20）
    }
    logins = [c.login_of(x) for x in contribs]
    logins = [l for l in logins if l and not _is_bot(l)][:8]

    return {
        "type": "project",
        "repo": f"{owner}/{repo}",
        "name": info.get("name") or repo,
        "description": desc,
        "topics": topics_top,
        "languages": (list(langs.keys()) if isinstance(langs, dict) else list(langs))[:8],
        "stars": stars, "forks": forks, "visits": visits,
        "contributors_count": len(contribs),
        "top_contributors": logins,
        "score": score,
        "score_total": sum(score.values()),
    }


# ---------------------------------------------------------------------------
# 领域核心学者
# ---------------------------------------------------------------------------

def category_scholars(category: str, top: int = 5, limit: int = 20) -> list[dict]:
    """领域核心学者：热点榜 core_scholars 取 top-N，逐个画像。"""
    raw = H.collect(category=category, limit=limit)
    scholars = (H.compute(raw).get("core_scholars") or [])[:top]
    out = []
    for s in scholars:
        login = s.get("login")
        if not login:
            continue
        prof = scholar_profile(login)
        prof["category_repos"] = s.get("repo_count")
        out.append(prof)
    return out


# ---------------------------------------------------------------------------
# 报告
# ---------------------------------------------------------------------------

def render_report(result: dict) -> str:
    lines = ["# 🪪 主体画像报告", ""]

    def _scholar_card(p: dict, depth: int = 2):
        h = "#" * depth
        lines.append(f"{h} 👤 {p.get('name')} (`{p.get('login')}`)")
        if p.get("bio"):
            lines.append(f"> {p['bio'][:120]}")
        lines.append("")
        lines.append(f"- 活跃度 **{p.get('activity')}** · 协作开放度 **{p.get('collab')}** · 公开仓库 **{p.get('repo_count')}**")
        if p.get("category_repos") is not None:
            lines.append(f"- 该领域关联仓库 **{p.get('category_repos')}**")
        if p.get("top_topics"):
            lines.append("- 研究主题：" + "、".join(f"`{t['topic']}`({t['count']})" for t in p["top_topics"][:6]))
        if p.get("languages"):
            lines.append("- 语言：" + "、".join(p["languages"][:6]))
        lines.append("")

    def _project_card(p: dict):
        lines.append(f"## 📦 {p.get('name')} (`{p.get('repo')}`)")
        if p.get("description"):
            lines.append(f"> {p['description'][:140]}")
        lines.append("")
        lines.append(f"- ★{p.get('stars')} ⑂{p.get('forks')} 👁{p.get('visits')} · 贡献者 {p.get('contributors_count')} · 研究维度评分 **{p.get('score_total')}/40**")
        if p.get("topics"):
            lines.append("- 主题：" + "、".join(f"`{t['topic']}`" for t in p["topics"][:6]))
        if p.get("top_contributors"):
            lines.append("- 核心贡献者：" + "、".join(f"`{l}`" for l in p["top_contributors"][:6]))
        lines.append("")

    profiles = result.get("profiles") or []
    if result.get("mode") == "project":
        _project_card(profiles[0] if profiles else {})
    else:
        for p in profiles:
            _scholar_card(p)
    lines.append("---\n*由 gitlink-research-profile 生成*")
    return "\n".join(lines)


# ---------------------------------------------------------------------------
# CLI
# ---------------------------------------------------------------------------

def main() -> None:
    ap = argparse.ArgumentParser(description="主体画像 — 学者 / 项目 / 领域核心学者")
    ap.add_argument("--login", default="", help="学者 login（学者画像）")
    ap.add_argument("--owner", default="", help="仓库 owner（项目画像）")
    ap.add_argument("--repo", default="", help="仓库名（项目画像）")
    ap.add_argument("--category", default="", help="领域分类（领域核心学者）")
    ap.add_argument("--top", type=int, default=5, help="领域模式取 top-N 学者（默认 5）")
    ap.add_argument("--limit", type=int, default=20, help="领域模式热点榜上限（默认 20）")
    ap.add_argument("--out", "-o", default="", help="输出目录（不传则打印 JSON）")
    args = ap.parse_args()

    if args.login:
        profiles = [scholar_profile(args.login)]
        mode = "scholar"
    elif args.owner and args.repo:
        profiles = [project_profile(args.owner, args.repo)]
        mode = "project"
    elif args.category:
        profiles = category_scholars(args.category, top=args.top, limit=args.limit)
        mode = "category_scholars"
    else:
        print(json.dumps({"ok": False, "error": "need --login OR --owner/--repo OR --category"},
                         ensure_ascii=False))
        sys.exit(1)

    result = {"scenario": "profile", "mode": mode, "profiles": profiles}

    json_text = json.dumps(result, ensure_ascii=False, indent=2)
    if args.out:
        os.makedirs(args.out, exist_ok=True)
        with open(os.path.join(args.out, "profile.json"), "w", encoding="utf-8") as f:
            f.write(json_text)
        with open(os.path.join(args.out, "report.md"), "w", encoding="utf-8") as f:
            f.write(render_report(result))
        sys.stderr.write(f"[profile] ✓ mode={mode} 产物落 {args.out}\n")
    else:
        print(json_text)


if __name__ == "__main__":
    main()
