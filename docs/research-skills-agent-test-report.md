# Research Skills Agent Test Report

## Scope

This report covers the seven read-only research Skills added for the GitLink competition:

- `gitlink-research-reproducibility`
- `gitlink-research-compliance`
- `gitlink-research-progress-tracker`
- `gitlink-research-collaboration-map`
- `gitlink-research-knowledge-graph`
- `gitlink-research-data-provenance`
- `gitlink-research-artifact-handbook`

The Skills are standalone `gitlink-cli` Agent Skills. They do not depend on external project code and can be submitted as an independent Skill PR.

## Automatic Validation

Run:

```bash
make validate-research-skills
git diff --check
```

The validation script checks:

- every Skill has valid `SKILL.md` frontmatter with the expected name and a useful trigger description;
- every Skill instructs the Agent to read `gitlink-shared`;
- every Skill is read-only by default and uses `gitlink-cli --format json`;
- every Skill includes command coverage for its scenario;
- every Skill has at least one reference file and one example file;
- examples contain user request, Agent steps, expected answer, and concrete `gitlink-cli` commands;
- `skills/README.md` and `doc/changes/research-skills-suite.md` index all seven Skills.

## Offline Agent Test Prompts

These prompts can be used in Claude Code, Cursor, Codex, or OpenClaw after installing the Skills.

### Reproducibility

```text
Use gitlink-research-reproducibility to audit songhui18/ICCV2021. Produce a reproducibility score, missing items, and top three remediation actions. Do not write back to GitLink.
```

Expected behavior:

- read `gitlink-shared`;
- run `repo +info` and `repo +tree`, or fall back to `api GET /sub_entries`;
- cite command sources;
- produce a checklist report.

### Compliance

```text
Use gitlink-research-compliance to check Gitconomy/Git4Research before open-source release. Report license, SECURITY, CONTRIBUTING, dependency, and sensitive-file risks.
```

Expected behavior:

- perform read-only file tree checks;
- avoid printing sensitive contents;
- state that the output is an engineering pre-check, not legal advice.

### Progress Tracking

```text
Use gitlink-research-progress-tracker to generate a weekly research progress report for songhui18/ICCV2021, including Issue/PR queues and risk warnings.
```

Expected behavior:

- collect repo, Issue, PR, Release, and contributor data;
- separate facts from recommendations;
- avoid posting a comment unless explicitly requested.

### Collaboration Map

```text
Use gitlink-research-collaboration-map to analyze collaboration structure for Gitconomy/Git4Research. Identify maintainers, contributors, open collaboration gaps, and suggested next actions.
```

Expected behavior:

- use public contributor, Issue, PR, language, and optional user data;
- avoid private contact information;
- base recommendations on evidence.

### Knowledge Graph

```text
Use gitlink-research-knowledge-graph to search GitLink for “论文复现” and “open research”, then output a Markdown trend summary and graph JSON.
```

Expected behavior:

- run bounded keyword searches;
- deep-analyze no more than eight repositories;
- produce `nodes` and `edges` using the documented graph schema;
- mark platform-data limits.

### Data Provenance

```text
Use gitlink-research-data-provenance to audit songhui18/ICCV2021 for dataset source, citation, license, and privacy risks. Do not print raw data contents.
```

Expected behavior:

- collect repo info and selected file trees;
- report paths and risk types only;
- distinguish missing evidence from confirmed risk.

### Artifact Handbook

```text
Use gitlink-research-artifact-handbook to generate a research artifact handbook for Gitconomy/Git4Research, suitable for project handoff and competition demo.
```

Expected behavior:

- collect repo, README, tree, Issue/PR, Release, language, and contributor signals when available;
- produce a structured handbook with missing fields marked as `待补充`;
- avoid writing Wiki, Issue, Release, or repository files unless explicitly requested.

## Real Agent Verification

For competition submission, record at least one Agent run with screenshots or video:

1. Install or expose the Skills to the Agent.
2. Run one prompt from this report against a public GitLink repository.
3. Show the Agent reading the Skill, running read-only `gitlink-cli` commands, and producing the report.
4. Show that no write operation was performed.

Recommended demo target:

```text
songhui18/ICCV2021
```

This repository has richer Issue and PR data, which makes progress tracking and collaboration mapping more visible.

## Known Limitations

- The automatic script validates Skill structure and workflow instructions; it does not prove semantic quality of an LLM-generated report.
- Real GitLink API verification requires network access and may require authentication for private repositories.
- Screenshots or video still need to be captured in the selected Agent UI for the competition submission package.
