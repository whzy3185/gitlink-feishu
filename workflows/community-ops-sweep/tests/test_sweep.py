#!/usr/bin/env python3
"""community_ops_sweep 引擎纯逻辑单测 —— 把"同输入→同 plan"钉成可回归事实。

只测确定性纯函数（标签增删 / 状态映射 / 链接校验 / plan 组装），不碰 gitlink-cli I/O。
纯标准库 unittest（与 gatekeeper test_scoring.py 同风格）。

运行：
    python3 examples/workflows/community-ops-sweep/tests/test_sweep.py
"""

from __future__ import annotations

import importlib.util
import sys
import unittest
from datetime import datetime, timezone
from pathlib import Path

# 以绝对路径加载被测脚本（非包，按文件载入；先注册 sys.modules 以兼容 dataclass+annotations）
_SCRIPT = (
    Path(__file__).resolve().parent.parent / "scripts" / "community_ops_sweep.py"
)
_spec = importlib.util.spec_from_file_location("community_ops_sweep", _SCRIPT)
assert _spec and _spec.loader, f"无法定位被测脚本：{_SCRIPT}"
sw = importlib.util.module_from_spec(_spec)
sys.modules["community_ops_sweep"] = sw
_spec.loader.exec_module(sw)  # type: ignore[union-attr]

TAG_MAP = {"bug": "3", "feature": "5", "docs": "7", "question": "9", "enhancement": "11"}
POLICY = {
    "status_map": {"duplicate": "closed", "answered": "closed", "spam": "closed"},
}


class TestMergeTagIds(unittest.TestCase):
    def test_add_to_empty(self):
        self.assertEqual(sw.merge_tag_ids([], ["bug"], [], TAG_MAP), ["3"])

    def test_add_preserves_existing_union(self):
        # 整体替换语义：现有 ∪ 新增
        self.assertEqual(sw.merge_tag_ids(["3", "7"], ["feature"], [], TAG_MAP), ["3", "5", "7"])

    def test_remove_drops_tag(self):
        self.assertEqual(sw.merge_tag_ids(["3", "7"], [], ["bug"], TAG_MAP), ["7"])

    def test_add_and_remove_together(self):
        # current[3,7] ∪ {5} \ {7} = {3,5}
        self.assertEqual(sw.merge_tag_ids(["3", "7"], ["feature"], ["docs"], TAG_MAP), ["3", "5"])

    def test_unknown_add_filtered_by_whitelist(self):
        self.assertEqual(sw.merge_tag_ids([], ["nonexistent"], [], TAG_MAP), [])

    def test_result_sorted_deduped(self):
        self.assertEqual(sw.merge_tag_ids(["7", "3"], ["bug"], [], TAG_MAP), ["3", "7"])


class TestMapStatus(unittest.TestCase):
    def test_duplicate_closes(self):
        self.assertEqual(sw.map_status("duplicate", POLICY), "closed")

    def test_unknown_action_no_change(self):
        self.assertIsNone(sw.map_status("schedule_fix", POLICY))

    def test_none_action_no_change(self):
        self.assertIsNone(sw.map_status(None, POLICY))


class TestBuildPlanTriage(unittest.TestCase):
    def _candidates(self, issues):
        return {"issues": issues, "merged_prs": [], "open_issues": {i["number"] for i in issues}}

    def test_plan_adds_tag(self):
        candidates = self._candidates([{"number": "42", "title": "登录失败", "description": "", "tags": [], "state": "open"}])
        triage = [{"issue": "42", "tag_names": ["bug"], "remove_tag_names": [], "recommended_action": "schedule_fix"}]
        plan = sw.build_plan(candidates, triage, [], [], {}, POLICY, TAG_MAP)
        tag_writes = [w for w in plan["writes"] if w.get("issue") == "42" and w["op"] == "update_tags"]
        self.assertTrue(any(w["tag_ids"] == ["3"] for w in tag_writes), plan["writes"])

    def test_plan_removes_tag(self):
        candidates = self._candidates([{"number": "42", "title": "x", "description": "", "tags": [{"id": "3", "name": "bug"}], "state": "open"}])
        triage = [{"issue": "42", "tag_names": [], "remove_tag_names": ["bug"], "recommended_action": "schedule_fix"}]
        plan = sw.build_plan(candidates, triage, [], [], {}, POLICY, TAG_MAP)
        tag_writes = [w for w in plan["writes"] if w.get("issue") == "42" and w["op"] == "update_tags"]
        # 期望全集为空 → tag_ids == []
        self.assertTrue(any(w["tag_ids"] == [] for w in tag_writes), plan["writes"])

    def test_plan_closes_duplicate(self):
        candidates = self._candidates([{"number": "43", "title": "dup", "description": "", "tags": [], "state": "open"}])
        triage = [{"issue": "43", "tag_names": [], "remove_tag_names": [], "recommended_action": "duplicate"}]
        plan = sw.build_plan(candidates, triage, [], [], {}, POLICY, TAG_MAP)
        status_writes = [w for w in plan["writes"] if w.get("issue") == "43" and w["op"] == "update_status"]
        self.assertTrue(any(w["state"] == "closed" for w in status_writes), plan["writes"])


