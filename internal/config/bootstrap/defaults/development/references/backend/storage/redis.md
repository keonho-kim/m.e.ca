# Redis and cache economics

Use Redis for a justified shared hot set, short-lived state or coordination mechanism. Define whether each key is reconstructible, how stale it may be, and what a missing key means. Separate correctness-critical state from an evictable cache when their operating policies conflict.

## Design coherence before adding levels

Cache keys bind tenant, authorization-relevant scope, query/filter identity and representation version. Do not share a value or negative lookup across callers whose visibility differs. Bound key/value bytes, count, TTL and aggregate retained memory.

Cache-aside has races: a reader can fill an old value after a concurrent writer invalidates it. Choose an appropriate mechanism—versioned values/generations, ordered invalidation, a transactional change stream or a product-approved stale window. Include version retention and rebuild cost. A second local cache can save round trips but duplicates memory and adds another coherence boundary.

Use the [cache consistency protocol](cache-consistency.md) to reason through the stale-fill schedule, atomic installation, generation loss and freshness contract. That reference also owns admission/eviction selection; selecting Redis does not settle those choices.

Server-assisted client caching requires handling invalidation/order races and flushing local state when the invalidation connection is lost. See [client-side caching](https://redis.io/docs/latest/develop/reference/client-side-caching/). Pub/Sub notification alone does not provide durable catch-up; its [delivery semantics](https://redis.io/docs/latest/develop/pubsub/) permit message loss.

Use request coalescing for a hot miss with separate waiter cancellation and a bounded shared loader. Jitter expiration/refresh where freshness permits. Set a short, scoped negative TTL for truly absent results; never cache transient authorization/database failure as absence. Limit miss concurrency so a Redis outage does not immediately overwhelm the authority.

## Account for physical memory and latency

`maxmemory` is not total RSS: persistence/replication buffers and allocator behavior require headroom. Choose eviction policy from actual value reuse; `noeviction` converts pressure into write errors. See [eviction](https://redis.io/docs/latest/develop/reference/eviction/).

Inspect per-key overhead, large collections, expiration bursts and encoding transitions. Small aggregate encodings can save memory, but grouping changes access, eviction and expiry behavior. Measure actual object bytes and conversion latency rather than copying historical savings from the [memory guide](https://redis.io/docs/latest/operate/oss_and_stack/management/optimization/memory-optimization/).

Bound pipeline batches by commands and bytes; reduce round trips without retaining unbounded queued replies or delaying urgent requests. [Pipelining](https://redis.io/docs/latest/develop/using-commands/pipelining/) does not make a sequence atomic. Long scripts/functions and large key operations can stall unrelated clients. Chunk bulk maintenance and measure server latency under it.

Cluster sharding does not split one hot key across servers. Review slot placement and same-slot requirements before grouping atomic operations; hash tags can preserve locality while also concentrating load. Budget client topology refresh/reconnect behavior during moves. See [Redis Cluster](https://redis.io/docs/latest/operate/oss_and_stack/management/scaling/).

## Durability and coordination are separate contracts

RDB/AOF policies trade latency, restart time and loss exposure; background persistence also needs fork/copy-on-write memory. Use the [persistence guide](https://redis.io/docs/latest/operate/oss_and_stack/management/persistence/) for the selected mode. `WAIT` improves replication acknowledgement but does not establish strong consistency or guarantee no failover loss; see its [consistency section](https://redis.io/docs/latest/commands/wait/).

A lease needs an unpredictable owner value, atomic owner-checked release and bounded validity. A paused owner can resume after expiry: a monotonically fenced write must be rejected by the actual protected authority. Redis's [lock discussion](https://redis.io/docs/latest/develop/clients/patterns/distributed-locks/) explicitly calls for fencing. Do not use a cache lease as the sole inventory, money or uniqueness invariant.

Test cold start, hot-key skew, invalidation loss, eviction, failover, stale owner and a full cache outage. Report useful throughput and origin load alongside hit rate.
