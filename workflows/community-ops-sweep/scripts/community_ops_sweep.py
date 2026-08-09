#!/usr/bin/env python3
"""community_ops_sweep — 社区运营增量对账 sweep 的确定性引擎。

三层架构里的「Python 引擎层」：只做事实性计算（采集 / 归一化 / id 解析 / 合并 /
守卫 / 渲染 / 写回）；判断（分类 / PR 关联 / owner 建议）由 CC workflow 的 LLM
agent 产出 JSON 喂入。设计见
docs/superpowers/specs/2026-07-01-community-ops-sweep-design.md。

子命令：
    collect    采集 since 之后的新 issue + 合并 PR + 标签/分配人映射 → candidates.json
    plan       读 candidates + LLM decision JSON → plan.json（写计划）
    apply      执行 plan.json（默认 dry-run，--apply 才真写）
    checkpoint 推进 .last-sweep + 追加 docs/issue-triage-log.md

字段名实测自 baoerjun/gitlink-cli：issue 在 data.issues[]，PR 在 data.pulls[]。
"""
from __future__ import annotations

import argparse
import importlib.util
import json
import subprocess
import sys
from datetime import datetime, timedelta, timezone
from pathlib import Path
from typing import Any, Iterable

# --------------------------------------------------------------------------- #
# 复用 community-ops-automation 的 CLI 封装与通用 helpers（best-effort：缺失
# 也不影响纯逻辑导入与单测）
# --------------------------------------------------------------------------- #
_GW_PATH = (
    Path(__file__).resolve().parents[2]
    / "community-ops-automation"
    / "scripts"
    / "gitlink_workflow.py"
)
try:
    _spec = importlib.util.spec_from_file_location("_gw", _GW_PATH)
    _gw = importlib.util.module_from_spec(_spec)  # type: ignore[union-attr]
    sys.modules["_gw"] = _gw
    _spec.loader.exec_module(_gw)  # type: ignore[union-attr]
    run_gitlink_cli = _gw.run_gitlink_cli
    first_value = _gw.first_value
    extract_first_list = _gw.extract_first_list
    parse_datetime = _gw.parse_datetime
    WorkflowError = _gw.WorkflowError
except Exception:  # pragma: no cover - 纯逻辑单测不应依赖 _gw
    _gw = None
    run_gitlink_cli = None  # type: ignore[assignment]
    first_value = lambda item, keys, default=None: next(  # noqa: E731
        (item[k] for k in keys if isinstance(item, dict) and k in item and item[k] not in (None, "", [])), default
    )
    extract_first_list = lambda payload, keys: payload if isinstance(payload, list) else []  # noqa: E731
    parse_datetime = lambda v: None  # type: ignore[assignment]  # noqa: E731

    class WorkflowError(RuntimeError):
        pass


DEFAULT_POLICY = {
    "status_map": {
        "duplicate": "closed",
        "answered": "closed",
        "spam": "closed",
        "invalid": "closed",
    }
}


# --------------------------------------------------------------------------- #
# 纯逻辑（确定性、可单测）—— TDD 覆盖
# --------------------------------------------------------------------------- #
def _id_sort_key(x: Any) -> tuple:
    try:
        return (0, int(x))
    except (TypeError, ValueError):
        return (1, str(x))


def merge_tag_ids(current_ids, add_names, remove_names, tag_map):
    """整体替换语义：返回 (current ∪ add) \\ remove；白名单过滤、去重、按数值排序。"""
    result = {str(x) for x in (current_ids or []) if x is not None and str(x) != ""}
    for name in add_names or []:
        tid = (tag_map or {}).get(name)
        if tid is not None:
            result.add(str(tid))
    for name in remove_names or []:
        tid = (tag_map or {}).get(name)
        if tid is not None:
            result.discard(str(tid))
    return sorted(result, key=_id_sort_key)


def map_status(recommended_action, policy):
    """recommended_action → 目标状态（如 duplicate→closed）；未映射或 None → None（不改）。"""
    if not recommended_action:
        return None
    sm = (policy or {}).get("status_map", {}) or {}
    return sm.get(recommended_action)


def _issue_current_tag_ids(issue: dict) -> list[str]:
    out = []
    for t in issue.get("tags") or []:
        if isinstance(t, dict) and t.get("id") is not None:
            out.append(str(t["id"]))
    return out


def coerce_dates(item, keys=("updated_at", "created_at")):
    """把 item 中字符串日期字段解析回 datetime（磁盘 round-trip 后的修复）。

    datetime/None 原样保留；只解析 str。缺失的键补齐为 None。非 dict 原样返回。"""
    if not isinstance(item, dict):
        return item
    out = dict(item)
    for k in keys:
        v = out.get(k)
        if isinstance(v, str):
            out[k] = parse_datetime(v)
        elif k not in out:
            out[k] = None
    return out


