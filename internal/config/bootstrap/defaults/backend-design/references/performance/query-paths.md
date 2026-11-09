# Query paths: avoid scanning and hydration work

Use while designing list/detail/search/export operations, or when a profile shows database, round-trip or materialization cost. Specify the exact result and ordering before optimizing its access path.

## Bound the work before the expensive stage

Select authorized candidate IDs and narrow fields, then hydrate only the required bounded result. Avoid selecting blobs/history solely to compute counts, list labels or existence. Push filtering into the authority where it preserves semantics; a remote full scan followed by application filtering still pays transfer and decoding costs.

Compare one set-based join/batch with N+1 calls and with a huge denormalized query. The goal is fewer visited rows/bytes/round trips, not the fewest SQL statements at any cost. A join that multiplies wide rows can cost more than two bounded queries. Request-local memoization is useful only when keys and values have actual reuse.

## Design pagination as an access path

For a non-null descending order `(created_at, id)` inside one tenant, an illustrative next page is:

```sql
SELECT id, created_at, title
FROM documents
WHERE tenant_id = $1
  AND (created_at, id) < ($2, $3)
ORDER BY created_at DESC, id DESC
LIMIT $4;
```

This assumes a bounded page size, unique stable tie-breaker, matching filters/index and a separately handled first page. Nulls, mixed sort directions, collation, changed sort fields and live deletions need explicit semantics. Bind the cursor to the authenticated scope and normalized query; a cursor is not authorization. Keyset pagination does not create a stable snapshot across requests.

Offset is acceptable for genuinely small bounded result sets. For deep pagination, avoid computing and discarding all preceding rows; PostgreSQL's [SELECT guidance](https://www.postgresql.org/docs/current/sql-select.html) explains ordering and skipped-row implications. Search-engine `search_after` plus a managed point-in-time has a different retention/resource cost; see [Elasticsearch pagination](https://www.elastic.co/docs/reference/elasticsearch/rest-apis/paginate-search-results).

## Choose materialization deliberately

| Repeated work | Candidate | Added responsibility |
| --- | --- | --- |
| Stable expensive join/aggregation | Materialized projection | Update lag, version, refresh and rebuild |
| Same immutable payload | Canonical scoped reference | Lifetime, access and deletion |
| Frequent bounded existence check | Unique/indexed key or coherent negative cache | Current-state authority and invalidation |
| Sparse changed fields | Conditional update/delta | Historical reconstruction and compaction |
| Reporting over many rows | Preaggregation/columnar analytical projection | Ingestion, freshness and extra storage |

Use indexes matching scope, filter and ordering. Statistics and real selectivity decide whether scans, bitmap access or nested/hash/merge joins fit; do not replace the planner with a fixed rule. See [PostgreSQL plans](../storage/postgres-queries.md) and [SQLite planning](https://sqlite.org/optoverview.html).

Validate first/deep pages, skewed tenants, changed membership and large bodies. Compare rows/bytes visited, query count, pool residence, encoded output, index/WAL growth and p99 at concurrent load. A faster single query is a regression if its new index makes the write workload miss its budget.
