"""gitlink-flow 单元测试。

覆盖三个子工作流（triage/pr-review/release-notes）与复用 Skill 步骤，
以及编排器 run_flow。使用合成数据 + FakeClient，不触网。

    python -m pytest tests/ -q
"""

from __future__ import annotations

import sys
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parent.parent / "src"))

import pytest

import steps
import report as report_mod
import cli
from flow import run_flow


# ---------------------------------------------------------------------------
# 子工作流 1：triage
# ---------------------------------------------------------------------------

class TestTriage:
    def test_classifies_bug(self):
        r = steps.triage_issues([{"name": "登录时 crash 崩溃", "description": "复现步骤"}])
        assert r["by_category"]["bug"] == 1

    def test_classifies_feature(self):
        r = steps.triage_issues([{"name": "建议新增搜索功能 feature", "description": ""}])
        assert r["by_category"]["feature"] == 1

    def test_good_first_detected(self):
        r = steps.triage_issues([{"name": "fix typo in docs", "description": "small"}])
        assert r["good_first_count"] == 1
        assert r["items"][0]["suggested_label"] == "good first issue"

    def test_empty(self):
        r = steps.triage_issues([])
        assert r["total"] == 0
        assert r["good_first_count"] == 0


# ---------------------------------------------------------------------------
# 子工作流 2：pr-review
# ---------------------------------------------------------------------------

class TestPRReview:
    def test_status_counts(self):
        pulls = [
            {"name": "a", "pull_request_status": 0, "author_login": "u1"},
            {"name": "b", "pull_request_status": 1, "author_login": "u2"},
            {"name": "c", "pull_request_status": 2, "author_login": "u3"},
        ]
        r = steps.pr_review_summary(pulls)
        assert r["open"] == 1 and r["merged"] == 1 and r["closed"] == 1
        assert r["merge_rate"] == round(1 / 3 * 100, 1)

    def test_pending_review_listed(self):
        pulls = [{"name": "open pr", "pull_request_status": 0,
                  "author_login": "dev", "fork_project_user": "dev"}]
        r = steps.pr_review_summary(pulls)
        assert len(r["pending_review"]) == 1
        assert r["pending_review"][0]["is_fork"] is True

    def test_empty(self):
        r = steps.pr_review_summary([])
        assert r["total"] == 0 and r["merge_rate"] == 0.0


# ---------------------------------------------------------------------------
# 子工作流 3：release-notes
# ---------------------------------------------------------------------------

class TestReleaseNotes:
    def test_groups_by_type(self):
        commits = [
            {"message": "feat: 新增登录"},
            {"message": "fix: 修复崩溃"},
            {"message": "feat(api): 新增接口"},
            {"message": "随便写的提交"},  # other，不计入
        ]
        r = steps.release_notes(commits, version="v1.0")
        assert r["groups"]["feat"] == 2
        assert r["groups"]["fix"] == 1
        assert r["typed_commits"] == 3
        assert "## v1.0" in r["markdown"]
        assert "新增登录" in r["markdown"]

    def test_strips_prefix(self):
        r = steps.release_notes([{"message": "fix(core): 修复空指针"}])
        assert "修复空指针" in r["markdown"]
        assert "fix(core):" not in r["markdown"]

    def test_empty(self):
        r = steps.release_notes([])
        assert r["typed_commits"] == 0


# ---------------------------------------------------------------------------
# 复用 Skill 步骤
# ---------------------------------------------------------------------------

class TestHealthAndContributors:
    def test_health_full(self):
        files = ["readme.md", "license", "contributing.md", "code_of_conduct.md"]
        r = steps.health_check(files)
        assert r["score"] == 100
        assert r["missing"] == []

    def test_health_partial(self):
        r = steps.health_check(["readme.md", "license"])
        assert r["score"] == 50
        assert "CONTRIBUTING" in r["missing"]

    def test_contributors_sorted(self):
        contribs = [
            {"login": "a", "contributions": 10},
            {"login": "b", "contributions": 50},
        ]
        r = steps.contributor_highlights(contribs)
        assert r["top"][0]["name"] == "b"
        assert r["total_contributions"] == 60


# ---------------------------------------------------------------------------
# 编排器 + 周报
# ---------------------------------------------------------------------------

class FakeClient:
    def repo_info(self, o, r):
        return {"name": r, "issues_count": 2, "pull_requests_count": 1,
                "praises_count": 3, "forked_count": 4}

    def issues(self, o, r, limit=50):
        return [{"id": "1", "name": "fix typo in docs", "description": "small"},
                {"id": "2", "name": "登录 crash", "description": "bug"}]

    def pulls(self, o, r, limit=50):
        return [{"name": "feat: x", "pull_request_status": 0, "author_login": "dev"}]

    def commits(self, o, r, max_pages=4):
        return [{"message": "feat: 新功能"}, {"message": "fix: 修复"}]

    def contributors(self, o, r):
        return [{"login": "alice", "contributions": 100}]

    def releases(self, o, r):
        return [{"tag_name": "v1.0", "name": "v1.0"}]

    def list_dir(self, o, r, path, ref):
        return [{"name": "README.md"}, {"name": "LICENSE"}]


class TestOrchestrator:
    def test_run_flow_all_steps(self):
        result = run_flow("o", "repo", client=FakeClient(), use_cli=False)
        assert "step1_triage" in result
        assert "step2_pr_review" in result
        assert "step3_release_notes" in result
        assert "step4_health" in result
        assert "step5_contributors" in result
        assert "step6_weekly_report" in result

    def test_weekly_report_renders(self):
        result = run_flow("o", "repo", client=FakeClient(), use_cli=False)
        md = result["step6_weekly_report"]
        assert "社区运营周报" in md
        assert "Issue 自动分拣" in md
        assert "PR Review 汇总" in md
        assert "Release Notes" in md

    def test_triage_in_flow(self):
        result = run_flow("o", "repo", client=FakeClient(), use_cli=False)
        # 一个 good-first（docs typo）+ 一个 bug
        assert result["step1_triage"]["good_first_count"] == 1

    def test_fallback_data_source(self):
        # 强制走 glapi 直连时，数据源应标注已回退
        result = run_flow("o", "repo", client=FakeClient(), use_cli=False)
        assert "回退" in result["data_source"]


class TestCliLayer:
    """gitlink-cli 封装层：解包信封与列表提取，不实际调用命令。"""

    def test_extract_list_from_dict(self):
        assert cli._extract_list({"issues": [1, 2]}, ("issues",)) == [1, 2]

    def test_extract_list_passthrough(self):
        assert cli._extract_list([1, 2], ("issues",)) == [1, 2]

    def test_extract_list_empty(self):
        assert cli._extract_list({"other": 1}, ("issues",)) == []

    def test_cli_available_returns_bool(self):
        assert isinstance(cli.cli_available(), bool)


if __name__ == "__main__":
    sys.exit(pytest.main([__file__, "-v"]))
