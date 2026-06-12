"""gitlink-cli 命令封装层。

本模块把 `gitlink-cli` 的子命令（issue +list、pr +list、repo +info、
release +list 等）封装为 Python 函数，是 gitlink-flow 工作流的**主调用链**。

设计依据赛题要求——工作流应优先组合 gitlink-cli 已有命令，而非自行直连
平台 HTTP API。因此 flow.py 默认通过本模块调用 gitlink-cli；仅当本机
未安装 gitlink-cli 时，才回退到 glapi.py 的直连方式（见 GitLinkClient）。

gitlink-cli 的 JSON 输出统一为 {"ok": bool, "data": {...}}，本模块负责
解包 data 并归一化为与 glapi 相同的返回结构，使上层步骤无感知。
"""

from __future__ import annotations

import json
import shutil
import subprocess
from typing import Any


class CliError(RuntimeError):
    """gitlink-cli 调用失败。"""


def cli_available() -> bool:
    """检测本机是否安装了 gitlink-cli。"""
    return shutil.which("gitlink-cli") is not None


def _run(args: list[str], timeout: int = 60) -> Any:
    """执行 gitlink-cli 子命令，返回解包后的 data。

    统一追加 --format json，解析 {"ok":..., "data":...} 信封。
    """
    cmd = ["gitlink-cli", *args, "--format", "json"]
    try:
        proc = subprocess.run(cmd, capture_output=True, text=True,
                              encoding="utf-8", timeout=timeout)
    except FileNotFoundError as exc:
        raise CliError("未找到 gitlink-cli，请先安装：npm i -g @gitlink-ai/cli") from exc
    except subprocess.TimeoutExpired as exc:
        raise CliError(f"gitlink-cli 调用超时：{' '.join(args)}") from exc

    out = (proc.stdout or "").strip()
    # 跳过可能的环境横幅行，定位 JSON 起始
    brace = out.find("{")
    if brace > 0:
        out = out[brace:]
    if not out:
        raise CliError(f"gitlink-cli 无输出：{' '.join(args)}；stderr={proc.stderr.strip()}")
    try:
        payload = json.loads(out)
    except json.JSONDecodeError as exc:
        raise CliError(f"gitlink-cli 输出非 JSON：{' '.join(args)}") from exc

    if isinstance(payload, dict) and payload.get("ok") is False:
        raise CliError(f"gitlink-cli 返回失败：{payload.get('message') or payload}")
    if isinstance(payload, dict) and "data" in payload:
        return payload["data"]
    return payload


def _extract_list(data: Any, keys: tuple[str, ...]) -> list[dict[str, Any]]:
    if isinstance(data, list):
        return data
    if isinstance(data, dict):
        for k in keys:
            v = data.get(k)
            if isinstance(v, list):
                return v
    return []


# ---------------------------------------------------------------------------
# 高层封装——签名与 glapi.GitLinkClient 对齐
# ---------------------------------------------------------------------------

def repo_info(owner: str, repo: str) -> dict[str, Any]:
    data = _run(["repo", "+info", "--owner", owner, "--repo", repo])
    return data if isinstance(data, dict) else {}


def issues(owner: str, repo: str, limit: int = 50) -> list[dict[str, Any]]:
    data = _run(["issue", "+list", "--owner", owner, "--repo", repo])
    return _extract_list(data, ("issues",))


def pulls(owner: str, repo: str, limit: int = 50) -> list[dict[str, Any]]:
    data = _run(["pr", "+list", "--owner", owner, "--repo", repo])
    return _extract_list(data, ("issues", "pulls"))


def releases(owner: str, repo: str) -> list[dict[str, Any]]:
    data = _run(["release", "+list", "--owner", owner, "--repo", repo])
    return _extract_list(data, ("releases",))
