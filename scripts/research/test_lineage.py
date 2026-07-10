"""test_lineage.py — lineage.py 纯函数单元测试。

不联网、不调 gitlink-cli。把「取数」与「算法」分离：算法函数接收已取好的
Python 数据结构，本测试用 mock 数据喂算法。

运行： python test_lineage.py
"""
from __future__ import annotations

import os
import sys
import unittest

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))

# 仅 import 算法纯函数，绝不触发联网取数
import lineage as L  # noqa: E402


class TestExperimentClassifier(unittest.TestCase):
    def test_is_experiment_file_hits(self):
        cases = [
            "experiments/mnist/run.py",
            "experiment_train.py",
            "benchmark/imagenet/eval.py",
            "eval/metrics.py",
            "tests/test_model.py",
            "test/foo.py",
            "data/dataset.csv",
            "src/benchmark_infer.py",
        ]
        for p in cases:
            with self.subTest(p=p):
                self.assertTrue(L.is_experiment_file(p), f"应判为实验文件: {p}")

    def test_is_experiment_file_misses(self):
        cases = [
            "src/model.py",
            "README.md",
            "docs/index.md",
            "main.go",
            "config.yaml",
            "pkg/utils/util.py",
        ]
        for p in cases:
            with self.subTest(p=p):
                self.assertFalse(L.is_experiment_file(p), f"不应判为实验文件: {p}")

    def test_empty(self):
        self.assertFalse(L.is_experiment_file(""))
        self.assertFalse(L.is_experiment_file(None))  # type: ignore[arg-type]


class TestDocClassifier(unittest.TestCase):
    def test_md_anywhere(self):
        self.assertTrue(L.is_doc_file("README.md"))
        self.assertTrue(L.is_doc_file("docs/guide.md"))
        self.assertTrue(L.is_doc_file("deep/nested/notes.md"))
        self.assertTrue(L.is_doc_file("GUIDE.MD"))  # 大小写不敏感

    def test_docs_dir(self):
        self.assertTrue(L.is_doc_file("docs/index.html"))
        self.assertTrue(L.is_doc_file("docs/config.yaml"))
        self.assertTrue(L.is_doc_file("a/docs/b.txt"))

    def test_non_doc(self):
        self.assertFalse(L.is_doc_file("src/main.py"))
        self.assertFalse(L.is_doc_file("tests/x.py"))
        self.assertFalse(L.is_doc_file("data.csv"))
        self.assertFalse(L.is_doc_file(""))


class TestTimestampParse(unittest.TestCase):
    def test_iso_string(self):
        t = L._to_epoch("2024-05-01T08:00:00Z")
        self.assertGreater(t, 0)
        # 反解回日期
        self.assertTrue(L._iso_date(t).startswith("2024-05-01"))

    def test_int_seconds(self):
        self.assertAlmostEqual(L._to_epoch(1714521600), 1714521600.0)

    def test_int_millis(self):
        self.assertAlmostEqual(L._to_epoch(1714521600000), 1714521600.0)

    def test_garbage(self):
        self.assertEqual(L._to_epoch(None), 0.0)
        self.assertEqual(L._to_epoch(""), 0.0)
        self.assertEqual(L._to_epoch("not-a-date"), 0.0)

    def test_iso_date_zero(self):
        self.assertEqual(L._iso_date(0.0), "")


class TestBranchMap(unittest.TestCase):
    def test_single_default_branch(self):
        commits = [
            {"timestamp": "2024-01-01T00:00:00Z"},
            {"timestamp": "2024-06-01T00:00:00Z"},
            {"timestamp": "2024-03-01T00:00:00Z"},
        ]
        bm = L.build_branch_map(commits, "master")
        self.assertEqual(len(bm), 1)
        b = bm[0]
        self.assertEqual(b["name"], "master")
        self.assertTrue(b["is_default"])
        self.assertEqual(b["commits"], 3)
        # 最后活跃应取最大值 2024-06-01
        self.assertTrue(b["last_active"].startswith("2024-06-01"))

    def test_empty(self):
        bm = L.build_branch_map([], "main")
        self.assertEqual(bm, [{"name": "main", "commits": 0,
                               "last_active": "", "is_default": True}])

    def test_non_list(self):
        bm = L.build_branch_map(None, "main")  # type: ignore[arg-type]
        self.assertEqual(bm[0]["commits"], 0)


class TestCommitTimeline(unittest.TestCase):
    def test_aggregation_and_sort(self):
        commits = [
            {"timestamp": "2024-01-02T00:00:00Z"},
            {"timestamp": "2024-01-02T12:00:00Z"},
            {"timestamp": "2024-01-01T00:00:00Z"},
        ]
        tl = L.commit_timeline(commits, bucket="day")
        self.assertEqual(tl, [
            {"date": "2024-01-01", "count": 1},
            {"date": "2024-01-02", "count": 2},
        ])

    def test_skips_garbage(self):
        commits = [{"timestamp": "bad"}, {"timestamp": ""}]
        self.assertEqual(L.commit_timeline(commits), [])


