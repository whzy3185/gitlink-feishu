## PR list supports direct lookup by PR number

`pr +list` previously only exposed keyword-based search, which made it awkward
to jump to a known PR from the web UI or from review notes. This change adds
`--number` / `-n` and a compatibility alias `--id` / `-i` to the list command.

When a PR number is provided, the CLI now reads that PR through the dedicated
detail endpoint and wraps the result into the usual list payload shape. This
keeps the output stable for automation while making exact-number lookup work
even when the target PR is not on the current list page.
