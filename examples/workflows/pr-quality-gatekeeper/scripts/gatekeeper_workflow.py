#!/usr/bin/env python3
"""gitlink-gatekeeper 端到端「PR 看门人闭环」编排脚本。

把 gitlink-gatekeeper（Policy-as-Code PR 合并门禁）包成一条可复现的闭环，串联
SSOT（docs/design.md）第 9 节定义的三步：

    1. 路由：按变更文件路径匹配 owner-rules.yaml 的 glob，推导建议 reviewer。
    2. 裁决：采集 PR 上下文 → 按 SSOT 第 3-5 节确定性算法算评分卡 + 三态裁决。
    3. 回写 + 善后：评分卡作为评论回写 PR + 打裁决标签；若 REQUEST_CHANGES，
       自动创建一条 tracking issue 汇总必修项并回链 PR。

设计要点（与 SSOT 第 8 节安全规则一致）：
    - **默认 dry-run**：不传 --apply 时只打印将要做什么，绝不写任何东西。
    - 写操作（回写评论 / 打标签 / 建 issue / 合并）一律需要显式 --apply。
    - **绝不默认自动合并**：仅当策略 behavior.auto_merge=true 且裁决=PASS 且
      命令显式带 --apply 才会合并。
    - 纯 Python 标准库，无第三方依赖（含内置的 YAML 子集解析器）。

CLI 命令映射见 SSOT 第 7 节；所引用的 gitlink-cli 命令均已在
upstream-gitlink-cli/shortcuts/ 下核验存在：
    pr +view / pr +files / pr +diff / pr +comment / pr +review / pr +merge
    issue +create / issue +comment
    label +create / label +list
    ci +builds
向 PR / issue 挂标签没有独立 shortcut，需走 Raw API
    POST /:owner/:repo/issues/:id   --body '{"issue_tag_ids":[...], ...}'
（PR 底层关联一个 issue，标签即挂在该 issue 上——与 gitlink-code-review 工作流 3 一致）。
挂载行为受 --apply 显式门控：dry-run 只预览「将把标签挂到 PR」，仅在 --apply 时
才真正发起 POST；回写 body 会带上原 subject/description，避免清空 PR 标题/描述。
"""

# SPDX-License-Identifier: MulanPSL-2.0

from __future__ import annotations

import argparse
import fnmatch
import json
import re
import subprocess
import sys
from dataclasses import dataclass, field
from pathlib import Path
from shutil import which
from typing import Any, Iterable


# --------------------------------------------------------------------------- #
# 常量 / 异常
# --------------------------------------------------------------------------- #

SEVERITY_ORDER = ("blocker", "major", "minor", "nit")

# SSOT 第 2 节「内置默认策略」——找不到 gatekeeper.yaml 时回退到这里。
DEFAULT_POLICY: dict[str, Any] = {
    "version": 1,
    "weights": {
        "review_findings": 40,
        "test_coverage": 20,
        "pr_hygiene": 15,
        "commit_quality": 15,
        "ci_status": 10,
    },
    "hard_gates": {
        "forbid_blocker_findings": True,
        "require_ci_pass": True,
        "require_tests_for_src_changes": True,
        "require_linked_issue": False,
        "max_changed_files": 80,
    },
    "severity_penalty": {"blocker": 100, "major": 25, "minor": 5, "nit": 1},
    "thresholds": {"pass": 85, "request_changes": 60},
    "labels": {
        "pass": "gatekeeper:pass",
        "request_changes": "gatekeeper:needs-changes",
        "comment": "gatekeeper:review",
    },
    "source_globs": ["**/*.go", "**/*.py", "**/*.js", "**/*.ts", "**/*.rs", "**/*.java"],
    "test_globs": ["**/*_test.go", "**/test_*.py", "**/*.test.*", "**/*.spec.*", "tests/**"],
    "behavior": {
        "dry_run_default": True,
        "post_comment": True,
        "apply_label": True,
        "auto_merge": False,
        "merge_method": "squash",
    },
}

VERDICT_EMOJI = {"PASS": "✅", "REQUEST_CHANGES": "❌", "COMMENT": "💬"}


class WorkflowError(RuntimeError):
    """工作流可预期的失败（缺配置、CLI 不存在、采集失败等）。"""


# --------------------------------------------------------------------------- #
# 极简 YAML 子集解析器（纯标准库，覆盖本工作流配置所需的语法）
#
# 支持：标量、嵌套映射（缩进）、`key: [a, b]` 行内列表、`- item` 块列表、
#       `#` 注释、true/false/整数/带引号字符串。
# 不支持：锚点、多文档、多行字符串——本项目配置不需要。
# --------------------------------------------------------------------------- #

def _parse_scalar(token: str) -> Any:
    token = token.strip()
    if token == "" or token == "~" or token.lower() == "null":
        return None
    if (token.startswith('"') and token.endswith('"')) or (
        token.startswith("'") and token.endswith("'")
    ):
        return token[1:-1]
    low = token.lower()
    if low == "true":
        return True
    if low == "false":
        return False
    try:
        return int(token)
    except ValueError:
        pass
    try:
        return float(token)
    except ValueError:
        return token


def _parse_inline_list(token: str) -> list[Any]:
    inner = token.strip()[1:-1].strip()
    if not inner:
        return []
    # 朴素逗号切分；配置里的列表元素不含逗号，足够。
    return [_parse_scalar(part) for part in inner.split(",")]


def _strip_comment(line: str) -> str:
    in_single = in_double = False
    for idx, ch in enumerate(line):
        if ch == "'" and not in_double:
            in_single = not in_single
        elif ch == '"' and not in_single:
            in_double = not in_double
        elif ch == "#" and not in_single and not in_double:
            return line[:idx]
    return line


def _parse_value_token(rest: str) -> Any:
    """解析 `key:` 右侧的标量 / 行内列表。"""
    if rest.startswith("["):
        return _parse_inline_list(rest)
    return _parse_scalar(rest)


def _clean_lines(text: str) -> list[tuple[int, str]]:
    """返回 [(indent, stripped_content)]，已去注释 / 空行。"""
    out: list[tuple[int, str]] = []
    for raw in text.splitlines():
        line = _strip_comment(raw).rstrip()
        if not line.strip():
            continue
        indent = len(line) - len(line.lstrip(" "))
        out.append((indent, line.strip()))
    return out


