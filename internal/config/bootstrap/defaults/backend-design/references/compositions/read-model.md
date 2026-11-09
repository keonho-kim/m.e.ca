# Query projection and optional cache

Choose when the representation needed to read efficiently differs from the representation that enforces mutations. Start with a scoped SQL projection in the same module/database. Separate command/query functions do not automatically require full CQRS, a second database or event sourcing. [Fowler's CQRS discussion](https://martinfowler.com/bliki/CQRS.html) explains the broader model separation and complexity risk.

## Three arrangements to compare

```text
A  caller → query function → indexed authority → narrow result
B  caller → query function → local materialized projection → narrow result
   transaction updates authority + projection together
C  caller → query function → cache/search projection
   authority commit → outbox/CDC → idempotent projector → versioned view
```

Choose A unless query cost/shape or isolation justifies B/C. B adds local write work; C adds delivery lag, duplicated storage and rebuild ownership. The query contract states the supported freshness. Current authorization must still apply to projected content, snippets and cached results.

| Owner | Contract |
| --- | --- |
| Query module | Scoped predicates/order, page budget and required representation |
| Authority | Business state and versions, including deletion |
| Projector | Ordered/idempotent application, checkpoint and rebuild generation |
| Cache | Admission, immutable key/value lifetime and stale behavior |
| Response adapter | Final projection/encoding; no hidden N+1 hydration |

## Share only useful immutable work

Memoize deterministic parsing/calculation by content identity plus relevant version. Cache-aside reads a separately owned source and needs invalidation/freshness. Flyweight/Type Object can share immutable descriptors across many instances; instance state remains separate. Prototype copying must specify which values are deep-copied and must not clone transactions, locks or file handles.

A cache wrapper is a behavioral boundary, not a transparent decoration: it changes freshness, errors and retained bytes. Its public contract must make those changes acceptable. Consult [cache consistency](../storage/cache-consistency.md) before adding L1/L2 layers.

## Query and rebuild sequence

Parse/filter/authorize → select bounded eligible IDs/fields → hydrate selected large data only if needed → encode. Keep cursor query/scope identity and order stable. For projection creation, use a consistent snapshot/change position and monotonic source versions; an auto-increment row ID alone is not a safe committed watermark. [Projection rebuild](../storage/projection-rebuild.md) owns that protocol.

Event sourcing means the event log is the authoritative history from which state is derived; an outbox/audit log beside authoritative rows does not make the service event-sourced. Adopt event sourcing only for a real historical/reconstruction requirement and budget replay, schema evolution, snapshots and deletion.

## Cost and verification

Compare scanned/hydrated bytes and round trips saved with projection writes, index/cache bytes, maintenance and cold-rebuild cost. A hot DB page cache may already eliminate the expensive I/O; a network cache can add a hop without useful savings.

Test a source update while a cache fill is running, deleted objects, restrictive tenants, a stalled projector and a generation swap. Verify the allowed staleness and denial behavior, not only response equality when the system is idle. Remove the extra representation if its measured benefit or isolation need disappears.
