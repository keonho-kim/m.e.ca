# Replication, sharding and failure cells

Use when one authority no longer meets capacity, geography or isolation requirements. Begin with the particular limit: write serialization, hot working set, tenant interference, failure blast radius or distance. Adding replicas, shards and regions solves different problems.

| Technique | Potential benefit | Cost to include in the design |
| --- | --- | --- |
| Read replicas | Isolate/read-scale eligible queries | Lag, read-after-write routing, replica conflicts and failover loss |
| Partitioned tables | Pruning, retention and smaller maintenance units | Key choice, uniqueness constraints and partition management |
| Sharding | Split ownership and aggregate capacity | Rebalancing, scatter/gather, cross-shard invariants and hotspots |
| Cells | Bound tenant/resource/failure impact | Placement, spare capacity, routing and per-cell operations |
| Several regions | Locality and regional recovery | Replication lag or consensus latency, residency and conflict policy |

PostgreSQL's [standby documentation](https://www.postgresql.org/docs/current/warm-standby.html) distinguishes asynchronous and synchronous replication tradeoffs. Do not route a freshly committed mutation to an arbitrary lagging replica and promise immediate visibility. Define how the caller requests freshness and what happens when it cannot be met.

## Partition ownership, not merely keys

Choose the key from access locality and atomic invariants. A tenant key can keep most operations local yet create one oversized tenant. Time partitions can make expiration cheap yet concentrate all writes in the current partition. Global uniqueness, quotas and joins still need explicit authority.

Estimate query fan-out, per-shard candidate counts and final merge memory. More shards can reduce local work while increasing coordination and tail exposure. Bound parallel subrequests; an unrestricted broadcast converts a small client request into cluster-wide overload. Cancellation must reach losing/expired work.

For moves, establish an ownership epoch/fence, copy a consistent base, catch up changes, verify and switch authority at a defined point. Reject stale writers. A routing update and a best-effort cache flush cannot alone establish one writer. Do not introduce dual authorities without a reconciliation and cutover contract.

## Design the degraded state

Keep a serving data path independent of control-plane availability where the product permits cached configuration. Define the expiry and security consequences of that state. Amazon's [static stability account](https://aws.amazon.com/builders-library/static-stability-using-availability-zones/) is a case study of surviving dependency impairment with pre-existing capacity, not a mandate to reproduce its topology.

Authorization has its own freshness boundary. Old permission state applied to new content can expose data after revocation; the [Zanzibar paper](https://storage.googleapis.com/gweb-research2023-media/pubtools/5068.pdf) explains why causal ordering matters. A design needs equivalent guarantees only for the product's actual threat and consistency contract.

Validate hot-tenant behavior, a failed shard, lagging replica, interrupted move and loss of one intended failure domain. Record accepted unavailability or loss explicitly. Avoid automatic retries to every replica when the operation's commit outcome is unknown.
