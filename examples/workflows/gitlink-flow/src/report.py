"""社区运营周报生成。

把工作流 6 个步骤的结果汇总成一份可直接发布到 Issue/Wiki 的社区运营周报。
"""

from __future__ import annotations

from typing import Any


def render_weekly(result: dict[str, Any]) -> str:
    """渲染社区运营周报（Markdown）。"""
    owner, repo = result["owner"], result["repo"]
    info = result["repo_info"]
    triage = result["step1_triage"]
    pr = result["step2_pr_review"]
    rel = result["step3_release_notes"]
    health = result["step4_health"]
    contrib = result["step5_contributors"]

    lines = [
        f"# 社区运营周报 — {owner}/{repo}",
        "",
        f"生成时间：{result['generated_at']}　|　工具：gitlink-flow 端到端工作流",
        "",
        "> 本周报由 gitlink-flow 自动串联 Issue 分拣、PR Review、Release Notes、"
        "社区健康体检、贡献者致谢等步骤生成，覆盖社区运营全链路。",
        "",
        "## 一、仓库概览",
        "",
        f"- Star {info.get('praises_count') or 0} / Fork {info.get('forked_count') or 0}"
        f" / Issue {info.get('issues_count') or 0} / PR {info.get('pull_requests_count') or 0}",
        f"- 数据采集：{result.get('data_source', 'gitlink-cli 命令')}",
        "",
        "## 二、Issue 自动分拣",
        "",
        f"共 {triage['total']} 个 Issue，自动分类：",
        "",
    ]
    for cat, n in triage["by_category"].items():
        if n:
            lines.append(f"- {cat}：{n} 个")
    lines.append("")
    if triage["good_first_count"]:
        lines.append(f"发现 **{triage['good_first_count']}** 个适合新人上手的任务，建议打 `good first issue` 标签：")
        lines.append("")
        for it in triage["items"]:
            if it["good_first"]:
                ref = f"#{it['id']}" if it["id"] else ""
                lines.append(f"- {ref} {it['title']}")
        lines.append("")

    lines += [
        "## 三、PR Review 汇总",
        "",
        f"共 {pr['total']} 个 PR：开放 {pr['open']} / 已合并 {pr['merged']} / 已关闭 {pr['closed']}，"
        f"合并率 {pr['merge_rate']}%。",
        "",
    ]
    if pr["pending_review"]:
        lines.append("待 Review 的 PR：")
        lines.append("")
        for p in pr["pending_review"][:10]:
            tag = "（来自 Fork）" if p["is_fork"] else ""
            ref = f"#{p['id']}" if p["id"] else ""
            lines.append(f"- {ref} {p['title']} — @{p['author']} {tag}")
        lines.append("")

    lines += [
        "## 四、社区健康体检",
        "",
        f"健康度评分：**{health['score']}/100**",
        f"- 已具备：{('、'.join(health['present'])) or '无'}",
        f"- 缺失：{('、'.join(health['missing'])) or '无'}",
        "",
        "## 五、贡献者致谢",
        "",
        f"共 {contrib['total_contributors']} 位贡献者，本周致谢榜前列：",
        "",
    ]
    for i, c in enumerate(contrib["top"], 1):
        medal = {1: "🥇", 2: "🥈", 3: "🥉"}.get(i, f"{i}.")
        lines.append(f"- {medal} {c['name']}（{c['contributions']} 次贡献）")
    lines.append("")

    lines += [
        "## 六、Release Notes（自动生成）",
        "",
        f"基于提交历史，版本 `{rel['version']}` 的变更摘要"
        f"（规范化提交 {rel['typed_commits']}/{rel['total_commits']}）：",
        "",
        rel["markdown"] or "（暂无符合 conventional commits 规范的提交）",
        "",
        "---",
        "",
        "由 gitlink-flow 社区运营自动化工作流生成。所有数据来自 GitLink 平台，分析全程只读。",
    ]
    return "\n".join(lines)
