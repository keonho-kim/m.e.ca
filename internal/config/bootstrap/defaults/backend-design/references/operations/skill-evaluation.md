# Evaluate the skill's decisions

Use when changing or evaluating this skill, not as a compulsory workflow for every backend task. File parsing, link traversal and bootstrap tests verify delivery; they do not prove useful design reasoning.

## Independent task cases

Give the evaluator the task, skill and relevant project facts, without the acceptance notes below. Keep all execution within authorized resources. These cases may be assessed manually; do not report independent/model evaluation unless it actually ran.

| Task prompt | Important evidence to inspect afterward |
| --- | --- |
| Implement rejection of stale approval updates in an existing backend. | Runs a focused behavioral test showing the missing guard before implementation, makes it pass, then refactors with relevant checks green; setup failures are not reported as Red |
| Design the same approval contract without implementing it. | Defines invariants, acceptance cases and test boundaries while preserving the design-only scope |
| Design a one-host records app with metadata pages, large bodies and short write bursts. No multi-host requirement. | Considers direct module/SQLite or another justified authority; projects narrow rows; does not require Redis/broker by habit |
| Redesign the same app for multiple independent writers and strict read-after-write. | Changes authority/freshness assumptions; accounts for pool/concurrency and cache consistency |
| Design a reservation with external payment; the client may lose the response after success. | Local invariant, durable command/step identity, unknown outcome, reconciliation and bounded retention |
| Build an index while documents are edited/deleted, then resume clients from a cursor. | Consistent snapshot/change boundary, versions/tombstones, generation switch; no ID-allocation-as-commit assumption |
| Add interchangeable HTTP/provider access while following the development prompt. | Narrow adapters and explicit selection; no Strategy hierarchy or generic wrapper around every function |
| Centralize diagnostics for HTTP and workers. | One catalog for codes/messages/events/fields, preserved causes, transport mapping, correct terminal owner and bounded sink behavior |
| Design encrypted document storage with KMS outage and old-backup recovery. | Threat/DEK scope, key-service cost, publication uncertainty, rotation distinction and usable restore |
| Improve a large parser path in Node/Go/JVM/Rust, with cancellation. | Uses target-runtime semantics, bounds real work and retained bytes; does not confuse caller cancellation with worker completion |

## Compare judgment, not wording

Check whether the result connects requirements to responsibilities, atomicity, input/state representation, calls/copies, resource budgets and failure outcomes. Alternatives should change when the facts change. Omitting an unnecessary pattern is a successful choice; naming many patterns is not a quality signal.

For each case, record concrete supported/missing/incorrect decisions and their observable consequences. Treat a violated invariant, credential exposure, invented benchmark or unauthorized effect as material regardless of presentation quality. Evaluate focused reference selection: an agent should not load the entire collection or ask the user to choose a subcommand.

Run paired cases with the same model/settings/context to assess a skill revision when feasible, and record variance rather than attributing one answer to the skill alone. Keep prompts, artifacts, model/configuration and human assessment separate from runtime unit-test evidence. External model/API calls and disruptive experiments require their normal authorization; this file does not grant it.
