# Transactional command with a pure decision

Choose when a local authority can commit the business invariant, command identity and necessary publication intent together. Begin with a scoped operation and one native transaction. A repository must express useful persistence behavior; an ORM [Unit of Work](https://martinfowler.com/eaaCatalog/unitOfWork.html) also tracks changes, so do not claim that every transaction wrapper implements that full pattern.

## Roles and sequence

| Role | Owns |
| --- | --- |
| Ingress adapter | Parsed command, authenticated caller, bounded input |
| Command operation | Identity/fingerprint, state transition and observed outcome |
| Pure decision | Invariant/eligibility given current facts; registered rejection code |
| Persistence boundary | Scoped reads, constraints/CAS, mutation and commit |
| Relay, if required | Committed outbox delivery and acknowledgement |

Illustrative sequence, not executable SQL:

```text
parse and authenticate
begin local transaction
  bind scoped command identity; reject changed-input reuse
  if already committed: return its authorized stored outcome/reference
  read current target and permission/version facts
  decide transition using those facts
  conditionally write the transition
  record outcome and required outbox/audit intent
commit and distinguish known success, known rollback, uncertain commit
respond; relay owns later delivery
```

The unique command claim and effect share the transaction; a process-local map is not duplicate protection across instances. A losing CAS triggers a fresh decision or a domain conflict according to the contract, not repeated execution with stale facts.

A consumer-owned transaction callback can expose only the needed operations; it must not leak beyond the callback or perform remote calls. Alternatively, a cohesive storage operation can invoke a pure decision on the facts it locked/read. Choose the form with fewer leaked SQL/lifetime assumptions; do not duplicate transaction state in a second generic framework.

## Explicit state, native errors

Use an enum/table/direct branches for valid transitions and guards. Introduce State objects or HSM only when real behavior/shared transitions justify them. Command identity describes intent; recorded outcome describes observation. A native result/error path stops subsequent success-only steps without ROP combinators or exception swallowing.

Classify commit uncertainty separately from safe retry. If a response disappears after commit, command lookup resolves it. If command lookup is also unavailable, preserve uncertainty and bounded reconciliation. Do not delete an uncertain claim to make a retry possible. See [idempotency](../api/commands-and-idempotency.md).

## Performance and failure checks

Keep transaction residence short: precompute deterministic formatting outside it, but keep mutable eligibility and conditional mutation inside the consistency boundary. Budget touched rows/indexes/WAL, lock waits and stored outcome bytes. Use a compact immutable outcome reference only if its visibility and replay semantics remain stable.

Test competing commands, the same key with changed input, stale approval, commit-before-response loss and failure before/after publication. Assert one accepted business transition and consistent durable identity. If no external delivery is required, omit the outbox/relay. If there is only one simple change, keep it as a direct atomic operation.
