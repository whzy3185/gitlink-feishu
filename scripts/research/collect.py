"""collect.py — 子赛题四共享数据采集器。

每个采集器是对 `gitlink-cli <domain> +<verb>` 或 Raw API 的薄封装，返回归一化后的
Python 对象（dict/list）。所有 GitLink 操作经 gitlink-cli（见 gitlink-shared 工具边界）。

形状说明：GitLink 不同端点字段差异较大，本模块只做“尽力归一”，把不确定字段原样透传；
具体字段解读留给各场景脚本（lineage/graph/match/...）并在其阶段用真实仓库验证后收敛。
"""
from __future__ import annotations

from typing import Any, Iterable

import gitlink_data as gd

# ---------------------------------------------------------------------------
# 归一化小工具
# ---------------------------------------------------------------------------

def as_str(v: Any) -> str:
    if v is None:
        return ""
    return str(v)


def as_int(v: Any, default: int = 0) -> int:
    try:
        return int(v)
    except (TypeError, ValueError):
        return default


def as_float(v: Any, default: float = 0.0) -> float:
    try:
        return float(v)
    except (TypeError, ValueError):
        return default


def login_of(obj: dict) -> str:
    """从用户/作者对象里尽量取出 login（GitLink 嵌套形式多变）。"""
    if not isinstance(obj, dict):
        return ""
    for path_keys in (("login",), ("name",), ("username",),
                      ("author", "login"), ("user", "login"),
                      ("owner", "login"), ("author", "name")):
        cur: Any = obj
        ok = True
        for k in path_keys:
            if isinstance(cur, dict) and k in cur:
                cur = cur[k]
            else:
                ok = False
                break
        if ok and isinstance(cur, str) and cur:
            return cur
    return ""


def repo_fullname(project: dict) -> str:
    """从 /projects 里的仓库对象取 'owner/identifier' 全名。"""
    if not isinstance(project, dict):
        return ""
    identifier = project.get("identifier") or project.get("name") or project.get("repo_name") or ""
    owner = login_of(project.get("author") or project.get("owner") or {}) or as_str(project.get("owner_login"))
    if owner and identifier:
        return f"{owner}/{identifier}"
    return identifier


# ---------------------------------------------------------------------------
# 仓库级
# ---------------------------------------------------------------------------

def repo_info(owner: str, repo: str) -> dict:
    return gd.run_data(["repo", "+info"], owner=owner, repo=repo) or {}


def readme(owner: str, repo: str, ref: str = "master") -> str:
    data = gd.run_data(["repo", "+readme", "--ref", ref], owner=owner, repo=repo)
    if isinstance(data, str):
        return data
    if isinstance(data, dict):
        for k in ("content", "text", "readme", "markdown", "data"):
            v = data.get(k)
            if isinstance(v, str) and v:
                return v
    return ""


def tree(owner: str, repo: str, path: str = "", ref: str = "master") -> list:
    flags = ["repo", "+tree", "--ref", ref]
    if path:
        flags += ["--path", path]
    data = gd.run_data(flags, owner=owner, repo=repo)
    return gd.first_list(data, ("entries", "trees", "files", "sub_entries"))


def languages(owner: str, repo: str) -> dict:
    data = gd.run_data(["repo", "+languages"], owner=owner, repo=repo)
    return data if isinstance(data, dict) else {}


def contributors(owner: str, repo: str, limit: int = 100) -> list:
    """贡献者列表。`repo +contributors` 无分页 flag（见 shortcuts/repo/repo.go），
    故单次取全量；limit 仅为兼容保留、不传给 CLI。"""
    data = gd.run_data(["repo", "+contributors"], owner=owner, repo=repo)
    return gd.first_list(data, ("contributors", "list"))


# ---------------------------------------------------------------------------
# Issue / PR / 里程碑（分页）
# ---------------------------------------------------------------------------

