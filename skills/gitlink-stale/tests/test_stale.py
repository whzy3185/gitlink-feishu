"""gitlink-stale 单元测试。"""
from __future__ import annotations
import sys
from pathlib import Path
sys.path.insert(0, str(Path(__file__).resolve().parent.parent / "scripts"))
import pytest
from stale import parse_relative_days, grade, analyze_stale, render_report


class TestParseRelativeDays:
    def test_days(self):
        assert parse_relative_days("3天前") == 3

    def test_weeks(self):
        assert parse_relative_days("2周前") == 14

    def test_months(self):
        assert parse_relative_days("2个月前") == 60

    def test_years(self):
        assert parse_relative_days("1年前") == 365

    def test_recent(self):
        assert parse_relative_days("刚刚") == 0
        assert parse_relative_days("3小时前") == 0

    def test_unknown(self):
        assert parse_relative_days("") is None
        assert parse_relative_days("奇怪的值") is None


class TestGrade:
    def test_levels(self):
        assert grade(5) == "活跃"
        assert grade(30) == "留意"
        assert grade(90) == "陈旧"
        assert grade(200) == "僵尸"
        assert grade(None) == "未知"


class TestAnalyzeStale:
    def test_filters_closed(self):
        issues = [
            {"id": "1", "name": "open old", "updated_at": "200天前", "issue_status": "正在解决"},
            {"id": "2", "name": "closed", "updated_at": "300天前", "issue_status": "关闭"},
        ]
        r = analyze_stale(issues, [])
        # 关闭的不计入
        assert r["total_open"] == 1
        assert r["by_grade"]["僵尸"] == 1

    def test_pr_open_only(self):
        pulls = [
            {"pull_request_number": "1", "name": "open pr", "pr_time": "100天前", "pull_request_status": 0},
            {"pull_request_number": "2", "name": "merged", "pr_time": "100天前", "pull_request_status": 1},
        ]
        r = analyze_stale([], pulls)
        assert r["total_open"] == 1

    def test_cleanup_list(self):
        issues = [{"id": "1", "name": "zombie", "updated_at": "200天前", "issue_status": "open"}]
        r = analyze_stale(issues, [])
        assert r["cleanup_count"] == 1
        assert r["cleanup"][0]["grade"] == "僵尸"

    def test_empty(self):
        r = analyze_stale([], [])
        assert r["total_open"] == 0 and r["cleanup_count"] == 0


class TestRender:
    def test_report(self):
        issues = [{"id": "1", "name": "old issue", "updated_at": "150天前", "issue_status": "open"}]
        r = analyze_stale(issues, [])
        md = render_report(r, "o", "r")
        assert "陈旧 Issue/PR 清理报告" in md
        assert "建议处理" in md

    def test_clean_repo(self):
        r = analyze_stale([{"id": "1", "name": "x", "updated_at": "1天前", "issue_status": "open"}], [])
        md = render_report(r, "o", "r")
        assert "积压控制良好" in md


if __name__ == "__main__":
    sys.exit(pytest.main([__file__, "-v"]))