def _parse_block(lines: list[tuple[int, str]], pos: int, indent: int) -> tuple[Any, int]:
    """递归下降解析一个缩进块，返回 (value, next_pos)。

    根据块内第一行判断是 list（`- ...`）还是 dict（`key: ...`）。
    支持「列表项是映射」（`- glob: x` 后跟同级缩进的 `reviewers: [...]`）。
    """
    if pos >= len(lines):
        return {}, pos

    first_indent = lines[pos][0]
    is_list = lines[pos][1].startswith("- ")
    container: Any = [] if is_list else {}

    while pos < len(lines):
        cur_indent, content = lines[pos]
        if cur_indent < first_indent:
            break

        if is_list:
            if not content.startswith("- "):
                break
            item_body = content[2:].strip()
            # 列表项内可能直接带 key: value（即列表元素是映射）
            if ":" in item_body and not item_body.startswith(("[", '"', "'")):
                key, _, rest = item_body.partition(":")
                key, rest = key.strip(), rest.strip()
                # 该列表元素起始的虚拟缩进 = 列表项内容的列位置
                inner_indent = cur_indent + 2
                item_map: dict[str, Any] = {}
                if rest == "":
                    pos += 1
                    sub, pos = _parse_block(lines, pos, inner_indent + 1)
                    item_map[key] = sub
                else:
                    item_map[key] = _parse_value_token(rest)
                    pos += 1
                # 收编后续属于同一列表元素的 key（缩进 >= inner_indent 且不是新 `- `）
                while pos < len(lines):
                    nxt_indent, nxt_content = lines[pos]
                    if nxt_indent < inner_indent or nxt_content.startswith("- "):
                        break
                    if ":" not in nxt_content:
                        break
                    k2, _, r2 = nxt_content.partition(":")
                    k2, r2 = k2.strip(), r2.strip()
                    if r2 == "":
                        pos += 1
                        sub2, pos = _parse_block(lines, pos, nxt_indent + 1)
                        item_map[k2] = sub2
                    else:
                        item_map[k2] = _parse_value_token(r2)
                        pos += 1
                container.append(item_map)
            else:
                container.append(_parse_scalar(item_body))
                pos += 1
        else:
            if content.startswith("- ") or ":" not in content:
                break
            key, _, rest = content.partition(":")
            key, rest = key.strip(), rest.strip()
            if rest == "":
                pos += 1
                sub, pos = _parse_block(lines, pos, cur_indent + 1)
                container[key] = sub
            else:
                container[key] = _parse_value_token(rest)
                pos += 1

    return container, pos


def parse_simple_yaml(text: str) -> dict[str, Any]:
    """解析本工作流所需的 YAML 子集，返回嵌套 dict。

    支持：嵌套映射、块列表、「列表项是映射」、行内列表、标量、注释。
    不支持：锚点 / 多文档 / 多行字符串 / 复杂流式语法（本项目配置不需要）。
    """
    lines = _clean_lines(text)
    if not lines:
        return {}
    value, _ = _parse_block(lines, 0, lines[0][0])
    if not isinstance(value, dict):
        # 顶层是列表的情况：包一层（本项目顶层均为映射，保险处理）
        return {"_root": value}
    return value


# --------------------------------------------------------------------------- #
# 配置加载
# --------------------------------------------------------------------------- #

def load_yaml_file(path: Path) -> dict[str, Any]:
    if not path.exists():
        raise WorkflowError(f"找不到配置文件：{path}")
    return parse_simple_yaml(path.read_text(encoding="utf-8"))


def deep_merge(base: dict[str, Any], override: dict[str, Any]) -> dict[str, Any]:
    """把 override 合并到 base 的副本上（用于策略覆盖默认值）。"""
    result = json.loads(json.dumps(base))  # 深拷贝
    for key, value in (override or {}).items():
        if isinstance(value, dict) and isinstance(result.get(key), dict):
            result[key] = deep_merge(result[key], value)
        else:
            result[key] = value
    return result


def load_policy(policy_path: Path | None) -> dict[str, Any]:
    """加载 gatekeeper.yaml；缺失时回退内置默认策略（SSOT 第 2 节）。"""
    if policy_path is None or not policy_path.exists():
        return json.loads(json.dumps(DEFAULT_POLICY))
    user_policy = load_yaml_file(policy_path)
    merged = deep_merge(DEFAULT_POLICY, user_policy)
    _validate_policy(merged)
    return merged


def _validate_policy(policy: dict[str, Any]) -> None:
    weights = policy.get("weights", {})
    total = sum(int(v) for v in weights.values())
    if total != 100:
        raise WorkflowError(
            f"策略非法：weights 之和必须为 100，当前为 {total}（{weights}）"
        )


# --------------------------------------------------------------------------- #
# gitlink-cli 调用层
# --------------------------------------------------------------------------- #

def cli_available(cli_bin: str) -> bool:
    return which(cli_bin) is not None


def run_cli_json(
    cli_bin: str,
    args: list[str],
    owner: str,
    repo: str,
) -> Any:
    """运行只读 gitlink-cli 命令并解析 JSON 输出（仅采集步骤用）。"""
    cmd = [cli_bin, *args, "--owner", owner, "--repo", repo, "--format", "json"]
    proc = subprocess.run(cmd, capture_output=True, text=True, encoding="utf-8")
    if proc.returncode != 0:
        stderr = (proc.stderr or proc.stdout or "未知错误").strip()
        raise WorkflowError(f"命令失败：{' '.join(cmd)}\n{stderr}")
    return _parse_cli_json(proc.stdout)


def _parse_cli_json(text: str) -> Any:
    stripped = (text or "").strip()
    if not stripped:
        return {}
    try:
        return json.loads(stripped)
    except json.JSONDecodeError:
        idxs = [i for i in (stripped.find("{"), stripped.find("[")) if i != -1]
        if idxs:
            return json.loads(stripped[min(idxs):])
        raise WorkflowError(f"无法解析 CLI JSON 输出：{stripped[:120]}")


@dataclass
class PlannedWrite:
    """一个待执行的写操作（dry-run 时只打印，apply 时执行）。"""

    label: str               # 人类可读说明
    command: list[str]       # gitlink-cli 子命令（不含 --owner/--repo/--format）
    note: str = ""           # 备注（如 body 摘要）


def render_planned(write: PlannedWrite, owner: str, repo: str, cli_bin: str) -> str:
    safe_cmd = " ".join([cli_bin, *write.command, "--owner", owner, "--repo", repo])
    # 折叠超长 body，避免刷屏
    return f"  • {write.label}\n    $ {safe_cmd[:400]}"


