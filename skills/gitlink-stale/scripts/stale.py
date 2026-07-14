"""gitlink-stale：陈旧 Issue/PR 清理。

检测仓库中久未更新的开放 Issue 与 PR，按陈旧程度分级（活跃/留意/陈旧/僵尸），
生成清理建议清单，帮助维护者控制积压。

数据来自 GitLink 公开 API（只读），无需登录。生成的只是建议清单，
是否关闭由维护者通过 gitlink-cli 自行决定。

用法：
    python stale.py --owner Gitlink --repo gitlink-cli
    python stale.py --owner Gitlink --repo gitlink-cli --format json
"""

from __future__ import annotations

import argparse
import json
import re
import sys
from pathlib import Path
from typing import Any

sys.path.insert(0, str(Path(__file__).resolve().parent))
from glapi import GitLinkClient, GitLinkError, split_owner_repo

if hasattr(sys.stdout, "reconfigure"):
    try:
        sys.stdout.reconfigure(encoding="utf-8")
    except Exception:
        pass


def parse_relative_days(text: str) -> int | None:
    """把相对时间（如 "3天前"/"2个月前"/"1年前"/"刚刚"）粗略换算成天数。"""
    if not text:
        return None
    t = str(text).strip()
    if "刚" in t or "分钟" in t or "秒" in t or "小时" in t:
        return 0
    m = re.search(r"(\d+)\s*天", t)
    if m:
        return int(m.group(1))
    m = re.search(r"(\d+)\s*周", t)
    if m:
        return int(m.group(1)) * 7
    m = re.search(r"(\d+)\s*个?月", t)
    if m:
        return int(m.group(1)) * 30
    m = re.search(r"(\d+)\s*年", t)
    if m:
        return int(m.group(1)) * 365
    return None


# 陈旧度分级阈值（天）
def grade(days: int | None) -> str:
    if days is None:
        return "未知"
    if days <= 14:
        return "活跃"
    if days <= 45:
        return "留意"
    if days <= 120:
        return "陈旧"
    return "僵尸"


GRADE_EMOJI = {"活跃": "🟢", "留意": "🟡", "陈旧": "🟠", "僵尸": "🔴", "未知": "⚪"}


def _age_days(item: dict[str, Any]) -> int | None:
    """优先用 updated_at/created_at 的相对时间估算停滞天数。"""
    for key in ("updated_at", "created_at", "time_ago", "pr_time"):
        v = item.get(key)
        d = parse_relative_days(v) if v else None
        if d is not None:
            return d
    return None


def analyze_stale(issues: list[dict[str, Any]], pulls: list[dict[str, Any]]) -> dict[str, Any]:
    """分析开放 Issue/PR 的陈旧度。"""
    def _scan(items, kind):
        out = []
        for it in items:
            # 只看开放项
            if kind == "pr" and it.get("pull_request_status") not in (0, None):
                continue
            if kind == "issue":
                status = str(it.get("issue_status") or "")
                if "关" in status or "closed" in status.lower():
                    continue
            days = _age_days(it)
            g = grade(days)
            out.append({
                "kind": kind,
                "id": str(it.get("pull_request_number") or it.get("id") or ""),
                "title": str(it.get("name") or it.get("subject") or it.get("title") or "")[:60],
                "age_days": days,
                "grade": g,
                "author": it.get("author_login") or it.get("author_name") or "",
            })
        return out

    records = _scan(issues, "issue") + _scan(pulls, "pr")
    by_grade: dict[str, int] = {}
    for r in records:
        by_grade[r["grade"]] = by_grade.get(r["grade"], 0) + 1
    # 需要清理的：陈旧 + 僵尸，按停滞天数降序
    cleanup = sorted(
        [r for r in records if r["grade"] in ("陈旧", "僵尸")],
        key=lambda x: x["age_days"] or 0, reverse=True,
    )
    return {
        "total_open": len(records),
        "by_grade": by_grade,
        "cleanup_count": len(cleanup),
        "cleanup": cleanup,
        "records": records,
    }


def render_report(result: dict[str, Any], owner: str, repo: str) -> str:
    lines = [
        f"# 陈旧 Issue/PR 清理报告 — {owner}/{repo}",
        "",
        f"开放项共 {result['total_open']} 个。陈旧度分布：",
        "",
    ]
    for g in ("活跃", "留意", "陈旧", "僵尸", "未知"):
        n = result["by_grade"].get(g, 0)
        if n:
            lines.append(f"- {GRADE_EMOJI[g]} {g}：{n}")
    lines.append("")
    lines.append("> 分级标准：活跃 ≤14 天 / 留意 ≤45 天 / 陈旧 ≤120 天 / 僵尸 >120 天（按最近更新估算）。")
    lines.append("")
    if result["cleanup"]:
        lines += [f"## 建议处理（{result['cleanup_count']} 个陈旧/僵尸项）", "",
                  "| 类型 | 编号 | 标题 | 停滞 | 等级 |", "|:----:|:----:|------|:----:|:----:|"]
        for c in result["cleanup"][:30]:
            kind = "Issue" if c["kind"] == "issue" else "PR"
            age = f"{c['age_days']}天" if c["age_days"] is not None else "未知"
            lines.append(f"| {kind} | {c['id']} | {c['title']} | {age} | {GRADE_EMOJI[c['grade']]}{c['grade']} |")
        lines += ["", "## 建议行动", "",
                  "1. 对「僵尸」项：评估是否仍有意义，无意义可关闭或加 `wontfix` 标签。",
                  "2. 对「陈旧」项：@相关人确认进展，或补充信息后重新激活。",
                  "3. 关闭操作：`gitlink-cli issue +close --number <web序号>`（写操作，需确认）。", ""]
    else:
        lines += ["🎉 没有发现明显陈旧的开放项，积压控制良好。", ""]
    lines.append("---\n\n由 gitlink-stale 生成。分析只读，关闭操作由维护者确认执行。")
    return "\n".join(lines)


def analyze(owner: str, repo: str, client: GitLinkClient | None = None) -> dict[str, Any]:
    client = client or GitLinkClient()
    issues = client.issues(owner, repo, limit=50)
    pulls = client.pulls(owner, repo, limit=50)
    return analyze_stale(issues, pulls)


def main(argv: list[str] | None = None) -> int:
    p = argparse.ArgumentParser(prog="gitlink-stale", description="陈旧 Issue/PR 清理")
    p.add_argument("--owner"); p.add_argument("--repo"); p.add_argument("--slug")
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
        result = analyze(owner, repo)
    except GitLinkError as exc:
        print(f"采集失败：{exc}", file=sys.stderr)
        return 1

    out = (json.dumps(result, ensure_ascii=False, indent=2) if args.format == "json"
           else render_report(result, owner, repo))
    if args.output:
        args.output.parent.mkdir(parents=True, exist_ok=True)
        args.output.write_text(out, encoding="utf-8")
        print(f"已写入 {args.output}")
    else:
        print(out)
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
