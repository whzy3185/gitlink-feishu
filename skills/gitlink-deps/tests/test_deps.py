"""gitlink-deps 单元测试。"""

from __future__ import annotations

import sys
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parent.parent / "scripts"))

import pytest

from deps import (
    parse_go_mod, parse_package_json, parse_requirements,
    parse_cargo_toml, parse_pom_xml, scan, render_report, _assess_risks,
)

GO_MOD = """module github.com/example/proj

go 1.26.1

require (
\tgithub.com/spf13/cobra v1.10.2
\tgopkg.in/yaml.v3 v3.0.1
)

require (
\tgithub.com/danieljoos/wincred v1.2.3 // indirect
)
"""

PACKAGE_JSON = """{
  "name": "x",
  "dependencies": {"react": "^18.0.0", "axios": "1.6.0"},
  "devDependencies": {"jest": "^29.0.0"}
}"""

REQUIREMENTS = """# comment
requests==2.31.0
flask>=2.0
numpy
-e .
"""


class TestGoMod:
    def test_parses_direct_and_indirect(self):
        deps = parse_go_mod(GO_MOD)
        names = {d["name"] for d in deps}
        assert "github.com/spf13/cobra" in names
        assert "gopkg.in/yaml.v3" in names
        indirect = [d for d in deps if d["indirect"]]
        assert any(d["name"] == "github.com/danieljoos/wincred" for d in indirect)

    def test_version_extracted(self):
        deps = parse_go_mod(GO_MOD)
        cobra = next(d for d in deps if "cobra" in d["name"])
        assert cobra["version"] == "v1.10.2"


class TestPackageJson:
    def test_deps_and_devdeps(self):
        deps = parse_package_json(PACKAGE_JSON)
        names = {d["name"] for d in deps}
        assert "react" in names and "axios" in names and "jest" in names
        jest = next(d for d in deps if d["name"] == "jest")
        assert jest["indirect"] is True  # devDependency

    def test_invalid_json(self):
        assert parse_package_json("{not json") == []


class TestRequirements:
    def test_parses_pins(self):
        deps = parse_requirements(REQUIREMENTS)
        names = {d["name"] for d in deps}
        assert "requests" in names and "flask" in names and "numpy" in names
        req = next(d for d in deps if d["name"] == "requests")
        assert "2.31.0" in req["version"]

    def test_skips_comments_and_flags(self):
        deps = parse_requirements(REQUIREMENTS)
        names = {d["name"] for d in deps}
        assert "-e" not in names


class TestCargoToml:
    def test_parses_deps(self):
        text = '[package]\nname="x"\n[dependencies]\nserde = "1.0"\ntokio = "1.35"\n'
        deps = parse_cargo_toml(text)
        names = {d["name"] for d in deps}
        assert "serde" in names and "tokio" in names


class TestPomXml:
    def test_parses_dependencies(self):
        text = """<project><dependencies>
        <dependency><groupId>org.junit</groupId><artifactId>junit</artifactId><version>5.0</version></dependency>
        </dependencies></project>"""
        deps = parse_pom_xml(text)
        assert deps[0]["name"] == "org.junit:junit"
        assert deps[0]["version"] == "5.0"


class TestRisks:
    def test_no_manifest(self):
        risks = _assess_risks([], [])
        assert any("未发现依赖声明" in r for r in risks)

    def test_unpinned_flagged(self):
        deps = [{"name": "react", "version": "^18.0.0", "indirect": False}]
        manifests = [{"file": "package.json", "count": 1}]
        risks = _assess_risks(deps, manifests)
        assert any("未锁定" in r for r in risks)


class TestScan:
    class FakeClient:
        def list_dir(self, owner, repo, path, ref):
            if path == "":
                return [{"name": "go.mod", "type": "file"}]
            return []

        def file_content(self, owner, repo, filepath, ref):
            return GO_MOD if filepath == "go.mod" else None

    def test_scan_go_repo(self):
        r = scan("o", "r", client=self.FakeClient())
        assert "Go" in r["ecosystems"]
        assert r["total_deps"] >= 3
        assert r["direct_count"] >= 2

    def test_report_renders(self):
        r = scan("o", "r", client=self.FakeClient())
        report = render_report(r)
        assert "依赖追踪报告" in report
        assert "cobra" in report


if __name__ == "__main__":
    sys.exit(pytest.main([__file__, "-v"]))
