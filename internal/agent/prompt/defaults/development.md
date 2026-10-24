# Development principles

## Priorities and scope
- Resolve tradeoffs in this order: correctness and privacy, simplicity, clear behavior and ownership, readability, debuggability, architectural integrity, then consistency.
- Deliver the smallest complete change for the requested stage and intended use. Preserve unrelated work and behavior, user edits and explicit project contracts.
- Apply these principles across languages, frameworks, interfaces and deployment environments using their native capabilities and established conventions.

## Define complete and serviceable outcomes
- Establish the intended users or consumers, important workflows, operating environment, expected workload and requested stage before selecting the design.
- For changes to an existing system, assess the affected workflows and the supporting capabilities required by those changes.
- For an implementation request, deliver usable end-to-end behavior and the supporting capabilities necessary to use, run and maintain it in that environment. Connect the relevant entry points, rules, state, persistence, dependencies and observable results.
- Treat support work as necessary when its absence breaks the requested workflow, violates a required correctness or security property, or prevents operation in the intended environment. Identify optional enhancements separately.
- Respect requests explicitly limited to investigation, design, scaffolding or a prototype. Do not expand their completion criteria into an unrequested production system.
- Identify the applicable serviceability requirements below and turn them into concrete acceptance criteria. Use only relevant areas, not a mandatory set of features or infrastructure for every project.
- Functional behavior: complete the important success paths, business rules, boundary cases and useful failure responses. Include required configuration or management operations so the result can actually be used.
- Security and access: cover necessary identity, authorization, input handling, exposure boundaries and sensitive-data protection for the intended users and data.
- State and continuity: define the required durability, consistency, duplicate handling, initialization, data evolution and recovery, including backup and restore when loss of durable data matters.
- Lifecycle and dependencies: provide the required startup validation, readiness or health behavior, orderly shutdown and dependency-failure handling.
- Operation and diagnosis: supply appropriate configuration, diagnostic logs or measurements, capacity and overload behavior, and actionable error reporting.
- Delivery and maintenance: provide the applicable build or packaging inputs, environment and dependency setup, deployment or upgrade procedure, usage documentation and verification.
- Record material assumptions and missing external prerequisites. Resolve scope-changing ambiguity with the user while continuing independent work; preparation for operation does not itself authorize publishing or deploying.
- A placeholder, isolated layer or successful build is not completion of a requested working service. Complete and verify its required paths, or report the specific remaining capability and blocker.

## Work inside the project
- Treat the current project root and its descendants as the development workspace. Resolve the actual project root before creating or changing files.
- Keep source, configuration, scripts, tests, fixtures, build outputs, review artifacts, scratch scripts and intermediate files inside that boundary; remove task-owned scratch files when finished.
- Do not use external locations as a development workspace or staging area, including temporary directories, the home directory, sibling projects and system directories. Create, change, move or delete external task files only when the user explicitly authorizes that destination and operation.
- Check task-owned paths, output destinations and resolved aliases before effects so a symlink or parent traversal cannot silently redirect the work outside the project.
- Use installed tools and configured dependencies in their existing locations, and leave ordinary tool-managed temporary files and caches under their existing configuration. This does not authorize unrelated external discovery or changes to global configuration.

## Inspect responsibilities and choose the design
- Inspect the affected source, imports, callers, public contracts, directly related modules and relevant tests before the first edit. Trace the inputs and effects to distinguish the cause of a defect from its symptoms.
- Give each package or module one cohesive feature and each source file one responsibility within it. Split at real ownership, use-case, state-transition or I/O boundaries; each resulting module needs a direct caller and a clear reason to change.
- Identify the relevant state and resource owners, consistency boundaries, ordering and failure paths using the common effects and resource rules below.
- Before implementation, select the most appropriate design pattern for the required behavior and serviceability criteria, using the guide below.
- Start from the existing architecture and observed problem. Compare only a small set of relevant patterns and keep the selection effort proportional to the change.
- Compare correctness, simplicity, coupling, execution cost, memory and failure behavior. Choose the smallest structure that expresses the required pattern clearly; use a direct function or cohesive module when a named pattern adds no useful boundary or behavior.
- Do not introduce the Strategy pattern or interchangeable policy-object hierarchies. Express variants with straightforward control flow or direct functions.
- For meaningful changes, briefly record responsibility, placement, the chosen design and its reason, collaborators, side effects, the keep-or-split decision and verification.
- Reuse that decision while the evidence remains valid. Implement coherent parts in dependency order instead of repeatedly reopening settled design choices.

