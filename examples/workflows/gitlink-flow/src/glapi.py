"""GitLink 公开 API 共享客户端。

供 gitlink-skills-pack 下各 Skill 的脚本复用。仅依赖 Python 标准库，
无需第三方包，便于在受限环境或 Agent 沙箱中运行。

数据全部来自 GitLink 平台公开接口（https://www.gitlink.org.cn/api），
默认无需 token；如需访问私有仓库，可传入 token。

所有方法均为只读，不修改任何远程数据。
"""

from __future__ import annotations

import base64
import json
import time
import urllib.error
import urllib.parse
import urllib.request
from pathlib import Path
from typing import Any

API_BASE = "https://www.gitlink.org.cn/api"
USER_AGENT = "gitlink-skills-pack/1.0 (+https://www.gitlink.org.cn)"
DEFAULT_TIMEOUT = 30
COMMIT_PAGE_SIZE = 50  # GitLink commits 接口每页硬上限


class GitLinkError(RuntimeError):
    """API 调用中不可恢复的错误。"""


class GitLinkClient:
    """GitLink 公开数据接口客户端。

    带可选文件缓存：同一资源重复读取不重复打网，对平台友好。
    """

    def __init__(self, base: str = API_BASE, token: str | None = None,
                 timeout: int = DEFAULT_TIMEOUT, cache_dir: Path | None = None) -> None:
        self.base = base.rstrip("/")
        self.token = token
        self.timeout = timeout
        self.cache_dir = cache_dir
        if self.cache_dir:
            self.cache_dir.mkdir(parents=True, exist_ok=True)

    # ------------------------------------------------------------------
    # 底层请求
    # ------------------------------------------------------------------
    def _cache_path(self, url: str) -> Path | None:
        if not self.cache_dir:
            return None
        safe = urllib.parse.quote(url, safe="")
        return self.cache_dir / f"{safe}.json"

    def get(self, path: str, query: dict[str, Any] | None = None) -> Any:
        """GET 请求，返回解析后的 JSON（dict/list）或 None。"""
        url = f"{self.base}/{path.lstrip('/')}"
        if query:
            url = f"{url}?{urllib.parse.urlencode(query)}"

        cache_path = self._cache_path(url)
        if cache_path and cache_path.exists():
            return json.loads(cache_path.read_text(encoding="utf-8"))

        headers = {"Accept": "application/json", "User-Agent": USER_AGENT}
        if self.token:
            headers["Authorization"] = f"Bearer {self.token}"

        req = urllib.request.Request(url, headers=headers)
        try:
            with urllib.request.urlopen(req, timeout=self.timeout) as resp:
                raw = resp.read().decode("utf-8", errors="replace")
        except urllib.error.HTTPError as exc:
            raise GitLinkError(f"HTTP {exc.code}: {url}") from exc
        except urllib.error.URLError as exc:
            raise GitLinkError(f"网络错误: {url} -> {exc.reason}") from exc

        text = raw.strip()
        if not text or text in ("null", "{}", "[]"):
            data: Any = None
        elif text[0] in "{[":
            try:
                data = json.loads(text)
            except json.JSONDecodeError as exc:
                raise GitLinkError(f"响应非 JSON: {url}") from exc
        else:
            raise GitLinkError(f"响应非 JSON（可能是 HTML）: {url}")

        if cache_path is not None:
            cache_path.write_text(json.dumps(data, ensure_ascii=False), encoding="utf-8")
        return data

    # ------------------------------------------------------------------
    # 资源访问（高层封装）
    # ------------------------------------------------------------------
    def repo_info(self, owner: str, repo: str) -> dict[str, Any]:
        """仓库元信息。"""
        data = self.get(f"{owner}/{repo}.json")
        return data if isinstance(data, dict) else {}

    def issues(self, owner: str, repo: str, limit: int = 50,
               page: int = 1) -> list[dict[str, Any]]:
        """Issue 列表。"""
        data = self.get(f"{owner}/{repo}/issues.json", {"page": page, "limit": limit})
        return _extract_list(data, ("issues",))

    def issue_detail(self, owner: str, repo: str, number: int) -> dict[str, Any]:
        """单个 Issue 详情（含完整字段）。"""
        data = self.get(f"{owner}/{repo}/issues/{number}.json")
        return data if isinstance(data, dict) else {}

    def pulls(self, owner: str, repo: str, limit: int = 50,
              page: int = 1) -> list[dict[str, Any]]:
        """PR 列表。"""
        data = self.get(f"{owner}/{repo}/pulls.json", {"page": page, "limit": limit})
        return _extract_list(data, ("issues", "pulls"))

    def contributors(self, owner: str, repo: str) -> list[dict[str, Any]]:
        """贡献者列表。"""
        data = self.get(f"{owner}/{repo}/contributors.json")
        return _extract_list(data, ("list",))

    def commits(self, owner: str, repo: str, max_pages: int = 4) -> list[dict[str, Any]]:
        """提交列表（按需翻页，每页 50 条，以 total_count 为终止依据）。"""
        out: list[dict[str, Any]] = []
        total: int | None = None
        for page in range(1, max(1, max_pages) + 1):
            data = self.get(f"{owner}/{repo}/commits.json",
                            {"page": page, "limit": COMMIT_PAGE_SIZE})
            if total is None and isinstance(data, dict):
                total = _safe_int(data.get("total_count")) or None
            page_items = _extract_list(data, ("commits",))
            if not page_items:
                break
            out.extend(page_items)
            if total is not None and len(out) >= total:
                break
        return out

    def list_dir(self, owner: str, repo: str, path: str = "",
                 ref: str = "master") -> list[dict[str, Any]]:
        """列出目录下的条目（文件与子目录）。

        返回的每个 entry 含 name / path / type(file|dir) / sha / size，
        文件类型的 entry 还可能直接带明文 content。
        """
        data = self.get(f"{owner}/{repo}/sub_entries.json",
                        {"filepath": path, "ref": ref})
        # 查询目录时 entries 为 list；查询单文件时 entries 为单个 dict。
        # 统一归一化为 list，便于下游处理。
        if isinstance(data, dict):
            entries = data.get("entries")
            if isinstance(entries, dict):
                return [entries]
            if isinstance(entries, list):
                return entries
        return _extract_list(data, ("entries",))

    def file_content(self, owner: str, repo: str, filepath: str,
                     ref: str = "master") -> str | None:
        """读取单个文件的文本内容。

        GitLink 的 sub_entries 接口对单文件查询会在 entries 中返回明文 content，
        据此取出。文件不存在或无内容时返回 None。
        """
        entries = self.list_dir(owner, repo, filepath, ref)
        target = filepath.rsplit("/", 1)[-1]
        for entry in entries:
            if entry.get("type") == "file" and entry.get("name") == target:
                content = entry.get("content")
                if isinstance(content, str):
                    return content
        # 回退：部分情况下单文件查询 entries 仅一项
        if len(entries) == 1 and entries[0].get("type") == "file":
            content = entries[0].get("content")
            if isinstance(content, str):
                return content
        return None

    def releases(self, owner: str, repo: str, limit: int = 50,
                 page: int = 1) -> list[dict[str, Any]]:
        """版本发布列表。"""
        data = self.get(f"{owner}/{repo}/releases.json", {"page": page, "limit": limit})
        return _extract_list(data, ("releases",))

    def readme(self, owner: str, repo: str, ref: str = "master") -> str | None:
        """读取仓库 README（自动 base64 解码）。"""
        data = self.get(f"{owner}/{repo}/readme.json", {"ref": ref})
        if not isinstance(data, dict):
            return None
        content = data.get("content")
        if not isinstance(content, str):
            return None
        # 注意：GitLink 的 readme.json 虽然 encoding 标为 base64，
        # 实测 content 多为明文 Markdown。先探测明文特征，命中则直接返回；
        # 否则再尝试 base64 解码。
        stripped = content.lstrip()
        if stripped.startswith(("#", "<", "[", "-", "*", "本", "这", "项")) or "\n" in content[:200]:
            return content
        try:
            raw = base64.b64decode(content.encode("ascii", "ignore"))
            decoded = raw.decode("utf-8", errors="replace")
            # 解码结果若不像文本（大量替换符），回退为原文
            if decoded.count("\ufffd") > len(decoded) * 0.1:
                return content
            return decoded
        except (ValueError, TypeError):
            return content


# ----------------------------------------------------------------------------
# 辅助
# ----------------------------------------------------------------------------

def _extract_list(payload: Any, keys: tuple[str, ...]) -> list[Any]:
    """从可能嵌套的响应中提取第一个匹配键的列表。"""
    if isinstance(payload, list):
        return payload
    if isinstance(payload, dict):
        for key in keys:
            value = payload.get(key)
            if isinstance(value, list):
                return value
    return []


def _safe_int(value: Any, default: int = 0) -> int:
    try:
        return int(value)
    except (TypeError, ValueError):
        return default


def split_owner_repo(slug: str) -> tuple[str, str]:
    """把 'owner/repo' 或完整 URL 解析为 (owner, repo)。"""
    s = slug.strip()
    if s.startswith("http"):
        parts = urllib.parse.urlparse(s).path.strip("/").split("/")
        if len(parts) >= 2:
            return parts[0], parts[1].replace(".git", "")
        raise GitLinkError(f"无法从 URL 解析 owner/repo: {slug}")
    if "/" in s:
        owner, repo = s.split("/", 1)
        return owner, repo.replace(".git", "")
    raise GitLinkError(f"格式应为 owner/repo: {slug}")
