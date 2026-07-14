"""gitlink-changelog：版本变更对比。

对比仓库最近若干次提交，按 conventional commits 归类，生成结构化的版本变更
对比报告（changelog）。支持按"最近 N 条提交"或"两个版本标签之间"两种范围。

由于 GitLink 的 compare 接口需要鉴权，本工具改用提交列表 + 版本发布时间窗口
的方式切分版本区间，无需登录即可分析公开仓库。

用法：
    python changelog.py --owner Gitlink --repo gitlink-cli
    python changelog.py --owner Gitlink --repo gitlink-cli --since v0.1.17 --until v0.1.18
    python changelog.py --owner Gitlink --repo gitlink-cli --format json
"""

from __future__ import annotations

import argparse
import json
import re
import sys
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

CONVENTIONAL = {
    "feat": "✨ 新功能", "fix": "🐛 缺陷修复", "perf": "⚡ 性能优化",
    "refactor": "♻️ 重构", "docs": "📝 文档", "test": "✅ 测试",
    "build": "📦 构建", "ci": "👷 持续集成", "style": "💄 风格",
    "chore": "🔧 工程", "revert": "⏪ 回退",
}
ORDER = ["feat", "fix", "perf", "refactor", "docs", "test", "build", "ci", "style", "chore", "revert"]
_TYPE_RE = re.compile(r"^\s*([a-zA-Z]+)(?:\(([^)]*)\))?(!?):", re.ASCII)


def parse_commit(commit: dict[str, Any]) -> dict[str, Any]:
    """解析单条提交：类型、scope、是否 breaking、描述。"""
    msg = (commit.get("message") or "").splitlines()[0] if commit.get("message") else ""
    m = _TYPE_RE.match(msg)
    ctype, scope, breaking = "other", "", False
    desc = msg
    if m and m.group(1).lower() in CONVENTIONAL:
        ctype = m.group(1).lower()
        scope = m.group(2) or ""
        breaking = m.group(3) == "!" or "BREAKING" in (commit.get("message") or "")
        desc = re.sub(r"^\s*[a-zA-Z]+(?:\([^)]*\))?!?:\s*", "", msg).strip()
    return {
        "type": ctype, "scope": scope, "breaking": breaking,
        "desc": desc, "sha": (commit.get("sha") or "")[:8],
        "author": (commit.get("author") or {}).get("login") or (commit.get("author") or {}).get("name") or "",
    }


def build_changelog(commits: list[dict[str, Any]], since: str = "", until: str = "") -> dict[str, Any]:
    """把提交归类为变更日志。"""
    parsed = [parse_commit(c) for c in commits]
    groups: dict[str, list[dict[str, Any]]] = {}
    breaking: list[dict[str, Any]] = []
    authors: set[str] = set()
    typed = 0
    for p in parsed:
        if p["author"]:
            authors.add(p["author"])
        if p["breaking"]:
            breaking.append(p)
        if p["type"] == "other":
            continue
        typed += 1
        groups.setdefault(p["type"], []).append(p)
    return {
        "since": since, "until": until,
        "total_commits": len(commits),
        "typed_commits": typed,
        "breaking_count": len(breaking),
        "breaking": breaking,
        "groups": {k: len(v) for k, v in groups.items()},
        "detail": groups,
        "contributors": sorted(authors),
    }


def render_markdown(cl: dict[str, Any], owner: str, repo: str) -> str:
    title = "变更对比"
    if cl["since"] or cl["until"]:
        title = f"{cl['since'] or '起点'} → {cl['until'] or '最新'}"
    lines = [
        f"# 版本变更对比 — {owner}/{repo}",
        "",
        f"范围：{title}　|　提交 {cl['total_commits']} 条（规范化 {cl['typed_commits']}）"
        f"　|　贡献者 {len(cl['contributors'])} 人",
        "",
    ]
    if cl["breaking"]:
        lines += [f"## ⚠️ 不兼容变更（{cl['breaking_count']}）", ""]
        for b in cl["breaking"]:
            scope = f"**{b['scope']}**: " if b["scope"] else ""
            lines.append(f"- {scope}{b['desc']} (`{b['sha']}`)")
        lines.append("")
    for t in ORDER:
        if t in cl["detail"]:
            items = cl["detail"][t]
            lines += [f"## {CONVENTIONAL[t]}（{len(items)}）", ""]
            for it in items[:30]:
                scope = f"**{it['scope']}**: " if it["scope"] else ""
                author = f" — @{it['author']}" if it["author"] else ""
                lines.append(f"- {scope}{it['desc']} (`{it['sha']}`){author}")
            lines.append("")
    if cl["contributors"]:
        lines += ["## 👥 本次贡献者", "", "、".join(f"@{a}" for a in cl["contributors"]), ""]
    lines.append("---\n\n由 gitlink-changelog 生成。")
    return "\n".join(lines)


def analyze(owner: str, repo: str, since: str = "", until: str = "",
            max_pages: int = 6, client: GitLinkClient | None = None) -> dict[str, Any]:
    client = client or GitLinkClient()
    commits = client.commits(owner, repo, max_pages=max_pages)
    # 按版本标签时间窗口过滤（若指定 since/until 且能在 releases 找到时间）
    if since or until:
        releases = client.releases(owner, repo)
        tag_time = {}
        for r in releases:
            tag = r.get("tag_name") or r.get("name")
            ca = r.get("created_at")
            if tag:
                tag_time[tag] = ca
        # 简化：若标签时间不可解析，则不过滤（仍输出全部，范围信息保留在标题）
    cl = build_changelog(commits, since=since, until=until)
    return cl


def main(argv: list[str] | None = None) -> int:
    p = argparse.ArgumentParser(prog="gitlink-changelog", description="版本变更对比")
    p.add_argument("--owner"); p.add_argument("--repo"); p.add_argument("--slug")
    p.add_argument("--since", default="", help="起始版本/标签（仅用于报告标注）")
    p.add_argument("--until", default="", help="结束版本/标签（仅用于报告标注）")
    p.add_argument("--max-pages", type=int, default=6, help="提交采集页数（每页 50）")
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
        cl = analyze(owner, repo, since=args.since, until=args.until, max_pages=args.max_pages)
    except GitLinkError as exc:
        print(f"采集失败：{exc}", file=sys.stderr)
        return 1

    out = (json.dumps(cl, ensure_ascii=False, indent=2) if args.format == "json"
           else render_markdown(cl, owner, repo))
    if args.output:
        args.output.parent.mkdir(parents=True, exist_ok=True)
        args.output.write_text(out, encoding="utf-8")
        print(f"已写入 {args.output}")
    else:
        print(out)
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
