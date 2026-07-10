"""graph_build.py — S2 科研热点追踪与知识图谱。

输入一组科研关键词，从 GitLink 平台按关键词搜索相关仓库（search +repos），
对每个候选仓库取 repo_info / contributors / languages / README，再用
networkx.MultiDiGraph 构建一张「仓库—学者—主题」科研知识图谱：

  节点
    - repo    : id = repo:owner/name        (props: language/stars/forks/desc)
    - scholar : id = scholar:login          (来自 contributors，过滤 bot/i-robot)
    - topic   : id = topic:x                (由 topics.py 词典抽取)
  边
    - contributes_to  : scholar → repo      (weight = contribution_perc 解析为 0~1)
    - owns            : scholar → repo      (当 author.login == contributor login)
    - covers_topic    : repo → topic        (weight = 出现次数 / max)
    - collaborates_with : scholar → scholar (共享同一 repo)
    - related_to      : topic ↔ topic       (在同一 repo 共现)

「取数」与「建图」严格分离：build_graph() 只接收已经取好的 Python 数据结构，
便于离线单测（不联网、不调 gitlink-cli）。collect() 负责在线取数。

数据全部经 gitlink-cli 获取（search +repos / repo +info / repo +contributors /
repo +languages / repo +readme）。

用法：
  python graph_build.py --keywords "deep learning,nlp" --repos-limit 20 --out ./out
  python graph_build.py --keywords "knowledge graph"   # 仅打印 JSON
"""
from __future__ import annotations

import argparse
import datetime as _dt
import json
import math
import os
import sys
import time
from collections import Counter, defaultdict
from typing import Any

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
import collect as c  # noqa: E402
import topics as T  # noqa: E402

import networkx as nx  # noqa: E402


# ---------------------------------------------------------------------------
# 小工具
# ---------------------------------------------------------------------------

BOT_LOGIN_HINTS = ("bot", "i-robot", "dependabot", "renovate", "semantic-release-bot")


def is_bot(login: str) -> bool:
    """识别明显机器人账号（不作为学者节点）。"""
    if not login:
        return True
    low = login.lower()
    return any(h in low for h in BOT_LOGIN_HINTS)


def parse_ratio(v: Any) -> float:
    """把 '1.18%' / '0.2' / 0.2 等统一解析为 0~1 比例（与 match.py 一致）。"""
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


def _topic_heat(descriptions: list[str], top: int = 10) -> list[dict]:
    """对全部仓库 description 跑 topics.topic_counter，取 top 热度榜。"""
    cnt = T.topic_counter(descriptions)
    return [{"topic": t, "count": n} for t, n in cnt.most_common(top)]


# ---------------------------------------------------------------------------
# 热点追踪：飙升项目 + 活跃讨论（快照代理；真·增长率需定时轮询存历史）
# ---------------------------------------------------------------------------

def _to_epoch(v: Any) -> float:
    """把 GitLink 时间（ISO 字符串或整数秒）解析为 epoch 秒，失败返回 0.0。"""
    if v is None:
        return 0.0
    if isinstance(v, (int, float)):
        return float(v) / 1000.0 if v > 1e12 else float(v)
    s = str(v).strip()
    if not s:
        return 0.0
    if s.isdigit():
        f = float(s)
        return f / 1000.0 if f > 1e12 else f
    iso = s.replace("Z", "+00:00")
    try:
        return _dt.datetime.fromisoformat(iso).timestamp()
    except (ValueError, TypeError):
        import re
        m = re.search(r"(\d{4})-(\d{2})-(\d{2})", s)
        if m:
            try:
                return _dt.datetime(int(m.group(1)), int(m.group(2)), int(m.group(3))).timestamp()
            except ValueError:
                return 0.0
        return 0.0


def _iso_day(epoch: float) -> str:
    if not epoch:
        return ""
    try:
        return _dt.datetime.utcfromtimestamp(epoch).strftime("%Y-%m-%d")
    except (OSError, ValueError, OverflowError):
        return ""


