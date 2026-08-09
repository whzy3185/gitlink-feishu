"""report.py — S5 科研进度智能跟踪与预警。

输入一个科研仓库，统计「本周 / 上周」的提交、Issue、PR 活跃度，结合里程碑进度，
用阈值规则产出风险预警（stale issue / stale PR / 逾期里程碑 / 低活跃 / bus_factor），
并给出本周相对上周的 commit 趋势。辅助科研负责人及时发现「项目停滞 / 单点风险」。

数据全部经 gitlink-cli 获取（commits / issue +list / pr +list / milestone +list /
repo +contributors）。算法为纯函数、零第三方依赖，单测以 mock 数据喂入。

用法：
  python report.py --owner mindspore-Ecosystem --repo mindspore --out ./out
  python report.py --owner O --repo R              # 仅打印 JSON
"""
from __future__ import annotations

import argparse
import json
import os
import sys
from collections import Counter
from datetime import datetime, timedelta, timezone
from typing import Any

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
import collect as c  # noqa: E402

# ---------------------------------------------------------------------------
# 常量（阈值，便于单测覆盖）
# ---------------------------------------------------------------------------

STALE_ISSUE_DAYS = 30          # 开放且无活动 > 30 天 → stale issue
STALE_PR_DAYS = 14             # 开放且未 review > 14 天 → stale PR
STALE_PR_WINDOW_DAYS = 90      # 统计 stale PR 时回溯窗口（避免扫全量历史）
LOW_ACTIVITY_COMMITS = 3       # 本周提交 < 3 → 低活跃
BUS_FACTOR_RATIO = 0.5         # 单一贡献者占本周提交 > 50% → bus factor

UTC = timezone.utc


# ---------------------------------------------------------------------------
# 时间解析
# ---------------------------------------------------------------------------

def parse_time(s: Any) -> datetime | None:
    """把时间字段解析为带 UTC 时区的 datetime；无法解析返回 None。

    兼容两类输入：
      - ISO 字串，如 '2024-06-01T08:30:00+08:00' / '2024-06-01T08:30:00Z'
        / '2024-06-01 08:30:00' / '2024-06-01'
      - 整数（或整数字串）秒级 Unix 时间戳，如 1717200000 / '1717200000'
    """
    if s is None:
        return None
    if isinstance(s, (int, float)):
        try:
            return datetime.fromtimestamp(float(s), tz=UTC)
        except (OverflowError, OSError, ValueError):
            return None
    if not isinstance(s, str):
        return None
    text = s.strip()
    if not text:
        return None
    # 纯数字 → 当作 Unix 秒级时间戳
    if text.lstrip("-").isdigit():
        try:
            return datetime.fromtimestamp(float(text), tz=UTC)
        except (OverflowError, OSError, ValueError):
            return None
    # ISO 字串：统一以 Z → +00:00
    iso = text.replace("Z", "+00:00")
    try:
        dt = datetime.fromisoformat(iso)
    except ValueError:
        # 尝试 'YYYY-MM-DD HH:MM:SS' / 'YYYY-MM-DD'
        for fmt in ("%Y-%m-%d %H:%M:%S", "%Y-%m-%d"):
            try:
                dt = datetime.strptime(text, fmt)
                break
            except ValueError:
                continue
        else:
            return None
    if dt.tzinfo is None:
        dt = dt.replace(tzinfo=UTC)
    return dt.astimezone(UTC)


def in_window(dt: datetime | None, days: int, now: datetime) -> bool:
    """判断 dt 是否落在 [now - days, now] 区间内。None 视为不在窗口内。"""
    if dt is None:
        return False
    if days < 0:
        return False
    now_utc = now.astimezone(UTC) if now.tzinfo else now.replace(tzinfo=UTC)
    dt_utc = dt.astimezone(UTC) if dt.tzinfo else dt.replace(tzinfo=UTC)
    return (now_utc - dt_utc) <= timedelta(days=days) and (now_utc - dt_utc) >= timedelta(0)


# ---------------------------------------------------------------------------
# 周统计
# ---------------------------------------------------------------------------

def _commit_time(item: dict) -> datetime | None:
    for key in ("timestamp", "commit_time", "committed_date", "created_at"):
        v = item.get(key)
        dt = parse_time(v)
        if dt is not None:
            return dt
    return None


def _issue_time(item: dict, prefer: tuple[str, ...]) -> datetime | None:
    for key in prefer:
        v = item.get(key)
        dt = parse_time(v)
        if dt is not None:
            return dt
    return None


def _issue_status(item: dict) -> str:
    s = item.get("status")
    if isinstance(s, str) and s:
        return s.lower()
    return ""


