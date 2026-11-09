# Backend TDD and invariant verification

Use TDD by default for testable backend implementation. Derive tests from the requested behavior and business/resource invariants, including concurrency, retries, cancellation and recovery where relevant. A race detector finds data races, not every logical ordering bug.

## Drive one behavior through the loop

Choose a small acceptance slice such as rejecting a stale approval or returning a committed command's outcome on retry. Define its observable result and effects before editing implementation.

| Stage | Required evidence |
| --- | --- |
| Red | Run the focused test and observe the intended missing behavior/invariant failure; inspect why it failed |
| Green | Make the smallest correct implementation and run that test plus directly affected checks |
| Refactor | Simplify the owning code while checks remain green; broaden only for changed shared contracts or unresolved risk |

For a bug, first reproduce the faulty behavior with a focused regression test when feasible. A test already passing does not demonstrate a missing behavior; inspect its input/assertion or whether the requirement already works. Do not sabotage existing code to manufacture Red. Resolve setup/compile/dependency failures separately and report them honestly.

Test at the boundary that can establish the claim: pure decisions with controlled facts, application effects through narrow controlled I/O, SQL constraints/isolation through a disposable real database, and transport mapping through the actual adapter. Avoid tests that merely repeat private helper branches or mock every layer so the real invariant is never exercised.

Add relevant success, limit and failure cases as small cycles. Keep clocks, identities and interleavings controllable. Do not implement the complete feature and then write tests to match it. Preserve existing checks during a structural refactor; characterize an uncovered observable contract only when that protects the change.

Design/review-only work records acceptance cases and intended test boundaries without implementing them. Documentation/mechanical edits use review and existing checks. If a behavior cannot be automated, state the observable verification and its limitations; a proposed test is not an executed result.

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
