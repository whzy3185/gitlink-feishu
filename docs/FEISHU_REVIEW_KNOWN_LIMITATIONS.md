# Feishu Review known limitations

1. The service is single-instance. The instance lock prevents two local writers
   but is not a distributed leader election mechanism.
2. SQLite is the only state store. Large installations need capacity testing,
   retention planning and regular verified backups.
3. GitLink API fields and webhook delivery contracts can vary by deployment.
   Stage Six real read and delivery validation has not been executed.
4. Feishu Channel, Card, Reply, Base, Doc and Task validators are implemented
   but were not run in Stage Six. Their evidence files remain `not_executed`.
5. Card/Reply/Doc/Task ambiguous remote results require manual reconciliation;
   only Base has a safe unique-key lookup path.
6. The first acknowledgement is best effort. Durable final operations use the
   outbox, leases and reconciliation paths.
7. Base has no assumed atomic business-key upsert; concurrent create races can
   produce a multiple-match reconciliation case.
8. Doc snapshots are append-oriented and are not automatically deleted.
9. Task cleanup does not delete remote tasks. Archived behavior depends on
   whether a task already exists.
10. GitLink common Review write-back is a separate controlled capability and is
    not enabled by Stage Six validation.
11. Full user OAuth and automatic identity binding are not complete.
12. Enterprise WeChat has regression tests and an adapter, but no production
    end-to-end acceptance is claimed.
13. Hosted CI status can only be known after the final commit is pushed; the
    committed matrix marks those final-SHA jobs pending.