def compute_trending(repos: list[dict], top: int = 15) -> list[dict]:
    """对搜索到的仓库按「热度分数」排序，作为飙升/热门项目代理。

    分数 = star + fork×2 + 近期更新加成；并给「日均增星(velocity)」作为
    day-1 可用的趋势代理（真·增长率需定时轮询存历史快照，见 trends 表规划）。
    """
    now = time.time()
    scored: list[dict] = []
    for r in repos:
        fullname = r.get("fullname") or c.repo_fullname(r)
        if not fullname:
            continue
        stars = c.as_int(r.get("praises_count"))
        forks = c.as_int(r.get("forked_count"))
        updated = _to_epoch(r.get("updated_at") or r.get("time")
                            or r.get("updated_on") or r.get("created_at"))
        age_days = max(1.0, (now - updated) / 86400.0) if updated else 99999.0
        recency = max(0.0, 60.0 - age_days)  # 60 天内更新有加成
        score = stars + forks * 2 + recency
        velocity = round(stars / age_days, 3) if age_days < 99990 else 0.0
        lang_obj = r.get("language")
        lang = lang_obj.get("name") if isinstance(lang_obj, dict) else c.as_str(lang_obj)
        scored.append({
            "repo": fullname,
            "description": c.as_str(r.get("description"))[:140],
            "language": lang or "",
            "stars": stars,
            "forks": forks,
            "updated": _iso_day(updated),
            "velocity": velocity,
            "score": round(score, 1),
        })
    scored.sort(key=lambda x: -x["score"])
    return scored[:top]


def compute_active(issues_map: dict[str, list], prs_map: dict[str, list],
                   top: int = 12) -> list[dict]:
    """跨仓库取评论/日志最多的 Issue / PR，作为「活跃讨论」信号。"""
    items: list[dict] = []
    for fullname, issues in issues_map.items():
        for iss in issues or []:
            if not isinstance(iss, dict):
                continue
            jc = c.as_int(iss.get("journals_count") or iss.get("comments_count"))
            items.append({
                "repo": fullname, "type": "issue",
                "number": iss.get("index") or iss.get("number") or iss.get("id"),
                "title": c.as_str(iss.get("subject") or iss.get("title"))[:120],
                "comments": jc,
                "state": c.as_str(iss.get("status") or iss.get("issue_status") or "open"),
            })
    for fullname, prs in prs_map.items():
        for pr in prs or []:
            if not isinstance(pr, dict):
                continue
            jc = c.as_int(pr.get("journals_count") or pr.get("comments_count"))
            items.append({
                "repo": fullname, "type": "pr",
                "number": pr.get("index") or pr.get("number") or pr.get("id"),
                "title": c.as_str(pr.get("title"))[:120],
                "comments": jc,
                "state": c.as_str(pr.get("status") or "open"),
            })
    items.sort(key=lambda x: -x["comments"])
    # 优先有评论的；不足则按已有顺序补齐
    commented = [it for it in items if it["comments"] > 0]
    return (commented or items)[:top]


# ---------------------------------------------------------------------------
# 建图（纯函数：不联网，只吃已取数据，便于单测）
# ---------------------------------------------------------------------------

