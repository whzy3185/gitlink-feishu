#!/usr/bin/env python3
"""科研辅助工具模板：采集 → 指标 → 报告 三段式骨架（纯标准库，Python >= 3.9）。

自带可运行示例：仓库科研活跃度快照。
改造为自己的工具时只需替换 collect() / compute_metrics() / 报告模板三处。
"""

from __future__ import annotations

import argparse
import json
import subprocess
import sys
from datetime import datetime, timedelta
from pathlib import Path

CLI = "gitlink-cli"
DATE_FMT = "%Y-%m-%d %H:%M"
TEMPLATE = Path(__file__).resolve().parent.parent / "report-template.md"


def run_cli(args: list[str]) -> dict:
    out = subprocess.run([CLI, *args, "--format", "json"], capture_output=True, text=True)
    if out.returncode != 0:
        raise RuntimeError(f"{CLI} {' '.join(args)} failed: {out.stderr.strip() or out.stdout.strip()}")
    payload = json.loads(out.stdout)
    if not payload.get("ok", False):
        raise RuntimeError(f"{CLI} {' '.join(args)} returned error")
    return payload.get("data") or {}


# ---------- 1. 采集（按需替换） ----------

def collect(owner: str, repo: str) -> dict:
    open_issues = run_cli(["issue", "+list", "--owner", owner, "--repo", repo, "--state", "open", "--limit", "50"])
    prs = run_cli(["pr", "+list", "--owner", owner, "--repo", repo, "--limit", "1"])
    branches = run_cli(["branch", "+list", "--owner", owner, "--repo", repo])
    return {"open_issues": open_issues, "prs": prs, "branches": branches}


# ---------- 2. 指标（确定性纯函数，单测覆盖） ----------

def compute_metrics(raw: dict, now: datetime, active_days: int) -> dict:
    issues_data = raw["open_issues"]
    open_count = issues_data.get("open_count") or len(issues_data.get("issues") or [])
    closed_count = issues_data.get("closed_count") or 0
    total = open_count + closed_count
    close_rate = f"{closed_count / total:.0%}" if total else "n/a"

    recently_active = False
    for issue in issues_data.get("issues") or []:
        raw_date = issue.get("updated_at") or issue.get("created_at") or ""
        try:
            if now - datetime.strptime(raw_date, DATE_FMT) <= timedelta(days=active_days):
                recently_active = True
                break
        except ValueError:
            continue

    prs = raw["prs"]
    branches = raw["branches"]
    branch_list = branches if isinstance(branches, list) else branches.get("branches") or []
    return {
        "open_issues": open_count,
        "closed_issues": closed_count,
        "close_rate": close_rate,
        "open_prs": prs.get("open_count") or prs.get("total_count") or 0,
        "branches": len(branch_list),
        "recently_active": "是" if recently_active else "否",
    }


def conclude(metrics: dict, active_days: int) -> str:
    if metrics["recently_active"] == "是":
        return f"仓库处于活跃维护状态（近 {active_days} 天内有 issue 更新），适合作为科研协作/复现对象。"
    return f"仓库近 {active_days} 天无 issue 更新，科研复用前建议先与维护者确认项目状态。"


# ---------- 3. 报告渲染 ----------

def render(owner: str, repo: str, metrics: dict, now: datetime, active_days: int) -> str:
    return TEMPLATE.read_text(encoding="utf-8").format(
        owner=owner,
        repo=repo,
        generated_at=now.strftime("%Y-%m-%d %H:%M"),
        conclusion=conclude(metrics, active_days),
        active_days=active_days,
        **metrics,
    )


def main() -> int:
    ap = argparse.ArgumentParser(description="科研仓库活跃度快照（科研辅助模板示例）")
    ap.add_argument("--owner", required=True)
    ap.add_argument("--repo", required=True)
    ap.add_argument("--active-days", type=int, default=30)
    ap.add_argument("--now", help="时间基准 YYYY-MM-DD（默认当前 UTC；注入固定值可复现）")
    ap.add_argument("--output", default="report.md")
    args = ap.parse_args()

    now = datetime.strptime(args.now, "%Y-%m-%d") if args.now else datetime.utcnow()
    raw = collect(args.owner, args.repo)
    metrics = compute_metrics(raw, now, args.active_days)
    Path(args.output).write_text(render(args.owner, args.repo, metrics, now, args.active_days), encoding="utf-8")
    print(f"report -> {args.output}")
    return 0


if __name__ == "__main__":
    sys.exit(main())
