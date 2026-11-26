# Optimization workflow

## Measure the path and define the target

Follow one affected operation from input through lookup, allocation, transformation, effects and delivery. State the workload, bottleneck and target before adding an optimization. For existing code, profile or instrument the actual path. For a new design, state the estimate and what measurements will confirm it.

Use the [practical selection map](../algorithms.md#start-from-the-observed-work) to connect that observation to a concrete representation or processing method. Carry the case's update, lifetime and verification rules into the change.

Compare equal inputs and outputs with the same build and environment. Include the cold and warm paths, common and worst input shapes, bursts and long-lived use where relevant. Measure elapsed time, CPU, allocations, retained heap, process memory and I/O separately. For latency under load, examine both a typical result and relevant tail percentiles.

## Reduce work at its owner

1. **Perform fewer operations.** Select needed fields, short-circuit when the result is known, replace repeated scans with a query-shaped index and batch work when the contract allows it. Include the cost of maintaining any new index.
2. **Move fewer bytes.** Avoid repeat copies, parsing or encoding of immutable results. Use views or shared buffers with a clear lifetime. Check if a small view keeps a large backing allocation alive.
3. **Bound materialization.** Stream or page inputs and incremental results. Preserve ordering and partial failure semantics. Bound input acceptance, working memory, output buffering and retained state separately.
4. **Reuse only when it pays.** For a cache or precomputation, give the key, owner, update/revision rule, capacity, eviction and release point. Include the inputs that affect identity or visibility. Compare the saved work with misses, invalidation, memory and synchronization.
5. **Schedule independently owned work.** Use bounded concurrency for operations with real overlap and enough available resources. Include waiting, queueing, contention, reordering and cancellation. Serialize effects that require ordering and release workers, reservations and buffers on each exit.

Keep a simple fallback when the fast path has narrower eligibility. For an incremental result, define which changes invalidate the derived state and when to recompute. For bounded overload, account for who can retry, which accepted results must survive and how the operation reports saturation. Follow the [backend guide](../backend.md) for transaction, service admission or recovery guarantees.

## Example decision

A renderer that repeatedly scans all records to find a selected item might benefit from a key index. Start by counting scans and comparing an indexed view built once per input revision with the current scan. For small lists or one lookup per revision, building the index can cost more than scanning. For stable large lists and many lookups, the index can save repeated work. Verify selection after replacement, deletion and a late update. Measure rebuild cost and retained memory along with lookup latency.

## Verify the change and its lifetime

Keep regression checks for the observable result, ordering, error, cancellation and cleanup affected by the change. Compare representative runs and state the break-even assumptions or remaining bottleneck. Check memory after repeated work and after resources are released. Report measured improvements, measurement scope and conditions that still need evidence.
