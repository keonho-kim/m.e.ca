# Performance primary sources

Checked 2026-11-09. Repository/default-branch documentation moves; record an exact package/extension revision when implementing. Published benchmark results belong to their workloads. These sources support the [performance map](../performance.md).

## Retrieval and representations

| Source | Use for |
| --- | --- |
| [pgvector](https://github.com/pgvector/pgvector) | Exact/ANN choices, filter behavior, iterative scans and index lifecycle |
| [Faiss index catalog](https://github.com/facebookresearch/faiss/wiki/Faiss-indexes) | Storage/accuracy tradeoffs across flat, graph, IVF and quantized indexes |
| [Faiss selection guide](https://github.com/facebookresearch/faiss/wiki/Guidelines-to-choose-an-index) | Index choice from size, training, reuse and hardware constraints |
| [RoaringBitmap](https://github.com/RoaringBitmap/RoaringBitmap) | Compressed-set representation, density and intersection tradeoffs |
| [RocksDB Bloom/Ribbon filters](https://github.com/meta/rocksdb/wiki/RocksDB-Bloom-Filter) | Negative-probe I/O, filter memory and construction cost |
| [Block-Max WAND account](https://www.elastic.co/blog/faster-retrieval-of-top-hits-in-elasticsearch-with-block-max-wand) | Score-bound pruning and exact-total/aggregation cost |
| [Elasticsearch pagination](https://www.elastic.co/docs/reference/elasticsearch/rest-apis/paginate-search-results) | Deep-page memory, tie-breakers, `search_after` and point-in-time |
| [Elasticsearch search tuning](https://www.elastic.co/docs/deploy-manage/production-guidance/optimize-performance/search-speed) | Query shape, shards, filesystem cache and indexing/search tradeoffs |
| [Nori](https://www.elastic.co/docs/reference/elasticsearch/plugins/analysis-nori) | Concrete Korean morphological analysis and matching deployment requirements |

SQLite FTS5 and PostgreSQL text/trigram/index documentation are in the [database source catalog](data.md). Faiss and pgvector expose mechanisms with different runtime/operational models; their presence here does not prescribe a dependency.

## Runtime, profiling and workload evidence

| Source | Use for |
| --- | --- |
| [Go GC guide](https://go.dev/doc/gc-guide) | Allocation, live-memory and soft-limit tradeoffs |
| [Go diagnostics](https://go.dev/doc/diagnostics) | CPU, memory, blocking and execution-trace tools |
| [Go PGO](https://go.dev/doc/pgo) | Profile representativeness and compiler feedback |
| [USE method, original author](https://www.brendangregg.com/usemethod.html) | Resource utilization, saturation and errors |
| [Off-CPU analysis, original author](https://www.brendangregg.com/offcpuanalysis.html) | Blocked time and context complementary to CPU sampling |
| [k6 open/closed workload models](https://grafana.com/docs/k6/latest/using-k6/scenarios/concepts/open-vs-closed/) | Arrival-rate behavior and coordinated omission |
| [Envoy adaptive concurrency](https://www.envoyproxy.io/docs/envoy/latest/configuration/http/http_filters/adaptive_concurrency_filter.html) | Latency feedback, minimum-RTT sampling and jitter; `latest` may be a development build |
| [Node event-loop guidance](https://nodejs.org/en/learn/asynchronous-work/dont-block-the-event-loop) | Blocking CPU/serialization and worker-transfer costs; historical timings are not current benchmarks |
| [Node worker API](https://nodejs.org/api/worker_threads.html) | Current worker/message/buffer lifetime contracts |
| [JDK 25 virtual threads](https://docs.oracle.com/en/java/javase/25/core/virtual-threads.html) | Waiting concurrency versus CPU/resource capacity and runtime-specific behavior |
| [Tokio spawn_blocking](https://docs.rs/tokio/latest/tokio/task/fn.spawn_blocking.html) | Blocking execution, admission and started-task cancellation limits |

Go is a concrete runtime example, not this skill's required implementation language. Choose the equivalent supported profiling, lifecycle and compiler tools for the target project. No profiling tool, package, kernel facility or benchmark runner is installed by this skill.
