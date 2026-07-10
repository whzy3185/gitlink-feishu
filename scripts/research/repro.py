"""repro.py — S3 科研项目合规与复现性检查。

输入一个科研仓库，检查其「合规性」（许可证、版权、依赖、安全策略、数据隐私）
与「可复现性」（CI 配置、lockfile、README 是否含数据集/环境/构建说明、版本 tag、
密钥泄露），分别给出 0-10 的复现分与合规分，并产出检查清单、风险项与中文报告。

数据全部经 gitlink-cli 获取：
  - repo +info（仓库信息、版本 tag）
  - file +get（LICENSE / README / go.mod / requirements.txt / package.json / .gitignore 等）
  - repo +tree（扫 data/、.env、config、.gitea/.github workflows 等是否存在）

用法：
  python repro.py --owner mindspore-Ecosystem --repo mindspore --out ./out
  python repro.py --owner O --repo R                 # 仅打印 JSON
"""
from __future__ import annotations

import argparse
import json
import os
import re
import sys
from typing import Any

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
import collect as c  # noqa: E402
import gitlink_data as gd  # noqa: E402

# ---------------------------------------------------------------------------
# 常量
# ---------------------------------------------------------------------------

# 复现性/合规相关的关键文件（相对仓库根）
KEY_FILES: tuple[str, ...] = (
    "LICENSE", "LICENSE.txt", "LICENSE.md",
    "README", "README.md", "README.rst",
    "go.mod", "requirements.txt", "package.json", "Cargo.toml",
    ".gitignore", "SECURITY.md", "CONTRIBUTING.md",
)

# CI 配置目录/文件（复现性信号）。
# 注意：根 tree 通常只列顶层目录（.github/.devops/.gitea），不一定展开到 workflows/，
# 故同时收录顶层目录名与深层路径；.devops 是 GitLink 专属 CI/CD 目录。
CI_PATHS: tuple[str, ...] = (
    ".devops",
    ".gitea", ".gitea/workflows",
    ".github", ".github/workflows",
    ".gitlab-ci.yml", ".circleci", ".travis.yml", "azure-pipelines.yml",
)

# 锁文件（复现性信号：依赖版本固定）
LOCKFILES: tuple[str, ...] = (
    "go.sum", "package-lock.json", "yarn.lock", "pnpm-lock.yaml",
    "requirements.txt", "poetry.lock", "Pipfile.lock", "Cargo.lock", "composer.lock",
)

# 敏感目录/文件（数据隐私信号）
SENSITIVE_PATHS: tuple[str, ...] = (
    "data/", "data", "dataset/", "datasets/", ".env", ".env.local",
    "config/secrets", "secrets",
)

# README 复现性关键词（数据集 / 环境 / 构建 / 运行说明）
README_REPRO_KEYWORDS: tuple[str, ...] = (
    "install", "setup", "环境", "依赖", "build", "构建", "运行", "run",
    "dataset", "数据集", "docker", "conda", "pip install", "npm install",
    "requirements", "reproduce", "复现", "环境配置", "usage", "用法",
)

# 许可证识别关键词（顺序即优先级）
LICENSE_PATTERNS: tuple[tuple[str, str], ...] = (
    ("MulanPSL", "MulanPSL-2.0"),
    ("木兰宽松许可证", "MulanPSL-2.0"),
    ("Apache License", "Apache-2.0"),
    ("MIT License", "MIT"),
    ("GNU GENERAL PUBLIC LICENSE", "GPL"),
    ("GNU Lesser General Public License", "LGPL"),
    ("BSD ", "BSD"),
    ("ISC License", "ISC"),
    ("Mozilla Public License", "MPL"),
    ("Unlicense", "Unlicense"),
)