def execute_write(
    write: PlannedWrite,
    owner: str,
    repo: str,
    cli_bin: str,
) -> dict[str, Any]:
    cmd = [cli_bin, *write.command, "--owner", owner, "--repo", repo, "--format", "json"]
    proc = subprocess.run(cmd, capture_output=True, text=True, encoding="utf-8")
    ok = proc.returncode == 0
    return {
        "label": write.label,
        "ok": ok,
        "stderr": (proc.stderr or "").strip() if not ok else "",
        "data": _parse_cli_json(proc.stdout) if ok and proc.stdout.strip() else None,
    }


# --------------------------------------------------------------------------- #
# 数据归一化（兼容 GitLink Envelope 的多种字段名）
# --------------------------------------------------------------------------- #

def unwrap(payload: Any) -> Any:
    """剥掉 Envelope 的 data 外层。"""
    if isinstance(payload, dict) and "data" in payload and "ok" in payload:
        return payload["data"]
    return payload


def first_value(item: dict[str, Any], keys: Iterable[str], default: Any = None) -> Any:
    for key in keys:
        if isinstance(item, dict) and key in item and item[key] not in (None, "", []):
            return item[key]
    return default


def extract_first_list(payload: Any, keys: Iterable[str]) -> list[Any]:
    if isinstance(payload, list):
        return payload
    if isinstance(payload, dict):
        for key in keys:
            if isinstance(payload.get(key), list):
                return payload[key]
        for value in payload.values():
            found = extract_first_list(value, keys)
            if found:
                return found
    return []


def extract_file_paths(files_payload: Any) -> list[str]:
    data = unwrap(files_payload)
    items = extract_first_list(data, ("files", "diff", "entries", "items", "list"))
    paths: list[str] = []
    for item in items:
        if isinstance(item, dict):
            p = first_value(item, ("path", "filename", "new_path", "name", "filepath"))
            if p:
                paths.append(str(p))
        elif isinstance(item, str):
            paths.append(item)
    return paths


# --------------------------------------------------------------------------- #
# 步骤 1：路由（owner-rules.yaml glob → reviewer）
# --------------------------------------------------------------------------- #

def load_owner_rules(path: Path) -> tuple[list[dict[str, Any]], list[str]]:
    """读取 owner-rules.yaml，返回 (rules, default_reviewers)，rules 为 [{glob, reviewers:[...]}, ...]（保序）。"""
    if not path.exists():
        raise WorkflowError(f"找不到 owner-rules：{path}")
    parsed = parse_simple_yaml(path.read_text(encoding="utf-8"))
    rules_raw = parsed.get("rules", [])
    rules: list[dict[str, Any]] = []
    if isinstance(rules_raw, list):
        for entry in rules_raw:
            if not isinstance(entry, dict):
                continue
            glob = entry.get("glob") or entry.get("path")
            reviewers = entry.get("reviewers")
            if isinstance(reviewers, str):
                reviewers = [reviewers]
            if glob and reviewers:
                rules.append({"glob": str(glob), "reviewers": list(reviewers)})
    fallback = parsed.get("default_reviewers") or parsed.get("fallback") or []
    if isinstance(fallback, str):
        fallback = [fallback]
    return rules, list(fallback)


def route_reviewers(
    changed_files: list[str],
    rules: list[dict[str, Any]],
    fallback: list[str],
) -> dict[str, Any]:
    """把变更文件映射到 reviewer。返回 reviewer→匹配文件，及未命中文件。"""
    assignments: dict[str, list[str]] = {}
    matched: set[str] = set()
    for path in changed_files:
        for rule in rules:
            if glob_match(path, rule["glob"]):
                matched.add(path)
                for reviewer in rule["reviewers"]:
                    assignments.setdefault(reviewer, []).append(path)
                break  # 首个命中规则生效（规则顺序即优先级）
    unmatched = [p for p in changed_files if p not in matched]
    if unmatched and fallback:
        for reviewer in fallback:
            assignments.setdefault(reviewer, []).extend(unmatched)
    return {
        "assignments": {k: sorted(set(v)) for k, v in assignments.items()},
        "unmatched": unmatched,
        "suggested_reviewers": sorted(assignments.keys()),
    }


# --------------------------------------------------------------------------- #
# 步骤 2：裁决（采集 + 评分，复用 SSOT 第 3-5 节确定性算法）
# --------------------------------------------------------------------------- #

def glob_match(path: str, pattern: str) -> bool:
    """fnmatch 包装：让 `**/X` 也能匹配顶层文件 `X`（fnmatch 默认要求至少一段目录）。

    这与 SSOT source_globs/test_globs（如 `**/*.go`）的直觉一致：`main.go`
    位于仓库根目录也应被视为源码。
    """
    if fnmatch.fnmatch(path, pattern):
        return True
    if pattern.startswith("**/"):
        return fnmatch.fnmatch(path, pattern[3:])
    return False


def matches_any(path: str, globs: list[str]) -> bool:
    return any(glob_match(path, g) for g in globs)


def count_src_test(changed_files: list[str], policy: dict[str, Any]) -> tuple[int, int]:
    # test_globs 优先：同时匹配 source 与 test 的文件只计为 test，不计为 src（SSOT §3.2）
    tests = sum(1 for p in changed_files if matches_any(p, policy["test_globs"]))
    src = sum(
        1
        for p in changed_files
        if matches_any(p, policy["source_globs"]) and not matches_any(p, policy["test_globs"])
    )
    return src, tests


CONVENTIONAL_RE = re.compile(
    r"^(feat|fix|docs|style|refactor|perf|test|build|ci|chore|revert)(\([^)]+\))?!?:\s+\S+"
)


def count_conventional_commits(commits: list[str]) -> tuple[int, int]:
    if not commits:
        return 0, 0
    conforming = sum(1 for msg in commits if CONVENTIONAL_RE.match((msg or "").strip()))
    return conforming, len(commits)


@dataclass
class Finding:
    severity: str
    message: str
    file: str = ""
    line: Any = ""


@dataclass
class ScoreInput:
    pr_id: str
    title: str
    description: str
    changed_files: list[str]
    changed_src: int
    changed_tests: int
    commits: list[str]
    ci_status: str            # passing / failing / unknown
    linked_issue: bool
    findings: list[Finding] = field(default_factory=list)


