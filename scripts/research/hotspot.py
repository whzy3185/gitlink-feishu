"""hotspot.py — 科研热点追踪（全栈重构版）。

输入一组科研关键词，从 GitLink 平台按关键词搜索相关仓库，对每个仓库：
  - 拉取 repo_info（stars / forks / 更新时间 / 贡献者）
  - 拉取 issue + pr 列表（活跃讨论按评论数排序）
  - 从 description + readme 抽取主题标签
  - 聚合学者/团队贡献网络

输出：hotspot.json（结构化数据）+ report.md（中文简报）。

取数与算法分离：collect() 在线取数，compute_*() 纯函数离线可测。

用法：
  # 分类精选源（GitLink 官方 pinned，自带 visits）—— 推荐用于缩小范围
  python hotspot.py --category 深度学习 --limit 30 --out ./out
  python hotspot.py --category 32 --limit 20 --days 90   # 近 90 天
  # 关键词源（原逻辑）
  python hotspot.py --keywords "deep learning,机器学习" --limit 12 --out ./out
"""
from __future__ import annotations

import argparse
import json
import os
import sys
import time
from collections import Counter, defaultdict
from typing import Any

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
import collect as c
import topics as T


# ---------------------------------------------------------------------------
# 小工具
# ---------------------------------------------------------------------------

def _now_ts() -> float:
    return time.time()


def _days_ago(ts: float) -> int:
    """粗略计算距今多少天（非精确日历差，但足以排序）。"""
    return max(0, int((_now_ts() - ts) / 86400))


# ---------------------------------------------------------------------------
# 热度评分（纯函数，离线可测）
# ---------------------------------------------------------------------------

def compute_trending_score(stars: int, forks: int, updated_days_ago: int, visits: int = 0) -> int:
    """仓库热度综合评分。

    公式：stars + forks × 2 + visits//10（GitLink 官方访问量）+ 近期更新加成。
    visits 来自 `explore +pinned` 的官方字段；关键词源无该字段时为 0（向后兼容）。
    满分无上限，用于仓库间横向排序。
    """
    base = stars + forks * 2 + visits // 10
    if updated_days_ago <= 7:
        recency = 30
    elif updated_days_ago <= 30:
        recency = 20
    elif updated_days_ago <= 90:
        recency = 10
    elif updated_days_ago <= 180:
        recency = 5
    else:
        recency = 0
    return base + recency


def compute_velocity(stars: int, updated_days_ago: int) -> float:
    """日均星标增速（近似值）。"""
    if updated_days_ago <= 0:
        updated_days_ago = 1
    return round(stars / max(updated_days_ago, 1), 2)


# ---------------------------------------------------------------------------
# 在线取数
# ---------------------------------------------------------------------------