def build_graph(repos: list[dict],
                contributors_map: dict[str, list[dict]],
                languages_map: dict[str, dict],
                readmes: dict[str, str],
                keywords: list[str] | None = None) -> dict[str, Any]:
    """构建科研知识图谱。

    参数（全部为「已取好」的 Python 数据，不联网）：
      repos            : list[dict]，每个元素至少含 fullname 与 repo_info 字段
                         （identifier, author.login, description, language.name,
                          praises_count, forked_count）。
      contributors_map : {fullname: [contributor, ...]}，contributor 至少含
                         login + contribution_perc。
      languages_map    : {fullname: {"Python": "99.7%", ...}}。
      readmes          : {fullname: readme 文本（已截断到前 4000 字符）}。

    返回结果 dict（与 graph.json 结构一致）。
    """
    G = nx.MultiDiGraph()

    repo_nodes: dict[str, dict] = {}
    scholar_repos: dict[str, set[str]] = defaultdict(set)
    repo_topics: dict[str, Counter] = {}
    descriptions: list[str] = []
    max_topic_count = 1  # 用于 covers_topic 归一化（防除零）

    # ---- 1) 仓库 + 主题节点 ----
    for r in repos:
        fullname = r.get("fullname") or c.repo_fullname(r)
        if not fullname:
            continue
        desc = c.as_str(r.get("description"))
        lang_name = ""
        lang_obj = r.get("language")
        if isinstance(lang_obj, dict):
            lang_name = c.as_str(lang_obj.get("name"))
        stars = c.as_int(r.get("praises_count"))
        forks = c.as_int(r.get("forked_count"))
        readme = c.as_str(readmes.get(fullname))[:4000]
        descriptions.append(desc)

        repo_id = f"repo:{fullname}"
        props = {
            "language": lang_name,
            "stars": stars,
            "forks": forks,
            "description": desc,
            "readme_head": readme[:200],
        }
        G.add_node(repo_id, type="repo", label=fullname, **props)
        repo_nodes[repo_id] = {"label": fullname, **props}

        # 抽取主题（description + readme）
        tps = T.extract_topics(desc + " " + readme)
        cnt: Counter = Counter()
        for tp in tps:
            cnt[tp] += 1
        repo_topics[repo_id] = cnt
        if cnt:
            max_topic_count = max(max_topic_count, max(cnt.values()))

    # ---- 2) 主题节点 + covers_topic 边 ----
    topic_repos: dict[str, set[str]] = defaultdict(set)
    repo_topic_pairs: dict[str, list[str]] = {}  # repo_id -> [topic_id]
    for repo_id, cnt in repo_topics.items():
        pairs: list[str] = []
        for tp, n in cnt.items():
            topic_id = f"topic:{tp}"
            if topic_id not in G:
                G.add_node(topic_id, type="topic", label=tp, count=0,
                           language="", stars=0, forks=0, description="")
            # 累计该主题被多少仓库覆盖
            G.nodes[topic_id]["count"] += 1
            weight = n / max_topic_count if max_topic_count else 0.0
            G.add_edge(repo_id, topic_id, type="covers_topic",
                       weight=round(weight, 4))
            topic_repos[tp].add(repo_id)
            pairs.append(topic_id)
        repo_topic_pairs[repo_id] = pairs

    # ---- 3) 学者节点 + contributes_to / owns ----
    for fullname, contribs in contributors_map.items():
        repo_id = f"repo:{fullname}"
        if repo_id not in G:
            continue
        owner_login = ""
        # 从 repos 列表里取该仓库 author.login（判定 owns）
        for r in repos:
            if (r.get("fullname") or c.repo_fullname(r)) == fullname:
                owner_login = c.login_of(r.get("author") or {})
                break
        for contrib in contribs:
            login = c.login_of(contrib)
            if is_bot(login):
                continue
            scholar_id = f"scholar:{login}"
            if scholar_id not in G:
                G.add_node(scholar_id, type="scholar", label=login,
                           language="", stars=0, forks=0, description="")
            weight = parse_ratio(contrib.get("contribution_perc"))
            G.add_edge(scholar_id, repo_id, type="contributes_to",
                       weight=round(weight, 4))
            if login == owner_login:
                G.add_edge(scholar_id, repo_id, type="owns", weight=1.0)
            scholar_repos[login].add(repo_id)

    # ---- 4) collaborates_with（共享同一 repo 的两两学者）----
    for contribs in contributors_map.values():
        logins = [c.login_of(x) for x in contribs if not is_bot(c.login_of(x))]
        logins = sorted(set(logins))
        if len(logins) < 2:
            continue
        for i in range(len(logins)):
            for j in range(i + 1, len(logins)):
                a = f"scholar:{logins[i]}"
                b = f"scholar:{logins[j]}"
                # 双向（无向语义；MultiDiGraph 用两条边近似）
                G.add_edge(a, b, type="collaborates_with", weight=1.0)
                G.add_edge(b, a, type="collaborates_with", weight=1.0)

    # ---- 5) related_to（同一 repo 内共现的两两主题）----
    for repo_id, tids in repo_topic_pairs.items():
        for i in range(len(tids)):
            for j in range(i + 1, len(tids)):
                a, b = tids[i], tids[j]
                G.add_edge(a, b, type="related_to", weight=1.0)
                G.add_edge(b, a, type="related_to", weight=1.0)

    # ---- 6) 导出 ----
    nodes_out = []
    for nid, attrs in G.nodes(data=True):
        nodes_out.append({
            "id": nid,
            "type": attrs.get("type", ""),
            "label": attrs.get("label", nid),
            "props": {k: v for k, v in attrs.items()
                      if k not in ("type", "label")},
        })
    edges_out = []
    for u, v, attrs in G.edges(data=True):
        edges_out.append({
            "source": u,
            "target": v,
            "type": attrs.get("type", ""),
            "weight": attrs.get("weight", 1.0),
        })

    # 核心学者：按出现 repo 数排序
    core_scholars = sorted(
        ({"login": lg, "repo_count": len(rs)} for lg, rs in scholar_repos.items()),
        key=lambda x: (-x["repo_count"], x["login"]),
    )[:15]

    # 核心团队：仅统计「组织」类型(非个人 User)的仓库拥有者，按拥有仓库数排序。
    # （个人账号不算团队；author.type 区分 User / Organization）
    owner_count: dict[str, int] = {}
    owner_is_org: dict[str, bool] = {}
    for r in repos:
        author = r.get("author") or {}
        lg = c.login_of(author)
        if not lg:
            continue
        owner_count[lg] = owner_count.get(lg, 0) + 1
        tp = str(author.get("type", "")).lower()
        if tp and tp not in ("user", ""):
            owner_is_org[lg] = True
    owner_logins = sorted(
        [{"login": lg, "repo_count": cnt, "type": "organization"}
         for lg, cnt in owner_count.items() if owner_is_org.get(lg)],
        key=lambda x: (-x["repo_count"], x["login"]),
    )

    # 主题热度：取所有 description 的 top10（含图谱里实际命中的 count）
    heat = _topic_heat(descriptions, top=10)

    return {
        "scenario": "S2_research_knowledge_graph",
        "keywords": list(keywords or []),
        "nodes": nodes_out,
        "edges": edges_out,
        "core_scholars": core_scholars,
        "core_teams": owner_logins,
        "topic_heat": heat,
        "meta": {
            "keywords": list(keywords or []),
            "repo_count": len(repo_nodes),
            "node_count": G.number_of_nodes(),
            "edge_count": G.number_of_edges(),
            "scholar_count": sum(1 for n in nodes_out if n["type"] == "scholar"),
            "topic_count": sum(1 for n in nodes_out if n["type"] == "topic"),
        },
    }


