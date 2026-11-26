# Algorithms, data structures and optimization

Define the result and its invariants first. Write down the input sizes, distributions and operation mix. Estimate time and auxiliary space for the largest supported case and measure the dominant costs in the actual environment. Keep the straightforward solution when it meets the workload and correctness requirements.

## Start from the observed work

Choose a matching case below. Each case explains when to apply a technique, how to construct it, who owns the resulting state, its costs and the behavior to verify. Use the project's existing containers, parsers or solvers where they satisfy that contract.

| Work that needs a design | Practical guide |
| --- | --- |
| Repeated ID lookups, prefix suggestions, snapshot reconciliation or expensive parsing reuse | [Lookup and changing collections](algorithms/lookup-and-updates.md) |
| Delayed tasks, top results, rolling activity windows or exports larger than memory | [Scheduling and streams](algorithms/scheduling-and-streams.md) |
| Prerequisite jobs, partial recomputation, eligible resource assignment or budgeted work selection | [Dependencies and constrained choices](algorithms/dependency-and-allocation.md) |
| Variable-height lists, interval overlap, incremental text parsing or combined membership filters | [Ranges, edits and representations](algorithms/ranges-and-edits.md) |

Record the concrete operation, selected representation, source of truth, update/invalidation rule and verification for the change. Compare construction and maintenance cost with the repeated work it saves.

## Resolve the underlying algorithm

| Decision | Read |
| --- | --- |
| Repeated search, overlapping subproblems, graph traversal or large sets of events | [Algorithm techniques](algorithms/techniques.md) |
| Membership, range queries, ordered events, connectivity or text lookup | [Data structures](algorithms/data-structures.md) |
| Latency, allocation, retention, I/O or concurrency is the limiting cost | [Optimization workflow](algorithms/optimization.md) |

Use the least complex technique that fits both the measured or estimated workload and the observable contract. For an existing path, compare equivalent work. For a new design, state the cost assumptions and how to validate them. Use the [backend guide](backend.md) when the deciding costs or guarantees belong to query planning, database transactions, storage or service capacity. For rendering and interaction costs, use the [UI guide](ui.md) with the same workload.
