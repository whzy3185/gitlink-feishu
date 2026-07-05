"""确定性回归护栏：同输入 → 同分 → 同等级。"""

import sys
import unittest
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parent.parent / "scripts"))

from repro_audit import audit, render_report  # noqa: E402

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


if __name__ == "__main__":
    unittest.main()
