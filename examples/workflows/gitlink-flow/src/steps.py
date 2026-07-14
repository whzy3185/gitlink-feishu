"""gitlink-flow 工作流步骤库。

每个函数是一个可独立测试的工作流步骤，接收采集到的原始数据，输出结构化结果。
编排器 flow.py 按顺序调用这些步骤，串联成端到端的社区运营自动化工作流。

三个子工作流对标官方 examples/workflows 的参考场景：
- triage：Issue 自动分拣（按关键词/标签分类，识别新手友好任务）
- pr_review_summary：PR Review 汇总（统计 PR 状态、识别待处理）
- release_notes：Release Notes 生成（从提交按 conventional commits 归类）

另复用 5 个自研 Skill 的核心分析：社区健康体检、依赖巡检、贡献者致谢、知识库索引。

全程只读，不修改远程数据。
"""

from __future__ import annotations

import re
from collections import Counter
from typing import Any

# ---------------------------------------------------------------------------
# 通用归一化
# ---------------------------------------------------------------------------

CONVENTIONAL_TYPES = {
    "feat": "新功能", "fix": "缺陷修复", "docs": "文档", "refactor": "重构",
    "perf": "性能", "test": "测试", "chore": "工程", "build": "构建",
    "ci": "持续集成", "style": "风格", "revert": "回退",
}

GOOD_FIRST_HINTS = ["typo", "docs", "doc", "readme", "test", "translation", "example",
                    "文档", "注释", "翻译", "示例", "拼写"]
BUG_HINTS = ["bug", "error", "fail", "crash", "panic", "错误", "失败", "崩溃", "异常"]
FEATURE_HINTS = ["feature", "support", "add", "enhance", "新增", "支持", "功能", "建议"]
QUESTION_HINTS = ["how", "why", "question", "如何", "怎么", "为什么", "请问"]


def _commit_type(message: str) -> str:
    m = re.match(r"^\s*([a-zA-Z]+)(?:\([^)]*\))?!?:", message or "")
    if not m:
        return "other"
    t = m.group(1).lower()
    return t if t in CONVENTIONAL_TYPES else "other"


# ---------------------------------------------------------------------------
# 子工作流 1：Issue 自动分拣（triage）
# ---------------------------------------------------------------------------

def triage_issues(issues: list[dict[str, Any]]) -> dict[str, Any]:
    """对 Issue 按内容自动分类，并识别新手友好任务。

    分类：bug / feature / question / good-first / other。
    每个 Issue 给出建议标签，供维护者打标参考。
    """
    buckets: dict[str, list[dict[str, Any]]] = {
        "bug": [], "feature": [], "question": [], "good-first": [], "other": [],
    }
    results: list[dict[str, Any]] = []
    for it in issues:
        title = str(it.get("name") or it.get("subject") or it.get("title") or "")
        body = str(it.get("description") or it.get("body") or "")
        text = f"{title}\n{body}".lower()

        category = "other"
        if any(h in text for h in BUG_HINTS):
            category = "bug"
        elif any(h in text for h in FEATURE_HINTS):
            category = "feature"
        elif any(h in text for h in QUESTION_HINTS):
            category = "question"

        good_first = any(h in text for h in GOOD_FIRST_HINTS) and len(body) < 800
        bucket_key = "good-first" if good_first else category
        rec = {
            "id": str(it.get("id") or ""),
            "title": title[:60],
            "category": category,
            "good_first": good_first,
            "suggested_label": "good first issue" if good_first else category,
        }
        buckets.setdefault(bucket_key, []).append(rec)
        results.append(rec)

    return {
        "total": len(issues),
        "by_category": {k: len(v) for k, v in buckets.items()},
        "good_first_count": len(buckets["good-first"]),
        "items": results,
    }


# ---------------------------------------------------------------------------
# 子工作流 2：PR Review 汇总（pr-review）
# ---------------------------------------------------------------------------

