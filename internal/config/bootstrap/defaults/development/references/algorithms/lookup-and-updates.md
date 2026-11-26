# Lookup and changing collections

Choose the case that matches the operation and update frequency. Keep the canonical records and derived indexes under explicit ownership. Use the [data structure guide](data-structures.md) for operation costs and the [optimization workflow](optimization.md) for measurement.

## Repeatedly joining records by ID

- **Use when** a request, import or view repeatedly scans the same collection to resolve an ID. For example, every order line needs the product record from one already-loaded catalog.
- **Build** a map from product ID to its record once for the catalog revision. Resolve lines through that map. Decide how duplicate IDs, missing records and deleted products behave before constructing it. Keep a separate ordered sequence when callers need stable display order.
- **Cost and ownership.** Building an index for `n` records and resolving `m` keys takes expected `O(n + m)` work with ordinary hashing, plus index memory. A single scan can be sufficient for one lookup. Reuse the index only while its source revision and access scope remain valid. If the source is a database, use its batch-query or join capability through the [query-path guide](../backend/performance/query-paths.md).
- **Verify** missing and duplicate IDs, source replacement, deletion and equal IDs from different tenants or workspaces. Compare the result with a simple scan and measure index construction as well as lookup time.

## Prefix suggestions over a known dictionary

- **Use when** a command palette, path picker or tag editor needs a bounded list of prefix matches from a mostly stable dictionary.
- **Build** a sorted sequence of normalized search keys paired with stable IDs and display text. Binary-search the first key at or after the prefix, then scan while keys match and the result limit allows it. Apply the same normalization and lexical order during construction and lookup. A trie or radix tree is useful when the operation mix and measured prefix workload justify its nodes and update machinery.
- **Cost and ownership.** Include normalization, string comparison, dictionary sorting, updates and returned matches. Preserve originals for display. Version the index with its dictionary and reject late results from an earlier query. Define how case, accents, Unicode normalization and duplicate display names affect matches. For fuzzy or full-text search, use the [search guide](../backend/performance/search-and-ranking.md).
- **Verify** empty and long prefixes, equivalent Unicode forms, dictionary changes and a query that completes after a newer one. Check that result limits and stable ranking match the product's requirements.

## Reconciling successive snapshots

- **Use when** a synchronization pass receives a new collection and must identify inserts, updates and removals while preserving existing item identity.
- **Build** a map of prior items by stable ID. Walk the incoming snapshot once, match each item, compare the fields or revision that define a meaningful change and record new membership. Remaining prior IDs identify removals only when the incoming snapshot represents the complete same scope. Apply canonical records and their index changes through one owner.
- **Cost and ownership.** A hash-based pass uses expected `O(n + m)` matching work and proportional index space. If both inputs already share a key order, a merge walk supports incremental consumption. Define how duplicate IDs and versions out of order are rejected. Preserve unchanged references when consumers rely on identity. Track order separately when moved items matter.
- **Verify** an incomplete page, filter change, deleted item, reordered items and overlapping updates. Keep a partial page's missing records distinct from an explicit deletion. For visible collections, align identity and retained selection with the [interaction contract](../ui/interaction-contracts.md).

## Reusing expensive parsed or derived results

- **Use when** profiling shows the same immutable document, expression or payload is parsed or transformed repeatedly with the same options.
- **Build** a cache key from the complete input identity and every option that affects the result, including parser version and relevant visibility or redaction scope. Give the cached value immutable ownership. For content deduplication, hash candidates and verify content identity according to the required integrity guarantee before reusing them.
- **Cost and ownership.** Count hashing, misses, retained backing buffers and eviction. Bound retained bytes as well as entries. Discard an obsolete generation when a slower computation finishes after its source changes. Choose a request-owned cache when reuse is limited to one operation. Give shared in-flight work a lifetime that accommodates each waiter's cancellation.
- **Verify** equal bytes with different options, changed content with a reused filename, concurrent fills, cancellation and low-reuse input streams. Use the [cache-consistency guide](../backend/storage/cache-consistency.md) when reuse crosses mutable storage or independently running services.
