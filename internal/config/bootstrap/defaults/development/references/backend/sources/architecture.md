# Pattern and verification provenance

The compositions are original backend arrangements within the development skill. The [entrypoint](../../../SKILL.md) owns the shared workflow and the [pattern guide](../../patterns.md) owns general structure selection. Backend references own persistence, delivery and operation details.

The shared principles are direct functions/modules first, actual consumer-owned boundaries, native language facilities, explicit effects and cleanup, and measured resource decisions.

| Primary source | Scope and reason to consult |
| --- | --- |
| [Fowler: Transaction Script](https://martinfowler.com/eaaCatalog/transactionScript.html) | Request-oriented procedural business logic; does not require a service framework |
| [Fowler: Unit of Work](https://martinfowler.com/eaaCatalog/unitOfWork.html) | Canonical change tracking/coordination terminology, distinct from merely using a DB transaction |
| [Fowler: CQRS](https://martinfowler.com/bliki/CQRS.html) | Separate read/write conceptual models and the complexity of broader adoption |
| [Microsoft: saga](https://learn.microsoft.com/en-us/azure/architecture/patterns/saga) | Independently committed steps, orchestration/choreography and compensation tradeoffs |
| [FoundationDB: simulation/testing](https://apple.github.io/foundationdb/testing.html) | Controlled reproducible failure histories in a system built for deterministic simulation |

GoF names in the composition map aid recognition; they are not a checklist or claims that every native function realizes a particular class-based GoF structure. Worked studies are fictional reasoning exercises with explicit arithmetic, not copied production systems or measured benchmarks. The diagnostic catalog example is a local design policy, not a required external-standard schema.

The source catalogs for databases, APIs and operations supply the concrete transaction/protocol guarantees used by the compositions. Correct terminology alone does not prove that a composition preserves those guarantees. Verify the actual implementation and selected versions.
