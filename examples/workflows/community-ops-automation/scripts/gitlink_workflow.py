from __future__ import annotations

import argparse
import json
import os
import subprocess
from collections import Counter, defaultdict
from datetime import datetime, timedelta, timezone
from pathlib import Path
from typing import Any, Iterable


class WorkflowError(RuntimeError):
    pass


CLI_PAGE_SIZE = 100


def parse_args(argv: list[str] | None = None) -> argparse.Namespace:
    parser = argparse.ArgumentParser(
        description="GitLink 社区运营自动化工作流：周报 + Release Notes + 风险提示"
    )
    parser.add_argument(
        "--config",
        type=Path,
        default=Path("examples/sample_config.json"),
        help="配置文件路径",
    )
    parser.add_argument("--owner", help="覆盖配置中的仓库所有者")
    parser.add_argument("--repo", help="覆盖配置中的仓库名称")
    parser.add_argument(
        "--window-days",
        type=int,
        help="统计窗口，默认从配置文件读取或使用 7 天",
    )
    parser.add_argument(
        "--output-dir",
        type=Path,
        help="输出目录，默认从配置文件读取或使用 outputs",
    )
    parser.add_argument(
        "--publish-issue-id",
        type=int,
        help="发布摘要到指定 Issue 评论，未提供则只生成本地报告",
    )
    parser.add_argument(
        "--now",
        help="固定当前时间，便于测试，格式为 ISO8601",
    )
    parser.add_argument(
        "--skip-releases",
        action="store_true",
        help="跳过 release 列表采集",
    )
    parser.add_argument(
        "--cli-bin",
        help="gitlink-cli 可执行文件路径；可配合 GITLINK_CLI_BIN 使用",
    )
    return parser.parse_args(argv)


def load_json_file(path: Path) -> dict[str, Any]:
    if not path.exists():
        return {}
    return json.loads(path.read_text(encoding="utf-8"))


def sanitize_repo_name(value: str) -> str:
    return value.replace("/", "_").replace("\\", "_")


def parse_datetime(value: Any) -> datetime | None:
    if value in (None, "", []):
        return None
    if isinstance(value, datetime):
        dt = value
    else:
        text = str(value).strip()
        if not text:
            return None
        text = text.replace("Z", "+00:00")
        try:
            dt = datetime.fromisoformat(text)
        except ValueError:
            return None
    if dt.tzinfo is None:
        dt = dt.replace(tzinfo=timezone.utc)
    return dt.astimezone(timezone.utc)


def parse_iso_now(value: str | None) -> datetime:
    if not value:
        return datetime.now(timezone.utc)
    dt = parse_datetime(value)
    if dt is None:
        raise WorkflowError(f"无法解析 --now 的值: {value}")
    return dt


def first_value(item: dict[str, Any], keys: Iterable[str], default: Any = None) -> Any:
    for key in keys:
        if key in item:
            value = item[key]
            if value not in (None, "", []):
                return value
    return default


def normalize_labels(value: Any) -> list[str]:
    labels: list[str] = []
    if isinstance(value, list):
        for item in value:
            if isinstance(item, dict):
                name = first_value(item, ("name", "title", "label_name"))
                if name:
                    labels.append(str(name))
            elif item not in (None, ""):
                labels.append(str(item))
    elif isinstance(value, str) and value:
        labels.append(value)
    return labels


def extract_first_list(payload: Any, keys: Iterable[str]) -> list[Any]:
    if isinstance(payload, list):
        return payload
    if isinstance(payload, dict):
        for key in keys:
            value = payload.get(key)
            if isinstance(value, list):
                return value
        for value in payload.values():
            found = extract_first_list(value, keys)
            if found:
                return found
    return []


def extract_first_dict(payload: Any, keys: Iterable[str]) -> dict[str, Any]:
    if isinstance(payload, dict):
        for key in keys:
            value = payload.get(key)
            if isinstance(value, dict):
                return value
        for value in payload.values():
            found = extract_first_dict(value, keys)
            if found:
                return found
    if isinstance(payload, list):
        for item in payload:
            found = extract_first_dict(item, keys)
            if found:
                return found
    return {}


