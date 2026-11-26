# BACKEND DESIGN

Turn backend requirements into a concrete structure whose correctness, CPU, memory, storage, I/O, network and latency costs can be explained. Use the target project's language/framework conventions and preserve the requested design, review or implementation scope.

Use the [development workflow](../SKILL.md) for shared design and verification. Carry the actors, state and accepted outcomes from the [application guide](applications.md) into backend authority and failure contracts. Align pending, committed and uncertain outcomes with the [UI guide](ui.md).

## Establish the operation and cost

For substantial work, establish the invariant, authenticated scope, authority/commit point, load/burst/skew, data size/growth, latency/resource budgets, allowed staleness and recovery needs. Reuse settled facts; ask only about consequential unknowns and label assumptions.

Trace one important mutation and one expensive read. Count scans, round trips, allocations/copies, retained bytes and durable writes through admission, decision, persistence, publication and response, including failure/recovery. New designs use explicit cost hypotheses; existing code supplies profiles/baselines. Estimates are not benchmarks, and reasoning does not require already-running code.

## Choose a practical composition

Use the [pattern guide](patterns.md) for general structures and the [algorithm guide](algorithms.md) for in-memory representations and computation. These backend references apply those decisions to transactions, persistence, protocols and service operation.

Start with a function or cohesive feature module. Select patterns for an actual ownership, variation, transaction, delivery or resource requirement. Use native values/functions and explicit control flow. **Do not introduce Strategy or interchangeable policy-object hierarchies.** A pattern does not require its own package, interface or service.

Use the [composition map](backend/compositions.md) to derive collaborators, call/dependency direction, state transitions and cleanup. Adapt, combine or omit the illustrated roles; folder names are examples. Compare viable alternatives through **requirement → structure → authority/effects → savings/additional cost → failure → verification → removal condition**. The [design studies](backend/design-studies.md) show fictional workloads and explicit arithmetic, not prescribed stacks.

## Resolve the relevant contracts

| Decision | Read |
| --- | --- |
| Workload, domain ownership, distribution and scenario selection | [Architecture](backend/architecture.md) |
| Feature modules, transactions, read models, workflows, pipelines and integrations | [Compositions](backend/compositions.md) |
| Complete schema/flow/cost comparisons | [Design studies](backend/design-studies.md) |
| Physical models, SQLite/PostgreSQL/Redis, caches and projection rebuilds | [Storage](backend/storage.md) |
| API outcomes, idempotency, streams, jobs, tenancy and large objects | [API execution](backend/api.md) |
| Encryption boundaries, key lifecycle and advanced privacy | [Security](backend/security.md) |
| Retrieval, scheduling, runtime/memory/I/O and measurement | [Performance](backend/performance.md) |
| Diagnostic contracts, observability, invariants and recovery | [Operations](backend/operations.md) |
| Source scope, versions and pattern provenance | [Primary source index](backend/sources.md) |

## Preserve ownership and guarantees

- Give each durable fact one authority. Derived data needs freshness, deletion and rebuild contracts. IDs/timestamps are not automatically committed watermarks.
- Protect invariants at the actual transaction/conditional-write boundary. Cancellation or lost response does not establish that nothing committed; preserve unknown outcomes.
- Keep domain decisions independent of transport/SQL. Construct clients, pools, workers and sinks at a composition root, with explicit owners and cleanup.
- Centralize error codes/messages, events, field schemas, severity and redaction in diagnostics. Callers supply registered identities and bounded values/causes. Operations own retry/reconciliation; adapters own transport mapping.
- Budget bytes, count, concurrency, time and retention separately. Include slow consumers/maintenance; logical budgets are not RSS guarantees. Shared immutable data still needs lifetime ownership.
- Select maintained cryptography from a threat model. Design key-service cost, rotation and restore together; keep keys private and acknowledge the authorized plaintext boundary.

## Verify, then simplify

Produce a reviewable arrangement: owners, schema/access paths, transaction/publication sequence, diagnostics, cost assumptions and failure checks. A small change needs only its affected decisions; do not output a pattern checklist.

When implementation is authorized, inspect callers/contracts/tests and apply the TDD loop at the owning boundary. Use [invariant verification](backend/operations/invariant-verification.md) for races/uncertain effects and the performance map for equivalent-work measurements. Remove abstractions/copies that no longer earn their cost. Report the tests/checks actually run and their observed outcomes. Do not impose MECA-specific source-layout/header rules on unrelated projects.

Distinguish source guarantees, this skill's synthesis, assumptions and observed results. Check actual versions, source maturity and cryptographic errata when applying specialized mechanisms. Report completed work and evidence honestly. Loading tests do not prove model reasoning; use [evaluation cases](backend/operations/skill-evaluation.md) when assessing the skill itself.
