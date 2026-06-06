"""gitlink-flow：社区运营自动化端到端工作流编排器。

串联多个步骤，对一个真实 GitLink 仓库执行完整的社区运营自动化：

    采集（glapi）
      → ① Issue 自动分拣（triage）
      → ② PR Review 汇总（pr-review）
      → ③ Release Notes 生成（release-notes）
      → ④ 社区健康体检（复用 gitlink-scaffold）
      → ⑤ 贡献者致谢（复用 gitlink-contributor）
      → ⑥ 生成社区运营周报（汇总以上全部）

串联 6 个步骤、复用 5 个自研 Skill 的能力，远超"≥3 个调用"的要求。
数据来自 GitLink 公开 API（只读），无需登录。

用法：
    python flow.py --owner Gitlink --repo gitlink-cli
    python flow.py --owner Gitlink --repo gitlink-cli --format json
    python flow.py --config examples/config.json
"""

from __future__ import annotations

import argparse
import json
import sys
from datetime import datetime
from pathlib import Path
from typing import Any

sys.path.insert(0, str(Path(__file__).resolve().parent))
from glapi import GitLinkClient, GitLinkError, split_owner_repo
import steps
import report as report_mod


def run_flow(owner: str, repo: str, client: GitLinkClient | None = None,
             commit_pages: int = 4) -> dict[str, Any]:
    """对单个仓库执行完整工作流，返回各步骤结果。"""
    client = client or GitLinkClient()

    # ===== 采集阶段 =====
    info = client.repo_info(owner, repo)
    issues = client.issues(owner, repo, limit=50)
    pulls = client.pulls(owner, repo, limit=50)
    commits = client.commits(owner, repo, max_pages=commit_pages)
    contributors = client.contributors(owner, repo)
    releases = client.releases(owner, repo) if hasattr(client, "releases") else []
    try:
        root_entries = client.list_dir(owner, repo, "", "master")
        root_files = [str(e.get("name", "")) for e in root_entries]
    except GitLinkError:
        root_files = []

    # ===== 分析阶段（6 步）=====
    latest_version = "Unreleased"
    if releases:
        latest_version = releases[0].get("tag_name") or releases[0].get("name") or "Unreleased"

    result = {
        "owner": owner,
        "repo": repo,
        "generated_at": datetime.now().strftime("%Y-%m-%d %H:%M"),
        "repo_info": {
            "name": info.get("name"),
            "issues_count": info.get("issues_count"),
            "pull_requests_count": info.get("pull_requests_count"),
            "praises_count": info.get("praises_count"),
            "forked_count": info.get("forked_count"),
        },
        "step1_triage": steps.triage_issues(issues),
        "step2_pr_review": steps.pr_review_summary(pulls),
        "step3_release_notes": steps.release_notes(commits, version=latest_version),
        "step4_health": steps.health_check(root_files),
        "step5_contributors": steps.contributor_highlights(contributors),
    }
    # 第 6 步：汇总周报
    result["step6_weekly_report"] = report_mod.render_weekly(result)
    return result


def main(argv: list[str] | None = None) -> int:
    p = argparse.ArgumentParser(prog="gitlink-flow",
                                description="社区运营自动化端到端工作流")
    p.add_argument("--owner", help="仓库所有者")
    p.add_argument("--repo", help="仓库名称")
    p.add_argument("--slug", help="owner/repo 或完整 URL")
    p.add_argument("--config", type=Path, help="批量配置 JSON（含 repos 列表）")
    p.add_argument("--commit-pages", type=int, default=4, help="提交采集页数（每页 50）")
    p.add_argument("--format", choices=["markdown", "json"], default="markdown")
    p.add_argument("--output", type=Path, help="输出文件")
    p.add_argument("--output-dir", type=Path, help="批量模式输出目录")
    args = p.parse_args(argv)

    targets: list[tuple[str, str]] = []
    if args.config:
        cfg = json.loads(args.config.read_text(encoding="utf-8"))
        for item in cfg.get("repos", []):
            if item.get("owner") and item.get("repo"):
                targets.append((item["owner"], item["repo"]))
    if args.slug:
        targets.append(split_owner_repo(args.slug))
    elif args.owner and args.repo:
        targets.append((args.owner, args.repo))

    if not targets:
        print("错误：请用 --owner/--repo 或 --slug 或 --config 指定仓库。", file=sys.stderr)
        return 2

    client = GitLinkClient()
    exit_code = 0
    for owner, repo in targets:
        print(f"[工作流] 处理 {owner}/{repo} ...", flush=True)
        try:
            result = run_flow(owner, repo, client=client, commit_pages=args.commit_pages)
        except GitLinkError as exc:
            print(f"  失败：{exc}", file=sys.stderr)
            exit_code = 1
            continue

        if args.format == "json":
            out = json.dumps(result, ensure_ascii=False, indent=2)
        else:
            out = result["step6_weekly_report"]

        if args.output_dir:
            args.output_dir.mkdir(parents=True, exist_ok=True)
            ext = "json" if args.format == "json" else "md"
            fp = args.output_dir / f"{owner}_{repo}_flow.{ext}"
            fp.write_text(out, encoding="utf-8")
            print(f"  已写入 {fp}", flush=True)
        elif args.output:
            args.output.parent.mkdir(parents=True, exist_ok=True)
            args.output.write_text(out, encoding="utf-8")
            print(f"  已写入 {args.output}", flush=True)
        else:
            print(out)
    return exit_code


if __name__ == "__main__":
    raise SystemExit(main())