def _pr_status(item: dict) -> int:
    """PR 状态 → 0=open,1=merged,2=closed。

    GitLink PR 列表 status 是字符串('merged'/'open'/'closed')；
    详情/health 可能是 pull_request_status 整数(0/1/2)。两者兼容。
    """
    v = item.get("status", item.get("pull_request_status", 0))
    if isinstance(v, str):
        s = v.lower()
        if "merged" in s:
            return 1
        if s in ("closed", "close", "reject", "rejected"):
            return 2
        return 0
    try:
        return int(v)
    except (TypeError, ValueError):
        return 0


def _author_login(item: dict, key: str = "author") -> str:
    """从 commit/issue/pr 的 author 子对象取 login；退化用顶层 login/name。"""
    obj = item.get(key)
    if isinstance(obj, dict):
        for k in ("login", "name", "username"):
            if obj.get(k):
                return str(obj[k])
    for k in ("login", "name", "username", "committer_login"):
        if item.get(k):
            return str(item[k])
    return ""


def _week_bounds(now: datetime) -> tuple[datetime, datetime, datetime, datetime]:
    """返回 (this_week_start, last_week_start, last_week_end, now)（UTC）。

    「周」按自然天对齐：本周 = [now-7d, now]，上周 = [now-14d, now-7d)。
    """
    now_utc = now.astimezone(UTC) if now.tzinfo else now.replace(tzinfo=UTC)
    this_start = now_utc - timedelta(days=7)
    last_start = now_utc - timedelta(days=14)
    last_end = this_start
    return this_start, last_start, last_end, now_utc


def _commits_in(commits: list[dict], lo: datetime, hi: datetime) -> list[dict]:
    out = []
    for it in commits:
        dt = _commit_time(it)
        if dt is None:
            continue
        if lo <= dt < hi:
            out.append(it)
    return out


def _window_summary(commits: list, issues: list, prs: list,
                    contributors: list, lo: datetime, hi: datetime,
                    now: datetime) -> dict[str, Any]:
    """统计 [lo, hi) 区间内的活动摘要（供 week_stats 复用）。"""
    w_commits = _commits_in(commits, lo, hi)

    opened = closed = stale_open = 0
    for iss in issues:
        created = _issue_time(iss, ("created_at", "created_unix"))
        if created is not None and lo <= created < hi:
            opened += 1
        st = _issue_status(iss)
        if st in ("closed", "reject", "rejected"):
            # 关闭时间优先 closed_at/journals_updated_at
            closed_dt = _issue_time(iss, ("closed_at", "journals_updated_at", "updated_at"))
            if closed_dt is not None and lo <= closed_dt < hi:
                closed += 1
        elif st in ("", "open", "opened"):
            # stale 判定：开放且距今 > STALE_ISSUE_DAYS 无活动
            last_dt = _issue_time(iss, ("journals_updated_at", "updated_at", "created_at"))
            if last_dt is not None and (now - last_dt) > timedelta(days=STALE_ISSUE_DAYS):
                stale_open += 1

    pr_opened = pr_merged = pr_open_stale = 0
    for pr in prs:
        created = _issue_time(pr, ("pr_created_unix", "created_at", "created_unix"))
        if created is None:
            created = _commit_time(pr)
        if created is not None and lo <= created < hi:
            pr_opened += 1
        st = _pr_status(pr)
        if st == 1:  # merged
            merged_dt = _issue_time(pr, ("pr_merged_unix", "merged_at", "updated_at"))
            if merged_dt is None:
                merged_dt = created
            if merged_dt is not None and lo <= merged_dt < hi:
                pr_merged += 1
        elif st == 0:  # open
            # stale PR：开放且 created > STALE_PR_DAYS 天前、近 STALE_PR_WINDOW 天内
            if created is not None and (now - created) > timedelta(days=STALE_PR_DAYS) \
                    and created > now - timedelta(days=STALE_PR_WINDOW_DAYS):
                pr_open_stale += 1

    # 本周活跃贡献者：去重 author login（commits）
    logins = [_author_login(it) for it in w_commits]
    contrib_active = {lg for lg in logins if lg}
    return {
        "commits": len(w_commits),
        "issues_opened": opened,
        "issues_closed": closed,
        "issues_stale": stale_open,
        "prs_opened": pr_opened,
        "prs_merged": pr_merged,
        "prs_open_stale": pr_open_stale,
        "contributors_active": len(contrib_active),
        "contributors_active_logins": sorted(contrib_active),
    }


