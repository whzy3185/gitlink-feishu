"""gitlink-scaffold 单元测试。"""

from __future__ import annotations

import sys
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parent.parent / "scripts"))

import pytest

from scaffold import (
    check_repo, generate_templates, render_report,
    HEALTH_FILES, TEMPLATE_BUILDERS,
)


class FakeClient:
    """用预设目录内容模拟 GitLinkClient.list_dir。"""

    def __init__(self, files_by_dir):
        self.files_by_dir = files_by_dir

    def list_dir(self, owner, repo, path, ref):
        names = self.files_by_dir.get(path, [])
        return [{"name": n, "type": "file"} for n in names]


class TestCheckRepo:
    def test_all_missing(self):
        client = FakeClient({"": []})
        r = check_repo("o", "r", client=client)
        assert r["health_score"] == 0
        assert len(r["missing"]) == len(HEALTH_FILES)

    def test_readme_license_present(self):
        client = FakeClient({"": ["readme.md", "license"]})
        r = check_repo("o", "r", client=client)
        # README(20) + LICENSE(20) = 40
        assert r["health_score"] == 40
        present_keys = {p["key"] for p in r["present"]}
        assert "readme" in present_keys
        assert "license" in present_keys

    def test_contributing_in_subdir(self):
        client = FakeClient({"": [], ".gitlink": ["contributing.md"]})
        r = check_repo("o", "r", client=client)
        assert any(p["key"] == "contributing" for p in r["present"])

    def test_missing_critical_flagged(self):
        client = FakeClient({"": ["contributing.md"]})
        r = check_repo("o", "r", client=client)
        crit_keys = {m["key"] for m in r["missing_critical"]}
        assert "readme" in crit_keys
        assert "license" in crit_keys

    def test_full_score(self):
        allfiles = ["readme.md", "license", "contributing.md", "code_of_conduct.md",
                    "security.md", "issue_template.md", "pull_request_template.md",
                    "changelog.md"]
        client = FakeClient({"": allfiles})
        r = check_repo("o", "r", client=client)
        assert r["health_score"] == 100
        assert r["missing"] == []


class TestGenerate:
    def test_generates_templates(self, tmp_path):
        missing = [{"key": "contributing", "name": "CONTRIBUTING", "critical": False},
                   {"key": "security", "name": "SECURITY", "critical": False}]
        gen = generate_templates(missing, "o", "r", tmp_path)
        assert len(gen) == 2
        assert (tmp_path / "CONTRIBUTING.md").exists()
        assert (tmp_path / "SECURITY.md").exists()

    def test_skips_no_template(self, tmp_path):
        # readme/license 无模板生成器
        missing = [{"key": "readme", "name": "README", "critical": True}]
        gen = generate_templates(missing, "o", "r", tmp_path)
        assert gen == []

    def test_template_content_has_repo(self, tmp_path):
        missing = [{"key": "contributing", "name": "CONTRIBUTING", "critical": False}]
        generate_templates(missing, "MyOrg", "MyRepo", tmp_path)
        text = (tmp_path / "CONTRIBUTING.md").read_text(encoding="utf-8")
        assert "MyOrg/MyRepo" in text


class TestReport:
    def test_report_renders(self):
        client = FakeClient({"": ["readme.md"]})
        r = check_repo("o", "r", client=client)
        report = render_report(r)
        assert "社区健康文件体检" in report
        assert "健康度评分" in report

    def test_all_present_message(self):
        allfiles = ["readme.md", "license", "contributing.md", "code_of_conduct.md",
                    "security.md", "issue_template.md", "pull_request_template.md", "changelog.md"]
        r = check_repo("o", "r", client=FakeClient({"": allfiles}))
        assert "全部齐全" in render_report(r)


if __name__ == "__main__":
    sys.exit(pytest.main([__file__, "-v"]))
