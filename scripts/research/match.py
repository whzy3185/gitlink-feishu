"""match.py — S4 科研协作智能匹配。

输入一个科研仓库，分析其技术缺口（未解决 Issue 的主题/语言、开放 PR、研究空缺），
再从 GitLink 平台候选池（本仓库贡献者 + 按缺口主题搜索到的用户）中，用「主题向量 + 语言匹配 +
活跃度 + 协作开放度」综合打分，推荐最合适的跨团队/跨学者协作伙伴。

数据全部经 gitlink-cli 获取（issue +list / repo +contributors / repo +list --user / search +users）。

用法：
  python match.py --owner mindspore-Ecosystem --repo mindspore --top 10 --out ./out
  python match.py --owner O --repo R --format json   # 仅打印 JSON
"""
from __future__ import annotations

import argparse
import json
import math
import os
import sys
from collections import Counter
from typing import Any

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
import collect as c  # noqa: E402
import topics as T  # noqa: E402

# 优先级 → 权重（Issue 缺口信号加权）
PRIORITY_WEIGHT = {"urgent": 3, "high": 3, "紧急": 3, "高": 3,
                   "normal": 2, "medium": 2, "普通": 2, "中": 2,
                   "low": 1, "低": 1}


# ---------------------------------------------------------------------------
# 向量与打分
# ---------------------------------------------------------------------------

def cosine(c1: dict[str, float], c2: dict[str, float]) -> float:
    keys = set(c1) | set(c2)
    dot = sum(c1.get(k, 0.0) * c2.get(k, 0.0) for k in keys)
    n1 = math.sqrt(sum(v * v for v in c1.values()))
    n2 = math.sqrt(sum(v * v for v in c2.values()))
    return dot / (n1 * n2) if n1 and n2 else 0.0


def jaccard(a: list[str], b: list[str]) -> float:
    sa, sb = set(a), set(b)
    if not sa or not sb:
        return 0.0
    return len(sa & sb) / len(sa | sb)


def _priority_weight(issue: dict) -> float:
    p = (issue.get("priority_name") or issue.get("priority") or "").lower()
    for k, w in PRIORITY_WEIGHT.items():
        if k in str(p):
            return float(w)
    return 1.0


def _parse_ratio(v: Any) -> float:
    """把 '1.18%' / '0.2' / 0.2 等统一解析为 0~1 比例。"""
    if v is None:
        return 0.0
    s = str(v).strip()
    pct = s.endswith("%")
    if pct:
        s = s[:-1]
    try:
        f = float(s)
    except ValueError:
        return 0.0
    return f / 100.0 if (pct or f > 1.0) else f


# ---------------------------------------------------------------------------
# 缺口信号
# ---------------------------------------------------------------------------

