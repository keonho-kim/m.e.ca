---
name: backend-design
description: Design production backends with performance and reliability built into their structure. Use for responsibility and pattern selection, SQLite/PostgreSQL/Redis architecture, API workflows, centralized diagnostics, secure data and advanced retrieval; connect choices to workload, resource cost and failure behavior.
license: Apache-2.0
---

# BACKEND DESIGN

Turn backend requirements into a concrete structure whose correctness, CPU, memory, storage, I/O, network and latency costs can be explained. Use the target project's language/framework conventions and preserve the requested design, review or implementation scope.

Invoke only `#backend-design`, without subcommands or required arguments. Infer the task from the conversation and project. Read **entry → relevant map → selected reference**; every resource is reachable within two file-link hops. Read only what changes the decision. The three reading levels and 8KiB Markdown budget are bundle authoring limits, not application limits or Agent Skills requirements.

## Default to TDD for implementation

For testable implementation work, use **Red → Green → Refactor**, one requested behavior at a time, using the project's native test tools.

1. **Red:** derive a focused behavioral test from an acceptance criterion or invariant. Run it before implementing that behavior and confirm failure for the intended missing behavior. Dependency, environment or tooling failures are not a valid Red result.
2. **Green:** implement the smallest correct change that passes the test. Cover the relevant success, boundary and failure behavior incrementally; do not build the whole feature before its tests.
3. **Refactor:** simplify responsibilities, dependencies and resource use while keeping tests green, then repeat. Verify performance changes with equivalent-work measurements as well as correctness tests.

For design/review-only requests, define acceptance cases without starting implementation. Wording/mechanical changes and behavior-preserving refactors use review and applicable existing checks; add a characterization test only for a meaningful coverage gap. If automation is impractical, use an observable check and report its limits. See [backend TDD and invariants](references/operations/invariant-verification.md) for test boundaries and controlled failures.

## Establish the operation and cost

For substantial work, establish the invariant, authenticated scope, authority/commit point, load/burst/skew, data size/growth, latency/resource budgets, allowed staleness and recovery needs. Reuse settled facts; ask only about consequential unknowns and label assumptions.

Trace one important mutation and one expensive read. Count scans, round trips, allocations/copies, retained bytes and durable writes through admission, decision, persistence, publication and response, including failure/recovery. New designs use explicit cost hypotheses; existing code supplies profiles/baselines. Estimates are not benchmarks, and reasoning does not require already-running code.

## Choose a practical composition

The development prompt supplies general pattern vocabulary; this self-contained skill extends it into backend arrangements. The target project need not contain that prompt file.

Start with a function or cohesive feature module. Select patterns for an actual ownership, variation, transaction, delivery or resource requirement. Use native values/functions and explicit control flow. **Do not introduce Strategy or interchangeable policy-object hierarchies.** A pattern does not require its own package, interface or service.

Use the [composition map](references/compositions.md) to derive collaborators, call/dependency direction, state transitions and cleanup. Adapt, combine or omit the illustrated roles; folder names are examples. Compare viable alternatives through **requirement → structure → authority/effects → savings/additional cost → failure → verification → removal condition**. The [design studies](references/design-studies.md) show fictional workloads and explicit arithmetic, not prescribed stacks.

## Resolve the relevant contracts

| Decision | Read |
| --- | --- |
| Workload, domain ownership, distribution and scenario selection | [Architecture](references/architecture.md) |
| Feature modules, transactions, read models, workflows, pipelines and integrations | [Compositions](references/compositions.md) |
| Complete schema/flow/cost comparisons | [Design studies](references/design-studies.md) |
| Physical models, SQLite/PostgreSQL/Redis, caches and projection rebuilds | [Storage](references/storage.md) |
| API outcomes, idempotency, streams, jobs, tenancy and large objects | [API execution](references/api.md) |
| Encryption boundaries, key lifecycle and advanced privacy | [Security](references/security.md) |
| Retrieval, scheduling, runtime/memory/I/O and measurement | [Performance](references/performance.md) |
| Diagnostic contracts, observability, invariants and recovery | [Operations](references/operations.md) |
| Source scope, versions and pattern provenance | [Primary source index](references/sources.md) |

## Preserve ownership and guarantees

- Give each durable fact one authority. Derived data needs freshness, deletion and rebuild contracts. IDs/timestamps are not automatically committed watermarks.
- Protect invariants at the actual transaction/conditional-write boundary. Cancellation or lost response does not establish that nothing committed; preserve unknown outcomes.
- Keep domain decisions independent of transport/SQL. Construct clients, pools, workers and sinks at a composition root, with explicit owners and cleanup.
- Centralize error codes/messages, events, field schemas, severity and redaction in diagnostics. Callers supply registered identities and bounded values/causes. Operations own retry/reconciliation; adapters own transport mapping.
- Budget bytes, count, concurrency, time and retention separately. Include slow consumers/maintenance; logical budgets are not RSS guarantees. Shared immutable data still needs lifetime ownership.
- Select maintained cryptography from a threat model. Design key-service cost, rotation and restore together; keep keys private and acknowledge the authorized plaintext boundary.

## Verify, then simplify

Produce a reviewable arrangement: owners, schema/access paths, transaction/publication sequence, diagnostics, cost assumptions and failure checks. A small change needs only its affected decisions; do not output a pattern checklist.

When implementation is authorized, inspect callers/contracts/tests and apply the TDD loop at the owning boundary. Use [invariant verification](references/operations/invariant-verification.md) for races/uncertain effects and the performance map for equivalent-work measurements. Remove abstractions/copies that no longer earn their cost. Report the tests/checks actually run and their observed outcomes. Do not impose MECA-specific source-layout/header rules on unrelated projects.

Distinguish source guarantees, this skill's synthesis, assumptions and observed results. Check actual versions, source maturity and cryptographic errata when applying specialized mechanisms. Report completed work and evidence honestly. Loading tests do not prove model reasoning; use [evaluation cases](references/operations/skill-evaluation.md) when assessing the skill itself.
