"""gitlink_data.py — 子赛题四数据访问共享层。

职责：
  1. 以子进程方式调用 gitlink-cli，解析统一 envelope（{ok,data,error,meta}）。
  2. 内置限速，避免触发 GitLink API 限流（参考 shortcuts/health 的 ~1.7 call/s）。
  3. 列表分页累积。
  4. 直接只读访问 health SQLite（shortcuts/health/schema.sql），供 S2 图谱 / S4 匹配取历史协作数据。

设计原则：本层只“取数 + 解析”，不做任何业务算法；算法在各场景脚本中实现。
所有 GitLink 操作一律经 gitlink-cli，绝不用 gh/glab（见 skills/gitlink-shared/SKILL.md 工具边界）。
"""
from __future__ import annotations

import json
import os
import sqlite3
import subprocess
import sys
import time
from pathlib import Path
from typing import Any, Iterable

# ---------------------------------------------------------------------------
# 配置
# ---------------------------------------------------------------------------

# CLI 可执行路径：默认走 PATH 上的 gitlink-cli；本地 Windows 开发可设 GITLINK_CLI=./gitlink-cli.exe
def cli_path() -> str:
    return os.environ.get("GITLINK_CLI", "gitlink-cli")


# 最小调用间隔（秒），限制对 GitLink API 的请求频率。
MIN_INTERVAL = float(os.environ.get("GITLINK_CLI_INTERVAL", "0.6"))

_last_call = 0.0


def _throttle() -> None:
    """简单的全局令牌限速：两次调用间至少间隔 MIN_INTERVAL 秒。"""
    global _last_call
    now = time.time()
    wait = MIN_INTERVAL - (now - _last_call)
    if wait > 0:
        time.sleep(wait)
    _last_call = time.time()


def _warn(msg: str) -> None:
    sys.stderr.write(f"[gitlink] {msg}\n")


# ---------------------------------------------------------------------------
# 核心：调用 CLI
# ---------------------------------------------------------------------------

def run(args: Iterable[str], owner: str | None = None, repo: str | None = None,
        fmt: str = "json") -> dict[str, Any]:
    """调用 `gitlink-cli [global flags] args... --format fmt`，返回解析后的 envelope dict。

    失败时返回 {"ok": False, "error": {...}}，不抛异常，便于调用方容错。
    """
    cmd = [cli_path()]
    if owner:
        cmd += ["--owner", owner]
    if repo:
        cmd += ["--repo", repo]
    cmd += list(args)
    if fmt:
        cmd += ["--format", fmt]

    _throttle()
    try:
        proc = subprocess.run(cmd, capture_output=True, text=True, timeout=120)
    except FileNotFoundError:
        _warn(f"gitlink-cli 未找到（cmd={cmd[0]}），请设置 GITLINK_CLI 环境变量")
        return {"ok": False, "error": {"message": f"gitlink-cli not found: {cmd[0]}"}}
    except subprocess.TimeoutExpired:
        _warn(f"调用超时：{' '.join(cmd)}")
        return {"ok": False, "error": {"message": "timeout"}}

    if proc.returncode != 0:
        _warn(f"exit={proc.returncode} cmd={' '.join(cmd)}\n{proc.stderr.strip()}")
        return {"ok": False, "error": {"message": (proc.stderr or proc.stdout).strip() or f"exit {proc.returncode}"}}

    out = proc.stdout.strip()
    if not out:
        return {"ok": True, "data": None}
    try:
        return json.loads(out)
    except json.JSONDecodeError:
        # 非 JSON（表格/原始文本），原样包裹返回
        return {"ok": True, "data": out}


def run_data(args: Iterable[str], owner: str | None = None, repo: str | None = None,
             fmt: str = "json") -> Any:
    """run() 的便捷封装：成功返回 data 字段，失败返回 None 并告警。"""
    env = run(args, owner=owner, repo=repo, fmt=fmt)
    if not env.get("ok"):
        msg = (env.get("error") or {}).get("message", "unknown error")
        _warn(f"{list(args)} -> {msg}")
        return None
    return env.get("data")


def api(method: str, path: str, query: str | None = None,
        owner: str | None = None, repo: str | None = None) -> Any:
    """调用 Raw API：`gitlink-cli api METHOD PATH --query ... --format json`。

    用于 shortcuts 未覆盖的端点（如仓库提交历史 GET /{owner}/{repo}/commits）。
    """
    args = ["api", method, path]
    if query:
        args += ["--query", query]
    return run_data(args, owner=owner, repo=repo)


# ---------------------------------------------------------------------------
# 分页 / 形状工具
# ---------------------------------------------------------------------------

# GitLink 各端点把列表放在不同键下；按优先级尝试这些键。
DEFAULT_LIST_KEYS = (
    "projects", "repos", "issues", "pulls", "users", "list",
    "contributors", "entries", "commits", "milestones", "tags",
)


def first_list(data: Any, keys: Iterable[str] = DEFAULT_LIST_KEYS) -> list:
    """从 envelope.data 里稳健地取出列表：data 本身是列表则直接返回，否则尝试已知键。"""
    if isinstance(data, list):
        return data
    if isinstance(data, dict):
        for k in keys:
            v = data.get(k)
            if isinstance(v, list):
                return v
        # 兜底：唯一一个 list 值
        list_vals = [v for v in data.values() if isinstance(v, list)]
        if len(list_vals) == 1:
            return list_vals[0]
    return []


def total_count(data: Any) -> int | None:
    if isinstance(data, dict):
        for k in ("total_count", "totalCount", "count", "total"):
            if isinstance(data.get(k), (int, float)):
                return int(data[k])
    return None


def paginate(domain: str, verb: str, list_keys=DEFAULT_LIST_KEYS,
             page_size: int = 50, max_pages: int = 20,
             owner: str | None = None, repo: str | None = None,
             extra_flags: Iterable[str] = ()) -> list:
    """对一个 `gitlink-cli <domain> +<verb>` 列表命令做多页累积。

    依赖命令支持 --page/--limit 两个 flag（repo/issue/pr/search/milestone 等均支持）。
    """
    collected: list = []
    for page in range(1, max_pages + 1):
        flags = [f"+{verb}", "--page", str(page), "--limit", str(page_size), *extra_flags]
        data = run_data([domain, *flags], owner=owner, repo=repo)
        if data is None:
            break
        items = first_list(data, list_keys)
        if not items:
            break
        collected.extend(items)
        total = total_count(data)
        if total is not None and len(collected) >= total:
            break
        if len(items) < page_size:
            break
    return collected


# ---------------------------------------------------------------------------
# health SQLite 只读访问
# ---------------------------------------------------------------------------

def health_db_path() -> str:
    return os.environ.get(
        "GITLINK_HEALTH_DB",
        str(Path.home() / ".agents" / "skills" / "gitlink-health" / "data" / "gitlink_health.db"),
    )


def open_health_db() -> sqlite3.Connection | None:
    """以只读方式打开 health SQLite；文件不存在则返回 None（调用方退化为纯 API 取数）。"""
    p = health_db_path()
    if not Path(p).exists():
        return None
    # mode=ro 防止误写；modernc/sqlite 已开 WAL，Python 只读并发安全。
    return sqlite3.connect(f"file:{p}?mode=ro", uri=True)
