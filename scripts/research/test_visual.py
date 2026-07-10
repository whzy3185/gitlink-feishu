"""test_visual.py — S6 科研成果可视化的纯单元测试（不联网、不调 gitlink-cli）。

把「取数」与「算法」分离：算法函数接收已构造好的 Python 数据结构（mock commits/issues/...），
测试只覆盖 bin_weekly / contribution_heatmap / extract_paper_links / classify_artifacts，
不测 plotly 渲染。

运行：`python scripts/research/test_visual.py`
"""
import os
import sys
from datetime import datetime, timedelta, timezone

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))

import visual as V  # noqa: E402


# ---------------------------------------------------------------------------
# 辅助：构造「最近 N 天」的时间戳/ISO 字符串
# ---------------------------------------------------------------------------

def _days_ago_ts(days: int) -> float:
    return (datetime.now(timezone.utc) - timedelta(days=days)).timestamp()


def _days_ago_iso(days: int) -> str:
    return (datetime.now(timezone.utc) - timedelta(days=days)).strftime("%Y-%m-%dT%H:%M:%SZ")


# ---------------------------------------------------------------------------
# _to_timestamp
# ---------------------------------------------------------------------------

def test_to_timestamp_int_seconds():
    assert V._to_timestamp(0) == 0.0
    assert V._to_timestamp(1719500000) == 1719500000.0
    assert V._to_timestamp("1719500000") == 1719500000.0


def test_to_timestamp_millis():
    # 13 位 → 视为毫秒
    assert abs(V._to_timestamp(1719500000000) - 1719500000.0) < 1e-3


def test_to_timestamp_iso():
    ts = V._to_timestamp("2024-06-15T12:00:00Z")
    assert ts > 0
    # 无效字符串 → 0
    assert V._to_timestamp("not-a-date") == 0.0
    assert V._to_timestamp(None) == 0.0
    assert V._to_timestamp("") == 0.0


# ---------------------------------------------------------------------------
# bin_weekly
# ---------------------------------------------------------------------------

def test_bin_weekly_shape_and_sum():
    weeks = 4
    commits = [
        {"timestamp": _days_ago_iso(2)},    # 本周
        {"timestamp": _days_ago_iso(9)},    # 上周
        {"timestamp": _days_ago_iso(9)},
        {"timestamp": _days_ago_iso(20)},   # 3 周前
    ]
    out = V.bin_weekly(commits, weeks, V._commit_time)
    assert set(out.keys()) == {"labels", "counts"}
    assert len(out["labels"]) == weeks
    assert len(out["counts"]) == weeks
    assert sum(out["counts"]) == len(commits)
    # 标签是 ISO 周（形如 2026-Wxx）
    assert all("-W" in lab for lab in out["labels"])


def test_bin_weekly_drops_out_of_window_and_invalid():
    weeks = 3
    commits = [
        {"timestamp": _days_ago_iso(1)},            # 窗口内
        {"timestamp": _days_ago_iso(365)},          # 太早（窗口外）→ 丢弃
        {"timestamp": "garbage"},                   # 非法 → 丢弃
        {},                                         # 无时间字段 → 丢弃
    ]
    out = V.bin_weekly(commits, weeks, V._commit_time)
    assert sum(out["counts"]) == 1


def test_bin_weekly_issues_uses_created_at():
    weeks = 2
    issues = [
        {"created_at": _days_ago_iso(3)},    # 窗口内
        {"created_at": _days_ago_iso(40)},   # 窗口外（>2 周）→ 丢弃
    ]
    out = V.bin_weekly(issues, weeks, V._issue_time)
    assert sum(out["counts"]) == 1  # 只有 3 天前那条进窗口


def test_bin_weekly_prs_uses_pr_created_unix():
    weeks = 2
    prs = [
        {"pr_created_unix": int(_days_ago_ts(2))},
        {"pr_created_unix": int(_days_ago_ts(50))},
    ]
    out = V.bin_weekly(prs, weeks, V._pr_time)
    assert sum(out["counts"]) == 1


def test_bin_weekly_empty():
    out = V.bin_weekly([], 5, V._commit_time)
    assert len(out["labels"]) == 5
    assert out["counts"] == [0, 0, 0, 0, 0]


# ---------------------------------------------------------------------------
# contribution_heatmap
# ---------------------------------------------------------------------------

def test_heatmap_matrix_shape():
    weeks = 3
    contributors = [
        {"login": "alice", "contributions": 100},
        {"login": "bob", "contributions": 5},
    ]
    commits = [
        {"timestamp": _days_ago_iso(2), "author": {"login": "alice"}},
        {"timestamp": _days_ago_iso(2), "author": {"login": "alice"}},
        {"timestamp": _days_ago_iso(9), "author": {"login": "bob"}},
    ]
    hm = V.contribution_heatmap(contributors, commits, weeks)
    assert set(hm.keys()) == {"users", "weeks", "matrix"}
    assert hm["users"][:2] == ["alice", "bob"]
    assert len(hm["matrix"]) == len(hm["users"])
    for row in hm["matrix"]:
        assert len(row) == weeks
    # alice 行总和 = 2
    alice_row = hm["matrix"][hm["users"].index("alice")]
    assert sum(alice_row) == 2
    bob_row = hm["matrix"][hm["users"].index("bob")]
    assert sum(bob_row) == 1


