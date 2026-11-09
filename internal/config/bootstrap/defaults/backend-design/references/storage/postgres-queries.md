# PostgreSQL: query and index design

Design from the query families and mutation rate. Record predicates, tenant scope, sort order, selected columns, expected selectivity/skew and consistency. One index per column is rarely a useful strategy for a compound workload.

## Select an access path

| Access shape | Candidate | Cost or limitation |
| --- | --- | --- |
| Equality/range and ordered pages | B-tree with matching key order | Wider keys and extra indexes increase writes and cache demand |
| Active subset or sparse condition | Partial index | Predicate must be implied by the actual planned query |
| Small returned payload on stable rows | Covering index with `INCLUDE` | Index growth; heap visibility can still require fetches |
| Arrays/JSON/text membership | Suitable GIN operator class | Update/maintenance overhead; operator coverage varies |
| Large physically correlated ranges | BRIN | Lossy candidate ranges need rechecks; correlation matters |
| Spatial/range/nearest-neighbor operators | Suitable GiST/SP-GiST class | Operator-specific behavior, rechecks and index maintenance |

These are starting hypotheses from PostgreSQL's [index types](https://www.postgresql.org/docs/current/indexes-types.html). A sequential scan is sensible when most rows are needed. Do not force an index simply because it exists.

An index-only scan also depends on visibility-map state; including every output column does not guarantee zero heap access. Verify heap fetches on realistic update churn using [index-only scan guidance](https://www.postgresql.org/docs/current/indexes-index-only-scans.html).

## Read the plan's multipliers

Compare estimated and actual rows, loops, rows removed, join input size, sort/hash spills, buffer reads and total output bytes. A small inner lookup repeated a million times may dominate a plan. A fast first page does not establish deep-page cost. PostgreSQL's [EXPLAIN guide](https://www.postgresql.org/docs/current/using-explain.html) describes interpretation and execution caveats.

`EXPLAIN ANALYZE` executes the statement. Use a controlled workload/replica or authorized test environment for expensive queries and mutations; rollback does not reverse arbitrary external effects or sequence consumption.

Fix cardinality estimation before inventing application caches. Keep statistics representative; correlated columns may need extended statistics rather than a global increase to every statistics target. Compare hot and cold parameter values and parameter-sensitive prepared plans. See [planner statistics](https://www.postgresql.org/docs/current/planner-stats.html).

## Reduce materialized work

Filter and project before expanding rows or decoding large values. Compare `EXISTS`, joins and bounded aggregation by actual plans; replacing every join with application calls can create N+1 round trips and inconsistent snapshots. Paginate eligible IDs before expensive detail joins only when that preserves filtering, ordering and authorization. Count totals separately when the product does not need an exact count on every page.

For CPU-heavy analytics, test parallel execution/JIT against setup and memory costs. JIT can cost more than it saves for short OLTP queries; do not copy analytical thresholds into an interactive service. See [when to JIT](https://www.postgresql.org/docs/current/jit-decision.html).

## Include the write side

Estimate index bytes per row, inserted/updated entries, WAL, vacuum and rebuild headroom. HOT updates depend on indexed-column changes and page space; frequently indexed mutable fields can defeat the benefit. Test fillfactor against storage and read cost using [HOT documentation](https://www.postgresql.org/docs/current/storage-hot.html). Every index needs a real read/constraint consumer and a removal criterion.
