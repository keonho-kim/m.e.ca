# Bounded processing and streaming pipeline

Choose for imports, exports, parsing, encryption, indexing or generated-output delivery where stages have different cost and lifetime. A direct sequential loop is the starting point. Parallel stages need independent work and a measured capacity benefit.

## Roles and flow

```text
bounded source iterator → parse/validate → transform → persist/emit
                              ↑ admission by bytes/cost
coordinator owns cancellation, stage errors and ordered completion
optional workers return results to one state/publication owner
```

The source owns its open body/cursor; each stage owns its current buffer and releases it on every exit. The coordinator closes/joins workers and resolves partial completion. The sink defines when bytes or state become committed/visible. Iterator exhaustion, decode failure and cancellation are distinct outcomes.

Use pure functions for calculations and explicit outer code for I/O. Native error propagation can express ROP-like short-circuiting without extra wrapper allocations. A Builder is useful only when a pipeline has meaningful configuration validity rules; do not make every stage a plugin.

## Parallelism and ordering

Choose bounded work units by input and expanded size. If output order matters, assign per-run sequence positions and bound the reorder buffer. One slow early item can otherwise retain every later result. Permit unordered completion only when the API/persistence contract allows it.

A single writer can own mutable assembly or publication without an actor framework. If independent partitions have their own mailbox/state, specify mailbox admission, ordering and epoch/lifetime. An actor that awaits external work may still need guards when later messages change the state; single-threaded execution alone is not transaction isolation across processes.

Apply backpressure before the next expensive allocation. For staged budgets, avoid workers holding one exhausted resource while waiting cyclically for another. Reserve a known envelope, acquire in a stable order, or release/fail growth explicitly. Include active items, queues and reordering in peak memory.

## Sharing, pooling and observers

Share immutable schema/type descriptors when many items really reuse them; bound their generation lifetime. A pool owns reusable buffers, a borrower owns the current lease, and no callback may use a returned buffer. Cap capacity and discard outliers. Pools must earn their reset/synchronization/retention cost.

Observers can receive disposable progress after a state change; each subscription owns a bounded queue and unsubscribe path. Required ordered collaborators should be direct calls or durable delivery. One slow observer must not pin the original input or prevent authoritative completion. Coalesce progress only when its semantic contract permits it.

## Partial effects and proof

For resumable imports, commit batch identity plus changes and checkpoint atomically where possible. A byte offset alone may not represent a valid parser/transaction boundary. For exports, state whether pages share a snapshot and who bounds that snapshot's age. A write to a socket does not prove the client consumed or retained it.

Measure peak bytes as active work + stage queues + reorder state + output buffering, then add runtime/native/shared overhead. Test oversized expansion, a stuck first item, a sink that stops reading, cancellation at each stage and crash after a batch commit. Verify cleanup as well as output. See [memory/I/O](../performance/memory-and-io.md) and [runtime boundaries](../performance/runtime-boundaries.md).
