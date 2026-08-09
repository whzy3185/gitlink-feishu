"""gitlink-deps：项目依赖追踪。

扫描一个 GitLink 仓库的依赖声明文件（go.mod / package.json /
requirements.txt / pom.xml / Cargo.toml / pyproject.toml 等），
解析出依赖清单、数量统计、技术栈识别与潜在风险提示，生成依赖报告。

数据来自 GitLink 公开 API（只读），无需登录。

用法：
    python deps.py --owner Gitlink --repo gitlink-cli
    python deps.py --owner Gitlink --repo gitlink-cli --format json
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

# 依赖文件 → 生态映射
MANIFESTS = {
    "go.mod": "Go",
    "package.json": "Node.js",
    "requirements.txt": "Python",
    "pyproject.toml": "Python",
    "Pipfile": "Python",
    "pom.xml": "Java (Maven)",
    "build.gradle": "Java (Gradle)",
    "Cargo.toml": "Rust",
    "composer.json": "PHP",
    "Gemfile": "Ruby",
}


# ---------------------------------------------------------------------------
# 各类清单解析器（纯函数，输入文本，输出依赖列表）
# ---------------------------------------------------------------------------

def parse_go_mod(text: str) -> list[dict[str, str]]:
    """解析 go.mod 的 require 块。"""
    deps: list[dict[str, str]] = []
    in_block = False
    for line in text.splitlines():
        s = line.strip()
        if s.startswith("require ("):
            in_block = True
            continue
        if in_block and s == ")":
            in_block = False
            continue
        # require 块内，或单行 require
        m = re.match(r"(?:require\s+)?([\w./\-]+)\s+(v[\w.\-+]+)", s)
        if m and ("/" in m.group(1)):
            deps.append({
                "name": m.group(1),
                "version": m.group(2),
                "indirect": "// indirect" in s,
            })
    return deps


def parse_package_json(text: str) -> list[dict[str, str]]:
    """解析 package.json 的 dependencies 与 devDependencies。"""
    deps: list[dict[str, str]] = []
    try:
        data = json.loads(text)
    except json.JSONDecodeError:
        return deps
    for field, dev in (("dependencies", False), ("devDependencies", True)):
        block = data.get(field)
        if isinstance(block, dict):
            for name, ver in block.items():
                deps.append({"name": name, "version": str(ver), "indirect": dev})
    return deps


def parse_requirements(text: str) -> list[dict[str, str]]:
    """解析 requirements.txt。"""
    deps: list[dict[str, str]] = []
    for line in text.splitlines():
        s = line.strip()
        if not s or s.startswith("#") or s.startswith("-"):
            continue
        m = re.match(r"([A-Za-z0-9_.\-]+)\s*([=<>!~]=?.*)?", s)
        if m:
            deps.append({
                "name": m.group(1),
                "version": (m.group(2) or "").strip() or "*",
                "indirect": False,
            })
    return deps


def parse_cargo_toml(text: str) -> list[dict[str, str]]:
    """解析 Cargo.toml 的 [dependencies] 段（简化）。"""
    deps: list[dict[str, str]] = []
    in_deps = False
    for line in text.splitlines():
        s = line.strip()
        if s.startswith("["):
            in_deps = "dependencies" in s
            continue
        if in_deps and "=" in s and not s.startswith("#"):
            name = s.split("=", 1)[0].strip()
            ver_part = s.split("=", 1)[1].strip().strip('"')
            if name:
                deps.append({"name": name, "version": ver_part or "*", "indirect": False})
    return deps


def parse_pom_xml(text: str) -> list[dict[str, str]]:
    """解析 pom.xml 的 <dependency> 块（正则简化）。"""
    deps: list[dict[str, str]] = []
    for block in re.findall(r"<dependency>(.*?)</dependency>", text, re.DOTALL):
        gid = re.search(r"<groupId>(.*?)</groupId>", block)
        aid = re.search(r"<artifactId>(.*?)</artifactId>", block)
        ver = re.search(r"<version>(.*?)</version>", block)
        if aid:
            name = f"{gid.group(1)}:{aid.group(1)}" if gid else aid.group(1)
            deps.append({"name": name.strip(),
                         "version": ver.group(1).strip() if ver else "*",
                         "indirect": False})
    return deps


PARSERS = {
    "go.mod": parse_go_mod,
    "package.json": parse_package_json,
    "requirements.txt": parse_requirements,
    "Cargo.toml": parse_cargo_toml,
    "pom.xml": parse_pom_xml,
}


def scan(owner: str, repo: str, ref: str = "master",
         client: GitLinkClient | None = None) -> dict[str, Any]:
    """扫描仓库根目录的依赖文件并解析。"""
    client = client or GitLinkClient()

    # 列根目录，找出存在的清单文件
    try:
        root_entries = client.list_dir(owner, repo, "", ref)
    except GitLinkError:
        root_entries = []
    root_names = {str(e.get("name", "")) for e in root_entries}

    manifests_found: list[dict[str, Any]] = []
    all_deps: list[dict[str, Any]] = []
    ecosystems: set[str] = set()

    for fname, eco in MANIFESTS.items():
        if fname not in root_names:
            continue
        ecosystems.add(eco)
        parser = PARSERS.get(fname)
        deps: list[dict[str, str]] = []
        if parser:
            content = client.file_content(owner, repo, fname, ref)
            if content:
                deps = parser(content)
        for d in deps:
            d["manifest"] = fname
            d["ecosystem"] = eco
        all_deps.extend(deps)
        manifests_found.append({
            "file": fname, "ecosystem": eco, "parsed": parser is not None,
            "count": len(deps),
        })

    direct = [d for d in all_deps if not d.get("indirect")]
    indirect = [d for d in all_deps if d.get("indirect")]

    return {
        "owner": owner, "repo": repo,
        "ecosystems": sorted(ecosystems),
        "manifests": manifests_found,
        "total_deps": len(all_deps),
        "direct_count": len(direct),
        "indirect_count": len(indirect),
        "dependencies": all_deps,
        "risks": _assess_risks(all_deps, manifests_found),
    }


def _assess_risks(deps: list[dict[str, Any]], manifests: list[dict[str, Any]]) -> list[str]:
    """基于依赖清单给出风险与改进提示。"""
    risks: list[str] = []
    if not manifests:
        risks.append("未发现依赖声明文件，无法分析依赖（可能是纯文档/资源仓库，或依赖文件不在根目录）。")
        return risks

    # 未锁定版本的依赖
    unpinned = [d for d in deps if d.get("version") in ("*", "", "latest")
                or str(d.get("version", "")).startswith("^")
                or str(d.get("version", "")).startswith("~")]
    if unpinned:
        risks.append(f"有 {len(unpinned)} 个依赖未锁定精确版本（使用 ^ / ~ / * / latest），"
                     "可能导致构建不可复现，建议在锁文件中固定版本。")

    # 依赖数量过多
    direct = [d for d in deps if not d.get("indirect")]
    if len(direct) > 50:
        risks.append(f"直接依赖较多（{len(direct)} 个），建议定期审查是否都必要，减少供应链攻击面。")

    if not risks:
        risks.append("未发现明显的依赖风险，依赖声明较为规范。")
    return risks


def render_report(result: dict[str, Any]) -> str:
    """渲染依赖报告（Markdown）。"""
    owner, repo = result["owner"], result["repo"]
    lines = [
        f"# 依赖追踪报告 — {owner}/{repo}",
        "",
        f"- 技术栈：{', '.join(result['ecosystems']) or '未识别'}",
        f"- 依赖声明文件：{len(result['manifests'])} 个",
        f"- 依赖总数：{result['total_deps']}（直接 {result['direct_count']} / 间接 {result['indirect_count']}）",
        "",
    ]
    if result["manifests"]:
        lines += ["## 依赖声明文件", "", "| 文件 | 生态 | 解析依赖数 |", "|------|------|:----------:|"]
        for m in result["manifests"]:
            lines.append(f"| `{m['file']}` | {m['ecosystem']} | {m['count']} |")
        lines.append("")

    direct = [d for d in result["dependencies"] if not d.get("indirect")]
    if direct:
        lines += ["## 直接依赖（前 30）", "", "| 依赖 | 版本 | 生态 |", "|------|------|------|"]
        for d in direct[:30]:
            lines.append(f"| `{d['name']}` | {d['version']} | {d['ecosystem']} |")
        if len(direct) > 30:
            lines.append(f"| … | 其余 {len(direct) - 30} 个 | |")
        lines.append("")

    lines += ["## 风险与建议", ""]
    for i, r in enumerate(result["risks"], 1):
        lines.append(f"{i}. {r}")
    lines.append("")
    return "\n".join(lines)


def main(argv: list[str] | None = None) -> int:
    p = argparse.ArgumentParser(prog="gitlink-deps", description="项目依赖追踪")
    p.add_argument("--owner", help="仓库所有者")
    p.add_argument("--repo", help="仓库名称")
    p.add_argument("--slug", help="owner/repo 或完整 URL")
    p.add_argument("--ref", default="master", help="分支或标签，默认 master")
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
        result = scan(owner, repo, ref=args.ref)
    except GitLinkError as exc:
        print(f"采集失败：{exc}", file=sys.stderr)
        return 1

    out = (json.dumps(result, ensure_ascii=False, indent=2)
           if args.format == "json" else render_report(result))

    if args.output:
        args.output.parent.mkdir(parents=True, exist_ok=True)
        args.output.write_text(out, encoding="utf-8")
        print(f"已写入 {args.output}")
    else:
        print(out)
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
