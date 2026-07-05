"""确定性回归护栏：同输入 → 同分 → 同等级。"""

import sys
import tempfile
import unittest
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parent.parent / "scripts"))

from repro_audit import audit, read_repos_file, render_report, render_summary  # noqa: E402

GOOD_README = """# Project

## How to run

```bash
make train
```

## Dataset

Download the dataset from ...

## Citation

```bibtex
@article{x2026}
```
"""


class AuditTest(unittest.TestCase):
    def test_full_marks(self):
        names = ["README.md", "LICENSE", "CITATION.cff", "requirements.txt", "Makefile"]
        dirs = ["data", "tests", "src"]
        results = audit(names, dirs, GOOD_README, release_count=2)
        report, total = render_report("o", "r", "", results)
        self.assertEqual(total, 100)
        self.assertIn("等级 A", report)

    def test_empty_repo(self):
        results = audit([], [], "", release_count=0)
        report, total = render_report("o", "r", "", results)
        self.assertEqual(total, 0)
        self.assertIn("等级 D", report)

    def test_readme_without_usage_partial(self):
        results = audit(["README.md"], [], "# hi", release_count=0)
        readme = next(r for r in results if r.name == "README 与运行说明")
        self.assertEqual(readme.score, 8)  # 有 README 无运行说明 → 部分分

    def test_citation_in_readme_counts(self):
        results = audit(["README.md"], [], "## Citation\n@article{x}", release_count=0)
        cit = next(r for r in results if r.name == "引用信息")
        self.assertEqual(cit.score, cit.weight)

    def test_scripts_dir_as_entry(self):
        results = audit([], ["scripts"], "", release_count=0)
        entry = next(r for r in results if r.name == "运行入口")
        self.assertEqual(entry.score, entry.weight)

    def test_deterministic(self):
        a = audit(["README.md", "go.mod"], ["tests"], GOOD_README, 1)
        b = audit(["README.md", "go.mod"], ["tests"], GOOD_README, 1)
        self.assertEqual([vars(x) for x in a], [vars(y) for y in b])

    def test_advice_present_for_failures(self):
        results = audit([], [], "", 0)
        for r in results:
            self.assertTrue(r.advice, f"{r.name} 应给出修复建议")

    def test_read_repos_file(self):
        with tempfile.NamedTemporaryFile("w", suffix=".txt", delete=False, encoding="utf-8") as f:
            f.write("# 注释\n\nowner1/repo1\n  owner2/repo2  \n")
            path = f.name
        self.assertEqual(read_repos_file(path), [("owner1", "repo1"), ("owner2", "repo2")])

    def test_read_repos_file_invalid_line(self):
        with tempfile.NamedTemporaryFile("w", suffix=".txt", delete=False, encoding="utf-8") as f:
            f.write("not-a-repo-line\n")
            path = f.name
        with self.assertRaises(ValueError):
            read_repos_file(path)

    def test_render_summary_sorted(self):
        summary = render_summary([("o/low", 8), ("o/high", 92), ("o/mid", 58)])
        rows = [line for line in summary.splitlines() if line.startswith("| o/")]
        self.assertEqual([r.split(" | ")[0] for r in rows], ["| o/high", "| o/mid", "| o/low"])
        self.assertIn("A（可复现性良好）", rows[0])
        self.assertIn("D（复现困难）", rows[2])


if __name__ == "__main__":
    unittest.main()