def run_gitlink_cli(command: list[str], owner: str, repo: str, cwd: Path | None = None) -> Any:
    if shutil_which("gitlink-cli") is None:
        raise WorkflowError("未找到 gitlink-cli，请先安装并确保它在 PATH 中")

    cli_path = shutil_which("gitlink-cli") or "gitlink-cli"
    if cli_path.lower().endswith((".cmd", ".bat")):
        cmd = [
            "cmd",
            "/c",
            cli_path,
            *command,
            "--owner",
            owner,
            "--repo",
            repo,
            "--format",
            "json",
        ]
    else:
        cmd = [
            cli_path,
            *command,
            "--owner",
            owner,
            "--repo",
            repo,
            "--format",
            "json",
        ]
    proc = subprocess.run(
        cmd,
        cwd=str(cwd) if cwd else None,
        capture_output=True,
        text=True,
        encoding="utf-8",
    )
    if proc.returncode != 0:
        stderr = proc.stderr.strip() or proc.stdout.strip() or "未知错误"
        raise WorkflowError(f"{' '.join(cmd)} 失败: {stderr}")
    return parse_json_output(proc.stdout)


def parse_json_output(text: str) -> Any:
    stripped = text.strip()
    if not stripped:
        raise WorkflowError("CLI 返回空结果")
    try:
        return json.loads(stripped)
    except json.JSONDecodeError:
        first_json = min(
            [idx for idx in (stripped.find("{"), stripped.find("[")) if idx != -1],
            default=-1,
        )
        if first_json > 0:
            return json.loads(stripped[first_json:])
        raise WorkflowError(f"无法解析 CLI JSON 输出: {stripped[:120]}")


def normalize_repo_info(payload: Any) -> dict[str, Any]:
    repo = extract_first_dict(payload, ("project", "repo", "repository", "data"))
    if not repo and isinstance(payload, dict):
        repo = payload
    return {
        "name": first_value(repo, ("name", "repo_name", "project_name", "identifier"), ""),
        "description": first_value(repo, ("description", "desc", "summary"), ""),
        "default_branch": first_value(repo, ("default_branch", "defaultBranch"), ""),
        "language": first_value(repo, ("language",), ""),
        "raw": repo,
    }


def normalize_issue_state(item: dict[str, Any], query_state: str | None = None) -> str:
    raw_status = first_value(item, ("status_id", "status", "state_id"), None)
    raw_name = str(
        first_value(item, ("issue_status", "status_name", "state", "status_name_cn"), "")
    ).strip().lower()
    if raw_status is not None:
        try:
            raw_status = int(raw_status)
        except (TypeError, ValueError):
            raw_status = str(raw_status).strip().lower()
    if raw_status in {5, "5", "closed", "close"} or "关" in raw_name or "closed" in raw_name:
        return "closed"
    if raw_status in {1, "1", 2, "2", 3, "3", "open", "opened"} or "开" in raw_name or "新" in raw_name:
        return "open"
    if query_state:
        return query_state
    return "open"


def normalize_issue(item: dict[str, Any], query_state: str | None = None) -> dict[str, Any]:
    return {
        "id": str(first_value(item, ("project_issues_index", "iid", "issue_id", "id", "number"), "")),
        "title": str(first_value(item, ("subject", "title", "name"), "(untitled)")),
        "state": normalize_issue_state(item, query_state=query_state),
        "created_at": parse_datetime(
            first_value(item, ("created_at", "createdAt", "created_time", "created", "format_time"))
        ),
        "updated_at": parse_datetime(
            first_value(item, ("updated_at", "updatedAt", "updated_time", "updated", "format_time"))
        ),
        "labels": normalize_labels(first_value(item, ("labels", "label_list", "label"), [])),
        "raw": item,
    }


def normalize_issues(payload: Any, query_state: str | None = None) -> list[dict[str, Any]]:
    items = extract_first_list(payload, ("issues", "issue_list", "items", "list"))
    normalized: list[dict[str, Any]] = []
    for item in items:
        if not isinstance(item, dict):
            continue
        normalized.append(normalize_issue(item, query_state=query_state))
    return normalized


def normalize_pr_state(item: dict[str, Any], query_state: str | None = None) -> str:
    raw_status = first_value(item, ("pull_request_status", "pull_request_staus", "status_id", "state_id"), None)
    if raw_status is not None:
        try:
            raw_status = int(raw_status)
        except (TypeError, ValueError):
            raw_status = str(raw_status).strip().lower()
    if raw_status in {1, "1", "merged"}:
        return "merged"
    if raw_status in {2, "2", "closed", "close"}:
        return "closed"
    if raw_status in {0, "0", "open", "opened"}:
        return "open"
    if query_state:
        return query_state
    return "open"


