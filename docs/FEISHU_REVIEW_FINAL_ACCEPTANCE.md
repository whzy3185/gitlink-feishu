# Feishu Review final acceptance

## Result at documentation time

- Original failed gate: run `30911026158`, job `91997535171`.
- Root cause: Windows-style paths were classified with host-dependent
  `filepath.Base` and `filepath.IsAbs` semantics on Linux.
- Repaired gate: run `30917906873`, job `92020647498`, success.
- Local `go test ./... -count=1`, `go vet ./...` and production build: pass.
- Offline failure injection: 20/20 pass, blind non-idempotent retries: 0.
- Backup/Restore, Retention/WAL, graceful shutdown, Admin and health contracts:
  pass on the local Windows environment.
- Secret scan: pass with exact fixture allowlist and zero findings.
- Final-SHA Linux/Windows/Race/PowerShell/Node jobs: pending until push.
- Real GitLink, public webhook, two-chat, Card/Reply, Base, Doc and Task Stage Six
  validators: not executed.

## Safe execution

Default offline acceptance:

```powershell
.\scripts\run-feishu-review-final-acceptance.ps1 `
  -OfflineOnly `
  -OutputDirectory .\evidence\final
```

Real acceptance is allowed only after offline success and requires both:

```text
FEISHU_REVIEW_REAL_VALIDATION=1
FEISHU_REVIEW_CONFIRM_EXTERNAL_WRITES=YES
```

Then run with `-RealPlatform -ConfirmExternalWrites`. Preflight requires two
different test chats, a test Base table, test Doc folder, test Task assignee,
dedicated state/backup paths, HTTPS webhook URL and a `review-final-` prefix.

## Evidence interpretation

The committed package always contains all 34 named files. A `real-*` file with
`validation_mode=not_executed` and `passed=false` is a gap record, not proof.
Only a separately generated result with `validation_mode=real_platform` and
`passed=true` closes that gate. The manifest hashes non-manifest files and
self-excludes its two manifest files to avoid recursive hashing.