def _render_owner_comment(evidence, suggested) -> str:
    names = "、".join(f"@{s}" for s in suggested)
    ev = f"（依据：{evidence}）" if evidence else ""
    return (
        f"🤖 建议负责人：{names}{ev}。\n"
        "如非本人负责，请自行重新指派或在下方留言说明。"
    )


def build_plan(candidates, triage, owners, links, routing, policy, tag_map):
    """把 LLM 判断 + 确定性规则汇编成写计划。纯函数：同输入 → 同输出。"""
    policy = policy or {}
    routing = routing or {}
    tag_map = tag_map or {}
    writes: list[dict] = []

    issues = candidates.get("issues", []) or []
    by_num = {str(i.get("number")): i for i in issues if i.get("number") is not None}
    open_issues = {str(x) for x in (candidates.get("open_issues") or [])}

    # R1 triage：标签增删 + 状态映射
    for t in triage or []:
        num = str(t.get("issue"))
        issue = by_num.get(num)
        if issue is None:
            continue  # 校验：候选里没有 → 跳过，不盲写
        current = _issue_current_tag_ids(issue)
        new_ids = merge_tag_ids(
            current, t.get("tag_names", []), t.get("remove_tag_names", []), tag_map
        )
        if sorted(new_ids, key=_id_sort_key) != sorted(current, key=_id_sort_key):
            writes.append({
                "op": "update_tags", "issue": num, "tag_ids": new_ids,
                "tag_names": t.get("tag_names", []) or [],
                "remove_tag_names": t.get("remove_tag_names", []) or [],
            })
        state = map_status(t.get("recommended_action"), policy)
        if state:
            writes.append({
                "op": "update_status", "issue": num, "state": state,
                "reason": f"triage: {t.get('recommended_action')}",
            })

    # R2 owner 建议（默认评论；routing.auto_assign=true 才真指派）
    auto_assign = bool(routing.get("auto_assign"))
    assigner_map = candidates.get("assigner_map") or {}
    for o in owners or []:
        num = str(o.get("issue"))
        if num not in by_num:
            continue
        suggested = o.get("suggested_owners") or []
        if not suggested:
            continue
        writes.append({
            "op": "comment", "issue": num, "kind": "owner_suggest",
            "body": _render_owner_comment(o.get("evidence"), suggested),
        })
        if auto_assign:
            aids = [str(assigner_map[s]) for s in suggested if s in assigner_map]
            if aids:
                writes.append({"op": "update_assigner", "issue": num, "assigner_ids": aids})

    # R5 PR→Issue 关联闭环：只关「存在且 open」的（幂等 + 防幻觉编号）
    for l in links or []:
        pr = l.get("pr")
        for inum in l.get("linked_issue_numbers", []) or []:
            inum = str(inum)
            if inum not in open_issues:
                continue  # 校验：不存在或非 open → 跳过
            writes.append({
                "op": "update_status", "issue": inum, "state": "closed",
                "reason": f"fixed by PR#{pr}",
            })
            writes.append({
                "op": "comment", "issue": inum, "kind": "link_close",
                "body": f"🤖 PR#{pr} 已合并且语义关联本 issue，自动关闭（可 reopen）。",
            })

    return {"writes": writes}


# --------------------------------------------------------------------------- #
# 归一化（实测字段名）
# --------------------------------------------------------------------------- #
def _issue_state(item: dict) -> str:
    sid = first_value(item, ("status_id", "state_id"), None)
    if sid in (5, "5", "closed", "close"):
        return "closed"
    return "open"


def normalize_issue(item: dict) -> dict:
    author = item.get("author") or {}
    tags = item.get("tags") or []
    return {
        "id": str(first_value(item, ("number", "project_issues_index", "iid"), "")),
        "number": str(first_value(item, ("number", "project_issues_index", "iid"), "")),
        "title": str(first_value(item, ("subject", "title", "name"), "(untitled)")),
        "description": str(first_value(item, ("description", "body", "content"), "")),
        "state": _issue_state(item),
        "labels": [str(t.get("name")) for t in tags if isinstance(t, dict) and t.get("name")],
        "tags": tags,
        "author_login": author.get("login") or author.get("name") or "",
        "author_id": author.get("id"),
        "created_at": parse_datetime(first_value(item, ("created_at", "createdAt"), None)),
        "updated_at": parse_datetime(first_value(item, ("updated_at", "updatedAt"), None)),
    }