def normalize_pr(item: dict[str, Any], query_state: str | None = None) -> dict[str, Any]:
    state = normalize_pr_state(item, query_state=query_state)
    merged_at = parse_datetime(first_value(item, ("merged_at", "mergedAt", "merged_time")))
    merged_flag = state == "merged" or merged_at is not None
    return {
        "id": str(
            first_value(item, ("pull_request_number", "iid", "pr_id", "merge_request_iid", "id", "number"), "")
        ),
        "title": str(first_value(item, ("title", "subject", "name"), "(untitled)")),
        "state": state,
        "created_at": parse_datetime(
            first_value(item, ("created_at", "createdAt", "created_time", "created", "pr_full_time"))
        ),
        "updated_at": parse_datetime(
            first_value(item, ("updated_at", "updatedAt", "updated_time", "updated", "pr_full_time"))
        ),
        "merged_at": merged_at
        or (parse_datetime(first_value(item, ("pr_full_time",))) if state == "merged" else None),
        "merged": merged_flag,
        "labels": normalize_labels(first_value(item, ("labels", "label_list", "label"), [])),
        "raw": item,
    }


def normalize_prs(payload: Any, query_state: str | None = None) -> list[dict[str, Any]]:
    items = extract_first_list(payload, ("pull_requests", "merge_requests", "prs", "items", "list"))
    normalized: list[dict[str, Any]] = []
    for item in items:
        if not isinstance(item, dict):
            continue
        normalized.append(normalize_pr(item, query_state=query_state))
    return normalized


def normalize_releases(payload: Any) -> list[dict[str, Any]]:
    items = extract_first_list(payload, ("releases", "items", "list"))
    normalized: list[dict[str, Any]] = []
    for item in items:
        if not isinstance(item, dict):
            continue
        normalized.append(
            {
                "id": str(first_value(item, ("version_id", "id", "release_id", "iid"), "")),
                "title": str(first_value(item, ("name", "title", "tag_name"), "(untitled)")),
                "created_at": parse_datetime(
                    first_value(item, ("created_at", "createdAt", "released_at", "releasedAt"))
                ),
                "raw": item,
            }
        )
    return normalized


def is_open(state: str) -> bool:
    return state == "open"


def is_closed(state: str) -> bool:
    return state in {"closed", "close", "done", "resolved"}


def classify_title(title: str) -> str:
    lowered = title.strip().lower()
    prefix = lowered.split(":", 1)[0]
    prefix = prefix.split("(", 1)[0].strip()
    mapping = {
        "feat": "feature",
        "feature": "feature",
        "fix": "fix",
        "bugfix": "fix",
        "docs": "docs",
        "doc": "docs",
        "refactor": "refactor",
        "test": "test",
        "chore": "chore",
        "ci": "ci",
    }
    return mapping.get(prefix, "other")


def within_window(dt: datetime | None, cutoff: datetime) -> bool:
    return dt is not None and dt >= cutoff


def dedupe_records(records: list[dict[str, Any]]) -> list[dict[str, Any]]:
    seen: set[str] = set()
    result: list[dict[str, Any]] = []
    for item in records:
        key = str(item.get("id", "")).strip()
        if not key or key in seen:
            continue
        seen.add(key)
        result.append(item)
    return result


def fetch_paginated_payload(
    command: list[str],
    owner: str,
    repo: str,
    item_keys: tuple[str, ...],
    page_size: int = CLI_PAGE_SIZE,
) -> list[dict[str, Any]]:
    items: list[dict[str, Any]] = []
    page = 1
    max_pages = 50
    while True:
        if page > max_pages:
            break
        payload = run_gitlink_cli(
            [*command, "--page", str(page), "--limit", str(page_size)],
            owner,
            repo,
        )
        page_items = extract_first_list(payload, item_keys)
        page_items = [item for item in page_items if isinstance(item, dict)]
        if not page_items:
            break
        items.extend(page_items)
        if len(page_items) < page_size:
            break
        page += 1
    return items


def fetch_issues(owner: str, repo: str) -> list[dict[str, Any]]:
    records: list[dict[str, Any]] = []
    for state in ("open", "closed"):
        payloads = fetch_paginated_payload(
            ["issue", "+list", "--state", state],
            owner,
            repo,
            ("issues", "issue_list", "items", "list"),
        )
        records.extend(normalize_issues({"issues": payloads}, query_state=state))
    return dedupe_records(records)


