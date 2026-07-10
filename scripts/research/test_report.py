"""test_report.py — S5 进度跟踪与预警的纯单元测试（不联网）。

运行：`pytest scripts/research/` 或 `python scripts/research/test_report.py`
"""
import os
import sys
from datetime import datetime, timedelta, timezone

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))

import report as R  # noqa: E402

UTC = timezone.utc
NOW = datetime(2026, 6, 29, 12, 0, 0, tzinfo=UTC)


def iso(dt):
    return dt.isoformat()


def test_parse_time_iso():
    assert R.parse_time("2026-06-01T08:30:00+00:00").year == 2026
    assert R.parse_time("2026-06-01T08:30:00Z").month == 6


def test_parse_time_unix():
    # 整数与整数字串都按秒级时间戳解析
    dt = R.parse_time(1717200000)
    assert dt is not None and dt.year == 2024
    assert R.parse_time("1717200000").year == 2024


def test_parse_time_none_and_garbage():
    assert R.parse_time(None) is None
    assert R.parse_time("") is None
    assert R.parse_time("not a date") is None


def test_in_window():
    lo = NOW - timedelta(days=7)
    assert R.in_window(NOW - timedelta(days=2), 7, NOW) is True
    assert R.in_window(NOW - timedelta(days=10), 7, NOW) is False
    assert R.in_window(None, 7, NOW) is False


def test_week_stats_splits_this_and_last_week():
    commits = [
        {"timestamp": iso(NOW - timedelta(days=2)), "author": {"login": "a"}},   # 本周
        {"timestamp": iso(NOW - timedelta(days=3)), "author": {"login": "b"}},   # 本周
        {"timestamp": iso(NOW - timedelta(days=10)), "author": {"login": "a"}},  # 上周
    ]
    issues = [
        {"created_at": iso(NOW - timedelta(days=1)), "status": "open"},          # 本周新增
        {"created_at": iso(NOW - timedelta(days=9)), "status": "open"},          # 上周新增
    ]
    prs = []
    stats = R.week_stats(commits, issues, prs, ["a", "b"], NOW)
    assert stats["this_week"]["commits"] == 2
    assert stats["last_week"]["commits"] == 1
    assert stats["this_week"]["issues_opened"] == 1
    assert stats["last_week"]["issues_opened"] == 1


def test_week_stats_stale_issue():
    # 开放、最近活动 > 30 天 → stale
    old = NOW - timedelta(days=40)
    issues = [{"status": "open", "created_at": iso(old), "journals_updated_at": iso(old)}]
    stats = R.week_stats([], issues, [], [], NOW)
    assert stats["this_week"]["issues_stale"] == 1


def test_trend():
    assert R.trend({"commits": 10}, {"commits": 5})["activity_level"] == "increasing"
    assert R.trend({"commits": 2}, {"commits": 10})["activity_level"] == "decreasing"
    assert R.trend({"commits": 10}, {"commits": 10})["activity_level"] == "stable"
    # 上周为 0、本周有提交 → 100%
    assert R.trend({"commits": 3}, {"commits": 0})["commit_delta_pct"] == 100.0


def test_risk_low_activity():
    stats = {"this_week": {"commits": 1, "issues_stale": 0, "prs_open_stale": 0}}
    warns = R.risk_warnings(stats, [], [], commits=None, now=NOW)
    assert any(w["type"] == "low_activity" for w in warns)


def test_risk_bus_factor():
    # 一人占本周全部提交 → bus factor
    commits = [{"timestamp": iso(NOW - timedelta(days=1)), "author": {"login": "only"}} for _ in range(5)]
    stats = {"this_week": {"commits": 5, "issues_stale": 0, "prs_open_stale": 0}}
    warns = R.risk_warnings(stats, [], [], commits=commits, now=NOW)
    assert any(w["type"] == "bus_factor" for w in warns)


def test_milestone_progress_overdue():
    ms = [{"name": "v1.0", "status": "open", "effective_date": iso(NOW - timedelta(days=5))}]
    issues = [
        {"milestone_name": "v1.0", "status": "closed"},
        {"milestone_name": "v1.0", "status": "open"},
        {"milestone_name": "v1.0", "status": "open"},
    ]
    out = R.milestone_progress(ms, issues, NOW)
    assert len(out) == 1
    assert out[0]["total"] == 3 and out[0]["closed"] == 1
    assert out[0]["completion_pct"] == round(100 / 3, 1)
    assert out[0]["overdue"] is True


def test_render_report_contains_sections():
    # 直接构造一个最小 result 喂渲染器（不触网）
    res = {
        "repo": "o/r", "generated_at": iso(NOW),
        "week_stats": {"this_week": {"commits": 1, "issues_opened": 0, "issues_closed": 0,
                                     "issues_stale": 0, "prs_opened": 0, "prs_merged": 0,
                                     "prs_open_stale": 0, "contributors_active": 1},
                       "last_week": {"commits": 0, "issues_opened": 0, "issues_closed": 0,
                                     "issues_stale": 0, "prs_opened": 0, "prs_merged": 0,
                                     "prs_open_stale": 0, "contributors_active": 0},
                       "window": {"this_week_start": iso(NOW), "now": iso(NOW),
                                  "last_week_start": iso(NOW), "last_week_end": iso(NOW)},
                       "total_contributors": 1},
        "trend": {"commit_delta_pct": 100.0, "activity_level": "increasing"},
        "milestones": [], "risk_warnings": [],
        "meta": {"commits_fetched": 1, "issues_fetched": 0, "prs_fetched": 0,
                 "milestones_fetched": 0, "contributors_fetched": 1},
    }
    md = R.render_report(res)
    assert "周报" in md and "趋势" in md


def test_pr_status_string_and_int():
    # GitLink PR 列表 status 是字符串 'merged'/'open'/'closed'
    assert R._pr_status({"status": "merged"}) == 1
    assert R._pr_status({"status": "open"}) == 0
    assert R._pr_status({"status": "closed"}) == 2
    # 详情/health 可能给 pull_request_status 整数 0/1/2
    assert R._pr_status({"pull_request_status": 1}) == 1
    assert R._pr_status({"pull_request_status": 0}) == 0
    assert R._pr_status({"pull_request_status": 2}) == 2


def _run_all():
    fns = [v for k, v in sorted(globals().items()) if k.startswith("test_")]
    for fn in fns:
        fn()
        print(f"PASS {fn.__name__}")
    print(f"\nAll {len(fns)} report tests passed.")


if __name__ == "__main__":
    _run_all()