# ---------------------------------------------------------------------------
# 取数（在线：调 gitlink-cli）
# ---------------------------------------------------------------------------

def collect(keywords: list[str], repos_limit: int = 20) -> dict[str, Any]:
    """按关键词搜索仓库并取其 info/contributors/languages/readme，返回原始数据。

    与 build_graph() 解耦：本函数可被替换为 mock（单测里直接构造数据喂 build_graph）。
    """
    seen: dict[str, dict] = {}  # fullname -> 归一化的 repo dict
    for kw in keywords:
        for r in c.search_repos(kw, limit=repos_limit):
            fullname = c.repo_fullname(r)
            if not fullname or fullname in seen:
                continue
            seen[fullname] = _normalize_search_hit(r, fullname)
        if len(seen) >= repos_limit:
            break

    repos = list(seen.values())[:repos_limit]
    contributors_map: dict[str, list[dict]] = {}
    languages_map: dict[str, dict] = {}
    readmes: dict[str, str] = {}
    issues_map: dict[str, list[dict]] = {}
    prs_map: dict[str, list[dict]] = {}
    for r in repos:
        fullname = r["fullname"]
        owner, _, name = fullname.partition("/")
        # 用仓库完整 info 覆盖搜索结果的稀疏字段
        info = c.repo_info(owner, name)
        if info:
            r["description"] = c.as_str(info.get("description")) or r.get("description", "")
            r["praises_count"] = c.as_int(info.get("praises_count") or info.get("watchers_count"))
            r["forked_count"] = c.as_int(info.get("forked_count"))
            # 更新时间（飙升/热度排序用；GitLink 字段名兜底多个）
            r["updated_at"] = (info.get("updated_at") or info.get("time")
                               or info.get("updated_on") or r.get("updated_at"))
            if info.get("language") and isinstance(info["language"], dict):
                r["language"] = info["language"]
        contributors_map[fullname] = c.contributors(owner, name, limit=100)
        languages_map[fullname] = c.languages(owner, name)
        readmes[fullname] = c.readme(owner, name)[:4000]
        # 活跃讨论：开放的 Issue / PR（评论多的=热讨论）
        issues_map[fullname] = c.issues(owner, name, state="open", max_pages=1, page_size=50)
        prs_map[fullname] = c.prs(owner, name, state="open", max_pages=1, page_size=50)
    return {"repos": repos, "contributors_map": contributors_map,
            "languages_map": languages_map, "readmes": readmes,
            "issues_map": issues_map, "prs_map": prs_map}


