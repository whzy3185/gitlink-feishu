"""
S1 仓库级科研项目谱系分析

分析维度: 提交时间线 / PR 演进模式 / 文档演化 / 创新点识别 / 分支地图
所有 GitLink 操作经 gitlink-cli，禁止用 gh/glab。统一使用 main 分支。

用法:
  python lineage.py --owner mindspore-Ecosystem --repo mindspore --out ./out
"""
from __future__ import annotations

import argparse
import json
import os
import re
import sys
from collections import Counter
from typing import Any

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
import collect as c  # noqa: E402

# ---------------------------------------------------------------------------
# 提交时间解析（commit timestamp 可能是 ISO 字符串或整数秒）
# ---------------------------------------------------------------------------

def _to_epoch(v: Any) -> float:
    """把 GitLink commit 的 timestamp（ISO 字符串或整数秒）统一解析为 epoch 秒。

    失败返回 0.0。
    """
    if v is None:
        return 0.0
    if isinstance(v, (int, float)):
        # 毫秒级时间戳兜底（GitLink 多为秒）
        return float(v) / 1000.0 if v > 1e12 else float(v)
    s = str(v).strip()
    if not s:
        return 0.0
    # 纯数字串
    if s.isdigit():
        f = float(s)
        return f / 1000.0 if f > 1e12 else f
    # ISO 8601：'2024-05-01T08:00:00Z' / '2024-05-01 08:00:00'
    s = s.replace("Z", "+00:00")
    try:
        import datetime as _dt
        return _dt.datetime.fromisoformat(s).timestamp()
    except (ValueError, TypeError):
        # 退而求其次：抽首个 YYYY-MM-DD
        m = re.search(r"(\d{4})-(\d{2})-(\d{2})", str(v))
        if m:
            try:
                import datetime as _dt
                return _dt.datetime(int(m.group(1)), int(m.group(2)),
                                    int(m.group(3))).timestamp()
            except ValueError:
                return 0.0
        return 0.0


def _iso_date(epoch: float) -> str:
    """epoch 秒 → 'YYYY-MM-DD' 字符串（0 → ''）。"""
    if not epoch:
        return ""
    import datetime as _dt
    try:
        return _dt.datetime.utcfromtimestamp(epoch).strftime("%Y-%m-%d")
    except (OSError, ValueError, OverflowError):
        return ""


# ---------------------------------------------------------------------------
# 分类器：实验/评测文件 vs 文档文件
# ---------------------------------------------------------------------------

# 命中即判为「实验/评测/数据」类文件（科研产物信号）。
# 匹配「目录段或文件名」：以 experiment/benchmark/eval/test(s)/data 开头
# （后接 [\w-]* 续写，或紧接分隔符/串尾）。
_EXPERIMENT_RE = re.compile(
    r"(^|[_/\-])(experiment[\w-]*|benchmark[\w-]*|eval[\w-]*|tests?|data)([/\-\._]|$)",
    re.IGNORECASE,
)


def is_experiment_file(path: str) -> bool:
    """判断文件路径是否属于「实验/评测/数据」类（科研产物）。

    命中 experiment*/benchmark*/eval*/tests?/data/ 任意一段（作为目录名或文件名前缀）
    返回 True。例如：
      - experiments/run.py, experiment_train.py, benchmark/eval.py
      - tests/test_x.py, data/dataset.csv, src/benchmark_infer.py
    """
    if not path:
        return False
    return bool(_EXPERIMENT_RE.search(path))


def is_doc_file(path: str) -> bool:
    """判断文件路径是否属于文档类：*.md（任意位置）或 docs/ 下任意文件。"""
    if not path:
        return False
    low = path.lower()
    if low.endswith(".md"):
        return True
    return low.startswith("docs/") or ("/docs/" in low)


# ---------------------------------------------------------------------------
# 分支地图（单分支简化：默认分支为唯一分支）
# ---------------------------------------------------------------------------

