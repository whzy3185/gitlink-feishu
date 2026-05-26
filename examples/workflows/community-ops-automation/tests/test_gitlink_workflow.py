from __future__ import annotations

import unittest
from datetime import datetime, timezone

from scripts.gitlink_workflow import (
    build_issue_comment_command,
    normalize_issues,
    normalize_prs,
    normalize_releases,
    render_markdown_report,
    render_release_notes,
    summarize_workflow,
)


class WorkflowTests(unittest.TestCase):
    def setUp(self) -> None:
        self.now = datetime(2026, 5, 15, 12, 0, tzinfo=timezone.utc)
        self.repo_info = {
            "name": "forgeplus",
            "description": "demo repo",
            "default_branch": "master",
        }

    def test_normalize_issue_payload(self) -> None:
        payload = {
            "data": {
                "issues": [
                    {
                        "project_issues_index": 1,
                        "subject": "feat: add report",
                        "status_id": 1,
                        "status_name": "新增",
                        "updated_at": "2026-05-10T10:00:00Z",
                        "labels": [{"name": "enhancement"}],
                    }
                ]
            }
        }
        issues = normalize_issues(payload)
        self.assertEqual(len(issues), 1)
        self.assertEqual(issues[0]["title"], "feat: add report")
        self.assertEqual(issues[0]["labels"], ["enhancement"])
        self.assertEqual(issues[0]["state"], "open")

    def test_normalize_pr_payload(self) -> None:
        payload = {
            "data": {
                "merge_requests": [
                    {
                        "pull_request_number": 10,
                        "title": "fix: bug",
                        "pull_request_status": 1,
                        "merged_at": "2026-05-14T10:00:00Z",
                    }
                ]
            }
        }
        prs = normalize_prs(payload)
        self.assertEqual(len(prs), 1)
        self.assertTrue(prs[0]["merged"])
        self.assertEqual(prs[0]["state"], "merged")

    def test_normalize_release_payload(self) -> None:
        payload = {"data": {"releases": [{"id": 5, "name": "v1.0.0"}]}}
        releases = normalize_releases(payload)
        self.assertEqual(len(releases), 1)
        self.assertEqual(releases[0]["title"], "v1.0.0")

    def test_summary_and_report(self) -> None:
        issues = [
            {
                "id": "1",
                "title": "feat: add report",
                "state": "open",
                "created_at": datetime(2026, 5, 5, 12, 0, tzinfo=timezone.utc),
                "updated_at": datetime(2026, 5, 10, 12, 0, tzinfo=timezone.utc),
                "labels": ["enhancement"],
            },
            {
                "id": "2",
                "title": "fix: stale issue",
                "state": "open",
                "created_at": datetime(2026, 4, 20, 12, 0, tzinfo=timezone.utc),
                "updated_at": datetime(2026, 5, 1, 12, 0, tzinfo=timezone.utc),
                "labels": ["bug"],
            },
        ]
        prs = [
            {
                "id": "10",
                "title": "feat: workflow",
                "state": "merged",
                "created_at": datetime(2026, 5, 12, 12, 0, tzinfo=timezone.utc),
                "updated_at": datetime(2026, 5, 14, 12, 0, tzinfo=timezone.utc),
                "merged_at": datetime(2026, 5, 14, 12, 0, tzinfo=timezone.utc),
                "merged": True,
                "labels": [],
            },
            {
                "id": "11",
                "title": "chore: cleanup",
                "state": "open",
                "created_at": datetime(2026, 5, 1, 12, 0, tzinfo=timezone.utc),
                "updated_at": datetime(2026, 5, 2, 12, 0, tzinfo=timezone.utc),
                "merged_at": None,
                "merged": False,
                "labels": [],
            },
        ]
        releases = [{"id": "1", "title": "v1.0.0", "created_at": datetime(2026, 5, 14, 12, 0, tzinfo=timezone.utc)}]
        summary = summarize_workflow(self.repo_info, issues, prs, releases, self.now, 7)
        report = render_markdown_report(summary)
        self.assertIn("# forgeplus 自动化周报", report)
        self.assertIn("Issues 总数", report)
        self.assertIn("超窗 Issue", report)
        self.assertIn("feature", report)
        release_notes = render_release_notes(summary)
        self.assertIn("Release Notes", release_notes)
        self.assertIn("变更分类", release_notes)
        self.assertEqual(summary["counts"]["issues_stale"], 1)
        self.assertEqual(summary["counts"]["prs_merged"], 1)
        self.assertIn("feature", summary["pr_buckets"])

    def test_issue_comment_command_uses_number_flag(self) -> None:
        command = build_issue_comment_command(2, "demo")
        self.assertEqual(command, ["issue", "+comment", "--number", "2", "--body", "demo"])
        self.assertNotIn("-i", command)


if __name__ == "__main__":
    unittest.main()
