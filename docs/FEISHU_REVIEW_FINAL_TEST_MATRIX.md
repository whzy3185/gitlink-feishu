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
| T15 | real GitLink read | Real GitLink | `validate-real-gitlink-review-read.ps1` | not executed | yes | 0 |
| T16 | real public webhook | Real reverse proxy/GitLink | `validate-real-gitlink-webhook.ps1` | not executed | yes | 0 |
| T17 | real two-chat isolation | Real Feishu | `validate-real-feishu-dual-chat.ps1` | not executed | yes | 0 |
| T18 | real card/reply | Real Feishu | `validate-real-feishu-card-reply.ps1` | not executed | yes | 0 |
| T19 | real Base | Real Feishu | `validate-real-feishu-base.ps1` | not executed | yes | 0 |
| T20 | real Doc | Real Feishu | `validate-real-feishu-doc.ps1` | not executed | yes | 0 |
| T21 | real Task | Real Feishu | `validate-real-feishu-task.ps1` | not executed | yes | 0 |

Durations and commit SHAs are stored in the corresponding JSON evidence when a
test is executed. `not executed` is intentionally not rewritten as `pass`.