def pr_review_summary(pulls: list[dict[str, Any]]) -> dict[str, Any]:
    """汇总 PR 状态，识别待 Review 的 PR。

    GitLink pull_request_status：0=open, 1=merged, 2=closed。
    """
    open_prs: list[dict[str, Any]] = []
    merged = closed = 0
    for p in pulls:
        status = p.get("pull_request_status")
        title = str(p.get("name") or p.get("title") or "")
        author = p.get("author_login") or p.get("author_name") or "unknown"
        if status == 1:
            merged += 1
        elif status == 2:
            closed += 1
        else:
            open_prs.append({
                "id": str(p.get("pull_request_number") or p.get("id") or ""),
                "title": title[:60],
                "author": author,
                "is_fork": bool(p.get("fork_project_user")),
            })
    return {
        "total": len(pulls),
        "open": len(open_prs),
        "merged": merged,
        "closed": closed,
        "merge_rate": round(merged / len(pulls) * 100, 1) if pulls else 0.0,
        "pending_review": open_prs,
    }


# ---------------------------------------------------------------------------
# 子工作流 3：Release Notes 生成（release-notes）
# ---------------------------------------------------------------------------

def release_notes(commits: list[dict[str, Any]], version: str = "Unreleased") -> dict[str, Any]:
    """从提交历史按 conventional commits 归类生成 Release Notes。"""
    groups: dict[str, list[str]] = {}
    typed = 0
    for c in commits:
        msg = (c.get("message") or "").splitlines()[0] if c.get("message") else ""
        t = _commit_type(msg)
        if t == "other":
            continue
        typed += 1
        # 去掉类型前缀，保留描述
        desc = re.sub(r"^\s*[a-zA-Z]+(?:\([^)]*\))?!?:\s*", "", msg).strip()
        groups.setdefault(t, []).append(desc)

    # 生成 Markdown
    order = ["feat", "fix", "perf", "refactor", "docs", "test", "build", "ci", "chore"]
    lines = [f"## {version}", ""]
    for t in order:
        if t in groups:
            lines.append(f"### {CONVENTIONAL_TYPES[t]}（{t}）")
            for d in groups[t][:20]:
                lines.append(f"- {d}")
            lines.append("")
    return {
        "version": version,
        "typed_commits": typed,
        "total_commits": len(commits),
        "groups": {k: len(v) for k, v in groups.items()},
        "markdown": "\n".join(lines).strip(),
    }


# ---------------------------------------------------------------------------
# 复用 Skill：社区健康体检（scaffold 核心）
# ---------------------------------------------------------------------------

HEALTH_FILES = {
    "README": ["readme.md", "readme.rst", "readme"],
    "LICENSE": ["license", "license.md", "copying"],
    "CONTRIBUTING": ["contributing.md", "contributing"],
    "贡献准则": ["code_of_conduct.md"],
}


def health_check(root_files: list[str]) -> dict[str, Any]:
    """基于根目录文件名检测社区健康文件齐全度。"""
    names = {f.lower() for f in root_files}
    present, missing = [], []
    for label, cands in HEALTH_FILES.items():
        if any(c in names for c in cands):
            present.append(label)
        else:
            missing.append(label)
    score = round(len(present) / len(HEALTH_FILES) * 100)
    return {"score": score, "present": present, "missing": missing}


# ---------------------------------------------------------------------------
# 复用 Skill：贡献者致谢（contributor 核心）
# ---------------------------------------------------------------------------

def contributor_highlights(contributors: list[dict[str, Any]], top: int = 5) -> dict[str, Any]:
    """提取贡献者亮点：总数、前 N 名。"""
    profiles = sorted(
        ({"name": c.get("login") or c.get("name") or "unknown",
          "contributions": int(c.get("contributions") or 0)} for c in contributors),
        key=lambda x: x["contributions"], reverse=True,
    )
    total = sum(p["contributions"] for p in profiles)
    return {
        "total_contributors": len(profiles),
        "total_contributions": total,
        "top": profiles[:top],
    }
