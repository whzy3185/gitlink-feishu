# Feishu Review final test matrix

| Test ID | Capability | Environment | Command/evidence | Result | Real platform | External writes |
| --- | --- | --- | --- | --- | --- | ---: |
| T01 | repaired collaboration gate | GitHub Actions Linux | run `30917906873`, job `92020647498` | pass | no | 0 |
| T02 | repository test baseline | Windows local | `go test ./... -count=1` | pass | no | 0 |
| T03 | repository vet baseline | Windows local | `go vet ./...` | pass | no | 0 |
| T04 | repository production build | Windows local | `go build` | pass | no | 0 |
| T05 | final failure matrix | offline fault injection | `inject-feishu-review-final-failures.ps1` | 20/20 pass | no | 0 |
| T06 | Secret scan | Windows local | `scan-feishu-review-secrets.ps1` | pass, findings 0 | no | 0 |
| T07 | Backup/Restore | Windows local | `verify-feishu-review-backup-restore.ps1` | pass | no | 0 |
| T08 | Retention/WAL | Windows local | `verify-feishu-review-retention-wal.ps1` | pass | no | 0 |
| T09 | Graceful shutdown | Windows local | `verify-feishu-review-graceful-shutdown.ps1` | pass | no | 0 |
| T10 | Linux full | GitHub Actions | final quality workflow | pending final SHA | no | 0 |
| T11 | Linux race | GitHub Actions | final quality workflow | pending final SHA | no | 0 |
| T12 | Windows review | GitHub Actions | final quality workflow | pending final SHA | no | 0 |
| T13 | PowerShell matrix | GitHub Actions | Ubuntu + Windows | pending final SHA | no | 0 |
| T14 | Node/WeCom | GitHub Actions | npm + Go adapter | pending final SHA | no | 0 |
| T15 | real GitLink read | Real GitLink | `real-gitlink-read.json` | pass | yes | 0 |
| T16 | real public webhook | Real reverse proxy/GitLink | `real-webhook.json` | blocked by environment | yes | 0 |
| T17 | real two-chat isolation | Real Feishu | `real-dual-chat.json` | blocked by environment | yes | 0 |
| T18 | real card/reply | Real Feishu | `real-card-reply.json` | pass | yes | 3 Feishu |
| T19 | real Base | Real Feishu | `real-base.json` | blocked by environment | yes | 0 |
| T20 | real Doc | Real Feishu | `real-doc.json` | blocked by environment | yes | 0 |
| T21 | real Task | Real Feishu | `real-task.json` | blocked by environment | yes | 0 |
| T22 | real claim/deadline/release | Real Feishu | `real-collaboration.json` | pass | yes | 9 Feishu |
| T23 | real common Review | Real Feishu + GitLink | `real-common-review.json` | pass | yes | 1 GitLink |
| T24 | real approve | Real Feishu + GitLink | `real-approve.json` | pass | yes | 1 GitLink |
| T25 | real reject Review | Real Feishu + GitLink | `real-reject.json` | pass | yes | 1 GitLink |
| T26 | real reject and close | Real Feishu + GitLink | `real-refuse.json` | pass | yes | 1 GitLink |
| T27 | real controlled merge | Real Feishu + GitLink | `real-merge.json` | pass | yes | 1 GitLink |
| T28 | real head stale | Real Feishu + GitLink | `real-stale.json` | pass | yes | 0 |

Durations and commit SHAs are stored in the corresponding JSON evidence when a
test is executed. `not executed` is intentionally not rewritten as `pass`.
