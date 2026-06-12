"""gitlink-onboard：新贡献者上手指南生成器。

为想参与一个 GitLink 项目的新人，一站式生成完整的「上手指南」，整合：
- 项目简介与社区指标
- 技术栈识别（从依赖文件推断）
- 核心目录/文件导航（帮新人定位代码入口）
- 社区文件检查（README/CONTRIBUTING/行为准则是否齐全）
- 适合上手的 good-first-issue
- 核心贡献者（遇到问题找谁）
- 标准上手步骤（Fork → 改 → PR）

相比 gitlink-newcomer（聚焦"找新手 Issue"），本技能输出的是一份覆盖"项目是什么、
代码在哪、找谁问、从哪个 Issue 开始、怎么提 PR"的完整上手指南，功能更全面。

数据来自 GitLink 公开 API（只读），无需登录。

用法：
    python onboard.py --owner Gitlink --repo gitlink-cli
    python onboard.py --owner Gitlink --repo gitlink-cli --format json
"""

from __future__ import annotations

import argparse
import json
import sys
from pathlib import Path
from typing import Any

sys.path.insert(0, str(Path(__file__).resolve().parent))
from glapi import GitLinkClient, GitLinkError, split_owner_repo

if hasattr(sys.stdout, "reconfigure"):
    try:
        sys.stdout.reconfigure(encoding="utf-8")
    except Exception:
        pass

# 依赖文件 → 技术栈
STACK_MARKERS = {
    "go.mod": "Go", "package.json": "Node.js / JavaScript", "requirements.txt": "Python",
    "pyproject.toml": "Python", "Cargo.toml": "Rust", "pom.xml": "Java (Maven)",
    "build.gradle": "Java (Gradle)", "composer.json": "PHP", "Gemfile": "Ruby",
    "Dockerfile": "Docker", "Makefile": "Make",
}
# 核心目录提示
CORE_DIR_HINTS = {
    "src": "源码主目录", "lib": "库代码", "cmd": "命令行入口", "internal": "内部包",
    "pkg": "公共包", "app": "应用代码", "core": "核心模块", "docs": "文档",
    "test": "测试", "tests": "测试", "examples": "示例", "skills": "Agent Skills",
}
GOOD_FIRST_HINTS = ["typo", "docs", "doc", "readme", "test", "translation", "example",
                    "文档", "注释", "翻译", "示例", "拼写"]
HEALTH_FILES = {"README": ["readme.md", "readme.rst", "readme"],
                "CONTRIBUTING": ["contributing.md", "contributing"],
                "行为准则": ["code_of_conduct.md"]}


def detect_stack(root_files: list[str]) -> list[str]:
    names = {f.lower() for f in root_files}
    stacks = []
    for marker, lang in STACK_MARKERS.items():
        if marker.lower() in names:
            stacks.append(lang)
    return sorted(set(stacks))


def navigate_dirs(entries: list[dict[str, Any]]) -> list[dict[str, str]]:
    nav = []
    for e in entries:
        if e.get("type") == "dir":
            name = str(e.get("name", ""))
            hint = CORE_DIR_HINTS.get(name.lower(), "")
            nav.append({"name": name, "hint": hint})
    # 有提示的排前面
    nav.sort(key=lambda x: (x["hint"] == "", x["name"]))
    return nav


def pick_good_first(issues: list[dict[str, Any]]) -> list[dict[str, str]]:
    out = []
    for it in issues:
        status = str(it.get("issue_status") or "")
        if "关" in status:
            continue
        title = str(it.get("name") or it.get("subject") or "")
        body = str(it.get("description") or "")
        if any(h in (title + body).lower() for h in GOOD_FIRST_HINTS) and len(body) < 800:
            out.append({"id": str(it.get("id") or ""), "title": title[:60]})
    return out[:8]


def check_health(root_files: list[str]) -> dict[str, bool]:
    names = {f.lower() for f in root_files}
    return {label: any(c in names for c in cands) for label, cands in HEALTH_FILES.items()}


def build_guide(owner: str, repo: str, info: dict[str, Any], root_files: list[str],
                entries: list[dict[str, Any]], issues: list[dict[str, Any]],
                contributors: list[dict[str, Any]]) -> dict[str, Any]:
    stacks = detect_stack(root_files)
    nav = navigate_dirs(entries)
    good_first = pick_good_first(issues)
    health = check_health(root_files)
    core_contribs = sorted(
        ({"name": c.get("login") or c.get("name"), "contributions": int(c.get("contributions") or 0)}
         for c in contributors), key=lambda x: x["contributions"], reverse=True)[:3]
    return {
        "owner": owner, "repo": repo,
        "description": info.get("description") or "",
        "default_branch": info.get("default_branch") or "master",
        "stars": info.get("praises_count") or 0,
        "forks": info.get("forked_count") or 0,
        "stacks": stacks,
        "navigation": nav,
        "good_first": good_first,
        "health": health,
        "core_contributors": core_contribs,
    }