def issues(owner: str, repo: str, state: str = "all", max_pages: int = 10,
           page_size: int = 50) -> list:
    extra = ["--state", state] if state and state != "all" else []
    return gd.paginate("issue", "list", owner=owner, repo=repo,
                       page_size=page_size, max_pages=max_pages, extra_flags=extra)


def prs(owner: str, repo: str, state: str = "all", max_pages: int = 10,
        page_size: int = 50) -> list:
    extra = ["--state", state] if state and state != "all" else []
    # GitLink 把 PR 列表也放在 data["issues"] 键下（见 shortcuts/health/api.go）。
    return gd.paginate("pr", "list", owner=owner, repo=repo,
                       list_keys=("issues", "pulls", "list"),
                       page_size=page_size, max_pages=max_pages, extra_flags=extra)


def issues_all(owner: str, repo: str, max_pages: int = 10,
               page_size: int = 50) -> list:
    """全部 Issue（open + closed 合并去重）。

    `issue +list` 默认只返 open；取 open/closed 两个状态并集按 id 去重，
    每条保留真实 status 字段。供 S5/S6 做全量统计。
    """
    seen: dict = {}
    for state in ("open", "closed"):
        for iss in issues(owner, repo, state=state,
                          max_pages=max_pages, page_size=page_size):
            key = iss.get("id") or iss.get("index")
            if key is not None and key not in seen:
                seen[key] = iss
    return list(seen.values())


def prs_all(owner: str, repo: str, max_pages: int = 10,
            page_size: int = 50) -> list:
    """全部 PR（open + merged + closed 合并去重）。

    GitLink 的 `pr +list` 状态过滤不可靠（不同仓库行为不一：有的默认只返 open，
    有的 --state 不生效返混合），故取三个状态并集按 index 去重，每条 PR 保留其
    真实 status 字段（'merged'/'open'/'closed'）供上层分类。仿 health 采集法。
    """
    seen: dict = {}
    for state in ("open", "merged", "closed"):
        for pr in prs(owner, repo, state=state,
                      max_pages=max_pages, page_size=page_size):
            key = pr.get("index") or pr.get("id") or pr.get("number")
            if key is None:
                continue
            if key not in seen:
                seen[key] = pr
    return list(seen.values())


def pr_detail(owner: str, repo: str, index: int) -> dict:
    """单个 PR 详情：`pr +view --id <index>`。

    PR 列表 API 不返回文件改动数；详情接口返回 `files_count` / `commits_count`，
    用于 S1 创新点识别的「大规模重构」判据。
    """
    return gd.run_data(["pr", "+view", "--id", str(index)], owner=owner, repo=repo) or {}


def milestones(owner: str, repo: str, state: str = "all") -> list:
    extra = ["--status", state] if state and state != "all" else []
    data = gd.run_data(["milestone", "+list", *extra, "--limit", "100"],
                       owner=owner, repo=repo)
    return gd.first_list(data, ("milestones", "list"))


# ---------------------------------------------------------------------------
# 搜索 / 用户
# ---------------------------------------------------------------------------

def search_repos(keyword: str, limit: int = 20) -> list:
    data = gd.run_data(["search", "+repos", "-k", keyword, "--limit", str(limit)])
    return gd.first_list(data, ("projects", "repos"))


def search_users(keyword: str, limit: int = 20) -> list:
    data = gd.run_data(["search", "+users", "-k", keyword, "--limit", str(limit)])
    return gd.first_list(data, ("users", "list"))


# ---------------------------------------------------------------------------
# GitLink 官方「分类精选 / 探索」源（pinned=d）—— 热点追踪的主力数据源
#   GET /api/project_categories.json  -> [{id,name}, ...]（约 44 个领域）
#   GET /api/projects.json?pinned=d&category_id=N&limit=M
# ---------------------------------------------------------------------------

def categories() -> list:
    """GitLink 全部领域分类（id + name，约 44 个）。公开、免鉴权。

    优先走 `explore +categories`；失败回退 Raw API。
    """
    data = gd.run_data(["explore", "+categories"])
    if isinstance(data, dict) and data.get("project_categories"):
        return data["project_categories"]
    data = gd.api("GET", "/project_categories") or {}
    return data.get("project_categories") if isinstance(data, dict) else []


