"""gitlink-metrics：仓库量化指标看板。

把仓库的协作数据转化为一组量化指标，生成数据驱动的运营看板：
- 规模指标：Issue/PR/贡献者/版本数
- PR 合并率与开放占比
- 提交活跃趋势（按月）
- 贡献集中度：基尼系数、CR3/CR5、巴士因子
- 多维评分卡（0-100）：活跃度 / 协作 / 社区 / 可维护性

相比 gitlink-insight（偏定性的健康度与周报），本技能聚焦**量化指标与评分卡**，
用可计算、可对比的数字刻画仓库状态，功能更偏数据分析。

数据来自 GitLink 公开 API（只读），无需登录。

用法：
    python metrics.py --owner Gitlink --repo gitlink-cli
    python metrics.py --owner Gitlink --repo gitlink-cli --format json
"""

from __future__ import annotations

import argparse
import json
import sys
from collections import Counter
from datetime import datetime, timezone
from pathlib import Path
from typing import Any

sys.path.insert(0, str(Path(__file__).resolve().parent))
from glapi import GitLinkClient, GitLinkError, split_owner_repo

if hasattr(sys.stdout, "reconfigure"):
    try:
        sys.stdout.reconfigure(encoding="utf-8")
    except Exception:
        pass


def gini(values: list[float]) -> float:
    xs = sorted(v for v in values if v >= 0)
    n = len(xs)
    total = sum(xs)
    if n == 0 or total == 0:
        return 0.0
    cum = sum(i * x for i, x in enumerate(xs, start=1))
    return round((2 * cum) / (n * total) - (n + 1) / n, 3)


def bus_factor(values: list[int]) -> int:
    total = sum(values)
    if total <= 0:
        return 0
    cum = 0
    for i, v in enumerate(sorted(values, reverse=True), start=1):
        cum += v
        if cum / total >= 0.5:
            return i
    return len(values)


def monthly_commits(commits: list[dict[str, Any]]) -> dict[str, int]:
    months: Counter[str] = Counter()
    for c in commits:
        ts = c.get("timestamp")
        try:
            dt = datetime.fromtimestamp(int(ts), tz=timezone.utc)
            months[f"{dt.year:04d}-{dt.month:02d}"] += 1
        except (TypeError, ValueError, OSError):
            continue
    return dict(sorted(months.items()))


def pr_metrics(pulls: list[dict[str, Any]]) -> dict[str, Any]:
    merged = sum(1 for p in pulls if p.get("pull_request_status") == 1)
    closed = sum(1 for p in pulls if p.get("pull_request_status") == 2)
    opened = sum(1 for p in pulls if p.get("pull_request_status") in (0, None))
    total = len(pulls)
    return {
        "total": total, "merged": merged, "closed": closed, "open": opened,
        "merge_rate": round(merged / total * 100, 1) if total else 0.0,
    }


def score_card(info, prm, contribs_vals, months) -> dict[str, int]:
    """多维评分卡（各 0-100）。"""
    # 活跃度：近月提交量
    recent = list(months.values())[-3:] if months else []
    avg_recent = sum(recent) / len(recent) if recent else 0
    activity = min(100, int(avg_recent * 2))
    # 协作：PR 合并率
    collab = int(prm["merge_rate"])
    # 社区：贡献者数 + star/fork
    community = min(100, len(contribs_vals) * 4 + int(info.get("praises_count") or 0)
                    + int(info.get("forked_count") or 0))
    # 可维护性：巴士因子越高越好（封顶）
    bf = bus_factor(contribs_vals)
    maintain = min(100, bf * 20)
    overall = round((activity + collab + community + maintain) / 4)
    return {"活跃度": activity, "协作": collab, "社区": community,
            "可维护性": maintain, "综合": overall}


