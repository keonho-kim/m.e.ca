# Vector and hybrid retrieval

Use when semantic retrieval earns the embedding/indexing cost. Establish an exact-search baseline, query set and quality metric before choosing ANN. Record corpus/tenant sizes, filter selectivity, update rate, dimensions, metric, model version and hardware.

## Choose the index from the cost envelope

| Option | Potential fit | Added cost |
| --- | --- | --- |
| Exact flat scan | Small eligible corpus, strict recall, batched/vectorized execution | Distance work proportional to eligible vectors |
| HNSW | Repeated low-latency queries with sufficient memory | Graph memory/builds, update/deletion behavior and tuning |
| IVF | Partitioned candidate search and tunable probing | Training, distribution drift and missed partitions |
| Scalar/product/binary quantization | Memory/bandwidth is limiting | Quantization error and candidate reranking |
| Disk-backed specialized index | Corpus exceeds memory | Random I/O, cache locality, rebuild and deployment cost |

Faiss's [index catalog](https://github.com/facebookresearch/faiss/wiki/Faiss-indexes) and [selection guide](https://github.com/facebookresearch/faiss/wiki/Guidelines-to-choose-an-index) document concrete index tradeoffs. Raw float32 vectors alone need roughly `4 × dimensions × count` bytes, excluding IDs, graphs, metadata, replicas and working buffers. Calculate those additions before choosing an index.

## Filters can dominate recall and latency

A tiny authorized subset may be cheaper to fetch and scan exactly than to traverse a shared ANN index then discard most results. With post-filtering, requesting k global neighbors does not establish k eligible neighbors. pgvector's [filtering/iterative-scan guidance](https://github.com/pgvector/pgvector) describes bounded scan expansion and strict/relaxed order behavior; check the deployed extension version.

Compare selective exact filters, suitable partial indexes/partitions, oversampling and iterative search using measured eligible recall. Cap extra candidates/time/memory and report partial results deliberately. A strict ordering of found neighbors is not a proof of exhaustive global recall. Tenants can affect one another's candidate search even when returned rows are correctly authorized.

## Build a complete retrieval pipeline

Keep metric/normalization consistent across embeddings, index and query. A model/dimension change requires a separate index generation and cutover, not silent insertion into an incompatible space. Do not persist redundant original text in every chunk, vector row, event and result cache unless it has an independent retention/access need.

For hybrid search, generate bounded lexical and semantic candidates, deduplicate stable document/chunk IDs, combine ranks/scores with a documented method, then rerank a bounded set. Different score scales require calibration; a simple sum is not automatically meaningful. Hydrate only authorized selected content. Reranker batching can improve throughput while adding queue latency and memory.

Quantized candidate generation followed by higher-precision reranking trades memory/bandwidth for extra computation. Retaining both representations can erase storage savings. Embedding/result caches bind model, normalization, filters, tenant and content version; a cache hit must not bypass permission or deletion checks.

## Validate quality under change

Measure recall@k or another appropriate judged metric, nDCG/task success, eligible result count, p95/p99, bytes/vector, build time, update freshness and restart warm-up. Slice by restrictive filters, rare queries, smallest/largest tenants and recent deletes. Compare against exact results on a tractable representative sample.

Load tests must include ingestion, deletion/vacuum/merge, index rebuild, cold start and failure recovery. A faster ANN query with unacceptable quality or unbounded catch-up is not a successful backend optimization.
