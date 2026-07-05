#!/usr/bin/env python3
"""中英文档一致性守护工作流（doc-sync-automation）。

采集 → 检测 → 报告 → 回写（可选）：
1. 采集：通过 gitlink-cli 发现双语文档对并拉取两版内容
2. 检测：确定性结构比对（章节大纲 / 代码块 / 表格行数 / 版本号）
3. 报告：输出分级（严重/中等/轻微）Markdown 漂移报告
4. 回写：--apply 时把报告作为 tracking issue 提交到 GitLink

纯标准库实现（Python >= 3.9），gitlink-cli 为唯一外部依赖。
默认 dry-run，不写远端。
"""

import argparse
import json
import re
import subprocess
import sys
from dataclasses import dataclass, field
from pathlib import Path

# 文档对命名约定：主文档 -> 可能的翻译文档
PAIR_PATTERNS = [
    ("README.md", ["README.zh-CN.md", "README_zh.md", "README.zh.md", "README-zh.md"]),
    ("CONTRIBUTING.md", ["CONTRIBUTING.zh-CN.md", "CONTRIBUTING_zh.md"]),
    ("CHANGELOG.md", ["CHANGELOG.zh-CN.md"]),
]

SEVERITY_ORDER = {"严重": 0, "中等": 1, "轻微": 2}
SEVERITY_ICON = {"严重": "🔴", "中等": "🟡", "轻微": "🟢"}


@dataclass
class Finding:
    severity: str  # 严重 | 中等 | 轻微
    category: str
    location: str
    detail: str


@dataclass
class DocStructure:
    headings: list = field(default_factory=list)  # [(level, text)]
    code_blocks: int = 0
    code_lines: int = 0
    table_rows: int = 0
    versions: list = field(default_factory=list)  # 形如 Go 1.26 / v0.2.0 的版本号


def run_cli(args, cli="gitlink-cli"):
    """调用 gitlink-cli 并返回 stdout 文本。"""
    result = subprocess.run(
        [cli, *args], capture_output=True, text=True, check=False
    )
    if result.returncode != 0:
        raise RuntimeError(
            f"gitlink-cli {' '.join(args)} 失败: {result.stderr.strip() or result.stdout.strip()}"
        )
    return result.stdout


def fetch_file(owner, repo, path, ref, cli="gitlink-cli"):
    args = ["file", "+view", "--owner", owner, "--repo", repo, "--path", path, "--raw"]
    if ref:
        args += ["--ref", ref]
    return run_cli(args, cli=cli)


def list_root_entries(owner, repo, ref, cli="gitlink-cli"):
    args = ["repo", "+tree", "--owner", owner, "--repo", repo, "--format", "json"]
    if ref:
        args += ["--ref", ref]
    out = run_cli(args, cli=cli)
    payload = json.loads(out)
    data = payload.get("data", payload)
    if isinstance(data, str):
        data = json.loads(data)
    entries = data.get("entries", data) if isinstance(data, dict) else data
    names = []
    if isinstance(entries, list):
        for e in entries:
            if isinstance(e, dict) and e.get("name"):
                names.append(e["name"])
    return names


def discover_pairs(names):
    """按命名约定从文件名列表中发现文档对。"""
    nameset = set(names)
    pairs = []
    for base, translations in PAIR_PATTERNS:
        if base not in nameset:
            continue
        for t in translations:
            if t in nameset:
                pairs.append((base, t))
                break
    return pairs


VERSION_RE = re.compile(r"\b(?:go|node(?:\.js)?|python|v)\s?(\d+\.\d+(?:\.\d+)?)\b", re.I)


def parse_structure(text):
    """提取文档结构：标题大纲、代码块、表格行、版本号。"""
    s = DocStructure()
    in_code = False
    code_lines = 0
    for line in text.splitlines():
        stripped = line.strip()
        if stripped.startswith("```"):
            if in_code:
                s.code_blocks += 1
                s.code_lines += code_lines
                code_lines = 0
            in_code = not in_code
            continue
        if in_code:
            code_lines += 1
            continue
        m = re.match(r"^(#{1,6})\s+(.*)$", stripped)
        if m:
            s.headings.append((len(m.group(1)), m.group(2).strip()))
            continue
        if stripped.startswith("|") and stripped.endswith("|") and not re.match(r"^\|[\s:|-]+\|$", stripped):
            s.table_rows += 1
        s.versions.extend(v for v in VERSION_RE.findall(line))
    return s