# 密钥/敏感信息正则（按类别）
SECRET_PATTERNS: tuple[tuple[str, str, str], ...] = (
    # (category, level, regex)
    ("private_key", "critical", r"-----BEGIN (?:RSA |EC |DSA |OPENSSH |PGP )?PRIVATE KEY-----"),
    ("aws_access_key", "critical", r"AKIA[0-9A-Z]{16}"),
    ("aws_secret", "critical", r"aws_secret_access_key\s*[:=]\s*['\"]?[A-Za-z0-9/+=]{40}"),
    ("generic_api_key", "high", r"(?i)api[_-]?key\s*[:=]\s*['\"]?[A-Za-z0-9_\-]{16,}"),
    ("google_api_key", "high", r"AIza[0-9A-Za-z_\-]{35}"),
    ("slack_token", "high", r"xox[baprs]-[0-9A-Za-z-]{10,}"),
    ("github_token", "high", r"gh[pousr]_[A-Za-z0-9]{36,}"),
    ("jwt", "medium", r"eyJ[A-Za-z0-9_\-]+\.eyJ[A-Za-z0-9_\-]+\.[A-Za-z0-9_\-]+"),
    ("email", "low", r"[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[A-Za-z]{2,}"),
    # 中国大陆手机号
    ("phone_cn", "low", r"(?<!\d)1[3-9]\d{9}(?!\d)"),
)


# ---------------------------------------------------------------------------
# 算法：纯函数（单测对象，不联网）
# ---------------------------------------------------------------------------

def identify_license(text: str) -> dict[str, Any]:
    """关键词匹配 LICENSE 文本，返回许可证信息。

    返回: {"license": str, "recognized": bool, "evidence": str}
    未识别返回 license="None"。
    """
    if not text or not text.strip():
        return {"license": "None", "recognized": False,
                "evidence": "LICENSE 文件为空或缺失"}
    low = text.lower()
    for pat, name in LICENSE_PATTERNS:
        if pat.lower() in low:
            # 找到关键词所在行作为证据
            idx = low.find(pat.lower())
            line_start = text.rfind("\n", 0, idx) + 1
            line_end = text.find("\n", idx)
            if line_end == -1:
                line_end = len(text)
            evidence = text[line_start:line_end].strip()[:120]
            return {"license": name, "recognized": True, "evidence": evidence}
    return {"license": "None", "recognized": False,
            "evidence": "未匹配到已知许可证关键词"}


def scan_secrets(text: str, file: str = "") -> list[dict[str, Any]]:
    """扫描文本中的密钥/敏感信息。

    返回: [{"level","category","file","line","detail"}, ...]
    """
    if not text:
        return []
    findings: list[dict[str, Any]] = []
    lines = text.splitlines()
    for category, level, pattern in SECRET_PATTERNS:
        for m in re.finditer(pattern, text):
            # 计算行号与所在行内容
            line_no = text.count("\n", 0, m.start()) + 1
            line_content = lines[line_no - 1] if 0 <= line_no - 1 < len(lines) else ""
            detail = m.group(0)
            # 脱敏：长串截断
            if len(detail) > 40:
                detail = detail[:20] + "..." + detail[-6:]
            findings.append({
                "level": level, "category": category, "file": file,
                "line": line_no, "detail": detail,
                "context": line_content.strip()[:80],
            })
    return findings


def _tree_paths(tree: list) -> list[str]:
    """从 tree 列表提取所有路径字符串（兼容多种字段名）。"""
    paths: list[str] = []
    for item in tree:
        if isinstance(item, dict):
            for k in ("path", "name", "filepath"):
                v = item.get(k)
                if isinstance(v, str) and v:
                    paths.append(v)
                    break
        elif isinstance(item, str) and item:
            paths.append(item)
    return paths


def _has_path(paths: list[str], targets: tuple[str, ...]) -> list[str]:
    """paths 中命中任一 target（前缀/精确都算）的命中项，返回命中原文（去重保序）。"""
    hits: list[str] = []
    seen: set[str] = set()
    low_targets = [t.lower().rstrip("/") for t in targets]
    for p in paths:
        pl = p.lower().rstrip("/")
        for tl in low_targets:
            if pl == tl or pl.startswith(tl + "/"):
                key = f"{p}::{tl}"
                if key not in seen:
                    seen.add(key)
                    hits.append(p)
                break
    return hits


