#!/usr/bin/env python3
"""可复现评分单测 —— 把「同输入 → 同分 → 同裁决」从口号变成可验证事实。

纯标准库 unittest（Python 3.9 兼容）。直接 import `scripts/gatekeeper_workflow.py`
的确定性算法（score_dimensions / evaluate_hard_gates / decide_verdict），对四个
权威裁决案例（skills/gitlink-gatekeeper/examples/decision-*.md 与
scorecard-sample.md）构造等价的 ScoreInput，断言**总分**与**三态裁决**与文档逐位一致。

任意一处算法改动若改变了这四个案例的分值，本测试立即变红——即为「确定性」的回归护栏。

运行：
    python3 workflow/tests/test_scoring.py
或：
    python3 -m unittest workflow.tests.test_scoring   # 在仓库根目录

数值来源（默认策略 gatekeeper.yaml，与脚本内置 DEFAULT_POLICY 一致）：
    权重 40/20/15/15/10；severity_penalty blocker=100/major=25/minor=5/nit=1；
    thresholds pass=85 / request_changes=60；max_changed_files=80。
"""

from __future__ import annotations

import importlib.util
import sys
import unittest
from pathlib import Path

# --------------------------------------------------------------------------- #
# 以绝对路径加载被测脚本（它在 scripts/ 下、非包，按文件直接载入最稳）
#
# 注意：必须先把模块塞进 sys.modules 再 exec —— 被测脚本用了
# `from __future__ import annotations`，Python 3.9 的 @dataclass 在解析字符串
# 注解时会回查 sys.modules[cls.__module__]，未注册会取到 None 而报
# AttributeError（'NoneType' object has no attribute '__dict__'）。
# --------------------------------------------------------------------------- #
_SCRIPT = (
    Path(__file__).resolve().parent.parent / "scripts" / "gatekeeper_workflow.py"
)
_spec = importlib.util.spec_from_file_location("gatekeeper_workflow", _SCRIPT)
assert _spec and _spec.loader, f"无法定位被测脚本：{_SCRIPT}"
gw = importlib.util.module_from_spec(_spec)
sys.modules["gatekeeper_workflow"] = gw
_spec.loader.exec_module(gw)  # type: ignore[union-attr]

ScoreInput = gw.ScoreInput
Finding = gw.Finding
score_dimensions = gw.score_dimensions
evaluate_hard_gates = gw.evaluate_hard_gates
decide_verdict = gw.decide_verdict
# 默认策略（深拷贝一份，避免任何用例意外改到共享 dict）
import json as _json  # noqa: E402

DEFAULT_POLICY = _json.loads(_json.dumps(gw.DEFAULT_POLICY))


# --------------------------------------------------------------------------- #
# 构造辅助：把「严重度计数 / 文件数 / commit 计数」翻译成 ScoreInput 字段
# --------------------------------------------------------------------------- #

def _findings(blocker: int = 0, major: int = 0, minor: int = 0, nit: int = 0):
    """按严重度计数生成 Finding 列表（message/file/line 对评分无影响，仅 severity 计 penalty）。"""
    out = []
    for sev, n in (("blocker", blocker), ("major", major), ("minor", minor), ("nit", nit)):
        for i in range(n):
            out.append(Finding(severity=sev, message=f"{sev} #{i}", file="f.go", line=i + 1))
    return out


def _commits(conforming: int, total: int):
    """生成 total 条 commit message，其中 conforming 条符合 Conventional Commits。"""
    assert conforming <= total
    msgs = [f"feat(mod{i}): conforming change {i}" for i in range(conforming)]
    msgs += [f"wip update {i}" for i in range(total - conforming)]  # 'wip ...' 不匹配规约
    return msgs


def _files(n: int):
    """生成 n 个占位变更文件路径（仅用于 size 维度计 len，src/test 计数由字段直接给定）。"""
    return [f"path/file_{i}.go" for i in range(n)]


def _build(
    *,
    pr_id: str,
    title: str,
    desc_len: int,
    linked_issue: bool,
    n_files: int,
    src: int,
    tests: int,
    commits: tuple,  # (conforming, total)
    ci: str,
    findings_counts: dict,
) -> ScoreInput:
    description = "x" * desc_len if desc_len else ""
    return ScoreInput(
        pr_id=pr_id,
        title=title,
        description=description,
        changed_files=_files(n_files),
        changed_src=src,
        changed_tests=tests,
        commits=_commits(*commits),
        ci_status=ci,
        linked_issue=linked_issue,
        findings=_findings(**findings_counts),
    )


def _run(inp: ScoreInput):
    """跑完整确定性链路，返回 (total, verdict)。"""
    dims = score_dimensions(inp, DEFAULT_POLICY)
    failures = evaluate_hard_gates(inp, DEFAULT_POLICY)
    verdict = decide_verdict(dims["total"], bool(failures), DEFAULT_POLICY)
    return dims, failures, verdict


# --------------------------------------------------------------------------- #
# 四个权威案例
# --------------------------------------------------------------------------- #