def score_dimensions(inp: ScoreInput, policy: dict[str, Any]) -> dict[str, Any]:
    """按 SSOT 第 3 节逐维评分。返回每维得分 + 备注。"""
    w = policy["weights"]
    sev = policy["severity_penalty"]
    max_changed = int(policy["hard_gates"].get("max_changed_files", 0))

    # 3.1 review_findings
    penalty = sum(int(sev.get(f.severity, 0)) for f in inp.findings)
    rf_w = w["review_findings"]
    review_score = round(rf_w * max(0.0, 1 - penalty / rf_w)) if rf_w else 0

    # 3.2 test_coverage
    tc_w = w["test_coverage"]
    if inp.changed_src == 0:
        test_score = tc_w
    elif inp.changed_tests == 0:
        test_score = 0
    else:
        ratio = min(1.0, inp.changed_tests / inp.changed_src)
        test_score = round(tc_w * (0.5 + 0.5 * ratio))

    # 3.3 pr_hygiene（三项各 1/3）
    hy_w = w["pr_hygiene"]
    hits = 0.0
    desc_ok = bool(inp.description) and len(inp.description.strip()) >= 30
    if desc_ok:
        hits += 1 / 3
    if inp.linked_issue:
        hits += 1 / 3
    n_files = len(inp.changed_files)
    if max_changed == 0 or n_files <= max_changed / 2:
        size_credit, size_mark = 1 / 3, "✓"  # 不限体量或体量适中 → 满分
    elif n_files <= max_changed:
        size_credit, size_mark = 1 / 6, "~"  # 偏大但未超上限 → 半分
    else:
        size_credit, size_mark = 0.0, "✗"  # 超上限（通常已被硬门禁拦截）
    hits += size_credit
    hygiene_score = round(hy_w * hits)

    # 3.4 commit_quality
    cq_w = w["commit_quality"]
    conforming, total = count_conventional_commits(inp.commits)
    commit_score = cq_w if total == 0 else round(cq_w * conforming / total)

    # 3.5 ci_status
    ci_w = w["ci_status"]
    if inp.ci_status == "passing":
        ci_score = ci_w
    elif inp.ci_status == "failing":
        ci_score = 0
    else:
        ci_score = round(ci_w * 0.5)

    sev_counts = {s: sum(1 for f in inp.findings if f.severity == s) for s in SEVERITY_ORDER}

    return {
        "review_findings": {
            "score": review_score,
            "weight": rf_w,
            "note": " / ".join(f"{sev_counts[s]} {s}" for s in SEVERITY_ORDER),
        },
        "test_coverage": {
            "score": test_score,
            "weight": tc_w,
            "note": f"{inp.changed_src} src / {inp.changed_tests} test files",
        },
        "pr_hygiene": {
            "score": hygiene_score,
            "weight": hy_w,
            "note": f"desc {'✓' if desc_ok else '✗'} / "
            f"linked issue {'✓' if inp.linked_issue else '✗'} / "
            f"size {size_mark}",
        },
        "commit_quality": {
            "score": commit_score,
            "weight": cq_w,
            "note": f"{conforming}/{total} conventional",
        },
        "ci_status": {"score": ci_score, "weight": ci_w, "note": inp.ci_status},
        "total": review_score + test_score + hygiene_score + commit_score + ci_score,
    }


def evaluate_hard_gates(inp: ScoreInput, policy: dict[str, Any]) -> list[dict[str, str]]:
    """按 SSOT 第 4 节逐项判定硬门禁，返回命中的失败列表。"""
    gates = policy["hard_gates"]
    failures: list[dict[str, str]] = []
    has_blocker = any(f.severity == "blocker" for f in inp.findings)
    if gates.get("forbid_blocker_findings") and has_blocker:
        failures.append({"gate": "forbid_blocker_findings", "detail": "存在 blocker 级审查发现"})
    # 仅在 CI 明确 failing 时触发；unknown/无 build 记录不触发（SSOT §4）
    if gates.get("require_ci_pass") and inp.ci_status == "failing":
        failures.append({"gate": "require_ci_pass", "detail": "CI 明确失败（failing）"})
    if (
        gates.get("require_tests_for_src_changes")
        and inp.changed_src > 0
        and inp.changed_tests == 0
    ):
        failures.append(
            {
                "gate": "require_tests_for_src_changes",
                "detail": f"改动了 {inp.changed_src} 个源码文件，但本 PR 未包含任何测试文件",
            }
        )
    if gates.get("require_linked_issue") and not inp.linked_issue:
        failures.append({"gate": "require_linked_issue", "detail": "PR 未关联任何 Issue"})
    max_changed = int(gates.get("max_changed_files", 0))
    if max_changed > 0 and len(inp.changed_files) > max_changed:
        failures.append(
            {
                "gate": "max_changed_files",
                "detail": f"变更文件 {len(inp.changed_files)} 超过上限 {max_changed}",
            }
        )
    return failures


def decide_verdict(
    total: int, hard_gate_failed: bool, policy: dict[str, Any]
) -> str:
    """SSOT 第 5 节裁决判定树。"""
    th = policy["thresholds"]
    if hard_gate_failed:
        return "REQUEST_CHANGES"
    if total >= int(th["pass"]):
        return "PASS"
    if total < int(th["request_changes"]):
        return "REQUEST_CHANGES"
    return "COMMENT"


# --------------------------------------------------------------------------- #
# 评分卡渲染（SSOT 第 6 节模板）
# --------------------------------------------------------------------------- #