def normalize_pr(pull: dict) -> dict:
    inner = pull.get("issue") or {}
    author = inner.get("author") or pull.get("author") or {}
    state = str(first_value(pull, ("status", "state"), "open")).lower()
    merged = state == "merged"
    return {
        "id": str(first_value(pull, ("index", "number", "pull_request_number"), "")),
        "number": str(first_value(pull, ("index", "number", "pull_request_number"), "")),
        "title": str(first_value(pull, ("title", "name"), "(untitled)")),
        "description": str(first_value(pull, ("body", "description", "content"), "")),
        "state": state,
        "merged": merged,
        "base": pull.get("base"),
        "head": pull.get("head"),
        "author_login": author.get("login") or author.get("name") or "",
        "author_id": author.get("id"),
        "merged_at": parse_datetime(first_value(pull, ("merged_at", "merge_time"), None)),
        "created_at": parse_datetime(first_value(pull, ("created_at", "pr_created_unix"), None)),
        "updated_at": parse_datetime(first_value(pull, ("updated_at", "created_at", "pr_created_unix"), None)),
        "labels": [str(t.get("name")) for t in (inner.get("issue_tags") or []) if isinstance(t, dict) and t.get("name")],
    }


def _load_tag_map(owner, repo) -> dict[str, str]:
    if run_gitlink_cli is None:
        return {}
    payload = run_gitlink_cli(["issue", "+tags", "--only-name"], owner, repo)
    items = extract_first_list(payload, ("issue_tags", "tags", "items", "list"))
    return {str(t.get("name")): str(t.get("id")) for t in items if isinstance(t, dict) and t.get("name") and t.get("id") is not None}


def _load_assigner_map(owner, repo) -> dict[str, str]:
    if run_gitlink_cli is None:
        return {}
    payload = run_gitlink_cli(["issue", "+assigners"], owner, repo)
    items = extract_first_list(payload, ("assigners", "issue_assigners", "users", "items", "list"))
    out = {}
    for u in items:
        if not isinstance(u, dict):
            continue
        login = u.get("login") or u.get("name") or u.get("username")
        if login and u.get("id") is not None:
            out[str(login)] = str(u["id"])
    return out


# --------------------------------------------------------------------------- #
# I/O 子命令
# --------------------------------------------------------------------------- #
def _read_json(path: Path) -> Any:
    return json.loads(path.read_text(encoding="utf-8"))


def _write_json(path: Path, data: Any) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text(json.dumps(data, ensure_ascii=False, indent=2, default=str), encoding="utf-8")


def _parse_since(value: str | None, checkpoint: Path | None) -> datetime:
    if value:
        dt = parse_datetime(value)
        if dt is None:
            raise WorkflowError(f"无法解析 --since: {value}")
        return dt
    if checkpoint and checkpoint.exists():
        dt = parse_datetime(checkpoint.read_text(encoding="utf-8").strip())
        if dt is not None:
            return dt
    return datetime.now(timezone.utc) - timedelta(days=7)


def _collect_all_prs(owner: str, repo: str):
    """采集所有 PR（open + merged + closed），返回 (open_prs, merged_prs, all_pr_authors)。"""
    all_prs: list[dict] = []
    for state in ("open", "merged", "closed"):
        raw = run_gitlink_cli(["pr", "+list", "--state", state, "--limit", "200"], owner, repo)
        items = [
            normalize_pr(p)
            for p in extract_first_list(raw, ("pulls", "pull_requests", "issues", "items", "list"))
            if isinstance(p, dict)
        ]
        all_prs.extend(items)
    # 去重 (同一条 PR 可能出现在多个 state 查询中)
    seen: set[str] = set()
    deduped: list[dict] = []
    for p in all_prs:
        if p["number"] not in seen:
            seen.add(p["number"])
            deduped.append(p)
    open_prs = [p for p in deduped if p["state"] == "open"]
    merged_prs = [p for p in deduped if p["merged"] or p["state"] == "merged"]
    authors = sorted({p["author_login"] for p in deduped if p["author_login"]})
    return open_prs, merged_prs, authors


def _contributor_ranking(issues: list[dict], merged_prs: list[dict]) -> list[dict]:
    """统计贡献者活跃度：合并 PR 数 + 提交 Issue 数。"""
    scores: dict[str, dict] = {}
    for i in issues:
        login = i.get("author_login", "")
        if not login:
            continue
        if login not in scores:
            scores[login] = {"login": login, "issues": 0, "prs": 0}
        scores[login]["issues"] += 1
    for p in merged_prs:
        login = p.get("author_login", "")
        if not login:
            continue
        if login not in scores:
            scores[login] = {"login": login, "issues": 0, "prs": 0}
        scores[login]["prs"] += 1
    ranked = sorted(scores.values(), key=lambda x: x["prs"] + x["issues"], reverse=True)
    return ranked[:10]


def _load_previous_summary(checkpoint_dir: Path) -> dict | None:
    """读取上一周期的汇总数据用于趋势对比。"""
    path = checkpoint_dir / ".last-sweep-summary.json"
    if not path.exists():
        return None
    try:
        return json.loads(path.read_text(encoding="utf-8"))
    except Exception:
        return None


