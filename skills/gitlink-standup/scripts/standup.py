"""gitlink-standup：个人/团队日报周报。

汇总一个或多个成员的近期活动（提交、Issue、PR、版本发布），按活动类型统计，
生成适合团队同步（standup）的日报/周报。

数据来自 GitLink 用户动态接口（公开，只读），无需登录。

用法：
    python standup.py --user wbtiger
    python standup.py --users wbtiger,wangyue789 --limit 50
    python standup.py --user wbtiger --format json
"""

from __future__ import annotations

import argparse
import json
import sys
from collections import Counter
from pathlib import Path
from typing import Any

sys.path.insert(0, str(Path(__file__).resolve().parent))
from glapi import GitLinkClient, GitLinkError

if hasattr(sys.stdout, "reconfigure"):
    try:
        sys.stdout.reconfigure(encoding="utf-8")
    except Exception:
        pass

TYPE_LABEL = {
    "CommitLog": "💻 代码提交",
    "Issue": "🐛 Issue",
    "PullRequest": "🔀 合并请求",
    "VersionRelease": "🏷️ 版本发布",
}


def summarize_user(login: str, trends: list[dict[str, Any]]) -> dict[str, Any]:
    """汇总单个成员的活动。"""
    by_type: Counter[str] = Counter()
    samples: dict[str, list[str]] = {}
    for t in trends:
        tt = t.get("trend_type") or "Other"
        by_type[tt] += 1
        name = (t.get("name") or "").splitlines()[0][:70] if t.get("name") else ""
        if name:
            samples.setdefault(tt, [])
            if len(samples[tt]) < 5:
                samples[tt].append(name)
    return {
        "login": login,
        "total_activities": len(trends),
        "by_type": dict(by_type),
        "samples": samples,
    }


def render_report(summaries: list[dict[str, Any]], period: str = "近期") -> str:
    """渲染日报/周报（Markdown）。"""
    lines = [f"# 团队活动{period}报", "",
             f"覆盖 {len(summaries)} 位成员，按 GitLink 活动动态汇总。", ""]
    total_all = sum(s["total_activities"] for s in summaries)
    lines.append(f"总活动数：**{total_all}**")
    lines.append("")
    for s in summaries:
        lines.append(f"## @{s['login']}（{s['total_activities']} 项活动）")
        lines.append("")
        if not s["total_activities"]:
            lines.append("- 该时间范围内暂无公开活动")
            lines.append("")
            continue
        for tt, n in s["by_type"].items():
            label = TYPE_LABEL.get(tt, tt)
            lines.append(f"### {label}：{n}")
            for name in s["samples"].get(tt, []):
                lines.append(f"- {name}")
            lines.append("")
    lines.append("---\n\n由 gitlink-standup 生成。活动数据来自 GitLink 用户动态，仅统计公开活动。")
    return "\n".join(lines)


def analyze(users: list[str], limit: int = 50,
            client: GitLinkClient | None = None) -> list[dict[str, Any]]:
    client = client or GitLinkClient()
    out = []
    for u in users:
        try:
            trends = client.user_trends(u, limit=limit)
        except GitLinkError:
            trends = []
        out.append(summarize_user(u, trends))
    return out


def main(argv: list[str] | None = None) -> int:
    p = argparse.ArgumentParser(prog="gitlink-standup", description="个人/团队日报周报")
    p.add_argument("--user", help="单个成员 login")
    p.add_argument("--users", help="多个成员 login，逗号分隔")
    p.add_argument("--limit", type=int, default=50, help="每人采集的活动条数上限")
    p.add_argument("--period", default="近期", help="报告周期标注，如 日 / 周")
    p.add_argument("--format", choices=["markdown", "json"], default="markdown")
    p.add_argument("--output", type=Path)
    args = p.parse_args(argv)

    users: list[str] = []
    if args.users:
        users = [u.strip() for u in args.users.split(",") if u.strip()]
    elif args.user:
        users = [args.user]
    else:
        print("错误：请用 --user 或 --users 指定成员。", file=sys.stderr)
        return 2

    summaries = analyze(users, limit=args.limit)
    out = (json.dumps(summaries, ensure_ascii=False, indent=2) if args.format == "json"
           else render_report(summaries, period=args.period))
    if args.output:
        args.output.parent.mkdir(parents=True, exist_ok=True)
        args.output.write_text(out, encoding="utf-8")
        print(f"已写入 {args.output}")
    else:
        print(out)
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
