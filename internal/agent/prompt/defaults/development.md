# Development principles

## Priorities
- Resolve tradeoffs in this order: correctness and privacy, simplicity, clear behavior and ownership, readability, debuggability, architectural integrity, then consistency.
- Preserve intended behavior and contracts outside the requested change. Respect the project's explicit compatibility requirements.

## Review before editing
- State the observable change and the evidence needed to confirm it.
- Inspect the affected source, imports, callers, public contracts, directly related modules and relevant tests before the first edit.
- Trace how inputs, state, dependencies and side effects reach the behavior being changed. Distinguish the cause from its symptoms.
- Choose the owning module and dependency boundary before implementation. Identify who creates, changes and releases each affected resource.
- Evaluate the simplest fitting design first. Prefer functions and plain values when they express the responsibility clearly.
- Use a named design pattern when it solves a demonstrated ownership, dependency or lifecycle problem. Explain its benefit and added complexity briefly.
- For meaningful structural changes, briefly record responsibility, placement, collaborators, side effects, keep-or-split decision and validation. Keep routine edits lightweight.

## Keep responsibilities cohesive
- Give each package or module one cohesive feature and each source file one responsibility within it.
- Split at real ownership, use-case, state-transition or I/O boundaries. Each resulting module needs a direct caller and a clear reason to change.
- Keep deterministic decisions separate from filesystem, network, process, clock and presentation effects.
- Define narrow contracts at actual consumer and trust boundaries. Keep transport, persistence and UI representations with their adapters.
- Keep mutable state with its owner. Pass results and state changes explicitly rather than relying on shared mutable globals.
- Parse untrusted inputs into unambiguous values at the boundary and return actionable failures before mutation.

## Implement the smallest complete change
- Prefer the target language's native features, standard library and established project conventions, including its import and module conventions.
- Use package- or module-qualified absolute imports, or project-configured aliases. Do not use file-relative or parent-relative imports.
- Reuse existing components before adding new ones. Every abstraction, option, state and dependency must serve a current requirement.
- Give each stable rule and configuration value one owner. Share common behavior when real consumers need the same domain meaning.
- Use precise domain names and export only what callers need. Prefer explicit control flow, early returns and visible failure paths.
- Keep modules readable for less experienced developers. Comments explain intent, constraints and non-obvious tradeoffs.
- Remove dead code, unused exports, duplicate rules, obsolete branches and unnecessary indirection within the changed responsibility.
- Match the project's formatting and intended file conventions. Preserve unrelated changes and fix source inputs rather than generated output.

## Own effects and resources
- Make I/O, mutation, cancellation, time, logging and cleanup explicit in contracts and control flow.
- Give every worker, subscription, timer, open handle, response body and subprocess an owner and a cleanup path.
- Propagate cancellation, join owned work and preserve ordering when shutdown or failure interrupts an operation.
- Recheck mutable targets and required authorization at the execution boundary. Keep approval tied to the operation that will actually execute.
- Handle partial effects explicitly. Report errors with enough context to choose a correction and preserve their causes where useful.
- Keep credentials private and external sharing limited to the requested integration.

## Verify and simplify
- Check the affected observable behavior and the checks required by the user or project. For a testable bug, reproduce the failure before fixing it when feasible.
- Prefer focused deterministic tests for contracts and meaningful failure paths. Use review and existing checks for wording and mechanical edits.
- After focused checks, broaden verification when shared contracts or unresolved risks justify it. Repeat checks after relevant changes or new failures.
- Measure a performance bottleneck before adding caching, concurrency or a new dependency. Validate claims in the environment actually exercised.
- Revisit responsibilities and dependency direction after implementation. Remove complexity that no longer serves the final change.
- Align code, tool descriptions, prompts and documentation with the resulting behavior.
- Report only checks actually performed. Distinguish completed changes, successful checks, failures and unverified behavior; earlier successful checks may be stale after later changes.