def _normalize_search_hit(r: dict, fullname: str) -> dict:
    """把 search_repos 返回项归一化为 build_graph 期望的形状。"""
    lang_obj = r.get("language")
    if not isinstance(lang_obj, dict):
        lang_obj = {"name": c.as_str(lang_obj)}
    return {
        "fullname": fullname,
        "identifier": r.get("identifier", fullname.split("/")[-1]),
        "author": r.get("author") or {},
        "description": c.as_str(r.get("description")),
        "language": lang_obj,
        "praises_count": c.as_int(r.get("praises_count")),
        "forked_count": c.as_int(r.get("forked_count")),
        "forked_from_project_id": r.get("forked_from_project_id"),
    }


# ---------------------------------------------------------------------------
# 渲染：Mermaid / DOT / Markdown 报告
# ---------------------------------------------------------------------------

_NODE_LIMIT = 40  # 防止 Mermaid 爆炸

_NODE_STYLE = {
    "repo": ("repoNode", "#4C78A8"),
    "scholar": ("scholarNode", "#F58518"),
    "topic": ("topicNode", "#54A24B"),
}


def _safe_id(nid: str) -> str:
    """Mermaid/DOT 节点 id 用安全字符（去冒号斜杠）。"""
    return nid.replace(":", "_").replace("/", "_").replace("-", "_")


def render_mermaid(result: dict[str, Any], node_limit: int = _NODE_LIMIT) -> str:
    """渲染前 ~40 节点的 Mermaid graph TD（带 classDef 着色）。"""
    lines = ["```mermaid", "graph TD"]
    # 类定义
    for t, (cls, color) in _NODE_STYLE.items():
        lines.append(f"  classDef {cls} fill:{color},stroke:#333,color:#fff;")

    nodes = result.get("nodes", [])
    edges = result.get("edges", [])
    # 取前 node_limit 个节点（repo 优先，再 scholar，再 topic）
    type_order = {"repo": 0, "scholar": 1, "topic": 2}
    ordered = sorted(nodes, key=lambda n: (type_order.get(n["type"], 9), n["id"]))
    picked = ordered[:node_limit]
    picked_ids = {n["id"] for n in picked}

    label_map: dict[str, str] = {}
    for n in picked:
        sid = _safe_id(n["id"])
        label = n["label"].replace('"', "'")
        lines.append(f'  {sid}["{label}"]')
        label_map[n["id"]] = sid
        cls = _NODE_STYLE.get(n["type"], ("", ""))[0]
        if cls:
            lines.append(f"  class {sid} {cls};")

    # 只画两端都在 picked 内的边；去重（同源同目标同类只画一条）
    seen_edge: set[tuple] = set()
    for e in edges:
        if e["source"] not in picked_ids or e["target"] not in picked_ids:
            continue
        key = (e["source"], e["target"], e["type"])
        if key in seen_edge:
            continue
        seen_edge.add(key)
        a = label_map[e["source"]]
        b = label_map[e["target"]]
        w = e.get("weight", 1.0)
        et = e["type"]
        # 不同边类型用不同箭头标签
        lines.append(f'  {a} -- "{et}({w:.2f})" --> {b}')

    lines.append("```")
    return "\n".join(lines)


