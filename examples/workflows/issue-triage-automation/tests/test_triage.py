import sys
import unittest
from datetime import datetime
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parent.parent / "scripts"))

from issue_triage import is_stale, match_rule, render_report, triage_issue

NOW = datetime(2026, 7, 10, 12, 0)
RULES = [
    {"name": "bug", "keywords_any": ["bug", "崩溃"], "set_priority": "紧急", "add_tags": ["bug"], "route_to": "alice"},
    {"name": "docs", "keywords_any": ["文档"], "set_priority": "低", "add_tags": ["documentation"], "route_to": None},
]


class TestMatchRule(unittest.TestCase):
    def test_keyword_in_subject_case_insensitive(self):
        self.assertTrue(match_rule({"subject": "发现一个 BUG"}, RULES[0]))

    def test_keyword_in_description(self):
        self.assertTrue(match_rule({"subject": "问题", "description": "程序崩溃了"}, RULES[0]))

    def test_no_match(self):
        self.assertFalse(match_rule({"subject": "hello", "description": "world"}, RULES[0]))


class TestStale(unittest.TestCase):
    def test_old_issue_is_stale(self):
        self.assertTrue(is_stale({"updated_at": "2026-05-01 10:00"}, NOW, 30))

    def test_fresh_issue_not_stale(self):
        self.assertFalse(is_stale({"updated_at": "2026-07-01 10:00"}, NOW, 30))

    def test_unparseable_date_not_stale(self):
        self.assertFalse(is_stale({"updated_at": "n/a"}, NOW, 30))


class TestTriageIssue(unittest.TestCase):
    def test_first_matching_rule_wins(self):
        issue = {"project_issues_index": 7, "subject": "文档里的 bug", "updated_at": "2026-07-09 10:00"}
        d = triage_issue(issue, RULES, NOW, 30)
        self.assertEqual(d["rule"], "bug")
        self.assertEqual(d["priority"], "紧急")
        self.assertEqual(d["tags"], ["bug"])
        self.assertEqual(d["assignee"], "alice")
        self.assertFalse(d["stale"])

    def test_no_rule_matched(self):
        d = triage_issue({"project_issues_index": 8, "subject": "hello"}, RULES, NOW, 30)
        self.assertIsNone(d["rule"])
        self.assertEqual(d["tags"], [])

    def test_deterministic(self):
        issue = {"project_issues_index": 9, "subject": "崩溃", "updated_at": "2026-01-01 00:00"}
        self.assertEqual(
            triage_issue(issue, RULES, NOW, 30),
            triage_issue(issue, RULES, NOW, 30),
        )


class TestReport(unittest.TestCase):
    def test_report_contains_counts_and_rows(self):
        decisions = [
            {"number": 2, "subject": "b", "rule": "bug", "priority": "紧急", "tags": ["bug"], "assignee": None, "stale": True},
            {"number": 1, "subject": "a", "rule": None, "priority": None, "tags": [], "assignee": None, "stale": False},
        ]
        md = render_report(decisions, "o", "r")
        self.assertIn("共 2 个 open issue，命中规则 1 个，stale 1 个。", md)
        self.assertLess(md.index("| 1 |"), md.index("| 2 |"))  # 按编号排序


if __name__ == "__main__":
    unittest.main()
