# Feature module and composition root

Choose this as the initial shape for a cohesive backend feature. A request-oriented function can own a complete use case without a network service, event bus or interface for every function. Fowler's [Transaction Script](https://martinfowler.com/eaaCatalog/transactionScript.html) describes a related procedural organization; choose richer domain objects only when their invariants/behavior warrant them.

## Concrete arrangement

Illustrative responsibilities, not required folder names:

```text
entrypoint/              construct shared DB/client/logger; own start/stop
records/
  approve                application operation; caller scope and effect order
  decision               pure transition from facts to accepted/rejected change
  read                   query projection required by real callers
adapters/http/           parse, authenticate, map outcomes, encode once
adapters/postgres/       scoped SQL, atomic persistence and driver errors
diagnostics/             registered faults/messages/events/field schemas
```

Call flow: `HTTP → records.approve → persistence operation → result → HTTP`.
Dependency direction: the adapter implements the consumer's needed I/O contract; the decision imports neither HTTP nor SQL. Construction happens at the entrypoint. If a feature-local SQL function is already a clear boundary, do not add a forwarding repository merely to match the diagram.

Define a narrow operation such as `Approve(ctx, actor, command) -> outcome/error`, not a generic request map. Parse once, use typed values and preserve a cause through the registered fault mechanism. A second caller such as a worker reuses the use case while supplying its own authenticated execution context.

## Where patterns help

| Requirement | Small fitting structure |
| --- | --- |
| Construct a configured DB/client | Factory function called once by the composition root |
| Same rule used by HTTP and job execution | Pure decision/value function with facts passed explicitly |
| Controllable external I/O | Consumer-owned narrow contract; concrete adapter |
| Several coordinated internal steps | Use-case entry point; a facade only if it hides a real subsystem |
| Complicated valid configuration | Builder only for meaningful staging; publish an immutable validated value |

Do not equate a factory function with GoF Factory Method or require a DI container. Shared clients have process lifetime; transactions and input buffers have operation lifetime. A module boundary should not force serialization, deep copying or another goroutine.

## State and failure

The use case owns outcome semantics and decides whether an effect is known absent, committed or uncertain. The storage adapter owns SQL and commit observation. The entrypoint owns client/sink shutdown. A failed notification after commit is not a failed uncommitted transaction; route required durable publication through the appropriate composition.

Keep input/authentication checks before effects, then recheck mutable authorization/preconditions at the actual authoritative boundary. A pure decision computed before a concurrent update needs a version guard or fresh facts inside the transaction.

## Cost and simplification

Count one parse/encode at each real external boundary. Keep one canonical large payload and pass a scoped reference or borrowed immutable value when lifetime permits. Avoid layered `DTO → map → JSON → map → domain` conversion and hidden lazy queries during response serialization.

Verify HTTP and worker callers observe the same domain outcome, denied calls have no effect, resources close on cancellation, and an adapter replacement preserves the contract. Remove unused ports, empty layers and wrappers that have no independent policy. See [ownership](../architecture/ownership.md) for the shared review rules.
