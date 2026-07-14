"""gitlink-kb 单元测试。"""

from __future__ import annotations

import sys
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parent.parent / "scripts"))

import pytest

from kb import (
    split_sections, build_index, search, extract_faq, build_map,
    collect_docs, _tokenize_query,
)

DOC = """# 项目标题

简介段落。

## 安装

使用 pip 安装这个工具。

## 如何配置？

先创建配置文件，然后运行 init。

### 子配置

细节内容。
"""


class TestSplitSections:
    def test_splits_by_heading(self):
        secs = split_sections(DOC)
        titles = [s["title"] for s in secs]
        assert "安装" in titles
        assert "如何配置？" in titles

    def test_level_recorded(self):
        secs = split_sections(DOC)
        sub = next(s for s in secs if s["title"] == "子配置")
        assert sub["level"] == 3

    def test_body_captured(self):
        secs = split_sections(DOC)
        install = next(s for s in secs if s["title"] == "安装")
        assert "pip" in install["body"]


class TestTokenize:
    def test_english(self):
        assert "install" in _tokenize_query("how to install")

    def test_chinese_bigram(self):
        tokens = _tokenize_query("如何安装")
        assert "如何" in tokens or "安装" in tokens


class TestSearch:
    def setup_method(self):
        self.index = build_index([{"path": "README", "content": DOC}])

    def test_finds_install(self):
        results = search(self.index, "安装")
        assert results
        assert any("安装" in r["title"] for r in results)

    def test_english_query(self):
        results = search(self.index, "pip")
        assert results
        assert "pip" in results[0]["snippet"]

    def test_no_match(self):
        assert search(self.index, "zzzznotexist") == []

    def test_empty_query(self):
        assert search(self.index, "") == []

    def test_title_weighted(self):
        # 标题命中应排在前面
        results = search(self.index, "配置")
        assert results
        assert "配置" in results[0]["title"]


class TestFaq:
    def test_extracts_question(self):
        index = build_index([{"path": "README", "content": DOC}])
        faq = extract_faq(index)
        assert any("配置" in f["question"] for f in faq)

    def test_no_question(self):
        index = build_index([{"path": "x", "content": "# Title\n\nbody"}])
        assert extract_faq(index) == []


class TestMap:
    def test_builds_map(self):
        index = build_index([{"path": "README", "content": DOC}])
        doc_map = build_map(index)
        assert "README" in doc_map
        titles = [s["title"] for s in doc_map["README"]]
        assert "安装" in titles


class TestCollectDocs:
    class FakeClient:
        def readme(self, owner, repo, ref):
            return "# README\n\n内容"

        def list_dir(self, owner, repo, path, ref):
            if path == "docs":
                return [{"name": "guide.md", "type": "file", "content": "# 指南\n\n步骤"}]
            return []

        def file_content(self, owner, repo, filepath, ref):
            return None

    def test_collects_readme_and_docs(self):
        docs = collect_docs("o", "r", "master", self.FakeClient())
        paths = {d["path"] for d in docs}
        assert "README" in paths
        assert any("guide" in p for p in paths)


if __name__ == "__main__":
    sys.exit(pytest.main([__file__, "-v"]))
