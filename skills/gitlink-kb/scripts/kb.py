"""gitlink-kb：仓库知识库问答助手。

把一个 GitLink 仓库的文档（README、docs/ 目录、各类 Markdown）索引起来，
支持关键词检索、文档地图生成与 FAQ 提取，让仓库沉淀的知识可被快速查询。

数据来自 GitLink 公开 API（只读），无需登录。

用法：
    python kb.py --owner Gitlink --repo gitlink-cli --query "如何安装"
    python kb.py --owner Gitlink --repo gitlink-cli --map
    python kb.py --owner Gitlink --repo gitlink-cli --faq
"""

from __future__ import annotations

import argparse
import json
import re
import sys
from pathlib import Path
from typing import Any

sys.path.insert(0, str(Path(__file__).resolve().parent))
from glapi import GitLinkClient, GitLinkError, split_owner_repo

# Windows 控制台默认 GBK，直接打印含 emoji 的 Markdown 会抛 UnicodeEncodeError。
# 重配置 stdout 为 UTF-8，确保跨平台正常输出。
if hasattr(sys.stdout, "reconfigure"):
    try:
        sys.stdout.reconfigure(encoding="utf-8")
    except Exception:
        pass

# 文档类文件扩展名
DOC_EXTS = (".md", ".markdown", ".rst", ".txt")
# 优先索引的文档目录
DOC_DIRS = ["", "docs", "doc", ".gitlink", "wiki"]


# ---------------------------------------------------------------------------
# 文档解析
# ---------------------------------------------------------------------------

def split_sections(markdown: str) -> list[dict[str, Any]]:
    """按 Markdown 标题切分为段落，每段含标题、层级、正文。"""
    sections: list[dict[str, Any]] = []
    current = {"title": "（开头）", "level": 0, "lines": []}
    for line in markdown.splitlines():
        m = re.match(r"^(#{1,6})\s+(.*)", line)
        if m:
            if current["lines"] or current["title"] != "（开头）":
                sections.append(current)
            current = {"title": m.group(2).strip(), "level": len(m.group(1)), "lines": []}
        else:
            current["lines"].append(line)
    if current["lines"] or current["title"] != "（开头）":
        sections.append(current)
    for s in sections:
        s["body"] = "\n".join(s["lines"]).strip()
        del s["lines"]
    return sections


def collect_docs(owner: str, repo: str, ref: str,
                 client: GitLinkClient, max_files: int = 20) -> list[dict[str, Any]]:
    """收集仓库中的文档文件及其内容。"""
    docs: list[dict[str, Any]] = []

    # README 优先
    readme = client.readme(owner, repo, ref)
    if readme:
        docs.append({"path": "README", "content": readme})

    seen = {"readme", "readme.md"}
    for d in DOC_DIRS:
        if len(docs) >= max_files:
            break
        try:
            entries = client.list_dir(owner, repo, d, ref)
        except GitLinkError:
            continue
        for e in entries:
            if len(docs) >= max_files:
                break
            name = str(e.get("name", ""))
            if e.get("type") != "file" or not name.lower().endswith(DOC_EXTS):
                continue
            path = f"{d}/{name}" if d else name
            if path.lower() in seen:
                continue
            seen.add(path.lower())
            # entries 里可能已带 content，否则单独取
            content = e.get("content")
            if not content:
                content = client.file_content(owner, repo, path, ref)
            if content:
                docs.append({"path": path, "content": content})
    return docs


def build_index(docs: list[dict[str, Any]]) -> list[dict[str, Any]]:
    """把文档切分为可检索的段落索引。"""
    index: list[dict[str, Any]] = []
    for doc in docs:
        for sec in split_sections(doc["content"]):
            if not sec["body"] and sec["title"] == "（开头）":
                continue
            index.append({
                "doc": doc["path"],
                "title": sec["title"],
                "level": sec["level"],
                "body": sec["body"],
            })
    return index


# ---------------------------------------------------------------------------
# 检索
# ---------------------------------------------------------------------------

def _tokenize_query(query: str) -> list[str]:
    """把查询拆为关键词（英文按词，中文按字/词粗切）。"""
    tokens = re.findall(r"[A-Za-z0-9_]+", query.lower())
    # 中文：粗略按 2-gram 补充
    zh = re.findall(r"[\u4e00-\u9fff]+", query)
    for seg in zh:
        tokens.append(seg)
        for i in range(len(seg) - 1):
            tokens.append(seg[i:i + 2])
    return [t for t in tokens if t]


def search(index: list[dict[str, Any]], query: str, top: int = 5) -> list[dict[str, Any]]:
    """在索引中检索与查询最相关的段落（基于关键词命中计分）。"""
    tokens = _tokenize_query(query)
    if not tokens:
        return []
    scored: list[tuple[int, dict[str, Any]]] = []
    for sec in index:
        haystack = (sec["title"] + "\n" + sec["body"]).lower()
        score = 0
        for t in tokens:
            score += haystack.count(t.lower())
        # 标题命中加权
        title_low = sec["title"].lower()
        for t in tokens:
            if t.lower() in title_low:
                score += 5
        if score > 0:
            scored.append((score, sec))
    scored.sort(key=lambda x: x[0], reverse=True)
    results = []
    for score, sec in scored[:top]:
        snippet = sec["body"][:300].strip()
        results.append({
            "doc": sec["doc"], "title": sec["title"],
            "score": score, "snippet": snippet,
        })
    return results


