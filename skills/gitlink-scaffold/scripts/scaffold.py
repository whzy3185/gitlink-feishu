"""gitlink-scaffold：社区健康文件体检与模板生成。

扫描一个 GitLink 仓库，检测开源社区推荐的健康文件是否齐全
（README / LICENSE / CONTRIBUTING / CODE_OF_CONDUCT / SECURITY /
Issue 模板 / PR 模板 / CHANGELOG 等），给出健康度评分，
并为缺失的文件生成可直接使用的中文模板。

数据来自 GitLink 公开 API（只读），无需登录。生成的模板仅输出到本地，
是否提交到仓库由用户决定。

用法：
    python scaffold.py --owner Gitlink --repo gitlink-cli
    python scaffold.py --owner Gitlink --repo gitlink-cli --format json
    python scaffold.py --owner Gitlink --repo gitlink-cli --generate --output-dir out
"""

from __future__ import annotations

import argparse
import json
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
# 社区健康文件清单
# ---------------------------------------------------------------------------

# 每项：键、展示名、候选文件名（不区分大小写）、候选所在目录、权重、是否关键
HEALTH_FILES = [
    {
        "key": "readme", "name": "README", "weight": 20, "critical": True,
        "candidates": ["readme.md", "readme.rst", "readme.txt", "readme"],
        "dirs": [""],
    },
    {
        "key": "license", "name": "LICENSE", "weight": 20, "critical": True,
        "candidates": ["license", "license.md", "license.txt", "copying", "license-mulanpsl2"],
        "dirs": [""],
    },
    {
        "key": "contributing", "name": "CONTRIBUTING", "weight": 15, "critical": False,
        "candidates": ["contributing.md", "contributing.rst", "contributing"],
        "dirs": ["", ".gitlink", ".github", "docs"],
    },
    {
        "key": "code_of_conduct", "name": "CODE_OF_CONDUCT", "weight": 10, "critical": False,
        "candidates": ["code_of_conduct.md", "code-of-conduct.md"],
        "dirs": ["", ".gitlink", ".github", "docs"],
    },
    {
        "key": "security", "name": "SECURITY", "weight": 10, "critical": False,
        "candidates": ["security.md", "security"],
        "dirs": ["", ".gitlink", ".github", "docs"],
    },
    {
        "key": "issue_template", "name": "Issue 模板", "weight": 10, "critical": False,
        "candidates": ["issue_template.md", "issue_template"],
        "dirs": ["", ".gitlink", ".github", ".gitlink/issue_template", ".github/ISSUE_TEMPLATE"],
    },
    {
        "key": "pr_template", "name": "PR 模板", "weight": 10, "critical": False,
        "candidates": ["pull_request_template.md", "pull_request_template", "merge_request_template.md"],
        "dirs": ["", ".gitlink", ".github"],
    },
    {
        "key": "changelog", "name": "CHANGELOG", "weight": 5, "critical": False,
        "candidates": ["changelog.md", "changelog", "changes.md", "history.md"],
        "dirs": [""],
    },
]


def _names_in_dir(client: GitLinkClient, owner: str, repo: str,
                  path: str, ref: str, cache: dict[str, set[str]]) -> set[str]:
    """列出某目录下所有条目名（小写），带本次运行内缓存。"""
    if path in cache:
        return cache[path]
    try:
        entries = client.list_dir(owner, repo, path, ref)
    except GitLinkError:
        entries = []
    names = {str(e.get("name", "")).lower() for e in entries if e.get("name")}
    cache[path] = names
    return names


def check_repo(owner: str, repo: str, ref: str = "master",
               client: GitLinkClient | None = None) -> dict[str, Any]:
    """检测仓库的社区健康文件齐全度。"""
    client = client or GitLinkClient()
    dir_cache: dict[str, set[str]] = {}

    present: list[dict[str, Any]] = []
    missing: list[dict[str, Any]] = []
    score = 0
    max_score = 0

    for spec in HEALTH_FILES:
        max_score += spec["weight"]
        found_at = None
        for d in spec["dirs"]:
            names = _names_in_dir(client, owner, repo, d, ref, dir_cache)
            hit = next((c for c in spec["candidates"] if c in names), None)
            if hit:
                found_at = f"{d}/{hit}" if d else hit
                break
        record = {"key": spec["key"], "name": spec["name"],
                  "weight": spec["weight"], "critical": spec["critical"]}
        if found_at:
            record["path"] = found_at
            present.append(record)
            score += spec["weight"]
        else:
            missing.append(record)

    health = round(score / max_score * 100) if max_score else 0
    return {
        "owner": owner, "repo": repo,
        "health_score": health,
        "present": present,
        "missing": missing,
        "missing_critical": [m for m in missing if m["critical"]],
    }


