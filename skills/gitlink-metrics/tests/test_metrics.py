"""gitlink-metrics 单元测试。"""
from __future__ import annotations
import sys
from pathlib import Path
sys.path.insert(0, str(Path(__file__).resolve().parent.parent / "scripts"))
import pytest
from metrics import gini, bus_factor, monthly_commits, pr_metrics, score_card, analyze, render_dashboard


class TestGini:
    def test_equal(self):
        assert gini([10, 10, 10]) == pytest.approx(0.0, abs=0.01)

    def test_concentrated(self):
        assert gini([100, 1, 1]) > 0.5

    def test_empty(self):
        assert gini([]) == 0.0


class TestBusFactor:
    def test_single_dominant(self):
        assert bus_factor([60, 20, 20]) == 1

    def test_even(self):
        assert bus_factor([25, 25, 25, 25]) == 2

    def test_empty(self):
        assert bus_factor([]) == 0


class TestMonthlyCommits:
    def test_groups_by_month(self):
        import datetime
        ts1 = int(datetime.datetime(2026, 1, 15, tzinfo=datetime.timezone.utc).timestamp())
        ts2 = int(datetime.datetime(2026, 1, 20, tzinfo=datetime.timezone.utc).timestamp())
        ts3 = int(datetime.datetime(2026, 2, 1, tzinfo=datetime.timezone.utc).timestamp())
        m = monthly_commits([{"timestamp": ts1}, {"timestamp": ts2}, {"timestamp": ts3}])
        assert m["2026-01"] == 2 and m["2026-02"] == 1

    def test_invalid_ts(self):
        assert monthly_commits([{"timestamp": None}]) == {}


class TestPRMetrics:
    def test_rates(self):
        pulls = [{"pull_request_status": 1}, {"pull_request_status": 1},
                 {"pull_request_status": 0}, {"pull_request_status": 2}]
        r = pr_metrics(pulls)
        assert r["merged"] == 2 and r["open"] == 1 and r["closed"] == 1
        assert r["merge_rate"] == 50.0

    def test_empty(self):
        assert pr_metrics([])["merge_rate"] == 0.0


class TestScoreCard:
    def test_all_dims(self):
        info = {"praises_count": 5, "forked_count": 10}
        prm = {"merge_rate": 80}
        card = score_card(info, prm, [50, 30, 20], {"2026-05": 40, "2026-06": 30})
        for k in ("活跃度", "协作", "社区", "可维护性", "综合"):
            assert 0 <= card[k] <= 100
        assert card["协作"] == 80


class TestAnalyze:
    class FakeClient:
        def repo_info(self, o, r):
            return {"issues_count": 10, "pull_requests_count": 5, "version_releases_count": 2,
                    "praises_count": 3, "forked_count": 4}
        def pulls(self, o, r, limit=50):
            return [{"pull_request_status": 1}, {"pull_request_status": 0}]
        def commits(self, o, r, max_pages=6):
            import datetime
            ts = int(datetime.datetime(2026, 5, 1, tzinfo=datetime.timezone.utc).timestamp())
            return [{"timestamp": ts}]
        def contributors(self, o, r):
            return [{"login": "a", "contributions": 80}, {"login": "b", "contributions": 20}]

    def test_full(self):
        m = analyze("o", "r", client=self.FakeClient())
        assert m["scale"]["contributors"] == 2
        assert m["pr_metrics"]["merge_rate"] == 50.0
        assert "综合" in m["score_card"]

    def test_render(self):
        m = analyze("o", "r", client=self.FakeClient())
        md = render_dashboard(m, "o", "r")
        assert "仓库指标看板" in md
        assert "多维评分卡" in md
        assert "巴士因子" in md


if __name__ == "__main__":
    sys.exit(pytest.main([__file__, "-v"]))
