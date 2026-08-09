"""test_match.py — S4 协作匹配的纯单元测试（不联网）。

运行：`pytest scripts/research/` 或 `python scripts/research/test_match.py`
"""
import os
import sys

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))

import match as M  # noqa: E402


def test_cosine_basic():
    assert M.cosine({"a": 1, "b": 0}, {"a": 1, "b": 0}) == 1.0
    assert M.cosine({"a": 1}, {"b": 1}) == 0.0
    # 对称且在 [0,1]
    v = M.cosine({"a": 1, "b": 2}, {"a": 2, "b": 1})
    assert 0.0 < v < 1.0


def test_cosine_empty_safe():
    assert M.cosine({}, {"a": 1}) == 0.0
    assert M.cosine({}, {}) == 0.0


def test_jaccard():
    assert M.jaccard(["python", "go"], ["python", "rust"]) == 1 / 3
    assert M.jaccard([], ["x"]) == 0.0


def test_priority_weight():
    assert M._priority_weight({"priority_name": "高"}) == 3.0
    assert M._priority_weight({"priority_name": "urgent"}) == 3.0
    assert M._priority_weight({"priority_name": "普通"}) == 2.0
    assert M._priority_weight({"priority_name": "低"}) == 1.0
    assert M._priority_weight({}) == 1.0


def test_parse_ratio_percent_string():
    assert M._parse_ratio("1.18%") == 0.0118
    assert abs(M._parse_ratio("50%") - 0.5) < 1e-9


def test_parse_ratio_plain():
    assert M._parse_ratio(0.5) == 0.5
    assert M._parse_ratio("0.2") == 0.2
    assert M._parse_ratio(None) == 0.0
    assert M._parse_ratio("n/a") == 0.0


def test_render_report_has_sections():
    result = {
        "repo": "o/r", "gap_topics": ["deep_learning"], "needed_languages": ["python"],
        "gap_signals": [{"type": "unresolved_issue", "topic": "deep_learning",
                         "evidence": "x", "priority": "高"}],
        "candidates": [{"login": "alice", "score": 20.0, "topic_overlap": 0.5,
                        "language_match": 0.5, "activity_level": "high",
                        "repo_languages": ["python"], "reasons": ["覆盖缺口主题"]}],
        "meta": {"pool_size": 1, "issue_sample": 1},
    }
    md = M.render_report(result)
    assert "缺口分析" in md or "技术缺口" in md
    assert "alice" in md
    mm = M.render_mermaid(result)
    assert mm.startswith("```mermaid") and "alice" in mm


def _run_all():
    fns = [v for k, v in sorted(globals().items()) if k.startswith("test_")]
    for fn in fns:
        fn()
        print(f"PASS {fn.__name__}")
    print(f"\nAll {len(fns)} match tests passed.")


if __name__ == "__main__":
    _run_all()
