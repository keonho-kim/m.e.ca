# Lexical search, filters and top-k

Design from the query language, corpus, update rate, authorization and result-quality target. Separate candidate generation, eligibility filtering, ranking, top-k selection, hydration and optional totals/facets. Each stage has a different optimization and consistency boundary.

## Fit the retrieval structure

| Work shape | Candidate | Tradeoff |
| --- | --- | --- |
| Equality/range and ordered lookup | Database B-tree | Ordered maintenance and key size |
| Terms/phrases over documents | Inverted index | Tokenization, postings storage and update lag |
| Short/fuzzy/substrings | Trigram/specialized analyzer | Index expansion, short-query selectivity and false candidates |
| Repeated set intersections | Sorted postings/bitmaps | Density, ID universe and update cost |
| Small top-k from many scored candidates | Bounded heap/engine top-k | O(n log k) selection instead of sorting all n; scoring still costs |
| Many negative key probes | Coherent Bloom/Ribbon filter | False positives, memory/build cost and versioned completeness |

SQLite [FTS5](https://sqlite.org/fts5.html) offers tokenizers, ranking and content-storage choices; an external-content index needs reliable synchronization/rebuild. PostgreSQL [text indexes](https://www.postgresql.org/docs/current/textsearch-indexes.html) and [pg_trgm](https://www.postgresql.org/docs/current/pgtrgm.html) solve different access patterns. Do not assume substring and natural-language relevance share one index.

For Korean content, evaluate morphological segmentation, compounds, spacing, normalization and mixed Korean/Latin identifiers with representative queries. [Nori](https://www.elastic.co/docs/reference/elasticsearch/plugins/analysis-nori) is a concrete Korean analyzer, not a reason by itself to add Elasticsearch. Version analyzer/dictionary changes and plan reindexing.

## Reduce candidates without changing the promised result

Intersect selective eligibility sets early where the engine permits it; choose sparse arrays, dense bitsets or compressed Roaring representations by actual density and operations. [Roaring's guidance](https://github.com/RoaringBitmap/RoaringBitmap) explains why compressed bitmaps are not always the smallest/faster choice. Reuse engine/library implementations before creating a custom search subsystem.

Block-Max WAND uses score upper bounds to skip blocks that cannot enter top-k under supported ranking assumptions. It can retain exact top hits while avoiding full evaluation; exact total counts or aggregations can remove the saving. Elastic's [implementation account](https://www.elastic.co/blog/faster-retrieval-of-top-hits-in-elasticsearch-with-block-max-wand) is mechanism evidence, not a portable speedup promise. Make exact count/facet requirements explicit instead of paying for them by default.

Bloom/Ribbon filters can avoid absent-key I/O in an LSM path; [RocksDB's filter documentation](https://github.com/meta/rocksdb/wiki/RocksDB-Bloom-Filter) describes construction and memory tradeoffs. A negative answer is safe only relative to the versioned data actually indexed. Stale filters cannot authoritatively reject newly committed objects.

## Include lifecycle and isolation

Version documents and deletion tombstones; ensure delayed updates cannot resurrect deleted content. Apply access policy to hits, snippets, facets, suggestions and counts. A shared candidate index must not expose other tenants through metadata or result caching. Bound wildcard/fuzzy expansions, clauses, candidate count, reranking and execution time.

Evaluate relevance and latency together on typo/rare/common/adversarial queries, tenant skew and concurrent ingestion. Include index bytes, merge/rebuild I/O, freshness and cold-cache behavior. Use [search tuning guidance](https://www.elastic.co/docs/deploy-manage/production-guidance/optimize-performance/search-speed) for the selected engine, without importing its defaults into another architecture.
