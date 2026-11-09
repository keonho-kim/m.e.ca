# Recovery and production-change cost

Use while selecting durability, replication, retention and rollout strategy. Define tolerated loss (RPO), time to useful service (RTO) and the invariants that recovery must preserve. Recovery throughput and spare storage are part of capacity design.

## A replica is not a backup

Replication can copy deletion/corruption and retain an acknowledged-loss window depending on configuration. Backups need an independently recoverable history, integrity checks, key access and a tested restore path. PostgreSQL [continuous archiving/PITR](https://www.postgresql.org/docs/current/continuous-archiving.html) requires a valid base plus the necessary WAL chain; do not declare recovery possible merely because archive files exist.

For SQLite, use an engine-supported consistent snapshot such as its [backup API](https://sqlite.org/backup.html); copying a live main file alone does not capture arbitrary WAL state. Redis persistence/restart guarantees depend on its [RDB/AOF policy](https://redis.io/docs/latest/operate/oss_and_stack/management/persistence/). State each system's actual durable boundary.

## Restore the application, not just the database

Inventory schema/code versions, encrypted data/key versions, object manifests/blobs, authoritative permission state and external command identities. Rebuildable indexes/caches need a correct source and a bounded rebuild process. Restore an old authority into isolation before making it visible; otherwise old event consumers or workers can replay external effects.

Preserve deletion/retention rules when historical data returns. A backup containing data that later became ineligible needs the documented reconciliation/expiry policy. Missing/corrupt/unsupported data must not silently become empty successful state.

Test an older restore point, lost key permissions, missing WAL/object, incomplete snapshot and a target with production-like size. Measure verification, download/decrypt/decompress, replay, index rebuild, warm-up and final readiness separately. A restore command exiting successfully is not a measured RTO or proof of application correctness.

## Failover needs a fence and capacity

Prevent a former primary/worker from continuing authoritative writes after promotion. Reconnect pools deliberately; bound retry/recovery traffic and resolve uncertain commands before replay. Account for replica lag and stale-read routing. Reserve capacity for catch-up plus serving, not only the healthy steady state.

Use failure cells or pre-existing spare capacity where justified by the blast-radius objective; [Amazon's static-stability design](https://aws.amazon.com/builders-library/static-stability-using-availability-zones/) illustrates serving through impaired control dependencies. Validate the actual failure domain rather than assuming all instances/zones fail independently.

## Make change reversible where possible

Canary representative traffic against a meaningful control with explicit stop criteria for correctness, latency, memory, storage growth and error budget. Shared backends can contaminate both control and canary; [Google's canarying chapter](https://sre.google/workbook/canarying-releases/) discusses that limitation. A binary rollback cannot automatically undo committed schema/data/external effects.

For schema/index/key transitions, plan temporary duplication, locks, WAL, rebuild space and the point after which rollback requires forward repair. Use bounded resumable backfills and verify equivalence before removing the previous authority/path. Keep operational approvals tied to the requested environment and actual mutation; a skill does not authorize production experiments or deletion.

The deliverable should name recovery owners, inputs, safe stop points and last observed exercise results. Label untested paths and estimates explicitly. Do not advertise availability or data-loss guarantees from architecture diagrams, build success or replica count.
