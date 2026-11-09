# Research record

Primary sources checked on **2026-11-09**. The topic references are original engineering synthesis; linked manuals, papers and articles are not copied into this bundle. Read the relevant original before depending on a version-specific guarantee or implementing cryptographic policy.

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

At this reading, PostgreSQL's `/docs/current/` pages identified PostgreSQL 18; OpenAPI's `latest` identified 3.2.1. Those observations are research context, not requirements to install or upgrade. OpenTelemetry's general recording-errors convention was marked **Development**; confirm the status of conventions and language SDKs before freezing a public telemetry contract.

For cryptographic material, inspect current errata, algorithm limitations and maintained implementation guidance. A source's publication date and this check date have different meanings. NIST's final ML-KEM page includes a potential-errata note. Do not infer that a library using an algorithm is a validated cryptographic module.

The FIPS 204/205 detail pages and the publisher-hosted full text of *The Tail at Scale* were unavailable in this environment. Post-quantum scope is supported by NIST's accessible announcement and ML-KEM publication; the Tail entry is publication metadata only. The skill does not treat those inaccessible full texts as reviewed implementation evidence.