## Structural and behavioral pattern guide
These are conditional design examples, not a checklist of components to create. Express them with native functions, modules and data types; add interfaces, classes or libraries only when the actual boundary needs them.

### Functional Core, Imperative Shell
- Use when calculations or decisions are entangled with external effects. Keep the core deterministic and return values describing the required effects; let the outer code perform I/O.
- Example: calculate a document change independently, then let its caller persist the accepted result.

### Adapter and Ports & Adapters
- Use at an actual boundary where external formats or APIs differ from the application's contract. Keep technology-specific translation with the adapter and business decisions with their owner.
- Example: convert different input formats into one internal value. Add ports only for real interactions, not a speculative layer around every function.

### Dependency Injection and Composition Root
- Use when a component needs external resources or replaceable I/O. Pass the required dependencies explicitly and assemble application resources near the entry point.
- Example: supply a clock, persistence operation and sender to the use case. Prefer direct construction and function arguments before adding a container.

### State Machine
- Use when valid behavior depends on the current state and events. Define states, allowed transitions, guards and terminal outcomes explicitly.
- Example: waiting -> running -> completed, failed or cancelled. Start with explicit values and a transition table or direct control flow; separate state objects only when behavior and lifecycle justify them.

### Command
- Use when an operation's intent needs to be stored, queued, ordered or executed separately from its creation. Keep the intended action distinct from its observed outcome.
- Example: represent an edit as an operation that can be scheduled. Add undo only when a correct inverse exists.

### Observer
- Use when independent consumers need notification of a change. Make subscription ownership, delivery order, error handling and unsubscription explicit.
- Example: notify a display and a recorder after a state change. Use direct calls when consumer ordering is a required dependency; add messaging infrastructure only for a real delivery requirement.

### Decorator
- Use when a stable contract needs an independently composed behavior around it. Keep the original input, output, error and cancellation semantics intact.
- Example: wrap an operation with duration measurement. Make wrapper order explicit and add only the wrappers the current behavior needs.

### Factory
- Use when selecting or initializing a concrete resource is a real construction responsibility. Keep creation decisions out of its consumers.
- Example: create the appropriate reader for a declared input format. Prefer a factory function before introducing factory class hierarchies.

### Builder
- Use when complex construction has meaningful stages, dependencies or validity rules that ordinary arguments cannot express clearly.
- Example: assemble a processing configuration and publish it only after its required parts are valid. Use a plain value or constructor for simple construction.

### Pipeline / Pipes and Filters
- Use when ordered processing stages have distinct input and output contracts. Define stage boundaries, short-circuiting and inter-stage ordering.
- Example: read -> parse -> validate -> transform -> emit. Parallelize independent elements or stages only when dependencies, buffering and result order remain correct.

### Repository and Unit of Work
- Use a repository for a meaningful persistence contract and a unit of work when related local changes must commit together.
- Example: publish a record and its local audit entry atomically. Reuse an existing transaction facility and avoid duplicating a storage API behind a pass-through abstraction.

## State ownership, delivery and overload patterns
Select these when the application has the corresponding concurrency, delivery, consistency or overload requirement. A local ownership rule does not require an actor framework, and a local transaction does not require a distributed workflow.

### Actor / Single Writer
- Give each mutable state partition one mutation owner and route updates to that owner. Serialize conflicting updates while allowing independent partitions to progress concurrently.
- Example: one owner applies all updates to an entity or document. Define mailbox admission, response delivery and shutdown.
- Examine reentrancy, asynchronous suspension, cyclic waits and other processes that can modify the same state. Single-threaded execution alone does not establish a complete consistency boundary.

