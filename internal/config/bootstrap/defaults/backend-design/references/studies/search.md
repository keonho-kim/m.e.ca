# Study: tenant-scoped lexical and vector search

Fictional inputs: one million 768-dimensional float32 vectors, mixed tenants, and queries often narrowed to about 500 eligible documents. Raw vector values alone occupy 3,072,000,000 bytes (about 2.86GiB); IDs, metadata, graphs, versions and replicas are additional. These are assumed shapes, not benchmark results.

## Authority and read representation

```text
documents: tenant/object/version/permission_version/body_ref/deletion
chunks: tenant/document/chunk/content_version/text_ref
vectors: chunk/model_version/dimensions/metric/vector
projection generation: source identity + schema/analyzer/model + committed progress
```

The document/permission authority owns visibility. Indexers build a [read model](../compositions/read-model.md) from versioned source changes. The query operation owns candidate budgets and authorization; rankers consume only appropriate candidates and content. Avoid copying full documents into every chunk, result cache and diagnostic event.

## Compare retrieval arrangements

| Option | Work/cost hypothesis | What decides |
| --- | --- | --- |
| Filter eligible rows, exact vector scan | Around 500×768 coordinate contributions per query before other overhead | Filter/query cost and actual eligible recall/latency |
| Shared ANN followed by eligibility filtering | Fewer distance evaluations in some distributions; graph memory and more candidates may be needed | Recall under restrictive tenants, update churn, scan cap |
| Lexical + ANN/exact candidates + bounded reranking | Better fit for mixed term/semantic queries if quality improves | Extra candidate/scoring/transfer latency and judged relevance |

A small eligible set can favor exact computation even when the global corpus is large. The approximate index must not hide insufficient eligible results. Strict ordering of found neighbors is not exhaustive recall. Use the [vector guide](../performance/vector-retrieval.md) to choose index/operator settings from actual measurements.

## End-to-end query and update

Parse an allowlisted filter → establish tenant/permission scope → obtain bounded lexical/vector candidates → deduplicate IDs → rank/recheck current visibility where required → hydrate selected fields → encode. Separate exact counts/facets when their full-corpus work would dominate first useful results, if the product permits it.

Rebuild a new index generation from a consistent snapshot/change position, apply versioned updates/deletes, catch up and atomically switch. A stale backfill must not resurrect a deleted document. Keep required history and old-generation readers bounded; [projection rebuild](../storage/projection-rebuild.md) owns the protocol.

## Quality and resource acceptance

Compare against exact/judged results for the relevant corpus sample. Slice by tenant, rare/common terms, restrictive filters, new content and revoked access. Measure recall/task quality, p99, candidate count, hydrated bytes, update freshness, index build/merge cost and resident memory together.

Quantization may reduce vector memory but retaining full-precision rerank copies changes total savings. Cache keys bind model, filters, source/permission versions and representation. Warm result reuse cannot override revocation. Add a dedicated search service only when its access features/capacity/isolation earn another projection and operating boundary.