def _save_current_summary(checkpoint_dir: Path, summary: dict) -> None:
    """保存本周期的汇总数据供下一次趋势对比。"""
    checkpoint_dir.mkdir(parents=True, exist_ok=True)
    (checkpoint_dir / ".last-sweep-summary.json").write_text(
        json.dumps(summary, ensure_ascii=False, indent=2, default=str),
        encoding="utf-8",
    )


def cmd_collect(args) -> int:
    owner, repo = args.owner, args.repo
    checkpoint = Path(args.checkpoint)
    since = _parse_since(args.since, checkpoint)

    raw_issues = run_gitlink_cli(["issue", "+list", "--limit", "200"], owner, repo)
    all_issues = [
        normalize_issue(i)
        for i in extract_first_list(raw_issues, ("issues", "issue_list", "items", "list"))
        if isinstance(i, dict)
    ]
    new_issues = [i for i in all_issues if i["created_at"] and i["created_at"] >= since]
    open_issue_numbers = sorted({i["number"] for i in all_issues if i["state"] == "open"})
    closed_issues = [i for i in all_issues if i["state"] == "closed"]

    open_prs, merged_prs, pr_authors = _collect_all_prs(owner, repo)

    # 趋势对比：加载上次汇总
    previous = _load_previous_summary(checkpoint.parent)
    current_counts = {
        "issues_total": len(all_issues), "issues_open": len(open_issue_numbers),
        "issues_closed": len(closed_issues), "prs_open": len(open_prs),
        "prs_merged": len(merged_prs), "contributors": len(
            {i.get("author_login", "") for i in all_issues if i.get("author_login")} | set(pr_authors)
        ),
    }
    trends = _compute_trends(current_counts, previous)

    # 贡献者排行
    contributors = _contributor_ranking(all_issues, merged_prs)

    # 保存当前汇总供下一次对比
    _save_current_summary(checkpoint.parent, current_counts)

    candidates = {
        "owner": owner, "repo": repo, "since": since.isoformat(),
        "issues": new_issues, "open_issues": open_issue_numbers,
        "closed_issues": [c["number"] for c in closed_issues],
        "merged_prs": merged_prs, "open_prs": open_prs,
        "trends": trends, "contributors": contributors,
        "current_counts": current_counts, "previous_counts": previous,
        "tag_map": _load_tag_map(owner, repo),
        "assigner_map": _load_assigner_map(owner, repo),
    }
    _write_json(Path(args.out), candidates)
    print(json.dumps({
        "since": since.isoformat(), "new_issues": len(new_issues),
        "merged_prs": len(merged_prs), "open_issues": len(open_issue_numbers),
        "open_prs": len(open_prs), "contributors": len(contributors),
    }, ensure_ascii=False))
    return 0


def _compute_trends(current: dict, previous: dict | None) -> dict[str, str]:
    """对比上周期数据，返回趋势箭头。"""
    if not previous:
        return {}
    trends = {}
    for key in ("issues_total", "issues_open", "issues_closed", "prs_open", "prs_merged", "contributors"):
        prev = previous.get(key, 0)
        curr = current.get(key, 0)
        if isinstance(prev, (int, float)) and isinstance(curr, (int, float)):
            if curr > prev:
                trends[key] = f"↑{curr - prev}"
            elif curr < prev:
                trends[key] = f"↓{prev - curr}"
            else:
                trends[key] = "→"
    return trends


def cmd_plan(args) -> int:
    candidates = _read_json(Path(args.candidates))
    tag_map = candidates.get("tag_map") or {}
    policy = _read_json(Path(args.policy)) if args.policy else DEFAULT_POLICY
    routing = _read_json(Path(args.routing)) if args.routing else {}
    triage = _read_json(Path(args.triage)) if args.triage and Path(args.triage).exists() else []
    owners = _read_json(Path(args.owners)) if args.owners and Path(args.owners).exists() else []
    links = _read_json(Path(args.links)) if args.links and Path(args.links).exists() else []

    # 读取 LLM 生成的执行摘要
    summary_path = getattr(args, "summary", None)
    if summary_path and Path(summary_path).exists():
        summary_data = _read_json(Path(summary_path))
        candidates["executive_summary"] = summary_data.get("summary", "") if isinstance(summary_data, dict) else ""

    plan = build_plan(candidates, triage, owners, links, routing, policy, tag_map)
    # R3: 周报 Markdown 生成（七段式）；R4: Release Notes
    plan["report_md"] = _generate_report(candidates, plan)
    plan["release_notes_md"] = _render_release_notes(candidates)
    _write_json(Path(args.out), plan)
    print(json.dumps({"writes": len(plan["writes"])}, ensure_ascii=False))
    return 0


