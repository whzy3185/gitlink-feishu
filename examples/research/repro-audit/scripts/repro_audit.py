#!/usr/bin/env python3
"""科研项目复现性审计（repro-audit）。

面向科研代码仓库的确定性复现性评估：
1. 采集：`repo +tree` 获取仓库结构，`file +view --raw` 拉取 README，`release +list` 查发布
2. 评估：八个维度的复现性检查项（文档/许可证/引用/依赖/入口/数据说明/测试/版本固化）
3. 报告：0-100 评分卡 + A/B/C/D 等级 + 逐项修复建议
4. 回写：--apply 时用 `issue +create` 把报告作为改进 tracking issue 提交

纯标准库实现（Python >= 3.9），gitlink-cli 为唯一外部依赖。默认 dry-run。
"""

import argparse
import json
import re
import subprocess
import sys
from dataclasses import dataclass
from pathlib import Path


@dataclass
class CheckResult:
    name: str
    weight: int
    score: int  # 0..weight
    evidence: str
    advice: str


def run_cli(args, cli="gitlink-cli"):
    result = subprocess.run([cli, *args], capture_output=True, text=True, check=False)
    if result.returncode != 0:
        raise RuntimeError(
            f"gitlink-cli {' '.join(args)} 失败: {result.stderr.strip() or result.stdout.strip()}"
        )
    return result.stdout


def get_json(args, cli="gitlink-cli"):
    payload = json.loads(run_cli([*args, "--format", "json"], cli=cli))
    return payload.get("data", payload)


def list_entries(owner, repo, ref, path="", cli="gitlink-cli"):
    args = ["repo", "+tree", "--owner", owner, "--repo", repo]
    if ref:
        args += ["--ref", ref]
    if path:
        args += ["--path", path]
    data = get_json(args, cli=cli)
    entries = data.get("entries", data) if isinstance(data, dict) else data
    out = []
    if isinstance(entries, list):
        for e in entries:
            if isinstance(e, dict) and e.get("name"):
                out.append((e["name"], e.get("type", "")))
    return out


def fetch_readme(owner, repo, ref, names, cli="gitlink-cli"):
    for name in names:
        if re.match(r"(?i)^readme(\.|$)", name):
            args = ["file", "+view", "--owner", owner, "--repo", repo, "--path", name, "--raw"]
            if ref:
                args += ["--ref", ref]
            try:
                return name, run_cli(args, cli=cli)
            except RuntimeError:
                continue
    return None, ""


def count_releases(owner, repo, cli="gitlink-cli"):
    try:
        data = get_json(["release", "+list", "--owner", owner, "--repo", repo], cli=cli)
    except RuntimeError:
        return 0
    releases = data.get("releases", data) if isinstance(data, dict) else data
    return len(releases) if isinstance(releases, list) else 0


DEP_FILES = [
    "requirements.txt", "environment.yml", "environment.yaml", "Pipfile",
    "pyproject.toml", "setup.py", "go.mod", "package.json", "Cargo.toml",
    "pom.xml", "build.gradle", "CMakeLists.txt", "DESCRIPTION", "renv.lock",
]
ENTRY_FILES = ["Makefile", "makefile", "run.sh", "train.sh", "main.py", "run.py", "train.py", "Dockerfile", "docker-compose.yml"]
CITATION_FILES = ["CITATION.cff", "CITATION", "CITATION.bib", "citation.bib"]
LICENSE_RE = re.compile(r"(?i)^(license|licence|copying)(\.|$)")
DATA_HINT_RE = re.compile(r"(?i)\b(dataset|data set|数据集|数据来源|download.*data|data/)\b")
REPRO_HINT_RE = re.compile(r"(?i)(reproduc|复现|实验设置|experiment setup|how to run|usage|快速开始|quick\s?start)")