def fetch_prs(owner: str, repo: str) -> list[dict[str, Any]]:
    records: list[dict[str, Any]] = []
    for state in ("open", "merged", "closed"):
        payloads = fetch_paginated_payload(
            ["pr", "+list", "--state", state],
            owner,
            repo,
            ("pull_requests", "merge_requests", "prs", "items", "list"),
        )
        records.extend(normalize_prs({"pull_requests": payloads}, query_state=state))
    return dedupe_records(records)


def fetch_releases(owner: str, repo: str) -> list[dict[str, Any]]:
    payloads = fetch_paginated_payload(
        ["release", "+list"],
        owner,
        repo,
        ("releases", "items", "list"),
    )
    return dedupe_records(normalize_releases({"releases": payloads}))


def summarize_workflow(
    repo_info: dict[str, Any],
    issues: list[dict[str, Any]],
    prs: list[dict[str, Any]],
    releases: list[dict[str, Any]],
    now: datetime,
    window_days: int,
) -> dict[str, Any]:
    cutoff = now - timedelta(days=window_days)

    open_issues = [item for item in issues if is_open(item["state"])]
    closed_issues = [item for item in issues if is_closed(item["state"])]
    stale_issues = [
        item
        for item in open_issues
        if item["updated_at"] is None or item["updated_at"] < cutoff
    ]

    merged_prs = [item for item in prs if item["merged"] or item["state"] == "merged"]
    open_prs = [item for item in prs if is_open(item["state"]) or (not item["merged"] and not is_closed(item["state"]))]
    stale_prs = [
        item
        for item in open_prs
        if item["updated_at"] is None or item["updated_at"] < cutoff
    ]
    recent_merged_prs = [
        item
        for item in merged_prs
        if within_window(item["merged_at"] or item["updated_at"] or item["created_at"], cutoff)
    ]

    issue_label_counter: Counter[str] = Counter()
    for item in issues:
        issue_label_counter.update(item["labels"])

    pr_buckets: dict[str, list[dict[str, Any]]] = defaultdict(list)
    for item in recent_merged_prs:
        pr_buckets[classify_title(item["title"])].append(item)

    actions: list[str] = []
    if stale_issues:
        actions.append(
            f"存在 {len(stale_issues)} 个超过 {window_days} 天未更新的开放 Issue，建议优先清理。"
        )
    if stale_prs:
        actions.append(
            f"存在 {len(stale_prs)} 个超过 {window_days} 天未更新的开放 PR，建议安排 review 或重新拆解。"
        )
    if not releases:
        actions.append("当前未采集到 Release 记录，建议补充发布说明或确认 Release 权限。")

    return {
        "repo": repo_info,
        "window_days": window_days,
        "now": now,
        "cutoff": cutoff,
        "counts": {
            "issues_total": len(issues),
            "issues_open": len(open_issues),
            "issues_closed": len(closed_issues),
            "issues_stale": len(stale_issues),
            "prs_total": len(prs),
            "prs_open": len(open_prs),
            "prs_merged": len(merged_prs),
            "prs_stale": len(stale_prs),
            "releases_total": len(releases),
        },
        "labels": issue_label_counter.most_common(8),
        "stale_issues": stale_issues,
        "stale_prs": stale_prs,
        "recent_merged_prs": recent_merged_prs,
        "pr_buckets": {key: value for key, value in pr_buckets.items()},
        "actions": actions,
    }


def render_list_block(items: list[dict[str, Any]], title_key: str = "title") -> str:
    if not items:
        return "- 无"
    lines = []
    for item in items[:10]:
        parts = [f"- {item.get('id', '')} {item.get(title_key, '')}".strip()]
        state = item.get("state")
        if state:
            parts.append(f"({state})")
        dt = item.get("updated_at") or item.get("merged_at") or item.get("created_at")
        if isinstance(dt, datetime):
            parts.append(dt.strftime("%Y-%m-%d"))
        lines.append(" ".join(parts))
    return "\n".join(lines)