def render_scorecard(
    inp: ScoreInput,
    dims: dict[str, Any],
    failures: list[dict[str, str]],
    verdict: str,
    policy_label: str,
    routing: dict[str, Any] | None,
    tracking_issue: Any = None,
) -> str:
    emoji = VERDICT_EMOJI[verdict]
    total = dims["total"]
    lines: list[str] = []
    lines.append(f"## 🛡️ Gatekeeper Report — PR #{inp.pr_id} {inp.title}")
    lines.append("")
    lines.append(
        f"**Verdict: {emoji} {verdict}**  ·  Score: {total}/100  ·  policy: {policy_label}"
    )
    lines.append("")
    lines.append("| Dimension | Weight | Score | Notes |")
    lines.append("|-----------|:------:|:-----:|-------|")
    lines.append(
        f"| Review findings | {dims['review_findings']['weight']} | "
        f"{dims['review_findings']['score']}/{dims['review_findings']['weight']} | "
        f"{dims['review_findings']['note']} |"
    )
    lines.append(
        f"| Test coverage   | {dims['test_coverage']['weight']} | "
        f"{dims['test_coverage']['score']}/{dims['test_coverage']['weight']} | "
        f"{dims['test_coverage']['note']} |"
    )
    lines.append(
        f"| PR hygiene      | {dims['pr_hygiene']['weight']} | "
        f"{dims['pr_hygiene']['score']}/{dims['pr_hygiene']['weight']} | "
        f"{dims['pr_hygiene']['note']} |"
    )
    lines.append(
        f"| Commit quality  | {dims['commit_quality']['weight']} | "
        f"{dims['commit_quality']['score']}/{dims['commit_quality']['weight']} | "
        f"{dims['commit_quality']['note']} |"
    )
    lines.append(
        f"| CI status       | {dims['ci_status']['weight']} | "
        f"{dims['ci_status']['score']}/{dims['ci_status']['weight']} | "
        f"{dims['ci_status']['note']} |"
    )
    lines.append("")

    if routing and routing.get("suggested_reviewers"):
        lines.append(f"### 👥 Suggested reviewers ({len(routing['suggested_reviewers'])})")
        for reviewer, paths in routing["assignments"].items():
            sample = ", ".join(paths[:3]) + (" …" if len(paths) > 3 else "")
            lines.append(f"- @{reviewer} — {len(paths)} file(s): {sample}")
        lines.append("")

    if failures:
        lines.append(f"### ⛔ Hard gate failures ({len(failures)})")
        for f in failures:
            lines.append(f"- `{f['gate']}`: {f['detail']}")
        lines.append("")

    must = [f for f in inp.findings if f.severity in ("blocker", "major")]
    should = [f for f in inp.findings if f.severity == "minor"]
    nits = [f for f in inp.findings if f.severity == "nit"]

    if must:
        lines.append(f"### 🔴 Must fix ({len(must)})")
        for f in must:
            loc = f" — {f.file}:{f.line}" if f.file else ""
            lines.append(f"- [{f.severity}] {f.message}{loc}")
        lines.append("")
    if should:
        lines.append(f"### 🟡 Should fix ({len(should)})")
        for f in should:
            loc = f" — {f.file}:{f.line}" if f.file else ""
            lines.append(f"- [{f.severity}] {f.message}{loc}")
        lines.append("")
    if nits:
        lines.append(f"### 🔵 Nits ({len(nits)})")
        for f in nits:
            loc = f" — {f.file}:{f.line}" if f.file else ""
            lines.append(f"- [{f.severity}] {f.message}{loc}")
        lines.append("")

    lines.append("### Next steps")
    if verdict == "PASS":
        lines.append("1. 满足合并门禁；如策略开启 auto_merge 且操作者带 --apply，可执行合并")
    elif verdict == "REQUEST_CHANGES":
        if failures:
            lines.append(f"1. 优先解除硬门禁：{failures[0]['gate']} — {failures[0]['detail']}")
        else:
            lines.append("1. 评分低于阈值，按上方 Must/Should fix 修复后重新触发 gatekeeper")
        if tracking_issue not in (None, ""):
            lines.append(f"2. 关联 tracking issue #{tracking_issue}（已自动汇总必修项，修复后逐项勾选）")
    else:
        lines.append("1. 处于观察区间，建议处理 Should fix 项后复跑以争取 PASS")
    lines.append("---")
    lines.append("*Generated by gitlink-gatekeeper · policy-as-code PR gate · re-run after changes*")
    return "\n".join(lines)


def render_tracking_issue_body(
    inp: ScoreInput,
    dims: dict[str, Any],
    failures: list[dict[str, str]],
    routing: dict[str, Any] | None,
    owner: str,
    repo: str,
) -> str:
    """REQUEST_CHANGES 时创建的 tracking issue 正文。"""
    must = [f for f in inp.findings if f.severity in ("blocker", "major")]
    lines = [
        f"## Tracking — gatekeeper 拦截 PR #{inp.pr_id}",
        "",
        f"PR：`{owner}/{repo}` #{inp.pr_id} {inp.title}",
        f"裁决：**REQUEST_CHANGES** · Score {dims['total']}/100",
        "",
    ]
    if failures:
        lines.append("### 必须解除的硬门禁")
        for f in failures:
            lines.append(f"- [ ] `{f['gate']}`: {f['detail']}")
        lines.append("")
    if must:
        lines.append("### 必修项（blocker / major）")
        for f in must:
            loc = f" — {f.file}:{f.line}" if f.file else ""
            lines.append(f"- [ ] [{f.severity}] {f.message}{loc}")
        lines.append("")
    if routing and routing.get("suggested_reviewers"):
        lines.append("### 建议 reviewer")
        lines.append("- " + ", ".join(f"@{r}" for r in routing["suggested_reviewers"]))
        lines.append("")
    lines.append("> 修复后请在 PR 上复跑 gatekeeper；全部勾选完成后关闭本 issue。")
    lines.append("> Generated by gitlink-gatekeeper workflow.")
    return "\n".join(lines)


# --------------------------------------------------------------------------- #
# 采集编排
# --------------------------------------------------------------------------- #

def detect_linked_issue(description: str) -> bool:
    return bool(re.search(r"#\d+", description or ""))


def normalize_ci_status(builds_payload: Any) -> str:
    data = unwrap(builds_payload)
    items = extract_first_list(data, ("builds", "items", "list"))
    if not items:
        return "unknown"
    statuses = []
    for item in items:
        if isinstance(item, dict):
            s = str(first_value(item, ("status", "state", "result"), "")).lower()
            statuses.append(s)
    if not statuses:
        return "unknown"
    latest = statuses[0]
    if latest in ("success", "passing", "passed", "ok", "1"):
        return "passing"
    if latest in ("failure", "failing", "failed", "error", "2"):
        return "failing"
    return "unknown"


