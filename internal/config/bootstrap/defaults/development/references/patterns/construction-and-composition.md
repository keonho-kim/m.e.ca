# Construction and composition

## GoF creational patterns

### Factory Method
- Use when a stable creator delegates a construction step to an extension that selects the concrete product. The GoF form specializes the creation method in a subclass. An ordinary factory function is a simpler construction mechanism, not the same structure.
- Example: let a specialized document importer supply its reader while the import procedure stays stable. Prefer native extension mechanisms and avoid a creator hierarchy when direct construction is sufficient.

### Abstract Factory
- Use when related products must be constructed as one compatible family, and consumers should depend on their contracts rather than concrete family members.
- Example: create a reader and writer that obey the same format rules. Keep the family small. Use direct construction when products vary independently.

### Builder
- Use when complex construction has meaningful stages, dependencies or validity rules that ordinary arguments cannot express clearly.
- Example: assemble a processing configuration and publish it only after its required parts are valid. Use a plain value or constructor for simple construction.

### Prototype
- Use when a new instance should start from an existing configured instance rather than repeat expensive or intricate setup.
- Example: clone a validated document template, then customize its instance values. Define shallow versus deep copying and keep mutable state and owned resources from being shared accidentally.

### Singleton
- Use only when a real invariant requires one instance within a defined scope. Make creation, access and lifecycle ownership explicit instead of introducing a global mutable service locator.
- Example: one process-owned resource registry is passed to its consumers. Check initialization races, shutdown and test isolation. A single instance does not make its operations concurrency-safe.

## GoF structural patterns

### Adapter
- Use at an actual boundary where external formats or APIs differ from the application's contract. Keep technology-specific translation with the adapter and business decisions with their owner.
- Example: convert different input formats into one internal value. An adapter translates a concrete boundary. The broader Ports & Adapters architecture is described separately below.

### Bridge
- Use when an abstraction and its implementation vary along independent dimensions. Connect them through composition so each dimension can evolve without a cross-product of subclasses.
- Example: compose a document output operation with a separate output-device implementation. Keep a direct dependency when only one dimension actually varies.

### Composite
- Use when leaves and groups form a tree and consumers need the same operations on both.
- Example: calculate a total for one item or a nested group. Define child ownership, traversal order and invalid cycles. Keep leaf-only operations out of the shared contract.

### Decorator
- Use when a stable contract needs an independently composed behavior around it. Keep the original input, output, error and cancellation semantics intact.
- Example: wrap an operation with duration measurement. Make wrapper order explicit and add only the wrappers the current behavior needs.

### Facade
- Use when consumers need a cohesive entry point to a complex subsystem. Keep the facade focused on its consumers' actual use cases and leave rules with their owners.
- Example: expose one import operation that coordinates parsing and storage. Avoid pass-through layers or a facade that accumulates unrelated responsibilities.

### Flyweight / Immutable Sharing
- Use when measured memory use comes from many equivalent immutable values. Share their common representation while keeping per-instance mutable state separate.
- Example: share identical immutable descriptors rather than copying them into every item. Define ownership and reclamation so sharing does not become permanent retention.

### Proxy
- Use when access to an object needs a distinct boundary for authorization, deferred initialization or remote interaction while preserving the consumer contract.
- Example: authorize access before opening a remote document. Make added latency, failures and ownership visible. A proxy cannot make remote effects behave like local atomic operations.

## Module construction and processing

### Factory Function
- Use when selecting or initializing a concrete resource is a real construction responsibility. Keep creation decisions out of its consumers.
- Example: create the appropriate reader for a declared input format. Prefer a factory function before introducing factory class hierarchies.

### Ports & Adapters
- Define narrow application-owned contracts for real external interactions and translate external APIs and formats in concrete adapters. Keep business decisions with the application owner.
- Example: connect an import use case to a concrete source through the required read contract. Add ports only for real interactions, not a speculative layer around every function.

### Dependency Injection and Composition Root
- Use when a component needs external resources or replaceable I/O. Pass the required dependencies explicitly and assemble application resources near the entry point.
- Example: supply a clock, persistence operation and sender to the use case. Prefer direct construction and function arguments before adding a container.

### Pipeline / Pipes and Filters
- Use when ordered processing stages have distinct input and output contracts. Define stage boundaries, short-circuiting and inter-stage ordering.
- Example: read -> parse -> validate -> transform -> emit. Parallelize independent elements or stages only when dependencies, buffering and result order remain correct.
