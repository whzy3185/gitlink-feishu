"""gitlink-changelog 单元测试。"""
from __future__ import annotations
import sys
from pathlib import Path
sys.path.insert(0, str(Path(__file__).resolve().parent.parent / "scripts"))
import pytest
from changelog import parse_commit, build_changelog, render_markdown


def commit(msg, login="dev", sha="abcdef123456"):
    return {"message": msg, "author": {"login": login}, "sha": sha}


class TestParseCommit:
    def test_plain(self):
        p = parse_commit(commit("feat: 新增登录"))
        assert p["type"] == "feat" and p["desc"] == "新增登录"

    def test_scope(self):
        p = parse_commit(commit("fix(auth): 修复 token"))
        assert p["type"] == "fix" and p["scope"] == "auth"

    def test_breaking(self):
        p = parse_commit(commit("feat!: 重构 API"))
        assert p["breaking"] is True

    def test_breaking_body(self):
        p = parse_commit({"message": "feat: x\n\nBREAKING CHANGE: 不兼容", "author": {}, "sha": "x"})
        assert p["breaking"] is True

    def test_other(self):
        p = parse_commit(commit("随便写的"))
        assert p["type"] == "other"

    def test_merge_not_typed(self):
        p = parse_commit(commit("Merge pull request '#1'"))
        assert p["type"] == "other"


class TestBuildChangelog:
    def test_groups(self):
        cl = build_changelog([commit("feat: a"), commit("fix: b"), commit("feat: c")])
        assert cl["groups"]["feat"] == 2
        assert cl["groups"]["fix"] == 1
        assert cl["typed_commits"] == 3

    def test_breaking_collected(self):
        cl = build_changelog([commit("feat!: x")])
        assert cl["breaking_count"] == 1

    def test_contributors(self):
        cl = build_changelog([commit("feat: a", login="alice"), commit("fix: b", login="bob")])
        assert set(cl["contributors"]) == {"alice", "bob"}

    def test_empty(self):
        cl = build_changelog([])
        assert cl["total_commits"] == 0 and cl["typed_commits"] == 0


class TestRender:
    def test_markdown(self):
        cl = build_changelog([commit("feat: 新功能"), commit("fix!: 重大修复")])
        md = render_markdown(cl, "o", "r")
        assert "版本变更对比" in md
        assert "新功能" in md
        assert "不兼容变更" in md


if __name__ == "__main__":
    sys.exit(pytest.main([__file__, "-v"]))
