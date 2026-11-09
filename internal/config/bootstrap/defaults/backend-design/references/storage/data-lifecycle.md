# Data lifecycle and amplification

Use while designing schema, retention and projections. Include creation, update, deletion, indexing, backup and restore costs in the data model instead of deferring them to cleanup jobs.

Derive row/key/type decisions with [physical models](physical-model.md). For a backfill or new projection, [snapshot/change-stream joining](projection-rebuild.md) defines the consistent boundary, generation and checkpoint requirements; a monotonic row ID alone is insufficient.

## One canonical payload, intentional representations

Keep frequently queried metadata narrow and hydrate large immutable bodies only when needed. Avoid journaling the same client-supplied history or attachment into a new session on every request. Store an authenticated scoped reference where the authority and recovery contract permit it.

Content-addressed storage can share immutable bytes, but its hash is not authorization. Enforce access through the requesting object/tenant; verify bytes on ingest and handle deletion/reference ownership transactionally. Cross-tenant deduplication can reveal content equality and complicate per-tenant erasure. Randomized encryption also changes deduplication economics.

Separate mutable latest state from retained versions when the product needs both. An append-only snapshot on every small update may multiply unchanged payloads. References/deltas save bytes but make compaction, corruption recovery and historical reads more complex; choose from actual retention and access requirements.

PostgreSQL may compress/store large values out of line through [TOAST](https://www.postgresql.org/docs/current/storage-toast.html). That does not make selecting or repeatedly rewriting a large value free. Measure table, index and auxiliary storage, not only the column's logical length.

## Plan retention as an access path

Define which timestamps/versions control expiry and which legal/product holds override it. Use indexes or [partitions](https://www.postgresql.org/docs/current/ddl-partitioning.html) that make bounded deletion feasible. A global periodic full scan can be more expensive than the feature it supports.

Deletion must cover authority, caches, search/vector projections, export artifacts and recoverable copies according to the documented retention contract. Mark deletion with an ordered version/tombstone where delayed updates could resurrect data. Cleanup must distinguish unreachable objects from objects whose publication/commit is merely uncertain. Measure backlog and oldest eligible item, not only deletion count.

## Change schema without permanent parallel models

Use an online transition only when rolling deployment or uninterrupted service requires it and project policy permits it. Establish the authoritative writer and a temporary expand/backfill/verify/cutover/contract sequence. Backfill in bounded key ranges with resumable checkpoints and compare-and-set/version rules so old data cannot overwrite concurrent new writes.

Stripe's [migration case study](https://stripe.com/blog/online-migrations) demonstrates staged changes at its scale. Do not infer that independent best-effort dual writes are atomic. Cross-store changes need ordered capture/outbox or explicit reconciliation. Set an end condition that removes the old path; projects that reject obsolete formats should reject them clearly rather than silently recreating user data.

Budget index creation/rebuild, extra WAL, temporary copies, replica lag and rollback space before cutover. PostgreSQL's [`CREATE INDEX CONCURRENTLY`](https://www.postgresql.org/docs/current/sql-createindex.html) has transaction/locking/failure constraints and can leave an invalid index; it is not cost-free online DDL.

Verify restore to a selected old point with matching schema, keys and projection rebuild procedures. Track logical/physical bytes per useful object and write bytes per business mutation, including maintenance. A smaller live table is not proof that retained WAL, backups and temporary artifacts shrank.
