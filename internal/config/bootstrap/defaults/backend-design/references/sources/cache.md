# Redis and cache primary sources

Checked 2026-11-09. `latest` documentation and Redis product names cover several deployment variants. Confirm commands, persistence, clustering and client support in the selected version/tier. These sources support [Redis/cache design](../storage/redis.md).

| Source | Use for |
| --- | --- |
| [Persistence](https://redis.io/docs/latest/operate/oss_and_stack/management/persistence/) | RDB/AOF, loss/restart tradeoffs and background persistence cost |
| [Eviction](https://redis.io/docs/latest/develop/reference/eviction/) | Policy selection and memory excluded from eviction accounting |
| [Client-side caching](https://redis.io/docs/latest/develop/reference/client-side-caching/) | Invalidation ordering, connection loss and local cache flushing |
| [Distributed locks](https://redis.io/docs/latest/develop/clients/patterns/distributed-locks/) | Ownership, lease assumptions and fencing requirements |
| [Redis Cluster](https://redis.io/docs/latest/operate/oss_and_stack/management/scaling/) | Slot placement, topology and cross-key limitations |
| [Streams](https://redis.io/docs/latest/develop/data-types/streams/) | Consumer groups, pending work, acknowledgement and trimming |
| [Pipelining](https://redis.io/docs/latest/develop/using-commands/pipelining/) | Round-trip savings versus queued reply memory |
| [WAIT](https://redis.io/docs/latest/commands/wait/) | Replication acknowledgement without a strong-consistency guarantee |
| [Pub/Sub](https://redis.io/docs/latest/develop/pubsub/) | At-most-once delivery and lost-notification consequences |
| [Memory optimization](https://redis.io/docs/latest/operate/oss_and_stack/management/optimization/memory-optimization/) | Small-object encodings and measured representation tradeoffs |

Redis's lock page presents its algorithm and assumptions. This skill does not treat a lease as a substitute for a durable invariant: the protected authority must reject stale owners. Historical examples and benchmark figures in documentation are not sizing guarantees for a present deployment.

[Caffeine's efficiency account](https://github.com/ben-manes/caffeine/wiki/Efficiency) describes Window TinyLFU admission/eviction and historical trace comparisons. Use it for cache-policy mechanisms, not a promise that a policy is best for every workload or a current legal/license assessment of alternatives mentioned in that historical article.