def compare_structures(base_path, trans_path, base, trans):
    """确定性漂移检测，返回 Finding 列表。"""
    findings = []

    # 1. 章节数量漂移（结构不对齐 = 严重信号）
    b_top = [h for h in base.headings if h[0] <= 2]
    t_top = [h for h in trans.headings if h[0] <= 2]
    if len(b_top) != len(t_top):
        more, fewer = (base_path, trans_path) if len(b_top) > len(t_top) else (trans_path, base_path)
        findings.append(Finding(
            "严重", "章节数不一致", "一级/二级标题",
            f"{more} 有 {max(len(b_top), len(t_top))} 节，{fewer} 只有 {min(len(b_top), len(t_top))} 节，疑似缺失章节",
        ))

    # 2. 代码块漂移（示例不同步 = 中等）
    if base.code_blocks != trans.code_blocks:
        findings.append(Finding(
            "中等", "代码块数不一致", "全文代码示例",
            f"{base_path} 有 {base.code_blocks} 个代码块，{trans_path} 有 {trans.code_blocks} 个",
        ))
    elif abs(base.code_lines - trans.code_lines) > max(5, base.code_lines // 20):
        findings.append(Finding(
            "中等", "代码行数漂移", "全文代码示例",
            f"代码行数 {base.code_lines} vs {trans.code_lines}，差异超过 5%",
        ))

    # 3. 表格行数漂移（功能表滞后 = 中等）
    if base.table_rows != trans.table_rows:
        findings.append(Finding(
            "中等", "表格行数不一致", "全文表格",
            f"{base_path} 共 {base.table_rows} 行表格，{trans_path} 共 {trans.table_rows} 行，疑似功能表滞后",
        ))

    # 4. 版本号漂移（轻微）
    b_ver, t_ver = sorted(set(base.versions)), sorted(set(trans.versions))
    if b_ver != t_ver:
        only_b = [v for v in b_ver if v not in t_ver]
        only_t = [v for v in t_ver if v not in b_ver]
        findings.append(Finding(
            "轻微", "版本号不一致", "安装/依赖说明",
            f"仅 {base_path} 出现: {only_b or '无'}；仅 {trans_path} 出现: {only_t or '无'}",
        ))

    findings.sort(key=lambda f: SEVERITY_ORDER[f.severity])
    return findings


def render_report(owner, repo, ref, results):
    lines = [f"# 文档一致性报告：{owner}/{repo}（ref: {ref or '默认分支'}）", ""]
    total = sum(len(f) for _, _, f in results)
    if total == 0:
        lines.append("✅ 所有文档对结构一致，未检测到漂移。")
    for base_path, trans_path, findings in results:
        lines.append(f"## {base_path} ⇄ {trans_path}")
        lines.append("")
        if not findings:
            lines.append("✅ 无漂移。")
            lines.append("")
            continue
        lines.append("| 等级 | 类型 | 位置 | 说明 |")
        lines.append("|------|------|------|------|")
        for f in findings:
            lines.append(f"| {SEVERITY_ICON[f.severity]} {f.severity} | {f.category} | {f.location} | {f.detail} |")
        lines.append("")
    lines.append("---")
    lines.append("*由 doc-sync-automation 工作流生成（确定性结构比对，同输入同输出）。*")
    return "\n".join(lines)


def create_tracking_issue(owner, repo, report, cli="gitlink-cli"):
    out = run_cli([
        "issue", "+create", "--owner", owner, "--repo", repo,
        "--title", "[doc-sync] 中英文档漂移报告",
        "--body", report,
        "--format", "json",
    ], cli=cli)
    return out


def main():
    parser = argparse.ArgumentParser(description="中英文档一致性守护工作流")
    parser.add_argument("--owner", required=True)
    parser.add_argument("--repo", required=True)
    parser.add_argument("--ref", default="")
    parser.add_argument("--pair", action="append", default=[],
                        help="手动指定文档对，格式 base.md:translation.md，可多次")
    parser.add_argument("--apply", action="store_true",
                        help="把漂移报告作为 tracking issue 回写到 GitLink（默认 dry-run）")
    parser.add_argument("--output-dir", default="outputs")
    parser.add_argument("--cli", default="gitlink-cli")
    args = parser.parse_args()

    if args.pair:
        pairs = [tuple(p.split(":", 1)) for p in args.pair]
    else:
        names = list_root_entries(args.owner, args.repo, args.ref, cli=args.cli)
        pairs = discover_pairs(names)
        if not pairs:
            print("未发现双语文档对（可用 --pair 手动指定）", file=sys.stderr)
            return 1

    results = []
    for base_path, trans_path in pairs:
        base_text = fetch_file(args.owner, args.repo, base_path, args.ref, cli=args.cli)
        trans_text = fetch_file(args.owner, args.repo, trans_path, args.ref, cli=args.cli)
        findings = compare_structures(
            base_path, trans_path, parse_structure(base_text), parse_structure(trans_text)
        )
        results.append((base_path, trans_path, findings))

    report = render_report(args.owner, args.repo, args.ref, results)
    out_dir = Path(args.output_dir)
    out_dir.mkdir(parents=True, exist_ok=True)
    report_path = out_dir / f"doc-sync-{args.owner}-{args.repo}.md"
    report_path.write_text(report, encoding="utf-8")
    print(report)
    print(f"\n报告已保存：{report_path}", file=sys.stderr)

    severe = sum(1 for _, _, fs in results for f in fs if f.severity == "严重")
    if args.apply and any(fs for _, _, fs in results):
        create_tracking_issue(args.owner, args.repo, report, cli=args.cli)
        print("已创建 tracking issue。", file=sys.stderr)

    return 2 if severe else 0


if __name__ == "__main__":
    sys.exit(main())