def extract_faq(index: list[dict[str, Any]]) -> list[dict[str, str]]:
    """从文档中提取 FAQ（标题形似问题，或 Q/问 开头的段落）。"""
    faq: list[dict[str, str]] = []
    for sec in index:
        title = sec["title"]
        is_question = (
            "?" in title or "？" in title
            or title.lower().startswith(("q:", "q ", "how", "what", "why", "when", "如何", "怎么", "为什么", "是否"))
        )
        if is_question and sec["body"]:
            faq.append({"question": title, "answer": sec["body"][:400].strip(), "doc": sec["doc"]})
    return faq


def build_map(index: list[dict[str, Any]]) -> dict[str, list[dict[str, Any]]]:
    """按文档归类标题，生成文档地图。"""
    doc_map: dict[str, list[dict[str, Any]]] = {}
    for sec in index:
        if sec["title"] == "（开头）":
            continue
        doc_map.setdefault(sec["doc"], []).append(
            {"title": sec["title"], "level": sec["level"]})
    return doc_map


# ---------------------------------------------------------------------------
# 渲染
# ---------------------------------------------------------------------------

def render_search(owner: str, repo: str, query: str,
                  results: list[dict[str, Any]]) -> str:
    lines = [f"# 知识库检索 — {owner}/{repo}", "", f"查询：**{query}**", ""]
    if not results:
        lines += ["未找到相关内容。建议换个关键词，或确认仓库是否有相关文档。", ""]
        return "\n".join(lines)
    for i, r in enumerate(results, 1):
        lines += [
            f"## {i}. {r['title']}　`{r['doc']}`（相关度 {r['score']}）",
            "",
            r["snippet"] + ("…" if len(r["snippet"]) >= 300 else ""),
            "",
        ]
    return "\n".join(lines)


def render_map(owner: str, repo: str, doc_map: dict[str, list[dict[str, Any]]]) -> str:
    lines = [f"# 文档地图 — {owner}/{repo}", "",
             f"共索引 {len(doc_map)} 个文档。", ""]
    for doc, secs in doc_map.items():
        lines.append(f"## 📄 {doc}")
        lines.append("")
        for s in secs:
            indent = "  " * max(0, s["level"] - 1)
            lines.append(f"{indent}- {s['title']}")
        lines.append("")
    return "\n".join(lines)


def render_faq(owner: str, repo: str, faq: list[dict[str, str]]) -> str:
    lines = [f"# 常见问题（FAQ）— {owner}/{repo}", ""]
    if not faq:
        lines += ["未从文档中识别出 FAQ 条目。", ""]
        return "\n".join(lines)
    for item in faq:
        lines += [f"### ❓ {item['question']}", "", item["answer"], "",
                  f"<sub>来源：{item['doc']}</sub>", ""]
    return "\n".join(lines)


def main(argv: list[str] | None = None) -> int:
    p = argparse.ArgumentParser(prog="gitlink-kb", description="仓库知识库问答助手")
    p.add_argument("--owner", help="仓库所有者")
    p.add_argument("--repo", help="仓库名称")
    p.add_argument("--slug", help="owner/repo 或完整 URL")
    p.add_argument("--ref", default="master", help="分支或标签，默认 master")
    p.add_argument("--query", help="检索关键词/问题")
    p.add_argument("--map", action="store_true", help="生成文档地图")
    p.add_argument("--faq", action="store_true", help="提取 FAQ")
    p.add_argument("--max-files", type=int, default=20, help="最多索引的文档数")
    p.add_argument("--format", choices=["markdown", "json"], default="markdown")
    p.add_argument("--output", type=Path, help="输出文件")
    args = p.parse_args(argv)

    if args.slug:
        owner, repo = split_owner_repo(args.slug)
    elif args.owner and args.repo:
        owner, repo = args.owner, args.repo
    else:
        print("错误：请用 --owner/--repo 或 --slug 指定仓库。", file=sys.stderr)
        return 2

    client = GitLinkClient()
    try:
        docs = collect_docs(owner, repo, args.ref, client, max_files=args.max_files)
        index = build_index(docs)
    except GitLinkError as exc:
        print(f"采集失败：{exc}", file=sys.stderr)
        return 1

    if args.query:
        results = search(index, args.query)
        out = (json.dumps({"query": args.query, "results": results}, ensure_ascii=False, indent=2)
               if args.format == "json" else render_search(owner, repo, args.query, results))
    elif args.map:
        doc_map = build_map(index)
        out = (json.dumps(doc_map, ensure_ascii=False, indent=2)
               if args.format == "json" else render_map(owner, repo, doc_map))
    elif args.faq:
        faq = extract_faq(index)
        out = (json.dumps({"faq": faq}, ensure_ascii=False, indent=2)
               if args.format == "json" else render_faq(owner, repo, faq))
    else:
        # 默认输出索引概况
        summary = {"owner": owner, "repo": repo,
                   "indexed_docs": len({s["doc"] for s in index}),
                   "sections": len(index)}
        out = (json.dumps(summary, ensure_ascii=False, indent=2)
               if args.format == "json"
               else f"# 知识库索引 — {owner}/{repo}\n\n已索引 {summary['indexed_docs']} 个文档、"
                    f"{summary['sections']} 个段落。\n\n用 `--query <问题>` 检索、`--map` 看文档地图、`--faq` 提取常见问题。")

    if args.output:
        args.output.parent.mkdir(parents=True, exist_ok=True)
        args.output.write_text(out, encoding="utf-8")
        print(f"已写入 {args.output}")
    else:
        print(out)
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
