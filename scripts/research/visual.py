"""
S6 科研成果可视化沉淀 —— 生成交互式 Plotly HTML 报告

图表: 开发时间线(commits+PRs 双线)、贡献者热力图(有数据才渲染)、
      语言占比饼图(<3% 合并为"其他")、里程碑甘特图(<2条任务跳过)
自适应布局: 空白/无效图表自动隐藏，仅展示有效数据

用法:
  python visual.py --owner mindspore-Ecosystem --repo mindspore --weeks 26 --out ./out
"""
from __future__ import annotations

import argparse
import json
import os
import re
import sys
from datetime import datetime, timezone
from typing import Any

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
import collect as c  # noqa: E402

# ---------------------------------------------------------------------------
# 时间解析工具
# ---------------------------------------------------------------------------

def _to_timestamp(value: Any) -> float:
    """把 GitLink 多种时间表示统一成 epoch 秒。

    支持整数秒（pr_created_unix）、ISO 字符串（created_at / timestamp 字符串）。
    无法解析返回 0.0（最远古时间，会被周分桶丢弃到「太早」一端）。
    """
    if value is None:
        return 0.0
    # 整数秒（commit.timestamp 形如 "1719500000" 也可走这里）
    if isinstance(value, (int, float)):
        f = float(value)
        # 毫秒级时间戳兜底（13 位）
        return f / 1000.0 if f > 1e12 else f
    s = str(value).strip()
    if not s:
        return 0.0
    # 纯数字字符串
    if re.fullmatch(r"\d+(\.\d+)?", s):
        f = float(s)
        return f / 1000.0 if f > 1e12 else f
    # ISO 8601（兼容带/不带 Z、带毫秒、带时区偏移）
    txt = s.replace("Z", "+00:00")
    fmts = ("%Y-%m-%dT%H:%M:%S%z",
            "%Y-%m-%dT%H:%M:%S.%f%z",
            "%Y-%m-%d %H:%M:%S",
            "%Y-%m-%d")
    for fmt in fmts:
        try:
            dt = datetime.strptime(txt, fmt)
            if dt.tzinfo is None:
                dt = dt.replace(tzinfo=timezone.utc)
            return dt.timestamp()
        except ValueError:
            continue
    return 0.0


def _commit_time(commit: dict) -> float:
    """提交对象取时间戳：优先 timestamp（字符串），退而 author.committed_unix。"""
    ts = commit.get("timestamp")
    if ts is not None:
        return _to_timestamp(ts)
    auth = commit.get("author") or {}
    if isinstance(auth, dict):
        for k in ("committed_unix", "committed_at", "time", "date"):
            if auth.get(k) is not None:
                return _to_timestamp(auth.get(k))
    return 0.0


def _issue_time(issue: dict) -> float:
    return _to_timestamp(issue.get("created_at"))


def _pr_time(pr: dict) -> float:
    return _to_timestamp(pr.get("pr_created_unix") or pr.get("created_at"))


# ---------------------------------------------------------------------------
# 算法 1：按周分桶
# ---------------------------------------------------------------------------