def repro_checks(file_texts: dict[str, str], tree: list,
                 repo_info: dict | None = None) -> list[dict[str, Any]]:
    """复现性检查。

    file_texts: {路径: 文本内容}（已取好）
    tree: tree 列表（已取好）
    repo_info: 仓库信息（取版本 tag；无 tag 字段则 unknown）

    返回检查项列表: [{"name","pass","score","evidence"}]
    每项 score 0-2（0=缺失/失败, 1=部分, 2=完备）。
    """
    paths = _tree_paths(tree)
    items: list[dict[str, Any]] = []

    # 1) CI 配置
    ci_hits = _has_path(paths, CI_PATHS)
    if ci_hits:
        items.append({"name": "CI 配置", "pass": True, "score": 2,
                      "evidence": f"检测到 CI 配置: {', '.join(ci_hits[:3])}"})
    else:
        items.append({"name": "CI 配置", "pass": False, "score": 0,
                      "evidence": "未找到 .gitea/.github/.gitlab 等 CI 配置"})

    # 2) lockfile（依赖版本固定）
    lock_hits = _has_path(paths, LOCKFILES)
    if lock_hits:
        items.append({"name": "依赖锁文件", "pass": True, "score": 2,
                      "evidence": f"存在 lockfile: {', '.join(lock_hits[:3])}"})
    else:
        items.append({"name": "依赖锁文件", "pass": False, "score": 0,
                      "evidence": "未找到 go.sum/package-lock.json/poetry.lock 等锁文件"})

    # 3) README 含数据集/环境/构建说明
    readme_text = ""
    for k in ("README.md", "README", "README.rst"):
        if k in file_texts and file_texts[k]:
            readme_text = file_texts[k]
            break
    if readme_text:
        low = readme_text.lower()
        hit_kw = [kw for kw in README_REPRO_KEYWORDS if kw.lower() in low]
        score = 2 if len(hit_kw) >= 4 else (1 if len(hit_kw) >= 1 else 0)
        items.append({"name": "README 复现说明", "pass": score > 0, "score": score,
                      "evidence": (f"README 含复现关键词 {len(hit_kw)} 个: {', '.join(hit_kw[:5])}"
                                   if hit_kw else "README 存在但缺少数据集/环境/构建说明")})
    else:
        items.append({"name": "README 复现说明", "pass": False, "score": 0,
                      "evidence": "未找到 README"})

    # 4) 版本 tag（用 repo_info，无 tag 字段则 unknown）
    info = repo_info or {}
    tag = (info.get("version") or info.get("tag") or info.get("release_tag")
           or info.get("default_branch") or "")
    # 多数 GitLink repo_info 无显式 tag 字段 → 标 unknown（不扣分但提示）
    has_tag = bool(info.get("version") or info.get("tag") or info.get("release_tag"))
    if has_tag:
        items.append({"name": "版本 tag", "pass": True, "score": 2,
                      "evidence": f"版本/release tag: {tag}"})
    else:
        items.append({"name": "版本 tag", "pass": False, "score": 1,
                      "evidence": f"repo_info 无显式 tag 字段（默认分支: {info.get('default_branch', 'unknown')}），建议打 tag 固定可复现版本"})

    # 5) 容器化（Dockerfile / docker-compose）—— 复现环境
    container_hits = _has_path(paths, ("Dockerfile", "docker-compose.yml",
                                       "docker-compose.yaml", ".devcontainer"))
    if container_hits:
        items.append({"name": "容器化环境", "pass": True, "score": 2,
                      "evidence": f"存在容器配置: {', '.join(container_hits[:3])}"})
    else:
        items.append({"name": "容器化环境", "pass": False, "score": 0,
                      "evidence": "未找到 Dockerfile/docker-compose，复现环境依赖手工描述"})

    return items