# ---------------------------------------------------------------------------
# 模板生成
# ---------------------------------------------------------------------------

def _tpl_contributing(owner: str, repo: str) -> str:
    return f"""# 贡献指南

感谢你考虑为 {owner}/{repo} 做贡献！本指南帮助你顺利参与。

## 如何贡献

1. Fork 本仓库并克隆你的 Fork。
2. 新建分支：`git checkout -b feat/your-feature`
3. 进行修改，并确保通过现有测试。
4. 提交时使用清晰的提交信息（推荐 Conventional Commits，如 `feat:`、`fix:`、`docs:`）。
5. 推送到你的 Fork，并向本仓库发起 Pull Request。

## 提交 PR 前的检查清单

- [ ] 代码可以正常构建/运行
- [ ] 新增或修改的功能有对应测试
- [ ] 文档已同步更新
- [ ] PR 描述清楚说明了变更内容与动机

## 报告问题

通过 Issue 报告 Bug 或提出建议时，请尽量包含：复现步骤、期望行为、实际行为、运行环境。

## 行为准则

参与本项目即表示你同意遵守 [行为准则](CODE_OF_CONDUCT.md)。
"""


def _tpl_code_of_conduct(owner: str, repo: str) -> str:
    return """# 行为准则

## 我们的承诺

为营造开放、友好的社区环境，我们承诺：无论年龄、性别、经验水平、国籍、个人外貌、
种族或宗教，参与本项目的每个人都能获得免受骚扰的体验。

## 行为标准

有助于营造积极环境的行为包括：

- 使用友好和包容的语言
- 尊重不同的观点和经验
- 优雅地接受建设性批评
- 关注对社区最有利的事情

不可接受的行为包括：

- 使用性化的语言或图像
- 挑衅、侮辱或贬损性评论
- 公开或私下骚扰
- 未经许可发布他人的私人信息

## 执行

如遇违反行为准则的情况，可通过仓库 Issue 或维护者联系方式报告。所有投诉都会被审查与处理。

本准则改编自 Contributor Covenant。
"""


def _tpl_security(owner: str, repo: str) -> str:
    return f"""# 安全策略

## 报告漏洞

我们重视 {owner}/{repo} 的安全。如果你发现安全漏洞，请**不要**直接在公开 Issue 中披露。

请通过以下方式私下报告：

- 在 GitLink 上私信仓库维护者
- 或发送邮件至维护者邮箱（见仓库主页）

报告时请包含：漏洞描述、复现步骤、可能的影响范围。我们会尽快响应并在修复后致谢。

## 支持的版本

| 版本 | 是否支持 |
|------|:--------:|
| 最新发布版 | ✅ |
| 历史版本 | 视情况 |
"""


def _tpl_issue_template(owner: str, repo: str) -> str:
    return """---
name: Bug 报告 / 功能建议
about: 报告问题或提出新功能
---

## 类型

- [ ] Bug 报告
- [ ] 功能建议
- [ ] 问题咨询

## 描述

<!-- 清晰简洁地描述问题或建议 -->

## 复现步骤（Bug）

1.
2.
3.

## 期望行为

<!-- 你期望发生什么 -->

## 实际行为

<!-- 实际发生了什么 -->

## 运行环境

- 操作系统：
- 版本：
"""


def _tpl_pr_template(owner: str, repo: str) -> str:
    return """## 变更说明

<!-- 简述本次 PR 做了什么、为什么 -->

## 关联 Issue

<!-- 如 Closes #123 -->

## 变更类型

- [ ] Bug 修复
- [ ] 新功能
- [ ] 文档更新
- [ ] 重构 / 性能优化

## 检查清单

- [ ] 代码可正常构建/运行
- [ ] 已添加或更新测试
- [ ] 已更新相关文档
- [ ] 提交信息清晰规范
"""


def _tpl_changelog(owner: str, repo: str) -> str:
    return """# 更新日志

本项目所有重要变更都会记录在本文件中。

格式参考 [Keep a Changelog](https://keepachangelog.com/zh-CN/)，
版本号遵循[语义化版本](https://semver.org/lang/zh-CN/)。

## [Unreleased]

### 新增
### 修复
### 变更
"""


TEMPLATE_BUILDERS = {
    "contributing": ("CONTRIBUTING.md", _tpl_contributing),
    "code_of_conduct": ("CODE_OF_CONDUCT.md", _tpl_code_of_conduct),
    "security": ("SECURITY.md", _tpl_security),
    "issue_template": (".gitlink/issue_template.md", _tpl_issue_template),
    "pr_template": (".gitlink/pull_request_template.md", _tpl_pr_template),
    "changelog": ("CHANGELOG.md", _tpl_changelog),
}


