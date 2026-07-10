#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
gitlink-cli 演示网页 · 后端（零依赖，仅 Python 标准库）
作用：接收前端命令 + 访客 token，真跑 gitlink-cli，返回真实输出。
启动：python server.py   →   http://0.0.0.0:$PORT （默认 8000）

位置：仓库内 demo/web/server.py。CLI 在仓库根 ../../gitlink-cli[.exe]。
安全：本地/演示用。已做白名单（只允许 gitlink-cli 子命令）、subprocess 列表参数（不经 shell）、
      30s 超时。访客 token 仅在请求内存中传给子进程，不写日志、不落盘。
"""
import http.server
import json
import os
import re
import socketserver
import subprocess
import sys
from pathlib import Path
from urllib.parse import urlparse, parse_qs

# Windows GBK 终端兼容
try:
    sys.stdout.reconfigure(encoding="utf-8", errors="replace")
    sys.stderr.reconfigure(encoding="utf-8", errors="replace")
except Exception:
    pass

PORT = int(os.getenv("PORT", "8000"))
HOST = os.getenv("HOST", "0.0.0.0")  # 云端需 0.0.0.0；只听本机设 HOST=127.0.0.1
ROOT = Path(__file__).resolve().parent           # .../demo/web
REPO_ROOT = ROOT.parent.parent                    # 仓库根 .../gitlink-cli


def _find_cli():
    """CLI 位置：GITLINK_BIN > 仓库内 > PATH。Linux=gitlink-cli，Windows=gitlink-cli.exe。"""
    env = os.getenv("GITLINK_BIN")
    if env and Path(env).exists():
        return Path(env)
    for name in ("gitlink-cli", "gitlink-cli.exe"):
        p = REPO_ROOT / name
        if p.exists():
            return p
    for d in os.getenv("PATH", "").split(os.pathsep):
        p = Path(d) / "gitlink-cli"
        if p.exists():
            return p
    return REPO_ROOT / "gitlink-cli"  # 占位


CLI = _find_cli()
CWD = str(REPO_ROOT)  # 让 --owner/--repo 可从 git remote 自动解析

ALLOWED_DOMAINS = {  # 全部 30 个顶层域
    "api", "auth", "branch", "ci", "compare", "config", "dataset", "doctor",
    "file", "health", "ignore", "issue", "label", "license", "member",
    "milestone", "org", "pipeline", "pm", "pr", "profile", "release", "repo",
    "search", "snippet", "user", "version", "webhook", "wiki", "workflow",
}
NEEDS_TOKEN_DOMAINS = {  # 平台域（访客需填 token）
    "repo", "issue", "pr", "branch", "release", "search", "label", "member",
    "milestone", "webhook", "wiki", "org", "user", "ci", "compare", "dataset",
    "health", "license", "pipeline", "pm", "profile", "workflow", "api",
}
LOCAL_DOMAINS = {"snippet", "auth", "config", "version", "doctor"}


def _run_cli(args, token="", timeout=30):
    env = dict(os.environ)
    if token:
        env["GITLINK_TOKEN"] = token
    r = subprocess.run(
        [str(CLI)] + args, capture_output=True, text=True, timeout=timeout,
        cwd=CWD, encoding="utf-8", errors="replace", env=env,
    )
    return r.stdout, r.stderr, r.returncode


def _parse_json(stdout):
    s = stdout.strip()
    i, j = s.find("{"), s.rfind("}")
    if i < 0 or j < 0:
        return None
    try:
        return json.loads(s[i:j + 1])
    except Exception:
        return None


def _license_name(text):
    t = (text or "").lower()
    if "mulan" in t: return "Mulan PSL v2"
    if t.startswith("mit") or "mit license" in t: return "MIT"
    if "apache" in t: return "Apache 2.0"
    if "gpl" in t: return "GPL"
    if "bsd" in t: return "BSD"
    return "有 LICENSE" if text else "无"


def analyze_repo(owner, repo, token):
    """采集 + 按 research-insight 评分表算四维 + 巴士因子。"""
    def cli(*a):
        return _run_cli(list(a), token=token, timeout=30)

    info_o, _, _ = cli("repo", "+info", "--owner", owner, "--repo", repo, "--format", "json")
    info = (_parse_json(info_o) or {}).get("data") or {}

    contrib_o, _, _ = cli("repo", "+contributors", "--owner", owner, "--repo", repo, "--format", "json")
    contribs = ((_parse_json(contrib_o) or {}).get("data") or {}).get("list") or []
    contribs_sorted = sorted(contribs, key=lambda c: -(c.get("contributions") or 0))

    rel_o, _, _ = cli("release", "+list", "--owner", owner, "--repo", repo, "--format", "json")
    releases = ((_parse_json(rel_o) or {}).get("data") or {}).get("releases") or []

    lic_o, _, _ = cli("file", "+get", "--owner", owner, "--repo", repo, "--path", "LICENSE", "--format", "json")
    lic_text = ""
    lic_parsed = _parse_json(lic_o)
    if lic_parsed:
        d = lic_parsed.get("data") or {}
        entries = d.get("entries") if isinstance(d, dict) else None
        if isinstance(entries, dict):
            lic_text = entries.get("content") or ""
        elif isinstance(d, str):
            lic_text = d

    ci_o, _, _ = cli("repo", "+tree", "--owner", owner, "--repo", repo, "--path", ".gitea/workflows", "--format", "json")
    ci_entries = ((_parse_json(ci_o) or {}).get("data") or {}).get("entries") or []
    has_ci = bool(ci_entries)

    tree_o, _, _ = cli("repo", "+tree", "--owner", owner, "--repo", repo, "--format", "json")
    root_files = [str(e.get("name", "")) for e in ((_parse_json(tree_o) or {}).get("data") or {}).get("entries") or []]
    lock_files = {"go.sum", "package-lock.json", "yarn.lock", "Cargo.lock", "requirements.txt", "poetry.lock", "pom.xml"}
    has_lock = any(f in lock_files for f in root_files)
    has_readme = any(f.lower().startswith("readme") for f in root_files)

    # 可复现性（工程类，满分 8：数据项 N/A）
    repro, repro_detail = 0, []
    repro += 2 if has_ci else 0; repro_detail.append(("CI 配置", has_ci))
    repro += 2 if has_lock else 0; repro_detail.append(("依赖锁定", has_lock))
    repro += 2 if has_readme else 0; repro_detail.append(("运行文档", has_readme))
    ver = bool(releases or info.get("version_releases_count"))
    repro += 2 if ver else 0; repro_detail.append(("版本归档", ver))

    n_contrib = len(contribs)
    activity = min(10, round(n_contrib / 3)) if n_contrib else 2  # 无 commits API，用贡献者规模近似

    citation = 0
    citation += 3 if lic_text else 0
    citation += 3 if ver else 0
    citation += 2 if has_readme else 0
    citation += 2 if (info.get("fork_info") or {}).get("fork_project_user_login") else 0
    citation = min(10, citation)

    top_perc = 0.0
    if contribs_sorted:
        try:
            top_perc = float(re.sub(r"[^\d.]", "", str(contribs_sorted[0].get("contribution_perc", "0"))))
        except Exception:
            top_perc = 0.0
    collab = 10 if top_perc < 33 else (6 if top_perc < 50 else 3)

    fork_from = (info.get("fork_info") or {}).get("fork_project_user_login")
    return {
        "ok": True, "owner": owner, "repo": repo,
        "is_fork": bool(fork_from), "fork_from": fork_from,
        "name": info.get("name", repo),
        "license": _license_name(lic_text),
        "contributor_count": n_contrib,
        "release_count": len(releases),
        "version_releases_count": info.get("version_releases_count", 0),
        "contributors": [
            {"name": c.get("name") or c.get("login") or "?",
             "contributions": c.get("contributions", 0),
             "perc": c.get("contribution_perc", "")}
            for c in contribs_sorted[:8]
        ],
        "scores": {"repro": repro, "activity": activity, "citation": citation, "collab": collab},
        "repro_max": 8, "repro_detail": repro_detail,
        "bus_factor": top_perc,
        "bus_risk": "低" if top_perc < 33 else ("中" if top_perc < 50 else "高"),
    }


class Handler(http.server.BaseHTTPRequestHandler):
    def _cors(self):
        self.send_header("Access-Control-Allow-Origin", "*")
        self.send_header("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
        self.send_header("Access-Control-Allow-Headers", "Content-Type")

    def do_OPTIONS(self):
        self.send_response(204); self._cors(); self.end_headers()

    def do_GET(self):
        path = urlparse(self.path).path
        if path in ("/", "/index.html"):
            self._serve_file("index.html", "text/html")
        elif path == "/api/cli":
            self._json({"ok": True, "cli": str(CLI), "exists": CLI.exists()})
        elif path == "/api/domains":
            self._json({"ok": True, "domains": sorted(ALLOWED_DOMAINS), "local": sorted(LOCAL_DOMAINS)})
        elif path == "/api/health":
            self._json({"ok": True, "cli_exists": CLI.exists()})
        elif path == "/api/skill":
            self._handle_skill()
        else:
            self.send_error(404)

    def _handle_skill(self):
        q = parse_qs(urlparse(self.path).query)
        name = (q.get("name") or [""])[0].strip()
        if not name:
            self._json({"ok": False, "error": "缺少 ?name="}); return
        skill_md = REPO_ROOT / "skills" / f"gitlink-{name}" / "SKILL.md"
        if not skill_md.exists():
            self._json({"ok": False, "error": f"找不到 SKILL.md：gitlink-{name}"}); return
        self._json({"ok": True, "name": name, "content": skill_md.read_text(encoding="utf-8")})

    def do_POST(self):
        path = urlparse(self.path).path
        body = self._read_body()
        if path == "/api/run":
            self._handle_run(body)
        elif path == "/api/analyze":
            owner = (body.get("owner") or "").strip()
            repo = (body.get("repo") or "").strip()
            token = (body.get("token") or "").strip()
            if not owner or not repo:
                self._json({"ok": False, "error": "缺少 owner/repo"}); return
            try:
                self._json(analyze_repo(owner, repo, token))
            except subprocess.TimeoutExpired:
                self._json({"ok": False, "error": "采集超时（>30s）"})
            except Exception as e:
                self._json({"ok": False, "error": str(e)})
        else:
            self.send_error(404)

    def _handle_run(self, body):
        cmd = (body.get("cmd") or "").strip()
        token = (body.get("token") or "").strip()
        if not cmd:
            self._json({"ok": False, "error": "空命令"}); return
        args = cmd.split()
        while args and args[0] in ("gitlink-cli", "gitlink-cli.exe", "./gitlink-cli.exe"):
            args = args[1:]
        if not args:
            self._json({"ok": False, "error": "缺少子命令"}); return
        domain = args[0]
        if domain not in ALLOWED_DOMAINS:
            self._json({"ok": False, "error": f"不允许的命令：{domain}（仅限 gitlink-cli 子命令）"}); return
        needs_token = domain in NEEDS_TOKEN_DOMAINS
        try:
            out, err, code = _run_cli(args, token=token, timeout=30)
            # 失败时从 stderr 取首行作为 error，前端绝不再显示 undefined
            err_msg = None
            if code != 0:
                first = (err.strip() or out.strip()).splitlines()
                err_msg = first[0][:200] if first else f"命令失败（退出码 {code}）"
            self._json({
                "ok": code == 0, "cmd": f"gitlink-cli {' '.join(args)}",
                "stdout": out, "stderr": err, "code": code, "error": err_msg,
                "needs_token": needs_token, "token_provided": bool(token),
            })
        except subprocess.TimeoutExpired:
            self._json({"ok": False, "error": "命令超时（>30s），可能涉及交互输入"})
        except Exception as e:
            self._json({"ok": False, "error": str(e)})

    def _read_body(self):
        length = int(self.headers.get("Content-Length", 0) or 0)
        raw = self.rfile.read(length) if length else b"{}"
        try:
            return json.loads(raw)
        except Exception:
            return {}

    def _json(self, obj):
        data = json.dumps(obj, ensure_ascii=False).encode("utf-8")
        self.send_response(200)
        self.send_header("Content-Type", "application/json; charset=utf-8")
        self._cors()
        self.send_header("Content-Length", str(len(data)))
        self.end_headers()
        self.wfile.write(data)

    def _serve_file(self, name, mime):
        p = ROOT / name
        if not p.exists():
            self.send_error(404, f"{name} 不存在"); return
        data = p.read_bytes()
        self.send_response(200)
        self.send_header("Content-Type", f"{mime}; charset=utf-8")
        self._cors()
        self.send_header("Content-Length", str(len(data)))
        self.end_headers()
        self.wfile.write(data)

    def log_message(self, *a):
        pass


class ReuseTCPServer(socketserver.ThreadingMixIn, socketserver.TCPServer):
    allow_reuse_address = True
    daemon_threads = True  # 每个请求独立线程，单个卡死不阻塞其他请求


if __name__ == "__main__":
    if not CLI.exists():
        print(f"[!] 找不到 gitlink-cli 二进制：{CLI}")
        print("    请先编译：cd <仓库根> && go build -o gitlink-cli . （Linux）")
        print("    或设环境变量 GITLINK_BIN 指向已有二进制。")
    with ReuseTCPServer((HOST, PORT), Handler) as httpd:
        print(f"[OK] gitlink-cli 演示后端已启动：http://{HOST}:{PORT}")
        print(f"     CLI：{CLI}（exists={CLI.exists()}）")
        print(f"     访客在网页顶栏填自己的 GitLink token 即可跑平台命令。Ctrl+C 停止。")
        try:
            httpd.serve_forever()
        except KeyboardInterrupt:
            print("\n已停止")