def render_dot(result: dict[str, Any], node_limit: int = _NODE_LIMIT) -> str:
    """渲染 Graphviz DOT 字符串（带节点着色）。"""
    lines = ["digraph G {", '  rankdir=LR;',
             '  graph [fontname="Helvetica"];',
             '  node [fontname="Helvetica", style="filled"];',
             '  edge [fontname="Helvetica"];']
    type_order = {"repo": 0, "scholar": 1, "topic": 2}
    nodes = result.get("nodes", [])
    edges = result.get("edges", [])
    ordered = sorted(nodes, key=lambda n: (type_order.get(n["type"], 9), n["id"]))
    picked = ordered[:node_limit]
    picked_ids = {n["id"] for n in picked}
    label_map: dict[str, str] = {}
    for n in picked:
        sid = _safe_id(n["id"])
        label = n["label"].replace('"', "'")
        color = _NODE_STYLE.get(n["type"], ("", "#CCCCCC"))[1]
        lines.append(f'  {sid} [label="{label}", fillcolor="{color}"];')
        label_map[n["id"]] = sid
    for e in edges:
        if e["source"] not in picked_ids or e["target"] not in picked_ids:
            continue
        a = label_map[e["source"]]
        b = label_map[e["target"]]
        lines.append(f'  {a} -> {b} [label="{e["type"]}"];')
    lines.append("}")
    return "\n".join(lines)


def render_report(result: dict[str, Any]) -> str:
    meta = result["meta"]
    heat = result.get("topic_heat", [])
    scholars = result.get("core_scholars", [])
    teams = result.get("core_teams", [])
    lines = [
        "# 科研热点追踪与知识图谱报告\n",
        f"> 场景 S2 · 子赛题四「应用 GitLink 辅助科研」\n",
        f"**关键词**: {', '.join(result.get('keywords') or []) or '—'}\n",
        "## 一、图谱概览\n",
        f"- 仓库节点: **{meta['repo_count']}**",
        f"- 学者节点: **{meta['scholar_count']}**",
        f"- 主题节点: **{meta['topic_count']}**",
        f"- 节点总数: **{meta['node_count']}**",
        f"- 边总数: **{meta['edge_count']}**\n",
        "## 二、主题热度榜（基于全部仓库 description）\n",
        "| 排名 | 主题 | 覆盖仓库数 |",
        "|------|------|-----------|",
    ]
    if heat:
        for i, h in enumerate(heat, 1):
            lines.append(f"| {i} | `{h['topic']}` | {h['count']} |")
    else:
        lines.append("| — | （未识别到明确主题） | — |")
    lines += ["\n## 三、核心学者（按出现仓库数排序）\n",
              "| 排名 | 学者 | 关联仓库数 |",
              "|------|------|-----------|"]
    if scholars:
        for i, s in enumerate(scholars[:10], 1):
            lines.append(f"| {i} | `{s['login']}` | {s['repo_count']} |")
    else:
        lines.append("| — | （未识别到学者） | — |")
    lines += [f"\n## 四、核心团队（组织型仓库拥有者）\n"]
    if teams:
        lines += ["| 团队/组织 | 拥有仓库数 |", "|-----------|-----------|"]
        for t in teams:
            if isinstance(t, dict):
                lines.append(f"| `{t.get('login')}` | {t.get('repo_count', 0)} |")
            else:
                lines.append(f"| `{t}` | — |")
    else:
        lines.append("（该批仓库均由个人账号拥有，无组织型团队）")
    # 飙升/热门项目（热度分数排序；日均增星 velocity 作趋势代理）
    trending = result.get("trending_repos", [])
    lines += ["\n## 五、热门 / 飙升项目（热度排序）\n",
              "| 仓库 | 语言 | ★ | ⑂ | 日均★ | 最近更新 |",
              "|------|------|---:|---:|---:|----------|"]
    if trending:
        for t in trending[:10]:
            lines.append(f"| `{t['repo']}` | {t['language'] or '—'} | {t['stars']} | "
                         f"{t['forks']} | {t['velocity']} | {t['updated'] or '—'} |")
    else:
        lines.append("| — | （未取到仓库） | | | | |")

    # 活跃讨论
    active = result.get("active_discussions", [])
    lines += ["\n## 六、活跃讨论（评论最多的 Issue / PR）\n",
              "# | 类型 | 仓库 | 标题 | 评论 |", "|-|------|------|------|---:|"]
    if active:
        for i, a in enumerate(active[:10], 1):
            lines.append(f"| {i} | {a['type']} | `{a['repo']}` | {a['title'][:50]} | {a['comments']} |")
    else:
        lines.append("| — | | | （暂无明显热讨论） | |")

    lines.append("\n_配套产物：graph.json（结构化）+ graph.mmd（Mermaid）+ "
                 "graph.dot（Graphviz DOT）_\n")
    return "\n".join(lines)


