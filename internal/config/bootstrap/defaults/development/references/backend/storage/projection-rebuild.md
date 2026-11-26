# Snapshot, change stream and projection generations

Use for search indexes, materialized read models, cache warm-up or replica-like consumers. The protocol must join a consistent base to subsequent changes without losing a committed update or resurrecting deleted data.

## Do not use allocation order as commit order

Illustrative PostgreSQL failure:

```text
A allocates row/event ID 10 and pauses before commit
B allocates ID 11 and commits
reader sees 11, stores cursor=11
A commits 10
reader asks for ID > 11 and misses A permanently
```

This schedule follows from separate concurrent transactions and [sequence allocation semantics](https://www.postgresql.org/docs/current/functions-sequence.html). Gaplessness or a wall-clock timestamp does not fix it. Choose a source with an actual committed-order/snapshot contract, or a polling protocol that tracks pending rows/acknowledgement without skipping them. Do not add arbitrary lookback and claim it proves completeness.

## A concrete generation protocol

1. Establish generation G and a consistent source snapshot S with its corresponding change position P. The source must guarantee which changes belong before/after that boundary. PostgreSQL [exported snapshots for logical decoding](https://www.postgresql.org/docs/current/logicaldecoding-explanation.html) provide a specific mechanism.
2. Build G from S in bounded batches. Retain source change history until replay catches up; monitor oldest snapshot and retained WAL.
3. Apply changes after P idempotently with source object version, deletion tombstones and committed progress. A checkpoint advances only after all effects it covers are durable, including all relevant records within a source transaction/position.
4. Verify counts/checksums or semantic samples, lag and authorization/deletion behavior. A count match alone is insufficient.
5. Switch the readable generation atomically according to the product's freshness contract. Fence old projectors and stop publishing into the wrong generation.
6. Retire old generation/history only after active readers and recovery requirements permit it. Bound overlap bytes and age.

Persist generation, source identity/epoch, schema/analyzer version and progress. After failover or lost history, require a valid continuation contract or rebuild. A numeric cursor from another source epoch is not automatically comparable.

## Concurrent state and consumers

For a mutable source, backfill and live updates can race. Apply each object only if its incoming version is newer; a delete is a versioned state, not just a missing row. Preserve tombstones until no eligible old event/backfill can reintroduce the object. If the sink cannot atomically update state plus checkpoint, replay must be harmless and progress must never run ahead of durable effects.

For snapshot plus live subscription, either register/buffer the bounded tail before the snapshot cut, or use a durable replay position bound to that cut. Define duplicate handling and resync when retention expires. Avoid a permanently open database snapshot for each client.

## Cost and proof

Budget source scan/read impact, WAL retention, both index generations, temporary sorting and catch-up capacity. Rate-limit rebuild independently so it cannot exhaust interactive memory/connections. Verify the ID-10/11 schedule, concurrent delete/backfill, replay duplicates, checkpoint crash, source failover, stale projector and a failed cutover. See [read-model composition](../compositions/read-model.md) for role placement.