def collect_pr_context(
    cli_bin: str,
    owner: str,
    repo: str,
    pr_id: str,
    skip_ci: bool,
) -> dict[str, Any]:
    """采集 PR 上下文（只读 CLI 命令，见 SSOT 第 7 节）。"""
    view = unwrap(run_cli_json(cli_bin, ["pr", "+view", "-i", pr_id], owner, repo))
    view = view if isinstance(view, dict) else {}
    # GitLink 的 PR 由 issue 承载：title/description 在 view.issue（subject/description），
    # view.pull_request 只有合并相关字段。兼容直接返回 PR 对象的情况。
    issue = view.get("issue") if isinstance(view.get("issue"), dict) else {}
    pr = view.get("pull_request") if isinstance(view.get("pull_request"), dict) else {}
    # PR 背后承载的 issue id —— 给 PR 挂标签时的 Raw API 路径占位需要它。
    issue_id = first_value(issue, ("id", "issue", "number"), "")
    issue_id = str(issue_id) if issue_id not in (None, "") else ""
    title = str(
        first_value(issue, ("subject", "title", "name"))
        or first_value(view, ("title", "subject", "name"))
        or first_value(pr, ("title", "subject", "name"), f"PR #{pr_id}")
    )
    description = str(
        first_value(issue, ("description", "body", "notes"))
        or first_value(view, ("body", "description", "notes"))
        or first_value(pr, ("body", "description", "notes"), "")
    )

    files_payload = run_cli_json(cli_bin, ["pr", "+files", "-i", pr_id], owner, repo)
    changed_files = extract_file_paths(files_payload)

    # commits：SSOT 第 7 节用 Raw API GET /:owner/:repo/pulls/:id/commits。
    # 路径里的 :id 需替换为实际 PR 号；该端点在部分实例可能未开放，失败则降级
    # （commit_quality 按 total=0 给满分，见 SSOT 3.4），不阻断整条闭环。
    commits_payload: Any = {}
    try:
        commits_payload = run_cli_json(
            cli_bin,
            ["api", "GET", f"/:owner/:repo/pulls/{pr_id}/commits"],
            owner,
            repo,
        )
    except WorkflowError:
        commits_payload = {}
    commit_msgs: list[str] = []
    citems = extract_first_list(unwrap(commits_payload), ("commits", "items", "list"))
    for c in citems:
        if isinstance(c, dict):
            msg = first_value(c, ("message", "title", "commit_message"), "")
            commit = c.get("commit") if isinstance(c.get("commit"), dict) else None
            if not msg and commit:
                msg = first_value(commit, ("message", "title"), "")
            if msg:
                commit_msgs.append(str(msg))

    ci_status = "unknown"
    if not skip_ci:
        try:
            ci_status = normalize_ci_status(
                run_cli_json(cli_bin, ["ci", "+builds"], owner, repo)
            )
        except WorkflowError:
            ci_status = "unknown"

    return {
        "title": title,
        "description": description,
        "issue_id": issue_id,
        "changed_files": changed_files,
        "commits": commit_msgs,
        "ci_status": ci_status,
        "linked_issue": detect_linked_issue(description),
    }


def load_findings(path: Path | None) -> list[Finding]:
    """从 --findings JSON 注入 AI 审查发现；缺省为空（结果仍确定性可复现）。"""
    if path is None:
        return []
    if not path.exists():
        raise WorkflowError(f"找不到 findings 文件：{path}")
    raw = json.loads(path.read_text(encoding="utf-8"))
    items = raw.get("findings", raw) if isinstance(raw, dict) else raw
    findings: list[Finding] = []
    for item in items or []:
        if not isinstance(item, dict):
            continue
        sev = str(item.get("severity", "")).lower()
        if sev not in SEVERITY_ORDER:
            continue
        findings.append(
            Finding(
                severity=sev,
                message=str(item.get("message", "")),
                file=str(item.get("file", "")),
                line=item.get("line", ""),
            )
        )
    return findings


# --------------------------------------------------------------------------- #
# 善后：构造写操作计划（标签 / 评论 / tracking issue / 合并）
# --------------------------------------------------------------------------- #

def build_label_command(label_name: str, color: str = "#1E90FF") -> PlannedWrite:
    """确保裁决标签「定义」存在（label +create）。

    这是「打标签」两步中的第一步——仅创建标签定义本身。第二步「把标签挂到
    PR 背后的 issue」没有独立 shortcut，需走 Raw API
    POST /:owner/:repo/issues/:id 带 issue_tag_ids（见模块 docstring 与
    attach_label_to_pr）。挂载是受 --apply 显式门控的安全写操作：
    dry-run 只预览将要挂载的标签，apply 时才真正发起 POST。
    """
    return PlannedWrite(
        label=f"确保标签存在：{label_name}",
        command=["label", "+create", "-n", label_name, "-c", color],
        note="如标签已存在会返回冲突，可忽略",
    )


def lookup_label_id(
    cli_bin: str, owner: str, repo: str, label_name: str
) -> Any:
    """用 `label +list` 按 name 找出标签 id；找不到返回 None。

    GitLink 返回 data.issue_tags = [{id, name, ...}]。只读命令，dry-run 不调用。
    """
    payload = run_cli_json(cli_bin, ["label", "+list"], owner, repo)
    tags = extract_first_list(unwrap(payload), ("issue_tags", "tags", "items", "list"))
    for tag in tags:
        if isinstance(tag, dict) and str(tag.get("name", "")) == label_name:
            return tag.get("id")
    return None


def build_attach_label_command(
    issue_id: str,
    tag_id: Any,
    title: str,
    description: str,
    label_name: str,
) -> PlannedWrite:
    """把标签挂到 PR 背后 issue 的 Raw API 写操作。

    必须带回原 subject/description，否则 GitLink 会把标题/描述清空
    （见 gitlink-shared 实测）。
    """
    body = json.dumps(
        {
            "issue_tag_ids": [tag_id],
            "done_ratio": 0,
            "subject": title,
            "description": description,
        },
        ensure_ascii=False,
    )
    return PlannedWrite(
        label=f"挂载标签到 PR(issue {issue_id})：{label_name}",
        command=["api", "POST", f"/:owner/:repo/issues/{issue_id}", "--body", body],
        note=f"issue_tag_ids=[{tag_id}]",
    )


def attach_label_to_pr(
    cli_bin: str,
    owner: str,
    repo: str,
    issue_id: str,
    label_name: str,
    title: str,
    description: str,
) -> dict[str, Any]:
    """apply 时：查 tag id → Raw API 挂到 PR 背后 issue。封装多步逻辑保持 main 可读。

    任何一步失败都返回 {ok: False, ...} 而非抛出，保证整条闭环优雅降级。
    """
    if not issue_id:
        return {
            "label": f"挂载标签到 PR：{label_name}",
            "ok": False,
            "stderr": "未取到 PR 背后 issue id，跳过挂载",
            "data": None,
        }
    try:
        tag_id = lookup_label_id(cli_bin, owner, repo, label_name)
    except WorkflowError as exc:
        return {
            "label": f"挂载标签到 PR：{label_name}",
            "ok": False,
            "stderr": f"label +list 失败：{exc}",
            "data": None,
        }
    if tag_id in (None, ""):
        return {
            "label": f"挂载标签到 PR：{label_name}",
            "ok": False,
            "stderr": f"label +list 未找到标签 {label_name} 的 id",
            "data": None,
        }
    write = build_attach_label_command(issue_id, tag_id, title, description, label_name)
    return execute_write(write, owner, repo, cli_bin)