def test_heatmap_includes_commit_authors_not_in_contributors():
    weeks = 2
    contributors = [{"login": "alice", "contributions": 1}]
    commits = [
        {"timestamp": _days_ago_iso(2), "author": {"login": "alice"}},
        {"timestamp": _days_ago_iso(3), "author": {"login": "carol"}},  # 不在 contributors
    ]
    hm = V.contribution_heatmap(contributors, commits, weeks)
    assert "carol" in hm["users"]


def test_heatmap_top_users_cap():
    weeks = 2
    contributors = [{"login": f"u{i}", "contributions": i} for i in range(20)]
    commits = [{"timestamp": _days_ago_iso(1), "author": {"login": f"u{i}"}}
               for i in range(20)]
    hm = V.contribution_heatmap(contributors, commits, weeks, top_users=5)
    assert len(hm["users"]) <= 5
    assert len(hm["matrix"]) == len(hm["users"])


def test_heatmap_empty():
    hm = V.contribution_heatmap([], [], 4)
    assert hm["users"] == []
    assert hm["matrix"] == []
    assert len(hm["weeks"]) == 4


# ---------------------------------------------------------------------------
# extract_paper_links
# ---------------------------------------------------------------------------

def test_extract_arxiv_url():
    text = "See https://arxiv.org/abs/2401.00012 for details."
    links = V.extract_paper_links(text)
    assert len(links) == 1
    assert links[0]["type"] == "arxiv"
    assert links[0]["target"] == "https://arxiv.org/abs/2401.00012"
    # snippet 截取自原文上下文（窗口较窄，断言前缀即可）
    assert links[0]["source_text_snippet"].startswith("See https://arxiv.org")


def test_extract_arxiv_pdf_url():
    text = "paper: https://arxiv.org/pdf/2305.12345.pdf"
    links = V.extract_paper_links(text)
    assert len(links) == 1
    # 归一成 abs 形式
    assert links[0]["target"] == "https://arxiv.org/abs/2305.12345"


def test_extract_arxiv_bare():
    text = "We use arXiv:2103.07018 in our method."
    links = V.extract_paper_links(text)
    assert any(l["target"] == "https://arxiv.org/abs/2103.07018" for l in links)


def test_extract_doi_url():
    text = "Cited from https://doi.org/10.1000/182"
    links = V.extract_paper_links(text)
    assert len(links) == 1
    assert links[0]["type"] == "doi"
    assert links[0]["target"] == "https://doi.org/10.1000/182"


def test_extract_doi_bare():
    text = "Reference 10.1109/5.771073 shows that."
    links = V.extract_paper_links(text)
    assert len(links) == 1
    assert links[0]["target"] == "https://doi.org/10.1109/5.771073"


def test_extract_dedup_same_id():
    text = ("arxiv 1 https://arxiv.org/abs/2401.00012 "
            "and again https://arxiv.org/abs/2401.00012")
    links = V.extract_paper_links(text)
    assert len(links) == 1


def test_extract_dedup_across_doi_forms():
    # doi.org 形式与裸 DOI 视为同一条
    text = "https://doi.org/10.1000/182 and bare 10.1000/182 again"
    links = V.extract_paper_links(text)
    # 同一 DOI 只出现一次
    targets = [l["target"] for l in links]
    assert targets.count("https://doi.org/10.1000/182") == 1


def test_extract_multiple_and_order():
    text = ("first https://arxiv.org/abs/2401.00012 "
            "then https://doi.org/10.1000/182")
    links = V.extract_paper_links(text)
    assert len(links) == 2
    # 按出现位置排序
    assert links[0]["type"] == "arxiv"
    assert links[1]["type"] == "doi"


def test_extract_none_in_text():
    assert V.extract_paper_links("no links here at all") == []
    assert V.extract_paper_links("") == []


def test_extract_strips_trailing_punct_from_doi():
    text = "see 10.1000/abc123, then more."
    links = V.extract_paper_links(text)
    assert links[0]["target"].endswith("/abc123")  # 末尾逗号/句号被清掉
    assert not links[0]["target"].rstrip().endswith(",")


# ---------------------------------------------------------------------------
# classify_artifacts
# ---------------------------------------------------------------------------

def test_classify_paper_and_ipynb():
    tree = [
        {"path": "docs/paper.pdf"},
        {"path": "notebooks/demo.ipynb"},
    ]
    out = V.classify_artifacts(tree)
    cats = {a["path"]: a["category"] for a in out}
    assert cats["docs/paper.pdf"] == "paper"
    assert cats["notebooks/demo.ipynb"] == "paper"


def test_classify_dataset():
    tree = [
        {"path": "data/train.csv"},
        {"path": "datasets/x.parquet"},
    ]
    out = V.classify_artifacts(tree)
    cats = {a["path"]: a["category"] for a in out}
    assert cats["data/train.csv"] == "dataset"


