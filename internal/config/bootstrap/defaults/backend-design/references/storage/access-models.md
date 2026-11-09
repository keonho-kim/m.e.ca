# Workload-shaped access models

Choose a model from the dominant reads and mutations, not an entity diagram alone. The examples are alternatives; they do not require another database product.

## Hierarchies and relationships

| Representation | Read benefit | Mutation/storage cost |
| --- | --- | --- |
| Parent adjacency + recursive query | Small canonical relation; flexible traversal | Recursive work and cycle/fan-out bounds |
| Materialized path | Prefix/subtree access with suitable indexing | Moving a subtree rewrites descendant paths |
| Closure relation | Direct ancestor/descendant joins | Potentially quadratic relation size; moves update many pairs |
| Cached reachability projection | Fast repeated permission/traversal checks | Revocation freshness, versioning and rebuild |

Use explicit cycle detection and a breadth/work budget as well as depth. A shallow tree can have enormous fan-out. PostgreSQL's [recursive-query guidance](https://www.postgresql.org/docs/current/queries-with.html) describes traversal/cycle and materialization behavior. Recheck permissions at the correct consistency boundary; cached ancestry is not automatically an authorization authority.

For a closure table, distinguish self-pairs and path multiplicity in a DAG. Removing one edge need not remove reachability provided by another path. Keep a tree-only implementation explicit rather than silently treating it as a general graph.

## Time-series and append-heavy workloads

Choose event-time versus ingestion-time semantics, late-arrival window, deduplication identity and retention granularity. Time partitions help only when queries/retention use their boundaries; the active partition can remain the write hotspot. Order/correlation may make compact summary indexes useful, but inspect actual plans and update churn.

Rollups require an owner for late corrections, duplicate input, recomputation and expiry. An average cannot be merged safely by averaging averages without counts/weights. Persist the sufficient aggregate state for the operations the product needs. Approximate distinct/quantile summaries require an explicit error/merge contract and a maintained implementation.

## OLTP, reporting and analytical projections

A row-oriented authority fits small conditional mutations and selective reads. Wide repeated scans may benefit from preaggregation, a column-oriented representation or separate analytical capacity. Compare reduced selected-column I/O against ingestion latency, duplicated storage, refresh/rebuild and another operational owner.

Keep expensive reporting from monopolizing interactive connections/work memory. A read replica provides isolation only within its lag, resource and recovery constraints; it does not change a fundamentally expensive access pattern. Materialize only the query families whose reuse earns the maintenance cost.

## Bulk ingestion and export

Batch by rows, bytes and lock/time budget; choose engine-native bulk interfaces when measurements and transaction semantics justify them. Stage and validate data before publishing it where partial visibility is unacceptable. Define duplicates, rejected rows and the restart checkpoint. Bulk speed gained by dropping required constraints or durability changes the contract.

Estimate peak temporary space, new indexes/WAL and rebuild overlap before choosing a batch size. Verify hierarchy moves/cycles, late time-series events, duplicate imports, aggregation corrections and concurrent foreground load. [Physical models](physical-model.md) own type/row choices; [data lifecycle](data-lifecycle.md) owns retention and cutover.
