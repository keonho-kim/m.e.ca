---
name: development
description: Engineering guidance for reasoning about software and interfaces, choosing designs, implementing changes, diagnosing behavior and reviewing tradeoffs. Connects application workflows, design patterns, algorithms, data structures, UI and backend references as needed by the user's request.
license: Apache-2.0. Bundled materials retain their notices in LICENSE.txt and NOTICE.md.
---

# Software development

Use this reference library to resolve the engineering questions in the user's request. Establish what result would help them now and which decision needs guidance. Apply the relevant knowledge to that result; a workflow in a reference is not an instruction to carry out additional work.

## Frame the engineering decision

- Ground the answer in the evidence the question needs: the user's scenario, supplied code, the current project or a measured workload. Inspect files, interfaces, dependencies and tests when the conclusion depends on their actual state.
- For a workflow or design decision, connect required behavior, actors, records, state ownership and operating constraints. Trace affected boundaries only as far as needed to understand the result and its tradeoffs.
- For a change, establish observable acceptance criteria and the smallest complete scope that meets them. For findings or recommendations, connect the evidence to consequences and identify uncertainty.
- Use the selected techniques to fulfill the requested outcome. The user may want to understand or compare an approach, apply it themselves, or have the agent implement it; follow that intent through the work and its completion.

## Choose the design

- Prioritize correctness and privacy, simplicity, clear ownership, readability, debuggability and architectural consistency. Use the smallest design that satisfies the required behavior.
- Start with a function or cohesive module. Give each stable rule, mutable state and side effect one clear owner. Introduce boundaries where callers, independently changing responsibilities or external effects need them.
- Keep interfaces narrow and owned by their consumers. Keep transport, storage and UI representations with their adapters. Use native functions, explicit selection and direct construction for the variations present today.
- For a substantial change, record the responsibility, owning module, collaborators, state and effects, any planned split and how to verify the design. For a review, connect each finding to observable consequences at the affected boundary.

## Read the guidance the decision needs

When a decision needs a reference that is not already supplied, search with `meca develop` or use the topic map below. The command runs the bundled [search script](scripts/search.sh). Choose two to five concrete English keywords describing the operation or symptom. In MECA, pass this function body to `codemode`:

```javascript
return await tools.execute_command({
  reason: "Find scheduling guidance",
  command: 'meca develop "delayed task queue"'
});
```

The command uses `sh`, `awk`, `find` and `sort`. It reads the current reference files and returns up to ten paths, line numbers and headings. Matching is case-insensitive and literal. Results favor sections matching more query terms, then terms in headings and paths. Each file contributes its best section, with matched-term counts shown for partial results. Refine weak queries with concrete synonyms or use the topic map below.

Read selected results through `tools.read_skill_file({id, path})` inside `codemode`, using the loaded skill ID and returned path. Line numbers locate the section within that resource. For a project or shared copy, read its script and invoke `sh <actual-skill-root>/scripts/search.sh "keywords"` through `tools.execute_command`. The installed MECA executable is `~/.local/bin/meca`.

| Decision | Read |
| --- | --- |
| Define a feature's actors, records and complete workflow | [Application workflows](references/applications.md) |
| Choose interface structure, controls, feedback and motion | [UI design](references/ui.md) |
| Design backend authority, transactions, APIs, diagnostics and recovery | [Backend design](references/backend.md) |
| Recognize and select a design pattern | [Pattern guide](references/patterns.md) |
| Choose algorithms and data structures or investigate performance | [Algorithms and optimization](references/algorithms.md) |
| Coordinate a design and implementation among MECA teammates | [Team workflow](references/team-workflow.md) |
| Define operational requirements and verify a change end to end | [Verification and serviceability](references/serviceability.md) |

Use the ID supplied by the engine or returned by `tools.load_skill`. Reference paths passed to `tools.read_skill_file` are skill-root-relative; resolve links against the containing document's directory. Read selected material at its `needs_model` boundary before generating dependent work, and reuse it for the current decision.

For a booking feature, use the application guide to establish a reservation's identity and capacity rule. Carry those facts into the UI's pending, confirmed and conflict states, then into the backend's concurrency and retry contract. If availability lookup is expensive, use the algorithm guide with the same capacity and freshness requirements. Verify the whole action through its visible result.

For a small local refactor, start with the affected responsibility and its relevant pattern or algorithm guide. For a UI-only review, start with the UI guide and inspect other boundaries when their behavior affects a finding. Team implementation also uses the team workflow. Match the reading and verification to the requested scope.

## Carry out requested changes

When implementation is part of the user's requested outcome, deliver a complete acceptance slice and revisit boundaries whose contracts it changes.

- Reuse the project's native features, established conventions and relevant components. Keep dependencies explicit, control flow readable and errors actionable. Parse inputs at the boundary and edit source inputs for generated files.
- Make mutation, I/O, ordering, cancellation and cleanup visible. Give each owned resource a release path. Use deterministic facts and controlled dependencies to verify decisions and effects.
- Follow the project's import, file header and documentation conventions. Where the project leaves the choice open, use module-qualified imports and brief headers that describe purpose and direct collaborators. Explain API contracts, ownership and non-obvious decisions in the user's language unless another language is specified.
- Keep existing behavior and unrelated edits intact. Simplify within the changed responsibility and remove rules, dependencies or indirection made redundant by the change.

## Verify each behavior

Use **Red → Green → Refactor** for a testable behavior change.

1. **Red.** Write a focused behavioral test from an acceptance criterion or invariant. Run it before implementing the behavior and confirm the intended failure. Resolve unrelated setup, dependency or tool failures as such.
2. **Green.** Implement the smallest correct behavior that passes. Cover relevant success, boundary and failure cases over focused iterations.
3. **Refactor.** Simplify the owning code with the checks still passing. Exercise affected entry points and broaden verification when a shared contract changes.

For design or review, identify the evidence and applicable acceptance cases. For mechanical changes, use review and existing checks. When automation is impractical, exercise the observable behavior with the available tools and report its limits. Report completed work, checks actually run and any remaining uncertainty.