def collect(keywords: list[str] | None = None, repos_limit: int = 12,
            category: str | None = None, limit: int | None = None,
            days: int = 0) -> dict:
    """在线取数：候选源（分类精选 OR 关键词搜索）→ 去重 → 截断 → 逐仓详情。

    - ``category`` 模式：走 ``explore +pinned --category <域>``，取 GitLink 官方精选
      项目（自带 visits/praises_count/forked_count/topics/language/author）。
    - 关键词模式：多关键词 ``search +repos``（原逻辑）。

    其余逐仓 enrichment（repo_info/issues/prs/contributors/readme/languages）两种源共用。
    返回原始数据字典，供 compute() 消费。
    """
    seen: dict[str, dict] = {}

    def _add(proj: dict, kw: str = ""):
        full = c.repo_fullname(proj)
        if not full:
            return
        if full not in seen:
            seen[full] = {
                "fullname": full,
                "matched_keywords": [],
                "project": proj,
                "info": None,
                "issues": [],
                "prs": [],
                "contributors": [],
                "readme": "",
                "languages": {},
            }
        if kw and kw not in seen[full]["matched_keywords"]:
            seen[full]["matched_keywords"].append(kw)

    # Step 1: 候选源
    if category:
        n = limit or repos_limit
        for proj in c.pinned(category, limit=n):
            _add(proj)
    else:
        for kw in (keywords or []):
            for proj in c.search_repos(kw, limit=repos_limit):
                _add(proj, kw)

    repos = list(seen.values())

    # 按 praises + visits//10 粗排，截断到 limit 再深度取数（节省 API）
    def _heat_sort_key(r: dict) -> int:
        p = r["project"]
        return -(c.as_int(p.get("praises_count") or p.get("stars_count") or 0)
                 + c.as_int(p.get("visits") or 0) // 10)

    cap = limit or repos_limit
    repos.sort(key=_heat_sort_key)
    repos = repos[:cap]

    # days 过滤（按 last_update_time 近 N 天；0 = 不过滤；过滤后为空则保留原列表）
    if days and days > 0:
        cutoff = _now_ts() - days * 86400
        fresh = [r for r in repos
                 if _parse_ts(r["project"].get("last_update_time") or "") >= cutoff]
        if fresh:
            repos = fresh

    # Step 2: 逐仓库取详情（两源共用）
    for r in repos:
        full = r["fullname"]
        parts = full.split("/", 1)
        if len(parts) != 2:
            continue
        owner, repo = parts[0], parts[1]
        info = c.repo_info(owner, repo)
        r["info"] = info if isinstance(info, dict) else {}
        r["issues"] = c.issues(owner, repo, state="open", max_pages=3, page_size=30)
        r["prs"] = c.prs(owner, repo, state="open", max_pages=3, page_size=30)
        r["contributors"] = c.contributors(owner, repo)
        r["readme"] = (c.readme(owner, repo) or "")[:4000]
        r["languages"] = c.languages(owner, repo)

    return {
        "keywords": keywords or [],
        "category": category or "",
        "repos": repos,
        "repos_limit": cap,
    }


# ---------------------------------------------------------------------------
# 离线计算（纯函数，不联网）
# ---------------------------------------------------------------------------

def compute(raw: dict) -> dict:
    """从 collect() 的原始数据计算出所有热点指标。

    输入结构见 collect() 返回值；输出为标准 hotspot.json 结构。
    """
    repos_raw = raw.get("repos") or []

    # ------ trending_repos ------
    trending: list[dict] = []
    now = _now_ts()
    for r in repos_raw:
        info = r.get("info") or {}
        proj = r.get("project") or {}

        stars = c.as_int(
            info.get("watchers_count")
            or info.get("praises_count")
            or proj.get("praises_count")
            or 0
        )
        forks = c.as_int(
            info.get("forked_count")
            or proj.get("forked_count")
            or 0
        )
        # 解析更新时间
        update_str = (
            info.get("full_last_update_time")
            or info.get("last_update_time")
            or proj.get("full_last_update_time")
            or proj.get("last_update_time")
            or ""
        )
        update_ts = _parse_ts(update_str)
        days = _days_ago(update_ts)
        visits = c.as_int(proj.get("visits") or info.get("visits") or 0)
        score = compute_trending_score(stars, forks, days, visits=visits)
        velocity = compute_velocity(stars, days)
        language = ""
        lang_obj = info.get("language") or proj.get("language")
        if isinstance(lang_obj, dict):
            language = lang_obj.get("name") or ""
        elif isinstance(lang_obj, str):
            language = lang_obj

        trending.append({
            "repo": r["fullname"],
            "description": (
                info.get("description")
                or proj.get("description")
                or ""
            ),
            "language": language,
            "stars": stars,
            "forks": forks,
            "visits": visits,
            "score": score,
            "velocity": velocity,
            "matched_keywords": r.get("matched_keywords", []),
            "updated": _fmt_ts(update_ts),
            "contributors_count": c.as_int(info.get("contributor_users_count") or 0),
            "releases_count": c.as_int(info.get("version_releases_count") or 0),
        })

    trending.sort(key=lambda x: -x["score"])

    # ------ active_discussions ------
    discussions: list[dict] = []
    for r in repos_raw:
        full = r["fullname"]
        # issues
        for iss in r.get("issues") or []:
            comments = c.as_int(iss.get("comment_count") or iss.get("comments") or 0)
            if comments > 0:
                discussions.append({
                    "type": "issue",
                    "repo": full,
                    "title": iss.get("title") or "(无标题)",
                    "number": iss.get("index") or iss.get("number") or "",
                    "state": iss.get("state") or iss.get("status") or "open",
                    "comments": comments,
                })
        # PRs
        for pr in r.get("prs") or []:
            comments = c.as_int(pr.get("comment_count") or pr.get("comments") or 0)
            if comments > 0:
                discussions.append({
                    "type": "pr",
                    "repo": full,
                    "title": pr.get("title") or "(无标题)",
                    "number": pr.get("index") or pr.get("number") or "",
                    "state": pr.get("state") or pr.get("status") or "open",
                    "comments": comments,
                })

    discussions.sort(key=lambda x: -x["comments"])

    # ------ topic_heat ------
    # 对所有仓库的 description + readme 跑 topic_counter
    texts = []
    for r in repos_raw:
        info = r.get("info") or {}
        proj = r.get("project") or {}
        desc = info.get("description") or proj.get("description") or ""
        texts.append(desc)
        readme = r.get("readme") or ""
        if readme:
            texts.append(readme)
    heat = T.topic_counter(texts)
    topic_heat = [
        {"topic": topic, "count": count}
        for topic, count in heat.most_common(10)
    ]

    # ------ core_scholars / core_teams ------
    scholar_repo_count: Counter = Counter()
    # 记录每个 scholar 关联的仓库名
    scholar_repos: dict[str, list[str]] = defaultdict(list)
    for r in repos_raw:
        full = r["fullname"]
        for contrib in r.get("contributors") or []:
            login = c.login_of(contrib)
            if not login or _is_bot(login):
                continue
            scholar_repo_count[login] += 1
            if full not in scholar_repos[login]:
                scholar_repos[login].append(full)

    core_scholars = [
        {
            "login": login,
            "repo_count": count,
            "repos": scholar_repos.get(login, []),
        }
        for login, count in scholar_repo_count.most_common(12)
    ]

    # 核心团队：按 owner（仓库第一段）聚合
    org_repo: Counter = Counter()
    for r in repos_raw:
        full = r["fullname"]
        org = full.split("/")[0] if "/" in full else full
        org_repo[org] += 1

    core_teams = [
        {"login": org, "repo_count": count}
        for org, count in org_repo.most_common(8)
    ]

    # ------ meta ------
    total_issues = sum(len(r.get("issues") or []) for r in repos_raw)
    total_prs = sum(len(r.get("prs") or []) for r in repos_raw)

    return {
        "scenario": "hotspot",
        "keywords": raw.get("keywords") or [],
        "category": raw.get("category") or "",
        "trending_repos": trending,
        "active_discussions": discussions,
        "topic_heat": topic_heat,
        "core_scholars": core_scholars,
        "core_teams": core_teams,
        "meta": {
            "repo_count": len(repos_raw),
            "issue_count": total_issues,
            "pr_count": total_prs,
            "scholar_count": len(scholar_repo_count),
            "discussion_count": len(discussions),
            "topic_count": len(topic_heat),
        },
    }


# ---------------------------------------------------------------------------
# 时间工具
# ---------------------------------------------------------------------------

def _parse_ts(v: Any) -> float:
    """把 GitLink 时间戳/字符串尽量解析为 Unix 浮点秒。"""
    if v is None:
        return 0.0
    if isinstance(v, (int, float)):
        if v > 1_000_000_000_000:
            return v / 1000.0
        return float(v)
    s = str(v).strip()
    if not s:
        return 0.0
    # ISO 8601 格式
    for fmt in ("%Y-%m-%dT%H:%M:%S", "%Y-%m-%d %H:%M:%S",
                "%Y-%m-%dT%H:%M:%SZ", "%Y-%m-%d"):
        try:
            return time.mktime(time.strptime(s[:19] if len(s) >= 19 else s, fmt))
        except ValueError:
            continue
    return 0.0


def _fmt_ts(ts: float) -> str:
    """Unix 浮点秒 → 'YYYY-MM-DD' 字符串。"""
    if ts <= 0:
        return "—"
    try:
        return time.strftime("%Y-%m-%d", time.localtime(ts))
    except (ValueError, OSError):
        return "—"


_BOT_HINTS = ("bot", "i-robot", "dependabot", "renovate", "semantic-release-bot")


def _is_bot(login: str) -> bool:
    low = login.lower()
    return any(h in low for h in _BOT_HINTS)


# ---------------------------------------------------------------------------
# 报告生成
# ---------------------------------------------------------------------------

def _report_md(output: dict) -> str:
    """从热点 JSON 产出中文简报 Markdown。"""
    kw = ", ".join(output.get("keywords") or [])
    category = output.get("category") or ""
    meta = output.get("meta") or {}
    source_line = f"> 领域分类：**{category}**（GitLink 官方精选）" if category else f"> 关键词：{kw}"
    lines = [
        f"# 🔬 科研热点追踪报告",
        f"",
        source_line,
        f"> 扫描时间：{_fmt_ts(_now_ts())}",
        f"> 覆盖仓库：{meta.get('repo_count', 0)} 个 "
        f"· 讨论 {meta.get('discussion_count', 0)} 条 "
        f"· 主题 {meta.get('topic_count', 0)} 个 "
        f"· 学者 {meta.get('scholar_count', 0)} 位",
        f"",
    ]

    # 飙升项目 top 5
    trending = output.get("trending_repos") or []
    if trending:
        lines.append("## 🔥 飙升项目 Top 5")
        lines.append("")
        lines.append("| # | 仓库 | 语言 | 👁 访问 | ★ Star | ⑂ Fork | 热度 | 更新 |")
        lines.append("|---|------|------|---------|--------|--------|------|------|")
        for i, r in enumerate(trending[:5], 1):
            lines.append(
                f"| {i} | `{r['repo']}` | {r['language'] or '—'} | "
                f"{r.get('visits', 0)} | {r['stars']} | {r['forks']} | {r['score']} | {r['updated']} |"
            )
        lines.append("")

    # 活跃讨论 top 5
    discussions = output.get("active_discussions") or []
    if discussions:
        lines.append("## 💬 活跃讨论 Top 5")
        lines.append("")
        for i, d in enumerate(discussions[:5], 1):
            tp = "🐛 Issue" if d["type"] == "issue" else "🔀 PR"
            lines.append(
                f"{i}. {tp} [{d['repo']}] {d['title']} "
                f"(#{d['number']} · {d['comments']} 💬)"
            )
        lines.append("")

    # 热门主题
    topic_heat = output.get("topic_heat") or []
    if topic_heat:
        lines.append("## 📊 热门主题")
        lines.append("")
        for t in topic_heat:
            bar = "█" * min(t["count"], 20)
            lines.append(f"- **{t['topic']}** — {t['count']} 个仓库 {bar}")
        lines.append("")

    # 核心学者
    scholars = output.get("core_scholars") or []
    if scholars:
        lines.append("## 👥 核心学者")
        lines.append("")
        for s in scholars[:5]:
            lines.append(f"- **{s['login']}** — 关联 {s['repo_count']} 个仓库")
        lines.append("")

    # 核心团队
    teams = output.get("core_teams") or []
    if teams:
        lines.append("## 🏛 活跃组织/团队")
        lines.append("")
        for t in teams:
            lines.append(f"- **{t['login']}** — {t['repo_count']} 个仓库")
        lines.append("")

    lines.append("---")
    lines.append("*由 gitlink-research-hotspot 自动生成*")
    return "\n".join(lines)


# ---------------------------------------------------------------------------
# CLI 入口
# ---------------------------------------------------------------------------

def main() -> None:
    ap = argparse.ArgumentParser(
        description="科研热点追踪 — 分类精选(GitLink 官方) / 关键词 双数据源"
    )
    src = ap.add_mutually_exclusive_group()
    src.add_argument(
        "--category", "-c", default="",
        help="GitLink 领域分类（中文名或 id，如 深度学习 / 32）—— 官方精选源",
    )
    src.add_argument(
        "--keywords", "-k", default="",
        help="搜索关键词，逗号分隔（关键词源）",
    )
    ap.add_argument("--limit", type=int, default=20, help="最大分析仓库数（默认 20）")
    ap.add_argument("--repos-limit", type=int, default=12, help="(兼容旧参数，等价 --limit)")
    ap.add_argument("--days", type=int, default=0,
                    help="只取近 N 天更新的仓库（0=不过滤；仅 --category 源生效）")
    ap.add_argument("--out", "-o", default="", help="输出目录（不传则仅打印 JSON 到 stdout）")
    args = ap.parse_args()

    category = args.category.strip()
    kw_list = [k.strip() for k in args.keywords.split(",") if k.strip()] if args.keywords else []
    if not category and not kw_list:
        print(json.dumps({"ok": False, "error": "need --category OR --keywords"},
                         ensure_ascii=False))
        sys.exit(1)

    cap = args.limit or args.repos_limit

    # 取数
    if category:
        sys.stderr.write(f"[hotspot] 分类精选: {category}  上限: {cap}  近 {args.days or '∞'} 天\n")
    else:
        sys.stderr.write(f"[hotspot] 关键词: {kw_list}  上限: {cap}\n")
    sys.stderr.flush()
    raw = collect(keywords=kw_list or None, repos_limit=cap,
                  category=category or None, limit=cap, days=args.days)

    # 计算
    sys.stderr.write(f"[hotspot] 仓库: {len(raw['repos'])}  计算热点…\n")
    sys.stderr.flush()
    output = compute(raw)

    json_text = json.dumps(output, ensure_ascii=False, indent=2)

    if args.out:
        os.makedirs(args.out, exist_ok=True)
        json_path = os.path.join(args.out, "hotspot.json")
        with open(json_path, "w", encoding="utf-8") as f:
            f.write(json_text)
        report_path = os.path.join(args.out, "report.md")
        with open(report_path, "w", encoding="utf-8") as f:
            f.write(_report_md(output))
        sys.stderr.write(
            f"[hotspot] ✓ 完成 "
            f"仓库={output['meta']['repo_count']} "
            f"讨论={output['meta']['discussion_count']} "
            f"主题={output['meta']['topic_count']} "
            f"学者={output['meta']['scholar_count']}\n"
        )
        sys.stderr.write(f"[hotspot] 产物: {json_path}, {report_path}\n")
        sys.stderr.flush()
    else:
        print(json_text)


if __name__ == "__main__":
    main()
