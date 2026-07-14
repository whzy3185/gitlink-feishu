#!/usr/bin/env python3
from __future__ import annotations

import re
import sys
from pathlib import Path


RESEARCH_SKILLS = {
    "gitlink-research-reproducibility": {
        "commands": ["repo +info", "repo +tree", "api GET"],
        "references": ["checklist.md"],
        "examples": ["reproducibility-audit.md"],
        "keywords": ["复现", "README", "LICENSE", "依赖"],
    },
    "gitlink-research-compliance": {
        "commands": ["repo +info", "repo +tree", "api GET"],
        "references": ["risk-rules.md"],
        "examples": ["compliance-audit.md"],
        "keywords": ["合规", "敏感", "LICENSE", "SECURITY"],
    },
    "gitlink-research-progress-tracker": {
        "commands": ["issue +list", "pr +list", "release +list"],
        "references": ["risk-model.md"],
        "examples": ["weekly-report.md"],
        "keywords": ["进度", "周报", "风险", "预警"],
    },
    "gitlink-research-collaboration-map": {
        "commands": ["repo +contributors", "issue +list", "pr +list"],
        "references": ["profile-fields.md"],
        "examples": ["collaboration-map.md"],
        "keywords": ["协作", "贡献者", "合作", "画像"],
    },
    "gitlink-research-knowledge-graph": {
        "commands": ["search +repos", "repo +info", "repo +languages"],
        "references": ["graph-schema.md"],
        "examples": ["keyword-map.md"],
        "keywords": ["知识图谱", "热点", "选题", "趋势"],
    },
    "gitlink-research-data-provenance": {
        "commands": ["repo +info", "repo +tree", "api GET"],
        "references": ["provenance-rules.md"],
        "examples": ["data-provenance-audit.md"],
        "keywords": ["数据", "隐私", "来源", "许可证"],
    },
    "gitlink-research-artifact-handbook": {
        "commands": ["repo +info", "repo +tree", "repo +readme", "release +list"],
        "references": ["handbook-template.md"],
        "examples": ["artifact-handbook.md"],
        "keywords": ["成果", "手册", "交接", "复现"],
    },
}


def fail(message: str) -> None:
    print(f"ERROR: {message}", file=sys.stderr)
    raise SystemExit(1)


def read(path: Path) -> str:
    try:
        return path.read_text(encoding="utf-8")
    except FileNotFoundError:
        fail(f"缺少文件：{path}")


def parse_frontmatter(text: str, path: Path) -> dict[str, str]:
    if not text.startswith("---\n"):
        fail(f"{path} 未以 YAML frontmatter 开头")
    try:
        _, frontmatter, _ = text.split("---", 2)
    except ValueError:
        fail(f"{path} 的 YAML frontmatter 不完整")
    parsed: dict[str, str] = {}
    for line in frontmatter.strip().splitlines():
        if ":" not in line:
            fail(f"{path} 的 frontmatter 行缺少冒号：{line}")
        key, value = line.split(":", 1)
        parsed[key.strip()] = value.strip().strip('"')
    return parsed


def assert_contains(text: str, needle: str, path: Path) -> None:
    if needle not in text:
        fail(f"{path} 缺少必需文本：{needle}")


def assert_regex(text: str, pattern: str, path: Path) -> None:
    if not re.search(pattern, text, flags=re.MULTILINE):
        fail(f"{path} 未匹配必需模式：{pattern}")


def validate_skill(root: Path, name: str, spec: dict[str, list[str]]) -> None:
    skill_dir = root / "skills" / name
    skill_md = skill_dir / "SKILL.md"
    text = read(skill_md)
    frontmatter = parse_frontmatter(text, skill_md)

    if frontmatter.get("name") != name:
        fail(f"{skill_md} 的 name 字段错误：{frontmatter.get('name')}")
    if not re.fullmatch(r"\d+\.\d+\.\d+", frontmatter.get("version", "")):
        fail(f"{skill_md} 的 version 必须使用 x.y.z 格式")
    if "gitlink-cli" not in frontmatter.get("bins", ""):
        fail(f"{skill_md} 的 metadata.requires.bins 缺少 gitlink-cli")
    if not frontmatter.get("cliHelp", "").startswith("gitlink-cli "):
        fail(f"{skill_md} 的 cliHelp 必须以 gitlink-cli 开头")
    description = frontmatter.get("description", "")
    if len(description) < 30:
        fail(f"{skill_md} 的 description 过短")
    for keyword in spec["keywords"]:
        if keyword not in text and keyword not in description:
            fail(f"{skill_md} 缺少场景关键词：{keyword}")

    assert_contains(text, "gitlink-shared", skill_md)
    assert_contains(text, "gitlink-cli", skill_md)
    assert_contains(text, "--format json", skill_md)
    assert_regex(text, r"只读|默认只读", skill_md)
    assert_regex(text, r"不输出 Token|Token", skill_md)

    for command in spec["commands"]:
        assert_contains(text, command, skill_md)

    for filename in spec["references"]:
        path = skill_dir / "references" / filename
        ref_text = read(path)
        if len(ref_text.strip().splitlines()) < 5:
            fail(f"{path} 内容过少，无法作为有效参考")

    for filename in spec["examples"]:
        path = skill_dir / "examples" / filename
        example_text = read(path)
        assert_contains(example_text, "用户请求", path)
        assert_contains(example_text, "Agent 步骤", path)
        assert_contains(example_text, "预期回答", path)
        assert_contains(example_text, "gitlink-cli", path)


def validate_readme(root: Path) -> None:
    readme = root / "skills" / "README.md"
    text = read(readme)
    for name in RESEARCH_SKILLS:
        assert_contains(text, name, readme)


def validate_change_note(root: Path) -> None:
    note = root / "doc" / "changes" / "research-skills-suite.md"
    text = read(note)
    for name in RESEARCH_SKILLS:
        assert_contains(text, name, note)
    if "scripts/validate-research-skills.py" not in text and "make validate-research-skills" not in text:
        fail(f"{note} 缺少科研 Skills 验证命令")


def main() -> int:
    root = Path(__file__).resolve().parents[1]
    for name, spec in RESEARCH_SKILLS.items():
        validate_skill(root, name, spec)
        print(f"ok {name}")
    validate_readme(root)
    validate_change_note(root)
    print("research skill validation passed")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
