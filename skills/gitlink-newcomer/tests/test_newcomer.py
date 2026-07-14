"""gitlink-newcomer 单元测试。

覆盖 good-first-issue 识别、友好度评分、ID 区分、引导生成与看板渲染。
使用合成数据，不触网，可离线运行：

    python -m pytest tests/ -v
"""

from __future__ import annotations

import sys
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parent.parent / "scripts"))

import pytest

from newcomer import (
    score_issue, build_guidance, render_board,
    _issue_number, _issue_gid, _labels_of,
)


def issue(name="", body="", labels=None, comments=0, number=None, gid=None):
    d = {"name": name, "description": body, "comment_journals_count": comments}
    if labels is not None:
        d["issue_tags"] = [{"name": x} for x in labels]
    if number is not None:
        d["number"] = number
    if gid is not None:
        d["id"] = gid
    return d


class TestLabels:
    def test_extract_dict_labels(self):
        assert _labels_of(issue(labels=["Bug", "good first issue"])) == ["bug", "good first issue"]

    def test_no_labels(self):
        assert _labels_of(issue()) == []


class TestIdDistinction:
    def test_web_number_preferred(self):
        assert _issue_number(issue(number=12)) == 12

    def test_no_web_number_returns_none(self):
        # 列表接口只有全局 id，不应被当作 web 序号
        assert _issue_number(issue(gid=140801)) is None

    def test_gid_extracted(self):
        assert _issue_gid(issue(gid=140801)) == 140801


class TestScoreIssue:
    def test_good_first_label_boosts(self):
        s = score_issue(issue(name="fix typo", labels=["good first issue"]))
        assert s["friendliness"] >= 75
        assert s["is_good_first"] is True
        assert "新手友好标签" in str(s["signals"])

    def test_hard_keyword_lowers(self):
        s = score_issue(issue(name="refactor concurrency architecture",
                              body="needs deep refactor of the core"))
        assert s["friendliness"] < 55
        assert s["is_good_first"] is False

    def test_easy_keyword(self):
        s = score_issue(issue(name="update docs and fix typo in readme"))
        assert s["friendliness"] > 50
        assert any("易上手" in sig for sig in s["signals"])

    def test_hard_label(self):
        s = score_issue(issue(name="something", labels=["hard"]))
        assert "高难度标签" in str(s["signals"])

    def test_difficulty_levels(self):
        easy = score_issue(issue(name="docs typo", labels=["good first issue"]))
        hard = score_issue(issue(name="refactor architecture performance concurrency"))
        assert easy["difficulty"] in ("入门", "较易")
        assert hard["difficulty"] in ("中等", "进阶")

    def test_score_bounded(self):
        s = score_issue(issue(name="good first " * 10, labels=["good first issue", "beginner"]))
        assert 0 <= s["friendliness"] <= 100

    def test_long_body_penalty(self):
        short = score_issue(issue(name="task", body="x" * 100))
        long = score_issue(issue(name="task", body="x" * 2000))
        assert long["friendliness"] <= short["friendliness"]


class TestGuidance:
    def test_uses_number_when_available(self):
        g = build_guidance(score_issue(issue(name="fix", number=12)), "o", "r")
        assert "#12" in g

    def test_uses_title_when_no_number(self):
        s = score_issue(issue(name="修复文档错别字", gid=999))
        g = build_guidance(s, "o", "r")
        # 无 web 序号时用标题引用，不出现 #999
        assert "#999" not in g
        assert "修复文档错别字" in g

    def test_contains_fork_flow(self):
        g = build_guidance(score_issue(issue(name="task", number=1)), "Gitlink", "gitlink-cli")
        assert "repo +fork" in g
        assert "Gitlink" in g


class TestBoard:
    def test_empty_candidates(self):
        result = {"owner": "o", "repo": "r", "total_issues": 3,
                  "candidate_count": 0, "candidates": [], "all_scored": []}
        board = render_board(result, "o", "r")
        assert "暂未发现" in board

    def test_board_with_candidates(self):
        cand = score_issue(issue(name="fix typo in docs", labels=["good first issue"], gid=100))
        result = {"owner": "o", "repo": "r", "total_issues": 5,
                  "candidate_count": 1, "candidates": [cand], "all_scored": [cand]}
        board = render_board(result, "o", "r")
        assert "新手任务看板" in board
        assert "id:100" in board  # 无 web 序号时标注全局 id


if __name__ == "__main__":
    sys.exit(pytest.main([__file__, "-v"]))