def build_branch_map(commits: list[dict], default_branch: str) -> list[dict]:
    """构建分支活跃度地图。本场景单分支简化：默认分支为唯一分支。

    返回 [{name, commits, last_active, is_default}]。
    """
    n = len(commits) if isinstance(commits, list) else 0
    last = 0.0
    for cm in commits or []:
        t = _to_epoch(cm.get("timestamp"))
        if t and t > last:
            last = t
    return [{
        "name": default_branch or "master",
        "commits": n,
        "last_active": _iso_date(last),
        "is_default": True,
    }]


# ---------------------------------------------------------------------------
# 提交时间线（按周/按日聚合）
# ---------------------------------------------------------------------------

def commit_timeline(commits: list[dict], bucket: str = "day") -> list[dict]:
    """把提交按日期聚合为时间线，返回按日期升序的 [{date, count}]。

    bucket ∈ {'day','week'}；week 以 ISO 年-周 表示。
    """
    counter: Counter = Counter()
    for cm in commits or []:
        t = _to_epoch(cm.get("timestamp"))
        if not t:
            continue
        if bucket == "week":
            import datetime as _dt
            iso = _dt.datetime.utcfromtimestamp(t).isocalendar()
            key = f"{iso[0]}-W{iso[1]:02d}"
        else:
            key = _iso_date(t)
        counter[key] += 1
    return [{"date": k, "count": counter[k]} for k in sorted(counter)]


# ---------------------------------------------------------------------------
# 合并 PR 的演进模式
# ---------------------------------------------------------------------------

def pr_merge_patterns(merged_prs: list[dict]) -> list[dict]:
    """从已合并 PR 抽取演进模式。

    返回 [{number, title, status, merged_time, changed_files}]。
    changed_files 取 pr 提供的 changed_files / changedFiles / additions/deletions 的近似。
    """
    out: list[dict] = []
    for pr in merged_prs or []:
        if not isinstance(pr, dict):
            continue
        # 时间：优先 merged 时间字段，否则 pr_created_unix
        merged_t = (pr.get("pr_merged_unix") or pr.get("merged_at")
                    or pr.get("pr_updated_unix") or pr.get("pr_created_unix"))
        # 文件改动数：优先详情接口的 files_count（列表 API 不返回）
        changed = (pr.get("files_count") or pr.get("changed_files")
                   or pr.get("changedFiles") or pr.get("file_nums") or 0)
        # 兜底：用 additions/deletions 之和近似
        if not changed:
            changed = c.as_int(pr.get("additions"), 0) + c.as_int(pr.get("deletions"), 0)
        # 标题只取首行 + 截断到 80 字（有些 PR 创建时把正文塞进了 title 字段）
        raw_title = c.as_str(pr.get("title")).split("\n", 1)[0].strip()
        title = raw_title[:80]
        out.append({
            "number": pr.get("index") or pr.get("number") or pr.get("id"),
            "title": title,
            "status": pr.get("status"),
            "merged_time": _iso_date(_to_epoch(merged_t)),
            "changed_files": c.as_int(changed),
        })
    # 按合并时间升序（空时间排末尾）
    out.sort(key=lambda x: (x["merged_time"] == "", x["merged_time"]))
    return out


# ---------------------------------------------------------------------------
# 文档演进（docs/*.md 的近似最后修改信息）
# ---------------------------------------------------------------------------

def doc_evolution(tree_entries: list[dict]) -> list[dict]:
    """从仓库树抽取 docs/ 下的文档清单，近似其最后修改日期。

    tree 来自 c.tree(owner, repo, path='docs')。每条 entry 形如
    {name, path, type, ...}。最后修改日期在树端点通常不可得，用文件名中的
    日期或留空（report 中标注「近似」）。
    返回 [{file, last_date}]。
    """
    out: list[dict] = []
    for e in tree_entries or []:
        if not isinstance(e, dict):
            continue
        name = c.as_str(e.get("name") or e.get("path"))
        path = c.as_str(e.get("path") or name)
        if not is_doc_file(path):
            continue
        last = c.as_str(e.get("last_commit") or e.get("commit_date")
                        or e.get("date"))
        if not last:
            m = re.search(r"(\d{4})-(\d{2})-(\d{2})", path)
            last = m.group(0) if m else ""
        out.append({"file": name, "last_date": last})
    out.sort(key=lambda x: (x["last_date"] == "", x["file"]))
    return out