def generate_templates(missing: list[dict[str, Any]], owner: str, repo: str,
                       out_dir: Path) -> list[str]:
    """为缺失且有模板的文件生成模板，返回生成的文件路径列表。"""
    generated: list[str] = []
    for m in missing:
        builder = TEMPLATE_BUILDERS.get(m["key"])
        if not builder:
            continue
        rel_path, fn = builder
        target = out_dir / rel_path
        target.parent.mkdir(parents=True, exist_ok=True)
        target.write_text(fn(owner, repo), encoding="utf-8")
        generated.append(str(target))
    return generated


def render_report(result: dict[str, Any]) -> str:
    """渲染社区健康文件体检报告（Markdown）。"""
    owner, repo = result["owner"], result["repo"]
    health = result["health_score"]
    bar = "█" * (health // 10) + "·" * (10 - health // 10)
    lines = [
        f"# 社区健康文件体检 — {owner}/{repo}",
        "",
        f"健康度评分：**{health}/100**  `{bar}`",
        "",
        "## 已具备",
        "",
    ]
    if result["present"]:
        for p in result["present"]:
            lines.append(f"- ✅ {p['name']}（`{p.get('path')}`）")
    else:
        lines.append("- （无）")
    lines += ["", "## 缺失", ""]
    if result["missing"]:
        for m in result["missing"]:
            mark = "❗" if m["critical"] else "⬜"
            tip = "（关键）" if m["critical"] else ""
            has_tpl = "，可自动生成模板" if m["key"] in TEMPLATE_BUILDERS else ""
            lines.append(f"- {mark} {m['name']}{tip}{has_tpl}")
    else:
        lines.append("- 🎉 全部齐全！")
    lines += ["", "## 建议", ""]
    if result["missing_critical"]:
        names = "、".join(m["name"] for m in result["missing_critical"])
        lines.append(f"1. 优先补齐关键文件：**{names}**。")
    gen_able = [m["name"] for m in result["missing"] if m["key"] in TEMPLATE_BUILDERS]
    if gen_able:
        lines.append(f"2. 以下文件可用本工具一键生成模板：{', '.join(gen_able)}。")
        lines.append("   运行：`python scaffold.py --owner %s --repo %s --generate`" % (owner, repo))
    if not result["missing"]:
        lines.append("社区健康文件已齐全，继续保持。")
    lines.append("")
    return "\n".join(lines)


def main(argv: list[str] | None = None) -> int:
    p = argparse.ArgumentParser(prog="gitlink-scaffold",
                                description="社区健康文件体检与模板生成")
    p.add_argument("--owner", help="仓库所有者")
    p.add_argument("--repo", help="仓库名称")
    p.add_argument("--slug", help="owner/repo 或完整 URL")
    p.add_argument("--ref", default="master", help="分支或标签，默认 master")
    p.add_argument("--generate", action="store_true", help="为缺失文件生成模板")
    p.add_argument("--output-dir", type=Path, default=Path("scaffold_out"),
                   help="模板输出目录（配合 --generate）")
    p.add_argument("--format", choices=["markdown", "json"], default="markdown")
    p.add_argument("--output", type=Path, help="报告输出文件")
    args = p.parse_args(argv)

    if args.slug:
        owner, repo = split_owner_repo(args.slug)
    elif args.owner and args.repo:
        owner, repo = args.owner, args.repo
    else:
        print("错误：请用 --owner/--repo 或 --slug 指定仓库。", file=sys.stderr)
        return 2

    try:
        result = check_repo(owner, repo, ref=args.ref)
    except GitLinkError as exc:
        print(f"采集失败：{exc}", file=sys.stderr)
        return 1

    generated: list[str] = []
    if args.generate:
        generated = generate_templates(result["missing"], owner, repo, args.output_dir)

    if args.format == "json":
        result["generated"] = generated
        out = json.dumps(result, ensure_ascii=False, indent=2)
    else:
        out = render_report(result)
        if generated:
            out += "\n## 已生成模板\n\n" + "\n".join(f"- `{g}`" for g in generated) + "\n"

    if args.output:
        args.output.parent.mkdir(parents=True, exist_ok=True)
        args.output.write_text(out, encoding="utf-8")
        print(f"已写入 {args.output}")
    else:
        print(out)
    if generated:
        print(f"\n已生成 {len(generated)} 个模板到 {args.output_dir}/", file=sys.stderr)
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