def _build_command(w: dict) -> list[str] | None:
    op = w.get("op")
    num = str(w.get("issue"))
    if op == "update_tags":
        return ["issue", "+update", "--number", num, "--tag-ids", ",".join(w.get("tag_ids") or [])]
    if op == "update_status":
        return ["issue", "+update", "--number", num, "--state", str(w.get("state"))]
    if op == "comment":
        return ["issue", "+comment", "--number", num, "--body", str(w.get("body"))]
    if op == "update_assigner":
        return ["issue", "+update", "--number", num, "--assigner-ids", ",".join(w.get("assigner_ids") or [])]
    return None


def _exec_or_dry(args, cmd, label) -> bool:
    """dry-run 打印；--apply 才真执行。返回是否执行成功。"""
    if not args.apply:
        preview = " ".join(cmd[:4]) + (" …" if len(cmd) > 4 else "")
        print(f"[dry-run] {label}: {preview}")
        return False
    try:
        run_gitlink_cli(cmd, args.owner, args.repo)
        return True
    except Exception as exc:  # noqa: BLE001
        print(f"[fail] {label}: {exc}", file=sys.stderr)
        return False


def _publish_report(args, plan) -> None:
    """R3：把 report_md 作为评论发到指定 issue。"""
    body = (plan.get("report_md") or "").strip()
    issue = getattr(args, "report_issue", None)
    if not body or not issue:
        return
    _exec_or_dry(args, ["issue", "+comment", "--number", str(issue), "--body", body],
                 f"report→issue#{issue}")


def _publish_release_notes(args, plan) -> None:
    """R4：把 release_notes_md 合并进指定 release 的 body（release +update --body 整体替换，先读后拼）。"""
    body = (plan.get("release_notes_md") or "").strip()
    rid = getattr(args, "release_id", None)
    if not body or not rid:
        return
    existing = ""
    if args.apply:
        try:
            rv = run_gitlink_cli(["release", "+view", "--id", str(rid)], args.owner, args.repo)
            data = rv.get("data", rv) if isinstance(rv, dict) else {}
            existing = str(first_value(data, ("body", "description", "content", "version_content"), "") or "").rstrip()
        except Exception as exc:  # noqa: BLE001
            print(f"[warn] read release body failed: {exc}", file=sys.stderr)
    merged = (existing + "\n\n" + body) if existing else body
    _exec_or_dry(args, ["release", "+update", "--id", str(rid), "--body", merged],
                 f"notes→release#{rid}")


def cmd_apply(args) -> int:
    plan = _read_json(Path(args.plan))
    writes = plan.get("writes", [])
    done = 0
    for w in writes:
        cmd = _build_command(w)
        if cmd is None:
            continue
        if _exec_or_dry(args, cmd, f"#{w.get('issue')} {w.get('op')}"):
            done += 1
    _publish_report(args, plan)
    _publish_release_notes(args, plan)
    _publish_report_to_wiki(args, plan)
    n = done if args.apply else len(writes)
    print(f"{'applied' if args.apply else 'dry-run'}: {n} write(s)")
    return 0


# --------------------------------------------------------------------------- #
# Wiki 发布（R3-wiki：周报上传到 Wiki 周报目录）
# --------------------------------------------------------------------------- #
_WIKI_CLI_BIN = "gitlink-cli"


def _find_wiki_cli() -> str:
    """找到 gitlink-cli 可执行文件；优先复用 _gw 的 which 逻辑。"""
    if _gw is not None and hasattr(_gw, "shutil_which"):
        path = _gw.shutil_which(_WIKI_CLI_BIN)
        if path:
            return path
    from shutil import which
    return which(_WIKI_CLI_BIN) or _WIKI_CLI_BIN


def _run_wiki(args, command: list[str], label: str) -> bool:
    """执行 gitlink-cli wiki 子命令；dry-run 打印，--apply 真写。返回是否实际执行。"""
    cli = _find_wiki_cli()
    cmd = [cli, "wiki", *command, "--owner", args.owner, "--repo", args.repo]
    if not args.apply:
        print(f"[dry-run] {label}: {' '.join(cmd)}")
        return False
    try:
        proc = subprocess.run(cmd, capture_output=True, text=True, timeout=30)
        if proc.returncode != 0:
            err = proc.stderr.strip() or proc.stdout.strip() or "未知错误"
            print(f"[fail] {label}: {err}", file=sys.stderr)
            return False
        print(f"[ok] {label}")
        return True
    except Exception as exc:  # noqa: BLE001
        print(f"[fail] {label}: {exc}", file=sys.stderr)
        return False


