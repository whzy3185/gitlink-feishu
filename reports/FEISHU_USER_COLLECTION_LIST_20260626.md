# Feishu / GitLink Project Completion Checklist

Date: 2026-06-26

This checklist records what still needs manual evidence or product decisions
after the real Feishu validation run. Do not write real secrets, webhook URLs,
app secrets, table IDs, document IDs, open IDs, union IDs, or GitLink tokens in
this file.

## Current Validation State

```text
Custom bot webhook: configured and real send passed.
Self-built app credentials: configured and tenant_access_token passed.
DocX target: configured and real append passed.
Bitable Base: configured.
Bitable one-table validation: passed.
Bitable split-table validation: passed.
Feishu task creation: basic task create passed.
GitLink repository source: Gitlink/gitlink-cli real workflow report generated.
zh-CN output: available for Feishu cards, digests, DocX blocks, and task candidates.
Image evidence: deferred and not part of this upload.
```

## Image Evidence

Screenshots and other image files are intentionally deferred for this round.

```text
Do not add screenshots or image files in this upload.
Use the text smoke report and permission matrix as current evidence.
```

If visual evidence is needed later, capture it in a separate documentation pass
and redact all visible IDs or secrets before committing.

## Bitable Demonstration State

The first validation used one test table with multiple views. The follow-up
validation created or reused five separate tables:

```text
gitlink_reports
gitlink_issues
gitlink_prs
gitlink_contributors
gitlink_tasks
```

Real split-table write result:

```text
reports: 1 record
issues: 5 records
prs: 2 records
contributors: 1 record
tasks: 7 records
```

Use the split-table text evidence for this upload because it demonstrates that
each supported table can receive records independently.

## Test Permission Note

The test enterprise used a self-built Feishu app with broad permissions so the
CLI could validate DocX, Base, and Task APIs end to end. This is only for local
validation. Production deployments should use least-privilege scopes and
resource-level access.

## Next-Stage Capability Boundary

These are intentionally not implemented in the current branch:

```text
Feishu Task project placement
Feishu Task section placement
Feishu Task assignees and followers
Feishu-side task dedupe/search
Bitable Base/table/field/view creation as a product command
Feishu card callback server
Feishu-triggered GitLink writes
GitLink issue comments from Feishu
GitLink PR reviews from Feishu
GitLink merge/close/member/webhook actions from Feishu
```

## Manual Decisions

```text
1. Decide later whether visual evidence is needed at all.
2. Decide whether experimental Open Platform writes should remain enabled in
   the submitted branch or stay documented as validation-only.
3. Decide whether the next implementation stage should prioritize Task
   assignees/followers or Bitable view/table automation.
```

## Safety

```text
Keep .local/feishu-gitlink.env.ps1 ignored.
Do not commit real Feishu or GitLink credentials.
Do not commit screenshots containing secrets or raw IDs.
Do not rerun task-create repeatedly unless duplicate test tasks are acceptable.
```
