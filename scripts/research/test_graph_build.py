"""test_graph_build.py — S2 知识图谱的纯单元测试（不联网、不调 gitlink-cli）。

把「取数」与「算法」分离：build_graph() 只吃已构造好的 mock 数据结构。
运行：`python test_graph_build.py` 或 `pytest scripts/research/`
"""
import os
import sys

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))

import graph_build as G  # noqa: E402


# ---------------------------------------------------------------------------
# mock 数据构造
# ---------------------------------------------------------------------------

def _mock_repos():
    return [
        {
            "fullname": "alice/mindspore-vision",
            "identifier": "mindspore-vision",
            "author": {"login": "alice"},
            "description": "deep learning library for computer vision, "
                           "image classification and object detection using CNN",
            "language": {"name": "Python"},
            "praises_count": 120,
            "forked_count": 30,
        },
        {
            "fullname": "bob/mindnlp",
            "identifier": "mindnlp",
            "author": {"login": "bob"},
            "description": "A nlp library: transformer, bert, text classification",
            "language": {"name": "Python"},
            "praises_count": 200,
            "forked_count": 50,
        },
    ]


def _mock_contributors_map():
    # alice 既贡献自己的 repo，也贡献 bob 的 repo（→ 核心学者 + collaborates_with）
    return {
        "alice/mindspore-vision": [
            {"login": "alice", "contribution_perc": "60.0%"},
            {"login": "carol", "contribution_perc": "40.0%"},
            {"login": "i-robot", "contribution_perc": "0.0%"},  # bot，应被过滤
        ],
        "bob/mindnlp": [
            {"login": "alice", "contribution_perc": "10.0%"},
            {"login": "bob", "contribution_perc": "90.0%"},
            {"login": "dependabot", "contribution_perc": "1.0%"},  # bot
        ],
    }


def _mock_readmes():
    return {
        "alice/mindspore-vision": "pytorch and resnet for object detection",
        "bob/mindnlp": "pretrain gpt and llm models",
    }


def _build():
    return G.build_graph(_mock_repos(), _mock_contributors_map(),
                         {}, _mock_readmes(), keywords=["vision", "nlp"])


# ---------------------------------------------------------------------------
# 工具函数
# ---------------------------------------------------------------------------

def test_is_bot():
    assert G.is_bot("i-robot") is True
    assert G.is_bot("dependabot[bot]") is True
    assert G.is_bot("alice") is False
    assert G.is_bot("") is True


def test_parse_ratio():
    assert G.parse_ratio("60.0%") == 0.6
    assert abs(G.parse_ratio("0.5") - 0.5) < 1e-9
    assert G.parse_ratio(0.3) == 0.3
    assert G.parse_ratio(None) == 0.0
    assert G.parse_ratio("n/a") == 0.0


# ---------------------------------------------------------------------------
# 建图：节点计数
# ---------------------------------------------------------------------------

def test_node_counts():
    result = _build()
    by_type = {}
    for n in result["nodes"]:
        by_type[n["type"]] = by_type.get(n["type"], 0) + 1
    assert by_type.get("repo", 0) == 2
    # alice / carol / bob = 3 个学者（bot 被过滤）
    assert by_type.get("scholar", 0) == 3
    # 主题：CV + deep_learning + nlp = 至少 3 个
    assert by_type.get("topic", 0) >= 3
    assert result["meta"]["repo_count"] == 2
    assert result["meta"]["scholar_count"] == 3


def test_repo_node_props():
    result = _build()
    repo_nodes = [n for n in result["nodes"] if n["type"] == "repo"]
    labels = {n["label"] for n in repo_nodes}
    assert "alice/mindspore-vision" in labels
    r = [n for n in repo_nodes if n["label"] == "alice/mindspore-vision"][0]
    assert r["props"]["language"] == "Python"
    assert r["props"]["stars"] == 120
    assert r["props"]["forks"] == 30


# ---------------------------------------------------------------------------
# 建图：边
# ---------------------------------------------------------------------------

def test_contributes_to_weight():
    result = _build()
    edges = result["edges"]
    contrib = [e for e in edges if e["type"] == "contributes_to"
               and e["source"] == "scholar:alice"
               and e["target"] == "repo:alice/mindspore-vision"]
    assert contrib, "应有 alice→自己repo 的 contributes_to 边"
    # 60% → 0.6
    assert abs(contrib[0]["weight"] - 0.6) < 1e-9


def test_owns_edge():
    result = _build()
    owns = [e for e in result["edges"] if e["type"] == "owns"
            and e["source"] == "scholar:bob"
            and e["target"] == "repo:bob/mindnlp"]
    assert owns, "bob 应有 owns 边到自己的 repo"