def _ensure_wiki_dir(args, dir_name: str) -> bool:
    """确保 Wiki 侧边栏中存在目录；不存在则创建。"""
    # 先检查目录是否已存在（通过读 Sidebar 判断）
    cli = _find_wiki_cli()
    check_cmd = [cli, "wiki", "+view", "--name", "_Sidebar",
                 "--owner", args.owner, "--repo", args.repo]
    try:
        proc = subprocess.run(check_cmd, capture_output=True, text=True, timeout=15)
        if proc.returncode == 0 and f"- {dir_name}" not in proc.stdout:
            return _run_wiki(args, ["+mkdir", "--name", dir_name],
                             f"wiki mkdir {dir_name}")
    except Exception:
        pass
    return True  # 无法判断时跳过，让 create --dir 自行报错


def _publish_report_to_wiki(args, plan) -> None:
    """R3-wiki：把周报上传到 Wiki 周报目录（支持幂等：同日期运行更新而非报错）。"""
    body = (plan.get("report_md") or "").strip()
    if not body:
        return
    if not getattr(args, "publish_wiki", False):
        return

    wiki_dir = getattr(args, "wiki_dir", None) or "周报专区"
    now = datetime.now(timezone.utc)
    page_name = f"周报-{now.strftime('%Y-%m-%d-%H%M')}"
    commit_msg = f"自动生成周报 {now.strftime('%Y-%m-%d %H:%M')}"

    # Step 1: 确保目录存在
    _ensure_wiki_dir(args, wiki_dir)

    # Step 2: 尝试创建 → 失败则更新（幂等）
    ok = _run_wiki(args, [
        "+create", "--name", page_name,
        "--content", body,
        "--message", commit_msg,
        "--dir", wiki_dir,
    ], f"wiki create {page_name} → {wiki_dir}")
    if not ok:
        _run_wiki(args, [
            "+update", "--name", page_name,
            "--content", body,
            "--message", commit_msg,
        ], f"wiki update {page_name} (fallback)")


def cmd_checkpoint(args) -> int:
    now = datetime.now(timezone.utc)
    Path(args.checkpoint).parent.mkdir(parents=True, exist_ok=True)
    Path(args.checkpoint).write_text(now.isoformat(), encoding="utf-8")
    log = Path(args.triage_log)
    entry = _read_json(Path(args.plan)) if args.plan and Path(args.plan).exists() else {"writes": []}
    batch = [
        f"\n## 批次 {now.isoformat()}（{args.owner}/{args.repo}）\n",
        f"- writes: {len(entry.get('writes', []))}\n",
    ]
    log.parent.mkdir(parents=True, exist_ok=True)
    with log.open("a", encoding="utf-8") as fh:
        fh.writelines(batch)
    print(f"checkpoint → {args.checkpoint} ({now.isoformat()})")
    return 0