def week_stats(commits: list, issues: list, prs: list,
               contributors: list, now: datetime) -> dict[str, Any]:
    """统计本周 / 上周活跃度摘要。contributors 形参保留以兼容，活跃度以 commits 的作者为准。"""
    this_start, last_start, last_end, now_utc = _week_bounds(now)
    this = _window_summary(commits, issues, prs, contributors,
                           this_start, now_utc, now_utc)
    last = _window_summary(commits, issues, prs, contributors,
                           last_start, last_end, now_utc)
    return {
        "this_week": this,
        "last_week": last,
        "window": {
            "this_week_start": this_start.isoformat(),
            "now": now_utc.isoformat(),
            "last_week_start": last_start.isoformat(),
            "last_week_end": last_end.isoformat(),
        },
        "total_contributors": len(contributors),
    }


# ---------------------------------------------------------------------------
# 里程碑进度
# ---------------------------------------------------------------------------

def _milestone_due(item: dict) -> datetime | None:
    for key in ("effective_date", "due_date", "deadline", "end_date"):
        dt = parse_time(item.get(key))
        if dt is not None:
            return dt
    return None


def milestone_progress(milestones: list, issues: list, now: datetime) -> list[dict]:
    """每个里程碑的 open/closed issue 数、完成率、是否逾期。

    Issue → Milestone 的关联字段优先用 milestone_name / milestone_id。
    """
    now_utc = now.astimezone(UTC) if now.tzinfo else now.replace(tzinfo=UTC)
    by_key: dict[Any, dict[str, int]] = {}
    for iss in issues:
        ms_name = iss.get("milestone_name") or iss.get("milestone")
        ms_id = iss.get("milestone_id") or iss.get("milestone_index")
        key = ms_name if ms_name else (ms_id if ms_id is not None else None)
        if key is None:
            continue
        bucket = by_key.setdefault(key, {"open": 0, "closed": 0})
        st = _issue_status(iss)
        if st in ("closed", "reject", "rejected"):
            bucket["closed"] += 1
        else:
            bucket["open"] += 1

    out: list[dict] = []
    for ms in milestones:
        name = ms.get("name") or ms.get("title") or "(未命名)"
        key = name
        counts = by_key.get(key, {"open": 0, "closed": 0})
        total = counts["open"] + counts["closed"]
        pct = round(100.0 * counts["closed"] / total, 1) if total else 0.0
        due = _milestone_due(ms)
        overdue = False
        # 仅当里程碑未关闭 + 有 due_date 且 due < now → 逾期
        ms_status = str(ms.get("status", "")).lower()
        is_closed = ms_status in ("closed", "reject", "rejected", "done", "completed")
        if due is not None and not is_closed and due < now_utc:
            overdue = True
        out.append({
            "name": name,
            "open": counts["open"],
            "closed": counts["closed"],
            "total": total,
            "completion_pct": pct,
            "due_date": due.isoformat() if due else None,
            "overdue": overdue,
            "status": ms.get("status", ""),
        })
    return out


# ---------------------------------------------------------------------------
# 趋势
# ---------------------------------------------------------------------------

def trend(this_week: dict, last_week: dict) -> dict[str, Any]:
    """本周 vs 上周 commit 增量与活跃度等级。"""
    tc = this_week.get("commits", 0)
    lc = last_week.get("commits", 0)
    if lc == 0:
        delta_pct = 100.0 if tc > 0 else 0.0
    else:
        delta_pct = round(100.0 * (tc - lc) / lc, 1)
    if delta_pct > 10:
        level = "increasing"
    elif delta_pct < -10:
        level = "decreasing"
    else:
        level = "stable"
    return {"commit_delta_pct": delta_pct, "activity_level": level,
            "this_week_commits": tc, "last_week_commits": lc}


# ---------------------------------------------------------------------------
# 风险预警
# ---------------------------------------------------------------------------

