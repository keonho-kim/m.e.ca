# Worked design studies

These fictional studies connect a product operation to schema, responsibility, effect order and resource cost. Their numbers are assumptions and arithmetic, not benchmarks or defaults. Choose the closest case and derive the target project's own design.

| Product pressure | Study | Compare |
| --- | --- | --- |
| List/detail latency, bursts and canonical documents | [Workspace records](studies/workspace.md) | SQLite/local authority; PostgreSQL/shared authority; optional cache |
| Scarce inventory and an external payment | [Reservation and payment](studies/reservation.md) | One local transaction; durable external workflow; compensation |
| Tenant-scoped lexical/vector queries and index updates | [Search service](studies/search.md) | Filtered exact search; ANN; materialized search projection |
| Large encrypted documents, KMS latency and key rotation | [Encrypted documents](studies/encrypted-documents.md) | DEK scope, bounded reuse, streaming and publication/recovery |

Read one study as a complete operation, then open only the technical references its unresolved decisions need. A study is not a second source of policy: the topic references own mechanism guarantees and failure contracts. Keep arithmetic and assumptions visible when adapting a case.

A useful result identifies which requirement changed the choice. For example, moving from one host to several independent writers changes authority requirements; a larger row count alone may only call for a better access path. If several alternatives satisfy the workload, explain the simplicity/operating-cost choice instead of claiming a universal winner.