def compliance_items(license_info: dict[str, Any], file_texts: dict[str, str],
                     tree: list) -> list[dict[str, Any]]:
    """合规性检查。

    返回检查项列表: [{"name","pass","score","evidence"}]，score 0-2。
    """
    paths = _tree_paths(tree)
    items: list[dict[str, Any]] = []

    # 1) LICENSE 声明
    lic = license_info.get("license", "None")
    recognized = license_info.get("recognized", False)
    if recognized and lic != "None":
        items.append({"name": "LICENSE 文件", "pass": True, "score": 2,
                      "evidence": f"LICENSE 声明为 {lic}"})
    elif lic == "None" and not (file_texts.get("LICENSE") or file_texts.get("LICENSE.txt")
                                or file_texts.get("LICENSE.md")):
        items.append({"name": "LICENSE 文件", "pass": False, "score": 0,
                      "evidence": "缺少 LICENSE 文件"})
    else:
        items.append({"name": "LICENSE 文件", "pass": False, "score": 1,
                      "evidence": "LICENSE 文件存在但类型未识别"})

    # 2) SECURITY.md
    sec_hits = _has_path(paths, ("SECURITY.md", "SECURITY", "security.md"))
    if sec_hits:
        items.append({"name": "安全策略 SECURITY.md", "pass": True, "score": 2,
                      "evidence": f"存在 {sec_hits[0]}"})
    else:
        items.append({"name": "安全策略 SECURITY.md", "pass": False, "score": 0,
                      "evidence": "缺少 SECURITY.md，无安全披露流程"})

    # 3) 版权头（采样 README/LICENSE 头部判断有无 Copyright）
    sample = (file_texts.get("LICENSE", "") + "\n" + file_texts.get("README.md", "")
              + "\n" + file_texts.get("README", ""))
    has_copyright = ("copyright" in sample.lower()) or ("版权" in sample) or ("©" in sample)
    if has_copyright:
        items.append({"name": "版权声明", "pass": True, "score": 2,
                      "evidence": "LICENSE/README 中含 copyright/版权 声明"})
    else:
        items.append({"name": "版权声明", "pass": False, "score": 1,
                      "evidence": "未在 LICENSE/README 中发现版权声明（建议源文件头补 Copyright 注释）"})

    # 4) 依赖合规（存在依赖清单即视为已声明，识别许可证更佳）
    dep_present = bool(_has_path(paths, ("go.mod", "requirements.txt", "package.json",
                                          "Cargo.toml", "pom.xml", "setup.py", "pyproject.toml")))
    if dep_present:
        items.append({"name": "依赖清单声明", "pass": True, "score": 2,
                      "evidence": "存在依赖管理文件（建议核对各依赖许可证兼容性）"})
    else:
        items.append({"name": "依赖清单声明", "pass": False, "score": 1,
                      "evidence": "未发现标准依赖管理文件"})

    # 5) CONTRIBUTING.md（社区合规）
    contrib_hits = _has_path(paths, ("CONTRIBUTING.md", "CONTRIBUTING", "contributing.md"))
    if contrib_hits:
        items.append({"name": "贡献指南", "pass": True, "score": 2,
                      "evidence": f"存在 {contrib_hits[0]}"})
    else:
        items.append({"name": "贡献指南", "pass": False, "score": 1,
                      "evidence": "缺少 CONTRIBUTING.md"})

    return items


def data_privacy(tree: list, gitignore_text: str) -> dict[str, Any]:
    """数据隐私检查。

    返回: {
        "items": [{"name","pass","score","evidence"}],
        "risks": [...],   # 高风险项明细
    }
    """
    paths = _tree_paths(tree)
    items: list[dict[str, Any]] = []
    risks: list[str] = []

    # 1) data/ 目录是否入库
    data_hits = _has_path(paths, ("data/", "dataset/", "datasets/"))
    if data_hits:
        items.append({"name": "数据目录入库", "pass": False, "score": 0,
                      "evidence": f"data/ 目录已入库: {', '.join(data_hits[:3])}（建议大文件走外部存储/DVC）"})
        risks.append(f"数据目录入库: {', '.join(data_hits[:3])}（可能含敏感数据）")
    else:
        items.append({"name": "数据目录入库", "pass": True, "score": 2,
                      "evidence": "未发现 data/ 目录入库"})

    # 2) .env 是否入库
    env_hits = _has_path(paths, (".env", ".env.local", ".env.production"))
    if env_hits:
        items.append({"name": ".env 入库", "pass": False, "score": 0,
                      "evidence": f".env 已入库: {', '.join(env_hits[:3])}（高风险，疑似凭据泄露）"})
        risks.append(f".env 已入库: {', '.join(env_hits[:3])}（凭据泄露风险）")
    else:
        items.append({"name": ".env 入库", "pass": True, "score": 2,
                      "evidence": ".env 未入库"})

    # 3) .gitignore 是否忽略 .env
    gi = (gitignore_text or "").lower()
    ignores_env = ".env" in gi
    if ignores_env:
        items.append({"name": ".gitignore 忽略 .env", "pass": True, "score": 2,
                      "evidence": ".gitignore 已配置忽略 .env"})
    else:
        items.append({"name": ".gitignore 忽略 .env", "pass": False, "score": 1,
                      "evidence": ".gitignore 未忽略 .env（建议添加 .env）"})
        if not env_hits:
            risks.append(".gitignore 未忽略 .env（预防性建议）")

    return {"items": items, "risks": risks}


