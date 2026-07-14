"""test_repro.py — repro.py 的纯单元测试。

不联网、不调 gitlink-cli。把"取数"和"算法"分离：直接给算法函数喂 mock 数据。

运行： python test_repro.py
"""
from __future__ import annotations

import os
import sys

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
import repro  # noqa: E402

# 用一个轻量测试框架，避免依赖 unittest（保持无第三方依赖）。
_failures: list[str] = []


def check(cond: bool, msg: str) -> None:
    status = "PASS" if cond else "FAIL"
    print(f"  [{status}] {msg}")
    if not cond:
        _failures.append(msg)


def check_eq(actual, expected, msg: str) -> None:
    ok = actual == expected
    status = "PASS" if ok else "FAIL"
    print(f"  [{status}] {msg}  (got={actual!r})")
    if not ok:
        _failures.append(f"{msg}: expected {expected!r}, got {actual!r}")


# ---------------------------------------------------------------------------
# 1) identify_license
# ---------------------------------------------------------------------------

def test_identify_license():
    print("\n== test_identify_license ==")
    cases = {
        "Apache License\nVersion 2.0": "Apache-2.0",
        "MIT License\n\nCopyright (c) 2024": "MIT",
        "GNU GENERAL PUBLIC LICENSE\nVersion 3": "GPL",
        "木兰宽松许可证, 第2版": "MulanPSL-2.0",
        "MulanPSL v2": "MulanPSL-2.0",
        "BSD 3-Clause License": "BSD",
        "ISC License": "ISC",
        "Mozilla Public License Version 2.0": "MPL",
        "random text without any license keyword": "None",
        "": "None",
        "   \n  ": "None",
    }
    for text, expected in cases.items():
        info = repro.identify_license(text)
        check_eq(info["license"], expected, f"identify_license({text[:24]!r})")
    # 识别标志
    check(repro.identify_license("MIT License")["recognized"] is True, "MIT recognized=True")
    check(repro.identify_license("nope")["recognized"] is False, "unknown recognized=False")
    # GPL 优先于 LGPL：LGPL 文本应命中 LGPL（因为 LGPL 模式在 GPL 之前）
    l = repro.identify_license("GNU Lesser General Public License v3")
    check_eq(l["license"], "LGPL", "LGPL not misidentified as GPL")


# ---------------------------------------------------------------------------
# 2) scan_secrets
# ---------------------------------------------------------------------------

def test_scan_secrets():
    print("\n== test_scan_secrets ==")
    sample = """\
API_KEY=sk_live_abcdef1234567890abcd
AWS_KEY=AKIAIOSFODNN7EXAMPLE
mail: someone@example.com
phone: 13812345678
-----BEGIN RSA PRIVATE KEY-----
MIIEpAIBAAKCAQEA...
"""
    findings = repro.scan_secrets(sample, file="config.env")
    cats = {f["category"] for f in findings}
    check("generic_api_key" in cats, "detect generic_api_key")
    check("aws_access_key" in cats, "detect aws_access_key")
    check("email" in cats, "detect email")
    check("phone_cn" in cats, "detect phone_cn")
    check("private_key" in cats, "detect private_key")
    # 每条都有必要字段
    for f in findings:
        check(set(["level", "category", "file", "line", "detail"]).issubset(f.keys()),
              f"finding {f['category']} has required keys")
        check_eq(f["file"], "config.env", f"{f['category']} file field")
    # critical 级别存在
    levels = {f["level"] for f in findings}
    check("critical" in levels, "critical level present (private key/aws)")
    # 空文本
    check_eq(repro.scan_secrets(""), [], "empty text -> no findings")
    # 无敏感信息文本
    check_eq(repro.scan_secrets("hello world\nnothing here"), [], "clean text -> no findings")
    # 长串脱敏（detail 不超长）
    long_token = "api_key=" + "a" * 60
    f2 = repro.scan_secrets(long_token, file="x")
    if f2:
        check(len(f2[0]["detail"]) <= 50, "long secret is masked in detail")


