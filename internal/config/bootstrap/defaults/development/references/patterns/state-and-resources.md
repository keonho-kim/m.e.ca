# State and resources

## Composition, content, decision and performance approaches

### Component
- Use when entities need independently composed capabilities rather than an expanding inheritance hierarchy.
- Example: compose movement, health and interaction capabilities on an entity. Define capability ownership and interactions. Avoid splitting behavior into tiny components with no independent responsibility.

### Entity Component System (ECS)
- Use when many entities benefit from separating identity, component data and systems that process matching data sets.
- Example: process positions and velocities in batches. Verify locality and workload benefits, and define structural-change ordering, entity identity lifetime and parallel write ownership before adding an ECS architecture.

### Type Object
- Use when many instance kinds differ mainly by shared definitions rather than distinct behavior implementations.
- Example: instances refer to immutable item-type definitions. Separate definition data from mutable instance state, validate definitions and define update or reload consistency.

### Behavior Tree
- Use when decisions can be expressed as composed conditions and actions with explicit success, failure and running outcomes.
- Example: select between finding a target, approaching it and acting. Define tick order, resumption and cancellation, bound repeated work and avoid a tree for a simple sequence.

### Blackboard
- Use when cooperating decision modules need a shared store of observations or intermediate conclusions.
- Example: related decision steps consult an observed target and its freshness. Define typed keys, writers, scope, lifetime and conflict rules so the blackboard does not become unowned global state.

### Event Queue
- Use when producing an event and handling it must occur at different times. Define queue ownership, admission, ordering, delivery and shutdown.
- Example: process one completed operation in independent notification and accounting consumers. Make loss and duplication semantics explicit. An in-memory queue does not provide durability.

### Object Pool
- Use only when measurement shows repeated allocation or initialization is material and safe reuse reduces that cost.
- Example: reuse bounded buffers with a clear borrower and return path. Reset contents, protect sensitive data, prevent use after return and cap retained memory so pooling does not extend unnecessary lifetimes.

### Spatial Partition
- Use when spatial queries spend excessive work comparing unrelated objects. Select a grid, tree or other index from the distribution and query workload.
- Example: search only nearby collision candidates. Maintain the index as objects move and verify boundaries, duplicate candidates, dense regions and update cost.

## Concurrency and state-ownership models
A local ownership rule does not require an actor framework. Select these models for an actual concurrency and consistency requirement.

### Actor Model
- Use independent state-owning units that communicate through messages when isolation and asynchronous coordination fit the workload.
- Example: route entity or document updates through its owner's mailbox. Define mailbox admission, ordering, response delivery and shutdown. Independent actors can progress concurrently.

### Single Writer
- Give each mutable state partition one mutation owner and route updates to that owner. Serialize conflicting updates while allowing independent partitions to progress concurrently. The owner can be an event loop or transaction owner without an actor framework.
- Examine reentrancy, asynchronous suspension, cyclic waits and other processes that can modify the same state. Single-threaded execution alone does not establish a complete consistency boundary.