def risk_warnings(stats: dict, milestones: list, contributors: list,
                  commits: list | None = None, now: datetime | None = None) -> list[dict]:
    """阈值规则 → 风险列表。

    依赖 week_stats 产出的 stats（含 this_week / last_week）以及 milestone_progress
    的 milestones 列表。commits 用于 bus_factor 复算（可选，避免 stats 内无明细）。
    """
    now_utc = (now or datetime.now(UTC)).astimezone(UTC) \
        if (now or datetime.now(UTC)).tzinfo else (now or datetime.now(UTC)).replace(tzinfo=UTC)
    this_start = now_utc - timedelta(days=7)
    warnings: list[dict] = []
    tw: dict = stats.get("this_week", {})

    # 1) 低活跃
    commits_this = tw.get("commits", 0)
    if commits_this < LOW_ACTIVITY_COMMITS:
        warnings.append({
            "level": "warning", "type": "low_activity",
            "message": f"本周提交仅 {commits_this} 次（低于阈值 {LOW_ACTIVITY_COMMITS}），项目可能进展缓慢",
            "metric": commits_this, "suggestion": "确认是否进入收尾阶段；若无，组织一次进度同步。",
        })

    # 2) bus_factor：单一贡献者本周提交占比 > 50%
    if commits:
        w_commits = _commits_in(commits, this_start, now_utc)
    else:
        w_commits = []  # 无明细 → 无法判 bus factor
    if w_commits:
        login_counts: Counter = Counter(_author_login(it) for it in w_commits)
        top_login, top_n = login_counts.most_common(1)[0]
        ratio = top_n / len(w_commits)
        active_n = len({lg for lg in login_counts if lg})
        if ratio > BUS_FACTOR_RATIO and active_n <= 2:
            warnings.append({
                "level": "critical", "type": "bus_factor",
                "message": f"bus factor 风险：{top_login or '(匿名)'} 一人贡献本周 {top_n}/{len(w_commits)} "
                           f"次提交（{round(ratio*100)}%），活跃贡献者仅 {active_n} 人",
                "metric": round(ratio, 3), "suggestion": "引入第二贡献者 / 文档化核心模块，降低单点依赖。",
            })

    # 3) stale issue / stale PR（沿用 week_stats 已统计的口径）
    stale_iss = tw.get("issues_stale", 0)
    if stale_iss >= 5:
        warnings.append({
            "level": "warning" if stale_iss < 20 else "critical",
            "type": "stale_issue",
            "message": f"存在 {stale_iss} 个开放 Issue 超过 {STALE_ISSUE_DAYS} 天无活动",
            "metric": stale_iss, "suggestion": "分诊：关闭无效 Issue、分配负责人或拆解。",
        })
    stale_pr = tw.get("prs_open_stale", 0)
    if stale_pr >= 1:
        warnings.append({
            "level": "warning" if stale_pr < 3 else "critical",
            "type": "stale_pr",
            "message": f"存在 {stale_pr} 个开放 PR 超过 {STALE_PR_DAYS} 天未 review/合并",
            "metric": stale_pr, "suggestion": "安排 review 或明确 reject，避免 PR 堆积。",
        })

    # 4) 逾期里程碑
    for ms in milestones:
        if isinstance(ms, dict) and ms.get("overdue"):
            warnings.append({
                "level": "critical", "type": "overdue_milestone",
                "message": f"里程碑「{ms.get('name', '?')}」已逾期"
                           + (f"（due {ms.get('due_date')}）" if ms.get("due_date") else "")
                           + f"，完成率 {ms.get('completion_pct', 0)}%",
                "metric": ms.get("completion_pct", 0),
                "suggestion": "重新评估范围或顺延 deadline，并同步干系人。",
            })

    # 排序：critical > warning > info
    rank = {"critical": 0, "warning": 1, "info": 2}
    warnings.sort(key=lambda w: (rank.get(w["level"], 9), w["type"]))
    return warnings


# ---------------------------------------------------------------------------
# 主入口（取数 + 算法）
# ---------------------------------------------------------------------------

def analyze(owner: str, repo: str, now: datetime | None = None) -> dict[str, Any]:
    """取数 + 算法：返回完整结果 dict。"""
    if now is None:
        now = datetime.now(UTC)
    commits = c.commits(owner, repo, max_pages=10)
    issues = c.issues_all(owner, repo)
    prs = c.prs_all(owner, repo)
    milestones = c.milestones(owner, repo)
    contributors = c.contributors(owner, repo)

    stats = week_stats(commits, issues, prs, contributors, now)
    ms_progress = milestone_progress(milestones, issues, now)
    warnings = risk_warnings(stats, ms_progress, contributors, commits=commits, now=now)
    tr = trend(stats["this_week"], stats["last_week"])

    return {
        "scenario": "S5_progress_tracking",
        "repo": f"{owner}/{repo}",
        "generated_at": now.astimezone(UTC).isoformat(),
        "week_stats": stats,
        "trend": tr,
        "milestones": ms_progress,
        "risk_warnings": warnings,
        "meta": {
            "commits_fetched": len(commits),
            "issues_fetched": len(issues),
            "prs_fetched": len(prs),
            "milestones_fetched": len(milestones),
            "contributors_fetched": len(contributors),
        },
    }


# ---------------------------------------------------------------------------
# 渲染：Markdown 周报
# ---------------------------------------------------------------------------

