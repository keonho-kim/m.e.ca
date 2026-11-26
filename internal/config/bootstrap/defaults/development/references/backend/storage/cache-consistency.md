# Cache admission, fills and version ownership

Use when the source/cache relationship is chosen. Define freshness first: immutable, bounded-stale, read-your-write or authoritative current state. Each requires a different protocol; TTL alone is not proof of immediate freshness.

## A stale-fill schedule

```text
reader starts loading source object v40
writer commits v41 and invalidates the cache
reader finishes and installs v40 after invalidation
```

A version-aware fill can prevent this only with a complete protocol. Keep a monotonic per-object generation/watermark; atomically compare and install a fill against it. A writer's invalidation at v41 raises the watermark and removes older cached data; the later v40 fill is rejected. A separate client GET then SET is racy.

That watermark must survive for the maximum eligible old fill/event lifetime. If it can disappear through eviction, expiry or reset, treat its state as unknown and re-establish coherence; do not accept an old fill merely because no newer marker exists. Bound fill lifetimes and generation retention together. Cross-store invalidation still has delivery delay, so this mechanism alone does not promise immediate read-after-write across a database and Redis.

For read-your-write, propagate a trusted minimum source version and serve only a compatible cache result or an authority read that can meet it. For strict current-state operations, use an authoritative consistency boundary. Pub/Sub disconnect handling and L1 flushing follow [Redis client-cache guidance](https://redis.io/docs/latest/develop/reference/client-side-caching/).

## Scope and coalescing

Cache identity binds tenant, visibility, normalized query and representation/model versions. A tombstone carries a source version so delayed updates cannot resurrect deleted content. Negative caching applies only to a known absent result, not denial or transient source failure.

Request coalescing gives one loader ownership of a bounded shared operation; waiters can cancel independently. Bound the number/bytes of keys being loaded, loader duration and waiter count. When the last waiter leaves, the loader's continuation/cancellation policy must be explicit. Do not let one canceled caller terminate work promised to others accidentally.

## Decide which values earn space

| Workload | Candidate to evaluate | Main cost |
| --- | --- | --- |
| Small changing hot set | LRU or existing simple bounded cache | Scans can displace reused entries |
| Repeated hot keys with scan pollution | Frequency-aware admission such as TinyLFU | Sketch/metadata, aging and synchronization |
| Variable value sizes | Byte-weighted admission/eviction | Hit rate by item may differ from bytes/cost saved |
| Stable immutable expensive result | Memoization by content/version | Retention and low-reuse keys |
| Synchronized expiry burst | Bounded refresh/coalescing with jitter | Refresh capacity and allowed stale interval |

[Caffeine's Window TinyLFU design](https://github.com/ben-manes/caffeine/wiki/Efficiency) is one concrete admission/eviction composition. Its historical traces are not a guarantee for the target workload. A cache entry's miss cost, size and update frequency matter alongside recency/frequency.

Include L1 copies across instances, L2 serialization/network, metadata, cold-start and origin protection in the comparison. Test reordered/lost invalidations, generation eviction, stale fills, revocation, loader cancellation and full cache outage. Simplify back to direct source reads if saved cost does not exceed coherence and operating cost.
