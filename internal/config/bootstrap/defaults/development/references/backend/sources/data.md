# Database primary sources

Vendor manuals describe their configured product, not a generic SQL guarantee. Use the manual for the actual deployment version; a moving `current` alias does not pin behavior. These pointers support the [storage map](../storage.md).

## SQLite

| Source | Use for |
| --- | --- |
| [Write-ahead logging](https://sqlite.org/wal.html) | Writer serialization, reader/checkpoint interaction, WAL growth and filesystem limits |
| [Isolation](https://sqlite.org/isolation.html) | Snapshot upgrades and transaction restart decisions |
| [Optimizer overview](https://sqlite.org/optoverview.html) | Composite access paths, covering indexes and query transformations |
| [PRAGMA reference](https://sqlite.org/pragma.html) | Connection/configuration scope and version-sensitive optimize behavior |
| [Online backup API](https://sqlite.org/backup.html) | Engine-supported consistent snapshots and incremental backup |
| [FTS5](https://sqlite.org/fts5.html) | Lexical retrieval, tokenizers, ranking and external-content synchronization |

## PostgreSQL

| Source | Use for |
| --- | --- |
| [Transaction isolation](https://www.postgresql.org/docs/current/transaction-iso.html) | Snapshot behavior, anomalies and complete-transaction retry |
| [Numeric types](https://www.postgresql.org/docs/current/datatype-numeric.html) | Precision, range and physical representation choices |
| [Collations](https://www.postgresql.org/docs/current/collation.html) | Equality/sort semantics and version-sensitive text behavior |
| [Sequences](https://www.postgresql.org/docs/current/functions-sequence.html) | Allocation semantics distinct from transaction commit ordering |
| [WITH/recursive queries](https://www.postgresql.org/docs/current/queries-with.html) | Hierarchy traversal, cycle handling and CTE materialization |
| [Explicit locking](https://www.postgresql.org/docs/current/explicit-locking.html) | Row/table/advisory ownership, lock scope and deadlocks |
| [Index types](https://www.postgresql.org/docs/current/indexes-types.html) | B-tree, GIN, GiST, SP-GiST and BRIN access patterns |
| [Index-only scans](https://www.postgresql.org/docs/current/indexes-index-only-scans.html) | Covering payload and visibility-map requirements |
| [Planner statistics](https://www.postgresql.org/docs/current/planner-stats.html) | Selectivity estimates and correlated-column statistics |
| [Using EXPLAIN](https://www.postgresql.org/docs/current/using-explain.html) | Plans, actual execution and measurement caveats |
| [Resource consumption](https://www.postgresql.org/docs/current/runtime-config-resource.html) | Per-operator work memory and concurrent maintenance costs |
| [Routine vacuuming](https://www.postgresql.org/docs/current/routine-vacuuming.html) | Dead rows, visibility, freeze safety and maintenance |
| [Table partitioning](https://www.postgresql.org/docs/current/ddl-partitioning.html) | Pruning, retention, constraints and management cost |
| [Row security](https://www.postgresql.org/docs/current/ddl-rowsecurity.html) | Policy enforcement and privileged-role exceptions |
| [Continuous archiving/PITR](https://www.postgresql.org/docs/current/continuous-archiving.html) | Recovery chains, base backups and WAL |
| [CREATE INDEX](https://www.postgresql.org/docs/current/sql-createindex.html) | Partial indexes, concurrent builds and failure constraints |
| [SELECT](https://www.postgresql.org/docs/current/sql-select.html) | Ordering, pagination and queue-oriented `SKIP LOCKED` |
| [Logical decoding](https://www.postgresql.org/docs/current/logicaldecoding-explanation.html) | Slots, retained state and duplicate replay after crash |
| [Standby servers](https://www.postgresql.org/docs/current/warm-standby.html) | Replication lag, acknowledgement and failover tradeoffs |
| [HOT](https://www.postgresql.org/docs/current/storage-hot.html) | Indexed-column changes, page space and write amplification |
| [JIT decisions](https://www.postgresql.org/docs/current/jit-decision.html) | Compilation cost versus long CPU-heavy queries |
| [TOAST](https://www.postgresql.org/docs/current/storage-toast.html) | Compression and out-of-line large-value representation |
| [Text-search indexes](https://www.postgresql.org/docs/current/textsearch-indexes.html) | Inverted indexes and rechecks for text retrieval |
| [pg_trgm](https://www.postgresql.org/docs/current/pgtrgm.html) | Trigram similarity and substring/fuzzy access paths |

## Connection boundary

[PgBouncer's feature map](https://www.pgbouncer.org/features.html) identifies which session assumptions change under transaction/statement pooling, including prepared-plan support conditions. Check pooler and driver versions together; a pooling-mode label is not enough.