# ---------------------------------------------------------------------------
# 3) repro_checks
# ---------------------------------------------------------------------------

def test_repro_checks():
    print("\n== test_repro_checks ==")
    file_texts = {
        "README.md": "# Project\n\n## Install\n pip install -e .\n## Build\ndocker build .\n"
                     "Dataset: download from xxx. reproduce: python run.py\nUsage: see docs.",
    }
    tree = [
        {"path": ".gitea/workflows/ci.yml"},
        {"path": "go.sum"},
        {"path": "requirements.txt"},
        {"path": "Dockerfile"},
        {"path": "src/main.go"},
    ]
    items = repro.repro_checks(file_texts, tree, repo_info={"default_branch": "master"})
    names = {it["name"]: it for it in items}
    check(names["CI 配置"]["pass"] is True, "CI detected")
    check_eq(names["CI 配置"]["score"], 2, "CI score=2")
    check(names["依赖锁文件"]["pass"] is True, "lockfile detected")
    check(names["README 复现说明"]["pass"] is True, "README repro keywords detected")
    check(names["README 复现说明"]["score"] >= 1, "README repro score>=1")
    check(names["容器化环境"]["pass"] is True, "Dockerfile detected")

    # 无 CI / 无 lockfile 场景
    items2 = repro.repro_checks({}, [], repo_info={})
    names2 = {it["name"]: it for it in items2}
    check(names2["CI 配置"]["pass"] is False, "no CI -> fail")
    check_eq(names2["CI 配置"]["score"], 0, "no CI score=0")
    check(names2["依赖锁文件"]["pass"] is False, "no lockfile -> fail")
    check(names2["README 复现说明"]["pass"] is False, "no README -> fail")
    # 版本 tag：无 tag 字段 → score=1（不扣满分但提示）
    check_eq(names2["版本 tag"]["score"], 1, "unknown tag -> score=1")
    # 有 tag 字段 → score=2
    items3 = repro.repro_checks({}, [], repo_info={"version": "v1.2.3"})
    names3 = {it["name"]: it for it in items3}
    check_eq(names3["版本 tag"]["score"], 2, "version present -> score=2")
    check(names3["版本 tag"]["pass"] is True, "version present -> pass")

    # score 范围合法
    for it in items:
        check(0 <= it["score"] <= 2, f"{it['name']} score in [0,2]")


# ---------------------------------------------------------------------------
# 4) compliance_items
# ---------------------------------------------------------------------------

def test_compliance_items():
    print("\n== test_compliance_items ==")
    license_info = {"license": "MIT", "recognized": True, "evidence": "MIT License"}
    file_texts = {"LICENSE": "MIT License\nCopyright (c) 2024 Test", "README.md": "see LICENSE"}
    tree = [
        {"path": "SECURITY.md"},
        {"path": "CONTRIBUTING.md"},
        {"path": "requirements.txt"},
    ]
    items = repro.compliance_items(license_info, file_texts, tree)
    names = {it["name"]: it for it in items}
    check(names["LICENSE 文件"]["pass"] is True, "LICENSE recognized")
    check_eq(names["LICENSE 文件"]["score"], 2, "LICENSE score=2")
    check(names["安全策略 SECURITY.md"]["pass"] is True, "SECURITY.md present")
    check(names["版权声明"]["pass"] is True, "copyright present")
    check(names["贡献指南"]["pass"] is True, "CONTRIBUTING present")
    check(names["依赖清单声明"]["pass"] is True, "dep manifest present")

    # 缺失场景
    license_none = {"license": "None", "recognized": False, "evidence": "missing"}
    items2 = repro.compliance_items(license_none, {}, [])
    names2 = {it["name"]: it for it in items2}
    check(names2["LICENSE 文件"]["pass"] is False, "no LICENSE -> fail")
    check_eq(names2["LICENSE 文件"]["score"], 0, "no LICENSE score=0")
    check(names2["安全策略 SECURITY.md"]["pass"] is False, "no SECURITY -> fail")


# ---------------------------------------------------------------------------
# 5) data_privacy
# ---------------------------------------------------------------------------

