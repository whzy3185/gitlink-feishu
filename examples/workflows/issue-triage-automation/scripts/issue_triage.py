#!/usr/bin/env python3
"""Issue 自动分拣端到端工作流（deterministic issue triage）。

采集 → 分类 → 报告 →（可选 --apply）回写：
1. 通过 gitlink-cli 拉取仓库 open issue 列表（含分页合并）
2. 按规则文件（JSON，纯标准库解析）做确定性分类：
   - 关键词命中 → 建议优先级 / 建议标签 / 建议负责人
   - 超龄未更新 → 标记 stale
3. 产出 triage 报告（markdown + json，同输入必得同输出）
4. 仅在 --apply 时通过 gitlink-cli 真实回写（优先级更新 + 分拣评论）

零第三方依赖，Python >= 3.9。
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


# ---------- 纯函数核心（单测覆盖，无网络） ----------

def match_rule(issue: dict, rule: dict) -> bool:
    """规则命中判定：keywords_any 任一关键词出现在标题或描述（大小写不敏感）。"""
    hay = ((issue.get("subject") or "") + "\n" + (issue.get("description") or "")).lower()
    return any(kw.lower() in hay for kw in rule.get("keywords_any", []))


def is_stale(issue: dict, now: datetime, stale_days: int) -> bool:
    raw = issue.get("updated_at") or issue.get("created_at") or ""
    try:
        updated = datetime.strptime(raw, DATE_FMT)
    except ValueError:
        return False
    return now - updated > timedelta(days=stale_days)


def triage_issue(issue: dict, rules: list[dict], now: datetime, stale_days: int) -> dict:
    """对单个 issue 产出确定性分拣决定（首个命中规则生效）。"""
    decision = {
        "number": issue.get("project_issues_index") or issue.get("number"),
        "subject": issue.get("subject", ""),
        "rule": None,
        "priority": None,
        "priority_id": None,
        "tags": [],
        "assignee": None,
        "stale": is_stale(issue, now, stale_days),
    }
    for rule in rules:
        if match_rule(issue, rule):
            decision["rule"] = rule.get("name")
            decision["priority"] = rule.get("set_priority")
            decision["priority_id"] = rule.get("set_priority_id")
            decision["tags"] = list(rule.get("add_tags", []))
            decision["assignee"] = rule.get("route_to")
            break
    return decision


def render_report(decisions: list[dict], owner: str, repo: str) -> str:
    lines = [
        f"# Issue 分拣报告：{owner}/{repo}",
        "",
        f"共 {len(decisions)} 个 open issue，"
        f"命中规则 {sum(1 for d in decisions if d['rule'])} 个，"
        f"stale {sum(1 for d in decisions if d['stale'])} 个。",
        "",
        "| # | 标题 | 命中规则 | 建议优先级 | 建议标签 | 建议负责人 | stale |",
        "|---|------|----------|------------|----------|------------|-------|",
    ]
    for d in sorted(decisions, key=lambda x: (x["number"] is None, x["number"])):
        lines.append(
            "| {number} | {subject} | {rule} | {priority} | {tags} | {assignee} | {stale} |".format(
                number=d["number"],
                subject=(d["subject"][:40] or "-"),
                rule=d["rule"] or "-",
                priority=d["priority"] or "-",
                tags=",".join(d["tags"]) or "-",
                assignee=d["assignee"] or "-",
                stale="是" if d["stale"] else "-",
            )
        )
    return "\n".join(lines) + "\n"


# ---------- CLI 采集与回写 ----------

def run_cli(args: list[str]) -> dict:
    out = subprocess.run([CLI, *args, "--format", "json"], capture_output=True, text=True)
    if out.returncode != 0:
        raise RuntimeError(f"{CLI} {' '.join(args)} failed: {out.stderr.strip() or out.stdout.strip()}")
    payload = json.loads(out.stdout)
    if not payload.get("ok", False):
        raise RuntimeError(f"{CLI} {' '.join(args)} returned error: {payload}")
    return payload.get("data") or {}


def fetch_open_issues(owner: str, repo: str) -> list[dict]:
    issues: list[dict] = []
    page = 1
    while True:
        data = run_cli([
            "issue", "+list", "--owner", owner, "--repo", repo,
            "--state", "open", "--page", str(page), "--limit", "50",
        ])
        batch = data.get("issues") or []
        issues.extend(batch)
        if len(batch) < 50:
            return issues
        page += 1


def apply_decision(owner: str, repo: str, decision: dict) -> None:
    number = str(decision["number"])
    if decision["priority_id"]:
        run_cli([
            "issue", "+update", "--owner", owner, "--repo", repo,
            "--number", number, "--priority-id", str(decision["priority_id"]),
        ])
    note = f"[issue-triage] 规则「{decision['rule']}」命中：建议标签 {','.join(decision['tags']) or '无'}，建议负责人 {decision['assignee'] or '无'}。"
    run_cli([
        "issue", "+comment", "--owner", owner, "--repo", repo,
        "--number", number, "--body", note,
    ])


def main() -> int:
    ap = argparse.ArgumentParser(description="GitLink issue 自动分拣工作流")
    ap.add_argument("--owner", required=True)
    ap.add_argument("--repo", required=True)
    ap.add_argument("--rules", default=str(Path(__file__).resolve().parent.parent / "rules.example.json"))
    ap.add_argument("--stale-days", type=int, default=30)
    ap.add_argument("--output-dir", default="triage-out")
    ap.add_argument("--apply", action="store_true", help="真实回写优先级与分拣评论（默认 dry-run）")
    args = ap.parse_args()

    rules = json.loads(Path(args.rules).read_text(encoding="utf-8"))["rules"]
    issues = fetch_open_issues(args.owner, args.repo)
    now = datetime.utcnow()
    decisions = [triage_issue(i, rules, now, args.stale_days) for i in issues]

    outdir = Path(args.output_dir)
    outdir.mkdir(parents=True, exist_ok=True)
    (outdir / "triage.json").write_text(
        json.dumps(decisions, ensure_ascii=False, indent=2), encoding="utf-8")
    (outdir / "triage.md").write_text(render_report(decisions, args.owner, args.repo), encoding="utf-8")
    print(f"triaged {len(decisions)} issues -> {outdir}/triage.md")

    if args.apply:
        applied = 0
        for d in decisions:
            if d["rule"]:
                apply_decision(args.owner, args.repo, d)
                applied += 1
        print(f"applied {applied} decisions")
    else:
        print("dry-run（未写远端）；加 --apply 真实回写")
    return 0


if __name__ == "__main__":
    sys.exit(main())