class TestPrMergePatterns(unittest.TestCase):
    def test_extract_and_sort(self):
        prs = [
            {"index": 10, "title": "feat A", "status": 1,
             "pr_created_unix": 1717200000, "changed_files": 5},
            {"index": 2, "title": "feat B", "status": 1,
             "pr_merged_unix": 1714521600, "changed_files": 20},
            {"index": 5, "title": "feat C", "status": 1,
             "pr_created_unix": 1715000000, "additions": 3, "deletions": 4},
        ]
        out = L.pr_merge_patterns(prs)
        self.assertEqual(len(out), 3)
        # 升序：1714521600(2024-05-01) < 1715000000 < 1717200000
        self.assertEqual(out[0]["number"], 2)
        self.assertEqual(out[0]["changed_files"], 20)
        self.assertTrue(out[0]["merged_time"].startswith("2024-05"))
        # additions/deletions 兜底近似
        last = next(p for p in out if p["number"] == 5)
        self.assertEqual(last["changed_files"], 7)
        # 空 merged_time 排末尾
        no_time = L.pr_merge_patterns([
            {"index": 1, "title": "x", "status": 1},
            {"index": 2, "title": "y", "status": 1, "pr_created_unix": 1714521600},
        ])
        self.assertEqual(no_time[-1]["number"], 1)

    def test_empty(self):
        self.assertEqual(L.pr_merge_patterns([]), [])


class TestDocEvolution(unittest.TestCase):
    def test_filter_docs(self):
        tree = [
            {"name": "guide.md", "path": "docs/guide.md", "date": "2024-03-01"},
            {"name": "index.html", "path": "docs/index.html"},
            {"name": "model.py", "path": "src/model.py"},  # 排除
            {"name": "old_2022-01-01.md", "path": "docs/old_2022-01-01.md"},
        ]
        out = L.doc_evolution(tree)
        files = {d["file"] for d in out}
        self.assertIn("guide.md", files)
        self.assertIn("index.html", files)
        self.assertIn("old_2022-01-01.md", files)
        self.assertNotIn("model.py", files)
        # 文件名日期回退
        old = next(d for d in out if d["file"] == "old_2022-01-01.md")
        self.assertEqual(old["last_date"], "2022-01-01")
        # 显式 date 字段优先
        g = next(d for d in out if d["file"] == "guide.md")
        self.assertEqual(g["last_date"], "2024-03-01")


class TestInnovationPoints(unittest.TestCase):
    def test_high_impact_and_milestone(self):
        prs = [
            {"index": 1, "title": "chore: typo", "status": 1,
             "pr_created_unix": 1714521600, "changed_files": 2},  # 低影响，无关键词
            {"index": 2, "title": "feat: add transformer model", "status": 1,
             "pr_created_unix": 1715000000, "changed_files": 25},  # 大规模+关键词
            {"index": 3, "title": "implement benchmark suite", "status": 1,
             "pr_created_unix": 1717200000, "changed_files": 4},  # 仅关键词
        ]
        out = L.innovation_points(prs, commits=[])
        # 应识别出 PR#2 和 PR#3，PR#1 被过滤
        nums = sorted(it["description"] for it in out)
        self.assertTrue(any("transformer" in n.lower() for n in nums))
        self.assertTrue(any("benchmark" in n.lower() for n in nums))
        cats = [it["category"] for it in out]
        self.assertIn("大规模重构/新特性", cats)
        self.assertIn("特性引入", cats)
        # 大规模 PR 排前（impact 更高）
        self.assertIn("transformer", out[0]["description"].lower())
        # 每条都带证据
        for it in out:
            self.assertTrue(it["evidence"])
            self.assertTrue(it["category"])

    def test_empty(self):
        self.assertEqual(L.innovation_points([], []), [])

    def test_top_limit(self):
        prs = [{"index": i, "title": f"add feature {i}", "status": 1,
                "pr_created_unix": 1714521600 + i * 86400, "changed_files": 15}
               for i in range(20)]
        out = L.innovation_points(prs, commits=[], top=5)
        self.assertEqual(len(out), 5)


class TestRender(unittest.TestCase):
    """渲染函数不抛异常、产出非空。"""

    def _result(self):
        return {
            "scenario": "S1_repository_research_insight",
            "repo": "o/r", "default_branch": "master",
            "commit_timeline": [{"date": "2024-01-01", "count": 3}],
            "branch_map": [{"name": "master", "commits": 5, "last_active": "2024-06-01",
                            "is_default": True}],
            "pr_merge_patterns": [{"number": 2, "title": "feat A", "status": 1,
                                   "merged_time": "2024-06-01", "changed_files": 9}],
            "doc_evolution": [{"file": "guide.md", "last_date": "2024-03-01"}],
            "experiment_files": ["benchmark/eval.py"],
            "innovation_points": [{"description": "feat A", "evidence": "PR #2",
                                   "category": "特性引入"}],
            "meta": {"commit_count": 5, "merged_pr_count": 1, "doc_count": 1,
                     "experiment_file_count": 1},
        }

    def test_report(self):
        r = L.render_report(self._result())
        self.assertIn("仓库级科研项目洞悉报告", r)
        self.assertIn("master", r)
        self.assertIn("feat A", r)

    def test_mermaid(self):
        m = L.render_mermaid(self._result())
        self.assertIn("gitGraph", m)
        self.assertIn("master", m)


if __name__ == "__main__":
    unittest.main(verbosity=2)
