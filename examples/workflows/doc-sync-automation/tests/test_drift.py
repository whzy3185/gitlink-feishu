"""确定性回归护栏：同输入 → 同发现 → 同退出语义。"""

import sys
import unittest
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parent.parent / "scripts"))

from doc_sync_workflow import (  # noqa: E402
    compare_structures,
    discover_pairs,
    parse_structure,
    render_report,
)

EN = """# Title

## Features

| a | b |
|---|---|
| 1 | 2 |
| 3 | 4 |

## Install

Requires Go 1.26+.

```bash
make install
```

## Usage

```bash
run --help
```
"""

ZH = """# 标题

## 功能

| a | b |
|---|---|
| 1 | 2 |

## 安装

需要 Go 1.25+。

```bash
make install
```
"""


class ParseStructureTest(unittest.TestCase):
    def test_parse(self):
        s = parse_structure(EN)
        self.assertEqual([h[1] for h in s.headings], ["Title", "Features", "Install", "Usage"])
        self.assertEqual(s.code_blocks, 2)
        self.assertEqual(s.table_rows, 3)  # 表头 1 行 + 数据 2 行
        self.assertIn("1.26", s.versions)

    def test_code_fence_content_not_parsed_as_heading(self):
        text = "```bash\n# not a heading\n```\n## Real\n"
        s = parse_structure(text)
        self.assertEqual([h[1] for h in s.headings], ["Real"])


class CompareTest(unittest.TestCase):
    def test_drift_detected(self):
        findings = compare_structures(
            "README.md", "README.zh-CN.md", parse_structure(EN), parse_structure(ZH)
        )
        categories = [f.category for f in findings]
        self.assertIn("章节数不一致", categories)      # 缺 Usage 节 → 严重
        self.assertIn("代码块数不一致", categories)    # 2 vs 1 → 中等
        self.assertIn("表格行数不一致", categories)    # 3 vs 2 → 中等
        self.assertIn("版本号不一致", categories)      # 1.26 vs 1.25 → 轻微
        # 严重排在最前
        self.assertEqual(findings[0].severity, "严重")

    def test_identical_docs_no_findings(self):
        findings = compare_structures(
            "a.md", "b.md", parse_structure(EN), parse_structure(EN)
        )
        self.assertEqual(findings, [])

    def test_deterministic(self):
        f1 = compare_structures("a", "b", parse_structure(EN), parse_structure(ZH))
        f2 = compare_structures("a", "b", parse_structure(EN), parse_structure(ZH))
        self.assertEqual([vars(f) for f in f1], [vars(f) for f in f2])


class DiscoverTest(unittest.TestCase):
    def test_discover(self):
        names = ["README.md", "README.zh-CN.md", "LICENSE", "CONTRIBUTING.md"]
        self.assertEqual(discover_pairs(names), [("README.md", "README.zh-CN.md")])

    def test_no_pair(self):
        self.assertEqual(discover_pairs(["README.md", "LICENSE"]), [])

    def test_discover_generic_and_nested(self):
        names = ["docs/guide.md", "docs/guide.zh-CN.md", "USAGE.md", "USAGE_zh.md", "NOTES.zh.md"]
        self.assertEqual(
            discover_pairs(names),
            [("USAGE.md", "USAGE_zh.md"), ("docs/guide.md", "docs/guide.zh-CN.md")],
        )

    def test_translation_file_not_treated_as_base(self):
        self.assertEqual(discover_pairs(["README.zh-CN.md", "README_zh.md"]), [])


class ReportTest(unittest.TestCase):
    def test_report_contains_findings(self):
        findings = compare_structures(
            "README.md", "README.zh-CN.md", parse_structure(EN), parse_structure(ZH)
        )
        report = render_report("o", "r", "master", [("README.md", "README.zh-CN.md", findings)])
        self.assertIn("🔴 严重", report)
        self.assertIn("README.md ⇄ README.zh-CN.md", report)

    def test_report_clean(self):
        report = render_report("o", "r", "", [("a.md", "b.md", [])])
        self.assertIn("无漂移", report)


if __name__ == "__main__":
    unittest.main()