def render_guide(g: dict[str, Any]) -> str:
    owner, repo = g["owner"], g["repo"]
    lines = [
        f"# 新贡献者上手指南 — {owner}/{repo}",
        "",
        "欢迎参与本项目！这份指南由 gitlink-onboard 自动生成，带你快速上手。",
        "",
        "## 一、项目是什么",
        "",
        f"- 简介：{g['description'] or '（仓库未提供简介）'}",
        f"- 技术栈：{('、'.join(g['stacks'])) or '未自动识别'}",
        f"- 社区：⭐ {g['stars']} / Fork {g['forks']}　默认分支 `{g['default_branch']}`",
        "",
        "## 二、代码在哪（核心目录导航）",
        "",
    ]
    if g["navigation"]:
        for n in g["navigation"][:12]:
            hint = f" — {n['hint']}" if n["hint"] else ""
            lines.append(f"- `{n['name']}/`{hint}")
    else:
        lines.append("- （未获取到目录结构）")
    lines += ["", "## 三、社区文件是否齐全", ""]
    for label, ok in g["health"].items():
        lines.append(f"- {'✅' if ok else '⬜'} {label}{'' if ok else '（建议先补充，新人可贡献）'}")
    lines += ["", "## 四、从哪个 Issue 开始", ""]
    if g["good_first"]:
        lines.append("以下是适合新人上手的 Issue：")
        lines.append("")
        for it in g["good_first"]:
            lines.append(f"- #{it['id']} {it['title']}")
    else:
        lines.append("- 暂未发现明显的新手友好 Issue，可在 Issue 区留言询问维护者。")
    lines += ["", "## 五、遇到问题找谁", ""]
    if g["core_contributors"]:
        for c in g["core_contributors"]:
            lines.append(f"- @{c['name']}（核心贡献者，{c['contributions']} 次贡献）")
    else:
        lines.append("- 在 Issue 区留言，社区会帮你。")
    lines += [
        "", "## 六、上手步骤", "",
        f"1. 阅读 README" + ("、CONTRIBUTING" if g["health"].get("CONTRIBUTING") else "") + " 了解规范。",
        f"2. Fork 本仓库：`gitlink-cli repo +fork --owner {owner} --repo {repo}`",
        "3. 克隆你的 Fork，新建分支进行修改。",
        f"4. 从你的 Fork 向 `{owner}/{repo}` 的 `{g['default_branch']}` 分支提交 PR。",
        "5. 在 PR 描述里关联你解决的 Issue，等待 Review。",
        "",
        "---", "", "由 gitlink-onboard 生成。祝你贡献顺利！🚀",
    ]
    return "\n".join(lines)


def analyze(owner: str, repo: str, client: GitLinkClient | None = None) -> dict[str, Any]:
    client = client or GitLinkClient()
    info = client.repo_info(owner, repo)
    branch = info.get("default_branch") or "master"
    try:
        entries = client.list_dir(owner, repo, "", branch)
    except GitLinkError:
        entries = []
    root_files = [str(e.get("name", "")) for e in entries]
    issues = client.issues(owner, repo, limit=50)
    contributors = client.contributors(owner, repo)
    return build_guide(owner, repo, info, root_files, entries, issues, contributors)


def main(argv: list[str] | None = None) -> int:
    p = argparse.ArgumentParser(prog="gitlink-onboard", description="新贡献者上手指南生成器")
    p.add_argument("--owner"); p.add_argument("--repo"); p.add_argument("--slug")
    p.add_argument("--format", choices=["markdown", "json"], default="markdown")
    p.add_argument("--output", type=Path)
    args = p.parse_args(argv)

    if args.slug:
        owner, repo = split_owner_repo(args.slug)
    elif args.owner and args.repo:
        owner, repo = args.owner, args.repo
    else:
        print("错误：请用 --owner/--repo 或 --slug 指定仓库。", file=sys.stderr)
        return 2

    try:
        g = analyze(owner, repo)
    except GitLinkError as exc:
        print(f"采集失败：{exc}", file=sys.stderr)
        return 1

    out = (json.dumps(g, ensure_ascii=False, indent=2) if args.format == "json"
           else render_guide(g))
    if args.output:
        args.output.parent.mkdir(parents=True, exist_ok=True)
        args.output.write_text(out, encoding="utf-8")
        print(f"已写入 {args.output}")
    else:
        print(out)
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
