# Design pattern selection

Start with a direct function or cohesive module. Give a pattern a role only when a requirement has an actual caller, collaboration, state transition or effect that needs it. Consider a small set of fitting options and compare correctness, coupling, resource cost, failure behavior and verifiability. Use native language facilities for the chosen structure.

| Situation | Read |
| --- | --- |
| Resource construction, incompatible interfaces or composing behavior | [Construction and composition](patterns/construction-and-composition.md) |
| Dispatch, state transitions, notifications or fallible rules | [Behavior and rules](patterns/behavior-and-rules.md) |
| Component state, event delivery, ownership and shared runtime resources | [State and resources](patterns/state-and-resources.md) |

The references cover object design, module composition and concurrency at different scopes. For transactions and repositories, outbox and saga coordination, or service overload and failure isolation, use the [backend guide](backend.md). For cache design and other resource tradeoffs in general code, see the [algorithms and optimization map](algorithms.md). Confirm that a structural choice preserves the state and completion semantics of the [application workflow](applications.md).