def audit(names, dirs, readme_text, release_count):
    """确定性打分：names=顶层文件名，dirs=顶层目录名。返回 CheckResult 列表。"""
    nameset = set(names)
    results = []

    def add(name, weight, ok, evidence, advice, partial=None):
        score = weight if ok else (partial if partial else 0)
        results.append(CheckResult(name, weight, score, evidence, "" if ok else advice))

    readme_name = next((n for n in names if re.match(r"(?i)^readme(\.|$)", n)), None)
    has_usage = bool(readme_text and REPRO_HINT_RE.search(readme_text))
    add("README 与运行说明", 15, bool(readme_name and has_usage),
        f"README={readme_name or '缺失'}；运行/复现章节={'有' if has_usage else '未检出'}",
        "补充 README 并加入 How to run / 复现步骤章节",
        partial=8 if readme_name else 0)

    lic = next((n for n in names if LICENSE_RE.match(n)), None)
    add("开源许可证", 15, bool(lic), f"许可证文件={lic or '缺失'}",
        "添加 LICENSE 文件（科研代码推荐 MIT/Apache-2.0/BSD）")

    cit = next((n for n in names if n in CITATION_FILES), None)
    cite_in_readme = bool(readme_text and re.search(r"(?i)(citation|引用|bibtex|@(article|inproceedings))", readme_text))
    add("引用信息", 10, bool(cit or cite_in_readme),
        f"CITATION 文件={cit or '无'}；README 引用章节={'有' if cite_in_readme else '无'}",
        "添加 CITATION.cff 或在 README 中给出 BibTeX")

    dep = next((n for n in names if n in DEP_FILES), None)
    add("依赖清单（环境固化）", 15, bool(dep), f"依赖文件={dep or '缺失'}",
        "添加 requirements.txt / environment.yml 等依赖清单并固定版本")

    entry = next((n for n in names if n in ENTRY_FILES), None)
    script_dir = next((d for d in dirs if d in ("scripts", "bin")), None)
    add("运行入口", 10, bool(entry or script_dir),
        f"入口={entry or script_dir or '未检出'}",
        "提供 Makefile / run.sh / main.py 等一键运行入口")

    data_dir = next((d for d in dirs if re.match(r"(?i)^(data|datasets?)$", d)), None)
    data_in_readme = bool(readme_text and DATA_HINT_RE.search(readme_text))
    add("数据可得性说明", 10, bool(data_dir or data_in_readme),
        f"数据目录={data_dir or '无'}；README 数据说明={'有' if data_in_readme else '无'}",
        "在 README 中说明数据集来源、获取方式与预处理步骤")

    test_dir = next((d for d in dirs if re.match(r"(?i)^tests?$", d)), None)
    add("测试/验证代码", 10, bool(test_dir), f"测试目录={test_dir or '无'}",
        "添加 tests/ 验证关键结果可复算")

    add("版本发布（成果固化）", 15, release_count > 0, f"release 数={release_count}",
        "为论文对应的代码状态打 tag 并创建 release")

    return results


GRADE = [(85, "A（可复现性良好）"), (70, "B（基本可复现）"), (50, "C（存在明显缺口）"), (0, "D（复现困难）")]


def render_report(owner, repo, ref, results):
    total = sum(r.score for r in results)
    grade = next(g for t, g in GRADE if total >= t)
    lines = [
        f"# 科研复现性审计报告：{owner}/{repo}（ref: {ref or '默认分支'}）",
        "",
        f"**总分：{total}/100 — 等级 {grade}**",
        "",
        "| 检查项 | 得分 | 证据 | 建议 |",
        "|--------|------|------|------|",
    ]
    for r in results:
        lines.append(f"| {r.name} | {r.score}/{r.weight} | {r.evidence} | {r.advice or '—'} |")
    lines += ["", "---", "*由 repro-audit 生成（确定性检查，同输入同输出）。*"]
    return "\n".join(lines), total


