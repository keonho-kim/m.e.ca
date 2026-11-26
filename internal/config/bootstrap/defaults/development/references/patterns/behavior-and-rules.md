# Behavior and rules

## GoF behavioral patterns

### Chain of Responsibility
- Use when a request should pass through ordered handlers that either handle it or explicitly delegate to the next handler.
- Example: route a request through the applicable resolvers until one accepts it. Define precedence, the unhandled outcome and which effects may occur before delegation.

### Command
- Use when an operation's intent needs to be stored, queued, ordered or executed separately from its creation. Keep the intended action distinct from its observed outcome.
- Example: represent an edit as an operation that can be scheduled. Add undo only when a correct inverse exists.

### Interpreter
- Use when a small, stable domain language benefits from explicit grammar nodes and evaluation rules.
- Example: evaluate a constrained filter expression. Parse untrusted text before evaluation, bound nesting and evaluation cost, and prefer an established parser when the grammar is substantial.

### Iterator
- Use when consumers need sequential access without depending on a collection's representation or fully materializing it.
- Example: traverse records incrementally. Define ordering, mutation visibility, errors, exhaustion and release of any resources held by the traversal.

### Mediator
- Use when several peers have coordination rules that would otherwise create tangled direct dependencies. Put those interaction rules in one bounded coordinator.
- Example: coordinate validation and enablement across related form fields. Keep each peer's own behavior local and avoid a mediator that becomes the owner of every feature.

### Memento
- Use when state must be captured and later restored without exposing its internal representation to the caretaker.
- Example: retain an editor snapshot for undo. Define snapshot consistency, identity, retention and resource ownership. Restoring memory does not reverse external side effects.

### Observer
- Use when independent consumers need notification of a change. Make subscription ownership, delivery order, error handling and unsubscription explicit.
- Example: notify a display and a recorder after a state change. Use direct calls when consumer ordering is a required dependency. Add messaging infrastructure only for a real delivery requirement.

### State / State Machine
- Use when valid behavior depends on the current state and events. Define states, allowed transitions, guards and terminal outcomes explicitly.
- Example: waiting -> running -> completed, failed or cancelled. Start with explicit values and a transition table or direct control flow. The GoF State form delegates behavior to state objects only when behavior and lifecycle justify them.

### Strategy
- Recognize this as encapsulating interchangeable algorithms or policies behind a common contract. Do not introduce the Strategy pattern or interchangeable policy-object hierarchies. Express variants with straightforward control flow or direct functions.
- Example: select a required calculation with an explicit branch rather than introducing a family of policy objects. Recognizing the pattern does not authorize adding that hierarchy.

### Template Method
- Use when a base operation has a stable sequence and deliberate subclass extension steps. Keep the invariant order and extension contracts explicit.
- Example: specialize a required decoding step within a fixed import procedure. Prefer a direct function with explicit collaborators when inheritance adds no useful constraint.

### Visitor
- Use when a stable set of element types needs several distinct operations without placing every operation on those elements.
- Example: evaluate or format a parsed document tree with separate traversals. Consider the cost of updating visitors when element types change. Prefer ordinary traversal when sufficient.

## Extended rule, state and error approaches

### Specification
- Use when named predicates must be reused or combined with AND, OR and NOT while retaining clear domain meaning.
- Example: combine eligibility and access conditions. Keep evaluation pure, define short-circuiting and useful rejection reasons, and avoid a rule framework for a few direct conditions.

### Hierarchical State Machine (HSM)
- Use when related states share behavior or transitions that a flat state machine would duplicate. Define parent and child states and event propagation explicitly.
- Example: running contains loading and processing while cancellation applies to both. Specify entry and exit order and terminal behavior. Add hierarchy only when the shared behavior warrants it.

### Functional Core / Imperative Shell
- Use when calculations or decisions are entangled with external effects. Keep the core deterministic and return values describing the required effects. Let the outer code perform I/O.
- Example: calculate a document change independently, then let its caller persist the accepted result.

### Result-Based Error Handling
- Use explicit success and failure values for expected fallible operations, following the target language's native error conventions.
- Example: return a parsed value or a meaningful parse failure. Preserve error causes and distinguish cancellation, partial effects and unexpected defects rather than reporting failure as an empty success.

### Railway-Oriented Programming (ROP)
- Use when several fallible transformations share a clear success path and failures should propagate without running later success-only steps.
- Example: parse -> validate -> calculate -> persist, stopping on the first failure. Keep effects and cleanup explicit. Use native error propagation rather than adding combinators or wrappers without a readability benefit.