def build_comment_command(pr_id: str, body: str) -> PlannedWrite:
    return PlannedWrite(
        label=f"回写评分卡评论到 PR #{pr_id}",
        command=["pr", "+comment", "-i", pr_id, "-b", body],
        note=f"评分卡 {len(body)} 字符",
    )


def build_tracking_issue_command(title: str, body: str) -> PlannedWrite:
    return PlannedWrite(
        label="创建 tracking issue 汇总必修项",
        command=["issue", "+create", "-t", title, "-b", body],
        note=title,
    )


def parse_created_issue_number(res: dict[str, Any]) -> Any:
    """从 issue +create 的 execute_write 结果里回捕新建 issue 的编号/ID。

    GitLink 字段命名不统一（issue / id / number / pull_request），用 first_value
    兼容多键；取不到返回 None。
    """
    if not res or not res.get("ok"):
        return None
    data = unwrap(res.get("data"))
    if isinstance(data, dict):
        return first_value(data, ("issue", "id", "number", "pull_request"))
    return None


def build_merge_command(pr_id: str, method: str) -> PlannedWrite:
    return PlannedWrite(
        label=f"合并 PR #{pr_id}（{method}）",
        command=["pr", "+merge", "-i", pr_id, "-m", method],
        note="仅 PASS + auto_merge + --apply 时出现",
    )


# --------------------------------------------------------------------------- #
# 主流程
# --------------------------------------------------------------------------- #

def parse_args(argv: list[str] | None = None) -> argparse.Namespace:
    parser = argparse.ArgumentParser(
        prog="gatekeeper_workflow.py",
        description="gitlink-gatekeeper 端到端 PR 看门人闭环：路由 reviewer → 裁决 → 回写评论/标签/tracking issue。"
        " 默认 dry-run，写操作需 --apply。",
    )
    parser.add_argument("--config", type=Path, help="工作流配置文件（YAML），含 owner/repo/policy/owner_rules 路径")
    parser.add_argument("--owner", help="仓库 owner（覆盖配置）")
    parser.add_argument("--repo", help="仓库名（覆盖配置）")
    parser.add_argument("--pr", dest="pr_id", help="目标 PR 编号（覆盖配置）")
    parser.add_argument("--policy", type=Path, help="gatekeeper.yaml 策略路径（缺省回退内置默认策略）")
    parser.add_argument("--owner-rules", dest="owner_rules", type=Path, help="owner-rules.yaml 路径")
    parser.add_argument("--findings", type=Path, help="AI 审查发现 JSON（注入 review_findings；缺省为空）")
    parser.add_argument("--cli-bin", default="gitlink-cli", help="gitlink-cli 可执行文件路径")
    parser.add_argument("--skip-ci", action="store_true", help="跳过 CI 采集（ci_status 记为 unknown）")
    parser.add_argument("--output-dir", type=Path, default=Path("outputs"), help="本地产物输出目录")
    parser.add_argument(
        "--apply",
        action="store_true",
        help="执行写操作（回写评论 / 打标签 / 建 issue / 合并）。不传则仅预览（安全默认）。",
    )
    parser.add_argument("--no-color", action="store_true", help="不输出彩色（保留位，当前纯文本）")
    return parser.parse_args(argv)


def resolve_config(args: argparse.Namespace) -> dict[str, Any]:
    cfg: dict[str, Any] = {}
    if args.config:
        cfg = load_yaml_file(args.config)
    base = args.config.parent if args.config else Path.cwd()

    def resolve_path(value: Any) -> Path | None:
        if not value:
            return None
        p = Path(value)
        return p if p.is_absolute() else (base / p)

    owner = args.owner or cfg.get("owner")
    repo = args.repo or cfg.get("repo")
    pr_id = args.pr_id or (str(cfg["pr"]) if cfg.get("pr") is not None else None)
    policy_path = args.policy or resolve_path(cfg.get("policy"))
    owner_rules_path = args.owner_rules or resolve_path(cfg.get("owner_rules"))
    findings_path = args.findings or resolve_path(cfg.get("findings"))

    if not owner or not repo:
        raise WorkflowError("必须提供 owner 和 repo（通过 --config 或 --owner/--repo）")
    if not pr_id:
        raise WorkflowError("必须提供 PR 编号（通过 --config 的 pr 字段或 --pr）")
    if not owner_rules_path:
        raise WorkflowError("必须提供 owner-rules.yaml（通过 --config 的 owner_rules 或 --owner-rules）")

    return {
        "owner": str(owner),
        "repo": str(repo),
        "pr_id": str(pr_id),
        "policy_path": policy_path,
        "owner_rules_path": owner_rules_path,
        "findings_path": findings_path,
    }


