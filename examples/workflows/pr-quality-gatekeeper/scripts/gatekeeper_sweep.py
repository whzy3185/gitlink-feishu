# SPDX-License-Identifier: MulanPSL-2.0
"""gatekeeper_sweep —— 对一个仓库的全部 open PR 批量跑门禁（只读 dry-run），出治理报告。

把单 PR 的「策略 → 评分卡 → 裁决」升级为仓库级体检：
  1. 翻页拉取 PR 列表，筛出 open；
  2. 逐个调用 gatekeeper_workflow.py（强制 dry-run，绝不 --apply，对远端零写入）；
  3. 汇总每个 PR 的 summary.json → 聚合统计 + 全量明细表 → sweep-report.md / sweep-summary.json。

诚实口径：批扫不注入 AI 审查发现（--findings），review_findings 维按 0 发现计满分，
报告中明确标注「该维度未评」；其余 4 维（测试/卫生/commit/CI）为真实采集结果。
纯标准库，无第三方依赖。
"""

from __future__ import annotations

import argparse
import json
import re
import subprocess
import sys
import time
import urllib.request
from pathlib import Path
from typing import Any

API_BASE = "https://www.gitlink.org.cn/api"
HYGIENE_RE = re.compile(r"desc (✓|✗) / linked issue (✓|✗) / size (✓|✗)")


def fetch_open_prs(owner: str, repo: str, limit_pages: int = 20) -> list[dict[str, Any]]:
    """翻页拉取 PR 列表并筛出 open（列表接口的 status 参数不可靠，按字段过滤）。"""
    items: list[dict[str, Any]] = []
    page = 1
    while page <= limit_pages:
        url = f"{API_BASE}/{owner}/{repo}/pulls.json?page={page}&limit=50"
        with urllib.request.urlopen(url, timeout=30) as resp:
            data = json.loads(resp.read().decode("utf-8"))
        batch = data.get("issues") or []
        if not batch:
            break
        items.extend(batch)
        if len(items) >= int(data.get("search_count") or 0):
            break
        page += 1
    return [it for it in items if it.get("pull_request_staus") == "open"]


def run_one(
    workflow_script: Path,
    owner: str,
    repo: str,
    number: int,
    policy: Path,
    owner_rules: Path,
    cli_bin: str,
    out_dir: Path,
) -> dict[str, Any]:
    """对单个 PR 跑一次 dry-run 门禁，返回解析后的行记录（失败不抛，记 error）。"""
    cmd = [
        sys.executable,
        str(workflow_script),
        "--owner", owner,
        "--repo", repo,
        "--pr", str(number),
        "--policy", str(policy),
        "--owner-rules", str(owner_rules),
        "--cli-bin", cli_bin,
        "--skip-ci",
        "--output-dir", str(out_dir),
    ]
    proc = subprocess.run(cmd, capture_output=True, text=True, timeout=180)
    slug = f"{owner}_{repo}_pr{number}".replace("/", "_")
    summary_path = out_dir / f"{slug}_summary.json"
    if proc.returncode == 1 or not summary_path.exists():
        return {"number": number, "error": (proc.stderr or proc.stdout)[-200:].strip()}
    summary = json.loads(summary_path.read_text(encoding="utf-8"))
    hygiene = ""
    scorecard_path = out_dir / f"{slug}_scorecard.md"
    if scorecard_path.exists():
        m = HYGIENE_RE.search(scorecard_path.read_text(encoding="utf-8"))
        if m:
            hygiene = "/".join(m.groups())  # 例如 "✓/✗/✓"：描述/关联issue/体量
    return {
        "number": number,
        "verdict": summary.get("verdict"),
        "total": summary.get("total"),
        "scores": summary.get("scores", {}),
        "hard_gate_failures": [f.get("gate") if isinstance(f, dict) else f
                               for f in summary.get("hard_gate_failures", [])],
        "hygiene": hygiene,
        "suggested_reviewers": summary.get("routing", {}).get("suggested_reviewers", []),
    }