def build_gap_signals(owner: str, repo: str, info: dict,
                      issue_sample: int = 100) -> tuple[Counter, list[str], list[dict]]:
    """返回 (缺口主题词频 Counter, 需求语言列表, 缺口信号明细)。"""
    gap_topics: Counter = Counter()
    gap_langs: set[str] = set()
    signals: list[dict] = []

    # 1) 仓库自身主题/语言（协作者应具备的基础方向）
    desc = (info.get("description") or "") + " " + c.readme(owner, repo)[:4000]
    for tp in T.extract_topics(desc):
        gap_topics[tp] += 1
    for lg in c.languages(owner, repo):
        gap_langs.add(lg)
    for lg in T.extract_languages(desc):
        gap_langs.add(lg)

    # 2) 未解决 Issue 的主题/语言（核心缺口）
    open_issues = c.issues(owner, repo, state="open", max_pages=max(1, issue_sample // 50),
                           page_size=50)
    for iss in open_issues[:issue_sample]:
        text = (iss.get("subject") or iss.get("title") or "")
        w = _priority_weight(iss)
        for tp in T.extract_topics(text):
            gap_topics[tp] += w
        for lg in T.extract_languages(text):
            gap_langs.add(lg)
        # 取每条 issue 的首个主题作为该条的证据
        tps = T.extract_topics(text)
        if tps:
            signals.append({"type": "unresolved_issue", "topic": tps[0],
                            "evidence": text[:80], "priority": iss.get("priority_name", "")})

    # 3) 开放 PR 正在推进的方向（轻量加成）
    for pr in c.prs(owner, repo, state="open", max_pages=1, page_size=30):
        text = (pr.get("title") or "") + " " + (pr.get("body") or "")
        for tp in T.extract_topics(text):
            gap_topics[tp] += 0.5

    needed_langs = sorted(gap_langs)
    return gap_topics, needed_langs, signals


# ---------------------------------------------------------------------------
# 候选人画像与匹配
# ---------------------------------------------------------------------------

def candidate_pool(owner: str, repo: str, gap_topics: Counter, pool_cap: int) -> list[str]:
    """候选人 login 池：本仓库贡献者 + 按缺口主题搜到的外部用户。"""
    seen: list[str] = []
    seen_set: set[str] = set()

    for contrib in c.contributors(owner, repo):
        login = c.login_of(contrib)
        # 过滤明显机器人账号
        if login and login not in seen_set and "bot" not in login.lower() and login.lower() != "i-robot":
            seen.append(login)
            seen_set.add(login)
        if len(seen) >= pool_cap:
            return seen

    # 取词频最高的若干主题，用其英文关键词搜外部用户
    top_topics = [t for t, _ in gap_topics.most_common(5)]
    eng_kw = {"nlp": "nlp", "deep_learning": "deep learning", "computer_vision": "cv",
              "reinforcement_learning": "reinforcement learning",
              "graph_learning": "gnn", "federated_learning": "federated",
              "scientific_computing": "cuda", "data_mining": "machine learning",
              "devops": "devops", "security": "security", "database": "database"}
    for tp in top_topics:
        kw = eng_kw.get(tp)
        if not kw:
            continue
        for u in c.search_users(kw, limit=10):
            login = c.login_of(u)
            if login and login not in seen_set:
                seen.append(login)
                seen_set.add(login)
        if len(seen) >= pool_cap:
            break
    return seen[:pool_cap]


def profile_candidate(login: str, repo_contribs: dict[str, dict]) -> dict[str, Any]:
    """构建候选人画像：主题向量 + 语言集合 + 活跃度 + 协作开放度。"""
    repos = c.user_repos(login, limit=15)
    texts = []
    langs: set[str] = set()
    fork_count = 0
    for r in repos:
        texts.append((r.get("description") or "") + " " + (r.get("identifier") or ""))
        if r.get("language") and isinstance(r["language"], dict):
            langs.add((r["language"].get("name") or "").lower())
        if r.get("forked_from_project_id") or r.get("forked_count"):
            fork_count += 1
    topic_vec: Counter = Counter()
    for t in texts:
        for tp in T.extract_topics(t):
            topic_vec[tp] += 1
    for lg in T.extract_languages(" ".join(texts)):
        langs.add(lg)

    activity = 0.4
    if login in repo_contribs:
        # 本仓库贡献者 → 高活跃（贡献占比越高加成越大）
        activity = 0.7 + 0.3 * min(_parse_ratio(repo_contribs[login].get("contribution_perc")), 1.0)
    elif len(repos) >= 5:
        activity = 0.7
    elif repos:
        activity = 0.4
    collab = min(fork_count / 5.0, 1.0)

    return {"topic_vec": dict(topic_vec), "langs": sorted(langs),
            "activity": activity, "collab": collab, "repo_count": len(repos)}


def match(owner: str, repo: str, top: int = 10, pool_cap: int = 15,
          issue_sample: int = 100) -> dict[str, Any]:
    info = c.repo_info(owner, repo)
    gap_topics, needed_langs, signals = build_gap_signals(owner, repo, info, issue_sample)
    contribs_list = c.contributors(owner, repo)
    repo_contribs = {c.login_of(x): x for x in contribs_list if c.login_of(x)}

    # 缺口主题向量（与候选人主题向量同空间）
    gap_vec = dict(gap_topics)

    pool = candidate_pool(owner, repo, gap_topics, pool_cap)
    scored = []
    for login in pool:
        prof = profile_candidate(login, repo_contribs)
        topic_overlap = cosine(prof["topic_vec"], gap_vec)
        lang_match = jaccard(prof["langs"], needed_langs) if needed_langs else 0.0
        score = (0.45 * topic_overlap + 0.20 * lang_match
                 + 0.20 * prof["activity"] + 0.15 * prof["collab"]) * 100
        reasons: list[str] = []
        overlap_topics = sorted(set(prof["topic_vec"]) & set(gap_vec),
                                key=lambda k: -prof["topic_vec"][k])
        if overlap_topics:
            reasons.append(f"覆盖缺口主题: {', '.join(overlap_topics[:4])}")
        matched_langs = sorted(set(prof["langs"]) & set(needed_langs))
        if matched_langs:
            reasons.append(f"语言匹配: {', '.join(matched_langs[:4])}")
        if login in repo_contribs:
            reasons.append("本仓库活跃贡献者")
        if prof["collab"] > 0:
            reasons.append(f"协作开放度高(fork={int(prof['collab']*5)})")
        activity_level = ("high" if prof["activity"] >= 0.7
                          else "medium" if prof["activity"] >= 0.4 else "low")
        scored.append({
            "login": login, "score": round(score, 1),
            "topic_overlap": round(topic_overlap, 3),
            "language_match": round(lang_match, 3),
            "activity_level": activity_level,
            "repo_languages": prof["langs"][:6],
            "repo_count": prof["repo_count"],
            "reasons": reasons or ["无明显主题/语言重叠"],
        })
    scored.sort(key=lambda x: -x["score"])

    top_topics = [t for t, _ in gap_topics.most_common(8)]
    return {
        "scenario": "S4_collaboration_matching",
        "repo": f"{owner}/{repo}",
        "gap_topics": top_topics,
        "needed_languages": needed_langs,
        "gap_signals": signals[:30],
        "candidates": scored[:top],
        "meta": {"pool_size": len(pool), "issue_sample": issue_sample},
    }


# ---------------------------------------------------------------------------
# 渲染：Markdown 报告 + Mermaid 协作网络
# ---------------------------------------------------------------------------

def render_report(result: dict[str, Any]) -> str:
    repo = result["repo"]
    cands = result["candidates"]
    lines = [
        f"# 科研协作智能匹配报告 — {repo}\n",
        f"> 场景 S4 · 子赛题四「应用 GitLink 辅助科研」\n",
        "## 一、仓库技术缺口分析\n",
        f"- **缺口主题**: {', '.join(result['gap_topics']) or '（未识别到明确主题）'}",
        f"- **需求语言**: {', '.join(result['needed_languages']) or '—'}",
        f"- **缺口信号样本**: {len(result['gap_signals'])} 条未解决 Issue/PR 主题证据\n",
        "| 缺口主题 | 证据（Issue/PR） | 优先级 |",
        "|----------|------------------|--------|",
    ]
    for s in result["gap_signals"][:8]:
        lines.append(f"| {s['topic']} | {s['evidence']} | {s.get('priority','')} |")
    lines += ["\n## 二、推荐协作伙伴（按综合匹配分排序）\n",
              "| 排名 | 用户 | 匹配分 | 主题重叠 | 语言匹配 | 活跃度 | 匹配理由 |",
              "|------|------|--------|----------|----------|--------|----------|"]
    for i, m in enumerate(cands, 1):
        lines.append(f"| {i} | `{m['login']}` | {m['score']} | {m['topic_overlap']} | "
                     f"{m['language_match']} | {m['activity_level']} | {'; '.join(m['reasons'][:2])} |")
    lines.append(f"\n_候选池规模 {result['meta']['pool_size']}，issue 采样 {result['meta']['issue_sample']}_\n")
    return "\n".join(lines)


def render_mermaid(result: dict[str, Any]) -> str:
    repo = result["repo"].replace("/", "_")
    lines = ["```mermaid", "graph TD", f'  R["{result["repo"]}<br/>(目标仓库)"]']
    for i, m in enumerate(result["candidates"][:8], 1):
        nid = f"C{i}"
        lines.append(f'  {nid}["{m["login"]}<br/>{m["score"]}分"]')
        # 边的粗细用文字标签近似
        lines.append(f'  R -- "{m["topic_overlap"]}" --> {nid}')
    lines.append("```")
    return "\n".join(lines)


# ---------------------------------------------------------------------------

def main():
    ap = argparse.ArgumentParser(description="S4 科研协作智能匹配")
    ap.add_argument("--owner", required=True)
    ap.add_argument("--repo", required=True)
    ap.add_argument("--top", type=int, default=10)
    ap.add_argument("--pool", type=int, default=15, help="候选池上限")
    ap.add_argument("--issue-sample", type=int, default=100)
    ap.add_argument("--out", help="输出目录（写 match.json/report.md/network.mmd）；省略则打印 JSON")
    args = ap.parse_args()

    result = match(args.owner, args.repo, top=args.top, pool_cap=args.pool,
                   issue_sample=args.issue_sample)

    if args.out:
        os.makedirs(args.out, exist_ok=True)
        with open(os.path.join(args.out, "match.json"), "w", encoding="utf-8") as f:
            json.dump(result, f, ensure_ascii=False, indent=2)
        with open(os.path.join(args.out, "report.md"), "w", encoding="utf-8") as f:
            f.write(render_report(result))
        with open(os.path.join(args.out, "network.mmd"), "w", encoding="utf-8") as f:
            f.write(render_mermaid(result))
        print(f"✓ S4 匹配完成 → {args.out}/match.json | report.md | network.mmd")
        print(f"  缺口主题: {', '.join(result['gap_topics'])}")
        print(f"  Top 推荐: {', '.join(m['login']+'('+str(m['score'])+')' for m in result['candidates'][:5])}")
    else:
        print(json.dumps(result, ensure_ascii=False, indent=2))


if __name__ == "__main__":
    main()