def main(argv: list[str] | None = None) -> int:
    args = parse_args(argv)
    try:
        conf = resolve_config(args)
        owner, repo, pr_id = conf["owner"], conf["repo"], conf["pr_id"]
        cli_bin = args.cli_bin

        policy = load_policy(conf["policy_path"])
        policy_label = (
            f"{conf['policy_path'].name}@v{policy['version']}"
            if conf["policy_path"] and conf["policy_path"].exists()
            else f"builtin-default@v{policy['version']}"
        )
        rules, fallback = load_owner_rules(conf["owner_rules_path"])
        findings = load_findings(conf["findings_path"])

        print(f"=== gitlink-gatekeeper PR 看门人闭环 ===")
        print(f"目标：{owner}/{repo} PR #{pr_id} · policy: {policy_label}")
        print(f"模式：{'APPLY（将执行写操作）' if args.apply else 'DRY-RUN（仅预览，不写任何东西）'}")
        print()

        if not cli_available(cli_bin):
            raise WorkflowError(
                f"未找到 {cli_bin}，请先安装 gitlink-cli 并完成 `gitlink-cli auth login`"
            )

        # --- 步骤 1+2 采集 ---
        print("[1/3] 采集 PR 上下文并路由 reviewer …")
        ctx = collect_pr_context(cli_bin, owner, repo, pr_id, args.skip_ci)
        changed_src, changed_tests = count_src_test(ctx["changed_files"], policy)
        routing = route_reviewers(ctx["changed_files"], rules, fallback)
        print(f"      变更文件 {len(ctx['changed_files'])} 个（src {changed_src} / test {changed_tests}）")
        print(f"      建议 reviewer：{', '.join(routing['suggested_reviewers']) or '（无规则命中）'}")
        if routing["unmatched"]:
            print(f"      未命中规则文件 {len(routing['unmatched'])} 个 → fallback")
        print()

        # --- 步骤 2 裁决 ---
        print("[2/3] 按策略评分并裁决 …")
        inp = ScoreInput(
            pr_id=pr_id,
            title=ctx["title"],
            description=ctx["description"],
            changed_files=ctx["changed_files"],
            changed_src=changed_src,
            changed_tests=changed_tests,
            commits=ctx["commits"],
            ci_status=ctx["ci_status"],
            linked_issue=ctx["linked_issue"],
            findings=findings,
        )
        dims = score_dimensions(inp, policy)
        failures = evaluate_hard_gates(inp, policy)
        verdict = decide_verdict(dims["total"], bool(failures), policy)
        scorecard = render_scorecard(inp, dims, failures, verdict, policy_label, routing)
        print(f"      Score: {dims['total']}/100 · 硬门禁失败 {len(failures)} 项 · 裁决: {verdict}")
        print()

        # 落盘本地产物（无论 dry-run 与否都生成，便于复核 / 验证记录）
        out_dir: Path = args.output_dir
        out_dir.mkdir(parents=True, exist_ok=True)
        slug = f"{owner}_{repo}_pr{pr_id}".replace("/", "_")
        scorecard_path = out_dir / f"{slug}_scorecard.md"
        summary_path = out_dir / f"{slug}_summary.json"
        scorecard_path.write_text(scorecard + "\n", encoding="utf-8")

        # --- 步骤 3 善后 ---
        print("[3/3] 回写 + 善后 …")
        behavior = policy["behavior"]
        labels = policy["labels"]
        verdict_label = {
            "PASS": labels["pass"],
            "REQUEST_CHANGES": labels["request_changes"],
            "COMMENT": labels["comment"],
        }[verdict]

        results: list[dict[str, Any]] = []
        tracking_issue_no: Any = None

        # REQUEST_CHANGES：apply 时「先建 tracking issue → 回捕编号 → 把编号补进评分卡
        # Next steps → 再回写评论」，让评论与 issue 双向可追溯。
        issue_title = issue_body = ""
        if verdict == "REQUEST_CHANGES":
            issue_title = f"[gatekeeper] PR #{pr_id} 未通过门禁：{ctx['title'][:60]}"
            issue_body = render_tracking_issue_body(inp, dims, failures, routing, owner, repo)
            if args.apply:
                print("      APPLY：执行写操作 …")
                tracking_write = build_tracking_issue_command(issue_title, issue_body)
                res = execute_write(tracking_write, owner, repo, cli_bin)
                results.append(res)
                tracking_issue_no = parse_created_issue_number(res)
                status = "OK" if res["ok"] else f"FAIL（{res['stderr'][:120]}）"
                print(f"  • {tracking_write.label} → {status}")
                if tracking_issue_no not in (None, ""):
                    print(f"  • tracking issue 已建：#{tracking_issue_no}")
                    # 把编号补进评分卡 Next steps 后重新落盘（评论将带上回链）
                    scorecard = render_scorecard(
                        inp, dims, failures, verdict, policy_label, routing,
                        tracking_issue=tracking_issue_no,
                    )
                    scorecard_path.write_text(scorecard + "\n", encoding="utf-8")

        # 构造（其余）写操作计划。评论用上面可能已带回链的 scorecard。
        plan: list[PlannedWrite] = []
        if behavior.get("post_comment", True):
            plan.append(build_comment_command(pr_id, scorecard))
        if behavior.get("apply_label", True):
            plan.append(build_label_command(verdict_label))
        # dry-run 下把 tracking issue 也列进计划预览（apply 路径已在上面执行掉）
        if verdict == "REQUEST_CHANGES" and not args.apply:
            plan.append(build_tracking_issue_command(issue_title, issue_body))
        if (
            verdict == "PASS"
            and behavior.get("auto_merge", False)
            and args.apply
        ):
            plan.append(build_merge_command(pr_id, behavior.get("merge_method", "squash")))

        if not args.apply:
            print("      DRY-RUN：以下写操作不会执行（加 --apply 才执行）：")
            for w in plan:
                print(render_planned(w, owner, repo, cli_bin))
            if behavior.get("apply_label", True):
                print(
                    f"  • （计划）apply 时将把标签 {verdict_label} 挂到 PR 背后 issue"
                    f"（label +list 查 id → Raw API POST /:owner/:repo/issues/<issue_id>）"
                )
        else:
            if verdict != "REQUEST_CHANGES":
                print("      APPLY：执行写操作 …")
            for w in plan:
                res = execute_write(w, owner, repo, cli_bin)
                status = "OK" if res["ok"] else f"FAIL（{res['stderr'][:120]}）"
                print(f"  • {w.label} → {status}")
                results.append(res)
            # 标签真正挂到 PR：create 之后查 id 并 Raw API 挂载（受 --apply 门控）
            if behavior.get("apply_label", True):
                attach_res = attach_label_to_pr(
                    cli_bin, owner, repo, ctx.get("issue_id", ""),
                    verdict_label, ctx["title"], ctx["description"],
                )
                status = "OK" if attach_res["ok"] else f"FAIL（{attach_res['stderr'][:120]}）"
                iid = ctx.get("issue_id", "") or "?"
                print(f"  • 挂载标签到 PR(issue {iid}) → {status}")
                results.append(attach_res)
        print()

        # 结构化摘要
        summary = {
            "owner": owner,
            "repo": repo,
            "pr_id": pr_id,
            "policy": policy_label,
            "mode": "apply" if args.apply else "dry-run",
            "routing": routing,
            "scores": {k: v for k, v in dims.items() if k != "total"},
            "total": dims["total"],
            "hard_gate_failures": failures,
            "verdict": verdict,
            "verdict_label": verdict_label,
            "tracking_issue": tracking_issue_no,
            "planned_writes": [
                {"label": w.label, "command": w.command, "note": w.note} for w in plan
            ],
            "executed": results,
            "artifacts": {
                "scorecard": scorecard_path.as_posix(),
                "summary": summary_path.as_posix(),
            },
        }
        summary_path.write_text(
            json.dumps(summary, ensure_ascii=False, indent=2), encoding="utf-8"
        )

        print(f"评分卡已落盘：{scorecard_path}")
        print(f"结构化摘要：{summary_path}")
        print(f"最终裁决：{VERDICT_EMOJI[verdict]} {verdict} · {dims['total']}/100")
        # REQUEST_CHANGES 时返回码 2，便于 CI 接入做门禁；PASS/COMMENT 返回 0。
        return 2 if verdict == "REQUEST_CHANGES" else 0

    except WorkflowError as exc:
        print(f"错误：{exc}", file=sys.stderr)
        return 1


if __name__ == "__main__":
    raise SystemExit(main())