def test_classify_model():
    tree = [
        {"path": "model/best.ckpt"},
        {"path": "models/v2.onnx"},
    ]
    out = V.classify_artifacts(tree)
    cats = {a["path"]: a["category"] for a in out}
    assert cats["model/best.ckpt"] == "model"
    assert cats["models/v2.onnx"] == "model"


def test_classify_benchmark():
    tree = [{"path": "benchmark/glue/run.py"}]
    out = V.classify_artifacts(tree)
    assert out[0]["category"] == "benchmark"


def test_classify_ignores_unrelated():
    tree = [
        {"path": "src/main.py"},
        {"path": "README.md"},
        {"path": "tools/util.go"},
    ]
    out = V.classify_artifacts(tree)
    assert out == []  # 都不命中任何类别


def test_classify_dedup_same_path():
    tree = [
        {"path": "data/a.csv"},
        {"path": "data/a.csv"},  # 重复
    ]
    out = V.classify_artifacts(tree)
    assert len(out) == 1


def test_classify_handles_name_only():
    # 没有 path 只有 name 的条目也能处理
    tree = [{"name": "paper.pdf"}]
    out = V.classify_artifacts(tree)
    assert len(out) == 1
    assert out[0]["category"] == "paper"


def test_classify_empty_and_non_dict():
    assert V.classify_artifacts([]) == []
    assert V.classify_artifacts(None) == []
    assert V.classify_artifacts(["str", 123, None]) == []


def test_artifact_summary():
    arts = [
        {"path": "a.pdf", "category": "paper"},
        {"path": "b.ipynb", "category": "paper"},
        {"path": "x.csv", "category": "dataset"},
        {"path": "m.ckpt", "category": "model"},
    ]
    s = V.artifact_summary(arts)
    assert s == {"paper": 2, "dataset": 1, "model": 1, "benchmark": 0}


# ---------------------------------------------------------------------------
# render_report（不渲染 plotly，只验证 markdown 结构）
# ---------------------------------------------------------------------------

def test_render_report_has_sections():
    result = {
        "repo": "o/r", "weeks": 4,
        "timeline": {"labels": ["W1", "W2", "W3", "W4"],
                     "commits": [1, 2, 3, 4], "issues": [0, 1, 0, 2],
                     "prs": [0, 0, 1, 0]},
        "heatmap": {"users": ["alice", "bob"], "weeks": ["W1", "W2"],
                    "matrix": [[1, 2], [0, 1]]},
        "languages": {"Python": "99%"},
        "milestones": [],
        "paper_links": [{"type": "arxiv", "target": "https://arxiv.org/abs/2401.00012",
                         "source_text_snippet": "see arxiv"}],
        "artifacts": [{"path": "p.pdf", "category": "paper", "name": "p.pdf"}],
        "artifact_summary": {"paper": 1, "dataset": 0, "model": 0, "benchmark": 0},
        "meta": {"commit_count": 10, "issue_count": 3, "pr_count": 1,
                 "contributor_count": 2, "milestone_count": 0},
    }
    md = V.render_report(result)
    assert "科研成果可视化" in md
    assert "o/r" in md
    assert "alice" in md
    assert "arxiv.org/abs/2401.00012" in md
    # 含周快照表头
    assert "commits" in md


# ---------------------------------------------------------------------------
# 端到端（算法层）：run() 复用 raw dict，不联网
# ---------------------------------------------------------------------------

def test_run_with_mock_raw():
    raw = {
        "commits": [{"timestamp": _days_ago_iso(2), "author": {"login": "alice"},
                     "message": "see https://arxiv.org/abs/2401.00012"}],
        "issues": [{"created_at": _days_ago_iso(3)}],
        "prs": [{"pr_created_unix": int(_days_ago_ts(4))}],
        "milestones": [{"name": "v1.0", "due_on": _days_ago_iso(30)}],
        "languages": {"Python": "99%"},
        "contributors": [{"login": "alice", "contributions": 1}],
        "readme": "ref https://doi.org/10.1000/182 here",
        "tree": [{"path": "data/x.csv"}, {"path": "paper.pdf"}],
    }
    result = V.run("owner", "repo", weeks=4, raw=raw)
    assert result["scenario"] == "S6_research_visualization"
    assert result["repo"] == "owner/repo"
    assert result["weeks"] == 4
    # 论文链接同时来自 readme 和 commit message
    targets = {p["target"] for p in result["paper_links"]}
    assert "https://arxiv.org/abs/2401.00012" in targets
    assert "https://doi.org/10.1000/182" in targets
    # 产物分类
    cats = {a["category"] for a in result["artifacts"]}
    assert cats == {"dataset", "paper"}
    # 时间线长度 = weeks
    assert len(result["timeline"]["labels"]) == 4


def _run_all():
    fns = [v for k, v in sorted(globals().items()) if k.startswith("test_")]
    for fn in fns:
        fn()
        print(f"PASS {fn.__name__}")
    print(f"\nAll {len(fns)} visual tests passed.")


if __name__ == "__main__":
    _run_all()