def render_markdown_report(summary: dict[str, Any]) -> str:
    repo = summary["repo"]
    counts = summary["counts"]
    lines: list[str] = []
    title = repo["name"] or "GitLink 仓库"
    lines.append(f"# {title} 自动化周报")
    if repo.get("description"):
        lines.append("")
        lines.append(repo["description"])
    lines.append("")
    lines.append(f"- 统计窗口：近 {summary['window_days']} 天")
    lines.append(f"- 生成时间：{summary['now'].strftime('%Y-%m-%d %H:%M:%S UTC')}")
    lines.append("")
    lines.append("## 核心指标")
    lines.append("")
    lines.append("| 指标 | 数值 |")
    lines.append("| --- | ---: |")
    lines.append(f"| Issues 总数 | {counts['issues_total']} |")
    lines.append(f"| 打开 Issues | {counts['issues_open']} |")
    lines.append(f"| 超窗 Issue | {counts['issues_stale']} |")
    lines.append(f"| PR 总数 | {counts['prs_total']} |")
    lines.append(f"| 打开 PR | {counts['prs_open']} |")
    lines.append(f"| 已合并 PR | {counts['prs_merged']} |")
    lines.append(f"| Release 数 | {counts['releases_total']} |")
    lines.append("")

    lines.append("## 热点标签")
    if summary["labels"]:
        for label, count in summary["labels"]:
            lines.append(f"- {label}: {count}")
    else:
        lines.append("- 无")
    lines.append("")

    lines.append("## 最近合并 PR")
    recent_groups = summary["pr_buckets"]
    if recent_groups:
        for bucket, items in recent_groups.items():
            lines.append(f"### {bucket}")
            for item in items[:8]:
                merged_at = item.get("merged_at") or item.get("updated_at") or item.get("created_at")
                suffix = f" ({merged_at.strftime('%Y-%m-%d')})" if isinstance(merged_at, datetime) else ""
                lines.append(f"- {item['title']}{suffix}")
    else:
        lines.append("- 无")
    lines.append("")

    lines.append("## 风险提示")
    if summary["stale_issues"]:
        lines.append("### 超窗 Issue")
        lines.append(render_list_block(summary["stale_issues"]))
        lines.append("")
    if summary["stale_prs"]:
        lines.append("### 超窗 PR")
        lines.append(render_list_block(summary["stale_prs"]))
        lines.append("")
    if summary["actions"]:
        lines.append("### 建议动作")
        for action in summary["actions"]:
            lines.append(f"- {action}")
    else:
        lines.append("- 当前未发现明显风险。")
    return "\n".join(lines).rstrip() + "\n"


def render_release_notes(summary: dict[str, Any]) -> str:
    repo = summary["repo"]
    lines: list[str] = []
    title = repo["name"] or "GitLink 仓库"
    lines.append(f"# {title} Release Notes 草稿")
    lines.append("")
    lines.append(f"- 统计窗口：近 {summary['window_days']} 天")
    lines.append(f"- 生成时间：{summary['now'].strftime('%Y-%m-%d %H:%M:%S UTC')}")
    lines.append("")
    lines.append("## 变更概览")
    lines.append(f"- 已合并 PR：{summary['counts']['prs_merged']} 个")
    lines.append(f"- 最近窗口内合并 PR：{len(summary['recent_merged_prs'])} 个")
    lines.append("")
    lines.append("## 变更分类")
    groups = summary["pr_buckets"]
    if groups:
        for bucket in ("feature", "fix", "docs", "refactor", "test", "chore", "ci", "other"):
            items = groups.get(bucket, [])
            if not items:
                continue
            lines.append(f"### {bucket}")
            for item in items[:10]:
                merged_at = item.get("merged_at") or item.get("updated_at") or item.get("created_at")
                suffix = f" ({merged_at.strftime('%Y-%m-%d')})" if isinstance(merged_at, datetime) else ""
                lines.append(f"- {item['title']}{suffix}")
            lines.append("")
    else:
        lines.append("- 无")
        lines.append("")
    lines.append("## 发布说明")
    if summary["actions"]:
        for action in summary["actions"]:
            lines.append(f"- {action}")
    else:
        lines.append("- 当前未发现明显风险。")
    return "\n".join(lines).rstrip() + "\n"


def render_publish_comment(
    summary: dict[str, Any],
    report_path: Path,
    release_notes_path: Path | None = None,
) -> str:
    repo = summary["repo"]
    counts = summary["counts"]
    lines = [
        f"## {repo['name'] or 'GitLink 仓库'} 自动化周报摘要",
        "",
        f"- 时间窗：近 {summary['window_days']} 天",
        f"- Issues：{counts['issues_open']} 个打开，{counts['issues_stale']} 个超窗",
        f"- PR：{counts['prs_open']} 个打开，{counts['prs_merged']} 个已合并",
        f"- Release：{counts['releases_total']} 条",
        "",
        f"完整报告已生成：`{report_path.as_posix()}`",
    ]
    if release_notes_path is not None:
        lines.append(f"Release Notes 草稿：`{release_notes_path.as_posix()}`")
    if summary["actions"]:
        lines.append("")
        lines.append("### 建议动作")
        for action in summary["actions"][:3]:
            lines.append(f"- {action}")
    return "\n".join(lines).rstrip()


