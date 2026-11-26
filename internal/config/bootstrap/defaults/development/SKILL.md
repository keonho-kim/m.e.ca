---
name: development
description: Plan, design, implement, debug, optimize or review software and interfaces. Connect application workflows, UI, backend architecture, algorithms and data structures through clear responsibilities, state and verification. Includes single-agent and team development.
license: Apache-2.0. Bundled materials retain their notices in LICENSE.txt and NOTICE.md.
---

# Software development

Start with the user's intended outcome and the requested stage. Inspect the affected code, interfaces, dependencies and tests. Trace the input and effects to their owners, including the visible result and recovery. Use this entrypoint to choose the design direction and the references the current decision needs.

## Connect the workflow and its owners

1. Identify the actors, relevant records, required behavior and observable acceptance criteria. Establish the workload and operating constraints that affect the design.
2. Follow the action through presentation, application decisions and effects. Name who owns each state transition, how success becomes authoritative and what callers observe while it is pending or fails.
3. Choose a fitting structure and representation. Connect the same identity, permissions, ordering and completion meaning across the affected boundaries. Apply the relevant guides below to resolve a concrete question.
4. Implement and verify a complete acceptance slice. Revisit the other affected boundaries when a decision changes their contract.

## Choose the design

- Prioritize correctness and privacy, simplicity, clear ownership, readability, debuggability and architectural consistency. Use the smallest design that satisfies the required behavior.
- Start with a function or cohesive module. Give each stable rule, mutable state and side effect one clear owner. Introduce boundaries where callers, independently changing responsibilities or external effects need them.
- Keep interfaces narrow and owned by their consumers. Keep transport, storage and UI representations with their adapters. Use native functions, explicit selection and direct construction for the variations present today.
- For a substantial change, record the responsibility, owning module, collaborators, state and effects, any planned split and how to verify the design. For a review, connect each finding to observable consequences at the affected boundary.

## Read the guidance the decision needs

For a focused question, search the reference library with `meca develop`. This MECA subcommand runs the bundled [search script](scripts/search.sh) through the existing command execution path. Use two to five concrete English keywords describing the operation or symptom. For the installed `meca:development` skill, run

```sh
meca develop "delayed task queue"
```

The command uses `sh`, `awk`, `find` and `sort`. It reads the current reference files and returns up to ten paths, line numbers and headings. Matching is case-insensitive and literal. Results favor sections matching more query terms, then terms in headings and paths. Each file contributes its best section, with matched-term counts shown for partial results. Refine weak queries with concrete synonyms or use the topic map below.

Read selected results with `read_skill_file`, using the loaded skill ID and only the returned path. Line numbers help locate the section after reading. For a project or shared copy, read its script and run `sh <actual-skill-root>/scripts/search.sh "keywords"` to search that copy. The installed MECA executable is `~/.local/bin/meca`.

| Decision | Read |
| --- | --- |
| Define a feature's actors, records and complete workflow | [Application workflows](references/applications.md) |
| Choose interface structure, controls, feedback and motion | [UI design](references/ui.md) |
| Design backend authority, transactions, APIs, diagnostics and recovery | [Backend design](references/backend.md) |
| Recognize and select a design pattern | [Pattern guide](references/patterns.md) |
| Choose algorithms and data structures or investigate performance | [Algorithms and optimization](references/algorithms.md) |
| Coordinate a design and implementation among MECA teammates | [Team workflow](references/team-workflow.md) |
| Define operational requirements and verify a change end to end | [Verification and serviceability](references/serviceability.md) |

Use `read_skill_file` with the ID returned by `load_skill`. Paths are relative to this skill root. Topic guides link to focused maps and resources. Read the selected material for the current decision and reuse it within this request.

For a booking feature, use the application guide to establish a reservation's identity and capacity rule. Carry those facts into the UI's pending, confirmed and conflict states, then into the backend's concurrency and retry contract. If availability lookup is expensive, use the algorithm guide with the same capacity and freshness requirements. Verify the whole action through its visible result.

For a small local refactor, start with the affected responsibility and its relevant pattern or algorithm guide. For a UI-only review, start with the UI guide and inspect other boundaries when their behavior affects a finding. Team implementation also uses the team workflow. Match the reading and verification to the requested scope.

## Implement the behavior

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
