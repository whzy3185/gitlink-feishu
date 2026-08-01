# Round 2 gateway deployment

This directory provides a conservative single-instance systemd example. It
starts the Feishu long connection, durable SQLite job store, and a loopback-only
GitLink webhook listener. It does not enable GitLink Review writes, Feishu
Base/Doc/Task writes, or the external Agent runner.

1. Build `gitlink-cli` and install it as `/opt/gitlink/bin/gitlink-cli`.
2. Create the `gitlink-review` system account and `/var/lib/gitlink-review`.
3. Copy `gitlink-feishu.env.example` to `/etc/gitlink/feishu.env`, replace the
   placeholders, and set mode `0600`.
4. Copy `docs/examples/feishu-review-bindings-v2.json` to
   `/etc/gitlink/review-bindings.json`, replace IDs, and set mode `0600`.
5. Install the unit, run `systemctl daemon-reload`, then start it.
6. Put the loopback webhook behind a TLS reverse proxy only when GitLink must
   reach it. Preserve the raw body and signature headers.

Before enabling any optional writer, complete the matching evidence gate in
`docs/ROUND2_PLATFORM_EVIDENCE_CHECKLIST.md`. Do not add write flags to the base
unit merely to simplify a demo.