def _generate_report(candidates: dict, plan: dict | None = None) -> str:
    """生成七段式社区运营周报（纯 Markdown 文本拼接，非 HTML 渲染）。

    Wiki 平台负责将 Markdown 转为页面展示，我们只负责数据→内容的编排。
    """
    repo = candidates.get("repo", "")
    owner = candidates.get("owner", "")
    since_str = candidates.get("since", "")[:10]
    now = datetime.now(timezone.utc)
    week_end = now.strftime("%Y-%m-%d")
    week_range = f"{since_str} ~ {week_end}"

    trends = candidates.get("trends", {}) or {}
    current = candidates.get("current_counts", {}) or {}
    contributors = candidates.get("contributors", []) or []
    writes = (plan or {}).get("writes", []) or []

    issues = candidates.get("issues", []) or []
    merged_prs = candidates.get("merged_prs", []) or []
    open_prs = candidates.get("open_prs", []) or []

    lines: list[str] = []

    # ── 标题 ──
    lines.append(f"# {repo} 社区运营周报")
    lines.append("")
    lines.append(f"> 📅 {week_range} | 🤖 由 community-ops-sweep 自动生成")
    lines.append("")

    # ── 一、执行摘要 ──
    lines.append("## 一、📊 执行摘要")
    lines.append("")
    summary_placeholder = candidates.get("executive_summary", "").strip()
    if summary_placeholder:
        lines.append(summary_placeholder)
    else:
        lines.append("> ⚠️ 执行摘要未生成。运行 Workflow 时由 LLM Agent 自动填充。")
    lines.append("")

    # ── 二、核心指标 ──
    lines.append("## 二、📈 核心指标")
    lines.append("")
    lines.append("| 指标 | 本周数值 | 趋势 |")
    lines.append("|------|:--------:|:----:|")
    _metric_row(lines, "Issue 总数", current, "issues_total", trends)
    _metric_row(lines, "打开 Issue", current, "issues_open", trends)
    _metric_row(lines, "已关闭 Issue", current, "issues_closed", trends)
    _metric_row(lines, "待审 PR", current, "prs_open", trends)
    _metric_row(lines, "已合并 PR", current, "prs_merged", trends)
    _metric_row(lines, "活跃贡献者", current, "contributors", trends)
    lines.append("")

    # ── 三、新增 Issue 清单 ──
    lines.append("## 三、🆕 新增 Issue 清单")
    lines.append("")
    if issues:
        lines.append("| 编号 | 标题 | 标签 | 提交者 |")
        lines.append("|------|------|------|--------|")
        for i in issues:
            num = i.get("number", "-")
            title = (i.get("title") or "(untitled)")[:60]
            labels = ", ".join(i.get("labels", [])[:4]) or "-"
            author = i.get("author_login", "-")
            lines.append(f"| #{num} | {title} | {labels} | @{author} |")
    else:
        lines.append("本周无新增 Issue。")
    lines.append("")

    # ── 四、PR 活动 ──
    lines.append("## 四、🔀 PR 活动")
    lines.append("")

    lines.append(f"### 已合并 PR（{len(merged_prs)} 个）")
    if merged_prs:
        lines.append("")
        lines.append("| 编号 | 标题 | 作者 | 合并时间 |")
        lines.append("|------|------|------|----------|")
        for p in merged_prs[:15]:
            num = p.get("number", "-")
            title = (p.get("title") or "(untitled)")[:60]
            author = p.get("author_login", "-")
            merged_at = _fmt_dt(p.get("merged_at") or p.get("updated_at"))
            lines.append(f"| !{num} | {title} | @{author} | {merged_at} |")
    else:
        lines.append("")
        lines.append("本周无合并 PR。")
    lines.append("")

    lines.append(f"### 待审 PR（{len(open_prs)} 个）")
    if open_prs:
        lines.append("")
        lines.append("| 编号 | 标题 | 作者 | 创建时间 |")
        lines.append("|------|------|------|----------|")
        for p in open_prs[:15]:
            num = p.get("number", "-")
            title = (p.get("title") or "(untitled)")[:60]
            author = p.get("author_login", "-")
            created = _fmt_dt(p.get("created_at"))
            lines.append(f"| !{num} | {title} | @{author} | {created} |")
    else:
        lines.append("")
        lines.append("无待审 PR 🎉")
    lines.append("")

    # ── 五、贡献者活跃榜 ──
    lines.append("## 五、👥 贡献者活跃榜")
    lines.append("")
    if contributors:
        lines.append("| 排名 | 贡献者 | Issue 提交 | PR 合并 | 活跃度 |")
        lines.append("|:----:|--------|:----------:|:-------:|:------:|")
        for idx, c in enumerate(contributors, 1):
            score = c["issues"] + c["prs"]
            bar = "█" * min(score, 10)
            lines.append(f"| {idx} | @{c['login']} | {c['issues']} | {c['prs']} | {bar} |")
    else:
        lines.append("本周无活跃贡献者。")
    lines.append("")

    # ── 六、自动 Triage 日志 ──
    lines.append("## 六、🤖 自动 Triage 日志")
    lines.append("")
    if writes:
        lines.append("| Issue | 操作 | 详情 |")
        lines.append("|-------|------|------|")
        for w in writes:
            issue_num = f"#{w.get('issue', '-')}"
            op = w.get("op", "-")
            detail = _triage_detail(w)
            lines.append(f"| {issue_num} | {op} | {detail} |")
    else:
        lines.append("本次 sweep 未执行写操作（无变更或 dry-run）。")
    lines.append("")

    # ── 七、待办与风险 ──
    lines.append("## 七、⚠️ 待办与风险")
    lines.append("")
    stale_issues = [i for i in issues if _is_stale(i, since_str)]
    stale_prs = [p for p in open_prs if _is_stale_pr(p, since_str)]
    if stale_issues:
        lines.append(f"- 🔴 **{len(stale_issues)} 个 Issue** 超过窗口期未更新，建议优先处理")
    if stale_prs:
        lines.append(f"- 🟡 **{len(stale_prs)} 个 PR** 等待 Review 超过窗口期，建议推进")
    if open_prs:
        lines.append(f"- 📋 **{len(open_prs)} 个 PR** 待 Review，建议分配 Reviewer")
    if not stale_issues and not stale_prs:
        lines.append("- ✅ 当前无超期未处理的 Issue 或 PR。")
    lines.append("")

    lines.append("---")
    lines.append("")
    lines.append(f"*报告生成时间：{now.strftime('%Y-%m-%d %H:%M:%S UTC')} | 数据来源：[{owner}/{repo}](https://gitlink.org.cn/{owner}/{repo})*")
    lines.append("")

    return "\n".join(lines)


