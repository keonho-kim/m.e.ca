# Operations and diagnostics decisions

Design observability, capacity and recovery together with the request path. They have resource and consistency costs of their own.

| Need | Read |
| --- | --- |
| Logger/error handler ownership, stable codes, safe messages and transport mapping | [Central diagnostics](operations/diagnostics.md) |
| Implementable catalog schema, caller contract and mapping example | [Diagnostic contracts](operations/diagnostic-contracts.md) |
| Traces/metrics/log correlation, sampling, cardinality and alerting | [Telemetry](operations/telemetry.md) |
| Deadlines, admission, retry storms, dependency outages and shutdown | [Resilience](operations/resilience.md) |
| Failover, backup restore, corruption boundaries and production change | [Recovery](operations/recovery.md) |
| Default backend TDD, controlled race/crash histories and state-model checks | [TDD and invariant verification](operations/invariant-verification.md) |
| Evaluating this skill's design decisions rather than its file delivery | [Skill evaluation cases](operations/skill-evaluation.md) |

Keep request diagnostics, durable audit records, business events and operation state distinct. They may correlate by identifier without copying full payloads into all four stores. Sampling a debug log is reasonable; sampling the sole record of an accepted financial mutation is a different product decision.

The diagnostics module owns definitions and publication policy. The application owns whether an operation failed, whether its effect is uncertain and whether another attempt is safe. The process composition owns sinks, queues, rotation, flush deadlines and shutdown joins. Centralization should remove duplicated policy without producing a global service locator or swallowing errors.

Operational evidence should answer: what users lost, what is saturated, where work is waiting, whether effects committed, and how to recover without replaying them. Pages should follow an actionable objective and an owner. Use the [operations sources](sources/operations.md) for concrete production mechanisms and their original scope.