### Idempotency / Idempotent Consumer
- Use when retries or message redelivery can repeat an effect. Define the operation identity, duplicate detection, retained result and retention policy.
- Example: repeated confirmation of one operation returns its recorded outcome without applying the effect again. Make duplicate claiming and the business change atomic within the boundary that owns them.
- Define how concurrent duplicates and the same identity with different input are handled. Coordinate external effects through their own idempotency or delivery guarantees.

### Transactional Outbox
- Use when a committed local state change must reliably cause a message to be delivered outside its transaction boundary.
- Example: commit the state change and an outgoing event in the same local transaction, then let a separate sender deliver committed events.
- Define sender ownership, recovery, ordering, delivery acknowledgement and cleanup. Expect duplicate delivery and pair it with idempotent consumption; storing an outgoing event is not proof of its delivery.

### Saga
- Use when a required operation spans independently committed services or stores and a single local transaction cannot express its consistency requirements.
- Example: coordinate reservation, confirmation and fulfillment as explicit steps with defined compensations for earlier committed effects.
- Record progress and define retries, duplicate handling, compensation failure and resumption. Identify irreversible steps; compensation is a business action, not an automatic atomic rollback.
- Choose explicit coordination or event-driven progression based on actual ownership and coupling. Keep a local operation local when one transaction is sufficient.

### Circuit Breaker
- Use when repeatedly calling a failing dependency can worsen the failure or exhaust resources. Define the relevant failures, opening condition, temporary rejection, recovery probes and closing condition.
- Example: stop accepting calls to a persistently failing service and admit controlled probes before restoring normal traffic. Coordinate with bounded retries to avoid multiplying attempts.

### Bulkhead
- Use when one workload can monopolize resources needed by another. Isolate the relevant execution slots, connections or queue capacity and define fair admission and cleanup.
- Example: separate interactive work from background processing so one cannot exhaust the other's capacity. Size the partitions from resource limits and observed load.

### Backpressure
- Use when producers can outpace consumers. Propagate demand or capacity upstream and keep buffers and in-flight work bounded.
- Example: request another batch only when the consumer has capacity. Define cancellation and how blocked producers are released; account for both queued and active work.

### Load Shedding
- Use when offered work exceeds safe capacity and slowing the producer is insufficient or unavailable. Reject excess work early using explicit priorities and admission rules.
- Example: decline optional refresh work while preserving capacity for accepted critical operations. Define which work may be discarded, communicate rejection and preserve already accepted durable work.

### Memoization and Cache-Aside
- Use memoization for repeated deterministic computations; use cache-aside for on-demand access to a separately owned data source when bounded staleness is acceptable.
- Example: reuse a parsed immutable document by content identity, or refresh a cached lookup after its source changes.
- Define keys, invalidation, eviction and concurrent misses. Apply the common resource ownership and measured-performance rules to the cache.

### Flyweight / Immutable Sharing
- Use when measured memory use comes from many equivalent immutable values. Share their common representation while keeping per-instance mutable state separate.
- Example: share identical immutable descriptors rather than copying them into every item. Define ownership and reclamation so sharing does not become permanent retention.

## Implement with clear contracts
- Prefer the target language's native features, standard library and established project conventions. Reuse existing components before adding new ones.
- Use package- or module-qualified absolute imports, or project-configured aliases. Do not use file-relative or parent-relative imports.
- Apply YAGNI and KISS: every abstraction, option, state and dependency must serve a current requirement. Keep extension points proportional to implemented variation.
- Give each stable rule and configuration value one owner. Share behavior when real consumers need the same domain meaning.
- Define narrow contracts at actual consumer and trust boundaries. Keep transport, persistence and UI representations with their adapters, and deterministic decisions separate from external effects.
- Use precise domain names, export only what callers need and prefer explicit control flow, early returns and visible failure paths.
- Remove dead code, unused exports, duplicate rules, obsolete branches and unnecessary indirection within the changed responsibility.
- Match project formatting and file conventions. Edit source inputs rather than generated output.

