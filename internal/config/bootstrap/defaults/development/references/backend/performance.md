# Performance decisions

Optimize useful completed work under the agreed quality and failure contract. A faster microbenchmark is insufficient when memory retention, index size, maintenance or tail latency worsens.

| Evidence | Read |
| --- | --- |
| Planning work admission, tenant fairness and expensive-job scheduling | [Admission and scheduling](performance/admission-and-scheduling.md) |
| Mapping the design to Go, Node, JVM or Rust execution/cancellation | [Runtime boundaries](performance/runtime-boundaries.md) |
| Too many rows/round trips, deep pagination, repeated joins or redundant hydration | [Query paths](performance/query-paths.md) |
| Text search, filters, facets, top-k ranking, Korean analysis or huge candidate sets | [Search and ranking](performance/search-and-ranking.md) |
| Embeddings, ANN filtering, hybrid search, quantization and reranking | [Vector retrieval](performance/vector-retrieval.md) |
| Heap growth, repeated encoding, queue buffers, copies or I/O stalls | [Memory and I/O](performance/memory-and-io.md) |
| Choosing a benchmark, proving a result or rejecting an optimization | [Measurement](performance/measurement.md) |

Use this escalation order as a starting hypothesis, not a law: remove unnecessary work → reduce candidate/byte movement → choose the access path/data representation → bound and batch work → exploit locality → consider specialized parallelism/native execution. Reorder it when the profile identifies a different bottleneck.

Any approximation needs a visible quality contract. Exact authorization, financial invariants and durable identity cannot inherit a search engine's approximation. Any accelerator needs update, eviction, failure and rebuild ownership.

A new design should state the assumed workload and cost hypothesis; existing code should use measurements to refine it. Name the candidate mechanism, expected asymptotic or constant-factor change, memory/storage/write cost, regression cases and a reproducible acceptance condition. Use [worked comparisons](design-studies.md) to connect the cost to a concrete arrangement. Prefer the storage engine's existing algorithms before writing an application-level replacement. [Source families](sources/performance.md) document mechanisms, not portable speedup promises.