class TestBuildPlanLinks(unittest.TestCase):
    def test_link_closes_only_open_existing_issue(self):
        candidates = {
            "issues": [],
            "merged_prs": [{"number": "128", "title": "fix login", "description": "fixes #45", "author": "alice"}],
            "open_issues": {"45"},  # 45 开着；99 不在 → 不存在/未开
        }
        links = [{"pr": "128", "linked_issue_numbers": ["45", "99"]}]
        plan = sw.build_plan(candidates, [], [], links, {}, POLICY, TAG_MAP)
        closes = [w for w in plan["writes"] if w["op"] == "update_status" and w.get("state") == "closed"]
        closed = {w["issue"] for w in closes}
        self.assertIn("45", closed)
        self.assertNotIn("99", closed)

    def test_link_no_match_no_write(self):
        candidates = {"issues": [], "merged_prs": [{"number": "1", "title": "x", "description": "", "author": "a"}], "open_issues": set()}
        links = [{"pr": "1", "linked_issue_numbers": []}]
        plan = sw.build_plan(candidates, [], [], links, {}, POLICY, TAG_MAP)
        self.assertFalse(any(w["op"] == "update_status" for w in plan["writes"]))


class TestBuildPlanOwnerSuggest(unittest.TestCase):
    def test_owner_suggestion_emits_comment_not_assign(self):
        # R2 默认建议：写 comment，不写 assign
        candidates = {"issues": [{"number": "50", "title": "t", "description": "", "tags": [], "state": "open"}], "merged_prs": [], "open_issues": {"50"}}
        owners = [{"issue": "50", "suggested_owners": ["alice", "bob"], "evidence": "近期 PR#128"}]
        plan = sw.build_plan(candidates, [], owners, [], {}, POLICY, TAG_MAP)
        comments = [w for w in plan["writes"] if w.get("issue") == "50" and w["op"] == "comment"]
        assigns = [w for w in plan["writes"] if w.get("issue") == "50" and w["op"] == "update_assigner"]
        self.assertTrue(comments, "应有建议评论")
        self.assertFalse(assigns, "默认不应自动指派")


class TestCoerceDates(unittest.TestCase):
    """candidates.json 经磁盘 round-trip 后日期变字符串；渲染前要 coerce 回 datetime。"""

    def test_parses_string_dates(self):
        out = sw.coerce_dates({"updated_at": "2026-06-15T10:00:00+00:00", "created_at": "2026-06-10T00:00:00+00:00", "x": 1})
        self.assertIsInstance(out["updated_at"], datetime)
        self.assertIsInstance(out["created_at"], datetime)
        self.assertEqual(out["x"], 1)

    def test_datetime_and_none_passthrough(self):
        dt = datetime(2026, 6, 1, tzinfo=timezone.utc)
        out = sw.coerce_dates({"updated_at": dt, "created_at": None})
        self.assertIs(out["updated_at"], dt)
        self.assertIsNone(out["created_at"])

    def test_custom_keys(self):
        out = sw.coerce_dates({"merged_at": "2026-06-15T10:00:00+00:00"}, keys=("merged_at",))
        self.assertIsInstance(out["merged_at"], datetime)


if __name__ == "__main__":
    unittest.main(verbosity=2)
