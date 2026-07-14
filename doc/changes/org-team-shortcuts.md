# org team shortcuts

This change expands the `org` shortcut group from basic organization lookup into an operational workflow for team and membership management.

The updated command set keeps `org +list`, `org +info`, and `org +create`, and adds stronger organization administration coverage:

- `org +members` now supports `--all`, `--team`, `--login`, and `--keyword`, and automatically fetches all pages before applying filters.
- `org +teams` lists organization teams with `--authorize`, `--unit`, `--keyword`, and `--include-users` support, returning normalized team summaries.
- `org +team-create` creates a team and supports `--dry-run` for request preview.
- `org +member-remove` removes an organization member by `--user-id` or resolves `--login` automatically, with `--dry-run` support.

The output shape is normalized for automation use. Member results now include matched counts, team summaries, and consistent user fields. Team results expose permission-level aggregation, unit information, and optional normalized user details.

The command examples in `README.md` were extended so the new team and member management flows are discoverable from the main project documentation.

Validation:

```bash
go test ./shortcuts/org/...
go test ./shortcuts/...
go test ./...
go build ./...
```