# ---------------------------------------------------------------------------
# 创新点识别（高影响合并）
# ---------------------------------------------------------------------------

def innovation_points(merged_prs: list[dict], commits: list[dict],
                      top: int = 8) -> list[dict]:
    """识别高影响合并作为项目的创新/里程碑点。

    判据（任一）：
      - 改动文件数高（>= 中位数的 1.5 倍，或绝对值 >= 10）→ 「大规模重构/新特性」
      - 标题含里程碑关键词（add/implement/feature/release/benchmark/...）→ 「特性引入」
    返回 [{description, evidence, category}]，按影响度降序取前 top。
    """
    patterns = pr_merge_patterns(merged_prs)
    if not patterns:
        return []

    files = [p["changed_files"] for p in patterns if p["changed_files"] > 0]
    median = sorted(files)[len(files) // 2] if files else 0

    MILESTONE_RE = re.compile(
        r"(add|implement|support|feature|release|benchmark|refactor|"
        r"experiment|dataset|train|inference|v\d+\.\d+)", re.IGNORECASE)

    scored: list[tuple[float, dict]] = []
    for p in patterns:
        cf = p["changed_files"]
        title = p["title"]
        category = ""
        impact = float(cf)
        if cf >= 10 or (median and cf >= median * 1.5):
            category = "大规模重构/新特性"
            impact += 10
        if MILESTONE_RE.search(title):
            category = category or "特性引入"
            impact += 5
        if not category:
            continue
        evidence = (f"PR #{p['number']} 「{title[:48]}」 "
                    f"改动 {cf} 文件，合并于 {p['merged_time'] or '未知时间'}")
        scored.append((impact, {
            "description": title.strip() or f"PR #{p['number']}",
            "evidence": evidence,
            "category": category,
        }))
    scored.sort(key=lambda x: -x[0])
    return [item for _, item in scored[:top]]


# ---------------------------------------------------------------------------
# 主流程：取数 + 算法
# ---------------------------------------------------------------------------

def lineage(owner: str, repo: str, branches_limit: int = 5) -> dict[str, Any]:
    """取数 + 分析，返回完整 lineage 结果 dict。"""
    info = c.repo_info(owner, repo)
    default_branch = "main"  # 统一使用 main 分支分析

    commits = c.commits(owner, repo, ref=default_branch, max_pages=10, page_size=100)
    # 已合并 PR：state=merged（collect 透传）
    merged_prs = c.prs(owner, repo, state="merged", max_pages=10, page_size=50)
    # 列表 API 不返回文件改动数 → 逐个取详情补 files_count（限速 + 上限 30 个，控 API 调用）
    for pr in merged_prs[:30]:
        idx = pr.get("index") or pr.get("number") or pr.get("id")
        if idx is None:
            continue
        try:
            det = c.pr_detail(owner, repo, int(idx))
        except (TypeError, ValueError):
            det = {}
        if det:
            if det.get("files_count") is not None:
                pr["files_count"] = det.get("files_count")
            if det.get("commits_count") is not None:
                pr["commits_count"] = det.get("commits_count")
    tree_root = c.tree(owner, repo, ref=default_branch)
    docs_tree = c.tree(owner, repo, path="docs", ref=default_branch)
    _readme = c.readme(owner, repo, ref=default_branch)

    # 扫描整棵树，挑出实验/评测文件
    exp_files: list[str] = []
    all_tree = (tree_root or []) + (docs_tree or [])
    for e in all_tree:
        if not isinstance(e, dict):
            continue
        p = c.as_str(e.get("path") or e.get("name"))
        if p and is_experiment_file(p):
            exp_files.append(p)

    timeline = commit_timeline(commits, bucket="day")
    branch_map = build_branch_map(commits, default_branch)[:branches_limit]
    pr_patterns = pr_merge_patterns(merged_prs)
    docs = doc_evolution(docs_tree if docs_tree else tree_root)
    innovations = innovation_points(merged_prs, commits)

    return {
        "scenario": "S1_repository_research_insight",
        "repo": f"{owner}/{repo}",
        "default_branch": default_branch,
        "commit_timeline": timeline,
        "branch_map": branch_map,
        "pr_merge_patterns": pr_patterns,
        "doc_evolution": docs,
        "experiment_files": sorted(set(exp_files)),
        "innovation_points": innovations,
        "meta": {
            "commit_count": len(commits),
            "merged_pr_count": len(merged_prs),
            "doc_count": len(docs),
            "experiment_file_count": len(exp_files),
        },
    }


# ---------------------------------------------------------------------------
# 渲染：Markdown 报告 + Mermaid gitGraph
# ---------------------------------------------------------------------------

def render_report(result: dict[str, Any]) -> str:
    repo = result["repo"]
    meta = result["meta"]
    lines = [
        f"# 仓库级科研项目洞悉报告 — {repo}\n",
        f"> 场景 S1 · 子赛题四「应用 GitLink 辅助科研」· 项目谱系（lineage）分析\n",
        "## 一、基础信息\n",
        f"- **默认分支**: `{result['default_branch']}`",
        f"- **采样提交**: {meta['commit_count']} 条（默认分支，最多 10×100）",
        f"- **已合并 PR**: {meta['merged_pr_count']} 个",
        f"- **文档文件**: {meta['doc_count']} 个",
        f"- **实验/评测文件**: {meta['experiment_file_count']} 个\n",
    ]

    lines.append("## 二、提交活跃度时间线\n")
    tl = result["commit_timeline"]
    if tl:
        peak = max(tl, key=lambda x: x["count"])
        lines.append(f"- 时间跨度: {tl[0]['date']} → {tl[-1]['date']}"
                     f"（共 {len(tl)} 个有提交的日期）")
        lines.append(f"- 峰值: {peak['date']} 当日 {peak['count']} 次提交\n")
    else:
        lines.append("- （未取到提交时间线）\n")

    lines.append("## 三、分支地图\n")
    lines.append("| 分支 | 提交数 | 最后活跃 | 是否默认 |")
    lines.append("|------|:------:|----------|:--------:|")
    for b in result["branch_map"]:
        lines.append(f"| `{b['name']}` | {b['commits']} | {b['last_active'] or '—'} |"
                     f" {'是' if b['is_default'] else '否'} |")
    lines.append("")

    lines.append("## 四、合并 PR 演进模式（高影响合并预览）\n")
    prs = result["pr_merge_patterns"]
    if prs:
        lines.append("| PR | 标题 | 改动文件 | 合并时间 |")
        lines.append("|----|------|:--------:|----------|")
        for p in prs[:10]:
            lines.append(f"| #{p['number']} | {p['title']} | "
                         f"{p['changed_files']} | {p['merged_time'] or '—'} |")
    else:
        lines.append("- （无已合并 PR）")
    lines.append("")

    lines.append("## 五、创新/里程碑点\n")
    inno = result["innovation_points"]
    if inno:
        for i, it in enumerate(inno, 1):
            lines.append(f"{i}. **[{it['category']}]** {it['description']}")
            lines.append(f"   - 证据: {it['evidence']}")
    else:
        lines.append("- （未识别到明显高影响合并）")
    lines.append("")

    lines.append("## 六、文档演进（docs/*）\n")
    docs = result["doc_evolution"]
    if docs:
        lines.append("| 文档 | 近似最后日期 |")
        lines.append("|------|--------------|")
        for d in docs[:15]:
            lines.append(f"| {d['file']} | {d['last_date'] or '—'} |")
    else:
        lines.append("- （docs/ 下无文档或树不可得）")
    lines.append("")

    lines.append("## 七、实验/评测文件组织\n")
    exps = result["experiment_files"]
    if exps:
        for p in exps[:20]:
            lines.append(f"- `{p}`")
        if len(exps) > 20:
            lines.append(f"- ...（共 {len(exps)} 个，此处仅列前 20）")
    else:
        lines.append("- （未在仓库树中识别到 experiment/benchmark/eval/test/data 目录）")
    lines.append("")
    return "\n".join(lines)


def render_mermaid(result: dict[str, Any]) -> str:
    """渲染 Mermaid gitGraph：用 commit 链展示项目演进。有 PR 时用 branch→merge，
    无 PR 时回退到按 commits 简化展示。"""
    lines = ["```mermaid", "gitGraph"]
    branch_name = result.get("default_branch", "main")
    lines.append(f"  commit id: \"{branch_name} 起点\"")
    prs = result.get("pr_merge_patterns") or []
    inno = result.get("innovation_points") or []
    inno_nums = set()
    for it in inno:
        ev = it.get("evidence", "")
        m = re.search(r"PR #(\d+)", ev)
        if m:
            inno_nums.add(int(m.group(1)))

    if prs:
        # 有 PR：每个 PR 作为 feature 分支 → merge 回主线
        for i, p in enumerate(prs[:15]):
            pnum = p.get("number") or i + 1
            title = (p.get("title") or "")[:24]
            is_innov = pnum in inno_nums
            tag = " ✨创新" if is_innov else ""
            safe_branch = f"pr{pnum}"
            lines.append(f"  branch {safe_branch}")
            lines.append(f"  checkout {safe_branch}")
            lines.append(f"  commit id: \"#{pnum}{tag}: {title}\"")
            lines.append(f"  checkout {branch_name}")
            lines.append(f"  merge {safe_branch}")
    else:
        # 无 PR 数据：用 commit 时间线简化展示
        timeline = result.get("commit_timeline") or []
        shown = 0
        for entry in timeline:
            if shown >= 12:
                break
            if entry.get("count", 0) > 0:
                lines.append(f"  commit id: \"{entry['date']} ({entry['count']} commits)\"")
                shown += 1
        if shown == 0:
            lines.append("  commit id: \"(暂无提交数据)\"")

    lines.append(f"  commit id: \"HEAD\"")
    lines.append("```")
    return "\n".join(lines)


# ---------------------------------------------------------------------------

def main():
    ap = argparse.ArgumentParser(description="S1 仓库级科研项目洞悉（lineage 谱系分析）")
    ap.add_argument("--owner", required=True)
    ap.add_argument("--repo", required=True)
    ap.add_argument("--branches-limit", type=int, default=5,
                    help="分支地图上限（本场景单分支简化，默认 5）")
    ap.add_argument("--out", help="输出目录（写 lineage.json/report.md/branch_graph.mmd）；"
                                  "省略则打印 JSON")
    args = ap.parse_args()

    result = lineage(args.owner, args.repo, branches_limit=args.branches_limit)

    if args.out:
        os.makedirs(args.out, exist_ok=True)
        with open(os.path.join(args.out, "lineage.json"), "w", encoding="utf-8") as f:
            json.dump(result, f, ensure_ascii=False, indent=2)
        with open(os.path.join(args.out, "report.md"), "w", encoding="utf-8") as f:
            f.write(render_report(result))
        with open(os.path.join(args.out, "branch_graph.mmd"), "w", encoding="utf-8") as f:
            f.write(render_mermaid(result))
        print(f"✓ S1 项目洞悉完成 → {args.out}/lineage.json | report.md | branch_graph.mmd")
        print(f"  提交 {result['meta']['commit_count']} 条 | "
              f"合并 PR {result['meta']['merged_pr_count']} 个 | "
              f"创新点 {len(result['innovation_points'])} 个")
    else:
        print(json.dumps(result, ensure_ascii=False, indent=2))


if __name__ == "__main__":
    main()