def render_report(result: dict[str, Any]) -> str:
    repo = result["repo"]
    stats = result["week_stats"]
    tw, lw = stats["this_week"], stats["last_week"]
    tr = result["trend"]
    lines = [
        f"# 科研进度智能跟踪周报 — {repo}\n",
        f"> 场景 S5 · 子赛题四「应用 GitLink 辅助科研」· 生成于 {result['generated_at']}\n",
        "## 一、本周 / 上周活动对比\n",
        "| 指标 | 本周 | 上周 |",
        "|------|------|------|",
        f"| 提交 commits | {tw['commits']} | {lw['commits']} |",
        f"| Issue 新增 | {tw['issues_opened']} | {lw['issues_opened']} |",
        f"| Issue 关闭 | {tw['issues_closed']} | {lw['issues_closed']} |",
        f"| 开放 stale issue (>{STALE_ISSUE_DAYS}天) | {tw['issues_stale']} | {lw['issues_stale']} |",
        f"| PR 新增 | {tw['prs_opened']} | {lw['prs_opened']} |",
        f"| PR 合并 | {tw['prs_merged']} | {lw['prs_merged']} |",
        f"| 开放 stale PR (>{STALE_PR_DAYS}天) | {tw['prs_open_stale']} | {lw['prs_open_stale']} |",
        f"| 活跃贡献者 | {tw['contributors_active']} | {lw['contributors_active']} |",
        "",
        f"- **趋势**：commit 周环比 **{tr['commit_delta_pct']}%**，活跃度等级 `{tr['activity_level']}`",
        "",
        "## 二、里程碑进度\n",
    ]
    ms = result["milestones"]
    if ms:
        lines += [
            "| 里程碑 | 完成/总数 | 完成率 | due_date | 状态 |",
            "|--------|-----------|--------|----------|------|",
        ]
        for m in ms:
            flag = " ⚠️逾期" if m["overdue"] else ""
            lines.append(
                f"| {m['name']}{flag} | {m['closed']}/{m['total']} | {m['completion_pct']}% "
                f"| {m['due_date'] or '—'} | {m['status'] or '—'} |"
            )
    else:
        lines.append("_（仓库无里程碑数据）_")

    lines += ["\n## 三、风险预警\n"]
    warns = result["risk_warnings"]
    if warns:
        lines += ["| 级别 | 类型 | 说明 | 建议 |", "|------|------|------|------|"]
        for w in warns:
            lines.append(f"| {w['level']} | {w['type']} | {w['message']} | {w['suggestion']} |")
    else:
        lines.append("_（未触发风险阈值，进度正常）_")

    lines += [
        "\n## 四、附\n",
        f"- 取数：commits={result['meta']['commits_fetched']} "
        f"issues={result['meta']['issues_fetched']} prs={result['meta']['prs_fetched']} "
        f"milestones={result['meta']['milestones_fetched']} "
        f"contributors={result['meta']['contributors_fetched']}",
        f"- 窗口：本周 [{stats['window']['this_week_start']}, {stats['window']['now']}]；"
        f"上周 [{stats['window']['last_week_start']}, {stats['window']['last_week_end']})",
        f"- 阈值：stale_issue>{STALE_ISSUE_DAYS}天 / stale_pr>{STALE_PR_DAYS}天 / "
        f"低活跃<{LOW_ACTIVITY_COMMITS}次/周 / bus_factor>{int(BUS_FACTOR_RATIO*100)}%",
        "",
    ]
    return "\n".join(lines)


# ---------------------------------------------------------------------------

def main():
    ap = argparse.ArgumentParser(description="S5 科研进度智能跟踪与预警")
    ap.add_argument("--owner", required=True)
    ap.add_argument("--repo", required=True)
    ap.add_argument("--out", help="输出目录（写 report.json + weekly_report.md）；省略则打印 JSON")
    args = ap.parse_args()

    result = analyze(args.owner, args.repo)

    if args.out:
        os.makedirs(args.out, exist_ok=True)
        with open(os.path.join(args.out, "report.json"), "w", encoding="utf-8") as f:
            json.dump(result, f, ensure_ascii=False, indent=2)
        with open(os.path.join(args.out, "weekly_report.md"), "w", encoding="utf-8") as f:
            f.write(render_report(result))
        print(f"✓ S5 周报完成 → {args.out}/report.json | weekly_report.md")
        print(f"  本周提交 {result['week_stats']['this_week']['commits']} "
              f"(趋势 {result['trend']['activity_level']})；风险 {len(result['risk_warnings'])} 条")
    else:
        print(json.dumps(result, ensure_ascii=False, indent=2))


if __name__ == "__main__":
    main()
