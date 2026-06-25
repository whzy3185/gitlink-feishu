## PR list output now shows the user-facing PR number

`pr +list` already returned the GitLink PR sequence as `index`, but the default
table output did not make that value easy to spot. This change copies the same
value into a stable `number` field during list normalization and prioritizes the
`number` column in table rendering.

As a result:

- `gitlink-cli pr +list --format table` shows the PR number in a dedicated
  leading column.
- JSON and YAML output also include `number`, making the list output align with
  `pr +view --id <number>` semantics and with the PR number shown in the web UI.
