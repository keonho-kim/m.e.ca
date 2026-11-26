# Study: workspace records and large bodies

Fictional assumptions: a desktop/shared workspace shows 40 metadata rows per page; each body averages 64KiB and each projected metadata row 256B. Peak read arrivals are assumed 300/s. Bodies are needed only on detail. These values are arithmetic inputs, not measured capacity.

## Schema and responsibility

```text
records(tenant, id, status, updated_at, version, body_ref)
  primary key: tenant + id
  candidate index: tenant + status + updated_at DESC + id DESC
bodies(tenant, immutable_ref, bytes, content_version)
commands(tenant, operation, key, fingerprint, outcome_ref) when retry identity is needed
```

The record module owns updates and query projections; the persistence adapter owns SQL/transactions; detail access authorizes the record before resolving its body. A body hash/reference is not access authority. The index is a hypothesis for the actual filter/order, not a required universal index.

## Compare viable arrangements

| Arrangement | Why it may fit | Cost or trigger to reject it |
| --- | --- | --- |
| One-host SQLite + direct projections | Embedded authority, no network DB hop | Writer contention, backup/host recovery or multi-host authority requirement |
| PostgreSQL + direct projections | Several service instances share transactional authority | Pool/maintenance/network cost; verify active concurrency |
| PostgreSQL + bounded result cache | Proven repeated expensive reads with acceptable freshness | Invalidation, scope/version state, L1/L2 duplication and cold-miss load |

Do not add Redis solely because reads are frequent. First stop hydrating bodies for a metadata-only result. If the old path materialized all 40 bodies, it moved about 2.5MiB of body data per request; the selected metadata is about 10KiB. At the assumed peak that is roughly 750MiB/s versus 2.93MiB/s of logical selected bytes. Actual disk/network/allocation savings depend on caches, encoding, page layout and the real baseline.

## Operation flow

List: parse tenant-scoped filter/cursor → indexed bounded metadata query → response. The cursor binds filters, ordering and tenant; detail separately authorizes and loads the selected immutable body. Updates use a current version guard; related command identity/outcome commits with the mutation if the API promises deduplication.

For a new body, stage bytes under private immutable identity, commit a valid metadata reference and reconcile uncertain/orphan publication. A single-database body table may keep publication atomic; external object storage changes that recovery boundary. Keep unchanged bodies out of every event/checkpoint.

## Design decision and verification

Start with the [feature module](../compositions/service-module.md) and a direct read projection. Select the authority from host/writer/recovery requirements, then inspect query cost before adding a cache. Estimate active memory as concurrent operations × selected/decoded/output representations, not peak QPS alone.

Test deep pages, changing sort fields, unauthorized body references, concurrent update, largest body and cancellation. Measure scans, hydrated bytes, round trips, writer/pool wait and retention after load. Add cache/projection machinery only if the verified workload earns its lifecycle. See [physical model](../storage/physical-model.md) and [cache protocol](../storage/cache-consistency.md).
