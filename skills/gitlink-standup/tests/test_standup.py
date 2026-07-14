"""gitlink-standup 单元测试。"""
from __future__ import annotations
import sys
from pathlib import Path
sys.path.insert(0, str(Path(__file__).resolve().parent.parent / "scripts"))
import pytest
from standup import summarize_user, render_report, analyze


def trend(tt, name="something"):
    return {"trend_type": tt, "action_type": "x", "action_time": "1天前", "name": name}


class TestSummarize:
    def test_counts_by_type(self):
        s = summarize_user("u", [trend("CommitLog"), trend("CommitLog"), trend("PullRequest")])
        assert s["total_activities"] == 3
        assert s["by_type"]["CommitLog"] == 2
        assert s["by_type"]["PullRequest"] == 1

    def test_samples_capped(self):
        s = summarize_user("u", [trend("CommitLog", f"c{i}") for i in range(10)])
        assert len(s["samples"]["CommitLog"]) == 5  # 每类最多 5 条样例

    def test_empty(self):
        s = summarize_user("u", [])
        assert s["total_activities"] == 0
        assert s["by_type"] == {}

    def test_multiline_name_first_line(self):
        s = summarize_user("u", [trend("Issue", "标题\n正文第二行")])
        assert s["samples"]["Issue"][0] == "标题"


class TestRender:
    def test_report(self):
        summaries = [summarize_user("alice", [trend("CommitLog")])]
        md = render_report(summaries, period="周")
        assert "团队活动周报" in md
        assert "@alice" in md
        assert "代码提交" in md

    def test_empty_member(self):
        md = render_report([summarize_user("bob", [])])
        assert "暂无公开活动" in md


class TestAnalyze:
    class FakeClient:
        def user_trends(self, login, limit=50):
            return [trend("CommitLog"), trend("Issue")] if login == "alice" else []

    def test_multi_user(self):
        r = analyze(["alice", "bob"], client=self.FakeClient())
        assert len(r) == 2
        assert r[0]["total_activities"] == 2
        assert r[1]["total_activities"] == 0


if __name__ == "__main__":
    sys.exit(pytest.main([__file__, "-v"]))
