"""gitlink-onboard 单元测试。"""
from __future__ import annotations
import sys
from pathlib import Path
sys.path.insert(0, str(Path(__file__).resolve().parent.parent / "scripts"))
import pytest
from onboard import detect_stack, navigate_dirs, pick_good_first, check_health, build_guide, render_guide


class TestDetectStack:
    def test_go(self):
        assert "Go" in detect_stack(["go.mod", "main.go"])

    def test_multi(self):
        s = detect_stack(["package.json", "Dockerfile"])
        assert "Node.js / JavaScript" in s and "Docker" in s

    def test_none(self):
        assert detect_stack(["README.md"]) == []


class TestNavigate:
    def test_dirs_with_hints(self):
        entries = [{"name": "cmd", "type": "dir"}, {"name": "weird", "type": "dir"},
                   {"name": "file.go", "type": "file"}]
        nav = navigate_dirs(entries)
        names = [n["name"] for n in nav]
        assert "cmd" in names and "weird" in names
        assert "file.go" not in names  # 文件不计入
        # 有提示的排前
        assert nav[0]["hint"] != ""


class TestGoodFirst:
    def test_picks_docs(self):
        issues = [{"id": "1", "name": "fix typo in docs", "description": "small", "issue_status": "open"}]
        r = pick_good_first(issues)
        assert len(r) == 1

    def test_skips_closed(self):
        issues = [{"id": "1", "name": "fix typo", "description": "x", "issue_status": "关闭"}]
        assert pick_good_first(issues) == []


class TestHealth:
    def test_detects(self):
        h = check_health(["readme.md", "contributing.md"])
        assert h["README"] is True
        assert h["CONTRIBUTING"] is True
        assert h["行为准则"] is False


class TestBuildAndRender:
    def _data(self):
        info = {"description": "测试项目", "default_branch": "master", "praises_count": 5, "forked_count": 2}
        entries = [{"name": "cmd", "type": "dir"}, {"name": "src", "type": "dir"}]
        root_files = ["go.mod", "readme.md"]
        issues = [{"id": "1", "name": "docs typo", "description": "x", "issue_status": "open"}]
        contributors = [{"login": "alice", "contributions": 100}]
        return build_guide("o", "r", info, root_files, entries, issues, contributors)

    def test_build(self):
        g = self._data()
        assert "Go" in g["stacks"]
        assert g["core_contributors"][0]["name"] == "alice"
        assert len(g["good_first"]) == 1

    def test_render(self):
        md = render_guide(self._data())
        assert "新贡献者上手指南" in md
        assert "技术栈" in md
        assert "上手步骤" in md
        assert "@alice" in md


if __name__ == "__main__":
    sys.exit(pytest.main([__file__, "-v"]))
