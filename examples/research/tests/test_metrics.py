import sys
import unittest
from datetime import datetime
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parent.parent / "scripts"))

from research_tool_template import compute_metrics, conclude

NOW = datetime(2026, 7, 10)


def raw(issues=None, open_count=0, closed_count=0, prs=None, branches=None):
    return {
        "open_issues": {"issues": issues or [], "open_count": open_count, "closed_count": closed_count},
        "prs": prs or {},
        "branches": branches if branches is not None else [],
    }


class TestComputeMetrics(unittest.TestCase):
    def test_close_rate(self):
        m = compute_metrics(raw(open_count=2, closed_count=8), NOW, 30)
        self.assertEqual(m["close_rate"], "80%")

    def test_close_rate_empty(self):
        m = compute_metrics(raw(), NOW, 30)
        self.assertEqual(m["close_rate"], "n/a")

    def test_recently_active(self):
        m = compute_metrics(raw(issues=[{"updated_at": "2026-07-01 10:00"}], open_count=1), NOW, 30)
        self.assertEqual(m["recently_active"], "是")

    def test_not_recently_active(self):
        m = compute_metrics(raw(issues=[{"updated_at": "2026-01-01 10:00"}], open_count=1), NOW, 30)
        self.assertEqual(m["recently_active"], "否")

    def test_bad_date_ignored(self):
        m = compute_metrics(raw(issues=[{"updated_at": "n/a"}], open_count=1), NOW, 30)
        self.assertEqual(m["recently_active"], "否")

    def test_branches_wrapped_shape(self):
        m = compute_metrics(raw(branches={"branches": [1, 2, 3]}), NOW, 30)
        self.assertEqual(m["branches"], 3)

    def test_deterministic(self):
        r = raw(issues=[{"updated_at": "2026-07-01 10:00"}], open_count=3, closed_count=7)
        self.assertEqual(compute_metrics(r, NOW, 30), compute_metrics(r, NOW, 30))


class TestConclude(unittest.TestCase):
    def test_active(self):
        self.assertIn("活跃维护", conclude({"recently_active": "是"}, 30))

    def test_inactive(self):
        self.assertIn("建议先与维护者确认", conclude({"recently_active": "否"}, 30))


if __name__ == "__main__":
    unittest.main()