def test_data_privacy():
    print("\n== test_data_privacy ==")
    # 健康场景：无 data/，无 .env，.gitignore 忽略 .env
    tree = [{"path": "src/main.py"}, {"path": ".gitignore"}]
    dp = repro.data_privacy(tree, ".env\n*.key\nnode_modules/")
    items = {it["name"]: it for it in dp["items"]}
    check(items["数据目录入库"]["pass"] is True, "no data dir -> pass")
    check(items[".env 入库"]["pass"] is True, "no .env -> pass")
    check(items[".gitignore 忽略 .env"]["pass"] is True, "gitignore ignores .env")
    check_eq(len(dp["risks"]), 0, "healthy repo -> 0 risks")

    # 风险场景：data/ 入库，.env 入库，gitignore 未忽略 .env
    tree2 = [{"path": "data/raw.csv"}, {"path": ".env"}, {"path": "config/secrets.yml"}]
    dp2 = repro.data_privacy(tree2, "node_modules/\n*.log")
    items2 = {it["name"]: it for it in dp2["items"]}
    check(items2["数据目录入库"]["pass"] is False, "data dir tracked -> fail")
    check(items2[".env 入库"]["pass"] is False, ".env tracked -> fail")
    check(items2[".gitignore 忽略 .env"]["pass"] is False, "gitignore missing .env")
    check(len(dp2["risks"]) >= 1, "risky repo -> has risks")


# ---------------------------------------------------------------------------
# 6) 打分 _score_10
# ---------------------------------------------------------------------------

def test_score():
    print("\n== test_score ==")
    # 5 项全 2 分 → 10
    full = [{"score": 2}] * 5
    check_eq(repro._score_10(full), 10.0, "all pass -> 10")
    # 5 项全 0 → 0
    zero = [{"score": 0}] * 5
    check_eq(repro._score_10(zero), 0.0, "all fail -> 0")
    # 混合：5 项中 3×2 + 2×0 = 6/10 = 6.0
    mixed = [{"score": 2}, {"score": 2}, {"score": 2}, {"score": 0}, {"score": 0}]
    check_eq(repro._score_10(mixed), 6.0, "mixed 6/10 -> 6.0")
    # 空列表
    check_eq(repro._score_10([]), 0.0, "empty -> 0")
    # cap 不超 10
    over = [{"score": 2}] * 8
    check(repro._score_10(over) <= 10.0, "capped at 10")


# ---------------------------------------------------------------------------
# 7) render_report 端到端（用 mock 结果）
# ---------------------------------------------------------------------------

def test_render_report():
    print("\n== test_render_report ==")
    mock = {
        "scenario": "S3_compliance_reproducibility",
        "repo": "o/r",
        "default_branch": "master",
        "license": "MIT",
        "repro_items": [{"name": "CI 配置", "pass": True, "score": 2, "evidence": "ci"}],
        "compliance_items": [{"name": "LICENSE 文件", "pass": True, "score": 2, "evidence": "MIT"}],
        "privacy_items": [{"name": ".env 入库", "pass": True, "score": 2, "evidence": "ok"}],
        "secrets": [],
        "risks": [],
        "repro_score": 10.0,
        "compliance_score": 10.0,
        "meta": {"tree_size": 5},
    }
    md = repro.render_report(mock)
    check("# 科研项目合规与复现性检查报告" in md, "report has title")
    check("MIT" in md, "report shows license")
    check("10.0/10" in md, "report shows scores")
    check("复现性检查清单" in md, "report has repro checklist")
    check("风险项" in md, "report has risk section")


# ---------------------------------------------------------------------------

def main():
    test_identify_license()
    test_scan_secrets()
    test_repro_checks()
    test_compliance_items()
    test_data_privacy()
    test_score()
    test_render_report()

    print()
    if _failures:
        print(f"RESULT: FAIL ({len(_failures)} failures)")
        for m in _failures:
            print(f"  - {m}")
        sys.exit(1)
    else:
        print("RESULT: ALL PASS")


if __name__ == "__main__":
    main()