def category_to_id(category: Any) -> str:
    """分类名 → id（已是数字原样返回）；查不到返回原值。"""
    s = as_str(category)
    if s.isdigit():
        return s
    for c in categories():
        if as_str(c.get("name")) == s:
            return as_str(c.get("id"))
    return s


def pinned(category: Any, limit: int = 20) -> list:
    """某分类下 GitLink 官方精选项目（pinned=d，curated）。

    category 可传中文名（如 "深度学习"）或 id（如 32）。每项含：
    identifier / name / visits / praises_count / forked_count / time_ago /
    author{login} / topics / language。owner 取自 author.login（见 repo_fullname）。
    """
    data = gd.run_data(["explore", "+pinned", "--category", str(category),
                        "--limit", str(limit)])
    projects = gd.first_list(data, ("projects", "repos")) if data is not None else []
    if not projects:  # 回退 Raw API（旧二进制无 explore 域时）
        cid = category_to_id(category)
        data = gd.api("GET", "/projects", query=f"pinned=d&category_id={cid}&limit={limit}")
        projects = gd.first_list(data, ("projects", "repos"))
    return projects or []


def user_info(login: str) -> dict:
    return gd.run_data(["user", "+info", "--login", login]) or {}


def user_repos(login: str, limit: int = 20) -> list:
    """某用户的公开仓库列表（`repo +list --user <login> --category all`）。

    必须显式传 `--category all`：`repo +list` 的 category 默认是 `manage`
    （"我管理的"），用来列别人的仓库时会 404/空。返回 data.projects，
    每项含 identifier(仓库名)/language({id,name})/description 等。
    """
    data = gd.run_data(["repo", "+list", "--user", login,
                        "--category", "all", "--limit", str(limit)])
    return gd.first_list(data, ("projects", "repos"))


# ---------------------------------------------------------------------------
# 提交历史（Raw API：shortcuts 未提供 repo +commits）
# ---------------------------------------------------------------------------

def commits(owner: str, repo: str, ref: str = "master", max_pages: int = 5,
            page_size: int = 100) -> list:
    """通过 `api GET /{owner}/{repo}/commits` 分页取提交。

    注：该端点确切路径/分页参数需在 Phase 0c 用真实仓库核验；若不通，
    退化方案见 doc 注释（按分支取或用 compare 端点）。
    """
    out: list = []
    for page in range(1, max_pages + 1):
        data = gd.api("GET", f"/{owner}/{repo}/commits",
                      query=f"sha={ref}&page={page}&limit={page_size}",
                      owner=owner, repo=repo)
        items = gd.first_list(data, ("commits", "list"))
        if not items:
            break
        out.extend(items)
        if len(items) < page_size:
            break
    return out


def file_text(owner: str, repo: str, path: str, ref: str = "master") -> str:
    """读取仓库内某文件的文本内容（LICENSE / CI 配置 / lockfile 等）。

    `file +get`（sub_entries）返回形如 {"entries": {"content": "...", "commit": {...}}}。
    """
    data = gd.run_data(["file", "+get", "--path", path, "--ref", ref],
                       owner=owner, repo=repo)
    if isinstance(data, str):
        return data
    if isinstance(data, dict):
        # 顶层直接带内容
        for k in ("content", "text", "data"):
            v = data.get(k)
            if isinstance(v, str) and v:
                return v
        entries = data.get("entries")
        # 形式一：entries 是 dict，内含 content 键（GitLink 实测）
        if isinstance(entries, dict):
            v = entries.get("content") or entries.get("text")
            if isinstance(v, str) and v:
                return v
        # 形式二：entries 是 list[dict]
        elif isinstance(entries, list) and entries and isinstance(entries[0], dict):
            return as_str(entries[0].get("content") or entries[0].get("text"))
    return ""
