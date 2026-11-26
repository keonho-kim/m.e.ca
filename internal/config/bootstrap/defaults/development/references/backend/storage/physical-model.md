# Physical models as execution decisions

Choose field types, keys and row boundaries from invariants and access/update patterns. Schema choices determine bytes compared, copied, indexed and rewritten before any application optimization is possible.

| Decision | Compare | Contract/cost to verify |
| --- | --- | --- |
| Identifier | Bounded integer, native UUID, external opaque ID | Width across indexes/references, generation/coordination, locality and client precision |
| Tenant identity | Scoped composite key or globally unique ID plus tenant | Cross-tenant constraints and leading access predicates |
| Numeric value | Fixed scale integer or exact decimal; float only where approximation fits | Range, rounding, overflow, arithmetic cost and serialization precision |
| Time | Instant, calendar date/time, duration, ordering position | Zone/clock semantics, indexing and expiry/replay behavior |
| Text equality | Original display value plus canonical comparison representation | Normalization/collation, uniqueness, search behavior and migration |
| Variable fields | Typed columns, JSON/document or related rows | Constraints, path indexes, update granularity, hydration and schema evolution |
| Large content | Inline, separate immutable row, object storage | Transaction boundary, extra lookup, copy/decrypt cost and orphan recovery |

An ID is not an access token or a commit-order guarantee. Sequential/ordered keys can improve some locality while concentrating writes; random keys have different cache/index behavior. Evaluate the actual engine and distribution rather than prescribing one UUID generation version everywhere.

Exact decimals and scaled integers have different range/arithmetic costs; choose the business precision and rounding policy first. Preserve values across database, driver and API encodings, especially clients with narrower exact integer ranges. See [PostgreSQL numeric types](https://www.postgresql.org/docs/current/datatype-numeric.html).

Store a scheduled user's zone/calendar intent where required; a UTC instant alone cannot reconstruct every recurring local-time rule. Use monotonic elapsed time for local duration measurement when supported; persistent leases/expiry need a distributed clock/fencing policy. Domain event time, ingestion time and commit position answer different questions.

Text normalization must agree across unique constraints, query compilation, cache identity and encrypted equality indexes. Do not normalize display content or secrets by habit. Compare deterministic/non-deterministic collation behavior and plan version changes; [PostgreSQL collation](https://www.postgresql.org/docs/current/collation.html) documents engine-specific semantics.

## Bound row and update amplification

Sketch the hot query projection and hot update before finalizing the entity. A 20-byte status change should not force application decoding/copying of a 2MiB immutable document. Splitting them may reduce working bytes but adds joins/references and publication ownership. Keep fields together when atomic access/locality benefits outweigh splitting.

Estimate `rows × row bytes + index entries + auxiliary data`, then include MVCC versions, WAL, replicas and backups. Distinguish logical payload size from physical representation and resident working set. Database compression does not remove application materialization cost.

Normalize authoritative rules and relationships first; denormalize specific query projections when the saved work justifies update/rebuild consistency. Duplicate a small stable display field only with an owner/version policy, not by copying the whole aggregate into every table/event.

Test boundary precision, collations, null/empty semantics, maximum key/body sizes, concurrent uniqueness, large-field updates and deletion. Record which query/constraint consumes each index. Use [access models](access-models.md) for hierarchical/time-series/aggregate structures and [query plans](postgres-queries.md) to validate the physical result.