def _score_10(items: list[dict[str, Any]], cap: float = 10.0) -> float:
    """把检查项的 0-2 分聚合为 0-10 分：sum(score)/sum(max=2) * 10。"""
    total = sum(it.get("score", 0) for it in items)
    max_total = sum(2 for _ in items)
    if max_total == 0:
        return 0.0
    return round(min(cap, total / max_total * cap), 1)


# ---------------------------------------------------------------------------
# 数据采集（调 gitlink-cli；算法不依赖本节）
# ---------------------------------------------------------------------------

def collect_file_texts(owner: str, repo: str, ref: str = "master") -> dict[str, str]:
    """批量取关键文件文本。缺失文件返回空串（不出现在 dict 中）。"""
    out: dict[str, str] = {}
    for path in KEY_FILES:
        try:
            txt = c.file_text(owner, repo, path, ref=ref)
        except Exception:
            txt = ""
        if txt and txt.strip():
            out[path] = txt
    return out


def collect_tree(owner: str, repo: str, ref: str = "master") -> list:
    """取根 tree（含扫 data/、.env、config、workflows 等）。"""
    try:
        return c.tree(owner, repo, path="", ref=ref)
    except Exception:
        return []


# ---------------------------------------------------------------------------
# 主流程
# ---------------------------------------------------------------------------

def run(owner: str, repo: str) -> dict[str, Any]:
    info = c.repo_info(owner, repo)
    ref = info.get("default_branch") or "master"

    file_texts = collect_file_texts(owner, repo, ref)
    tree = collect_tree(owner, repo, ref)

    license_text = (file_texts.get("LICENSE") or file_texts.get("LICENSE.txt")
                    or file_texts.get("LICENSE.md") or "")
    license_info = identify_license(license_text)

    repro = repro_checks(file_texts, tree, repo_info=info)
    compliance = compliance_items(license_info, file_texts, tree)
    gitignore_text = file_texts.get(".gitignore", "")
    dp = data_privacy(tree, gitignore_text)

    # 汇总密钥扫描（扫所有已取文本 + gitignore）
    all_secrets: list[dict[str, Any]] = []
    for path, txt in file_texts.items():
        all_secrets.extend(scan_secrets(txt, file=path))

    repro_score = _score_10(repro)
    compliance_score = _score_10(compliance + dp["items"])

    # 风险项汇总
    risks: list[dict[str, Any]] = []
    for it in repro + compliance + dp["items"]:
        if not it["pass"]:
            risks.append({"area": "repro/compliance", "name": it["name"],
                          "evidence": it["evidence"], "level": "medium"})
    for r in dp["risks"]:
        risks.append({"area": "privacy", "name": "数据隐私", "evidence": r, "level": "high"})
    for s in all_secrets:
        risks.append({"area": "secret", "name": s["category"], "file": s["file"],
                      "line": s["line"], "detail": s["detail"],
                      "level": s["level"]})
    # 按级别排序
    level_rank = {"critical": 0, "high": 1, "medium": 2, "low": 3}
    risks.sort(key=lambda r: level_rank.get(r.get("level", "low"), 9))

    return {
        "scenario": "S3_compliance_reproducibility",
        "repo": f"{owner}/{repo}",
        "default_branch": ref,
        "license": license_info["license"],
        "repro_items": repro,
        "compliance_items": compliance,
        "privacy_items": dp["items"],
        "secrets": all_secrets,
        "risks": risks,
        "repro_score": repro_score,
        "compliance_score": compliance_score,
        "meta": {"key_files_found": sorted(file_texts.keys()),
                 "tree_size": len(tree),
                 "languages": c.languages(owner, repo)},
    }


# ---------------------------------------------------------------------------
# 渲染：Markdown 报告
# ---------------------------------------------------------------------------

def _grade(score: float) -> str:
    if score >= 8:
        return "良好"
    if score >= 6:
        return "及格"
    if score >= 4:
        return "偏弱"
    return "较差"


