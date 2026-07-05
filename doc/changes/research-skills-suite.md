# Research skills suite

Adds seven read-only research-oriented Agent Skills for GitLink repositories:

- `gitlink-research-reproducibility`: reproducibility checklist and remediation plan for research code.
- `gitlink-research-compliance`: license, security policy, dependency, and sensitive-file risk audit.
- `gitlink-research-progress-tracker`: weekly progress report and risk early warning for research projects.
- `gitlink-research-collaboration-map`: contributor, Issue, and PR collaboration mapping for research teams.
- `gitlink-research-knowledge-graph`: keyword-based research landscape and lightweight knowledge graph workflow.
- `gitlink-research-data-provenance`: dataset source, citation, license, and privacy-risk audit.
- `gitlink-research-artifact-handbook`: research artifact handbook generation for demos, handoff, and open-source release.

These Skills are designed as standalone `gitlink-cli` Skill PR deliverables. They do not depend on external project code and can also be composed by end-to-end research workflows.

Validation:

```bash
make validate-research-skills
git diff --check
```

Agent verification guidance is documented in `docs/research-skills-agent-test-report.md`.