def _metric_row(lines: list[str], label: str, counts: dict, key: str, trends: dict):
    val = counts.get(key, "-")
    trend = trends.get(key, "→")
    lines.append(f"| {label} | {val} | {trend} |")


def _fmt_dt(dt) -> str:
    if isinstance(dt, datetime):
        return dt.strftime("%m-%d")
    if isinstance(dt, str):
        return dt[:10]
    return "-"


def _is_stale(issue: dict, since_str: str) -> bool:
    updated = issue.get("updated_at")
    if isinstance(updated, datetime):
        return updated.strftime("%Y-%m-%d") < since_str
    if isinstance(updated, str):
        return updated[:10] < since_str
    return False


def _is_stale_pr(pr: dict, since_str: str) -> bool:
    updated = pr.get("updated_at")
    if isinstance(updated, datetime):
        return updated.strftime("%Y-%m-%d") < since_str
    if isinstance(updated, str):
        return updated[:10] < since_str
    return False


def _triage_detail(w: dict) -> str:
    op = w.get("op", "")
    if op == "update_tags":
        added = ", ".join(w.get("tag_names", []))
        removed = ", ".join(w.get("remove_tag_names", []))
        parts = []
        if added:
            parts.append(f"+{added}")
        if removed:
            parts.append(f"-{removed}")
        return " ".join(parts) if parts else "标签调整"
    if op == "update_status":
        return f"→ {w.get('state', '?')} ({w.get('reason', '')})"
    if op == "comment":
        kind = w.get("kind", "")
        return {"owner_suggest": "建议负责人", "link_close": "自动关闭通知"}.get(kind, "评论")
    if op == "update_assigner":
        return f"指派 {', '.join(w.get('assigner_ids', []))}"
    return "-"


def _render_release_notes(candidates: dict) -> str:
    if _gw is None:
        return ""
    try:
        repo_info = {"name": candidates.get("repo", ""), "description": ""}
        issues = [coerce_dates(i) for i in candidates.get("issues", [])]
        prs = [coerce_dates(p, ("updated_at", "created_at", "merged_at")) for p in candidates.get("merged_prs", [])]
        summary = _gw.summarize_workflow(
            repo_info, issues, prs, [],
            datetime.now(timezone.utc), 7,
        )
        return _gw.render_release_notes(summary)
    except Exception as exc:  # noqa: BLE001
        print(f"[warn] render_release_notes failed: {exc}", file=sys.stderr)
        return ""


def main(argv: list[str] | None = None) -> int:
    p = argparse.ArgumentParser(description="community-ops-sweep 确定性引擎")
    sub = p.add_subparsers(dest="cmd", required=True)

    pc = sub.add_parser("collect", help="采集 → candidates.json")
    pc.add_argument("--owner", required=True); pc.add_argument("--repo", required=True)
    pc.add_argument("--since"); pc.add_argument("--out", default="candidates.json")
    pc.add_argument("--checkpoint", default=".last-sweep")
    pc.set_defaults(func=cmd_collect)

    pp = sub.add_parser("plan", help="candidates + decisions → plan.json")
    pp.add_argument("--candidates", default="candidates.json")
    pp.add_argument("--triage"); pp.add_argument("--owners"); pp.add_argument("--links")
    pp.add_argument("--summary", help="LLM 生成的执行摘要 JSON 文件")
    pp.add_argument("--routing"); pp.add_argument("--policy")
    pp.add_argument("--out", default="plan.json")
    pp.set_defaults(func=cmd_plan)

    pa = sub.add_parser("apply", help="执行 plan.json（默认 dry-run）")
    pa.add_argument("--plan", default="plan.json")
    pa.add_argument("--owner", required=True); pa.add_argument("--repo", required=True)
    pa.add_argument("--apply", action="store_true")
    pa.add_argument("--report-issue", dest="report_issue", help="R3：把周报作为评论发到此 issue")
    pa.add_argument("--release-id", dest="release_id", help="R4：把 release notes 合并进此 release body")
    pa.add_argument("--publish-wiki", action="store_true", dest="publish_wiki",
                    help="R3-wiki：把周报上传到 Wiki 周报目录")
    pa.add_argument("--wiki-dir", dest="wiki_dir", default="周报专区",
                    help="Wiki 周报目录名（默认：周报专区）")
    pa.set_defaults(func=cmd_apply)

    pk = sub.add_parser("checkpoint", help="推进 .last-sweep + 追加 triage-log")
    pk.add_argument("--owner", required=True); pk.add_argument("--repo", required=True)
    pk.add_argument("--checkpoint", default=".last-sweep")
    pk.add_argument("--triage-log", default="docs/issue-triage-log.md")
    pk.add_argument("--plan")
    pk.set_defaults(func=cmd_checkpoint)

    args = p.parse_args(argv)
    return args.func(args)


if __name__ == "__main__":
    raise SystemExit(main())
