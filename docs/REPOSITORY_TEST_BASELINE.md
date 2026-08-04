# Repository test baseline

## Scope

This document records the repository-wide baseline restored during Stage Six on
`chore/feishu-review-final-hardening`. The product baseline is Stage Five commit
`e2bcb35efc56db659eb58af141a3fd8018e6d5be`; the cross-platform CI baseline is
`12a83b5038a84329dbff7e5af6c3d7a316b33d0b`.

This work does not add any GitLink or Feishu platform write path. All commands
below were run locally against tests, fixtures and compiler inputs only.

## Initial failures

The first repository-wide `go test ./... -count=1` exposed historical contract
drift that predated the Stage One through Stage Five review platform work. The
failures fell into the following groups.

| Area | Failure type | Representative symptom | Resolution |
| --- | --- | --- | --- |
| HTTP client | compilation and response classification | missing multipart compatibility and full HTML responses treated as JSON/API failures | restore streaming multipart helper and deterministic HTML response detection |
| API command | obsolete command contract | pagination, variable expansion and dry-run paths disagreed with tests | restore the documented single-request and paginated command contracts |
| root command | missing registration | `completion` and `doctor` were implemented or tested but not exposed consistently | register both commands and restore completion implementation |
| output | nondeterministic or incomplete formatting | map order, resource envelopes and error codes did not match the CLI contract | sort fields and preserve resource/error metadata |
| shortcut compatibility | missing symbols and renamed helpers | issue batch, compare summary, release assets and legacy test helpers did not compile | add compatibility wrappers or restore the missing implementation |
| shortcut behavior | production/test drift | CI, commit, issue, label, member, milestone, notification, organization, release, repository, search, snippet, user and webhook contracts diverged | restore the production behavior expected by the existing tests without deleting assertions |
| skill metadata | schema drift | existing optional frontmatter and `bins_any` were rejected | extend validation while preserving required-field checks |
| localization | missing keys | contributor chart output referenced absent locale keys | add the English and Chinese keys |

The initial baseline also contained duplicate or legacy test helper names. These
were resolved by uniquely naming the legacy coverage and adding narrowly scoped
test helpers. No failing test was deleted, skipped, weakened, or excluded from
the full repository command.

## Platform note

The first Windows link attempt exhausted free space on the system drive and
failed with `link.exe: There is not enough space on disk`. That was an execution
environment failure, not a code failure. Go build and test caches were moved to
dedicated directories on drive `E:` for verification:

```powershell
$env:GOCACHE = 'E:\GitLinkCLI-Competition\.gocache-stage6'
$env:GOTMPDIR = 'E:\GitLinkCLI-Competition\.gotmp-stage6'
```

The cache directories and `.local` logs are not committed.

## Restored baseline

The following commands completed successfully on Windows with Go caches on
drive `E:`:

```text
go test ./... -count=1  exit 0
go vet ./...            exit 0
go build -o .local/gitlink-cli-stage6.exe .  exit 0
```

The resulting Windows binary was 25,253,888 bytes. Linux, Linux race and hosted
Windows results remain GitHub Actions evidence and must not be inferred from
this local run.

## Safety and provenance

- GitLink network writes executed: 0.
- Feishu network writes executed: 0.
- Existing test assertions removed: 0.
- New `t.Skip` calls added to hide failures: 0.
- CI packages excluded from `go test ./...`: 0.
- Repository history rewritten: no.

The complete fix is isolated in commit
`8cd9dbab989d29fad80dcb47bf6dfd2ee2cd8376` (`fix(repo): restore full repository
test baseline`).
