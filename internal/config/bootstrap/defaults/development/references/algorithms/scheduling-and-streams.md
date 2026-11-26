# Scheduling and stream processing

Define ordering, completion and retention before choosing a queue or summary. The cases below describe the in-process representation. Use the backend guides for durable acceptance, retry ownership and service admission.

## Delayed tasks and expiring entries

- **Use when** many active items have different deadlines and repeatedly scanning every item to find the next expiry dominates the work.
- **Build** a min-heap ordered by deadline and a stable tie-breaker. Keep one timer aimed at the current earliest deadline and re-evaluate it when that entry changes. For priority updates or cancellation, use indexed removal or replace the item's generation and discard stale heap entries when encountered. The [heap queue implementation notes](https://docs.python.org/3/library/heapq.html#priority-queue-implementation-notes) illustrate tie-breakers and changed priorities.
- **Cost and ownership.** Heap insertion and removal are logarithmic in queued entries. Lazy cancellation needs a bound or rebuild policy for stale entries, including those with far-future deadlines. Keep mutable heap state with one scheduler owner or a defined lock. Use a monotonic clock for elapsed waits. If deadlines must survive restarts, reconstruct from authoritative persisted time and policy.
- **Verify** equal deadlines, a new earlier deadline, repeated rescheduling, cancellation while due and shutdown with waiting workers. Use controlled time. Recheck an item's current generation before its effect and use the [backend scheduling guide](../backend/performance/admission-and-scheduling.md) for capacity and fairness.

## Selecting the best few results from many candidates

- **Use when** a local query or batch needs the best `k` items and a full sort performs unnecessary work.
- **Build** one total comparison order, such as score descending and stable ID ascending. Keep at most `k` accepted candidates in a heap whose root is the least desirable retained item. Replace that root only when a better candidate arrives. Sort the retained items into display order once input processing finishes.
- **Cost and ownership.** For `1 < k <= n`, heap selection costs `O(n log k)` and `O(k)` space before final output ordering. Define zero, one, oversized limits and ties at the cutoff. Use the data source's ordered limit when it can enforce the same filters and rank before transfer. For a nearly complete result set, compare with ordinary sorting.
- **Verify** ties, duplicate candidates, invalid scores and filters applied before ranking. Compare with full sorting on small inputs. Streaming previews must reflect that later candidates can change the result.

## Recent activity counts and rolling extremes

- **Use when** metrics or interaction history repeatedly ask for a count, sum, minimum or maximum over a moving recent window.
- **Build** an ordered deque of relevant events for exact expiry. On each read or update, remove expired entries and update the running aggregate. For a minimum or maximum, use a monotonic deque of candidate values with their positions or expiry times. Aggregate into fixed time buckets when the product accepts the bucket resolution and boundary error.
- **Cost and ownership.** Each event enters and leaves a deque at most once under monotone processing order. Memory depends on events still relevant to the window. A ring bounds bucket count when time resolution is fixed, and each slot needs a time identity to distinguish reused positions. Define arrival time versus event time and how late or reordered events are handled.
- **Verify** exact boundary expiry, long idle gaps, bursts and equal values. A processing-time deque can lose correctness when late event-time inputs are inserted without an ordering policy. For distributed enforcement of a request limit, use the [backend admission guide](../backend/performance/admission-and-scheduling.md).

## Sorting or merging an export larger than memory

- **Use when** an export, import or log merge must produce ordered results from more data than one safe working set.
- **Build** sorted runs from bounded input chunks, then merge their heads with a heap. If sources are already sorted by the required total order, start with the merge. Keep the same comparator and stable tie-breaker across runs. Limit simultaneously open runs and use multiple merge passes when file-descriptor or memory budgets require it.
- **Cost and ownership.** Budget run storage, input/output buffers, open files and merge-head state. Tie temporary files to the operation and clean them on success, failure and cancellation. Define the source snapshot and publication point so an interrupted export has an actionable outcome. Use the [pipeline guide](../backend/compositions/pipeline.md) for persistence and partial completion.
- **Verify** empty runs, equal keys across runs, one oversized record, a slow consumer, disk exhaustion and interrupted publication. Check sorted order and record multiplicity against a small in-memory result. Blocked stages must observe cancellation and release their resources. [Pipeline cancellation](https://go.dev/blog/pipelines)