class TestAuthoritativeCases(unittest.TestCase):
    """对照 examples/ 下四个裁决记录，断言总分与裁决。"""

    def test_decision_pass(self):
        # decision-pass.md：3 src / 2 test、desc 142(含#198)、4/4 commit、CI passing、
        # 0/0/1/2 findings → 33+17+15+15+10 = 90 → PASS
        inp = _build(
            pr_id="214",
            title="feat(search): validate pagination params",
            desc_len=142,
            linked_issue=True,
            n_files=5,
            src=3,
            tests=2,
            commits=(4, 4),
            ci="passing",
            findings_counts={"minor": 1, "nit": 2},
        )
        dims, failures, verdict = _run(inp)
        self.assertEqual(dims["review_findings"]["score"], 33)
        self.assertEqual(dims["test_coverage"]["score"], 17)
        self.assertEqual(dims["pr_hygiene"]["score"], 15)
        self.assertEqual(dims["commit_quality"]["score"], 15)
        self.assertEqual(dims["ci_status"]["score"], 10)
        self.assertEqual(failures, [])
        self.assertEqual(dims["total"], 90)
        self.assertEqual(verdict, "PASS")

    def test_decision_request_changes(self):
        # decision-request-changes.md：4 src / 0 test（触发硬门禁
        # require_tests_for_src_changes）、desc 88 无关联、2/3 commit、CI passing、
        # 0/1/1/2 findings → 8+0+10+10+10 = 38 → REQUEST_CHANGES
        inp = _build(
            pr_id="305",
            title="refactor(billing): rework settlement pipeline",
            desc_len=88,
            linked_issue=False,
            n_files=4,
            src=4,
            tests=0,
            commits=(2, 3),
            ci="passing",
            findings_counts={"major": 1, "minor": 1, "nit": 2},
        )
        dims, failures, verdict = _run(inp)
        self.assertEqual(dims["review_findings"]["score"], 8)
        self.assertEqual(dims["test_coverage"]["score"], 0)
        self.assertEqual(dims["pr_hygiene"]["score"], 10)
        self.assertEqual(dims["commit_quality"]["score"], 10)
        self.assertEqual(dims["ci_status"]["score"], 10)
        gate_names = {f["gate"] for f in failures}
        self.assertIn("require_tests_for_src_changes", gate_names)
        self.assertEqual(dims["total"], 38)
        self.assertEqual(verdict, "REQUEST_CHANGES")

    def test_decision_comment(self):
        # decision-comment.md：2 src / 1 test、desc 52 无关联、2/3 commit、CI passing、
        # 0/0/3/2 findings → 23+15+10+10+10 = 68 ∈ [60,85) 且无硬门禁 → COMMENT
        inp = _build(
            pr_id="277",
            title="feat(config): merge defaults on load",
            desc_len=52,
            linked_issue=False,
            n_files=2,
            src=2,
            tests=1,
            commits=(2, 3),
            ci="passing",
            findings_counts={"minor": 3, "nit": 2},
        )
        dims, failures, verdict = _run(inp)
        self.assertEqual(dims["review_findings"]["score"], 23)
        self.assertEqual(dims["test_coverage"]["score"], 15)
        self.assertEqual(dims["pr_hygiene"]["score"], 10)
        self.assertEqual(dims["commit_quality"]["score"], 10)
        self.assertEqual(dims["ci_status"]["score"], 10)
        self.assertEqual(failures, [])
        self.assertEqual(dims["total"], 68)
        self.assertEqual(verdict, "COMMENT")

    def test_scorecard_sample(self):
        # scorecard-sample.md：4 src / 0 test（触发硬门禁）、desc 64 无关联、3/4 commit、
        # CI passing、0/1/2/1 findings → 4+0+10+11+10 = 35 → REQUEST_CHANGES
        inp = _build(
            pr_id="128",
            title="feat(auth): add refresh-token rotation",
            desc_len=64,
            linked_issue=False,
            n_files=6,
            src=4,
            tests=0,
            commits=(3, 4),
            ci="passing",
            findings_counts={"major": 1, "minor": 2, "nit": 1},
        )
        dims, failures, verdict = _run(inp)
        self.assertEqual(dims["review_findings"]["score"], 4)
        self.assertEqual(dims["test_coverage"]["score"], 0)
        self.assertEqual(dims["pr_hygiene"]["score"], 10)
        self.assertEqual(dims["commit_quality"]["score"], 11)
        self.assertEqual(dims["ci_status"]["score"], 10)
        gate_names = {f["gate"] for f in failures}
        self.assertIn("require_tests_for_src_changes", gate_names)
        self.assertEqual(dims["total"], 35)
        self.assertEqual(verdict, "REQUEST_CHANGES")


class TestVerdictBoundaries(unittest.TestCase):
    """裁决判定树（decide_verdict）边界：与 thresholds pass=85 / request_changes=60 一致。"""

    def test_pass_threshold_inclusive(self):
        self.assertEqual(decide_verdict(85, False, DEFAULT_POLICY), "PASS")

    def test_comment_band(self):
        self.assertEqual(decide_verdict(60, False, DEFAULT_POLICY), "COMMENT")
        self.assertEqual(decide_verdict(84, False, DEFAULT_POLICY), "COMMENT")

    def test_request_changes_below_band(self):
        self.assertEqual(decide_verdict(59, False, DEFAULT_POLICY), "REQUEST_CHANGES")

    def test_hard_gate_short_circuits_high_score(self):
        # 即便满分，硬门禁失败也直接 REQUEST_CHANGES
        self.assertEqual(decide_verdict(100, True, DEFAULT_POLICY), "REQUEST_CHANGES")


if __name__ == "__main__":
    unittest.main(verbosity=2)
