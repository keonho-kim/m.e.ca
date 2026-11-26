# PostgreSQL: pools, memory and maintenance

Use when determining whether a query design remains viable under concurrency, updates and failure. Pool size, query memory, vacuum and replication consume one shared capacity budget.

## Pool admission is part of latency

Separate pool wait, connection establishment, server execution and result consumption. Choose active database concurrency from useful throughput and latency at saturation; increasing connections can increase contention and memory while reducing completed work. Reserve capacity for maintenance and operational access. Apply bounded queues and deadlines before acquiring a connection.

Release transactions/rows before network callbacks or slow client delivery. Long exports should use bounded pages or a managed snapshot/job with explicit age and cancellation. Idle transactions retain resources even when CPU looks quiet.

Transaction pooling changes session assumptions. Session advisory locks, `LISTEN`, temporary state and prepared statements need deliberate driver/pooler support; use transaction-local tenant state and verify cleanup. PgBouncer's [feature map](https://www.pgbouncer.org/features.html) documents the differences. Do not assume every pooling mode preserves arbitrary `SET` state.

## Calculate memory at the operation level

`work_mem` is not one allowance per connection: one query may run several memory-consuming operators and parallel workers, alongside other sessions. Hash operations have additional multipliers. Budget those workspaces, maintenance workers, shared buffers, OS cache, connection overhead and native extensions together. Use conservative simultaneous-work estimates and inspect spill behavior. See [resource consumption](https://www.postgresql.org/docs/current/runtime-config-resource.html).

Prefer operation-local tuning for a justified expensive path to a global setting that makes all concurrent queries dangerous. A spill may be an acceptable latency/storage tradeoff; raising every workspace can convert it into an OOM incident. Put temporary-space limits and observability beside memory budgets.

## Design for update churn

Track dead tuples, oldest transaction/snapshot, vacuum progress, visibility, freeze age, index/table growth and long waits. Autovacuum is correctness work as well as cleanup; `VACUUM FULL` rewrites/locks and should not become a routine request-path response to bloat. [Routine vacuuming](https://www.postgresql.org/docs/current/routine-vacuuming.html) explains the different duties.

Partition by a real pruning/retention benefit. Excessive partitions add planning/management cost, and uniqueness/foreign-key constraints need inspection. Plan partition creation before the next interval and verify pruned plans. See [table partitioning](https://www.postgresql.org/docs/current/ddl-partitioning.html).

## Bound replication and change capture

Monitor lag in both time and bytes, archive failures and retained WAL from consumers/slots. A disconnected CDC consumer can retain storage while application queries remain fast. Logical decoding can resend after restart, so consumers need idempotent application/checkpoints; see [logical decoding](https://www.postgresql.org/docs/current/logicaldecoding-explanation.html).

Decide whether to retain, invalidate or rebuild a stalled consumer before disk exhaustion. Dropping a live slot can force resynchronization and must have an owner. Failover also requires connection recovery and read-freshness policy; acknowledged loss depends on replication configuration, not just replica count. Validate maintenance plus foreground load, reconnect storms, failed archives and largest-tenant skew before claiming capacity.