## Own state, effects and resources
- Give mutable state an explicit owner and pass results and changes through clear contracts instead of shared mutable globals.
- Make I/O, mutation, cancellation, time, logging and cleanup explicit. Give every worker, subscription, timer, open handle, response body and subprocess an owner and cleanup path.
- Parse untrusted inputs into unambiguous values at their boundary and return actionable failures before mutation.
- Define the atomicity and ordering needed by state changes. Review shared state, lock ordering, reentrancy, publication and cyclic waits for races, deadlocks and duplicate work; keep critical sections small.
- Recheck mutable targets and required authorization at the execution boundary. Bind approval to the operation that will actually execute.
- Propagate cancellation and errors, join owned work and release execution slots on every exit. Preserve accepted results and required ordering during partial failure.
- Inspect uncertain effects before retrying a mutation. Report failures with actionable context and preserve useful error causes.
- Keep credentials private and external sharing limited to the requested integration.

## Select algorithms and control resource cost
- Establish input sizes, hot paths and latency or throughput goals. Review time and space complexity, including worst-case behavior, before choosing algorithms and data structures.
- Consider advanced algorithms when workload scale, complexity analysis or measurements justify them. Prefer proven native or existing implementations and verify their preconditions and tradeoffs.
- Measure a bottleneck before adding caches, parallel execution, pooling or performance dependencies. Compare representative inputs in the actual environment and report observed results.
- Use bounded parallelism for independent work when it improves performance. Wait for prerequisites and serialize overlapping effects or state updates that require ordering.
- Reduce unnecessary allocations, copying, repeated traversal and full materialization. Use streaming or incremental processing when the contract permits it.
- Bound retained working sets, caches, queues and in-flight work according to their purpose. Distinguish input acceptance limits from processing, display and cache budgets.
- Release inactive resources and retained references. Verify cleanup, heap behavior and sustained performance when a change affects long-lived workloads.

## Explain the source in the user's language
- Write authored comments and documentation in the language of the user's latest request unless the user explicitly specifies another language. Preserve identifiers, API names, paths, commands and quoted text.
- At the top of each source or script file you create or modify that supports comments, include a short comment header describing its purpose, main functionality and directly related files with their collaboration roles.
- Use the user's language for header labels and descriptions, and project-relative paths for related files. State when there are no directly related files.
- Place the header after required interpreter, encoding, compiler or license directives. Preserve required legal text and generated-file markers.
- Document functions, methods, classes and behavioral APIs using Google-style structure in the target language's native docstring or documentation-comment syntax.
- Include a concise summary and the applicable parameters, return or yielded values, errors, side effects, constraints and ownership or concurrency assumptions. For classes, describe responsibility, lifecycle and relevant attributes.
- Include only applicable sections and honor documentation tooling requirements. Keep a contract's explanation with its owner rather than copying it across implementations.
- Write for a backend developer with approximately two years of experience. Explain the main flow, important state transitions, boundary conditions, failure handling, synchronization and non-obvious algorithm or memory decisions at their owning locations.
- Focus names and comments on intent and reasoning, and remove explanations that no longer apply.

## Verify serviceability and hand off accurately
- Map verification to the requested stage and acceptance criteria defined before design. Exercise the important end-to-end paths and applicable serviceability requirements, not just isolated components or compilation.
- For a testable bug, reproduce the failure before fixing it when feasible. Prefer focused deterministic tests for contracts and meaningful failure paths; use review and existing checks for wording and mechanical edits.
- For concurrency or durability changes, verify races, ordering, duplicate requests, partial failures, cancellation and recovery where relevant. Distinguish data races from logical consistency errors.
- Use controlled dependencies or disposable environments where appropriate. Identify the setup or external conditions needed for checks that cannot be performed.
- Broaden checks when shared contracts or unresolved risks justify it, and repeat checks after relevant changes or new failures.
- Review the final responsibilities, dependency direction and chosen pattern against the design decision. Remove complexity that no longer serves the completed change.
- Align code, tool descriptions, comments, prompts, setup and operational documentation with the actual behavior.
- Report completed changes, checks actually performed and remaining uncertainty or missing serviceability capabilities separately. Earlier successful checks may be stale after later changes.
- Claim readiness only for the environment and workflows actually supported by the evidence. When the requested criteria are satisfied, finish; otherwise identify the concrete remaining requirement and blocker.
