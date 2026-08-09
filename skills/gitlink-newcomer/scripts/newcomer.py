"""gitlink-newcomer：新人引导分析。

识别一个 GitLink 仓库中适合新贡献者上手的 Issue，评估上手难度与友好度，
为每个候选 Issue 生成个性化引导评论，并产出新手任务看板。

数据来自 GitLink 公开 API（只读），无需登录。生成的引导评论仅作为建议输出，
是否发布由用户通过 gitlink-cli 自行决定。

用法：
    python newcomer.py --owner Gitlink --repo gitlink-cli
    python newcomer.py --owner Gitlink --repo gitlink-cli --format json
    python newcomer.py --owner Gitlink --repo gitlink-cli --issue 12   # 只为某个 Issue 生成引导
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

# ---------------------------------------------------------------------------
# 识别规则
# ---------------------------------------------------------------------------

# good-first-issue 的标签信号（不同项目命名习惯）
GOOD_FIRST_LABELS = {
    "good first issue", "good-first-issue", "goodfirstissue",
    "good first", "first-timers-only", "first timers only",
    "beginner", "beginner-friendly", "easy", "starter",
    "新手", "新手友好", "新人", "入门", "简单",
}

# 标题/正文中暗示「适合新手」的关键词
EASY_KEYWORDS = [
    "typo", "document", "docs", "readme", "comment", "translation", "translate",
    "rename", "format", "lint", "test", "example", "i18n",
    "文档", "注释", "拼写", "翻译", "示例", "格式", "重命名",
]

# 暗示「难度较高、不适合新手」的关键词
HARD_KEYWORDS = [
    "refactor", "architecture", "performance", "concurrency", "race",
    "security", "deadlock", "memory leak", "breaking change",
    "重构", "架构", "性能", "并发", "安全", "死锁", "内存",
]

# 难度高的标签
HARD_LABELS = {"hard", "complex", "advanced", "epic", "困难", "复杂"}


def _text_of(issue: dict[str, Any]) -> str:
    """合并 Issue 的标题与正文用于关键词分析。"""
    title = issue.get("name") or issue.get("subject") or issue.get("title") or ""
    body = issue.get("description") or issue.get("body") or ""
    return f"{title}\n{body}".lower()


def _labels_of(issue: dict[str, Any]) -> list[str]:
    """提取 Issue 标签名（兼容多种字段结构）。"""
    labels: list[str] = []
    raw = issue.get("issue_tags") or issue.get("labels") or issue.get("tags")
    if isinstance(raw, list):
        for item in raw:
            if isinstance(item, dict):
                name = item.get("name") or item.get("title")
                if name:
                    labels.append(str(name).lower())
            elif item:
                labels.append(str(item).lower())
    return labels


def _issue_number(issue: dict[str, Any]) -> int | None:
    """提取仓库内 Issue 序号（web URL 中显示的编号）。

    注意：GitLink 的 Issue 列表接口通常只返回全局数据库 id，不含 web 序号；
    只有单 Issue 详情或带 number/index 字段时才有可靠序号。取不到返回 None，
    避免把全局 id 误当作 web 序号引用。
    """
    for key in ("number", "index"):
        val = issue.get(key)
        if isinstance(val, int):
            return val
    return None


def _issue_gid(issue: dict[str, Any]) -> int | None:
    """全局数据库 id（仅用于去重/展示，不可用于 web 链接 / PR 关联）。"""
    val = issue.get("id")
    return val if isinstance(val, int) else None


def score_issue(issue: dict[str, Any]) -> dict[str, Any]:
    """评估单个 Issue 的新手友好度。

    返回友好度评分（0-100）、难度等级、命中的信号，
    评分越高越适合新人上手。
    """
    text = _text_of(issue)
    labels = _labels_of(issue)
    signals: list[str] = []
    score = 50  # 基准分

    # 标签信号（最强）
    if any(lb in GOOD_FIRST_LABELS for lb in labels):
        score += 35
        signals.append("带有新手友好标签")
    if any(lb in HARD_LABELS for lb in labels):
        score -= 30
        signals.append("带有高难度标签")

    # 关键词信号
    easy_hits = [k for k in EASY_KEYWORDS if k in text]
    if easy_hits:
        score += min(20, len(easy_hits) * 7)
        signals.append(f"内容涉及易上手主题（{', '.join(easy_hits[:3])}）")
    hard_hits = [k for k in HARD_KEYWORDS if k in text]
    if hard_hits:
        score -= min(25, len(hard_hits) * 10)
        signals.append(f"内容涉及高难度主题（{', '.join(hard_hits[:3])}）")

    # 描述长度：太长往往复杂
    body = issue.get("description") or issue.get("body") or ""
    if len(body) > 1500:
        score -= 8
        signals.append("描述较长，可能较复杂")
    elif 30 <= len(body) <= 600:
        score += 5
        signals.append("描述长度适中")

    # 评论数：讨论太多可能有争议或难度大
    comments = issue.get("comment_journals_count") or issue.get("journals_count") or 0
    if isinstance(comments, int) and comments > 15:
        score -= 8
        signals.append("讨论较多，可能存在分歧")

    score = max(0, min(100, score))
    if score >= 75:
        difficulty = "入门"
    elif score >= 55:
        difficulty = "较易"
    elif score >= 40:
        difficulty = "中等"
    else:
        difficulty = "进阶"

    return {
        "number": _issue_number(issue),
        "gid": _issue_gid(issue),
        "title": issue.get("name") or issue.get("subject") or issue.get("title") or "(无标题)",
        "labels": labels,
        "friendliness": score,
        "difficulty": difficulty,
        "signals": signals,
        "author": issue.get("author_login") or issue.get("author_name"),
        "comments": comments if isinstance(comments, int) else 0,
        "is_good_first": score >= 55,
    }


def build_guidance(scored: dict[str, Any], owner: str, repo: str) -> str:
    """为一个候选 Issue 生成个性化的新手引导评论。"""
    num = scored.get("number")
    title = scored["title"]
    diff = scored["difficulty"]
    # 有可靠 web 序号时用 #num 引用，否则用标题引用（不误用全局 id）
    ref = f"#{num}" if num else f"《{title}》"

    lines = [
        f"👋 欢迎！这个 Issue（{ref}）被 gitlink-newcomer 评估为 **{diff}** 难度，适合作为参与本项目的起点。",
        "",
        "如果你想认领它，建议按以下步骤上手：",
        "",
        f"1. 阅读项目的 `README` 和 `CONTRIBUTING`（如有），了解开发与提交规范。",
        f"2. Fork 本仓库并克隆你的 Fork：`gitlink-cli repo +fork --owner {owner} --repo {repo}`",
        "3. 新建一个分支进行修改，保持改动聚焦于本 Issue。",
        f"4. 完成后从你的 Fork 向 `{owner}/{repo}` 提交 PR，并在描述里关联本 Issue。",
        "",
    ]

    if "文档" in str(scored["signals"]) or "docs" in str(scored["signals"]).lower():
        lines.append("> 提示：这看起来是一个文档/文本类改动，通常不需要改动核心逻辑，很适合第一次贡献。")
    if scored["comments"] > 8:
        lines.append("> 提示：该 Issue 已有较多讨论，动手前建议先通读评论，确认当前结论与分工。")
    lines.append("")
    lines.append("有任何问题都可以在本 Issue 下留言，社区很乐意帮助新人。祝贡献顺利！🚀")
    return "\n".join(lines)


def analyze(owner: str, repo: str, limit: int = 50,
            client: GitLinkClient | None = None) -> dict[str, Any]:
    """分析仓库的 Issue，识别并排序新手友好的候选。"""
    client = client or GitLinkClient()
    issues = client.issues(owner, repo, limit=limit)
    scored = [score_issue(i) for i in issues]
    candidates = [s for s in scored if s["is_good_first"]]
    candidates.sort(key=lambda s: s["friendliness"], reverse=True)

    return {
        "owner": owner,
        "repo": repo,
        "total_issues": len(issues),
        "candidate_count": len(candidates),
        "candidates": candidates,
        "all_scored": scored,
    }


def render_board(result: dict[str, Any], owner: str, repo: str) -> str:
    """渲染新手任务看板（Markdown）。"""
    lines = [
        f"# 新手任务看板 — {owner}/{repo}",
        "",
        f"由 gitlink-newcomer 生成。共扫描 {result['total_issues']} 个开放 Issue，"
        f"识别出 **{result['candidate_count']}** 个适合新贡献者上手的任务。",
        "",
    ]
    if not result["candidates"]:
        lines += [
            "暂未发现明显适合新手的 Issue。建议维护者：",
            "",
            "- 为简单任务打上 `good first issue` 标签",
            "- 在 Issue 描述里补充清晰的上手说明与验收标准",
            "",
        ]
        return "\n".join(lines)

    lines += [
        "| 推荐度 | 难度 | Issue | 标题 | 命中信号 |",
        "|:------:|:----:|:-----:|------|----------|",
    ]
    for c in result["candidates"]:
        stars = "⭐" * max(1, round(c["friendliness"] / 20))
        signal = c["signals"][0] if c["signals"] else "-"
        title = c["title"][:40]
        # 有 web 序号用 #num，否则标注全局 id（gid）以便定位
        if c.get("number"):
            ref = f"#{c['number']}"
        elif c.get("gid"):
            ref = f"id:{c['gid']}"
        else:
            ref = "-"
        lines.append(
            f"| {stars} | {c['difficulty']} | {ref} | {title} | {signal} |"
        )
    lines += [
        "",
        "## 建议行动",
        "",
        "1. 为上述 Issue 添加引导评论，欢迎新贡献者认领（见各 Issue 的引导文案）。",
        "2. 确认这些 Issue 的描述包含足够的上手信息。",
        "3. 可在仓库 README 中链接本看板，方便新人发现。",
        "",
    ]
    return "\n".join(lines)


def main(argv: list[str] | None = None) -> int:
    p = argparse.ArgumentParser(
        prog="gitlink-newcomer",
        description="识别 good-first-issue 并生成新手引导",
    )
    p.add_argument("--owner", help="仓库所有者，如 Gitlink")
    p.add_argument("--repo", help="仓库名称，如 gitlink-cli")
    p.add_argument("--slug", help="owner/repo 形式，或完整仓库 URL")
    p.add_argument("--issue", type=int, help="只为指定 Issue 编号生成引导评论")
    p.add_argument("--limit", type=int, default=50, help="扫描的 Issue 数量上限（默认 50）")
    p.add_argument("--format", choices=["markdown", "json"], default="markdown",
                   help="输出格式（默认 markdown）")
    p.add_argument("--output", type=Path, help="输出文件路径，缺省打印到标准输出")
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
        # 单 Issue 引导模式
        if args.issue is not None:
            detail = client.issue_detail(owner, repo, args.issue)
            if not detail:
                print(f"未找到 Issue #{args.issue}", file=sys.stderr)
                return 1
            scored = score_issue(detail)
            guidance = build_guidance(scored, owner, repo)
            if args.format == "json":
                out = json.dumps({"issue": scored, "guidance": guidance},
                                 ensure_ascii=False, indent=2)
            else:
                out = guidance
        else:
            result = analyze(owner, repo, limit=args.limit, client=client)
            if args.format == "json":
                # 为每个候选附上引导文案
                for c in result["candidates"]:
                    c["guidance"] = build_guidance(c, owner, repo)
                out = json.dumps(result, ensure_ascii=False, indent=2)
            else:
                out = render_board(result, owner, repo)
    except GitLinkError as exc:
        print(f"采集失败：{exc}", file=sys.stderr)
        return 1

    if args.output:
        args.output.parent.mkdir(parents=True, exist_ok=True)
        args.output.write_text(out, encoding="utf-8")
        print(f"已写入 {args.output}")
    else:
        print(out)
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
