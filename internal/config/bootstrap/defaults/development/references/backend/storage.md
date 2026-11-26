# Storage decisions

Select an authority before selecting accelerators. Avoid a default SQLite → PostgreSQL → Redis ladder: deployment, write contention, recovery and access shape determine the choice.

| Need | Read |
| --- | --- |
| Key/type/row boundaries and precision, text and time semantics | [Physical model](storage/physical-model.md) |
| Hierarchies, time-series, aggregation and bulk workloads | [Access models](storage/access-models.md) |
| Embedded/local authority, a single host, predictable write admission | [SQLite](storage/sqlite.md) |
| PostgreSQL plan quality, index choice, joins and write amplification | [PostgreSQL query design](storage/postgres-queries.md) |
| Pools, query memory, VACUUM, replication and retained WAL | [PostgreSQL operations](storage/postgres-operations.md) |
| Cache coherence, Redis memory/durability, hot keys and leases | [Redis and caches](storage/redis.md) |
| Stale fills, L1/L2 generations and admission/eviction choices | [Cache consistency protocol](storage/cache-consistency.md) |
| Snapshot plus changes, deletion, cursors and index-generation cutover | [Projection rebuild](storage/projection-rebuild.md) |
| Concurrent updates, uniqueness, reservations, retries and outbox | [Transactions and consistency](storage/transactions.md) |
| Online schema changes, canonical payloads, retention and deletion | [Data lifecycle](storage/data-lifecycle.md) |

## Compare the actual operating model

| Candidate | Fits when | Cost or boundary to prove |
| --- | --- | --- |
| SQLite | Embedded access and one host can own durable state; serialized writes fit the deadline | Writer contention, checkpoint progress, disk safety and host recovery |
| PostgreSQL | Concurrent transactions, shared network authority and relational access justify a managed server | Pool saturation, write/index/WAL amplification, maintenance and failover semantics |
| Redis | Reconstructible hot data, bounded counters or short-lived coordination need very low latency | Eviction, stale reads, replication loss, fork memory and cache-miss load |
| Object storage | Large immutable bytes should leave transactional rows small | Metadata/object publication, integrity, lifecycle and orphan recovery |
| Dedicated search engine | Retrieval features or scale exceed the relational access path | Another projection, freshness, deletion, authorization and rebuild ownership |

Redis can be an authority only after its persistence, acknowledgement and failover model satisfies the product's loss budget. A broker or cache is not made durable by naming it a database. A database is not too small merely because it is embedded.

For combinations, document authority → projection/update mechanism → visibility delay → stale/missing behavior → rebuild. Put bytes and write amplification beside latency. A read cache that duplicates an already hot database working set may add more memory and invalidation work than useful throughput.

Place the chosen storage work inside a [transactional command](compositions/transactional-command.md), [read model](compositions/read-model.md) or another fitting composition. Storage choice alone does not assign business, transaction or publication ownership. Use the [worked studies](design-studies.md) for concrete cost comparisons.