def aggregate(rows: list[dict[str, Any]]) -> dict[str, Any]:
    ok = [r for r in rows if "error" not in r]
    totals = sorted(r["total"] for r in ok)
    verdicts: dict[str, int] = {}
    gate_hits: dict[str, int] = {}
    for r in ok:
        verdicts[r["verdict"]] = verdicts.get(r["verdict"], 0) + 1
        for g in r["hard_gate_failures"]:
            gate_hits[str(g)] = gate_hits.get(str(g), 0) + 1
    def pct(n: int) -> str:
        return f"{100 * n / len(ok):.0f}%" if ok else "0%"
    no_linked = sum(1 for r in ok if r["hygiene"] and r["hygiene"].split("/")[1] == "✗")
    zero_cov = sum(1 for r in ok if r["scores"].get("test_coverage") == 0)
    return {
        "scanned": len(rows),
        "succeeded": len(ok),
        "failed": len(rows) - len(ok),
        "verdicts": verdicts,
        "score_min": totals[0] if totals else None,
        "score_median": totals[len(totals) // 2] if totals else None,
        "score_avg": round(sum(totals) / len(totals), 1) if totals else None,
        "score_max": totals[-1] if totals else None,
        "hard_gate_hits": gate_hits,
        "pct_zero_test_coverage": pct(zero_cov),
        "pct_no_linked_issue": pct(no_linked),
        "pct_request_changes": pct(verdicts.get("REQUEST_CHANGES", 0)),
    }


def render_report(
    owner: str, repo: str, policy_label: str, date_label: str,
    rows: list[dict[str, Any]], agg: dict[str, Any],
    pr_meta: dict[int, dict[str, Any]],
) -> str:
    ok = [r for r in rows if "error" not in r]
    lines = [
        f"# gatekeeper 仓库体检报告 —— {owner}/{repo}（{date_label}）",
        "",
        f"> 对 **{agg['scanned']} 个 open PR** 全量 dry-run（**只读，零写入**）· 策略 `{policy_label}` · "
        f"成功 {agg['succeeded']} / 失败 {agg['failed']}",
        ">",
        "> **诚实口径**：批扫未注入 AI 审查发现，review_findings 维按 0 发现计满分（**该维度未评**）；"
        "CI 维按 `--skip-ci` 统一记 unknown（半分）。其余维度为真实采集。"
        "因此**总分代表「除人工/AI 审查外的工程卫生分」，偏乐观**；裁决分布同理。",
        "",
        "## 总览",
        "",
        f"- 裁决分布：{' · '.join(f'{k} **{v}**' for k, v in sorted(agg['verdicts'].items()))}",
        f"- 分数：min {agg['score_min']} / 中位 {agg['score_median']} / 均值 {agg['score_avg']} / max {agg['score_max']}",
        f"- **{agg['pct_zero_test_coverage']}** 的 PR 测试覆盖维 0 分（改动不带任何测试）",
        f"- **{agg['pct_no_linked_issue']}** 的 PR 未关联 issue",
        f"- **{agg['pct_request_changes']}** 的 PR 触发 REQUEST_CHANGES（硬门禁或低分）",
        "",
        "硬门禁命中：" + ("；".join(f"`{g}` × {n}" for g, n in sorted(agg["hard_gate_hits"].items(), key=lambda x: -x[1])) or "无"),
        "",
        "## 全量明细（按分数降序）",
        "",
        "| PR | 标题 | 作者 | 总分 | 裁决 | 硬门禁失败 | 卫生(描述/关联/体量) |",
        "|----|------|------|-----:|------|-----------|---------------------|",
    ]
    for r in sorted(ok, key=lambda x: -x["total"]):
        meta = pr_meta.get(r["number"], {})
        title = str(meta.get("name", ""))[:48].replace("|", "\\|")
        gates = ", ".join(str(g) for g in r["hard_gate_failures"]) or "—"
        lines.append(
            f"| [#{r['number']}](https://www.gitlink.org.cn/{owner}/{repo}/pulls/{r['number']}) "
            f"| {title} | {meta.get('author_name', '?')} | {r['total']} | {r['verdict']} | {gates} | {r['hygiene'] or '—'} |"
        )
    errs = [r for r in rows if "error" in r]
    if errs:
        lines += ["", "## 跑失败的 PR", ""]
        lines += [f"- #{r['number']}：`{r['error']}`" for r in errs]
    lines += [
        "",
        "## 这份报告说明了什么",
        "",
        "- 同一份 `gatekeeper.yaml` 策略可以**无人值守地体检一个真实活跃仓库的全部积压**——"
        "确定性评分意味着大规模治理零 AI 成本，AI 只在需要语义判断（review_findings）时按需介入。",
        "- 任何人重跑本报告（`python3 scripts/gatekeeper_sweep.py`）会对同一组 PR 得到同样的分数与裁决。",
        "",
    ]
    return "\n".join(lines)


def main(argv: list[str] | None = None) -> int:
    parser = argparse.ArgumentParser(description="对全部 open PR 批量 dry-run 出治理报告")
    parser.add_argument("--owner", default="Gitlink")
    parser.add_argument("--repo", default="gitlink-cli")
    parser.add_argument("--policy", type=Path, required=True)
    parser.add_argument("--owner-rules", dest="owner_rules", type=Path, required=True)
    parser.add_argument("--cli-bin", default="gitlink-cli")
    parser.add_argument("--output-dir", type=Path, default=Path("sweep-outputs"))
    parser.add_argument("--date-label", default="sweep", help="报告日期标签（可复现：不取系统时间）")
    parser.add_argument("--max", type=int, default=0, help="只跑前 N 个（0=全量），用于试跑")
    parser.add_argument("--sleep", type=float, default=0.2, help="相邻 PR 间隔秒数（对平台礼貌）")
    args = parser.parse_args(argv)

    workflow_script = Path(__file__).with_name("gatekeeper_workflow.py")
    runs_dir = args.output_dir / "runs"
    runs_dir.mkdir(parents=True, exist_ok=True)

    prs = fetch_open_prs(args.owner, args.repo)
    if args.max:
        prs = prs[: args.max]
    pr_meta = {int(p["pull_request_number"]): p for p in prs}
    print(f"open PR 共 {len(prs)} 个，开始批扫（dry-run，零写入）…", flush=True)

    rows: list[dict[str, Any]] = []
    for i, p in enumerate(prs, 1):
        number = int(p["pull_request_number"])
        row = run_one(workflow_script, args.owner, args.repo, number,
                      args.policy, args.owner_rules, args.cli_bin, runs_dir)
        rows.append(row)
        tag = row.get("verdict", "ERROR")
        print(f"[{i}/{len(prs)}] PR #{number} → {tag} {row.get('total', '')}", flush=True)
        time.sleep(args.sleep)

    agg = aggregate(rows)
    policy_label = args.policy.name
    report = render_report(args.owner, args.repo, policy_label, args.date_label, rows, agg, pr_meta)
    (args.output_dir / "sweep-report.md").write_text(report, encoding="utf-8")
    (args.output_dir / "sweep-summary.json").write_text(
        json.dumps({"aggregate": agg, "rows": rows}, ensure_ascii=False, indent=2),
        encoding="utf-8",
    )
    print(f"\n报告：{args.output_dir / 'sweep-report.md'}")
    print(f"汇总：{args.output_dir / 'sweep-summary.json'}")
    print(f"裁决分布：{agg['verdicts']} · 均分 {agg['score_avg']}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
