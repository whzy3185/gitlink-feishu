# Feishu Permission Matrix

Date: 2026-06-26

GitLink write permission is `No` for every implemented command in this branch.

| Capability | Command | Layer | Needs webhook? | Needs app_id/app_secret? | Needs DocX/Wiki scope? | Needs Base scope? | Needs Task scope? | Needs GitLink token? | Needs GitLink write permission? | Tested locally? | Test result | Known limitation |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| Custom bot test | `feishu +bot-test` | Stable webhook export | Yes for `--send` | No | No | No | No | No | No | Yes | unit/mock and real send passed | Custom bot only posts to configured chat |
| Workflow card | `feishu +notify` | Stable webhook export | Yes for `--send` | No | No | No | No | No | No | Yes | preview and real send passed, including zh-CN | Consumes workflow JSON; no direct Feishu identity routing |
| Weekly report | `feishu +weekly-report` | Stable webhook export | Yes for `--send` | No | No | No | No | No | No | Yes | preview and real send passed | Card is summary-level |
| Owner digest | `feishu +owner-digest` | Stable webhook export | Yes for `--send` | No | No | No | No | No | No | Yes | unit, preview, and real send passed, including zh-CN | Role-oriented, not personalized |
| Contributor digest | `feishu +contributor-digest` | Stable webhook export | Yes for `--send` | No | No | No | No | No | No | Yes | unit, preview, and real send passed, including zh-CN | Role-oriented, not open_id routed |
| App diagnostics | `feishu +app-check` | Stable diagnostics | No | Yes for `--remote` | No | No | No | No | No | Yes | unit, mock remote, and real remote passed | Remote mode only gets tenant_access_token; no writes |
| Doc diagnostics | `feishu +doc-check` | Stable diagnostics | No | Yes for `--remote` | Yes for Wiki node read | No | No | No | No | Yes | local and real remote diagnostics passed | Does not prove edit permission without append |
| Bitable diagnostics | `feishu +bitable-check` | Stable diagnostics | No | Yes for `--remote` | No | Yes for remote search | No | No | No | Yes | unit, mock remote, and five-table real remote passed | Checks table access and unique_key search; does not create fields |
| Task diagnostics | `feishu +task-check` | Stable diagnostics | No | Yes for `--remote` | No | No | No; token check only | No | No | Yes | mock remote and real remote passed with expected warnings | Does not create tasks; project/section still next-stage |
| Bitable schema | `feishu +bitable-schema` | Stable dry-run | No | No | No | No | No | No | No | Yes | preview passed | Does not create tables or views |
| Bitable records | `feishu +bitable-records` | Stable dry-run | No | No | No | No | No | No | No | Yes | preview passed | Summary records, not one row per raw issue/PR |
| Task preview | `feishu +task-preview` | Stable dry-run | No | No | No | No | No | No | No | Yes | preview passed, including zh-CN | Local candidates only |
| DocX / Wiki export | `feishu +doc-export` | Experimental Open Platform | No | Yes for `--send` | Yes | No | No | No | No | Yes | mock, preview, and real DocX append passed, including zh-CN | App must have scopes and document/folder permission |
| Bitable sync | `feishu +bitable-sync` | Experimental Open Platform | No | Yes for `--send` | No | Yes | No | No | No | Yes | mock, preview, one-table real write, and split-table real write passed | Requires existing tables and compatible fields; CLI does not create production tables/views |
| Task create | `feishu +task-create` | Experimental Open Platform | No | Yes for `--send` | No | No | Yes | No | No | Yes | mock, preview, and real create passed | Dedupe is local unique_key only; project/section/assignee/follower support is next-stage boundary |
| GitLink action gateway | not implemented | Future planning | No | Planned | No | No | No | Planned | Yes | No | not implemented | Requires official authorization model |