def analyze(owner: str, repo: str, client: GitLinkClient | None = None,
            commit_pages: int = 6) -> dict[str, Any]:
    client = client or GitLinkClient()
    info = client.repo_info(owner, repo)
    pulls = client.pulls(owner, repo, limit=50)
    commits = client.commits(owner, repo, max_pages=commit_pages)
    contributors = client.contributors(owner, repo)
    contribs_vals = [int(c.get("contributions") or 0) for c in contributors]
    months = monthly_commits(commits)
    prm = pr_metrics(pulls)
    return {
        "owner": owner, "repo": repo,
        "scale": {
            "issues": info.get("issues_count"),
            "pulls": info.get("pull_requests_count"),
            "contributors": len(contributors),
            "releases": info.get("version_releases_count"),
            "stars": info.get("praises_count"),
            "forks": info.get("forked_count"),
        },
        "pr_metrics": prm,
        "monthly_commits": months,
        "concentration": {
            "gini": gini([float(v) for v in contribs_vals]),
            "cr3": round(sum(sorted(contribs_vals, reverse=True)[:3]) / (sum(contribs_vals) or 1) * 100, 1),
            "cr5": round(sum(sorted(contribs_vals, reverse=True)[:5]) / (sum(contribs_vals) or 1) * 100, 1),
            "bus_factor": bus_factor(contribs_vals),
        },
        "score_card": score_card(info, prm, contribs_vals, months),
    }


def _bar(n: int, peak: int, width: int = 18) -> str:
    filled = int(round(n / peak * width)) if peak else 0
    return "█" * filled + "·" * (width - filled)


def render_dashboard(m: dict[str, Any], owner: str, repo: str) -> str:
    sc = m["scale"]; prm = m["pr_metrics"]; con = m["concentration"]; card = m["score_card"]
    lines = [
        f"# 仓库指标看板 — {owner}/{repo}",
        "",
        f"综合评分：**{card['综合']}/100**",
        "",
        "## 一、规模指标",
        "",
        f"| Issue | PR | 贡献者 | 版本 | Star | Fork |",
        f"|:--:|:--:|:--:|:--:|:--:|:--:|",
        f"| {sc['issues']} | {sc['pulls']} | {sc['contributors']} | {sc['releases']} | {sc['stars']} | {sc['forks']} |",
        "",
        "## 二、PR 指标",
        "",
        f"- 合并率：**{prm['merge_rate']}%**（合并 {prm['merged']} / 开放 {prm['open']} / 关闭 {prm['closed']}，共 {prm['total']}）",
        "",
        "## 三、提交活跃趋势（按月）",
        "",
    ]
    months = m["monthly_commits"]
    if months:
        peak = max(months.values())
        for mo, n in months.items():
            lines.append(f"- `{mo}`  {_bar(n, peak)}  {n}")
    else:
        lines.append("- （无可定位时间的提交）")
    lines += [
        "", "## 四、贡献集中度", "",
        f"- 基尼系数：{con['gini']}（0=均衡，1=集中）",
        f"- CR3 / CR5：{con['cr3']}% / {con['cr5']}%",
        f"- 巴士因子：{con['bus_factor']}（约需多少人离开会影响过半知识延续）",
        "", "## 五、多维评分卡", "",
        "| 维度 | 评分 |", "|------|:----:|",
    ]
    for k in ("活跃度", "协作", "社区", "可维护性", "综合"):
        lines.append(f"| {k} | {card[k]}/100 |")
    lines += ["", "---", "", "由 gitlink-metrics 生成。所有指标基于 GitLink 公开数据，分析只读。"]
    return "\n".join(lines)


def main(argv: list[str] | None = None) -> int:
    p = argparse.ArgumentParser(prog="gitlink-metrics", description="仓库量化指标看板")
    p.add_argument("--owner"); p.add_argument("--repo"); p.add_argument("--slug")
    p.add_argument("--commit-pages", type=int, default=6)
    p.add_argument("--format", choices=["markdown", "json"], default="markdown")
    p.add_argument("--output", type=Path)
    args = p.parse_args(argv)

    if args.slug:
        owner, repo = split_owner_repo(args.slug)
    elif args.owner and args.repo:
        owner, repo = args.owner, args.repo
    else:
        print("错误：请用 --owner/--repo 或 --slug 指定仓库。", file=sys.stderr)
        return 2

    try:
        m = analyze(owner, repo, commit_pages=args.commit_pages)
    except GitLinkError as exc:
        print(f"采集失败：{exc}", file=sys.stderr)
        return 1

    out = (json.dumps(m, ensure_ascii=False, indent=2) if args.format == "json"
           else render_dashboard(m, owner, repo))
    if args.output:
        args.output.parent.mkdir(parents=True, exist_ok=True)
        args.output.write_text(out, encoding="utf-8")
        print(f"已写入 {args.output}")
    else:
        print(out)
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
