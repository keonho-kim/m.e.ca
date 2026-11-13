# Primary source index

The topic references are original engineering synthesis; linked manuals, papers and articles are not copied into this bundle. Read the relevant original before depending on a version-specific guarantee or implementing cryptographic policy.

| Source family | What it supports |
| --- | --- |
| [Pattern and verification sources](sources/architecture.md) | Relationship to the default development prompt, canonical pattern terminology and controlled failure testing |
| [Database sources](sources/data.md) | SQLite concurrency/planning/backup; PostgreSQL transactions/plans/maintenance and pooling |
| [Cache and Redis sources](sources/cache.md) | Durability, memory, invalidation, leases, messaging and batching |
| [API and execution sources](sources/api.md) | Protocol standards, authentication, idempotency, publication, file transfer and authorization consistency |
| [Security sources](sources/security.md) | Cryptographic primitives, key lifecycle, searchable encryption and specialized privacy techniques |
| [Performance sources](sources/performance.md) | Retrieval algorithms, vector indexes, data representations, profiling and load measurement |
| [Operations sources](sources/operations.md) | Production overload/retry/failure practices, diagnostics specifications, telemetry and migrations |

## Interpret evidence correctly

- **Standards:** distinguish Standards Track, Best Current Practice and Informational/IRTF RFCs. A published algorithm does not automatically standardize a complete deployment protocol.
- **Product documentation:** guarantees depend on version, configuration, driver and service tier. `current`, `latest` and repository default branches move. Record the actual deployed versions when applying the guidance.
- **Research and company reports:** useful mechanisms and observed tradeoffs; their benchmarks and scale describe their systems. They are not evidence of this project's throughput, security or reliability.
- **This skill's recommendations:** workload-dependent synthesis. The reference's tests and decision conditions explain how to validate them; they are not additional requirements of an external standard.

The [composition map](compositions.md) turns relevant development-pattern principles into concrete roles and execution paths. The [design studies](design-studies.md) use fictional workloads and arithmetic; neither is evidence of measured production performance. The [evaluation cases](operations/skill-evaluation.md) specify a reasoning assessment, not a report that model evaluation has already run.

Confirm specification maturity, errata and SDK support for the versions used by the target service. Moving documentation aliases do not define deployment requirements.

For cryptographic material, inspect algorithm limitations, maintained implementation guidance and applicable module validation. A library's use of an algorithm does not by itself establish a validated cryptographic module. Publication metadata and announcements identify source scope; read the full specification or paper before treating it as implementation evidence.
