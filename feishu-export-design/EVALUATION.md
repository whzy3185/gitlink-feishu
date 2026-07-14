# Feishu Export Design Evaluation

## Current Baseline

Working directory:

```text
E:\GitLinkCLI-Competition\gitlink-cli-feishu-clean
```

Branch and base:

```text
feat/feishu-export-clean
origin/master ef7a2c6
```

Baseline tests:

```powershell
$env:GOPROXY='https://goproxy.cn,direct'; go test ./...
```

Result: passed.

The first default `go test ./...` attempt failed only because `proxy.golang.org` was unreachable for new dependencies. With a temporary Go proxy override, the latest master baseline is clean.

## Judgment

The v2 task chain is substantially better than the earlier version and is close to executable. It correctly narrows the feature into a safe one-way export module:

```text
workflow JSON -> local preview
workflow JSON -> Feishu bot card
workflow JSON -> weekly report
workflow JSON -> Bitable schema
workflow JSON -> Bitable-ready records
```

The remaining design adjustments are:

1. Keep the first implementation limited to local workflow JSON input.
2. Use `--send` as the only real Feishu bot send switch.
3. Use `prs` in flags and table keys, while displaying `Pull Requests` to users.
4. Replace any `sync-bitable` wording with `bitable-records`.
5. Do not implement real Bitable OpenAPI writes in this task.
6. Keep `doctor` out of the first implementation unless the other commands already exist.

After checking Feishu Open Platform documentation and the BotBuilder shutdown notice, add one more adjustment:

7. Add Feishu Docs export as the first real custom-app integration after the custom bot MVP.

Reason:

```text
Feishu custom bot = notification.
Feishu DocX = collaborative report artifact.
Bitable = structured tracking.
```

The previous design handled notification and Bitable dry-run, but did not use Feishu Docs. That makes the workflow less useful as a collaboration handoff.

## What Is Complete

The v2 task chain is complete enough for:

- Command skeleton design.
- Safety options and redaction design.
- Feishu bot card and signature design.
- Mocked webhook client design.
- Workflow JSON mapping design.
- Local weekly report generation.
- Bitable schema generation.
- Bitable-ready record generation.
- Agent Skill and documentation outline.

## What Is Not Closed Yet

The following should not be implemented in the first pass:

- Real Bitable record creation.
- Real Bitable record update.
- Bitable unique-key lookup through Feishu OpenAPI.
- Tenant token acquisition.
- Token cache behavior.
- Feishu Base/table/view creation.
- Any GitLink remote write.

The following should be designed before real Bitable writes:

- `feishu +doc-export`
- app_id/app_secret loading
- tenant_access_token fetch and cache
- document folder permission diagnostics
- DocX create-document and create-block behavior

These require a separate authentication and data consistency design.

## Practical Utility

The design is useful in a real CLI workflow:

1. A user generates a workflow report with the existing `workflow +repo-report` command.
2. The new `feishu` command consumes that JSON file.
3. The user previews a card, weekly report, schema, or records locally.
4. Only when `--send` is explicit does the CLI send a Feishu bot message.

This produces a small, testable feature that is useful without requiring Feishu enterprise app credentials.

The stronger practical workflow is:

```text
workflow JSON -> markdown report -> Feishu DocX -> Feishu bot card with doc link -> Bitable dry-run records
```

That path gives users both an immediate group notification and a persistent collaborative report.

## Implementation Scope Rating

```text
Direction: strong
Safety boundary: strong
Repository fit: strong
First implementation size: acceptable after removing real Bitable writes
Direct executability: good after using the clean task chain in this directory
```