def build_issue_comment_command(issue_number: int, comment: str) -> list[str]:
    return ["issue", "+comment", "--number", str(issue_number), "--body", comment]


def safe_fetch(
    label: str,
    func,
    warnings: list[str],
    default: Any,
) -> Any:
    try:
        return func()
    except Exception as exc:  # noqa: BLE001
        warnings.append(f"{label} 失败：{exc}")
        return default


def shutil_which(name: str) -> str | None:
    from shutil import which

    return which(name)


def build_artifacts(
    owner: str,
    repo: str,
    window_days: int,
    output_dir: Path,
    now: datetime,
    publish_issue_id: int | None,
    skip_releases: bool,
) -> tuple[dict[str, Any], Path, Path, Path, list[str]]:
    warnings: list[str] = []
    repo_info = safe_fetch(
        "repo +info",
        lambda: normalize_repo_info(run_gitlink_cli(["repo", "+info"], owner, repo)),
        warnings,
        {"name": repo, "description": "", "default_branch": "", "language": "", "raw": {}},
    )
    issues = safe_fetch("issue +list", lambda: fetch_issues(owner, repo), warnings, [])
    prs = safe_fetch("pr +list", lambda: fetch_prs(owner, repo), warnings, [])
    releases = [] if skip_releases else safe_fetch(
        "release +list",
        lambda: fetch_releases(owner, repo),
        warnings,
        [],
    )

    summary = summarize_workflow(repo_info, issues, prs, releases, now, window_days)
    summary["warnings"] = warnings
    summary["owner"] = owner
    summary["repo_name"] = repo
    summary["publish_issue_id"] = publish_issue_id

    output_dir.mkdir(parents=True, exist_ok=True)
    stamp = now.strftime("%Y%m%d_%H%M%S")
    repo_slug = sanitize_repo_name(repo)
    base_name = f"{owner}_{repo_slug}_{stamp}"
    report_path = output_dir / f"{base_name}_report.md"
    summary_path = output_dir / f"{base_name}_summary.json"
    release_notes_path = output_dir / f"{base_name}_release_notes.md"

    report_text = render_markdown_report(summary)
    release_notes_text = render_release_notes(summary)
    report_path.write_text(report_text, encoding="utf-8")
    release_notes_path.write_text(release_notes_text, encoding="utf-8")
    summary_path.write_text(
        json.dumps(
            {
                **summary,
                "now": summary["now"].isoformat(),
                "cutoff": summary["cutoff"].isoformat(),
                "artifacts": {
                    "report": report_path.as_posix(),
                    "summary": summary_path.as_posix(),
                    "release_notes": release_notes_path.as_posix(),
                },
            },
            ensure_ascii=False,
            indent=2,
            default=str,
        ),
        encoding="utf-8",
    )

    if publish_issue_id is not None:
        comment = render_publish_comment(summary, report_path, release_notes_path)
        try:
            run_gitlink_cli(
                build_issue_comment_command(publish_issue_id, comment),
                owner,
                repo,
            )
        except Exception as exc:  # noqa: BLE001
            warnings.append(f"issue +comment 失败：{exc}")

    return summary, report_path, summary_path, release_notes_path, warnings


def main(argv: list[str] | None = None) -> int:
    args = parse_args(argv)
    config = load_json_file(args.config)

    owner = args.owner or config.get("owner")
    repo = args.repo or config.get("repo")
    if not owner or not repo:
        raise WorkflowError("请在配置文件或命令行中提供 owner 和 repo")

    window_days = args.window_days or int(config.get("window_days", 7))
    output_dir = args.output_dir or Path(config.get("output_dir", "outputs"))
    now = parse_iso_now(args.now)

    summary, report_path, summary_path, release_notes_path, warnings = build_artifacts(
        owner=owner,
        repo=repo,
        window_days=window_days,
        output_dir=output_dir,
        now=now,
        publish_issue_id=args.publish_issue_id,
        skip_releases=args.skip_releases,
    )

    print(f"已生成报告: {report_path}")
    print(f"已生成摘要: {summary_path}")
    print(f"已生成 Release Notes: {release_notes_path}")
    if warnings:
        print("警告:")
        for warning in warnings:
            print(f"- {warning}")
    print(
        "指标概览: "
        f"Issues={summary['counts']['issues_total']}, "
        f"PR={summary['counts']['prs_total']}, "
        f"Release={summary['counts']['releases_total']}"
    )
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