def audit_repo(owner, repo, ref, out_dir, cli, apply_issue=False, echo=True):
    """审计单个仓库，落盘报告，返回总分。"""
    entries = list_entries(owner, repo, ref, cli=cli)
    names = [n for n, t in entries if t != "dir"]
    dirs = [n for n, t in entries if t == "dir"]
    _, readme_text = fetch_readme(owner, repo, ref, names, cli=cli)
    releases = count_releases(owner, repo, cli=cli)

    results = audit(names, dirs, readme_text, releases)
    report, total = render_report(owner, repo, ref, results)

    path = out_dir / f"repro-audit-{owner}-{repo}.md"
    path.write_text(report, encoding="utf-8")
    if echo:
        print(report)
    print(f"\n报告已保存：{path}", file=sys.stderr)

    if apply_issue:
        run_cli([
            "issue", "+create", "--owner", owner, "--repo", repo,
            "--title", f"[repro-audit] 复现性审计报告（{total}/100）",
            "--body", report, "--format", "json",
        ], cli=cli)
        print("已创建 tracking issue。", file=sys.stderr)
    return total


def read_repos_file(path):
    """读取批量仓库清单：每行 owner/repo，# 开头为注释。"""
    repos = []
    for line in Path(path).read_text(encoding="utf-8").splitlines():
        line = line.strip()
        if not line or line.startswith("#"):
            continue
        owner, _, repo = line.partition("/")
        if not owner or not repo:
            raise ValueError(f"无效的仓库行（应为 owner/repo）：{line}")
        repos.append((owner, repo))
    return repos


def render_summary(rows, failures=()):
    """批量审计汇总表（确定性：按得分降序、同分按名称；失败仓库单独列出）。"""
    rows = sorted(rows, key=lambda r: (-r[1], r[0]))
    lines = ["# 批量复现性审计汇总", "", "| 仓库 | 得分 | 等级 |", "|------|------|------|"]
    for name, total in rows:
        grade = next(g for t, g in GRADE if total >= t)
        lines.append(f"| {name} | {total}/100 | {grade} |")
    if failures:
        lines += ["", "## 无法审计的仓库", ""]
        lines += [f"- {name}：{reason}" for name, reason in sorted(failures)]
    return "\n".join(lines) + "\n"


def main():
    parser = argparse.ArgumentParser(description="科研项目复现性审计")
    parser.add_argument("--owner")
    parser.add_argument("--repo")
    parser.add_argument("--repos-file", help="批量审计清单文件（每行 owner/repo，# 注释）")
    parser.add_argument("--ref", default="")
    parser.add_argument("--apply", action="store_true", help="把报告作为 tracking issue 回写（默认 dry-run）")
    parser.add_argument("--output-dir", default="outputs")
    parser.add_argument("--cli", default="gitlink-cli")
    args = parser.parse_args()

    if not args.repos_file and not (args.owner and args.repo):
        parser.error("需要 --owner 与 --repo，或 --repos-file")

    out_dir = Path(args.output_dir)
    out_dir.mkdir(parents=True, exist_ok=True)

    if args.repos_file:
        rows, failures = [], []
        for owner, repo in read_repos_file(args.repos_file):
            name = f"{owner}/{repo}"
            try:
                total = audit_repo(owner, repo, args.ref, out_dir, args.cli,
                                   apply_issue=args.apply, echo=False)
                rows.append((name, total))
            except RuntimeError as exc:
                print(f"跳过 {name}：{exc}", file=sys.stderr)
                failures.append((name, str(exc)))
        summary = render_summary(rows, failures)
        summary_path = out_dir / "repro-audit-summary.md"
        summary_path.write_text(summary, encoding="utf-8")
        print(summary)
        print(f"汇总已保存：{summary_path}", file=sys.stderr)
        return 0 if not failures and all(t >= 70 for _, t in rows) else 2

    total = audit_repo(args.owner, args.repo, args.ref, out_dir, args.cli,
                       apply_issue=args.apply)
    return 0 if total >= 70 else 2


if __name__ == "__main__":
    sys.exit(main())
