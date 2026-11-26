# Backend invariant verification

Apply the [shared behavior verification cycle](../../../SKILL.md#verify-each-behavior) at the boundary that owns the invariant. Use controlled facts for pure decisions, controlled I/O for application effects, real disposable databases for SQL constraints and the actual adapter for transport contracts. Choose histories that expose concurrency, retries, cancellation and recovery.

## Turn a design into executable questions

Write the authoritative state, allowed operations, preconditions, effect identity and possible observed outcomes. Include unknown outcome explicitly. Choose an oracle that does not simply repeat implementation branches: a simpler state model, an invariant over recorded history or comparison with a trusted exact computation.

| Contract | Controlled history | Assertion |
| --- | --- | --- |
| One reservation per command | Concurrent duplicates; crash after commit before response | One durable effect and a queryable outcome |
| No stale owner writes | Pause worker, expire/reclaim, resume old worker | Authority rejects old epoch |
| Projection completeness | Snapshot plus concurrent update/delete; checkpoint crash | No missing committed changes or resurrection |
| Cache freshness | Pause old fill; commit/invalidate; resume fill | Behavior matches declared version/staleness contract |
| Resource ownership | Cancel while queued/active/native work persists | No leak or premature capacity reuse |
| Key recoverability | Fail at key creation, manifest commit and epoch switch | Every committed ciphertext remains decryptable under policy |

Use barriers, fake clocks and controlled adapters to force the schedule; arbitrary sleeps rarely prove that the intended interleaving occurred. Keep tests on isolated fixtures. Property-based generation can explore valid/invalid operation sequences and shrink failures; fuzzing is useful for parsers/encodings and budget enforcement. Neither substitutes for a specified oracle.

## Effects, histories and fault injection

Record command invocation, observed completion, durable identity and authority version. A timeout can leave an operation pending/unknown in the history; treating it as a definite failure can make a checker report a false violation or miss a duplicate.

Inject errors before effects, after effects but before acknowledgement, during cleanup and at restart. Simulate reordering/duplication only where the actual transport permits it. Verify retry ownership, deduplication lifetime and recovery of partial state. For linearizability/serializability requirements, select a checker/model matching the promised semantics; do not label an eventual projection linearizable because final values match.

FoundationDB's [deterministic simulation](https://apple.github.io/foundationdb/testing.html) illustrates reproducible schedules and faults in a system built for that model. A small backend usually needs controlled boundary tests first, not a new simulator framework. Production-disruptive experiments require the appropriate authorization/environment.

## Pair correctness with cost

Run the same scenario at representative load and record useful completions, attempted work, p99, memory and retained storage. Rejected/duplicated/skipped work must not inflate throughput. Assert eventual return to the expected idle resource bound when relevant, allowing intentional caches and runtime behavior.

Use contract tests at adapter boundaries and targeted integration tests for real database isolation/driver behavior. A mock cannot prove SQL constraints, WAL durability or pool cleanup. Keep deterministic correctness checks separate from timing-sensitive performance experiments; do not introduce flaky wall-clock assertions merely to obtain Red for an optimization. Preserve behavior tests, capture a reproducible baseline and compare equivalent work. Report model-only, mock, real-adapter and production evidence distinctly, with the checks actually run and observed failures/successes.
