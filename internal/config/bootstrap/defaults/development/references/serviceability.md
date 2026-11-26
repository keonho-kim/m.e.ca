# Verification and serviceability

Define what the intended user or caller can complete, where the change will run and which supporting capabilities are essential. Trace the relevant entry points, decisions, effects and final observations. For an implementation request, connect those paths so the affected workflow can be exercised.

Select the areas the requested change affects.

| Area | Observable criteria |
| --- | --- |
| Behavior | Important outcomes, boundaries, relevant failure responses and setup required to use them |
| Security and access | Identity, authorization, input handling, exposure boundaries and private data handling |
| Continuity | Required ordering, duplicate handling, consistency, durability and recovery |
| Lifecycle | Startup validation, dependency failures, cancellation, orderly shutdown and owned resource release |
| Operation | Actionable diagnostics, configuration, capacity and behavior when a budget is exhausted |
| Delivery | Supported environment, reproducible build or setup, changed operating procedures and acceptance checks |

Check the actual authorized target before applying side effects. Use explicit ownership and ordering for shared state. Recheck concurrent changes and recover partial or uncertain outcomes before retrying. Avoid holding scarce resources across unrelated waiting. Follow the [backend guide](backend.md) to verify durable or transactional guarantees at the appropriate real persistence boundary. Use the [UI guide](ui.md) to exercise the observed status and recovery in the same workflow.

Use controlled inputs and disposable fixtures where the project supports them. For shared state, check applicable races, logical ordering, cancellation, duplicate effects and recovery. Match evidence to the claim: a build checks compilation, a direct call checks its contract, and a workflow check exercises the connected behavior. State any environment and external prerequisites that limit that evidence.

Revisit the responsibility split, dependency direction and actual verification after implementation. Keep instructions, setup, documentation and emitted behavior aligned. Report the checks run and remaining limitations with the completed outcome.
