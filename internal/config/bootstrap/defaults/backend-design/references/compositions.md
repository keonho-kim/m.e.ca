# Practical backend compositions

Read the arrangement that matches the operation. The general development prompt already describes pattern recognition; here a pattern earns its place through a real collaborator, owner, effect and cost. These are adaptable arrangements, not starter frameworks or required folders.

| Requirement | Arrangement | Useful pattern vocabulary |
| --- | --- | --- |
| One feature exposed through real callers | [Feature module](compositions/service-module.md) | Functional core, composition root, factory function, ports/adapters, bounded facade |
| One atomic business change with retries | [Transactional command](compositions/transactional-command.md) | Command, state machine, persistence contract, transaction/unit-of-work boundary |
| Reads need a cheaper representation | [Read model](compositions/read-model.md) | Projection, query separation, cache-aside, memoization, immutable sharing |
| Work spans independently committed effects | [Durable workflow](compositions/workflow.md) | Outbox, idempotent consumer, saga, state/HSM, event queue |
| Large processing or streaming with bounded memory | [Bounded pipeline](compositions/pipeline.md) | Pipes/filters, iterator, single writer, actor mailbox, pool, observer |
| Several external systems with real failure boundaries | [External integrations](compositions/integrations.md) | Adapter, decorator, proxy, explicit resolver chain, bulkhead/circuit breaker |
| User filters or nested domain rules | [Query/rule compilation](compositions/query-language.md) | Specification, composite AST, interpreter/visitor, bounded compilation |

## Read each arrangement as a design decision

Trace the call path separately from the dependency graph. A handler calling a persistence adapter through an application-owned contract does not require the domain to import SQL. Describe what is atomic, who can mutate state, who owns cancellation and what survives a crash. Include the cost of wrappers, copies, queues, callbacks and retained state.

Use the simplest fitting variant. A factory function is not GoF Factory Method; a directly composed family is not automatically Abstract Factory. Repository and unit-of-work concepts need meaningful persistence/commit behavior; a raw transaction alone is not an ORM change-tracking implementation. Direct error returns can express a fallible pipeline without ROP combinators. One process-owned client is not permission for a global Singleton service locator.

## Patterns with narrower backend uses

| Pattern family | Real trigger and boundary |
| --- | --- |
| Builder / Prototype / Type Object | Complex validated construction or reusable immutable definitions; publish once and separate mutable instance state. Do not clone handles or create builders for plain arguments. |
| Bridge / Component | Two actual independent dimensions/capabilities; compose their narrow contracts without a cross-product of subclasses. |
| Memento | A consistent checkpoint with version, retention and replay boundary; restoring a value does not reverse external effects. |
| Mediator / Blackboard | A bounded coordinator or typed request-scoped observations with named writers; do not accumulate global business state. |
| Behavior tree / HSM | Implemented resumable decisions or shared transitions that justify hierarchy; otherwise use a direct state table. |
| ECS / Spatial Partition | Proven bulk locality or spatial access requirements; use a suitable existing data/index implementation and account for update/rebuild cost. |
| Template Method / Factory Method | An existing inheritance extension contract actually requires it; prefer explicit collaborators in a new simple module. |

Use explicit branches/direct functions for variants; preserve the development prompt's exclusion of Strategy hierarchies. Named patterns have different scopes and are not all architecture-level alternatives.

For a chosen composition, write a short note identifying why it exists, the simpler rejected option, atomicity/lifetime, execution cost and a behavioral check. When the requirement disappears, remove the extra structure. [Pattern sources](sources/architecture.md) distinguish canonical terminology from this bundle's engineering arrangements.