def bin_weekly(items: list, weeks: int, time_getter) -> dict[str, list]:
    """把带时间戳的对象按「最近 weeks 周」分桶（含本周在内的 weeks 个连续周桶）。

    Args:
        items: 待分桶对象列表。
        weeks: 保留最近多少个周桶。
        time_getter: 从单个对象取 epoch 秒的函数。

    返回 ``{"labels": [...], "counts": [...]}``：
    - labels[i] 形如 "2026-W13"（ISO 周标签），从最近一周倒序到最老一周。
    - counts[i] 为该周命中数；时间非法或越界（早于窗口左端）的对象不计入。
    """
    weeks = max(1, int(weeks))
    now = datetime.now(timezone.utc)
    # 右端边界对齐到「下周一 00:00 UTC」（不含），使窗口包含当前周在内共 weeks 周。
    # 若右端用「本周一」，当前周会被整体排除，最近一周的数据被吞掉。
    today = now.replace(hour=0, minute=0, second=0, microsecond=0)
    # Monday=0 .. Sunday=6；toordinal - weekday = 本周一，+7 = 下周一
    next_monday = today.fromordinal(today.toordinal() - today.weekday() + 7).replace(tzinfo=timezone.utc)
    end_ts = next_monday.timestamp()
    start_ts = end_ts - weeks * 7 * 86400

    counts = [0] * weeks
    iso_labels: list[str] = []
    for i in range(weeks):
        # 桶 i 的周一 = end - (weeks-1-i) 周
        bucket_monday_ts = start_ts + i * 7 * 86400
        bucket_monday = datetime.fromtimestamp(bucket_monday_ts, tz=timezone.utc)
        iso_year, iso_week, _ = bucket_monday.isocalendar()
        iso_labels.append(f"{iso_year}-W{iso_week:02d}")

    for item in items:
        ts = time_getter(item)
        if ts <= 0:
            continue
        if ts < start_ts or ts >= end_ts:
            continue
        offset = ts - start_ts
        idx = int(offset // (7 * 86400))
        if 0 <= idx < weeks:
            counts[idx] += 1

    return {"labels": iso_labels, "counts": counts}


# ---------------------------------------------------------------------------
# 算法 2：贡献者 × 周热力矩阵
# ---------------------------------------------------------------------------

def contribution_heatmap(contributors: list, commits: list, weeks: int,
                         top_users: int = 12) -> dict[str, list]:
    """构建 top 贡献者 × 周桶的提交数矩阵。

    Args:
        contributors: GitLink contributors[]（用于排序与展示名）。
        commits: GitLink commits[]（含 author.login）。
        weeks: 周桶数（与 bin_weekly 同口径）。
        top_users: 矩阵最多保留多少个贡献者（按贡献数 desc）。

    返回 ``{"users": [login...], "weeks": [label...], "matrix": [[cnt...]]}``：
    matrix[user_i][week_j] = 该用户在该周的提交数。无 contributor 信息时也按 commit 作者聚合。
    """
    weeks = max(1, int(weeks))
    # 用 bin_weekly 的同口径周标签（取贡献数排序后的 login 列表）
    window = bin_weekly(commits, weeks, _commit_time)
    week_labels = window["labels"]
    now = datetime.now(timezone.utc)
    end = now.replace(hour=0, minute=0, second=0, microsecond=0)
    end = end.fromordinal(end.toordinal() - end.weekday()).replace(tzinfo=timezone.utc)
    start_ts = end.timestamp() - weeks * 7 * 86400

    # 候选用户顺序：contributors（按 contributions desc）+ 提交里出现但不在 contributors 的作者
    ordered: list[str] = []
    seen: set[str] = set()
    for contrib in contributors or []:
        login = c.login_of(contrib) or ""
        if login and login not in seen:
            ordered.append(login)
            seen.add(login)
    for cm in commits or []:
        auth = cm.get("author") or {}
        login = c.login_of(auth) if isinstance(auth, dict) else ""
        if login and login not in seen:
            ordered.append(login)
            seen.add(login)

    users = ordered[:top_users]
    matrix = [[0] * weeks for _ in users]
    user_idx = {u: i for i, u in enumerate(users)}

    for cm in commits or []:
        ts = _commit_time(cm)
        if ts <= 0 or ts < start_ts or ts >= end.timestamp():
            continue
        auth = cm.get("author") or {}
        login = c.login_of(auth) if isinstance(auth, dict) else ""
        if not login or login not in user_idx:
            continue
        offset = ts - start_ts
        idx = int(offset // (7 * 86400))
        if 0 <= idx < weeks:
            matrix[user_idx[login]][idx] += 1

    return {"users": users, "weeks": week_labels, "matrix": matrix}


# ---------------------------------------------------------------------------
# 算法 3：论文引用链接抽取
# ---------------------------------------------------------------------------

# arXiv: arxiv.org/abs/2401.00012 / arxiv.org/pdf/... / arxiv:2401.00012
_ARXIV_RE = re.compile(
    r"(?:https?://)?(?:www\.)?arxiv\.org/(?:abs|pdf)/(\d{4}\.\d{4,5})(?:v\d+)?(?:\.pdf)?",
    re.IGNORECASE,
)
_ARXIV_BARE_RE = re.compile(r"\barXiv:\s*(\d{4}\.\d{4,5})", re.IGNORECASE)
# DOI: doi.org/10.xxxx/... 或裸 10.xxxx/...（论文里的 DOI 形式）
_DOI_URL_RE = re.compile(
    r"(?:https?://)?(?:dx\.)?doi\.org/(10\.\d{4,9}/[^\s)\"'<>]+)", re.IGNORECASE,
)
_DOI_BARE_RE = re.compile(
    r"\b(10\.\d{4,9}/[^\s)\"'<>]+)", re.IGNORECASE,
)

# 在 DOI 字符串里清掉常见尾部分隔符（避免吃进句号、逗号）
_TRAILING_PUNCT = ".,;:)\"'>]"


def _clean_doi(doi: str) -> str:
    return doi.rstrip(_TRAILING_PUNCT)


def _snippet(text: str, pos: int, span: int = 60) -> str:
    """以匹配位置为中心截一段上下文。"""
    a = max(0, pos - span // 2)
    b = min(len(text), pos + span // 2)
    frag = text[a:b].replace("\n", " ").strip()
    return ("…" + frag) if a > 0 else frag


def extract_paper_links(text: str) -> list[dict[str, str]]:
    """从文本里抽取 arXiv 与 DOI 论文引用链接。

    返回 ``[{"source_text_snippet": str, "target": url, "type": "arxiv"|"doi"}]``，
    按 (出现位置, 类型优先 arxiv) 排序，去重（同一 arxiv id / doi 只保留首次）。
    """
    if not text:
        return []
    out: list[dict[str, str]] = []
    seen_arxiv: set[str] = set()
    seen_doi: set[str] = set()
    hits: list[tuple[int, dict[str, str]]] = []

    for m in _ARXIV_RE.finditer(text):
        aid = m.group(1)
        if aid in seen_arxiv:
            continue
        seen_arxiv.add(aid)
        hits.append((m.start(), {
            "source_text_snippet": _snippet(text, m.start()),
            "target": f"https://arxiv.org/abs/{aid}",
            "type": "arxiv",
        }))
    for m in _ARXIV_BARE_RE.finditer(text):
        aid = m.group(1)
        if aid in seen_arxiv:
            continue
        seen_arxiv.add(aid)
        hits.append((m.start(), {
            "source_text_snippet": _snippet(text, m.start()),
            "target": f"https://arxiv.org/abs/{aid}",
            "type": "arxiv",
        }))
    for m in _DOI_URL_RE.finditer(text):
        doi = _clean_doi(m.group(1))
        if doi.lower() in seen_doi:
            continue
        seen_doi.add(doi.lower())
        hits.append((m.start(), {
            "source_text_snippet": _snippet(text, m.start()),
            "target": f"https://doi.org/{doi}",
            "type": "doi",
        }))
    for m in _DOI_BARE_RE.finditer(text):
        doi = _clean_doi(m.group(1))
        if doi.lower() in seen_doi:
            continue
        seen_doi.add(doi.lower())
        hits.append((m.start(), {
            "source_text_snippet": _snippet(text, m.start()),
            "target": f"https://doi.org/{doi}",
            "type": "doi",
        }))

    hits.sort(key=lambda x: (x[0], 0 if x[1]["type"] == "arxiv" else 1))
    return [h[1] for h in hits]


# ---------------------------------------------------------------------------
# 算法 4：仓库产物分类
# ---------------------------------------------------------------------------

def classify_artifacts(tree_entries: list) -> list[dict[str, Any]]:
    """按路径把仓库文件/目录归入科研产物类别。

    规则（按优先级，先匹配先归类）：
      - path 含 ``benchmark/`` 段        → benchmark
      - path 含 ``model/`` 段或 *.ckpt/*.safetensors/*.onnx → model
      - path 含 ``data/`` 段或 *.csv/*.parquet → dataset
      - *.pdf / *.ipynb / paper 关键词   → paper

    每个产物 ``{"path", "category", "name"}``。tree_entries 既可能是文件列表
    （含 path/name/type）也可能是目录项；本函数尽力取 path/name 字段。
    """
    out: list[dict[str, Any]] = []
    seen: set[str] = set()
    for entry in tree_entries or []:
        if not isinstance(entry, dict):
            continue
        path = entry.get("path") or entry.get("name") or ""
        if not path:
            continue
        norm = path.replace("\\", "/").lower()
        name = norm.rsplit("/", 1)[-1]
        category = None
        # benchmark（必须含 benchmark 目录段，避免误把文件名含词的归入）
        if "/benchmark/" in norm or norm.startswith("benchmark/"):
            category = "benchmark"
        elif "/model/" in norm or norm.startswith("model/") or name.endswith(
                (".ckpt", ".safetensors", ".onnx", ".pb", ".h5", ".pt")):
            category = "model"
        elif "/data/" in norm or norm.startswith("data/") or name.endswith(
                (".csv", ".parquet", ".npy", ".npz", ".hdf5", ".h5")):
            # .h5 已先被 model 吃掉，这里主要 csv/parquet/npy
            category = "dataset"
        elif (name.endswith(".pdf") or name.endswith(".ipynb")
              or "paper" in norm or "arxiv" in norm):
            category = "paper"
        if category and path not in seen:
            seen.add(path)
            out.append({"path": path, "category": category,
                        "name": name or path.rsplit("/", 1)[-1]})
    return out


def artifact_summary(artifacts: list[dict[str, Any]]) -> dict[str, int]:
    """统计各类产物数量，返回 {paper: n, dataset: n, model: n, benchmark: n}。"""
    summary: dict[str, int] = {"paper": 0, "dataset": 0, "model": 0, "benchmark": 0}
    for a in artifacts or []:
        cat = a.get("category")
        if cat in summary:
            summary[cat] += 1
    return summary


# ---------------------------------------------------------------------------
# 数据采集（取数层，主流程调用；单测不触达）
# ---------------------------------------------------------------------------

def collect(owner: str, repo: str, weeks: int) -> dict[str, Any]:
    """从 GitLink 取本场景所需的全部数据。"""
    # commits 取够 ~weeks 周（每周按 30 条粗估，上限 max_pages=10）
    cm_pages = max(2, min(10, (weeks // 3) + 1))
    commits = c.commits(owner, repo, ref="main", max_pages=cm_pages, page_size=100)
    issues = c.issues_all(owner, repo, max_pages=10, page_size=50)
    pullreqs = c.prs_all(owner, repo, max_pages=10, page_size=50)
    milestones = c.milestones(owner, repo, state="all")
    langs = c.languages(owner, repo)
    contribs = c.contributors(owner, repo)
    readme = c.readme(owner, repo)
    tree = c.tree(owner, repo)
    return {
        "info": c.repo_info(owner, repo),
        "commits": commits,
        "issues": issues,
        "prs": pullreqs,
        "milestones": milestones,
        "languages": langs,
        "contributors": contribs,
        "readme": readme,
        "tree": tree,
    }


# ---------------------------------------------------------------------------
# 主算法：组装结果 dict
# ---------------------------------------------------------------------------

def run(owner: str, repo: str, weeks: int, raw: dict[str, Any] | None = None) -> dict[str, Any]:
    """主入口：取数（或复用传入的 raw）→ 算法 → 结果 dict。"""
    if raw is None:
        raw = collect(owner, repo, weeks)

    commits = raw.get("commits") or []
    issues = raw.get("issues") or []
    prs = raw.get("prs") or []
    milestones = raw.get("milestones") or []
    langs = raw.get("languages") or {}
    contribs = raw.get("contributors") or []
    readme = raw.get("readme") or ""
    tree = raw.get("tree") or []

    commits_ts = bin_weekly(commits, weeks, _commit_time)
    issues_ts = bin_weekly(issues, weeks, _issue_time)
    prs_ts = bin_weekly(prs, weeks, _pr_time)
    heatmap = contribution_heatmap(contribs, commits, weeks)

    # 合并 readme + 提交信息作为论文链接抽取语料
    corpus_parts = [readme]
    for cm in commits[:50]:
        msg = cm.get("message") or ""
        if isinstance(msg, str):
            corpus_parts.append(msg)
    paper_links = extract_paper_links("\n".join(corpus_parts))

    artifacts = classify_artifacts(tree)
    art_summary = artifact_summary(artifacts)

    # 里程碑甘特数据：取有 due_on 的，转成 [start, end, title]
    gantt: list[dict[str, Any]] = []
    for ms in milestones:
        if not isinstance(ms, dict):
            continue
        title = ms.get("name") or ms.get("title") or ""
        due = _to_timestamp(ms.get("due_on") or ms.get("effective_date"))
        start = _to_timestamp(ms.get("start_date"))
        if due > 0:
            gantt.append({
                "title": title,
                "start": start if start > 0 else due - 14 * 86400,
                "due": due,
            })

    return {
        "scenario": "S6_research_visualization",
        "repo": f"{owner}/{repo}",
        "weeks": weeks,
        "timeline": {
            "labels": commits_ts["labels"],
            "commits": commits_ts["counts"],
            "issues": issues_ts["counts"],
            "prs": prs_ts["counts"],
        },
        "heatmap": heatmap,
        "languages": langs,
        "milestones": gantt,
        "paper_links": paper_links,
        "artifacts": artifacts,
        "artifact_summary": art_summary,
        "meta": {
            "commit_count": len(commits),
            "issue_count": len(issues),
            "pr_count": len(prs),
            "milestone_count": len(milestones),
            "contributor_count": len(contribs),
        },
    }


# ---------------------------------------------------------------------------
# 渲染：Markdown 摘要报告
# ---------------------------------------------------------------------------

def render_report(result: dict[str, Any]) -> str:
    repo = result["repo"]
    tl = result["timeline"]
    weeks = result["weeks"]
    meta = result["meta"]
    art = result["artifact_summary"]
    lines = [
        f"# 科研成果可视化沉淀报告 — {repo}\n",
        f"> 场景 S6 · 子赛题四「应用 GitLink 辅助科研」\n",
        f"## 一、活跃度概览（最近 {weeks} 周）\n",
        f"- 提交数: **{meta['commit_count']}**（窗口内峰值 "
        f"{max(tl['commits']) if tl['commits'] else 0} 提交/周）",
        f"- 新增 Issue: **{meta['issue_count']}**，新增 PR: **{meta['pr_count']}**",
        f"- 贡献者: **{meta['contributor_count']}**，里程碑: **{meta['milestone_count']}**\n",
        "## 二、开发节奏（最近 8 周快照）\n",
        "| 周 | commits | issues | prs |",
        "|----|---------|--------|-----|",
    ]
    tail = tl["labels"][-8:]
    for i, label in enumerate(tail):
        idx = len(tl["labels"]) - len(tail) + i
        lines.append(f"| {label} | {tl['commits'][idx]} | {tl['issues'][idx]} | {tl['prs'][idx]} |")

    lines += ["\n## 三、核心贡献者热力（贡献者 × 周提交数）\n",
              "| 贡献者 | 窗口内提交 |",
              "|--------|-----------|"]
    hm = result["heatmap"]
    for i, user in enumerate(hm["users"][:10]):
        total = sum(hm["matrix"][i])
        lines.append(f"| `{user}` | {total} |")

    lines += ["\n## 四、科研产物分类\n",
              f"- 论文/笔记 (paper): **{art['paper']}**",
              f"- 数据集 (dataset): **{art['dataset']}**",
              f"- 模型 (model): **{art['model']}**",
              f"- 基准 (benchmark): **{art['benchmark']}**\n"]
    if result["paper_links"]:
        lines += ["## 五、抽取到的论文引用\n",
                  "| 类型 | 链接 |",
                  "|------|------|"]
        for p in result["paper_links"][:15]:
            lines.append(f"| {p['type']} | {p['target']} |")
    else:
        lines.append("## 五、抽取到的论文引用\n\n_未在 README/提交信息中发现 arXiv 或 DOI 引用_\n")

    lines.append(f"\n_交互可视化见 visual.html（或原始数据 visual.json）_\n")
    return "\n".join(lines)


# ---------------------------------------------------------------------------
# 渲染：交互 HTML（plotly 多子图）+ JSON bundle
# ---------------------------------------------------------------------------

def render_html(result: dict[str, Any]) -> str | None:
    """构建单一交互 HTML（多子图），修复图例/空白/饼图/甘特问题。无 plotly 返回 None。"""
    try:
        import plotly.graph_objects as go
        from plotly.subplots import make_subplots
    except ImportError:
        return None

    tl = result["timeline"]
    hm = result["heatmap"]
    langs = result["languages"] or {}
    gantt = result["milestones"]

    # ---- 预处理：时间线截断到最近有数据的周 ----
    labels = tl["labels"]
    commits_arr = tl["commits"]
    prs_arr = tl["prs"]
    # 找到最后一个非零周，只展示到该周 + 前面 buffer
    last_data_idx = -1
    for i in range(len(labels) - 1, -1, -1):
        if commits_arr[i] > 0 or prs_arr[i] > 0:
            last_data_idx = i
            break
    if last_data_idx < 0:
        last_data_idx = len(labels) - 1
    # 截取：从最早有数据的周的前 4 周开始，到最新周，最少 12 周
    first_nonzero = -1
    for i in range(len(labels)):
        if commits_arr[i] > 0 or prs_arr[i] > 0:
            first_nonzero = i
            break
    if first_nonzero >= 0:
        start_idx = max(0, first_nonzero - 4)
        end_idx = min(len(labels), last_data_idx + 2)
        if end_idx - start_idx < 12:
            start_idx = max(0, end_idx - 12)
    else:
        start_idx = 0
        end_idx = len(labels)
    trim_labels = labels[start_idx:end_idx]
    trim_commits = commits_arr[start_idx:end_idx]
    trim_prs = prs_arr[start_idx:end_idx]

    # 简化周标签：去掉年份，只显示 "W01", "W05" ...
    def _short_week(label: str) -> str:
        parts = label.split("-W")
        return f"W{parts[1]}" if len(parts) == 2 else label
    trim_labels_short = [_short_week(lb) for lb in trim_labels]
    # 计算 tick 间隔：标签数 <= 12 则全部显示，否则每 2~4 个显示一个
    n_labels = len(trim_labels_short)
    if n_labels <= 12:
        tick_step = 1
    elif n_labels <= 20:
        tick_step = 2
    else:
        tick_step = max(2, n_labels // 10)
    tick_vals = trim_labels_short[::tick_step]
    tick_text = tick_vals

    # ---- 语言数据预处理：<3% 合并为「其他」----
    pie_labels, pie_values, pie_text = [], [], []
    if langs:
        lang_items = []
        for k, v in langs.items():
            s = str(v).strip().rstrip("%")
            try:
                lang_items.append((k, float(s)))
            except ValueError:
                lang_items.append((k, 0.0))
        total = sum(x[1] for x in lang_items) or 1.0
        pie_labels, pie_values, pie_text = [], [], []
        other_val = 0.0
        for name, val in lang_items:
            pct = val / total * 100
            if pct < 3.0:
                other_val += val
            else:
                pie_labels.append(name)
                pie_values.append(val)
                pie_text.append(f"{name}: {pct:.1f}% ({val:.1f}% lines)")
        if other_val > 0:
            pie_labels.append("其他")
            pie_values.append(other_val)
            pie_text.append(f"其他: {other_val/total*100:.1f}% ({other_val:.1f}% lines)")

    # ---- 决定布局（跳过空白图）----
    has_heatmap = bool(hm["users"]) and any(sum(row) > 0 for row in hm["matrix"])
    has_pie = bool(langs and len(pie_labels) > 1)
    has_gantt = len(gantt) >= 2

    rows = 1
    if has_heatmap:
        rows += 1
    if has_pie:
        rows += 1
    if has_gantt:
        rows += 1

    # ---- 构建 specs（pie 需要 domain 类型，其他用 xy）----
    spec_list = [{"type": "xy"}]  # row 1: timeline (always present)
    if has_heatmap:
        spec_list.append({"type": "xy"})
    if has_pie:
        spec_list.append({"type": "domain"})
    if has_gantt:
        spec_list.append({"type": "xy"})

    fig = make_subplots(
        rows=rows, cols=1,
        specs=[[s] for s in spec_list],
        vertical_spacing=0.10,
        row_heights=[max(0.35, 1.0 / rows)] * rows,
    )
    cur = 1

    # ---- 1) 开发时间线 ----
    fig.add_trace(go.Scatter(
        x=trim_labels_short, y=trim_commits, name="Commits (提交)",
        mode="lines+markers", line=dict(color="#4F6BED", width=2),
        marker=dict(size=5),
    ), row=cur, col=1)
    fig.add_trace(go.Scatter(
        x=trim_labels_short, y=trim_prs, name="PRs (合并请求)",
        mode="lines+markers", line=dict(color="#2EC4B6", width=2),
        marker=dict(size=5),
    ), row=cur, col=1)
    fig.update_yaxes(title_text="数量 (条)", row=cur, col=1)
    fig.update_xaxes(tickangle=0, tickvals=tick_vals, ticktext=tick_text,
                     tickfont=dict(size=10), row=cur, col=1)
    cur += 1

    # ---- 2) 贡献热力图（有数据才画）----
    if has_heatmap:
        hm_weeks_short = [_short_week(w) for w in hm["weeks"]]
        hm_n = len(hm_weeks_short)
        hm_step = 1 if hm_n <= 12 else (2 if hm_n <= 20 else max(2, hm_n // 10))
        hm_tick_vals = hm_weeks_short[::hm_step]
        fig.add_trace(go.Heatmap(
            z=hm["matrix"], x=hm_weeks_short, y=hm["users"],
            colorscale="Blues", name="提交数",
            colorbar=dict(title="次", len=0.4, y=0.5 + 0.3 / rows),
        ), row=cur, col=1)
        fig.update_xaxes(tickangle=0, tickvals=hm_tick_vals, ticktext=hm_tick_vals,
                         tickfont=dict(size=9), row=cur, col=1)
        cur += 1

    # ---- 3) 语言饼图 ----
    if has_pie:
        fig.add_trace(go.Pie(
            labels=pie_labels, values=pie_values,
            text=pie_text, textinfo="text",
            textfont=dict(size=11),
            marker=dict(colors=["#4F6BED", "#2EC4B6", "#F26B5E", "#F59E0B", "#06B6D4",
                                "#8B5CF6", "#94A3B8", "#64748B"]),
        ), row=cur, col=1)
        cur += 1

    # ---- 4) 里程碑甘特 ----
    if has_gantt:
        for g in gantt[:10]:
            title = g["title"] or "(milestone)"
            start_str = datetime.fromtimestamp(g["start"], tz=timezone.utc).strftime("%m-%d")
            due_str = datetime.fromtimestamp(g["due"], tz=timezone.utc).strftime("%m-%d")
            hover = f"{title}<br>{start_str} → {due_str}"
            fig.add_trace(go.Bar(
                x=[g["due"] - g["start"]], y=[title],
                base=g["start"], orientation="h",
                marker=dict(color="#ffa15a", line=dict(color="#d97706", width=1)),
                hovertemplate=hover, showlegend=False,
                width=0.5,
            ), row=cur, col=1)
        fig.update_xaxes(title_text="日期 (UTC)", row=cur, col=1,
                         tickformat="%m-%d", dtick=7 * 86400 * 1000)
        cur += 1

    fig.update_layout(
        title=dict(
            text=f"科研成果可视化 — {result['repo']}（最近 {result['weeks']} 周）",
            font=dict(size=16),
        ),
        height=max(500, rows * 340),
        legend=dict(orientation="h", yanchor="top", y=-0.12, x=0.5, xanchor="center",
                     font=dict(size=11)),
        margin=dict(l=60, r=40, t=60, b=60),
        hovermode="x unified",
    )

    return fig.to_html(full_html=True, include_plotlyjs="cdn",
                       default_width="100%", default_height=f"{max(500, rows * 340)}px")


# ---------------------------------------------------------------------------
# 主入口
# ---------------------------------------------------------------------------

def main():
    ap = argparse.ArgumentParser(description="S6 科研成果可视化沉淀")
    ap.add_argument("--owner", required=True)
    ap.add_argument("--repo", required=True)
    ap.add_argument("--weeks", type=int, default=26, help="回溯多少周（默认 26）")
    ap.add_argument("--out", help="输出目录（写 visual.html + visual.json + report.md）；"
                                  "省略则打印 JSON")
    args = ap.parse_args()

    result = run(args.owner, args.repo, args.weeks)

    if args.out:
        os.makedirs(args.out, exist_ok=True)
        # 原始数据 bundle（供前端二次开发）
        with open(os.path.join(args.out, "visual.json"), "w", encoding="utf-8") as f:
            json.dump(result, f, ensure_ascii=False, indent=2)
        with open(os.path.join(args.out, "report.md"), "w", encoding="utf-8") as f:
            f.write(render_report(result))

        html = render_html(result)
        if html is not None:
            with open(os.path.join(args.out, "visual.html"), "w", encoding="utf-8") as f:
                f.write(html)
            print(f"✓ S6 可视化完成 → {args.out}/visual.html | visual.json | report.md")
        else:
            print(f"✓ S6 可视化完成（无 plotly，已降级）→ {args.out}/visual.json | report.md")
            print("  提示：pip install plotly 后可生成交互 HTML")
        meta = result["meta"]
        art = result["artifact_summary"]
        print(f"  commits={meta['commit_count']} issues={meta['issue_count']} "
              f"prs={meta['pr_count']} 贡献者={meta['contributor_count']}")
        print(f"  产物 paper={art['paper']} dataset={art['dataset']} "
              f"model={art['model']} benchmark={art['benchmark']}")
        print(f"  论文引用: {len(result['paper_links'])} 条")
    else:
        print(json.dumps(result, ensure_ascii=False, indent=2))


if __name__ == "__main__":
    main()
