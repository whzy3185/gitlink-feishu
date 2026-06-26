# Feishu Permission Matrix

Date: 2026-06-26

GitLink write permission is `No` for every implemented command in this branch.

| Capability | Command | Layer | Needs webhook? | Needs app_id/app_secret? | Needs DocX/Wiki scope? | Needs Base scope? | Needs Task scope? | Needs GitLink token? | Needs GitLink write permission? | Tested locally? | Test result | Known limitation |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| Custom bot test | `feishu +bot-test` | Stable webhook export | Yes for `--send` | No | No | No | No | No | No | Yes | unit/mock; real if webhook env exists | Custom bot only posts to configured chat |
| Workflow card | `feishu +notify` | Stable webhook export | Yes for `--send` | No | No | No | No | No | No | Yes | preview passed | Consumes workflow JSON; no direct Feishu identity routing |
| Weekly report | `feishu +weekly-report` | Stable webhook export | Yes for `--send` | No | No | No | No | No | No | Yes | preview passed | Card is summary-level |
| Owner digest | `feishu +owner-digest` | Stable webhook export | Yes for `--send` | No | No | No | No | No | No | Yes | unit and preview passed | Role-oriented, not personalized |
| Contributor digest | `feishu +contributor-digest` | Stable webhook export | Yes for `--send` | No | No | No | No | No | No | Yes | unit and preview passed | Role-oriented, not open_id routed |
| Bitable schema | `feishu +bitable-schema` | Stable dry-run | No | No | No | No | No | No | No | Yes | preview passed | Does not create tables or views |
| Bitable records | `feishu +bitable-records` | Stable dry-run | No | No | No | No | No | No | No | Yes | preview passed | Summary records, not one row per raw issue/PR |
| Task preview | `feishu +task-preview` | Stable dry-run | No | No | No | No | No | No | No | Yes | preview passed | Local candidates only |
| DocX / Wiki export | `feishu +doc-export` | Experimental Open Platform | No | Yes for `--send` | Yes | No | No | No | No | Mocked; real depends on env | mock passed | App must have scopes and document/folder permission |
| Bitable sync | `feishu +bitable-sync` | Experimental Open Platform | No | Yes for `--send` | No | Yes | No | No | No | Mocked; real depends on env | mock passed | Requires existing tables and `unique_key` field |
| Task create | `feishu +task-create` | Experimental Open Platform | No | Yes for `--send` | No | No | Yes | No | No | Mocked; real depends on env | mock passed | Dedupe is local unique_key only |
| GitLink action gateway | not implemented | Future planning | No | Planned | No | No | No | Planned | Yes | No | not implemented | Requires official authorization model |
