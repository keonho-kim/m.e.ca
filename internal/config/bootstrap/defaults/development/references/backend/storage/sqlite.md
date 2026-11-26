# SQLite: concurrency and physical cost

Choose SQLite when one host can own the authority and serialized writes meet the workload. Measure transaction residence and contention, not only inserts/s on one connection. WAL permits reader/writer overlap but still has one writer; long readers can prevent checkpoint completion. It is not a multi-host network-filesystem coordination mechanism. See [WAL](https://sqlite.org/wal.html).

## Design write ownership

- Keep write transactions short and free of network calls, waiting subscribers, user interaction or large serialization work. Prepare deterministic input before acquiring the writer.
- An application write queue can smooth bursts and provide fairness, but requires bounded count/bytes, admission cancellation and shutdown ownership. It coordinates only participating processes; the database remains the cross-process authority.
- A known read-then-write operation may benefit from `BEGIN IMMEDIATE` to claim the writer before reading. A stale WAL snapshot that cannot upgrade requires ending/restarting the transaction and recomputing its decision. Extending a busy timeout does not refresh it. Distinguish contention from corruption, I/O and full-disk errors. [Isolation](https://sqlite.org/isolation.html) explains this boundary.
- Retry only a known uncommitted, repeatable database unit. Resolve an uncertain commit before replaying external effects. Keep the total retry budget inside the caller's deadline.

## Budget connections and WAL

Choose read concurrency from workload and memory, rather than CPU count alone. Connection-local caches and prepared statements multiply with the pool. Verify connection-local settings on every actual connection, including foreign-key enforcement and busy handling; check that important PRAGMAs took effect because unknown ones may be ignored. Select durability and checkpoint behavior deliberately using the [PRAGMA reference](https://sqlite.org/pragma.html).

Do not hold a cursor/read transaction while streaming to a slow client. Fetch bounded data or establish an explicit snapshot/export job. Monitor oldest-reader age, WAL bytes, checkpoint progress, writer wait, commit latency and filesystem free space. A checkpoint that reports incomplete work is different from corruption. Avoid forcing blocking truncation on every request or deleting live WAL files to reclaim space.

## Make the access path cheap

Use a composite index matching scope, predicates and ordering; return only the selected columns. Apply pagination before hydrating blobs. Covering indexes can avoid table lookup but increase write work and file size. Avoid redundant prefix indexes unless uniqueness, collations or another actual query justify them. Check plans and maintain statistics; use the version-appropriate `PRAGMA optimize` behavior. The [optimizer overview](https://sqlite.org/optoverview.html) describes index constraints and planning tradeoffs.

Normalize authoritative fields used by filtering and integrity instead of repeatedly parsing whole JSON payloads. Separate large immutable bodies from small mutable rows when access frequency warrants it. Evaluate `WITHOUT ROWID` only for the actual key/storage pattern, not as a blanket speed switch.

## Verify the degraded state

Use the [online backup API](https://sqlite.org/backup.html) or another engine-supported consistent snapshot. Copying only a live main file can omit committed WAL state. Exercise two writers, stale read upgrade, canceled admission, long reader, failed checkpoint, full disk and restore. Compare p99 write latency, peak memory, WAL growth, rows/bytes visited and physical file growth under the same workload.
