# Architecture decisions

Choose the reference that resolves the current uncertainty. The complete collection is not a required reading list.

| Situation | Read | Decision to leave with |
| --- | --- | --- |
| New service or uncertain performance requirements | [Workload and budgets](architecture/workloads.md) | Explicit invariants, load distribution and capacity assumptions |
| Handlers, business rules, storage and logging are entangled | [Ownership and boundaries](architecture/ownership.md) | One owner for each decision, effect and lifecycle |
| Replication, sharding, cells or several regions are proposed | [Distribution](architecture/distribution.md) | Which guarantee or failure boundary justifies distribution |
| Translating a product into a backend | [Product scenarios](architecture/scenarios.md) | A fitting data/consistency/execution shape, with omitted machinery explained |

Start by tracing one important mutation and one expensive read. Show the commit boundary, subsequent publication, authorization checks and any retained representation. A diagram is useful only if those relationships become clearer.

Use the [composition map](compositions.md) to turn the boundary into concrete collaborators and call/dependency paths. The [design studies](design-studies.md) compare complete schema/flow/cost alternatives. Pattern selection follows the requirement; it does not begin with filling a layer diagram.

Separate three questions: what is authoritative, which code owns the rule, and where it runs. They can have different answers. Several modules may share one transaction without becoming separate network services; moving a module to a worker changes failure and delivery semantics.

For a proposed boundary, identify its real consumer and benefit: independent scaling, isolation, security, deployment or transaction ownership. Add the new network hop, timeout, deployment skew and operational ownership to the comparison. A directory named after a pattern does not establish a boundary.

Use the [research record](sources.md) to distinguish source guarantees from this bundle's synthesis. Company scale and component count are not evidence of suitability.
