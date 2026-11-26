# Choose data structures by operations

List the operations and their mix before choosing a representation. Include build cost, updates, queries, removals, iteration, memory and invalidation. Prefer a maintained native container when it fits. Define key identity, ordering and lifetime in the owner that uses the structure.

Apply these choices through the practical [lookup and update cases](lookup-and-updates.md), [queue and stream cases](scheduling-and-streams.md), or [range and edit cases](ranges-and-edits.md). Each describes the state to keep, how to update it and which failures to check.

| Need | Fit and constraint |
| --- | --- |
| Exact membership or lookup | Use a set or hash map for expected constant-time lookup. Budget table overhead and define equality and key normalization. Use an explicit order when output must be stable. |
| Mostly static sorted keys with ranges | A sorted array gives binary search, contiguous iteration and cheap range traversal. Arbitrary insertions cost linear movement. Compare a batch merge or rebuild with a balanced ordered index when changes are frequent. |
| Next deadline or best item | Use a heap for logarithmic insertion/removal. Use stable tie-breakers when the result contract requires them. For mutable priorities, track handles or discard stale entries and bound their retention. |
| Exact top `k` from a stream | Keep a heap of at most `k` entries for `O(n log k)` time when `k > 1` and `O(k)` space. Sort the result at the end if required. Account for a tie at the cutoff and the case where the caller asks for nearly every item. |
| Moving minimum or maximum in a fixed-order window | A monotonic deque removes expired entries at the front and dominated entries at the back. Each entry enters and leaves at most once, so processing is linear. Keep positions for expiry and choose how to handle equal values. |
| Prefix sums and point updates over indexed keys | A Fenwick tree supports updates and prefix sums in `O(log n)` with linear storage. Use differences for range sums. Prefix lower-bound search requires monotone prefix totals. [AtCoder Fenwick tree](https://atcoder.github.io/ac-library/production/document_en/fenwicktree.html) |
| Range aggregates under changing values | Use a segment tree for associative combine operations. Lazy range updates require update tags that compose and can update the summary with the information stored at each node. Check identity, tag order and range lengths. [AtCoder lazy segment tree](https://atcoder.github.io/ac-library/master/document_en/lazysegtree.html) |
| Dense bounded membership or set algebra | A bitset uses one bit per possible position before padding and metadata. Use word operations for unions and intersections. Use a sparse set or compressed representation when keys have a large sparse universe. |
| Undirected connectivity with edge additions | Union by size and path compression give disjoint sets nearly constant amortized operations. For offline rollback, use reversible changes and avoid ordinary path compression. For arbitrary online deletion, choose a different representation or rebuild when justified. |
| Many prefix lookups | A trie or radix tree shares paths at the cost of additional nodes and references. Compare a sorted key index for read-heavy use. Define the alphabet, normalization and whether input is indexed in bytes or characters. |

## Map keys to the right representation

Coordinate compression maps sparse ordered keys to dense ranks. Use it when queries depend on key order and the key set is known at build time. Preserve original endpoints for calculations that depend on distances or durations. New keys can require a rebuild or an ordered structure that accepts inserts.

For example, to count occupied slots over arbitrary time ranges, sorted interval events and prefix sums may be enough for immutable inputs. For online point updates and sum queries on a known coordinate set, a Fenwick tree avoids rebuilding every prefix. A maximum-overlap query under range updates needs a different aggregate and can use a segment tree whose nodes maintain range maxima and pending additions.

## Check cost at the owning boundary

Account for object headers, pointers, duplicated keys and capacity retained after a spike. Measure bytes per item and cache behavior with representative cardinalities. Avoid keeping all originals plus a full index unless both serve a real consumer. Use an eviction, rebuild or release rule appropriate to the structure's lifetime.

Exercise updates, deletes, empty ranges, duplicate keys, adversarial ordering and any stale-entry handling. Compare results with a simple map, sorted scan or direct calculation on controlled inputs. Keep concurrency and snapshots explicit when multiple consumers see the structure.
