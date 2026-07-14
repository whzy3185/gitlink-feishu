"""test_helpers.py — gitlink_data / collect 归一化工具的纯单元测试（不联网）。

运行：`pytest scripts/research/` 或 `python scripts/research/test_helpers.py`
"""
import sys
import os

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))

import gitlink_data as gd  # noqa: E402
import collect  # noqa: E402


def test_first_list_list_input():
    assert gd.first_list([1, 2, 3]) == [1, 2, 3]


def test_first_list_known_key():
    assert gd.first_list({"projects": [{"id": 1}], "total_count": 1}) == [{"id": 1}]


def test_first_list_single_list_value():
    # 无已知键但只有一个列表值时，兜底返回它
    assert gd.first_list({"whatever": [9, 9]}) == [9, 9]


def test_first_list_empty():
    assert gd.first_list({"total_count": 0}) == []
    assert gd.first_list(None) == []


def test_total_count():
    assert gd.total_count({"total_count": 42}) == 42
    assert gd.total_count({"totalCount": 7}) == 7
    assert gd.total_count({"projects": []}) is None


def test_login_of_flat():
    assert collect.login_of({"login": "whale"}) == "whale"


def test_login_of_nested_author():
    assert collect.login_of({"author": {"login": "baoerjun"}}) == "baoerjun"


def test_login_of_name_fallback():
    assert collect.login_of({"name": "surponess"}) == "surponess"


def test_login_of_empty():
    assert collect.login_of({}) == ""
    assert collect.login_of("not a dict") == ""


def test_repo_fullname_with_author():
    project = {"identifier": "gitlink-cli", "author": {"login": "whale_hihihi"}}
    assert collect.repo_fullname(project) == "whale_hihihi/gitlink-cli"


def test_repo_fullname_missing_owner():
    assert collect.repo_fullname({"identifier": "foo"}) == "foo"


def test_as_int_as_float():
    assert collect.as_int("12") == 12
    assert collect.as_int(None) == 0
    assert collect.as_float("1.5") == 1.5
    assert collect.as_float("x", -1.0) == -1.0


def _run_all():
    fns = [v for k, v in sorted(globals().items()) if k.startswith("test_")]
    for fn in fns:
        fn()
        print(f"PASS {fn.__name__}")
    print(f"\nAll {len(fns)} helper tests passed.")


if __name__ == "__main__":
    _run_all()