def render_report(result: dict[str, Any]) -> str:
    repo = result["repo"]
    lic = result["license"]
    rs = result["repro_score"]
    cs = result["compliance_score"]
    lines = [
        f"# 科研项目合规与复现性检查报告 — {repo}\n",
        f"> 场景 S3 · 子赛题四「应用 GitLink 辅助科研」\n",
        f"- **默认分支**: `{result.get('default_branch', 'master')}`",
        f"- **识别许可证**: `{lic}`",
        f"- **复现性评分**: **{rs}/10**（{_grade(rs)}）",
        f"- **合规性评分**: **{cs}/10**（{_grade(cs)}）\n",
    ]

    # 检查清单表
    lines += ["## 一、复现性检查清单\n",
              "| 检查项 | 通过 | 得分 | 证据 |",
              "|--------|:----:|:----:|------|"]
    for it in result["repro_items"]:
        mark = "PASS" if it["pass"] else "FAIL"
        lines.append(f"| {it['name']} | {mark} | {it['score']}/2 | {it['evidence']} |")

    lines += ["\n## 二、合规性检查清单\n",
              "| 检查项 | 通过 | 得分 | 证据 |",
              "|--------|:----:|:----:|------|"]
    for it in result["compliance_items"]:
        mark = "PASS" if it["pass"] else "FAIL"
        lines.append(f"| {it['name']} | {mark} | {it['score']}/2 | {it['evidence']} |")

    lines += ["\n## 三、数据隐私检查\n",
              "| 检查项 | 通过 | 得分 | 证据 |",
              "|--------|:----:|:----:|------|"]
    for it in result["privacy_items"]:
        mark = "PASS" if it["pass"] else "FAIL"
        lines.append(f"| {it['name']} | {mark} | {it['score']}/2 | {it['evidence']} |")

    # 风险项表
    risks = result["risks"]
    lines += ["\n## 四、风险项（按严重程度排序）\n",
              "| 级别 | 类别 | 名称 | 文件:行 | 证据 |",
              "|:----:|------|------|---------|------|"]
    if risks:
        for r in risks:
            lvl = r.get("level", "medium")
            area = r.get("area", "")
            name = r.get("name", "")
            file_loc = ""
            if r.get("file"):
                file_loc = f"{r['file']}:{r.get('line', '')}"
            ev = r.get("evidence") or r.get("detail", "")
            lines.append(f"| {lvl} | {area} | {name} | {file_loc} | {ev} |")
    else:
        lines.append("| — | — | 无风险项 | — | 全部检查通过 |")

    # 密钥小结
    secrets = result.get("secrets", [])
    if secrets:
        lines.append(f"\n> 检出 **{len(secrets)}** 处疑似敏感信息（见风险项表），请人工复核确认。\n")

    lines.append(f"\n_复现分 {rs}/10 · 合规分 {cs}/10 · 树节点 {result.get('meta',{}).get('tree_size','?')}_\n")
    return "\n".join(lines)


# ---------------------------------------------------------------------------

def main():
    ap = argparse.ArgumentParser(description="S3 科研项目合规与复现性检查")
    ap.add_argument("--owner", required=True)
    ap.add_argument("--repo", required=True)
    ap.add_argument("--out", help="输出目录（写 repro.json + compliance_report.md）；省略则打印 JSON")
    args = ap.parse_args()

    result = run(args.owner, args.repo)

    if args.out:
        os.makedirs(args.out, exist_ok=True)
        with open(os.path.join(args.out, "repro.json"), "w", encoding="utf-8") as f:
            json.dump(result, f, ensure_ascii=False, indent=2)
        with open(os.path.join(args.out, "compliance_report.md"), "w", encoding="utf-8") as f:
            f.write(render_report(result))
        print(f"✓ S3 合规/复现检查完成 → {args.out}/repro.json | compliance_report.md")
        print(f"  许可证: {result['license']}")
        print(f"  复现分: {result['repro_score']}/10  合规分: {result['compliance_score']}/10")
        print(f"  风险项: {len(result['risks'])} 处")
    else:
        print(json.dumps(result, ensure_ascii=False, indent=2))


if __name__ == "__main__":
    main()
