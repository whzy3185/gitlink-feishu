# Workflow Shortcut-Backed Fetches

## Summary

Adds `workflow +review-context`, a read-only workflow command that returns a deterministic PR review context bundle for AI agents and maintainers.

```bash
gitlink-cli workflow +review-context --owner Gitlink --repo gitlink-cli --number 1 --format json
```

## Behavior

The command aggregates shortcut-backed read-only fetches into one structured output:

- Repository info, equivalent to `repo +info`
- Pull request details, equivalent to `pr +view`
- Changed files, equivalent to `pr +files`
- Existing reviews, equivalent to `pr +reviews`
- Open issue context, equivalent to `issue +list --state open`
- Issue labels, equivalent to `label +list`

Each optional section can be disabled with `--include-...=false`, and `--issue-limit` / `--label-limit` bound the amount of context returned. Partial section failures are recorded in `notes` so an Agent can continue with the available data.

## Why

Code review and gatekeeper Skills previously instructed Agents to stitch together multiple raw API calls or separate shortcut calls before analysis. A single read-only workflow command makes review context deterministic, easier to test, and safer for Agent usage.

## Safety

- The command is read-only.
- It does not comment, approve, reject, merge, label, close, or modify remote resources.
- Write operations such as PR review submission remain explicit commands and still require user confirmation.

## Documentation

- Updates README examples in English and Chinese.
- Updates `gitlink-workflow` with `workflow +review-context`.
- Updates code review and gatekeeper Skills to prefer the workflow context bundle.
- Updates insight guidance to use shortcut-covered repo and user fetches before Raw API.

## Verification

```bash
go test ./shortcuts/workflow ./shortcuts
go test ./...
```