def test_covers_topic_weight_range():
    """covers_topic 权重应在 (0, 1] 且 <= 1。"""
    result = _build()
    covers = [e for e in result["edges"] if e["type"] == "covers_topic"]
    assert covers, "应至少有一条 covers_topic 边"
    for e in covers:
        assert 0.0 <= e["weight"] <= 1.0
    # 命中 computer_vision 主题（description 含 object detection/image classification）
    cv_edges = [e for e in covers if e["target"] == "topic:computer_vision"]
    assert cv_edges, "应识别出 computer_vision 主题"


def test_collaborates_with():
    """alice 与 carol 同在 alice/mindspore-vision → 至少一条 collaborates_with。"""
    result = _build()
    collab = [e for e in result["edges"] if e["type"] == "collaborates_with"]
    assert collab, "应有 collaborates_with 边"
    pair = {e["source"] for e in collab}
    assert "scholar:alice" in pair


def test_related_to_when_topic_cooccur():
    """computer_vision 与 deep_learning 在同一 repo 共现 → related_to。"""
    result = _build()
    related = [e for e in result["edges"] if e["type"] == "related_to"
               and e["source"] == "topic:computer_vision"]
    # mindspore-vision 的 description 同时命中 CV + deep_learning
    assert related, "共现主题应有 related_to 边"
    assert any(e["target"] == "topic:deep_learning" for e in related)


# ---------------------------------------------------------------------------
# 衍生统计
# ---------------------------------------------------------------------------

def test_core_scholars():
    result = _build()
    # alice 出现在 2 个 repo → 排首位
    top = result["core_scholars"][0]
    assert top["login"] == "alice"
    assert top["repo_count"] == 2


def test_topic_heat_top():
    result = _build()
    heat = result["topic_heat"]
    assert heat, "应有主题热度榜"
    # deep_learning 在两个 repo 都命中 → 应在前列
    topics = [h["topic"] for h in heat]
    assert "deep_learning" in topics


# ---------------------------------------------------------------------------
# 渲染
# ---------------------------------------------------------------------------

def test_render_mermaid_header():
    result = _build()
    mmd = G.render_mermaid(result)
    assert "graph TD" in mmd
    assert "classDef" in mmd  # 着色定义
    assert "```" in mmd


def test_render_mermaid_node_limit():
    """节点过多时应被截断到 node_limit。"""
    big_repos = []
    big_contribs = {}
    big_readmes = {}
    for i in range(60):
        fn = f"u{i}/repo{i}"
        big_repos.append({
            "fullname": fn, "identifier": f"repo{i}",
            "author": {"login": f"u{i}"}, "description": "deep learning",
            "language": {"name": "Python"}, "praises_count": 0, "forked_count": 0,
        })
        big_contribs[fn] = [{"login": f"u{i}", "contribution_perc": "100%"}]
        big_readmes[fn] = "deep learning"
    result = G.build_graph(big_repos, big_contribs, {}, big_readmes,
                           keywords=["dl"])
    mmd = G.render_mermaid(result, node_limit=40)
    # mermaid 里出现的节点声明数应 <= 40
    node_lines = [ln for ln in mmd.splitlines() if '["' in ln and "-->" not in ln]
    assert len(node_lines) <= 40


def test_render_dot_header():
    result = _build()
    dot = G.render_dot(result)
    assert dot.startswith("digraph G")
    assert "fillcolor" in dot


def test_render_report_sections():
    result = _build()
    md = G.render_report(result)
    assert "知识图谱报告" in md
    assert "主题热度榜" in md
    assert "核心学者" in md
    assert "deep_learning" in md or "computer_vision" in md


def test_render_report_empty_safe():
    result = {"scenario": "S2", "keywords": [], "nodes": [], "edges": [],
              "core_scholars": [], "core_teams": [], "topic_heat": [],
              "meta": {"repo_count": 0, "node_count": 0, "edge_count": 0,
                       "scholar_count": 0, "topic_count": 0}}
    md = G.render_report(result)
    assert "知识图谱报告" in md
    assert "未识别" in md or "暂无" in md


def test_empty_inputs():
    """空输入不应抛异常。"""
    result = G.build_graph([], {}, {}, {})
    assert result["meta"]["node_count"] == 0
    assert result["meta"]["edge_count"] == 0
    assert result["nodes"] == []


# ---------------------------------------------------------------------------

def _run_all():
    fns = [v for k, v in sorted(globals().items()) if k.startswith("test_")]
    for fn in fns:
        fn()
        print(f"PASS {fn.__name__}")
    print(f"\nAll {len(fns)} graph_build tests passed.")


if __name__ == "__main__":
    _run_all()