# ---------------------------------------------------------------------------

def main():
    ap = argparse.ArgumentParser(description="S2 科研热点追踪与知识图谱")
    ap.add_argument("--keywords", required=True,
                    help='逗号分隔的关键词，如 "deep learning,nlp"')
    ap.add_argument("--repos-limit", type=int, default=20,
                    help="每关键词搜索后去重取 top N 仓库（默认 20）")
    ap.add_argument("--out", help="输出目录（写 graph.json/report.md/graph.mmd/graph.dot）；"
                    "省略则打印 JSON")
    args = ap.parse_args()

    keywords = [k.strip() for k in args.keywords.split(",") if k.strip()]
    data = collect(keywords, repos_limit=args.repos_limit)
    result = build_graph(data["repos"], data["contributors_map"],
                         data["languages_map"], data["readmes"], keywords=keywords)
    # 热点追踪两翼：飙升项目 + 活跃讨论（图谱之外的"追踪"信号）
    result["trending_repos"] = compute_trending(data["repos"])
    result["active_discussions"] = compute_active(data["issues_map"], data["prs_map"])

    if args.out:
        os.makedirs(args.out, exist_ok=True)
        with open(os.path.join(args.out, "graph.json"), "w", encoding="utf-8") as f:
            json.dump(result, f, ensure_ascii=False, indent=2)
        with open(os.path.join(args.out, "report.md"), "w", encoding="utf-8") as f:
            f.write(render_report(result))
        with open(os.path.join(args.out, "graph.mmd"), "w", encoding="utf-8") as f:
            f.write(render_mermaid(result))
        with open(os.path.join(args.out, "graph.dot"), "w", encoding="utf-8") as f:
            f.write(render_dot(result))
        print(f"✓ S2 知识图谱完成 → {args.out}/graph.json | report.md | graph.mmd | graph.dot")
        print(f"  节点: {result['meta']['node_count']}  边: {result['meta']['edge_count']}  "
              f"仓库: {result['meta']['repo_count']}")
        heat = result["topic_heat"][:5]
        print(f"  Top 主题: {', '.join(h['topic'] for h in heat)}")
    else:
        print(json.dumps(result, ensure_ascii=False, indent=2))


if __name__ == "__main__":
    main()
